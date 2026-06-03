import { readFileSync } from 'node:fs';
import { resolve } from 'node:path';
import { describe, expect, it } from 'vitest';

describe('FinderFileList responsive grid contract', () => {
  it('keeps real finder grid tiles from overlapping in narrow panes', () => {
    const listSource = readFileSync(
      resolve(process.cwd(), 'src/features/learning-hub/components/finder/FinderFileList.tsx'),
      'utf-8'
    );
    const itemSource = readFileSync(
      resolve(process.cwd(), 'src/features/learning-hub/components/finder/FinderFileItem.tsx'),
      'utf-8'
    );

    expect(listSource).toContain('gridTemplateColumns: `repeat(auto-fill, minmax(min(${GRID_ITEM_MIN_WIDTH}px, 100%), 1fr))`');
    expect(itemSource).toContain('className="w-full min-w-0"');
    expect(itemSource).toContain('w-full min-w-0 h-[104px]');
    expect(itemSource).not.toContain('max-w-[96px]');
    expect(itemSource).not.toContain('w-[88px]');
    expect(itemSource).toContain('[overflow-wrap:anywhere]');
  });

  it('keeps the grid on native wrapping instead of virtual rows that can reveal blank after resize', () => {
    const source = readFileSync(
      resolve(process.cwd(), 'src/features/learning-hub/components/finder/FinderFileList.tsx'),
      'utf-8'
    );

    expect(source).toContain('window.requestAnimationFrame');
    expect(source).toContain('listVirtualizer.measure();');
    expect(source).not.toContain('gridVirtualizer');
    expect(source).toContain('[viewMode, items.length, listVirtualizer]');
    expect(source).toContain('grid w-full min-w-0 gap-2');
    expect(source).toContain('gridTemplateColumns');
    expect(source).not.toContain('flex w-full min-w-0 flex-wrap');
  });

  it('treats inner virtualized blank space as finder background clicks', () => {
    const source = readFileSync(
      resolve(process.cwd(), 'src/features/learning-hub/components/finder/FinderFileList.tsx'),
      'utf-8'
    );

    expect(source).toContain("if (!target.closest('[data-finder-item]') && onContainerClick)");
    expect(source).not.toContain('e.target === e.currentTarget && onContainerClick');
  });

  it('does not turn the fullscreen finder grid into a vertical list when the shell sidebar is collapsed', () => {
    const source = readFileSync(
      resolve(process.cwd(), 'src/features/learning-hub/LearningHubSidebar.tsx'),
      'utf-8'
    );
    const quickAccessSource = readFileSync(
      resolve(process.cwd(), 'src/features/learning-hub/components/finder/FinderQuickAccess.tsx'),
      'utf-8'
    );
    const pageSource = readFileSync(
      resolve(process.cwd(), 'src/features/learning-hub/LearningHubPage.tsx'),
      'utf-8'
    );
    const desktopSource = readFileSync(
      resolve(process.cwd(), 'src/features/learning-hub/components/finder/DesktopView.tsx'),
      'utf-8'
    );

    expect(source).toContain("viewMode={mode === 'canvas' ? 'list' : viewMode}");
    expect(source).toContain("compact={mode === 'canvas'}");
    expect(source).toContain("onViewModeChange={setViewMode}");
    expect(source).toContain("{!hideToolbarAndNav && (mode === 'canvas' || !isSmallScreen) && (");
    expect(source).toContain("title={t('finder.toolbar.up', '上一级')}");
    expect(source).toContain('onClick={goUp}');
    expect(source).toContain('className="study-shell-toolbar flex items-center gap-1.5 px-2 py-1 border-b shrink-0 min-w-0 overflow-visible"');
    expect(source).toContain('className="!h-7 !w-7 !p-0 shrink-0"');
    expect(quickAccessSource).toContain("collapsed ? 'w-14' : 'w-52'");
    expect(desktopSource).toContain("gridTemplateColumns: 'repeat(auto-fill, minmax(min(112px, 100%), 1fr))'");
    expect(desktopSource).toContain('w-full min-w-0 text-xs text-center');
    expect(desktopSource).not.toContain('max-w-[112px] text-xs text-center');
    expect(pageSource).toContain('setLocalSidebarCollapsed(false);');
    expect(pageSource).toContain('minSize={hasOpenApp ? (sidebarCollapsed ? 8 : 20) : 15}');
    expect(pageSource).not.toContain('collapsedSize={8}');
    expect(pageSource).not.toContain('onCollapse={() => setLocalSidebarCollapsed(true)}');
    expect(pageSource).not.toContain('onExpand={() => setLocalSidebarCollapsed(false)}');
    expect(pageSource).not.toContain('useDesktopShellSidebarPortal');
    expect(source).not.toContain('quickAccessPortalTarget');
    expect(source).not.toContain('createPortal');
    expect(source).not.toContain('showFinderCollapseRail');
    expect(source).not.toContain('showFinderCollapseRail && "hidden"');
    expect(source).toContain("{!isSmallScreen && mode !== 'canvas' && (");
    expect(source).toContain('collapsed={effectiveQuickAccessCollapsed}');
    expect(source).not.toContain('FILE_PANE_COLLAPSE_WIDTH');
    expect(source).not.toContain('filePaneCollapsed');
    expect(source).not.toContain('filePaneRef');
    expect(source).not.toContain("viewMode={isCollapsed || mode === 'canvas' ? 'list' : viewMode}");
    expect(source).not.toContain('filePaneAutoHidden');
    expect(source).not.toContain('autoCollapseQuickAccess');
  });
});
