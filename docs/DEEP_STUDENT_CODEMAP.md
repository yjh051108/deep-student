# Deep Student Codemap

Last updated: 2026-05-30

This codemap is a working engineering map for systematic debugging. It is based on read-only repository scans and is meant to answer: "where is the source of truth, which path moves data, and where can state drift?"

## Top Level

- `src/`: React/Vite frontend, app shell, feature UIs, Zustand stores, Tauri invoke wrappers, event listeners, debug panels, i18n, shared UI.
- `src-tauri/`: Tauri 2 Rust backend, command registry, SQLite services, VFS/resource storage, Chat V2 pipeline, OCR/PDF/media processing, LLM providers, MCP, sync and data governance.
- `src-tauri/migrations/`: SQL migrations for `vfs`, `chat_v2`, `llm_usage`, `mistakes`.
- `tests/`: frontend Vitest and Playwright suites.
- `src-tauri/tests/`: Rust integration, security, sync, migration, and tool tests.
- `scripts/`: build/release/dev helpers.
- Generated/heavy artifacts: `dist/`, `target/`, `node_modules/`, `test-results/`.

## App Shell

- `src/main.tsx`: React root. Initializes platform classes, error boundaries, overlay/dialog providers, frontend logging to Tauri, settings/MCP bootstrap.
- `src/App.tsx`: main shell and view orchestration. This is the closest thing to routing.
- `src/types/navigation.ts`: canonical `CurrentView` union.
- `src/app/navigation/canonicalView.ts`: legacy/current view canonicalization.
- `src/app/components/ViewLayerRenderer.tsx`: persistent view renderer. Visited views can stay mounted while hidden.
- `src/lazyComponents.tsx`: lazy imports for major pages.
- `src/config/navigation.ts`: sidebar/navigation definitions.

Main drift risk: view state is split across `App.tsx`, `useNavigationHistory`, `viewStore`, Learning Hub navigation context, and feature-specific stores. A hidden mounted view can keep listeners/timers alive.

## Frontend State Sources

- `src/stores/viewStore.ts`: global current-view mirror.
- `src/stores/uiStore.ts`: shell UI state such as panel collapse.
- `src/stores/systemStatusStore.ts`: migration/maintenance/system banners.
- `src/stores/settingsShellStore.ts`: settings tab/routing state.
- `src/features/chat/core/store/createChatStore.ts`: per-session vanilla Zustand Chat V2 store.
- `src/features/chat/adapters/TauriAdapter.ts`: Chat V2 frontend/backend bridge.
- `src/features/pdf/stores/pdfProcessingStore.ts`: global media/PDF processing status map, keyed by `sourceId`.
- `src/features/learning-hub/stores/finderStore.ts`: Learning Hub finder path, items, history, search, selection, stale request guards.
- `src/features/learning-hub/stores/recentStore.ts`: persisted frontend-only recent resources.
- `src/features/learning-hub/stores/desktopStore.ts`: persisted frontend-only desktop shortcuts.
- `src/mcp-debug/registerStores.ts`: useful index of major debug-visible stores.

Important pattern: many UI bugs are not missing data, but different stores disagreeing about the same entity.

## Tauri Backend

- `src-tauri/src/main.rs`: thin binary entrypoint calling `deep_student_lib::run()`.
- `src-tauri/src/lib.rs`: Tauri builder, plugin setup, managed state, command registration, custom protocols.
- `src-tauri/src/lib.rs` around `generate_handler!`: backend command index and a key mismatch point for frontend invoke names.
- `src-tauri/src/lib.rs` `build_app_state`: constructs managers/services/databases.
- `src-tauri/src/commands.rs`: shared command definitions and `AppState`.

Backend databases and services:

- `src-tauri/src/database/mod.rs`: primary SQLite wrapper, settings, document tasks, Anki, legacy/session/control tables.
- `src-tauri/src/database/manager.rs`: pooled primary DB access.
- `src-tauri/src/chat_v2/database.rs`: separate `chat_v2.db`.
- `src-tauri/src/vfs/database.rs`: separate `vfs.db` plus blob storage.
- `src-tauri/src/data_governance/`: backup, restore, migration status, diagnostics.
- `src-tauri/src/llm_manager/` and `src-tauri/src/providers/`: LLM config/provider execution.

Backend drift risk: some paths use primary DB, some use `chat_v2.db`, some use `vfs.db`, and frontend also persists state locally.

## Chat V2

Frontend:

- `src/features/chat/pages/ChatV2Page.tsx`: Chat V2 page, Learning Hub embedded sidebar/sheet, session-level UI.
- `src/features/chat/core/store/createChatStore.ts`: ChatStore creation and optimistic local state.
- `src/features/chat/core/store/*Actions.ts`: store actions split by concern.
- `src/features/chat/core/middleware/eventBridge.ts`: backend stream event application.
- `src/features/chat/adapters/TauriAdapter.ts`: invokes backend, listens to `chat_v2_event_{sessionId}` and `chat_v2_session_{sessionId}`.
- `src/features/chat/context/*`: context definitions and VFS ref formatting.
- `src/features/chat/registry/*`: block, event, and mode registries.
- `src/features/chat/skills/*`: builtin skills and tool schemas.

Backend:

- `src-tauri/src/chat_v2/handlers/send_message.rs`: `chat_v2_send_message`.
- `src-tauri/src/chat_v2/pipeline/`: prompt assembly, retrieval, tool loop, variants, stream handling.
- `src-tauri/src/chat_v2/tools/`: tool executors, including memory and resource/mindmap executors.
- `src-tauri/src/chat_v2/repo.rs`: Chat V2 persistence.

Send flow:

```text
ChatV2Page/InputBar
 -> ChatStore optimistic state
 -> TauriAdapter.executeSendMessage
 -> chat_v2_send_message
 -> ChatV2Pipeline
 -> LLM/tool/retrieval work
 -> backend emits chat_v2_event_{sessionId}
 -> TauriAdapter/eventBridge applies stream blocks
 -> ChatStore and chat_v2.db converge
```

Risk points:

- optimistic frontend state plus streamed backend events plus DB persistence.
- retry/edit/continue/variant state.
- hidden mounted views retaining session listeners.
- stringly typed events and command names.

## Chat Attachment And OCR Pipeline

Frontend files:

- `src/features/chat/components/input-bar/InputBarUI.tsx`: drag/drop, upload, polling, attachment chip/status UI.
- `src/features/chat/components/input-bar/useInputBarV2.ts`: send gating and non-multimodal fallback.
- `src/features/chat/components/input-bar/injectModeUtils.ts`: selected/ready inject mode source of truth.
- `src/features/chat/components/input-bar/AttachmentInjectModeSelector.tsx`: image/PDF mode toggles.
- `src/features/pdf/stores/pdfProcessingStore.ts`: `statusMap[sourceId]`.
- `src/api/vfsPdfProcessingApi.ts`: `vfs_start_pdf_processing`, batch status, retry/cancel wrappers.
- `src/features/chat/context/vfsRefApi.ts`: `vfs_upload_attachment` and VFS ref resolution.
- `src/features/chat/context/definitions/image.ts`: converts resolved image resource into image blocks, OCR XML, or `<ocr_status>`.

Backend files:

- `src-tauri/src/vfs/handlers.rs`: `vfs_upload_attachment`, processing status APIs.
- `src-tauri/src/vfs/pdf_processing_service.rs`: image/PDF media pipeline, OCR, ready modes, events.
- `src-tauri/src/chat_v2/handlers/ocr.rs`: direct Chat V2 OCR command.
- `src-tauri/src/ocr_adapters/`: OCR adapter types and engine factory.
- `src-tauri/src/local_paddle_ocr.rs`: local Paddle OCR runtime.
- `src-tauri/src/llm_manager/exam_engine.rs` and `model2_pipeline.rs`: OCR model config and raw OCR calls.

Current data flow:

```text
User attaches image
 -> InputBarUI.processFilesToAttachments
 -> vfs_upload_attachment
 -> Rust VFS stores file/resource/blob
 -> upload result returns sourceId + readyModes, often ["image"]
 -> Chat resource/context ref created
 -> attachment.status becomes ready if image mode is ready
 -> media pipeline emits media-processing-progress/completed
 -> usePdfProcessingProgress updates pdfProcessingStore[sourceId]
 -> InputBarUI syncs attachment status
 -> user sends
 -> useInputBarV2 checks selected model capability
 -> text-only model downgrades image ["image"] to ["ocr"]
 -> if "ocr" is not ready, send is blocked with attachment parsing state
 -> when allowed, TauriAdapter resolves context refs
 -> image.formatToBlocks injects image and/or OCR text
```

High-priority mismatch points:

1. `ready` images are not always kept in the poll/sync loop. A later OCR event can update `pdfProcessingStore` but not the attachment object, then terminal store cleanup can leave stale attachment status.
2. `start_pipeline` no-ops if an image task is already running. A later explicit OCR need may not upgrade the in-flight task.
3. Text-only fallback requires usable OCR text. If OCR is disabled, fails, times out, or returns low-quality text, `readyModes` never gains `ocr`, so the UI can remain blocked.
4. `image.ts` has a fallback `<ocr_status>` when OCR is unavailable, but send gating can block before reaching that graceful degradation.

Useful tests:

- `src/features/chat/components/input-bar/__tests__/InputBarUI.mediaReadyContract.source.test.ts`
- `tests/vitest/chat-v2/injectModeUtils.test.ts`
- `tests/vitest/chat-v2/pdfProcessingStore.failedStages.test.ts`
- `tests/vitest/chat-v2/vfsUploadReadyModes.source.test.ts`
- `tests/vitest/chat-v2/vfsRefOcrQuality.source.test.ts`
- `tests/vitest/chat-v2/pdfProcessingService.source.test.ts`

Useful debug panels:

- `src/debug-panel/plugins/MediaProcessingDebugPlugin.tsx`
- `src/debug-panel/plugins/AttachmentPipelineTestPlugin.tsx`

## Learning Hub And VFS

Frontend:

- `src/features/learning-hub/LearningHubPage.tsx`: page shell, tabs, active tab, split layout, persisted panel layout.
- `src/features/learning-hub/LearningHubSidebar.tsx`: main resource list/sidebar, refresh, delete, DSTU watch handling.
- `src/features/learning-hub/LearningHubSidebarV2.tsx`: alternate finder sidebar.
- `src/features/learning-hub/LearningHubNavigationContext.tsx`: global Learning Hub navigation bridge used by `App.tsx`.
- `src/features/learning-hub/stores/finderStore.ts`: finder source of truth.
- `src/features/learning-hub/apps/UnifiedAppPanel.tsx`: app/content panel.
- `src/features/learning-hub/apps/views/*ContentView.tsx`: file/image/PDF/note/textbook previews.
- `src/features/learning-hub/components/finder/FinderFileList.tsx`: grid/list rendering.
- `src/features/learning-hub/components/finder/FinderFileItem.tsx`: item card rendering.
- `src/features/learning-hub/views/IndexStatusView.tsx`: index status UI.

Frontend APIs:

- `src/dstu/api.ts`: `dstu_list`, `dstu_get`, `dstu_delete`, `dstu_watch`, search.
- `src/api/vfsFileApi.ts`: file upload/get/list/delete/content.
- `src/api/vfsUnifiedIndexApi.ts`: index status, reindexing, dimensions.
- `src/api/attachmentConfigApi.ts`: attachment root folder config.

Backend:

- `src-tauri/src/dstu/handlers.rs`: DSTU list/get/delete/watch handlers.
- `src-tauri/src/vfs/handlers.rs`: broad VFS command surface.
- `src-tauri/src/vfs/ref_handlers.rs`: VFS reference resolution.
- `src-tauri/src/vfs/index_handlers.rs`: indexing commands.
- `src-tauri/src/vfs/repos/folder_repo.rs`: `folder_items` tree source of truth.
- `src-tauri/src/vfs/repos/file_repo.rs`: file CRUD/deletion.
- `src-tauri/src/vfs/repos/attachment_repo.rs`: attachment compatibility/deletion.
- `src-tauri/src/vfs/repos/resource_repo.rs`: resources.
- `src-tauri/src/vfs/repos/blob_repo.rs`: blob storage.
- `src-tauri/src/vfs/indexing.rs`, `index_service.rs`, `lance_store.rs`, `multimodal_service.rs`: text and multimodal indexing.

Known risk points:

1. ID confusion: finder/DSTU nodes use source/business ids, while index rows expose both `resourceId` and `sourceId`. `UnifiedAppPanel` currently loads via `dstu.get("/" + resourceId)`; if a caller passes `res_*`, the panel can go blank.
2. Special views such as `indexStatus`, `memory`, and `desktop` intentionally use empty finder items. If navigation state sticks there, the resource list can look blank.
3. Delete paths differ. `dstu.delete` and `vfs_delete_attachment` are permissive, but `vfs_delete_file` has stricter `file_` id expectations. Mixed routes can make attachments look undeletable.
4. Recents and desktop shortcuts are frontend-persisted; they can survive backend deletes/moves unless watch/fallback cleanup runs.
5. Index counts come from several sources: visible rows, backend summary, text state, multimodal state, model config readiness, embedding dimensions.

## Memory, Tools, And Mindmap

Memory backend:

- `src-tauri/src/memory/scope.rs`: scope roots `全局` and `课题`; topic/general scope resolution.
- `src-tauri/src/memory/service.rs`: memory types/purposes and note storage.
- `src-tauri/src/memory/category_manager.rs`: category summaries.
- `src-tauri/src/memory/auto_extractor.rs`: automatic extraction.
- `src-tauri/src/memory/audit_log.rs`: audit log.
- `src-tauri/src/memory/config.rs`: memory root and settings.

Memory frontend:

- `src/api/memoryApi.ts`: memory command wrappers.
- `src/features/learning-hub/views/MemoryView.tsx`: memory UI.
- `src/features/learning-hub/components/MemoryTreePreview.tsx`: tree preview.

Prompt and routing:

- `src-tauri/src/chat_v2/pipeline/prompt.rs`: memory context assembly for prompts.
- `src-tauri/src/chat_v2/prompt_builder.rs`: prompt blocks.
- `src-tauri/src/chat_v2/pipeline/retrieval.rs`: memory retrieval is tool-first.
- `src-tauri/src/chat_v2/pipeline.rs`: backend enriches/validates group/topic scope.
- `src-tauri/src/chat_v2/tools/memory_executor.rs`: memory tool executor and scope checks.

Tool dispatch:

- `src-tauri/src/chat_v2/pipeline.rs`: executor registration order: resource tools, memory, skills, general executor.
- `src-tauri/src/chat_v2/tools/builtin_resource_executor.rs`: resource and mindmap tool executor.
- `src-tauri/src/chat_v2/tools/skills_executor.rs`: skill load/resolve/persist executor.
- `src-tauri/src/chat_v2/pipeline/tool_loop.rs`: dynamic tool schema loop.
- `src/features/chat/skills/builtin-tools/index.ts`: frontend skill/tool exports.
- `src/features/chat/skills/builtin/dstu-memory-orchestrator.ts`: broad orchestrator skill.

Mindmap:

- `src/features/chat/skills/builtin-tools/mindmap-tools.ts`: prompt/tool contract.
- `src-tauri/src/chat_v2/tools/builtin_resource_executor.rs`: `execute_mindmap_create`, update, delete, edit nodes, versions, diff.
- `src-tauri/src/vfs/repos/mindmap_repo.rs`: normalize/store/version mindmaps.
- `src/features/chat/utils/mindmapCitationParser.ts`: citation parsing.
- `src/features/chat/utils/citationRemarkPlugin.ts`: markdown citation transform.
- `src/features/mindmap/components/mindmap/MindMapEmbed.tsx`: inline embed.

Memory/mindmap drift risks:

1. Backend DB scope wins, but frontend, prompt, session DB, and tool executor all derive scope.
2. General sessions intentionally read global plus all topics, but writes default to global only.
3. Memory UI can expose implementation folders: `全局`, `课题/<name>`, legacy `课题/<id>`, category summaries, and system notes.
4. Mindmap tool stability depends on skill loading before `builtin-mindmap_create` is available.
5. Backend supports very large mindmaps; practical model behavior is controlled mostly by prompt/tool contract.

Relevant tests:

- `tests/vitest/memory/memoryStaleGroupScope.source.test.ts`
- `tests/vitest/memory/memoryScopeWriteContract.source.test.ts`
- `src/features/chat/skills/__tests__/resourceToolContract.test.ts`
- `tests/vitest/chat-v2/skills/activeSkillToolAccess.test.ts`
- `tests/vitest/chat-v2/mindmapEmbedScrollContract.source.test.ts`

## Indexing

Frontend:

- `src/features/learning-hub/views/IndexStatusView.tsx`: index UI, progress listeners, embedding config checks.
- `src/stores/unifiedIndexStore.ts`: global index state.
- `src/services/multimodalRagService.ts`: multimodal feature flag/service calls.
- `src/api/vfsUnifiedIndexApi.ts`: command wrappers.

Backend:

- `src-tauri/src/vfs/handlers.rs`: `vfs_get_all_index_status`, `vfs_batch_index_pending`, `vfs_reindex_resource`.
- `src-tauri/src/vfs/indexing.rs`: text extraction/indexing pipeline.
- `src-tauri/src/vfs/multimodal_service.rs`: multimodal indexing.
- `src-tauri/src/vfs/lance_store.rs`: vector store.
- `src-tauri/src/vfs/repos/embedding_dim_repo.rs`: embedding dimension registry.
- `src-tauri/src/llm_manager/rag_extension.rs`: embedding/reranking model resolution.

Risk points:

- Text and image/multimodal indexing progress are separate timelines but often shown in one UI.
- `resources` rows can outlive active source rows if cleanup/linkage drifts.
- Frontend capability checks may differ from backend model selection.
- Dimensions are model-dependent; stale dimension registration can make status numbers disagree.

## Cross-Cutting Risk Register

1. Multiple truth sources: frontend stores, backend DB rows, emitted events, localStorage/persisted UI, and blob filesystem.
2. Stringly typed invoke/event names.
3. Hidden mounted views with active effects.
4. Resource id ambiguity: `resourceId` (`res_*`) vs source/business ids (`file_*`, `att_*`, `tb_*`, `note_*`, `mm_*`).
5. Terminal processing state cleanup can erase the latest frontend status before attachment state has synchronized.
6. Delete/move operations must update VFS rows, folder items, vector indexes, frontend recents, and open tabs.
7. Memory UI is an implementation projection, not the same abstraction the AI uses.

## Current Debug Priorities

1. Chat image stuck at `附件解析中`: fix readiness/sync so text-only fallback either receives actual OCR or gracefully sends `<ocr_status>` without permanent spinner.
2. Learning Hub blank preview: normalize `UnifiedAppPanel` loading around source id vs `res_*` id.
3. Attachment deletion and ghost recents: unify delete route and recents cleanup.
4. Index status mismatch: derive visible status from one backend summary contract and map text/multimodal subprogress explicitly.
5. Memory display cleanup: keep backend memory scope as-is, but hide implementation artifacts in Learning Hub display.
