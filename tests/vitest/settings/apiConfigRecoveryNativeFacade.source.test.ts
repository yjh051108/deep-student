import { readFileSync } from 'node:fs';
import { resolve } from 'node:path';
import { describe, expect, it } from 'vitest';

describe('API config recovery native facade contract', () => {
  it('routes recovery commands through the native facade and Wails bridge', () => {
    const recoverySource = readFileSync(resolve(process.cwd(), 'src/components/ApiConfigRecovery.tsx'), 'utf-8');
    const nativeRuntime = readFileSync(resolve(process.cwd(), 'src/runtime/native.ts'), 'utf-8');
    const wailsBridge = readFileSync(resolve(process.cwd(), 'src/runtime/wailsBridge.ts'), 'utf-8');

    expect(recoverySource).toContain("import { invoke as nativeInvoke } from '@/runtime/native'");
    expect(recoverySource).toContain("nativeInvoke<ConfigStatus>('check_api_config_status')");
    expect(recoverySource).toContain("nativeInvoke<string>('restore_default_api_configs')");
    expect(recoverySource).not.toContain("@tauri-apps/api/core");

    for (const command of ['check_api_config_status', 'restore_default_api_configs']) {
      expect(nativeRuntime).toContain(`command === '${command}'`);
      expect(wailsBridge).toContain(`command === '${command}'`);
    }
    expect(wailsBridge).toContain('SettingsService.CheckAPIConfigStatus');
    expect(wailsBridge).toContain('SettingsService.RestoreDefaultAPIConfigs');
  });
});
