package bindings

import (
	"deep-student-go/internal/app"
	"deep-student-go/internal/qbank"
)

type QbankService struct {
	app *app.App
}

func NewQbankService(app *app.App) *QbankService {
	return &QbankService{app: app}
}

func (s *QbankService) ListQuestions(request qbank.ListQuestionsRequest) (qbank.QuestionListResult, error) {
	return s.app.Qbank.ListQuestions(request)
}

func (s *QbankService) SearchQuestions(request qbank.SearchQuestionsRequest) (qbank.QuestionSearchListResult, error) {
	return s.app.Qbank.SearchQuestions(request)
}

func (s *QbankService) NotesMentionsSearch(keyword string, subject *string, limit int) (qbank.NotesMentionSearchResult, error) {
	return s.app.Qbank.NotesMentionsSearch(keyword, subject, limit)
}

func (s *QbankService) RebuildFTSIndex() (int, error) {
	return s.app.Qbank.RebuildFTSIndex(), nil
}

func (s *QbankService) GetQuestion(questionID string) (*qbank.Question, error) {
	return s.app.Qbank.GetQuestion(questionID)
}

func (s *QbankService) CreateQuestion(params qbank.CreateQuestionParams) (qbank.Question, error) {
	return s.app.Qbank.CreateQuestion(params)
}

func (s *QbankService) UpdateQuestion(request qbank.UpdateQuestionRequest) (qbank.Question, error) {
	return s.app.Qbank.UpdateQuestion(request)
}

func (s *QbankService) DeleteQuestion(questionID string) (bool, error) {
	return true, s.app.Qbank.DeleteQuestion(questionID)
}

func (s *QbankService) BatchDeleteQuestions(questionIDs []string) (qbank.BatchResult, error) {
	return s.app.Qbank.BatchDeleteQuestions(questionIDs)
}

func (s *QbankService) SubmitAnswer(request qbank.SubmitAnswerRequest) (qbank.SubmitAnswerResult, error) {
	return s.app.Qbank.SubmitAnswer(request)
}

func (s *QbankService) AIGrade(request qbank.QbankGradingRequest) (*qbank.QbankGradingResponse, error) {
	return s.app.Qbank.AIGrade(request)
}

func (s *QbankService) CancelGrading(streamEventName string) (bool, error) {
	return true, s.app.Qbank.CancelGrading(streamEventName)
}

func (s *QbankService) CheckSyncStatus(examID string) (qbank.SyncStatusResult, error) {
	return s.app.Qbank.CheckSyncStatus(examID)
}

func (s *QbankService) GetSyncConflicts(examID string) ([]qbank.SyncConflict, error) {
	return s.app.Qbank.GetSyncConflicts(examID)
}

func (s *QbankService) ResolveSyncConflict(conflictID string, strategy string) (qbank.Question, error) {
	return s.app.Qbank.ResolveSyncConflict(conflictID, strategy)
}

func (s *QbankService) BatchResolveConflicts(examID string, strategy string) ([]qbank.Question, error) {
	return s.app.Qbank.BatchResolveConflicts(examID, strategy)
}

func (s *QbankService) SetSyncEnabled(examID string, enabled bool) (bool, error) {
	return true, s.app.Qbank.SetSyncEnabled(examID, enabled)
}

func (s *QbankService) UpdateSyncConfig(examID string, config qbank.SyncConfig) (bool, error) {
	return true, s.app.Qbank.UpdateSyncConfig(examID, config)
}

func (s *QbankService) ToggleFavorite(questionID string) (qbank.Question, error) {
	return s.app.Qbank.ToggleFavorite(questionID)
}

func (s *QbankService) GetStats(examID string) (*qbank.QuestionBankStats, error) {
	return s.app.Qbank.GetStats(examID)
}

func (s *QbankService) RefreshStats(examID string) (qbank.QuestionBankStats, error) {
	return s.app.Qbank.RefreshStats(examID)
}

func (s *QbankService) ResetProgress(examID string) (qbank.QuestionBankStats, error) {
	return s.app.Qbank.ResetProgress(examID)
}

func (s *QbankService) ResetQuestionsProgress(questionIDs []string) (qbank.BatchResult, error) {
	return s.app.Qbank.ResetQuestionsProgress(questionIDs)
}

func (s *QbankService) GetHistory(questionID string, limit int) ([]qbank.QuestionHistory, error) {
	return s.app.Qbank.GetHistory(questionID, limit)
}

func (s *QbankService) GetSubmissions(questionID string, limit int) ([]qbank.AnswerSubmission, error) {
	return s.app.Qbank.GetSubmissions(questionID, limit)
}

func (s *QbankService) GetLearningTrend(request qbank.GetLearningTrendRequest) ([]qbank.LearningTrendPoint, error) {
	return s.app.Qbank.GetLearningTrend(request)
}

func (s *QbankService) GetActivityHeatmap(request qbank.GetActivityHeatmapRequest) ([]qbank.ActivityHeatmapPoint, error) {
	return s.app.Qbank.GetActivityHeatmap(request)
}

func (s *QbankService) GetKnowledgeStats(request qbank.GetKnowledgeStatsRequest) ([]qbank.KnowledgePoint, error) {
	return s.app.Qbank.GetKnowledgeStats(request)
}

func (s *QbankService) GetKnowledgeStatsWithComparison(request qbank.GetKnowledgeStatsRequest) (qbank.KnowledgeStatsComparison, error) {
	return s.app.Qbank.GetKnowledgeStatsWithComparison(request)
}

func (s *QbankService) StartTimedPractice(request qbank.StartTimedPracticeRequest) (qbank.TimedPracticeSession, error) {
	return s.app.Qbank.StartTimedPractice(request)
}

func (s *QbankService) GenerateMockExam(request qbank.GenerateMockExamRequest) (qbank.MockExamSession, error) {
	return s.app.Qbank.GenerateMockExam(request)
}

func (s *QbankService) SubmitMockExam(request qbank.SubmitMockExamRequest) (qbank.MockExamScoreCard, error) {
	return s.app.Qbank.SubmitMockExam(request)
}

func (s *QbankService) GetDailyPractice(request qbank.GetDailyPracticeRequest) (qbank.DailyPracticeResult, error) {
	return s.app.Qbank.GetDailyPractice(request)
}

func (s *QbankService) GeneratePaper(request qbank.GeneratePaperRequest) (qbank.GeneratedPaper, error) {
	return s.app.Qbank.GeneratePaper(request)
}

func (s *QbankService) GetCheckInCalendar(request qbank.GetCheckInCalendarRequest) (qbank.CheckInCalendar, error) {
	return s.app.Qbank.GetCheckInCalendar(request)
}

func (s *QbankService) GetCsvPreview(filePath string, rows int) (qbank.CsvPreviewResult, error) {
	return s.app.Qbank.GetCsvPreview(filePath, rows)
}

func (s *QbankService) ImportQuestionsCsv(request qbank.CsvImportRequest) (qbank.CsvImportResult, error) {
	return s.app.Qbank.ImportQuestionsCsv(request)
}

func (s *QbankService) ExportQuestionsCsv(request qbank.CsvExportRequest) (qbank.CsvExportResult, error) {
	return s.app.Qbank.ExportQuestionsCsv(request)
}

func (s *QbankService) GetCsvExportableFields() ([][]string, error) {
	return s.app.Qbank.GetCsvExportableFields(), nil
}
