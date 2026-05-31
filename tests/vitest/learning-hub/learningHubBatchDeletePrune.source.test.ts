import { readFileSync } from 'node:fs';
import { resolve } from 'node:path';
import { describe, expect, it } from 'vitest';

describe('LearningHub batch delete prune contract', () => {
  const source = readFileSync(
    resolve(process.cwd(), 'src/features/learning-hub/LearningHubSidebar.tsx'),
    'utf-8'
  );

  it('removes successful batch-deleted resources from finder and recent state immediately', () => {
    expect(source).toContain('const failedIdSet = new Set(failedIds);');
    expect(source).toContain('const succeededResourceEntries = resourceEntries.filter(entry => !failedIdSet.has(entry.id));');
    expect(source).toContain('for (const entry of succeededResourceEntries)');
    expect(source).toContain('removeRecentByIdentity(entry.id, entry.path)');
    expect(source).toContain('pruneFinderResource(entry.id, entry.path)');
  });
});
