// ============================================================
// Tauri → Wails 适配层：@tauri-apps/api/webview（最小实现）
// ============================================================

export class Webview {
  label = 'main';

  async setZoom(scale: number): Promise<void> {
    void scale;
    console.debug('[tauri-shim] webview.setZoom no-op');
  }
  async openDevTools(): Promise<void> {
    console.debug('[tauri-shim] webview.openDevTools no-op');
  }
  async closeDevTools(): Promise<void> {
    console.debug('[tauri-shim] webview.closeDevTools no-op');
  }
  /** 拖放事件（原版 Tauri onDragDropEvent）——Wails 无对应物，注册浏览器原生拖放 */
  async onDragDropEvent(
    handler: (event: {
      payload: {
        type: 'enter' | 'over' | 'drop' | 'leave';
        paths: string[];
        position: { x: number; y: number };
      };
    }) => void
  ): Promise<() => void> {
    const el = window;
    const wrap = (e: DragEvent) => {
      const paths = Array.from(e.dataTransfer?.files ?? []).map((f) => f.name);
      const type =
        e.type === 'dragenter' ? 'enter' : e.type === 'dragover' ? 'over' : e.type === 'drop' ? 'drop' : 'leave';
      handler({
        payload: {
          type,
          paths,
          position: { x: e.clientX, y: e.clientY },
        },
      });
    };
    const stop = (e: Event) => e.preventDefault();
    el.addEventListener('dragenter', wrap as EventListener);
    el.addEventListener('dragover', (e) => {
      stop(e);
      wrap(e as DragEvent);
    });
    el.addEventListener('drop', (e) => {
      stop(e);
      wrap(e as DragEvent);
    });
    el.addEventListener('dragleave', wrap as EventListener);
    return () => {
      el.removeEventListener('dragenter', wrap as EventListener);
      el.removeEventListener('dragover', wrap as EventListener);
      el.removeEventListener('drop', wrap as EventListener);
      el.removeEventListener('dragleave', wrap as EventListener);
    };
  }
}

export function getCurrentWebview(): Webview {
  return new Webview();
}
