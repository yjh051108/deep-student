import { existsSync, readFileSync } from 'node:fs';
import { resolve } from 'node:path';
import { describe, expect, it } from 'vitest';

function readProjectFile(path: string): string {
  const absolutePath = resolve(process.cwd(), path);
  expect(existsSync(absolutePath), `${path} should exist`).toBe(true);
  return readFileSync(absolutePath, 'utf-8');
}

describe('save_anki_cards Wails bridge contract', () => {
  it('routes local Anki card saves through Go/Wails instead of old Rust', () => {
    const wailsBridge = readProjectFile('src/runtime/wailsBridge.ts');
    const bindingGo = readProjectFile('desktop-go/internal/bindings/anki_service.go');
    const serviceGo = readProjectFile('desktop-go/internal/anki/service.go');
    const generatedBinding = readProjectFile('src/runtime/wails-bindings/deep-student-go/internal/bindings/ankiservice.ts');
    const generatedModels = readProjectFile('src/runtime/wails-bindings/deep-student-go/internal/anki/models.ts');
    const libRs = readProjectFile('src-tauri/src/lib.rs');
    const ankiConnectRs = readProjectFile('src-tauri/src/cmd/anki_connect.rs');

    expect(wailsBridge).toContain("command === 'save_anki_cards'");
    expect(wailsBridge).toContain("requireObjectArg(command, args, 'request')");
    expect(wailsBridge).toContain('AnkiService.SaveAnkiCards(request as any)');
    expect(wailsBridge).toContain('savedIds: Array.isArray(result?.saved_ids)');
    expect(wailsBridge).toContain("taskId: typeof result?.task_id === 'string'");

    expect(bindingGo).toContain('func (s *AnkiService) SaveAnkiCards(request anki.SaveAnkiCardsRequest) (anki.SaveAnkiCardsResponse, error)');
    expect(serviceGo).toContain('func (s *Service) SaveAnkiCards(request SaveAnkiCardsRequest) (SaveAnkiCardsResponse, error)');
    expect(serviceGo).toContain('"source":        "go_save_anki_cards"');
    expect(generatedBinding).toContain('export function SaveAnkiCards(request: anki$0.SaveAnkiCardsRequest)');
    expect(generatedModels).toContain('export class SaveAnkiCardsRequest');
    expect(generatedModels).toContain('export class SaveAnkiCardsResponse');

    expect(libRs).not.toContain('crate::commands::save_anki_cards');
    expect(ankiConnectRs).not.toContain('pub async fn save_anki_cards');
  });
});
