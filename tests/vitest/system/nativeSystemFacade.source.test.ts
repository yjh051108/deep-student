import { readFileSync } from 'node:fs';
import { resolve } from 'node:path';
import { describe, expect, it } from 'vitest';

describe('system native facade contract', () => {
  it('routes save_webview_settings through the native facade and Wails bridge', () => {
    const systemApi = readFileSync(resolve(process.cwd(), 'src/utils/systemApi.ts'), 'utf-8');
    const nativeRuntime = readFileSync(resolve(process.cwd(), 'src/runtime/native.ts'), 'utf-8');
    const wailsBridge = readFileSync(resolve(process.cwd(), 'src/runtime/wailsBridge.ts'), 'utf-8');

    expect(systemApi).toContain("nativeInvoke<{ success: boolean; path?: string; size?: number }>('save_webview_settings'");
    expect(systemApi).not.toContain("return invoke<{ success: boolean; path?: string; size?: number }>('save_webview_settings'");
    expect(nativeRuntime).toContain("command === 'save_webview_settings'");
    expect(wailsBridge).toContain("command === 'save_webview_settings'");
    expect(wailsBridge).toContain('SystemService.SaveWebviewSettings');
  });
});
