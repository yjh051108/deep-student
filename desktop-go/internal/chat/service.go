package chat

import (
	"bufio"
	"bytes"
	"context"
	"crypto/rand"
	"deep-student-go/internal/storage"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math/big"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"
)

type Service struct {
	mu              sync.RWMutex
	path            string
	state           store
	emit            func(name string, payload any)
	apiConfigLoader func() ([]ApiConfig, error)
	httpClient      *http.Client
	activeStreams   map[string]activeStream
}

type activeStream struct {
	Token  string
	Cancel context.CancelFunc
}

type store struct {
	Sessions             []Session               `json:"sessions"`
	Groups               []Group                 `json:"groups"`
	States               map[string]SessionState `json:"states"`
	Messages             []Message               `json:"messages"`
	Blocks               []Block                 `json:"blocks"`
	InteractionResponses []InteractionResponse   `json:"interactionResponses"`
	ApprovalChoices      []ApprovalChoice        `json:"approvalChoices"`
}

type Session struct {
	ID            string         `json:"id"`
	Mode          string         `json:"mode"`
	Title         *string        `json:"title,omitempty"`
	Description   *string        `json:"description,omitempty"`
	TitleLocked   bool           `json:"titleLocked,omitempty"`
	PersistStatus string         `json:"persistStatus"`
	CreatedAt     string         `json:"createdAt"`
	UpdatedAt     string         `json:"updatedAt"`
	GroupID       *string        `json:"groupId,omitempty"`
	Metadata      map[string]any `json:"metadata,omitempty"`
}

type Group struct {
	ID                string   `json:"id"`
	Name              string   `json:"name"`
	Description       *string  `json:"description,omitempty"`
	Icon              *string  `json:"icon,omitempty"`
	Color             *string  `json:"color,omitempty"`
	SystemPrompt      *string  `json:"systemPrompt,omitempty"`
	DefaultSkillIDs   []string `json:"defaultSkillIds"`
	PinnedResourceIDs []string `json:"pinnedResourceIds"`
	WorkspaceID       *string  `json:"workspaceId,omitempty"`
	SortOrder         int      `json:"sortOrder"`
	PersistStatus     string   `json:"persistStatus"`
	CreatedAt         string   `json:"createdAt"`
	UpdatedAt         string   `json:"updatedAt"`
}

type CreateGroupRequest struct {
	Name              string   `json:"name"`
	Description       *string  `json:"description,omitempty"`
	Icon              *string  `json:"icon,omitempty"`
	Color             *string  `json:"color,omitempty"`
	SystemPrompt      *string  `json:"systemPrompt,omitempty"`
	DefaultSkillIDs   []string `json:"defaultSkillIds,omitempty"`
	PinnedResourceIDs []string `json:"pinnedResourceIds,omitempty"`
	WorkspaceID       *string  `json:"workspaceId,omitempty"`
}

type UpdateGroupRequest struct {
	Name              *string  `json:"name,omitempty"`
	Description       *string  `json:"description,omitempty"`
	Icon              *string  `json:"icon,omitempty"`
	Color             *string  `json:"color,omitempty"`
	SystemPrompt      *string  `json:"systemPrompt,omitempty"`
	DefaultSkillIDs   []string `json:"defaultSkillIds,omitempty"`
	PinnedResourceIDs []string `json:"pinnedResourceIds,omitempty"`
	WorkspaceID       *string  `json:"workspaceId,omitempty"`
	SortOrder         *int     `json:"sortOrder,omitempty"`
	PersistStatus     *string  `json:"persistStatus,omitempty"`
}

type SessionSettings struct {
	Title       *string        `json:"title,omitempty"`
	Description *string        `json:"description,omitempty"`
	Metadata    map[string]any `json:"metadata,omitempty"`
	GroupID     *string        `json:"groupId,omitempty"`
}

type SessionState map[string]any

type Message struct {
	ID                 string         `json:"id"`
	SessionID          string         `json:"sessionId"`
	Role               string         `json:"role"`
	BlockIDs           []string       `json:"blockIds"`
	Timestamp          int64          `json:"timestamp"`
	PersistentStableID *string        `json:"persistentStableId,omitempty"`
	ParentID           *string        `json:"parentId,omitempty"`
	Supersedes         *string        `json:"supersedes,omitempty"`
	Meta               map[string]any `json:"_meta,omitempty"`
	Attachments        []any          `json:"attachments,omitempty"`
}

type Block struct {
	ID           string  `json:"id"`
	MessageID    string  `json:"messageId"`
	Type         string  `json:"type"`
	Status       string  `json:"status"`
	Content      string  `json:"content,omitempty"`
	ToolName     *string `json:"toolName,omitempty"`
	ToolCallID   *string `json:"toolCallId,omitempty"`
	ToolInput    any     `json:"toolInput,omitempty"`
	ToolOutput   any     `json:"toolOutput,omitempty"`
	Citations    []any   `json:"citations,omitempty"`
	Error        *string `json:"error,omitempty"`
	StartedAt    *int64  `json:"startedAt,omitempty"`
	EndedAt      *int64  `json:"endedAt,omitempty"`
	FirstChunkAt *int64  `json:"firstChunkAt,omitempty"`
}

type SendMessageRequest struct {
	SessionID          string            `json:"sessionId"`
	Content            string            `json:"content"`
	Options            map[string]any    `json:"options,omitempty"`
	UserMessageID      *string           `json:"userMessageId,omitempty"`
	AssistantMessageID *string           `json:"assistantMessageId,omitempty"`
	UserContextRefs    []any             `json:"userContextRefs,omitempty"`
	PathMap            map[string]string `json:"pathMap,omitempty"`
	WorkspaceID        *string           `json:"workspaceId,omitempty"`
}

type ApiConfig struct {
	ID                string            `json:"id"`
	Name              string            `json:"name"`
	ApiKey            string            `json:"apiKey"`
	BaseUrl           string            `json:"baseUrl"`
	Model             string            `json:"model"`
	Enabled           bool              `json:"enabled"`
	ModelAdapter      string            `json:"modelAdapter"`
	MaxOutputTokens   uint32            `json:"maxOutputTokens,omitempty"`
	Temperature       float32           `json:"temperature,omitempty"`
	SupportsTools     bool              `json:"supportsTools,omitempty"`
	IsBuiltin         bool              `json:"isBuiltin,omitempty"`
	Headers           map[string]string `json:"headers,omitempty"`
	ProviderType      *string           `json:"providerType,omitempty"`
	ProviderScope     *string           `json:"providerScope,omitempty"`
	ApiProtocol       *string           `json:"apiProtocol,omitempty"`
	VendorID          *string           `json:"vendorId,omitempty"`
	IsMultimodal      bool              `json:"isMultimodal"`
	IsReasoning       bool              `json:"isReasoning"`
	IsEmbedding       bool              `json:"isEmbedding"`
	IsReranker        bool              `json:"isReranker"`
	ContextWindow     *uint32           `json:"contextWindow,omitempty"`
	MaxTokensLimit    *uint32           `json:"maxTokensLimit,omitempty"`
	ReasoningEffort   *string           `json:"reasoningEffort,omitempty"`
	ThinkingEnabled   bool              `json:"thinkingEnabled,omitempty"`
	ThinkingBudget    *int              `json:"thinkingBudget,omitempty"`
	SupportsReasoning bool              `json:"supportsReasoning,omitempty"`
}

type RetryMessageResponse struct {
	MessageID         string   `json:"message_id"`
	DeletedMessageIDs []string `json:"deleted_message_ids,omitempty"`
	DeletedVariantIDs []string `json:"deleted_variant_ids,omitempty"`
	NewVariantID      *string  `json:"new_variant_id,omitempty"`
}

type EditAndResendRequest struct {
	SessionID          string            `json:"sessionId"`
	MessageID          string            `json:"messageId"`
	NewContent         string            `json:"newContent"`
	AssistantMessageID *string           `json:"assistantMessageId,omitempty"`
	NewContextRefs     []any             `json:"newContextRefs,omitempty"`
	NewPathMap         map[string]string `json:"newPathMap,omitempty"`
	Options            map[string]any    `json:"options,omitempty"`
}

type EditAndResendResponse struct {
	NewMessageID      string   `json:"new_message_id"`
	DeletedMessageIDs []string `json:"deleted_message_ids,omitempty"`
	NewVariantID      *string  `json:"new_variant_id,omitempty"`
}

type MessageSummary struct {
	TotalMessages        int `json:"total_messages"`
	UserMessages         int `json:"user_messages"`
	AssistantMessages    int `json:"assistant_messages"`
	SessionsWithMessages int `json:"sessions_with_messages"`
}

type InteractionResponse struct {
	ToolCallID string         `json:"toolCallId"`
	Kind       string         `json:"kind"`
	SessionID  *string        `json:"sessionId,omitempty"`
	Response   map[string]any `json:"response"`
	CreatedAt  string         `json:"createdAt"`
}

type ApprovalChoice struct {
	ToolName  string         `json:"toolName"`
	Approved  bool           `json:"approved"`
	Arguments map[string]any `json:"arguments,omitempty"`
	UpdatedAt string         `json:"updatedAt"`
}

type UsageTrendPoint struct {
	TimeLabel        string   `json:"timeLabel"`
	Timestamp        int64    `json:"timestamp"`
	TotalTokens      int      `json:"totalTokens"`
	PromptTokens     int      `json:"promptTokens"`
	CompletionTokens int      `json:"completionTokens"`
	RequestCount     int      `json:"requestCount"`
	EstimatedCostUSD *float64 `json:"estimatedCostUsd,omitempty"`
	SuccessRate      *float64 `json:"successRate,omitempty"`
}

type ModelSummary struct {
	ModelID             string   `json:"modelId"`
	RequestCount        int      `json:"requestCount"`
	TotalTokens         int      `json:"totalTokens"`
	PromptTokens        int      `json:"promptTokens"`
	CompletionTokens    int      `json:"completionTokens"`
	EstimatedCostUSD    *float64 `json:"estimatedCostUsd,omitempty"`
	Percentage          *float64 `json:"percentage,omitempty"`
	AvgTokensPerRequest *float64 `json:"avgTokensPerRequest,omitempty"`
}

type CallerTypeSummary struct {
	CallerType       string   `json:"callerType"`
	DisplayName      string   `json:"displayName"`
	RequestCount     int      `json:"requestCount"`
	TotalTokens      int      `json:"totalTokens"`
	EstimatedCostUSD *float64 `json:"estimatedCostUsd,omitempty"`
	Percentage       *float64 `json:"percentage,omitempty"`
}

type DailySummary struct {
	DateKey               string   `json:"dateKey"`
	Model                 string   `json:"model"`
	CallerType            string   `json:"callerType"`
	CallCount             int      `json:"callCount"`
	TotalPromptTokens     int      `json:"totalPromptTokens"`
	TotalCompletionTokens int      `json:"totalCompletionTokens"`
	TotalTokens           int      `json:"totalTokens"`
	SuccessCount          int      `json:"successCount"`
	ErrorCount            int      `json:"errorCount"`
	TotalCostEstimate     *float64 `json:"totalCostEstimate,omitempty"`
}

type UsageSummary struct {
	StartDate             string              `json:"startDate"`
	EndDate               string              `json:"endDate"`
	TotalRequests         int                 `json:"totalRequests"`
	SuccessRequests       int                 `json:"successRequests"`
	ErrorRequests         int                 `json:"errorRequests"`
	TotalPromptTokens     int                 `json:"totalPromptTokens"`
	TotalCompletionTokens int                 `json:"totalCompletionTokens"`
	TotalTokens           int                 `json:"totalTokens"`
	TotalReasoningTokens  *int                `json:"totalReasoningTokens,omitempty"`
	TotalCachedTokens     *int                `json:"totalCachedTokens,omitempty"`
	TotalEstimatedCostUSD *float64            `json:"totalEstimatedCostUsd,omitempty"`
	AvgTokensPerRequest   *float64            `json:"avgTokensPerRequest,omitempty"`
	AvgDurationMs         *float64            `json:"avgDurationMs,omitempty"`
	ByCallerType          []CallerTypeSummary `json:"byCallerType,omitempty"`
	ByModel               []ModelSummary      `json:"byModel,omitempty"`
	TrendPoints           []UsageTrendPoint   `json:"trendPoints,omitempty"`
}

type UsageRecord struct {
	ID                string   `json:"id"`
	CallerType        string   `json:"callerType"`
	CallerID          *string  `json:"callerId,omitempty"`
	ModelID           string   `json:"modelId"`
	ConfigID          *string  `json:"configId,omitempty"`
	ProviderID        *string  `json:"providerId,omitempty"`
	PromptTokens      int      `json:"promptTokens"`
	CompletionTokens  int      `json:"completionTokens"`
	TotalTokens       int      `json:"totalTokens"`
	ReasoningTokens   *int     `json:"reasoningTokens,omitempty"`
	CachedTokens      *int     `json:"cachedTokens,omitempty"`
	EstimatedCostUSD  *float64 `json:"estimatedCostUsd,omitempty"`
	DurationMs        *int64   `json:"durationMs,omitempty"`
	Success           bool     `json:"success"`
	ErrorMessage      *string  `json:"errorMessage,omitempty"`
	CreatedAt         string   `json:"createdAt"`
	WorkspaceID       *string  `json:"workspaceId,omitempty"`
	Timestamp         int64    `json:"-"`
	TotalPromptWindow int      `json:"-"`
}

type LoadSessionResponse struct {
	Session  Session      `json:"session"`
	Messages []Message    `json:"messages"`
	Blocks   []Block      `json:"blocks"`
	State    SessionState `json:"state,omitempty"`
}

type BackendEvent struct {
	SessionID  string         `json:"sessionId,omitempty"`
	Type       string         `json:"type"`
	Phase      string         `json:"phase"`
	MessageID  string         `json:"messageId,omitempty"`
	BlockID    string         `json:"blockId,omitempty"`
	BlockType  string         `json:"blockType,omitempty"`
	Chunk      string         `json:"chunk,omitempty"`
	Result     any            `json:"result,omitempty"`
	Error      string         `json:"error,omitempty"`
	Payload    map[string]any `json:"payload,omitempty"`
	SequenceID int            `json:"sequenceId,omitempty"`
	ModelID    string         `json:"modelId,omitempty"`
}

type SessionEventPayload struct {
	SessionID  string         `json:"sessionId"`
	EventType  string         `json:"eventType"`
	MessageID  string         `json:"messageId,omitempty"`
	ModelID    string         `json:"modelId,omitempty"`
	Error      string         `json:"error,omitempty"`
	DurationMs int64          `json:"durationMs,omitempty"`
	Timestamp  int64          `json:"timestamp"`
	Usage      map[string]any `json:"usage,omitempty"`
}

type streamResult struct {
	Chunks []string
	Usage  map[string]any
}

func NewService(dataDir string) (*Service, error) {
	service := &Service{
		httpClient:    &http.Client{Timeout: 60 * time.Second},
		activeStreams: map[string]activeStream{},
		path:          filepath.Join(dataDir, "chat-go.json"),
		state: store{
			Sessions:             []Session{},
			Groups:               []Group{},
			States:               map[string]SessionState{},
			Messages:             []Message{},
			Blocks:               []Block{},
			InteractionResponses: []InteractionResponse{},
			ApprovalChoices:      []ApprovalChoice{},
		},
	}
	if err := service.load(); err != nil {
		return nil, err
	}
	return service, nil
}

func (s *Service) SetEventEmitter(emit func(name string, payload any)) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.emit = emit
}

func (s *Service) SetAPIConfigLoader(loader func() ([]ApiConfig, error)) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.apiConfigLoader = loader
}

func (s *Service) SetHTTPClient(client *http.Client) {
	if client == nil {
		return
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.httpClient = client
}

func (s *Service) emitLocalAssistantStream(sessionID string, messageID string, blockID string, content string, modelID string, startedAt int64) {
	emit := s.currentEmitter()
	if emit == nil {
		return
	}

	sessionChannel := "chat_v2_session_" + sessionID
	blockChannel := "chat_v2_event_" + sessionID
	nowMillis := time.Now().UnixMilli()

	emit(sessionChannel, SessionEventPayload{
		SessionID: sessionID,
		EventType: "stream_start",
		MessageID: messageID,
		ModelID:   modelID,
		Timestamp: startedAt,
	})
	emit(blockChannel, BackendEvent{
		SessionID:  sessionID,
		Type:       "content",
		Phase:      "start",
		MessageID:  messageID,
		BlockID:    blockID,
		BlockType:  "content",
		SequenceID: 1,
		ModelID:    modelID,
	})
	chunks := streamChunks(content, 80)
	for index, chunk := range chunks {
		emit(blockChannel, BackendEvent{
			SessionID:  sessionID,
			Type:       "content",
			Phase:      "chunk",
			MessageID:  messageID,
			BlockID:    blockID,
			Chunk:      chunk,
			SequenceID: index + 2,
			ModelID:    modelID,
		})
	}
	emit(blockChannel, BackendEvent{
		SessionID:  sessionID,
		Type:       "content",
		Phase:      "end",
		MessageID:  messageID,
		BlockID:    blockID,
		SequenceID: len(chunks) + 2,
		ModelID:    modelID,
	})
	emit(sessionChannel, SessionEventPayload{
		SessionID:  sessionID,
		EventType:  "stream_complete",
		MessageID:  messageID,
		ModelID:    modelID,
		DurationMs: nowMillis - startedAt,
		Timestamp:  nowMillis,
	})
}

func (s *Service) currentEmitter() func(name string, payload any) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.emit
}

func (s *Service) currentAPIConfigLoader() func() ([]ApiConfig, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.apiConfigLoader
}

func (s *Service) currentHTTPClient() *http.Client {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if s.httpClient == nil {
		return http.DefaultClient
	}
	return s.httpClient
}

func (s *Service) runAssistantStream(sessionID string, messageID string, blockID string, userContent string, options map[string]any, startedAt int64) {
	ctx, cancel := context.WithCancel(context.Background())
	streamToken := s.registerActiveStream(messageID, cancel)
	defer s.unregisterActiveStream(messageID, streamToken)

	config, ok, err := s.resolveChatAPIConfig(options)
	if err != nil {
		s.emitAssistantError(sessionID, messageID, blockID, err, startedAt)
		return
	}
	if !ok {
		content := "Go shell chat streaming is not connected to a configured remote provider yet. Your message was saved locally and replayed through the Wails event bridge so the frontend streaming path can run."
		s.persistAssistantBlock(sessionID, messageID, blockID, content, "complete", nil, "go-local-placeholder", nil)
		s.emitLocalAssistantStream(sessionID, messageID, blockID, content, "go-local-placeholder", startedAt)
		return
	}

	modelID := config.ID
	if modelID == "" {
		modelID = config.Model
	}
	s.updateAssistantModelMeta(messageID, modelID)
	s.emitSessionEvent("chat_v2_session_"+sessionID, SessionEventPayload{
		SessionID: sessionID,
		EventType: "stream_start",
		MessageID: messageID,
		ModelID:   modelID,
		Timestamp: startedAt,
	})
	s.emitBackendEvent("chat_v2_event_"+sessionID, BackendEvent{
		SessionID:  sessionID,
		Type:       "content",
		Phase:      "start",
		MessageID:  messageID,
		BlockID:    blockID,
		BlockType:  "content",
		SequenceID: 1,
		ModelID:    modelID,
	})
	sequenceID := 2
	result, err := s.streamOpenAICompatible(ctx, config, userContent, options, func(chunk string) {
		s.emitBackendEvent("chat_v2_event_"+sessionID, BackendEvent{
			SessionID:  sessionID,
			Type:       "content",
			Phase:      "chunk",
			MessageID:  messageID,
			BlockID:    blockID,
			Chunk:      chunk,
			SequenceID: sequenceID,
			ModelID:    modelID,
		})
		sequenceID++
	})
	if err != nil {
		if errors.Is(err, context.Canceled) {
			s.emitAssistantCancelled(sessionID, messageID, blockID, modelID, startedAt)
			return
		}
		s.emitAssistantError(sessionID, messageID, blockID, err, startedAt)
		return
	}
	content := strings.Join(result.Chunks, "")
	s.persistAssistantBlock(sessionID, messageID, blockID, content, "complete", nil, modelID, result.Usage)
	nowMillis := time.Now().UnixMilli()
	s.emitBackendEvent("chat_v2_event_"+sessionID, BackendEvent{
		SessionID:  sessionID,
		Type:       "content",
		Phase:      "end",
		MessageID:  messageID,
		BlockID:    blockID,
		SequenceID: sequenceID,
		ModelID:    modelID,
	})
	s.emitSessionEvent("chat_v2_session_"+sessionID, SessionEventPayload{
		SessionID:  sessionID,
		EventType:  "stream_complete",
		MessageID:  messageID,
		ModelID:    modelID,
		DurationMs: nowMillis - startedAt,
		Timestamp:  nowMillis,
		Usage:      result.Usage,
	})
}

func (s *Service) resolveChatAPIConfig(options map[string]any) (ApiConfig, bool, error) {
	loader := s.currentAPIConfigLoader()
	if loader == nil {
		return ApiConfig{}, false, nil
	}
	configs, err := loader()
	if err != nil {
		return ApiConfig{}, false, err
	}
	preferredID := stringOption(options, "model2OverrideId")
	if preferredID == "" {
		preferredID = stringOption(options, "modelId")
	}
	if preferredID != "" {
		for _, config := range configs {
			if config.ID == preferredID && config.Enabled && strings.TrimSpace(config.ApiKey) != "" && strings.TrimSpace(config.BaseUrl) != "" && strings.TrimSpace(config.Model) != "" {
				return config, true, nil
			}
		}
	}
	for _, config := range configs {
		if config.Enabled && strings.TrimSpace(config.ApiKey) != "" && strings.TrimSpace(config.BaseUrl) != "" && strings.TrimSpace(config.Model) != "" && !config.IsEmbedding && !config.IsReranker {
			return config, true, nil
		}
	}
	return ApiConfig{}, false, nil
}

func (s *Service) streamOpenAICompatible(ctx context.Context, config ApiConfig, userContent string, options map[string]any, onChunk func(string)) (streamResult, error) {
	baseURL := strings.TrimRight(strings.TrimSpace(config.BaseUrl), "/")
	if baseURL == "" {
		return streamResult{}, errors.New("api baseUrl is required")
	}
	endpoint := baseURL
	if !strings.HasSuffix(endpoint, "/chat/completions") {
		endpoint += "/chat/completions"
	}
	maxTokens := intOption(options, "maxTokens")
	if maxTokens <= 0 && config.MaxOutputTokens > 0 {
		maxTokens = int(config.MaxOutputTokens)
	}
	body := map[string]any{
		"model":  strings.TrimSpace(config.Model),
		"stream": true,
		"stream_options": map[string]any{
			"include_usage": true,
		},
		"messages": []map[string]string{
			{"role": "user", "content": userContent},
		},
	}
	if maxTokens > 0 {
		body["max_tokens"] = maxTokens
	}
	if temperature, ok := floatOption(options, "temperature"); ok {
		body["temperature"] = temperature
	} else if config.Temperature > 0 {
		body["temperature"] = config.Temperature
	}
	if topP, ok := floatOption(options, "topP"); ok {
		body["top_p"] = topP
	}
	requestBody, err := json.Marshal(body)
	if err != nil {
		return streamResult{}, err
	}
	request, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(requestBody))
	if err != nil {
		return streamResult{}, err
	}
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("Accept", "text/event-stream")
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
		return streamResult{}, err
	}
	defer response.Body.Close()
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		bytes, _ := io.ReadAll(io.LimitReader(response.Body, 4096))
		return streamResult{}, fmt.Errorf("provider returned %s: %s", response.Status, strings.TrimSpace(string(bytes)))
	}
	result, err := readOpenAIStream(response.Body, onChunk)
	if err != nil {
		return streamResult{}, err
	}
	if len(result.Chunks) == 0 {
		result.Chunks = []string{""}
	}
	return result, nil
}

func readOpenAIStream(reader io.Reader, onChunk func(string)) (streamResult, error) {
	scanner := bufio.NewScanner(reader)
	scanner.Buffer(make([]byte, 0, 4096), 1024*1024)
	result := streamResult{Chunks: []string{}}
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, ":") {
			continue
		}
		if !strings.HasPrefix(line, "data:") {
			continue
		}
		data := strings.TrimSpace(strings.TrimPrefix(line, "data:"))
		if data == "[DONE]" {
			break
		}
		event, err := parseOpenAIStreamEvent(data)
		if err != nil {
			return streamResult{}, err
		}
		if event.Usage != nil {
			result.Usage = event.Usage
		}
		if event.Chunk == "" {
			continue
		}
		result.Chunks = append(result.Chunks, event.Chunk)
		if onChunk != nil {
			onChunk(event.Chunk)
		}
	}
	if err := scanner.Err(); err != nil {
		return streamResult{}, err
	}
	return result, nil
}

type openAIStreamEvent struct {
	Chunk string
	Usage map[string]any
}

func parseOpenAIStreamEvent(data string) (openAIStreamEvent, error) {
	var event struct {
		Usage   map[string]any `json:"usage"`
		Choices []struct {
			Delta struct {
				Content any `json:"content"`
			} `json:"delta"`
		} `json:"choices"`
	}
	if err := json.Unmarshal([]byte(data), &event); err != nil {
		return openAIStreamEvent{}, err
	}
	normalized := openAIStreamEvent{
		Usage: normalizeOpenAIUsage(event.Usage),
	}
	if len(event.Choices) == 0 {
		return normalized, nil
	}
	switch content := event.Choices[0].Delta.Content.(type) {
	case string:
		normalized.Chunk = content
		return normalized, nil
	case []any:
		parts := make([]string, 0, len(content))
		for _, item := range content {
			if object, ok := item.(map[string]any); ok {
				if text, ok := object["text"].(string); ok {
					parts = append(parts, text)
				}
			}
		}
		normalized.Chunk = strings.Join(parts, "")
		return normalized, nil
	default:
		return normalized, nil
	}
}

func normalizeOpenAIUsage(raw map[string]any) map[string]any {
	if len(raw) == 0 {
		return nil
	}
	promptTokens, hasPrompt := numberFromMap(raw, "prompt_tokens")
	completionTokens, hasCompletion := numberFromMap(raw, "completion_tokens")
	totalTokens, hasTotal := numberFromMap(raw, "total_tokens")
	if !hasPrompt && !hasCompletion && !hasTotal {
		return nil
	}
	if !hasPrompt {
		promptTokens = 0
	}
	if !hasCompletion {
		completionTokens = 0
	}
	if !hasTotal {
		totalTokens = promptTokens + completionTokens
	}
	usage := map[string]any{
		"promptTokens":          promptTokens,
		"completionTokens":      completionTokens,
		"totalTokens":           totalTokens,
		"source":                "api",
		"lastRoundPromptTokens": promptTokens + completionTokens,
	}
	if details, ok := raw["completion_tokens_details"].(map[string]any); ok {
		if reasoningTokens, ok := numberFromMap(details, "reasoning_tokens"); ok {
			usage["reasoningTokens"] = reasoningTokens
		}
	}
	if details, ok := raw["prompt_tokens_details"].(map[string]any); ok {
		if cachedTokens, ok := numberFromMap(details, "cached_tokens"); ok {
			usage["cachedTokens"] = cachedTokens
		}
	}
	return usage
}

func numberFromMap(values map[string]any, key string) (int, bool) {
	switch value := values[key].(type) {
	case float64:
		return int(value), true
	case float32:
		return int(value), true
	case int:
		return value, true
	case int64:
		return int(value), true
	case json.Number:
		number, err := value.Int64()
		if err == nil {
			return int(number), true
		}
		float, err := value.Float64()
		if err == nil {
			return int(float), true
		}
	case string:
		number, err := strconv.Atoi(strings.TrimSpace(value))
		if err == nil {
			return number, true
		}
	}
	return 0, false
}

func (s *Service) usageRecordsInRangeLocked(start time.Time, end time.Time) []UsageRecord {
	records := []UsageRecord{}
	for _, message := range s.state.Messages {
		if message.Role != "assistant" || len(message.Meta) == 0 {
			continue
		}
		usage, ok := message.Meta["usage"].(map[string]any)
		if !ok {
			continue
		}
		timestamp := time.UnixMilli(message.Timestamp)
		if !start.IsZero() && timestamp.Before(start) {
			continue
		}
		if !end.IsZero() && timestamp.After(end) {
			continue
		}
		promptTokens, hasPrompt := numberFromMap(usage, "promptTokens")
		completionTokens, hasCompletion := numberFromMap(usage, "completionTokens")
		totalTokens, hasTotal := numberFromMap(usage, "totalTokens")
		if !hasPrompt && !hasCompletion && !hasTotal {
			continue
		}
		if !hasPrompt {
			promptTokens = 0
		}
		if !hasCompletion {
			completionTokens = 0
		}
		if !hasTotal {
			totalTokens = promptTokens + completionTokens
		}
		modelID := stringFromMap(message.Meta, "modelId")
		if modelID == "" {
			modelID = "unknown"
		}
		callerID := message.SessionID
		record := UsageRecord{
			ID:               "usage_" + message.ID,
			CallerType:       "chat",
			CallerID:         &callerID,
			ModelID:          modelID,
			ConfigID:         &modelID,
			PromptTokens:     promptTokens,
			CompletionTokens: completionTokens,
			TotalTokens:      totalTokens,
			Success:          stringFromMap(message.Meta, "terminalError") == "",
			CreatedAt:        timestamp.Format(time.RFC3339),
			Timestamp:        message.Timestamp,
		}
		if reasoningTokens, ok := numberFromMap(usage, "reasoningTokens"); ok {
			record.ReasoningTokens = intPtr(reasoningTokens)
		}
		if cachedTokens, ok := numberFromMap(usage, "cachedTokens"); ok {
			record.CachedTokens = intPtr(cachedTokens)
		}
		if lastRoundPromptTokens, ok := numberFromMap(usage, "lastRoundPromptTokens"); ok {
			record.TotalPromptWindow = lastRoundPromptTokens
		}
		records = append(records, record)
	}
	return records
}

func parseUsageDateRange(startDate *string, endDate *string) (time.Time, time.Time, error) {
	now := time.Now()
	start := now.AddDate(0, 0, -29)
	end := now
	if startDate != nil && strings.TrimSpace(*startDate) != "" {
		parsed, err := time.ParseInLocation("2006-01-02", strings.TrimSpace(*startDate), time.Local)
		if err != nil {
			return time.Time{}, time.Time{}, fmt.Errorf("invalid startDate: %w", err)
		}
		start = parsed
	}
	if endDate != nil && strings.TrimSpace(*endDate) != "" {
		parsed, err := time.ParseInLocation("2006-01-02", strings.TrimSpace(*endDate), time.Local)
		if err != nil {
			return time.Time{}, time.Time{}, fmt.Errorf("invalid endDate: %w", err)
		}
		end = parsed.Add(24*time.Hour - time.Nanosecond)
	}
	if start.After(end) {
		return time.Time{}, time.Time{}, errors.New("startDate must be before endDate")
	}
	return start, end, nil
}

func buildUsageByModel(records []UsageRecord) []ModelSummary {
	totalTokens := 0
	byModel := map[string]*ModelSummary{}
	for _, record := range records {
		totalTokens += record.TotalTokens
		model := byModel[record.ModelID]
		if model == nil {
			model = &ModelSummary{ModelID: record.ModelID}
			byModel[record.ModelID] = model
		}
		model.RequestCount++
		model.TotalTokens += record.TotalTokens
		model.PromptTokens += record.PromptTokens
		model.CompletionTokens += record.CompletionTokens
	}
	out := make([]ModelSummary, 0, len(byModel))
	for _, item := range byModel {
		if item.RequestCount > 0 {
			item.AvgTokensPerRequest = floatPtr(float64(item.TotalTokens) / float64(item.RequestCount))
		}
		if totalTokens > 0 {
			item.Percentage = floatPtr(float64(item.TotalTokens) * 100 / float64(totalTokens))
		}
		out = append(out, *item)
	}
	sort.Slice(out, func(a, b int) bool {
		if out[a].TotalTokens != out[b].TotalTokens {
			return out[a].TotalTokens > out[b].TotalTokens
		}
		return out[a].ModelID < out[b].ModelID
	})
	return out
}

func buildUsageByCaller(records []UsageRecord) []CallerTypeSummary {
	totalTokens := 0
	byCaller := map[string]*CallerTypeSummary{}
	for _, record := range records {
		totalTokens += record.TotalTokens
		caller := byCaller[record.CallerType]
		if caller == nil {
			caller = &CallerTypeSummary{
				CallerType:  record.CallerType,
				DisplayName: displayNameForCaller(record.CallerType),
			}
			byCaller[record.CallerType] = caller
		}
		caller.RequestCount++
		caller.TotalTokens += record.TotalTokens
	}
	out := make([]CallerTypeSummary, 0, len(byCaller))
	for _, item := range byCaller {
		if totalTokens > 0 {
			item.Percentage = floatPtr(float64(item.TotalTokens) * 100 / float64(totalTokens))
		}
		out = append(out, *item)
	}
	sort.Slice(out, func(a, b int) bool {
		if out[a].TotalTokens != out[b].TotalTokens {
			return out[a].TotalTokens > out[b].TotalTokens
		}
		return out[a].CallerType < out[b].CallerType
	})
	return out
}

func buildUsageTrends(records []UsageRecord, granularity string) []UsageTrendPoint {
	if strings.TrimSpace(granularity) == "" {
		granularity = "day"
	}
	byBucket := map[int64]*UsageTrendPoint{}
	for _, record := range records {
		bucket := usageBucketStart(time.UnixMilli(record.Timestamp), granularity)
		key := bucket.UnixMilli()
		point := byBucket[key]
		if point == nil {
			point = &UsageTrendPoint{
				TimeLabel:   usageBucketLabel(bucket, granularity),
				Timestamp:   key,
				SuccessRate: floatPtr(100),
			}
			byBucket[key] = point
		}
		point.TotalTokens += record.TotalTokens
		point.PromptTokens += record.PromptTokens
		point.CompletionTokens += record.CompletionTokens
		point.RequestCount++
	}
	out := make([]UsageTrendPoint, 0, len(byBucket))
	for _, point := range byBucket {
		out = append(out, *point)
	}
	sort.Slice(out, func(a, b int) bool {
		return out[a].Timestamp < out[b].Timestamp
	})
	return out
}

func usageBucketStart(value time.Time, granularity string) time.Time {
	switch granularity {
	case "hour":
		return time.Date(value.Year(), value.Month(), value.Day(), value.Hour(), 0, 0, 0, value.Location())
	case "week":
		weekday := int(value.Weekday())
		if weekday == 0 {
			weekday = 7
		}
		base := time.Date(value.Year(), value.Month(), value.Day(), 0, 0, 0, 0, value.Location())
		return base.AddDate(0, 0, -(weekday - 1))
	case "month":
		return time.Date(value.Year(), value.Month(), 1, 0, 0, 0, 0, value.Location())
	default:
		return time.Date(value.Year(), value.Month(), value.Day(), 0, 0, 0, 0, value.Location())
	}
}

func usageBucketLabel(value time.Time, granularity string) string {
	switch granularity {
	case "hour":
		return value.Format("01-02 15:00")
	case "week":
		return value.Format("2006-01-02")
	case "month":
		return value.Format("2006-01")
	default:
		return value.Format("2006-01-02")
	}
}

func displayNameForCaller(callerType string) string {
	switch callerType {
	case "chat":
		return "Chat"
	default:
		if strings.TrimSpace(callerType) == "" {
			return "Unknown"
		}
		return callerType
	}
}

func stringFromMap(values map[string]any, key string) string {
	value, ok := values[key].(string)
	if !ok {
		return ""
	}
	return strings.TrimSpace(value)
}

func (s *Service) emitAssistantError(sessionID string, messageID string, blockID string, err error, startedAt int64) {
	message := err.Error()
	s.persistAssistantBlock(sessionID, messageID, blockID, "", "error", &message, "", nil)
	nowMillis := time.Now().UnixMilli()
	s.emitBackendEvent("chat_v2_event_"+sessionID, BackendEvent{
		SessionID: sessionID,
		Type:      "content",
		Phase:     "error",
		MessageID: messageID,
		BlockID:   blockID,
		Error:     message,
	})
	s.emitSessionEvent("chat_v2_session_"+sessionID, SessionEventPayload{
		SessionID:  sessionID,
		EventType:  "stream_error",
		MessageID:  messageID,
		Error:      message,
		DurationMs: nowMillis - startedAt,
		Timestamp:  nowMillis,
	})
}

func (s *Service) emitAssistantCancelled(sessionID string, messageID string, blockID string, modelID string, startedAt int64) {
	message := "Stream cancelled"
	s.persistAssistantBlock(sessionID, messageID, blockID, "", "cancelled", &message, modelID, nil)
	nowMillis := time.Now().UnixMilli()
	s.emitBackendEvent("chat_v2_event_"+sessionID, BackendEvent{
		SessionID: sessionID,
		Type:      "content",
		Phase:     "error",
		MessageID: messageID,
		BlockID:   blockID,
		Error:     message,
		ModelID:   modelID,
	})
	s.emitSessionEvent("chat_v2_session_"+sessionID, SessionEventPayload{
		SessionID:  sessionID,
		EventType:  "stream_cancelled",
		MessageID:  messageID,
		ModelID:    modelID,
		DurationMs: nowMillis - startedAt,
		Timestamp:  nowMillis,
	})
}

func (s *Service) persistAssistantBlock(sessionID string, messageID string, blockID string, content string, status string, blockError *string, modelID string, usage map[string]any) {
	nowMillis := time.Now().UnixMilli()
	s.mu.Lock()
	defer s.mu.Unlock()
	if index, ok := s.findBlockIndexLocked(blockID); ok {
		s.state.Blocks[index].Content = content
		s.state.Blocks[index].Status = status
		s.state.Blocks[index].Error = blockError
		s.state.Blocks[index].EndedAt = &nowMillis
	}
	if sessionIndex, ok := s.findSessionIndexLocked(sessionID); ok {
		s.state.Sessions[sessionIndex].UpdatedAt = nowISO()
	}
	if strings.TrimSpace(modelID) != "" {
		if messageIndex, ok := s.findMessageIndexLocked(messageID); ok {
			if s.state.Messages[messageIndex].Meta == nil {
				s.state.Messages[messageIndex].Meta = map[string]any{}
			}
			s.state.Messages[messageIndex].Meta["modelId"] = strings.TrimSpace(modelID)
			if usage != nil {
				s.state.Messages[messageIndex].Meta["usage"] = usage
			}
		}
	}
	_ = s.flushLocked()
}

func (s *Service) updateAssistantModelMeta(messageID string, modelID string) {
	modelID = strings.TrimSpace(modelID)
	if modelID == "" {
		return
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if messageIndex, ok := s.findMessageIndexLocked(messageID); ok {
		if s.state.Messages[messageIndex].Meta == nil {
			s.state.Messages[messageIndex].Meta = map[string]any{}
		}
		s.state.Messages[messageIndex].Meta["modelId"] = modelID
		_ = s.flushLocked()
	}
}

func (s *Service) registerActiveStream(messageID string, cancel context.CancelFunc) string {
	messageID = strings.TrimSpace(messageID)
	if messageID == "" || cancel == nil {
		return ""
	}
	token := randomToken(16)
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.activeStreams == nil {
		s.activeStreams = map[string]activeStream{}
	}
	s.activeStreams[messageID] = activeStream{Token: token, Cancel: cancel}
	return token
}

func (s *Service) unregisterActiveStream(messageID string, token string) {
	messageID = strings.TrimSpace(messageID)
	if messageID == "" || token == "" {
		return
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.activeStreams == nil {
		return
	}
	if current, ok := s.activeStreams[messageID]; ok && current.Token == token {
		delete(s.activeStreams, messageID)
	}
}

func (s *Service) cancelActiveStream(messageID string) bool {
	messageID = strings.TrimSpace(messageID)
	if messageID == "" {
		return false
	}
	s.mu.Lock()
	stream := s.activeStreams[messageID]
	s.mu.Unlock()
	if stream.Cancel == nil {
		return false
	}
	stream.Cancel()
	return true
}

func (s *Service) emitBackendEvent(channel string, payload BackendEvent) {
	emit := s.currentEmitter()
	if emit != nil {
		emit(channel, payload)
	}
}

func (s *Service) emitSessionEvent(channel string, payload SessionEventPayload) {
	emit := s.currentEmitter()
	if emit != nil {
		emit(channel, payload)
	}
}

func streamChunks(content string, maxRunes int) []string {
	if maxRunes <= 0 {
		maxRunes = 80
	}
	runes := []rune(content)
	if len(runes) == 0 {
		return []string{""}
	}
	out := make([]string, 0, (len(runes)/maxRunes)+1)
	for start := 0; start < len(runes); start += maxRunes {
		end := start + maxRunes
		if end > len(runes) {
			end = len(runes)
		}
		out = append(out, string(runes[start:end]))
	}
	return out
}

func stringOption(options map[string]any, key string) string {
	if options == nil {
		return ""
	}
	value, ok := options[key]
	if !ok {
		return ""
	}
	if text, ok := value.(string); ok {
		return strings.TrimSpace(text)
	}
	return ""
}

func intOption(options map[string]any, key string) int {
	if options == nil {
		return 0
	}
	switch value := options[key].(type) {
	case int:
		return value
	case int64:
		return int(value)
	case float64:
		return int(value)
	case json.Number:
		parsed, _ := value.Int64()
		return int(parsed)
	default:
		return 0
	}
}

func floatOption(options map[string]any, key string) (float64, bool) {
	if options == nil {
		return 0, false
	}
	switch value := options[key].(type) {
	case float32:
		return float64(value), true
	case float64:
		return value, true
	case int:
		return float64(value), true
	case json.Number:
		parsed, err := value.Float64()
		return parsed, err == nil
	default:
		return 0, false
	}
}

func (s *Service) CreateSession(mode string, title *string, metadata map[string]any, groupID *string) (Session, error) {
	mode = strings.TrimSpace(mode)
	if mode == "" {
		mode = "chat"
	}
	if !validMode(mode) {
		return Session{}, fmt.Errorf("invalid session mode: %s", mode)
	}
	groupID = normalizeOptionalString(groupID)

	s.mu.Lock()
	defer s.mu.Unlock()

	if groupID != nil && !s.groupIsActiveLocked(*groupID) {
		return Session{}, fmt.Errorf("group not found or inactive: %s", *groupID)
	}
	now := nowISO()
	session := Session{
		ID:            "sess_" + randomToken(16),
		Mode:          mode,
		Title:         normalizeOptionalString(title),
		PersistStatus: "active",
		CreatedAt:     now,
		UpdatedAt:     now,
		GroupID:       groupID,
		Metadata:      normalizeMetadata(metadata),
	}
	s.state.Sessions = append(s.state.Sessions, session)
	if err := s.flushLocked(); err != nil {
		return Session{}, err
	}
	return session, nil
}

func (s *Service) GetSession(sessionID string) (*Session, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	session, ok := s.findSessionLocked(sessionID)
	if !ok {
		return nil, nil
	}
	return &session, nil
}

func (s *Service) LoadSession(sessionID string) (LoadSessionResponse, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	session, ok := s.findSessionLocked(sessionID)
	if !ok {
		return LoadSessionResponse{}, fmt.Errorf("session not found: %s", sessionID)
	}
	state := s.state.States[sessionID]
	messages := s.messagesForSessionLocked(sessionID)
	blocks := s.blocksForMessagesLocked(messageIDs(messages))
	return LoadSessionResponse{
		Session:  session,
		Messages: messages,
		Blocks:   blocks,
		State:    state,
	}, nil
}

func (s *Service) SaveSession(sessionID string, sessionState SessionState) (bool, error) {
	if strings.TrimSpace(sessionID) == "" {
		return false, errors.New("sessionId is required")
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	index, ok := s.findSessionIndexLocked(sessionID)
	if !ok {
		return false, fmt.Errorf("session not found: %s", sessionID)
	}
	s.state.States[sessionID] = sessionState
	s.state.Sessions[index].UpdatedAt = nowISO()
	return true, s.flushLocked()
}

func (s *Service) UpdateSessionSettings(sessionID string, settings SessionSettings) (Session, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	index, ok := s.findSessionIndexLocked(sessionID)
	if !ok {
		return Session{}, fmt.Errorf("session not found: %s", sessionID)
	}
	session := s.state.Sessions[index]
	if settings.Title != nil {
		session.Title = normalizeOptionalString(settings.Title)
		session.TitleLocked = true
	}
	if settings.Description != nil {
		session.Description = normalizeOptionalString(settings.Description)
	}
	if settings.Metadata != nil {
		session.Metadata = normalizeMetadata(settings.Metadata)
	}
	if settings.GroupID != nil {
		normalized := normalizeOptionalString(settings.GroupID)
		if normalized != nil && !s.groupIsActiveLocked(*normalized) {
			return Session{}, fmt.Errorf("group not found or inactive: %s", *normalized)
		}
		session.GroupID = normalized
	}
	session.UpdatedAt = nowISO()
	s.state.Sessions[index] = session
	if err := s.flushLocked(); err != nil {
		return Session{}, err
	}
	return session, nil
}

func (s *Service) ArchiveSession(sessionID string) (bool, error) {
	return s.setSessionStatus(sessionID, "archived")
}

func (s *Service) RestoreSession(sessionID string) (Session, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	index, ok := s.findSessionIndexLocked(sessionID)
	if !ok {
		return Session{}, fmt.Errorf("session not found: %s", sessionID)
	}
	s.state.Sessions[index].PersistStatus = "active"
	s.state.Sessions[index].UpdatedAt = nowISO()
	if err := s.flushLocked(); err != nil {
		return Session{}, err
	}
	return s.state.Sessions[index], nil
}

func (s *Service) DeleteSession(sessionID string) (bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	index, ok := s.findSessionIndexLocked(sessionID)
	if !ok {
		return false, fmt.Errorf("session not found: %s", sessionID)
	}

	messageIDs := map[string]bool{}
	for _, message := range s.state.Messages {
		if message.SessionID == sessionID {
			messageIDs[message.ID] = true
			if stream, ok := s.activeStreams[message.ID]; ok && stream.Cancel != nil {
				return false, errors.New("cannot delete session while streaming")
			}
		}
	}

	s.state.Sessions = append(s.state.Sessions[:index], s.state.Sessions[index+1:]...)
	delete(s.state.States, sessionID)
	nextMessages := make([]Message, 0, len(s.state.Messages))
	for _, message := range s.state.Messages {
		if message.SessionID != sessionID {
			nextMessages = append(nextMessages, message)
		}
	}
	s.state.Messages = nextMessages
	nextBlocks := make([]Block, 0, len(s.state.Blocks))
	for _, block := range s.state.Blocks {
		if !messageIDs[block.MessageID] {
			nextBlocks = append(nextBlocks, block)
		}
	}
	s.state.Blocks = nextBlocks
	for messageID := range messageIDs {
		delete(s.activeStreams, messageID)
	}
	return true, s.flushLocked()
}

func (s *Service) MoveSessionToGroup(sessionID string, groupID *string) (bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	index, ok := s.findSessionIndexLocked(sessionID)
	if !ok {
		return false, fmt.Errorf("session not found: %s", sessionID)
	}
	normalized := normalizeOptionalString(groupID)
	if normalized != nil && !s.groupIsActiveLocked(*normalized) {
		return false, fmt.Errorf("group not found or inactive: %s", *normalized)
	}
	s.state.Sessions[index].GroupID = normalized
	s.state.Sessions[index].UpdatedAt = nowISO()
	return true, s.flushLocked()
}

func (s *Service) ListSessions(status *string, groupID *string, limit int, offset int) ([]Session, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if limit <= 0 {
		limit = 50
	}
	if offset < 0 {
		offset = 0
	}
	out := make([]Session, 0, len(s.state.Sessions))
	for _, session := range s.state.Sessions {
		if status != nil && *status != "" && session.PersistStatus != *status {
			continue
		}
		if !matchesGroupFilter(session, groupID) {
			continue
		}
		out = append(out, session)
	}
	sortSessions(out)
	if offset >= len(out) {
		return []Session{}, nil
	}
	end := offset + limit
	if end > len(out) {
		end = len(out)
	}
	return out[offset:end], nil
}

func (s *Service) CountSessions(status *string, groupID *string) (int, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	count := 0
	for _, session := range s.state.Sessions {
		if status != nil && *status != "" && session.PersistStatus != *status {
			continue
		}
		if !matchesGroupFilter(session, groupID) {
			continue
		}
		count++
	}
	return count, nil
}

func (s *Service) CreateGroup(request CreateGroupRequest) (Group, error) {
	name := strings.TrimSpace(request.Name)
	if name == "" {
		return Group{}, errors.New("group name is required")
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	now := nowISO()
	group := Group{
		ID:                "grp_" + randomToken(16),
		Name:              name,
		Description:       normalizeOptionalString(request.Description),
		Icon:              normalizeOptionalString(request.Icon),
		Color:             normalizeOptionalString(request.Color),
		SystemPrompt:      normalizeOptionalString(request.SystemPrompt),
		DefaultSkillIDs:   nonNilStrings(request.DefaultSkillIDs),
		PinnedResourceIDs: nonNilStrings(request.PinnedResourceIDs),
		WorkspaceID:       normalizeOptionalString(request.WorkspaceID),
		SortOrder:         s.nextGroupSortLocked(request.WorkspaceID),
		PersistStatus:     "active",
		CreatedAt:         now,
		UpdatedAt:         now,
	}
	s.state.Groups = append(s.state.Groups, group)
	if err := s.flushLocked(); err != nil {
		return Group{}, err
	}
	return group, nil
}

func (s *Service) GetGroup(groupID string) (*Group, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	group, ok := s.findGroupLocked(groupID)
	if !ok {
		return nil, nil
	}
	return &group, nil
}

func (s *Service) UpdateGroup(groupID string, request UpdateGroupRequest) (Group, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	index, ok := s.findGroupIndexLocked(groupID)
	if !ok {
		return Group{}, fmt.Errorf("group not found: %s", groupID)
	}
	group := s.state.Groups[index]
	if request.Name != nil && strings.TrimSpace(*request.Name) != "" {
		group.Name = strings.TrimSpace(*request.Name)
	}
	if request.Description != nil {
		group.Description = normalizeOptionalString(request.Description)
	}
	if request.Icon != nil {
		group.Icon = normalizeOptionalString(request.Icon)
	}
	if request.Color != nil {
		group.Color = normalizeOptionalString(request.Color)
	}
	if request.SystemPrompt != nil {
		group.SystemPrompt = normalizeOptionalString(request.SystemPrompt)
	}
	if request.DefaultSkillIDs != nil {
		group.DefaultSkillIDs = nonNilStrings(request.DefaultSkillIDs)
	}
	if request.PinnedResourceIDs != nil {
		group.PinnedResourceIDs = nonNilStrings(request.PinnedResourceIDs)
	}
	if request.WorkspaceID != nil {
		group.WorkspaceID = normalizeOptionalString(request.WorkspaceID)
	}
	if request.SortOrder != nil {
		group.SortOrder = *request.SortOrder
	}
	if request.PersistStatus != nil && *request.PersistStatus != "" {
		group.PersistStatus = *request.PersistStatus
	}
	group.UpdatedAt = nowISO()
	s.state.Groups[index] = group
	if err := s.flushLocked(); err != nil {
		return Group{}, err
	}
	return group, nil
}

func (s *Service) DeleteGroup(groupID string) (bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	index, ok := s.findGroupIndexLocked(groupID)
	if !ok {
		return false, fmt.Errorf("group not found: %s", groupID)
	}
	now := nowISO()
	s.state.Groups[index].PersistStatus = "deleted"
	s.state.Groups[index].UpdatedAt = now
	for sessionIndex := range s.state.Sessions {
		if s.state.Sessions[sessionIndex].GroupID != nil && *s.state.Sessions[sessionIndex].GroupID == groupID {
			s.state.Sessions[sessionIndex].GroupID = nil
			s.state.Sessions[sessionIndex].UpdatedAt = now
		}
	}
	return true, s.flushLocked()
}

func (s *Service) RestoreGroup(groupID string) (Group, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	index, ok := s.findGroupIndexLocked(groupID)
	if !ok {
		return Group{}, fmt.Errorf("group not found: %s", groupID)
	}
	s.state.Groups[index].PersistStatus = "active"
	s.state.Groups[index].UpdatedAt = nowISO()
	if err := s.flushLocked(); err != nil {
		return Group{}, err
	}
	return s.state.Groups[index], nil
}

func (s *Service) ListGroups(status *string, workspaceID *string) ([]Group, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]Group, 0, len(s.state.Groups))
	for _, group := range s.state.Groups {
		if status != nil && *status != "" && group.PersistStatus != *status {
			continue
		}
		if workspaceID != nil && *workspaceID != "" {
			if group.WorkspaceID == nil || *group.WorkspaceID != *workspaceID {
				continue
			}
		}
		out = append(out, group)
	}
	sortGroups(out)
	return out, nil
}

func (s *Service) ReorderGroups(groupIDs []string) (bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	order := map[string]int{}
	for index, groupID := range groupIDs {
		order[groupID] = index
	}
	now := nowISO()
	for index := range s.state.Groups {
		if sortOrder, ok := order[s.state.Groups[index].ID]; ok {
			s.state.Groups[index].SortOrder = sortOrder
			s.state.Groups[index].UpdatedAt = now
		}
	}
	return true, s.flushLocked()
}

func (s *Service) GetSessionTags(sessionID string) ([]string, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	session, ok := s.findSessionLocked(sessionID)
	if !ok {
		return []string{}, nil
	}
	return tagsFromMetadata(session.Metadata), nil
}

func (s *Service) GetTagsBatch(sessionIDs []string) (map[string][]string, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make(map[string][]string, len(sessionIDs))
	for _, sessionID := range sessionIDs {
		out[sessionID] = []string{}
	}
	for _, session := range s.state.Sessions {
		if _, ok := out[session.ID]; ok {
			out[session.ID] = tagsFromMetadata(session.Metadata)
		}
	}
	return out, nil
}

func (s *Service) AddTag(sessionID string, tag string) (bool, error) {
	tag = strings.TrimSpace(tag)
	if tag == "" {
		return false, errors.New("tag is required")
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	index, ok := s.findSessionIndexLocked(sessionID)
	if !ok {
		return false, fmt.Errorf("session not found: %s", sessionID)
	}
	tags := tagsFromMetadata(s.state.Sessions[index].Metadata)
	if !containsString(tags, tag) {
		tags = append(tags, tag)
	}
	s.setSessionTagsLocked(index, tags)
	s.state.Sessions[index].UpdatedAt = nowISO()
	return true, s.flushLocked()
}

func (s *Service) RemoveTag(sessionID string, tag string) (bool, error) {
	tag = strings.TrimSpace(tag)
	s.mu.Lock()
	defer s.mu.Unlock()
	index, ok := s.findSessionIndexLocked(sessionID)
	if !ok {
		return false, fmt.Errorf("session not found: %s", sessionID)
	}
	tags := tagsFromMetadata(s.state.Sessions[index].Metadata)
	next := make([]string, 0, len(tags))
	for _, existing := range tags {
		if existing != tag {
			next = append(next, existing)
		}
	}
	s.setSessionTagsLocked(index, next)
	s.state.Sessions[index].UpdatedAt = nowISO()
	return true, s.flushLocked()
}

func (s *Service) ListAllTags() ([][]any, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	counts := map[string]int{}
	for _, session := range s.state.Sessions {
		if session.PersistStatus == "deleted" {
			continue
		}
		for _, tag := range tagsFromMetadata(session.Metadata) {
			counts[tag]++
		}
	}
	tags := make([]string, 0, len(counts))
	for tag := range counts {
		tags = append(tags, tag)
	}
	sort.Slice(tags, func(a, b int) bool {
		if counts[tags[a]] != counts[tags[b]] {
			return counts[tags[a]] > counts[tags[b]]
		}
		return tags[a] < tags[b]
	})
	out := make([][]any, 0, len(tags))
	for _, tag := range tags {
		out = append(out, []any{tag, counts[tag]})
	}
	return out, nil
}

func (s *Service) GetMessageSummary() (MessageSummary, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	sessionsWithMessages := map[string]bool{}
	summary := MessageSummary{}
	for _, message := range s.state.Messages {
		summary.TotalMessages++
		sessionsWithMessages[message.SessionID] = true
		switch message.Role {
		case "user":
			summary.UserMessages++
		case "assistant":
			summary.AssistantMessages++
		}
	}
	summary.SessionsWithMessages = len(sessionsWithMessages)
	return summary, nil
}

func (s *Service) LLMUsageGetTrends(days int, granularity string) ([]UsageTrendPoint, error) {
	if days <= 0 {
		days = 30
	}
	end := time.Now()
	start := end.AddDate(0, 0, -days+1)
	s.mu.RLock()
	defer s.mu.RUnlock()
	return buildUsageTrends(s.usageRecordsInRangeLocked(start, end), granularity), nil
}

func (s *Service) LLMUsageByModel(startDate string, endDate string) ([]ModelSummary, error) {
	start, end, err := parseUsageDateRange(&startDate, &endDate)
	if err != nil {
		return nil, err
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	return buildUsageByModel(s.usageRecordsInRangeLocked(start, end)), nil
}

func (s *Service) LLMUsageByCaller(startDate string, endDate string) ([]CallerTypeSummary, error) {
	start, end, err := parseUsageDateRange(&startDate, &endDate)
	if err != nil {
		return nil, err
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	return buildUsageByCaller(s.usageRecordsInRangeLocked(start, end)), nil
}

func (s *Service) LLMUsageSummary(startDate *string, endDate *string) (UsageSummary, error) {
	start, end, err := parseUsageDateRange(startDate, endDate)
	if err != nil {
		return UsageSummary{}, err
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	records := s.usageRecordsInRangeLocked(start, end)
	summary := UsageSummary{
		StartDate:    start.Format("2006-01-02"),
		EndDate:      end.Format("2006-01-02"),
		ByModel:      buildUsageByModel(records),
		ByCallerType: buildUsageByCaller(records),
		TrendPoints:  buildUsageTrends(records, "day"),
	}
	var reasoningTotal int
	var hasReasoning bool
	var cachedTotal int
	var hasCached bool
	var durationTotal int64
	var durationCount int
	for _, record := range records {
		summary.TotalRequests++
		if record.Success {
			summary.SuccessRequests++
		} else {
			summary.ErrorRequests++
		}
		summary.TotalPromptTokens += record.PromptTokens
		summary.TotalCompletionTokens += record.CompletionTokens
		summary.TotalTokens += record.TotalTokens
		if record.ReasoningTokens != nil {
			hasReasoning = true
			reasoningTotal += *record.ReasoningTokens
		}
		if record.CachedTokens != nil {
			hasCached = true
			cachedTotal += *record.CachedTokens
		}
		if record.DurationMs != nil {
			durationCount++
			durationTotal += *record.DurationMs
		}
	}
	if summary.TotalRequests > 0 {
		summary.AvgTokensPerRequest = floatPtr(float64(summary.TotalTokens) / float64(summary.TotalRequests))
	}
	if durationCount > 0 {
		summary.AvgDurationMs = floatPtr(float64(durationTotal) / float64(durationCount))
	}
	if hasReasoning {
		summary.TotalReasoningTokens = intPtr(reasoningTotal)
	}
	if hasCached {
		summary.TotalCachedTokens = intPtr(cachedTotal)
	}
	return summary, nil
}

func (s *Service) LLMUsageRecent(limit int) ([]UsageRecord, error) {
	if limit <= 0 {
		limit = 20
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	records := s.usageRecordsInRangeLocked(time.Time{}, time.Now())
	sort.Slice(records, func(a, b int) bool {
		return records[a].Timestamp > records[b].Timestamp
	})
	if len(records) > limit {
		records = records[:limit]
	}
	return records, nil
}

func (s *Service) LLMUsageDaily(startDate string, endDate string) ([]DailySummary, error) {
	start, end, err := parseUsageDateRange(&startDate, &endDate)
	if err != nil {
		return nil, err
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	records := s.usageRecordsInRangeLocked(start, end)
	byKey := map[string]*DailySummary{}
	for _, record := range records {
		dateKey := time.UnixMilli(record.Timestamp).Format("2006-01-02")
		key := dateKey + "\x00" + record.ModelID + "\x00" + record.CallerType
		item := byKey[key]
		if item == nil {
			item = &DailySummary{
				DateKey:    dateKey,
				Model:      record.ModelID,
				CallerType: record.CallerType,
			}
			byKey[key] = item
		}
		item.CallCount++
		item.TotalPromptTokens += record.PromptTokens
		item.TotalCompletionTokens += record.CompletionTokens
		item.TotalTokens += record.TotalTokens
		if record.Success {
			item.SuccessCount++
		} else {
			item.ErrorCount++
		}
	}
	out := make([]DailySummary, 0, len(byKey))
	for _, item := range byKey {
		out = append(out, *item)
	}
	sort.Slice(out, func(a, b int) bool {
		if out[a].DateKey != out[b].DateKey {
			return out[a].DateKey < out[b].DateKey
		}
		if out[a].Model != out[b].Model {
			return out[a].Model < out[b].Model
		}
		return out[a].CallerType < out[b].CallerType
	})
	return out, nil
}

func (s *Service) LLMUsageCleanup(beforeDate string) (int, error) {
	beforeDate = strings.TrimSpace(beforeDate)
	if beforeDate == "" {
		return 0, errors.New("beforeDate is required")
	}
	before, err := time.ParseInLocation("2006-01-02", beforeDate, time.Local)
	if err != nil {
		return 0, fmt.Errorf("invalid beforeDate: %w", err)
	}
	beforeMillis := before.UnixMilli()
	s.mu.Lock()
	defer s.mu.Unlock()
	cleared := 0
	for index := range s.state.Messages {
		if s.state.Messages[index].Timestamp >= beforeMillis || s.state.Messages[index].Meta == nil {
			continue
		}
		if _, ok := s.state.Messages[index].Meta["usage"]; !ok {
			continue
		}
		delete(s.state.Messages[index].Meta, "usage")
		cleared++
	}
	if cleared == 0 {
		return 0, nil
	}
	return cleared, s.flushLocked()
}

func (s *Service) BranchSession(sourceSessionID string, upToMessageID string) (Session, error) {
	sourceSessionID = strings.TrimSpace(sourceSessionID)
	upToMessageID = strings.TrimSpace(upToMessageID)
	if sourceSessionID == "" {
		return Session{}, errors.New("sourceSessionId is required")
	}
	if upToMessageID == "" {
		return Session{}, errors.New("upToMessageId is required")
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	sourceSession, ok := s.findSessionLocked(sourceSessionID)
	if !ok {
		return Session{}, fmt.Errorf("source session not found: %s", sourceSessionID)
	}
	if sourceSession.PersistStatus != "active" {
		return Session{}, fmt.Errorf("source session is not active: %s", sourceSessionID)
	}

	sourceMessages := s.messagesForSessionLocked(sourceSessionID)
	cutIndex := -1
	for index, message := range sourceMessages {
		if message.ID == upToMessageID {
			cutIndex = index
			break
		}
	}
	if cutIndex < 0 {
		return Session{}, fmt.Errorf("message %s not found in session %s", upToMessageID, sourceSessionID)
	}
	messagesToCopy := sourceMessages[:cutIndex+1]

	now := nowISO()
	newSessionID := "sess_" + randomToken(16)
	metadata := cloneMap(sourceSession.Metadata)
	if metadata == nil {
		metadata = map[string]any{}
	}
	metadata["branchedFrom"] = map[string]any{
		"sessionId":  sourceSessionID,
		"messageId":  upToMessageID,
		"branchedAt": now,
	}
	if len(metadata) == 0 {
		metadata = nil
	}

	var title *string
	if sourceSession.Title != nil {
		branchTitle := strings.TrimSpace(*sourceSession.Title)
		if branchTitle != "" {
			title = stringPtr(branchTitle + " (branch)")
		}
	}
	newSession := Session{
		ID:            newSessionID,
		Mode:          "chat",
		Title:         title,
		Description:   cloneStringPtr(sourceSession.Description),
		TitleLocked:   true,
		PersistStatus: "active",
		CreatedAt:     now,
		UpdatedAt:     now,
		GroupID:       cloneStringPtr(sourceSession.GroupID),
		Metadata:      metadata,
	}

	messageIDMap := make(map[string]string, len(messagesToCopy))
	blockIDMap := map[string]string{}
	for _, message := range messagesToCopy {
		messageIDMap[message.ID] = "msg_" + randomToken(16)
		for _, blockID := range message.BlockIDs {
			if _, ok := blockIDMap[blockID]; !ok {
				blockIDMap[blockID] = "blk_" + randomToken(16)
			}
		}
	}
	combinedIDMap := make(map[string]string, len(messageIDMap)+len(blockIDMap))
	for oldID, newID := range messageIDMap {
		combinedIDMap[oldID] = newID
	}
	for oldID, newID := range blockIDMap {
		combinedIDMap[oldID] = newID
	}

	s.state.Sessions = append(s.state.Sessions, newSession)
	for _, sourceMessage := range messagesToCopy {
		newMessage := cloneMessage(sourceMessage)
		newMessage.ID = messageIDMap[sourceMessage.ID]
		newMessage.SessionID = newSessionID
		newMessage.BlockIDs = remapStringSlice(sourceMessage.BlockIDs, blockIDMap)
		newMessage.ParentID = remapOptionalID(sourceMessage.ParentID, messageIDMap)
		newMessage.Supersedes = remapOptionalID(sourceMessage.Supersedes, messageIDMap)
		s.state.Messages = append(s.state.Messages, newMessage)
	}

	copiedMessageIDs := messageIDs(messagesToCopy)
	for _, sourceBlock := range s.state.Blocks {
		if !copiedMessageIDs[sourceBlock.MessageID] {
			continue
		}
		newBlockID, ok := blockIDMap[sourceBlock.ID]
		if !ok {
			continue
		}
		newMessageID, ok := messageIDMap[sourceBlock.MessageID]
		if !ok {
			continue
		}
		newBlock := cloneBlock(sourceBlock)
		newBlock.ID = newBlockID
		newBlock.MessageID = newMessageID
		newBlock.ToolInput = remapIDsInValue(newBlock.ToolInput, combinedIDMap)
		newBlock.ToolOutput = remapIDsInValue(newBlock.ToolOutput, combinedIDMap)
		s.state.Blocks = append(s.state.Blocks, newBlock)
	}

	if sourceState, ok := s.state.States[sourceSessionID]; ok {
		branchedState := cloneSessionState(sourceState)
		branchedState["sessionId"] = newSessionID
		branchedState["updatedAt"] = now
		delete(branchedState, "inputValue")
		delete(branchedState, "panelStates")
		delete(branchedState, "pendingContextRefs")
		delete(branchedState, "pendingContextRefsJson")
		s.state.States[newSessionID] = branchedState
	}

	if err := s.flushLocked(); err != nil {
		return Session{}, err
	}
	return newSession, nil
}

func (s *Service) SendMessage(request SendMessageRequest) (string, error) {
	if strings.TrimSpace(request.SessionID) == "" {
		return "", errors.New("sessionId is required")
	}
	content := request.Content
	userMessageID := valueOrID(request.UserMessageID, "msg_user_")
	assistantMessageID := valueOrID(request.AssistantMessageID, "msg_asst_")
	userBlockID := "blk_" + randomToken(16)
	assistantBlockID := "blk_" + randomToken(16)
	nowMillis := time.Now().UnixMilli()
	now := nowISO()

	s.mu.Lock()
	sessionIndex, ok := s.findSessionIndexLocked(request.SessionID)
	if !ok {
		s.mu.Unlock()
		return "", fmt.Errorf("session not found: %s", request.SessionID)
	}
	if !s.messageExistsLocked(userMessageID) {
		s.state.Messages = append(s.state.Messages, Message{
			ID:        userMessageID,
			SessionID: request.SessionID,
			Role:      "user",
			BlockIDs:  []string{userBlockID},
			Timestamp: nowMillis,
			Meta:      contextMeta(request),
		})
		s.state.Blocks = append(s.state.Blocks, Block{
			ID:        userBlockID,
			MessageID: userMessageID,
			Type:      "markdown",
			Status:    "complete",
			Content:   content,
			StartedAt: &nowMillis,
			EndedAt:   &nowMillis,
		})
	}
	if !s.messageExistsLocked(assistantMessageID) {
		s.state.Messages = append(s.state.Messages, Message{
			ID:        assistantMessageID,
			SessionID: request.SessionID,
			Role:      "assistant",
			BlockIDs:  []string{assistantBlockID},
			Timestamp: nowMillis + 1,
			Meta: map[string]any{
				"modelId":    "go-local-placeholder",
				"rawRequest": request,
			},
		})
		s.state.Blocks = append(s.state.Blocks, Block{
			ID:        assistantBlockID,
			MessageID: assistantMessageID,
			Type:      "markdown",
			Status:    "streaming",
			StartedAt: &nowMillis,
		})
	}
	if s.state.Sessions[sessionIndex].Title == nil || strings.TrimSpace(*s.state.Sessions[sessionIndex].Title) == "" {
		s.state.Sessions[sessionIndex].Title = stringPtr(deriveTitle(content))
	}
	s.state.Sessions[sessionIndex].UpdatedAt = now
	if err := s.flushLocked(); err != nil {
		s.mu.Unlock()
		return "", err
	}
	s.mu.Unlock()

	s.runAssistantStream(request.SessionID, assistantMessageID, assistantBlockID, content, request.Options, nowMillis)
	return assistantMessageID, nil
}

func (s *Service) ContinueMessage(sessionID string, messageID string, options map[string]any) (string, error) {
	if strings.TrimSpace(messageID) == "" {
		return "", errors.New("messageId is required")
	}
	nowMillis := time.Now().UnixMilli()
	s.mu.Lock()
	sessionIndex, ok := s.findSessionIndexLocked(sessionID)
	if !ok {
		s.mu.Unlock()
		return "", fmt.Errorf("session not found: %s", sessionID)
	}
	message, ok := s.findMessageLocked(messageID)
	if !ok || message.SessionID != sessionID {
		s.mu.Unlock()
		return "", fmt.Errorf("message not found: %s", messageID)
	}
	if message.Role == "user" {
		s.mu.Unlock()
		return "", errors.New("continue is only for assistant messages")
	}
	userContent, ok := s.previousUserContentBeforeLocked(sessionID, messageID)
	if !ok {
		userContent = "Continue the previous assistant response."
	}
	blockID := "blk_" + randomToken(16)
	s.state.Blocks = append(s.state.Blocks, Block{
		ID:        blockID,
		MessageID: messageID,
		Type:      "markdown",
		Status:    "streaming",
		StartedAt: &nowMillis,
	})
	index, _ := s.findMessageIndexLocked(messageID)
	s.state.Messages[index].BlockIDs = append(s.state.Messages[index].BlockIDs, blockID)
	s.state.Messages[index].Timestamp = nowMillis
	s.state.Messages[index].Meta = map[string]any{
		"modelId":    "go-local-placeholder",
		"rawOptions": options,
	}
	s.state.Sessions[sessionIndex].UpdatedAt = nowISO()
	if err := s.flushLocked(); err != nil {
		s.mu.Unlock()
		return "", err
	}
	s.mu.Unlock()

	s.runAssistantStream(sessionID, messageID, blockID, userContent, options, nowMillis)
	return messageID, nil
}

func (s *Service) CancelStream(sessionID string, messageID string) (bool, error) {
	s.cancelActiveStream(messageID)
	nowMillis := time.Now().UnixMilli()
	s.mu.Lock()
	defer s.mu.Unlock()
	sessionIndex, ok := s.findSessionIndexLocked(sessionID)
	if !ok {
		return false, fmt.Errorf("session not found: %s", sessionID)
	}
	message, ok := s.findMessageLocked(messageID)
	if !ok || message.SessionID != sessionID {
		return false, fmt.Errorf("message not found: %s", messageID)
	}
	for index := range s.state.Blocks {
		if s.state.Blocks[index].MessageID == messageID && s.state.Blocks[index].Status == "streaming" {
			s.state.Blocks[index].Status = "cancelled"
			s.state.Blocks[index].EndedAt = &nowMillis
		}
	}
	s.state.Sessions[sessionIndex].UpdatedAt = nowISO()
	return true, s.flushLocked()
}

func (s *Service) RetryMessage(sessionID string, messageID string, options map[string]any) (RetryMessageResponse, error) {
	nowMillis := time.Now().UnixMilli()
	s.mu.Lock()
	sessionIndex, ok := s.findSessionIndexLocked(sessionID)
	if !ok {
		s.mu.Unlock()
		return RetryMessageResponse{}, fmt.Errorf("session not found: %s", sessionID)
	}
	messageIndex, ok := s.findMessageIndexLocked(messageID)
	if !ok || s.state.Messages[messageIndex].SessionID != sessionID {
		s.mu.Unlock()
		return RetryMessageResponse{}, fmt.Errorf("message not found: %s", messageID)
	}
	if s.state.Messages[messageIndex].Role == "user" {
		s.mu.Unlock()
		return RetryMessageResponse{}, errors.New("retry is only for assistant messages; edit and resend user messages")
	}
	userContent, ok := s.previousUserContentBeforeLocked(sessionID, messageID)
	if !ok {
		userContent = "Retry the previous assistant response."
	}
	blockID := "blk_" + randomToken(16)
	s.deleteBlocksForMessageLocked(messageID)
	s.state.Blocks = append(s.state.Blocks, Block{
		ID:        blockID,
		MessageID: messageID,
		Type:      "markdown",
		Status:    "streaming",
		StartedAt: &nowMillis,
	})
	s.state.Messages[messageIndex].BlockIDs = []string{blockID}
	s.state.Messages[messageIndex].Timestamp = nowMillis
	s.state.Messages[messageIndex].Meta = map[string]any{
		"modelId":    "go-local-placeholder",
		"rawOptions": options,
	}
	s.state.Sessions[sessionIndex].UpdatedAt = nowISO()
	if err := s.flushLocked(); err != nil {
		s.mu.Unlock()
		return RetryMessageResponse{}, err
	}
	s.mu.Unlock()

	s.runAssistantStream(sessionID, messageID, blockID, userContent, options, nowMillis)
	return RetryMessageResponse{
		MessageID:         messageID,
		DeletedMessageIDs: []string{},
		DeletedVariantIDs: []string{},
	}, nil
}

func (s *Service) EditAndResend(request EditAndResendRequest) (EditAndResendResponse, error) {
	if strings.TrimSpace(request.SessionID) == "" {
		return EditAndResendResponse{}, errors.New("sessionId is required")
	}
	if strings.TrimSpace(request.MessageID) == "" {
		return EditAndResendResponse{}, errors.New("messageId is required")
	}
	if strings.TrimSpace(request.NewContent) == "" {
		return EditAndResendResponse{}, errors.New("newContent is required")
	}
	nowMillis := time.Now().UnixMilli()
	now := nowISO()
	assistantMessageID := valueOrID(request.AssistantMessageID, "msg_asst_")
	assistantBlockID := "blk_" + randomToken(16)

	s.mu.Lock()
	sessionIndex, ok := s.findSessionIndexLocked(request.SessionID)
	if !ok {
		s.mu.Unlock()
		return EditAndResendResponse{}, fmt.Errorf("session not found: %s", request.SessionID)
	}
	messageIndex, ok := s.findMessageIndexLocked(request.MessageID)
	if !ok || s.state.Messages[messageIndex].SessionID != request.SessionID {
		s.mu.Unlock()
		return EditAndResendResponse{}, fmt.Errorf("message not found: %s", request.MessageID)
	}
	if s.state.Messages[messageIndex].Role != "user" {
		s.mu.Unlock()
		return EditAndResendResponse{}, errors.New("can only edit user messages")
	}

	deletedIDs := s.deleteMessagesAfterLocked(request.SessionID, request.MessageID)
	s.replaceUserContentLocked(messageIndex, request.NewContent, nowMillis)
	if meta := editContextMeta(request); meta != nil {
		s.state.Messages[messageIndex].Meta = meta
	}
	s.state.Messages[messageIndex].Timestamp = nowMillis

	s.state.Messages = append(s.state.Messages, Message{
		ID:        assistantMessageID,
		SessionID: request.SessionID,
		Role:      "assistant",
		BlockIDs:  []string{assistantBlockID},
		Timestamp: nowMillis + 1,
		Meta: map[string]any{
			"modelId":    "go-local-placeholder",
			"rawOptions": request.Options,
		},
	})
	s.state.Blocks = append(s.state.Blocks, Block{
		ID:        assistantBlockID,
		MessageID: assistantMessageID,
		Type:      "markdown",
		Status:    "streaming",
		StartedAt: &nowMillis,
	})
	if s.state.Sessions[sessionIndex].Title == nil || strings.TrimSpace(*s.state.Sessions[sessionIndex].Title) == "" {
		s.state.Sessions[sessionIndex].Title = stringPtr(deriveTitle(request.NewContent))
	}
	s.state.Sessions[sessionIndex].UpdatedAt = now
	if err := s.flushLocked(); err != nil {
		s.mu.Unlock()
		return EditAndResendResponse{}, err
	}
	s.mu.Unlock()

	s.runAssistantStream(request.SessionID, assistantMessageID, assistantBlockID, request.NewContent, request.Options, nowMillis)
	return EditAndResendResponse{
		NewMessageID:      assistantMessageID,
		DeletedMessageIDs: deletedIDs,
	}, nil
}

func (s *Service) RespondToolApproval(sessionID string, toolCallID string, toolName string, approved bool, reason *string, remember bool, arguments map[string]any) (bool, error) {
	toolCallID = strings.TrimSpace(toolCallID)
	if toolCallID == "" {
		return false, errors.New("toolCallId is required")
	}
	now := nowISO()
	nowMillis := time.Now().UnixMilli()
	status := "rejected"
	if approved {
		status = "approved"
	}
	response := map[string]any{
		"toolCallId": toolCallID,
		"toolName":   toolName,
		"approved":   approved,
		"remember":   remember,
		"status":     status,
	}
	if reason != nil && strings.TrimSpace(*reason) != "" {
		response["reason"] = strings.TrimSpace(*reason)
	}
	if len(arguments) > 0 {
		response["arguments"] = arguments
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	s.upsertInteractionResponseLocked(InteractionResponse{
		ToolCallID: toolCallID,
		Kind:       "tool_approval",
		SessionID:  normalizeOptionalString(&sessionID),
		Response:   response,
		CreatedAt:  now,
	})
	if remember && strings.TrimSpace(toolName) != "" {
		s.upsertApprovalChoiceLocked(ApprovalChoice{
			ToolName:  strings.TrimSpace(toolName),
			Approved:  approved,
			Arguments: arguments,
			UpdatedAt: now,
		})
	}
	s.resolveBlocksByToolCallLocked(toolCallID, status, response, &nowMillis)
	return true, s.flushLocked()
}

func (s *Service) ClearApprovalHistory() (int, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	count := len(s.state.ApprovalChoices)
	s.state.ApprovalChoices = []ApprovalChoice{}
	return count, s.flushLocked()
}

func (s *Service) RespondAskUser(toolCallID string, selectedTexts []string, selectedIndices []int, customText *string, source string) (bool, error) {
	toolCallID = strings.TrimSpace(toolCallID)
	if toolCallID == "" {
		return false, errors.New("toolCallId is required")
	}
	if strings.TrimSpace(source) == "" {
		source = "user_click"
	}
	now := nowISO()
	nowMillis := time.Now().UnixMilli()
	response := map[string]any{
		"toolCallId":       toolCallID,
		"selected":         nonNilStrings(selectedTexts),
		"selected_texts":   nonNilStrings(selectedTexts),
		"selected_indices": selectedIndices,
		"custom_text":      nil,
		"source":           source,
	}
	if customText != nil && strings.TrimSpace(*customText) != "" {
		response["custom_text"] = strings.TrimSpace(*customText)
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	s.upsertInteractionResponseLocked(InteractionResponse{
		ToolCallID: toolCallID,
		Kind:       "ask_user",
		Response:   response,
		CreatedAt:  now,
	})
	s.resolveBlocksByToolCallLocked(toolCallID, "success", map[string]any{"result": response}, &nowMillis)
	return true, s.flushLocked()
}

func (s *Service) DeleteMessage(sessionID string, messageID string) (bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.findSessionIndexLocked(sessionID); !ok {
		return false, fmt.Errorf("session not found: %s", sessionID)
	}
	nextMessages := make([]Message, 0, len(s.state.Messages))
	deleted := false
	for _, message := range s.state.Messages {
		if message.ID == messageID && message.SessionID == sessionID {
			deleted = true
			continue
		}
		nextMessages = append(nextMessages, message)
	}
	s.state.Messages = nextMessages
	nextBlocks := make([]Block, 0, len(s.state.Blocks))
	for _, block := range s.state.Blocks {
		if block.MessageID != messageID {
			nextBlocks = append(nextBlocks, block)
		}
	}
	s.state.Blocks = nextBlocks
	return deleted, s.flushLocked()
}

func (s *Service) UpdateBlockContent(blockID string, content string) (bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	index, ok := s.findBlockIndexLocked(blockID)
	if !ok {
		return false, fmt.Errorf("block not found: %s", blockID)
	}
	nowMillis := time.Now().UnixMilli()
	s.state.Blocks[index].Content = content
	s.state.Blocks[index].Status = "complete"
	s.state.Blocks[index].EndedAt = &nowMillis
	return true, s.flushLocked()
}

func (s *Service) UpsertStreamingBlock(blockID string, messageID string, blockType string, content string, sessionID *string) (bool, error) {
	if strings.TrimSpace(blockID) == "" || strings.TrimSpace(messageID) == "" {
		return false, errors.New("blockId and messageId are required")
	}
	if strings.TrimSpace(blockType) == "" {
		blockType = "markdown"
	}
	nowMillis := time.Now().UnixMilli()
	s.mu.Lock()
	defer s.mu.Unlock()
	if index, ok := s.findBlockIndexLocked(blockID); ok {
		s.state.Blocks[index].Content = content
		s.state.Blocks[index].Status = "streaming"
		s.state.Blocks[index].EndedAt = nil
		return true, s.flushLocked()
	}
	if _, ok := s.findMessageIndexLocked(messageID); !ok {
		resolvedSessionID := ""
		if sessionID != nil {
			resolvedSessionID = *sessionID
		}
		if resolvedSessionID == "" {
			return false, fmt.Errorf("message not found: %s", messageID)
		}
		s.state.Messages = append(s.state.Messages, Message{
			ID:        messageID,
			SessionID: resolvedSessionID,
			Role:      "assistant",
			BlockIDs:  []string{blockID},
			Timestamp: nowMillis,
		})
	} else {
		messageIndex, _ := s.findMessageIndexLocked(messageID)
		if !containsString(s.state.Messages[messageIndex].BlockIDs, blockID) {
			s.state.Messages[messageIndex].BlockIDs = append(s.state.Messages[messageIndex].BlockIDs, blockID)
		}
	}
	s.state.Blocks = append(s.state.Blocks, Block{
		ID:        blockID,
		MessageID: messageID,
		Type:      blockType,
		Status:    "streaming",
		Content:   content,
		StartedAt: &nowMillis,
	})
	return true, s.flushLocked()
}

func (s *Service) setSessionStatus(sessionID string, status string) (bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	index, ok := s.findSessionIndexLocked(sessionID)
	if !ok {
		return false, fmt.Errorf("session not found: %s", sessionID)
	}
	s.state.Sessions[index].PersistStatus = status
	s.state.Sessions[index].UpdatedAt = nowISO()
	return true, s.flushLocked()
}

func (s *Service) findSessionLocked(sessionID string) (Session, bool) {
	for _, session := range s.state.Sessions {
		if session.ID == sessionID {
			return session, true
		}
	}
	return Session{}, false
}

func (s *Service) findSessionIndexLocked(sessionID string) (int, bool) {
	for index, session := range s.state.Sessions {
		if session.ID == sessionID {
			return index, true
		}
	}
	return -1, false
}

func (s *Service) messageExistsLocked(messageID string) bool {
	_, ok := s.findMessageLocked(messageID)
	return ok
}

func (s *Service) findMessageLocked(messageID string) (Message, bool) {
	for _, message := range s.state.Messages {
		if message.ID == messageID {
			return message, true
		}
	}
	return Message{}, false
}

func (s *Service) findMessageIndexLocked(messageID string) (int, bool) {
	for index, message := range s.state.Messages {
		if message.ID == messageID {
			return index, true
		}
	}
	return -1, false
}

func (s *Service) findBlockIndexLocked(blockID string) (int, bool) {
	for index, block := range s.state.Blocks {
		if block.ID == blockID {
			return index, true
		}
	}
	return -1, false
}

func (s *Service) upsertInteractionResponseLocked(response InteractionResponse) {
	for index := range s.state.InteractionResponses {
		if s.state.InteractionResponses[index].ToolCallID == response.ToolCallID && s.state.InteractionResponses[index].Kind == response.Kind {
			s.state.InteractionResponses[index] = response
			return
		}
	}
	s.state.InteractionResponses = append(s.state.InteractionResponses, response)
}

func (s *Service) upsertApprovalChoiceLocked(choice ApprovalChoice) {
	for index := range s.state.ApprovalChoices {
		if s.state.ApprovalChoices[index].ToolName == choice.ToolName {
			s.state.ApprovalChoices[index] = choice
			return
		}
	}
	s.state.ApprovalChoices = append(s.state.ApprovalChoices, choice)
}

func (s *Service) resolveBlocksByToolCallLocked(toolCallID string, status string, output any, endedAt *int64) {
	for index := range s.state.Blocks {
		if !blockMatchesToolCall(s.state.Blocks[index], toolCallID) {
			continue
		}
		s.state.Blocks[index].Status = status
		s.state.Blocks[index].ToolOutput = output
		s.state.Blocks[index].EndedAt = endedAt
	}
}

func blockMatchesToolCall(block Block, toolCallID string) bool {
	if block.ToolCallID != nil && *block.ToolCallID == toolCallID {
		return true
	}
	if block.ID == toolCallID || block.ID == "approval_"+toolCallID || block.ID == "ask_user_"+toolCallID {
		return true
	}
	if block.ToolInput != nil {
		if object, ok := block.ToolInput.(map[string]any); ok {
			if value, ok := object["toolCallId"].(string); ok && value == toolCallID {
				return true
			}
		}
	}
	return false
}

func (s *Service) deleteBlocksForMessageLocked(messageID string) {
	nextBlocks := make([]Block, 0, len(s.state.Blocks))
	for _, block := range s.state.Blocks {
		if block.MessageID != messageID {
			nextBlocks = append(nextBlocks, block)
		}
	}
	s.state.Blocks = nextBlocks
}

func (s *Service) deleteMessagesAfterLocked(sessionID string, messageID string) []string {
	messages := s.messagesForSessionLocked(sessionID)
	deleteSet := map[string]bool{}
	seenTarget := false
	for _, message := range messages {
		if seenTarget {
			deleteSet[message.ID] = true
			continue
		}
		if message.ID == messageID {
			seenTarget = true
		}
	}
	if len(deleteSet) == 0 {
		return []string{}
	}
	deletedIDs := make([]string, 0, len(deleteSet))
	nextMessages := make([]Message, 0, len(s.state.Messages))
	for _, message := range s.state.Messages {
		if deleteSet[message.ID] {
			deletedIDs = append(deletedIDs, message.ID)
			continue
		}
		nextMessages = append(nextMessages, message)
	}
	s.state.Messages = nextMessages
	nextBlocks := make([]Block, 0, len(s.state.Blocks))
	for _, block := range s.state.Blocks {
		if !deleteSet[block.MessageID] {
			nextBlocks = append(nextBlocks, block)
		}
	}
	s.state.Blocks = nextBlocks
	sort.Strings(deletedIDs)
	return deletedIDs
}

func (s *Service) replaceUserContentLocked(messageIndex int, content string, timestamp int64) {
	messageID := s.state.Messages[messageIndex].ID
	if len(s.state.Messages[messageIndex].BlockIDs) == 0 {
		blockID := "blk_" + randomToken(16)
		s.state.Messages[messageIndex].BlockIDs = []string{blockID}
		s.state.Blocks = append(s.state.Blocks, Block{
			ID:        blockID,
			MessageID: messageID,
			Type:      "markdown",
			Status:    "complete",
			Content:   content,
			StartedAt: &timestamp,
			EndedAt:   &timestamp,
		})
		return
	}
	for _, blockID := range s.state.Messages[messageIndex].BlockIDs {
		if blockIndex, ok := s.findBlockIndexLocked(blockID); ok {
			s.state.Blocks[blockIndex].Content = content
			s.state.Blocks[blockIndex].Status = "complete"
			s.state.Blocks[blockIndex].EndedAt = &timestamp
			return
		}
	}
}

func (s *Service) previousUserContentBeforeLocked(sessionID string, beforeMessageID string) (string, bool) {
	messages := s.messagesForSessionLocked(sessionID)
	latestContent := ""
	for _, message := range messages {
		if message.ID == beforeMessageID {
			break
		}
		if message.Role != "user" {
			continue
		}
		content, ok := s.firstBlockContentLocked(message)
		if ok {
			latestContent = content
		}
	}
	if strings.TrimSpace(latestContent) == "" {
		return "", false
	}
	return latestContent, true
}

func (s *Service) firstBlockContentLocked(message Message) (string, bool) {
	for _, blockID := range message.BlockIDs {
		blockIndex, ok := s.findBlockIndexLocked(blockID)
		if !ok {
			continue
		}
		content := s.state.Blocks[blockIndex].Content
		if strings.TrimSpace(content) != "" {
			return content, true
		}
	}
	return "", false
}

func (s *Service) setSessionTagsLocked(sessionIndex int, tags []string) {
	metadata := s.state.Sessions[sessionIndex].Metadata
	if metadata == nil {
		metadata = map[string]any{}
	}
	if len(tags) == 0 {
		delete(metadata, "tags")
	} else {
		metadata["tags"] = tags
	}
	if len(metadata) == 0 {
		s.state.Sessions[sessionIndex].Metadata = nil
		return
	}
	s.state.Sessions[sessionIndex].Metadata = metadata
}

func (s *Service) findGroupLocked(groupID string) (Group, bool) {
	for _, group := range s.state.Groups {
		if group.ID == groupID {
			return group, true
		}
	}
	return Group{}, false
}

func (s *Service) findGroupIndexLocked(groupID string) (int, bool) {
	for index, group := range s.state.Groups {
		if group.ID == groupID {
			return index, true
		}
	}
	return -1, false
}

func (s *Service) groupIsActiveLocked(groupID string) bool {
	group, ok := s.findGroupLocked(groupID)
	return ok && group.PersistStatus == "active"
}

func (s *Service) nextGroupSortLocked(workspaceID *string) int {
	maxOrder := -1
	for _, group := range s.state.Groups {
		if workspaceID != nil && *workspaceID != "" {
			if group.WorkspaceID == nil || *group.WorkspaceID != *workspaceID {
				continue
			}
		}
		if group.PersistStatus == "active" && group.SortOrder > maxOrder {
			maxOrder = group.SortOrder
		}
	}
	return maxOrder + 1
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
	if s.state.Sessions == nil {
		s.state.Sessions = []Session{}
	}
	if s.state.Groups == nil {
		s.state.Groups = []Group{}
	}
	if s.state.States == nil {
		s.state.States = map[string]SessionState{}
	}
	if s.state.Messages == nil {
		s.state.Messages = []Message{}
	}
	if s.state.Blocks == nil {
		s.state.Blocks = []Block{}
	}
	if s.state.InteractionResponses == nil {
		s.state.InteractionResponses = []InteractionResponse{}
	}
	if s.state.ApprovalChoices == nil {
		s.state.ApprovalChoices = []ApprovalChoice{}
	}
	return nil
}

func (s *Service) flushLocked() error {
	return storage.WriteJSONAtomic(s.path, s.state)
}

func matchesGroupFilter(session Session, groupID *string) bool {
	if groupID == nil {
		return true
	}
	switch *groupID {
	case "*":
		return session.GroupID != nil && *session.GroupID != ""
	case "":
		return session.GroupID == nil || *session.GroupID == ""
	default:
		return session.GroupID != nil && *session.GroupID == *groupID
	}
}

func sortSessions(sessions []Session) {
	sort.SliceStable(sessions, func(a, b int) bool {
		return sessions[a].UpdatedAt > sessions[b].UpdatedAt
	})
}

func sortGroups(groups []Group) {
	sort.SliceStable(groups, func(a, b int) bool {
		if groups[a].SortOrder != groups[b].SortOrder {
			return groups[a].SortOrder < groups[b].SortOrder
		}
		return groups[a].UpdatedAt > groups[b].UpdatedAt
	})
}

func (s *Service) messagesForSessionLocked(sessionID string) []Message {
	out := make([]Message, 0)
	for _, message := range s.state.Messages {
		if message.SessionID == sessionID {
			out = append(out, message)
		}
	}
	sort.SliceStable(out, func(a, b int) bool {
		return out[a].Timestamp < out[b].Timestamp
	})
	return out
}

func (s *Service) blocksForMessagesLocked(ids map[string]bool) []Block {
	out := make([]Block, 0)
	for _, block := range s.state.Blocks {
		if ids[block.MessageID] {
			out = append(out, block)
		}
	}
	sort.SliceStable(out, func(a, b int) bool {
		left := int64(0)
		right := int64(0)
		if out[a].StartedAt != nil {
			left = *out[a].StartedAt
		}
		if out[b].StartedAt != nil {
			right = *out[b].StartedAt
		}
		return left < right
	})
	return out
}

func messageIDs(messages []Message) map[string]bool {
	out := make(map[string]bool, len(messages))
	for _, message := range messages {
		out[message.ID] = true
	}
	return out
}

func valueOrID(value *string, prefix string) string {
	if value != nil && strings.TrimSpace(*value) != "" {
		return strings.TrimSpace(*value)
	}
	return prefix + randomToken(16)
}

func contextMeta(request SendMessageRequest) map[string]any {
	meta := map[string]any{}
	if len(request.UserContextRefs) > 0 || len(request.PathMap) > 0 {
		snapshot := map[string]any{}
		if len(request.UserContextRefs) > 0 {
			snapshot["userRefs"] = request.UserContextRefs
		}
		if len(request.PathMap) > 0 {
			snapshot["pathMap"] = request.PathMap
		}
		meta["contextSnapshot"] = snapshot
	}
	if len(meta) == 0 {
		return nil
	}
	return meta
}

func editContextMeta(request EditAndResendRequest) map[string]any {
	meta := map[string]any{}
	if len(request.NewContextRefs) > 0 || len(request.NewPathMap) > 0 {
		snapshot := map[string]any{}
		if len(request.NewContextRefs) > 0 {
			snapshot["userRefs"] = request.NewContextRefs
		}
		if len(request.NewPathMap) > 0 {
			snapshot["pathMap"] = request.NewPathMap
		}
		meta["contextSnapshot"] = snapshot
	}
	if len(meta) == 0 {
		return nil
	}
	return meta
}

func cloneMessage(message Message) Message {
	return Message{
		ID:                 message.ID,
		SessionID:          message.SessionID,
		Role:               message.Role,
		BlockIDs:           append([]string{}, message.BlockIDs...),
		Timestamp:          message.Timestamp,
		PersistentStableID: cloneStringPtr(message.PersistentStableID),
		ParentID:           cloneStringPtr(message.ParentID),
		Supersedes:         cloneStringPtr(message.Supersedes),
		Meta:               cloneMap(message.Meta),
		Attachments:        cloneAnySlice(message.Attachments),
	}
}

func cloneBlock(block Block) Block {
	return Block{
		ID:           block.ID,
		MessageID:    block.MessageID,
		Type:         block.Type,
		Status:       block.Status,
		Content:      block.Content,
		ToolName:     cloneStringPtr(block.ToolName),
		ToolCallID:   cloneStringPtr(block.ToolCallID),
		ToolInput:    cloneAny(block.ToolInput),
		ToolOutput:   cloneAny(block.ToolOutput),
		Citations:    cloneAnySlice(block.Citations),
		Error:        cloneStringPtr(block.Error),
		StartedAt:    cloneInt64Ptr(block.StartedAt),
		EndedAt:      cloneInt64Ptr(block.EndedAt),
		FirstChunkAt: cloneInt64Ptr(block.FirstChunkAt),
	}
}

func cloneSessionState(state SessionState) SessionState {
	if state == nil {
		return nil
	}
	return SessionState(cloneMap(map[string]any(state)))
}

func cloneMap(value map[string]any) map[string]any {
	if value == nil {
		return nil
	}
	out := make(map[string]any, len(value))
	for key, item := range value {
		out[key] = cloneAny(item)
	}
	return out
}

func cloneAnySlice(value []any) []any {
	if value == nil {
		return nil
	}
	out := make([]any, len(value))
	for index, item := range value {
		out[index] = cloneAny(item)
	}
	return out
}

func cloneAny(value any) any {
	switch typed := value.(type) {
	case map[string]any:
		return cloneMap(typed)
	case []any:
		return cloneAnySlice(typed)
	case []string:
		return append([]string{}, typed...)
	case []map[string]any:
		out := make([]map[string]any, len(typed))
		for index, item := range typed {
			out[index] = cloneMap(item)
		}
		return out
	default:
		return typed
	}
}

func remapIDsInValue(value any, ids map[string]string) any {
	switch typed := value.(type) {
	case string:
		if replacement, ok := ids[typed]; ok {
			return replacement
		}
		return typed
	case map[string]any:
		out := make(map[string]any, len(typed))
		for key, item := range typed {
			out[key] = remapIDsInValue(item, ids)
		}
		return out
	case []any:
		out := make([]any, len(typed))
		for index, item := range typed {
			out[index] = remapIDsInValue(item, ids)
		}
		return out
	case []string:
		return remapStringSlice(typed, ids)
	default:
		return cloneAny(typed)
	}
}

func remapStringSlice(values []string, ids map[string]string) []string {
	if values == nil {
		return nil
	}
	out := make([]string, len(values))
	for index, value := range values {
		if replacement, ok := ids[value]; ok {
			out[index] = replacement
			continue
		}
		out[index] = value
	}
	return out
}

func remapOptionalID(value *string, ids map[string]string) *string {
	if value == nil {
		return nil
	}
	replacement, ok := ids[*value]
	if !ok {
		return nil
	}
	return &replacement
}

func cloneStringPtr(value *string) *string {
	if value == nil {
		return nil
	}
	cloned := *value
	return &cloned
}

func cloneInt64Ptr(value *int64) *int64 {
	if value == nil {
		return nil
	}
	cloned := *value
	return &cloned
}

func deriveTitle(content string) string {
	title := strings.TrimSpace(content)
	if title == "" {
		return "New chat"
	}
	title = strings.ReplaceAll(title, "\n", " ")
	if len([]rune(title)) > 40 {
		runes := []rune(title)
		return string(runes[:40])
	}
	return title
}

func containsString(values []string, needle string) bool {
	for _, value := range values {
		if value == needle {
			return true
		}
	}
	return false
}

func stringPtr(value string) *string {
	return &value
}

func intPtr(value int) *int {
	return &value
}

func floatPtr(value float64) *float64 {
	return &value
}

func normalizeOptionalString(value *string) *string {
	if value == nil {
		return nil
	}
	trimmed := strings.TrimSpace(*value)
	if trimmed == "" {
		return nil
	}
	return &trimmed
}

func normalizeMetadata(metadata map[string]any) map[string]any {
	if len(metadata) == 0 {
		return nil
	}
	return metadata
}

func tagsFromMetadata(metadata map[string]any) []string {
	if len(metadata) == 0 {
		return []string{}
	}
	switch raw := metadata["tags"].(type) {
	case []string:
		return dedupeTags(raw)
	case []any:
		tags := make([]string, 0, len(raw))
		for _, value := range raw {
			if tag, ok := value.(string); ok {
				tags = append(tags, tag)
			}
		}
		return dedupeTags(tags)
	default:
		return []string{}
	}
}

func dedupeTags(values []string) []string {
	out := make([]string, 0, len(values))
	seen := map[string]bool{}
	for _, value := range values {
		tag := strings.TrimSpace(value)
		if tag == "" || seen[tag] {
			continue
		}
		seen[tag] = true
		out = append(out, tag)
	}
	return out
}

func nonNilStrings(values []string) []string {
	if values == nil {
		return []string{}
	}
	return values
}

func validMode(mode string) bool {
	switch mode {
	case "chat", "analysis", "review", "textbook", "bridge", "general_chat":
		return true
	default:
		return false
	}
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
