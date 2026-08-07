// Research Store —— 深度调研状态管理
// ------------------------------------------------------------
// 对接后端 Wails 绑定：
// - ResearchPlan(topic, depth, format) —— 生成调研计划
// - ResearchRun(topic, engines) —— 执行多引擎聚合调研
// - ResearchSave(r) —— 保存调研报告到 Hub

import { create } from "zustand";
import { callWails } from "@/lib/wails";

/** 调研深度 */
export type ResearchDepth = "quick" | "normal" | "deep";

/** 报告格式 */
export type ResearchFormat = "markdown" | "report" | "brief";

/** 报告章节 —— 与后端 research.ReportSection 对齐 */
export interface ReportSection {
  title: string;
  content: string;
}

/** 搜索结果 —— 与后端 research.SearchResult 对齐 */
export interface SearchResult {
  engine: string;
  title: string;
  url: string;
  snippet: string;
}

/** 调研计划 —— 与后端 research.Plan 对齐 */
export interface Plan {
  steps: string[];
}

/** 调研报告 —— 与后端 research.Report 对齐 */
export interface Report {
  id: string;
  topic: string;
  sections: ReportSection[];
  sources: SearchResult[];
  created_at?: unknown;
}

/** 可用搜索引擎 */
export const AVAILABLE_ENGINES: string[] = [
  "google",
  "serpapi",
  "tavily",
  "brave",
  "searxng",
  "zhipu",
  "bocha",
];

/** 深度元信息 */
export const DEPTH_META: Record<ResearchDepth, string> = {
  quick: "快速",
  normal: "标准",
  deep: "深度",
};

/** 格式元信息 */
export const FORMAT_META: Record<ResearchFormat, string> = {
  markdown: "Markdown",
  report: "报告",
  brief: "简报",
};

interface ResearchState {
  /** 调研主题 */
  topic: string;
  /** 调研深度 */
  depth: ResearchDepth;
  /** 报告格式 */
  format: ResearchFormat;
  /** 选中引擎列表 */
  engines: string[];
  /** 生成的计划 */
  plan: Plan | null;
  /** 调研报告 */
  report: Report | null;
  /** 当前选中的章节索引 */
  activeSectionIdx: number;
  /** 加载状态：生成计划中 */
  planning: boolean;
  /** 加载状态：执行调研中 */
  running: boolean;
  /** 加载状态：保存中 */
  saving: boolean;
  /** 错误信息 */
  error: string | null;
  /** 已保存的 URI */
  savedUri: string | null;

  // —— Actions ——
  setTopic: (t: string) => void;
  setDepth: (d: ResearchDepth) => void;
  setFormat: (f: ResearchFormat) => void;
  toggleEngine: (e: string) => void;
  generatePlan: () => Promise<void>;
  run: () => Promise<void>;
  save: () => Promise<string | null>;
  selectSection: (idx: number) => void;
  reset: () => void;
}

export const useResearchStore = create<ResearchState>((set, get) => ({
  topic: "",
  depth: "normal",
  format: "report",
  engines: ["google", "zhipu"],
  plan: null,
  report: null,
  activeSectionIdx: 0,
  planning: false,
  running: false,
  saving: false,
  error: null,
  savedUri: null,

  setTopic: (t) => set({ topic: t, savedUri: null }),
  setDepth: (d) => set({ depth: d }),
  setFormat: (f) => set({ format: f }),
  toggleEngine: (e) =>
    set((s) => ({
      engines: s.engines.includes(e)
        ? s.engines.filter((x) => x !== e)
        : [...s.engines, e],
    })),

  // —— 生成计划 ——
  generatePlan: async () => {
    const { topic, depth, format } = get();
    if (!topic.trim()) {
      set({ error: "请输入调研主题" });
      return;
    }
    set({ planning: true, error: null, plan: null });
    try {
      const p = await callWails<Plan>(
        "ResearchPlan",
        topic,
        depth,
        format
      );
      if (!p) {
        set({ error: "[后端未连接] 计划生成不可用" });
        return;
      }
      set({
        plan: { steps: p.steps ?? [] },
      });
    } catch (err) {
      set({ error: err instanceof Error ? err.message : String(err) });
    } finally {
      set({ planning: false });
    }
  },

  // —— 执行调研 ——
  run: async () => {
    const { topic, engines } = get();
    if (!topic.trim()) {
      set({ error: "请输入调研主题" });
      return;
    }
    if (engines.length === 0) {
      set({ error: "至少选择一个搜索引擎" });
      return;
    }
    set({
      running: true,
      error: null,
      report: null,
      activeSectionIdx: 0,
      savedUri: null,
    });
    try {
      const r = await callWails<Report>("ResearchRun", topic, engines);
      if (!r) {
        set({ error: "[后端未连接] 调研不可用" });
        return;
      }
      const safe: Report = {
        id: r.id || `report_${Date.now()}`,
        topic: r.topic ?? topic,
        sections: r.sections ?? [],
        sources: r.sources ?? [],
        created_at: r.created_at,
      };
      set({ report: safe, activeSectionIdx: 0 });
    } catch (err) {
      set({ error: err instanceof Error ? err.message : String(err) });
    } finally {
      set({ running: false });
    }
  },

  // —— 保存报告 ——
  save: async () => {
    const { report } = get();
    if (!report) return null;
    set({ saving: true, error: null });
    try {
      const uri = await callWails<string>("ResearchSave", report);
      if (uri) {
        set({ savedUri: uri });
      } else {
        set({ error: "[后端未连接] 保存不可用" });
      }
      return uri;
    } catch (err) {
      set({ error: err instanceof Error ? err.message : String(err) });
      return null;
    } finally {
      set({ saving: false });
    }
  },

  // —— 选中章节 ——
  selectSection: (idx) =>
    set((s) => ({
      activeSectionIdx: Math.max(
        0,
        Math.min(idx, Math.max(0, (s.report?.sections.length ?? 1) - 1))
      ),
    })),

  // —— 重置 ——
  reset: () =>
    set({
      topic: "",
      plan: null,
      report: null,
      activeSectionIdx: 0,
      savedUri: null,
      error: null,
    }),
}));
