// ============================================================
// Tauri → Wails 适配层：@tauri-apps/plugin-updater（stub）
// ------------------------------------------------------------
// Go 重构版没有更新服务；check 返回 null（无可用更新），
// 避免真实 tauri 插件包被加载。
// ============================================================

export interface Update {
  version: string;
  date?: string;
  body?: string;
  download(): Promise<void>;
  downloadAndInstall(): Promise<void>;
}

export async function check(): Promise<Update | null> {
  console.debug('[tauri-shim] plugin-updater disabled (Go build)');
  return null;
}
