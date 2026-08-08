// ============================================================
// Tauri → Wails 适配层：@tauri-apps/api/mocks（仅 dev 预览用）
// ============================================================

type IpcHandler = (cmd: string, args?: unknown) => unknown;

const handlers = new Map<string, IpcHandler>();

export function mockIPC(cb: (cmd: string, args?: unknown) => unknown): void {
  handlers.set('*', cb);
}

export function mockWindows(current: string, others?: string[]): void {
  void current;
  void others;
}

export function clearMocks(): void {
  handlers.clear();
}

export function mockIPCInternal(cmd: string, args?: unknown): unknown {
  const h = handlers.get('*');
  if (!h) return undefined;
  return h(cmd, args);
}
