// Package llmcfg 提供模型厂商配置系统的核心类型与持久化能力。
//
// 100% 复刻 Rust 原版 src-tauri/src/llm_manager 的 ApiConfig / VendorConfig /
// ModelProfile 设计，JSON 字段统一 camelCase，与 serde rename_all = "camelCase" 对齐。
package llmcfg

// VendorConfig 供应商配置。
//
// 对齐 Rust VendorConfig：描述一个 LLM 供应商入口（baseURL、API Key、协议等）。
type VendorConfig struct {
	ID                      string            `json:"id"`
	Name                    string            `json:"name"`
	ProviderType            string            `json:"providerType"`
	APIProtocol             *string           `json:"apiProtocol,omitempty"`
	SupportsOpenAIResponses *bool             `json:"supportsOpenAIResponses,omitempty"`
	BaseURL                 string            `json:"baseUrl"`
	APIKey                  string            `json:"apiKey"`
	Headers                 map[string]string `json:"headers"`
	RateLimitPerMinute      *int              `json:"rateLimitPerMinute,omitempty"`
	DefaultTimeoutMs        *int64            `json:"defaultTimeoutMs,omitempty"`
	Notes                   *string           `json:"notes,omitempty"`
	IsBuiltin               bool              `json:"isBuiltin"`
	IsReadOnly              bool              `json:"isReadOnly"`
	SortOrder               *int              `json:"sortOrder,omitempty"`
	MaxTokensLimit          *uint32           `json:"maxTokensLimit,omitempty"`
	WebsiteURL              *string           `json:"websiteUrl,omitempty"`
}

// ModelProfile 模型配置。
//
// 对齐 Rust ModelProfile：描述一个具体的模型条目（model ID、能力、温度等）。
type ModelProfile struct {
	ID                string   `json:"id"`
	VendorID          string   `json:"vendorId"`
	Label             string   `json:"label"`
	Model             string   `json:"model"`
	ProviderScope     *string  `json:"providerScope,omitempty"`
	APIProtocol       *string  `json:"apiProtocol,omitempty"`
	ModelAdapter      string   `json:"modelAdapter"`
	IsMultimodal      bool     `json:"isMultimodal"`
	IsReasoning       bool     `json:"isReasoning"`
	IsEmbedding       bool     `json:"isEmbedding"`
	IsReranker        bool     `json:"isReranker"`
	IsImageGeneration bool     `json:"isImageGeneration"`
	SupportsTools     bool     `json:"supportsTools"`
	SupportsReasoning bool     `json:"supportsReasoning"`
	Status            string   `json:"status"`
	Enabled           bool     `json:"enabled"`
	MaxOutputTokens   uint32   `json:"maxOutputTokens"`
	Temperature       float32  `json:"temperature"`
	ReasoningEffort   *string  `json:"reasoningEffort,omitempty"`
	ThinkingEnabled   bool     `json:"thinkingEnabled"`
	ThinkingBudget    *int32   `json:"thinkingBudget,omitempty"`
	IncludeThoughts   bool     `json:"includeThoughts"`
	EnableThinking    *bool    `json:"enableThinking,omitempty"`
	MinP              *float32 `json:"minP,omitempty"`
	TopK              *uint32  `json:"topK,omitempty"`
	GeminiAPIVersion  *string  `json:"geminiApiVersion,omitempty"`
	IsBuiltin         bool     `json:"isBuiltin"`
	IsFavorite        bool     `json:"isFavorite"`
	MaxTokensLimit    *uint32  `json:"maxTokensLimit,omitempty"`
	ContextWindow     *uint32  `json:"contextWindow,omitempty"`
	RepetitionPenalty *float32 `json:"repetitionPenalty,omitempty"`
	ReasoningSplit    *bool    `json:"reasoningSplit,omitempty"`
	Effort            *string  `json:"effort,omitempty"`
	Verbosity         *string  `json:"verbosity,omitempty"`
}

// ModelAssignments 模型分配（哪个 profile 给哪个用途）。
//
// 对齐 Rust ModelAssignments：每个字段是一个 profile ID，可空。
type ModelAssignments struct {
	Model2ConfigID              *string `json:"model2ConfigId,omitempty"`
	ReviewAnalysisModelConfigID *string `json:"reviewAnalysisModelConfigId,omitempty"`
	AnkiCardModelConfigID       *string `json:"ankiCardModelConfigId,omitempty"`
	QBankAIGradingModelConfigID *string `json:"qbankAiGradingModelConfigId,omitempty"`
	EmbeddingModelConfigID      *string `json:"embeddingModelConfigId,omitempty"`
	RerankerModelConfigID       *string `json:"rerankerModelConfigId,omitempty"`
	ChatTitleModelConfigID      *string `json:"chatTitleModelConfigId,omitempty"`
	ExamSheetOCRModelConfigID   *string `json:"examSheetOcrModelConfigId,omitempty"`
	TranslationModelConfigID    *string `json:"translationModelConfigId,omitempty"`
	VLEmbeddingModelConfigID    *string `json:"vlEmbeddingModelConfigId,omitempty"`
	VLRerankerModelConfigID     *string `json:"vlRerankerModelConfigId,omitempty"`
	MemoryDecisionModelConfigID *string `json:"memoryDecisionModelConfigId,omitempty"`
	VoiceInputASRModelConfigID  *string `json:"voiceInputAsrModelConfigId,omitempty"`
	ImageGenerationModelConfigID *string `json:"imageGenerationModelConfigId,omitempty"`
	TranslationDisplayMode      *string `json:"translationDisplayMode,omitempty"`
}

// ApiConfig 运行时合并配置（Vendor + Profile → 单一可调用的配置）。
//
// 对齐 Rust ApiConfig：把 vendor 与 profile 字段扁平化成调用层直接可用的结构。
type ApiConfig struct {
	ID                       string            `json:"id"`
	Name                     string            `json:"name"`
	VendorID                 *string           `json:"vendorId,omitempty"`
	VendorName               *string           `json:"vendorName,omitempty"`
	ProviderType             *string           `json:"providerType,omitempty"`
	ProviderScope            *string           `json:"providerScope,omitempty"`
	APIProtocol              *string           `json:"apiProtocol,omitempty"`
	SupportsOpenAIResponses  *bool             `json:"supportsOpenAIResponses,omitempty"`
	APIKey                   string            `json:"apiKey"`
	BaseURL                  string            `json:"baseUrl"`
	Model                    string            `json:"model"`
	IsMultimodal             bool              `json:"isMultimodal"`
	IsReasoning              bool              `json:"isReasoning"`
	IsEmbedding              bool              `json:"isEmbedding"`
	IsReranker               bool              `json:"isReranker"`
	IsImageGeneration        bool              `json:"isImageGeneration"`
	Enabled                  bool              `json:"enabled"`
	ModelAdapter             string            `json:"modelAdapter"`
	MaxOutputTokens          uint32            `json:"maxOutputTokens"`
	Temperature              float32           `json:"temperature"`
	SupportsTools            bool              `json:"supportsTools"`
	GeminiAPIVersion         string            `json:"geminiApiVersion"`
	IsBuiltin                bool              `json:"isBuiltin"`
	IsReadOnly               bool              `json:"isReadOnly"`
	ReasoningEffort          *string           `json:"reasoningEffort,omitempty"`
	ThinkingEnabled          bool              `json:"thinkingEnabled"`
	ThinkingBudget           *int32            `json:"thinkingBudget,omitempty"`
	IncludeThoughts          bool              `json:"includeThoughts"`
	MinP                     *float32          `json:"minP,omitempty"`
	TopK                     *uint32           `json:"topK,omitempty"`
	EnableThinking           *bool             `json:"enableThinking,omitempty"`
	SupportsReasoning        bool              `json:"supportsReasoning"`
	Headers                  map[string]string `json:"headers,omitempty"`
	TopPOverride             *float32          `json:"topPOverride,omitempty"`
	FrequencyPenaltyOverride *float32          `json:"frequencyPenaltyOverride,omitempty"`
	PresencePenaltyOverride  *float32          `json:"presencePenaltyOverride,omitempty"`
	RepetitionPenalty        *float32          `json:"repetitionPenalty,omitempty"`
	ReasoningSplit           *bool             `json:"reasoningSplit,omitempty"`
	Effort                   *string           `json:"effort,omitempty"`
	Verbosity                *string           `json:"verbosity,omitempty"`
	IsFavorite               bool              `json:"isFavorite"`
	MaxTokensLimit           *uint32           `json:"maxTokensLimit,omitempty"`
	ContextWindow            *uint32           `json:"contextWindow,omitempty"`
}

// TestConnectionResult 测试连接结果。
type TestConnectionResult struct {
	OK         bool   `json:"ok"`
	Message    string `json:"message"`
	LatencyMs  int64  `json:"latencyMs"`
	Model      string `json:"model,omitempty"`
	VendorName string `json:"vendorName,omitempty"`
}

// 默认值常量（与 Rust default_model_adapter / default_max_output_tokens / default_temperature 等对齐）。
const (
	defaultModelAdapter    = "general"
	defaultMaxOutputTokens = uint32(4096)
	defaultTemperature     = float32(0.7)
	defaultProfileStatus   = "enabled"
	defaultProfileEnabled  = true
	defaultGeminiAPIVer    = "v1beta"

	// 协议枚举（简化版，覆盖任务要求四种）。
	protocolOpenAIChat        = "openai_chat"
	protocolOpenAIResponses   = "openai_responses"
	protocolAnthropicMessages = "anthropic_messages"
	protocolGeminiGenerate    = "gemini_generateContent"
)

// strPtr 工具：把字符串字面量转成 *string，便于构造可选字段。
func strPtr(s string) *string { return &s }

// boolPtr 工具：把 bool 转成 *bool。
func boolPtr(b bool) *bool { return &b }

// intPtr 工具：把 int 转成 *int。
func intPtr(i int) *int { return &i }

// u32Ptr 工具：把 uint32 转成 *uint32。
func u32Ptr(v uint32) *uint32 { return &v }

// resolvePreferredProtocol 简化版协议解析。
//
// 对齐 Rust resolve_preferred_protocol_for_provider 的核心分支：
//   - adapter == "google" → gemini_generateContent
//   - adapter == "anthropic" → anthropic_messages
//   - OpenAI 官方入口 → openai_responses（官方建议默认 Responses）
//   - 其余 OpenAI 兼容入口 → openai_chat
//
// 10 家内置厂商（除 Gemini 外）均为 OpenAI 兼容 chat 协议；OpenAI 额外支持 Responses。
func resolvePreferredProtocol(providerType, baseURL string, supportsResponses *bool) string {
	if supportsResponses != nil && *supportsResponses {
		return protocolOpenAIResponses
	}
	switch providerType {
	case "anthropic":
		return protocolAnthropicMessages
	case "gemini", "google":
		return protocolGeminiGenerate
	case "openai":
		// OpenAI 官方入口默认走 Responses
		if isOfficialOpenAI(providerType, baseURL) {
			return protocolOpenAIResponses
		}
		return protocolOpenAIChat
	default:
		return protocolOpenAIChat
	}
}

// providerSupportsOpenAIResponses 简化版：仅 OpenAI 官方入口返回 true。
func providerSupportsOpenAIResponses(providerType, baseURL string, explicit *bool) bool {
	if explicit != nil {
		return *explicit
	}
	return isOfficialOpenAI(providerType, baseURL)
}

// isOfficialOpenAI 判断是否为 OpenAI 官方入口。
func isOfficialOpenAI(providerType, baseURL string) bool {
	if providerType != "openai" {
		return false
	}
	return baseURL == "https://api.openai.com/v1" || baseURL == "https://api.openai.com"
}
