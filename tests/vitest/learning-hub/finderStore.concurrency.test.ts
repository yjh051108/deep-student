import { beforeEach, describe, expect, it, vi } from 'vitest';

const { dstuListMock, getBreadcrumbsMock, reportErrorMock } = vi.hoisted(() => ({
  dstuListMock: vi.fn(),
  getBreadcrumbsMock: vi.fn(),
  reportErrorMock: vi.fn(),
}));

vi.mock('@/dstu/api', () => ({
  dstu: {
    list: dstuListMock,
    get: vi.fn(),
    search: vi.fn(),
    searchInFolder: vi.fn(),
  },
}));

vi.mock('@/dstu', () => ({
  folderApi: {
    getBreadcrumbs: getBreadcrumbsMock,
  },
  trashApi: {
    listTrash: vi.fn(),
  },
}));

vi.mock('@/shared/result', () => ({
  reportError: reportErrorMock,
}));

vi.mock('@/i18n', () => ({
  default: {
    language: 'zh-CN',
  },
}));

import { useFinderStore } from '@/features/learning-hub/stores/finderStore';
import type { DstuNode } from '@/dstu/types';

function deferred<T>() {
  let resolve!: (value: T) => void;
  const promise = new Promise<T>((innerResolve) => {
    resolve = innerResolve;
  });
  return { promise, resolve };
}

function node(id: string, name: string): DstuNode {
  return {
    id,
    name,
    type: 'note',
    path: `/${name}`,
    createdAt: '2026-05-01T00:00:00.000Z',
    updatedAt: '2026-05-02T00:00:00.000Z',
  } as DstuNode;
}

describe('finderStore concurrency guards', () => {
  beforeEach(() => {
    vi.clearAllMocks();
    useFinderStore.getState().reset();
  });

  it('does not let an older load failure clear newer successful items', async () => {
    const oldRequest = deferred<{ ok: false; error: { message: string } }>();
    const newRequest = deferred<{ ok: true; value: DstuNode[] }>();
    dstuListMock
      .mockReturnValueOnce(oldRequest.promise)
      .mockReturnValueOnce(newRequest.promise);

    const firstLoad = useFinderStore.getState().loadItems();
    const secondLoad = useFinderStore.getState().loadItems();

    newRequest.resolve({ ok: true, value: [node('note-2', 'new')] });
    await secondLoad;

    expect(useFinderStore.getState().items.map(item => item.id)).toEqual(['note-2']);

    oldRequest.resolve({ ok: false, error: { message: 'old folder failed' } });
    await firstLoad;

    expect(useFinderStore.getState().items.map(item => item.id)).toEqual(['note-2']);
    expect(useFinderStore.getState().error).toBeNull();
  });

  it('does not let an older breadcrumb response overwrite newer folder navigation', async () => {
    const folderA = deferred<{ ok: true; value: Array<{ id: string; name: string }> }>();
    const folderB = deferred<{ ok: true; value: Array<{ id: string; name: string }> }>();
    getBreadcrumbsMock
      .mockReturnValueOnce(folderA.promise)
      .mockReturnValueOnce(folderB.promise);

    const firstNavigation = useFinderStore.getState().enterFolder('folder-a');
    const secondNavigation = useFinderStore.getState().enterFolder('folder-b');

    folderB.resolve({ ok: true, value: [{ id: 'folder-b', name: 'B' }] });
    await secondNavigation;

    expect(useFinderStore.getState().currentPath.folderId).toBe('folder-b');

    folderA.resolve({ ok: true, value: [{ id: 'folder-a', name: 'A' }] });
    await firstNavigation;

    expect(useFinderStore.getState().currentPath.folderId).toBe('folder-b');
  });

  it('refreshes immediately after quick access navigation', async () => {
    dstuListMock.mockResolvedValue({ ok: true, value: [node('note-quick', 'quick')] });

    useFinderStore.getState().quickAccessNavigate('allFiles');

    await vi.waitFor(() => {
      expect(dstuListMock).toHaveBeenCalled();
      expect(useFinderStore.getState().items.map(item => item.id)).toEqual(['note-quick']);
    });
  });
});
