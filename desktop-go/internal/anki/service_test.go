package anki

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestMissingDocumentDefaults(t *testing.T) {
	service := newTestService(t)

	tasks, err := service.GetDocumentTasks("missing")
	if err != nil {
		t.Fatalf("GetDocumentTasks returned error: %v", err)
	}
	if len(tasks) != 0 {
		t.Fatalf("expected no tasks, got %d", len(tasks))
	}

	cards, err := service.GetDocumentCards("missing")
	if err != nil {
		t.Fatalf("GetDocumentCards returned error: %v", err)
	}
	if len(cards) != 0 {
		t.Fatalf("expected no cards, got %d", len(cards))
	}

	state, err := service.GetDocumentState("missing")
	if err != nil {
		t.Fatalf("GetDocumentState returned error: %v", err)
	}
	if state.Status != "pending" || state.TotalTasks != 0 || state.CompletedTasks != 0 || state.FailedTasks != 0 || state.PausedTasks != 0 {
		t.Fatalf("unexpected default state: %#v", state)
	}

	counts, err := service.GetDocumentTaskCounts("missing")
	if err != nil {
		t.Fatalf("GetDocumentTaskCounts returned error: %v", err)
	}
	if counts.Total != 0 || counts.Pending != 0 || counts.Completed != 0 {
		t.Fatalf("unexpected default counts: %#v", counts)
	}
}

func TestAnkiConnectMetadataQueries(t *testing.T) {
	actions := make(chan string, 3)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Fatalf("expected POST, got %s", r.Method)
		}
		var request struct {
			Action  string `json:"action"`
			Version int    `json:"version"`
		}
		if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
			t.Fatalf("decode request: %v", err)
		}
		if request.Version != 6 {
			t.Fatalf("expected AnkiConnect version 6, got %d", request.Version)
		}
		actions <- request.Action
		w.Header().Set("Content-Type", "application/json")
		switch request.Action {
		case "version":
			_ = json.NewEncoder(w).Encode(map[string]any{"result": 6, "error": nil})
		case "deckNames":
			_ = json.NewEncoder(w).Encode(map[string]any{"result": []string{"Default", "Biology"}, "error": nil})
		case "modelNames":
			_ = json.NewEncoder(w).Encode(map[string]any{"result": []string{"Basic", "Cloze"}, "error": nil})
		default:
			t.Fatalf("unexpected action %s", request.Action)
		}
	}))
	defer server.Close()

	service := newTestService(t)
	service.SetAnkiConnectURL(server.URL)

	ok, err := service.CheckConnectStatus()
	if err != nil || !ok {
		t.Fatalf("CheckConnectStatus returned %v, %v", ok, err)
	}
	decks, err := service.ListDeckNames()
	if err != nil {
		t.Fatalf("ListDeckNames returned error: %v", err)
	}
	if !containsString(decks, "Biology") {
		t.Fatalf("expected Biology deck, got %#v", decks)
	}
	models, err := service.ListModelNames()
	if err != nil {
		t.Fatalf("ListModelNames returned error: %v", err)
	}
	if !containsString(models, "Basic") || !containsString(models, "Cloze") {
		t.Fatalf("expected Basic and Cloze models, got %#v", models)
	}

	expectedActions := []string{"version", "deckNames", "modelNames"}
	for _, expected := range expectedActions {
		select {
		case action := <-actions:
			if action != expected {
				t.Fatalf("expected action %s, got %s", expected, action)
			}
		case <-time.After(time.Second):
			t.Fatalf("missing action %s", expected)
		}
	}
}

func TestAnkiConnectMetadataQueriesReturnAnkiErrors(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{"result": nil, "error": "collection unavailable"})
	}))
	defer server.Close()

	service := newTestService(t)
	service.SetAnkiConnectURL(server.URL)

	if _, err := service.ListModelNames(); err == nil || !strings.Contains(err.Error(), "collection unavailable") {
		t.Fatalf("expected AnkiConnect error, got %v", err)
	}
}

func TestPauseResumeCreatesAndUpdatesDocumentState(t *testing.T) {
	service := newTestService(t)

	ok, err := service.PauseDocumentProcessing("doc-1")
	if err != nil || !ok {
		t.Fatalf("PauseDocumentProcessing returned %v, %v", ok, err)
	}
	state, err := service.GetDocumentProcessingState("doc-1")
	if err != nil {
		t.Fatalf("GetDocumentProcessingState returned error: %v", err)
	}
	if state.Status != "paused" {
		t.Fatalf("expected paused state, got %#v", state)
	}

	ok, err = service.ResumeDocumentProcessing("doc-1")
	if err != nil || !ok {
		t.Fatalf("ResumeDocumentProcessing returned %v, %v", ok, err)
	}
	state, err = service.GetDocumentState("doc-1")
	if err != nil {
		t.Fatalf("GetDocumentState returned error: %v", err)
	}
	if state.Status != "pending" {
		t.Fatalf("expected pending state after resume, got %#v", state)
	}
}

func TestTriggerTaskProcessingResetsMatchedTask(t *testing.T) {
	service := newTestService(t)
	message := "model failed"
	seedDocuments(t, service, []DocumentSession{{
		DocumentID: "doc-1",
		Status:     "failed",
		Tasks: []DocumentTask{{
			ID:           "task-1",
			TaskID:       "legacy-task-1",
			DocumentID:   "doc-1",
			SegmentIndex: 1,
			Status:       "Failed",
			ErrorMessage: &message,
		}},
	}})

	if err := service.TriggerTaskProcessing("legacy-task-1"); err != nil {
		t.Fatalf("TriggerTaskProcessing returned error: %v", err)
	}
	tasks, err := service.GetDocumentTasks("doc-1")
	if err != nil {
		t.Fatalf("GetDocumentTasks returned error: %v", err)
	}
	if len(tasks) != 1 {
		t.Fatalf("expected one task, got %d", len(tasks))
	}
	if tasks[0].Status != "Pending" || tasks[0].ErrorMessage != nil {
		t.Fatalf("expected task reset to pending without error, got %#v", tasks[0])
	}
}

func TestDeleteDocumentSessionRemovesStoredDocument(t *testing.T) {
	service := newTestService(t)
	seedDocuments(t, service, []DocumentSession{{
		DocumentID: "doc-1",
		Status:     "pending",
		Tasks: []DocumentTask{{
			ID:         "task-1",
			DocumentID: "doc-1",
			Status:     "Pending",
		}},
	}})

	ok, err := service.DeleteDocumentSession("doc-1")
	if err != nil || !ok {
		t.Fatalf("DeleteDocumentSession returned %v, %v", ok, err)
	}
	tasks, err := service.GetDocumentTasks("doc-1")
	if err != nil {
		t.Fatalf("GetDocumentTasks returned error: %v", err)
	}
	if len(tasks) != 0 {
		t.Fatalf("expected deleted session to have no tasks, got %d", len(tasks))
	}
}

func TestSaveAnkiCardsPersistsCardsAndCompletedTask(t *testing.T) {
	service := newTestService(t)
	cardID := "card-explicit"
	documentID := "doc-save"
	templateID := "template-basic"

	response, err := service.SaveAnkiCards(SaveAnkiCardsRequest{
		DocumentID: &documentID,
		TemplateID: &templateID,
		Cards: []SaveAnkiCardPayload{
			{
				ID:     &cardID,
				Tags:   []string{" biology ", ""},
				Images: []string{"img-1"},
				Fields: map[string]string{
					"Front": "What does chlorophyll capture?",
					"Back":  "Sunlight.",
					"Text":  "Chlorophyll captures sunlight.",
				},
			},
			{
				Front: strPtr("What is produced?"),
				Back:  strPtr("Oxygen."),
				Extra: map[string]string{"Mnemonic": "Leaves release oxygen"},
			},
		},
		Options: map[string]any{"deck_name": "Biology"},
	})
	if err != nil {
		t.Fatalf("SaveAnkiCards returned error: %v", err)
	}
	if response.TaskID == "" || len(response.SavedIDs) != 2 || response.SavedIDs[0] != cardID || response.SavedIDs[1] == "" {
		t.Fatalf("unexpected response: %+v", response)
	}

	tasks, err := service.GetDocumentTasks(documentID)
	if err != nil {
		t.Fatalf("GetDocumentTasks returned error: %v", err)
	}
	if len(tasks) != 1 || tasks[0].ID != response.TaskID || tasks[0].Status != "Completed" || tasks[0].CardsGenerated != 2 {
		t.Fatalf("expected one completed task, got %+v", tasks)
	}
	if !strings.Contains(tasks[0].AnkiGenerationOptionsJSON, "Biology") {
		t.Fatalf("expected options JSON to be persisted, got %s", tasks[0].AnkiGenerationOptionsJSON)
	}

	cards, err := service.GetDocumentCards(documentID)
	if err != nil {
		t.Fatalf("GetDocumentCards returned error: %v", err)
	}
	if len(cards) != 2 {
		t.Fatalf("expected two cards, got %d: %+v", len(cards), cards)
	}
	first := cards[0]
	if first["id"] != cardID || first["front"] != "What does chlorophyll capture?" || first["back"] != "Sunlight." || first["template_id"] != templateID {
		t.Fatalf("unexpected first card: %+v", first)
	}
	if tags, ok := first["tags"].([]string); !ok || len(tags) != 1 || tags[0] != "biology" {
		t.Fatalf("expected cleaned tags, got %#v", first["tags"])
	}
	if fields, ok := first["extra_fields"].(map[string]string); !ok || fields["Front"] == "" || fields["Back"] == "" || fields["Text"] == "" {
		t.Fatalf("expected extra_fields with Front/Back/Text, got %#v", first["extra_fields"])
	}
	second := cards[1]
	if second["id"] == "" || second["task_id"] != response.TaskID || second["front"] != "What is produced?" || second["back"] != "Oxygen." {
		t.Fatalf("unexpected generated-id card: %+v", second)
	}
}

func TestSaveAnkiCardsRejectsEmptyCards(t *testing.T) {
	service := newTestService(t)
	if _, err := service.SaveAnkiCards(SaveAnkiCardsRequest{}); err == nil {
		t.Fatal("expected SaveAnkiCards to reject empty card list")
	}
}

func TestSaveAnkiCardsUsesFallbackDocumentIdentity(t *testing.T) {
	service := newTestService(t)
	blockID := "block-save"

	response, err := service.SaveAnkiCards(SaveAnkiCardsRequest{
		DocumentID: strPtr(" "),
		BlockID:    &blockID,
		Cards: []SaveAnkiCardPayload{{
			Fields: map[string]string{"Front": "Block card", "Back": "Stored by block id"},
		}},
	})
	if err != nil {
		t.Fatalf("SaveAnkiCards returned error: %v", err)
	}
	if response.TaskID == "" || len(response.SavedIDs) != 1 {
		t.Fatalf("unexpected response: %+v", response)
	}

	tasks, err := service.GetDocumentTasks(blockID)
	if err != nil {
		t.Fatalf("GetDocumentTasks returned error: %v", err)
	}
	if len(tasks) != 1 || tasks[0].DocumentID != blockID || tasks[0].ContentSegment != "chat_block:block-save" {
		t.Fatalf("expected block id fallback identity, got %+v", tasks)
	}
}

func TestSaveAnkiCardsUpsertsExistingCardID(t *testing.T) {
	service := newTestService(t)
	documentID := "doc-upsert"
	cardID := "card-same"

	if _, err := service.SaveAnkiCards(SaveAnkiCardsRequest{
		DocumentID: &documentID,
		Cards: []SaveAnkiCardPayload{{
			ID:    &cardID,
			Front: strPtr("Old front"),
			Back:  strPtr("Old back"),
		}},
	}); err != nil {
		t.Fatalf("SaveAnkiCards(first) returned error: %v", err)
	}
	if _, err := service.SaveAnkiCards(SaveAnkiCardsRequest{
		DocumentID: &documentID,
		Cards: []SaveAnkiCardPayload{{
			ID:    &cardID,
			Front: strPtr("New front"),
			Back:  strPtr("New back"),
		}},
	}); err != nil {
		t.Fatalf("SaveAnkiCards(second) returned error: %v", err)
	}
	cards, err := service.GetDocumentCards(documentID)
	if err != nil {
		t.Fatalf("GetDocumentCards returned error: %v", err)
	}
	if len(cards) != 1 || cards[0]["front"] != "New front" || cards[0]["back"] != "New back" {
		t.Fatalf("expected existing card id to be updated, got %+v", cards)
	}
}

func TestRecoverStuckDocumentTasksReturnsProcessingAndStreamingTasksToPending(t *testing.T) {
	service := newTestService(t)
	seedDocuments(t, service, []DocumentSession{{
		DocumentID: "doc-1",
		Status:     "processing",
		Tasks: []DocumentTask{
			{ID: "task-1", DocumentID: "doc-1", Status: "Processing"},
			{ID: "task-2", DocumentID: "doc-1", Status: "Streaming"},
			{ID: "task-3", DocumentID: "doc-1", Status: "Completed"},
		},
	}})

	recovered, err := service.RecoverStuckDocumentTasks()
	if err != nil {
		t.Fatalf("RecoverStuckDocumentTasks returned error: %v", err)
	}
	if recovered != 2 {
		t.Fatalf("expected 2 recovered tasks, got %d", recovered)
	}

	counts, err := service.GetDocumentTaskCounts("doc-1")
	if err != nil {
		t.Fatalf("GetDocumentTaskCounts returned error: %v", err)
	}
	if counts.Pending != 2 || counts.Completed != 1 || counts.Processing != 0 || counts.Streaming != 0 {
		t.Fatalf("unexpected recovered counts: %#v", counts)
	}
}

func TestStartEnhancedDocumentProcessingCreatesTasksCardsAndEvents(t *testing.T) {
	service := newTestService(t)
	events := make(chan map[string]any, 16)
	service.SetEventEmitter(func(name string, payload any) {
		if name != "anki_generation_event" {
			return
		}
		if event, ok := payload.(map[string]any); ok {
			events <- event
		}
	})

	documentID, err := service.StartEnhancedDocumentProcessing(
		"Photosynthesis converts light into stored chemical energy. Chlorophyll captures sunlight in plant leaves. The process releases oxygen as a byproduct.",
		"biology-notes",
		map[string]any{"max_cards_per_mistake": 3, "template_id": "basic"},
	)
	if err != nil {
		t.Fatalf("StartEnhancedDocumentProcessing returned error: %v", err)
	}
	if documentID == "" {
		t.Fatal("expected document ID")
	}

	if !waitForEvent(t, events, "DocumentProcessingCompleted") {
		t.Fatal("expected completion event")
	}

	tasks, err := service.GetDocumentTasks(documentID)
	if err != nil {
		t.Fatalf("GetDocumentTasks returned error: %v", err)
	}
	if len(tasks) == 0 {
		t.Fatal("expected generated tasks")
	}
	if tasks[0].Status != "Completed" || tasks[0].CardsGenerated == 0 {
		t.Fatalf("expected completed task with generated cards, got %#v", tasks[0])
	}

	cards, err := service.GetDocumentCards(documentID)
	if err != nil {
		t.Fatalf("GetDocumentCards returned error: %v", err)
	}
	if len(cards) == 0 {
		t.Fatal("expected generated cards")
	}
	if cards[0]["front"] == "" || cards[0]["back"] == "" || cards[0]["task_id"] != tasks[0].ID {
		t.Fatalf("unexpected generated card: %#v", cards[0])
	}

	state, err := service.GetDocumentState(documentID)
	if err != nil {
		t.Fatalf("GetDocumentState returned error: %v", err)
	}
	if state.Status != "completed" || state.CompletedTasks != len(tasks) {
		t.Fatalf("expected completed document state, got %#v", state)
	}
}

func TestStartEnhancedDocumentProcessingUsesAssignedProvider(t *testing.T) {
	requests := make(chan string, 1)
	provider := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/chat/completions" {
			t.Fatalf("unexpected provider path %s", r.URL.Path)
		}
		if r.Header.Get("Authorization") != "Bearer anki-key" {
			t.Fatalf("unexpected authorization header %q", r.Header.Get("Authorization"))
		}
		body, err := io.ReadAll(r.Body)
		if err != nil {
			t.Fatalf("read provider body: %v", err)
		}
		requests <- string(body)
		w.Header().Set("Content-Type", "application/json")
		content := `{"cards":[{"front":"What does chlorophyll capture?","back":"Chlorophyll captures sunlight for photosynthesis.","tags":["biology","photosynthesis"]}]}`
		_ = json.NewEncoder(w).Encode(map[string]any{
			"choices": []map[string]any{{
				"message": map[string]any{"content": content},
			}},
		})
	}))
	defer provider.Close()

	service := newTestService(t)
	service.SetAPIConfigLoader(func() (APIConfigState, error) {
		return APIConfigState{
			AnkiCardModelConfigID: "api_anki",
			Configs: []ApiConfig{
				{
					ID:         "api_chat",
					ApiKey:     "chat-key",
					BaseUrl:    provider.URL + "/v1",
					Model:      "chat-model",
					Enabled:    true,
					Headers:    map[string]string{},
					IsReranker: false,
				},
				{
					ID:              "api_anki",
					Name:            "Anki model",
					ApiKey:          "anki-key",
					BaseUrl:         provider.URL + "/v1",
					Model:           "anki-model",
					Enabled:         true,
					MaxOutputTokens: 512,
					Temperature:     0.1,
					Headers:         map[string]string{"X-Anki-Test": "yes"},
				},
			},
		}, nil
	})
	events := make(chan map[string]any, 16)
	service.SetEventEmitter(func(name string, payload any) {
		if name != "anki_generation_event" {
			return
		}
		if event, ok := payload.(map[string]any); ok {
			events <- event
		}
	})

	documentID, err := service.StartEnhancedDocumentProcessing(
		"Photosynthesis converts light into chemical energy. Chlorophyll captures sunlight in plant leaves.",
		"biology-provider",
		map[string]any{"max_cards_per_mistake": 2, "template_id": "basic"},
	)
	if err != nil {
		t.Fatalf("StartEnhancedDocumentProcessing returned error: %v", err)
	}
	if !waitForEvent(t, events, "DocumentProcessingCompleted") {
		t.Fatal("expected completion event")
	}

	select {
	case body := <-requests:
		if !strings.Contains(body, `"model":"anki-model"`) || !strings.Contains(body, `"stream":false`) || !strings.Contains(body, `"max_tokens":512`) {
			t.Fatalf("provider request did not use assigned Anki config: %s", body)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("provider was not called")
	}

	cards, err := service.GetDocumentCards(documentID)
	if err != nil {
		t.Fatalf("GetDocumentCards returned error: %v", err)
	}
	if len(cards) != 1 {
		t.Fatalf("expected one provider card, got %d: %#v", len(cards), cards)
	}
	card := cards[0]
	if card["front"] != "What does chlorophyll capture?" || card["source"] != "go_provider_document_worker" || card["provider_config_id"] != "api_anki" || card["template_id"] != "basic" {
		t.Fatalf("unexpected provider card: %#v", card)
	}
	tags, ok := card["tags"].([]string)
	if !ok || !containsString(tags, "photosynthesis") || !containsString(tags, "provider") {
		t.Fatalf("provider tags were not preserved: %#v", card["tags"])
	}
}

func TestStartEnhancedDocumentProcessingValidatesContent(t *testing.T) {
	service := newTestService(t)

	if _, err := service.StartEnhancedDocumentProcessing("  ", "empty", nil); !errors.Is(err, errDocumentContentRequired) {
		t.Fatalf("expected content required error, got %v", err)
	}
}

func TestDocumentIDIsRequiredForMutatingSessionControls(t *testing.T) {
	service := newTestService(t)

	if _, err := service.PauseDocumentProcessing(" "); !errors.Is(err, errDocumentIDRequired) {
		t.Fatalf("expected document ID required error for pause, got %v", err)
	}
	if _, err := service.DeleteDocumentSession(""); !errors.Is(err, errDocumentIDRequired) {
		t.Fatalf("expected document ID required error for delete, got %v", err)
	}
}

func waitForEvent(t *testing.T, events <-chan map[string]any, key string) bool {
	t.Helper()
	deadline := time.After(2 * time.Second)
	for {
		select {
		case event := <-events:
			if _, ok := event[key]; ok {
				return true
			}
		case <-deadline:
			return false
		}
	}
}

func newTestService(t *testing.T) *Service {
	t.Helper()
	service, err := NewService(t.TempDir())
	if err != nil {
		t.Fatalf("NewService returned error: %v", err)
	}
	return service
}

func seedDocuments(t *testing.T, service *Service, documents []DocumentSession) {
	t.Helper()
	service.mu.Lock()
	defer service.mu.Unlock()
	service.state.Documents = documents
	if err := service.flushLocked(); err != nil {
		t.Fatalf("flushLocked returned error: %v", err)
	}
}

func containsString(values []string, expected string) bool {
	for _, value := range values {
		if value == expected {
			return true
		}
	}
	return false
}

func strPtr(value string) *string {
	return &value
}
