import { existsSync, readFileSync } from 'node:fs';
import { resolve } from 'node:path';
import { describe, expect, it } from 'vitest';

function readProjectFile(path: string): string {
  const absolutePath = resolve(process.cwd(), path);
  expect(existsSync(absolutePath), `${path} should exist`).toBe(true);
  return readFileSync(absolutePath, 'utf-8');
}

describe('model adapter options native facade contract', () => {
  it('routes the settings modal adapter list through Go/Wails instead of direct Tauri', () => {
    const modal = readProjectFile('src/features/settings/components/ShadApiEditModal.tsx');
    const nativeRuntime = readProjectFile('src/runtime/native.ts');
    const wailsBridge = readProjectFile('src/runtime/wailsBridge.ts');
    const settingsBinding = readProjectFile('desktop-go/internal/bindings/settings_service.go');
    const settingsService = readProjectFile('desktop-go/internal/settings/service.go');
    const generatedBinding = readProjectFile('src/runtime/wails-bindings/deep-student-go/internal/bindings/settingsservice.ts');
    const libRs = readProjectFile('src-tauri/src/lib.rs');
    const ankiCardsRs = readProjectFile('src-tauri/src/cmd/anki_cards.rs');

    expect(modal).toContain("import { invoke } from '@/runtime/native'");
    expect(modal).not.toContain("@tauri-apps/api/core");
    expect(modal).toContain("await invoke('get_model_adapter_options')");
    expect(modal).toContain('fallbackByValue');

    expect(nativeRuntime).toContain("'get_model_adapter_options'");
    expect(nativeRuntime).toContain('fallbackModelAdapterOptions');
    expect(wailsBridge).toContain("command === 'get_model_adapter_options'");
    expect(wailsBridge).toContain('SettingsService.GetModelAdapterOptions');

    expect(settingsBinding).toContain('func (s *SettingsService) GetModelAdapterOptions() []settings.ModelAdapterOption');
    expect(settingsService).toContain('type ModelAdapterOption struct');
    expect(settingsService).toContain('func (s *Service) GetModelAdapterOptions() []ModelAdapterOption');
    expect(generatedBinding).toContain('export function GetModelAdapterOptions');

    expect(libRs).not.toContain('crate::commands::get_model_adapter_options');
    expect(ankiCardsRs).not.toContain('pub async fn get_model_adapter_options');
  });
});
