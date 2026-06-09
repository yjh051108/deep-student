import { existsSync, readFileSync } from 'node:fs';
import { resolve } from 'node:path';
import { describe, expect, it } from 'vitest';

function readProjectFile(path: string): string {
  const absolutePath = resolve(process.cwd(), path);
  expect(existsSync(absolutePath), `${path} should exist`).toBe(true);
  return readFileSync(absolutePath, 'utf-8');
}

describe('qbank sync config native facade contract', () => {
  it('routes qbank_update_sync_config through the Go/Wails qbank service', () => {
    const store = readProjectFile('src/stores/questionBankStore.ts');
    const nativeRuntime = readProjectFile('src/runtime/native.ts');
    const wailsBridge = readProjectFile('src/runtime/wailsBridge.ts');
    const bindingGo = readProjectFile('desktop-go/internal/bindings/qbank_service.go');
    const serviceGo = readProjectFile('desktop-go/internal/qbank/service.go');
    const generatedBinding = readProjectFile('src/runtime/wails-bindings/deep-student-go/internal/bindings/qbankservice.ts');
    const generatedModels = readProjectFile('src/runtime/wails-bindings/deep-student-go/internal/qbank/models.ts');
    const libRs = readProjectFile('src-tauri/src/lib.rs');
    const questionSyncRs = readProjectFile('src-tauri/src/question_sync_service.rs');

    expect(store).toContain("import { invoke } from '@/runtime/native'");
    expect(store).not.toContain("@tauri-apps/api/core");
    expect(store).toContain("await invoke('qbank_update_sync_config'");

    expect(nativeRuntime).toContain("'qbank_update_sync_config'");
    const fallbackBeforeTauriBlock = nativeRuntime.match(/const mcpFallbackBeforeTauriCommands = new Set<string>\(\[[\s\S]*?\]\);/)?.[0] ?? '';
    expect(fallbackBeforeTauriBlock).toContain("'get_model_adapter_options'");
    expect(fallbackBeforeTauriBlock).not.toContain("'qbank_update_sync_config'");
    expect(nativeRuntime).toContain("const fallbackQbankSyncConfigsKey = 'go_qbank_sync_configs'");
    expect(nativeRuntime).toContain("command === 'qbank_update_sync_config'");
    expect(nativeRuntime).toContain("requireFallbackStringArg(command, args, 'examId')");
    expect(nativeRuntime).toContain("requireFallbackObjectArg(command, args, 'config')");
    expect(nativeRuntime).toContain('normalizeFallbackQbankSyncConfig(config)');
    expect(nativeRuntime).toContain('saveFallbackQbankSyncConfigs(configs)');

    expect(wailsBridge).toContain("command === 'qbank_update_sync_config'");
    expect(wailsBridge).toContain("requireStringArg(command, args, 'examId')");
    expect(wailsBridge).toContain("requireObjectArg(command, args, 'config')");
    expect(wailsBridge).toContain('QbankService.UpdateSyncConfig(examId');

    expect(bindingGo).toContain('func (s *QbankService) UpdateSyncConfig(examID string, config qbank.SyncConfig) (bool, error)');
    expect(serviceGo).toContain('func (s *Service) UpdateSyncConfig(examID string, config SyncConfig) error');
    expect(generatedBinding).toContain('export function UpdateSyncConfig(examID: string, config: qbank$0.SyncConfig)');
    expect(generatedModels).toContain('export class SyncConfig');

    expect(libRs).not.toContain('crate::question_sync_service::qbank_update_sync_config');
    expect(questionSyncRs).not.toContain('pub async fn qbank_update_sync_config');
  });
});
