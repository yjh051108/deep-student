// Notes Store —— 笔记系统状态管理
// ------------------------------------------------------------
// 对接后端 Wails 绑定（NotesXxx 方法）。
// 设计要点：
// - 三栏布局：左文件夹/回收站 / 中笔记列表 / 右编辑器
// - 支持搜索、标签筛选、排序
// - 软删除 + 回收站 + 恢复 + 永久删除 + 清空
// - 导入导出 Markdown / HTML / JSON

import { create } from "zustand";
import {
  notesApi,
  encodeBytes,
  decodeBytes,
  type Note,
  type Folder,
  type Asset,
  type ListOptions,
  type CreateParams,
  type UpdateParams,
  type ExportFormat,
} from "@/lib/notes";

/** 排序键 */
export type SortBy = "updated" | "created" | "title" | "wordCount";
export type SortDir = "asc" | "desc";

interface NotesState {
  // —— 列表数据 ——
  /** 笔记列表（列表视图用元数据，不含正文） */
  items: Note[];
  /** 文件夹列表 */
  folders: Folder[];
  /** 当前选中文件夹 ID（null = 全部 / 根目录） */
  selectedFolderId: string | null;
  /** 是否查看回收站 */
  viewingTrash: boolean;
  /** 总数 */
  total: number;

  // —— 搜索与筛选 ——
  keyword: string;
  sortBy: SortBy;
  sortDir: SortDir;

  // —— 选中与编辑 ——
  /** 当前选中笔记 ID */
  selectedId: string | null;
  /** 当前选中笔记（含正文） */
  current: Note | null;
  /** 编辑草稿（标题 + 正文 + 标签） */
  draftTitle: string;
  draftContent: string;
  draftTags: string[];
  /** 草稿是否与 current 一致（无未保存修改） */
  dirty: boolean;

  // —— 资产 ——
  assets: Asset[];

  // —— 统计 ——
  stats: { total: number; trash: number; pinned: number; assets: number };

  // —— 加载状态 ——
  loading: boolean;
  saving: boolean;
  error: string | null;
  toast: string | null;

  // —— Actions ——
  // 初始化
  init: () => Promise<void>;
  // 列表
  refresh: () => Promise<void>;
  refreshFolders: () => Promise<void>;
  refreshStats: () => Promise<void>;
  setFolder: (id: string | null) => void;
  viewTrash: (on: boolean) => void;
  setKeyword: (k: string) => void;
  setSort: (by: SortBy, dir: SortDir) => void;
  // 选中
  select: (id: string | null) => Promise<void>;
  // 草稿
  setDraftTitle: (t: string) => void;
  setDraftContent: (c: string) => void;
  setDraftTags: (t: string[]) => void;
  // CRUD
  create: () => Promise<void>;
  save: () => Promise<void>;
  moveToTrash: (id: string) => Promise<void>;
  restore: (id: string) => Promise<void>;
  hardDelete: (id: string) => Promise<void>;
  emptyTrash: () => Promise<void>;
  togglePinned: (id: string, pinned: boolean) => Promise<void>;
  // 文件夹
  createFolder: (name: string) => Promise<void>;
  renameFolder: (id: string, name: string) => Promise<void>;
  deleteFolder: (id: string) => Promise<void>;
  // 资产
  loadAssets: (noteId: string) => Promise<void>;
  addAsset: (noteId: string, filename: string, data: number[], mime: string) => Promise<void>;
  deleteAsset: (assetId: string) => Promise<void>;
  // 导入导出
  importMarkdown: (filename: string, content: string) => Promise<void>;
  exportNote: (id: string, format: ExportFormat) => Promise<string>;
  exportAll: (format: ExportFormat) => Promise<string>;
  // 内部
  setToast: (msg: string | null) => void;
  clearError: () => void;
}

export const useNotesStore = create<NotesState>((set, get) => ({
  items: [],
  folders: [],
  selectedFolderId: null,
  viewingTrash: false,
  total: 0,
  keyword: "",
  sortBy: "updated",
  sortDir: "desc",
  selectedId: null,
  current: null,
  draftTitle: "",
  draftContent: "",
  draftTags: [],
  dirty: false,
  assets: [],
  stats: { total: 0, trash: 0, pinned: 0, assets: 0 },
  loading: false,
  saving: false,
  error: null,
  toast: null,

  // —— 初始化：加载文件夹 + 统计 + 列表 ——
  init: async () => {
    await Promise.all([
      get().refreshFolders(),
      get().refresh(),
      get().refreshStats(),
    ]);
  },

  // 内部辅助：刷新文件夹
  refreshFolders: async () => {
    try {
      const list = await notesApi.listFolders();
      set({ folders: list ?? [] });
    } catch (err) {
      set({ error: err instanceof Error ? err.message : String(err) });
    }
  },

  // 内部辅助：刷新统计
  refreshStats: async () => {
    try {
      const s = await notesApi.stats();
      if (s) {
        set({
          stats: {
            total: s.total ?? 0,
            trash: s.trash ?? 0,
            pinned: s.pinned ?? 0,
            assets: s.assets ?? 0,
          },
        });
      }
    } catch (err) {
      set({ error: err instanceof Error ? err.message : String(err) });
    }
  },

  // —— 刷新笔记列表 ——
  refresh: async () => {
    const { selectedFolderId, viewingTrash, keyword, sortBy, sortDir } = get();
    set({ loading: true, error: null });
    try {
      const opts: ListOptions = {
        limit: 200,
        offset: 0,
        sortBy,
        sortDir,
      };
      if (viewingTrash) {
        opts.onlyDeleted = true;
      } else {
        if (selectedFolderId !== null) opts.folderId = selectedFolderId;
        if (keyword.trim()) opts.keyword = keyword.trim();
      }
      const res = await notesApi.listMeta(opts);
      if (res) {
        set({ items: res.items ?? [], total: res.total ?? 0 });
      } else {
        set({ items: [], total: 0 });
      }
    } catch (err) {
      set({ error: err instanceof Error ? err.message : String(err) });
    } finally {
      set({ loading: false });
    }
  },

  setFolder: (id) => {
    set({ selectedFolderId: id, viewingTrash: false, selectedId: null, current: null });
    void get().refresh();
  },

  viewTrash: (on) => {
    set({ viewingTrash: on, selectedId: null, current: null, selectedFolderId: null });
    void get().refresh();
  },

  setKeyword: (k) => {
    set({ keyword: k });
  },

  setSort: (by, dir) => {
    set({ sortBy: by, sortDir: dir });
    void get().refresh();
  },

  // —— 选中笔记 ——
  select: async (id) => {
    if (!id) {
      set({ selectedId: null, current: null, draftTitle: "", draftContent: "", draftTags: [], dirty: false, assets: [] });
      return;
    }
    set({ selectedId: id, loading: true, error: null });
    try {
      const n = await notesApi.get(id);
      if (n) {
        set({
          current: n,
          draftTitle: n.title,
          draftContent: n.contentMd,
          draftTags: [...(n.tags ?? [])],
          dirty: false,
        });
        // 加载资产
        await get().loadAssets(id);
      }
    } catch (err) {
      set({ error: err instanceof Error ? err.message : String(err) });
    } finally {
      set({ loading: false });
    }
  },

  // —— 草稿编辑 ——
  setDraftTitle: (t) => set((s) => ({ draftTitle: t, dirty: true })),
  setDraftContent: (c) => set((s) => ({ draftContent: c, dirty: true })),
  setDraftTags: (t) => set((s) => ({ draftTags: t, dirty: true })),

  // —— 创建新笔记 ——
  create: async () => {
    const { selectedFolderId } = get();
    set({ saving: true, error: null });
    try {
      const params: CreateParams = {
        title: "未命名笔记",
        contentMd: "",
        tags: [],
        folderId: selectedFolderId,
      };
      const n = await notesApi.create(params);
      if (n) {
        set({
          selectedId: n.id,
          current: n,
          draftTitle: n.title,
          draftContent: n.contentMd,
          draftTags: [...(n.tags ?? [])],
          dirty: false,
          assets: [],
        });
        set({ toast: "已创建新笔记" });
        setTimeout(() => get().setToast(null), 2000);
        await get().refresh();
        await get().refreshStats();
      }
    } catch (err) {
      set({ error: err instanceof Error ? err.message : String(err) });
    } finally {
      set({ saving: false });
    }
  },

  // —— 保存草稿 ——
  save: async () => {
    const { selectedId, current, draftTitle, draftContent, draftTags } = get();
    if (!selectedId || !current) return;
    set({ saving: true, error: null });
    try {
      const params: UpdateParams = {
        id: selectedId,
        title: draftTitle,
        contentMd: draftContent,
        tags: draftTags,
        expectedUpdate: current.updatedAt,
      };
      const n = await notesApi.update(params);
      if (n) {
        set({
          current: n,
          draftTitle: n.title,
          draftContent: n.contentMd,
          draftTags: [...(n.tags ?? [])],
          dirty: false,
        });
        set({ toast: "已保存" });
        setTimeout(() => get().setToast(null), 1500);
        await get().refresh();
      }
    } catch (err) {
      set({ error: err instanceof Error ? err.message : String(err) });
    } finally {
      set({ saving: false });
    }
  },

  // —— 移入回收站 ——
  moveToTrash: async (id) => {
    set({ error: null });
    try {
      await notesApi.moveToTrash(id);
      if (get().selectedId === id) {
        set({ selectedId: null, current: null, draftTitle: "", draftContent: "", draftTags: [], dirty: false, assets: [] });
      }
      set({ toast: "已移入回收站" });
      setTimeout(() => get().setToast(null), 1500);
      await get().refresh();
      await get().refreshStats();
    } catch (err) {
      set({ error: err instanceof Error ? err.message : String(err) });
    }
  },

  // —— 恢复 ——
  restore: async (id) => {
    set({ error: null });
    try {
      await notesApi.restore(id);
      set({ toast: "已恢复" });
      setTimeout(() => get().setToast(null), 1500);
      await get().refresh();
      await get().refreshStats();
    } catch (err) {
      set({ error: err instanceof Error ? err.message : String(err) });
    }
  },

  // —— 永久删除 ——
  hardDelete: async (id) => {
    set({ error: null });
    try {
      await notesApi.hardDelete(id);
      if (get().selectedId === id) {
        set({ selectedId: null, current: null, draftTitle: "", draftContent: "", draftTags: [], dirty: false, assets: [] });
      }
      set({ toast: "已永久删除" });
      setTimeout(() => get().setToast(null), 1500);
      await get().refresh();
      await get().refreshStats();
    } catch (err) {
      set({ error: err instanceof Error ? err.message : String(err) });
    }
  },

  // —— 清空回收站 ——
  emptyTrash: async () => {
    set({ error: null });
    try {
      const count = await notesApi.emptyTrash();
      set({ toast: `已清空 ${count} 条` });
      setTimeout(() => get().setToast(null), 2000);
      await get().refresh();
      await get().refreshStats();
    } catch (err) {
      set({ error: err instanceof Error ? err.message : String(err) });
    }
  },

  // —— 切换置顶 ——
  togglePinned: async (id, pinned) => {
    set({ error: null });
    try {
      await notesApi.update({ id, isPinned: pinned });
      await get().refresh();
      await get().refreshStats();
    } catch (err) {
      set({ error: err instanceof Error ? err.message : String(err) });
    }
  },

  // —— 文件夹 CRUD ——
  createFolder: async (name) => {
    set({ error: null });
    try {
      await notesApi.createFolder(name, null);
      set({ toast: "文件夹已创建" });
      setTimeout(() => get().setToast(null), 1500);
      await get().refreshFolders();
    } catch (err) {
      set({ error: err instanceof Error ? err.message : String(err) });
    }
  },

  renameFolder: async (id, name) => {
    set({ error: null });
    try {
      await notesApi.updateFolder(id, name);
      await get().refreshFolders();
    } catch (err) {
      set({ error: err instanceof Error ? err.message : String(err) });
    }
  },

  deleteFolder: async (id) => {
    set({ error: null });
    try {
      await notesApi.deleteFolder(id);
      if (get().selectedFolderId === id) {
        set({ selectedFolderId: null });
      }
      set({ toast: "文件夹已删除" });
      setTimeout(() => get().setToast(null), 1500);
      await get().refreshFolders();
      await get().refresh();
    } catch (err) {
      set({ error: err instanceof Error ? err.message : String(err) });
    }
  },

  // —— 资产 ——
  loadAssets: async (noteId) => {
    try {
      const list = await notesApi.listAssets(noteId);
      set({ assets: list ?? [] });
    } catch (err) {
      set({ error: err instanceof Error ? err.message : String(err) });
    }
  },

  addAsset: async (noteId, filename, data, mime) => {
    set({ error: null });
    try {
      await notesApi.addAsset(noteId, filename, data, mime);
      set({ toast: "附件已添加" });
      setTimeout(() => get().setToast(null), 1500);
      await get().loadAssets(noteId);
      // 更新当前笔记的资产标志
      if (get().current?.id === noteId) {
        const n = await notesApi.get(noteId);
        if (n) set({ current: n });
      }
      await get().refresh();
      await get().refreshStats();
    } catch (err) {
      set({ error: err instanceof Error ? err.message : String(err) });
    }
  },

  deleteAsset: async (assetId) => {
    set({ error: null });
    try {
      const noteId = get().current?.id;
      await notesApi.deleteAsset(assetId);
      if (noteId) await get().loadAssets(noteId);
      set({ toast: "附件已删除" });
      setTimeout(() => get().setToast(null), 1500);
      await get().refresh();
      await get().refreshStats();
    } catch (err) {
      set({ error: err instanceof Error ? err.message : String(err) });
    }
  },

  // —— 导入 ——
  importMarkdown: async (filename, content) => {
    const { selectedFolderId } = get();
    set({ saving: true, error: null });
    try {
      const bytes = encodeBytes(content);
      const n = await notesApi.importMarkdown(filename, bytes, selectedFolderId);
      if (n) {
        set({ toast: `已导入：${n.title}` });
        setTimeout(() => get().setToast(null), 2000);
        await get().refresh();
        await get().refreshStats();
      }
    } catch (err) {
      set({ error: err instanceof Error ? err.message : String(err) });
    } finally {
      set({ saving: false });
    }
  },

  // —— 导出单条 ——
  exportNote: async (id, format) => {
    try {
      const bytes = await notesApi.exportNote(id, format);
      if (!bytes) return "";
      return decodeBytes(bytes);
    } catch (err) {
      set({ error: err instanceof Error ? err.message : String(err) });
      return "";
    }
  },

  // —— 导出全部 ——
  exportAll: async (format) => {
    try {
      const bytes = await notesApi.exportAll(format);
      if (!bytes) return "";
      return decodeBytes(bytes);
    } catch (err) {
      set({ error: err instanceof Error ? err.message : String(err) });
      return "";
    }
  },

  setToast: (msg) => set({ toast: msg }),
  clearError: () => set({ error: null }),
}));
