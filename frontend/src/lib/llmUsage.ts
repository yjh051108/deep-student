// LLM 用量前端类型与 Wails 封装
// ------------------------------------------------------------
// 与后端 internal/llmusage/types.go 对齐。

import { callWails } from "@/lib/wails";

/** 单次调用日志 —— 与后端 llmusage.Log 对齐 */
export interface UsageLog {
  id: string;
  timestamp: string;
  provider: string;
  model: string;
  adapter?: string;
  apiConfigId?: string;
  promptTokens: number;
  completionTokens: number;
  totalTokens: number;
  reasoningTokens?: number | null;
  cachedTokens?: number | null;
  tokenSource: string;
  durationMs?: number | null;
  callerType: string;
  sessionId?: string;
  status: string;
  errorMessage?: string;
  costEstimate?: number | null;
}

/** 按日聚合 —— 与后端 llmusage.DailyAggregate 对齐 */
export interface DailyAggregate {
  date: string;
  callerType: string;
  model: string;
  provider: string;
  requestCount: number;
  successCount: number;
  errorCount: number;
  totalPromptTokens: number;
  totalCompletionTokens: number;
  totalTokens: number;
  totalReasoningTokens: number;
  totalCachedTokens: number;
  totalCostEstimate: number;
  totalDurationMs: number;
}

/** 总览 —— 与后端 llmusage.Summary 对齐 */
export interface UsageSummary {
  totalRequests: number;
  totalTokens: number;
  totalPromptTokens: number;
  totalCompletionTokens: number;
  totalCost: number;
  todayRequests: number;
  todayTokens: number;
  last7DaysRequests: number;
  last7DaysTokens: number;
}

/** 日志过滤 */
export interface LogFilter {
  provider?: string;
  model?: string;
  callerType?: string;
  status?: string;
  since?: string | null;
  until?: string | null;
  limit?: number;
}

/** 按日聚合过滤 */
export interface DailyFilter {
  dateStart?: string;
  dateEnd?: string;
  callerType?: string;
  model?: string;
}

/** 记录调用参数 */
export interface LogEntry {
  provider: string;
  model: string;
  adapter?: string;
  apiConfigId?: string;
  promptTokens?: number;
  completionTokens?: number;
  reasoningTokens?: number | null;
  cachedTokens?: number | null;
  tokenSource?: string;
  durationMs?: number | null;
  callerType: string;
  sessionId?: string;
  status?: string;
  errorMessage?: string;
  costEstimate?: number | null;
}

export const llmUsageApi = {
  record: (e: LogEntry) => callWails<UsageLog>("LLMUsageRecord", e),
  query: (filter: LogFilter) => callWails<UsageLog[]>("LLMUsageQuery", filter),
  queryDaily: (filter: DailyFilter) =>
    callWails<DailyAggregate[]>("LLMUsageQueryDaily", filter),
  summary: () => callWails<UsageSummary>("LLMUsageSummary"),
  cleanup: (before: string) => callWails<number>("LLMUsageCleanup", before),
};
