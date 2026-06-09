import { readFileSync } from 'node:fs';
import { resolve } from 'node:path';
import { describe, expect, it } from 'vitest';

describe('memory config native facade contract', () => {
  it('routes memory_get_config through native and Wails while leaving broader memory APIs explicit', () => {
    const memoryApi = readFileSync(resolve(process.cwd(), 'src/api/memoryApi.ts'), 'utf-8');
    const nativeRuntime = readFileSync(resolve(process.cwd(), 'src/runtime/native.ts'), 'utf-8');
    const wailsBridge = readFileSync(resolve(process.cwd(), 'src/runtime/wailsBridge.ts'), 'utf-8');

    expect(memoryApi).toContain("nativeInvoke<MemoryConfig>('memory_get_config')");
    expect(memoryApi).not.toContain("return invoke<MemoryConfig>('memory_get_config')");
    expect(nativeRuntime).toContain("command === 'memory_get_config'");
    expect(wailsBridge).toContain("command === 'memory_get_config'");
    expect(wailsBridge).toContain('SettingsService.GetMemoryConfig');
  });
});
