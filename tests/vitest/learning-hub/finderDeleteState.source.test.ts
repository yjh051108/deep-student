import { readFileSync } from 'node:fs';
import { resolve } from 'node:path';
import { describe, expect, it } from 'vitest';

describe('learning hub delete state contract', () => {
  const sidebarSource = readFileSync(
    resolve(process.cwd(), 'src/features/learning-hub/LearningHubSidebar.tsx'),
    'utf-8'
  );
  const recentStoreSource = readFileSync(
    resolve(process.cwd(), 'src/features/learning-hub/stores/recentStore.ts'),
    'utf-8'
  );

  it('removes deleted resources from visible finder state and recent state by id or path', () => {
    expect(recentStoreSource).toContain('removeRecentByIdentity');
    expect(sidebarSource).toContain('const pruneFinderResource = (resourceId: string, path?: string | null)');
    expect(sidebarSource).toContain('item.resourceId !== resourceId');
    expect(sidebarSource).toContain('(!path || item.path !== path)');
    expect(sidebarSource).toContain('removeRecentByIdentity(resourceId, event.path)');
    expect(sidebarSource).toContain('pruneFinderResource(resourceId, event.path)');
    expect(sidebarSource).toContain('const visiblePath = resourcePath ?? deletePath');
    expect(sidebarSource).toContain('removeRecentByIdentity(resource.id, visiblePath)');
    expect(sidebarSource).toContain('pruneFinderResource(resource.id, visiblePath)');
  });

  it('deletes resources by stable id instead of display path', () => {
    expect(sidebarSource).toContain('const resourcePath = resource.path ?? items.find');
    expect(sidebarSource).toContain('const deletePath = `/${resource.id}`');
    expect(sidebarSource).toContain('const deleteResult = await dstu.delete(deletePath)');
    expect(sidebarSource).not.toContain('let deletePath = resource.path');
  });
});
