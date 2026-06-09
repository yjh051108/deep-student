import { existsSync, readFileSync } from 'node:fs';
import { resolve } from 'node:path';
import { describe, expect, it } from 'vitest';

describe('Wails SkillService bridge source contract', () => {
  const bridgeSource = readFileSync(resolve(process.cwd(), 'src/runtime/wailsBridge.ts'), 'utf-8');
  const apiSource = readFileSync(resolve(process.cwd(), 'src/features/chat/skills/api.ts'), 'utf-8');
  const loaderSource = readFileSync(resolve(process.cwd(), 'src/features/chat/skills/loader.ts'), 'utf-8');
  const bindingPath = resolve(
    process.cwd(),
    'src/runtime/wails-bindings/deep-student-go/internal/bindings/skillservice.ts',
  );

  it('has generated SkillService bindings and Wails bridge routes for every skill command', () => {
    expect(existsSync(bindingPath)).toBe(true);
    expect(bridgeSource).toContain("bindings/skillservice");

    const routes = [
      ['skill_list_directories', 'SkillService.ListDirectories'],
      ['skill_read_file', 'SkillService.ReadFile'],
      ['skill_create', 'SkillService.Create'],
      ['skill_update', 'SkillService.Update'],
      ['skill_delete', 'SkillService.Delete'],
    ] as const;

    for (const [command, serviceCall] of routes) {
      expect(bridgeSource).toContain(command);
      expect(bridgeSource).toContain(serviceCall);
    }
  });

  it('keeps chat skills callers on the runtime native facade', () => {
    expect(apiSource).toContain("from '@/runtime/native'");
    expect(loaderSource).toContain("from '@/runtime/native'");
    expect(loaderSource).toContain('getAppDataDir');

    for (const source of [apiSource, loaderSource]) {
      expect(source).not.toContain('@tauri-apps/api/core');
      expect(source).not.toContain('@tauri-apps/api/path');
    }
  });
});
