import { readFileSync } from 'node:fs';
import { resolve } from 'node:path';
import { describe, expect, it } from 'vitest';

describe('web search diagnostics native facade contract', () => {
  it('routes visible search diagnostics and provider strategies through native and Wails instead of direct Tauri invoke', () => {
    const settingsApi = readFileSync(resolve(process.cwd(), 'src/utils/settingsApi.ts'), 'utf-8');
    const engineSettings = readFileSync(resolve(process.cwd(), 'src/features/settings/components/EngineSettingsSection.tsx'), 'utf-8');
    const searchStatus = readFileSync(resolve(process.cwd(), 'src/components/SearchEngineStatus.tsx'), 'utf-8');
    const nativeRuntime = readFileSync(resolve(process.cwd(), 'src/runtime/native.ts'), 'utf-8');
    const wailsBridge = readFileSync(resolve(process.cwd(), 'src/runtime/wailsBridge.ts'), 'utf-8');

    expect(settingsApi).toContain("nativeInvoke<SearchEngineTestResponse>('test_search_engine'");
    expect(settingsApi).toContain("nativeInvoke<any>('test_web_search_connectivity'");
    expect(settingsApi).toContain("nativeInvoke<SearchEngineHealthReport>('test_all_search_engines'");
    expect(settingsApi).not.toContain("invoke<any>('test_web_search_connectivity'");
    expect(settingsApi).not.toContain("invoke('test_all_search_engines'");

    expect(engineSettings).toContain("import { testSearchEngine } from '@/utils/settingsApi'");
    expect(engineSettings).toContain("import { invoke as nativeInvoke } from '@/runtime/native'");
    expect(engineSettings).toContain('testSearchEngine(id)');
    expect(engineSettings).toContain("nativeInvoke<{ provider_strategies?: ProviderStrategiesMap } | null>('get_provider_strategies_config'");
    expect(engineSettings).toContain("nativeInvoke('save_provider_strategies_config', { strategies: providerStrategies })");
    expect(engineSettings).not.toContain("from '@tauri-apps/api/core'");
    expect(engineSettings).not.toContain("invoke('test_search_engine'");
    expect(searchStatus).toContain("import { testSearchEngine } from '@/utils/settingsApi'");
    expect(searchStatus).not.toContain("from '@tauri-apps/api/core'");
    expect(searchStatus).not.toContain("invoke<{ok: boolean, message: string, response_time?: number}>('test_search_engine'");

    expect(nativeRuntime).toContain("| 'test_search_engine'");
    expect(nativeRuntime).toContain("| 'test_web_search_connectivity'");
    expect(nativeRuntime).toContain("| 'test_all_search_engines'");
    expect(nativeRuntime).toContain("| 'get_provider_strategies_config'");
    expect(nativeRuntime).toContain("| 'save_provider_strategies_config'");
    expect(wailsBridge).toContain("command === 'test_search_engine'");
    expect(wailsBridge).toContain("command === 'test_web_search_connectivity'");
    expect(wailsBridge).toContain("command === 'test_all_search_engines'");
    expect(wailsBridge).toContain("command === 'get_provider_strategies_config'");
    expect(wailsBridge).toContain("command === 'save_provider_strategies_config'");
    expect(wailsBridge).toContain('SettingsService.TestSearchEngine');
    expect(wailsBridge).toContain('SettingsService.TestWebSearchConnectivity');
    expect(wailsBridge).toContain('SettingsService.TestAllSearchEngines');
    expect(wailsBridge).toContain('SettingsService.GetProviderStrategiesConfig');
    expect(wailsBridge).toContain('SettingsService.SaveProviderStrategiesConfig');
  });
});
