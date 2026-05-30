import { readFileSync } from 'node:fs';
import { resolve } from 'node:path';
import { describe, expect, it } from 'vitest';

describe('IndexStatusView progress display contract', () => {
  const source = readFileSync(
    resolve(process.cwd(), 'src/features/learning-hub/views/IndexStatusView.tsx'),
    'utf-8'
  );

  it('does not render a separate multimodal progress bar after aggregate status is complete', () => {
    expect(source).not.toContain('mmProgress');
    expect(source).not.toContain('mmMessage');
    expect(source).not.toContain('[&>div]:bg-purple-600');
    expect(source).not.toContain('bg-purple-500/5 p-2 rounded-md');
  });

  it('uses multimodal progress events only to lock indexing and refresh aggregate status', () => {
    expect(source).toContain("setMmIndexing(payload.phase !== 'completed' && payload.phase !== 'failed');");
    expect(source).toContain("if (payload.phase === 'completed') {");
    expect(source).toContain('loadData();');
    expect(source).toContain('showGlobalNotification(failCount > 0 ?');
  });

  it('does not run image indexing when the backend reports multimodal indexing unavailable', () => {
    expect(source).toContain("const canRunImageIndex = imageIndexCapability === 'ready';");
    expect(source).toContain('const pendingMmCount = mmResources.length;');
    expect(source).toContain('pendingTextCount === 0 && pendingMmCount > 0 && !canRunImageIndex');
  });
});
