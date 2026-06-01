import React from 'react';
import { fireEvent, render, screen, waitFor } from '@testing-library/react';
import { beforeEach, describe, expect, it, vi } from 'vitest';

import { DSTU_NAVIGATE_TO_KNOWLEDGE_BASE_EVENT } from '@/components/learning-hub/learningHubContracts';

const pageMocks = vi.hoisted(() => ({
  dstuGet: vi.fn(),
  finderQuickAccessNavigate: vi.fn(),
  panelOps: [] as string[],
  setPendingMemoryLocate: vi.fn(),
}));

const finderState = {
  currentPath: { viewKind: 'folder', folderId: null, typeFilter: null, breadcrumbs: [] },
  goUp: vi.fn(),
  jumpToBreadcrumb: vi.fn(),
  refresh: vi.fn(),
  quickAccessNavigate: pageMocks.finderQuickAccessNavigate,
};

vi.mock('react-i18next', async () => {
  const actual = await vi.importActual<typeof import('react-i18next')>('react-i18next');
  return {
    ...actual,
    useTranslation: () => ({
      t: (key: string, fallback?: string) => fallback ?? key,
    }),
  };
});
vi.mock('react-resizable-panels', async () => {
  const React = await vi.importActual<typeof import('react')>('react');
  return {
    PanelGroup: ({ children }: { children: React.ReactNode }) => <div>{children}</div>,
    Panel: React.forwardRef(({
      children,
      id,
    }: {
      children: React.ReactNode;
      id?: string;
    }, ref: React.ForwardedRef<unknown>) => {
      React.useImperativeHandle(ref, () => ({
        expand: () => pageMocks.panelOps.push(`${id ?? 'panel'}:expand`),
        resize: (size: number) => pageMocks.panelOps.push(`${id ?? 'panel'}:resize:${size}`),
      }));
      return <div data-testid={id ? `panel-${id}` : undefined}>{children}</div>;
    }),
    PanelResizeHandle: () => null,
  };
});
vi.mock('@/dstu/openResource', () => ({ registerOpenResourceHandler: vi.fn(() => () => {}), type: {} }));
vi.mock('@/dstu', () => ({
  createEmpty: vi.fn(),
  dstu: {
    watch: vi.fn(() => () => {}),
    get: pageMocks.dstuGet,
  },
}));
vi.mock('@/components/UnifiedNotification', () => ({ showGlobalNotification: vi.fn() }));
vi.mock('@/utils/pendingMemoryLocate', () => ({ setPendingMemoryLocate: pageMocks.setPendingMemoryLocate }));
vi.mock('@/components/learning-hub/LearningHubSidebar', () => ({
  LearningHubSidebar: ({ onOpenApp, activeFileId }: {
    onOpenApp?: (item: { id: string; type: string; title: string; path: string }) => void;
    activeFileId?: string | null;
  }) => (
    <div>
      <button
        onClick={() => onOpenApp?.({
          id: 'file_1',
          type: 'file',
          title: 'Handout.pdf',
          path: '/Course/Handout.pdf',
        })}
      >
        open-sidebar-file
      </button>
      <button
        onClick={() => onOpenApp?.({
          id: 'file_2',
          type: 'file',
          title: 'Notes.pdf',
          path: '/Course/Notes.pdf',
        })}
      >
        open-sidebar-file-2
      </button>
      <div data-testid="active-file-id">{activeFileId ?? ''}</div>
    </div>
  ),
}));
vi.mock('@/stores/uiStore', () => ({ useUIStore: (selector: (state: { leftPanelCollapsed: boolean; setLeftPanelCollapsed: () => void }) => unknown) => selector({ leftPanelCollapsed: false, setLeftPanelCollapsed: vi.fn() }) }));
vi.mock('@/components/layout', () => ({ useMobileHeader: vi.fn() }));
vi.mock('@/hooks/useBreakpoint', () => ({ useBreakpoint: () => ({ isSmallScreen: false }) }));
vi.mock('@/components/learning-hub/stores/finderStore', () => ({
  useFinderStore: (selector: (state: typeof finderState) => unknown) => selector(finderState),
}));
vi.mock('@/components/learning-hub/components/DstuAppLauncher', () => ({ DstuAppLauncher: () => null }));
vi.mock('@/components/learning-hub/components/TabBar', () => ({
  TabBar: ({
    tabs,
    activeTabId,
    splitView,
    onSplitView,
    onClose,
  }: {
    tabs: Array<{ tabId: string; title: string }>;
    activeTabId: string | null;
    splitView?: { rightTabId: string } | null;
    onSplitView?: (tabId: string) => void;
    onClose?: (tabId: string) => void;
  }) => (
    <div>
      <div data-testid="tabbar-active">{activeTabId ?? ''}</div>
      <div data-testid="tabbar-split">{splitView?.rightTabId ?? ''}</div>
      <button onClick={() => tabs[1] && onSplitView?.(tabs[1].tabId)}>split-second-tab</button>
      <button onClick={() => activeTabId && onClose?.(activeTabId)}>close-active-tab</button>
    </div>
  ),
}));
vi.mock('@/components/learning-hub/apps/TabPanelContainer', () => ({
  TabPanelContainer: ({ tabs }: { tabs: Array<{ title: string }> }) => <div data-testid="tab-count">{tabs.length}:{tabs[0]?.title ?? ''}</div>,
}));
vi.mock('@/components/learning-hub/activeTabAccessor', () => ({ setActiveTabForExternal: vi.fn() }));
vi.mock('@/command-palette/hooks/useCommandEvents', () => ({ COMMAND_EVENTS: {}, useCommandEvents: vi.fn() }));
vi.mock('@/debug-panel/hooks/usePageLifecycle', () => ({ usePageMount: vi.fn() }));
vi.mock('@/debug-panel/debugMasterSwitch', () => ({ debugLog: { log: vi.fn(), warn: vi.fn(), error: vi.fn() } }));
vi.mock('@/components/learning-hub/hooks', async () => {
  const actual = await vi.importActual<typeof import('@/components/learning-hub/hooks')>('@/components/learning-hub/hooks');
  return {
    ...actual,
    useVfsContextInject: () => ({ injectToChat: vi.fn(), canInject: false, isInjecting: false }),
  };
});

import LearningHubPage from '@/components/learning-hub/LearningHubPage';

describe('LearningHubPage events', () => {
  beforeEach(() => {
    pageMocks.dstuGet.mockReset();
    pageMocks.finderQuickAccessNavigate.mockReset();
    pageMocks.panelOps.length = 0;
    pageMocks.setPendingMemoryLocate.mockReset();
    pageMocks.dstuGet.mockResolvedValue({ ok: true, value: { id: 'tb_1', type: 'textbook', name: '代数.pdf' } });
  });

  it('opens a document tab for manage locator events instead of routing to memory', async () => {
    render(<LearningHubPage />);

    window.dispatchEvent(new CustomEvent(DSTU_NAVIGATE_TO_KNOWLEDGE_BASE_EVENT, {
      detail: {
        preferTab: 'manage',
        locator: {
          resourceId: 'tb_1',
          resourceType: 'textbook',
          title: '代数.pdf',
        },
      },
    }));

    await waitFor(() => expect(screen.getByTestId('tab-count')).toHaveTextContent('1:代数.pdf'));
    expect(pageMocks.dstuGet).toHaveBeenCalledWith('/tb_1');
    expect(pageMocks.finderQuickAccessNavigate).not.toHaveBeenCalledWith('memory');
    expect(pageMocks.setPendingMemoryLocate).not.toHaveBeenCalled();
  });

  it('opens and reuses a finder resource tab while feeding the active resource back to the sidebar', async () => {
    render(<LearningHubPage />);

    fireEvent.click(screen.getByText('open-sidebar-file'));
    await waitFor(() => expect(screen.getByTestId('tab-count')).toHaveTextContent('1:Handout.pdf'));
    expect(screen.getByTestId('active-file-id')).toHaveTextContent('file_1');

    fireEvent.click(screen.getByText('open-sidebar-file'));
    await waitFor(() => expect(screen.getByTestId('tab-count')).toHaveTextContent('1:Handout.pdf'));
    expect(screen.getByTestId('active-file-id')).toHaveTextContent('file_1');
  });

  it('restores the desktop split when the first resource tab opens', async () => {
    render(<LearningHubPage />);

    fireEvent.click(screen.getByText('open-sidebar-file'));

    await waitFor(() => {
      expect(pageMocks.panelOps).toEqual(expect.arrayContaining([
        'learning-hub-sidebar:expand',
        'learning-hub-app:expand',
        'learning-hub-sidebar:resize:35',
        'learning-hub-app:resize:65',
      ]));
    });
  });

  it('clears split view when the active left tab is closed', async () => {
    render(<LearningHubPage />);

    fireEvent.click(screen.getByText('open-sidebar-file'));
    fireEvent.click(screen.getByText('open-sidebar-file-2'));
    await waitFor(() => expect(screen.getByTestId('tab-count')).toHaveTextContent('2:Handout.pdf'));

    fireEvent.click(screen.getByText('split-second-tab'));
    await waitFor(() => expect(screen.getByTestId('tabbar-split')).not.toHaveTextContent(''));

    fireEvent.click(screen.getByText('close-active-tab'));
    await waitFor(() => expect(screen.getByTestId('tabbar-split').textContent).toBe(''));
  });
});
