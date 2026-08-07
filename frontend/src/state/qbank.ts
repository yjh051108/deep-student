// QBank Store —— 题库与练习状态管理
// ------------------------------------------------------------
// 对接后端 Wails 绑定：
// - QBankExtract(uri, title) —— 从文档抽题生成题集
// - QBankSave(set) —— 保存题集
// - QBankStartAttempt(setID) —— 开始一次答题
// - QBankAnswer(attemptID, qID, ans) —— 提交单题答案
// - QBankSubmit(attemptID) —— 提交整卷评分
// - QBankAnalyze(setID, qID) —— 单题 AI 解析
// - QBankMastery() —— 知识点掌握度映射

import { create } from "zustand";
import { callWails } from "@/lib/wails";

/** 题目类型：单选 / 多选 / 填空 / 简答 */
export type QuestionType = "single" | "multi" | "fill" | "essay";

/** 题目结构 —— 与后端 qbank.Question 对齐 */
export interface Question {
  id: string;
  stem: string;
  options?: string[];
  answer: string;
  type: string;
  points?: string[];
  knowledge?: string[];
}

/** 题集结构 —— 与后端 qbank.Set 对齐 */
export interface QBSet {
  id: string;
  title: string;
  questions: Question[];
  created_at?: unknown;
}

/** 答题记录 —— 与后端 qbank.Attempt 对齐 */
export interface Attempt {
  id: string;
  set_id: string;
  answers: Record<string, string>;
  score: number;
  total: number;
  started_at?: unknown;
  finished_at?: unknown;
}

interface QBankState {
  /** 已保存的题集列表 */
  sets: QBSet[];
  /** 当前选中的题集 ID */
  activeSetId: string | null;
  /** 当前题集对象（来自 sets 缓存或最近 Extract 的结果） */
  activeSet: QBSet | null;
  /** 当前答题 attempt */
  attempt: Attempt | null;
  /** 当前题目索引（0-based） */
  currentIndex: number;
  /** 用户作答草稿（qID -> 答案字符串） */
  draftAnswers: Record<string, string>;
  /** 知识点掌握度映射 */
  mastery: Record<string, number>;
  /** 单题 AI 解析缓存（qID -> 解析文本） */
  analysisMap: Record<string, string>;
  /** 加载状态：通用 */
  loading: boolean;
  /** 加载状态：抽题 */
  extracting: boolean;
  /** 加载状态：提交评分 */
  submitting: boolean;
  /** 加载状态：AI 解析（按 qID 索引） */
  analyzingQid: string | null;
  /** 错误信息 */
  error: string | null;

  // —— Actions ——
  extract: (uri: string, title: string) => Promise<QBSet | null>;
  save: (set: QBSet) => Promise<string | null>;
  selectSet: (id: string) => void;
  startAttempt: () => Promise<void>;
  answer: (qID: string, ans: string) => Promise<void>;
  setDraft: (qID: string, ans: string) => void;
  submit: () => Promise<void>;
  analyze: (qID: string) => Promise<void>;
  loadMastery: () => Promise<void>;
  next: () => void;
  prev: () => void;
  jumpTo: (idx: number) => void;
  reset: () => void;
}

export const useQBankStore = create<QBankState>((set, get) => ({
  sets: [],
  activeSetId: null,
  activeSet: null,
  attempt: null,
  currentIndex: 0,
  draftAnswers: {},
  mastery: {},
  analysisMap: {},
  loading: false,
  extracting: false,
  submitting: false,
  analyzingQid: null,
  error: null,

  // —— 从 URI 抽题 ——
  extract: async (uri, title) => {
    set({ extracting: true, error: null });
    try {
      const result = await callWails<QBSet>("QBankExtract", uri, title);
      if (!result) {
        set({ error: "[后端未连接] 抽题不可用" });
        return null;
      }
      // 抽题成功后立即设为当前题集（用户可选择保存）
      set({
        activeSet: result,
        activeSetId: result.id,
        attempt: null,
        currentIndex: 0,
        draftAnswers: {},
        analysisMap: {},
      });
      return result;
    } catch (err) {
      set({ error: err instanceof Error ? err.message : String(err) });
      return null;
    } finally {
      set({ extracting: false });
    }
  },

  // —— 保存题集 ——
  save: async (qbSet) => {
    set({ loading: true, error: null });
    try {
      const uri = await callWails<string>("QBankSave", qbSet);
      if (uri) {
        // 保存成功后追加到本地列表（去重）
        set((s) => ({
          sets: s.sets.some((x) => x.id === qbSet.id)
            ? s.sets.map((x) => (x.id === qbSet.id ? qbSet : x))
            : [...s.sets, qbSet],
        }));
      }
      return uri;
    } catch (err) {
      set({ error: err instanceof Error ? err.message : String(err) });
      return null;
    } finally {
      set({ loading: false });
    }
  },

  // —— 选中题集 ——
  selectSet: (id) => {
    const set_ = get().sets.find((s) => s.id === id) ?? null;
    set({
          activeSetId: id,
          activeSet: set_,
          attempt: null,
          currentIndex: 0,
          draftAnswers: {},
          analysisMap: {},
        });
  },

  // —— 开始一次答题 ——
  startAttempt: async () => {
    const { activeSetId } = get();
    if (!activeSetId) {
      set({ error: "未选中题集" });
      return;
    }
    set({ loading: true, error: null });
    try {
      const att = await callWails<Attempt>("QBankStartAttempt", activeSetId);
      if (!att) {
        set({ error: "[后端未连接] 无法开始答题" });
        return;
      }
      set({
        attempt: att,
        currentIndex: 0,
        draftAnswers: { ...att.answers },
      });
    } catch (err) {
      set({ error: err instanceof Error ? err.message : String(err) });
    } finally {
      set({ loading: false });
    }
  },

  // —— 草稿作答（不立即提交，仅本地保存） ——
  setDraft: (qID, ans) => {
    set((s) => ({ draftAnswers: { ...s.draftAnswers, [qID]: ans } }));
  },

  // —— 提交单题答案到后端 ——
  answer: async (qID, ans) => {
    const { attempt } = get();
    if (!attempt) return;
    // 先更新本地草稿
    set((s) => ({ draftAnswers: { ...s.draftAnswers, [qID]: ans } }));
    // 调用后端持久化（失败不阻塞，最终以 submit 为准）
    await callWails<void>("QBankAnswer", attempt.id, qID, ans);
  },

  // —— 提交整卷 ——
  submit: async () => {
    const { attempt, draftAnswers } = get();
    if (!attempt) return;
    set({ submitting: true, error: null });
    try {
      // 先把所有草稿答案同步到后端
      for (const [qID, ans] of Object.entries(draftAnswers)) {
        await callWails<void>("QBankAnswer", attempt.id, qID, ans);
      }
      const finalAttempt = await callWails<Attempt>(
        "QBankSubmit",
        attempt.id
      );
      if (finalAttempt) {
        set({
          attempt: { ...finalAttempt, answers: { ...draftAnswers } },
        });
      } else {
        set({ error: "[后端未连接] 提交评分不可用" });
      }
    } catch (err) {
      set({ error: err instanceof Error ? err.message : String(err) });
    } finally {
      set({ submitting: false });
    }
  },

  // —— 单题 AI 解析 ——
  analyze: async (qID) => {
    const { activeSetId } = get();
    if (!activeSetId) return;
    set({ analyzingQid: qID, error: null });
    try {
      const text = await callWails<string>(
        "QBankAnalyze",
        activeSetId,
        qID
      );
      set((s) => ({
        analysisMap: {
          ...s.analysisMap,
          [qID]: text ?? "[后端未连接] 解析不可用",
        },
      }));
    } catch (err) {
      set((s) => ({
        analysisMap: {
          ...s.analysisMap,
          [qID]: "[错误] " + (err instanceof Error ? err.message : String(err)),
        },
      }));
    } finally {
      set({ analyzingQid: null });
    }
  },

  // —— 加载知识点掌握度 ——
  loadMastery: async () => {
    set({ loading: true, error: null });
    try {
      const m = await callWails<Record<string, number>>("QBankMastery");
      set({ mastery: m ?? {} });
    } catch (err) {
      set({ error: err instanceof Error ? err.message : String(err) });
    } finally {
      set({ loading: false });
    }
  },

  // —— 题目导航 ——
  next: () =>
    set((s) => ({
      currentIndex: Math.min(
        s.currentIndex + 1,
        Math.max(0, (s.activeSet?.questions.length ?? 1) - 1)
      ),
    })),
  prev: () => set((s) => ({ currentIndex: Math.max(0, s.currentIndex - 1) })),
  jumpTo: (idx) =>
    set((s) => ({
      currentIndex: Math.max(
        0,
        Math.min(idx, Math.max(0, (s.activeSet?.questions.length ?? 1) - 1))
      ),
    })),

  // —— 重置答题 ——
  reset: () =>
    set({
      attempt: null,
      currentIndex: 0,
      draftAnswers: {},
      analysisMap: {},
    }),
}));
