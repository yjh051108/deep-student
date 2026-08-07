// 内置厂商和模型定义 —— 完全照搬 Rust builtin_vendors.rs。
//
// 包含 10 家厂商（SiliconFlow / DeepSeek / 通义千问 / 智谱 / 豆包 / MiniMax /
// 月之暗面 / OpenAI / NVIDIA / MiMo）及其全部模型。Gemini 走单独 registry，此处省略。
//
// 注意（与 Rust 一致）：
//   - 供应商 is_builtin=true 表示入口不可删除
//   - 模型 is_builtin=false 表示用户可自由编辑和删除模型配置
package llmcfg

import "strings"

// builtinVendor 内置供应商定义（对应 Rust BuiltinVendor）。
type builtinVendor struct {
	id             string
	name           string
	providerType   string
	baseURL        string
	notes          string
	maxTokensLimit *uint32
	websiteURL     string
}

// builtinModel 内置模型定义（对应 Rust BuiltinModel）。
type builtinModel struct {
	id             string
	vendorID       string
	label          string
	model          string
	isMultimodal   bool
	isReasoning    bool
	supportsTools  bool
	maxOutputTokens uint32
	temperature    float32
}

// builtinVendors 所有内置供应商列表（对应 Rust BUILTIN_VENDORS）。
//
// 顺序与 Rust 一致：SiliconFlow / DeepSeek / 通义千问 / 智谱 / 豆包 / MiniMax /
// 月之暗面 / OpenAI / NVIDIA / MiMo。
var builtinVendors = []builtinVendor{
	{
		id:           "builtin-siliconflow",
		name:         "SiliconFlow",
		providerType: "siliconflow",
		baseURL:      "https://api.siliconflow.cn/v1",
		notes:        "Built-in template for SiliconFlow. Please enter your API Key.",
		websiteURL:   "https://cloud.siliconflow.cn/i/deadXN1B",
	},
	{
		id:             "builtin-deepseek",
		name:           "DeepSeek",
		providerType:   "deepseek",
		baseURL:        "https://api.deepseek.com/v1",
		notes:          "DeepSeek 官方 API。推荐模型: deepseek-v4-flash, deepseek-v4-pro。兼容别名: deepseek-chat, deepseek-reasoner（官方计划于 2026-07-24 后逐步弃用）。根据 Thinking Mode 文档，当前请求层 max_tokens 默认 32K、最大 64K。",
		maxTokensLimit: u32Ptr(65_536),
		websiteURL:     "https://deepseek.com",
	},
	{
		id:           "builtin-qwen",
		name:         "通义千问",
		providerType: "qwen",
		baseURL:      "https://dashscope.aliyuncs.com/compatible-mode/v1",
		notes:        "阿里云百炼 API（兼容 OpenAI Chat；平台亦支持 Responses / DashScope 原生）。推荐模型: qwen3.5-plus, qwen3.5-flash, qwen3-max, qwen3.5-397b-a17b, qwen3.5-122b-a10b, qwq-plus",
		websiteURL:   "https://bailian.console.aliyun.com",
	},
	{
		id:           "builtin-zhipu",
		name:         "智谱AI",
		providerType: "zhipu",
		baseURL:      "https://open.bigmodel.cn/api/paas/v4",
		notes:        "智谱AI 开放平台。可用模型: glm-5(最新旗舰), glm-4.7, glm-4.6, glm-4.7-flash(免费)",
		websiteURL:   "https://open.bigmodel.cn",
	},
	{
		id:           "builtin-doubao",
		name:         "字节豆包",
		providerType: "doubao",
		baseURL:      "https://ark.cn-beijing.volces.com/api/v3",
		notes:        "火山方舟大模型平台。推荐模型: Seed 2.0 Pro/Lite/Mini/Code (可直接用模型名调用), Seed 1.8",
		websiteURL:   "https://www.volcengine.com/product/doubao",
	},
	{
		id:           "builtin-minimax",
		name:         "MiniMax",
		providerType: "minimax",
		baseURL:      "https://api.minimax.io/v1",
		notes:        "MiniMax API。可用模型: MiniMax-M2.5(最新), M2.5-highspeed, M2.1, M2",
		websiteURL:   "https://platform.minimaxi.com",
	},
	{
		id:           "builtin-moonshot",
		name:         "月之暗面",
		providerType: "moonshot",
		baseURL:      "https://api.moonshot.cn/v1",
		notes:        "Kimi API。可用模型: kimi-k2.5(多模态), kimi-k2, kimi-k2-thinking, kimi-latest",
		websiteURL:   "https://platform.moonshot.cn",
	},
	{
		id:           "builtin-openai",
		name:         "OpenAI",
		providerType: "openai",
		baseURL:      "https://api.openai.com/v1",
		notes:        "OpenAI 官方 API。根据 OpenAI 官方模型文档，当前 GPT-5.x 家族可用模型包括: gpt-5.5, gpt-5.5-pro, gpt-5.4, gpt-5.4-pro, gpt-5.4-mini, gpt-5.4-nano；全部模型页仍列出 gpt-5.2, gpt-5.2-pro, gpt-5.1, gpt-5, gpt-5-pro, gpt-5-mini, gpt-5-nano，以及 o3-pro/o3/o4-mini。默认协议建议使用 Responses。",
		websiteURL:   "https://platform.openai.com",
	},
	{
		id:           "builtin-nvidia",
		name:         "NVIDIA",
		providerType: "nvidia",
		baseURL:      "https://integrate.api.nvidia.com/v1",
		notes:        "NVIDIA NIM hosted API。OpenAI-compatible Chat Completions；模型可通过 /models 拉取。默认不注入 thinking/reasoning 专用参数，避免不同 NIM 模型参数格式不一致。",
		websiteURL:   "https://build.nvidia.com/nim",
	},
	{
		id:           "builtin-mimo",
		name:         "Xiaomi MiMo",
		providerType: "mimo",
		baseURL:      "https://api.xiaomimimo.com/v1",
		notes:        "Xiaomi MiMo API。优先内置 MiMo V2.5-Pro 与 MiMo V2.5（1M context，OpenAI-compatible Chat Completions）；Token Plan 可将 Base URL 改为 token-plan-*.xiaomimimo.com/v1。支持 thinking: { type } 与 reasoning_content 回传。V2.5 TTS/ASR 属语音专项能力，当前不放入聊天模型默认列表。",
		websiteURL:   "https://platform.xiaomimimo.com",
	},
}

// builtinModels 所有内置模型列表（对应 Rust BUILTIN_MODELS）。
//
// 顺序与 Rust 一致：DeepSeek / 通义千问 / 智谱 / 豆包 / MiniMax / 月之暗面 /
// OpenAI / NVIDIA / MiMo。
var builtinModels = []builtinModel{
	// ===== DeepSeek 模型 =====
	{id: "builtin-deepseek-v4-flash", vendorID: "builtin-deepseek", label: "DeepSeek V4 Flash (官方推荐)", model: "deepseek-v4-flash", isReasoning: true, supportsTools: true, maxOutputTokens: 32_768, temperature: 0.6},
	{id: "builtin-deepseek-v4-pro", vendorID: "builtin-deepseek", label: "DeepSeek V4 Pro (官方推荐)", model: "deepseek-v4-pro", isReasoning: true, supportsTools: true, maxOutputTokens: 32_768, temperature: 0.6},
	{id: "builtin-deepseek-chat", vendorID: "builtin-deepseek", label: "DeepSeek Chat (兼容别名/非思考)", model: "deepseek-chat", supportsTools: true, maxOutputTokens: 32_768, temperature: 0.7},
	{id: "builtin-deepseek-reasoner", vendorID: "builtin-deepseek", label: "DeepSeek Reasoner (兼容别名/思考)", model: "deepseek-reasoner", isReasoning: true, supportsTools: true, maxOutputTokens: 32_768, temperature: 0.7},
	// ===== 通义千问模型 =====
	{id: "builtin-qwen3-max", vendorID: "builtin-qwen", label: "Qwen3 Max (旗舰)", model: "qwen3-max", supportsTools: true, maxOutputTokens: 65_536, temperature: 0.7},
	{id: "builtin-qwen3.5-plus", vendorID: "builtin-qwen", label: "Qwen3.5 Plus (多模态/混合思考)", model: "qwen3.5-plus", isMultimodal: true, isReasoning: true, supportsTools: true, maxOutputTokens: 65_536, temperature: 0.7},
	{id: "builtin-qwen3.5-flash", vendorID: "builtin-qwen", label: "Qwen3.5 Flash (快速/混合思考)", model: "qwen3.5-flash", isReasoning: true, supportsTools: true, maxOutputTokens: 65_536, temperature: 0.7},
	{id: "builtin-qwen-plus", vendorID: "builtin-qwen", label: "Qwen Plus (支持思考)", model: "qwen-plus", isReasoning: true, supportsTools: true, maxOutputTokens: 32_768, temperature: 0.7},
	{id: "builtin-qwq-plus", vendorID: "builtin-qwen", label: "QwQ Plus (推理模型)", model: "qwq-plus", isReasoning: true, supportsTools: true, maxOutputTokens: 8_192, temperature: 0.7},
	{id: "builtin-qwen3.5-397b-a17b", vendorID: "builtin-qwen", label: "Qwen3.5 397B A17B (开源旗舰)", model: "qwen3.5-397b-a17b", isReasoning: true, supportsTools: true, maxOutputTokens: 65_536, temperature: 0.7},
	{id: "builtin-qwen3.5-122b-a10b", vendorID: "builtin-qwen", label: "Qwen3.5 122B A10B (开源旗舰)", model: "qwen3.5-122b-a10b", isReasoning: true, supportsTools: true, maxOutputTokens: 65_536, temperature: 0.7},
	// ===== 智谱AI模型 =====
	{id: "builtin-glm-5", vendorID: "builtin-zhipu", label: "GLM-5 (最新旗舰)", model: "glm-5", isReasoning: true, supportsTools: true, maxOutputTokens: 16_384, temperature: 0.7},
	{id: "builtin-glm-4.7", vendorID: "builtin-zhipu", label: "GLM-4.7 (高性价比)", model: "glm-4.7", isReasoning: true, supportsTools: true, maxOutputTokens: 16_384, temperature: 0.7},
	{id: "builtin-glm-4.6", vendorID: "builtin-zhipu", label: "GLM-4.6 (上一代)", model: "glm-4.6", isReasoning: true, supportsTools: true, maxOutputTokens: 16_384, temperature: 0.7},
	{id: "builtin-glm-4.7-flash", vendorID: "builtin-zhipu", label: "GLM-4.7 Flash (免费)", model: "glm-4.7-flash", supportsTools: true, maxOutputTokens: 8_192, temperature: 0.7},
	// ===== 字节豆包模型 =====
	{id: "builtin-doubao-seed-2.0-pro", vendorID: "builtin-doubao", label: "Seed 2.0 Pro (旗舰全能)", model: "doubao-seed-2-0-pro-260215", isMultimodal: true, isReasoning: true, supportsTools: true, maxOutputTokens: 65_535, temperature: 0.7},
	{id: "builtin-doubao-seed-2.0-lite", vendorID: "builtin-doubao", label: "Seed 2.0 Lite (均衡)", model: "doubao-seed-2-0-lite-260215", isMultimodal: true, isReasoning: true, supportsTools: true, maxOutputTokens: 65_535, temperature: 0.7},
	{id: "builtin-doubao-seed-2.0-mini", vendorID: "builtin-doubao", label: "Seed 2.0 Mini (快速)", model: "doubao-seed-2-0-mini-260215", isMultimodal: true, isReasoning: true, supportsTools: true, maxOutputTokens: 65_535, temperature: 0.7},
	{id: "builtin-doubao-seed-2.0-code", vendorID: "builtin-doubao", label: "Seed 2.0 Code (编程)", model: "doubao-seed-2-0-code-preview-260215", isReasoning: true, supportsTools: true, maxOutputTokens: 65_535, temperature: 0.7},
	{id: "builtin-doubao-1.8-pro", vendorID: "builtin-doubao", label: "Seed 1.8 (上一代)", model: "doubao-seed-1-8-251215", isMultimodal: true, isReasoning: true, supportsTools: true, maxOutputTokens: 65_535, temperature: 0.7},
	// ===== MiniMax 模型 =====
	{id: "builtin-minimax-m2.5", vendorID: "builtin-minimax", label: "MiniMax M2.5 (最新旗舰)", model: "MiniMax-M2.5", isReasoning: true, supportsTools: true, maxOutputTokens: 16_384, temperature: 1.0},
	{id: "builtin-minimax-m2.5-highspeed", vendorID: "builtin-minimax", label: "MiniMax M2.5 Highspeed (极速)", model: "MiniMax-M2.5-highspeed", supportsTools: true, maxOutputTokens: 8_192, temperature: 1.0},
	{id: "builtin-minimax-m2.1", vendorID: "builtin-minimax", label: "MiniMax M2.1 (上一代)", model: "MiniMax-M2.1", isReasoning: true, supportsTools: true, maxOutputTokens: 16_384, temperature: 1.0},
	// ===== 月之暗面模型 =====
	{id: "builtin-kimi-k2.5", vendorID: "builtin-moonshot", label: "Kimi K2.5 (多模态旗舰)", model: "kimi-k2.5", isMultimodal: true, isReasoning: true, supportsTools: true, maxOutputTokens: 32_768, temperature: 1.0},
	{id: "builtin-kimi-k2", vendorID: "builtin-moonshot", label: "Kimi K2 (1T参数)", model: "kimi-k2", supportsTools: true, maxOutputTokens: 16_384, temperature: 0.7},
	{id: "builtin-kimi-k2-thinking", vendorID: "builtin-moonshot", label: "Kimi K2 Thinking (推理)", model: "kimi-k2-thinking", isReasoning: true, supportsTools: true, maxOutputTokens: 16_384, temperature: 0.7},
	{id: "builtin-kimi-latest", vendorID: "builtin-moonshot", label: "Kimi Latest (自动更新)", model: "kimi-latest", supportsTools: true, maxOutputTokens: 8_192, temperature: 0.7},
	{id: "builtin-moonshot-v1-128k", vendorID: "builtin-moonshot", label: "Moonshot V1 (旧版)", model: "moonshot-v1-128k", supportsTools: true, maxOutputTokens: 8_192, temperature: 0.7},
	// ===== OpenAI 模型 (GPT-5.x 和 o 系列) =====
	// --- GPT-5.5 系列 (当前旗舰) ---
	{id: "builtin-gpt-5.5", vendorID: "builtin-openai", label: "GPT-5.5 (当前旗舰)", model: "gpt-5.5", isMultimodal: true, isReasoning: true, supportsTools: true, maxOutputTokens: 128_000, temperature: 1.0},
	{id: "builtin-gpt-5.5-pro", vendorID: "builtin-openai", label: "GPT-5.5 Pro (高精度)", model: "gpt-5.5-pro", isMultimodal: true, isReasoning: true, supportsTools: true, maxOutputTokens: 128_000, temperature: 1.0},
	// --- GPT-5.4 系列 (当前均衡主力) ---
	{id: "builtin-gpt-5.4", vendorID: "builtin-openai", label: "GPT-5.4 (均衡主力)", model: "gpt-5.4", isMultimodal: true, isReasoning: true, supportsTools: true, maxOutputTokens: 128_000, temperature: 1.0},
	{id: "builtin-gpt-5.4-pro", vendorID: "builtin-openai", label: "GPT-5.4 Pro (高计算)", model: "gpt-5.4-pro", isMultimodal: true, isReasoning: true, supportsTools: true, maxOutputTokens: 128_000, temperature: 1.0},
	{id: "builtin-gpt-5.4-mini", vendorID: "builtin-openai", label: "GPT-5.4 Mini (高性价比)", model: "gpt-5.4-mini", isMultimodal: true, isReasoning: true, supportsTools: true, maxOutputTokens: 128_000, temperature: 1.0},
	{id: "builtin-gpt-5.4-nano", vendorID: "builtin-openai", label: "GPT-5.4 Nano (超低成本)", model: "gpt-5.4-nano", isMultimodal: true, isReasoning: true, supportsTools: true, maxOutputTokens: 128_000, temperature: 1.0},
	// --- GPT-5.2 / 5.1 / 5.0 系列 (官方全部模型页仍列出) ---
	{id: "builtin-gpt-5.2", vendorID: "builtin-openai", label: "GPT-5.2 (上一代旗舰)", model: "gpt-5.2", isMultimodal: true, isReasoning: true, supportsTools: true, maxOutputTokens: 128_000, temperature: 1.0},
	{id: "builtin-gpt-5.2-pro", vendorID: "builtin-openai", label: "GPT-5.2 Pro (上一代高精度)", model: "gpt-5.2-pro", isMultimodal: true, isReasoning: true, supportsTools: true, maxOutputTokens: 128_000, temperature: 1.0},
	// --- GPT-5.1 系列 (Codex 优化) ---
	{id: "builtin-gpt-5.1", vendorID: "builtin-openai", label: "GPT-5.1 (上一代 Coding/Agent)", model: "gpt-5.1", isMultimodal: true, isReasoning: true, supportsTools: true, maxOutputTokens: 128_000, temperature: 1.0},
	// --- GPT-5 系列 (2025年8月发布，400K 上下文) ---
	{id: "builtin-gpt-5", vendorID: "builtin-openai", label: "GPT-5 (基础代)", model: "gpt-5", isMultimodal: true, isReasoning: true, supportsTools: true, maxOutputTokens: 128_000, temperature: 1.0},
	{id: "builtin-gpt-5-pro", vendorID: "builtin-openai", label: "GPT-5 Pro (高精度)", model: "gpt-5-pro", isMultimodal: true, isReasoning: true, supportsTools: true, maxOutputTokens: 128_000, temperature: 1.0},
	{id: "builtin-gpt-5-mini", vendorID: "builtin-openai", label: "GPT-5 Mini (轻量)", model: "gpt-5-mini", isMultimodal: true, isReasoning: true, supportsTools: true, maxOutputTokens: 128_000, temperature: 1.0},
	{id: "builtin-gpt-5-nano", vendorID: "builtin-openai", label: "GPT-5 Nano (经济)", model: "gpt-5-nano", isMultimodal: true, isReasoning: true, supportsTools: true, maxOutputTokens: 128_000, temperature: 1.0},
	// --- o 系列推理模型 ---
	{id: "builtin-o3-pro", vendorID: "builtin-openai", label: "o3-pro (深度推理)", model: "o3-pro", isMultimodal: true, isReasoning: true, supportsTools: true, maxOutputTokens: 100_000, temperature: 1.0},
	{id: "builtin-o3", vendorID: "builtin-openai", label: "o3 (推理)", model: "o3", isMultimodal: true, isReasoning: true, supportsTools: true, maxOutputTokens: 100_000, temperature: 1.0},
	{id: "builtin-o3-mini", vendorID: "builtin-openai", label: "o3-mini (推理轻量)", model: "o3-mini", isMultimodal: true, isReasoning: true, supportsTools: true, maxOutputTokens: 100_000, temperature: 1.0},
	{id: "builtin-o4-mini", vendorID: "builtin-openai", label: "o4-mini (最新推理)", model: "o4-mini", isMultimodal: true, isReasoning: true, supportsTools: true, maxOutputTokens: 100_000, temperature: 1.0},
	// ===== NVIDIA NIM 模型 =====
	{id: "builtin-nvidia-nemotron-3-nano", vendorID: "builtin-nvidia", label: "NVIDIA Nemotron 3 Nano", model: "nvidia/nemotron-3-nano-30b-a3b", isReasoning: true, supportsTools: true, maxOutputTokens: 8_192, temperature: 0.7},
	{id: "builtin-nvidia-llama-3.1-405b", vendorID: "builtin-nvidia", label: "Llama 3.1 405B Instruct", model: "meta/llama-3.1-405b-instruct", maxOutputTokens: 8_192, temperature: 0.7},
	{id: "builtin-nvidia-yi-large", vendorID: "builtin-nvidia", label: "Yi Large", model: "01-ai/yi-large", maxOutputTokens: 8_192, temperature: 0.7},
	// ===== Xiaomi MiMo 模型 =====
	{id: "builtin-mimo-v2.5-pro", vendorID: "builtin-mimo", label: "MiMo V2.5 Pro", model: "mimo-v2.5-pro", isReasoning: true, supportsTools: true, maxOutputTokens: 131_072, temperature: 1.0},
	{id: "builtin-mimo-v2.5", vendorID: "builtin-mimo", label: "MiMo V2.5", model: "mimo-v2.5", isMultimodal: true, isReasoning: true, supportsTools: true, maxOutputTokens: 32_768, temperature: 1.0},
	{id: "builtin-mimo-v2-pro", vendorID: "builtin-mimo", label: "MiMo V2 Pro", model: "mimo-v2-pro", isReasoning: true, supportsTools: true, maxOutputTokens: 131_072, temperature: 1.0},
	{id: "builtin-mimo-v2-omni", vendorID: "builtin-mimo", label: "MiMo V2 Omni", model: "mimo-v2-omni", isMultimodal: true, isReasoning: true, supportsTools: true, maxOutputTokens: 32_768, temperature: 1.0},
	{id: "builtin-mimo-v2-flash", vendorID: "builtin-mimo", label: "MiMo V2 Flash", model: "mimo-v2-flash", isReasoning: true, supportsTools: true, maxOutputTokens: 65_536, temperature: 0.3},
}

// toVendorConfig 将内置供应商定义转换为 VendorConfig（对应 Rust BuiltinVendor::to_vendor_config）。
func (v builtinVendor) toVendorConfig() VendorConfig {
	protocol := resolvePreferredProtocol(v.providerType, v.baseURL, nil)
	supportsResponses := providerSupportsOpenAIResponses(v.providerType, v.baseURL, nil)
	var website *string
	if v.websiteURL != "" {
		website = strPtr(v.websiteURL)
	}
	return VendorConfig{
		ID:                      v.id,
		Name:                    v.name,
		ProviderType:            v.providerType,
		APIProtocol:             strPtr(protocol),
		SupportsOpenAIResponses: boolPtr(supportsResponses),
		BaseURL:                 v.baseURL,
		APIKey:                  "",
		Headers:                 map[string]string{},
		Notes:                   strPtr(v.notes),
		IsBuiltin:               true,
		IsReadOnly:              false, // 允许用户编辑（主要是填 Key）
		MaxTokensLimit:          v.maxTokensLimit,
		WebsiteURL:              website,
	}
}

// getBuiltinVendorMaxTokensLimit 根据供应商 ID 查找其 max_tokens_limit。
func getBuiltinVendorMaxTokensLimit(vendorID string) *uint32 {
	for _, v := range builtinVendors {
		if v.id == vendorID {
			return v.maxTokensLimit
		}
	}
	return nil
}

// getBuiltinVendorProviderType 根据供应商 ID 查找其 provider_type，默认 "openai"。
func getBuiltinVendorProviderType(vendorID string) string {
	for _, v := range builtinVendors {
		if v.id == vendorID {
			return v.providerType
		}
	}
	return "openai"
}

// toModelProfile 将内置模型定义转换为 ModelProfile（对应 Rust BuiltinModel::to_model_profile）。
//
// 关键规则（与 Rust 一致）：
//   - modelAdapter：gemini→google，deepseek→deepseek，nvidia→general，mimo→mimo，其他→general
//   - reasoningEffort：DeepSeek 推理模型为 "high"；OpenAI 推理模型按 model 分级
//   - verbosity：OpenAI 推理模型：nano 为 "low"，其他 "medium"
//   - thinkingEnabled / includeThoughts：推理模型默认 true，但 NVIDIA 全部为 false，
//     MiMo 的 v2-flash 和 v2.5-flash 为 false
//   - contextWindow：DeepSeek v4/chat/reasoner 1_000_000；v3.2/v3.1 128_000；
//     NVIDIA nemotron 1_000_000；MiMo v2.5-pro/v2-pro/v2.5 1_000_000，
//     v2-flash/v2.5-flash/v2-omni 256_000
func (m builtinModel) toModelProfile() ModelProfile {
	maxTokensLimit := getBuiltinVendorMaxTokensLimit(m.vendorID)
	providerScope := getBuiltinVendorProviderType(m.vendorID)

	// 根据供应商确定 model_adapter 与 gemini_api_version
	var modelAdapter string
	var geminiAPIVersion *string
	switch m.vendorID {
	case "builtin-gemini":
		modelAdapter = "google"
		geminiAPIVersion = strPtr("v1beta")
	case "builtin-deepseek":
		modelAdapter = "deepseek"
	case "builtin-nvidia":
		modelAdapter = "general"
	case "builtin-mimo":
		modelAdapter = "mimo"
	default:
		modelAdapter = "general"
	}

	// reasoningEffort 规则
	var reasoningEffort *string
	if m.vendorID == "builtin-deepseek" && m.isReasoning {
		reasoningEffort = strPtr("high")
	} else if m.vendorID == "builtin-openai" && m.isReasoning {
		var effort string
		switch m.model {
		case "gpt-5.5-pro", "gpt-5.4-pro", "gpt-5.2-pro", "gpt-5-pro", "o3-pro":
			effort = "high"
		case "gpt-5.4-nano":
			effort = "low"
		default:
			effort = "medium"
		}
		reasoningEffort = strPtr(effort)
	}

	// verbosity 规则（仅 OpenAI 推理模型）
	var verbosity *string
	if m.vendorID == "builtin-openai" && m.isReasoning {
		if m.model == "gpt-5.4-nano" {
			verbosity = strPtr("low")
		} else {
			verbosity = strPtr("medium")
		}
	}

	// thinkingEnabled / includeThoughts 规则
	useReasoningDefaults := m.isReasoning &&
		m.vendorID != "builtin-nvidia" &&
		!(m.vendorID == "builtin-mimo" && (m.model == "mimo-v2-flash" || m.model == "mimo-v2.5-flash"))

	return ModelProfile{
		ID:                m.id,
		VendorID:          m.vendorID,
		Label:             m.label,
		Model:             m.model,
		ProviderScope:     strPtr(providerScope),
		APIProtocol:       nil,
		ModelAdapter:      modelAdapter,
		IsMultimodal:      m.isMultimodal,
		IsReasoning:       m.isReasoning,
		IsEmbedding:       false,
		IsReranker:        false,
		IsImageGeneration: false,
		SupportsTools:     m.supportsTools,
		SupportsReasoning: m.isReasoning,
		Status:            defaultProfileStatus,
		Enabled:           defaultProfileEnabled,
		MaxOutputTokens:   m.maxOutputTokens,
		Temperature:       m.temperature,
		ReasoningEffort:   reasoningEffort,
		ThinkingEnabled:   useReasoningDefaults,
		IncludeThoughts:   useReasoningDefaults,
		GeminiAPIVersion:  geminiAPIVersion,
		IsBuiltin:         false, // 允许用户编辑和删除模型配置
		IsFavorite:        false,
		MaxTokensLimit:    maxTokensLimit,
		ContextWindow:     deepseekContextWindow(m.model),
		Verbosity:         verbosity,
	}
}

// deepseekContextWindow 推断模型上下文窗口大小（对应 Rust deepseek_context_window）。
//
// 规则：
//   - deepseek-v4 / deepseek-chat / deepseek-reasoner → 1_000_000
//   - deepseek-v3.2 / deepseek-v3.1 → 128_000
//   - nemotron-3-nano / nemotron-3-super / nemotron-3-ultra → 1_000_000
//   - mimo-v2.5-pro / mimo-v2-pro / mimo-v2.5 → 1_000_000
//   - mimo-v2-flash / mimo-v2.5-flash / mimo-v2-omni → 256_000
func deepseekContextWindow(model string) *uint32 {
	normalized := strings.ToLower(strings.TrimSpace(model))
	switch {
	case strings.Contains(normalized, "deepseek-v4"),
		normalized == "deepseek-chat",
		normalized == "deepseek-reasoner":
		return u32Ptr(1_000_000)
	case strings.Contains(normalized, "deepseek-v3.2"),
		strings.Contains(normalized, "deepseek-v3.1"):
		return u32Ptr(128_000)
	case strings.Contains(normalized, "nemotron-3-nano"),
		strings.Contains(normalized, "nemotron-3-super"),
		strings.Contains(normalized, "nemotron-3-ultra"):
		return u32Ptr(1_000_000)
	case normalized == "mimo-v2.5-pro",
		normalized == "mimo-v2-pro",
		normalized == "mimo-v2.5":
		return u32Ptr(1_000_000)
	case normalized == "mimo-v2-flash",
		normalized == "mimo-v2.5-flash",
		normalized == "mimo-v2-omni":
		return u32Ptr(256_000)
	default:
		return nil
	}
}

// loadBuiltinVendors 加载所有内置供应商（不包含已存在的）。
//
// 对应 Rust load_builtin_vendors：仅返回 existingIDs 中不存在的项。
func loadBuiltinVendors(existingIDs []string) []VendorConfig {
	exist := make(map[string]struct{}, len(existingIDs))
	for _, id := range existingIDs {
		exist[id] = struct{}{}
	}
	out := make([]VendorConfig, 0, len(builtinVendors))
	for _, v := range builtinVendors {
		if _, ok := exist[v.id]; ok {
			continue
		}
		out = append(out, v.toVendorConfig())
	}
	return out
}

// loadBuiltinModels 加载所有内置模型（不包含已存在的）。
//
// 对应 Rust load_builtin_models：仅返回 existingIDs 中不存在的项。
func loadBuiltinModels(existingIDs []string) []ModelProfile {
	exist := make(map[string]struct{}, len(existingIDs))
	for _, id := range existingIDs {
		exist[id] = struct{}{}
	}
	out := make([]ModelProfile, 0, len(builtinModels))
	for _, m := range builtinModels {
		if _, ok := exist[m.id]; ok {
			continue
		}
		out = append(out, m.toModelProfile())
	}
	return out
}
