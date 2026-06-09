import i18next from 'i18next';
import {
  invoke,
  isInjectedNativeRuntime,
  isTauriRuntime,
  isWailsRuntime,
} from '../runtime/native';
import { listen } from '../runtime/nativeEvents';
import { getErrorMessage } from '../utils/errorUtils';

// 调试事件触发辅助函数
const emitStdioDebugEvent = (eventType: string, detail: any) => {
  try {
    if (typeof window !== 'undefined') {
      const event = new CustomEvent(eventType, { detail });
      window.dispatchEvent(event);
    }
  } catch (e: unknown) {
    // 静默失败，避免影响主逻辑
  }
};

export type StdioFraming = 'jsonl' | 'content_length';

export interface TauriStdioServerConfig {
  command: string;
  args?: string[];
  env?: Record<string, string>;
  cwd?: string;
  framing?: StdioFraming;
}

type JsonRpcMessage = Record<string, unknown>;

function isMobilePlatform(): boolean {
  if (typeof navigator === 'undefined') return false;
  const ua = navigator.userAgent.toLowerCase();
  return ua.includes('android') || ua.includes('iphone') || ua.includes('ipad');
}

function isNativeStdioEnvironment(): boolean {
  const result = isInjectedNativeRuntime() || isWailsRuntime() || isTauriRuntime();

  // 调试日志
  console.log('[Native Stdio] Environment check:', {
    isInjectedNative: isInjectedNativeRuntime(),
    isWails: isWailsRuntime(),
    isTauri: isTauriRuntime(),
    isNative: result,
    userAgent: typeof navigator !== 'undefined' ? navigator.userAgent : 'N/A',
  });

  return result;
}

export function isTauriStdioSupported(): boolean {
  const isNative = isNativeStdioEnvironment();
  const isMobile = isMobilePlatform();
  const isSupported = isNative && !isMobile;
  
  console.log('[Native Stdio] Support check:', {
    isNative,
    isMobile,
    isSupported,
  });

  return isSupported;
}

const STDIO_SPAWN_ERROR_FLAG = Symbol('dstu-stdio-spawn-error');

function enrichStdioSpawnError(error: unknown, command: string): Error {
  if (error && typeof error === 'object' && (error as any)[STDIO_SPAWN_ERROR_FLAG]) {
    return error as Error;
  }
  
  // 调试日志：查看原始错误
  console.log('[Native Stdio] enrichStdioSpawnError - original error:', {
    error,
    type: typeof error,
    isError: error instanceof Error,
    message: (error as any)?.message,
    constructor: (error as any)?.constructor?.name,
    keys: error && typeof error === 'object' ? Object.keys(error) : [],
  });

  // 使用 getErrorMessage 正确提取错误信息，避免 [object Object]
  const originalMessage = getErrorMessage(error);
  console.log('[Native Stdio] enrichStdioSpawnError - extracted message:', originalMessage);
  const lowered = originalMessage.toLowerCase();
  const isSpawnFailure =
    lowered.includes('failed to spawn') ||
    lowered.includes('enoent') ||
    lowered.includes('not found') ||
    lowered.includes('no such file or directory') ||
    lowered.includes('could not find the specified file');
  if (!isSpawnFailure) {
    return error instanceof Error ? error : new Error(originalMessage);
  }
  const trimmedCommand = (command || '').trim() || 'mcp-server';
  const hints: string[] = [
    `- ${i18next.t('mcp:stdio.spawn_hint_check_path', { command: trimmedCommand })}`,
    `- ${i18next.t('mcp:stdio.spawn_hint_npx')}`,
    `- ${i18next.t('mcp:stdio.spawn_hint_install')}`,
  ];
  if (!trimmedCommand.includes(' ')) {
    hints.push(`- ${i18next.t('mcp:stdio.spawn_hint_where', { command: trimmedCommand })}`);
  }
  const enhancedMessage = `${originalMessage}\n${hints.join('\n')}`;
  const enriched = new Error(enhancedMessage);
  (enriched as any)[STDIO_SPAWN_ERROR_FLAG] = true;
  return enriched;
}

export class TauriStdioClientTransport {
  onclose?: () => void;
  onerror?: (error: unknown) => void;
  onmessage?: (message: JsonRpcMessage) => void;

  public sessionId: string | undefined;
  private nativeSessionId: string | null = null;
  private readonly params: TauriStdioServerConfig;
  private unlistenFns: Array<() => void> = [];
  private eventPrefix: string | null = null;
  private serverId?: string; // 用于调试事件关联

  constructor(server: TauriStdioServerConfig, serverId?: string) {
    this.params = server;
    this.serverId = serverId;
  }

  get transport_name() {
    return 'stdio';
  }

  get sessionIdValue(): string | null {
    return this.sessionId;
  }

  async start(): Promise<void> {
    if (!isTauriStdioSupported()) {
      throw new Error(i18next.t('mcp:stdio.desktop_only'));
    }
    if (!this.params.command || !this.params.command.trim()) {
      throw new Error(i18next.t('mcp:stdio.missing_command'));
    }

    try {
      const sessionId = await invoke<string>('mcp_stdio_start', {
        command: this.params.command,
        args: this.params.args ?? [],
        env: this.params.env ?? {},
        framing: this.params.framing ?? 'content_length',
        cwd: this.params.cwd ?? null,
      });

      this.nativeSessionId = sessionId;
      this.sessionId = undefined; // 让 Protocol 按正常流程初始化
      await this.registerListeners(sessionId);
    } catch (error: unknown) {
      const enriched = enrichStdioSpawnError(error, this.params.command);
      this.onerror?.(enriched);
      throw enriched;
    }
  }

  async send(message: JsonRpcMessage): Promise<void> {
    if (!this.nativeSessionId) {
      throw new Error(i18next.t('mcp:stdio.not_initialized'));
    }
    const payload = JSON.stringify(message);

    // 触发发送事件
    emitStdioDebugEvent('mcp-stdio-send', {
      serverId: this.serverId,
      sessionId: this.nativeSessionId,
      payload: summarizeStdioPayload(payload),
    });

    try {
      await invoke('mcp_stdio_send', { sessionId: this.nativeSessionId, payload });
    } catch (error: unknown) {
      this.onerror?.(error);
      throw error;
    }
  }

  async close(): Promise<void> {
    if (!this.nativeSessionId) return;
    try {
      await invoke('mcp_stdio_close', { sessionId: this.nativeSessionId });
    } catch (error: unknown) {
      this.onerror?.(error);
      throw error;
    } finally {
      this.cleanupListeners();
      this.nativeSessionId = null;
      this.sessionId = undefined;
      this.onclose?.();
    }
  }

  is_connected(): boolean {
    return Boolean(this.nativeSessionId);
  }

  private async registerListeners(sessionId: string) {
    const eventPrefix = `mcp-stdio-${sessionId}`;
    this.eventPrefix = eventPrefix;

    const messageUnlisten = await listen(`${eventPrefix}-message`, event => {
      const payload = (event.payload as any) ?? {};
      if (!payload || typeof payload.message !== 'string') return;

      // 触发接收事件
      emitStdioDebugEvent('mcp-stdio-recv', {
        serverId: this.serverId,
        sessionId,
        payload: summarizeStdioPayload(payload.message),
      });

      try {
        const parsed = JSON.parse(payload.message) as JsonRpcMessage;
        this.onmessage?.(parsed);
      } catch (error: unknown) {
        console.warn('[MCP][stdio] Failed to parse message', {
          byteLength: typeof payload.message === 'string' ? payload.message.length : 0,
          error,
        });
        this.onerror?.(error);
      }
    });

    const errorUnlisten = await listen(`${eventPrefix}-error`, event => {
      const payload = (event.payload as any) ?? {};
      const error = payload?.error ?? i18next.t('mcp:stdio.unknown_error');

      // 触发错误事件
      emitStdioDebugEvent('mcp-stdio-error', {
        serverId: this.serverId,
        sessionId,
        error,
      });

      if (this.onerror) {
        this.onerror(error);
      }
    });

    const closeUnlisten = await listen(`${eventPrefix}-closed`, () => {
      // 触发关闭事件
      emitStdioDebugEvent('mcp-stdio-closed', {
        serverId: this.serverId,
        sessionId,
      });

      this.cleanupListeners();
      this.nativeSessionId = null;
      this.sessionId = undefined;
      this.onclose?.();
    });

    this.unlistenFns.push(messageUnlisten, errorUnlisten, closeUnlisten);
  }

  private cleanupListeners() {
    while (this.unlistenFns.length) {
      const dispose = this.unlistenFns.pop();
      try {
        dispose && dispose();
      } catch (e: unknown) {
        console.warn('[MCP][stdio] Failed to remove event listener', e);
      }
    }
    this.eventPrefix = null;
  }

  get nativeSession(): string | null {
    return this.nativeSessionId;
  }
}

function summarizeStdioPayload(payload: string): Record<string, unknown> {
  const summary: Record<string, unknown> = {
    byteLength: new TextEncoder().encode(payload).byteLength,
  };
  try {
    const parsed = JSON.parse(payload) as Record<string, unknown>;
    if (typeof parsed.id === 'string' || typeof parsed.id === 'number') {
      summary.id = parsed.id;
    }
    if (typeof parsed.method === 'string') {
      summary.method = parsed.method;
    }
    if (parsed.params && typeof parsed.params === 'object' && !Array.isArray(parsed.params)) {
      summary.paramKeys = Object.keys(parsed.params as Record<string, unknown>);
    }
    if (parsed.result && typeof parsed.result === 'object' && !Array.isArray(parsed.result)) {
      summary.resultKeys = Object.keys(parsed.result as Record<string, unknown>);
    }
    if (parsed.error && typeof parsed.error === 'object' && !Array.isArray(parsed.error)) {
      const errorObject = parsed.error as Record<string, unknown>;
      summary.errorCode = errorObject.code;
      if (typeof errorObject.message === 'string') {
        summary.errorMessageLength = errorObject.message.length;
      }
    }
  } catch {
    summary.parseable = false;
  }

  if (shouldEmitRawStdioPayload()) {
    summary.rawPayload = payload;
  }
  return summary;
}

function shouldEmitRawStdioPayload(): boolean {
  try {
    return typeof localStorage !== 'undefined' && localStorage.getItem('deep_student_mcp_stdio_debug_raw') === '1';
  } catch {
    return false;
  }
}
