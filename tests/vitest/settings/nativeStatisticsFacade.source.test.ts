import { readFileSync } from 'node:fs';
import { resolve } from 'node:path';
import { describe, expect, it } from 'vitest';

describe('statistics native facade contract', () => {
  it('routes statistics commands through native and Wails instead of direct Tauri invoke', () => {
    const settingsApi = readFileSync(resolve(process.cwd(), 'src/utils/settingsApi.ts'), 'utf-8');
    const nativeRuntime = readFileSync(resolve(process.cwd(), 'src/runtime/native.ts'), 'utf-8');
    const wailsBridge = readFileSync(resolve(process.cwd(), 'src/runtime/wailsBridge.ts'), 'utf-8');

    expect(settingsApi).toContain("nativeInvoke<any>('get_statistics')");
    expect(settingsApi).toContain("nativeInvoke<any>('get_enhanced_statistics')");
    expect(settingsApi).not.toContain("invoke<any>('get_statistics')");
    expect(settingsApi).not.toContain("invoke<any>('get_enhanced_statistics')");
    expect(nativeRuntime).toContain("command === 'get_statistics'");
    expect(nativeRuntime).toContain("command === 'get_enhanced_statistics'");
    expect(wailsBridge).toContain("command === 'get_statistics'");
    expect(wailsBridge).toContain("command === 'get_enhanced_statistics'");
    expect(wailsBridge).toContain('SettingsService.GetStatistics');
    expect(wailsBridge).toContain('SettingsService.GetEnhancedStatistics');
  });
});
