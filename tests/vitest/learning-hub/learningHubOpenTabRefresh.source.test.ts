import { readFileSync } from 'node:fs';
import { resolve } from 'node:path';
import { describe, expect, it } from 'vitest';

describe('LearningHub open tab refresh contract', () => {
  it('refreshes existing tab metadata and expands the local resource pane when opening a resource', () => {
    const source = readFileSync(
      resolve(process.cwd(), 'src/features/learning-hub/LearningHubPage.tsx'),
      'utf-8'
    );

    expect(source).toContain('setLocalSidebarCollapsed(false);');
    expect(source).toContain('return prev.map(t => t.tabId === existing.tabId ? { ...t, ...app, openedAt: Date.now() } : t);');
    expect(source).not.toContain('return prev.map(t => t.tabId === existing.tabId ? { ...t, openedAt: Date.now() } : t);');
  });
});
