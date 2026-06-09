import { describe, it, expect, vi, beforeEach } from 'vitest';

vi.mock('@/runtime/native', () => ({
  invoke: vi.fn(),
}));

import { invoke } from '@/runtime/native';
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
          readyModes: ['ocr'],
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
      readyModes: ['ocr'],
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

  it('normalizes completed-with-issues failed stages from nested progress', async () => {
    invokeMock.mockResolvedValueOnce({
      fileId: 'att_3',
      stage: 'completed',
      progress: {
        stage: 'completed_with_issues',
        percent: 100,
        ready_modes: ['text'],
        media_type: 'pdf',
        failed_stages: [
          {
            stage: 'raster_preview',
            message: 'PDFium raster preview is unavailable; using text-layer SVG preview fallback.',
            retriable: true,
          },
        ],
      },
      error: null,
    });

    const result = await getPdfProcessingStatus('att_3');

    expect(result).toEqual({
      stage: 'completed_with_issues',
      percent: 100,
      readyModes: ['text'],
      mediaType: 'pdf',
      failedStages: [
        {
          stage: 'raster_preview',
          message: 'PDFium raster preview is unavailable; using text-layer SVG preview fallback.',
          retriable: true,
        },
      ],
    });
  });

  it('normalizes null backend status to pending fallback', async () => {
    invokeMock.mockResolvedValueOnce(null);

    const result = await getPdfProcessingStatus('missing');

    expect(result).toEqual({
      stage: 'pending',
      percent: 0,
      readyModes: [],
    });
  });
});
