package qbank

import (
	"bytes"
	"deep-student-go/internal/vfs"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
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

func TestCreateListFilterAndPagination(t *testing.T) {
	service := newTestService(t)
	examID := "exam_1"
	if _, err := service.CreateQuestion(CreateQuestionParams{
		ExamID:       examID,
		Content:      "Derivative",
		Answer:       strPtr("A"),
		Tags:         []string{"math"},
		Difficulty:   strPtr("easy"),
		QuestionType: strPtr("single_choice"),
	}); err != nil {
		t.Fatalf("CreateQuestion(first) error = %v", err)
	}
	if _, err := service.CreateQuestion(CreateQuestionParams{
		ExamID:  examID,
		Content: "Grammar",
		Answer:  strPtr("B"),
		Tags:    []string{"english"},
	}); err != nil {
		t.Fatalf("CreateQuestion(second) error = %v", err)
	}

	result, err := service.ListQuestions(ListQuestionsRequest{
		ExamID:   examID,
		Filters:  &QuestionFilters{Tags: []string{"math"}},
		Page:     1,
		PageSize: 1,
	})
	if err != nil {
		t.Fatalf("ListQuestions() error = %v", err)
	}
	if result.Total != 1 || len(result.Questions) != 1 || result.Questions[0].Content != "Derivative" {
		t.Fatalf("unexpected list result: %+v", result)
	}
}

func TestNotesMentionsSearchUsesQbankQuestions(t *testing.T) {
	service := newTestService(t)
	cardID := "card_physics_1"
	if _, err := service.CreateQuestion(CreateQuestionParams{
		ExamID:        "exam_physics",
		CardID:        &cardID,
		QuestionLabel: strPtr("Newton inertia card"),
		Content:       "Explain Newton's first law.",
		Answer:        strPtr("Objects keep moving unless an external force acts."),
		Explanation:   strPtr("Inertia means motion does not change without a net external force."),
		Tags:          []string{"physics", "inertia"},
	}); err != nil {
		t.Fatalf("CreateQuestion(physics) error = %v", err)
	}
	if _, err := service.CreateQuestion(CreateQuestionParams{
		ExamID:      "exam_math",
		Content:     "Use inertia-like reasoning as a distractor in this word problem.",
		Explanation: strPtr("This should be hidden by the subject filter."),
		Tags:        []string{"math"},
	}); err != nil {
		t.Fatalf("CreateQuestion(math) error = %v", err)
	}
	if _, err := service.CreateQuestion(CreateQuestionParams{
		ExamID:      "exam_physics",
		Content:     "Describe conservation of momentum.",
		Explanation: strPtr("No matching keyword here."),
		Tags:        []string{"physics"},
	}); err != nil {
		t.Fatalf("CreateQuestion(other) error = %v", err)
	}

	all, err := service.NotesMentionsSearch("inertia", nil, 8)
	if err != nil {
		t.Fatalf("NotesMentionsSearch(all) error = %v", err)
	}
	if len(all.IrecCards) != 2 || len(all.Mistakes) != 2 {
		t.Fatalf("expected two all-scope mention hits, got %+v", all)
	}
	if all.IrecCards[0].ID != cardID || all.IrecCards[0].MistakeID == nil || *all.IrecCards[0].MistakeID == "" {
		t.Fatalf("expected card id plus backing question id, got %+v", all.IrecCards[0])
	}
	if !strings.Contains(strings.ToLower(all.IrecCards[0].Insight), "inertia") {
		t.Fatalf("expected matching insight snippet, got %+v", all.IrecCards[0])
	}

	filtered, err := service.NotesMentionsSearch("inertia", strPtr("exam_physics"), 8)
	if err != nil {
		t.Fatalf("NotesMentionsSearch(filtered) error = %v", err)
	}
	if len(filtered.IrecCards) != 1 || filtered.IrecCards[0].ID != cardID {
		t.Fatalf("expected subject-filtered physics card, got %+v", filtered)
	}

	blank, err := service.NotesMentionsSearch("  ", nil, 8)
	if err != nil {
		t.Fatalf("NotesMentionsSearch(blank) error = %v", err)
	}
	if len(blank.IrecCards) != 0 || len(blank.Mistakes) != 0 {
		t.Fatalf("expected blank search to return empty result, got %+v", blank)
	}
}

func TestSubmitAnswerUpdatesQuestionAndStats(t *testing.T) {
	service := newTestService(t)
	question, err := service.CreateQuestion(CreateQuestionParams{
		ExamID:  "exam_1",
		Content: "1+1",
		Answer:  strPtr("2"),
	})
	if err != nil {
		t.Fatalf("CreateQuestion() error = %v", err)
	}

	result, err := service.SubmitAnswer(SubmitAnswerRequest{
		QuestionID: question.ID,
		UserAnswer: " 2 ",
	})
	if err != nil {
		t.Fatalf("SubmitAnswer() error = %v", err)
	}
	if result.IsCorrect == nil || !*result.IsCorrect || result.UpdatedQuestion.AttemptCount != 1 {
		t.Fatalf("unexpected submit result: %+v", result)
	}
	if result.UpdatedStats.TotalAttempts != 1 || result.UpdatedStats.TotalCorrect != 1 {
		t.Fatalf("unexpected stats: %+v", result.UpdatedStats)
	}
	history, err := service.GetHistory(question.ID, 10)
	if err != nil {
		t.Fatalf("GetHistory() error = %v", err)
	}
	if len(history) == 0 {
		t.Fatalf("expected answer history")
	}
}

func TestAIGradeUpdatesQuestionAndEmitsEvents(t *testing.T) {
	dir := t.TempDir()
	vfsService, err := vfs.NewService(dir)
	if err != nil {
		t.Fatalf("vfs.NewService() error = %v", err)
	}
	service, err := NewService(dir, vfsService)
	if err != nil {
		t.Fatalf("NewService() error = %v", err)
	}
	var events []string
	var payloads []any
	service.SetEventEmitter(func(name string, payload any) {
		events = append(events, name)
		payloads = append(payloads, payload)
	})

	question, err := service.CreateQuestion(CreateQuestionParams{
		ExamID:       "exam_1",
		Content:      "Explain Newton's first law.",
		Answer:       strPtr("An object remains at rest or in uniform motion unless acted on by an external force."),
		Explanation:  strPtr("This is the law of inertia."),
		QuestionType: strPtr("short_answer"),
		Tags:         []string{"physics"},
	})
	if err != nil {
		t.Fatalf("CreateQuestion() error = %v", err)
	}
	submitted, err := service.SubmitAnswer(SubmitAnswerRequest{
		QuestionID: question.ID,
		UserAnswer: "Objects keep uniform motion unless an external force acts.",
	})
	if err != nil {
		t.Fatalf("SubmitAnswer() error = %v", err)
	}
	if !submitted.NeedsManualGrading || submitted.IsCorrect != nil {
		t.Fatalf("expected manual grading before AI result: %+v", submitted)
	}

	response, err := service.AIGrade(QbankGradingRequest{
		QuestionID:      question.ID,
		SubmissionID:    submitted.SubmissionID,
		StreamSessionID: "session_1",
		Mode:            "grade",
	})
	if err != nil {
		t.Fatalf("AIGrade() error = %v", err)
	}
	if response == nil || response.Verdict == nil || response.Score == nil || response.Feedback == "" {
		t.Fatalf("unexpected AI grade response: %+v", response)
	}
	if len(events) != 2 || events[0] != "qbank_grading_stream_session_1" || events[1] != "qbank_grading_stream_session_1" {
		t.Fatalf("unexpected events: %#v", events)
	}
	if payload, ok := payloads[1].(map[string]any); !ok || payload["type"] != "complete" || payload["submission_id"] != submitted.SubmissionID {
		t.Fatalf("unexpected complete payload: %#v", payloads[1])
	}

	updated, err := service.GetQuestion(question.ID)
	if err != nil {
		t.Fatalf("GetQuestion() error = %v", err)
	}
	if updated == nil || updated.AIFeedback == nil || !strings.Contains(*updated.AIFeedback, "AI Grading") || updated.AIScore == nil || updated.AIGradedAt == nil {
		t.Fatalf("expected persisted AI feedback fields, got %+v", updated)
	}
	if updated.IsCorrect == nil {
		t.Fatalf("expected AI grading to update correctness: %+v", updated)
	}
	submissions, err := service.GetSubmissions(question.ID, 5)
	if err != nil {
		t.Fatalf("GetSubmissions() error = %v", err)
	}
	if len(submissions) != 1 || submissions[0].IsCorrect == nil {
		t.Fatalf("expected graded submission, got %+v", submissions)
	}
	resolved, err := vfsService.ResolveResourceRefs([]vfs.ResourceRef{{
		SourceID:     question.ID,
		ResourceHash: updated.ResourceHash,
		Type:         "exam",
		Name:         "question",
	}})
	if err != nil {
		t.Fatalf("ResolveResourceRefs() error = %v", err)
	}
	if len(resolved) != 1 || resolved[0].Content == nil || !strings.Contains(*resolved[0].Content, "AI Feedback") {
		t.Fatalf("expected VFS AI feedback sync, got %+v", resolved)
	}
}

func TestAIGradeUsesAssignedProvider(t *testing.T) {
	requests := make(chan string, 1)
	provider := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/chat/completions" {
			t.Fatalf("unexpected provider path %s", r.URL.Path)
		}
		if r.Header.Get("Authorization") != "Bearer qbank-key" {
			t.Fatalf("unexpected authorization header %q", r.Header.Get("Authorization"))
		}
		body, err := io.ReadAll(r.Body)
		if err != nil {
			t.Fatalf("read provider body: %v", err)
		}
		requests <- string(body)
		w.Header().Set("Content-Type", "application/json")
		content := `{"verdict":"correct","score":94,"feedback":"### Provider Feedback\n\nThe answer captures inertia and the external force condition."}`
		_ = json.NewEncoder(w).Encode(map[string]any{
			"choices": []map[string]any{{
				"message": map[string]any{"content": content},
			}},
		})
	}))
	defer provider.Close()

	dir := t.TempDir()
	vfsService, err := vfs.NewService(dir)
	if err != nil {
		t.Fatalf("vfs.NewService() error = %v", err)
	}
	service, err := NewService(dir, vfsService)
	if err != nil {
		t.Fatalf("NewService() error = %v", err)
	}
	service.SetAPIConfigLoader(func() (APIConfigState, error) {
		return APIConfigState{
			QbankAIGradingModelConfigID: "api_qbank",
			Configs: []ApiConfig{
				{
					ID:      "api_chat",
					ApiKey:  "chat-key",
					BaseUrl: provider.URL + "/v1",
					Model:   "chat-model",
					Enabled: true,
				},
				{
					ID:              "api_qbank",
					Name:            "Qbank grader",
					ApiKey:          "qbank-key",
					BaseUrl:         provider.URL + "/v1",
					Model:           "qbank-model",
					Enabled:         true,
					MaxOutputTokens: 777,
					Temperature:     0.15,
				},
			},
		}, nil
	})
	var payloads []any
	service.SetEventEmitter(func(name string, payload any) {
		if name == "qbank_grading_stream_session_provider" {
			payloads = append(payloads, payload)
		}
	})

	question, err := service.CreateQuestion(CreateQuestionParams{
		ExamID:       "exam_provider",
		Content:      "Explain Newton's first law.",
		Answer:       strPtr("An object remains at rest or in uniform motion unless acted on by an external force."),
		Explanation:  strPtr("The law of inertia describes unchanged motion without net external force."),
		QuestionType: strPtr("short_answer"),
		Tags:         []string{"physics"},
	})
	if err != nil {
		t.Fatalf("CreateQuestion() error = %v", err)
	}
	submitted, err := service.SubmitAnswer(SubmitAnswerRequest{
		QuestionID: question.ID,
		UserAnswer: "Objects keep moving the same way until a force acts.",
	})
	if err != nil {
		t.Fatalf("SubmitAnswer() error = %v", err)
	}

	response, err := service.AIGrade(QbankGradingRequest{
		QuestionID:      question.ID,
		SubmissionID:    submitted.SubmissionID,
		StreamSessionID: "session_provider",
		Mode:            "grade",
	})
	if err != nil {
		t.Fatalf("AIGrade() error = %v", err)
	}
	if response == nil || response.Verdict == nil || *response.Verdict != "correct" || response.Score == nil || *response.Score != 94 || !strings.Contains(response.Feedback, "Provider Feedback") {
		t.Fatalf("unexpected provider grading response: %+v", response)
	}
	select {
	case body := <-requests:
		if !strings.Contains(body, `"model":"qbank-model"`) || !strings.Contains(body, `"stream":false`) || !strings.Contains(body, `"max_tokens":777`) {
			t.Fatalf("provider request did not use assigned qbank config: %s", body)
		}
		if !strings.Contains(body, "Newton") || !strings.Contains(body, "Objects keep moving") {
			t.Fatalf("provider request missing grading context: %s", body)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("provider was not called")
	}
	if len(payloads) != 2 {
		t.Fatalf("expected data and complete events, got %#v", payloads)
	}
	complete, ok := payloads[1].(map[string]any)
	if !ok || complete["type"] != "complete" || complete["submission_id"] != submitted.SubmissionID {
		t.Fatalf("unexpected complete payload: %#v", payloads[1])
	}

	updated, err := service.GetQuestion(question.ID)
	if err != nil {
		t.Fatalf("GetQuestion() error = %v", err)
	}
	if updated == nil || updated.AIFeedback == nil || !strings.Contains(*updated.AIFeedback, "Provider Feedback") || updated.AIScore == nil || *updated.AIScore != 94 {
		t.Fatalf("expected provider feedback fields, got %+v", updated)
	}
	if updated.IsCorrect == nil || !*updated.IsCorrect {
		t.Fatalf("expected provider verdict to mark question correct: %+v", updated)
	}
	submissions, err := service.GetSubmissions(question.ID, 5)
	if err != nil {
		t.Fatalf("GetSubmissions() error = %v", err)
	}
	if len(submissions) != 1 || submissions[0].IsCorrect == nil || !*submissions[0].IsCorrect {
		t.Fatalf("expected provider-graded submission, got %+v", submissions)
	}
	resolved, err := vfsService.ResolveResourceRefs([]vfs.ResourceRef{{
		SourceID:     question.ID,
		ResourceHash: updated.ResourceHash,
		Type:         "exam",
		Name:         "question",
	}})
	if err != nil {
		t.Fatalf("ResolveResourceRefs() error = %v", err)
	}
	if len(resolved) != 1 || resolved[0].Content == nil || !strings.Contains(*resolved[0].Content, "Provider Feedback") {
		t.Fatalf("expected provider feedback in VFS resource, got %+v", resolved)
	}
}

func TestSyncStatusLocalCompatibility(t *testing.T) {
	dir := t.TempDir()
	service, err := NewService(dir)
	if err != nil {
		t.Fatalf("NewService() error = %v", err)
	}
	if _, err := service.CreateQuestion(CreateQuestionParams{
		ExamID:  "exam_sync",
		Content: "Sync placeholder",
		Answer:  strPtr("A"),
	}); err != nil {
		t.Fatalf("CreateQuestion() error = %v", err)
	}

	status, err := service.CheckSyncStatus("exam_sync")
	if err != nil {
		t.Fatalf("CheckSyncStatus() error = %v", err)
	}
	if status.SyncEnabled || status.TotalCount != 1 || status.SyncedCount != 1 || status.PendingConflictCount != 0 || status.SyncConfig == nil {
		t.Fatalf("unexpected initial sync status: %+v", status)
	}
	conflicts, err := service.GetSyncConflicts("exam_sync")
	if err != nil {
		t.Fatalf("GetSyncConflicts() error = %v", err)
	}
	if len(conflicts) != 0 {
		t.Fatalf("expected no local Go sync conflicts, got %+v", conflicts)
	}
	if resolved, err := service.BatchResolveConflicts("exam_sync", "keep_local"); err != nil || len(resolved) != 0 {
		t.Fatalf("BatchResolveConflicts() = %+v, %v", resolved, err)
	}
	if err := service.SetSyncEnabled("exam_sync", true); err != nil {
		t.Fatalf("SetSyncEnabled() error = %v", err)
	}
	if err := service.UpdateSyncConfig("exam_sync", SyncConfig{
		DefaultStrategy:  "keep_local",
		AutoSync:         true,
		SyncIntervalSecs: 900,
		SyncProgress:     false,
		SyncNotes:        false,
	}); err != nil {
		t.Fatalf("UpdateSyncConfig() error = %v", err)
	}

	reloaded, err := NewService(dir)
	if err != nil {
		t.Fatalf("NewService(reload) error = %v", err)
	}
	reloadedStatus, err := reloaded.CheckSyncStatus("exam_sync")
	if err != nil {
		t.Fatalf("CheckSyncStatus(reload) error = %v", err)
	}
	if !reloadedStatus.SyncEnabled || reloadedStatus.SyncConfig == nil || !reloadedStatus.SyncConfig.AutoSync {
		t.Fatalf("expected persisted sync enabled config, got %+v", reloadedStatus)
	}
	if reloadedStatus.SyncConfig.DefaultStrategy != "keep_local" ||
		reloadedStatus.SyncConfig.SyncIntervalSecs != 900 ||
		reloadedStatus.SyncConfig.SyncProgress ||
		reloadedStatus.SyncConfig.SyncNotes {
		t.Fatalf("expected persisted custom sync config, got %+v", reloadedStatus.SyncConfig)
	}
}

func TestUpdateSyncConfigPersistsConfigByExamID(t *testing.T) {
	dir := t.TempDir()
	service, err := NewService(dir)
	if err != nil {
		t.Fatalf("NewService() error = %v", err)
	}
	if err := service.UpdateSyncConfig(" exam_sync ", SyncConfig{
		DefaultStrategy:  "keep_local",
		AutoSync:         true,
		SyncIntervalSecs: 900,
		SyncProgress:     false,
		SyncNotes:        false,
	}); err != nil {
		t.Fatalf("UpdateSyncConfig() error = %v", err)
	}

	reloaded, err := NewService(dir)
	if err != nil {
		t.Fatalf("NewService(reload) error = %v", err)
	}
	status, err := reloaded.CheckSyncStatus("exam_sync")
	if err != nil {
		t.Fatalf("CheckSyncStatus() error = %v", err)
	}
	if status.SyncConfig == nil {
		t.Fatalf("expected sync config, got %+v", status)
	}
	if !status.SyncEnabled ||
		status.SyncConfig.DefaultStrategy != "keep_local" ||
		status.SyncConfig.SyncIntervalSecs != 900 ||
		status.SyncConfig.SyncProgress ||
		status.SyncConfig.SyncNotes {
		t.Fatalf("expected persisted custom sync config, got %+v", status.SyncConfig)
	}
}

func TestUpdateSyncConfigNormalizesEmptyStrategyAndInvalidInterval(t *testing.T) {
	service := newTestService(t)
	if err := service.UpdateSyncConfig("exam_sync", SyncConfig{
		AutoSync:         true,
		SyncIntervalSecs: -10,
		SyncProgress:     false,
		SyncNotes:        false,
	}); err != nil {
		t.Fatalf("UpdateSyncConfig() error = %v", err)
	}

	status, err := service.CheckSyncStatus("exam_sync")
	if err != nil {
		t.Fatalf("CheckSyncStatus() error = %v", err)
	}
	if status.SyncConfig == nil {
		t.Fatalf("expected sync config, got %+v", status)
	}
	if status.SyncConfig.DefaultStrategy != "keep_newer" || status.SyncConfig.SyncIntervalSecs != 300 {
		t.Fatalf("expected normalized defaults, got %+v", status.SyncConfig)
	}
}

func TestUpdateSyncConfigRejectsBlankExamID(t *testing.T) {
	service := newTestService(t)
	if err := service.UpdateSyncConfig(" \t ", SyncConfig{}); err == nil {
		t.Fatal("expected UpdateSyncConfig to reject blank exam id")
	}
}

func TestSetSyncEnabledPreservesExistingSyncConfigFields(t *testing.T) {
	service := newTestService(t)
	if err := service.UpdateSyncConfig("exam_sync", SyncConfig{
		DefaultStrategy:  "keep_local",
		AutoSync:         true,
		SyncIntervalSecs: 900,
		SyncProgress:     false,
		SyncNotes:        false,
	}); err != nil {
		t.Fatalf("UpdateSyncConfig() error = %v", err)
	}
	if err := service.SetSyncEnabled("exam_sync", false); err != nil {
		t.Fatalf("SetSyncEnabled(false) error = %v", err)
	}

	status, err := service.CheckSyncStatus("exam_sync")
	if err != nil {
		t.Fatalf("CheckSyncStatus() error = %v", err)
	}
	if status.SyncConfig == nil {
		t.Fatalf("expected sync config, got %+v", status)
	}
	if status.SyncEnabled ||
		status.SyncConfig.DefaultStrategy != "keep_local" ||
		status.SyncConfig.SyncIntervalSecs != 900 ||
		status.SyncConfig.SyncProgress ||
		status.SyncConfig.SyncNotes {
		t.Fatalf("expected SetSyncEnabled to preserve config fields, got %+v", status.SyncConfig)
	}
}

func TestCreateAndUpdateSyncsQuestionIntoHybridVfs(t *testing.T) {
	dir := t.TempDir()
	vfsService, err := vfs.NewService(dir)
	if err != nil {
		t.Fatalf("vfs.NewService() error = %v", err)
	}
	service, err := NewService(dir, vfsService)
	if err != nil {
		t.Fatalf("NewService() error = %v", err)
	}

	label := "Limits practice"
	question, err := service.CreateQuestion(CreateQuestionParams{
		ExamID:        "exam_1",
		Content:       "What is lim x->0 sin(x)/x?",
		QuestionLabel: &label,
		QuestionType:  strPtr("single_choice"),
		Options: []QuestionOption{
			{Key: "A", Content: "0"},
			{Key: "B", Content: "1"},
		},
		Answer:      strPtr("B"),
		Explanation: strPtr("Standard trigonometric limit."),
		Images: []QuestionImage{
			{ID: "img_1", Name: "limit.png", Mime: "image/png", Hash: "hash_1"},
		},
	})
	if err != nil {
		t.Fatalf("CreateQuestion() error = %v", err)
	}
	if question.ResourceID == "" || question.ResourceID == question.ID || question.ResourceHash == "" {
		t.Fatalf("expected VFS-backed question, got %+v", question)
	}

	refData, err := vfsService.GetResourceRefs(vfs.GetResourceRefsInput{SourceIDs: []string{question.ID}})
	if err != nil {
		t.Fatalf("GetResourceRefs() error = %v", err)
	}
	if len(refData.Refs) != 1 || refData.Refs[0].ResourceHash != question.ResourceHash || refData.Refs[0].ResourceID == nil || *refData.Refs[0].ResourceID != question.ResourceID {
		t.Fatalf("unexpected VFS refs: question=%+v refs=%+v", question, refData)
	}
	resolved, err := vfsService.ResolveResourceRefs(refData.Refs)
	if err != nil {
		t.Fatalf("ResolveResourceRefs() error = %v", err)
	}
	if len(resolved) != 1 || !resolved[0].Found || resolved[0].Content == nil || !strings.Contains(*resolved[0].Content, "Standard trigonometric limit.") {
		t.Fatalf("unexpected resolved question: %+v", resolved)
	}

	updatedContent := "Updated limit question"
	updatedExplanation := "Use the squeeze theorem."
	updated, err := service.UpdateQuestion(UpdateQuestionRequest{
		QuestionID: question.ID,
		Params: UpdateQuestionParams{
			Content:     &updatedContent,
			Explanation: &updatedExplanation,
		},
	})
	if err != nil {
		t.Fatalf("UpdateQuestion() error = %v", err)
	}
	if updated.ResourceID != question.ResourceID || updated.ResourceHash == question.ResourceHash {
		t.Fatalf("expected stable resource id and changed hash: before=%+v after=%+v", question, updated)
	}

	updatedRefs, err := vfsService.GetResourceRefs(vfs.GetResourceRefsInput{SourceIDs: []string{question.ID}})
	if err != nil {
		t.Fatalf("GetResourceRefs(updated) error = %v", err)
	}
	if len(updatedRefs.Refs) != 1 || updatedRefs.Refs[0].ResourceHash != updated.ResourceHash {
		t.Fatalf("unexpected updated refs: question=%+v refs=%+v", updated, updatedRefs)
	}
	updatedResolved, err := vfsService.ResolveResourceRefs(updatedRefs.Refs)
	if err != nil {
		t.Fatalf("ResolveResourceRefs(updated) error = %v", err)
	}
	if len(updatedResolved) != 1 || updatedResolved[0].Content == nil || !strings.Contains(*updatedResolved[0].Content, updatedExplanation) {
		t.Fatalf("unexpected updated resolved question: %+v", updatedResolved)
	}

	submitResult, err := service.SubmitAnswer(SubmitAnswerRequest{QuestionID: question.ID, UserAnswer: "B"})
	if err != nil {
		t.Fatalf("SubmitAnswer() error = %v", err)
	}
	afterSubmitRefs, err := vfsService.GetResourceRefs(vfs.GetResourceRefsInput{SourceIDs: []string{question.ID}})
	if err != nil {
		t.Fatalf("GetResourceRefs(after submit) error = %v", err)
	}
	if len(afterSubmitRefs.Refs) != 1 || afterSubmitRefs.Refs[0].ResourceHash != submitResult.UpdatedQuestion.ResourceHash {
		t.Fatalf("expected submit to sync VFS hash: result=%+v refs=%+v", submitResult.UpdatedQuestion, afterSubmitRefs)
	}
}

func TestLearningLoopStatsPracticePaperCalendarAndHistory(t *testing.T) {
	service := newTestService(t)
	examID := "exam_loop"
	qReview, err := service.CreateQuestion(CreateQuestionParams{
		ExamID:       examID,
		Content:      "Review algebra",
		Answer:       strPtr("A"),
		QuestionType: strPtr("single_choice"),
		Difficulty:   strPtr("easy"),
		Tags:         []string{"algebra", "basics"},
	})
	if err != nil {
		t.Fatalf("CreateQuestion(review) error = %v", err)
	}
	qNew, err := service.CreateQuestion(CreateQuestionParams{
		ExamID:       examID,
		Content:      "New geometry",
		Answer:       strPtr("B"),
		QuestionType: strPtr("single_choice"),
		Difficulty:   strPtr("medium"),
		Tags:         []string{"geometry"},
	})
	if err != nil {
		t.Fatalf("CreateQuestion(new) error = %v", err)
	}
	qPaper, err := service.CreateQuestion(CreateQuestionParams{
		ExamID:       examID,
		Content:      "Essay proof",
		Answer:       strPtr("C"),
		Explanation:  strPtr("Proof explanation"),
		QuestionType: strPtr("essay"),
		Difficulty:   strPtr("hard"),
		Tags:         []string{"algebra"},
	})
	if err != nil {
		t.Fatalf("CreateQuestion(paper) error = %v", err)
	}
	if _, err := service.SubmitAnswer(SubmitAnswerRequest{QuestionID: qReview.ID, UserAnswer: "wrong"}); err != nil {
		t.Fatalf("SubmitAnswer(review wrong) error = %v", err)
	}
	if _, err := service.SubmitAnswer(SubmitAnswerRequest{QuestionID: qPaper.ID, UserAnswer: "draft", IsCorrectOverride: boolPtr(true)}); err != nil {
		t.Fatalf("SubmitAnswer(paper correct) error = %v", err)
	}

	today := time.Now().UTC().Format("2006-01-02")
	yesterday := time.Now().UTC().AddDate(0, 0, -1).Format("2006-01-02")
	trend, err := service.GetLearningTrend(GetLearningTrendRequest{
		ExamID:    &examID,
		StartDate: yesterday,
		EndDate:   today,
	})
	if err != nil {
		t.Fatalf("GetLearningTrend() error = %v", err)
	}
	if len(trend) != 2 || trend[0].AttemptCount != 0 || trend[1].AttemptCount != 2 || trend[1].CorrectCount != 1 || trend[1].CorrectRate != 50 {
		t.Fatalf("unexpected trend: %+v", trend)
	}

	heatmap, err := service.GetActivityHeatmap(GetActivityHeatmapRequest{ExamID: &examID, Year: time.Now().UTC().Year()})
	if err != nil {
		t.Fatalf("GetActivityHeatmap() error = %v", err)
	}
	if len(heatmap) != 1 || heatmap[0].Date != today || heatmap[0].Count != 2 || heatmap[0].Level != 1 {
		t.Fatalf("unexpected heatmap: %+v", heatmap)
	}

	knowledge, err := service.GetKnowledgeStats(GetKnowledgeStatsRequest{ExamID: &examID})
	if err != nil {
		t.Fatalf("GetKnowledgeStats() error = %v", err)
	}
	if len(knowledge) < 2 || knowledge[0].Tag != "algebra" || knowledge[0].Total != 2 || knowledge[0].CorrectRate != 50 {
		t.Fatalf("unexpected knowledge stats: %+v", knowledge)
	}
	comparison, err := service.GetKnowledgeStatsWithComparison(GetKnowledgeStatsRequest{ExamID: &examID})
	if err != nil {
		t.Fatalf("GetKnowledgeStatsWithComparison() error = %v", err)
	}
	if len(comparison.Current) != len(knowledge) || comparison.Previous == nil {
		t.Fatalf("unexpected comparison: %+v", comparison)
	}

	timed, err := service.StartTimedPractice(StartTimedPracticeRequest{ExamID: examID, DurationMinutes: 15, QuestionCount: 2})
	if err != nil {
		t.Fatalf("StartTimedPractice() error = %v", err)
	}
	if timed.QuestionCount != 2 || len(timed.QuestionIDs) != 2 || timed.DurationMinutes != 15 {
		t.Fatalf("unexpected timed session: %+v", timed)
	}

	daily, err := service.GetDailyPractice(GetDailyPracticeRequest{ExamID: examID, Count: 3})
	if err != nil {
		t.Fatalf("GetDailyPractice() error = %v", err)
	}
	if len(daily.QuestionIDs) != 3 || daily.SourceDistribution.MistakeCount != 1 || daily.SourceDistribution.NewCount == 0 || daily.CompletedCount != 2 || daily.CorrectCount != 1 {
		t.Fatalf("unexpected daily practice: %+v", daily)
	}

	paper, err := service.GeneratePaper(GeneratePaperRequest{
		ExamID: examID,
		Config: PaperConfig{
			Title:               "Lean paper",
			TypeSelection:       map[string]int{"essay": 1},
			TagsFilter:          []string{"algebra"},
			IncludeAnswers:      false,
			IncludeExplanations: false,
		},
	})
	if err != nil {
		t.Fatalf("GeneratePaper() error = %v", err)
	}
	if len(paper.Questions) != 1 || paper.Questions[0].ID != qPaper.ID || paper.Questions[0].Answer != "" || paper.Questions[0].Explanation != "" {
		t.Fatalf("unexpected paper: %+v", paper)
	}

	calendar, err := service.GetCheckInCalendar(GetCheckInCalendarRequest{ExamID: &examID, Year: time.Now().UTC().Year(), Month: int(time.Now().UTC().Month())})
	if err != nil {
		t.Fatalf("GetCheckInCalendar() error = %v", err)
	}
	if calendar.MonthCheckInDays != 1 || calendar.MonthTotalQuestions != 2 || len(calendar.Days) != 1 || calendar.Days[0].CorrectCount != 1 {
		t.Fatalf("unexpected calendar: %+v", calendar)
	}

	updatedContent := "Review algebra updated"
	if _, err := service.UpdateQuestion(UpdateQuestionRequest{
		QuestionID:    qReview.ID,
		Params:        UpdateQuestionParams{Content: &updatedContent},
		RecordHistory: true,
	}); err != nil {
		t.Fatalf("UpdateQuestion(record history) error = %v", err)
	}
	history, err := service.GetHistory(qReview.ID, 20)
	if err != nil {
		t.Fatalf("GetHistory() error = %v", err)
	}
	foundContentHistory := false
	foundAnswerHistory := false
	for _, item := range history {
		if item.FieldName == "content" {
			foundContentHistory = true
		}
		if item.FieldName == "user_answer" {
			foundAnswerHistory = true
		}
	}
	if !foundContentHistory || !foundAnswerHistory {
		t.Fatalf("expected content and answer history, got %+v", history)
	}

	submissions, err := service.GetSubmissions(qReview.ID, 5)
	if err != nil {
		t.Fatalf("GetSubmissions() error = %v", err)
	}
	if len(submissions) != 1 || submissions[0].UserAnswer != "wrong" {
		t.Fatalf("unexpected submissions: %+v", submissions)
	}

	_ = qNew
}

func TestMockExamGenerationAndScoring(t *testing.T) {
	service := newTestService(t)
	examID := "exam_mock"
	first, err := service.CreateQuestion(CreateQuestionParams{
		ExamID:       examID,
		Content:      "Single",
		Answer:       strPtr("A"),
		QuestionType: strPtr("single_choice"),
		Difficulty:   strPtr("easy"),
	})
	if err != nil {
		t.Fatalf("CreateQuestion(first) error = %v", err)
	}
	second, err := service.CreateQuestion(CreateQuestionParams{
		ExamID:       examID,
		Content:      "Fill",
		Answer:       strPtr("B"),
		QuestionType: strPtr("fill_blank"),
		Difficulty:   strPtr("medium"),
	})
	if err != nil {
		t.Fatalf("CreateQuestion(second) error = %v", err)
	}
	total := 2
	session, err := service.GenerateMockExam(GenerateMockExamRequest{
		ExamID: examID,
		Config: MockExamConfig{
			DurationMinutes:        30,
			TypeDistribution:       map[string]int{"single_choice": 1},
			DifficultyDistribution: map[string]int{"medium": 1},
			TotalCount:             &total,
			Shuffle:                false,
			IncludeMistakes:        true,
		},
	})
	if err != nil {
		t.Fatalf("GenerateMockExam() error = %v", err)
	}
	if len(session.QuestionIDs) != 2 {
		t.Fatalf("unexpected mock session: %+v", session)
	}
	ended := time.Now().UTC().Add(time.Minute).Format(time.RFC3339)
	score, err := service.SubmitMockExam(SubmitMockExamRequest{Session: MockExamSession{
		ID:          session.ID,
		ExamID:      examID,
		Config:      session.Config,
		QuestionIDs: []string{first.ID, second.ID},
		StartedAt:   time.Now().UTC().Format(time.RFC3339),
		EndedAt:     &ended,
		Answers:     map[string]string{first.ID: "A", second.ID: "wrong"},
		Results:     map[string]bool{first.ID: true, second.ID: false},
		IsSubmitted: true,
	}})
	if err != nil {
		t.Fatalf("SubmitMockExam() error = %v", err)
	}
	if score.TotalCount != 2 || score.AnsweredCount != 2 || score.CorrectCount != 1 || score.CorrectRate != 50 || len(score.WrongQuestionIDs) != 1 {
		t.Fatalf("unexpected score: %+v", score)
	}
	if score.TypeStats["single_choice"].Correct != 1 || score.DifficultyStats["medium"].Total != 1 {
		t.Fatalf("unexpected score breakdown: %+v", score)
	}
}

func TestUpdateFavoriteDeleteResetAndPersistence(t *testing.T) {
	dir := t.TempDir()
	service, err := NewService(dir)
	if err != nil {
		t.Fatalf("NewService() error = %v", err)
	}
	question, err := service.CreateQuestion(CreateQuestionParams{
		ExamID:  "exam_1",
		Content: "old",
		Answer:  strPtr("x"),
	})
	if err != nil {
		t.Fatalf("CreateQuestion() error = %v", err)
	}

	updated, err := service.UpdateQuestion(UpdateQuestionRequest{
		QuestionID: question.ID,
		Params: UpdateQuestionParams{
			Content: strPtr("new"),
			Tags:    []string{"tag"},
		},
	})
	if err != nil {
		t.Fatalf("UpdateQuestion() error = %v", err)
	}
	if updated.Content != "new" || len(updated.Tags) != 1 {
		t.Fatalf("unexpected update: %+v", updated)
	}
	favorite, err := service.ToggleFavorite(question.ID)
	if err != nil {
		t.Fatalf("ToggleFavorite() error = %v", err)
	}
	if !favorite.IsFavorite {
		t.Fatalf("favorite not toggled: %+v", favorite)
	}
	if _, err := service.SubmitAnswer(SubmitAnswerRequest{QuestionID: question.ID, UserAnswer: "bad"}); err != nil {
		t.Fatalf("SubmitAnswer() error = %v", err)
	}
	stats, err := service.ResetProgress("exam_1")
	if err != nil {
		t.Fatalf("ResetProgress() error = %v", err)
	}
	if stats.TotalAttempts != 0 || stats.NewCount != 1 {
		t.Fatalf("unexpected reset stats: %+v", stats)
	}

	reloaded, err := NewService(dir)
	if err != nil {
		t.Fatalf("NewService(reloaded) error = %v", err)
	}
	got, err := reloaded.GetQuestion(question.ID)
	if err != nil {
		t.Fatalf("GetQuestion() error = %v", err)
	}
	if got == nil || got.Content != "new" || !got.IsFavorite || got.AttemptCount != 0 {
		t.Fatalf("unexpected reloaded question: %+v", got)
	}
	if err := reloaded.DeleteQuestion(question.ID); err != nil {
		t.Fatalf("DeleteQuestion() error = %v", err)
	}
	missing, err := reloaded.GetQuestion(question.ID)
	if err != nil {
		t.Fatalf("GetQuestion(deleted) error = %v", err)
	}
	if missing != nil {
		t.Fatalf("deleted question still exists: %+v", missing)
	}
}

func TestCsvPreviewImportAndVfsSync(t *testing.T) {
	dir := t.TempDir()
	vfsService, err := vfs.NewService(dir)
	if err != nil {
		t.Fatalf("vfs.NewService() error = %v", err)
	}
	service, err := NewService(dir, vfsService)
	if err != nil {
		t.Fatalf("NewService() error = %v", err)
	}
	csvPath := filepath.Join(dir, "questions.csv")
	if err := os.WriteFile(csvPath, []byte("\xEF\xBB\xBF题干,答案,解析,选项,标签,难度,题型\n1+1=?,2,加法,A. 1; B. 2,math;basic,简单,单选题\n"), 0o600); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	preview, err := service.GetCsvPreview(csvPath, 5)
	if err != nil {
		t.Fatalf("GetCsvPreview() error = %v", err)
	}
	if preview.Encoding != "UTF-8 BOM" || preview.TotalRows != 1 || len(preview.Headers) != 7 || preview.Headers[0] != "题干" {
		t.Fatalf("unexpected preview: %+v", preview)
	}

	result, err := service.ImportQuestionsCsv(CsvImportRequest{
		FilePath: csvPath,
		ExamID:   "exam_csv",
		FieldMapping: map[string]string{
			"题干": "content",
			"答案": "answer",
			"解析": "explanation",
			"选项": "options",
			"标签": "tags",
			"难度": "difficulty",
			"题型": "question_type",
		},
		DuplicateStrategy: CsvDuplicateSkip,
	})
	if err != nil {
		t.Fatalf("ImportQuestionsCsv() error = %v", err)
	}
	if result.SuccessCount != 1 || result.TotalRows != 1 || result.ExamID != "exam_csv" {
		t.Fatalf("unexpected import result: %+v", result)
	}

	list, err := service.ListQuestions(ListQuestionsRequest{ExamID: "exam_csv"})
	if err != nil {
		t.Fatalf("ListQuestions() error = %v", err)
	}
	if list.Total != 1 {
		t.Fatalf("unexpected list: %+v", list)
	}
	question := list.Questions[0]
	if question.Answer != "2" || question.Explanation != "加法" || question.QuestionType != "single_choice" || question.Difficulty != "easy" {
		t.Fatalf("unexpected imported question: %+v", question)
	}
	if len(question.Options) != 2 || question.Options[1].Key != "B" || question.ResourceID == "" || question.ResourceHash == "" {
		t.Fatalf("expected parsed options and VFS resource: %+v", question)
	}
	refs, err := vfsService.GetResourceRefs(vfs.GetResourceRefsInput{SourceIDs: []string{question.ID}})
	if err != nil {
		t.Fatalf("GetResourceRefs() error = %v", err)
	}
	if len(refs.Refs) != 1 || refs.Refs[0].ResourceHash != question.ResourceHash {
		t.Fatalf("unexpected refs: %+v question=%+v", refs, question)
	}
}

func TestCsvDuplicateSkipOverwriteAndMerge(t *testing.T) {
	service := newTestService(t)
	dir := t.TempDir()
	csvPath := filepath.Join(dir, "questions.csv")
	mapping := map[string]string{
		"content":     "content",
		"answer":      "answer",
		"explanation": "explanation",
		"tags":        "tags",
	}
	if err := os.WriteFile(csvPath, []byte("content,answer,explanation,tags\nSame?,A,first,one\n"), 0o600); err != nil {
		t.Fatalf("WriteFile(first) error = %v", err)
	}
	if result, err := service.ImportQuestionsCsv(CsvImportRequest{FilePath: csvPath, ExamID: "exam_csv", FieldMapping: mapping}); err != nil || result.SuccessCount != 1 {
		t.Fatalf("first import result=%+v err=%v", result, err)
	}
	if result, err := service.ImportQuestionsCsv(CsvImportRequest{FilePath: csvPath, ExamID: "exam_csv", FieldMapping: mapping, DuplicateStrategy: CsvDuplicateSkip}); err != nil || result.SkippedCount != 1 {
		t.Fatalf("skip import result=%+v err=%v", result, err)
	}

	if err := os.WriteFile(csvPath, []byte("content,answer,explanation,tags\nSame?,B,second,two\n"), 0o600); err != nil {
		t.Fatalf("WriteFile(overwrite) error = %v", err)
	}
	if result, err := service.ImportQuestionsCsv(CsvImportRequest{FilePath: csvPath, ExamID: "exam_csv", FieldMapping: mapping, DuplicateStrategy: CsvDuplicateOverwrite}); err != nil || result.SuccessCount != 1 {
		t.Fatalf("overwrite import result=%+v err=%v", result, err)
	}
	list, err := service.ListQuestions(ListQuestionsRequest{ExamID: "exam_csv"})
	if err != nil {
		t.Fatalf("ListQuestions() error = %v", err)
	}
	if list.Total != 1 || list.Questions[0].Answer != "B" || list.Questions[0].Explanation != "second" || list.Questions[0].Tags[0] != "two" {
		t.Fatalf("overwrite did not update existing question: %+v", list)
	}

	noAnswerPath := filepath.Join(dir, "no-answer.csv")
	if err := os.WriteFile(noAnswerPath, []byte("content\nMerge me\n"), 0o600); err != nil {
		t.Fatalf("WriteFile(noAnswer) error = %v", err)
	}
	if _, err := service.ImportQuestionsCsv(CsvImportRequest{
		FilePath:     noAnswerPath,
		ExamID:       "exam_csv",
		FieldMapping: map[string]string{"content": "content"},
	}); err != nil {
		t.Fatalf("import no-answer question error = %v", err)
	}
	if err := os.WriteFile(noAnswerPath, []byte("content,answer,explanation\nMerge me,C,filled\n"), 0o600); err != nil {
		t.Fatalf("WriteFile(merge) error = %v", err)
	}
	if result, err := service.ImportQuestionsCsv(CsvImportRequest{
		FilePath:          noAnswerPath,
		ExamID:            "exam_csv",
		FieldMapping:      map[string]string{"content": "content", "answer": "answer", "explanation": "explanation"},
		DuplicateStrategy: CsvDuplicateMerge,
	}); err != nil || result.SuccessCount != 1 {
		t.Fatalf("merge import result=%+v err=%v", result, err)
	}
	list, err = service.ListQuestions(ListQuestionsRequest{ExamID: "exam_csv"})
	if err != nil {
		t.Fatalf("ListQuestions(after merge) error = %v", err)
	}
	merged := false
	for _, question := range list.Questions {
		if question.Content == "Merge me" && question.Answer == "C" && question.Explanation == "filled" {
			merged = true
		}
	}
	if !merged {
		t.Fatalf("merge did not fill blank fields: %+v", list.Questions)
	}
}

func TestCsvExportFieldsEncodingAndFormulaNeutralization(t *testing.T) {
	service := newTestService(t)
	if _, err := service.CreateQuestion(CreateQuestionParams{
		ExamID:      "exam_csv",
		Content:     "=SUM(A1:A2)",
		Answer:      strPtr("A"),
		Explanation: strPtr("formula-like content should be neutralized"),
		Options: []QuestionOption{
			{Key: "A", Content: "ok"},
		},
		Tags: []string{"tag"},
	}); err != nil {
		t.Fatalf("CreateQuestion() error = %v", err)
	}
	exportPath := filepath.Join(t.TempDir(), "export.csv")
	result, err := service.ExportQuestionsCsv(CsvExportRequest{
		ExamID:   "exam_csv",
		FilePath: exportPath,
		Fields:   []string{"content", "answer", "options", "tags"},
		Encoding: "utf8_bom",
	})
	if err != nil {
		t.Fatalf("ExportQuestionsCsv() error = %v", err)
	}
	if result.ExportedCount != 1 || result.FileSize <= 3 {
		t.Fatalf("unexpected export result: %+v", result)
	}
	data, err := os.ReadFile(exportPath)
	if err != nil {
		t.Fatalf("ReadFile(export) error = %v", err)
	}
	if !bytes.HasPrefix(data, []byte{0xEF, 0xBB, 0xBF}) {
		t.Fatalf("expected UTF-8 BOM, got %q", string(data[:3]))
	}
	text := string(data[3:])
	if !strings.Contains(text, "题干内容,答案,选项,标签") || !strings.Contains(text, "\"\t=SUM(A1:A2)\"") {
		t.Fatalf("unexpected export text: %q", text)
	}

	fields := service.GetCsvExportableFields()
	if len(fields) == 0 || fields[0][0] != "content" || fields[0][1] != "题干内容" {
		t.Fatalf("unexpected exportable fields: %+v", fields)
	}
}

func strPtr(value string) *string {
	return &value
}

func boolPtr(value bool) *bool {
	return &value
}
