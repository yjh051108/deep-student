// Reader Store —— 阅读器状态管理
// ------------------------------------------------------------
// 对接后端 Wails 绑定：
// - ReaderOpen(uri) —— 打开文档，返回分页内容
// - ReaderSummarize(uri, page) —— 总结指定页
// - ReaderInject(uri, start, end, sel) —— 选段注入生成

import { create } from "zustand";
import { callWails } from "@/lib/wails";

/** 单页内容 —— 与后端 reader.Page 对齐 */
export interface ReaderPage_ {
  index: number;
  content: string;
}

/** 文档结构 —— 与后端 reader.Document 对齐 */
export interface Document {
  uri: string;
  title: string;
  pages: ReaderPage_[];
}

interface ReaderState {
  /** 当前打开的文档 */
  doc: Document | null;
  /** 当前页码（0-based） */
  currentPageIdx: number;
  /** 当前页总结 */
  summary: string;
  /** 注入结果 */
  injection: string;
  /** 加载状态：打开文档 */
  opening: boolean;
  /** 加载状态：总结中 */
  summarizing: boolean;
  /** 加载状态：注入中 */
  injecting: boolean;
  /** 错误信息 */
  error: string | null;

  // —— Actions ——
  open: (uri: string) => Promise<void>;
  setPage: (idx: number) => void;
  nextPage: () => void;
  prevPage: () => void;
  summarize: () => Promise<void>;
  inject: (start: number, end: number, sel: string) => Promise<void>;
  clearSummary: () => void;
  clearInjection: () => void;
  reset: () => void;
}

export const useReaderStore = create<ReaderState>((set, get) => ({
  doc: null,
  currentPageIdx: 0,
  summary: "",
  injection: "",
  opening: false,
  summarizing: false,
  injecting: false,
  error: null,

  // —— 打开文档 ——
  open: async (uri) => {
    if (!uri.trim()) {
      set({ error: "请输入文档 URI" });
      return;
    }
    set({ opening: true, error: null, summary: "", injection: "" });
    try {
      const d = await callWails<Document>("ReaderOpen", uri);
      if (!d) {
        set({ error: "[后端未连接] 无法打开文档" });
        return;
      }
      const safe: Document = {
        uri: d.uri ?? uri,
        title: d.title ?? uri,
        pages: d.pages ?? [],
      };
      set({
        doc: safe,
        currentPageIdx: 0,
        summary: "",
        injection: "",
      });
    } catch (err) {
      set({ error: err instanceof Error ? err.message : String(err) });
    } finally {
      set({ opening: false });
    }
  },

  // —— 设置当前页 ——
  setPage: (idx) =>
    set((s) => ({
      currentPageIdx: Math.max(
        0,
        Math.min(idx, Math.max(0, (s.doc?.pages.length ?? 1) - 1))
      ),
      summary: "",
    })),

  nextPage: () => {
    const { doc, currentPageIdx } = get();
    if (!doc) return;
    if (currentPageIdx < doc.pages.length - 1) {
      set({ currentPageIdx: currentPageIdx + 1, summary: "" });
    }
  },

  prevPage: () => {
    const { currentPageIdx } = get();
    if (currentPageIdx > 0) {
      set({ currentPageIdx: currentPageIdx - 1, summary: "" });
    }
  },

  // —— 总结当前页 ——
  summarize: async () => {
    const { doc, currentPageIdx } = get();
    if (!doc) return;
    set({ summarizing: true, error: null, summary: "" });
    try {
      const text = await callWails<string>(
        "ReaderSummarize",
        doc.uri,
        currentPageIdx
      );
      set({ summary: text ?? "[后端未连接] 总结不可用" });
    } catch (err) {
      set({
        summary: "[错误] " + (err instanceof Error ? err.message : String(err)),
      });
    } finally {
      set({ summarizing: false });
    }
  },

  // —— 选段注入 ——
  inject: async (start, end, sel) => {
    const { doc } = get();
    if (!doc) {
      set({ error: "未打开文档" });
      return;
    }
    if (!sel.trim()) {
      set({ error: "请输入选段文本" });
      return;
    }
    set({ injecting: true, error: null, injection: "" });
    try {
      const text = await callWails<string>(
        "ReaderInject",
        doc.uri,
        start,
        end,
        sel
      );
      set({ injection: text ?? "[后端未连接] 注入不可用" });
    } catch (err) {
      set({
        injection:
          "[错误] " + (err instanceof Error ? err.message : String(err)),
      });
    } finally {
      set({ injecting: false });
    }
  },

  clearSummary: () => set({ summary: "" }),
  clearInjection: () => set({ injection: "" }),

  // —— 重置 ——
  reset: () =>
    set({
      doc: null,
      currentPageIdx: 0,
      summary: "",
      injection: "",
      error: null,
    }),
}));
