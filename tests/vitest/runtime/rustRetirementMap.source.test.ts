import { existsSync, readFileSync } from 'node:fs';
import { resolve } from 'node:path';
import { describe, expect, it } from 'vitest';

function readProjectFile(path: string): string {
  const absolutePath = resolve(process.cwd(), path);
  expect(existsSync(absolutePath), `${path} should exist`).toBe(true);
  return readFileSync(absolutePath, 'utf-8');
}

interface RetirementItem {
  name: string;
  domain: string;
  status: string;
  goBridgeImplemented: boolean;
  rustRegistered: boolean;
  rustDefinitions: unknown[];
  directTauriFiles: string[];
  retirementCandidate: boolean;
}

function retirementItem(name: string): RetirementItem {
  const map = JSON.parse(readProjectFile('docs/generated/rust-retirement-map.json')) as {
    items: RetirementItem[];
  };
  const item = map.items.find(entry => entry.name === name);
  expect(item, `${name} should be present in rust-retirement-map.json`).toBeTruthy();
  return item as RetirementItem;
}

describe('Rust retirement map source contract', () => {
  it('counts only Tauri generate_handler entries as command registrations', () => {
    const script = readProjectFile('scripts/rust-retirement-map.mjs');
    const libRs = readProjectFile('src-tauri/src/lib.rs');

    expect(script).toContain('function generateHandlerBodies');
    expect(script).toContain('function commandRegistrationsInGenerateHandler');
    expect(script).toContain("const marker = 'tauri::generate_handler!['");

    expect(libRs).toContain('.get_setting(');
    expect(retirementItem('get_setting').rustRegistered).toBe(false);

    expect(libRs).toContain('recover_stuck_document_tasks()');
    expect(retirementItem('recover_stuck_document_tasks').rustRegistered).toBe(false);
  });

  it('keeps memory_get_config retired from Rust while preserving the Go/Wails route', () => {
    const libRs = readProjectFile('src-tauri/src/lib.rs');
    const memoryHandlers = readProjectFile('src-tauri/src/memory/handlers.rs');
    const wailsBridge = readProjectFile('src/runtime/wailsBridge.ts');

    expect(libRs).not.toContain('crate::memory::handlers::memory_get_config');
    expect(memoryHandlers).not.toContain('pub async fn memory_get_config');

    const memoryConfig = retirementItem('memory_get_config');
    expect(memoryConfig.rustRegistered).toBe(false);
    expect(memoryConfig.rustDefinitions).toEqual([]);

    expect(wailsBridge).toContain("command === 'memory_get_config'");
    expect(wailsBridge).toContain('SettingsService.GetMemoryConfig');
  });

  it('keeps qbank_update_sync_config retired from Rust while preserving the Go/Wails route', () => {
    const libRs = readProjectFile('src-tauri/src/lib.rs');
    const questionSync = readProjectFile('src-tauri/src/question_sync_service.rs');
    const wailsBridge = readProjectFile('src/runtime/wailsBridge.ts');

    expect(libRs).not.toContain('crate::question_sync_service::qbank_update_sync_config');
    expect(questionSync).not.toContain('pub async fn qbank_update_sync_config');

    const qbankSyncConfig = retirementItem('qbank_update_sync_config');
    expect(qbankSyncConfig.domain).toBe('study-data');
    expect(qbankSyncConfig.status).toBe('merge');
    expect(qbankSyncConfig.goBridgeImplemented).toBe(true);
    expect(qbankSyncConfig.rustRegistered).toBe(false);
    expect(qbankSyncConfig.rustDefinitions).toEqual([]);
    expect(qbankSyncConfig.directTauriFiles).toEqual([]);
    expect(qbankSyncConfig.retirementCandidate).toBe(false);

    expect(wailsBridge).toContain("command === 'qbank_update_sync_config'");
    expect(wailsBridge).toContain('QbankService.UpdateSyncConfig');
  });

  it('keeps save_anki_cards retired from Rust while preserving the Go/Wails route', () => {
    const libRs = readProjectFile('src-tauri/src/lib.rs');
    const ankiConnect = readProjectFile('src-tauri/src/cmd/anki_connect.rs');
    const wailsBridge = readProjectFile('src/runtime/wailsBridge.ts');

    expect(libRs).not.toContain('crate::commands::save_anki_cards');
    expect(ankiConnect).not.toContain('pub async fn save_anki_cards');

    const saveAnkiCards = retirementItem('save_anki_cards');
    expect(saveAnkiCards.domain).toBe('study-data');
    expect(saveAnkiCards.status).toBe('merge');
    expect(saveAnkiCards.goBridgeImplemented).toBe(true);
    expect(saveAnkiCards.rustRegistered).toBe(false);
    expect(saveAnkiCards.rustDefinitions).toEqual([]);
    expect(saveAnkiCards.directTauriFiles).toEqual([]);
    expect(saveAnkiCards.retirementCandidate).toBe(false);

    expect(wailsBridge).toContain("command === 'save_anki_cards'");
    expect(wailsBridge).toContain('AnkiService.SaveAnkiCards');
  });
});
