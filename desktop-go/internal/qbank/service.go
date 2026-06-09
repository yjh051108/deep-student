package qbank

import (
	"bytes"
	"context"
	"crypto/rand"
	"deep-student-go/internal/storage"
	"deep-student-go/internal/vfs"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math"
	"math/big"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"
)

type Service struct {
	mu              sync.RWMutex
	configMu        sync.RWMutex
	path            string
	state           store
	vfs             *vfs.Service
	emit            func(name string, payload any)
	apiConfigLoader func() (APIConfigState, error)
	httpClient      *http.Client
}

type store struct {
	Questions   []Question            `json:"questions"`
	Submissions []AnswerSubmission    `json:"submissions"`
	Histories   []QuestionHistory     `json:"histories"`
	SyncConfigs map[string]SyncConfig `json:"sync_configs,omitempty"`
}

type Question struct {
	ID            string           `json:"id"`
	ExamID        string           `json:"exam_id"`
	CardID        *string          `json:"card_id,omitempty"`
	QuestionLabel *string          `json:"question_label,omitempty"`
	Content       string           `json:"content"`
	Options       []QuestionOption `json:"options"`
	Answer        string           `json:"answer"`
	Explanation   string           `json:"explanation"`
	QuestionType  string           `json:"question_type"`
	Difficulty    string           `json:"difficulty"`
	Tags          []string         `json:"tags"`
	Status        string           `json:"status"`
	UserAnswer    string           `json:"user_answer"`
	IsCorrect     *bool            `json:"is_correct"`
	AttemptCount  int              `json:"attempt_count"`
	CorrectCount  int              `json:"correct_count"`
	LastAttemptAt *string          `json:"last_attempt_at"`
	UserNote      string           `json:"user_note"`
	IsFavorite    bool             `json:"is_favorite"`
	Images        []QuestionImage  `json:"images"`
	SourceType    string           `json:"source_type"`
	SourceRef     *string          `json:"source_ref,omitempty"`
	ParentID      *string          `json:"parent_id,omitempty"`
	CreatedAt     string           `json:"created_at"`
	UpdatedAt     string           `json:"updated_at"`
	ResourceID    string           `json:"resource_id,omitempty"`
	ResourceHash  string           `json:"resource_hash,omitempty"`
	AIFeedback    *string          `json:"ai_feedback,omitempty"`
	AIScore       *float64         `json:"ai_score,omitempty"`
	AIGradedAt    *string          `json:"ai_graded_at,omitempty"`
}

type QuestionOption struct {
	Key     string `json:"key"`
	Content string `json:"content"`
}

type QuestionImage struct {
	ID   string `json:"id"`
	Name string `json:"name"`
	Mime string `json:"mime"`
	Hash string `json:"hash"`
}

type QuestionFilters struct {
	Status       []string `json:"status,omitempty"`
	Difficulty   []string `json:"difficulty,omitempty"`
	QuestionType []string `json:"question_type,omitempty"`
	Tags         []string `json:"tags,omitempty"`
	Search       *string  `json:"search,omitempty"`
	IsFavorite   *bool    `json:"is_favorite,omitempty"`
}

type ListQuestionsRequest struct {
	ExamID   string           `json:"exam_id"`
	Filters  *QuestionFilters `json:"filters,omitempty"`
	Page     int              `json:"page,omitempty"`
	PageSize int              `json:"page_size,omitempty"`
}

type QuestionListResult struct {
	Questions []Question `json:"questions"`
	Total     int        `json:"total"`
	Page      int        `json:"page"`
	PageSize  int        `json:"page_size"`
	HasMore   bool       `json:"has_more"`
}

type SearchQuestionsRequest struct {
	Keyword  string           `json:"keyword"`
	ExamID   *string          `json:"exam_id,omitempty"`
	Filters  *QuestionFilters `json:"filters,omitempty"`
	Page     int              `json:"page,omitempty"`
	PageSize int              `json:"page_size,omitempty"`
}

type QuestionSearchResult struct {
	Question             Question `json:"question"`
	HighlightContent     *string  `json:"highlight_content,omitempty"`
	HighlightAnswer      *string  `json:"highlight_answer,omitempty"`
	HighlightExplanation *string  `json:"highlight_explanation,omitempty"`
	RelevanceScore       float64  `json:"relevance_score"`
}

type QuestionSearchListResult struct {
	Results      []QuestionSearchResult `json:"results"`
	Total        int                    `json:"total"`
	Page         int                    `json:"page"`
	PageSize     int                    `json:"page_size"`
	HasMore      bool                   `json:"has_more"`
	SearchTimeMS int                    `json:"search_time_ms"`
}

type NotesMentionMistakeHit struct {
	ID      string   `json:"id"`
	Subject string   `json:"subject"`
	Title   string   `json:"title"`
	Summary *string  `json:"summary,omitempty"`
	Tags    []string `json:"tags"`
}

type NotesMentionIrecCardHit struct {
	ID        string   `json:"id"`
	Title     string   `json:"title"`
	Insight   string   `json:"insight"`
	Subject   *string  `json:"subject,omitempty"`
	Tags      []string `json:"tags"`
	MistakeID *string  `json:"mistake_id,omitempty"`
}

type NotesMentionSearchResult struct {
	Mistakes  []NotesMentionMistakeHit  `json:"mistakes"`
	IrecCards []NotesMentionIrecCardHit `json:"irec_cards"`
}

type CreateQuestionParams struct {
	ExamID        string           `json:"exam_id"`
	Content       string           `json:"content"`
	QuestionType  *string          `json:"question_type,omitempty"`
	Options       []QuestionOption `json:"options,omitempty"`
	Answer        *string          `json:"answer,omitempty"`
	Explanation   *string          `json:"explanation,omitempty"`
	Difficulty    *string          `json:"difficulty,omitempty"`
	Tags          []string         `json:"tags,omitempty"`
	QuestionLabel *string          `json:"question_label,omitempty"`
	CardID        *string          `json:"card_id,omitempty"`
	SourceType    *string          `json:"source_type,omitempty"`
	SourceRef     *string          `json:"source_ref,omitempty"`
	Images        []QuestionImage  `json:"images,omitempty"`
	ParentID      *string          `json:"parent_id,omitempty"`
}

type UpdateQuestionParams struct {
	Content      *string          `json:"content,omitempty"`
	QuestionType *string          `json:"question_type,omitempty"`
	Options      []QuestionOption `json:"options,omitempty"`
	Answer       *string          `json:"answer,omitempty"`
	Explanation  *string          `json:"explanation,omitempty"`
	Difficulty   *string          `json:"difficulty,omitempty"`
	Tags         []string         `json:"tags,omitempty"`
	Status       *string          `json:"status,omitempty"`
	UserAnswer   *string          `json:"user_answer,omitempty"`
	IsCorrect    *bool            `json:"is_correct,omitempty"`
	UserNote     *string          `json:"user_note,omitempty"`
	Images       []QuestionImage  `json:"images,omitempty"`
}

type UpdateQuestionRequest struct {
	QuestionID    string               `json:"question_id"`
	Params        UpdateQuestionParams `json:"params"`
	RecordHistory bool                 `json:"record_history,omitempty"`
}

type SubmitAnswerRequest struct {
	QuestionID        string  `json:"question_id"`
	UserAnswer        string  `json:"user_answer"`
	IsCorrectOverride *bool   `json:"is_correct_override,omitempty"`
	ClientRequestID   *string `json:"client_request_id,omitempty"`
}

type SubmitAnswerResult struct {
	IsCorrect          *bool             `json:"is_correct"`
	CorrectAnswer      *string           `json:"correct_answer,omitempty"`
	NeedsManualGrading bool              `json:"needs_manual_grading"`
	Message            string            `json:"message"`
	UpdatedQuestion    Question          `json:"updated_question"`
	UpdatedStats       QuestionBankStats `json:"updated_stats"`
	SubmissionID       string            `json:"submission_id"`
}

type QbankGradingRequest struct {
	QuestionID      string  `json:"question_id"`
	SubmissionID    string  `json:"submission_id"`
	StreamSessionID string  `json:"stream_session_id"`
	Mode            string  `json:"mode"`
	ModelConfigID   *string `json:"model_config_id,omitempty"`
}

type QbankGradingResponse struct {
	SubmissionID string  `json:"submission_id"`
	Verdict      *string `json:"verdict,omitempty"`
	Score        *int    `json:"score,omitempty"`
	Feedback     string  `json:"feedback"`
}

type APIConfigState struct {
	Configs                     []ApiConfig
	QbankAIGradingModelConfigID string
}

type ApiConfig struct {
	ID              string            `json:"id"`
	Name            string            `json:"name"`
	ApiKey          string            `json:"apiKey"`
	BaseUrl         string            `json:"baseUrl"`
	Model           string            `json:"model"`
	Enabled         bool              `json:"enabled"`
	MaxOutputTokens uint32            `json:"maxOutputTokens,omitempty"`
	Temperature     float32           `json:"temperature,omitempty"`
	Headers         map[string]string `json:"headers,omitempty"`
	IsEmbedding     bool              `json:"isEmbedding,omitempty"`
	IsReranker      bool              `json:"isReranker,omitempty"`
}

type SyncConfig struct {
	DefaultStrategy  string `json:"default_strategy"`
	AutoSync         bool   `json:"auto_sync"`
	SyncIntervalSecs int    `json:"sync_interval_secs"`
	SyncProgress     bool   `json:"sync_progress"`
	SyncNotes        bool   `json:"sync_notes"`
}

type SyncStatusResult struct {
	SyncEnabled          bool        `json:"sync_enabled"`
	LastSyncedAt         *string     `json:"last_synced_at,omitempty"`
	LocalModifiedCount   int         `json:"local_modified_count"`
	PendingConflictCount int         `json:"pending_conflict_count"`
	TotalCount           int         `json:"total_count"`
	SyncedCount          int         `json:"synced_count"`
	SyncConfig           *SyncConfig `json:"sync_config,omitempty"`
}

type QuestionVersion struct {
	ID            string           `json:"id"`
	Content       string           `json:"content"`
	Options       []QuestionOption `json:"options,omitempty"`
	Answer        string           `json:"answer,omitempty"`
	Explanation   string           `json:"explanation,omitempty"`
	QuestionType  string           `json:"question_type"`
	Difficulty    string           `json:"difficulty,omitempty"`
	Tags          []string         `json:"tags"`
	Status        string           `json:"status"`
	UserAnswer    string           `json:"user_answer,omitempty"`
	IsCorrect     *bool            `json:"is_correct,omitempty"`
	AttemptCount  int              `json:"attempt_count"`
	CorrectCount  int              `json:"correct_count"`
	UserNote      string           `json:"user_note,omitempty"`
	IsFavorite    bool             `json:"is_favorite"`
	ContentHash   string           `json:"content_hash"`
	UpdatedAt     string           `json:"updated_at"`
	RemoteVersion int              `json:"remote_version"`
}

type SyncConflict struct {
	ID               string          `json:"id"`
	QuestionID       string          `json:"question_id"`
	ExamID           string          `json:"exam_id"`
	ConflictType     string          `json:"conflict_type"`
	LocalVersion     QuestionVersion `json:"local_version"`
	RemoteVersion    QuestionVersion `json:"remote_version"`
	Status           string          `json:"status"`
	ResolvedStrategy *string         `json:"resolved_strategy,omitempty"`
	ResolvedAt       *string         `json:"resolved_at,omitempty"`
	CreatedAt        string          `json:"created_at"`
}

type QuestionBankStats struct {
	ExamID          string  `json:"exam_id"`
	TotalCount      int     `json:"total_count"`
	NewCount        int     `json:"new_count"`
	InProgressCount int     `json:"in_progress_count"`
	MasteredCount   int     `json:"mastered_count"`
	ReviewCount     int     `json:"review_count"`
	TotalAttempts   int     `json:"total_attempts"`
	TotalCorrect    int     `json:"total_correct"`
	CorrectRate     float64 `json:"correct_rate"`
	UpdatedAt       string  `json:"updated_at"`
}

type AnswerSubmission struct {
	ID              string  `json:"id"`
	QuestionID      string  `json:"question_id"`
	UserAnswer      string  `json:"user_answer"`
	IsCorrect       *bool   `json:"is_correct"`
	ClientRequestID *string `json:"client_request_id,omitempty"`
	CreatedAt       string  `json:"created_at"`
}

type BatchResult struct {
	SuccessCount int      `json:"success_count"`
	FailedCount  int      `json:"failed_count"`
	Errors       []string `json:"errors"`
}

type QuestionHistory struct {
	ID         string  `json:"id"`
	QuestionID string  `json:"question_id"`
	FieldName  string  `json:"field_name"`
	OldValue   *string `json:"old_value,omitempty"`
	NewValue   *string `json:"new_value,omitempty"`
	Operator   string  `json:"operator"`
	Reason     *string `json:"reason,omitempty"`
	CreatedAt  string  `json:"created_at"`
}

type LearningTrendPoint struct {
	Date         string  `json:"date"`
	AttemptCount int     `json:"attempt_count"`
	CorrectCount int     `json:"correct_count"`
	CorrectRate  float64 `json:"correct_rate"`
}

type ActivityHeatmapPoint struct {
	Date         string `json:"date"`
	Count        int    `json:"count"`
	CorrectCount int    `json:"correct_count"`
	Level        int    `json:"level"`
}

type KnowledgePoint struct {
	Tag         string  `json:"tag"`
	Total       int     `json:"total"`
	Mastered    int     `json:"mastered"`
	InProgress  int     `json:"in_progress"`
	Review      int     `json:"review"`
	NewCount    int     `json:"new_count"`
	MasteryRate float64 `json:"mastery_rate"`
	CorrectRate float64 `json:"correct_rate"`
}

type KnowledgeStatsComparison struct {
	Current  []KnowledgePoint `json:"current"`
	Previous []KnowledgePoint `json:"previous"`
}

type GetLearningTrendRequest struct {
	ExamID    *string `json:"exam_id,omitempty"`
	StartDate string  `json:"start_date"`
	EndDate   string  `json:"end_date"`
}

type GetActivityHeatmapRequest struct {
	ExamID *string `json:"exam_id,omitempty"`
	Year   int     `json:"year"`
}

type GetKnowledgeStatsRequest struct {
	ExamID *string `json:"exam_id,omitempty"`
}

type StartTimedPracticeRequest struct {
	ExamID          string `json:"exam_id"`
	DurationMinutes int    `json:"duration_minutes"`
	QuestionCount   int    `json:"question_count"`
}

type TimedPracticeSession struct {
	ID              string   `json:"id"`
	ExamID          string   `json:"exam_id"`
	DurationMinutes int      `json:"duration_minutes"`
	QuestionCount   int      `json:"question_count"`
	QuestionIDs     []string `json:"question_ids"`
	StartedAt       string   `json:"started_at"`
	EndedAt         *string  `json:"ended_at,omitempty"`
	AnsweredCount   int      `json:"answered_count"`
	CorrectCount    int      `json:"correct_count"`
	IsTimeout       bool     `json:"is_timeout"`
	IsSubmitted     bool     `json:"is_submitted"`
	PausedSeconds   int      `json:"paused_seconds"`
	IsPaused        bool     `json:"is_paused"`
}

type MockExamConfig struct {
	DurationMinutes        int            `json:"duration_minutes"`
	TypeDistribution       map[string]int `json:"type_distribution"`
	DifficultyDistribution map[string]int `json:"difficulty_distribution"`
	TotalCount             *int           `json:"total_count,omitempty"`
	Shuffle                bool           `json:"shuffle"`
	IncludeMistakes        bool           `json:"include_mistakes"`
	Tags                   []string       `json:"tags,omitempty"`
}

type GenerateMockExamRequest struct {
	ExamID string         `json:"exam_id"`
	Config MockExamConfig `json:"config"`
}

type MockExamSession struct {
	ID          string            `json:"id"`
	ExamID      string            `json:"exam_id"`
	Config      MockExamConfig    `json:"config"`
	QuestionIDs []string          `json:"question_ids"`
	StartedAt   string            `json:"started_at"`
	EndedAt     *string           `json:"ended_at,omitempty"`
	Answers     map[string]string `json:"answers"`
	Results     map[string]bool   `json:"results"`
	IsSubmitted bool              `json:"is_submitted"`
	Score       *float64          `json:"score,omitempty"`
	CorrectRate *float64          `json:"correct_rate,omitempty"`
}

type SubmitMockExamRequest struct {
	Session MockExamSession `json:"session"`
}

type TypeStatItem struct {
	Total   int     `json:"total"`
	Correct int     `json:"correct"`
	Rate    float64 `json:"rate"`
}

type DifficultyStatItem struct {
	Total   int     `json:"total"`
	Correct int     `json:"correct"`
	Rate    float64 `json:"rate"`
}

type MockExamScoreCard struct {
	SessionID        string                        `json:"session_id"`
	ExamID           string                        `json:"exam_id"`
	TotalCount       int                           `json:"total_count"`
	AnsweredCount    int                           `json:"answered_count"`
	CorrectCount     int                           `json:"correct_count"`
	WrongCount       int                           `json:"wrong_count"`
	UnansweredCount  int                           `json:"unanswered_count"`
	CorrectRate      float64                       `json:"correct_rate"`
	TimeSpentSeconds int                           `json:"time_spent_seconds"`
	TypeStats        map[string]TypeStatItem       `json:"type_stats"`
	DifficultyStats  map[string]DifficultyStatItem `json:"difficulty_stats"`
	WrongQuestionIDs []string                      `json:"wrong_question_ids"`
	Comment          string                        `json:"comment"`
	CompletedAt      string                        `json:"completed_at"`
}

type GetDailyPracticeRequest struct {
	ExamID string `json:"exam_id"`
	Count  int    `json:"count"`
}

type DailyPracticeResult struct {
	Date               string                  `json:"date"`
	ExamID             string                  `json:"exam_id"`
	QuestionIDs        []string                `json:"question_ids"`
	DailyTarget        int                     `json:"daily_target"`
	CompletedCount     int                     `json:"completed_count"`
	CorrectCount       int                     `json:"correct_count"`
	SourceDistribution DailySourceDistribution `json:"source_distribution"`
	IsCompleted        bool                    `json:"is_completed"`
}

type DailySourceDistribution struct {
	MistakeCount int `json:"mistake_count"`
	NewCount     int `json:"new_count"`
	ReviewCount  int `json:"review_count"`
}

type PaperConfig struct {
	Title               string         `json:"title"`
	TypeSelection       map[string]int `json:"type_selection"`
	DifficultyFilter    []string       `json:"difficulty_filter,omitempty"`
	TagsFilter          []string       `json:"tags_filter,omitempty"`
	Shuffle             bool           `json:"shuffle"`
	IncludeAnswers      bool           `json:"include_answers"`
	IncludeExplanations bool           `json:"include_explanations"`
	ExportFormat        string         `json:"export_format"`
}

type GeneratePaperRequest struct {
	ExamID string      `json:"exam_id"`
	Config PaperConfig `json:"config"`
}

type GeneratedPaper struct {
	ID         string      `json:"id"`
	Title      string      `json:"title"`
	ExamID     string      `json:"exam_id"`
	Questions  []Question  `json:"questions"`
	TotalScore int         `json:"total_score"`
	Config     PaperConfig `json:"config"`
	CreatedAt  string      `json:"created_at"`
	ExportPath *string     `json:"export_path,omitempty"`
}

type GetCheckInCalendarRequest struct {
	ExamID *string `json:"exam_id,omitempty"`
	Year   int     `json:"year"`
	Month  int     `json:"month"`
}

type DailyCheckIn struct {
	Date                 string  `json:"date"`
	ExamID               *string `json:"exam_id,omitempty"`
	QuestionCount        int     `json:"question_count"`
	CorrectCount         int     `json:"correct_count"`
	StudyDurationSeconds int     `json:"study_duration_seconds"`
	TargetAchieved       bool    `json:"target_achieved"`
}

type CheckInCalendar struct {
	Year                int            `json:"year"`
	Month               int            `json:"month"`
	Days                []DailyCheckIn `json:"days"`
	StreakDays          int            `json:"streak_days"`
	MonthCheckInDays    int            `json:"month_check_in_days"`
	MonthTotalQuestions int            `json:"month_total_questions"`
}

func NewService(dataDir string, vfsServices ...*vfs.Service) (*Service, error) {
	service := &Service{
		httpClient: &http.Client{Timeout: 60 * time.Second},
		path:       filepath.Join(dataDir, "qbank-go.json"),
		state: store{
			Questions:   []Question{},
			Submissions: []AnswerSubmission{},
			Histories:   []QuestionHistory{},
			SyncConfigs: map[string]SyncConfig{},
		},
	}
	if len(vfsServices) > 0 {
		service.vfs = vfsServices[0]
	}
	if err := service.load(); err != nil {
		return nil, err
	}
	return service, nil
}

func (s *Service) SetEventEmitter(emit func(name string, payload any)) {
	s.emit = emit
}

func (s *Service) SetAPIConfigLoader(loader func() (APIConfigState, error)) {
	s.configMu.Lock()
	defer s.configMu.Unlock()
	s.apiConfigLoader = loader
}

func (s *Service) SetHTTPClient(client *http.Client) {
	if client == nil {
		return
	}
	s.configMu.Lock()
	defer s.configMu.Unlock()
	s.httpClient = client
}

func (s *Service) ListQuestions(request ListQuestionsRequest) (QuestionListResult, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	page, pageSize := normalizePage(request.Page, request.PageSize)
	filtered := make([]Question, 0)
	for _, question := range s.state.Questions {
		if question.ExamID != request.ExamID {
			continue
		}
		if !matchesFilters(question, request.Filters) {
			continue
		}
		filtered = append(filtered, question)
	}
	sortQuestions(filtered)
	return pageQuestions(filtered, page, pageSize), nil
}

func (s *Service) SearchQuestions(request SearchQuestionsRequest) (QuestionSearchListResult, error) {
	started := time.Now()
	page, pageSize := normalizePage(request.Page, request.PageSize)
	query := strings.ToLower(strings.TrimSpace(request.Keyword))

	s.mu.RLock()
	defer s.mu.RUnlock()

	results := make([]QuestionSearchResult, 0)
	for _, question := range s.state.Questions {
		if request.ExamID != nil && question.ExamID != *request.ExamID {
			continue
		}
		if !matchesFilters(question, request.Filters) {
			continue
		}
		haystack := strings.ToLower(strings.Join([]string{
			question.Content,
			question.Answer,
			question.Explanation,
			strings.Join(question.Tags, " "),
		}, "\n"))
		if query != "" && !strings.Contains(haystack, query) {
			continue
		}
		highlight := question.Content
		results = append(results, QuestionSearchResult{
			Question:         question,
			HighlightContent: &highlight,
			RelevanceScore:   0,
		})
	}
	sort.SliceStable(results, func(a, b int) bool {
		return results[a].Question.UpdatedAt > results[b].Question.UpdatedAt
	})
	total := len(results)
	offset := (page - 1) * pageSize
	if offset >= total {
		results = []QuestionSearchResult{}
	} else {
		end := offset + pageSize
		if end > total {
			end = total
		}
		results = results[offset:end]
	}
	return QuestionSearchListResult{
		Results:      results,
		Total:        total,
		Page:         page,
		PageSize:     pageSize,
		HasMore:      page*pageSize < total,
		SearchTimeMS: int(time.Since(started).Milliseconds()),
	}, nil
}

func (s *Service) NotesMentionsSearch(keyword string, subject *string, limit int) (NotesMentionSearchResult, error) {
	query := strings.TrimSpace(keyword)
	if query == "" {
		return NotesMentionSearchResult{
			Mistakes:  []NotesMentionMistakeHit{},
			IrecCards: []NotesMentionIrecCardHit{},
		}, nil
	}

	maxResults := normalizeMentionLimit(limit)
	var subjectFilter *string
	if subject != nil {
		trimmedSubject := strings.TrimSpace(*subject)
		if trimmedSubject != "" && trimmedSubject != "_global" {
			subjectFilter = &trimmedSubject
		}
	}
	terms := strings.Fields(strings.ToLower(query))

	s.mu.RLock()
	defer s.mu.RUnlock()

	type mentionCandidate struct {
		question Question
		score    int
	}
	candidates := make([]mentionCandidate, 0)
	for _, question := range s.state.Questions {
		if !matchesMentionSubject(question, subjectFilter) {
			continue
		}
		score := scoreMentionQuestion(question, strings.ToLower(query), terms)
		if score <= 0 {
			continue
		}
		candidates = append(candidates, mentionCandidate{question: question, score: score})
	}

	sort.SliceStable(candidates, func(a, b int) bool {
		if candidates[a].score != candidates[b].score {
			return candidates[a].score > candidates[b].score
		}
		return candidates[a].question.UpdatedAt > candidates[b].question.UpdatedAt
	})

	if len(candidates) > maxResults {
		candidates = candidates[:maxResults]
	}

	result := NotesMentionSearchResult{
		Mistakes:  make([]NotesMentionMistakeHit, 0, len(candidates)),
		IrecCards: make([]NotesMentionIrecCardHit, 0, len(candidates)),
	}
	for _, candidate := range candidates {
		question := candidate.question
		title := questionMentionTitle(question)
		insight := questionMentionInsight(question, query)
		mistakeID := question.ID
		cardID := question.ID
		if question.CardID != nil && strings.TrimSpace(*question.CardID) != "" {
			cardID = strings.TrimSpace(*question.CardID)
		}
		subjectValue := question.ExamID
		result.Mistakes = append(result.Mistakes, NotesMentionMistakeHit{
			ID:      question.ID,
			Subject: question.ExamID,
			Title:   title,
			Summary: &insight,
			Tags:    append([]string(nil), question.Tags...),
		})
		result.IrecCards = append(result.IrecCards, NotesMentionIrecCardHit{
			ID:        cardID,
			Title:     title,
			Insight:   insight,
			Subject:   &subjectValue,
			Tags:      append([]string(nil), question.Tags...),
			MistakeID: &mistakeID,
		})
	}
	return result, nil
}

func (s *Service) RebuildFTSIndex() int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return len(s.state.Questions)
}

func (s *Service) GetQuestion(questionID string) (*Question, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	question, ok := s.findQuestionLocked(questionID)
	if !ok {
		return nil, nil
	}
	return &question, nil
}

func (s *Service) CreateQuestion(params CreateQuestionParams) (Question, error) {
	if strings.TrimSpace(params.ExamID) == "" {
		return Question{}, errors.New("exam_id is required")
	}
	if strings.TrimSpace(params.Content) == "" {
		return Question{}, errors.New("content is required")
	}
	now := nowISO()
	questionType := valueOr(params.QuestionType, "other")
	difficulty := valueOr(params.Difficulty, "medium")
	sourceType := valueOr(params.SourceType, "manual")
	question := Question{
		ID:            "q_" + randomToken(16),
		ExamID:        params.ExamID,
		CardID:        params.CardID,
		QuestionLabel: params.QuestionLabel,
		Content:       params.Content,
		Options:       nonNilOptions(params.Options),
		Answer:        valueOr(params.Answer, ""),
		Explanation:   valueOr(params.Explanation, ""),
		QuestionType:  questionType,
		Difficulty:    difficulty,
		Tags:          nonNilStrings(params.Tags),
		Status:        "new",
		Images:        nonNilImages(params.Images),
		SourceType:    sourceType,
		SourceRef:     params.SourceRef,
		ParentID:      params.ParentID,
		CreatedAt:     now,
		UpdatedAt:     now,
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	s.state.Questions = append(s.state.Questions, question)
	index := len(s.state.Questions) - 1
	if err := s.syncQuestionResourceLocked(index); err != nil {
		s.state.Questions = s.state.Questions[:index]
		return Question{}, err
	}
	if err := s.flushLocked(); err != nil {
		return Question{}, err
	}
	return s.state.Questions[index], nil
}

func (s *Service) UpdateQuestion(request UpdateQuestionRequest) (Question, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	index, ok := s.findQuestionIndexLocked(request.QuestionID)
	if !ok {
		return Question{}, fmt.Errorf("question not found: %s", request.QuestionID)
	}
	previous := s.state.Questions[index]
	question := previous
	if request.Params.Content != nil {
		question.Content = *request.Params.Content
	}
	if request.Params.QuestionType != nil {
		question.QuestionType = *request.Params.QuestionType
	}
	if request.Params.Options != nil {
		question.Options = nonNilOptions(request.Params.Options)
	}
	if request.Params.Answer != nil {
		question.Answer = *request.Params.Answer
	}
	if request.Params.Explanation != nil {
		question.Explanation = *request.Params.Explanation
	}
	if request.Params.Difficulty != nil {
		question.Difficulty = *request.Params.Difficulty
	}
	if request.Params.Tags != nil {
		question.Tags = nonNilStrings(request.Params.Tags)
	}
	if request.Params.Status != nil {
		question.Status = *request.Params.Status
	}
	if request.Params.UserAnswer != nil {
		question.UserAnswer = *request.Params.UserAnswer
	}
	if request.Params.IsCorrect != nil {
		question.IsCorrect = request.Params.IsCorrect
	}
	if request.Params.UserNote != nil {
		question.UserNote = *request.Params.UserNote
	}
	if request.Params.Images != nil {
		question.Images = nonNilImages(request.Params.Images)
	}
	question.UpdatedAt = nowISO()
	s.state.Questions[index] = question
	if err := s.syncQuestionResourceLocked(index); err != nil {
		s.state.Questions[index] = previous
		return Question{}, err
	}
	if request.RecordHistory {
		s.recordQuestionChangesLocked(previous, s.state.Questions[index], "user")
	}
	if err := s.flushLocked(); err != nil {
		return Question{}, err
	}
	return s.state.Questions[index], nil
}

func (s *Service) DeleteQuestion(questionID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	index, ok := s.findQuestionIndexLocked(questionID)
	if !ok {
		return nil
	}
	s.state.Questions = append(s.state.Questions[:index], s.state.Questions[index+1:]...)
	s.state.Submissions = filterSubmissions(s.state.Submissions, questionID)
	return s.flushLocked()
}

func (s *Service) BatchDeleteQuestions(questionIDs []string) (BatchResult, error) {
	result := BatchResult{Errors: []string{}}
	for _, questionID := range questionIDs {
		if err := s.DeleteQuestion(questionID); err != nil {
			result.FailedCount++
			result.Errors = append(result.Errors, err.Error())
		} else {
			result.SuccessCount++
		}
	}
	return result, nil
}

func (s *Service) SubmitAnswer(request SubmitAnswerRequest) (SubmitAnswerResult, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	index, ok := s.findQuestionIndexLocked(request.QuestionID)
	if !ok {
		return SubmitAnswerResult{}, fmt.Errorf("question not found: %s", request.QuestionID)
	}
	previous := s.state.Questions[index]
	question := s.state.Questions[index]
	isCorrect := request.IsCorrectOverride
	needsManual := false
	if isCorrect == nil {
		if isSubjective(question.QuestionType) {
			needsManual = true
		} else {
			calculated := normalizedAnswer(request.UserAnswer) == normalizedAnswer(question.Answer)
			isCorrect = &calculated
		}
	}

	now := nowISO()
	question.UserAnswer = request.UserAnswer
	question.IsCorrect = isCorrect
	question.AttemptCount++
	if isCorrect != nil && *isCorrect {
		question.CorrectCount++
	}
	question.LastAttemptAt = &now
	question.Status = nextStatus(question.AttemptCount, question.CorrectCount, isCorrect)
	question.UpdatedAt = now
	s.state.Questions[index] = question

	submissionID := "sub_" + randomToken(16)
	previousSubmissionCount := len(s.state.Submissions)
	s.state.Submissions = append(s.state.Submissions, AnswerSubmission{
		ID:              submissionID,
		QuestionID:      question.ID,
		UserAnswer:      request.UserAnswer,
		IsCorrect:       isCorrect,
		ClientRequestID: request.ClientRequestID,
		CreatedAt:       now,
	})
	if err := s.syncQuestionResourceLocked(index); err != nil {
		s.state.Questions[index] = previous
		s.state.Submissions = s.state.Submissions[:previousSubmissionCount]
		return SubmitAnswerResult{}, err
	}
	question = s.state.Questions[index]
	s.recordQuestionChangesLocked(previous, question, "answer")
	stats := calculateStats(question.ExamID, s.state.Questions)
	if err := s.flushLocked(); err != nil {
		return SubmitAnswerResult{}, err
	}

	correctAnswer := question.Answer
	message := "Answer submitted"
	if needsManual {
		message = "Subjective answer submitted"
	} else if isCorrect != nil && *isCorrect {
		message = "Correct"
	} else {
		message = "Incorrect"
	}
	return SubmitAnswerResult{
		IsCorrect:          isCorrect,
		CorrectAnswer:      &correctAnswer,
		NeedsManualGrading: needsManual,
		Message:            message,
		UpdatedQuestion:    question,
		UpdatedStats:       stats,
		SubmissionID:       submissionID,
	}, nil
}

func (s *Service) AIGrade(request QbankGradingRequest) (*QbankGradingResponse, error) {
	request.QuestionID = strings.TrimSpace(request.QuestionID)
	request.SubmissionID = strings.TrimSpace(request.SubmissionID)
	request.StreamSessionID = strings.TrimSpace(request.StreamSessionID)
	request.Mode = strings.TrimSpace(request.Mode)
	if request.Mode == "" {
		request.Mode = "grade"
	}
	if request.QuestionID == "" || request.SubmissionID == "" || request.StreamSessionID == "" {
		err := errors.New("question_id, submission_id, and stream_session_id are required")
		s.emitQbankGradingError(request.StreamSessionID, err.Error())
		return nil, err
	}
	if request.Mode != "grade" && request.Mode != "analyze" {
		err := fmt.Errorf("unsupported qbank grading mode: %s", request.Mode)
		s.emitQbankGradingError(request.StreamSessionID, err.Error())
		return nil, err
	}

	s.mu.RLock()
	index, ok := s.findQuestionIndexLocked(request.QuestionID)
	if !ok {
		s.mu.RUnlock()
		err := fmt.Errorf("question not found: %s", request.QuestionID)
		s.emitQbankGradingError(request.StreamSessionID, err.Error())
		return nil, err
	}
	submissionIndex, ok := s.findSubmissionIndexLocked(request.SubmissionID)
	if !ok {
		s.mu.RUnlock()
		err := fmt.Errorf("answer submission not found: %s", request.SubmissionID)
		s.emitQbankGradingError(request.StreamSessionID, err.Error())
		return nil, err
	}
	if s.state.Submissions[submissionIndex].QuestionID != request.QuestionID {
		s.mu.RUnlock()
		err := fmt.Errorf("answer submission %s does not belong to question %s", request.SubmissionID, request.QuestionID)
		s.emitQbankGradingError(request.StreamSessionID, err.Error())
		return nil, err
	}

	questionSnapshot := s.state.Questions[index]
	submissionSnapshot := s.state.Submissions[submissionIndex]
	s.mu.RUnlock()

	feedback, verdict, score := s.buildGradingFeedback(questionSnapshot, submissionSnapshot, request)
	now := nowISO()

	s.mu.Lock()
	defer s.mu.Unlock()

	index, ok = s.findQuestionIndexLocked(request.QuestionID)
	if !ok {
		err := fmt.Errorf("question not found: %s", request.QuestionID)
		s.emitQbankGradingError(request.StreamSessionID, err.Error())
		return nil, err
	}
	submissionIndex, ok = s.findSubmissionIndexLocked(request.SubmissionID)
	if !ok {
		err := fmt.Errorf("answer submission not found: %s", request.SubmissionID)
		s.emitQbankGradingError(request.StreamSessionID, err.Error())
		return nil, err
	}
	if s.state.Submissions[submissionIndex].QuestionID != request.QuestionID {
		err := fmt.Errorf("answer submission %s does not belong to question %s", request.SubmissionID, request.QuestionID)
		s.emitQbankGradingError(request.StreamSessionID, err.Error())
		return nil, err
	}

	previous := s.state.Questions[index]
	previousSubmission := s.state.Submissions[submissionIndex]
	question := s.state.Questions[index]

	question.AIFeedback = &feedback
	if score != nil {
		scoreFloat := float64(*score)
		question.AIScore = &scoreFloat
	} else {
		question.AIScore = nil
	}
	question.AIGradedAt = &now
	question.UpdatedAt = now

	if request.Mode == "grade" {
		isCorrect := verdict != nil && *verdict == "correct"
		wasUngraded := question.IsCorrect == nil
		question.IsCorrect = &isCorrect
		s.state.Submissions[submissionIndex].IsCorrect = &isCorrect
		if wasUngraded && isCorrect {
			question.CorrectCount++
		}
		question.Status = nextStatus(question.AttemptCount, question.CorrectCount, question.IsCorrect)
	}

	s.state.Questions[index] = question
	if err := s.syncQuestionResourceLocked(index); err != nil {
		s.state.Questions[index] = previous
		s.state.Submissions[submissionIndex] = previousSubmission
		s.emitQbankGradingError(request.StreamSessionID, err.Error())
		return nil, err
	}
	question = s.state.Questions[index]
	s.recordQuestionChangesLocked(previous, question, "ai_grading")
	if err := s.flushLocked(); err != nil {
		s.state.Questions[index] = previous
		s.state.Submissions[submissionIndex] = previousSubmission
		s.emitQbankGradingError(request.StreamSessionID, err.Error())
		return nil, err
	}

	s.emitQbankGradingData(request.StreamSessionID, feedback, feedback)
	s.emitQbankGradingComplete(request.StreamSessionID, request.SubmissionID, verdict, score, feedback)
	return &QbankGradingResponse{
		SubmissionID: request.SubmissionID,
		Verdict:      verdict,
		Score:        score,
		Feedback:     feedback,
	}, nil
}

func (s *Service) CancelGrading(streamEventName string) error {
	streamEventName = strings.TrimSpace(streamEventName)
	if streamEventName == "" {
		return errors.New("streamEventName is required")
	}
	s.emitEvent(streamEventName, map[string]any{"type": "cancelled"})
	return nil
}

func (s *Service) CheckSyncStatus(examID string) (SyncStatusResult, error) {
	examID = strings.TrimSpace(examID)
	if examID == "" {
		return SyncStatusResult{}, errors.New("examId is required")
	}
	s.mu.RLock()
	defer s.mu.RUnlock()

	total := 0
	for _, question := range s.state.Questions {
		if question.ExamID == examID {
			total++
		}
	}
	config := s.syncConfigLocked(examID)
	return SyncStatusResult{
		SyncEnabled:          config.AutoSync,
		LocalModifiedCount:   0,
		PendingConflictCount: 0,
		TotalCount:           total,
		SyncedCount:          total,
		SyncConfig:           &config,
	}, nil
}

func (s *Service) GetSyncConflicts(examID string) ([]SyncConflict, error) {
	examID = strings.TrimSpace(examID)
	if examID == "" {
		return nil, errors.New("examId is required")
	}
	return []SyncConflict{}, nil
}

func (s *Service) ResolveSyncConflict(conflictID string, strategy string) (Question, error) {
	conflictID = strings.TrimSpace(conflictID)
	strategy = strings.TrimSpace(strategy)
	if conflictID == "" {
		return Question{}, errors.New("conflictId is required")
	}
	if strategy == "" {
		return Question{}, errors.New("strategy is required")
	}
	return Question{}, fmt.Errorf("sync conflict not found in local Go qbank store: %s", conflictID)
}

func (s *Service) BatchResolveConflicts(examID string, strategy string) ([]Question, error) {
	examID = strings.TrimSpace(examID)
	strategy = strings.TrimSpace(strategy)
	if examID == "" {
		return nil, errors.New("examId is required")
	}
	if strategy == "" {
		return nil, errors.New("strategy is required")
	}
	return []Question{}, nil
}

func (s *Service) SetSyncEnabled(examID string, enabled bool) error {
	examID = strings.TrimSpace(examID)
	if examID == "" {
		return errors.New("examId is required")
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.state.SyncConfigs == nil {
		s.state.SyncConfigs = map[string]SyncConfig{}
	}
	config := s.syncConfigLocked(examID)
	config.AutoSync = enabled
	s.state.SyncConfigs[examID] = config
	return s.flushLocked()
}

func (s *Service) UpdateSyncConfig(examID string, config SyncConfig) error {
	examID = strings.TrimSpace(examID)
	if examID == "" {
		return errors.New("examId is required")
	}
	config = normalizeSyncConfig(config)
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.state.SyncConfigs == nil {
		s.state.SyncConfigs = map[string]SyncConfig{}
	}
	s.state.SyncConfigs[examID] = config
	return s.flushLocked()
}

func (s *Service) syncConfigLocked(examID string) SyncConfig {
	if s.state.SyncConfigs != nil {
		if config, ok := s.state.SyncConfigs[examID]; ok {
			return normalizeSyncConfig(config)
		}
	}
	return defaultSyncConfig()
}

func (s *Service) emitQbankGradingData(streamSessionID string, chunk string, accumulated string) {
	if strings.TrimSpace(streamSessionID) == "" {
		return
	}
	s.emitEvent("qbank_grading_stream_"+streamSessionID, map[string]any{
		"type":        "data",
		"chunk":       chunk,
		"accumulated": accumulated,
	})
}

func (s *Service) emitQbankGradingComplete(streamSessionID string, submissionID string, verdict *string, score *int, feedback string) {
	if strings.TrimSpace(streamSessionID) == "" {
		return
	}
	s.emitEvent("qbank_grading_stream_"+streamSessionID, map[string]any{
		"type":          "complete",
		"submission_id": submissionID,
		"verdict":       verdict,
		"score":         score,
		"feedback":      feedback,
	})
}

func (s *Service) emitQbankGradingError(streamSessionID string, message string) {
	if strings.TrimSpace(streamSessionID) == "" {
		return
	}
	s.emitEvent("qbank_grading_stream_"+streamSessionID, map[string]any{
		"type":    "error",
		"message": message,
	})
}

func (s *Service) emitEvent(name string, payload any) {
	if s.emit != nil {
		s.emit(name, payload)
	}
}

func (s *Service) currentAPIConfigLoader() func() (APIConfigState, error) {
	s.configMu.RLock()
	defer s.configMu.RUnlock()
	return s.apiConfigLoader
}

func (s *Service) currentHTTPClient() *http.Client {
	s.configMu.RLock()
	defer s.configMu.RUnlock()
	if s.httpClient == nil {
		return http.DefaultClient
	}
	return s.httpClient
}

func (s *Service) buildGradingFeedback(question Question, submission AnswerSubmission, request QbankGradingRequest) (string, *string, *int) {
	if result, err := s.generateProviderGrading(question, submission, request); err == nil && strings.TrimSpace(result.Feedback) != "" {
		return result.Feedback, result.Verdict, result.Score
	}
	return buildLeanGradingFeedback(question, submission, request.Mode)
}

type providerGradingResult struct {
	Feedback string
	Verdict  *string
	Score    *int
}

func (s *Service) generateProviderGrading(question Question, submission AnswerSubmission, request QbankGradingRequest) (providerGradingResult, error) {
	config, ok, err := s.resolveQbankAPIConfig(request)
	if err != nil || !ok {
		return providerGradingResult{}, err
	}
	ctx, cancel := context.WithTimeout(context.Background(), 45*time.Second)
	defer cancel()

	content, err := s.requestProviderGrading(ctx, config, question, submission, request.Mode)
	if err != nil {
		return providerGradingResult{}, err
	}
	result := parseProviderGrading(content, request.Mode)
	if strings.TrimSpace(result.Feedback) == "" {
		return providerGradingResult{}, errors.New("provider returned no feedback")
	}
	if request.Mode == "grade" && result.Verdict == nil && result.Score == nil {
		return providerGradingResult{}, errors.New("provider returned no grade verdict or score")
	}
	return result, nil
}

func (s *Service) resolveQbankAPIConfig(request QbankGradingRequest) (ApiConfig, bool, error) {
	loader := s.currentAPIConfigLoader()
	if loader == nil {
		return ApiConfig{}, false, nil
	}
	state, err := loader()
	if err != nil {
		return ApiConfig{}, false, err
	}
	for _, preferredID := range qbankPreferredConfigIDs(request, state.QbankAIGradingModelConfigID) {
		for _, config := range state.Configs {
			if config.ID == preferredID && usableAPIConfig(config) {
				return config, true, nil
			}
		}
	}
	for _, config := range state.Configs {
		if usableAPIConfig(config) {
			return config, true, nil
		}
	}
	return ApiConfig{}, false, nil
}

func (s *Service) requestProviderGrading(ctx context.Context, config ApiConfig, question Question, submission AnswerSubmission, mode string) (string, error) {
	baseURL := strings.TrimRight(strings.TrimSpace(config.BaseUrl), "/")
	if baseURL == "" {
		return "", errors.New("api baseUrl is required")
	}
	endpoint := baseURL
	if !strings.HasSuffix(endpoint, "/chat/completions") {
		endpoint += "/chat/completions"
	}

	maxTokens := int(config.MaxOutputTokens)
	if maxTokens <= 0 {
		maxTokens = 1200
	}
	body := map[string]any{
		"model":      strings.TrimSpace(config.Model),
		"stream":     false,
		"max_tokens": maxTokens,
		"messages": []map[string]string{
			{
				"role":    "system",
				"content": "You grade student answers for a question bank. Return JSON only. For grade mode return {\"verdict\":\"correct|partial|incorrect\",\"score\":0-100,\"feedback\":\"markdown feedback\"}. For analyze mode return {\"feedback\":\"markdown analysis\"}. Do not include markdown fences.",
			},
			{
				"role":    "user",
				"content": qbankProviderPrompt(question, submission, mode),
			},
		},
	}
	if config.Temperature > 0 {
		body["temperature"] = config.Temperature
	} else {
		body["temperature"] = 0.2
	}

	requestBody, err := json.Marshal(body)
	if err != nil {
		return "", err
	}
	httpRequest, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(requestBody))
	if err != nil {
		return "", err
	}
	httpRequest.Header.Set("Content-Type", "application/json")
	httpRequest.Header.Set("Authorization", "Bearer "+strings.TrimSpace(config.ApiKey))
	for key, value := range config.Headers {
		key = strings.TrimSpace(key)
		if key == "" {
			continue
		}
		httpRequest.Header.Set(key, value)
	}

	response, err := s.currentHTTPClient().Do(httpRequest)
	if err != nil {
		return "", err
	}
	defer response.Body.Close()
	responseBody, err := io.ReadAll(io.LimitReader(response.Body, 2*1024*1024))
	if err != nil {
		return "", err
	}
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		return "", fmt.Errorf("provider returned %s: %s", response.Status, strings.TrimSpace(string(responseBody)))
	}

	var decoded struct {
		Choices []struct {
			Message struct {
				Content any `json:"content"`
			} `json:"message"`
		} `json:"choices"`
		Error *struct {
			Message string `json:"message"`
		} `json:"error,omitempty"`
	}
	if err := json.Unmarshal(responseBody, &decoded); err != nil {
		return "", err
	}
	if decoded.Error != nil && strings.TrimSpace(decoded.Error.Message) != "" {
		return "", errors.New(strings.TrimSpace(decoded.Error.Message))
	}
	if len(decoded.Choices) == 0 {
		return "", errors.New("provider returned no choices")
	}
	content := providerTextContent(decoded.Choices[0].Message.Content)
	if strings.TrimSpace(content) == "" {
		return "", errors.New("provider returned empty content")
	}
	return content, nil
}

func qbankProviderPrompt(question Question, submission AnswerSubmission, mode string) string {
	return strings.Join([]string{
		"Mode: " + mode,
		"Question type: " + strings.TrimSpace(question.QuestionType),
		"Question:",
		truncateForProvider(question.Content, 4000),
		"Student answer:",
		truncateForProvider(submission.UserAnswer, 3000),
		"Reference answer:",
		truncateForProvider(question.Answer, 3000),
		"Reference explanation:",
		truncateForProvider(question.Explanation, 3000),
		"Tags: " + strings.Join(nonNilStrings(question.Tags), ", "),
	}, "\n\n")
}

func parseProviderGrading(content string, mode string) providerGradingResult {
	var payload map[string]any
	if err := json.Unmarshal([]byte(extractProviderJSON(content)), &payload); err != nil {
		return providerGradingResult{}
	}
	feedback := firstProviderString(payload, "feedback", "Feedback", "analysis", "Analysis", "comment", "Comment")
	if feedback == "" {
		feedback = firstProviderString(payload, "message", "Message", "text", "Text")
	}
	if mode == "analyze" {
		return providerGradingResult{Feedback: feedback}
	}

	verdict := normalizeProviderVerdict(firstProviderString(payload, "verdict", "Verdict", "result", "Result", "grade", "Grade"))
	score := providerScore(payload)
	if verdict == nil && score != nil {
		verdict = verdictFromScore(*score)
	}
	if verdict != nil && score == nil {
		score = scoreFromVerdict(*verdict)
	}
	return providerGradingResult{
		Feedback: feedback,
		Verdict:  verdict,
		Score:    score,
	}
}

func usableAPIConfig(config ApiConfig) bool {
	return config.Enabled &&
		strings.TrimSpace(config.ID) != "" &&
		strings.TrimSpace(config.ApiKey) != "" &&
		strings.TrimSpace(config.BaseUrl) != "" &&
		strings.TrimSpace(config.Model) != "" &&
		!config.IsEmbedding &&
		!config.IsReranker
}

func qbankPreferredConfigIDs(request QbankGradingRequest, assignmentID string) []string {
	values := []string{strings.TrimSpace(valueOrEmpty(request.ModelConfigID)), strings.TrimSpace(assignmentID)}
	out := []string{}
	seen := map[string]bool{}
	for _, value := range values {
		if value == "" || seen[value] {
			continue
		}
		seen[value] = true
		out = append(out, value)
	}
	return out
}

func providerTextContent(content any) string {
	switch typed := content.(type) {
	case string:
		return strings.TrimSpace(typed)
	case []any:
		parts := []string{}
		for _, item := range typed {
			if object, ok := item.(map[string]any); ok {
				text := firstProviderString(object, "text", "content")
				if text != "" {
					parts = append(parts, text)
				}
			}
		}
		return strings.TrimSpace(strings.Join(parts, ""))
	case map[string]any:
		return firstProviderString(typed, "text", "content")
	default:
		return ""
	}
}

func extractProviderJSON(content string) string {
	cleaned := strings.TrimSpace(content)
	if strings.HasPrefix(cleaned, "```") {
		lines := strings.Split(cleaned, "\n")
		if len(lines) > 1 {
			if strings.HasPrefix(strings.TrimSpace(lines[0]), "```") {
				lines = lines[1:]
			}
			if len(lines) > 0 && strings.HasPrefix(strings.TrimSpace(lines[len(lines)-1]), "```") {
				lines = lines[:len(lines)-1]
			}
			cleaned = strings.TrimSpace(strings.Join(lines, "\n"))
		}
	}
	if json.Valid([]byte(cleaned)) {
		return cleaned
	}
	start := strings.IndexAny(cleaned, "{")
	if start < 0 {
		return cleaned
	}
	end := strings.LastIndex(cleaned, "}")
	if end <= start {
		return cleaned
	}
	return strings.TrimSpace(cleaned[start : end+1])
}

func firstProviderString(values map[string]any, keys ...string) string {
	for _, key := range keys {
		value, ok := values[key]
		if !ok {
			continue
		}
		if text, ok := value.(string); ok && strings.TrimSpace(text) != "" {
			return strings.TrimSpace(text)
		}
	}
	return ""
}

func providerScore(values map[string]any) *int {
	for _, key := range []string{"score", "Score", "points", "Points"} {
		value, ok := values[key]
		if !ok {
			continue
		}
		var parsed int
		switch typed := value.(type) {
		case int:
			parsed = typed
		case int64:
			parsed = int(typed)
		case float64:
			parsed = int(math.Round(typed))
		case float32:
			parsed = int(math.Round(float64(typed)))
		case json.Number:
			number, err := typed.Float64()
			if err != nil {
				continue
			}
			parsed = int(math.Round(number))
		case string:
			var number float64
			if _, err := fmt.Sscanf(typed, "%f", &number); err != nil {
				continue
			}
			parsed = int(math.Round(number))
		default:
			continue
		}
		if parsed < 0 {
			parsed = 0
		}
		if parsed > 100 {
			parsed = 100
		}
		return &parsed
	}
	return nil
}

func normalizeProviderVerdict(value string) *string {
	normalized := strings.ToLower(strings.TrimSpace(value))
	switch normalized {
	case "correct", "right", "yes", "pass", "passed":
		verdict := "correct"
		return &verdict
	case "partial", "partially_correct", "partially correct", "mixed":
		verdict := "partial"
		return &verdict
	case "incorrect", "wrong", "no", "fail", "failed":
		verdict := "incorrect"
		return &verdict
	default:
		return nil
	}
}

func verdictFromScore(score int) *string {
	verdict := "incorrect"
	if score >= 80 {
		verdict = "correct"
	} else if score >= 40 {
		verdict = "partial"
	}
	return &verdict
}

func scoreFromVerdict(verdict string) *int {
	score := 0
	switch verdict {
	case "correct":
		score = 100
	case "partial":
		score = 60
	}
	return &score
}

func valueOrEmpty(value *string) string {
	if value == nil {
		return ""
	}
	return *value
}

func truncateForProvider(value string, max int) string {
	value = strings.TrimSpace(value)
	if max <= 0 || len(value) <= max {
		return value
	}
	return strings.TrimSpace(value[:max]) + "..."
}

func (s *Service) ToggleFavorite(questionID string) (Question, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	index, ok := s.findQuestionIndexLocked(questionID)
	if !ok {
		return Question{}, fmt.Errorf("question not found: %s", questionID)
	}
	question := s.state.Questions[index]
	question.IsFavorite = !question.IsFavorite
	question.UpdatedAt = nowISO()
	s.state.Questions[index] = question
	if err := s.flushLocked(); err != nil {
		return Question{}, err
	}
	return question, nil
}

func (s *Service) GetStats(examID string) (*QuestionBankStats, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	stats := calculateStats(examID, s.state.Questions)
	if stats.TotalCount == 0 {
		return nil, nil
	}
	return &stats, nil
}

func (s *Service) RefreshStats(examID string) (QuestionBankStats, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return calculateStats(examID, s.state.Questions), nil
}

func (s *Service) ResetProgress(examID string) (QuestionBankStats, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	for index := range s.state.Questions {
		if s.state.Questions[index].ExamID != examID {
			continue
		}
		resetQuestionProgress(&s.state.Questions[index])
	}
	s.state.Submissions = filterSubmissionsByExam(s.state.Submissions, s.state.Questions, examID)
	stats := calculateStats(examID, s.state.Questions)
	return stats, s.flushLocked()
}

func (s *Service) ResetQuestionsProgress(questionIDs []string) (BatchResult, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	idSet := make(map[string]bool, len(questionIDs))
	for _, questionID := range questionIDs {
		idSet[questionID] = true
	}
	result := BatchResult{Errors: []string{}}
	for index := range s.state.Questions {
		if !idSet[s.state.Questions[index].ID] {
			continue
		}
		resetQuestionProgress(&s.state.Questions[index])
		result.SuccessCount++
		delete(idSet, s.state.Questions[index].ID)
	}
	for missing := range idSet {
		result.FailedCount++
		result.Errors = append(result.Errors, fmt.Sprintf("question not found: %s", missing))
	}
	return result, s.flushLocked()
}

func (s *Service) GetHistory(questionID string, limit int) ([]QuestionHistory, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if limit <= 0 {
		limit = 50
	}
	histories := make([]QuestionHistory, 0)
	for index := len(s.state.Histories) - 1; index >= 0 && len(histories) < limit; index-- {
		history := s.state.Histories[index]
		if history.QuestionID == questionID {
			histories = append(histories, history)
		}
	}
	return histories, nil
}

func (s *Service) GetSubmissions(questionID string, limit int) ([]AnswerSubmission, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if limit <= 0 {
		limit = 20
	}
	submissions := make([]AnswerSubmission, 0)
	for index := len(s.state.Submissions) - 1; index >= 0 && len(submissions) < limit; index-- {
		submission := s.state.Submissions[index]
		if submission.QuestionID == questionID {
			submissions = append(submissions, submission)
		}
	}
	return submissions, nil
}

func (s *Service) GetLearningTrend(request GetLearningTrendRequest) ([]LearningTrendPoint, error) {
	start, end, err := parseDateRange(request.StartDate, request.EndDate)
	if err != nil {
		return nil, err
	}

	s.mu.RLock()
	defer s.mu.RUnlock()

	points := map[string]*LearningTrendPoint{}
	for _, submission := range s.state.Submissions {
		question, ok := s.findQuestionLocked(submission.QuestionID)
		if !ok || !matchesOptionalExam(question.ExamID, request.ExamID) {
			continue
		}
		date, ok := isoDate(submission.CreatedAt)
		if !ok || date.Before(start) || date.After(end) {
			continue
		}
		key := formatDate(date)
		point := points[key]
		if point == nil {
			point = &LearningTrendPoint{Date: key}
			points[key] = point
		}
		point.AttemptCount++
		if submission.IsCorrect != nil && *submission.IsCorrect {
			point.CorrectCount++
		}
	}

	trend := make([]LearningTrendPoint, 0)
	for day := start; !day.After(end); day = day.AddDate(0, 0, 1) {
		key := formatDate(day)
		point := LearningTrendPoint{Date: key}
		if existing := points[key]; existing != nil {
			point = *existing
		}
		point.CorrectRate = percentage(point.CorrectCount, point.AttemptCount)
		trend = append(trend, point)
	}
	return trend, nil
}

func (s *Service) GetActivityHeatmap(request GetActivityHeatmapRequest) ([]ActivityHeatmapPoint, error) {
	if request.Year <= 0 {
		return nil, errors.New("year is required")
	}
	start := time.Date(request.Year, 1, 1, 0, 0, 0, 0, time.UTC)
	end := time.Date(request.Year, 12, 31, 0, 0, 0, 0, time.UTC)

	s.mu.RLock()
	defer s.mu.RUnlock()

	points := map[string]*ActivityHeatmapPoint{}
	for _, submission := range s.state.Submissions {
		question, ok := s.findQuestionLocked(submission.QuestionID)
		if !ok || !matchesOptionalExam(question.ExamID, request.ExamID) {
			continue
		}
		date, ok := isoDate(submission.CreatedAt)
		if !ok || date.Before(start) || date.After(end) {
			continue
		}
		key := formatDate(date)
		point := points[key]
		if point == nil {
			point = &ActivityHeatmapPoint{Date: key}
			points[key] = point
		}
		point.Count++
		if submission.IsCorrect != nil && *submission.IsCorrect {
			point.CorrectCount++
		}
	}

	heatmap := make([]ActivityHeatmapPoint, 0, len(points))
	for _, point := range points {
		point.Level = activityLevel(point.Count)
		heatmap = append(heatmap, *point)
	}
	sort.SliceStable(heatmap, func(a, b int) bool {
		return heatmap[a].Date < heatmap[b].Date
	})
	return heatmap, nil
}

func (s *Service) GetKnowledgeStats(request GetKnowledgeStatsRequest) ([]KnowledgePoint, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	type totals struct {
		point         KnowledgePoint
		totalAttempts int
		totalCorrect  int
	}
	byTag := map[string]*totals{}
	for _, question := range s.state.Questions {
		if !matchesOptionalExam(question.ExamID, request.ExamID) {
			continue
		}
		for _, tag := range question.Tags {
			tag = strings.TrimSpace(tag)
			if tag == "" {
				continue
			}
			entry := byTag[tag]
			if entry == nil {
				entry = &totals{point: KnowledgePoint{Tag: tag}}
				byTag[tag] = entry
			}
			entry.point.Total++
			entry.totalAttempts += question.AttemptCount
			entry.totalCorrect += question.CorrectCount
			switch question.Status {
			case "mastered":
				entry.point.Mastered++
			case "in_progress":
				entry.point.InProgress++
			case "review":
				entry.point.Review++
			default:
				entry.point.NewCount++
			}
		}
	}

	stats := make([]KnowledgePoint, 0, len(byTag))
	for _, entry := range byTag {
		point := entry.point
		if point.Total > 0 {
			point.MasteryRate = ((float64(point.Mastered) + float64(point.InProgress)*0.5) / float64(point.Total)) * 100
		}
		point.CorrectRate = percentage(entry.totalCorrect, entry.totalAttempts)
		stats = append(stats, point)
	}
	sort.SliceStable(stats, func(a, b int) bool {
		if stats[a].Total == stats[b].Total {
			return stats[a].Tag < stats[b].Tag
		}
		return stats[a].Total > stats[b].Total
	})
	if len(stats) > 10 {
		stats = stats[:10]
	}
	return stats, nil
}

func (s *Service) GetKnowledgeStatsWithComparison(request GetKnowledgeStatsRequest) (KnowledgeStatsComparison, error) {
	current, err := s.GetKnowledgeStats(request)
	if err != nil {
		return KnowledgeStatsComparison{}, err
	}
	return KnowledgeStatsComparison{
		Current:  current,
		Previous: []KnowledgePoint{},
	}, nil
}

func (s *Service) StartTimedPractice(request StartTimedPracticeRequest) (TimedPracticeSession, error) {
	if strings.TrimSpace(request.ExamID) == "" {
		return TimedPracticeSession{}, errors.New("exam_id is required")
	}
	count := request.QuestionCount
	if count <= 0 {
		count = 10
	}
	duration := request.DurationMinutes
	if duration <= 0 {
		duration = 30
	}

	s.mu.RLock()
	defer s.mu.RUnlock()

	questions := s.selectQuestionsLocked(request.ExamID, QuestionFilters{}, nil, count)
	if len(questions) == 0 {
		return TimedPracticeSession{}, errors.New("question bank has no questions")
	}
	ids := questionIDs(questions)
	return TimedPracticeSession{
		ID:              "timed_" + randomToken(16),
		ExamID:          request.ExamID,
		DurationMinutes: duration,
		QuestionCount:   len(ids),
		QuestionIDs:     ids,
		StartedAt:       nowISO(),
		AnsweredCount:   0,
		CorrectCount:    0,
		IsTimeout:       false,
		IsSubmitted:     false,
		PausedSeconds:   0,
		IsPaused:        false,
	}, nil
}

func (s *Service) GenerateMockExam(request GenerateMockExamRequest) (MockExamSession, error) {
	if strings.TrimSpace(request.ExamID) == "" {
		return MockExamSession{}, errors.New("exam_id is required")
	}

	s.mu.RLock()
	defer s.mu.RUnlock()

	selected := make([]Question, 0)
	exclude := map[string]bool{}
	baseFilters := QuestionFilters{Tags: nonNilStrings(request.Config.Tags)}
	if !request.Config.IncludeMistakes {
		baseFilters.Status = []string{"new", "in_progress", "mastered"}
	}
	for _, question := range s.selectByDistributionLocked(request.ExamID, baseFilters, "type", request.Config.TypeDistribution, exclude) {
		selected = append(selected, question)
		exclude[question.ID] = true
	}
	for _, question := range s.selectByDistributionLocked(request.ExamID, baseFilters, "difficulty", request.Config.DifficultyDistribution, exclude) {
		selected = append(selected, question)
		exclude[question.ID] = true
	}

	total := 0
	if request.Config.TotalCount != nil {
		total = *request.Config.TotalCount
	}
	if total <= 0 {
		total = distributionTotal(request.Config.TypeDistribution)
		if diffTotal := distributionTotal(request.Config.DifficultyDistribution); diffTotal > total {
			total = diffTotal
		}
	}
	if total <= 0 {
		total = 30
	}
	if len(selected) < total {
		for _, question := range s.selectQuestionsLocked(request.ExamID, baseFilters, exclude, total-len(selected)) {
			selected = append(selected, question)
			exclude[question.ID] = true
		}
	}
	if len(selected) > total {
		selected = selected[:total]
	}
	if request.Config.Shuffle {
		stableShuffleQuestions(selected)
	}
	if len(selected) == 0 {
		return MockExamSession{}, errors.New("unable to select questions for mock exam")
	}

	return MockExamSession{
		ID:          "mock_" + randomToken(16),
		ExamID:      request.ExamID,
		Config:      request.Config,
		QuestionIDs: questionIDs(selected),
		StartedAt:   nowISO(),
		Answers:     map[string]string{},
		Results:     map[string]bool{},
		IsSubmitted: false,
	}, nil
}

func (s *Service) SubmitMockExam(request SubmitMockExamRequest) (MockExamScoreCard, error) {
	session := request.Session
	if strings.TrimSpace(session.ID) == "" {
		return MockExamScoreCard{}, errors.New("session.id is required")
	}

	s.mu.RLock()
	defer s.mu.RUnlock()

	total := len(session.QuestionIDs)
	answered := len(session.Answers)
	correct := 0
	wrongIDs := []string{}
	typeStats := map[string]TypeStatItem{}
	difficultyStats := map[string]DifficultyStatItem{}

	for _, questionID := range session.QuestionIDs {
		question, ok := s.findQuestionLocked(questionID)
		if !ok {
			continue
		}
		isCorrect := session.Results[questionID]
		if isCorrect {
			correct++
		} else if _, hasAnswer := session.Answers[questionID]; hasAnswer {
			wrongIDs = append(wrongIDs, questionID)
		}

		typeItem := typeStats[question.QuestionType]
		typeItem.Total++
		if isCorrect {
			typeItem.Correct++
		}
		typeStats[question.QuestionType] = typeItem

		diffKey := strings.TrimSpace(question.Difficulty)
		if diffKey != "" {
			diffItem := difficultyStats[diffKey]
			diffItem.Total++
			if isCorrect {
				diffItem.Correct++
			}
			difficultyStats[diffKey] = diffItem
		}
	}
	for key, item := range typeStats {
		item.Rate = percentage(item.Correct, item.Total)
		typeStats[key] = item
	}
	for key, item := range difficultyStats {
		item.Rate = percentage(item.Correct, item.Total)
		difficultyStats[key] = item
	}

	endedAt := session.EndedAt
	if endedAt == nil || strings.TrimSpace(*endedAt) == "" {
		now := nowISO()
		endedAt = &now
	}
	timeSpent := secondsBetween(session.StartedAt, *endedAt)
	correctRate := percentage(correct, total)
	return MockExamScoreCard{
		SessionID:        session.ID,
		ExamID:           session.ExamID,
		TotalCount:       total,
		AnsweredCount:    answered,
		CorrectCount:     correct,
		WrongCount:       maxInt(answered-correct, 0),
		UnansweredCount:  maxInt(total-answered, 0),
		CorrectRate:      correctRate,
		TimeSpentSeconds: timeSpent,
		TypeStats:        typeStats,
		DifficultyStats:  difficultyStats,
		WrongQuestionIDs: wrongIDs,
		Comment:          scoreComment(correctRate),
		CompletedAt:      nowISO(),
	}, nil
}

func (s *Service) GetDailyPractice(request GetDailyPracticeRequest) (DailyPracticeResult, error) {
	if strings.TrimSpace(request.ExamID) == "" {
		return DailyPracticeResult{}, errors.New("exam_id is required")
	}
	count := request.Count
	if count <= 0 {
		count = 10
	}

	s.mu.RLock()
	defer s.mu.RUnlock()

	selected := make([]Question, 0)
	exclude := map[string]bool{}
	pick := func(filters QuestionFilters, remaining int) int {
		if remaining <= 0 {
			return 0
		}
		questions := s.selectQuestionsLocked(request.ExamID, filters, exclude, remaining)
		for _, question := range questions {
			selected = append(selected, question)
			exclude[question.ID] = true
		}
		return len(questions)
	}
	mistakeCount := pick(QuestionFilters{Status: []string{"review"}}, maxInt(count/2, 1))
	newCount := pick(QuestionFilters{Status: []string{"new"}}, count-len(selected))
	reviewCount := pick(QuestionFilters{Status: []string{"mastered", "in_progress"}}, count-len(selected))
	_ = pick(QuestionFilters{}, count-len(selected))
	if len(selected) == 0 {
		return DailyPracticeResult{}, errors.New("question bank has no questions")
	}

	completed, correct := todaySubmissionCounts(s.state.Submissions, s.state.Questions, request.ExamID)
	return DailyPracticeResult{
		Date:           formatDate(time.Now().UTC()),
		ExamID:         request.ExamID,
		QuestionIDs:    questionIDs(selected),
		DailyTarget:    count,
		CompletedCount: completed,
		CorrectCount:   correct,
		SourceDistribution: DailySourceDistribution{
			MistakeCount: mistakeCount,
			NewCount:     newCount,
			ReviewCount:  reviewCount,
		},
		IsCompleted: completed >= count,
	}, nil
}

func (s *Service) GeneratePaper(request GeneratePaperRequest) (GeneratedPaper, error) {
	if strings.TrimSpace(request.ExamID) == "" {
		return GeneratedPaper{}, errors.New("exam_id is required")
	}

	s.mu.RLock()
	defer s.mu.RUnlock()

	filters := QuestionFilters{
		Difficulty: nonNilStrings(request.Config.DifficultyFilter),
		Tags:       nonNilStrings(request.Config.TagsFilter),
	}
	selected := make([]Question, 0)
	exclude := map[string]bool{}
	for questionType, count := range request.Config.TypeSelection {
		if count <= 0 {
			continue
		}
		typeFilters := filters
		typeFilters.QuestionType = []string{questionType}
		for _, question := range s.selectQuestionsLocked(request.ExamID, typeFilters, exclude, count) {
			selected = append(selected, question)
			exclude[question.ID] = true
		}
	}
	if len(request.Config.TypeSelection) == 0 {
		selected = s.selectQuestionsLocked(request.ExamID, filters, exclude, 500)
	}
	if request.Config.Shuffle {
		stableShuffleQuestions(selected)
	}
	if len(selected) == 0 {
		return GeneratedPaper{}, errors.New("unable to select questions for paper")
	}
	questions := make([]Question, len(selected))
	copy(questions, selected)
	for index := range questions {
		if !request.Config.IncludeAnswers {
			questions[index].Answer = ""
		}
		if !request.Config.IncludeExplanations {
			questions[index].Explanation = ""
		}
	}
	title := strings.TrimSpace(request.Config.Title)
	if title == "" {
		title = "Practice Paper"
	}
	return GeneratedPaper{
		ID:         "paper_" + randomToken(16),
		Title:      title,
		ExamID:     request.ExamID,
		Questions:  questions,
		TotalScore: len(questions),
		Config:     request.Config,
		CreatedAt:  nowISO(),
	}, nil
}

func (s *Service) GetCheckInCalendar(request GetCheckInCalendarRequest) (CheckInCalendar, error) {
	if request.Year <= 0 {
		return CheckInCalendar{}, errors.New("year is required")
	}
	if request.Month < 1 || request.Month > 12 {
		return CheckInCalendar{}, errors.New("month must be 1-12")
	}
	start := time.Date(request.Year, time.Month(request.Month), 1, 0, 0, 0, 0, time.UTC)
	end := start.AddDate(0, 1, 0)

	s.mu.RLock()
	defer s.mu.RUnlock()

	byDate := map[string]*DailyCheckIn{}
	for _, submission := range s.state.Submissions {
		question, ok := s.findQuestionLocked(submission.QuestionID)
		if !ok || !matchesOptionalExam(question.ExamID, request.ExamID) {
			continue
		}
		date, ok := isoDate(submission.CreatedAt)
		if !ok || date.Before(start) || !date.Before(end) {
			continue
		}
		key := formatDate(date)
		day := byDate[key]
		if day == nil {
			day = &DailyCheckIn{
				Date:   key,
				ExamID: request.ExamID,
			}
			byDate[key] = day
		}
		day.QuestionCount++
		if submission.IsCorrect != nil && *submission.IsCorrect {
			day.CorrectCount++
		}
		day.TargetAchieved = day.QuestionCount >= 10
	}
	days := make([]DailyCheckIn, 0, len(byDate))
	totalQuestions := 0
	for _, day := range byDate {
		days = append(days, *day)
		totalQuestions += day.QuestionCount
	}
	sort.SliceStable(days, func(a, b int) bool {
		return days[a].Date < days[b].Date
	})
	return CheckInCalendar{
		Year:                request.Year,
		Month:               request.Month,
		Days:                days,
		StreakDays:          streakDays(days),
		MonthCheckInDays:    len(days),
		MonthTotalQuestions: totalQuestions,
	}, nil
}

func (s *Service) findQuestionLocked(questionID string) (Question, bool) {
	for _, question := range s.state.Questions {
		if question.ID == questionID {
			return question, true
		}
	}
	return Question{}, false
}

func (s *Service) findQuestionIndexLocked(questionID string) (int, bool) {
	for index, question := range s.state.Questions {
		if question.ID == questionID {
			return index, true
		}
	}
	return -1, false
}

func (s *Service) findSubmissionIndexLocked(submissionID string) (int, bool) {
	for index, submission := range s.state.Submissions {
		if submission.ID == submissionID {
			return index, true
		}
	}
	return -1, false
}

func (s *Service) syncQuestionResourceLocked(index int) error {
	if s.vfs == nil {
		return nil
	}
	question := s.state.Questions[index]
	result, err := s.vfs.CreateOrUpdateSource(vfs.CreateResourceInput{
		Type:     "exam",
		Data:     questionResourceData(question),
		SourceID: &question.ID,
		Metadata: questionResourceMetadata(question),
	})
	if err != nil {
		return err
	}
	s.state.Questions[index].ResourceID = result.ResourceID
	s.state.Questions[index].ResourceHash = result.Hash
	return nil
}

func (s *Service) recordQuestionChangesLocked(previous Question, next Question, operator string) {
	s.recordStringChangeLocked(next.ID, "content", previous.Content, next.Content, operator)
	s.recordStringChangeLocked(next.ID, "question_type", previous.QuestionType, next.QuestionType, operator)
	s.recordStringChangeLocked(next.ID, "answer", previous.Answer, next.Answer, operator)
	s.recordStringChangeLocked(next.ID, "explanation", previous.Explanation, next.Explanation, operator)
	s.recordStringChangeLocked(next.ID, "difficulty", previous.Difficulty, next.Difficulty, operator)
	s.recordStringChangeLocked(next.ID, "status", previous.Status, next.Status, operator)
	s.recordStringChangeLocked(next.ID, "user_answer", previous.UserAnswer, next.UserAnswer, operator)
	s.recordStringChangeLocked(next.ID, "user_note", previous.UserNote, next.UserNote, operator)
	s.recordJSONChangeLocked(next.ID, "options", previous.Options, next.Options, operator)
	s.recordJSONChangeLocked(next.ID, "tags", previous.Tags, next.Tags, operator)
	s.recordJSONChangeLocked(next.ID, "images", previous.Images, next.Images, operator)
	s.recordBoolPointerChangeLocked(next.ID, "is_correct", previous.IsCorrect, next.IsCorrect, operator)
	s.recordIntChangeLocked(next.ID, "attempt_count", previous.AttemptCount, next.AttemptCount, operator)
	s.recordIntChangeLocked(next.ID, "correct_count", previous.CorrectCount, next.CorrectCount, operator)
}

func (s *Service) recordStringChangeLocked(questionID string, field string, oldValue string, newValue string, operator string) {
	if oldValue == newValue {
		return
	}
	s.recordHistoryLocked(questionID, field, &oldValue, &newValue, operator, "")
}

func (s *Service) recordIntChangeLocked(questionID string, field string, oldValue int, newValue int, operator string) {
	if oldValue == newValue {
		return
	}
	oldText := fmt.Sprintf("%d", oldValue)
	newText := fmt.Sprintf("%d", newValue)
	s.recordHistoryLocked(questionID, field, &oldText, &newText, operator, "")
}

func (s *Service) recordBoolPointerChangeLocked(questionID string, field string, oldValue *bool, newValue *bool, operator string) {
	oldText := pointerBoolText(oldValue)
	newText := pointerBoolText(newValue)
	if oldText == newText {
		return
	}
	s.recordHistoryLocked(questionID, field, oldText, newText, operator, "")
}

func (s *Service) recordJSONChangeLocked(questionID string, field string, oldValue any, newValue any, operator string) {
	oldText := jsonStableText(oldValue)
	newText := jsonStableText(newValue)
	if oldText == newText {
		return
	}
	s.recordHistoryLocked(questionID, field, &oldText, &newText, operator, "")
}

func (s *Service) recordHistoryLocked(questionID string, field string, oldValue *string, newValue *string, operator string, reason string) {
	var reasonPtr *string
	if strings.TrimSpace(reason) != "" {
		reasonPtr = &reason
	}
	s.state.Histories = append(s.state.Histories, QuestionHistory{
		ID:         "hist_" + randomToken(16),
		QuestionID: questionID,
		FieldName:  field,
		OldValue:   oldValue,
		NewValue:   newValue,
		Operator:   operator,
		Reason:     reasonPtr,
		CreatedAt:  nowISO(),
	})
}

func (s *Service) selectByDistributionLocked(examID string, base QuestionFilters, dimension string, distribution map[string]int, exclude map[string]bool) []Question {
	selected := make([]Question, 0)
	keys := make([]string, 0, len(distribution))
	for key := range distribution {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	for _, key := range keys {
		count := distribution[key]
		if count <= 0 {
			continue
		}
		filters := base
		switch dimension {
		case "type":
			filters.QuestionType = []string{key}
		case "difficulty":
			filters.Difficulty = []string{key}
		}
		questions := s.selectQuestionsLocked(examID, filters, exclude, count)
		for _, question := range questions {
			selected = append(selected, question)
			exclude[question.ID] = true
		}
	}
	return selected
}

func (s *Service) selectQuestionsLocked(examID string, filters QuestionFilters, exclude map[string]bool, limit int) []Question {
	if limit <= 0 {
		return []Question{}
	}
	candidates := make([]Question, 0)
	for _, question := range s.state.Questions {
		if question.ExamID != examID {
			continue
		}
		if exclude != nil && exclude[question.ID] {
			continue
		}
		if !matchesFilters(question, &filters) {
			continue
		}
		candidates = append(candidates, question)
	}
	sort.SliceStable(candidates, func(a, b int) bool {
		ap := statusPriority(candidates[a].Status)
		bp := statusPriority(candidates[b].Status)
		if ap != bp {
			return ap < bp
		}
		if candidates[a].LastAttemptAt == nil && candidates[b].LastAttemptAt != nil {
			return true
		}
		if candidates[a].LastAttemptAt != nil && candidates[b].LastAttemptAt == nil {
			return false
		}
		if candidates[a].LastAttemptAt != nil && candidates[b].LastAttemptAt != nil && *candidates[a].LastAttemptAt != *candidates[b].LastAttemptAt {
			return *candidates[a].LastAttemptAt < *candidates[b].LastAttemptAt
		}
		if candidates[a].CreatedAt != candidates[b].CreatedAt {
			return candidates[a].CreatedAt < candidates[b].CreatedAt
		}
		return candidates[a].ID < candidates[b].ID
	})
	if len(candidates) > limit {
		candidates = candidates[:limit]
	}
	return candidates
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
	if s.state.Questions == nil {
		s.state.Questions = []Question{}
	}
	if s.state.Submissions == nil {
		s.state.Submissions = []AnswerSubmission{}
	}
	if s.state.Histories == nil {
		s.state.Histories = []QuestionHistory{}
	}
	if s.state.SyncConfigs == nil {
		s.state.SyncConfigs = map[string]SyncConfig{}
	}
	return nil
}

func (s *Service) flushLocked() error {
	return storage.WriteJSONAtomic(s.path, s.state)
}

func questionResourceData(question Question) string {
	var builder strings.Builder
	builder.WriteString("# ")
	builder.WriteString(questionTitle(question))
	builder.WriteString("\n\n")
	builder.WriteString(strings.TrimSpace(question.Content))
	builder.WriteString("\n")

	if len(question.Options) > 0 {
		builder.WriteString("\n## Options\n")
		for _, option := range question.Options {
			key := strings.TrimSpace(option.Key)
			if key == "" {
				key = "-"
			}
			builder.WriteString("- ")
			builder.WriteString(key)
			builder.WriteString(". ")
			builder.WriteString(strings.TrimSpace(option.Content))
			builder.WriteString("\n")
		}
	}
	if strings.TrimSpace(question.Answer) != "" {
		builder.WriteString("\n## Answer\n")
		builder.WriteString(strings.TrimSpace(question.Answer))
		builder.WriteString("\n")
	}
	if strings.TrimSpace(question.Explanation) != "" {
		builder.WriteString("\n## Explanation\n")
		builder.WriteString(strings.TrimSpace(question.Explanation))
		builder.WriteString("\n")
	}
	if strings.TrimSpace(question.UserNote) != "" {
		builder.WriteString("\n## User Note\n")
		builder.WriteString(strings.TrimSpace(question.UserNote))
		builder.WriteString("\n")
	}
	if question.AIFeedback != nil && strings.TrimSpace(*question.AIFeedback) != "" {
		builder.WriteString("\n## AI Feedback\n")
		builder.WriteString(strings.TrimSpace(*question.AIFeedback))
		builder.WriteString("\n")
	}
	if len(question.Images) > 0 {
		builder.WriteString("\n## Images\n")
		for _, image := range question.Images {
			builder.WriteString("- ")
			builder.WriteString(strings.TrimSpace(image.Name))
			if strings.TrimSpace(image.ID) != "" {
				builder.WriteString(" (")
				builder.WriteString(strings.TrimSpace(image.ID))
				builder.WriteString(")")
			}
			builder.WriteString("\n")
		}
	}
	return strings.TrimSpace(builder.String()) + "\n"
}

func questionResourceMetadata(question Question) map[string]any {
	title := questionTitle(question)
	metadata := map[string]any{
		"name":         title,
		"title":        title,
		"path":         "/" + question.ID,
		"sourceId":     question.ID,
		"sourceDb":     "qbank",
		"sourceType":   "question",
		"resourceType": "exam",
		"previewType":  "exam",
		"examId":       question.ExamID,
		"questionType": question.QuestionType,
		"difficulty":   question.Difficulty,
		"status":       question.Status,
		"tags":         nonNilStrings(question.Tags),
		"isFavorite":   question.IsFavorite,
		"imageCount":   len(question.Images),
		"createdAt":    question.CreatedAt,
		"updatedAt":    question.UpdatedAt,
	}
	if question.CardID != nil && strings.TrimSpace(*question.CardID) != "" {
		metadata["cardId"] = strings.TrimSpace(*question.CardID)
	}
	if question.QuestionLabel != nil && strings.TrimSpace(*question.QuestionLabel) != "" {
		metadata["questionLabel"] = strings.TrimSpace(*question.QuestionLabel)
	}
	if question.SourceRef != nil && strings.TrimSpace(*question.SourceRef) != "" {
		metadata["sourceRef"] = strings.TrimSpace(*question.SourceRef)
	}
	if question.ParentID != nil && strings.TrimSpace(*question.ParentID) != "" {
		metadata["parentId"] = strings.TrimSpace(*question.ParentID)
	}
	if question.AIFeedback != nil && strings.TrimSpace(*question.AIFeedback) != "" {
		metadata["aiFeedback"] = strings.TrimSpace(*question.AIFeedback)
	}
	if question.AIScore != nil {
		metadata["aiScore"] = *question.AIScore
	}
	if question.AIGradedAt != nil && strings.TrimSpace(*question.AIGradedAt) != "" {
		metadata["aiGradedAt"] = strings.TrimSpace(*question.AIGradedAt)
	}
	return metadata
}

func questionTitle(question Question) string {
	if question.QuestionLabel != nil && strings.TrimSpace(*question.QuestionLabel) != "" {
		return strings.TrimSpace(*question.QuestionLabel)
	}
	content := strings.Join(strings.Fields(question.Content), " ")
	if content == "" {
		return question.ID
	}
	runes := []rune(content)
	if len(runes) > 80 {
		return string(runes[:80])
	}
	return content
}

func matchesFilters(question Question, filters *QuestionFilters) bool {
	if filters == nil {
		return true
	}
	if len(filters.Status) > 0 && !contains(filters.Status, question.Status) {
		return false
	}
	if len(filters.Difficulty) > 0 && !contains(filters.Difficulty, question.Difficulty) {
		return false
	}
	if len(filters.QuestionType) > 0 && !contains(filters.QuestionType, question.QuestionType) {
		return false
	}
	if len(filters.Tags) > 0 && !hasAll(question.Tags, filters.Tags) {
		return false
	}
	if filters.IsFavorite != nil && question.IsFavorite != *filters.IsFavorite {
		return false
	}
	if filters.Search != nil && strings.TrimSpace(*filters.Search) != "" {
		query := strings.ToLower(strings.TrimSpace(*filters.Search))
		haystack := strings.ToLower(question.Content + "\n" + question.Answer + "\n" + question.Explanation + "\n" + strings.Join(question.Tags, "\n"))
		if !strings.Contains(haystack, query) {
			return false
		}
	}
	return true
}

func normalizeMentionLimit(limit int) int {
	if limit <= 0 {
		return 8
	}
	if limit > 40 {
		return 40
	}
	return limit
}

func matchesMentionSubject(question Question, subject *string) bool {
	if subject == nil {
		return true
	}
	needle := strings.ToLower(strings.TrimSpace(*subject))
	if needle == "" || needle == "_global" {
		return true
	}
	if strings.ToLower(question.ExamID) == needle {
		return true
	}
	if question.SourceRef != nil && strings.ToLower(strings.TrimSpace(*question.SourceRef)) == needle {
		return true
	}
	for _, tag := range question.Tags {
		if strings.ToLower(strings.TrimSpace(tag)) == needle {
			return true
		}
	}
	return false
}

func scoreMentionQuestion(question Question, query string, terms []string) int {
	fields := []struct {
		text   string
		weight int
	}{
		{derefQuestionLabel(question.QuestionLabel), 42},
		{question.Content, 36},
		{question.UserNote, 30},
		{question.Explanation, 26},
		{question.Answer, 18},
		{strings.Join(question.Tags, " "), 22},
	}
	score := 0
	for _, field := range fields {
		lower := strings.ToLower(field.text)
		if lower == "" {
			continue
		}
		if strings.Contains(lower, query) {
			score += field.weight * 2
		}
		for _, term := range terms {
			if term != "" && strings.Contains(lower, term) {
				score += field.weight
			}
		}
	}
	if score > 0 && question.IsFavorite {
		score += 6
	}
	if score > 0 && question.AttemptCount > 0 && question.IsCorrect != nil && !*question.IsCorrect {
		score += 8
	}
	return score
}

func questionMentionTitle(question Question) string {
	if label := strings.TrimSpace(derefQuestionLabel(question.QuestionLabel)); label != "" {
		return label
	}
	if content := previewText(question.Content, 96); content != "" {
		return content
	}
	return question.ID
}

func questionMentionInsight(question Question, query string) string {
	candidates := []string{
		question.Explanation,
		question.UserNote,
		question.Answer,
		question.Content,
	}
	for _, candidate := range candidates {
		if snippet := querySnippet(candidate, query, 160); snippet != "" {
			return snippet
		}
	}
	for _, candidate := range candidates {
		if snippet := previewText(candidate, 160); snippet != "" {
			return snippet
		}
	}
	return ""
}

func derefQuestionLabel(value *string) string {
	if value == nil {
		return ""
	}
	return *value
}

func querySnippet(text string, query string, maxLength int) string {
	trimmedText := strings.TrimSpace(text)
	trimmedQuery := strings.TrimSpace(query)
	if trimmedText == "" || trimmedQuery == "" {
		return ""
	}
	lowerText := strings.ToLower(trimmedText)
	lowerQuery := strings.ToLower(trimmedQuery)
	byteIndex := strings.Index(lowerText, lowerQuery)
	if byteIndex < 0 {
		return ""
	}
	charIndex := len([]rune(trimmedText[:byteIndex]))
	runes := []rune(trimmedText)
	if len(runes) <= maxLength {
		return trimmedText
	}
	half := maxLength / 2
	start := charIndex - half
	if start < 0 {
		start = 0
	}
	end := start + maxLength
	if end > len(runes) {
		end = len(runes)
		start = end - maxLength
		if start < 0 {
			start = 0
		}
	}
	snippet := strings.TrimSpace(string(runes[start:end]))
	if start > 0 {
		snippet = "..." + snippet
	}
	if end < len(runes) {
		snippet += "..."
	}
	return snippet
}

func previewText(text string, maxLength int) string {
	trimmed := strings.TrimSpace(text)
	if trimmed == "" {
		return ""
	}
	runes := []rune(trimmed)
	if maxLength <= 0 || len(runes) <= maxLength {
		return trimmed
	}
	return strings.TrimSpace(string(runes[:maxLength])) + "..."
}

func sortQuestions(questions []Question) {
	sort.SliceStable(questions, func(a, b int) bool {
		return questions[a].CreatedAt < questions[b].CreatedAt
	})
}

func pageQuestions(questions []Question, page int, pageSize int) QuestionListResult {
	total := len(questions)
	offset := (page - 1) * pageSize
	paged := []Question{}
	if offset < total {
		end := offset + pageSize
		if end > total {
			end = total
		}
		paged = questions[offset:end]
	}
	return QuestionListResult{
		Questions: paged,
		Total:     total,
		Page:      page,
		PageSize:  pageSize,
		HasMore:   page*pageSize < total,
	}
}

func normalizePage(page int, pageSize int) (int, int) {
	if page < 1 {
		page = 1
	}
	if pageSize < 1 {
		pageSize = 50
	}
	if pageSize > 100 {
		pageSize = 100
	}
	return page, pageSize
}

func calculateStats(examID string, questions []Question) QuestionBankStats {
	stats := QuestionBankStats{ExamID: examID, UpdatedAt: nowISO()}
	for _, question := range questions {
		if question.ExamID != examID {
			continue
		}
		stats.TotalCount++
		stats.TotalAttempts += question.AttemptCount
		stats.TotalCorrect += question.CorrectCount
		switch question.Status {
		case "mastered":
			stats.MasteredCount++
		case "review":
			stats.ReviewCount++
		case "in_progress":
			stats.InProgressCount++
		default:
			stats.NewCount++
		}
	}
	if stats.TotalAttempts > 0 {
		stats.CorrectRate = float64(stats.TotalCorrect) / float64(stats.TotalAttempts)
	}
	return stats
}

func nextStatus(attempts int, correct int, isCorrect *bool) string {
	if attempts == 0 {
		return "new"
	}
	if isCorrect != nil && !*isCorrect {
		return "review"
	}
	if correct >= 2 {
		return "mastered"
	}
	return "in_progress"
}

func resetQuestionProgress(question *Question) {
	question.Status = "new"
	question.UserAnswer = ""
	question.IsCorrect = nil
	question.AttemptCount = 0
	question.CorrectCount = 0
	question.LastAttemptAt = nil
	question.UpdatedAt = nowISO()
}

func normalizedAnswer(value string) string {
	return strings.ToLower(strings.Join(strings.Fields(value), ""))
}

func isSubjective(questionType string) bool {
	switch questionType {
	case "essay", "short_answer", "calculation", "proof":
		return true
	default:
		return false
	}
}

func defaultSyncConfig() SyncConfig {
	return SyncConfig{
		DefaultStrategy:  "keep_newer",
		AutoSync:         false,
		SyncIntervalSecs: 300,
		SyncProgress:     true,
		SyncNotes:        true,
	}
}

func normalizeSyncConfig(config SyncConfig) SyncConfig {
	defaults := defaultSyncConfig()
	if strings.TrimSpace(config.DefaultStrategy) == "" {
		config.DefaultStrategy = defaults.DefaultStrategy
	}
	if config.SyncIntervalSecs <= 0 {
		config.SyncIntervalSecs = defaults.SyncIntervalSecs
	}
	return config
}

func buildLeanGradingFeedback(question Question, submission AnswerSubmission, mode string) (string, *string, *int) {
	answer := strings.TrimSpace(submission.UserAnswer)
	reference := strings.TrimSpace(question.Answer)
	explanation := strings.TrimSpace(question.Explanation)
	if mode == "analyze" {
		feedback := buildLeanAnalysisFeedback(question, answer, reference, explanation)
		return feedback, nil, nil
	}

	score := estimateAnswerScore(answer, reference)
	verdictValue := "incorrect"
	if score >= 80 {
		verdictValue = "correct"
	} else if score >= 40 {
		verdictValue = "partial"
	}
	feedback := buildLeanGradeFeedback(question, answer, reference, explanation, verdictValue, score)
	return feedback, &verdictValue, &score
}

func buildLeanGradeFeedback(question Question, userAnswer string, reference string, explanation string, verdict string, score int) string {
	var builder strings.Builder
	builder.WriteString("### AI Grading\n\n")
	builder.WriteString("This Go migration shell used deterministic local grading because no usable remote grading result was available.\n\n")
	builder.WriteString("- Verdict: ")
	builder.WriteString(verdict)
	builder.WriteString("\n- Score: ")
	builder.WriteString(fmt.Sprintf("%d/100", score))
	builder.WriteString("\n- Question type: ")
	builder.WriteString(question.QuestionType)
	builder.WriteString("\n\n")
	builder.WriteString("#### Student Answer\n\n")
	builder.WriteString(nonEmptyForFeedback(userAnswer))
	builder.WriteString("\n\n")
	builder.WriteString("#### Reference Answer\n\n")
	builder.WriteString(nonEmptyForFeedback(reference))
	if explanation != "" {
		builder.WriteString("\n\n#### Reference Explanation\n\n")
		builder.WriteString(explanation)
	}
	builder.WriteString("\n\n#### Feedback\n\n")
	switch verdict {
	case "correct":
		builder.WriteString("The submitted answer closely matches the reference answer. Review the explanation to consolidate the method.")
	case "partial":
		builder.WriteString("The submitted answer overlaps with the reference answer but appears incomplete. Compare the missing key points against the reference answer.")
	default:
		builder.WriteString("The submitted answer does not sufficiently match the reference answer. Revisit the core concept and try restating the solution before checking the explanation.")
	}
	return builder.String()
}

func buildLeanAnalysisFeedback(question Question, userAnswer string, reference string, explanation string) string {
	var builder strings.Builder
	builder.WriteString("### AI Analysis\n\n")
	builder.WriteString("This Go migration shell generated a deterministic local analysis summary because no usable remote analysis result was available.\n\n")
	builder.WriteString("#### Solving Direction\n\n")
	if explanation != "" {
		builder.WriteString(explanation)
	} else if reference != "" {
		builder.WriteString("Use the reference answer as the target form: ")
		builder.WriteString(reference)
	} else {
		builder.WriteString("No reference explanation is available yet. Compare your answer with the question requirements and identify the core concept.")
	}
	builder.WriteString("\n\n#### Your Current Answer\n\n")
	builder.WriteString(nonEmptyForFeedback(userAnswer))
	if len(question.Tags) > 0 {
		builder.WriteString("\n\n#### Related Tags\n\n")
		for _, tag := range question.Tags {
			if strings.TrimSpace(tag) == "" {
				continue
			}
			builder.WriteString("- ")
			builder.WriteString(strings.TrimSpace(tag))
			builder.WriteString("\n")
		}
	}
	return builder.String()
}

func estimateAnswerScore(userAnswer string, reference string) int {
	userTokens := answerTokens(userAnswer)
	referenceTokens := answerTokens(reference)
	if len(referenceTokens) == 0 {
		if len(userTokens) == 0 {
			return 0
		}
		return 50
	}
	if normalizedAnswer(userAnswer) == normalizedAnswer(reference) {
		return 100
	}
	matched := 0
	userSet := make(map[string]bool, len(userTokens))
	for _, token := range userTokens {
		userSet[token] = true
	}
	for _, token := range referenceTokens {
		if userSet[token] {
			matched++
		}
	}
	score := int(math.Round(float64(matched) / float64(len(referenceTokens)) * 100))
	if score > 100 {
		return 100
	}
	if score < 0 {
		return 0
	}
	return score
}

func answerTokens(value string) []string {
	normalized := strings.ToLower(value)
	fields := strings.FieldsFunc(normalized, func(r rune) bool {
		if r >= 'a' && r <= 'z' {
			return false
		}
		if r >= '0' && r <= '9' {
			return false
		}
		if r >= '\u4e00' && r <= '\u9fff' {
			return false
		}
		return true
	})
	seen := map[string]bool{}
	out := make([]string, 0, len(fields))
	for _, field := range fields {
		field = strings.TrimSpace(field)
		if field == "" || seen[field] {
			continue
		}
		seen[field] = true
		out = append(out, field)
	}
	return out
}

func nonEmptyForFeedback(value string) string {
	if strings.TrimSpace(value) == "" {
		return "_No answer provided._"
	}
	return strings.TrimSpace(value)
}

func filterSubmissions(submissions []AnswerSubmission, deletedQuestionID string) []AnswerSubmission {
	out := make([]AnswerSubmission, 0, len(submissions))
	for _, submission := range submissions {
		if submission.QuestionID != deletedQuestionID {
			out = append(out, submission)
		}
	}
	return out
}

func filterSubmissionsByExam(submissions []AnswerSubmission, questions []Question, examID string) []AnswerSubmission {
	resetIDs := map[string]bool{}
	for _, question := range questions {
		if question.ExamID == examID {
			resetIDs[question.ID] = true
		}
	}
	out := make([]AnswerSubmission, 0, len(submissions))
	for _, submission := range submissions {
		if !resetIDs[submission.QuestionID] {
			out = append(out, submission)
		}
	}
	return out
}

func valueOr(value *string, fallback string) string {
	if value == nil {
		return fallback
	}
	trimmed := strings.TrimSpace(*value)
	if trimmed == "" {
		return fallback
	}
	return trimmed
}

func nonNilOptions(options []QuestionOption) []QuestionOption {
	if options == nil {
		return []QuestionOption{}
	}
	return options
}

func nonNilImages(images []QuestionImage) []QuestionImage {
	if images == nil {
		return []QuestionImage{}
	}
	return images
}

func nonNilStrings(values []string) []string {
	if values == nil {
		return []string{}
	}
	return values
}

func contains(values []string, needle string) bool {
	for _, value := range values {
		if value == needle {
			return true
		}
	}
	return false
}

func hasAll(actual []string, expected []string) bool {
	for _, value := range expected {
		if !contains(actual, value) {
			return false
		}
	}
	return true
}

func matchesOptionalExam(questionExamID string, examID *string) bool {
	return examID == nil || strings.TrimSpace(*examID) == "" || questionExamID == *examID
}

func parseDateRange(startDate string, endDate string) (time.Time, time.Time, error) {
	start, err := time.ParseInLocation("2006-01-02", strings.TrimSpace(startDate), time.UTC)
	if err != nil {
		return time.Time{}, time.Time{}, fmt.Errorf("invalid start_date: %w", err)
	}
	end, err := time.ParseInLocation("2006-01-02", strings.TrimSpace(endDate), time.UTC)
	if err != nil {
		return time.Time{}, time.Time{}, fmt.Errorf("invalid end_date: %w", err)
	}
	if end.Before(start) {
		return time.Time{}, time.Time{}, errors.New("end_date must be on or after start_date")
	}
	return start, end, nil
}

func isoDate(value string) (time.Time, bool) {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return time.Time{}, false
	}
	if parsed, err := time.Parse(time.RFC3339, trimmed); err == nil {
		return time.Date(parsed.UTC().Year(), parsed.UTC().Month(), parsed.UTC().Day(), 0, 0, 0, 0, time.UTC), true
	}
	if parsed, err := time.ParseInLocation("2006-01-02", trimmed, time.UTC); err == nil {
		return parsed, true
	}
	return time.Time{}, false
}

func formatDate(value time.Time) string {
	return value.UTC().Format("2006-01-02")
}

func percentage(numerator int, denominator int) float64 {
	if denominator <= 0 {
		return 0
	}
	return float64(numerator) / float64(denominator) * 100
}

func activityLevel(count int) int {
	switch {
	case count <= 0:
		return 0
	case count <= 3:
		return 1
	case count <= 6:
		return 2
	case count <= 10:
		return 3
	default:
		return 4
	}
}

func questionIDs(questions []Question) []string {
	ids := make([]string, 0, len(questions))
	for _, question := range questions {
		ids = append(ids, question.ID)
	}
	return ids
}

func distributionTotal(distribution map[string]int) int {
	total := 0
	for _, count := range distribution {
		if count > 0 {
			total += count
		}
	}
	return total
}

func stableShuffleQuestions(questions []Question) {
	sort.SliceStable(questions, func(a, b int) bool {
		return questions[a].ID > questions[b].ID
	})
}

func todaySubmissionCounts(submissions []AnswerSubmission, questions []Question, examID string) (int, int) {
	questionExam := map[string]string{}
	for _, question := range questions {
		questionExam[question.ID] = question.ExamID
	}
	today := formatDate(time.Now().UTC())
	completed := 0
	correct := 0
	for _, submission := range submissions {
		if questionExam[submission.QuestionID] != examID {
			continue
		}
		date, ok := isoDate(submission.CreatedAt)
		if !ok || formatDate(date) != today {
			continue
		}
		completed++
		if submission.IsCorrect != nil && *submission.IsCorrect {
			correct++
		}
	}
	return completed, correct
}

func streakDays(days []DailyCheckIn) int {
	active := map[string]bool{}
	for _, day := range days {
		if day.QuestionCount > 0 {
			active[day.Date] = true
		}
	}
	if len(active) == 0 {
		return 0
	}
	start := time.Now().UTC()
	today := formatDate(start)
	yesterday := formatDate(start.AddDate(0, 0, -1))
	if !active[today] {
		if !active[yesterday] {
			return 0
		}
		start = start.AddDate(0, 0, -1)
	}
	streak := 0
	for day := start; active[formatDate(day)]; day = day.AddDate(0, 0, -1) {
		streak++
	}
	return streak
}

func statusPriority(status string) int {
	switch status {
	case "review":
		return 0
	case "new":
		return 1
	case "in_progress":
		return 2
	case "mastered":
		return 3
	default:
		return 4
	}
}

func pointerBoolText(value *bool) *string {
	if value == nil {
		return nil
	}
	text := "false"
	if *value {
		text = "true"
	}
	return &text
}

func jsonStableText(value any) string {
	bytes, err := json.Marshal(value)
	if err != nil {
		return fmt.Sprintf("%v", value)
	}
	return string(bytes)
}

func secondsBetween(startedAt string, endedAt string) int {
	start, err := time.Parse(time.RFC3339, strings.TrimSpace(startedAt))
	if err != nil {
		return 0
	}
	end, err := time.Parse(time.RFC3339, strings.TrimSpace(endedAt))
	if err != nil {
		return 0
	}
	seconds := int(end.Sub(start).Seconds())
	if seconds < 0 {
		return 0
	}
	return seconds
}

func scoreComment(correctRate float64) string {
	switch {
	case correctRate >= 90:
		return "Excellent"
	case correctRate >= 80:
		return "Good"
	case correctRate >= 60:
		return "Passed"
	default:
		return "Needs review"
	}
}

func maxInt(a int, b int) int {
	if a > b {
		return a
	}
	return b
}

func nowISO() string {
	return time.Now().UTC().Format(time.RFC3339)
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
