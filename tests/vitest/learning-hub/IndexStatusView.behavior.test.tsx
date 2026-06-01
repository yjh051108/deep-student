import React from 'react';
import { act, fireEvent, render, screen, waitFor } from '@testing-library/react';
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';

const invokeMock = vi.fn();
const getAllIndexStatusMock = vi.fn();
const batchIndexPendingMock = vi.fn();
const listDimensionsMock = vi.fn();
const showGlobalNotificationMock = vi.fn();
const vfsIndexResourceBySourceMock = vi.fn();
const reindexResourceMock = vi.fn();
const eventListeners = new Map<string, Array<(event: { payload: unknown }) => void>>();

vi.mock('@tauri-apps/api/core', () => ({
  invoke: (...args: unknown[]) => invokeMock(...args),
}));

vi.mock('@tauri-apps/api/event', () => ({
  listen: vi.fn(async (event: string, callback: (event: { payload: unknown }) => void) => {
    eventListeners.set(event, [...(eventListeners.get(event) ?? []), callback]);
    return vi.fn();
  }),
}));

vi.mock('@/hooks/useBreakpoint', () => ({
  useIsMobile: () => false,
}));

vi.mock('@/components/custom-scroll-area', () => ({
  CustomScrollArea: ({ children }: { children: React.ReactNode }) => <div>{children}</div>,
}));

vi.mock('@/components/ui/NotionButton', () => ({
  NotionButton: ({
    children,
    disabled,
    onClick,
    title,
  }: {
    children: React.ReactNode;
    disabled?: boolean;
    onClick?: () => void;
    title?: string;
  }) => (
    <button disabled={disabled} onClick={onClick} title={title}>
      {children}
    </button>
  ),
}));

vi.mock('@/components/ui/shad/Progress', () => ({
  Progress: ({ value }: { value?: number }) => <div role="progressbar" aria-valuenow={value ?? 0} />,
}));

vi.mock('@/components/ui/shad/Input', () => ({
  Input: (props: React.InputHTMLAttributes<HTMLInputElement>) => <input {...props} />,
}));

vi.mock('@/components/UnifiedNotification', () => ({
  showGlobalNotification: (...args: unknown[]) => showGlobalNotificationMock(...args),
}));

vi.mock('@/utils/unifiedDialogs', () => ({
  unifiedAlert: vi.fn(),
  unifiedConfirm: vi.fn(async () => true),
}));

vi.mock('@/debug-panel/debugMasterSwitch', () => ({
  debugLog: {
    log: vi.fn(),
    warn: vi.fn(),
    error: vi.fn(),
    info: vi.fn(),
    debug: vi.fn(),
  },
}));

vi.mock('@/api/vfsUnifiedIndexApi', () => ({
  getAllIndexStatus: (...args: unknown[]) => getAllIndexStatusMock(...args),
  reindexResource: (...args: unknown[]) => reindexResourceMock(...args),
  batchIndexPendingLegacy: (...args: unknown[]) => batchIndexPendingMock(...args),
  listDimensions: (...args: unknown[]) => listDimensionsMock(...args),
  getResourceOcrInfo: vi.fn(),
  clearResourceOcr: vi.fn(),
  getResourceTextChunks: vi.fn(),
}));

vi.mock('@/api/vfsRagApi', () => ({
  vfsRagSearch: vi.fn(),
  resetAllIndexState: vi.fn(),
}));

vi.mock('@/services/multimodalRagService', () => ({
  MULTIMODAL_INDEX_ENABLED: true,
  default: {
    vfsIndexResourceBySource: (...args: unknown[]) => vfsIndexResourceBySourceMock(...args),
  },
}));

import { IndexStatusView } from '@/features/learning-hub/views/IndexStatusView';

const resource = (overrides = {}) => ({
  resourceId: 'res_1',
  sourceId: 'source_1',
  resourceType: 'image',
  name: 'Image 1',
  hasOcr: true,
  ocrCount: 12,
  textIndexState: 'indexed',
  textChunkCount: 1,
  nativeTextChunkCount: 1,
  ocrTextChunkCount: 0,
  textIndexRetryable: false,
  mmIndexState: 'indexed',
  mmIndexedPages: 1,
  displayIndexState: 'indexed',
  updatedAt: 1,
  isStale: false,
  ...overrides,
});

const summary = (overrides = {}) => ({
  totalResources: 9,
  indexedCount: 7,
  pendingCount: 0,
  indexingCount: 0,
  failedCount: 0,
  disabledCount: 0,
  staleCount: 0,
  textQueueCount: 0,
  displayTotalResources: 10,
  displayIndexedCount: 8,
  displayPendingCount: 2,
  displayIndexingCount: 0,
  displayFailedCount: 0,
  displayDisabledCount: 0,
  mmTotalResources: 4,
  mmIndexedCount: 2,
  mmPendingCount: 2,
  mmIndexingCount: 0,
  mmFailedCount: 0,
  mmDisabledCount: 0,
  resources: [resource()],
  ...overrides,
});

describe('IndexStatusView behavior', () => {
  beforeEach(() => {
    vi.clearAllMocks();
    vi.useRealTimers();
    eventListeners.clear();
    listDimensionsMock.mockResolvedValue([]);
    batchIndexPendingMock.mockResolvedValue({ successCount: 2, failCount: 0, total: 2 });
    reindexResourceMock.mockResolvedValue(1);
    vfsIndexResourceBySourceMock.mockResolvedValue({ indexedPages: 1 });
    invokeMock.mockResolvedValue({
      status: 'ready',
      ready: true,
      modelConfigId: 'vl_model',
      modelName: 'VL Embedding',
      model: 'vl-embedding',
    });
  });

  afterEach(() => {
    vi.useRealTimers();
  });

  it('renders text, image, and aggregate progress from distinct backend counters', async () => {
    getAllIndexStatusMock.mockResolvedValue(summary());

    render(<IndexStatusView />);

    expect(await screen.findAllByLabelText('indexStatus.progress.textIndexProgress 7/9')).not.toHaveLength(0);
    expect(screen.getAllByLabelText('indexStatus.progress.imageIndexProgress 2/4')).not.toHaveLength(0);
    expect(screen.getAllByText('80%')).not.toHaveLength(0);
  });

  it('runs backend text indexing even when no visible row is text-pending', async () => {
    getAllIndexStatusMock.mockResolvedValue(summary({
      indexedCount: 7,
      pendingCount: 2,
      textQueueCount: 2,
      displayPendingCount: 2,
      resources: [],
    }));

    render(<IndexStatusView />);

    fireEvent.click(await screen.findByRole('button', { name: /indexStatus\.action\.oneClickIndex/ }));

    await waitFor(() => expect(batchIndexPendingMock).toHaveBeenCalledTimes(1));
    expect(batchIndexPendingMock).toHaveBeenCalledWith();
  });

  it('indexes only filtered text resources when a resource type filter is active', async () => {
    getAllIndexStatusMock
      .mockResolvedValueOnce(summary({
        totalResources: 1,
        displayTotalResources: 1,
      }))
      .mockResolvedValueOnce(summary({
        totalResources: 2,
        indexedCount: 0,
        pendingCount: 2,
        textQueueCount: 2,
        displayTotalResources: 2,
        displayIndexedCount: 0,
        displayPendingCount: 2,
        resources: [
          resource({
            resourceId: 'file_pending',
            resourceType: 'file',
            name: 'File Pending',
            textIndexState: 'pending',
            textIndexRetryable: true,
            displayIndexState: 'pending',
          }),
          resource({
            resourceId: 'file_indexed',
            resourceType: 'file',
            name: 'File Indexed',
            textIndexState: 'indexed',
            displayIndexState: 'indexed',
          }),
        ],
      }));

    render(<IndexStatusView />);

    fireEvent.click(await screen.findByRole('button', { name: /indexStatus\.resourceType\.file/ }));
    await waitFor(() => expect(getAllIndexStatusMock).toHaveBeenLastCalledWith(expect.objectContaining({
      resourceType: 'file',
    })));
    await screen.findByText('File Pending');

    fireEvent.click(await screen.findByRole('button', { name: /indexStatus\.action\.oneClickIndex/ }));

    await waitFor(() => expect(reindexResourceMock).toHaveBeenCalledWith('file_pending'));
    expect(reindexResourceMock).toHaveBeenCalledTimes(1);
    expect(batchIndexPendingMock).not.toHaveBeenCalled();
  });

  it('does not run image indexing when multimodal capability is unavailable', async () => {
    invokeMock.mockResolvedValue({
      status: 'notConfigured',
      ready: false,
    });
    getAllIndexStatusMock.mockResolvedValue(summary({
      totalResources: 1,
      indexedCount: 1,
      pendingCount: 0,
      failedCount: 0,
      displayTotalResources: 1,
      displayIndexedCount: 1,
      displayPendingCount: 0,
      mmTotalResources: 1,
      mmIndexedCount: 0,
      mmPendingCount: 1,
      resources: [resource({
        textIndexState: 'indexed',
        mmIndexState: 'pending',
        displayIndexState: 'indexed',
      })],
    }));

    render(<IndexStatusView />);

    fireEvent.click(await screen.findByRole('button', { name: /indexStatus\.action\.oneClickIndex/ }));

    await waitFor(() => expect(showGlobalNotificationMock).toHaveBeenCalled());
    expect(batchIndexPendingMock).not.toHaveBeenCalled();
    expect(vfsIndexResourceBySourceMock).not.toHaveBeenCalled();
    expect(showGlobalNotificationMock).toHaveBeenCalledWith(
      'warning',
      'indexStatus.notification.hint',
      'indexStatus.progress.imageIndexCapability.notConfigured'
    );
  });

  it('keeps batch completion visible until the completion timer clears it', async () => {
    let resolveBatch: (value: unknown) => void = () => {};
    batchIndexPendingMock.mockReturnValue(new Promise((resolve) => {
      resolveBatch = resolve;
    }));
    getAllIndexStatusMock.mockResolvedValue(summary({
      indexedCount: 7,
      pendingCount: 2,
      textQueueCount: 2,
      displayPendingCount: 2,
      resources: [],
    }));

    render(<IndexStatusView />);

    fireEvent.click(await screen.findByRole('button', { name: /indexStatus\.action\.oneClickIndex/ }));
    await waitFor(() => expect(batchIndexPendingMock).toHaveBeenCalledTimes(1));

    vi.useFakeTimers();
    act(() => {
      for (const listener of eventListeners.get('vfs-index-progress') ?? []) {
        listener({
          payload: {
            type: 'batch_completed',
            total: 2,
            successCount: 2,
            failCount: 0,
            message: 'Done',
          },
        });
      }
    });

    expect(screen.getAllByRole('progressbar').some((node) => node.getAttribute('aria-valuenow') === '100')).toBe(true);

    await act(async () => {
      resolveBatch({ successCount: 2, failCount: 0, total: 2 });
      await Promise.resolve();
    });
    expect(screen.getAllByRole('progressbar').some((node) => node.getAttribute('aria-valuenow') === '100')).toBe(true);

    act(() => {
      vi.advanceTimersByTime(600);
    });
    expect(screen.queryByText('Done')).toBeNull();

    vi.useRealTimers();
  });
});
