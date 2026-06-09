import { existsSync, readFileSync } from 'node:fs';
import { resolve } from 'node:path';
import { describe, expect, it } from 'vitest';

function readProjectFile(path: string): string {
  const absolutePath = resolve(process.cwd(), path);
  expect(existsSync(absolutePath), `${path} should exist`).toBe(true);
  return readFileSync(absolutePath, 'utf-8');
}

describe('save Anki cards native facade contract', () => {
  it('keeps the local library save path on the native facade payload shape', () => {
    const adapter = readProjectFile('src/services/ankiApiAdapter.ts');

    expect(adapter).toContain("import { invoke } from '@/runtime/native'");
    expect(adapter).not.toContain('@tauri-apps/api/core');
    expect(adapter).toContain("invoke<SaveAnkiCardsResponse>('save_anki_cards'");
    expect(adapter).toContain('request: {');
    expect(adapter).toContain('document_id: params.documentId');
    expect(adapter).toContain('business_session_id: params.businessSessionId');
    expect(adapter).toContain('message_stable_id: params.messageStableId');
    expect(adapter).toContain('block_id: params.blockId');
    expect(adapter).toContain('template_id: params.templateId');
    expect(adapter).toContain('cards: cardsPayload');
    expect(adapter).toContain('front: card.front');
    expect(adapter).toContain('back: card.back');
    expect(adapter).toContain('tags: card.tags');
    expect(adapter).toContain('images: card.images');
    expect(adapter).toContain('fields: card.fields');
  });
});
