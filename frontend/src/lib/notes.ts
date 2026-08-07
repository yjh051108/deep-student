// Notes 前端类型与 Wails 封装
// ------------------------------------------------------------
// 与后端 internal/notes/types.go 对齐。
// 所有调用统一通过 callWails 走 window.go.deepstudent.App 绑定。

import { callWails } from "@/lib/wails";

/** 笔记主体 —— 与后端 notes.Note 对齐 */
export interface Note {
  id: string;
  title: string;
  contentMd: string;
  tags: string[];
  folderId?: string | null;
  hasAssets: boolean;
  assetCount: number;
  isPinned: boolean;
  isDeleted: boolean;
  deletedAt?: string | null;
  wordCount: number;
  charCount: number;
  createdAt: string;
  updatedAt: string;
  metadata?: Record<string, string>;
}

/** 文件夹 —— 与后端 notes.Folder 对齐 */
export interface Folder {
  id: string;
  name: string;
  parentId?: string | null;
  sortOrder: number;
  createdAt: string;
  updatedAt: string;
}

/** 笔记附件 —— 与后端 notes.Asset 对齐 */
export interface Asset {
  id: string;
  noteId: string;
  filename: string;
  mimeType: string;
  size: number;
  blobRef: string;
  createdAt: string;
}

/** 列表查询选项 —— 与后端 notes.ListOptions 对齐 */
export interface ListOptions {
  folderId?: string | null;
  tags?: string[];
  keyword?: string;
  includeDeleted?: boolean;
  onlyDeleted?: boolean;
  hasAssets?: boolean | null;
  dateStart?: string | null;
  dateEnd?: string | null;
  sortBy?: string; // "updated" | "created" | "title" | "wordCount"
  sortDir?: string; // "asc" | "desc"
  limit?: number;
  offset?: number;
}

/** 列表结果 —— 与后端 notes.ListResult 对齐 */
export interface ListResult {
  items: Note[];
  total: number;
  limit: number;
  offset: number;
}

/** 创建参数 —— 与后端 notes.CreateParams 对齐 */
export interface CreateParams {
  title: string;
  contentMd: string;
  tags?: string[];
  folderId?: string | null;
}

/** 更新参数 —— 与后端 notes.UpdateParams 对齐 */
export interface UpdateParams {
  id: string;
  title?: string;
  contentMd?: string;
  tags?: string[];
  folderId?: string | null;
  isPinned?: boolean;
  expectedUpdate?: string;
}

/** 导出格式 */
export type ExportFormat = "markdown" | "html" | "json";

// ===================== Wails API 封装 =====================

export const notesApi = {
  create: (params: CreateParams) =>
    callWails<Note>("NotesCreate", params),
  get: (id: string) => callWails<Note>("NotesGet", id),
  update: (params: UpdateParams) =>
    callWails<Note>("NotesUpdate", params),
  list: (opts: ListOptions) => callWails<ListResult>("NotesList", opts),
  listMeta: (opts: ListOptions) =>
    callWails<ListResult>("NotesListMeta", opts),
  moveToTrash: (id: string) => callWails<void>("NotesMoveToTrash", id),
  restore: (id: string) => callWails<void>("NotesRestore", id),
  hardDelete: (id: string) => callWails<void>("NotesHardDelete", id),
  emptyTrash: () => callWails<number>("NotesEmptyTrash"),
  trashCount: () => callWails<number>("NotesTrashCount"),
  listFolders: () => callWails<Folder[]>("NotesListFolders"),
  createFolder: (name: string, parentId?: string | null) =>
    callWails<Folder>("NotesCreateFolder", name, parentId ?? null),
  updateFolder: (id: string, name: string) =>
    callWails<void>("NotesUpdateFolder", id, name),
  deleteFolder: (id: string) => callWails<void>("NotesDeleteFolder", id),
  addAsset: (noteId: string, filename: string, data: number[], mime: string) =>
    callWails<Asset>("NotesAddAsset", noteId, filename, data, mime),
  listAssets: (noteId: string) =>
    callWails<Asset[]>("NotesListAssets", noteId),
  getAsset: (assetId: string) =>
    callWails<[number[], Asset] | null>("NotesGetAsset", assetId),
  deleteAsset: (assetId: string) => callWails<void>("NotesDeleteAsset", assetId),
  importMarkdown: (filename: string, content: number[], folderId?: string | null) =>
    callWails<Note>("NotesImportMarkdown", filename, content, folderId ?? null),
  importBatch: (files: Record<string, number[]>, folderId?: string | null) =>
    callWails<Note[]>("NotesImportBatch", files, folderId ?? null),
  exportNote: (id: string, format: ExportFormat) =>
    callWails<number[]>("NotesExportNote", id, format),
  exportAll: (format: ExportFormat) =>
    callWails<number[]>("NotesExportAll", format),
  search: (keyword: string, limit?: number) =>
    callWails<Note[]>("NotesSearch", keyword, limit ?? 20),
  stats: () => callWails<Record<string, number>>("NotesStats"),
};

// ===================== 工具函数 =====================

/** 将字符串编码为字节数组（用于导入/资产接口） */
export function encodeBytes(s: string): number[] {
  const enc = new TextEncoder();
  return Array.from(enc.encode(s));
}

/** 将字节数组解码为字符串（用于导出/资产读取） */
export function decodeBytes(b: number[] | Uint8Array): string {
  const arr = b instanceof Uint8Array ? b : new Uint8Array(b);
  return new TextDecoder().decode(arr);
}

/** 格式化文件大小 */
export function formatSize(bytes: number): string {
  if (bytes < 1024) return `${bytes} B`;
  if (bytes < 1024 * 1024) return `${(bytes / 1024).toFixed(1)} KB`;
  return `${(bytes / (1024 * 1024)).toFixed(1)} MB`;
}

/** 格式化日期时间 */
export function formatDateTime(ts: string): string {
  if (!ts) return "-";
  try {
    const d = new Date(ts);
    if (isNaN(d.getTime())) return ts;
    const y = d.getFullYear();
    const m = String(d.getMonth() + 1).padStart(2, "0");
    const day = String(d.getDate()).padStart(2, "0");
    const hh = String(d.getHours()).padStart(2, "0");
    const mm = String(d.getMinutes()).padStart(2, "0");
    return `${y}-${m}-${day} ${hh}:${mm}`;
  } catch {
    return ts;
  }
}
