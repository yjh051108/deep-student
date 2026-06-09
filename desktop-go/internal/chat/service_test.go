package chat

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"
)

func newTestService(t *testing.T) *Service {
	t.Helper()
	service, err := NewService(t.TempDir())
	if err != nil {
		t.Fatalf("NewService() error = %v", err)
	}
	return service
}

func TestSessionLifecycleAndPersistence(t *testing.T) {
	dir := t.TempDir()
	service, err := NewService(dir)
	if err != nil {
		t.Fatalf("NewService() error = %v", err)
	}
	title := "Draft"
	session, err := service.CreateSession("chat", &title, map[string]any{"hidden": true}, nil)
	if err != nil {
		t.Fatalf("CreateSession() error = %v", err)
	}
	if session.ID == "" || session.PersistStatus != "active" || session.Title == nil || *session.Title != "Draft" {
		t.Fatalf("unexpected session: %+v", session)
	}
	if _, err := service.SaveSession(session.ID, SessionState{"inputValue": "hello"}); err != nil {
		t.Fatalf("SaveSession() error = %v", err)
	}

	reloaded, err := NewService(dir)
	if err != nil {
		t.Fatalf("NewService(reloaded) error = %v", err)
	}
	loaded, err := reloaded.LoadSession(session.ID)
	if err != nil {
		t.Fatalf("LoadSession() error = %v", err)
	}
	if loaded.State["inputValue"] != "hello" || len(loaded.Messages) != 0 || len(loaded.Blocks) != 0 {
		t.Fatalf("unexpected loaded session: %+v", loaded)
	}
	count, err := reloaded.CountSessions(strPtr("active"), nil)
	if err != nil {
		t.Fatalf("CountSessions() error = %v", err)
	}
	if count != 1 {
		t.Fatalf("CountSessions() = %d", count)
	}
	if _, err := reloaded.ArchiveSession(session.ID); err != nil {
		t.Fatalf("ArchiveSession() error = %v", err)
	}
	count, _ = reloaded.CountSessions(strPtr("active"), nil)
	if count != 0 {
		t.Fatalf("active count after archive = %d", count)
	}
}

func TestGroupsAndSessionFiltering(t *testing.T) {
	service := newTestService(t)
	group, err := service.CreateGroup(CreateGroupRequest{
		Name:            "Math",
		DefaultSkillIDs: []string{"skill_a"},
	})
	if err != nil {
		t.Fatalf("CreateGroup() error = %v", err)
	}
	if group.SortOrder != 0 || len(group.DefaultSkillIDs) != 1 {
		t.Fatalf("unexpected group: %+v", group)
	}
	session, err := service.CreateSession("chat", nil, nil, &group.ID)
	if err != nil {
		t.Fatalf("CreateSession(group) error = %v", err)
	}
	grouped, err := service.ListSessions(strPtr("active"), strPtr("*"), 50, 0)
	if err != nil {
		t.Fatalf("ListSessions(grouped) error = %v", err)
	}
	if len(grouped) != 1 || grouped[0].ID != session.ID {
		t.Fatalf("unexpected grouped sessions: %+v", grouped)
	}
	ungrouped, err := service.ListSessions(strPtr("active"), strPtr(""), 50, 0)
	if err != nil {
		t.Fatalf("ListSessions(ungrouped) error = %v", err)
	}
	if len(ungrouped) != 0 {
		t.Fatalf("unexpected ungrouped sessions: %+v", ungrouped)
	}
	if _, err := service.MoveSessionToGroup(session.ID, nil); err != nil {
		t.Fatalf("MoveSessionToGroup(nil) error = %v", err)
	}
	ungrouped, _ = service.ListSessions(strPtr("active"), strPtr(""), 50, 0)
	if len(ungrouped) != 1 {
		t.Fatalf("ungrouped after move = %+v", ungrouped)
	}
}

func TestGroupUpdateAndReorder(t *testing.T) {
	service := newTestService(t)
	first, err := service.CreateGroup(CreateGroupRequest{Name: "A"})
	if err != nil {
		t.Fatalf("CreateGroup(A) error = %v", err)
	}
	second, err := service.CreateGroup(CreateGroupRequest{Name: "B"})
	if err != nil {
		t.Fatalf("CreateGroup(B) error = %v", err)
	}
	if _, err := service.ReorderGroups([]string{second.ID, first.ID}); err != nil {
		t.Fatalf("ReorderGroups() error = %v", err)
	}
	groups, err := service.ListGroups(strPtr("active"), nil)
	if err != nil {
		t.Fatalf("ListGroups() error = %v", err)
	}
	if len(groups) != 2 || groups[0].ID != second.ID {
		t.Fatalf("unexpected reordered groups: %+v", groups)
	}
	archived, err := service.UpdateGroup(first.ID, UpdateGroupRequest{PersistStatus: strPtr("archived")})
	if err != nil {
		t.Fatalf("UpdateGroup() error = %v", err)
	}
	if archived.PersistStatus != "archived" {
		t.Fatalf("unexpected archived group: %+v", archived)
	}
	active, _ := service.ListGroups(strPtr("active"), nil)
	if len(active) != 1 || active[0].ID != second.ID {
		t.Fatalf("unexpected active groups: %+v", active)
	}
}

func TestRestoreAndDeleteSession(t *testing.T) {
	service := newTestService(t)
	session, err := service.CreateSession("chat", nil, nil, nil)
	if err != nil {
		t.Fatalf("CreateSession() error = %v", err)
	}
	if _, err := service.SaveSession(session.ID, SessionState{"draft": "hello"}); err != nil {
		t.Fatalf("SaveSession() error = %v", err)
	}
	if _, err := service.SendMessage(SendMessageRequest{
		SessionID:          session.ID,
		Content:            "hello",
		UserMessageID:      strPtr("msg_restore_user"),
		AssistantMessageID: strPtr("msg_restore_assistant"),
	}); err != nil {
		t.Fatalf("SendMessage() error = %v", err)
	}

	if _, err := service.ArchiveSession(session.ID); err != nil {
		t.Fatalf("ArchiveSession() error = %v", err)
	}
	restored, err := service.RestoreSession(session.ID)
	if err != nil {
		t.Fatalf("RestoreSession() error = %v", err)
	}
	if restored.PersistStatus != "active" {
		t.Fatalf("restored status = %s", restored.PersistStatus)
	}

	if ok, err := service.DeleteSession(session.ID); err != nil || !ok {
		t.Fatalf("DeleteSession() returned %v, %v", ok, err)
	}
	if _, err := service.LoadSession(session.ID); err == nil {
		t.Fatal("LoadSession() succeeded after permanent delete")
	}
	service.mu.RLock()
	defer service.mu.RUnlock()
	if _, ok := service.state.States[session.ID]; ok {
		t.Fatal("session state was not removed")
	}
	for _, message := range service.state.Messages {
		if message.SessionID == session.ID {
			t.Fatalf("message was not removed: %+v", message)
		}
	}
	for _, block := range service.state.Blocks {
		if block.MessageID == "msg_restore_user" || block.MessageID == "msg_restore_assistant" {
			t.Fatalf("block was not removed: %+v", block)
		}
	}
}

func TestDeleteAndRestoreGroup(t *testing.T) {
	service := newTestService(t)
	group, err := service.CreateGroup(CreateGroupRequest{Name: "Archive"})
	if err != nil {
		t.Fatalf("CreateGroup() error = %v", err)
	}
	session, err := service.CreateSession("chat", nil, nil, &group.ID)
	if err != nil {
		t.Fatalf("CreateSession(group) error = %v", err)
	}

	deleted, err := service.DeleteGroup(group.ID)
	if err != nil || !deleted {
		t.Fatalf("DeleteGroup() returned %v, %v", deleted, err)
	}
	deletedGroups, err := service.ListGroups(strPtr("deleted"), nil)
	if err != nil {
		t.Fatalf("ListGroups(deleted) error = %v", err)
	}
	if len(deletedGroups) != 1 || deletedGroups[0].ID != group.ID {
		t.Fatalf("unexpected deleted groups: %+v", deletedGroups)
	}
	loadedSession, err := service.GetSession(session.ID)
	if err != nil {
		t.Fatalf("GetSession() error = %v", err)
	}
	if loadedSession == nil || loadedSession.GroupID != nil {
		t.Fatalf("session should be ungrouped after DeleteGroup: %+v", loadedSession)
	}

	restored, err := service.RestoreGroup(group.ID)
	if err != nil {
		t.Fatalf("RestoreGroup() error = %v", err)
	}
	if restored.PersistStatus != "active" {
		t.Fatalf("restored group status = %s", restored.PersistStatus)
	}
	activeGroups, _ := service.ListGroups(strPtr("active"), nil)
	if len(activeGroups) != 1 || activeGroups[0].ID != group.ID {
		t.Fatalf("unexpected active groups after restore: %+v", activeGroups)
	}
}

func TestSendLoadUpdateAndDeleteMessages(t *testing.T) {
	service := newTestService(t)
	session, err := service.CreateSession("chat", nil, nil, nil)
	if err != nil {
		t.Fatalf("CreateSession() error = %v", err)
	}
	userID := "msg_user_fixed"
	assistantID := "msg_asst_fixed"
	returnedID, err := service.SendMessage(SendMessageRequest{
		SessionID:          session.ID,
		Content:            "hello world",
		UserMessageID:      &userID,
		AssistantMessageID: &assistantID,
		PathMap:            map[string]string{"res_1": "/note_1"},
	})
	if err != nil {
		t.Fatalf("SendMessage() error = %v", err)
	}
	if returnedID != assistantID {
		t.Fatalf("returned assistant id = %s", returnedID)
	}
	loaded, err := service.LoadSession(session.ID)
	if err != nil {
		t.Fatalf("LoadSession() error = %v", err)
	}
	if len(loaded.Messages) != 2 || len(loaded.Blocks) != 2 {
		t.Fatalf("unexpected loaded messages/blocks: %+v", loaded)
	}
	if loaded.Messages[0].ID != userID || loaded.Messages[1].ID != assistantID {
		t.Fatalf("unexpected message order: %+v", loaded.Messages)
	}
	userBlockID := loaded.Messages[0].BlockIDs[0]
	if _, err := service.UpdateBlockContent(userBlockID, "edited"); err != nil {
		t.Fatalf("UpdateBlockContent() error = %v", err)
	}
	loaded, _ = service.LoadSession(session.ID)
	if loaded.Blocks[0].Content != "edited" {
		t.Fatalf("block content not updated: %+v", loaded.Blocks)
	}
	if _, err := service.ContinueMessage(session.ID, assistantID, nil); err != nil {
		t.Fatalf("ContinueMessage() error = %v", err)
	}
	loaded, _ = service.LoadSession(session.ID)
	if len(loaded.Blocks) != 3 {
		t.Fatalf("continue did not append block: %+v", loaded.Blocks)
	}
	deleted, err := service.DeleteMessage(session.ID, userID)
	if err != nil {
		t.Fatalf("DeleteMessage() error = %v", err)
	}
	if !deleted {
		t.Fatal("DeleteMessage() = false")
	}
	loaded, _ = service.LoadSession(session.ID)
	if len(loaded.Messages) != 1 || loaded.Messages[0].ID != assistantID {
		t.Fatalf("unexpected messages after delete: %+v", loaded.Messages)
	}
}

func TestSendMessageEmitsLocalStreamEvents(t *testing.T) {
	service := newTestService(t)
	session, err := service.CreateSession("chat", nil, nil, nil)
	if err != nil {
		t.Fatalf("CreateSession() error = %v", err)
	}

	events := []struct {
		name    string
		payload any
	}{}
	service.SetEventEmitter(func(name string, payload any) {
		events = append(events, struct {
			name    string
			payload any
		}{name: name, payload: payload})
	})

	userID := "msg_user_events"
	assistantID := "msg_asst_events"
	if _, err := service.SendMessage(SendMessageRequest{
		SessionID:          session.ID,
		Content:            "hello event bridge",
		UserMessageID:      &userID,
		AssistantMessageID: &assistantID,
	}); err != nil {
		t.Fatalf("SendMessage() error = %v", err)
	}

	if len(events) < 5 {
		t.Fatalf("expected stream event sequence, got %+v", events)
	}
	if events[0].name != "chat_v2_session_"+session.ID {
		t.Fatalf("unexpected first event channel: %+v", events[0])
	}
	start, ok := events[0].payload.(SessionEventPayload)
	if !ok || start.EventType != "stream_start" || start.MessageID != assistantID {
		t.Fatalf("unexpected stream_start payload: %#v", events[0].payload)
	}

	blockStart, ok := events[1].payload.(BackendEvent)
	if !ok || blockStart.Type != "content" || blockStart.Phase != "start" || blockStart.MessageID != assistantID || blockStart.BlockID == "" || blockStart.SequenceID != 1 {
		t.Fatalf("unexpected block start payload: %#v", events[1].payload)
	}
	var sawChunk bool
	var sawEnd bool
	for _, event := range events {
		backend, ok := event.payload.(BackendEvent)
		if !ok {
			continue
		}
		if backend.Phase == "chunk" && backend.BlockID == blockStart.BlockID && backend.Chunk != "" && backend.SequenceID > 1 {
			sawChunk = true
		}
		if backend.Phase == "end" && backend.BlockID == blockStart.BlockID && backend.SequenceID > 1 {
			sawEnd = true
		}
	}
	if !sawChunk || !sawEnd {
		t.Fatalf("missing chunk/end events: %+v", events)
	}

	last, ok := events[len(events)-1].payload.(SessionEventPayload)
	if !ok || last.EventType != "stream_complete" || last.MessageID != assistantID {
		t.Fatalf("unexpected final event: %#v", events[len(events)-1].payload)
	}
}

func TestSendMessageStreamsOpenAICompatibleProvider(t *testing.T) {
	var requestBody string
	provider := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/chat/completions" {
			t.Fatalf("unexpected provider path %s", r.URL.Path)
		}
		if r.Header.Get("Authorization") != "Bearer test-key" {
			t.Fatalf("unexpected authorization header %q", r.Header.Get("Authorization"))
		}
		body, err := io.ReadAll(r.Body)
		if err != nil {
			t.Fatalf("read body: %v", err)
		}
		requestBody = string(body)
		w.Header().Set("Content-Type", "text/event-stream")
		_, _ = w.Write([]byte("data: {\"choices\":[{\"delta\":{\"content\":\"hello \"}}]}\n\n"))
		_, _ = w.Write([]byte("data: {\"choices\":[{\"delta\":{\"content\":\"provider\"}}]}\n\n"))
		_, _ = w.Write([]byte("data: {\"choices\":[],\"usage\":{\"prompt_tokens\":11,\"completion_tokens\":7,\"total_tokens\":18,\"prompt_tokens_details\":{\"cached_tokens\":3},\"completion_tokens_details\":{\"reasoning_tokens\":2}}}\n\n"))
		_, _ = w.Write([]byte("data: [DONE]\n\n"))
	}))
	defer provider.Close()

	service := newTestService(t)
	service.SetAPIConfigLoader(func() ([]ApiConfig, error) {
		return []ApiConfig{{
			ID:           "api_provider",
			ApiKey:       "test-key",
			BaseUrl:      provider.URL + "/v1",
			Model:        "model-a",
			Enabled:      true,
			Headers:      map[string]string{"X-Test": "yes"},
			ModelAdapter: "general",
		}}, nil
	})
	session, err := service.CreateSession("chat", nil, nil, nil)
	if err != nil {
		t.Fatalf("CreateSession() error = %v", err)
	}
	events := []any{}
	service.SetEventEmitter(func(_ string, payload any) {
		events = append(events, payload)
	})

	assistantID := "msg_provider"
	if _, err := service.SendMessage(SendMessageRequest{
		SessionID:          session.ID,
		Content:            "call provider",
		AssistantMessageID: &assistantID,
		Options:            map[string]any{"modelId": "api_provider", "temperature": 0.2, "maxTokens": 33},
	}); err != nil {
		t.Fatalf("SendMessage() error = %v", err)
	}
	if !strings.Contains(requestBody, `"model":"model-a"`) || !strings.Contains(requestBody, `"stream":true`) || !strings.Contains(requestBody, `"max_tokens":33`) || !strings.Contains(requestBody, `"include_usage":true`) {
		t.Fatalf("request body missing expected fields: %s", requestBody)
	}
	loaded, err := service.LoadSession(session.ID)
	if err != nil {
		t.Fatalf("LoadSession() error = %v", err)
	}
	assistant := findTestMessage(loaded.Messages, assistantID)
	if assistant == nil || len(assistant.BlockIDs) != 1 {
		t.Fatalf("assistant message missing provider block: %+v", loaded.Messages)
	}
	block := findTestBlock(loaded.Blocks, assistant.BlockIDs[0])
	if block == nil || block.Content != "hello provider" || block.Status != "complete" {
		t.Fatalf("provider block not persisted: %+v", block)
	}
	if assistant.Meta["modelId"] != "api_provider" {
		t.Fatalf("assistant model meta was not updated: %+v", assistant.Meta)
	}
	usage, ok := assistant.Meta["usage"].(map[string]any)
	if !ok {
		t.Fatalf("assistant usage meta missing: %+v", assistant.Meta)
	}
	assertTestUsage(t, usage, 11, 7, 18, 2, 3)
	var sawProviderChunk bool
	var sawUsageComplete bool
	for _, event := range events {
		backend, ok := event.(BackendEvent)
		if ok && backend.Phase == "chunk" && backend.Chunk == "provider" {
			sawProviderChunk = true
		}
		sessionEvent, ok := event.(SessionEventPayload)
		if ok && sessionEvent.EventType == "stream_complete" && sessionEvent.MessageID == assistantID {
			assertTestUsage(t, sessionEvent.Usage, 11, 7, 18, 2, 3)
			sawUsageComplete = true
		}
	}
	if !sawProviderChunk {
		t.Fatalf("provider chunks were not emitted: %+v", events)
	}
	if !sawUsageComplete {
		t.Fatalf("stream_complete usage was not emitted: %+v", events)
	}
}

func TestLLMUsageStatsDeriveFromChatMessageMeta(t *testing.T) {
	service := newTestService(t)
	session, err := service.CreateSession("chat", nil, nil, nil)
	if err != nil {
		t.Fatalf("CreateSession() error = %v", err)
	}
	now := time.Now()
	firstAt := now.AddDate(0, 0, -1)
	secondAt := now
	service.mu.Lock()
	service.state.Messages = append(service.state.Messages,
		Message{
			ID:        "msg_usage_one",
			SessionID: session.ID,
			Role:      "assistant",
			BlockIDs:  []string{"blk_usage_one"},
			Timestamp: firstAt.UnixMilli(),
			Meta: map[string]any{
				"modelId": "api_a",
				"usage": map[string]any{
					"promptTokens":          10,
					"completionTokens":      5,
					"totalTokens":           15,
					"reasoningTokens":       2,
					"cachedTokens":          3,
					"lastRoundPromptTokens": 15,
					"source":                "api",
				},
			},
		},
		Message{
			ID:        "msg_usage_two",
			SessionID: session.ID,
			Role:      "assistant",
			BlockIDs:  []string{"blk_usage_two"},
			Timestamp: secondAt.UnixMilli(),
			Meta: map[string]any{
				"modelId": "api_b",
				"usage": map[string]any{
					"promptTokens":          20,
					"completionTokens":      10,
					"totalTokens":           30,
					"lastRoundPromptTokens": 30,
					"source":                "api",
				},
			},
		},
	)
	service.mu.Unlock()

	start := firstAt.AddDate(0, 0, -1).Format("2006-01-02")
	end := secondAt.Format("2006-01-02")
	summary, err := service.LLMUsageSummary(&start, &end)
	if err != nil {
		t.Fatalf("LLMUsageSummary() error = %v", err)
	}
	if summary.TotalRequests != 2 || summary.TotalPromptTokens != 30 || summary.TotalCompletionTokens != 15 || summary.TotalTokens != 45 {
		t.Fatalf("unexpected usage summary: %+v", summary)
	}
	if summary.TotalReasoningTokens == nil || *summary.TotalReasoningTokens != 2 || summary.TotalCachedTokens == nil || *summary.TotalCachedTokens != 3 {
		t.Fatalf("unexpected optional totals: %+v", summary)
	}
	if summary.AvgTokensPerRequest == nil || *summary.AvgTokensPerRequest != 22.5 {
		t.Fatalf("unexpected avg tokens: %+v", summary.AvgTokensPerRequest)
	}

	byModel, err := service.LLMUsageByModel(start, end)
	if err != nil {
		t.Fatalf("LLMUsageByModel() error = %v", err)
	}
	if len(byModel) != 2 || byModel[0].ModelID != "api_b" || byModel[0].TotalTokens != 30 || byModel[1].ModelID != "api_a" {
		t.Fatalf("unexpected by-model stats: %+v", byModel)
	}
	byCaller, err := service.LLMUsageByCaller(start, end)
	if err != nil {
		t.Fatalf("LLMUsageByCaller() error = %v", err)
	}
	if len(byCaller) != 1 || byCaller[0].CallerType != "chat" || byCaller[0].RequestCount != 2 || byCaller[0].TotalTokens != 45 {
		t.Fatalf("unexpected by-caller stats: %+v", byCaller)
	}
	recent, err := service.LLMUsageRecent(1)
	if err != nil {
		t.Fatalf("LLMUsageRecent() error = %v", err)
	}
	if len(recent) != 1 || recent[0].ID != "usage_msg_usage_two" || recent[0].ModelID != "api_b" {
		t.Fatalf("unexpected recent records: %+v", recent)
	}
	daily, err := service.LLMUsageDaily(start, end)
	if err != nil {
		t.Fatalf("LLMUsageDaily() error = %v", err)
	}
	if len(daily) == 0 {
		t.Fatalf("daily stats missing: %+v", daily)
	}
	trends, err := service.LLMUsageGetTrends(7, "day")
	if err != nil {
		t.Fatalf("LLMUsageGetTrends() error = %v", err)
	}
	if len(trends) == 0 {
		t.Fatalf("trend stats missing: %+v", trends)
	}

	before := now.AddDate(0, 0, 1).Format("2006-01-02")
	cleared, err := service.LLMUsageCleanup(before)
	if err != nil {
		t.Fatalf("LLMUsageCleanup() error = %v", err)
	}
	if cleared != 2 {
		t.Fatalf("LLMUsageCleanup() cleared %d, want 2", cleared)
	}
	summaryAfterCleanup, err := service.LLMUsageSummary(&start, &end)
	if err != nil {
		t.Fatalf("LLMUsageSummary(after cleanup) error = %v", err)
	}
	if summaryAfterCleanup.TotalRequests != 0 || summaryAfterCleanup.TotalTokens != 0 {
		t.Fatalf("usage cleanup did not clear derived stats: %+v", summaryAfterCleanup)
	}
}

func TestRetryMessageStreamsOpenAICompatibleProvider(t *testing.T) {
	requestBodies := []string{}
	provider := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, err := io.ReadAll(r.Body)
		if err != nil {
			t.Fatalf("read body: %v", err)
		}
		requestBodies = append(requestBodies, string(body))
		w.Header().Set("Content-Type", "text/event-stream")
		if len(requestBodies) == 1 {
			_, _ = w.Write([]byte("data: {\"choices\":[{\"delta\":{\"content\":\"first answer\"}}]}\n\n"))
		} else {
			_, _ = w.Write([]byte("data: {\"choices\":[{\"delta\":{\"content\":\"retry answer\"}}]}\n\n"))
		}
		_, _ = w.Write([]byte("data: [DONE]\n\n"))
	}))
	defer provider.Close()

	service := newTestService(t)
	service.SetAPIConfigLoader(func() ([]ApiConfig, error) {
		return []ApiConfig{{
			ID:      "api_retry",
			ApiKey:  "test-key",
			BaseUrl: provider.URL + "/v1",
			Model:   "model-retry",
			Enabled: true,
		}}, nil
	})
	session, err := service.CreateSession("chat", nil, nil, nil)
	if err != nil {
		t.Fatalf("CreateSession() error = %v", err)
	}
	assistantID := "msg_retry_provider"
	if _, err := service.SendMessage(SendMessageRequest{
		SessionID:          session.ID,
		Content:            "original prompt",
		AssistantMessageID: &assistantID,
		Options:            map[string]any{"modelId": "api_retry"},
	}); err != nil {
		t.Fatalf("SendMessage() error = %v", err)
	}
	if _, err := service.RetryMessage(session.ID, assistantID, map[string]any{"modelId": "api_retry"}); err != nil {
		t.Fatalf("RetryMessage() error = %v", err)
	}
	if len(requestBodies) != 2 {
		t.Fatalf("expected two provider requests, got %d: %+v", len(requestBodies), requestBodies)
	}
	if !strings.Contains(requestBodies[1], `"content":"original prompt"`) {
		t.Fatalf("retry did not replay original user content: %s", requestBodies[1])
	}
	loaded, err := service.LoadSession(session.ID)
	if err != nil {
		t.Fatalf("LoadSession() error = %v", err)
	}
	assistant := findTestMessage(loaded.Messages, assistantID)
	if assistant == nil || len(assistant.BlockIDs) != 1 {
		t.Fatalf("assistant message missing retry block: %+v", loaded.Messages)
	}
	block := findTestBlock(loaded.Blocks, assistant.BlockIDs[0])
	if block == nil || block.Content != "retry answer" || block.Status != "complete" {
		t.Fatalf("retry provider block not persisted: %+v", block)
	}
}

func TestContinueMessageStreamsOpenAICompatibleProvider(t *testing.T) {
	requestBodies := []string{}
	provider := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, err := io.ReadAll(r.Body)
		if err != nil {
			t.Fatalf("read body: %v", err)
		}
		requestBodies = append(requestBodies, string(body))
		w.Header().Set("Content-Type", "text/event-stream")
		if len(requestBodies) == 1 {
			_, _ = w.Write([]byte("data: {\"choices\":[{\"delta\":{\"content\":\"first part\"}}]}\n\n"))
		} else {
			_, _ = w.Write([]byte("data: {\"choices\":[{\"delta\":{\"content\":\"continued part\"}}]}\n\n"))
		}
		_, _ = w.Write([]byte("data: [DONE]\n\n"))
	}))
	defer provider.Close()

	service := newTestService(t)
	service.SetAPIConfigLoader(func() ([]ApiConfig, error) {
		return []ApiConfig{{
			ID:      "api_continue",
			ApiKey:  "test-key",
			BaseUrl: provider.URL + "/v1",
			Model:   "model-continue",
			Enabled: true,
		}}, nil
	})
	session, err := service.CreateSession("chat", nil, nil, nil)
	if err != nil {
		t.Fatalf("CreateSession() error = %v", err)
	}
	events := []any{}
	service.SetEventEmitter(func(_ string, payload any) {
		events = append(events, payload)
	})

	assistantID := "msg_continue_provider"
	if _, err := service.SendMessage(SendMessageRequest{
		SessionID:          session.ID,
		Content:            "continue prompt",
		AssistantMessageID: &assistantID,
		Options:            map[string]any{"modelId": "api_continue"},
	}); err != nil {
		t.Fatalf("SendMessage() error = %v", err)
	}
	if _, err := service.ContinueMessage(session.ID, assistantID, map[string]any{"modelId": "api_continue"}); err != nil {
		t.Fatalf("ContinueMessage() error = %v", err)
	}
	if len(requestBodies) != 2 {
		t.Fatalf("expected two provider requests, got %d: %+v", len(requestBodies), requestBodies)
	}
	if !strings.Contains(requestBodies[1], `"content":"continue prompt"`) {
		t.Fatalf("continue did not replay original user content: %s", requestBodies[1])
	}
	loaded, err := service.LoadSession(session.ID)
	if err != nil {
		t.Fatalf("LoadSession() error = %v", err)
	}
	assistant := findTestMessage(loaded.Messages, assistantID)
	if assistant == nil || len(assistant.BlockIDs) != 2 {
		t.Fatalf("assistant message should keep original and continuation blocks: %+v", loaded.Messages)
	}
	block := findTestBlock(loaded.Blocks, assistant.BlockIDs[1])
	if block == nil || block.Content != "continued part" || block.Status != "complete" {
		t.Fatalf("continue provider block not persisted: %+v", block)
	}
	var sawContinueChunk bool
	for _, event := range events {
		backend, ok := event.(BackendEvent)
		if ok && backend.Phase == "chunk" && backend.Chunk == "continued part" {
			sawContinueChunk = true
		}
	}
	if !sawContinueChunk {
		t.Fatalf("continue chunks were not emitted: %+v", events)
	}
}

func TestEditAndResendStreamsOpenAICompatibleProvider(t *testing.T) {
	requestBodies := []string{}
	provider := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, err := io.ReadAll(r.Body)
		if err != nil {
			t.Fatalf("read body: %v", err)
		}
		requestBodies = append(requestBodies, string(body))
		w.Header().Set("Content-Type", "text/event-stream")
		if len(requestBodies) == 1 {
			_, _ = w.Write([]byte("data: {\"choices\":[{\"delta\":{\"content\":\"first answer\"}}]}\n\n"))
		} else {
			_, _ = w.Write([]byte("data: {\"choices\":[{\"delta\":{\"content\":\"edited answer\"}}]}\n\n"))
		}
		_, _ = w.Write([]byte("data: [DONE]\n\n"))
	}))
	defer provider.Close()

	service := newTestService(t)
	service.SetAPIConfigLoader(func() ([]ApiConfig, error) {
		return []ApiConfig{{
			ID:      "api_edit",
			ApiKey:  "test-key",
			BaseUrl: provider.URL + "/v1",
			Model:   "model-edit",
			Enabled: true,
		}}, nil
	})
	session, err := service.CreateSession("chat", nil, nil, nil)
	if err != nil {
		t.Fatalf("CreateSession() error = %v", err)
	}
	events := []any{}
	service.SetEventEmitter(func(_ string, payload any) {
		events = append(events, payload)
	})

	userID := "msg_edit_user"
	oldAssistantID := "msg_edit_old_assistant"
	if _, err := service.SendMessage(SendMessageRequest{
		SessionID:          session.ID,
		Content:            "original prompt",
		UserMessageID:      &userID,
		AssistantMessageID: &oldAssistantID,
		Options:            map[string]any{"modelId": "api_edit"},
	}); err != nil {
		t.Fatalf("SendMessage() error = %v", err)
	}

	newAssistantID := "msg_edit_new_assistant"
	edit, err := service.EditAndResend(EditAndResendRequest{
		SessionID:          session.ID,
		MessageID:          userID,
		NewContent:         "edited prompt",
		AssistantMessageID: &newAssistantID,
		NewContextRefs:     []any{map[string]any{"id": "ctx_edit"}},
		NewPathMap:         map[string]string{"ctx_edit": "/notes/edit.md"},
		Options:            map[string]any{"modelId": "api_edit"},
	})
	if err != nil {
		t.Fatalf("EditAndResend() error = %v", err)
	}
	if edit.NewMessageID != newAssistantID {
		t.Fatalf("EditAndResend() new message id = %s, want %s", edit.NewMessageID, newAssistantID)
	}
	if len(edit.DeletedMessageIDs) != 1 || edit.DeletedMessageIDs[0] != oldAssistantID {
		t.Fatalf("unexpected deleted messages: %+v", edit.DeletedMessageIDs)
	}
	if len(requestBodies) != 2 {
		t.Fatalf("expected two provider requests, got %d: %+v", len(requestBodies), requestBodies)
	}
	if !strings.Contains(requestBodies[1], `"content":"edited prompt"`) || strings.Contains(requestBodies[1], `"content":"original prompt"`) {
		t.Fatalf("edit resend did not send edited prompt only: %s", requestBodies[1])
	}

	loaded, err := service.LoadSession(session.ID)
	if err != nil {
		t.Fatalf("LoadSession() error = %v", err)
	}
	if findTestMessage(loaded.Messages, oldAssistantID) != nil {
		t.Fatalf("old assistant message still present: %+v", loaded.Messages)
	}
	user := findTestMessage(loaded.Messages, userID)
	if user == nil || len(user.BlockIDs) != 1 {
		t.Fatalf("edited user message missing: %+v", loaded.Messages)
	}
	userBlock := findTestBlock(loaded.Blocks, user.BlockIDs[0])
	if userBlock == nil || userBlock.Content != "edited prompt" {
		t.Fatalf("user block was not edited: %+v", userBlock)
	}
	assistant := findTestMessage(loaded.Messages, newAssistantID)
	if assistant == nil || len(assistant.BlockIDs) != 1 {
		t.Fatalf("new assistant missing provider block: %+v", loaded.Messages)
	}
	block := findTestBlock(loaded.Blocks, assistant.BlockIDs[0])
	if block == nil || block.Content != "edited answer" || block.Status != "complete" {
		t.Fatalf("edit provider block not persisted: %+v", block)
	}
	if assistant.Meta["modelId"] != "api_edit" {
		t.Fatalf("assistant model meta was not updated: %+v", assistant.Meta)
	}

	var sawEditStart bool
	var sawEditChunk bool
	for _, event := range events {
		if sessionEvent, ok := event.(SessionEventPayload); ok && sessionEvent.EventType == "stream_start" && sessionEvent.MessageID == newAssistantID {
			sawEditStart = true
		}
		if backendEvent, ok := event.(BackendEvent); ok && backendEvent.Phase == "chunk" && backendEvent.MessageID == newAssistantID && backendEvent.Chunk == "edited answer" {
			sawEditChunk = true
		}
	}
	if !sawEditStart || !sawEditChunk {
		t.Fatalf("missing edit stream events: %+v", events)
	}
}

func TestCancelStreamCancelsOpenAICompatibleProviderRequest(t *testing.T) {
	requestStarted := make(chan struct{})
	requestCancelled := make(chan struct{})
	releaseProvider := make(chan struct{})
	var startedOnce sync.Once

	provider := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		startedOnce.Do(func() { close(requestStarted) })
		w.Header().Set("Content-Type", "text/event-stream")
		if flusher, ok := w.(http.Flusher); ok {
			flusher.Flush()
		}
		select {
		case <-r.Context().Done():
			close(requestCancelled)
			return
		case <-releaseProvider:
			_, _ = w.Write([]byte("data: [DONE]\n\n"))
			return
		}
	}))
	defer provider.Close()
	defer close(releaseProvider)

	service := newTestService(t)
	service.SetAPIConfigLoader(func() ([]ApiConfig, error) {
		return []ApiConfig{{
			ID:      "api_cancel",
			ApiKey:  "test-key",
			BaseUrl: provider.URL + "/v1",
			Model:   "model-cancel",
			Enabled: true,
		}}, nil
	})
	session, err := service.CreateSession("chat", nil, nil, nil)
	if err != nil {
		t.Fatalf("CreateSession() error = %v", err)
	}
	events := []any{}
	var eventsMu sync.Mutex
	service.SetEventEmitter(func(_ string, payload any) {
		eventsMu.Lock()
		defer eventsMu.Unlock()
		events = append(events, payload)
	})

	assistantID := "msg_cancel_provider"
	sendDone := make(chan error, 1)
	go func() {
		_, err := service.SendMessage(SendMessageRequest{
			SessionID:          session.ID,
			Content:            "cancel prompt",
			AssistantMessageID: &assistantID,
			Options:            map[string]any{"modelId": "api_cancel"},
		})
		sendDone <- err
	}()

	select {
	case <-requestStarted:
	case <-time.After(2 * time.Second):
		t.Fatal("provider request did not start")
	}
	if ok, err := service.CancelStream(session.ID, assistantID); err != nil || !ok {
		t.Fatalf("CancelStream() = %v, %v", ok, err)
	}
	select {
	case <-requestCancelled:
	case <-time.After(2 * time.Second):
		t.Fatal("provider request was not cancelled")
	}
	select {
	case err := <-sendDone:
		if err != nil {
			t.Fatalf("SendMessage() error = %v", err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("SendMessage did not return after cancellation")
	}

	loaded, err := service.LoadSession(session.ID)
	if err != nil {
		t.Fatalf("LoadSession() error = %v", err)
	}
	assistant := findTestMessage(loaded.Messages, assistantID)
	if assistant == nil || len(assistant.BlockIDs) != 1 {
		t.Fatalf("assistant message missing cancel block: %+v", loaded.Messages)
	}
	block := findTestBlock(loaded.Blocks, assistant.BlockIDs[0])
	if block == nil || block.Status != "cancelled" || block.Error == nil || *block.Error == "" {
		t.Fatalf("cancelled provider block not persisted: %+v", block)
	}

	eventsMu.Lock()
	defer eventsMu.Unlock()
	var sawCancelled bool
	for _, event := range events {
		sessionEvent, ok := event.(SessionEventPayload)
		if ok && sessionEvent.EventType == "stream_cancelled" && sessionEvent.MessageID == assistantID {
			sawCancelled = true
		}
	}
	if !sawCancelled {
		t.Fatalf("stream_cancelled was not emitted: %+v", events)
	}
}

func TestUpsertStreamingBlockCreatesAndUpdates(t *testing.T) {
	service := newTestService(t)
	session, err := service.CreateSession("chat", nil, nil, nil)
	if err != nil {
		t.Fatalf("CreateSession() error = %v", err)
	}
	sessionID := session.ID
	if _, err := service.UpsertStreamingBlock("blk_1", "msg_1", "markdown", "first", &sessionID); err != nil {
		t.Fatalf("UpsertStreamingBlock(create) error = %v", err)
	}
	if _, err := service.UpsertStreamingBlock("blk_1", "msg_1", "markdown", "second", &sessionID); err != nil {
		t.Fatalf("UpsertStreamingBlock(update) error = %v", err)
	}
	loaded, err := service.LoadSession(session.ID)
	if err != nil {
		t.Fatalf("LoadSession() error = %v", err)
	}
	if len(loaded.Messages) != 1 || len(loaded.Blocks) != 1 || loaded.Blocks[0].Content != "second" {
		t.Fatalf("unexpected streaming block state: %+v", loaded)
	}
}

func TestCancelRetryAndEditMessages(t *testing.T) {
	service := newTestService(t)
	session, err := service.CreateSession("chat", nil, nil, nil)
	if err != nil {
		t.Fatalf("CreateSession() error = %v", err)
	}
	userID := "msg_user_edit"
	assistantID := "msg_asst_retry"
	if _, err := service.SendMessage(SendMessageRequest{
		SessionID:          session.ID,
		Content:            "first prompt",
		UserMessageID:      &userID,
		AssistantMessageID: &assistantID,
	}); err != nil {
		t.Fatalf("SendMessage() error = %v", err)
	}
	sessionID := session.ID
	if _, err := service.UpsertStreamingBlock("blk_stream", assistantID, "markdown", "partial", &sessionID); err != nil {
		t.Fatalf("UpsertStreamingBlock() error = %v", err)
	}
	if _, err := service.CancelStream(session.ID, assistantID); err != nil {
		t.Fatalf("CancelStream() error = %v", err)
	}
	loaded, err := service.LoadSession(session.ID)
	if err != nil {
		t.Fatalf("LoadSession() error = %v", err)
	}
	streamBlock := findTestBlock(loaded.Blocks, "blk_stream")
	if streamBlock == nil || streamBlock.Status != "cancelled" {
		t.Fatalf("stream block was not cancelled: %+v", loaded.Blocks)
	}

	retry, err := service.RetryMessage(session.ID, assistantID, map[string]any{"modelId": "test-model"})
	if err != nil {
		t.Fatalf("RetryMessage() error = %v", err)
	}
	if retry.MessageID != assistantID {
		t.Fatalf("RetryMessage() message id = %s", retry.MessageID)
	}
	loaded, _ = service.LoadSession(session.ID)
	assistant := findTestMessage(loaded.Messages, assistantID)
	if assistant == nil || len(assistant.BlockIDs) != 1 {
		t.Fatalf("assistant message was not reused: %+v", loaded.Messages)
	}
	retryBlock := findTestBlock(loaded.Blocks, assistant.BlockIDs[0])
	if retryBlock == nil || retryBlock.Status != "complete" || retryBlock.Content == "" {
		t.Fatalf("retry block not replaced: %+v", loaded.Blocks)
	}

	edit, err := service.EditAndResend(EditAndResendRequest{
		SessionID:      session.ID,
		MessageID:      userID,
		NewContent:     "edited prompt",
		NewContextRefs: []any{map[string]any{"id": "ctx_1"}},
		NewPathMap:     map[string]string{"ctx_1": "/notes/a.md"},
		Options:        map[string]any{"modelId": "test-model"},
	})
	if err != nil {
		t.Fatalf("EditAndResend() error = %v", err)
	}
	if edit.NewMessageID == "" || len(edit.DeletedMessageIDs) != 1 || edit.DeletedMessageIDs[0] != assistantID {
		t.Fatalf("unexpected edit result: %+v", edit)
	}
	loaded, _ = service.LoadSession(session.ID)
	if findTestMessage(loaded.Messages, assistantID) != nil {
		t.Fatalf("old assistant message still present: %+v", loaded.Messages)
	}
	user := findTestMessage(loaded.Messages, userID)
	if user == nil {
		t.Fatalf("edited user message missing: %+v", loaded.Messages)
	}
	userBlock := findTestBlock(loaded.Blocks, user.BlockIDs[0])
	if userBlock == nil || userBlock.Content != "edited prompt" {
		t.Fatalf("user block was not edited: %+v", loaded.Blocks)
	}
	if findTestMessage(loaded.Messages, edit.NewMessageID) == nil {
		t.Fatalf("new assistant message missing: %+v", loaded.Messages)
	}
}

func TestSessionTags(t *testing.T) {
	service := newTestService(t)
	first, err := service.CreateSession("chat", nil, nil, nil)
	if err != nil {
		t.Fatalf("CreateSession(first) error = %v", err)
	}
	second, err := service.CreateSession("chat", nil, map[string]any{"tags": []any{"math"}}, nil)
	if err != nil {
		t.Fatalf("CreateSession(second) error = %v", err)
	}
	if _, err := service.AddTag(first.ID, "math"); err != nil {
		t.Fatalf("AddTag(math) error = %v", err)
	}
	if _, err := service.AddTag(first.ID, "math"); err != nil {
		t.Fatalf("AddTag(duplicate) error = %v", err)
	}
	if _, err := service.AddTag(first.ID, "calculus"); err != nil {
		t.Fatalf("AddTag(calculus) error = %v", err)
	}
	tags, err := service.GetSessionTags(first.ID)
	if err != nil {
		t.Fatalf("GetSessionTags() error = %v", err)
	}
	if len(tags) != 2 || tags[0] != "math" || tags[1] != "calculus" {
		t.Fatalf("unexpected tags: %+v", tags)
	}
	batch, err := service.GetTagsBatch([]string{first.ID, second.ID, "missing"})
	if err != nil {
		t.Fatalf("GetTagsBatch() error = %v", err)
	}
	if len(batch[first.ID]) != 2 || len(batch[second.ID]) != 1 || len(batch["missing"]) != 0 {
		t.Fatalf("unexpected batch tags: %+v", batch)
	}
	all, err := service.ListAllTags()
	if err != nil {
		t.Fatalf("ListAllTags() error = %v", err)
	}
	if len(all) != 2 || all[0][0] != "math" || all[0][1] != 2 {
		t.Fatalf("unexpected all tags: %+v", all)
	}
	if _, err := service.RemoveTag(first.ID, "math"); err != nil {
		t.Fatalf("RemoveTag() error = %v", err)
	}
	tags, _ = service.GetSessionTags(first.ID)
	if len(tags) != 1 || tags[0] != "calculus" {
		t.Fatalf("unexpected tags after remove: %+v", tags)
	}
}

func TestInteractionResponsesResolveBlocks(t *testing.T) {
	service := newTestService(t)
	session, err := service.CreateSession("chat", nil, nil, nil)
	if err != nil {
		t.Fatalf("CreateSession() error = %v", err)
	}
	assistantID := "msg_asst_tools"
	approvalCallID := "tool_approval_1"
	askCallID := "ask_user_1"
	now := timeNowMillisForTest()
	service.mu.Lock()
	service.state.Messages = append(service.state.Messages, Message{
		ID:        assistantID,
		SessionID: session.ID,
		Role:      "assistant",
		BlockIDs:  []string{"approval_" + approvalCallID, "ask_user_" + askCallID},
		Timestamp: now,
	})
	service.state.Blocks = append(service.state.Blocks,
		Block{
			ID:         "approval_" + approvalCallID,
			MessageID:  assistantID,
			Type:       "mcp_tool",
			Status:     "running",
			ToolName:   strPtr("write_file"),
			ToolCallID: strPtr(approvalCallID),
			ToolInput:  map[string]any{"path": "notes.md"},
		},
		Block{
			ID:         "ask_user_" + askCallID,
			MessageID:  assistantID,
			Type:       "ask_user",
			Status:     "running",
			ToolName:   strPtr("ask_user"),
			ToolCallID: strPtr(askCallID),
			ToolInput: map[string]any{
				"question": "Pick one",
				"options":  []any{"A", "B"},
			},
		},
	)
	service.mu.Unlock()

	reason := "ok"
	if _, err := service.RespondToolApproval(session.ID, approvalCallID, "write_file", true, &reason, true, map[string]any{"path": "notes.md"}); err != nil {
		t.Fatalf("RespondToolApproval() error = %v", err)
	}
	custom := "custom answer"
	if _, err := service.RespondAskUser(askCallID, []string{"A"}, []int{0}, &custom, "mixed"); err != nil {
		t.Fatalf("RespondAskUser() error = %v", err)
	}

	loaded, err := service.LoadSession(session.ID)
	if err != nil {
		t.Fatalf("LoadSession() error = %v", err)
	}
	approvalBlock := findTestBlock(loaded.Blocks, "approval_"+approvalCallID)
	if approvalBlock == nil || approvalBlock.Status != "approved" || approvalBlock.ToolOutput == nil {
		t.Fatalf("approval block not resolved: %+v", loaded.Blocks)
	}
	askBlock := findTestBlock(loaded.Blocks, "ask_user_"+askCallID)
	if askBlock == nil || askBlock.Status != "success" || askBlock.ToolOutput == nil {
		t.Fatalf("ask user block not resolved: %+v", loaded.Blocks)
	}
	if len(service.state.InteractionResponses) != 2 {
		t.Fatalf("responses were not recorded: %+v", service.state.InteractionResponses)
	}
	if len(service.state.ApprovalChoices) != 1 || !service.state.ApprovalChoices[0].Approved {
		t.Fatalf("approval choice was not remembered: %+v", service.state.ApprovalChoices)
	}
	cleared, err := service.ClearApprovalHistory()
	if err != nil {
		t.Fatalf("ClearApprovalHistory() error = %v", err)
	}
	if cleared != 1 || len(service.state.ApprovalChoices) != 0 {
		t.Fatalf("approval choices were not cleared: cleared=%d choices=%+v", cleared, service.state.ApprovalChoices)
	}
}

func TestMessageSummary(t *testing.T) {
	service := newTestService(t)
	first, err := service.CreateSession("chat", nil, nil, nil)
	if err != nil {
		t.Fatalf("CreateSession(first) error = %v", err)
	}
	second, err := service.CreateSession("chat", nil, nil, nil)
	if err != nil {
		t.Fatalf("CreateSession(second) error = %v", err)
	}
	if _, err := service.SendMessage(SendMessageRequest{SessionID: first.ID, Content: "one"}); err != nil {
		t.Fatalf("SendMessage(first) error = %v", err)
	}
	if _, err := service.SendMessage(SendMessageRequest{SessionID: second.ID, Content: "two"}); err != nil {
		t.Fatalf("SendMessage(second) error = %v", err)
	}
	summary, err := service.GetMessageSummary()
	if err != nil {
		t.Fatalf("GetMessageSummary() error = %v", err)
	}
	if summary.TotalMessages != 4 || summary.UserMessages != 2 || summary.AssistantMessages != 2 || summary.SessionsWithMessages != 2 {
		t.Fatalf("unexpected summary: %+v", summary)
	}
}

func TestBranchSessionCopiesHistoryThroughTarget(t *testing.T) {
	service := newTestService(t)
	group, err := service.CreateGroup(CreateGroupRequest{Name: "Study"})
	if err != nil {
		t.Fatalf("CreateGroup() error = %v", err)
	}
	title := "Source thread"
	session, err := service.CreateSession("analysis", &title, map[string]any{"tags": []any{"math"}}, &group.ID)
	if err != nil {
		t.Fatalf("CreateSession() error = %v", err)
	}
	if _, err := service.SaveSession(session.ID, SessionState{
		"inputValue":             "draft",
		"panelStates":            map[string]any{"left": true},
		"pendingContextRefsJson": "[]",
		"chatParams":             map[string]any{"temperature": 0.2},
	}); err != nil {
		t.Fatalf("SaveSession() error = %v", err)
	}

	userOne := "msg_user_one"
	assistantOne := "msg_asst_one"
	userTwo := "msg_user_two"
	assistantTwo := "msg_asst_two"
	if _, err := service.SendMessage(SendMessageRequest{
		SessionID:          session.ID,
		Content:            "first prompt",
		UserMessageID:      &userOne,
		AssistantMessageID: &assistantOne,
	}); err != nil {
		t.Fatalf("SendMessage(first) error = %v", err)
	}
	if _, err := service.SendMessage(SendMessageRequest{
		SessionID:          session.ID,
		Content:            "second prompt",
		UserMessageID:      &userTwo,
		AssistantMessageID: &assistantTwo,
	}); err != nil {
		t.Fatalf("SendMessage(second) error = %v", err)
	}
	service.mu.Lock()
	service.state.Messages[1].ParentID = &userOne
	assistantBlockID := service.state.Messages[1].BlockIDs[0]
	service.state.Blocks[1].ToolInput = map[string]any{"originating_block_id": assistantBlockID}
	service.state.Blocks[1].ToolOutput = map[string]any{"message_id": assistantOne}
	service.mu.Unlock()

	branched, err := service.BranchSession(session.ID, assistantOne)
	if err != nil {
		t.Fatalf("BranchSession() error = %v", err)
	}
	if branched.ID == session.ID || branched.PersistStatus != "active" || branched.Mode != "chat" {
		t.Fatalf("unexpected branched session: %+v", branched)
	}
	if branched.GroupID == nil || *branched.GroupID != group.ID {
		t.Fatalf("branch did not keep group: %+v", branched)
	}
	if branched.Title == nil || *branched.Title != "Source thread (branch)" || !branched.TitleLocked {
		t.Fatalf("branch title not locked from source: %+v", branched)
	}
	branchedFrom, ok := branched.Metadata["branchedFrom"].(map[string]any)
	if !ok || branchedFrom["sessionId"] != session.ID || branchedFrom["messageId"] != assistantOne {
		t.Fatalf("missing branchedFrom metadata: %+v", branched.Metadata)
	}

	loaded, err := service.LoadSession(branched.ID)
	if err != nil {
		t.Fatalf("LoadSession(branch) error = %v", err)
	}
	if len(loaded.Messages) != 2 || len(loaded.Blocks) != 2 {
		t.Fatalf("branch copied wrong history length: %+v", loaded)
	}
	if findTestMessage(loaded.Messages, userTwo) != nil || findTestMessage(loaded.Messages, assistantTwo) != nil {
		t.Fatalf("branch copied messages after target: %+v", loaded.Messages)
	}
	if loaded.Messages[0].ID == userOne || loaded.Messages[1].ID == assistantOne {
		t.Fatalf("branch reused source message IDs: %+v", loaded.Messages)
	}
	if loaded.Messages[1].ParentID == nil || *loaded.Messages[1].ParentID != loaded.Messages[0].ID {
		t.Fatalf("branch did not remap parent id: %+v", loaded.Messages)
	}
	if loaded.Blocks[0].ID == assistantBlockID || loaded.Blocks[1].ID == assistantBlockID {
		t.Fatalf("branch reused source block IDs: %+v", loaded.Blocks)
	}
	assistantBlock := findTestBlock(loaded.Blocks, loaded.Messages[1].BlockIDs[0])
	if assistantBlock == nil {
		t.Fatalf("assistant block missing in branch: %+v", loaded.Blocks)
	}
	input, ok := assistantBlock.ToolInput.(map[string]any)
	if !ok || input["originating_block_id"] != assistantBlock.ID {
		t.Fatalf("tool input IDs were not remapped: %+v", assistantBlock.ToolInput)
	}
	output, ok := assistantBlock.ToolOutput.(map[string]any)
	if !ok || output["message_id"] != loaded.Messages[1].ID {
		t.Fatalf("tool output IDs were not remapped: %+v", assistantBlock.ToolOutput)
	}
	if _, ok := loaded.State["inputValue"]; ok {
		t.Fatalf("branch kept input draft state: %+v", loaded.State)
	}
	if _, ok := loaded.State["panelStates"]; ok {
		t.Fatalf("branch kept panel UI state: %+v", loaded.State)
	}
	if loaded.State["chatParams"] == nil || loaded.State["sessionId"] != branched.ID {
		t.Fatalf("branch did not keep useful session state: %+v", loaded.State)
	}
}

func TestBranchSessionRejectsMissingTarget(t *testing.T) {
	service := newTestService(t)
	session, err := service.CreateSession("chat", nil, nil, nil)
	if err != nil {
		t.Fatalf("CreateSession() error = %v", err)
	}
	if _, err := service.BranchSession(session.ID, "missing"); err == nil {
		t.Fatal("BranchSession() succeeded for missing target")
	}
	if sessions, err := service.ListSessions(strPtr("active"), nil, 50, 0); err != nil || len(sessions) != 1 {
		t.Fatalf("missing target should not create a session, sessions=%+v err=%v", sessions, err)
	}
}

func findTestMessage(messages []Message, id string) *Message {
	for index := range messages {
		if messages[index].ID == id {
			return &messages[index]
		}
	}
	return nil
}

func findTestBlock(blocks []Block, id string) *Block {
	for index := range blocks {
		if blocks[index].ID == id {
			return &blocks[index]
		}
	}
	return nil
}

func assertTestUsage(t *testing.T, usage map[string]any, prompt int, completion int, total int, reasoning int, cached int) {
	t.Helper()
	if usage == nil {
		t.Fatal("usage is nil")
	}
	assertUsageInt := func(key string, want int) {
		t.Helper()
		got, ok := testIntFromAny(usage[key])
		if !ok || got != want {
			t.Fatalf("usage[%s] = %#v, want %d in %+v", key, usage[key], want, usage)
		}
	}
	assertUsageInt("promptTokens", prompt)
	assertUsageInt("completionTokens", completion)
	assertUsageInt("totalTokens", total)
	assertUsageInt("reasoningTokens", reasoning)
	assertUsageInt("cachedTokens", cached)
	assertUsageInt("lastRoundPromptTokens", prompt+completion)
	if usage["source"] != "api" {
		t.Fatalf("usage source = %#v, want api in %+v", usage["source"], usage)
	}
}

func testIntFromAny(value any) (int, bool) {
	switch typed := value.(type) {
	case int:
		return typed, true
	case int64:
		return int(typed), true
	case float64:
		return int(typed), true
	case json.Number:
		number, err := typed.Int64()
		if err == nil {
			return int(number), true
		}
	}
	return 0, false
}

func strPtr(value string) *string {
	return &value
}

func timeNowMillisForTest() int64 {
	return 1_700_000_000_000
}
