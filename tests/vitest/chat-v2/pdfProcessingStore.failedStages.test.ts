import { beforeEach, describe, expect, it, vi } from 'vitest';

vi.mock('@/i18n', () => ({
  default: {
    t: (_key: string, fallback?: string) => fallback ?? _key,
  },
}));

import { usePdfProcessingStore } from '@/features/pdf/stores/pdfProcessingStore';

describe('pdf processing store failed stage preservation', () => {
  beforeEach(() => {
    usePdfProcessingStore.getState().clear();
  });

  it('preserves failed stages when a completed_with_issues event follows progress', () => {
    const issue = {
      stage: 'ocr_processing',
      message: 'OCR returned unusable text',
      retriable: true,
    };

    usePdfProcessingStore.getState().update('att_image', {
      stage: 'ocr_processing',
      percent: 82,
      readyModes: ['image'],
      mediaType: 'image',
      failedStages: [issue],
    });

    usePdfProcessingStore.getState().setCompleted(
      'att_image',
      ['image'],
      'completed_with_issues'
    );

    expect(usePdfProcessingStore.getState().get('att_image')).toMatchObject({
      stage: 'completed_with_issues',
      percent: 100,
      readyModes: ['image'],
      mediaType: 'image',
      failedStages: [issue],
    });
  });

  it('clears stale failed stages for a clean completed status', () => {
    usePdfProcessingStore.getState().update('att_pdf', {
      stage: 'vector_indexing',
      percent: 90,
      readyModes: ['text'],
      mediaType: 'pdf',
      failedStages: [
        {
          stage: 'vector_indexing',
          message: 'temporary failure',
          retriable: true,
        },
      ],
    });

    usePdfProcessingStore.getState().setCompleted('att_pdf', ['text'], 'completed');

    expect(usePdfProcessingStore.getState().get('att_pdf')).toMatchObject({
      stage: 'completed',
      readyModes: ['text'],
    });
    expect(usePdfProcessingStore.getState().get('att_pdf')?.failedStages).toBeUndefined();
  });
});
