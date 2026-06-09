import { readFileSync } from 'node:fs';
import { resolve } from 'node:path';
import { describe, expect, it } from 'vitest';

describe('AnkiConnect native facade contract', () => {
  it('routes readonly AnkiConnect metadata through native/Wails instead of direct Tauri imports', () => {
    const client = readFileSync(resolve(process.cwd(), 'src/services/ankiConnectClient.ts'), 'utf-8');
    const settingsSection = readFileSync(resolve(process.cwd(), 'src/features/settings/components/AnkiConnectSettingsSection.tsx'), 'utf-8');
    const ankiCardsBlock = readFileSync(resolve(process.cwd(), 'src/features/chat/plugins/blocks/ankiCardsBlock.tsx'), 'utf-8');
    const nativeRuntime = readFileSync(resolve(process.cwd(), 'src/runtime/native.ts'), 'utf-8');
    const wailsBridge = readFileSync(resolve(process.cwd(), 'src/runtime/wailsBridge.ts'), 'utf-8');

    expect(client).toContain("nativeInvoke<boolean>('check_anki_connect_status')");
    expect(client).toContain("nativeInvoke<string[]>('get_anki_deck_names')");
    expect(client).toContain("nativeInvoke<string[]>('get_anki_model_names')");
    expect(settingsSection).toContain('ankiConnectClient.listDecks()');
    expect(settingsSection).toContain('ankiConnectClient.listModels()');
    expect(settingsSection).not.toContain("import('@tauri-apps/api/core')");
    expect(ankiCardsBlock).toContain("import { ankiConnectClient } from '@/services/ankiConnectClient'");
    expect(ankiCardsBlock).toContain('ankiConnectClient.check()');
    expect(ankiCardsBlock).not.toContain("invoke<boolean>('check_anki_connect_status')");
    expect(nativeRuntime).toContain("command === 'check_anki_connect_status'");
    expect(nativeRuntime).toContain("command === 'get_anki_model_names'");
    expect(wailsBridge).toContain("command === 'check_anki_connect_status'");
    expect(wailsBridge).toContain("command === 'get_anki_model_names'");
    expect(wailsBridge).toContain('AnkiService.ListModelNames');
  });
});
