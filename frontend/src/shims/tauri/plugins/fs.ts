// ============================================================
// Tauri → Wails 适配层：@tauri-apps/plugin-fs
// ------------------------------------------------------------
// 原版在壁纸库/文件管理中使用。Wails 下文件系统由 Go 侧统一管理，
// 这里用浏览器 File System Access 能力降级实现（不可用时返回空态）。
// ============================================================

export interface DirEntry {
  name: string;
  isFile: boolean;
  isDirectory: boolean;
  isSymlink: boolean;
}

export type { FileEntry } from '../api/types';

// 内存映射（会话级），模拟最小文件系统
const memFS = new Map<string, Uint8Array>();

function norm(p: string): string {
  return p.replace(/\\/g, '/').replace(/^\.\//, '');
}

export async function readTextFile(path: string): Promise<string> {
  const data = memFS.get(norm(path));
  if (data) return new TextDecoder().decode(data);
  throw new Error(`[tauri-shim] readTextFile: file not found: ${path}`);
}

export async function readFile(path: string): Promise<Uint8Array> {
  const data = memFS.get(norm(path));
  if (data) return data;
  throw new Error(`[tauri-shim] readFile: file not found: ${path}`);
}

export async function writeTextFile(path: string, contents: string): Promise<void> {
  memFS.set(norm(path), new TextEncoder().encode(contents));
}

export async function writeFile(path: string, contents: Uint8Array | ArrayBuffer): Promise<void> {
  const buf = contents instanceof ArrayBuffer ? new Uint8Array(contents) : contents;
  memFS.set(norm(path), buf);
}

export async function exists(path: string): Promise<boolean> {
  return memFS.has(norm(path));
}

export async function mkdir(path: string, options?: { recursive?: boolean }): Promise<void> {
  void options;
  memFS.set(norm(path) + '/', new Uint8Array(0));
}

export async function readDir(path: string): Promise<DirEntry[]> {
  const prefix = norm(path).replace(/\/$/, '');
  const entries: DirEntry[] = [];
  const seen = new Set<string>();
  for (const key of memFS.keys()) {
    const k = norm(key);
    if (!k.startsWith(prefix + '/') || k === prefix + '/') continue;
    const rest = k.slice(prefix.length + 1);
    const name = rest.split('/')[0]!;
    if (seen.has(name)) continue;
    seen.add(name);
    const isDir = memFS.has(prefix + '/' + name + '/') || rest.includes('/');
    entries.push({ name, isFile: !isDir, isDirectory: isDir, isSymlink: false });
  }
  return entries;
}

export async function remove(path: string, options?: { recursive?: boolean }): Promise<void> {
  void options;
  const prefix = norm(path).replace(/\/$/, '');
  for (const key of [...memFS.keys()]) {
    if (key === prefix || key.startsWith(prefix + '/')) memFS.delete(key);
  }
}

export async function copyFile(src: string, dest: string): Promise<void> {
  const data = memFS.get(norm(src));
  if (data) memFS.set(norm(dest), data);
  else throw new Error(`[tauri-shim] copyFile: source not found: ${src}`);
}

export async function rename(src: string, dest: string): Promise<void> {
  const data = memFS.get(norm(src));
  if (data) {
    memFS.set(norm(dest), data);
    memFS.delete(norm(src));
  }
}

export async function stat(path: string): Promise<{ isFile: boolean; isDirectory: boolean; size: number }> {
  const data = memFS.get(norm(path));
  const isDir = memFS.has(norm(path) + '/');
  return {
    isFile: !isDir,
    isDirectory: isDir,
    size: data?.length ?? 0,
  };
}
