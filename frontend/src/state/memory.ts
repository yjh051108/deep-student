// Memory Store —— 智能记忆状态管理
// ------------------------------------------------------------
// 对接后端 Wails 绑定：
// - MemoryIngest(conversation) → Item[]
// - MemorySearch(q) → Item[]
// - MemoryProfile() → string / MemoryPrivacyMode(on)
// - MemoryDecay()
//
// 设计要点：
// - 三栏布局：左操作面板 / 中记忆列表 / 右详情 + 摄入
// - 记忆按 category 分组：identity / preference / goal / subject / other
// - 支持搜索、用户画像、隐私模式、衰减

import { create } from "zustand";
import { callWails } from "@/lib/wails";

/** 记忆条目 —— 与后端 memory.Item 对齐 */
export interface MemoryItem {
  id: string;
  category: string; // identity | preference | goal | subject | other
  content: string;
  tags: string[];
  weight: number;
  hit_count: number;
  last_hit: string;
  created_at: string;
  updated_at: string;
  source?: string;
  metadata?: Record<string, string>;
}

/** 记忆分类 */
export const MEMORY_CATEGORIES = [
  { key: "identity", label: "身份" },
  { key: "preference", label: "偏好" },
  { key: "goal", label: "目标" },
  { key: "subject", label: "学科" },
  { key: "other", label: "其他" },
] as const;

interface MemoryState {
  /** 全部记忆（搜索结果或全量） */
  items: MemoryItem[];
  /** 搜索关键词 */
  searchQuery: string;
  /** 是否处于搜索结果模式 */
  searching: boolean;
  /** 当前选中记忆 ID */
  selectedId: string | null;
  /** 用户画像文本 */
  profile: string;
  /** 隐私模式开关 */
  privacyMode: boolean;
  /** 摄入对话文本 */
  ingestText: string;
  /** 摄入后新增的记忆 */
  ingestedItems: MemoryItem[];
  /** 加载状态 */
  loading: boolean;
  /** 画像加载状态 */
  profileLoading: boolean;
  /** 错误信息 */
  error: string | null;
  /** 操作提示 */
  toast: string | null;

  // —— Actions ——
  setSearchQuery: (q: string) => void;
  setIngestText: (t: string) => void;
  selectItem: (id: string | null) => void;
  search: () => Promise<void>;
  clearSearch: () => void;
  loadProfile: () => Promise<void>;
  ingest: () => Promise<void>;
  togglePrivacy: () => Promise<void>;
  decay: () => Promise<void>;
}

export const useMemoryStore = create<MemoryState>((set, get) => ({
  items: [],
  searchQuery: "",
  searching: false,
  selectedId: null,
  profile: "",
  privacyMode: false,
  ingestText: "",
  ingestedItems: [],
  loading: false,
  profileLoading: false,
  error: null,
  toast: null,

  setSearchQuery: (q) => set({ searchQuery: q }),
  setIngestText: (t) => set({ ingestText: t }),
  selectItem: (id) => set({ selectedId: id }),

  // —— 搜索记忆 ——
  search: async () => {
    const { searchQuery } = get();
    if (!searchQuery.trim()) {
      set({ error: "请输入搜索关键词" });
      return;
    }
    set({ loading: true, error: null, searching: true, selectedId: null });
    try {
      const list = await callWails<MemoryItem[]>("MemorySearch", searchQuery.trim());
      set({ items: list ?? [] });
    } catch (err) {
      set({ error: err instanceof Error ? err.message : String(err) });
    } finally {
      set({ loading: false });
    }
  },

  clearSearch: () =>
    set({ searching: false, items: [], searchQuery: "", selectedId: null }),

  // —— 加载用户画像 ——
  loadProfile: async () => {
    set({ profileLoading: true, error: null });
    try {
      const text = await callWails<string>("MemoryProfile");
      set({ profile: text ?? "[后端未连接] 画像不可用" });
    } catch (err) {
      set({ error: err instanceof Error ? err.message : String(err) });
    } finally {
      set({ profileLoading: false });
    }
  },

  // —— 摄入对话 ——
  ingest: async () => {
    const { ingestText } = get();
    if (!ingestText.trim()) {
      set({ error: "请粘贴对话文本" });
      return;
    }
    set({ loading: true, error: null });
    try {
      const newItems = await callWails<MemoryItem[]>(
        "MemoryIngest",
        ingestText
      );
      const items = newItems ?? [];
      set({ ingestedItems: items, ingestText: "" });
      set({ toast: `已摄入 ${items.length} 条记忆` });
      // 2 秒后清除提示
      setTimeout(() => set({ toast: null }), 2000);
    } catch (err) {
      set({ error: err instanceof Error ? err.message : String(err) });
    } finally {
      set({ loading: false });
    }
  },

  // —— 切换隐私模式 ——
  togglePrivacy: async () => {
    const { privacyMode } = get();
    const next = !privacyMode;
    set({ privacyMode: next, error: null });
    try {
      await callWails<void>("MemoryPrivacyMode", next);
      set({ toast: next ? "隐私模式已开启" : "隐私模式已关闭" });
      setTimeout(() => set({ toast: null }), 2000);
    } catch (err) {
      set({ error: err instanceof Error ? err.message : String(err) });
    }
  },

  // —— 衰减 ——
  decay: async () => {
    set({ loading: true, error: null });
    try {
      await callWails<void>("MemoryDecay");
      set({ toast: "衰减已完成" });
      setTimeout(() => set({ toast: null }), 2000);
    } catch (err) {
      set({ error: err instanceof Error ? err.message : String(err) });
    } finally {
      set({ loading: false });
    }
  },
}));
