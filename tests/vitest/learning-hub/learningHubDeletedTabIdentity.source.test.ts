import { readFileSync } from 'node:fs';
import { resolve } from 'node:path';
import { describe, expect, it } from 'vitest';

describe('LearningHub deleted tab identity contract', () => {
  const source = readFileSync(
    resolve(process.cwd(), 'src/features/learning-hub/LearningHubPage.tsx'),
    'utf-8'
  );

  it('closes deleted resource tabs by explicit event id before path fallback', () => {
    expect(source).toContain('const getDstuResourceIdFromPath = (path?: string | null): string | null =>');
    expect(source).toContain('const affectedPath = event.path || event.oldPath;');
    expect(source).toContain('const affectedResourceId = event.id ?? getDstuResourceIdFromPath(affectedPath);');
    expect(source).toContain('prev.tabs.filter(tab => tab.resourceId !== affectedResourceId)');
    expect(source).not.toContain("affectedPath.split('/').filter(Boolean).pop()");
  });
});
