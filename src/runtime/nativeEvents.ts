import { emit as tauriEmit, listen as tauriListen } from '@tauri-apps/api/event';
import { Events as WailsEvents } from '@wailsio/runtime';
import { isTauriRuntime, isWailsRuntime } from './native';

export type NativeUnlistenFn = () => void;

export interface NativeEvent<T = unknown> {
  event: string;
  payload: T;
}

function isWailsEventEnvelope<T>(payload: unknown, event: string): payload is { name: string; data: T } {
  return Boolean(
    payload &&
    typeof payload === 'object' &&
    'name' in payload &&
    (payload as { name?: unknown }).name === event &&
    'data' in payload,
  );
}

export async function listen<T>(
  event: string,
  handler: (event: NativeEvent<T>) => void
): Promise<NativeUnlistenFn> {
  if (isWailsRuntime()) {
    return WailsEvents.On(event, payload => {
      handler({
        event,
        payload: isWailsEventEnvelope<T>(payload, event) ? payload.data : (payload as T),
      });
    });
  }

  if (isTauriRuntime()) {
    return await tauriListen<T>(event, handler as Parameters<typeof tauriListen<T>>[1]);
  }

  return () => {};
}

export async function emit<T = unknown>(event: string, payload?: T): Promise<void> {
  if (isWailsRuntime()) {
    await WailsEvents.Emit(event, payload);
    return;
  }

  if (isTauriRuntime()) {
    await tauriEmit(event, payload);
  }
}
