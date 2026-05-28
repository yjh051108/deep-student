import { readFileSync } from 'node:fs';
import { resolve } from 'node:path';
import { describe, expect, it } from 'vitest';

describe('LearningHubPage sidebar collapse contract', () => {
  const source = readFileSync(
    resolve(process.cwd(), 'src/features/learning-hub/LearningHubPage.tsx'),
    'utf-8'
  );

  it('expands the global left panel when a resource app is open on desktop', () => {
    expect(source).toContain('const globalLeftPanelCollapsed = useUIStore((state) => state.leftPanelCollapsed);');
    expect(source).toContain('if (hasOpenApp && globalLeftPanelCollapsed && !isSmallScreen)');
    expect(source).toContain('useUIStore.getState().setLeftPanelCollapsed(false);');
  });

  it('does not repeatedly squeeze the resource tree when switching or opening more tabs', () => {
    expect(source).toContain('const hadOpenAppRef = useRef(false);');
    expect(source).toContain('const wasOpen = hadOpenAppRef.current;');
    expect(source).toContain('if (wasOpen) {');
    expect(source).toContain('return;');
    expect(source).toContain('sidebarPanelRef.current?.resize(35);');
    expect(source).toContain('appPanelRef.current?.resize(65);');
    expect(source).not.toContain('sidebarPanelRef.current?.resize(25);');
    expect(source).not.toContain('appPanelRef.current?.resize(75);');
  });
});
