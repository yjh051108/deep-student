package reviewplan

import (
	"crypto/rand"
	"deep-student-go/internal/qbank"
	"deep-student-go/internal/storage"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"math/big"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"
)

const (
	defaultEaseFactor         = 2.5
	passingQuality            = 3
	graduationIntervalDays    = 21
	graduationRepetitions     = 3
	difficultFailureThreshold = 3
	defaultDueLimit           = 100
	defaultListLimit          = 100
	defaultHistoryLimit       = 50
)

type Service struct {
	mu    sync.RWMutex
	path  string
	state store
	qbank *qbank.Service
}

type store struct {
	Plans     []ReviewPlan    `json:"plans"`
	Histories []ReviewHistory `json:"histories"`
}

type ReviewPlan struct {
	ID                  string  `json:"id"`
	QuestionID          string  `json:"question_id"`
	ExamID              string  `json:"exam_id"`
	EaseFactor          float64 `json:"ease_factor"`
	IntervalDays        int     `json:"interval_days"`
	Repetitions         int     `json:"repetitions"`
	NextReviewDate      string  `json:"next_review_date"`
	LastReviewDate      *string `json:"last_review_date"`
	Status              string  `json:"status"`
	TotalReviews        int     `json:"total_reviews"`
	TotalCorrect        int     `json:"total_correct"`
	ConsecutiveFailures int     `json:"consecutive_failures"`
	IsDifficult         bool    `json:"is_difficult"`
	CreatedAt           string  `json:"created_at"`
	UpdatedAt           string  `json:"updated_at"`
}

type ReviewHistory struct {
	ID                string  `json:"id"`
	PlanID            string  `json:"plan_id"`
	QuestionID        string  `json:"question_id"`
	Quality           int     `json:"quality"`
	Passed            bool    `json:"passed"`
	EaseFactorBefore  float64 `json:"ease_factor_before"`
	EaseFactorAfter   float64 `json:"ease_factor_after"`
	IntervalBefore    int     `json:"interval_before"`
	IntervalAfter     int     `json:"interval_after"`
	RepetitionsBefore int     `json:"repetitions_before"`
	RepetitionsAfter  int     `json:"repetitions_after"`
	ReviewedAt        string  `json:"reviewed_at"`
	UserAnswer        *string `json:"user_answer"`
	TimeSpentSeconds  *int    `json:"time_spent_seconds"`
}

type DueReviewsFilter struct {
	ExamID        *string  `json:"exam_id,omitempty"`
	UntilDate     *string  `json:"until_date,omitempty"`
	Status        []string `json:"status,omitempty"`
	DifficultOnly *bool    `json:"difficult_only,omitempty"`
	Limit         *int     `json:"limit,omitempty"`
	Offset        *int     `json:"offset,omitempty"`
}

type DueReviewsResult struct {
	Plans   []ReviewPlan `json:"plans"`
	Total   int          `json:"total"`
	HasMore bool         `json:"has_more"`
}

type ReviewStats struct {
	ExamID         *string `json:"exam_id"`
	TotalPlans     int     `json:"total_plans"`
	NewCount       int     `json:"new_count"`
	LearningCount  int     `json:"learning_count"`
	ReviewingCount int     `json:"reviewing_count"`
	GraduatedCount int     `json:"graduated_count"`
	SuspendedCount int     `json:"suspended_count"`
	DueToday       int     `json:"due_today"`
	OverdueCount   int     `json:"overdue_count"`
	DifficultCount int     `json:"difficult_count"`
	TotalReviews   int     `json:"total_reviews"`
	TotalCorrect   int     `json:"total_correct"`
	AvgCorrectRate float64 `json:"avg_correct_rate"`
	AvgEaseFactor  float64 `json:"avg_ease_factor"`
	UpdatedAt      string  `json:"updated_at"`
}

type ProcessReviewResult struct {
	Plan           ReviewPlan    `json:"plan"`
	Passed         bool          `json:"passed"`
	NewInterval    int           `json:"new_interval"`
	NextReviewDate string        `json:"next_review_date"`
	History        ReviewHistory `json:"history"`
}

type BatchCreateResult struct {
	Created int          `json:"created"`
	Skipped int          `json:"skipped"`
	Failed  int          `json:"failed"`
	Plans   []ReviewPlan `json:"plans"`
}

type CalendarHeatmapData struct {
	Date   string `json:"date"`
	Count  int    `json:"count"`
	Passed int    `json:"passed"`
	Failed int    `json:"failed"`
}

func NewService(dataDir string, qbankService *qbank.Service) (*Service, error) {
	service := &Service{
		path:  filepath.Join(dataDir, "review-plan-go.json"),
		qbank: qbankService,
		state: store{
			Plans:     []ReviewPlan{},
			Histories: []ReviewHistory{},
		},
	}
	if err := service.load(); err != nil {
		return nil, err
	}
	return service, nil
}

func (s *Service) Create(questionID string, examID string) (ReviewPlan, error) {
	questionID = strings.TrimSpace(questionID)
	examID = strings.TrimSpace(examID)
	if questionID == "" || examID == "" {
		return ReviewPlan{}, errors.New("questionId and examId are required")
	}
	if err := s.validateQuestionExam(questionID, examID); err != nil {
		return ReviewPlan{}, err
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	if existing, ok := s.findPlanByQuestionLocked(questionID); ok {
		return existing, fmt.Errorf("review plan already exists for question: %s", questionID)
	}
	now := nowISO()
	today := todayDate()
	plan := ReviewPlan{
		ID:             "rp_" + randomToken(16),
		QuestionID:     questionID,
		ExamID:         examID,
		EaseFactor:     defaultEaseFactor,
		IntervalDays:   0,
		Repetitions:    0,
		NextReviewDate: today,
		Status:         "new",
		CreatedAt:      now,
		UpdatedAt:      now,
	}
	s.state.Plans = append(s.state.Plans, plan)
	if err := s.flushLocked(); err != nil {
		return ReviewPlan{}, err
	}
	return plan, nil
}

func (s *Service) Process(planID string, quality int, userAnswer *string, timeSpentSeconds *int) (ProcessReviewResult, error) {
	if quality < 0 || quality > 5 {
		return ProcessReviewResult{}, errors.New("quality must be between 0 and 5")
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	index, ok := s.findPlanIndexLocked(planID)
	if !ok {
		return ProcessReviewResult{}, fmt.Errorf("review plan not found: %s", planID)
	}
	previous := s.state.Plans[index]
	interval, easeFactor, repetitions := nextReview(quality, previous.Repetitions, previous.EaseFactor, previous.IntervalDays)
	passed := quality >= passingQuality
	today := todayDate()
	nextDate := dateAfterDays(today, interval)
	status, failures, difficult := nextStatus(previous, passed, interval, repetitions)

	updated := previous
	updated.EaseFactor = easeFactor
	updated.IntervalDays = interval
	updated.Repetitions = repetitions
	updated.NextReviewDate = nextDate
	updated.LastReviewDate = &today
	updated.Status = status
	updated.TotalReviews++
	if passed {
		updated.TotalCorrect++
	}
	updated.ConsecutiveFailures = failures
	updated.IsDifficult = difficult
	updated.UpdatedAt = nowISO()
	s.state.Plans[index] = updated

	history := ReviewHistory{
		ID:                "rh_" + randomToken(16),
		PlanID:            updated.ID,
		QuestionID:        updated.QuestionID,
		Quality:           quality,
		Passed:            passed,
		EaseFactorBefore:  previous.EaseFactor,
		EaseFactorAfter:   updated.EaseFactor,
		IntervalBefore:    previous.IntervalDays,
		IntervalAfter:     updated.IntervalDays,
		RepetitionsBefore: previous.Repetitions,
		RepetitionsAfter:  updated.Repetitions,
		ReviewedAt:        nowISO(),
		UserAnswer:        userAnswer,
		TimeSpentSeconds:  timeSpentSeconds,
	}
	s.state.Histories = append(s.state.Histories, history)
	if err := s.flushLocked(); err != nil {
		return ProcessReviewResult{}, err
	}

	return ProcessReviewResult{
		Plan:           updated,
		Passed:         passed,
		NewInterval:    interval,
		NextReviewDate: nextDate,
		History:        history,
	}, nil
}

func (s *Service) GetDue(examID *string, untilDate *string) (DueReviewsResult, error) {
	return s.GetDueWithFilter(DueReviewsFilter{
		ExamID:    examID,
		UntilDate: untilDate,
		Limit:     intPtr(defaultDueLimit),
	})
}

func (s *Service) GetDueWithFilter(filter DueReviewsFilter) (DueReviewsResult, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	until := todayDate()
	if filter.UntilDate != nil && strings.TrimSpace(*filter.UntilDate) != "" {
		until = strings.TrimSpace(*filter.UntilDate)
	}
	filtered := make([]ReviewPlan, 0)
	for _, plan := range s.state.Plans {
		if !planMatchesFilter(plan, filter) {
			continue
		}
		if plan.Status == "suspended" {
			continue
		}
		if plan.NextReviewDate > until {
			continue
		}
		filtered = append(filtered, plan)
	}
	sortPlans(filtered)
	return pagePlans(filtered, valueOrInt(filter.Limit, defaultDueLimit), valueOrInt(filter.Offset, 0)), nil
}

func (s *Service) GetStats(examID *string) (ReviewStats, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return calculateStats(s.state.Plans, examID), nil
}

func (s *Service) RefreshStats(examID *string) (ReviewStats, error) {
	return s.GetStats(examID)
}

func (s *Service) GetByQuestion(questionID string) (*ReviewPlan, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	plan, ok := s.findPlanByQuestionLocked(questionID)
	if !ok {
		return nil, nil
	}
	return &plan, nil
}

func (s *Service) Get(planID string) (*ReviewPlan, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	plan, ok := s.findPlanLocked(planID)
	if !ok {
		return nil, nil
	}
	return &plan, nil
}

func (s *Service) Suspend(planID string) (ReviewPlan, error) {
	return s.setStatus(planID, "suspended")
}

func (s *Service) Resume(planID string) (ReviewPlan, error) {
	return s.setStatus(planID, "reviewing")
}

func (s *Service) Delete(planID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	index, ok := s.findPlanIndexLocked(planID)
	if !ok {
		return nil
	}
	s.state.Plans = append(s.state.Plans[:index], s.state.Plans[index+1:]...)
	s.state.Histories = filterHistories(s.state.Histories, planID)
	return s.flushLocked()
}

func (s *Service) GetHistory(planID string, limit int) ([]ReviewHistory, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if limit <= 0 {
		limit = defaultHistoryLimit
	}
	out := make([]ReviewHistory, 0)
	for index := len(s.state.Histories) - 1; index >= 0 && len(out) < limit; index-- {
		history := s.state.Histories[index]
		if history.PlanID == planID {
			out = append(out, history)
		}
	}
	return out, nil
}

func (s *Service) BatchCreate(questionIDs []string, examID string) (BatchCreateResult, error) {
	result := BatchCreateResult{Plans: []ReviewPlan{}}
	for _, questionID := range questionIDs {
		plan, err := s.Create(questionID, examID)
		if err != nil {
			if strings.Contains(err.Error(), "already exists") {
				result.Skipped++
			} else {
				result.Failed++
			}
			continue
		}
		result.Created++
		result.Plans = append(result.Plans, plan)
	}
	return result, nil
}

func (s *Service) CreateForExam(examID string) (BatchCreateResult, error) {
	if s.qbank == nil {
		return BatchCreateResult{}, errors.New("qbank service is required")
	}
	list, err := s.qbank.ListQuestions(qbank.ListQuestionsRequest{ExamID: examID, Page: 1, PageSize: 100})
	if err != nil {
		return BatchCreateResult{}, err
	}
	questionIDs := make([]string, 0, list.Total)
	for {
		for _, question := range list.Questions {
			questionIDs = append(questionIDs, question.ID)
		}
		if !list.HasMore {
			break
		}
		list.Page++
		list, err = s.qbank.ListQuestions(qbank.ListQuestionsRequest{ExamID: examID, Page: list.Page, PageSize: list.PageSize})
		if err != nil {
			return BatchCreateResult{}, err
		}
	}
	return s.BatchCreate(questionIDs, examID)
}

func (s *Service) ListByExam(examID string, limit int, offset int) (DueReviewsResult, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	filtered := make([]ReviewPlan, 0)
	for _, plan := range s.state.Plans {
		if plan.ExamID == examID {
			filtered = append(filtered, plan)
		}
	}
	sortPlans(filtered)
	if limit <= 0 {
		limit = defaultListLimit
	}
	return pagePlans(filtered, limit, offset), nil
}

func (s *Service) GetOrCreate(questionID string, examID string) (ReviewPlan, error) {
	s.mu.RLock()
	existing, ok := s.findPlanByQuestionLocked(questionID)
	s.mu.RUnlock()
	if ok {
		return existing, nil
	}
	return s.Create(questionID, examID)
}

func (s *Service) GetCalendarData(startDate *string, endDate *string, examID *string) ([]CalendarHeatmapData, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	start := ""
	end := ""
	if startDate != nil {
		start = strings.TrimSpace(*startDate)
	}
	if endDate != nil {
		end = strings.TrimSpace(*endDate)
	}
	byDate := map[string]*CalendarHeatmapData{}
	for _, history := range s.state.Histories {
		plan, ok := s.findPlanLocked(history.PlanID)
		if !ok || !matchesOptionalExam(plan.ExamID, examID) {
			continue
		}
		date := datePart(history.ReviewedAt)
		if date == "" || (start != "" && date < start) || (end != "" && date > end) {
			continue
		}
		item := byDate[date]
		if item == nil {
			item = &CalendarHeatmapData{Date: date}
			byDate[date] = item
		}
		item.Count++
		if history.Passed {
			item.Passed++
		} else {
			item.Failed++
		}
	}
	out := make([]CalendarHeatmapData, 0, len(byDate))
	for _, item := range byDate {
		out = append(out, *item)
	}
	sort.SliceStable(out, func(a, b int) bool { return out[a].Date < out[b].Date })
	return out, nil
}

func (s *Service) validateQuestionExam(questionID string, examID string) error {
	if s.qbank == nil {
		return nil
	}
	question, err := s.qbank.GetQuestion(questionID)
	if err != nil {
		return err
	}
	if question == nil {
		return fmt.Errorf("question not found: %s", questionID)
	}
	if question.ExamID != examID {
		return fmt.Errorf("question %s belongs to exam_id=%s, got exam_id=%s", questionID, question.ExamID, examID)
	}
	return nil
}

func (s *Service) setStatus(planID string, status string) (ReviewPlan, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	index, ok := s.findPlanIndexLocked(planID)
	if !ok {
		return ReviewPlan{}, fmt.Errorf("review plan not found: %s", planID)
	}
	s.state.Plans[index].Status = status
	s.state.Plans[index].UpdatedAt = nowISO()
	if status == "reviewing" && s.state.Plans[index].NextReviewDate < todayDate() {
		s.state.Plans[index].NextReviewDate = todayDate()
	}
	if err := s.flushLocked(); err != nil {
		return ReviewPlan{}, err
	}
	return s.state.Plans[index], nil
}

func (s *Service) findPlanLocked(planID string) (ReviewPlan, bool) {
	for _, plan := range s.state.Plans {
		if plan.ID == planID {
			return plan, true
		}
	}
	return ReviewPlan{}, false
}

func (s *Service) findPlanIndexLocked(planID string) (int, bool) {
	for index, plan := range s.state.Plans {
		if plan.ID == planID {
			return index, true
		}
	}
	return -1, false
}

func (s *Service) findPlanByQuestionLocked(questionID string) (ReviewPlan, bool) {
	for _, plan := range s.state.Plans {
		if plan.QuestionID == questionID {
			return plan, true
		}
	}
	return ReviewPlan{}, false
}

func (s *Service) load() error {
	bytes, err := os.ReadFile(s.path)
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	if err != nil {
		return err
	}
	if len(bytes) == 0 {
		return nil
	}
	if err := json.Unmarshal(bytes, &s.state); err != nil {
		return err
	}
	if s.state.Plans == nil {
		s.state.Plans = []ReviewPlan{}
	}
	if s.state.Histories == nil {
		s.state.Histories = []ReviewHistory{}
	}
	return nil
}

func (s *Service) flushLocked() error {
	return storage.WriteJSONAtomic(s.path, s.state)
}

func nextReview(quality int, repetitions int, easeFactor float64, intervalDays int) (int, float64, int) {
	newEase := easeFactor + (0.1 - float64(5-quality)*(0.08+float64(5-quality)*0.02))
	if newEase < 1.3 {
		newEase = 1.3
	}
	newEase = math.Round(newEase*100) / 100
	if quality < passingQuality {
		return 1, newEase, 0
	}
	newRepetitions := repetitions + 1
	switch newRepetitions {
	case 1:
		return 1, newEase, newRepetitions
	case 2:
		return 6, newEase, newRepetitions
	default:
		interval := int(math.Round(float64(maxInt(intervalDays, 1)) * newEase))
		return maxInt(interval, 1), newEase, newRepetitions
	}
}

func nextStatus(plan ReviewPlan, passed bool, interval int, repetitions int) (string, int, bool) {
	failures := 0
	if !passed {
		failures = plan.ConsecutiveFailures + 1
	}
	difficult := failures >= difficultFailureThreshold
	if plan.Status == "suspended" {
		return "suspended", failures, difficult
	}
	if repetitions == 0 {
		return "new", failures, difficult
	}
	if repetitions < 2 {
		return "learning", failures, difficult
	}
	if interval >= graduationIntervalDays && repetitions >= graduationRepetitions {
		return "graduated", failures, difficult
	}
	return "reviewing", failures, difficult
}

func calculateStats(plans []ReviewPlan, examID *string) ReviewStats {
	stats := ReviewStats{ExamID: examID, UpdatedAt: nowISO()}
	today := todayDate()
	easeSum := 0.0
	for _, plan := range plans {
		if !matchesOptionalExam(plan.ExamID, examID) {
			continue
		}
		stats.TotalPlans++
		stats.TotalReviews += plan.TotalReviews
		stats.TotalCorrect += plan.TotalCorrect
		easeSum += plan.EaseFactor
		if plan.IsDifficult {
			stats.DifficultCount++
		}
		if plan.Status != "suspended" && plan.NextReviewDate <= today {
			stats.DueToday++
			if plan.NextReviewDate < today {
				stats.OverdueCount++
			}
		}
		switch plan.Status {
		case "learning":
			stats.LearningCount++
		case "reviewing":
			stats.ReviewingCount++
		case "graduated":
			stats.GraduatedCount++
		case "suspended":
			stats.SuspendedCount++
		default:
			stats.NewCount++
		}
	}
	if stats.TotalReviews > 0 {
		stats.AvgCorrectRate = float64(stats.TotalCorrect) / float64(stats.TotalReviews) * 100
	}
	if stats.TotalPlans > 0 {
		stats.AvgEaseFactor = easeSum / float64(stats.TotalPlans)
	}
	return stats
}

func planMatchesFilter(plan ReviewPlan, filter DueReviewsFilter) bool {
	if !matchesOptionalExam(plan.ExamID, filter.ExamID) {
		return false
	}
	if len(filter.Status) > 0 && !contains(filter.Status, plan.Status) {
		return false
	}
	if filter.DifficultOnly != nil && *filter.DifficultOnly && !plan.IsDifficult {
		return false
	}
	return true
}

func pagePlans(plans []ReviewPlan, limit int, offset int) DueReviewsResult {
	total := len(plans)
	if offset < 0 {
		offset = 0
	}
	if limit <= 0 {
		limit = defaultListLimit
	}
	if offset >= total {
		return DueReviewsResult{Plans: []ReviewPlan{}, Total: total, HasMore: false}
	}
	end := offset + limit
	if end > total {
		end = total
	}
	return DueReviewsResult{
		Plans:   plans[offset:end],
		Total:   total,
		HasMore: end < total,
	}
}

func sortPlans(plans []ReviewPlan) {
	sort.SliceStable(plans, func(a, b int) bool {
		if plans[a].NextReviewDate != plans[b].NextReviewDate {
			return plans[a].NextReviewDate < plans[b].NextReviewDate
		}
		if plans[a].UpdatedAt != plans[b].UpdatedAt {
			return plans[a].UpdatedAt < plans[b].UpdatedAt
		}
		return plans[a].ID < plans[b].ID
	})
}

func filterHistories(histories []ReviewHistory, planID string) []ReviewHistory {
	out := make([]ReviewHistory, 0, len(histories))
	for _, history := range histories {
		if history.PlanID != planID {
			out = append(out, history)
		}
	}
	return out
}

func matchesOptionalExam(planExamID string, examID *string) bool {
	return examID == nil || strings.TrimSpace(*examID) == "" || planExamID == *examID
}

func contains(values []string, needle string) bool {
	for _, value := range values {
		if value == needle {
			return true
		}
	}
	return false
}

func dateAfterDays(date string, days int) string {
	parsed, err := time.ParseInLocation("2006-01-02", date, time.UTC)
	if err != nil {
		parsed = time.Now().UTC()
	}
	return parsed.AddDate(0, 0, days).Format("2006-01-02")
}

func datePart(value string) string {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return ""
	}
	if parsed, err := time.Parse(time.RFC3339, trimmed); err == nil {
		return parsed.UTC().Format("2006-01-02")
	}
	if len(trimmed) >= 10 {
		return trimmed[:10]
	}
	return ""
}

func todayDate() string {
	return time.Now().UTC().Format("2006-01-02")
}

func nowISO() string {
	return time.Now().UTC().Format(time.RFC3339)
}

func valueOrInt(value *int, fallback int) int {
	if value == nil {
		return fallback
	}
	return *value
}

func intPtr(value int) *int {
	return &value
}

func maxInt(a int, b int) int {
	if a > b {
		return a
	}
	return b
}

func randomToken(length int) string {
	const alphabet = "0123456789abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ"
	out := make([]byte, length)
	max := big.NewInt(int64(len(alphabet)))
	for i := range out {
		n, err := rand.Int(rand.Reader, max)
		if err != nil {
			out[i] = alphabet[int(time.Now().UnixNano())%len(alphabet)]
			continue
		}
		out[i] = alphabet[n.Int64()]
	}
	return string(out)
}
