import { readFileSync } from 'node:fs';
import { resolve } from 'node:path';
import { describe, expect, it } from 'vitest';

describe('data governance native facade contract', () => {
  it('routes backup config through the native facade and Wails bridge', () => {
    const apiSource = readFileSync(resolve(process.cwd(), 'src/api/dataGovernance.ts'), 'utf-8');
    const nativeRuntime = readFileSync(resolve(process.cwd(), 'src/runtime/native.ts'), 'utf-8');
    const wailsBridge = readFileSync(resolve(process.cwd(), 'src/runtime/wailsBridge.ts'), 'utf-8');

    expect(apiSource).toContain("import { invoke as nativeInvoke } from '@/runtime/native'");
    expect(apiSource).toContain("nativeInvoke<BackupConfig>('get_backup_config')");
    expect(apiSource).toContain("nativeInvoke<void>('set_backup_config'");
    expect(apiSource).not.toContain("return invoke<BackupConfig>('get_backup_config')");
    expect(apiSource).not.toContain("return invoke<void>('set_backup_config'");

    for (const command of ['get_backup_config', 'set_backup_config']) {
      expect(nativeRuntime).toContain(`command === '${command}'`);
      expect(wailsBridge).toContain(`command === '${command}'`);
    }
    expect(wailsBridge).toContain('SettingsService.GetBackupConfig');
    expect(wailsBridge).toContain('SettingsService.SetBackupConfig');
  });
});
