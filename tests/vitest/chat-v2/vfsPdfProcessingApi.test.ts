import { describe, it, expect, vi, beforeEach } from 'vitest';

vi.mock('@tauri-apps/api/core', () => ({
  invoke: vi.fn(),
}));

import { invoke } from '@tauri-apps/api/core';
import {
  getBatchPdfProcessingStatus,
  getPdfProcessingStatus,
} from '@/api/vfsPdfProcessingApi';

const invokeMock = vi.mocked(invoke);

describe('vfsPdfProcessingApi normalization', () => {
  beforeEach(() => {
    invokeMock.mockReset();
  });

  it('normalizes backend batch HashMap payload into { statuses } shape', async () => {
    invokeMock.mockResolvedValueOnce({
      att_1: {
        fileId: 'att_1',
        stage: 'ocr_processing',
        progress: {
          stage: 'ocr_processing',
          currentPage: 2,
          totalPages: 5,
          percent: 45,
          readyModes: ['text'],
          mediaType: 'pdf',
        },
      },
    });

    const result = await getBatchPdfProcessingStatus(['att_1']);

    expect(result.statuses.att_1).toEqual({
      stage: 'ocr_processing',
      currentPage: 2,
      totalPages: 5,
      percent: 45,
      readyModes: ['text'],
      mediaType: 'pdf',
    });
  });

  it('normalizes single status payload with nested progress', async () => {
    invokeMock.mockResolvedValueOnce({
      fileId: 'att_2',
      stage: 'vector_indexing',
      progress: {
        stage: 'vector_indexing',
        percent: 90,
        readyModes: ['text', 'image'],
      },
      error: null,
    });

    const result = await getPdfProcessingStatus('att_2');

    expect(result).toEqual({
      stage: 'vector_indexing',
      percent: 90,
      readyModes: ['text', 'image'],
    });
  });

  it('normalizes snake_case progress fields from the backend', async () => {
    invokeMock.mockResolvedValueOnce({
      statuses: {
        att_3: {
          stage: 'ocr_processing',
          progress: {
            stage: 'ocr_processing',
            current_page: 3,
            total_pages: 8,
            percent: 62,
            ready_modes: ['text'],
            media_type: 'pdf',
          },
        },
      },
    });

    const result = await getBatchPdfProcessingStatus(['att_3']);

    expect(result.statuses.att_3).toEqual({
      stage: 'ocr_processing',
      currentPage: 3,
      totalPages: 8,
      percent: 62,
      readyModes: ['text'],
      mediaType: 'pdf',
    });
  });

  it('preserves completed_with_issues with failed stages and usable modes', async () => {
    invokeMock.mockResolvedValueOnce({
      fileId: 'att_4',
      stage: 'completed_with_issues',
      progress: {
        stage: 'completed_with_issues',
        percent: 100,
        ready_modes: ['text'],
        failed_stages: [
          {
            stage: 'vector_indexing',
            message: 'embedding failed',
            retriable: true,
          },
        ],
      },
      error: null,
    });

    const result = await getPdfProcessingStatus('att_4');

    expect(result).toEqual({
      stage: 'completed_with_issues',
      percent: 100,
      readyModes: ['text'],
      failedStages: [
        {
          stage: 'vector_indexing',
          message: 'embedding failed',
          retriable: true,
        },
      ],
    });
  });
});
