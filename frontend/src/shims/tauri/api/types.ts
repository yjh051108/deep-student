// ============================================================
// Tauri → Wails 适配层：共享类型
// ============================================================

export interface FileEntry {
  path: string;
  name?: string;
  children?: FileEntry[];
}
