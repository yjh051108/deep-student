import { existsSync, readFileSync } from 'node:fs';
import { resolve } from 'node:path';
import { describe, expect, it } from 'vitest';

function readProjectFile(path: string): string {
  const absolutePath = resolve(process.cwd(), path);
  expect(existsSync(absolutePath), `${path} should exist`).toBe(true);
  return readFileSync(absolutePath, 'utf-8');
}

describe('native command triage qbank sync contract', () => {
  it('classifies qbank_update_sync_config as study data instead of generic settings', () => {
    const triageScript = readProjectFile('scripts/native-triage.mjs');
    const triage = JSON.parse(readProjectFile('docs/generated/native-command-triage.json')) as {
      items: Array<{
        name: string;
        domain: string;
        status: string;
        files: string[];
      }>;
    };
    const item = triage.items.find(entry => entry.name === 'qbank_update_sync_config');

    expect(triageScript).toContain("['qbank_update_sync_config', { domain: 'study-data', status: 'merge' }]");
    expect(item).toBeTruthy();
    expect(item?.domain).toBe('study-data');
    expect(item?.status).toBe('merge');
    expect(item?.files).toEqual(['src/stores/questionBankStore.ts']);
  });
});
