import { readFileSync } from 'node:fs';
import { resolve } from 'node:path';
import { describe, expect, it } from 'vitest';

describe('IndexStatusView progress display contract', () => {
  const source = readFileSync(
    resolve(process.cwd(), 'src/features/learning-hub/views/IndexStatusView.tsx'),
    'utf-8'
  );

  it('does not render a separate multimodal progress bar after aggregate status is complete', () => {
    expect(source).not.toContain('imageIndexStats');
    expect(source).not.toContain('mmProgress');
    expect(source).not.toContain('mmMessage');
    expect(source).not.toContain('[&>div]:bg-purple-600');
    expect(source).not.toContain('bg-purple-500/5 p-2 rounded-md');
  });

  it('keeps image index capability as metadata instead of a second progress count', () => {
    expect(source).toContain("t('indexStatus.progress.imageIncludedInOverall'");
    expect(source).toContain('renderCount(displayIndexStats.indexed, displayIndexStats.total)');
    expect(source).toContain('total: summary?.displayTotalResources ?? 0');
    expect(source).toContain('indexed: summary?.displayIndexedCount ?? 0');
    expect(source).not.toContain('renderCount(imageIndexStats.indexed, imageIndexStats.total)');
  });

  it('uses multimodal progress events only to lock indexing and refresh aggregate status', () => {
    expect(source).toContain("setMmIndexing(payload.phase !== 'completed' && payload.phase !== 'failed');");
    expect(source).toContain("if (payload.phase === 'completed') {");
    expect(source).toContain('loadData();');
    expect(source).toContain('showGlobalNotification(failCount > 0 ?');
  });

  it('does not run image indexing when the backend reports multimodal indexing unavailable', () => {
    expect(source).toContain("const canRunImageIndex = imageIndexCapability === 'ready';");
    expect(source).toContain('const displayWorkCount = summary.displayPendingCount + summary.displayFailedCount;');
    expect(source).toContain('const pendingMmCount = mmResources.length;');
    expect(source).toContain('displayWorkCount > 0 && pendingTextCount === 0 && pendingMmCount > 0 && !canRunImageIndex');
    expect(source).toContain('if (mmResources.length > 0 && canRunImageIndex)');
    expect(source).toContain('includeImageIndex,');
  });

  it('does not start progress animation when display state has no work left', () => {
    expect(source).toContain('if (displayWorkCount === 0) {');
    expect(source).toContain("t('indexStatus.notification.noResourcesToIndex')");
    expect(source).toContain('return;');
    expect(source).not.toContain('pendingTextCount > 0 || displayWorkCount === 0');
  });
});
