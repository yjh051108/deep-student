import { afterEach, describe, expect, it, vi } from 'vitest';

const mockTauriInvoke = vi.hoisted(() => vi.fn());
const mockInvokeWails = vi.hoisted(() => vi.fn());

vi.mock('@tauri-apps/api/core', () => ({
  invoke: mockTauriInvoke,
}));

vi.mock('@/runtime/wailsBridge', () => ({
  invokeWails: mockInvokeWails,
  isWailsRuntime: () => false,
}));

function setWindowProperty(name: string, value: unknown): () => void {
  const previous = Object.getOwnPropertyDescriptor(window, name);
  Object.defineProperty(window, name, {
    configurable: true,
    writable: true,
    value,
  });

  return () => {
    if (previous) {
      Object.defineProperty(window, name, previous);
    } else {
      delete (window as unknown as Record<string, unknown>)[name];
    }
  };
}

describe('native MCP Tauri compatibility fallback', () => {
  afterEach(() => {
    localStorage.clear();
    mockTauriInvoke.mockReset();
    mockInvokeWails.mockReset();
    vi.resetModules();
  });

  it('handles retired MCP compatibility commands without calling Tauri invoke', async () => {
    const restoreTauri = setWindowProperty('__TAURI_INTERNALS__', {});
    localStorage.setItem('mcp.tools.list', JSON.stringify([{ name: 'tool-a' }, { name: 'tool-b' }]));
    localStorage.setItem('session.selected_mcp_tools', 'tool-a');
    localStorage.setItem('mcp.tools.namespace_prefix', 'study');
    localStorage.setItem('mcp.tools.conflict_resolution', 'prefer_existing');

    try {
      const { invoke, isTauriRuntime } = await import('@/runtime/native');

      expect(isTauriRuntime()).toBe(true);
      await expect(invoke('preheat_mcp_tools')).resolves.toEqual({ ok: true, count: 2 });
      await expect(invoke('get_mcp_tools')).resolves.toEqual([]);
      await expect(invoke('reload_mcp_client')).resolves.toEqual({
        success: true,
        message: 'Backend MCP disabled; frontend SDK in use',
      });
      await expect(invoke('get_mcp_status')).resolves.toMatchObject({
        available: false,
        enabled: true,
        connected: false,
        last_error: 'backend_mcp_disabled',
        namespace_prefix: 'study',
        conflict_resolution: 'prefer_existing',
      });

      expect(mockTauriInvoke).not.toHaveBeenCalled();
      expect(mockInvokeWails).not.toHaveBeenCalled();
    } finally {
      restoreTauri();
    }
  });

  it('still routes unrelated commands through Tauri when only Tauri is available', async () => {
    const restoreTauri = setWindowProperty('__TAURI_INTERNALS__', {});
    mockTauriInvoke.mockResolvedValue('native-value');

    try {
      const { invoke } = await import('@/runtime/native');

      await expect(invoke('get_setting', { key: 'x' })).resolves.toBe('native-value');

      expect(mockTauriInvoke).toHaveBeenCalledWith('get_setting', { key: 'x' });
      expect(mockInvokeWails).not.toHaveBeenCalled();
    } finally {
      restoreTauri();
    }
  });

  it('does not use browser fallback for retired qbank sync config in Tauri runtime', async () => {
    const restoreTauri = setWindowProperty('__TAURI_INTERNALS__', {});
    mockTauriInvoke.mockResolvedValue(true);

    try {
      const { invoke } = await import('@/runtime/native');

      await expect(invoke('qbank_update_sync_config', {
        examId: 'exam_sync',
        config: {
          default_strategy: 'keep_local',
          auto_sync: true,
          sync_interval_secs: 900,
          sync_progress: false,
          sync_notes: false,
        },
      })).resolves.toBe(true);

      expect(mockTauriInvoke).toHaveBeenCalledWith('qbank_update_sync_config', {
        examId: 'exam_sync',
        config: {
          default_strategy: 'keep_local',
          auto_sync: true,
          sync_interval_secs: 900,
          sync_progress: false,
          sync_notes: false,
        },
      });
      expect(mockInvokeWails).not.toHaveBeenCalled();
      expect(localStorage.getItem('go_qbank_sync_configs')).toBeNull();
    } finally {
      restoreTauri();
    }
  });
});
