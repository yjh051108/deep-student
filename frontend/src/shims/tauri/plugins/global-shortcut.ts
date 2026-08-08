// Tauri → Wails 适配层：@tauri-apps/plugin-global-shortcut（stub）
// Go 版快捷键由应用内快捷键处理，不注册系统级
export async function register(shortcut: string, handler: () => void): Promise<void> {
  void shortcut;
  void handler;
  console.debug('[tauri-shim] global-shortcut.register no-op');
}
export async function unregister(shortcut: string): Promise<void> {
  void shortcut;
}
export async function unregisterAll(): Promise<void> {}
export async function isRegistered(shortcut: string): Promise<boolean> {
  void shortcut;
  return false;
}
