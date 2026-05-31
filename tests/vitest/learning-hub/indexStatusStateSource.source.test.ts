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
  const ragExtensionSource = readFileSync(
    resolve(process.cwd(), 'src-tauri/src/llm_manager/rag_extension.rs'),
    'utf-8'
  );
  const embeddingRepoSource = readFileSync(
    resolve(process.cwd(), 'src-tauri/src/vfs/repos/embedding_repo.rs'),
    'utf-8'
  );

  it('uses backend multimodal capability instead of duplicating model matching in the UI', () => {
    expect(viewSource).toContain("invoke<MultimodalIndexCapability>('vfs_get_multimodal_index_capability')");
    expect(viewSource).not.toContain("invoke<IndexModelConfig[]>('get_api_configurations')");
    expect(viewSource).not.toContain('resolveImageIndexCapability');
  });

  it('recognizes VL embedding model assignment as multimodal index capability fallback', () => {
    expect(handlerSource).toContain('vfs_get_multimodal_index_capability');
    expect(handlerSource).toContain('get_vl_embedding_model_config().await');
    expect(handlerSource).toContain('fn is_usable_vl_embedding_config(config: &ApiConfig) -> bool');
    expect(handlerSource).toContain('Ok(config) if is_usable_vl_embedding_config(&config)');
    expect(handlerSource).toContain('vl_embedding_capability_accepts_explicit_embedding_assignment_without_multimodal_flag');
    expect(handlerSource).toContain('vl_embedding_capability_rejects_disabled_non_embedding_or_reranker_configs');
    expect(ragExtensionSource).toContain('get_setting("embedding.default_multimodal_model_config_id")');
    expect(ragExtensionSource).toContain('.get_model_assignments()');
    expect(ragExtensionSource).toContain('assignments.vl_embedding_model_config_id');
    expect(ragExtensionSource).toContain('dimension_default_available || assignment_available');
  });

  it('groups visible rows by backend display index state', () => {
    expect(viewSource).not.toContain('resolveResourceDisplayState');
    expect(viewSource).not.toContain("const states = [textState, normalizeIndexState(resource.mmIndexState)]");
    expect(viewSource).toContain('const resolvedDisplayRows = useMemo<DisplayIndexRow[]>');
    expect(viewSource).toContain('displayState: normalizeIndexState(resource.displayIndexState)');
    expect(viewSource).toContain('const groupedDisplayRows = useMemo(() =>');
    expect(viewSource).toContain('displayIndexStats');
    expect(viewSource).toContain('indexed: summary?.displayIndexedCount ?? 0');
    expect(viewSource).toContain('const displayedRows = selectedState ===');
    expect(viewSource).not.toContain("stateFilter: selectedState === 'all' ? undefined : selectedState");
  });

  it('loads every index status page before deriving visible rows or one-click workload', () => {
    expect(viewSource).toContain('const getCompleteIndexStatus = async (');
    expect(viewSource).toContain('while (resources.length < firstPage.totalResources)');
    expect(viewSource).toContain('offset: resources.length');
    expect(viewSource).toContain('resources.push(...page.resources)');
    expect(viewSource).toContain('getCompleteIndexStatus({');
    expect(viewSource).not.toContain('limit: 200');
  });

  it('derives one-click workload from backend summary and lets backend drain pending work', () => {
    expect(viewSource).toContain('const displayWorkCount = summary.displayPendingCount + summary.displayFailedCount;');
    expect(viewSource).toContain('const pendingTextResources = summary.resources.filter(isPendingTextResource);');
    expect(viewSource).toContain('const pendingTextCount = pendingTextResources.length;');
    expect(viewSource).toContain('const pendingMmCount = mmResources.length;');
    expect(viewSource).toContain('await batchIndexPending();');
    expect(viewSource).toContain('if (displayWorkCount === 0) {');
    expect(viewSource).not.toContain('Math.max(pendingTextCount, 10)');
    expect(viewSource).not.toContain('const pendingTextCount = summary.pendingCount + summary.failedCount;');
    expect(viewSource).not.toContain('const pendingMmCount = summary.mmPendingCount + summary.mmFailedCount;');
  });

  it('asks the backend for image-aware display state only after capability is known', () => {
    expect(viewSource).toContain('const includeImageIndex = MULTIMODAL_INDEX_ENABLED && multimodalCapability.ready;');
    expect(viewSource).toContain('includeImageIndex,');
  });

  it('exposes backend display state and display counts in the API contract', () => {
    const apiSource = readFileSync(
      resolve(process.cwd(), 'src/api/vfsUnifiedIndexApi.ts'),
      'utf-8'
    );
    expect(apiSource).toContain('displayIndexState: string;');
    expect(apiSource).toContain('displayTotalResources: number;');
    expect(apiSource).toContain('displayIndexedCount: number;');
    expect(apiSource).toContain('includeImageIndex?: boolean;');
  });

  it('filters deleted source rows before index status list and summary counts are built', () => {
    expect(handlerSource).toContain("WHERE resource_id IS NOT NULL AND status = 'active' AND deleted_at IS NULL");
    expect(handlerSource).toContain("LEFT JOIN files fs ON fs.id = r.source_id AND fs.status = 'active' AND fs.deleted_at IS NULL");
    expect(handlerSource).toContain("LEFT JOIN files fs_mm ON fs_mm.id = r.source_id AND fs_mm.status = 'active' AND fs_mm.deleted_at IS NULL");
    expect(handlerSource).toContain('EXISTS (SELECT 1 FROM notes WHERE resource_id = r.id AND deleted_at IS NULL)');
  });

  it('uses the same active source-row universe for one-click pending queue counts', () => {
    expect(embeddingRepoSource).toContain('const PENDING_RESOURCES_WHERE_SQL');
    expect(embeddingRepoSource).toContain('EXISTS (SELECT 1 FROM notes WHERE resource_id = r.id AND deleted_at IS NULL)');
    expect(embeddingRepoSource).toContain("EXISTS (SELECT 1 FROM files WHERE status = 'active' AND deleted_at IS NULL AND (resource_id = r.id OR id = r.source_id))");
    expect(embeddingRepoSource).toContain('EXISTS (SELECT 1 FROM exam_sheets WHERE deleted_at IS NULL AND (resource_id = r.id OR id = r.source_id))');
    expect(embeddingRepoSource).toContain('EXISTS (SELECT 1 FROM translations WHERE resource_id = r.id AND deleted_at IS NULL)');
    expect(embeddingRepoSource).toContain('EXISTS (SELECT 1 FROM essays WHERE resource_id = r.id AND deleted_at IS NULL)');
    expect(embeddingRepoSource).toContain('EXISTS (SELECT 1 FROM mindmaps WHERE resource_id = r.id AND deleted_at IS NULL)');
  });

  it('computes display state and display counts in the backend contract', () => {
    expect(handlerSource).toContain('pub display_index_state: String');
    expect(handlerSource).toContain('pub display_total_resources: i32');
    expect(handlerSource).toContain('fn display_index_state_sql(');
    expect(handlerSource).toContain('include_image_index: Option<bool>');
    expect(handlerSource).toContain('{display_state} as display_index_state');
    expect(handlerSource).toContain('list_conditions.push(format!("({}) = ?", list_display_state_sql));');
    expect(handlerSource).toContain("COALESCE(SUM(CASE WHEN {display_state} = 'indexed' THEN 1 ELSE 0 END), 0) as display_indexed");
  });
});
