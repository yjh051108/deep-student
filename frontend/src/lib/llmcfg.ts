// llmcfg —— 前端类型定义 + Wails 调用封装
// ------------------------------------------------------------
// 与 Go 端 internal/llmcfg 的 JSON 字段一一对应（camelCase）。
// 所有调用走 callWails，绑定命名空间为 window.go.deepstudent.App。

import { callWails } from "@/lib/wails";

/** 供应商配置（对齐 Go VendorConfig） */
export interface VendorConfig {
  id: string;
  name: string;
  providerType: string;
  apiProtocol?: string;
  supportsOpenAIResponses?: boolean;
  baseUrl: string;
  apiKey: string;
  headers: Record<string, string>;
  rateLimitPerMinute?: number;
  defaultTimeoutMs?: number;
  notes?: string;
  isBuiltin: boolean;
  isReadOnly: boolean;
  sortOrder?: number;
  maxTokensLimit?: number;
  websiteUrl?: string;
}

/** 模型配置（对齐 Go ModelProfile） */
export interface ModelProfile {
  id: string;
  vendorId: string;
  label: string;
  model: string;
  providerScope?: string;
  apiProtocol?: string;
  modelAdapter: string;
  isMultimodal: boolean;
  isReasoning: boolean;
  isEmbedding: boolean;
  isReranker: boolean;
  isImageGeneration: boolean;
  supportsTools: boolean;
  supportsReasoning: boolean;
  status: string;
  enabled: boolean;
  maxOutputTokens: number;
  temperature: number;
  reasoningEffort?: string;
  thinkingEnabled: boolean;
  thinkingBudget?: number;
  includeThoughts: boolean;
  enableThinking?: boolean;
  minP?: number;
  topK?: number;
  geminiApiVersion?: string;
  isBuiltin: boolean;
  isFavorite: boolean;
  maxTokensLimit?: number;
  contextWindow?: number;
  repetitionPenalty?: number;
  reasoningSplit?: boolean;
  effort?: string;
  verbosity?: string;
}

/** 模型分配（对齐 Go ModelAssignments） */
export interface ModelAssignments {
  model2ConfigId?: string;
  reviewAnalysisModelConfigId?: string;
  ankiCardModelConfigId?: string;
  qbankAiGradingModelConfigId?: string;
  embeddingModelConfigId?: string;
  rerankerModelConfigId?: string;
  chatTitleModelConfigId?: string;
  examSheetOcrModelConfigId?: string;
  translationModelConfigId?: string;
  vlEmbeddingModelConfigId?: string;
  vlRerankerModelConfigId?: string;
  memoryDecisionModelConfigId?: string;
  voiceInputAsrModelConfigId?: string;
  imageGenerationModelConfigId?: string;
  translationDisplayMode?: string;
}

/** 运行时合并配置（对齐 Go ApiConfig） */
export interface ApiConfig {
  id: string;
  name: string;
  vendorId?: string;
  vendorName?: string;
  providerType?: string;
  providerScope?: string;
  apiProtocol?: string;
  supportsOpenAIResponses?: boolean;
  apiKey: string;
  baseUrl: string;
  model: string;
  isMultimodal: boolean;
  isReasoning: boolean;
  isEmbedding: boolean;
  isReranker: boolean;
  isImageGeneration: boolean;
  enabled: boolean;
  modelAdapter: string;
  maxOutputTokens: number;
  temperature: number;
  supportsTools: boolean;
  geminiApiVersion: string;
  isBuiltin: boolean;
  isReadOnly: boolean;
  reasoningEffort?: string;
  thinkingEnabled: boolean;
  thinkingBudget?: number;
  includeThoughts: boolean;
  minP?: number;
  topK?: number;
  enableThinking?: boolean;
  supportsReasoning: boolean;
  headers?: Record<string, string>;
  topPOverride?: number;
  frequencyPenaltyOverride?: number;
  presencePenaltyOverride?: number;
  repetitionPenalty?: number;
  reasoningSplit?: boolean;
  effort?: string;
  verbosity?: string;
  isFavorite: boolean;
  maxTokensLimit?: number;
  contextWindow?: number;
}

/** 测试连接结果 */
export interface TestConnectionResult {
  ok: boolean;
  message: string;
  latencyMs: number;
  model?: string;
  vendorName?: string;
}

// ===== Wails 调用封装 =====

/** 拉取所有供应商 */
export async function fetchVendors(): Promise<VendorConfig[]> {
  const list = await callWails<VendorConfig[]>("LLMCfgGetVendors");
  return list ?? [];
}

/** 保存供应商（upsert），返回是否成功 */
export async function saveVendor(v: VendorConfig): Promise<boolean> {
  const err = await callWails<string | null>("LLMCfgSaveVendor", v);
  return err === null || err === undefined;
}

/** 删除供应商（内置不可删），返回是否成功 */
export async function deleteVendor(id: string): Promise<boolean> {
  const err = await callWails<string | null>("LLMCfgDeleteVendor", id);
  return err === null || err === undefined;
}

/** 拉取所有模型 */
export async function fetchProfiles(): Promise<ModelProfile[]> {
  const list = await callWails<ModelProfile[]>("LLMCfgGetProfiles");
  return list ?? [];
}

/** 按供应商筛选模型 */
export async function fetchProfilesByVendor(
  vendorId: string
): Promise<ModelProfile[]> {
  const list = await callWails<ModelProfile[]>(
    "LLMCfgGetProfilesByVendor",
    vendorId
  );
  return list ?? [];
}

/** 保存模型（upsert），返回是否成功 */
export async function saveProfile(p: ModelProfile): Promise<boolean> {
  const err = await callWails<string | null>("LLMCfgSaveProfile", p);
  return err === null || err === undefined;
}

/** 删除模型（内置不可删），返回是否成功 */
export async function deleteProfile(id: string): Promise<boolean> {
  const err = await callWails<string | null>("LLMCfgDeleteProfile", id);
  return err === null || err === undefined;
}

/** 获取模型分配 */
export async function fetchAssignments(): Promise<ModelAssignments> {
  const a = await callWails<ModelAssignments>("LLMCfgGetAssignments");
  return a ?? {};
}

/** 保存模型分配，返回是否成功 */
export async function saveAssignments(a: ModelAssignments): Promise<boolean> {
  const err = await callWails<string | null>("LLMCfgSaveAssignments", a);
  return err === null || err === undefined;
}

/** 测试连接 */
export async function testConnection(
  profileId: string
): Promise<TestConnectionResult | null> {
  return await callWails<TestConnectionResult>(
    "LLMCfgTestConnection",
    profileId
  );
}

/** 解析为运行时 ApiConfig */
export async function resolveApiConfig(
  profileId: string
): Promise<ApiConfig | null> {
  return await callWails<ApiConfig>("LLMCfgResolveApiConfig", profileId);
}

/** 重新加载内置（用于重置） */
export async function reloadBuiltins(): Promise<boolean> {
  const err = await callWails<string | null>("LLMCfgReloadBuiltins");
  return err === null || err === undefined;
}

// ===== 角色分配 key 枚举 =====

/** 模型角色定义（用于角色分配 Tab 渲染） */
export interface ModelRole {
  key: keyof ModelAssignments;
  label: string;
  hint: string;
}

/** 全部模型角色（顺序与 UI 网格一致） */
export const MODEL_ROLES: ModelRole[] = [
  { key: "model2ConfigId", label: "聊天主力", hint: "默认聊天模型" },
  { key: "reviewAnalysisModelConfigId", label: "回顾分析", hint: "错题回顾分析" },
  { key: "ankiCardModelConfigId", label: "Anki 制卡", hint: "卡片自动生成" },
  { key: "qbankAiGradingModelConfigId", label: "题库批改", hint: "AI 批改/解析" },
  { key: "embeddingModelConfigId", label: "嵌入", hint: "文本向量化" },
  { key: "rerankerModelConfigId", label: "重排序", hint: "检索结果重排" },
  { key: "chatTitleModelConfigId", label: "标题生成", hint: "会话标题" },
  { key: "examSheetOcrModelConfigId", label: "试卷 OCR", hint: "题目集识别" },
  { key: "translationModelConfigId", label: "翻译", hint: "翻译专用" },
  { key: "vlEmbeddingModelConfigId", label: "多模态嵌入", hint: "图文向量化" },
  { key: "vlRerankerModelConfigId", label: "多模态重排序", hint: "图文重排" },
  { key: "memoryDecisionModelConfigId", label: "记忆决策", hint: "去重判断" },
  { key: "voiceInputAsrModelConfigId", label: "语音 ASR", hint: "语音输入" },
  { key: "imageGenerationModelConfigId", label: "生图", hint: "图像生成" },
];
