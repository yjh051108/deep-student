// Essay Store —— 作文批改状态管理
// ------------------------------------------------------------
// 对接后端 Wails 绑定：
// - EssayGrade(text, scenario, dims) —— 多维度评分 + 润色 + 建议
// - EssaySave(r) —— 保存批改结果到 Hub

import { create } from "zustand";
import { callWails } from "@/lib/wails";

/** 评分维度结构 —— 与后端 essay.Dimension 对齐 */
export interface Dimension {
  name: string;
  score: number;
  weight: number;
  note: string;
}

/** 批改结果 —— 与后端 essay.Result 对齐 */
export interface EssayResult {
  id: string;
  scenario: string;
  original: string;
  polished: string;
  dimensions: Dimension[];
  total: number;
  suggestions: string[];
  highlights: string[];
}

/** 作文场景 */
export type EssayScenario =
  | "gaokao"
  | "ielts"
  | "toefl"
  | "cet"
  | "postgrad";

/** 评分维度名（与后端约定） */
export type EssayDim =
  | "vocabulary"
  | "grammar"
  | "coherence"
  | "structure"
  | "content";

/** 场景元信息 */
export const SCENARIO_META: Record<EssayScenario, string> = {
  gaokao: "高考",
  ielts: "IELTS 雅思",
  toefl: "TOEFL 托福",
  cet: "CET 大学英语",
  postgrad: "考研",
};

/** 维度元信息 */
export const DIM_META: Record<EssayDim, string> = {
  vocabulary: "词汇",
  grammar: "语法",
  coherence: "连贯性",
  structure: "结构",
  content: "内容",
};

interface EssayState {
  /** 作文文本 */
  text: string;
  /** 当前场景 */
  scenario: EssayScenario;
  /** 选中的评分维度 */
  dims: EssayDim[];
  /** 当前批改结果 */
  result: EssayResult | null;
  /** 历史批改结果（卡片堆叠） */
  history: EssayResult[];
  /** 加载状态：批改中 */
  grading: boolean;
  /** 加载状态：保存中 */
  saving: boolean;
  /** 错误信息 */
  error: string | null;
  /** 已保存的 URI */
  savedUri: string | null;

  // —— Actions ——
  setText: (t: string) => void;
  setScenario: (s: EssayScenario) => void;
  toggleDim: (d: EssayDim) => void;
  grade: () => Promise<void>;
  save: () => Promise<string | null>;
  clear: () => void;
}

export const useEssayStore = create<EssayState>((set, get) => ({
  text: "",
  scenario: "ielts",
  dims: ["vocabulary", "grammar", "coherence", "content"],
  result: null,
  history: [],
  grading: false,
  saving: false,
  error: null,
  savedUri: null,

  setText: (t) => set({ text: t, savedUri: null }),
  setScenario: (s) => set({ scenario: s, savedUri: null }),
  toggleDim: (d) =>
    set((s) => ({
      dims: s.dims.includes(d)
        ? s.dims.filter((x) => x !== d)
        : [...s.dims, d],
      savedUri: null,
    })),

  // —— 批改 ——
  grade: async () => {
    const { text, scenario, dims } = get();
    if (!text.trim()) {
      set({ error: "请输入作文内容" });
      return;
    }
    if (dims.length === 0) {
      set({ error: "至少选择一个评分维度" });
      return;
    }
    set({ grading: true, error: null, savedUri: null });
    try {
      const r = await callWails<EssayResult>(
        "EssayGrade",
        text,
        scenario,
        dims
      );
      if (!r) {
        set({ error: "[后端未连接] 批改不可用" });
        return;
      }
      // 补全后端可能省略的字段
      const safe: EssayResult = {
        id: r.id || `essay_${Date.now()}`,
        scenario: r.scenario || scenario,
        original: r.original ?? text,
        polished: r.polished ?? "",
        dimensions: r.dimensions ?? [],
        total: r.total ?? 0,
        suggestions: r.suggestions ?? [],
        highlights: r.highlights ?? [],
      };
      set((s) => ({
        result: safe,
        history: [safe, ...s.history].slice(0, 20),
      }));
    } catch (err) {
      set({ error: err instanceof Error ? err.message : String(err) });
    } finally {
      set({ grading: false });
    }
  },

  // —— 保存到 Hub ——
  save: async () => {
    const { result } = get();
    if (!result) return null;
    set({ saving: true, error: null });
    try {
      const uri = await callWails<string>("EssaySave", result);
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

  // —— 清空 ——
  clear: () =>
    set({
      text: "",
      result: null,
      savedUri: null,
      error: null,
    }),
}));
