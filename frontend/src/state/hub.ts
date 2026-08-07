// Hub Store —— 学习中心资源管理状态
// ------------------------------------------------------------
// 对接后端 Wails 绑定：
// - HubList(type) / HubSearch(type, tag) / HubGet(uri)
// - HubImportResource(type, title, data, tags) / HubDelete(uri)
// - HubContinueNote(uri, prompt) —— AI 续写笔记
//
// 设计要点：
// - 资源类型树（左侧）：note / textbook / qbank / mindmap / translation / flashcard / paper / chat / todo / skill
// - 当前选中类型 + 标签过滤 + 关键字搜索（中间列表）
// - 当前选中资源的预览（右侧）

import { create } from "zustand";
import { callWails } from "@/lib/wails";

/** VFS 资源类型 —— 与后端 pkg/vfs.ResourceType 保持一致 */
export type ResourceType =
  | "note"
  | "textbook"
  | "qbank"
  | "mindmap"
  | "translation"
  | "flashcard"
  | "paper"
  | "chat"
  | "todo"
  | "skill";

/** VFS Entry —— 与后端 pkg/vfs.Entry JSON 标签对齐 */
export interface VFSEntry {
  uri: string;
  type: ResourceType;
  id: string;
  title: string;
  tags: string[];
  metadata: Record<string, string>;
  blob_ref: string;
  size: number;
  created_at: number;
  updated_at: number;
}

/** 资源类型元信息：用于左侧树渲染 */
export interface ResourceTypeMeta {
  type: ResourceType;
  label: string;
  description: string;
}

export const RESOURCE_TYPES: ResourceTypeMeta[] = [
  { type: "note", label: "笔记", description: "富文本笔记 / 知识点整理" },
  { type: "textbook", label: "教材", description: "PDF / DOCX 教材文件" },
  { type: "qbank", label: "题库", description: "题集 / 试卷 / 练习" },
  { type: "mindmap", label: "思维导图", description: "节点树 / 大纲" },
  { type: "translation", label: "翻译", description: "全文翻译 / 双语对照" },
  { type: "flashcard", label: "卡片", description: "Anki 制卡任务" },
  { type: "paper", label: "论文", description: "arXiv / OpenAlex 论文" },
  { type: "chat", label: "会话", description: "聊天会话存档" },
  { type: "todo", label: "待办", description: "任务清单" },
  { type: "skill", label: "技能", description: "SKILL.md 技能定义" },
];

interface HubState {
  /** 当前选中的资源类型（空表示"全部"） */
  activeType: ResourceType | "";
  /** 当前选中资源的 URI */
  activeUri: string | null;
  /** 资源列表 */
  entries: VFSEntry[];
  /** 标签过滤（为空表示不过滤） */
  tagFilter: string;
  /** 关键字搜索 */
  keyword: string;
  /** 加载状态 */
  loading: boolean;
  /** 错误信息 */
  error: string | null;
  /** 当前预览的资源内容（文本形式） */
  previewContent: string;
  /** 预览加载状态 */
  previewLoading: boolean;
  /** AI 续写输出 */
  continuation: string;
  /** 续写加载状态 */
  continuing: boolean;

  // —— Actions ——
  selectType: (t: ResourceType | "") => void;
  selectResource: (uri: string | null) => void;
  setTagFilter: (tag: string) => void;
  setKeyword: (kw: string) => void;
  refresh: () => Promise<void>;
  importResource: (
    type: ResourceType,
    title: string,
    data: Uint8Array,
    tags: string[]
  ) => Promise<string | null>;
  removeResource: (uri: string) => Promise<void>;
  continueNote: (uri: string, prompt: string) => Promise<void>;
  clearContinuation: () => void;
}

export const useHubStore = create<HubState>((set, get) => ({
  activeType: "note",
  activeUri: null,
  entries: [],
  tagFilter: "",
  keyword: "",
  loading: false,
  error: null,
  previewContent: "",
  previewLoading: false,
  continuation: "",
  continuing: false,

  selectType: (t) => {
    set({ activeType: t, activeUri: null, previewContent: "", tagFilter: "" });
    void get().refresh();
  },

  selectResource: (uri) => {
    set({ activeUri: uri, continuation: "" });
    if (uri) void loadPreview(uri, set);
    else set({ previewContent: "" });
  },

  setTagFilter: (tag) => {
    set({ tagFilter: tag });
    void get().refresh();
  },

  setKeyword: (kw) => {
    set({ keyword: kw });
  },

  refresh: async () => {
    const { activeType, tagFilter } = get();
    set({ loading: true, error: null });
    try {
      let list: VFSEntry[] | null;
      if (tagFilter) {
        list = await callWails<VFSEntry[]>("HubSearch", activeType, tagFilter);
      } else {
        list = await callWails<VFSEntry[]>("HubList", activeType);
      }
      // 后端不可用时给空数组（避免 UI 崩溃）
      set({ entries: list ?? [] });
    } catch (err) {
      set({ error: err instanceof Error ? err.message : String(err) });
    } finally {
      set({ loading: false });
    }
  },

  importResource: async (type, title, data, tags) => {
    set({ loading: true, error: null });
    try {
      // callWails 接受 Uint8Array，Wails 会自动转 Go []byte
      const uri = await callWails<string>(
        "HubImportResource",
        type,
        title,
        Array.from(data),
        tags
      );
      if (uri) {
        await get().refresh();
      }
      return uri;
    } catch (err) {
      set({ error: err instanceof Error ? err.message : String(err) });
      return null;
    } finally {
      set({ loading: false });
    }
  },

  removeResource: async (uri) => {
    set({ loading: true, error: null });
    try {
      await callWails<void>("HubDelete", uri);
      // 删除成功后清除选中状态并刷新
      if (get().activeUri === uri) {
        set({ activeUri: null, previewContent: "" });
      }
      await get().refresh();
    } catch (err) {
      set({ error: err instanceof Error ? err.message : String(err) });
    } finally {
      set({ loading: false });
    }
  },

  continueNote: async (uri, prompt) => {
    set({ continuing: true, continuation: "" });
    try {
      const out = await callWails<string>("HubContinueNote", uri, prompt);
      set({ continuation: out ?? "[后端未连接] 续写不可用" });
    } catch (err) {
      set({
        continuation: "[错误] " + (err instanceof Error ? err.message : String(err)),
      });
    } finally {
      set({ continuing: false });
    }
  },

  clearContinuation: () => set({ continuation: "" }),
}));

/** zustand set 函数类型：接受 partial 对象或 updater 函数 */
type HubSetFn = (
  partial: Partial<HubState> | ((state: HubState) => Partial<HubState>)
) => void;

/** 加载资源预览内容 */
async function loadPreview(uri: string, set: HubSetFn) {
  set({ previewLoading: true, previewContent: "" });
  try {
    const res = await callWails<{ data: number[]; entry: VFSEntry }>(
      "HubGet",
      uri
    );
    if (!res) {
      set({ previewContent: "[后端未连接] 无法读取资源内容" });
      return;
    }
    // 把 number[] 转 Uint8Array 再 decode
    const bytes = new Uint8Array(res.data ?? []);
    // 尝试 UTF-8 解码；二进制资源给出大小提示
    try {
      const text = new TextDecoder("utf-8", { fatal: false }).decode(bytes);
      set({ previewContent: text });
    } catch {
      set({
        previewContent: `[二进制资源] ${bytes.length} 字节，类型：${res.entry?.type ?? "未知"}`,
      });
    }
  } catch (err) {
    set({
      previewContent:
        "[错误] " + (err instanceof Error ? err.message : String(err)),
    });
  } finally {
    set({ previewLoading: false });
  }
}
