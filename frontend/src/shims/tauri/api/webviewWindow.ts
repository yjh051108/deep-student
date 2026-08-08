// ============================================================
// Tauri → Wails 适配层：@tauri-apps/api/webviewWindow（最小实现）
// ============================================================

import { Window } from './window';
import { listen, once } from './event';

export class WebviewWindow extends Window {
  declare label: string;

  constructor(label?: string, options?: Record<string, unknown>) {
    super();
    this.label = label ?? 'main';
    void options;
  }

  static getByLabel(label: string): WebviewWindow | null {
    void label;
    return new WebviewWindow();
  }
  static getCurrent(): WebviewWindow {
    return new WebviewWindow();
  }
  static getAll(): WebviewWindow[] {
    return [new WebviewWindow()];
  }

  async setFocus(): Promise<void> {
    // 浏览器环境无焦点抢占语义
  }
  async once(event: string, handler: (e: unknown) => void): Promise<() => void> {
    return once(event, handler);
  }
  async listen(event: string, handler: (e: unknown) => void): Promise<() => void> {
    return listen(event, handler);
  }
}

export function getCurrentWebviewWindow(): WebviewWindow {
  return new WebviewWindow();
}
