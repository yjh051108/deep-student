import { readFileSync } from 'node:fs';
import { resolve } from 'node:path';
import { describe, expect, it } from 'vitest';

describe('LearningHub deleted tab identity contract', () => {
  const source = readFileSync(
    resolve(process.cwd(), 'src/features/learning-hub/LearningHubPage.tsx'),
    'utf-8'
  );

  it('closes deleted resource tabs by every stable identity from the event', () => {
    expect(source).toContain('const getDstuResourceIdFromPath = (path?: string | null): string | null =>');
    expect(source).toContain('const collectAffectedDstuIds = (event:');
    expect(source).toContain('event.node?.resourceId');
    expect(source).toContain('event.node?.sourceId');
    expect(source).toContain('getDstuResourceIdFromPath(event.oldPath)');
    expect(source).toContain('const affectedIds = collectAffectedDstuIds(event);');
    expect(source).toContain('prev.tabs.filter(tab => !affectedIds.has(tab.resourceId))');
    expect(source).not.toContain('const affectedResourceId = event.id ?? getDstuResourceIdFromPath(affectedPath);');
    expect(source).not.toContain("affectedPath.split('/').filter(Boolean).pop()");
  });
});
