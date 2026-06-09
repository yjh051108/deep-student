package bindings

import (
	"deep-student-go/internal/anki"
	"deep-student-go/internal/app"
)

type AnkiService struct {
	app *app.App
}

func NewAnkiService(app *app.App) *AnkiService {
	return &AnkiService{app: app}
}

func (s *AnkiService) CheckConnectStatus() (bool, error) {
	return s.app.Anki.CheckConnectStatus()
}

func (s *AnkiService) ListDeckNames() ([]string, error) {
	return s.app.Anki.ListDeckNames()
}

func (s *AnkiService) ListModelNames() ([]string, error) {
	return s.app.Anki.ListModelNames()
}

func (s *AnkiService) StartEnhancedDocumentProcessing(documentContent string, originalDocumentName string, options map[string]any) (string, error) {
	return s.app.Anki.StartEnhancedDocumentProcessing(documentContent, originalDocumentName, options)
}

func (s *AnkiService) GetDocumentTasks(documentID string) ([]anki.DocumentTask, error) {
	return s.app.Anki.GetDocumentTasks(documentID)
}

func (s *AnkiService) PauseDocumentProcessing(documentID string) (bool, error) {
	return s.app.Anki.PauseDocumentProcessing(documentID)
}

func (s *AnkiService) ResumeDocumentProcessing(documentID string) (bool, error) {
	return s.app.Anki.ResumeDocumentProcessing(documentID)
}

func (s *AnkiService) GetDocumentProcessingState(documentID string) (anki.DocumentProcessingState, error) {
	return s.app.Anki.GetDocumentProcessingState(documentID)
}

func (s *AnkiService) GetDocumentState(documentID string) (anki.DocumentProcessingState, error) {
	return s.app.Anki.GetDocumentState(documentID)
}

func (s *AnkiService) GetDocumentTaskCounts(documentID string) (anki.DocumentTaskCounts, error) {
	return s.app.Anki.GetDocumentTaskCounts(documentID)
}

func (s *AnkiService) TriggerTaskProcessing(taskID string) error {
	return s.app.Anki.TriggerTaskProcessing(taskID)
}

func (s *AnkiService) DeleteDocumentSession(documentID string) (bool, error) {
	return s.app.Anki.DeleteDocumentSession(documentID)
}

func (s *AnkiService) GetDocumentCards(documentID string) ([]map[string]any, error) {
	return s.app.Anki.GetDocumentCards(documentID)
}

func (s *AnkiService) SaveAnkiCards(request anki.SaveAnkiCardsRequest) (anki.SaveAnkiCardsResponse, error) {
	return s.app.Anki.SaveAnkiCards(request)
}

func (s *AnkiService) RecoverStuckDocumentTasks() (int, error) {
	return s.app.Anki.RecoverStuckDocumentTasks()
}
