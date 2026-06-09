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
  const goVfsServiceSource = readFileSync(
    resolve(process.cwd(), 'desktop-go/internal/vfs/service.go'),
    'utf-8'
  );
  const wailsBridgeSource = readFileSync(
    resolve(process.cwd(), 'src/runtime/wailsBridge.ts'),
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

  it('uses the native runtime facade for direct command invocations', () => {
    expect(viewSource).toContain("import { invoke } from '@/runtime/native';");
    expect(viewSource).not.toMatch(/import\s*\{[^}]*\binvoke\b[^}]*\}\s*from '@tauri-apps\/api\/core'/);
  });

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
    expect(viewSource).toContain('const normalizedCounts = useMemo(');
    expect(viewSource).toContain('normalizeIndexSummaryCounts(summary)');
    expect(viewSource).toContain('indexed: normalizedCounts?.display.indexed ?? 0');
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

  it('derives one-click workload from the same scope as the action', () => {
    expect(viewSource).toContain("const isActionFiltered = selectedType !== 'all' || selectedState !== 'all';");
    expect(viewSource).toContain('const matchesSelectedState = (resource: ResourceIndexStatus): boolean =>');
    expect(viewSource).toContain("selectedState === 'all' || normalizeIndexState(resource.displayIndexState) === selectedState");
    expect(viewSource).toContain('const filteredTextResources = isActionFiltered');
    expect(viewSource).toContain('const textWorkCount = isActionFiltered');
    expect(viewSource).toContain('const pendingMmCount = mmResources.length;');
    expect(viewSource).toContain('await reindexResource(resource.resourceId);');
    expect(viewSource).toContain('await batchIndexPending();');
    expect(viewSource).toContain('if (textWorkCount === 0 && pendingMmCount === 0) {');
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
    expect(apiSource).toContain('textIndexRetryable: boolean;');
    expect(apiSource).toContain('textQueueCount: number;');
    expect(apiSource).toContain('textTotalResources: number;');
    expect(apiSource).toContain('textIndexedCount: number;');
    expect(apiSource).toContain('displayTotalResources: number;');
    expect(apiSource).toContain('displayIndexedCount: number;');
    expect(apiSource).toContain('includeImageIndex?: boolean;');
  });

  it('filters deleted source rows before index status list and summary counts are built', () => {
    expect(goVfsServiceSource).toContain('func (s *Service) UnifiedIndexStatus() (IndexStatusSummary, error)');
    expect(goVfsServiceSource).toContain('func (s *Service) GetResourceUnits(resourceID string) ([]UnitIndexStatus, error)');
    expect(goVfsServiceSource).toContain('func (s *Service) GetAllIndexStatus(input GetIndexStatusInput) (ResourceIndexStatusSummary, error)');
    expect(goVfsServiceSource).toContain('if resourceIsDeleted(resource) {');
    expect(goVfsServiceSource).toContain('if !ok || resourceIsDeleted(resource) {');
    expect(goVfsServiceSource).toContain('func resourceIsDeleted(resource Resource) bool');
    expect(goVfsServiceSource).toContain('status == "deleted" || status == "trash"');
    expect(goVfsServiceSource).toContain('metadataString(resource.Metadata, "deletedAt", "") != ""');
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
    expect(goVfsServiceSource).toContain('type ResourceIndexStatus struct');
    expect(goVfsServiceSource).toMatch(/DisplayIndexState\s+string\s+`json:"displayIndexState"`/);
    expect(goVfsServiceSource).toMatch(/TextIndexRetryable\s+bool\s+`json:"textIndexRetryable"`/);
    expect(goVfsServiceSource).toContain('TextQueueCount        int                   `json:"textQueueCount"`');
    expect(goVfsServiceSource).toContain('TextTotalResources    int                   `json:"textTotalResources"`');
    expect(goVfsServiceSource).toContain('TextIndexedCount      int                   `json:"textIndexedCount"`');
    expect(goVfsServiceSource).toContain('DisplayTotalResources int                   `json:"displayTotalResources"`');
    expect(goVfsServiceSource).toContain('DisplayIndexedCount   int                   `json:"displayIndexedCount"`');
    expect(goVfsServiceSource).toContain('func resourceToIndexStatus(resource Resource) ResourceIndexStatus');
    expect(goVfsServiceSource).toContain('func accumulateResourceIndexSummary(summary *ResourceIndexStatusSummary, status ResourceIndexStatus)');
    expect(goVfsServiceSource).toContain('addResourceDisplayStat(summary, status.DisplayIndexState)');
    expect(goVfsServiceSource).toContain('addResourceTextStat(summary, status.TextIndexState)');
    expect(goVfsServiceSource).toContain('addResourceMMStat(summary, status.MMIndexState)');
    expect(goVfsServiceSource).toContain('summary.TextQueueCount++');
    expect(goVfsServiceSource).toContain('func addResourceDisplayStat(summary *ResourceIndexStatusSummary, state string)');
  });

  it('keeps multimodal business state and unit state in sync at the backend writer', () => {
    expect(multimodalServiceSource).toContain('fn update_mm_index_state_in_business_table');
    expect(multimodalServiceSource).toContain('UPDATE resources SET mm_index_state = ?1');
    expect(multimodalServiceSource).toContain('UPDATE vfs_index_units');
    expect(multimodalServiceSource).toContain("SET mm_state = 'indexed'");
    expect(multimodalServiceSource).toContain('serde_json::from_str::<Vec<serde_json::Value>>');
    expect(multimodalServiceSource).toContain('WHERE resource_id = ?3 AND unit_index = ?4 AND mm_required = 1');
    expect(multimodalServiceSource).toContain("SET mm_state = 'failed'");
    expect(multimodalServiceSource).toContain('WHERE resource_id = ?4 AND mm_required = 1');
  });

  it('routes legacy index-status commands through Wails instead of old Rust handlers', () => {
    for (const command of [
      'vfs_get_all_index_status',
      'vfs_reindex_resource',
      'vfs_batch_index_pending',
    ]) {
      expect(handlerSource).not.toContain(`pub async fn ${command}`);
      expect(handlerSource).not.toContain(`crate::vfs::handlers::${command}`);
    }
    expect(wailsBridgeSource).toContain("if (command === 'vfs_get_all_index_status')");
    expect(wailsBridgeSource).toContain('return await VfsService.GetAllIndexStatus({');
    expect(wailsBridgeSource).toContain("if (command === 'vfs_reindex_resource')");
    expect(wailsBridgeSource).toContain('return await VfsService.ReindexResource(resourceId) as T;');
    expect(wailsBridgeSource).toContain("command === 'vfs_unified_batch_index' || command === 'vfs_batch_index_pending'");
    expect(wailsBridgeSource).toContain('return await VfsService.BatchIndexPending(batchSize) as T;');
    expect(goVfsServiceSource).toContain('func (s *Service) ReindexResource(resourceID string) (int, error)');
    expect(goVfsServiceSource).toContain('func (s *Service) BatchIndexPending(batchSize int) (BatchIndexResult, error)');
  });

  it('keeps old Rust PDF/index processing controls retired while Wails routes the legacy names', () => {
    for (const command of [
      'vfs_cancel_pdf_processing',
      'vfs_retry_pdf_processing',
      'vfs_start_pdf_processing',
    ]) {
      expect(handlerSource).not.toContain(`pub async fn ${command}`);
      expect(handlerSource).not.toContain(`crate::vfs::handlers::${command}`);
    }
    expect(wailsBridgeSource).toContain("if (command === 'vfs_cancel_pdf_processing')");
    expect(wailsBridgeSource).toContain('return await VfsService.CancelPdfProcessing(fileId) as T;');
    expect(wailsBridgeSource).toContain("if (command === 'vfs_retry_pdf_processing')");
    expect(wailsBridgeSource).toContain('return await VfsService.RetryPdfProcessing(fileId) as T;');
    expect(wailsBridgeSource).toContain("if (command === 'vfs_start_pdf_processing')");
    expect(wailsBridgeSource).toContain('return await VfsService.StartPdfProcessing(fileId, optionalStringArg(args, \'startFromStage\') ?? null) as T;');
  });
});
