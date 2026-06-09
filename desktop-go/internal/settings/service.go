package settings

import (
	"bytes"
	"context"
	"crypto/rand"
	"deep-student-go/internal/storage"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math/big"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"
)

const (
	attachmentRootFolderIDKey      = "attachment_root_folder_id"
	attachmentRootFolderTitleKey   = "attachment_root_folder_title"
	backupConfigKey                = "backup.config"
	apiConfigurationsKey           = "api_configurations"
	vendorConfigsKey               = "vendor_configs"
	modelProfilesKey               = "model_profiles"
	modelAssignmentsKey            = "model_assignments"
	webSearchProviderStrategiesKey = "web_search.provider_strategies"
	mcpToolsListKey                = "mcp.tools.list"
	ocrAvailableModelsKey          = "ocr.available_models"
	ocrEngineTypeKey               = "ocr.engine_type"
	ocrEnableThinkingKey           = "ocr.enable_thinking"
	memoryRootFolderIDKey          = "memory_root_folder_id"
	memoryRootFolderTitleKey       = "memory_root_folder_title"
	memoryAutoCreateSubfoldersKey  = "auto_create_subfolders"
	memoryDefaultCategoryKey       = "default_category"
	memoryPrivacyModeKey           = "privacy_mode"
	memoryAutoExtractFrequencyKey  = "auto_extract_frequency"
	systemOCRConfigID              = "__system_ocr__"
)

var glmVisionRE = regexp.MustCompile(`(?i)glm-(?:4\.[5-9]|4\.\d{2,}|[5-9](?:\.\d+)?)v`)

var cnTrustedSites = []string{
	"edu.cn",
	"tsinghua.edu.cn",
	"pku.edu.cn",
	"fudan.edu.cn",
	"sjtu.edu.cn",
	"zju.edu.cn",
	"nju.edu.cn",
	"ustc.edu.cn",
	"bit.edu.cn",
	"buaa.edu.cn",
	"gov.cn",
	"beijing.gov.cn",
	"shanghai.gov.cn",
	"guangzhou.gov.cn",
	"shenzhen.gov.cn",
	"xinhuanet.com",
	"people.com.cn",
	"cctv.com",
	"chinanews.com.cn",
	"ce.cn",
	"runoob.com",
	"w3school.com.cn",
	"liaoxuefeng.com",
	"cnblogs.com",
	"csdn.net",
	"jianshu.com",
	"segmentfault.com",
	"juejin.cn",
	"zhihu.com",
	"oschina.net",
	"github.com",
	"gitee.com",
	"coding.net",
	"cas.cn",
	"cass.cn",
	"cnki.net",
	"wanfangdata.com.cn",
	"baidu.com",
	"tencent.com",
	"alibaba.com",
	"huawei.com",
	"xiaomi.com",
	"bytedance.com",
	"infoq.cn",
	"51cto.com",
	"iteye.com",
	"ibiblio.org",
	"apache.org",
	"python.org",
	"nodejs.org",
	"mysql.com",
	"postgresql.org",
}

type webSearchEngineDefinition struct {
	ID   string
	Name string
}

var webSearchEngines = []webSearchEngineDefinition{
	{ID: "google_cse", Name: "Google CSE"},
	{ID: "serpapi", Name: "SerpAPI"},
	{ID: "tavily", Name: "Tavily"},
	{ID: "brave", Name: "Brave"},
	{ID: "searxng", Name: "SearXNG"},
	{ID: "zhipu", Name: "智谱 AI"},
	{ID: "bocha", Name: "博查 AI"},
}

var defaultModelAdapterOptions = []ModelAdapterOption{
	{Value: "general", Label: "OpenAI Compatible", Description: "Standard OpenAI-compatible request format", IsDefault: true},
	{Value: "google", Label: "Google Gemini", Description: "Gemini request format with thinking controls", IsDefault: true},
	{Value: "anthropic", Label: "Anthropic Claude", Description: "Claude request format with extended thinking controls", IsDefault: true},
	{Value: "deepseek", Label: "DeepSeek", Description: "DeepSeek reasoning and thinking parameter format", IsDefault: true},
	{Value: "qwen", Label: "Qwen", Description: "DashScope/Qwen thinking parameter format", IsDefault: true},
	{Value: "zhipu", Label: "Zhipu GLM", Description: "GLM reasoning parameter format", IsDefault: true},
	{Value: "doubao", Label: "Doubao", Description: "Doubao thinking model format", IsDefault: true},
	{Value: "moonshot", Label: "Kimi/Moonshot", Description: "Kimi/Moonshot reasoning content format", IsDefault: true},
	{Value: "grok", Label: "xAI Grok", Description: "Grok request constraints and reasoning effort format", IsDefault: true},
	{Value: "minimax", Label: "MiniMax", Description: "MiniMax reasoning split format", IsDefault: true},
	{Value: "mimo", Label: "Xiaomi MiMo", Description: "MiMo thinking.type and reasoning_content format", IsDefault: true},
}

type webSearchConfig struct {
	Engine          string
	Timeout         time.Duration
	GoogleCSE       string
	GoogleCSECX     string
	SerpAPI         string
	Tavily          string
	TavilyDepth     string
	Brave           string
	SearxngEndpoint string
	SearxngAPIKey   string
	Zhipu           string
	Bocha           string
}

type Service struct {
	mu         sync.RWMutex
	path       string
	data       map[string]string
	httpClient *http.Client
}

type AttachmentConfig struct {
	AttachmentRootFolderID    *string `json:"attachmentRootFolderId"`
	AttachmentRootFolderTitle *string `json:"attachmentRootFolderTitle"`
}

type MemoryConfig struct {
	MemoryRootFolderID    *string `json:"memoryRootFolderId"`
	MemoryRootFolderTitle *string `json:"memoryRootFolderTitle"`
	AutoCreateSubfolders  bool    `json:"autoCreateSubfolders"`
	DefaultCategory       string  `json:"defaultCategory"`
	PrivacyMode           bool    `json:"privacyMode"`
	AutoExtractFrequency  string  `json:"autoExtractFrequency"`
}

type ModelAdapterOption struct {
	Value       string `json:"value"`
	Label       string `json:"label"`
	Description string `json:"description,omitempty"`
	IsDefault   bool   `json:"is_default"`
}

type CNWhitelistUserConfig struct {
	Enabled        bool     `json:"enabled"`
	UseDefaultList bool     `json:"use_default_list"`
	CustomSites    []string `json:"custom_sites"`
}

type CNWhitelistConfigResult struct {
	DefaultSites []string              `json:"default_sites"`
	UserConfig   CNWhitelistUserConfig `json:"user_config"`
}

type ProviderSpecialHandling struct {
	Handle429RetryAfter             bool    `json:"handle_429_retry_after"`
	ExponentialBackoffOn5xx         bool    `json:"exponential_backoff_on_5xx"`
	CircuitBreakerEnabled           bool    `json:"circuit_breaker_enabled"`
	CircuitBreakerFailureThreshold  *uint32 `json:"circuit_breaker_failure_threshold,omitempty"`
	CircuitBreakerRecoveryTimeoutMS *uint64 `json:"circuit_breaker_recovery_timeout_ms,omitempty"`
}

type ProviderStrategy struct {
	TimeoutMS             *uint64                  `json:"timeout_ms,omitempty"`
	MaxRetries            *uint32                  `json:"max_retries,omitempty"`
	InitialRetryDelayMS   *uint64                  `json:"initial_retry_delay_ms,omitempty"`
	MaxRetryDelayMS       *uint64                  `json:"max_retry_delay_ms,omitempty"`
	BackoffMultiplier     *float64                 `json:"backoff_multiplier,omitempty"`
	MaxConcurrentRequests *uint32                  `json:"max_concurrent_requests,omitempty"`
	RateLimitPerMinute    *uint32                  `json:"rate_limit_per_minute,omitempty"`
	CacheEnabled          *bool                    `json:"cache_enabled,omitempty"`
	CacheTTLSeconds       *uint64                  `json:"cache_ttl_seconds,omitempty"`
	CacheMaxEntries       *uint64                  `json:"cache_max_entries,omitempty"`
	SpecialHandling       *ProviderSpecialHandling `json:"special_handling,omitempty"`
}

type ProviderStrategies struct {
	Default   ProviderStrategy  `json:"default"`
	GoogleCSE *ProviderStrategy `json:"google_cse,omitempty"`
	SerpAPI   *ProviderStrategy `json:"serpapi,omitempty"`
	Tavily    *ProviderStrategy `json:"tavily,omitempty"`
	Brave     *ProviderStrategy `json:"brave,omitempty"`
	Searxng   *ProviderStrategy `json:"searxng,omitempty"`
	Zhipu     *ProviderStrategy `json:"zhipu,omitempty"`
	Bocha     *ProviderStrategy `json:"bocha,omitempty"`
}

type ProviderStrategiesConfigResult struct {
	ProviderStrategies ProviderStrategies `json:"provider_strategies"`
	ConfigKeys         map[string]string  `json:"config_keys"`
}

type BackupConfig struct {
	BackupDirectory         *string  `json:"backupDirectory"`
	AutoBackupEnabled       bool     `json:"autoBackupEnabled"`
	AutoBackupIntervalHours uint32   `json:"autoBackupIntervalHours"`
	MaxBackupCount          *uint32  `json:"maxBackupCount"`
	SlimBackup              bool     `json:"slimBackup"`
	BackupTiers             []string `json:"backupTiers,omitempty"`
}

type BasicStatistics struct {
	TotalMistakes  int            `json:"total_mistakes"`
	TotalReviews   int            `json:"total_reviews"`
	TypeStats      map[string]int `json:"type_stats"`
	TagStats       map[string]int `json:"tag_stats"`
	RecentMistakes []any          `json:"recent_mistakes"`
}

type ImageStatistics struct {
	TotalFiles     int   `json:"total_files"`
	TotalSizeBytes int64 `json:"total_size_bytes"`
}

type EnhancedStatistics struct {
	BasicStats      BasicStatistics `json:"basic_stats"`
	ImageStats      ImageStatistics `json:"image_stats"`
	RecentAdditions int             `json:"recent_additions"`
	QualityScore    float64         `json:"quality_score"`
	MonthlyTrend    []any           `json:"monthly_trend"`
	Timestamp       string          `json:"timestamp"`
}

type APIConfigStatus struct {
	ConfigCount    int  `json:"config_count"`
	EnabledCount   int  `json:"enabled_count"`
	HasAssignments bool `json:"has_assignments"`
	NeedsRecovery  bool `json:"needs_recovery"`
}

type SearchEngineTestResult struct {
	Ok           bool    `json:"ok"`
	Message      string  `json:"message"`
	ResponseTime uint64  `json:"response_time,omitempty"`
	TestQuery    string  `json:"test_query,omitempty"`
	ErrorDetails *string `json:"error_details,omitempty"`
	ResultsCount *int    `json:"results_count,omitempty"`
}

type WebSearchConnectivityResult struct {
	Success bool                   `json:"success"`
	Usage   map[string]any         `json:"usage,omitempty"`
	Detail  SearchEngineTestResult `json:"detail,omitempty"`
}

type SearchEngineHealthStatus struct {
	Name         string `json:"name"`
	Status       string `json:"status"`
	Message      string `json:"message"`
	ElapsedMS    uint64 `json:"elapsed_ms"`
	ResultsCount *int   `json:"results_count,omitempty"`
}

type SearchEngineHealthSummary struct {
	Total      int `json:"total"`
	Configured int `json:"configured"`
	Success    int `json:"success"`
	Failed     int `json:"failed"`
}

type SearchEngineHealthReport struct {
	Results   map[string]SearchEngineHealthStatus `json:"results"`
	Summary   SearchEngineHealthSummary           `json:"summary"`
	Timestamp string                              `json:"timestamp"`
}

type ApiConfig struct {
	ID                       string            `json:"id"`
	Name                     string            `json:"name"`
	VendorID                 *string           `json:"vendorId,omitempty"`
	VendorName               *string           `json:"vendorName,omitempty"`
	ProviderType             *string           `json:"providerType,omitempty"`
	ProviderScope            *string           `json:"providerScope,omitempty"`
	ApiProtocol              *string           `json:"apiProtocol,omitempty"`
	SupportsOpenAIResponses  *bool             `json:"supportsOpenAIResponses,omitempty"`
	ApiKey                   string            `json:"apiKey"`
	BaseUrl                  string            `json:"baseUrl"`
	Model                    string            `json:"model"`
	IsMultimodal             bool              `json:"isMultimodal"`
	IsReasoning              bool              `json:"isReasoning"`
	IsEmbedding              bool              `json:"isEmbedding"`
	IsReranker               bool              `json:"isReranker"`
	IsImageGeneration        bool              `json:"isImageGeneration,omitempty"`
	Enabled                  bool              `json:"enabled"`
	ModelAdapter             string            `json:"modelAdapter"`
	MaxOutputTokens          uint32            `json:"maxOutputTokens,omitempty"`
	Temperature              float32           `json:"temperature,omitempty"`
	SupportsTools            bool              `json:"supportsTools,omitempty"`
	GeminiApiVersion         string            `json:"geminiApiVersion,omitempty"`
	IsBuiltin                bool              `json:"isBuiltin,omitempty"`
	IsReadOnly               bool              `json:"isReadOnly,omitempty"`
	ReasoningEffort          *string           `json:"reasoningEffort,omitempty"`
	ThinkingEnabled          bool              `json:"thinkingEnabled,omitempty"`
	ThinkingBudget           *int              `json:"thinkingBudget,omitempty"`
	IncludeThoughts          bool              `json:"includeThoughts,omitempty"`
	MinP                     *float32          `json:"minP,omitempty"`
	TopK                     *uint32           `json:"topK,omitempty"`
	EnableThinking           *bool             `json:"enableThinking,omitempty"`
	SupportsReasoning        bool              `json:"supportsReasoning,omitempty"`
	Headers                  map[string]string `json:"headers,omitempty"`
	TopPOverride             *float32          `json:"topPOverride,omitempty"`
	FrequencyPenaltyOverride *float32          `json:"frequencyPenaltyOverride,omitempty"`
	PresencePenaltyOverride  *float32          `json:"presencePenaltyOverride,omitempty"`
	RepetitionPenalty        *float32          `json:"repetitionPenalty,omitempty"`
	ReasoningSplit           *bool             `json:"reasoningSplit,omitempty"`
	Effort                   *string           `json:"effort,omitempty"`
	Verbosity                *string           `json:"verbosity,omitempty"`
	IsFavorite               bool              `json:"isFavorite,omitempty"`
	MaxTokensLimit           *uint32           `json:"maxTokensLimit,omitempty"`
	ContextWindow            *uint32           `json:"contextWindow,omitempty"`
	IsAudioTranscription     bool              `json:"isAudioTranscription,omitempty"`
}

type VendorConfig struct {
	ID                      string            `json:"id"`
	Name                    string            `json:"name"`
	ProviderType            string            `json:"providerType"`
	ApiProtocol             *string           `json:"apiProtocol,omitempty"`
	SupportsOpenAIResponses *bool             `json:"supportsOpenAIResponses,omitempty"`
	BaseUrl                 string            `json:"baseUrl"`
	ApiKey                  string            `json:"apiKey"`
	Headers                 map[string]string `json:"headers,omitempty"`
	RateLimitPerMinute      *uint32           `json:"rateLimitPerMinute,omitempty"`
	DefaultTimeoutMs        *uint64           `json:"defaultTimeoutMs,omitempty"`
	Notes                   *string           `json:"notes,omitempty"`
	IsBuiltin               bool              `json:"isBuiltin,omitempty"`
	IsReadOnly              bool              `json:"isReadOnly,omitempty"`
	SortOrder               *int              `json:"sortOrder,omitempty"`
	MaxTokensLimit          *uint32           `json:"maxTokensLimit,omitempty"`
	WebsiteUrl              *string           `json:"websiteUrl,omitempty"`
}

type ModelProfile struct {
	ID                string   `json:"id"`
	VendorID          string   `json:"vendorId"`
	Label             string   `json:"label"`
	Model             string   `json:"model"`
	ProviderScope     *string  `json:"providerScope,omitempty"`
	ApiProtocol       *string  `json:"apiProtocol,omitempty"`
	ModelAdapter      string   `json:"modelAdapter"`
	IsMultimodal      bool     `json:"isMultimodal"`
	IsReasoning       bool     `json:"isReasoning"`
	IsEmbedding       bool     `json:"isEmbedding"`
	IsReranker        bool     `json:"isReranker"`
	IsImageGeneration bool     `json:"isImageGeneration,omitempty"`
	SupportsTools     bool     `json:"supportsTools,omitempty"`
	SupportsReasoning bool     `json:"supportsReasoning,omitempty"`
	Status            string   `json:"status,omitempty"`
	Enabled           bool     `json:"enabled"`
	MaxOutputTokens   uint32   `json:"maxOutputTokens,omitempty"`
	Temperature       float32  `json:"temperature,omitempty"`
	ReasoningEffort   *string  `json:"reasoningEffort,omitempty"`
	ThinkingEnabled   bool     `json:"thinkingEnabled,omitempty"`
	ThinkingBudget    *int     `json:"thinkingBudget,omitempty"`
	IncludeThoughts   bool     `json:"includeThoughts,omitempty"`
	EnableThinking    *bool    `json:"enableThinking,omitempty"`
	MinP              *float32 `json:"minP,omitempty"`
	TopK              *uint32  `json:"topK,omitempty"`
	GeminiApiVersion  *string  `json:"geminiApiVersion,omitempty"`
	IsBuiltin         bool     `json:"isBuiltin,omitempty"`
	RepetitionPenalty *float32 `json:"repetitionPenalty,omitempty"`
	ReasoningSplit    *bool    `json:"reasoningSplit,omitempty"`
	Effort            *string  `json:"effort,omitempty"`
	Verbosity         *string  `json:"verbosity,omitempty"`
	IsFavorite        bool     `json:"isFavorite,omitempty"`
	MaxTokensLimit    *uint32  `json:"maxTokensLimit,omitempty"`
	ContextWindow     *uint32  `json:"contextWindow,omitempty"`
}

type ModelAssignments struct {
	Model2ConfigID               *string `json:"model2_config_id"`
	ReviewAnalysisModelConfigID  *string `json:"review_analysis_model_config_id"`
	AnkiCardModelConfigID        *string `json:"anki_card_model_config_id"`
	QbankAIGradingModelConfigID  *string `json:"qbank_ai_grading_model_config_id"`
	EmbeddingModelConfigID       *string `json:"embedding_model_config_id"`
	RerankerModelConfigID        *string `json:"reranker_model_config_id"`
	ChatTitleModelConfigID       *string `json:"chat_title_model_config_id"`
	ExamSheetOCRModelConfigID    *string `json:"exam_sheet_ocr_model_config_id"`
	TranslationModelConfigID     *string `json:"translation_model_config_id"`
	VLEmbeddingModelConfigID     *string `json:"vl_embedding_model_config_id"`
	VLRerankerModelConfigID      *string `json:"vl_reranker_model_config_id"`
	MemoryDecisionModelConfigID  *string `json:"memory_decision_model_config_id"`
	VoiceInputASRModelConfigID   *string `json:"voice_input_asr_model_config_id"`
	ImageGenerationModelConfigID *string `json:"image_generation_model_config_id"`
	TranslationDisplayMode       *string `json:"translation_display_mode"`
}

type OCREngineInfo struct {
	EngineType        string `json:"engineType"`
	Name              string `json:"name"`
	Description       string `json:"description"`
	RecommendedModel  string `json:"recommendedModel"`
	SupportsGrounding bool   `json:"supportsGrounding"`
	IsFree            bool   `json:"isFree"`
}

type OCRModelConfig struct {
	ConfigID   string `json:"configId"`
	Model      string `json:"model"`
	EngineType string `json:"engineType"`
	Name       string `json:"name"`
	IsFree     bool   `json:"isFree"`
	Enabled    bool   `json:"enabled"`
	Priority   uint32 `json:"priority"`
}

type SaveOCRModelRequest struct {
	ConfigID   string  `json:"configId"`
	Model      string  `json:"model"`
	EngineType string  `json:"engineType"`
	Name       string  `json:"name"`
	IsFree     bool    `json:"isFree"`
	Enabled    *bool   `json:"enabled,omitempty"`
	Priority   *uint32 `json:"priority,omitempty"`
}

type AvailableOCRModel struct {
	ConfigID          string  `json:"configId"`
	Model             string  `json:"model"`
	EngineType        string  `json:"engineType"`
	Name              string  `json:"name"`
	IsFree            bool    `json:"isFree"`
	Description       *string `json:"description,omitempty"`
	SupportsGrounding bool    `json:"supportsGrounding"`
	Enabled           bool    `json:"enabled"`
	Priority          uint32  `json:"priority"`
}

type OCRTestRequest struct {
	ImageBase64 string  `json:"imageBase64"`
	EngineType  string  `json:"engineType"`
	ConfigID    *string `json:"configId,omitempty"`
}

type OCRTestRegion struct {
	Text  string      `json:"text"`
	BBox  *[4]float64 `json:"bbox"`
	Label *string     `json:"label"`
}

type OCRTestResponse struct {
	EngineType string          `json:"engineType"`
	EngineName string          `json:"engineName"`
	Text       string          `json:"text"`
	Regions    []OCRTestRegion `json:"regions"`
	ElapsedMS  uint64          `json:"elapsedMs"`
	Success    bool            `json:"success"`
	Error      *string         `json:"error"`
}

type UpdateOCRPriorityItem struct {
	ConfigID string `json:"configId"`
	Enabled  bool   `json:"enabled"`
}

type ocrTestConfig struct {
	engineType string
	engineName string
	model      string
	apiKey     string
	baseURL    string
	headers    map[string]string
	maxTokens  int
	isSystem   bool
}

type MCPPreheatResult struct {
	Ok    bool `json:"ok"`
	Count int  `json:"count"`
}

type MCPCacheState struct {
	TTLMs       int64   `json:"ttl_ms"`
	LastBuiltAt *string `json:"last_built_at"`
}

type MCPStatus struct {
	Available          bool          `json:"available"`
	Enabled            bool          `json:"enabled"`
	Connected          bool          `json:"connected"`
	EnabledReason      *string       `json:"enabled_reason"`
	ServerInfo         any           `json:"server_info"`
	ToolsCount         int           `json:"tools_count"`
	LastError          string        `json:"last_error"`
	NamespacePrefix    string        `json:"namespace_prefix"`
	ConflictResolution string        `json:"conflict_resolution"`
	CacheState         MCPCacheState `json:"cache_state"`
}

type MCPReloadResult struct {
	Success bool    `json:"success"`
	Message string  `json:"message,omitempty"`
	Error   *string `json:"error,omitempty"`
}

type MCPToolInfo struct {
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
	InputSchema any    `json:"input_schema,omitempty"`
}

func NewService(dataDir string) (*Service, error) {
	service := &Service{
		path:       filepath.Join(dataDir, "settings-go.json"),
		data:       make(map[string]string),
		httpClient: &http.Client{Timeout: 10 * time.Second},
	}
	if err := service.load(); err != nil {
		return nil, err
	}
	return service, nil
}

func (s *Service) SetHTTPClient(client *http.Client) {
	if client == nil {
		return
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.httpClient = client
}

func (s *Service) currentHTTPClient() *http.Client {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if s.httpClient != nil {
		return s.httpClient
	}
	return &http.Client{Timeout: 10 * time.Second}
}

func (s *Service) GetSetting(key string) (string, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	value, ok := s.data[key]
	return value, ok
}

func (s *Service) GetSettings(keys []string) map[string]string {
	s.mu.RLock()
	defer s.mu.RUnlock()

	out := make(map[string]string, len(keys))
	for _, key := range keys {
		if value, ok := s.data[key]; ok {
			out[key] = value
		}
	}
	return out
}

func (s *Service) GetSettingsByPrefix(prefix string) [][]string {
	s.mu.RLock()
	defer s.mu.RUnlock()

	keys := make([]string, 0)
	for key := range s.data {
		if strings.HasPrefix(key, prefix) {
			keys = append(keys, key)
		}
	}
	sort.Strings(keys)

	out := make([][]string, 0, len(keys))
	for _, key := range keys {
		out = append(out, []string{key, s.data[key], ""})
	}
	return out
}

func (s *Service) SaveSetting(key, value string) error {
	if key == "" {
		return errors.New("setting key cannot be empty")
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	s.data[key] = value
	return s.flushLocked()
}

func (s *Service) SaveSettings(values map[string]string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	for key, value := range values {
		if key == "" {
			return errors.New("setting key cannot be empty")
		}
		s.data[key] = value
	}
	return s.flushLocked()
}

func (s *Service) DeleteSetting(key string) error {
	if key == "" {
		return errors.New("setting key cannot be empty")
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.data, key)
	return s.flushLocked()
}

func (s *Service) GetBackupConfig() (BackupConfig, error) {
	raw, ok := s.GetSetting(backupConfigKey)
	if !ok || strings.TrimSpace(raw) == "" {
		return defaultBackupConfig(), nil
	}

	config := BackupConfig{
		AutoBackupIntervalHours: 24,
	}
	if err := json.Unmarshal([]byte(raw), &config); err != nil {
		return BackupConfig{}, fmt.Errorf("parse backup config: %w", err)
	}
	return config, nil
}

func (s *Service) SetBackupConfig(config BackupConfig) error {
	bytes, err := json.Marshal(config)
	if err != nil {
		return fmt.Errorf("serialize backup config: %w", err)
	}
	return s.SaveSetting(backupConfigKey, string(bytes))
}

func (s *Service) GetStatistics() BasicStatistics {
	return BasicStatistics{
		TotalMistakes:  0,
		TotalReviews:   0,
		TypeStats:      map[string]int{},
		TagStats:       map[string]int{},
		RecentMistakes: []any{},
	}
}

func (s *Service) GetEnhancedStatistics(imageStats ImageStatistics) EnhancedStatistics {
	return EnhancedStatistics{
		BasicStats:      s.GetStatistics(),
		ImageStats:      imageStats,
		RecentAdditions: 0,
		QualityScore:    0,
		MonthlyTrend:    []any{},
		Timestamp:       time.Now().UTC().Format(time.RFC3339),
	}
}

func (s *Service) CheckAPIConfigStatus() (APIConfigStatus, error) {
	configs, err := s.GetAPIConfigurations()
	if err != nil {
		return APIConfigStatus{}, err
	}
	assignments, err := s.GetModelAssignments()
	hasAssignments := false
	if err == nil {
		hasAssignments = assignments.Model2ConfigID != nil || assignments.ReviewAnalysisModelConfigID != nil
	}

	enabledCount := 0
	for _, config := range configs {
		if config.Enabled {
			enabledCount++
		}
	}

	return APIConfigStatus{
		ConfigCount:    len(configs),
		EnabledCount:   enabledCount,
		HasAssignments: hasAssignments,
		NeedsRecovery:  len(configs) == 0,
	}, nil
}

func (s *Service) RestoreDefaultAPIConfigs() (string, error) {
	if err := s.SaveAPIConfigurations(defaultRecoveryAPIConfigs()); err != nil {
		return "", err
	}
	if err := s.SaveModelAssignments(defaultModelAssignments()); err != nil {
		return "", err
	}
	return "✅ 默认API配置已恢复！请填入您的API密钥并启用相应配置。", nil
}

func (s *Service) TestAPIConnection(apiKey string, apiBase string, apiProtocol *string, supportsOpenAIResponses *bool, model *string, vendorID *string) (bool, error) {
	baseURL := strings.TrimRight(strings.TrimSpace(apiBase), "/")
	if baseURL == "" {
		baseURL = strings.TrimRight(strings.TrimSpace(s.apiBaseForVendor(vendorID)), "/")
	}
	if baseURL == "" {
		return false, errors.New("API base URL is required")
	}

	effectiveKey, err := s.resolveAPIKeyForTest(apiKey, vendorID)
	if err != nil {
		return false, err
	}

	modelID := strings.TrimSpace(valueOrEmpty(model))
	if modelID == "" {
		modelID = "gpt-4o-mini"
	}

	protocol := resolveTestAPIProtocol(baseURL, apiProtocol, supportsOpenAIResponses)
	endpoint := appendAPIEndpoint(baseURL, protocol)
	bodyBytes, err := json.Marshal(apiTestRequestBody(modelID, protocol))
	if err != nil {
		return false, err
	}

	req, err := http.NewRequest(http.MethodPost, endpoint, bytes.NewReader(bodyBytes))
	if err != nil {
		return false, err
	}
	req.Header.Set("Authorization", "Bearer "+strings.TrimSpace(effectiveKey))
	req.Header.Set("Content-Type", "application/json")
	for key, value := range s.apiHeadersForVendor(vendorID) {
		if strings.TrimSpace(key) == "" || strings.TrimSpace(value) == "" {
			continue
		}
		req.Header.Set(key, value)
	}

	resp, err := s.currentHTTPClient().Do(req)
	if err != nil {
		return false, fmt.Errorf("API connection test failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		return true, nil
	}

	text, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
	return false, fmt.Errorf("API connection test failed: %s - %s", resp.Status, strings.TrimSpace(string(text)))
}

func (s *Service) TestSearchEngine(engine string) (SearchEngineTestResult, error) {
	return s.testSearchEngineWithQuery(engine, "AI artificial intelligence", 1)
}

func (s *Service) TestWebSearchConnectivity(engine *string) (WebSearchConnectivityResult, error) {
	cfg := s.webSearchConfig()
	engineID := strings.TrimSpace(valueOrEmpty(engine))
	if engineID == "" {
		engineID = cfg.Engine
	}
	if engineID == "" || !isKnownSearchEngine(engineID) || !cfg.hasEngine(engineID) {
		engineID = firstConfiguredSearchEngine(cfg)
	}
	if engineID == "" {
		detail := SearchEngineTestResult{
			Ok:        false,
			Message:   "未配置可用的搜索引擎",
			TestQuery: "connectivity test",
		}
		return WebSearchConnectivityResult{Success: false, Detail: detail}, nil
	}

	result, err := s.testSearchEngineWithQuery(engineID, "connectivity test", 1)
	if err != nil {
		return WebSearchConnectivityResult{}, err
	}
	if !result.Ok {
		return WebSearchConnectivityResult{Success: false, Detail: result}, nil
	}
	return WebSearchConnectivityResult{
		Success: true,
		Usage: map[string]any{
			"elapsed_ms":    result.ResponseTime,
			"provider":      engineID,
			"results_count": valueOrZero(result.ResultsCount),
		},
	}, nil
}

func (s *Service) TestAllSearchEngines() (SearchEngineHealthReport, error) {
	cfg := s.webSearchConfig()
	results := make(map[string]SearchEngineHealthStatus, len(webSearchEngines))
	successCount := 0
	configuredCount := 0

	for _, engine := range webSearchEngines {
		if !cfg.hasEngine(engine.ID) {
			results[engine.ID] = SearchEngineHealthStatus{
				Name:      engine.Name,
				Status:    "not_configured",
				Message:   "缺少API密钥或端点配置",
				ElapsedMS: 0,
			}
			continue
		}

		configuredCount++
		result, err := s.testSearchEngineWithQuery(engine.ID, "test connectivity", 1)
		if err != nil {
			return SearchEngineHealthReport{}, err
		}
		status := SearchEngineHealthStatus{
			Name:         engine.Name,
			Status:       "failed",
			Message:      result.Message,
			ElapsedMS:    result.ResponseTime,
			ResultsCount: result.ResultsCount,
		}
		if result.Ok {
			status.Status = "success"
			status.Message = "连接成功"
			successCount++
		}
		results[engine.ID] = status
	}

	return SearchEngineHealthReport{
		Results: results,
		Summary: SearchEngineHealthSummary{
			Total:      len(webSearchEngines),
			Configured: configuredCount,
			Success:    successCount,
			Failed:     configuredCount - successCount,
		},
		Timestamp: time.Now().UTC().Format(time.RFC3339),
	}, nil
}

func (s *Service) GetAttachmentConfig() AttachmentConfig {
	values := s.GetSettings([]string{attachmentRootFolderIDKey, attachmentRootFolderTitleKey})
	return AttachmentConfig{
		AttachmentRootFolderID:    optionalString(values[attachmentRootFolderIDKey]),
		AttachmentRootFolderTitle: optionalString(values[attachmentRootFolderTitleKey]),
	}
}

func (s *Service) SetAttachmentRootFolder(folderID string) error {
	folderID = strings.TrimSpace(folderID)
	if folderID == "" {
		return errors.New("folderId is required")
	}
	return s.SaveSettings(map[string]string{
		attachmentRootFolderIDKey:    folderID,
		attachmentRootFolderTitleKey: "",
	})
}

func (s *Service) CreateAttachmentRootFolder(title string) (string, error) {
	title = strings.TrimSpace(title)
	if title == "" {
		title = "Attachments"
	}
	folderID := "folder_" + randomToken(16)
	if err := s.SaveSettings(map[string]string{
		attachmentRootFolderIDKey:    folderID,
		attachmentRootFolderTitleKey: title,
	}); err != nil {
		return "", err
	}
	return folderID, nil
}

func (s *Service) GetMemoryConfig() MemoryConfig {
	values := s.GetSettings([]string{
		memoryRootFolderIDKey,
		memoryRootFolderTitleKey,
		memoryAutoCreateSubfoldersKey,
		memoryDefaultCategoryKey,
		memoryPrivacyModeKey,
		memoryAutoExtractFrequencyKey,
	})
	rootID := optionalString(values[memoryRootFolderIDKey])
	rootTitle := optionalString(values[memoryRootFolderTitleKey])
	if rootID == nil {
		rootTitle = nil
	}
	defaultCategory := strings.TrimSpace(values[memoryDefaultCategoryKey])
	if defaultCategory == "" {
		defaultCategory = "通用"
	}
	return MemoryConfig{
		MemoryRootFolderID:    rootID,
		MemoryRootFolderTitle: rootTitle,
		AutoCreateSubfolders:  parseBoolSetting(values[memoryAutoCreateSubfoldersKey], true),
		DefaultCategory:       defaultCategory,
		PrivacyMode:           parseBoolSetting(values[memoryPrivacyModeKey], false),
		AutoExtractFrequency:  normalizeMemoryAutoExtractFrequency(values[memoryAutoExtractFrequencyKey]),
	}
}

func (s *Service) GetModelAdapterOptions() []ModelAdapterOption {
	options := make([]ModelAdapterOption, len(defaultModelAdapterOptions))
	copy(options, defaultModelAdapterOptions)
	return options
}

func (s *Service) GetCNWhitelistConfig() CNWhitelistConfigResult {
	values := s.GetSettings([]string{
		"web_search.cn_whitelist.enabled",
		"web_search.cn_whitelist.use_default",
		"web_search.cn_whitelist.custom_sites",
	})

	return CNWhitelistConfigResult{
		DefaultSites: append([]string(nil), cnTrustedSites...),
		UserConfig: CNWhitelistUserConfig{
			Enabled:        parseBoolSetting(values["web_search.cn_whitelist.enabled"], false),
			UseDefaultList: parseBoolSetting(values["web_search.cn_whitelist.use_default"], true),
			CustomSites:    parseSiteList(values["web_search.cn_whitelist.custom_sites"]),
		},
	}
}

func (s *Service) GetProviderStrategiesConfig() ProviderStrategiesConfigResult {
	strategies := defaultProviderStrategies()
	if raw, ok := s.GetSetting(webSearchProviderStrategiesKey); ok && strings.TrimSpace(raw) != "" {
		var stored ProviderStrategies
		if err := json.Unmarshal([]byte(raw), &stored); err == nil {
			strategies = normalizeProviderStrategies(stored)
		}
	}
	return ProviderStrategiesConfigResult{
		ProviderStrategies: strategies,
		ConfigKeys: map[string]string{
			"provider_strategies": webSearchProviderStrategiesKey,
		},
	}
}

func (s *Service) SaveProviderStrategiesConfig(strategies ProviderStrategies) (bool, error) {
	normalized := normalizeProviderStrategies(strategies)
	body, err := json.Marshal(normalized)
	if err != nil {
		return false, err
	}
	if err := s.SaveSetting(webSearchProviderStrategiesKey, string(body)); err != nil {
		return false, err
	}
	return true, nil
}

func (s *Service) PreheatMCPTools() MCPPreheatResult {
	raw, ok := s.GetSetting(mcpToolsListKey)
	if !ok || strings.TrimSpace(raw) == "" {
		return MCPPreheatResult{Ok: true, Count: 0}
	}

	var entries []any
	if err := json.Unmarshal([]byte(raw), &entries); err != nil {
		return MCPPreheatResult{Ok: true, Count: 0}
	}

	return MCPPreheatResult{Ok: true, Count: len(entries)}
}

func (s *Service) GetMCPStatus() MCPStatus {
	values := s.GetSettings([]string{
		"mcp.tools.namespace_prefix",
		"mcp.tools.conflict_resolution",
		"mcp.tools.cache_ttl_ms",
		"session.selected_mcp_tools",
	})

	status := MCPStatus{
		Available:          false,
		Enabled:            false,
		Connected:          false,
		EnabledReason:      nil,
		ServerInfo:         nil,
		ToolsCount:         0,
		LastError:          "backend_mcp_disabled",
		NamespacePrefix:    values["mcp.tools.namespace_prefix"],
		ConflictResolution: stringOrDefault(values["mcp.tools.conflict_resolution"], "use_namespace"),
		CacheState: MCPCacheState{
			TTLMs:       parseInt64Setting(values["mcp.tools.cache_ttl_ms"], 300_000),
			LastBuiltAt: nil,
		},
	}

	if selected, ok := values["session.selected_mcp_tools"]; ok {
		status.Enabled = strings.TrimSpace(selected) != ""
		if !status.Enabled {
			status.EnabledReason = stringPtr("会话未选择MCP工具")
		}
	}

	return status
}

func (s *Service) ReloadMCPClient() MCPReloadResult {
	return MCPReloadResult{
		Success: true,
		Message: "Backend MCP disabled; frontend SDK in use",
	}
}

func (s *Service) GetMCPTools() []MCPToolInfo {
	return []MCPToolInfo{}
}

func (s *Service) GetOCREngines() []OCREngineInfo {
	return ocrEngineInfoList()
}

func (s *Service) GetOCREngineType() string {
	value, ok := s.GetSetting(ocrEngineTypeKey)
	if !ok || strings.TrimSpace(value) == "" {
		return "paddle_ocr_vl"
	}
	return normalizeOCREngineType(value)
}

func (s *Service) GetOCRThinkingEnabled() bool {
	value, ok := s.GetSetting(ocrEnableThinkingKey)
	if !ok {
		return false
	}
	return parseBoolSetting(value, false)
}

func (s *Service) SetOCRThinkingEnabled(enabled bool) (bool, error) {
	value := "false"
	if enabled {
		value = "true"
	}
	if err := s.SaveSetting(ocrEnableThinkingKey, value); err != nil {
		return false, err
	}
	return true, nil
}

func (s *Service) GetAvailableOCRModels() ([]AvailableOCRModel, error) {
	models, err := s.loadOCRModels()
	if err != nil {
		return nil, err
	}
	changed := migrateOCRModels(models)

	configs, err := s.GetAPIConfigurations()
	if err == nil {
		validIDs := make(map[string]bool, len(configs))
		for _, config := range configs {
			validIDs[config.ID] = true
		}
		filtered := models[:0]
		for _, model := range models {
			if model.ConfigID == systemOCRConfigID || validIDs[model.ConfigID] {
				filtered = append(filtered, model)
			}
		}
		if len(filtered) != len(models) {
			models = filtered
			changed = true
		}
	}

	if systemOCRSupported() && !hasOCRModel(models, systemOCRConfigID) {
		models = append(models, OCRModelConfig{
			ConfigID:   systemOCRConfigID,
			Model:      "system",
			EngineType: "system_ocr",
			Name:       "系统 OCR",
			IsFree:     true,
			Enabled:    true,
			Priority:   uint32(len(models)),
		})
		changed = true
	}

	if normalizeOCRModelPriorities(models) {
		changed = true
	}
	if changed {
		if err := s.saveOCRModels(models); err != nil {
			return nil, err
		}
	}

	infoByType := ocrEngineInfoByType()
	result := make([]AvailableOCRModel, 0, len(models))
	for _, model := range models {
		info, ok := infoByType[model.EngineType]
		var description *string
		supportsGrounding := false
		if ok {
			description = &info.Description
			supportsGrounding = info.SupportsGrounding
		}
		result = append(result, AvailableOCRModel{
			ConfigID:          model.ConfigID,
			Model:             model.Model,
			EngineType:        model.EngineType,
			Name:              model.Name,
			IsFree:            model.IsFree,
			Description:       description,
			SupportsGrounding: supportsGrounding,
			Enabled:           model.Enabled,
			Priority:          model.Priority,
		})
	}
	return result, nil
}

func (s *Service) TestOCREngine(request OCRTestRequest) (OCRTestResponse, error) {
	startedAt := time.Now()
	engineType := normalizeOCREngineType(request.EngineType)
	engineName := ocrEngineDisplayName(engineType)
	response := OCRTestResponse{
		EngineType: engineType,
		EngineName: engineName,
		Text:       "",
		Regions:    []OCRTestRegion{},
		ElapsedMS:  0,
		Success:    false,
		Error:      nil,
	}

	imageDataURL, err := normalizeOCRImageDataURL(request.ImageBase64)
	if err != nil {
		return OCRTestResponse{}, err
	}

	config, err := s.resolveOCRTestConfig(request)
	if config.engineType != "" {
		response.EngineType = config.engineType
	}
	if config.engineName != "" {
		response.EngineName = config.engineName
	}
	if err != nil {
		response.ElapsedMS = uint64(time.Since(startedAt).Milliseconds())
		response.Error = stringPtr(err.Error())
		return response, nil
	}
	if config.isSystem {
		response.ElapsedMS = uint64(time.Since(startedAt).Milliseconds())
		response.Error = stringPtr("系统 OCR Go/Wails 诊断尚未接入；请选择已配置的多模态 OCR API")
		return response, nil
	}

	text, regions, err := s.callOCRTestProvider(config, imageDataURL)
	response.ElapsedMS = uint64(time.Since(startedAt).Milliseconds())
	if err != nil {
		response.Error = stringPtr(err.Error())
		return response, nil
	}
	response.Text = text
	response.Regions = regions
	response.Success = true
	return response, nil
}

func (s *Service) SaveAvailableOCRModels(models []SaveOCRModelRequest) (bool, error) {
	configs := make([]OCRModelConfig, 0, len(models))
	for index, model := range models {
		enabled := true
		if model.Enabled != nil {
			enabled = *model.Enabled
		}
		priority := uint32(index)
		if model.Priority != nil {
			priority = *model.Priority
		}
		configs = append(configs, OCRModelConfig{
			ConfigID:   strings.TrimSpace(model.ConfigID),
			Model:      strings.TrimSpace(model.Model),
			EngineType: normalizeOCREngineType(model.EngineType),
			Name:       strings.TrimSpace(model.Name),
			IsFree:     model.IsFree,
			Enabled:    enabled,
			Priority:   priority,
		})
	}
	normalizeOCRModelPriorities(configs)
	if err := s.saveOCRModels(configs); err != nil {
		return false, err
	}
	return true, nil
}

func (s *Service) UpdateOCREnginePriority(engineList []UpdateOCRPriorityItem) (bool, error) {
	models, err := s.loadOCRModels()
	if err != nil {
		return false, err
	}
	byID := make(map[string]int, len(models))
	for index, model := range models {
		byID[model.ConfigID] = index
	}
	for index, item := range engineList {
		if modelIndex, ok := byID[item.ConfigID]; ok {
			models[modelIndex].Priority = uint32(index)
			models[modelIndex].Enabled = item.Enabled
		}
	}
	normalizeOCRModelPriorities(models)
	if err := s.saveOCRModels(models); err != nil {
		return false, err
	}
	return true, nil
}

func (s *Service) AddOCREngine(configID, model, name string, engineType *string) (bool, error) {
	configID = strings.TrimSpace(configID)
	model = strings.TrimSpace(model)
	name = strings.TrimSpace(name)
	if configID == "" {
		return false, errors.New("configId is required")
	}
	if configID == systemOCRConfigID {
		return false, errors.New("system OCR is managed automatically")
	}
	models, err := s.loadOCRModels()
	if err != nil {
		return false, err
	}
	if hasOCRModel(models, configID) {
		return false, errors.New("OCR engine already exists")
	}
	effectiveEngine := inferOCREngineFromModel(model)
	if engineType != nil && strings.TrimSpace(*engineType) != "" {
		effectiveEngine = normalizeOCREngineType(*engineType)
	}
	infoByType := ocrEngineInfoByType()
	info, ok := infoByType[effectiveEngine]
	models = append(models, OCRModelConfig{
		ConfigID:   configID,
		Model:      model,
		EngineType: effectiveEngine,
		Name:       firstNonEmpty(name, model, configID),
		IsFree:     ok && info.IsFree,
		Enabled:    true,
		Priority:   uint32(len(models)),
	})
	normalizeOCRModelPriorities(models)
	if err := s.saveOCRModels(models); err != nil {
		return false, err
	}
	return true, nil
}

func (s *Service) RemoveOCREngine(configID string) (bool, error) {
	configID = strings.TrimSpace(configID)
	if configID == "" {
		return false, errors.New("configId is required")
	}
	if configID == systemOCRConfigID {
		return false, errors.New("system OCR is managed automatically")
	}
	models, err := s.loadOCRModels()
	if err != nil {
		return false, err
	}
	filtered := models[:0]
	for _, model := range models {
		if model.ConfigID != configID {
			filtered = append(filtered, model)
		}
	}
	if len(filtered) == len(models) {
		return false, errors.New("OCR engine not found")
	}
	normalizeOCRModelPriorities(filtered)
	if err := s.saveOCRModels(filtered); err != nil {
		return false, err
	}
	return true, nil
}

func (s *Service) resolveOCRTestConfig(request OCRTestRequest) (ocrTestConfig, error) {
	engineType := normalizeOCREngineType(request.EngineType)
	result := ocrTestConfig{
		engineType: engineType,
		engineName: ocrEngineDisplayName(engineType),
		headers:    map[string]string{},
	}

	configID := strings.TrimSpace(valueOrEmpty(request.ConfigID))
	if configID == systemOCRConfigID || engineType == "system_ocr" {
		result.engineType = "system_ocr"
		result.engineName = ocrEngineDisplayName("system_ocr")
		result.isSystem = true
		return result, nil
	}

	models, err := s.loadOCRModels()
	if err != nil {
		return result, err
	}
	if migrateOCRModels(models) {
		normalizeOCRModelPriorities(models)
	}
	configs, err := s.GetAPIConfigurations()
	if err != nil {
		return result, err
	}

	var ocrModel *OCRModelConfig
	if configID != "" {
		ocrModel = findOCRModelConfig(models, configID)
	} else {
		ocrModel = selectOCRModelForEngine(models, engineType)
		if ocrModel == nil {
			assignments, assignmentErr := s.GetModelAssignments()
			if assignmentErr == nil && assignments.ExamSheetOCRModelConfigID != nil {
				ocrModel = findOCRModelConfig(models, strings.TrimSpace(*assignments.ExamSheetOCRModelConfigID))
			}
		}
		if ocrModel != nil {
			configID = ocrModel.ConfigID
		}
	}

	if ocrModel != nil {
		result.engineType = normalizeOCREngineType(ocrModel.EngineType)
		result.engineName = firstNonEmpty(ocrModel.Name, ocrEngineDisplayName(result.engineType))
		if result.engineType == "system_ocr" || ocrModel.ConfigID == systemOCRConfigID {
			result.isSystem = true
			return result, nil
		}
	}
	if configID == "" {
		return result, fmt.Errorf("未找到可用于 %s 的 OCR 模型配置", result.engineName)
	}

	apiConfig, ok := findAPIConfig(configs, configID)
	if !ok {
		return result, fmt.Errorf("未找到 OCR API 配置: %s", configID)
	}
	if apiConfig.IsEmbedding || apiConfig.IsReranker || apiConfig.IsImageGeneration || apiConfig.IsAudioTranscription {
		return result, fmt.Errorf("OCR API 配置 %s 不是多模态识别模型", configID)
	}

	modelID := strings.TrimSpace(apiConfig.Model)
	if modelID == "" && ocrModel != nil {
		modelID = strings.TrimSpace(ocrModel.Model)
	}
	if modelID == "" {
		return result, fmt.Errorf("OCR API 配置 %s 缺少模型名称", configID)
	}

	baseURL := strings.TrimRight(strings.TrimSpace(apiConfig.BaseUrl), "/")
	if baseURL == "" {
		baseURL = strings.TrimRight(strings.TrimSpace(s.apiBaseForVendor(apiConfig.VendorID)), "/")
	}
	if baseURL == "" {
		return result, fmt.Errorf("OCR API 配置 %s 缺少 Base URL", configID)
	}

	apiKey, err := s.resolveAPIKeyForTest(apiConfig.ApiKey, apiConfig.VendorID)
	if err != nil {
		return result, err
	}

	headers := s.apiHeadersForVendor(apiConfig.VendorID)
	if headers == nil {
		headers = map[string]string{}
	}
	for key, value := range apiConfig.Headers {
		headers[key] = value
	}

	result.model = modelID
	result.apiKey = apiKey
	result.baseURL = baseURL
	result.headers = headers
	result.maxTokens = ocrTestMaxTokens(apiConfig)
	return result, nil
}

func (s *Service) callOCRTestProvider(config ocrTestConfig, imageDataURL string) (string, []OCRTestRegion, error) {
	bodyBytes, err := json.Marshal(ocrTestRequestBody(config, imageDataURL))
	if err != nil {
		return "", nil, err
	}
	req, err := http.NewRequest(http.MethodPost, appendChatCompletionsEndpoint(config.baseURL), bytes.NewReader(bodyBytes))
	if err != nil {
		return "", nil, err
	}
	req.Header.Set("Authorization", "Bearer "+strings.TrimSpace(config.apiKey))
	req.Header.Set("Content-Type", "application/json")
	for key, value := range config.headers {
		if strings.TrimSpace(key) == "" || strings.TrimSpace(value) == "" {
			continue
		}
		req.Header.Set(key, value)
	}

	resp, err := s.currentHTTPClient().Do(req)
	if err != nil {
		return "", nil, fmt.Errorf("OCR API request failed: %w", err)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(io.LimitReader(resp.Body, 1024*1024))
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return "", nil, fmt.Errorf("OCR API request failed: %s - %s", resp.Status, strings.TrimSpace(string(body)))
	}

	content, err := chatCompletionTextContent(body)
	if err != nil {
		return "", nil, err
	}
	regions := parseOCRTestRegions(config.engineType, content)
	if len(regions) == 0 {
		regions = []OCRTestRegion{{
			Text:  strings.TrimSpace(content),
			BBox:  nil,
			Label: stringPtr("text"),
		}}
	}
	textParts := make([]string, 0, len(regions))
	for _, region := range regions {
		if trimmed := strings.TrimSpace(region.Text); trimmed != "" {
			textParts = append(textParts, trimmed)
		}
	}
	text := strings.Join(textParts, "\n")
	if strings.TrimSpace(text) == "" {
		text = strings.TrimSpace(content)
	}
	return text, regions, nil
}

func (s *Service) GetAPIConfigurations() ([]ApiConfig, error) {
	var configs []ApiConfig
	if err := s.getJSON(apiConfigurationsKey, &configs); err != nil {
		return nil, err
	}
	return normalizeAPIConfigs(configs), nil
}

func (s *Service) SaveAPIConfigurations(configs []ApiConfig) error {
	return s.saveJSON(apiConfigurationsKey, normalizeAPIConfigs(configs))
}

func (s *Service) GetVendorConfigs() ([]VendorConfig, error) {
	var configs []VendorConfig
	if err := s.getJSON(vendorConfigsKey, &configs); err != nil {
		return nil, err
	}
	return normalizeVendorConfigs(configs), nil
}

func (s *Service) SaveVendorConfigs(configs []VendorConfig) error {
	return s.saveJSON(vendorConfigsKey, normalizeVendorConfigs(configs))
}

func (s *Service) GetModelProfiles() ([]ModelProfile, error) {
	var profiles []ModelProfile
	if err := s.getJSON(modelProfilesKey, &profiles); err != nil {
		return nil, err
	}
	return normalizeModelProfiles(profiles), nil
}

func (s *Service) SaveModelProfiles(profiles []ModelProfile) error {
	return s.saveJSON(modelProfilesKey, normalizeModelProfiles(profiles))
}

func (s *Service) GetModelAssignments() (ModelAssignments, error) {
	assignments := defaultModelAssignments()
	if err := s.getJSON(modelAssignmentsKey, &assignments); err != nil {
		return ModelAssignments{}, err
	}
	return normalizeModelAssignments(assignments), nil
}

func (s *Service) SaveModelAssignments(assignments ModelAssignments) error {
	return s.saveJSON(modelAssignmentsKey, normalizeModelAssignments(assignments))
}

func (s *Service) getJSON(key string, out any) error {
	s.mu.RLock()
	raw, ok := s.data[key]
	s.mu.RUnlock()
	if !ok || strings.TrimSpace(raw) == "" {
		return nil
	}
	if err := json.Unmarshal([]byte(raw), out); err != nil {
		return fmt.Errorf("decode %s: %w", key, err)
	}
	return nil
}

func (s *Service) saveJSON(key string, value any) error {
	bytes, err := json.Marshal(value)
	if err != nil {
		return fmt.Errorf("encode %s: %w", key, err)
	}
	return s.SaveSetting(key, string(bytes))
}

func (s *Service) loadOCRModels() ([]OCRModelConfig, error) {
	var models []OCRModelConfig
	if err := s.getJSON(ocrAvailableModelsKey, &models); err != nil {
		return nil, err
	}
	for index := range models {
		models[index].ConfigID = strings.TrimSpace(models[index].ConfigID)
		models[index].Model = strings.TrimSpace(models[index].Model)
		models[index].EngineType = normalizeOCREngineType(models[index].EngineType)
		models[index].Name = strings.TrimSpace(models[index].Name)
		if models[index].Name == "" {
			models[index].Name = firstNonEmpty(models[index].Model, models[index].ConfigID)
		}
	}
	return models, nil
}

func (s *Service) saveOCRModels(models []OCRModelConfig) error {
	return s.saveJSON(ocrAvailableModelsKey, models)
}

func ocrEngineInfoList() []OCREngineInfo {
	list := []OCREngineInfo{
		{
			EngineType:        "deepseek_ocr",
			Name:              "DeepSeek-OCR",
			Description:       "专业 OCR 模型，支持 Grounding 坐标输出，适合题目集识别",
			RecommendedModel:  "deepseek-ai/DeepSeek-OCR",
			SupportsGrounding: true,
			IsFree:            false,
		},
		{
			EngineType:        "paddle_ocr_vl",
			Name:              "PaddleOCR-VL-1.5",
			Description:       "百度开源 OCR 视觉语言模型 1.5 版，支持 109 种语言，精度 94.5%，完全免费",
			RecommendedModel:  "PaddlePaddle/PaddleOCR-VL-1.5",
			SupportsGrounding: true,
			IsFree:            true,
		},
		{
			EngineType:        "paddle_ocr_vl_v1",
			Name:              "PaddleOCR-VL",
			Description:       "百度开源 OCR 视觉语言模型旧版，支持坐标输出，完全免费，作为 1.5 版的备用",
			RecommendedModel:  "PaddlePaddle/PaddleOCR-VL",
			SupportsGrounding: true,
			IsFree:            true,
		},
		{
			EngineType:        "glm4v_ocr",
			Name:              "GLM-4.6V",
			Description:       "智谱 106B MoE 多模态模型，支持 bbox_2d 坐标输出，题目集导入优先引擎",
			RecommendedModel:  "zai-org/GLM-4.6V",
			SupportsGrounding: true,
			IsFree:            false,
		},
		{
			EngineType:        "generic_vlm",
			Name:              "通用多模态模型",
			Description:       "使用通用 VLM 进行 OCR，适合简单文档识别",
			RecommendedModel:  "Qwen/Qwen2.5-VL-7B-Instruct",
			SupportsGrounding: false,
			IsFree:            false,
		},
	}
	if systemOCRSupported() {
		list = append(list, OCREngineInfo{
			EngineType:        "system_ocr",
			Name:              "系统 OCR",
			Description:       "调用操作系统内置 OCR 引擎，免费离线，无需 API Key",
			RecommendedModel:  "system",
			SupportsGrounding: false,
			IsFree:            true,
		})
	}
	return list
}

func ocrEngineInfoByType() map[string]OCREngineInfo {
	out := make(map[string]OCREngineInfo)
	for _, info := range ocrEngineInfoList() {
		out[info.EngineType] = info
	}
	return out
}

func normalizeOCREngineType(value string) string {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "deepseek_ocr", "deepseek-ocr", "deepseek":
		return "deepseek_ocr"
	case "paddle_ocr_vl", "paddleocr-vl", "paddleocr_vl", "paddle":
		return "paddle_ocr_vl"
	case "paddle_ocr_vl_v1", "paddleocr-vl-v1", "paddleocr_vl_v1":
		return "paddle_ocr_vl_v1"
	case "glm4v_ocr", "glm-4.6v", "glm4v", "glm-4v":
		return "glm4v_ocr"
	case "generic_vlm", "generic", "vlm":
		return "generic_vlm"
	case "system_ocr", "system", "native":
		return "system_ocr"
	default:
		return "paddle_ocr_vl"
	}
}

func inferOCREngineFromModel(model string) string {
	modelLower := strings.ToLower(strings.TrimSpace(model))
	switch {
	case modelLower == "system":
		return "system_ocr"
	case glmVisionRE.MatchString(model):
		return "glm4v_ocr"
	case strings.Contains(modelLower, "deepseek") && strings.Contains(modelLower, "ocr"):
		return "deepseek_ocr"
	case strings.Contains(modelLower, "paddleocr-vl-1") || strings.Contains(modelLower, "paddleocr_vl_1"):
		return "paddle_ocr_vl"
	case strings.Contains(modelLower, "paddle") || strings.Contains(modelLower, "paddleocr"):
		return "paddle_ocr_vl_v1"
	default:
		return "generic_vlm"
	}
}

func migrateOCRModels(models []OCRModelConfig) bool {
	changed := false
	for index := range models {
		if models[index].Model == "PaddlePaddle/PaddleOCR-VL" && models[index].EngineType != "paddle_ocr_vl_v1" {
			models[index].Model = "PaddlePaddle/PaddleOCR-VL-1.5"
			if strings.Contains(models[index].Name, "PaddleOCR-VL") && !strings.Contains(models[index].Name, "1.5") {
				models[index].Name = strings.ReplaceAll(models[index].Name, "PaddleOCR-VL", "PaddleOCR-VL-1.5")
			}
			changed = true
		}
		if models[index].EngineType == "glm4v_ocr" && strings.Contains(strings.ToLower(models[index].Model), "glm-4.1v") {
			models[index].Model = "zai-org/GLM-4.6V"
			models[index].Name = strings.ReplaceAll(strings.ReplaceAll(models[index].Name, "4.1V", "4.6V"), "4.1v", "4.6V")
			changed = true
		}
	}
	return changed
}

func hasOCRModel(models []OCRModelConfig, configID string) bool {
	for _, model := range models {
		if model.ConfigID == configID {
			return true
		}
	}
	return false
}

func normalizeOCRModelPriorities(models []OCRModelConfig) bool {
	before := make([]OCRModelConfig, len(models))
	copy(before, models)

	sort.SliceStable(models, func(i, j int) bool {
		if models[i].Priority == models[j].Priority {
			return models[i].ConfigID < models[j].ConfigID
		}
		return models[i].Priority < models[j].Priority
	})

	changed := false
	for index := range models {
		models[index].ConfigID = strings.TrimSpace(models[index].ConfigID)
		models[index].Model = strings.TrimSpace(models[index].Model)
		models[index].EngineType = normalizeOCREngineType(models[index].EngineType)
		models[index].Name = strings.TrimSpace(models[index].Name)
		if models[index].ConfigID == "" {
			models[index].ConfigID = "ocr_" + randomToken(12)
		}
		if models[index].Name == "" {
			models[index].Name = firstNonEmpty(models[index].Model, models[index].ConfigID)
		}
		models[index].Priority = uint32(index)
		if before[index] != models[index] {
			changed = true
		}
	}
	return changed
}

func systemOCRSupported() bool {
	return runtime.GOOS == "darwin" || runtime.GOOS == "windows" || runtime.GOOS == "ios"
}

func normalizeOCRImageDataURL(value string) (string, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return "", errors.New("图片解析失败: imageBase64 is required")
	}

	if strings.HasPrefix(strings.ToLower(value), "data:") {
		commaIndex := strings.Index(value, ",")
		if commaIndex < 0 {
			return "", errors.New("图片解析失败: Invalid data URL format")
		}
		encoded := value[commaIndex+1:]
		if _, err := base64.StdEncoding.DecodeString(encoded); err != nil {
			return "", fmt.Errorf("图片解析失败: Base64 decode error: %w", err)
		}
		return value, nil
	}

	if _, err := base64.StdEncoding.DecodeString(value); err != nil {
		return "", fmt.Errorf("图片解析失败: Base64 decode error: %w", err)
	}
	return "data:image/jpeg;base64," + value, nil
}

func findOCRModelConfig(models []OCRModelConfig, configID string) *OCRModelConfig {
	configID = strings.TrimSpace(configID)
	for index := range models {
		if strings.TrimSpace(models[index].ConfigID) == configID {
			return &models[index]
		}
	}
	return nil
}

func selectOCRModelForEngine(models []OCRModelConfig, engineType string) *OCRModelConfig {
	engineType = normalizeOCREngineType(engineType)
	candidates := make([]OCRModelConfig, 0, len(models))
	for _, model := range models {
		if model.Enabled && normalizeOCREngineType(model.EngineType) == engineType {
			candidates = append(candidates, model)
		}
	}
	if len(candidates) == 0 {
		return nil
	}
	normalizeOCRModelPriorities(candidates)
	return &candidates[0]
}

func findAPIConfig(configs []ApiConfig, configID string) (ApiConfig, bool) {
	configID = strings.TrimSpace(configID)
	for _, config := range configs {
		if strings.TrimSpace(config.ID) == configID {
			return config, true
		}
	}
	return ApiConfig{}, false
}

func ocrEngineDisplayName(engineType string) string {
	info, ok := ocrEngineInfoByType()[normalizeOCREngineType(engineType)]
	if ok {
		return info.Name
	}
	return ocrEngineInfoByType()["paddle_ocr_vl"].Name
}

func ocrTestMaxTokens(config ApiConfig) int {
	value := int(config.MaxOutputTokens)
	if config.MaxTokensLimit != nil && *config.MaxTokensLimit > 0 {
		limit := int(*config.MaxTokensLimit)
		if value == 0 || limit < value {
			value = limit
		}
	}
	if value == 0 {
		value = 4096
	}
	return clampInt(value, 2048, 8000)
}

func ocrTestRequestBody(config ocrTestConfig, imageDataURL string) map[string]any {
	body := map[string]any{
		"model": strings.TrimSpace(config.model),
		"messages": []map[string]any{
			{
				"role": "user",
				"content": []map[string]any{
					{
						"type": "image_url",
						"image_url": map[string]any{
							"url":    imageDataURL,
							"detail": "high",
						},
					},
					{
						"type": "text",
						"text": ocrPromptTemplate(config.engineType),
					},
				},
			},
		},
		"temperature": 0.0,
		"max_tokens":  config.maxTokens,
		"stream":      false,
	}
	if config.engineType == "paddle_ocr_vl" || config.engineType == "paddle_ocr_vl_v1" {
		body["repetition_penalty"] = 1.1
	}
	return body
}

func ocrPromptTemplate(engineType string) string {
	switch normalizeOCREngineType(engineType) {
	case "deepseek_ocr":
		return "<|grounding|>Convert the document to markdown."
	case "paddle_ocr_vl", "paddle_ocr_vl_v1":
		return "OCR:"
	case "glm4v_ocr":
		return "请识别图片中的文字并转换为 Markdown，尽量保留原始结构；如能定位区域，请使用 bbox_2d [x1,y1,x2,y2] 表示坐标。"
	default:
		return "Convert the document to markdown. Preserve the structure and formatting as much as possible."
	}
}

func chatCompletionTextContent(body []byte) (string, error) {
	var payload struct {
		Choices []struct {
			Message struct {
				Content any `json:"content"`
			} `json:"message"`
		} `json:"choices"`
	}
	if err := json.Unmarshal(body, &payload); err != nil {
		return "", fmt.Errorf("OCR API response parse failed: %w", err)
	}
	if len(payload.Choices) == 0 {
		return "", errors.New("OCR API response missing choices")
	}
	content := chatMessageContentString(payload.Choices[0].Message.Content)
	if strings.TrimSpace(content) == "" {
		return "", errors.New("OCR API response missing message content")
	}
	return content, nil
}

func chatMessageContentString(value any) string {
	switch typed := value.(type) {
	case string:
		return typed
	case []any:
		parts := make([]string, 0, len(typed))
		for _, item := range typed {
			switch itemValue := item.(type) {
			case string:
				parts = append(parts, itemValue)
			case map[string]any:
				if text, ok := itemValue["text"].(string); ok {
					parts = append(parts, text)
				}
			}
		}
		return strings.Join(parts, "\n")
	default:
		return ""
	}
}

func parseOCRTestRegions(engineType string, content string) []OCRTestRegion {
	engineType = normalizeOCREngineType(engineType)
	if engineType == "paddle_ocr_vl" || engineType == "paddle_ocr_vl_v1" {
		if regions := parsePaddleOCRRegions(content); len(regions) > 0 {
			return regions
		}
	}
	if engineType == "deepseek_ocr" || strings.Contains(content, "<|ref|>") {
		if regions := parseDeepSeekOCRRegions(content); len(regions) > 0 {
			return regions
		}
	}
	trimmed := strings.TrimSpace(content)
	if trimmed == "" {
		return []OCRTestRegion{}
	}
	return []OCRTestRegion{{
		Text:  trimmed,
		BBox:  nil,
		Label: stringPtr("text"),
	}}
}

func parsePaddleOCRRegions(content string) []OCRTestRegion {
	jsonText := extractOCRJSONText(content)
	if jsonText == "" {
		return []OCRTestRegion{}
	}

	type paddleBlock struct {
		Type      string    `json:"type"`
		Label     string    `json:"label"`
		Content   string    `json:"content"`
		Text      string    `json:"text"`
		BBox      []float64 `json:"bbox"`
		BlockBBox []float64 `json:"block_bbox"`
	}
	type paddleDocument struct {
		Blocks []paddleBlock `json:"blocks"`
	}

	blocks := []paddleBlock{}
	var document paddleDocument
	if err := json.Unmarshal([]byte(jsonText), &document); err == nil && len(document.Blocks) > 0 {
		blocks = document.Blocks
	} else {
		_ = json.Unmarshal([]byte(jsonText), &blocks)
	}
	if len(blocks) == 0 {
		return []OCRTestRegion{}
	}

	regions := make([]OCRTestRegion, 0, len(blocks))
	for _, block := range blocks {
		text := firstNonEmpty(block.Content, block.Text)
		if strings.TrimSpace(text) == "" {
			continue
		}
		label := firstNonEmpty(block.Label, block.Type, "text")
		bbox := normalizeOCRBBox(firstNonEmptyBBox(block.BBox, block.BlockBBox))
		regions = append(regions, OCRTestRegion{
			Text:  strings.TrimSpace(text),
			BBox:  bbox,
			Label: stringPtr(label),
		})
	}
	return regions
}

func extractOCRJSONText(content string) string {
	trimmed := strings.TrimSpace(content)
	if strings.HasPrefix(trimmed, "```") {
		firstNewline := strings.Index(trimmed, "\n")
		lastFence := strings.LastIndex(trimmed, "```")
		if firstNewline >= 0 && lastFence > firstNewline {
			trimmed = strings.TrimSpace(trimmed[firstNewline+1 : lastFence])
		}
	}
	if strings.HasPrefix(trimmed, "{") || strings.HasPrefix(trimmed, "[") {
		return trimmed
	}
	objectStart := strings.Index(trimmed, "{")
	objectEnd := strings.LastIndex(trimmed, "}")
	if objectStart >= 0 && objectEnd > objectStart {
		return trimmed[objectStart : objectEnd+1]
	}
	arrayStart := strings.Index(trimmed, "[")
	arrayEnd := strings.LastIndex(trimmed, "]")
	if arrayStart >= 0 && arrayEnd > arrayStart {
		return trimmed[arrayStart : arrayEnd+1]
	}
	return ""
}

func parseDeepSeekOCRRegions(content string) []OCRTestRegion {
	pattern := regexp.MustCompile(`(?s)<\|ref\|>(.*?)<\|/ref\|>\s*<\|det\|>\s*\[?\[\s*([0-9.]+)\s*,\s*([0-9.]+)\s*,\s*([0-9.]+)\s*,\s*([0-9.]+)\s*\]?\]\s*<\|/det\|>`)
	matches := pattern.FindAllStringSubmatch(content, -1)
	regions := make([]OCRTestRegion, 0, len(matches))
	for _, match := range matches {
		if len(match) != 6 {
			continue
		}
		values := make([]float64, 0, 4)
		for _, raw := range match[2:] {
			value, err := strconv.ParseFloat(raw, 64)
			if err != nil {
				values = nil
				break
			}
			values = append(values, value)
		}
		if len(values) != 4 {
			continue
		}
		label := strings.TrimSpace(match[1])
		regions = append(regions, OCRTestRegion{
			Text:  label,
			BBox:  normalizeOCRBBox(values),
			Label: stringPtr(firstNonEmpty(label, "text")),
		})
	}
	return regions
}

func firstNonEmptyBBox(values ...[]float64) []float64 {
	for _, value := range values {
		if len(value) >= 4 {
			return value
		}
	}
	return nil
}

func normalizeOCRBBox(values []float64) *[4]float64 {
	if len(values) < 4 {
		return nil
	}
	out := [4]float64{values[0], values[1], values[2], values[3]}
	if out[0] > 1 || out[1] > 1 || out[2] > 1 || out[3] > 1 {
		x1 := out[0] / 999.0
		y1 := out[1] / 999.0
		width := (out[2] - out[0]) / 999.0
		height := (out[3] - out[1]) / 999.0
		out = [4]float64{x1, y1, width, height}
	}
	return &out
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
	return json.Unmarshal(bytes, &s.data)
}

func (s *Service) flushLocked() error {
	return storage.WriteJSONAtomic(s.path, s.data)
}

func optionalString(value string) *string {
	value = strings.TrimSpace(value)
	if value == "" {
		return nil
	}
	return &value
}

func stringPtr(value string) *string {
	return &value
}

func boolSettingPtr(value bool) *bool {
	return &value
}

func uint32Ptr(value uint32) *uint32 {
	return &value
}

func uint64Ptr(value uint64) *uint64 {
	return &value
}

func float64Ptr(value float64) *float64 {
	return &value
}

func defaultProviderSpecialHandling() ProviderSpecialHandling {
	return ProviderSpecialHandling{
		Handle429RetryAfter:             true,
		ExponentialBackoffOn5xx:         true,
		CircuitBreakerEnabled:           false,
		CircuitBreakerFailureThreshold:  uint32Ptr(5),
		CircuitBreakerRecoveryTimeoutMS: uint64Ptr(30000),
	}
}

func defaultProviderStrategy() ProviderStrategy {
	handling := defaultProviderSpecialHandling()
	return ProviderStrategy{
		TimeoutMS:             uint64Ptr(8000),
		MaxRetries:            uint32Ptr(2),
		InitialRetryDelayMS:   uint64Ptr(200),
		MaxRetryDelayMS:       uint64Ptr(5000),
		BackoffMultiplier:     float64Ptr(2.0),
		MaxConcurrentRequests: uint32Ptr(5),
		RateLimitPerMinute:    uint32Ptr(60),
		CacheEnabled:          boolSettingPtr(true),
		CacheTTLSeconds:       uint64Ptr(300),
		CacheMaxEntries:       uint64Ptr(128),
		SpecialHandling:       &handling,
	}
}

func providerStrategyWith(overrides func(*ProviderStrategy)) *ProviderStrategy {
	strategy := defaultProviderStrategy()
	if overrides != nil {
		overrides(&strategy)
	}
	return &strategy
}

func defaultProviderStrategies() ProviderStrategies {
	return ProviderStrategies{
		Default: defaultProviderStrategy(),
		GoogleCSE: providerStrategyWith(func(strategy *ProviderStrategy) {
			strategy.TimeoutMS = uint64Ptr(6000)
			strategy.MaxRetries = uint32Ptr(2)
			strategy.RateLimitPerMinute = uint32Ptr(100)
		}),
		SerpAPI: providerStrategyWith(func(strategy *ProviderStrategy) {
			strategy.TimeoutMS = uint64Ptr(15000)
			strategy.MaxRetries = uint32Ptr(2)
			strategy.RateLimitPerMinute = uint32Ptr(20)
			strategy.SpecialHandling = &ProviderSpecialHandling{
				Handle429RetryAfter:             true,
				ExponentialBackoffOn5xx:         true,
				CircuitBreakerEnabled:           true,
				CircuitBreakerFailureThreshold:  uint32Ptr(3),
				CircuitBreakerRecoveryTimeoutMS: uint64Ptr(60000),
			}
		}),
		Tavily: providerStrategyWith(func(strategy *ProviderStrategy) {
			strategy.TimeoutMS = uint64Ptr(8000)
			strategy.MaxRetries = uint32Ptr(3)
			strategy.RateLimitPerMinute = uint32Ptr(50)
		}),
		Brave: providerStrategyWith(func(strategy *ProviderStrategy) {
			strategy.TimeoutMS = uint64Ptr(12000)
			strategy.MaxRetries = uint32Ptr(2)
			strategy.RateLimitPerMinute = uint32Ptr(30)
		}),
		Searxng: providerStrategyWith(func(strategy *ProviderStrategy) {
			strategy.TimeoutMS = uint64Ptr(20000)
			strategy.MaxRetries = uint32Ptr(1)
			strategy.RateLimitPerMinute = uint32Ptr(30)
			strategy.SpecialHandling = &ProviderSpecialHandling{
				Handle429RetryAfter:             false,
				ExponentialBackoffOn5xx:         false,
				CircuitBreakerEnabled:           false,
				CircuitBreakerFailureThreshold:  uint32Ptr(5),
				CircuitBreakerRecoveryTimeoutMS: uint64Ptr(30000),
			}
		}),
		Zhipu: providerStrategyWith(func(strategy *ProviderStrategy) {
			strategy.TimeoutMS = uint64Ptr(10000)
			strategy.MaxRetries = uint32Ptr(2)
			strategy.RateLimitPerMinute = uint32Ptr(60)
		}),
		Bocha: providerStrategyWith(func(strategy *ProviderStrategy) {
			strategy.TimeoutMS = uint64Ptr(10000)
			strategy.MaxRetries = uint32Ptr(2)
			strategy.RateLimitPerMinute = uint32Ptr(60)
		}),
	}
}

func normalizeProviderStrategies(strategies ProviderStrategies) ProviderStrategies {
	defaults := defaultProviderStrategies()
	if providerStrategyEmpty(strategies.Default) {
		strategies.Default = defaults.Default
	}
	return strategies
}

func providerStrategyEmpty(strategy ProviderStrategy) bool {
	return strategy.TimeoutMS == nil &&
		strategy.MaxRetries == nil &&
		strategy.InitialRetryDelayMS == nil &&
		strategy.MaxRetryDelayMS == nil &&
		strategy.BackoffMultiplier == nil &&
		strategy.MaxConcurrentRequests == nil &&
		strategy.RateLimitPerMinute == nil &&
		strategy.CacheEnabled == nil &&
		strategy.CacheTTLSeconds == nil &&
		strategy.CacheMaxEntries == nil &&
		strategy.SpecialHandling == nil
}

func defaultBackupConfig() BackupConfig {
	return BackupConfig{
		BackupDirectory:         nil,
		AutoBackupEnabled:       false,
		AutoBackupIntervalHours: 24,
		MaxBackupCount:          uint32Ptr(5),
		SlimBackup:              false,
		BackupTiers:             nil,
	}
}

func defaultRecoveryAPIConfigs() []ApiConfig {
	openAIProtocol := "openai_responses"
	openAISupportsResponses := true
	openAIProvider := "openai"
	anthropicProtocol := "anthropic"
	anthropicSupportsResponses := false
	anthropicProvider := "anthropic"

	return []ApiConfig{
		{
			ID:                      "openai-gpt4",
			Name:                    "OpenAI GPT-4",
			ProviderType:            &openAIProvider,
			ProviderScope:           &openAIProvider,
			ApiProtocol:             &openAIProtocol,
			SupportsOpenAIResponses: &openAISupportsResponses,
			ApiKey:                  "",
			BaseUrl:                 "https://api.openai.com/v1",
			Model:                   "gpt-4-turbo-preview",
			IsMultimodal:            true,
			Enabled:                 false,
			ModelAdapter:            "general",
			MaxOutputTokens:         4096,
			Temperature:             0.7,
			SupportsTools:           true,
			GeminiApiVersion:        "v1",
		},
		{
			ID:                      "claude-sonnet",
			Name:                    "Claude 3.5 Sonnet",
			ProviderType:            &anthropicProvider,
			ProviderScope:           &anthropicProvider,
			ApiProtocol:             &anthropicProtocol,
			SupportsOpenAIResponses: &anthropicSupportsResponses,
			ApiKey:                  "",
			BaseUrl:                 "https://api.anthropic.com/v1",
			Model:                   "claude-3-5-sonnet-20241022",
			IsMultimodal:            true,
			Enabled:                 false,
			ModelAdapter:            "anthropic",
			MaxOutputTokens:         4096,
			Temperature:             0.7,
			SupportsTools:           true,
			GeminiApiVersion:        "v1",
		},
	}
}

func stringOrDefault(value string, fallback string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return fallback
	}
	return value
}

func parseInt64Setting(value string, fallback int64) int64 {
	value = strings.TrimSpace(value)
	if value == "" {
		return fallback
	}
	parsed, err := strconv.ParseInt(value, 10, 64)
	if err != nil || parsed < 0 {
		return fallback
	}
	return parsed
}

func parseBoolSetting(value string, fallback bool) bool {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "true", "1", "yes", "on":
		return true
	case "false", "0", "no", "off":
		return false
	default:
		return fallback
	}
}

func (s *Service) resolveAPIKeyForTest(apiKey string, vendorID *string) (string, error) {
	trimmed := strings.TrimSpace(apiKey)
	if trimmed != "" && !isMaskedAPIKey(trimmed) {
		return trimmed, nil
	}

	vendorKey := strings.TrimSpace(valueOrEmpty(vendorID))
	if vendorKey == "" {
		return "", errors.New("API key is required")
	}

	vendors, err := s.GetVendorConfigs()
	if err == nil {
		for _, vendor := range vendors {
			if vendor.ID != vendorKey {
				continue
			}
			key := strings.TrimSpace(vendor.ApiKey)
			if key != "" && !isMaskedAPIKey(key) {
				return key, nil
			}
		}
	}

	configs, err := s.GetAPIConfigurations()
	if err == nil {
		for _, config := range configs {
			if config.VendorID == nil || strings.TrimSpace(*config.VendorID) != vendorKey {
				continue
			}
			key := strings.TrimSpace(config.ApiKey)
			if key != "" && !isMaskedAPIKey(key) {
				return key, nil
			}
		}
	}

	if strings.Contains(strings.ToLower(vendorKey), "siliconflow") {
		if key, ok := s.GetSetting("siliconflow.api_key"); ok {
			key = strings.TrimSpace(key)
			if key != "" && !isMaskedAPIKey(key) {
				return key, nil
			}
		}
	}

	return "", errors.New("API key is required")
}

func (s *Service) apiBaseForVendor(vendorID *string) string {
	vendorKey := strings.TrimSpace(valueOrEmpty(vendorID))
	if vendorKey == "" {
		return ""
	}
	vendors, err := s.GetVendorConfigs()
	if err == nil {
		for _, vendor := range vendors {
			if vendor.ID == vendorKey && strings.TrimSpace(vendor.BaseUrl) != "" {
				return vendor.BaseUrl
			}
		}
	}
	configs, err := s.GetAPIConfigurations()
	if err == nil {
		for _, config := range configs {
			if config.VendorID != nil && strings.TrimSpace(*config.VendorID) == vendorKey && strings.TrimSpace(config.BaseUrl) != "" {
				return config.BaseUrl
			}
		}
	}
	return ""
}

func (s *Service) apiHeadersForVendor(vendorID *string) map[string]string {
	vendorKey := strings.TrimSpace(valueOrEmpty(vendorID))
	if vendorKey == "" {
		return map[string]string{}
	}
	vendors, err := s.GetVendorConfigs()
	if err == nil {
		for _, vendor := range vendors {
			if vendor.ID == vendorKey {
				return vendor.Headers
			}
		}
	}
	return map[string]string{}
}

func isMaskedAPIKey(value string) bool {
	value = strings.TrimSpace(value)
	if value == "" {
		return true
	}
	for _, ch := range value {
		if ch != '*' {
			return false
		}
	}
	return true
}

func appendChatCompletionsEndpoint(baseURL string) string {
	return appendAPIEndpoint(baseURL, "openai_chat_completions")
}

func appendResponsesEndpoint(baseURL string) string {
	return appendAPIEndpoint(baseURL, "openai_responses")
}

func appendAPIEndpoint(baseURL string, protocol string) string {
	baseURL = strings.TrimRight(strings.TrimSpace(baseURL), "/")
	endpoint := "/chat/completions"
	if protocol == "openai_responses" {
		endpoint = "/responses"
	}
	if strings.HasSuffix(baseURL, endpoint) {
		return baseURL
	}
	for _, existing := range []string{"/chat/completions", "/responses"} {
		if strings.HasSuffix(baseURL, existing) {
			baseURL = strings.TrimSuffix(baseURL, existing)
			break
		}
	}
	return baseURL + endpoint
}

func resolveTestAPIProtocol(baseURL string, apiProtocol *string, supportsOpenAIResponses *bool) string {
	explicit := strings.ToLower(strings.TrimSpace(valueOrEmpty(apiProtocol)))
	supportsResponses := supportsOpenAIResponses != nil && *supportsOpenAIResponses
	isOfficialOpenAI := strings.Contains(strings.ToLower(strings.TrimSpace(baseURL)), "api.openai.com")
	if explicit == "openai_chat_completions" {
		return "openai_chat_completions"
	}
	if explicit == "openai_responses" {
		if supportsResponses || isOfficialOpenAI {
			return "openai_responses"
		}
		return "openai_chat_completions"
	}
	if supportsResponses || isOfficialOpenAI {
		return "openai_responses"
	}
	return "openai_chat_completions"
}

func apiTestRequestBody(modelID string, protocol string) map[string]any {
	if protocol == "openai_responses" {
		return map[string]any{
			"model":             strings.TrimSpace(modelID),
			"input":             "Hi",
			"max_output_tokens": 1,
			"stream":            false,
		}
	}
	return map[string]any{
		"model":      strings.TrimSpace(modelID),
		"messages":   []map[string]string{{"role": "user", "content": "Hi"}},
		"max_tokens": 1,
		"stream":     false,
	}
}

func (s *Service) testSearchEngineWithQuery(engine string, query string, topK int) (SearchEngineTestResult, error) {
	engine = normalizeSearchEngineID(engine)
	query = strings.TrimSpace(query)
	if query == "" {
		query = "connectivity test"
	}
	if topK <= 0 {
		topK = 1
	}

	if !isKnownSearchEngine(engine) {
		return SearchEngineTestResult{
			Ok:        false,
			Message:   fmt.Sprintf("未知搜索引擎: %s", engine),
			TestQuery: query,
		}, nil
	}

	cfg := s.webSearchConfig()
	if !cfg.hasEngine(engine) {
		return SearchEngineTestResult{
			Ok:        false,
			Message:   fmt.Sprintf("%s搜索引擎测试失败: 缺少API密钥或端点配置", engine),
			TestQuery: query,
		}, nil
	}

	timeout := cfg.Timeout
	if timeout <= 0 {
		timeout = 15 * time.Second
	}
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	start := time.Now()
	resultsCount, err := s.probeSearchEngine(ctx, cfg, engine, query, topK)
	elapsed := uint64(time.Since(start).Milliseconds())
	if err != nil {
		detail := err.Error()
		return SearchEngineTestResult{
			Ok:           false,
			Message:      fmt.Sprintf("%s搜索引擎测试失败: %s", engine, detail),
			ResponseTime: elapsed,
			TestQuery:    query,
			ErrorDetails: &detail,
		}, nil
	}

	return SearchEngineTestResult{
		Ok:           true,
		Message:      fmt.Sprintf("%s搜索引擎连接正常", engine),
		ResponseTime: elapsed,
		TestQuery:    query,
		ResultsCount: &resultsCount,
	}, nil
}

func (s *Service) webSearchConfig() webSearchConfig {
	values := s.GetSettings([]string{
		"web_search.engine",
		"web_search.timeout_ms",
		"web_search.api_key.google_cse",
		"web_search.google_cse.cx",
		"web_search.api_key.serpapi",
		"web_search.api_key.tavily",
		"web_search.tavily.search_depth",
		"web_search.api_key.brave",
		"web_search.searxng.endpoint",
		"web_search.searxng.api_key",
		"web_search.api_key.zhipu",
		"web_search.api_key.bocha",
	})

	timeoutMS := int(parseInt64Setting(values["web_search.timeout_ms"], 15000))
	if timeoutMS < 1000 {
		timeoutMS = 15000
	}
	if timeoutMS > 60000 {
		timeoutMS = 60000
	}

	tavilyDepth := strings.TrimSpace(values["web_search.tavily.search_depth"])
	if tavilyDepth == "" {
		tavilyDepth = "basic"
	}

	return webSearchConfig{
		Engine:          normalizeSearchEngineID(values["web_search.engine"]),
		Timeout:         time.Duration(timeoutMS) * time.Millisecond,
		GoogleCSE:       searchSecretOrEnv(values, "web_search.api_key.google_cse", "GOOGLE_API_KEY"),
		GoogleCSECX:     searchSettingOrEnv(values, "web_search.google_cse.cx", "GOOGLE_CSE_CX"),
		SerpAPI:         searchSecretOrEnv(values, "web_search.api_key.serpapi", "SERPAPI_KEY"),
		Tavily:          searchSecretOrEnv(values, "web_search.api_key.tavily", "TAVILY_API_KEY"),
		TavilyDepth:     tavilyDepth,
		Brave:           searchSecretOrEnv(values, "web_search.api_key.brave", "BRAVE_API_KEY"),
		SearxngEndpoint: searchSettingOrEnv(values, "web_search.searxng.endpoint", "SEARXNG_ENDPOINT"),
		SearxngAPIKey:   searchSecretOrEnv(values, "web_search.searxng.api_key", "SEARXNG_API_KEY"),
		Zhipu:           searchSecretOrEnv(values, "web_search.api_key.zhipu", "ZHIPU_API_KEY"),
		Bocha:           searchSecretOrEnv(values, "web_search.api_key.bocha", "BOCHA_API_KEY"),
	}
}

func (cfg webSearchConfig) hasEngine(engine string) bool {
	switch normalizeSearchEngineID(engine) {
	case "google_cse":
		return cfg.GoogleCSE != "" && cfg.GoogleCSECX != ""
	case "serpapi":
		return cfg.SerpAPI != ""
	case "tavily":
		return cfg.Tavily != ""
	case "brave":
		return cfg.Brave != ""
	case "searxng":
		return cfg.SearxngEndpoint != ""
	case "zhipu":
		return cfg.Zhipu != ""
	case "bocha":
		return cfg.Bocha != ""
	default:
		return false
	}
}

func (s *Service) probeSearchEngine(ctx context.Context, cfg webSearchConfig, engine string, query string, topK int) (int, error) {
	topK = clampInt(topK, 1, 10)
	switch engine {
	case "google_cse":
		u, _ := url.Parse("https://www.googleapis.com/customsearch/v1")
		q := u.Query()
		q.Set("key", cfg.GoogleCSE)
		q.Set("cx", cfg.GoogleCSECX)
		q.Set("q", query)
		q.Set("num", strconv.Itoa(topK))
		q.Set("start", "1")
		u.RawQuery = q.Encode()
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), nil)
		if err != nil {
			return 0, err
		}
		return s.executeSearchProbe(req, engine)
	case "serpapi":
		u, _ := url.Parse("https://serpapi.com/search.json")
		q := u.Query()
		q.Set("api_key", cfg.SerpAPI)
		q.Set("engine", "google")
		q.Set("q", query)
		q.Set("num", strconv.Itoa(topK))
		q.Set("start", "1")
		u.RawQuery = q.Encode()
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), nil)
		if err != nil {
			return 0, err
		}
		return s.executeSearchProbe(req, engine)
	case "tavily":
		return s.executeSearchJSONPost(ctx, engine, "https://api.tavily.com/search", map[string]any{
			"query":        query,
			"max_results":  topK,
			"search_depth": cfg.TavilyDepth,
		}, map[string]string{"Authorization": "Bearer " + cfg.Tavily})
	case "brave":
		u, _ := url.Parse("https://api.search.brave.com/res/v1/web/search")
		q := u.Query()
		q.Set("q", query)
		q.Set("count", strconv.Itoa(topK))
		u.RawQuery = q.Encode()
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), nil)
		if err != nil {
			return 0, err
		}
		req.Header.Set("Accept", "application/json")
		req.Header.Set("X-Subscription-Token", cfg.Brave)
		return s.executeSearchProbe(req, engine)
	case "searxng":
		endpoint := strings.TrimRight(strings.TrimSpace(cfg.SearxngEndpoint), "/") + "/search"
		u, err := url.Parse(endpoint)
		if err != nil || u.Scheme == "" || u.Host == "" {
			return 0, fmt.Errorf("invalid searxng endpoint: %s", cfg.SearxngEndpoint)
		}
		q := u.Query()
		q.Set("q", query)
		q.Set("format", "json")
		q.Set("categories", "general")
		q.Set("language", "all")
		q.Set("safesearch", "0")
		if cfg.SearxngAPIKey != "" && !strings.Contains(cfg.SearxngAPIKey, ":") {
			q.Set("apikey", cfg.SearxngAPIKey)
		}
		u.RawQuery = q.Encode()
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), nil)
		if err != nil {
			return 0, err
		}
		if user, pass, ok := strings.Cut(cfg.SearxngAPIKey, ":"); ok {
			req.SetBasicAuth(user, pass)
		} else if cfg.SearxngAPIKey != "" {
			req.Header.Set("Authorization", "Bearer "+cfg.SearxngAPIKey)
			req.Header.Set("X-API-Key", cfg.SearxngAPIKey)
		}
		return s.executeSearchProbe(req, engine)
	case "zhipu":
		return s.executeSearchJSONPost(ctx, engine, "https://open.bigmodel.cn/api/paas/v4/web_search", map[string]any{
			"search_engine": "search-prime",
			"search_query":  query,
			"count":         topK,
			"content_size":  "high",
		}, map[string]string{"Authorization": "Bearer " + cfg.Zhipu})
	case "bocha":
		return s.executeSearchJSONPost(ctx, engine, "https://api.bochaai.com/v1/web-search", map[string]any{
			"query":   query,
			"count":   topK,
			"summary": false,
		}, map[string]string{"Authorization": "Bearer " + cfg.Bocha})
	default:
		return 0, fmt.Errorf("unsupported search engine: %s", engine)
	}
}

func (s *Service) executeSearchJSONPost(ctx context.Context, engine string, endpoint string, body map[string]any, headers map[string]string) (int, error) {
	bodyBytes, err := json.Marshal(body)
	if err != nil {
		return 0, err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(bodyBytes))
	if err != nil {
		return 0, err
	}
	req.Header.Set("Content-Type", "application/json")
	for key, value := range headers {
		if strings.TrimSpace(key) != "" && strings.TrimSpace(value) != "" {
			req.Header.Set(key, value)
		}
	}
	return s.executeSearchProbe(req, engine)
}

func (s *Service) executeSearchProbe(req *http.Request, engine string) (int, error) {
	req.Header.Set("User-Agent", "deep-student-go/0.1")
	resp, err := s.currentHTTPClient().Do(req)
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(io.LimitReader(resp.Body, 256*1024))
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return 0, searchProviderHTTPError(engine, resp.Status, body)
	}
	if len(strings.TrimSpace(string(body))) == 0 {
		return 0, nil
	}

	var payload map[string]any
	if err := json.Unmarshal(body, &payload); err != nil {
		return 0, fmt.Errorf("%s response JSON parse failed: %w", engine, err)
	}
	return resultCountForSearchEngine(engine, payload), nil
}

func searchProviderHTTPError(engine string, status string, body []byte) error {
	bodyText := strings.TrimSpace(string(body))
	message := ""
	var payload map[string]any
	if err := json.Unmarshal(body, &payload); err == nil {
		message = firstJSONText(payload, []string{"error", "message"}, []string{"message"}, []string{"error"})
	}
	if message == "" {
		message = bodyText
	}
	if message == "" {
		message = "empty response"
	}
	if len(message) > 512 {
		message = message[:512]
	}
	return fmt.Errorf("%s http %s: %s", engine, status, message)
}

func resultCountForSearchEngine(engine string, payload map[string]any) int {
	switch engine {
	case "google_cse":
		return jsonArrayLen(payload, "items")
	case "serpapi":
		return jsonArrayLen(payload, "organic_results")
	case "tavily", "searxng":
		return jsonArrayLen(payload, "results")
	case "brave":
		return jsonArrayLen(payload, "web", "results")
	case "zhipu":
		return jsonArrayLen(payload, "search_result")
	case "bocha":
		if count := jsonArrayLen(payload, "data", "webPages", "value"); count > 0 {
			return count
		}
		return jsonArrayLen(payload, "webPages", "value")
	default:
		return 0
	}
}

func jsonArrayLen(value any, path ...string) int {
	current := value
	for _, key := range path {
		obj, ok := current.(map[string]any)
		if !ok {
			return 0
		}
		current = obj[key]
	}
	items, ok := current.([]any)
	if !ok {
		return 0
	}
	return len(items)
}

func firstJSONText(value any, paths ...[]string) string {
	for _, path := range paths {
		current := value
		for _, key := range path {
			obj, ok := current.(map[string]any)
			if !ok {
				current = nil
				break
			}
			current = obj[key]
		}
		if text, ok := current.(string); ok && strings.TrimSpace(text) != "" {
			return strings.TrimSpace(text)
		}
	}
	return ""
}

func firstConfiguredSearchEngine(cfg webSearchConfig) string {
	for _, engine := range []string{"tavily", "google_cse", "brave", "serpapi", "searxng", "bocha", "zhipu"} {
		if cfg.hasEngine(engine) {
			return engine
		}
	}
	return ""
}

func isKnownSearchEngine(engine string) bool {
	engine = normalizeSearchEngineID(engine)
	for _, item := range webSearchEngines {
		if item.ID == engine {
			return true
		}
	}
	return false
}

func normalizeSearchEngineID(engine string) string {
	return strings.TrimSpace(strings.ToLower(engine))
}

func searchSettingOrEnv(values map[string]string, key string, envKey string) string {
	value := strings.TrimSpace(values[key])
	if value == "" {
		value = strings.TrimSpace(os.Getenv(envKey))
	}
	return value
}

func searchSecretOrEnv(values map[string]string, key string, envKey string) string {
	value := searchSettingOrEnv(values, key, envKey)
	if isMaskedAPIKey(value) {
		return ""
	}
	return value
}

func clampInt(value int, minValue int, maxValue int) int {
	if value < minValue {
		return minValue
	}
	if value > maxValue {
		return maxValue
	}
	return value
}

func valueOrZero(value *int) int {
	if value == nil {
		return 0
	}
	return *value
}

func normalizeMemoryAutoExtractFrequency(value string) string {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "off", "balanced", "aggressive":
		return strings.ToLower(strings.TrimSpace(value))
	default:
		return "balanced"
	}
}

func parseSiteList(value string) []string {
	value = strings.TrimSpace(value)
	if value == "" {
		return []string{}
	}

	var jsonSites []string
	if err := json.Unmarshal([]byte(value), &jsonSites); err == nil {
		return cleanSiteList(jsonSites)
	}

	return cleanSiteList(strings.Split(value, ","))
}

func cleanSiteList(values []string) []string {
	out := make([]string, 0, len(values))
	seen := make(map[string]bool, len(values))
	for _, value := range values {
		site := strings.TrimSpace(value)
		if site == "" || seen[site] {
			continue
		}
		seen[site] = true
		out = append(out, site)
	}
	return out
}

func normalizeAPIConfigs(configs []ApiConfig) []ApiConfig {
	out := make([]ApiConfig, 0, len(configs))
	for _, config := range configs {
		config.ID = strings.TrimSpace(config.ID)
		config.Name = strings.TrimSpace(config.Name)
		config.Model = strings.TrimSpace(config.Model)
		config.BaseUrl = strings.TrimSpace(config.BaseUrl)
		if config.ID == "" {
			config.ID = "api_" + randomToken(12)
		}
		if config.Name == "" {
			config.Name = firstNonEmpty(config.Model, config.ID)
		}
		if config.ModelAdapter == "" {
			config.ModelAdapter = "general"
		}
		if config.MaxOutputTokens == 0 {
			config.MaxOutputTokens = 4096
		}
		if config.Temperature == 0 {
			config.Temperature = 0.7
		}
		if config.GeminiApiVersion == "" {
			config.GeminiApiVersion = "v1"
		}
		out = append(out, config)
	}
	return out
}

func normalizeVendorConfigs(configs []VendorConfig) []VendorConfig {
	out := make([]VendorConfig, 0, len(configs))
	for _, config := range configs {
		config.ID = strings.TrimSpace(config.ID)
		config.Name = strings.TrimSpace(config.Name)
		config.ProviderType = strings.TrimSpace(config.ProviderType)
		config.BaseUrl = strings.TrimSpace(config.BaseUrl)
		if config.ID == "" {
			config.ID = "vendor_" + randomToken(12)
		}
		if config.Name == "" {
			config.Name = firstNonEmpty(config.ProviderType, config.ID)
		}
		if config.ProviderType == "" {
			config.ProviderType = "openai"
		}
		if config.Headers == nil {
			config.Headers = map[string]string{}
		}
		out = append(out, config)
	}
	return out
}

func normalizeModelProfiles(profiles []ModelProfile) []ModelProfile {
	out := make([]ModelProfile, 0, len(profiles))
	for _, profile := range profiles {
		profile.ID = strings.TrimSpace(profile.ID)
		profile.VendorID = strings.TrimSpace(profile.VendorID)
		profile.Label = strings.TrimSpace(profile.Label)
		profile.Model = strings.TrimSpace(profile.Model)
		if profile.ID == "" {
			profile.ID = "model_" + randomToken(12)
		}
		if profile.Label == "" {
			profile.Label = firstNonEmpty(profile.Model, profile.ID)
		}
		if profile.ModelAdapter == "" {
			profile.ModelAdapter = "general"
		}
		if profile.Status == "" {
			profile.Status = "active"
		}
		if profile.MaxOutputTokens == 0 {
			profile.MaxOutputTokens = 4096
		}
		if profile.Temperature == 0 {
			profile.Temperature = 0.7
		}
		out = append(out, profile)
	}
	return out
}

func normalizeModelAssignments(assignments ModelAssignments) ModelAssignments {
	mode := strings.TrimSpace(valueOrEmpty(assignments.TranslationDisplayMode))
	if mode != "" && mode != "aligned" && mode != "streaming" {
		mode = "aligned"
	}
	if mode != "" {
		assignments.TranslationDisplayMode = &mode
	}
	assignments.Model2ConfigID = cleanOptional(assignments.Model2ConfigID)
	assignments.ReviewAnalysisModelConfigID = cleanOptional(assignments.ReviewAnalysisModelConfigID)
	assignments.AnkiCardModelConfigID = cleanOptional(assignments.AnkiCardModelConfigID)
	assignments.QbankAIGradingModelConfigID = cleanOptional(assignments.QbankAIGradingModelConfigID)
	assignments.EmbeddingModelConfigID = cleanOptional(assignments.EmbeddingModelConfigID)
	assignments.RerankerModelConfigID = cleanOptional(assignments.RerankerModelConfigID)
	assignments.ChatTitleModelConfigID = cleanOptional(assignments.ChatTitleModelConfigID)
	assignments.ExamSheetOCRModelConfigID = cleanOptional(assignments.ExamSheetOCRModelConfigID)
	assignments.TranslationModelConfigID = cleanOptional(assignments.TranslationModelConfigID)
	assignments.VLEmbeddingModelConfigID = cleanOptional(assignments.VLEmbeddingModelConfigID)
	assignments.VLRerankerModelConfigID = cleanOptional(assignments.VLRerankerModelConfigID)
	assignments.MemoryDecisionModelConfigID = cleanOptional(assignments.MemoryDecisionModelConfigID)
	assignments.VoiceInputASRModelConfigID = cleanOptional(assignments.VoiceInputASRModelConfigID)
	assignments.ImageGenerationModelConfigID = cleanOptional(assignments.ImageGenerationModelConfigID)
	return assignments
}

func defaultModelAssignments() ModelAssignments {
	return ModelAssignments{}
}

func cleanOptional(value *string) *string {
	if value == nil {
		return nil
	}
	trimmed := strings.TrimSpace(*value)
	if trimmed == "" {
		return nil
	}
	return &trimmed
}

func valueOrEmpty(value *string) string {
	if value == nil {
		return ""
	}
	return *value
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if trimmed := strings.TrimSpace(value); trimmed != "" {
			return trimmed
		}
	}
	return ""
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
