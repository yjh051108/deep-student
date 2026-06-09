package reviewplan

import (
	"deep-student-go/internal/qbank"
	"testing"
	"time"
)

func newTestService(t *testing.T) (*Service, *qbank.Service, qbank.Question) {
	t.Helper()
	dir := t.TempDir()
	qbankService, err := qbank.NewService(dir)
	if err != nil {
		t.Fatalf("qbank.NewService() error = %v", err)
	}
	question, err := qbankService.CreateQuestion(qbank.CreateQuestionParams{
		ExamID:       "exam_review",
		Content:      "Review me",
		Answer:       strPtr("A"),
		QuestionType: strPtr("single_choice"),
	})
	if err != nil {
		t.Fatalf("CreateQuestion() error = %v", err)
	}
	service, err := NewService(dir, qbankService)
	if err != nil {
		t.Fatalf("NewService() error = %v", err)
	}
	return service, qbankService, question
}

func TestCreateProcessStatsHistoryCalendarAndPersistence(t *testing.T) {
	service, _, question := newTestService(t)

	plan, err := service.Create(question.ID, question.ExamID)
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}
	if plan.QuestionID != question.ID || plan.ExamID != question.ExamID || plan.Status != "new" || plan.NextReviewDate != todayDate() {
		t.Fatalf("unexpected plan: %+v", plan)
	}
	if _, err := service.Create(question.ID, question.ExamID); err == nil {
		t.Fatalf("expected duplicate create error")
	}

	answer := "A"
	spent := 12
	result, err := service.Process(plan.ID, 5, &answer, &spent)
	if err != nil {
		t.Fatalf("Process() error = %v", err)
	}
	if !result.Passed || result.NewInterval != 1 || result.Plan.TotalReviews != 1 || result.Plan.TotalCorrect != 1 || result.History.UserAnswer == nil || *result.History.UserAnswer != "A" {
		t.Fatalf("unexpected process result: %+v", result)
	}

	due, err := service.GetDue(&question.ExamID, nil)
	if err != nil {
		t.Fatalf("GetDue() error = %v", err)
	}
	if due.Total != 0 {
		t.Fatalf("expected no due plans after successful review, got %+v", due)
	}

	stats, err := service.GetStats(&question.ExamID)
	if err != nil {
		t.Fatalf("GetStats() error = %v", err)
	}
	if stats.TotalPlans != 1 || stats.LearningCount != 1 || stats.TotalReviews != 1 || stats.TotalCorrect != 1 || stats.AvgCorrectRate != 100 {
		t.Fatalf("unexpected stats: %+v", stats)
	}

	history, err := service.GetHistory(plan.ID, 10)
	if err != nil {
		t.Fatalf("GetHistory() error = %v", err)
	}
	if len(history) != 1 || history[0].Quality != 5 || history[0].TimeSpentSeconds == nil || *history[0].TimeSpentSeconds != 12 {
		t.Fatalf("unexpected history: %+v", history)
	}

	today := time.Now().UTC().Format("2006-01-02")
	calendar, err := service.GetCalendarData(&today, &today, &question.ExamID)
	if err != nil {
		t.Fatalf("GetCalendarData() error = %v", err)
	}
	if len(calendar) != 1 || calendar[0].Date != today || calendar[0].Count != 1 || calendar[0].Passed != 1 {
		t.Fatalf("unexpected calendar: %+v", calendar)
	}

	reloaded, err := NewService(service.pathDir(), service.qbank)
	if err != nil {
		t.Fatalf("NewService(reloaded) error = %v", err)
	}
	got, err := reloaded.Get(plan.ID)
	if err != nil {
		t.Fatalf("Get(reloaded) error = %v", err)
	}
	if got == nil || got.TotalReviews != 1 {
		t.Fatalf("unexpected reloaded plan: %+v", got)
	}
}

func TestBatchCreateFilterSuspendResumeDeleteAndQuestionValidation(t *testing.T) {
	service, qbankService, first := newTestService(t)
	second, err := qbankService.CreateQuestion(qbank.CreateQuestionParams{
		ExamID:  first.ExamID,
		Content: "Second",
		Answer:  strPtr("B"),
	})
	if err != nil {
		t.Fatalf("CreateQuestion(second) error = %v", err)
	}
	other, err := qbankService.CreateQuestion(qbank.CreateQuestionParams{
		ExamID:  "other_exam",
		Content: "Other",
		Answer:  strPtr("C"),
	})
	if err != nil {
		t.Fatalf("CreateQuestion(other) error = %v", err)
	}

	if _, err := service.Create(other.ID, first.ExamID); err == nil {
		t.Fatalf("expected exam mismatch error")
	}

	batch, err := service.BatchCreate([]string{first.ID, second.ID}, first.ExamID)
	if err != nil {
		t.Fatalf("BatchCreate() error = %v", err)
	}
	if batch.Created != 2 || len(batch.Plans) != 2 {
		t.Fatalf("unexpected batch: %+v", batch)
	}
	repeat, err := service.BatchCreate([]string{first.ID, "missing"}, first.ExamID)
	if err != nil {
		t.Fatalf("BatchCreate(repeat) error = %v", err)
	}
	if repeat.Skipped != 1 || repeat.Failed != 1 {
		t.Fatalf("unexpected repeat batch: %+v", repeat)
	}

	filtered, err := service.GetDueWithFilter(DueReviewsFilter{
		ExamID: &first.ExamID,
		Limit:  intPtr(1),
	})
	if err != nil {
		t.Fatalf("GetDueWithFilter() error = %v", err)
	}
	if filtered.Total != 2 || len(filtered.Plans) != 1 || !filtered.HasMore {
		t.Fatalf("unexpected filtered due: %+v", filtered)
	}

	suspended, err := service.Suspend(batch.Plans[0].ID)
	if err != nil {
		t.Fatalf("Suspend() error = %v", err)
	}
	if suspended.Status != "suspended" {
		t.Fatalf("unexpected suspended plan: %+v", suspended)
	}
	resumed, err := service.Resume(suspended.ID)
	if err != nil {
		t.Fatalf("Resume() error = %v", err)
	}
	if resumed.Status != "reviewing" {
		t.Fatalf("unexpected resumed plan: %+v", resumed)
	}

	byQuestion, err := service.GetByQuestion(first.ID)
	if err != nil {
		t.Fatalf("GetByQuestion() error = %v", err)
	}
	if byQuestion == nil || byQuestion.ID != batch.Plans[0].ID {
		t.Fatalf("unexpected by question: %+v", byQuestion)
	}

	if err := service.Delete(batch.Plans[0].ID); err != nil {
		t.Fatalf("Delete() error = %v", err)
	}
	missing, err := service.Get(batch.Plans[0].ID)
	if err != nil {
		t.Fatalf("Get(deleted) error = %v", err)
	}
	if missing != nil {
		t.Fatalf("deleted plan still exists: %+v", missing)
	}
}

func TestCreateForExamAndGetOrCreate(t *testing.T) {
	service, qbankService, first := newTestService(t)
	if _, err := qbankService.CreateQuestion(qbank.CreateQuestionParams{
		ExamID:  first.ExamID,
		Content: "Third",
		Answer:  strPtr("C"),
	}); err != nil {
		t.Fatalf("CreateQuestion(third) error = %v", err)
	}

	result, err := service.CreateForExam(first.ExamID)
	if err != nil {
		t.Fatalf("CreateForExam() error = %v", err)
	}
	if result.Created != 2 {
		t.Fatalf("unexpected create for exam result: %+v", result)
	}
	existing, err := service.GetOrCreate(first.ID, first.ExamID)
	if err != nil {
		t.Fatalf("GetOrCreate(existing) error = %v", err)
	}
	if existing.QuestionID != first.ID {
		t.Fatalf("unexpected existing plan: %+v", existing)
	}
	list, err := service.ListByExam(first.ExamID, 10, 0)
	if err != nil {
		t.Fatalf("ListByExam() error = %v", err)
	}
	if list.Total != 2 || len(list.Plans) != 2 {
		t.Fatalf("unexpected list: %+v", list)
	}
}

func (s *Service) pathDir() string {
	return filepathDir(s.path)
}

func filepathDir(path string) string {
	for i := len(path) - 1; i >= 0; i-- {
		if path[i] == '/' || path[i] == '\\' {
			return path[:i]
		}
	}
	return "."
}

func strPtr(value string) *string {
	return &value
}
