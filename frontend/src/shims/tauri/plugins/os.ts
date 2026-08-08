// Tauri → Wails 适配层：@tauri-apps/plugin-os（stub）
function ua(): string {
  return typeof navigator !== 'undefined' ? navigator.userAgent : '';
}
export async function platform(): Promise<string> {
  const u = ua();
  if (/Windows/i.test(u)) return 'windows';
  if (/Mac/i.test(u)) return 'macos';
  if (/Linux/i.test(u)) return 'linux';
  return 'unknown';
}
export async function version(): Promise<string> {
  return ua();
}
export async function arch(): Promise<string> {
  return '';
}
export async function locale(): Promise<string> {
  return typeof navigator !== 'undefined' ? navigator.language : 'zh-CN';
}
export async function hostname(): Promise<string> {
  return 'localhost';
}
export async function type(): Promise<string> {
  return 'desktop';
}
