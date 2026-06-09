import { readFileSync } from 'node:fs';
import { resolve } from 'node:path';
import { describe, expect, it } from 'vitest';

describe('retired deep research settings facade contract', () => {
  it('does not expose retired settingsApi research wrappers or direct Tauri research invokes', () => {
    const settingsApi = readFileSync(resolve(process.cwd(), 'src/utils/settingsApi.ts'), 'utf-8');

    expect(settingsApi).not.toMatch(/export\s+async\s+function\s+research[A-Z]/);
    expect(settingsApi).not.toMatch(/invoke(?:<[^>]+>)?\(\s*['"]research_/);
    expect(settingsApi).toContain('Deep Research command wrappers were retired');
  });
});
