// LLM 用量 Store —— 用量统计状态管理
// ------------------------------------------------------------
// 对接后端 llmUsageApi（LLMUsageXxx 方法）。
// 支持：总览卡片 / 最近日志 / 按日聚合表 / 过滤 / 清理。

import { create } from "zustand";
import {
  llmUsageApi,
  type UsageLog,
  type DailyAggregate,
  type UsageSummary,
  type LogFilter,
} from "@/lib/llmUsage";

interface LLMUsageState {
  summary: UsageSummary | null;
  logs: UsageLog[];
  daily: DailyAggregate[];
  filter: LogFilter;
  loading: boolean;
  error: string | null;
  toast: string | null;

  loadAll: () => Promise<void>;
  loadSummary: () => Promise<void>;
  loadLogs: (filter?: LogFilter) => Promise<void>;
  loadDaily: () => Promise<void>;
  setFilter: (f: Partial<LogFilter>) => void;
  cleanup: (beforeDays: number) => Promise<void>;
  clearToast: () => void;
}

export const useLLMUsageStore = create<LLMUsageState>((set, get) => ({
  summary: null,
  logs: [],
  daily: [],
  filter: { limit: 100 },
  loading: false,
  error: null,
  toast: null,

  loadAll: async () => {
    set({ loading: true, error: null });
    try {
      const [summary, logs, daily] = await Promise.all([
        llmUsageApi.summary(),
        llmUsageApi.query(get().filter),
        llmUsageApi.queryDaily({}),
      ]);
      set({
        summary: summary ?? null,
        logs: logs ?? [],
        daily: daily ?? [],
        loading: false,
      });
    } catch (err) {
      set({ error: String(err), loading: false });
    }
  },

  loadSummary: async () => {
    const summary = await llmUsageApi.summary();
    if (summary) set({ summary });
  },

  loadLogs: async (filter) => {
    const f = filter ?? get().filter;
    const logs = await llmUsageApi.query(f);
    if (logs) set({ logs });
  },

  loadDaily: async () => {
    const daily = await llmUsageApi.queryDaily({});
    if (daily) set({ daily });
  },

  setFilter: (f) => {
    set({ filter: { ...get().filter, ...f } });
    void get().loadLogs();
  },

  cleanup: async (beforeDays) => {
    const before = new Date(Date.now() - beforeDays * 86400000).toISOString();
    const n = await llmUsageApi.cleanup(before);
    if (n !== null) {
      set({ toast: `已清理 ${n} 条日志` });
      await get().loadAll();
    }
  },

  clearToast: () => set({ toast: null }),
}));
