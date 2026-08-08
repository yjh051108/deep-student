// ============================================================
// Tauri → Wails 适配层：@tauri-apps/api/event
// ------------------------------------------------------------
// 原版用 listen/emit 做前后端事件通信（流式 chunk、进度通知等）。
// Wails v2 提供 wailsjs/runtime 的 EventsOn/EventsEmit，语义等价。
// ============================================================

// 直接引用 wails 生成的 runtime（构建时存在 frontend/wailsjs/runtime）
import {
  EventsOn,
  EventsOnce,
  EventsOff,
  EventsEmit,
} from '../../../../wailsjs/runtime/runtime';

export type UnlistenFn = () => void;
export type EventName = string;

export interface Event<T = unknown> {
  event: EventName;
  id: number;
  payload: T;
}

export type EventCallback<T> = (event: Event<T>) => void;

export async function listen<T>(
  event: EventName,
  handler: EventCallback<T>
): Promise<UnlistenFn> {
  let seq = 0;
  EventsOn(event, (payload: T) => {
    handler({ event, id: ++seq, payload });
  });
  return () => EventsOff(event);
}

export async function once<T>(
  event: EventName,
  handler: EventCallback<T>
): Promise<UnlistenFn> {
  let seq = 0;
  EventsOnce(event, (payload: T) => {
    handler({ event, id: ++seq, payload });
  });
  return () => EventsOff(event);
}

export async function emit(event: EventName, payload?: unknown): Promise<void> {
  EventsEmit(event, payload);
}

export async function emitTo(
  target: string,
  event: EventName,
  payload?: unknown
): Promise<void> {
  void target;
  EventsEmit(event, payload);
}

export type { EventsOn as tauriEventsOn };
