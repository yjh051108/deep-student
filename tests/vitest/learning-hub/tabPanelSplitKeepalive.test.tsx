import React from 'react';
import { render, screen, waitFor } from '@testing-library/react';
import { describe, expect, it, vi } from 'vitest';

import { TabPanelContainer } from '@/features/learning-hub/apps/TabPanelContainer';
import type { OpenTab } from '@/features/learning-hub/types/tabs';

vi.mock('react-i18next', () => ({
  useTranslation: () => ({
    t: (_key: string, fallback?: string) => fallback ?? _key,
  }),
}));

vi.mock('react-resizable-panels', () => ({
  PanelGroup: ({ children }: { children: React.ReactNode }) => <div>{children}</div>,
  Panel: ({ children }: { children: React.ReactNode }) => <div>{children}</div>,
  PanelResizeHandle: () => null,
}));

vi.mock('@/features/learning-hub/apps/UnifiedAppPanel', () => ({
  UnifiedAppPanel: ({
    resourceId,
    isActive,
  }: {
    resourceId: string;
    isActive?: boolean;
  }) => <div data-testid={`panel-${resourceId}`}>{isActive ? 'active' : 'hidden'}</div>,
}));

const tab = (tabId: string, resourceId: string): OpenTab => ({
  tabId,
  resourceId,
  type: 'file',
  dstuPath: `/${resourceId}`,
  title: resourceId,
  openedAt: 1,
});

describe('TabPanelContainer split keepalive contract', () => {
  it('shows a valid tab when activeTabId is stale instead of rendering a blank panel', async () => {
    render(
      <TabPanelContainer
        tabs={[tab('old', 'old-file'), tab('latest', 'latest-file')]}
        activeTabId="missing"
        onClose={vi.fn()}
        onTitleChange={vi.fn()}
      />
    );

    await waitFor(() => expect(screen.getByTestId('panel-latest-file')).toHaveTextContent('active'));
    expect(screen.getByTestId('panel-old-file')).toHaveTextContent('hidden');
  });

  it('keeps non-visible tabs mounted while split view renders normal panes', async () => {
    render(
      <TabPanelContainer
        tabs={[tab('left', 'left-file'), tab('right', 'right-file'), tab('hidden', 'hidden-file')]}
        activeTabId="left"
        splitView={{ rightTabId: 'right' }}
        onClose={vi.fn()}
        onTitleChange={vi.fn()}
        onCloseSplitView={vi.fn()}
      />
    );

    await waitFor(() => expect(screen.getByTestId('panel-left-file')).toHaveTextContent('active'));
    expect(screen.getByTestId('panel-right-file')).toHaveTextContent('active');
    expect(screen.getByTestId('panel-hidden-file')).toHaveTextContent('hidden');
  });
});
