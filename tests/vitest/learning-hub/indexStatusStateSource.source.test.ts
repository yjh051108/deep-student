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
  const multimodalServiceSource = readFileSync(
    resolve(process.cwd(), 'src-tauri/src/vfs/multimodal_service.rs'),
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
    expect(handlerSource).toContain('database: State<\'_, Arc<crate::database::Database>>');
    expect(handlerSource).toContain('vfs_db: State<\'_, Arc<VfsDatabase>>');
    expect(handlerSource).toContain('resolve_bound_multimodal_embedding_model_id');
    expect(handlerSource).toContain('match resolve_bound_multimodal_embedding_model_id(&database, &vfs_db)');
    expect(handlerSource).toContain('.map(str::trim)');
    expect(handlerSource).toContain('.filter(|id| !id.is_empty())');
    expect(handlerSource).toContain('save_setting(');
    expect(handlerSource).toContain('embedding.default_multimodal_model_config_id');
    expect(handlerSource).toContain('embedding.default_multimodal_dimension');
    expect(handlerSource).toContain('list_by_modality(&conn, "multimodal")');
    expect(handlerSource).toContain('默认多模态维度 {} 不存在，请重新设置');
    expect(handlerSource).toContain('检测到多个多模态维度绑定了不同模型，请设置默认多模态维度');
    expect(handlerSource).toContain('vl_embedding_capability_accepts_explicit_embedding_assignment_without_multimodal_flag');
    expect(handlerSource).toContain('vl_embedding_capability_rejects_disabled_non_embedding_or_reranker_configs');
    expect(ragExtensionSource).toContain('get_setting("embedding.default_multimodal_model_config_id")');
    expect(ragExtensionSource).toContain('.get_model_assignments()');
    expect(ragExtensionSource).toContain('assignments.vl_embedding_model_config_id');
    expect(ragExtensionSource).toContain('resolve_vl_embedding_model_config_id(dimension_default_id, assignments).is_some()');
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
    expect(viewSource).toContain('const expectedTotal = firstPage.displayTotalResources ?? firstPage.totalResources;');
    expect(viewSource).toContain('while (resources.length < expectedTotal)');
    expect(viewSource).toContain('offset: resources.length');
    expect(viewSource).toContain('if (page.resources.length === 0) break;');
    expect(viewSource).toContain('resources.push(...page.resources)');
    expect(viewSource).toContain('getCompleteIndexStatus({');
    expect(viewSource).not.toContain('limit: 200');
  });

  it('derives one-click workload from backend summary and lets backend drain pending work', () => {
    expect(viewSource).toContain('const textWorkCount = summary.textQueueCount;');
    expect(viewSource).toContain('const pendingMmCount = mmResources.length;');
    expect(viewSource).toContain('await batchIndexPending();');
    expect(viewSource).toContain('if (textWorkCount === 0 && pendingMmCount === 0) {');
    expect(viewSource).not.toContain('Math.max(pendingTextCount, 10)');
    expect(viewSource).not.toContain('const pendingTextCount = summary.pendingCount + summary.failedCount;');
    expect(viewSource).not.toContain('const pendingTextResources = summary.resources.filter(isPendingTextResource);');
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
    expect(apiSource).toContain('textQueueCount: number;');
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
    expect(handlerSource).toContain('pub text_queue_count: i32');
    expect(handlerSource).toContain('pub display_total_resources: i32');
    expect(handlerSource).toContain('fn display_index_state_sql(');
    expect(handlerSource).toContain('include_image_index: Option<bool>');
    expect(handlerSource).toContain('{display_state} as display_index_state');
    expect(handlerSource).toContain('list_conditions.push(format!("({}) = ?", list_display_state_sql));');
    expect(handlerSource).toContain("COALESCE(SUM(CASE WHEN {display_state} = 'indexed' THEN 1 ELSE 0 END), 0) as display_indexed");
    expect(handlerSource).toContain('as text_queue_count');
  });

  it('keeps multimodal business state and unit state in sync at the backend writer', () => {
    expect(multimodalServiceSource).toContain('fn update_mm_index_state_in_business_table');
    expect(multimodalServiceSource).toContain('UPDATE resources SET mm_index_state = ?1');
    expect(multimodalServiceSource).toContain('UPDATE vfs_index_units');
    expect(multimodalServiceSource).toContain("SET mm_state = 'indexed'");
    expect(multimodalServiceSource).toContain('WHERE resource_id = ?3 AND mm_required = 1');
    expect(multimodalServiceSource).toContain('WHERE resource_id = ?4 AND mm_required = 1');
  });

  it('does not silently skip index status rows when list parsing fails', () => {
    expect(handlerSource).toContain('fn read_optional_millis(');
    expect(handlerSource).toContain('fn read_optional_i32(');
    expect(handlerSource).toContain('return Err(format!');
    expect(handlerSource).toContain('Index status row {} parse error: {}');
    expect(handlerSource).not.toContain('rows had parse errors');
  });
});
