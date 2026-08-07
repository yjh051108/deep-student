// Paper Store —— 论文检索状态管理
// ------------------------------------------------------------
// 对接后端 Wails 绑定：
// - PaperSearchArXiv(q, max) / PaperSearchOpenAlex(q, max)
// - PaperDownload(src) → uri / PaperCite(src, format) → string
// - PaperResolveDOI(doi) → string
//
// 设计要点：
// - 搜索框 + 引擎切换（arXiv / OpenAlex）+ 结果数
// - 结果列表展开后显示摘要与操作按钮
// - 引用格式：bibtex / gbt7714 / apa

import { create } from "zustand";
import { callWails } from "@/lib/wails";

/** 论文元数据 —— 与后端 paper.Source 对齐 */
export interface PaperSource {
  id: string;
  title: string;
  authors: string[];
  abstract: string;
  year: number;
  venue: string;
  url: string;
  pdf_url: string;
  doi: string;
  source: string; // arxiv | openalex
  created_at: string;
}

export type PaperEngine = "arxiv" | "openalex";
export type CitationFormat = "bibtex" | "gbt7714" | "apa";

interface PaperState {
  /** 搜索关键词 */
  query: string;
  /** 搜索引擎 */
  engine: PaperEngine;
  /** 最大结果数 */
  maxResults: number;
  /** 搜索结果 */
  results: PaperSource[];
  /** 当前展开的论文 ID */
  expandedId: string | null;
  /** 引用格式 */
  citationFormat: CitationFormat;
  /** 引用文本缓存（key = paperId:format） */
  citations: Record<string, string>;
  /** 下载后的 URI 缓存（key = paperId） */
  downloadUris: Record<string, string>;
  /** 搜索状态 */
  loading: boolean;
  /** 操作进行中（下载 / 引用 / DOI）的论文 ID */
  busyId: string | null;
  /** 错误信息 */
  error: string | null;

  // —— Actions ——
  setQuery: (q: string) => void;
  setEngine: (e: PaperEngine) => void;
  setMaxResults: (n: number) => void;
  setCitationFormat: (f: CitationFormat) => void;
  toggleExpand: (id: string) => void;
  search: () => Promise<void>;
  download: (src: PaperSource) => Promise<void>;
  cite: (src: PaperSource) => Promise<void>;
  resolveDOI: (doi: string) => Promise<void>;
}

export const usePaperStore = create<PaperState>((set, get) => ({
  query: "",
  engine: "arxiv",
  maxResults: 10,
  results: [],
  expandedId: null,
  citationFormat: "bibtex",
  citations: {},
  downloadUris: {},
  loading: false,
  busyId: null,
  error: null,

  setQuery: (q) => set({ query: q }),
  setEngine: (e) => set({ engine: e }),
  setMaxResults: (n) => set({ maxResults: n }),
  setCitationFormat: (f) => set({ citationFormat: f }),

  toggleExpand: (id) =>
    set((s) => ({
      expandedId: s.expandedId === id ? null : id,
    })),

  // —— 搜索 ——
  search: async () => {
    const { query, engine, maxResults } = get();
    if (!query.trim()) {
      set({ error: "请输入搜索关键词" });
      return;
    }
    set({ loading: true, error: null, expandedId: null });
    try {
      const method =
        engine === "arxiv" ? "PaperSearchArXiv" : "PaperSearchOpenAlex";
      const list = await callWails<PaperSource[]>(method, query.trim(), maxResults);
      set({ results: list ?? [] });
    } catch (err) {
      set({ error: err instanceof Error ? err.message : String(err) });
    } finally {
      set({ loading: false });
    }
  },

  // —— 下载 PDF ——
  download: async (src) => {
    set({ busyId: src.id, error: null });
    try {
      const uri = await callWails<string>("PaperDownload", src);
      if (!uri) {
        set({ error: "[后端未连接] 下载不可用" });
        return;
      }
      set((s) => ({ downloadUris: { ...s.downloadUris, [src.id]: uri } }));
    } catch (err) {
      set({ error: err instanceof Error ? err.message : String(err) });
    } finally {
      set({ busyId: null });
    }
  },

  // —— 生成引用 ——
  cite: async (src) => {
    const { citationFormat } = get();
    const cacheKey = `${src.id}:${citationFormat}`;
    set({ busyId: src.id, error: null });
    try {
      const text = await callWails<string>("PaperCite", src, citationFormat);
      if (text === null) {
        set({ error: "[后端未连接] 引用生成不可用" });
        return;
      }
      set((s) => ({ citations: { ...s.citations, [cacheKey]: text } }));
    } catch (err) {
      set({ error: err instanceof Error ? err.message : String(err) });
    } finally {
      set({ busyId: null });
    }
  },

  // —— 解析 DOI ——
  resolveDOI: async (doi) => {
    if (!doi.trim()) {
      set({ error: "请输入 DOI" });
      return;
    }
    set({ busyId: "doi", error: null });
    try {
      const url = await callWails<string>("PaperResolveDOI", doi.trim());
      if (!url) {
        set({ error: "[后端未连接] DOI 解析不可用" });
        return;
      }
      // 在新窗口打开解析后的链接
      window.open(url, "_blank");
    } catch (err) {
      set({ error: err instanceof Error ? err.message : String(err) });
    } finally {
      set({ busyId: null });
    }
  },
}));
