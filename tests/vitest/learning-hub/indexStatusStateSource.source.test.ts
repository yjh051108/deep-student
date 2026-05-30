import { readFileSync } from 'node:fs';
import { resolve } from 'node:path';
import { describe, expect, it } from 'vitest';

describe('Index status state source contract', () => {
  const viewSource = readFileSync(
    resolve(process.cwd(), 'src/features/learning-hub/views/IndexStatusView.tsx'),
    'utf-8'
  );
  const handlerSource = readFileSync(
    resolve(process.cwd(), 'src-tauri/src/vfs/handlers.rs'),
    'utf-8'
  );

  it('uses backend multimodal capability instead of duplicating model matching in the UI', () => {
    expect(viewSource).toContain("invoke<MultimodalIndexCapability>('vfs_get_multimodal_index_capability')");
    expect(viewSource).not.toContain("invoke<IndexModelConfig[]>('get_api_configurations')");
    expect(viewSource).not.toContain('resolveImageIndexCapability');
  });

  it('groups visible rows by combined text and image index state', () => {
    expect(viewSource).toContain('resolveResourceDisplayState');
    expect(viewSource).toContain("const states = [textState, normalizeIndexState(resource.mmIndexState)]");
    expect(viewSource).toContain('groupedCount');
    expect(viewSource).toContain('displayIndexStats');
    expect(viewSource).toContain('indexed: groupedResources.indexed?.length ?? 0');
    expect(viewSource).toContain('displayedResources');
    expect(viewSource).not.toContain("stateFilter: selectedState === 'all' ? undefined : selectedState");
  });

  it('derives one-click workload from the same visible resource rows', () => {
    expect(viewSource).toContain('const pendingTextCount = summary.resources.filter(isPendingTextResource).length;');
    expect(viewSource).toContain('const pendingMmCount = mmResources.length;');
    expect(viewSource).not.toContain('const pendingTextCount = summary.pendingCount + summary.failedCount;');
    expect(viewSource).not.toContain('const pendingMmCount = summary.mmPendingCount + summary.mmFailedCount;');
  });

  it('filters deleted source rows before index status list and summary counts are built', () => {
    expect(handlerSource).toContain("WHERE resource_id IS NOT NULL AND status = 'active' AND deleted_at IS NULL");
    expect(handlerSource).toContain("LEFT JOIN files fs ON fs.id = r.source_id AND fs.status = 'active' AND fs.deleted_at IS NULL");
    expect(handlerSource).toContain("LEFT JOIN files fs_mm ON fs_mm.id = r.source_id AND fs_mm.status = 'active' AND fs_mm.deleted_at IS NULL");
    expect(handlerSource).toContain('EXISTS (SELECT 1 FROM notes WHERE resource_id = r.id AND deleted_at IS NULL)');
  });
});
