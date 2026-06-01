import React from 'react';
import { act, render, screen, waitFor } from '@testing-library/react';
import { describe, expect, it, vi } from 'vitest';

import { UnifiedAppPanel } from '@/features/learning-hub/apps/UnifiedAppPanel';

const panelMocks = vi.hoisted(() => ({
  dstuGet: vi.fn(),
}));

const stableT = (_key: string, fallback?: string) => fallback ?? _key;

vi.mock('react-i18next', () => ({
  initReactI18next: { type: '3rdParty', init: vi.fn() },
  useTranslation: () => ({ t: stableT }),
}));

vi.mock('@/dstu', () => ({
  dstu: {
    get: panelMocks.dstuGet,
  },
}));

vi.mock('@/shared/result', () => ({
  reportError: vi.fn(),
}));

vi.mock('@/features/learning-hub/apps/views/FileContentView', () => ({
  default: ({ node }: { node: { id: string; name?: string } }) => (
    <div data-testid="file-content">{node.id}:{node.name}</div>
  ),
}));

vi.mock('@/features/learning-hub/apps/views/NoteContentView', () => ({
  default: ({ node }: { node: { id: string; name?: string } }) => (
    <div data-testid="note-content">{node.id}:{node.name}</div>
  ),
}));

vi.mock('@/features/learning-hub/apps/views/TextbookContentView', () => ({ default: () => null }));
vi.mock('@/features/learning-hub/apps/views/ExamContentView', () => ({ default: () => null }));
vi.mock('@/features/learning-hub/apps/views/TranslationContentView', () => ({ default: () => null }));
vi.mock('@/features/learning-hub/apps/views/EssayContentView', () => ({ default: () => null }));
vi.mock('@/features/learning-hub/apps/views/ImageContentView', () => ({ default: () => null }));
vi.mock('@/features/mindmap/MindMapContentView', () => ({ MindMapContentView: () => null }));

function deferred<T>() {
  let resolve!: (value: T) => void;
  const promise = new Promise<T>((res) => {
    resolve = res;
  });
  return { promise, resolve };
}

const okNode = (id: string, name = id) => ({
  ok: true,
  value: {
    id,
    type: 'file',
    name,
  },
});

describe('UnifiedAppPanel resource loading contract', () => {
  it('loads by stable resource id instead of refetching on display path metadata changes', async () => {
    panelMocks.dstuGet.mockResolvedValue(okNode('file_1', 'First'));

    const { rerender } = render(
      <UnifiedAppPanel type="file" resourceId="file_1" dstuPath="/Folder A/First.pdf" />
    );

    await waitFor(() => expect(screen.getByTestId('file-content')).toHaveTextContent('file_1:First'));

    rerender(<UnifiedAppPanel type="file" resourceId="file_1" dstuPath="/Renamed/First.pdf" />);

    expect(panelMocks.dstuGet).toHaveBeenCalledTimes(1);
    expect(panelMocks.dstuGet).toHaveBeenCalledWith('/file_1');
  });

  it('does not let an older resource request overwrite the current resource', async () => {
    const first = deferred<ReturnType<typeof okNode>>();
    const second = deferred<ReturnType<typeof okNode>>();
    panelMocks.dstuGet
      .mockReturnValueOnce(first.promise)
      .mockReturnValueOnce(second.promise);

    const { rerender } = render(
      <UnifiedAppPanel type="file" resourceId="file_a" dstuPath="/A.pdf" />
    );
    rerender(<UnifiedAppPanel type="file" resourceId="file_b" dstuPath="/B.pdf" />);

    await act(async () => {
      second.resolve(okNode('file_b', 'Second'));
      await second.promise;
    });

    await waitFor(() => expect(screen.getByTestId('file-content')).toHaveTextContent('file_b:Second'));

    await act(async () => {
      first.resolve(okNode('file_a', 'First'));
      await first.promise;
    });

    expect(screen.getByTestId('file-content')).toHaveTextContent('file_b:Second');
  });
});
