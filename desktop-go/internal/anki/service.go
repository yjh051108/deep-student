package anki

import (
	"bytes"
	"context"
	"crypto/rand"
	"deep-student-go/internal/storage"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"
)

type Service struct {
	mu              sync.RWMutex
	eventMu         sync.RWMutex
	configMu        sync.RWMutex
	path            string
	emit            func(name string, payload any)
	apiConfigLoader func() (APIConfigState, error)
	httpClient      *http.Client
	ankiConnectURL  string
	state           store
}

var errDocumentIDRequired = errors.New("documentId is required")
var errDocumentContentRequired = errors.New("document content is required")

const defaultAnkiConnectURL = "http://127.0.0.1:8765"

type store struct {
	Documents []DocumentSession `json:"documents"`
}

type DocumentSession struct {
	DocumentID           string           `json:"documentId"`
	OriginalDocumentName string           `json:"originalDocumentName,omitempty"`
	Status               string           `json:"status"`
	Paused               bool             `json:"paused"`
	Tasks                []DocumentTask   `json:"tasks"`
	Cards                []map[string]any `json:"cards"`
	CreatedAt            string           `json:"createdAt"`
	UpdatedAt            string           `json:"updatedAt"`
	RecoveryNote         *string          `json:"recoveryNote,omitempty"`
}

type DocumentTask struct {
	ID                        string  `json:"id"`
	TaskID                    string  `json:"task_id,omitempty"`
	DocumentID                string  `json:"document_id"`
	OriginalDocumentName      string  `json:"original_document_name"`
	SegmentIndex              int     `json:"segment_index"`
	ContentSegment            string  `json:"content_segment"`
	Status                    string  `json:"status"`
	CreatedAt                 string  `json:"created_at"`
	UpdatedAt                 string  `json:"updated_at"`
	ErrorMessage              *string `json:"error_message,omitempty"`
	AnkiGenerationOptionsJSON string  `json:"anki_generation_options_json"`
	CardsGenerated            int     `json:"cards_generated,omitempty"`
	IsRetry                   bool    `json:"is_retry,omitempty"`
}

type DocumentProcessingState struct {
	Status         string `json:"status"`
	TotalTasks     int    `json:"total_tasks"`
	CompletedTasks int    `json:"completed_tasks"`
	FailedTasks    int    `json:"failed_tasks"`
	PausedTasks    int    `json:"paused_tasks"`
}

type DocumentTaskCounts struct {
	Total      int `json:"total"`
	Pending    int `json:"pending"`
	Processing int `json:"processing"`
	Streaming  int `json:"streaming"`
	Paused     int `json:"paused"`
	Completed  int `json:"completed"`
	Failed     int `json:"failed"`
	Truncated  int `json:"truncated"`
	Cancelled  int `json:"cancelled"`
}

type SaveAnkiCardPayload struct {
	ID         *string           `json:"id,omitempty"`
	Front      *string           `json:"front,omitempty"`
	Back       *string           `json:"back,omitempty"`
	Text       *string           `json:"text,omitempty"`
	Tags       []string          `json:"tags,omitempty"`
	Images     []string          `json:"images,omitempty"`
	Fields     map[string]string `json:"fields,omitempty"`
	Extra      map[string]string `json:"extra_fields,omitempty"`
	TemplateID *string           `json:"template_id,omitempty"`
}

type SaveAnkiCardsRequest struct {
	DocumentID        *string               `json:"document_id,omitempty"`
	BusinessSessionID *string               `json:"business_session_id,omitempty"`
	MessageStableID   *string               `json:"message_stable_id,omitempty"`
	BlockID           *string               `json:"block_id,omitempty"`
	TemplateID        *string               `json:"template_id,omitempty"`
	Cards             []SaveAnkiCardPayload `json:"cards"`
	Options           map[string]any        `json:"options,omitempty"`
}

type SaveAnkiCardsResponse struct {
	SavedIDs []string `json:"saved_ids"`
	TaskID   string   `json:"task_id"`
}

type APIConfigState struct {
	Configs               []ApiConfig
	AnkiCardModelConfigID string
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

func NewService(dataDir string) (*Service, error) {
	service := &Service{
		httpClient:     &http.Client{Timeout: 60 * time.Second},
		ankiConnectURL: defaultAnkiConnectURL,
		path:           filepath.Join(dataDir, "anki-go.json"),
		state: store{
			Documents: []DocumentSession{},
		},
	}
	if err := service.load(); err != nil {
		return nil, err
	}
	return service, nil
}

func (s *Service) SetEventEmitter(emit func(name string, payload any)) {
	s.eventMu.Lock()
	defer s.eventMu.Unlock()
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

func (s *Service) SetAnkiConnectURL(url string) {
	s.configMu.Lock()
	defer s.configMu.Unlock()
	trimmed := strings.TrimSpace(url)
	if trimmed == "" {
		trimmed = defaultAnkiConnectURL
	}
	s.ankiConnectURL = trimmed
}

func (s *Service) CheckConnectStatus() (bool, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if _, err := s.requestAnkiConnect(ctx, "version", nil); err != nil {
		return false, err
	}
	return true, nil
}

func (s *Service) ListDeckNames() ([]string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	return s.requestAnkiConnectStringList(ctx, "deckNames", "牌组列表")
}

func (s *Service) ListModelNames() ([]string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	return s.requestAnkiConnectStringList(ctx, "modelNames", "笔记类型列表")
}

func (s *Service) StartEnhancedDocumentProcessing(documentContent string, originalDocumentName string, options map[string]any) (string, error) {
	content := strings.TrimSpace(documentContent)
	if content == "" {
		return "", errDocumentContentRequired
	}
	const maxDocumentBytes = 10_000_000
	if len(documentContent) > maxDocumentBytes {
		return "", errors.New("document content is too large")
	}

	documentName := normalizeDocumentName(originalDocumentName)
	normalizedOptions := normalizeOptions(options)
	optionsJSON := mustJSON(normalizedOptions)
	segments := splitDocumentSegments(content, intOption(normalizedOptions, "segment_size", 1800))
	if len(segments) == 0 {
		return "", errDocumentContentRequired
	}

	now := nowISO()
	documentID := newID("doc")
	tasks := make([]DocumentTask, 0, len(segments))
	for index, segment := range segments {
		taskID := newID("task")
		tasks = append(tasks, DocumentTask{
			ID:                        taskID,
			TaskID:                    taskID,
			DocumentID:                documentID,
			OriginalDocumentName:      documentName,
			SegmentIndex:              index,
			ContentSegment:            segment,
			Status:                    "Pending",
			CreatedAt:                 now,
			UpdatedAt:                 now,
			AnkiGenerationOptionsJSON: optionsJSON,
		})
	}

	s.mu.Lock()
	s.state.Documents = append(s.state.Documents, DocumentSession{
		DocumentID:           documentID,
		OriginalDocumentName: documentName,
		Status:               "processing",
		Paused:               false,
		Tasks:                tasks,
		Cards:                []map[string]any{},
		CreatedAt:            now,
		UpdatedAt:            now,
	})
	err := s.flushLocked()
	s.mu.Unlock()
	if err != nil {
		return "", err
	}

	go s.processDocumentAsync(documentID, true, 30*time.Millisecond)
	return documentID, nil
}

func (s *Service) GetDocumentTasks(documentID string) ([]DocumentTask, error) {
	documentID = strings.TrimSpace(documentID)
	if documentID == "" {
		return []DocumentTask{}, nil
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	if index := s.findDocumentIndexLocked(documentID); index >= 0 {
		return copyTasks(s.state.Documents[index].Tasks), nil
	}
	return []DocumentTask{}, nil
}

func (s *Service) PauseDocumentProcessing(documentID string) (bool, error) {
	return s.setDocumentPaused(documentID, true)
}

func (s *Service) ResumeDocumentProcessing(documentID string) (bool, error) {
	ok, err := s.setDocumentPaused(documentID, false)
	if err == nil && ok && s.hasRunnablePendingTasks(strings.TrimSpace(documentID)) {
		go s.processDocumentAsync(strings.TrimSpace(documentID), false, 0)
	}
	return ok, err
}

func (s *Service) GetDocumentProcessingState(documentID string) (DocumentProcessingState, error) {
	return s.GetDocumentState(documentID)
}

func (s *Service) GetDocumentState(documentID string) (DocumentProcessingState, error) {
	documentID = strings.TrimSpace(documentID)
	if documentID == "" {
		return emptyDocumentState("pending"), nil
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	if index := s.findDocumentIndexLocked(documentID); index >= 0 {
		return documentState(s.state.Documents[index]), nil
	}
	return emptyDocumentState("pending"), nil
}

func (s *Service) GetDocumentTaskCounts(documentID string) (DocumentTaskCounts, error) {
	tasks, err := s.GetDocumentTasks(documentID)
	if err != nil {
		return DocumentTaskCounts{}, err
	}
	counts := DocumentTaskCounts{Total: len(tasks)}
	for _, task := range tasks {
		switch normalizeTaskStatus(task.Status) {
		case "Pending":
			counts.Pending++
		case "Processing":
			counts.Processing++
		case "Streaming":
			counts.Streaming++
		case "Paused":
			counts.Paused++
		case "Completed":
			counts.Completed++
		case "Failed":
			counts.Failed++
		case "Truncated":
			counts.Truncated++
		case "Cancelled":
			counts.Cancelled++
		}
	}
	return counts, nil
}

func (s *Service) TriggerTaskProcessing(taskID string) error {
	taskID = strings.TrimSpace(taskID)
	if taskID == "" {
		return nil
	}
	now := nowISO()
	documentID := ""
	s.mu.Lock()
	for documentIndex := range s.state.Documents {
		for taskIndex := range s.state.Documents[documentIndex].Tasks {
			task := &s.state.Documents[documentIndex].Tasks[taskIndex]
			if task.ID != taskID && task.TaskID != taskID {
				continue
			}
			task.Status = "Pending"
			task.UpdatedAt = now
			task.ErrorMessage = nil
			task.IsRetry = true
			s.state.Documents[documentIndex].UpdatedAt = now
			s.state.Documents[documentIndex].Paused = false
			s.state.Documents[documentIndex].Status = deriveDocumentStatus(s.state.Documents[documentIndex])
			documentID = s.state.Documents[documentIndex].DocumentID
			err := s.flushLocked()
			s.mu.Unlock()
			if err == nil && documentID != "" && s.hasRunnablePendingTasks(documentID) {
				go s.processDocumentAsync(documentID, false, 0)
			}
			return err
		}
	}
	s.mu.Unlock()
	return nil
}

func (s *Service) DeleteDocumentSession(documentID string) (bool, error) {
	documentID = strings.TrimSpace(documentID)
	if documentID == "" {
		return false, errDocumentIDRequired
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	index := s.findDocumentIndexLocked(documentID)
	if index < 0 {
		return true, nil
	}
	s.state.Documents = append(s.state.Documents[:index], s.state.Documents[index+1:]...)
	return true, s.flushLocked()
}

func (s *Service) GetDocumentCards(documentID string) ([]map[string]any, error) {
	documentID = strings.TrimSpace(documentID)
	if documentID == "" {
		return []map[string]any{}, nil
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	if index := s.findDocumentIndexLocked(documentID); index >= 0 {
		return copyCards(s.state.Documents[index].Cards), nil
	}
	return []map[string]any{}, nil
}

func (s *Service) SaveAnkiCards(request SaveAnkiCardsRequest) (SaveAnkiCardsResponse, error) {
	if len(request.Cards) == 0 {
		return SaveAnkiCardsResponse{}, errors.New("No cards provided for saving")
	}

	documentID := firstNonEmptyString(
		stringPtrValue(request.DocumentID),
		stringPtrValue(request.BlockID),
		stringPtrValue(request.MessageStableID),
	)
	if documentID == "" {
		documentID = newID("doc")
	}
	taskID := newID("task")
	now := nowISO()
	options := normalizeOptions(request.Options)
	optionsJSON := mustJSON(options)
	contentSegment := saveCardsContentSegment(request)

	cards := make([]map[string]any, 0, len(request.Cards))
	savedIDs := make([]string, 0, len(request.Cards))
	for index, payload := range request.Cards {
		card, cardID := normalizeSavedCardPayload(payload, request.TemplateID, taskID, index, now)
		cards = append(cards, card)
		savedIDs = append(savedIDs, cardID)
	}
	if len(savedIDs) == 0 {
		return SaveAnkiCardsResponse{}, errors.New("未能保存任何卡片，请检查输入数据")
	}

	task := DocumentTask{
		ID:                        taskID,
		TaskID:                    taskID,
		DocumentID:                documentID,
		OriginalDocumentName:      "Chat Cards",
		SegmentIndex:              0,
		ContentSegment:            contentSegment,
		Status:                    "Completed",
		CreatedAt:                 now,
		UpdatedAt:                 now,
		AnkiGenerationOptionsJSON: optionsJSON,
		CardsGenerated:            len(cards),
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	index := s.findDocumentIndexLocked(documentID)
	if index < 0 {
		s.state.Documents = append(s.state.Documents, DocumentSession{
			DocumentID:           documentID,
			OriginalDocumentName: "Chat Cards",
			Status:               "completed",
			Paused:               false,
			Tasks:                []DocumentTask{task},
			Cards:                cards,
			CreatedAt:            now,
			UpdatedAt:            now,
		})
	} else {
		document := &s.state.Documents[index]
		document.Tasks = upsertTask(document.Tasks, task)
		document.Cards = upsertCards(document.Cards, cards)
		document.Status = deriveDocumentStatus(*document)
		document.UpdatedAt = now
		if strings.TrimSpace(document.OriginalDocumentName) == "" {
			document.OriginalDocumentName = "Chat Cards"
		}
	}
	if err := s.flushLocked(); err != nil {
		return SaveAnkiCardsResponse{}, err
	}
	return SaveAnkiCardsResponse{SavedIDs: savedIDs, TaskID: taskID}, nil
}

func (s *Service) RecoverStuckDocumentTasks() (int, error) {
	now := nowISO()
	recovered := 0
	s.mu.Lock()
	defer s.mu.Unlock()
	for documentIndex := range s.state.Documents {
		documentRecovered := 0
		for taskIndex := range s.state.Documents[documentIndex].Tasks {
			task := &s.state.Documents[documentIndex].Tasks[taskIndex]
			switch normalizeTaskStatus(task.Status) {
			case "Processing", "Streaming":
				task.Status = "Pending"
				task.UpdatedAt = now
				recovered++
				documentRecovered++
			}
		}
		if documentRecovered > 0 {
			s.state.Documents[documentIndex].UpdatedAt = now
			s.state.Documents[documentIndex].Status = deriveDocumentStatus(s.state.Documents[documentIndex])
		}
	}
	if recovered == 0 {
		return 0, nil
	}
	return recovered, s.flushLocked()
}

func (s *Service) setDocumentPaused(documentID string, paused bool) (bool, error) {
	documentID = strings.TrimSpace(documentID)
	if documentID == "" {
		return false, errDocumentIDRequired
	}
	now := nowISO()
	s.mu.Lock()
	defer s.mu.Unlock()
	index := s.findDocumentIndexLocked(documentID)
	if index < 0 {
		status := "pending"
		if paused {
			status = "paused"
		}
		s.state.Documents = append(s.state.Documents, DocumentSession{
			DocumentID: documentID,
			Status:     status,
			Paused:     paused,
			Tasks:      []DocumentTask{},
			Cards:      []map[string]any{},
			CreatedAt:  now,
			UpdatedAt:  now,
		})
		return true, s.flushLocked()
	}
	document := &s.state.Documents[index]
	document.Paused = paused
	for taskIndex := range document.Tasks {
		task := &document.Tasks[taskIndex]
		switch normalizeTaskStatus(task.Status) {
		case "Pending", "Processing", "Streaming", "Paused":
			if paused {
				task.Status = "Paused"
			} else {
				task.Status = "Pending"
			}
			task.UpdatedAt = now
		}
	}
	document.UpdatedAt = now
	if len(document.Tasks) == 0 {
		if paused {
			document.Status = "paused"
		} else {
			document.Status = "pending"
		}
	} else {
		document.Status = deriveDocumentStatus(*document)
	}
	return true, s.flushLocked()
}

func (s *Service) processDocumentAsync(documentID string, emitStart bool, initialDelay time.Duration) {
	documentID = strings.TrimSpace(documentID)
	if documentID == "" {
		return
	}
	if initialDelay > 0 {
		time.Sleep(initialDelay)
	}

	if emitStart {
		totalSegments := s.documentTaskTotal(documentID)
		if totalSegments > 0 {
			s.emitAnki(map[string]any{
				"DocumentProcessingStarted": map[string]any{
					"document_id":    documentID,
					"total_segments": totalSegments,
				},
			})
		}
	}

	for {
		task, ok, paused := s.claimNextPendingTask(documentID)
		if paused {
			s.emitAnki(map[string]any{
				"DocumentProcessingPaused": map[string]any{"document_id": documentID},
			})
			return
		}
		if !ok {
			state, err := s.GetDocumentState(documentID)
			if err == nil && state.Status == "completed" {
				s.emitAnki(map[string]any{
					"DocumentProcessingCompleted": map[string]any{"document_id": documentID},
				})
			}
			return
		}

		s.emitAnki(map[string]any{
			"TaskStatusUpdate": map[string]any{
				"task_id":       task.ID,
				"status":        "Processing",
				"segment_index": task.SegmentIndex,
				"document_id":   documentID,
			},
		})

		cards := s.generateCardsForTask(task)
		if len(cards) == 0 {
			message := "no usable learning unit was found in this segment"
			s.failTask(documentID, task.ID, message)
			s.emitAnki(map[string]any{
				"TaskProcessingError": map[string]any{
					"task_id":       task.ID,
					"error_message": message,
					"document_id":   documentID,
				},
			})
			continue
		}

		if paused := s.completeTask(documentID, task.ID, cards); paused {
			s.emitAnki(map[string]any{
				"DocumentProcessingPaused": map[string]any{"document_id": documentID},
			})
			return
		}
		for _, card := range cards {
			s.emitAnki(map[string]any{
				"NewCard": map[string]any{
					"card":        card,
					"document_id": documentID,
				},
			})
		}
		s.emitAnki(map[string]any{
			"TaskCompleted": map[string]any{
				"task_id":               task.ID,
				"final_status":          "Completed",
				"total_cards_generated": len(cards),
				"document_id":           documentID,
			},
		})
	}
}

func (s *Service) documentTaskTotal(documentID string) int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if index := s.findDocumentIndexLocked(documentID); index >= 0 {
		return len(s.state.Documents[index].Tasks)
	}
	return 0
}

func (s *Service) hasRunnablePendingTasks(documentID string) bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	index := s.findDocumentIndexLocked(documentID)
	if index < 0 || s.state.Documents[index].Paused {
		return false
	}
	for _, task := range s.state.Documents[index].Tasks {
		if normalizeTaskStatus(task.Status) == "Pending" && strings.TrimSpace(task.ContentSegment) != "" {
			return true
		}
	}
	return false
}

func (s *Service) claimNextPendingTask(documentID string) (DocumentTask, bool, bool) {
	now := nowISO()
	s.mu.Lock()
	defer s.mu.Unlock()
	index := s.findDocumentIndexLocked(documentID)
	if index < 0 {
		return DocumentTask{}, false, false
	}
	document := &s.state.Documents[index]
	if document.Paused {
		return DocumentTask{}, false, true
	}
	for taskIndex := range document.Tasks {
		task := &document.Tasks[taskIndex]
		if normalizeTaskStatus(task.Status) != "Pending" {
			continue
		}
		task.Status = "Processing"
		task.UpdatedAt = now
		document.Status = "processing"
		document.UpdatedAt = now
		_ = s.flushLocked()
		return *task, true, false
	}
	document.Status = deriveDocumentStatus(*document)
	document.UpdatedAt = now
	_ = s.flushLocked()
	return DocumentTask{}, false, false
}

func (s *Service) completeTask(documentID string, taskID string, cards []map[string]any) bool {
	now := nowISO()
	s.mu.Lock()
	defer s.mu.Unlock()
	index := s.findDocumentIndexLocked(documentID)
	if index < 0 {
		return false
	}
	document := &s.state.Documents[index]
	if document.Paused {
		for taskIndex := range document.Tasks {
			if document.Tasks[taskIndex].ID == taskID {
				document.Tasks[taskIndex].Status = "Paused"
				document.Tasks[taskIndex].UpdatedAt = now
				break
			}
		}
		document.Status = "paused"
		document.UpdatedAt = now
		_ = s.flushLocked()
		return true
	}
	document.Cards = append(document.Cards, copyCards(cards)...)
	for taskIndex := range document.Tasks {
		task := &document.Tasks[taskIndex]
		if task.ID != taskID {
			continue
		}
		task.Status = "Completed"
		task.CardsGenerated = len(cards)
		task.UpdatedAt = now
		task.ErrorMessage = nil
		break
	}
	document.Status = deriveDocumentStatus(*document)
	document.UpdatedAt = now
	_ = s.flushLocked()
	return false
}

func (s *Service) failTask(documentID string, taskID string, message string) {
	now := nowISO()
	s.mu.Lock()
	defer s.mu.Unlock()
	index := s.findDocumentIndexLocked(documentID)
	if index < 0 {
		return
	}
	document := &s.state.Documents[index]
	for taskIndex := range document.Tasks {
		task := &document.Tasks[taskIndex]
		if task.ID != taskID {
			continue
		}
		task.Status = "Failed"
		task.ErrorMessage = &message
		task.UpdatedAt = now
		break
	}
	document.Status = deriveDocumentStatus(*document)
	document.UpdatedAt = now
	_ = s.flushLocked()
}

func (s *Service) emitAnki(payload any) {
	emit := s.currentEmitter()
	if emit == nil {
		return
	}
	emit("anki_generation_event", payload)
}

func (s *Service) currentEmitter() func(name string, payload any) {
	s.eventMu.RLock()
	defer s.eventMu.RUnlock()
	return s.emit
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

func (s *Service) currentAnkiConnectURL() string {
	s.configMu.RLock()
	defer s.configMu.RUnlock()
	if strings.TrimSpace(s.ankiConnectURL) == "" {
		return defaultAnkiConnectURL
	}
	return s.ankiConnectURL
}

func (s *Service) requestAnkiConnectStringList(ctx context.Context, action string, label string) ([]string, error) {
	result, err := s.requestAnkiConnect(ctx, action, nil)
	if err != nil {
		return []string{}, err
	}
	if result == nil || string(*result) == "null" {
		return []string{}, errors.New("AnkiConnect返回空结果")
	}
	var names []string
	if err := json.Unmarshal(*result, &names); err != nil {
		return []string{}, fmt.Errorf("解析%s失败: %w", label, err)
	}
	return names, nil
}

func (s *Service) requestAnkiConnect(ctx context.Context, action string, params any) (*json.RawMessage, error) {
	body := struct {
		Action  string `json:"action"`
		Version int    `json:"version"`
		Params  any    `json:"params,omitempty"`
	}{
		Action:  action,
		Version: 6,
		Params:  params,
	}
	requestBody, err := json.Marshal(body)
	if err != nil {
		return nil, err
	}
	request, err := http.NewRequestWithContext(ctx, http.MethodPost, s.currentAnkiConnectURL(), bytes.NewReader(requestBody))
	if err != nil {
		return nil, err
	}
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("Accept", "application/json")
	request.Header.Set("User-Agent", "DeepStudent-Go/1.0")

	response, err := s.currentHTTPClient().Do(request)
	if err != nil {
		return nil, fmt.Errorf("请求AnkiConnect失败: %w", err)
	}
	defer response.Body.Close()
	bytes, err := io.ReadAll(io.LimitReader(response.Body, 1024*1024))
	if err != nil {
		return nil, err
	}
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		return nil, fmt.Errorf("AnkiConnect HTTP错误: %s - 内容: %s", response.Status, strings.TrimSpace(string(bytes)))
	}

	var decoded struct {
		Result *json.RawMessage `json:"result"`
		Error  any              `json:"error"`
	}
	if err := json.Unmarshal(bytes, &decoded); err != nil {
		return nil, fmt.Errorf("解析AnkiConnect响应失败: %w", err)
	}
	if message := ankiConnectErrorMessage(decoded.Error); message != "" {
		return nil, fmt.Errorf("AnkiConnect错误: %s", message)
	}
	return decoded.Result, nil
}

func ankiConnectErrorMessage(value any) string {
	switch typed := value.(type) {
	case nil:
		return ""
	case string:
		return strings.TrimSpace(typed)
	default:
		bytes, err := json.Marshal(typed)
		if err == nil {
			return strings.TrimSpace(string(bytes))
		}
		return strings.TrimSpace(fmt.Sprint(typed))
	}
}

func (s *Service) load() error {
	data, err := os.ReadFile(s.path)
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	if err != nil {
		return err
	}
	if len(data) == 0 {
		return nil
	}
	if err := json.Unmarshal(data, &s.state); err != nil {
		return err
	}
	if s.state.Documents == nil {
		s.state.Documents = []DocumentSession{}
	}
	return nil
}

func (s *Service) flushLocked() error {
	return storage.WriteJSONAtomic(s.path, s.state)
}

func (s *Service) findDocumentIndexLocked(documentID string) int {
	for index, document := range s.state.Documents {
		if document.DocumentID == documentID {
			return index
		}
	}
	return -1
}

func documentState(document DocumentSession) DocumentProcessingState {
	counts := DocumentProcessingState{Status: deriveDocumentStatus(document)}
	counts.TotalTasks = len(document.Tasks)
	for _, task := range document.Tasks {
		switch normalizeTaskStatus(task.Status) {
		case "Completed":
			counts.CompletedTasks++
		case "Failed", "Truncated":
			counts.FailedTasks++
		case "Paused":
			counts.PausedTasks++
		}
	}
	return counts
}

func emptyDocumentState(status string) DocumentProcessingState {
	return DocumentProcessingState{Status: status}
}

func deriveDocumentStatus(document DocumentSession) string {
	if document.Paused {
		return "paused"
	}
	if len(document.Tasks) == 0 {
		status := strings.TrimSpace(document.Status)
		if status != "" {
			return strings.ToLower(status)
		}
		return "pending"
	}
	completed := 0
	failed := 0
	paused := 0
	inFlight := 0
	for _, task := range document.Tasks {
		switch normalizeTaskStatus(task.Status) {
		case "Completed":
			completed++
		case "Failed", "Truncated":
			failed++
		case "Paused":
			paused++
		case "Processing", "Streaming":
			inFlight++
		}
	}
	if completed == len(document.Tasks) {
		return "completed"
	}
	if failed == len(document.Tasks) {
		return "failed"
	}
	if paused > 0 {
		return "paused"
	}
	if inFlight > 0 {
		return "processing"
	}
	return "pending"
}

func normalizeTaskStatus(value string) string {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "processing":
		return "Processing"
	case "streaming":
		return "Streaming"
	case "paused":
		return "Paused"
	case "completed":
		return "Completed"
	case "failed":
		return "Failed"
	case "truncated":
		return "Truncated"
	case "cancelled", "canceled":
		return "Cancelled"
	default:
		return "Pending"
	}
}

func normalizeOptions(options map[string]any) map[string]any {
	if options == nil {
		options = map[string]any{}
	}
	normalized := make(map[string]any, len(options)+4)
	for key, value := range options {
		normalized[key] = value
	}
	if _, ok := normalized["max_cards_per_mistake"]; !ok {
		normalized["max_cards_per_mistake"] = 10
	}
	if _, ok := normalized["note_type"]; !ok {
		normalized["note_type"] = "Basic"
	}
	if _, ok := normalized["deck_name"]; !ok {
		normalized["deck_name"] = "Default"
	}
	return normalized
}

func normalizeDocumentName(value string) string {
	name := strings.TrimSpace(value)
	name = strings.ReplaceAll(name, "../", "")
	name = strings.ReplaceAll(name, "..\\", "")
	name = strings.ReplaceAll(name, "./", "")
	if name == "" {
		return "Document"
	}
	if len(name) > 255 {
		return name[:255]
	}
	return name
}

func splitDocumentSegments(content string, targetSize int) []string {
	if targetSize < 600 {
		targetSize = 600
	}
	if targetSize > 4000 {
		targetSize = 4000
	}
	paragraphs := strings.FieldsFunc(content, func(r rune) bool {
		return r == '\n' || r == '\r'
	})
	segments := []string{}
	var current strings.Builder
	flush := func() {
		segment := strings.TrimSpace(current.String())
		if segment != "" {
			segments = append(segments, segment)
		}
		current.Reset()
	}
	for _, paragraph := range paragraphs {
		paragraph = strings.TrimSpace(paragraph)
		if paragraph == "" {
			continue
		}
		if current.Len() > 0 && current.Len()+len(paragraph)+1 > targetSize {
			flush()
		}
		if len(paragraph) > targetSize {
			for len(paragraph) > targetSize {
				segments = append(segments, strings.TrimSpace(paragraph[:targetSize]))
				paragraph = paragraph[targetSize:]
			}
			if strings.TrimSpace(paragraph) == "" {
				continue
			}
		}
		if current.Len() > 0 {
			current.WriteString("\n")
		}
		current.WriteString(paragraph)
	}
	flush()
	return segments
}

func (s *Service) generateCardsForTask(task DocumentTask) []map[string]any {
	if cards, err := s.generateProviderCardsForTask(task); err == nil && len(cards) > 0 {
		return cards
	}
	return generateLocalCardsForTask(task)
}

func generateLocalCardsForTask(task DocumentTask) []map[string]any {
	options := map[string]any{}
	if task.AnkiGenerationOptionsJSON != "" {
		_ = json.Unmarshal([]byte(task.AnkiGenerationOptionsJSON), &options)
	}
	maxCards := intOption(options, "max_cards_per_mistake", 10)
	if maxCards < 1 {
		maxCards = 1
	}
	if maxCards > 100 {
		maxCards = 100
	}
	templateID := stringOption(options, "template_id")
	if templateID == "" {
		templateIDs := stringArrayOption(options, "template_ids")
		if len(templateIDs) > 0 {
			templateID = templateIDs[0]
		}
	}

	units := learningUnits(task.ContentSegment)
	if len(units) == 0 {
		return []map[string]any{}
	}
	if len(units) > maxCards {
		units = units[:maxCards]
	}

	now := nowISO()
	cards := make([]map[string]any, 0, len(units))
	for index, unit := range units {
		front := "Summarize the key learning point from document segment " + intString(task.SegmentIndex+1) + ":\n\n" + truncateText(unit, 260)
		back := unit
		card := map[string]any{
			"id":            newID("card"),
			"task_id":       task.ID,
			"front":         front,
			"back":          back,
			"text":          unit,
			"tags":          []string{"deep-student", "document", "segment-" + intString(task.SegmentIndex+1)},
			"images":        []string{},
			"is_error_card": false,
			"created_at":    now,
			"updated_at":    now,
			"extra_fields": map[string]string{
				"Front": front,
				"Back":  back,
				"Text":  unit,
			},
		}
		if templateID != "" {
			card["template_id"] = templateID
		}
		if index == 0 {
			card["source"] = "go_local_document_worker"
		}
		cards = append(cards, card)
	}
	return cards
}

func normalizeSavedCardPayload(payload SaveAnkiCardPayload, requestTemplateID *string, taskID string, index int, now string) (map[string]any, string) {
	fields := mergeStringMaps(payload.Extra, payload.Fields)
	front := firstNonEmptyString(stringPtrValue(payload.Front), fields["Front"])
	if front == "" {
		front = "Chat card " + intString(index+1)
	}
	back := firstNonEmptyString(stringPtrValue(payload.Back), fields["Back"])
	text := firstNonEmptyString(stringPtrValue(payload.Text), fields["Text"])
	if text == "" {
		text = back
	}
	fields["Front"] = front
	fields["Back"] = back
	if text != "" {
		fields["Text"] = text
	}

	cardID := strings.TrimSpace(stringPtrValue(payload.ID))
	if cardID == "" {
		cardID = newID("card")
	}
	templateID := firstNonEmptyString(stringPtrValue(payload.TemplateID), stringPtrValue(requestTemplateID))
	card := map[string]any{
		"id":            cardID,
		"task_id":       taskID,
		"front":         front,
		"back":          back,
		"text":          text,
		"tags":          cleanStringSlice(payload.Tags),
		"images":        cleanStringSlice(payload.Images),
		"is_error_card": false,
		"created_at":    now,
		"updated_at":    now,
		"extra_fields":  fields,
		"fields":        fields,
		"source":        "go_save_anki_cards",
	}
	if templateID != "" {
		card["template_id"] = templateID
	}
	return card, cardID
}

func saveCardsContentSegment(request SaveAnkiCardsRequest) string {
	return firstNonEmptyString(
		prefixedValue("chat_document", stringPtrValue(request.DocumentID)),
		prefixedValue("chat_block", stringPtrValue(request.BlockID)),
		prefixedValue("chat_message", stringPtrValue(request.MessageStableID)),
		prefixedValue("chat_session", stringPtrValue(request.BusinessSessionID)),
		"chat_session:anonymous",
	)
}

func prefixedValue(prefix string, value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return ""
	}
	return prefix + ":" + value
}

func mergeStringMaps(maps ...map[string]string) map[string]string {
	out := map[string]string{}
	for _, values := range maps {
		for key, value := range values {
			key = strings.TrimSpace(key)
			if key == "" {
				continue
			}
			out[key] = value
		}
	}
	return out
}

func cleanStringSlice(values []string) []string {
	if len(values) == 0 {
		return []string{}
	}
	out := make([]string, 0, len(values))
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value != "" {
			out = append(out, value)
		}
	}
	return out
}

func stringPtrValue(value *string) string {
	if value == nil {
		return ""
	}
	return strings.TrimSpace(*value)
}

func upsertTask(tasks []DocumentTask, task DocumentTask) []DocumentTask {
	for index := range tasks {
		if tasks[index].ID == task.ID || tasks[index].TaskID == task.TaskID {
			tasks[index] = task
			return tasks
		}
	}
	return append(tasks, task)
}

func upsertCards(existing []map[string]any, cards []map[string]any) []map[string]any {
	out := copyCards(existing)
	for _, card := range cards {
		cardID := mapString(card, "id")
		if cardID == "" {
			out = append(out, copyCard(card))
			continue
		}
		replaced := false
		for index := range out {
			if mapString(out[index], "id") == cardID {
				out[index] = copyCard(card)
				replaced = true
				break
			}
		}
		if !replaced {
			out = append(out, copyCard(card))
		}
	}
	return out
}

func mapString(values map[string]any, key string) string {
	if value, ok := values[key].(string); ok {
		return strings.TrimSpace(value)
	}
	return ""
}

func (s *Service) generateProviderCardsForTask(task DocumentTask) ([]map[string]any, error) {
	options := decodeTaskOptions(task)
	config, ok, err := s.resolveAnkiAPIConfig(options)
	if err != nil || !ok {
		return []map[string]any{}, err
	}

	ctx, cancel := context.WithTimeout(context.Background(), 45*time.Second)
	defer cancel()

	content, err := s.requestProviderCards(ctx, config, task, options)
	if err != nil {
		return []map[string]any{}, err
	}
	return parseProviderCards(content, task, options, config), nil
}

func (s *Service) resolveAnkiAPIConfig(options map[string]any) (ApiConfig, bool, error) {
	loader := s.currentAPIConfigLoader()
	if loader == nil {
		return ApiConfig{}, false, nil
	}
	state, err := loader()
	if err != nil {
		return ApiConfig{}, false, err
	}

	preferredIDs := ankiPreferredConfigIDs(options, state.AnkiCardModelConfigID)
	for _, preferredID := range preferredIDs {
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

func (s *Service) requestProviderCards(ctx context.Context, config ApiConfig, task DocumentTask, options map[string]any) (string, error) {
	baseURL := strings.TrimRight(strings.TrimSpace(config.BaseUrl), "/")
	if baseURL == "" {
		return "", errors.New("api baseUrl is required")
	}
	endpoint := baseURL
	if !strings.HasSuffix(endpoint, "/chat/completions") {
		endpoint += "/chat/completions"
	}

	maxCards := maxCardsFromOptions(options)
	maxTokens := intOption(options, "maxTokens", 0)
	if maxTokens <= 0 {
		maxTokens = intOption(options, "max_tokens", 0)
	}
	if maxTokens <= 0 && config.MaxOutputTokens > 0 {
		maxTokens = int(config.MaxOutputTokens)
	}
	if maxTokens <= 0 {
		maxTokens = 1600
	}

	body := map[string]any{
		"model":  strings.TrimSpace(config.Model),
		"stream": false,
		"messages": []map[string]string{
			{
				"role":    "system",
				"content": "You generate high quality Anki flashcards from study notes. Return JSON only in this exact shape: {\"cards\":[{\"front\":\"question\",\"back\":\"answer\",\"tags\":[\"tag\"]}]}. Do not include markdown fences.",
			},
			{
				"role":    "user",
				"content": providerCardPrompt(task, options, maxCards),
			},
		},
		"max_tokens": maxTokens,
	}
	if temperature, ok := floatOption(options, "temperature"); ok {
		body["temperature"] = temperature
	} else if config.Temperature > 0 {
		body["temperature"] = config.Temperature
	} else {
		body["temperature"] = 0.2
	}

	requestBody, err := json.Marshal(body)
	if err != nil {
		return "", err
	}
	request, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(requestBody))
	if err != nil {
		return "", err
	}
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("Authorization", "Bearer "+strings.TrimSpace(config.ApiKey))
	for key, value := range config.Headers {
		key = strings.TrimSpace(key)
		if key == "" {
			continue
		}
		request.Header.Set(key, value)
	}

	response, err := s.currentHTTPClient().Do(request)
	if err != nil {
		return "", err
	}
	defer response.Body.Close()
	bytes, err := io.ReadAll(io.LimitReader(response.Body, 2*1024*1024))
	if err != nil {
		return "", err
	}
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		return "", fmt.Errorf("provider returned %s: %s", response.Status, strings.TrimSpace(string(bytes)))
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
	if err := json.Unmarshal(bytes, &decoded); err != nil {
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

func providerCardPrompt(task DocumentTask, options map[string]any, maxCards int) string {
	documentName := strings.TrimSpace(task.OriginalDocumentName)
	if documentName == "" {
		documentName = "Document"
	}
	deckName := stringOption(options, "deck_name")
	if deckName == "" {
		deckName = "Default"
	}
	return strings.Join([]string{
		"Document: " + documentName,
		"Deck: " + deckName,
		"Segment index: " + intString(task.SegmentIndex+1),
		"Generate at most " + intString(maxCards) + " cards.",
		"Prefer atomic question/answer cards. Avoid cloze markup unless the source demands it.",
		"Use concise Chinese or English matching the source language.",
		"Source segment:",
		truncateText(task.ContentSegment, 6000),
	}, "\n")
}

func parseProviderCards(content string, task DocumentTask, options map[string]any, config ApiConfig) []map[string]any {
	var payload any
	if err := json.Unmarshal([]byte(extractProviderJSON(content)), &payload); err != nil {
		return []map[string]any{}
	}
	items := providerCardItems(payload)
	if len(items) == 0 {
		return []map[string]any{}
	}
	maxCards := maxCardsFromOptions(options)
	if len(items) > maxCards {
		items = items[:maxCards]
	}

	cards := make([]map[string]any, 0, len(items))
	for index, item := range items {
		card := normalizeProviderCard(item, task, options, config, index)
		if len(card) > 0 {
			cards = append(cards, card)
		}
	}
	return cards
}

func normalizeProviderCard(item map[string]any, task DocumentTask, options map[string]any, config ApiConfig, index int) map[string]any {
	front := firstStringFromMap(item, "front", "Front", "question", "Question", "q", "prompt", "Prompt")
	back := firstStringFromMap(item, "back", "Back", "answer", "Answer", "a", "explanation", "Explanation")
	if front == "" || back == "" {
		return map[string]any{}
	}

	now := nowISO()
	tags := mergeTags(
		[]string{"deep-student", "document", "provider", "segment-" + intString(task.SegmentIndex+1)},
		stringSliceFromMap(item, "tags", "Tags"),
	)
	card := map[string]any{
		"id":                 newID("card"),
		"task_id":            task.ID,
		"front":              front,
		"back":               back,
		"text":               firstNonEmptyString(firstStringFromMap(item, "text", "Text", "source", "Source"), back),
		"tags":               tags,
		"images":             []string{},
		"is_error_card":      false,
		"created_at":         now,
		"updated_at":         now,
		"source":             "go_provider_document_worker",
		"provider_config_id": config.ID,
		"provider_model":     config.Model,
		"extra_fields": map[string]string{
			"Front": front,
			"Back":  back,
			"Text":  firstNonEmptyString(firstStringFromMap(item, "text", "Text", "source", "Source"), back),
		},
	}
	if templateID := templateIDFromOptions(options); templateID != "" {
		card["template_id"] = templateID
	}
	if index == 0 && config.Name != "" {
		card["provider_name"] = config.Name
	}
	return card
}

func learningUnits(content string) []string {
	normalized := strings.Join(strings.Fields(content), " ")
	if normalized == "" {
		return []string{}
	}
	parts := strings.FieldsFunc(normalized, func(r rune) bool {
		return r == '。' || r == '！' || r == '？' || r == '.' || r == '!' || r == '?' || r == ';' || r == '；'
	})
	units := []string{}
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if len(part) < 12 {
			continue
		}
		units = append(units, part)
	}
	if len(units) == 0 {
		return []string{truncateText(normalized, 500)}
	}
	return units
}

func decodeTaskOptions(task DocumentTask) map[string]any {
	options := map[string]any{}
	if task.AnkiGenerationOptionsJSON != "" {
		_ = json.Unmarshal([]byte(task.AnkiGenerationOptionsJSON), &options)
	}
	return options
}

func maxCardsFromOptions(options map[string]any) int {
	maxCards := intOption(options, "max_cards_per_mistake", 10)
	if maxCards < 1 {
		return 1
	}
	if maxCards > 100 {
		return 100
	}
	return maxCards
}

func templateIDFromOptions(options map[string]any) string {
	templateID := stringOption(options, "template_id")
	if templateID != "" {
		return templateID
	}
	templateIDs := stringArrayOption(options, "template_ids")
	if len(templateIDs) == 0 {
		return ""
	}
	return templateIDs[0]
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

func ankiPreferredConfigIDs(options map[string]any, assignmentID string) []string {
	values := []string{
		stringOption(options, "model_config_id"),
		stringOption(options, "modelConfigId"),
		stringOption(options, "anki_card_model_config_id"),
		stringOption(options, "ankiCardModelConfigId"),
		stringOption(options, "modelId"),
		stringOption(options, "model_id"),
		stringOption(options, "configId"),
		strings.TrimSpace(assignmentID),
	}
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
				text := firstStringFromMap(object, "text", "content")
				if text != "" {
					parts = append(parts, text)
				}
			}
		}
		return strings.TrimSpace(strings.Join(parts, ""))
	case map[string]any:
		return firstStringFromMap(typed, "text", "content")
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
	start := strings.IndexAny(cleaned, "[{")
	if start < 0 {
		return cleaned
	}
	end := -1
	if cleaned[start] == '[' {
		end = strings.LastIndex(cleaned, "]")
	} else {
		end = strings.LastIndex(cleaned, "}")
	}
	if end <= start {
		return cleaned
	}
	return strings.TrimSpace(cleaned[start : end+1])
}

func providerCardItems(payload any) []map[string]any {
	switch typed := payload.(type) {
	case []any:
		out := make([]map[string]any, 0, len(typed))
		for _, item := range typed {
			if object, ok := item.(map[string]any); ok {
				out = append(out, object)
			}
		}
		return out
	case map[string]any:
		if cards, ok := typed["cards"].([]any); ok {
			return providerCardItems(cards)
		}
		if _, hasFront := typed["front"]; hasFront {
			return []map[string]any{typed}
		}
		if _, hasQuestion := typed["question"]; hasQuestion {
			return []map[string]any{typed}
		}
	}
	return []map[string]any{}
}

func firstStringFromMap(values map[string]any, keys ...string) string {
	for _, key := range keys {
		value, ok := values[key]
		if !ok {
			continue
		}
		switch typed := value.(type) {
		case string:
			if strings.TrimSpace(typed) != "" {
				return strings.TrimSpace(typed)
			}
		case []any:
			parts := []string{}
			for _, item := range typed {
				if text, ok := item.(string); ok && strings.TrimSpace(text) != "" {
					parts = append(parts, strings.TrimSpace(text))
				}
			}
			if len(parts) > 0 {
				return strings.Join(parts, "\n")
			}
		}
	}
	return ""
}

func stringSliceFromMap(values map[string]any, keys ...string) []string {
	for _, key := range keys {
		value, ok := values[key]
		if !ok {
			continue
		}
		switch typed := value.(type) {
		case []string:
			return typed
		case []any:
			out := []string{}
			for _, item := range typed {
				if text, ok := item.(string); ok {
					out = append(out, text)
				}
			}
			return out
		case string:
			return strings.FieldsFunc(typed, func(r rune) bool {
				return r == ',' || r == ';' || r == '，' || r == '；'
			})
		}
	}
	return []string{}
}

func mergeTags(groups ...[]string) []string {
	out := []string{}
	seen := map[string]bool{}
	for _, group := range groups {
		for _, tag := range group {
			tag = strings.TrimSpace(tag)
			if tag == "" || seen[tag] {
				continue
			}
			seen[tag] = true
			out = append(out, tag)
		}
	}
	return out
}

func firstNonEmptyString(values ...string) string {
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value != "" {
			return value
		}
	}
	return ""
}

func intOption(options map[string]any, key string, fallback int) int {
	value, ok := options[key]
	if !ok {
		return fallback
	}
	switch typed := value.(type) {
	case int:
		return typed
	case int64:
		return int(typed)
	case float64:
		return int(typed)
	case float32:
		return int(typed)
	case json.Number:
		parsed, err := typed.Int64()
		if err == nil {
			return int(parsed)
		}
	case string:
		var parsed int
		if _, err := fmt.Sscanf(typed, "%d", &parsed); err == nil {
			return parsed
		}
	}
	return fallback
}

func floatOption(options map[string]any, key string) (float64, bool) {
	value, ok := options[key]
	if !ok {
		return 0, false
	}
	switch typed := value.(type) {
	case float64:
		return typed, true
	case float32:
		return float64(typed), true
	case int:
		return float64(typed), true
	case int64:
		return float64(typed), true
	case json.Number:
		parsed, err := typed.Float64()
		return parsed, err == nil
	case string:
		var parsed float64
		if _, err := fmt.Sscanf(typed, "%f", &parsed); err == nil {
			return parsed, true
		}
	}
	return 0, false
}

func stringOption(options map[string]any, key string) string {
	value, ok := options[key]
	if !ok {
		return ""
	}
	if typed, ok := value.(string); ok {
		return strings.TrimSpace(typed)
	}
	return ""
}

func stringArrayOption(options map[string]any, key string) []string {
	value, ok := options[key]
	if !ok {
		return []string{}
	}
	switch typed := value.(type) {
	case []string:
		return typed
	case []any:
		out := []string{}
		for _, item := range typed {
			if text, ok := item.(string); ok && strings.TrimSpace(text) != "" {
				out = append(out, strings.TrimSpace(text))
			}
		}
		return out
	default:
		return []string{}
	}
}

func truncateText(value string, max int) string {
	value = strings.TrimSpace(value)
	if max <= 0 || len(value) <= max {
		return value
	}
	return strings.TrimSpace(value[:max]) + "..."
}

func mustJSON(value any) string {
	bytes, err := json.Marshal(value)
	if err != nil {
		return "{}"
	}
	return string(bytes)
}

func intString(value int) string {
	return strconv.Itoa(value)
}

func newID(prefix string) string {
	return prefix + "_" + randomToken(12)
}

func randomToken(length int) string {
	if length <= 0 {
		return ""
	}
	bytes := make([]byte, (length+1)/2)
	if _, err := rand.Read(bytes); err != nil {
		return strings.Repeat("0", length)
	}
	return hex.EncodeToString(bytes)[:length]
}

func copyTasks(tasks []DocumentTask) []DocumentTask {
	if len(tasks) == 0 {
		return []DocumentTask{}
	}
	out := make([]DocumentTask, len(tasks))
	copy(out, tasks)
	return out
}

func copyCards(cards []map[string]any) []map[string]any {
	if len(cards) == 0 {
		return []map[string]any{}
	}
	out := make([]map[string]any, 0, len(cards))
	for _, card := range cards {
		out = append(out, copyCard(card))
	}
	return out
}

func copyCard(card map[string]any) map[string]any {
	copied := make(map[string]any, len(card))
	for key, value := range card {
		copied[key] = copyCardValue(value)
	}
	return copied
}

func copyCardValue(value any) any {
	switch typed := value.(type) {
	case map[string]string:
		out := make(map[string]string, len(typed))
		for key, item := range typed {
			out[key] = item
		}
		return out
	case map[string]any:
		out := make(map[string]any, len(typed))
		for key, item := range typed {
			out[key] = copyCardValue(item)
		}
		return out
	case []string:
		return append([]string{}, typed...)
	case []any:
		return append([]any{}, typed...)
	default:
		return value
	}
}

func nowISO() string {
	return time.Now().UTC().Format(time.RFC3339Nano)
}
