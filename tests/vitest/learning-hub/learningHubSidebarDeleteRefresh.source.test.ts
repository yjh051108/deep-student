import { readFileSync } from 'node:fs';
import { resolve } from 'node:path';
import { describe, expect, it } from 'vitest';

describe('LearningHubSidebar delete refresh contract', () => {
  const source = readFileSync(
    resolve(process.cwd(), 'src/features/learning-hub/LearningHubSidebar.tsx'),
    'utf-8'
  );

  it('does not patch finder items directly after delete events', () => {
    expect(source).not.toContain('removeResourcesFromRecentAndFinder');
    expect(source).not.toContain('finderState.setItems(finderState.items.filter');
    expect(source).toContain('event.id ?? getDstuResourceIdFromPath(event.path)');
    expect(source).toContain('useRecentStore.getState().removeRecent(resourceId)');
    expect(source).toContain('handleRefresh();');
  });
});
