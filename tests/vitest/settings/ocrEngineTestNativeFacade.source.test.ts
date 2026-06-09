import { readFileSync } from 'node:fs';
import { resolve } from 'node:path';
import { describe, expect, it } from 'vitest';

describe('OCR engine test native facade contract', () => {
  it('routes settings OCR diagnostics through native and Wails', () => {
    const panel = readFileSync(resolve(process.cwd(), 'src/features/settings/components/OcrEngineTestPanel.tsx'), 'utf-8');
    const nativeRuntime = readFileSync(resolve(process.cwd(), 'src/runtime/native.ts'), 'utf-8');
    const wailsBridge = readFileSync(resolve(process.cwd(), 'src/runtime/wailsBridge.ts'), 'utf-8');
    const generatedSettings = readFileSync(
      resolve(process.cwd(), 'src/runtime/wails-bindings/deep-student-go/internal/bindings/settingsservice.ts'),
      'utf-8'
    );
    const triage = readFileSync(resolve(process.cwd(), 'scripts/native-triage.mjs'), 'utf-8');

    expect(panel).toContain("from '@/runtime/native'");
    expect(panel).not.toContain("from '@tauri-apps/api/core'");
    expect(panel).toContain("invoke<OcrTestResponse>('test_ocr_engine'");
    expect(panel).toContain('request: {');
    expect(panel).toContain('imageBase64: selectedImage');
    expect(panel).toContain('engineType: model.engineType');
    expect(panel).toContain('configId: model.configId');

    expect(nativeRuntime).toContain("'test_ocr_engine'");
    expect(wailsBridge).toContain("command === 'test_ocr_engine'");
    expect(wailsBridge).toContain('SettingsService.TestOCREngine');
    expect(generatedSettings).toContain('export function TestOCREngine');
    expect(triage).toContain("['test_ocr_engine', { domain: 'settings', status: 'merge' }]");
  });
});
