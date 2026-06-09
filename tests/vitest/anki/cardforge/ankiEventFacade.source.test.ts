import { readFileSync } from 'node:fs';
import { resolve } from 'node:path';
import { describe, expect, it } from 'vitest';

describe('Anki generation event facade contract', () => {
  const listenerFiles = [
    'src/components/anki/cardforge/engines/CardEngine.ts',
    'src/components/anki/cardforge/engines/CardAgent.ts',
    'src/services/ankiApiAdapter.ts',
    'src/features/chat/adapters/TauriAdapter.ts',
    'src/debug-panel/plugins/ChatAnkiWorkflowDebugPlugin.tsx',
    'src/features/chat/debug/chatAnkiIntegrationTestPlugin.ts',
  ];

  it('listens for anki_generation_event through the native event facade', () => {
    for (const file of listenerFiles) {
      const source = readFileSync(resolve(process.cwd(), file), 'utf-8');
      expect(source).toContain('anki_generation_event');
      expect(source).toContain('@/runtime/nativeEvents');
      expect(source).not.toContain("@tauri-apps/api/event");
    }
  });
});
