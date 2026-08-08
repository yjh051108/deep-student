// Tauri → Wails 适配层：@tauri-apps/plugin-process（stub）
export async function exit(exitCode?: number): Promise<void> {
  void exitCode;
  console.debug('[tauri-shim] process.exit no-op');
}
export async function relaunch(): Promise<void> {
  console.debug('[tauri-shim] process.relaunch no-op');
}
