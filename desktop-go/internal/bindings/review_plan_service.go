package bindings

import (
	"deep-student-go/internal/app"
	"deep-student-go/internal/reviewplan"
)

type ReviewPlanService struct {
	app *app.App
}

func NewReviewPlanService(app *app.App) *ReviewPlanService {
	return &ReviewPlanService{app: app}
}

func (s *ReviewPlanService) Create(questionID string, examID string) (reviewplan.ReviewPlan, error) {
	return s.app.Review.Create(questionID, examID)
}

func (s *ReviewPlanService) Process(planID string, quality int, userAnswer *string, timeSpentSeconds *int) (reviewplan.ProcessReviewResult, error) {
	return s.app.Review.Process(planID, quality, userAnswer, timeSpentSeconds)
}

func (s *ReviewPlanService) GetDue(examID *string, untilDate *string) (reviewplan.DueReviewsResult, error) {
	return s.app.Review.GetDue(examID, untilDate)
}

func (s *ReviewPlanService) GetDueWithFilter(filter reviewplan.DueReviewsFilter) (reviewplan.DueReviewsResult, error) {
	return s.app.Review.GetDueWithFilter(filter)
}

func (s *ReviewPlanService) GetStats(examID *string) (reviewplan.ReviewStats, error) {
	return s.app.Review.GetStats(examID)
}

func (s *ReviewPlanService) RefreshStats(examID *string) (reviewplan.ReviewStats, error) {
	return s.app.Review.RefreshStats(examID)
}

func (s *ReviewPlanService) GetByQuestion(questionID string) (*reviewplan.ReviewPlan, error) {
	return s.app.Review.GetByQuestion(questionID)
}

func (s *ReviewPlanService) Get(planID string) (*reviewplan.ReviewPlan, error) {
	return s.app.Review.Get(planID)
}

func (s *ReviewPlanService) Suspend(planID string) (reviewplan.ReviewPlan, error) {
	return s.app.Review.Suspend(planID)
}

func (s *ReviewPlanService) Resume(planID string) (reviewplan.ReviewPlan, error) {
	return s.app.Review.Resume(planID)
}

func (s *ReviewPlanService) Delete(planID string) (bool, error) {
	return true, s.app.Review.Delete(planID)
}

func (s *ReviewPlanService) GetHistory(planID string, limit int) ([]reviewplan.ReviewHistory, error) {
	return s.app.Review.GetHistory(planID, limit)
}

func (s *ReviewPlanService) BatchCreate(questionIDs []string, examID string) (reviewplan.BatchCreateResult, error) {
	return s.app.Review.BatchCreate(questionIDs, examID)
}

func (s *ReviewPlanService) CreateForExam(examID string) (reviewplan.BatchCreateResult, error) {
	return s.app.Review.CreateForExam(examID)
}

func (s *ReviewPlanService) ListByExam(examID string, limit int, offset int) (reviewplan.DueReviewsResult, error) {
	return s.app.Review.ListByExam(examID, limit, offset)
}

func (s *ReviewPlanService) GetOrCreate(questionID string, examID string) (reviewplan.ReviewPlan, error) {
	return s.app.Review.GetOrCreate(questionID, examID)
}

func (s *ReviewPlanService) GetCalendarData(startDate *string, endDate *string, examID *string) ([]reviewplan.CalendarHeatmapData, error) {
	return s.app.Review.GetCalendarData(startDate, endDate, examID)
}
