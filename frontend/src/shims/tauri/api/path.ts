// ============================================================
// Tauri → Wails 适配层：@tauri-apps/api/path
// ------------------------------------------------------------
// 纯路径函数原样实现；目录常量返回占位值（真实文件系统由 Go 侧管理）。
// ============================================================

const SEP = '/';

export function join(...paths: string[]): string {
  return paths
    .filter((p) => p != null && p !== '')
    .join(SEP)
    .replace(/\/+/g, SEP);
}

export function dirname(path: string): string {
  const idx = path.lastIndexOf(SEP);
  if (idx <= 0) return path === '' ? '.' : '.';
  return path.slice(0, idx) || SEP;
}

export function basename(path: string, ext?: string): string {
  const name = path.slice(path.lastIndexOf(SEP) + 1);
  if (ext && name.endsWith(ext)) return name.slice(0, -ext.length);
  return name;
}

export function extname(path: string): string {
  const name = basename(path);
  const idx = name.lastIndexOf('.');
  return idx >= 0 ? name.slice(idx) : '';
}

export function isAbsolute(path: string): boolean {
  return /^([a-zA-Z]:[\\/]|\/)/.test(path);
}

export function resolve(...paths: string[]): string {
  return join(...paths);
}

export function normalize(path: string): string {
  return path.replace(/\\/g, SEP).replace(/\/+/g, SEP);
}

export async function homeDir(): Promise<string> {
  return '~/Documents/DeepStudent';
}

export async function appDataDir(): Promise<string> {
  return '~/AppData/DeepStudent';
}

export async function appConfigDir(): Promise<string> {
  return '~/AppData/DeepStudent/config';
}

export async function appLocalDataDir(): Promise<string> {
  return '~/AppData/DeepStudent/local';
}

export async function appCacheDir(): Promise<string> {
  return '~/AppData/DeepStudent/cache';
}

export async function documentDir(): Promise<string> {
  return '~/Documents';
}

export async function downloadDir(): Promise<string> {
  return '~/Downloads';
}

export async function desktopDir(): Promise<string> {
  return '~/Desktop';
}

export async function tempDir(): Promise<string> {
  return '~/AppData/Local/Temp';
}

export async function currentDir(): Promise<string> {
  return '.';
}
