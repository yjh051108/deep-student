import { readFileSync } from 'node:fs';
import { resolve } from 'node:path';
import { describe, expect, it } from 'vitest';

const filesWithoutRuntimeInvoke = [
  'src/components/BatchOperationToolbar/index.tsx',
  'src/components/shared/UnifiedDragDropZone.tsx',
  'src/features/chat/adapters/contextHelper.ts',
  'src/features/chat/components/input-bar/InputBarV2.tsx',
  'src/features/chat/plugins/blocks/subagentEmbed.tsx',
];

describe('unused direct Tauri invoke imports', () => {
  it('keeps files with no invoke calls from importing Tauri core invoke', () => {
    for (const file of filesWithoutRuntimeInvoke) {
      const source = readFileSync(resolve(process.cwd(), file), 'utf-8');
      expect(source).not.toContain("import { invoke } from '@tauri-apps/api/core'");
      expect(source).not.toContain('import { invoke } from "@tauri-apps/api/core"');
    }
  });
});
