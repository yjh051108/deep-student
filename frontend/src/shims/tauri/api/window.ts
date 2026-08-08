// ============================================================
// Tauri → Wails 适配层：@tauri-apps/api/window
// ------------------------------------------------------------
// 原版窗口控制（最大化/最小化/全屏/拖拽等）→ Wails runtime JS。
// ============================================================

import {
  WindowMaximise,
  WindowUnmaximise,
  WindowIsMaximised,
  WindowMinimise,
  WindowUnminimise,
  WindowIsMinimised,
  WindowFullscreen,
  WindowUnfullscreen,
  WindowIsFullscreen,
  WindowSetTitle,
  WindowCenter,
  WindowSetSize,
  WindowGetSize,
  WindowSetPosition,
  WindowGetPosition,
  WindowShow,
  WindowHide,
  WindowSetAlwaysOnTop,
  WindowReload,
  WindowSetMinSize,
  WindowSetMaxSize,
  WindowToggleMaximise,
} from '../../../../wailsjs/runtime/runtime';

export class Window {
  label = 'main';

  async maximize(): Promise<void> {
    WindowMaximise();
  }
  async unmaximize(): Promise<void> {
    WindowUnmaximise();
  }
  async toggleMaximize(): Promise<void> {
    WindowToggleMaximise();
  }
  async isMaximized(): Promise<boolean> {
    return WindowIsMaximised();
  }
  async minimize(): Promise<void> {
    WindowMinimise();
  }
  async unminimize(): Promise<void> {
    WindowUnminimise();
  }
  async isMinimized(): Promise<boolean> {
    return WindowIsMinimised();
  }
  async setFullscreen(flag: boolean): Promise<void> {
    if (flag) WindowFullscreen();
    else WindowUnfullscreen();
  }
  async isFullscreen(): Promise<boolean> {
    return WindowIsFullscreen();
  }
  async setTitle(title: string): Promise<void> {
    WindowSetTitle(title);
  }
  async center(): Promise<void> {
    WindowCenter();
  }
  async show(): Promise<void> {
    WindowShow();
  }
  async hide(): Promise<void> {
    WindowHide();
  }
  async setAlwaysOnTop(flag: boolean): Promise<void> {
    WindowSetAlwaysOnTop(flag);
  }
  async setSize(size: { width: number; height: number }): Promise<void> {
    WindowSetSize(size.width, size.height);
  }
  async getSize(): Promise<{ width: number; height: number }> {
    const size = await WindowGetSize();
    return { width: size.w, height: size.h };
  }
  async setMinSize(size: { width: number; height: number }): Promise<void> {
    WindowSetMinSize(size.width, size.height);
  }
  async setMaxSize(size: { width: number; height: number }): Promise<void> {
    WindowSetMaxSize(size.width, size.height);
  }
  async setPosition(pos: { x: number; y: number }): Promise<void> {
    WindowSetPosition(pos.x, pos.y);
  }
  async getPosition(): Promise<{ x: number; y: number }> {
    const pos = await WindowGetPosition();
    return { x: pos.x, y: pos.y };
  }
  async startDragging(): Promise<void> {
    // Wails 无公开 JS 拖拽 API；由 window.go 侧或 CSS app-region 处理
    console.debug('[tauri-shim] startDragging no-op');
  }
  async reload(): Promise<void> {
    WindowReload();
  }
  async close(): Promise<void> {
    // 交由 Go 侧窗口关闭（避免前端直接退出）
    console.debug('[tauri-shim] window.close no-op');
  }
  async listen(_event?: string, _handler?: (e: unknown) => void): Promise<() => void> {
    return () => {};
  }
  async onResized(): Promise<() => void> {
    return () => {};
  }
  async onMoved(): Promise<() => void> {
    return () => {};
  }
  async onCloseRequested(): Promise<() => void> {
    return () => {};
  }
  async onFocusChanged(): Promise<() => void> {
    return () => {};
  }
  async isVisible(): Promise<boolean> {
    return true;
  }
  async setZoom(scale: number): Promise<void> {
    void scale;
  }
  async onDragDropEvent(): Promise<() => void> {
    return () => {};
  }
}

export function getCurrentWindow(): Window {
  return new Window();
}

export const appWindow = getCurrentWindow();

// 兼容旧版命名
export class WebviewWindow extends Window {}
export function getCurrentWebviewWindow(): Window {
  return new Window();
}
