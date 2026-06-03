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

  it('renders text and image progress from their backend counters', () => {
    expect(source).toContain('indexed: summary.textIndexedCount,');
    expect(source).toContain('total: summary.textTotalResources,');
    expect(source).toContain('indexed: summary.mmIndexedCount,');
    expect(source).toContain('total: summary.mmTotalResources,');
    expect(source).toContain("renderCount(textProgress.indexed, textProgress.total, t('indexStatus.progress.textIndexProgress'))");
    expect(source).toContain("renderCount(imageProgress.indexed, imageProgress.total, t('indexStatus.progress.imageIndexProgress'))");
    expect(source).toContain('total: summary?.displayTotalResources ?? 0');
    expect(source).toContain('indexed: summary?.displayIndexedCount ?? 0');
    expect(source).not.toContain('renderCount(displayIndexStats.indexed, displayIndexStats.total)');
  });

  it('uses multimodal progress events only to lock indexing and refresh aggregate status', () => {
    expect(source).toContain("setMmIndexing(payload.phase !== 'completed' && payload.phase !== 'failed');");
    expect(source).toContain("if (payload.phase === 'completed') {");
    expect(source).toContain('loadData();');
    expect(source).toContain('showGlobalNotification(failCount > 0 ?');
  });

  it('does not run image indexing when the backend reports multimodal indexing unavailable', () => {
    expect(source).toContain("const canRunImageIndex = imageIndexCapability === 'ready';");
    expect(source).toContain('const textWorkCount = isActionFiltered');
    expect(source).toContain('const pendingMmCount = mmResources.length;');
    expect(source).toContain('textWorkCount === 0 && pendingMmCount > 0 && !canRunImageIndex');
    expect(source).toContain('if (mmResources.length > 0 && canRunImageIndex)');
    expect(source).toContain('includeImageIndex,');
  });

  it('does not start progress animation when display state has no work left', () => {
    expect(source).toContain('if (textWorkCount === 0 && pendingMmCount === 0) {');
    expect(source).toContain("t('indexStatus.notification.noResourcesToIndex')");
    expect(source).toContain('return;');
    expect(source).not.toContain('pendingTextCount > 0 || displayWorkCount === 0');
  });

  it('lets backend progress events own batch completion progress and message', () => {
    const invokeSuccessBlock = source.slice(
      source.indexOf('await batchIndexPending();'),
      source.indexOf('} catch (err: unknown)', source.indexOf('await batchIndexPending();'))
    );

    expect(invokeSuccessBlock).toContain('setBatchIndexing(false);');
    expect(invokeSuccessBlock).toContain('await loadData();');
    expect(invokeSuccessBlock).not.toContain('setBatchProgress(100)');
    expect(invokeSuccessBlock).not.toContain("setBatchMessage(t('indexStatus.notification.batchCompleted'))");
  });
});
