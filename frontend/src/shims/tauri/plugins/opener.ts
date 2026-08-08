// Tauri → Wails 适配层：@tauri-apps/plugin-opener
// 打开 URL / 文件 / 资源管理器 —— 浏览器等价实现
export async function openUrl(url: string): Promise<void> {
  const a = document.createElement('a');
  a.href = url;
  a.target = '_blank';
  a.rel = 'noopener noreferrer';
  document.body.appendChild(a);
  a.click();
  document.body.removeChild(a);
}
export async function openPath(path: string): Promise<void> {
  // 浏览器无法打开本地路径；交给后端（若暴露了打开方法）或 no-op
  console.debug('[tauri-shim] openPath no-op:', path);
}
export async function revealItemInDir(path: string): Promise<void> {
  console.debug('[tauri-shim] revealItemInDir no-op:', path);
}
export async function open(path: string): Promise<void> {
  await openUrl(path);
}
