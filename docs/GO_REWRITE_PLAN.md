# Go Rewrite Plan

## Decision

Rewrite the desktop shell and native backend in Go as a lean functional rebuild, while preserving the existing React/Vite frontend and user experience one-to-one.

Recommended shell: Wails v3.

Operational checkpoint: progress management is part of the rewrite goal. Keep `docs/GO_REWRITE_PROGRESS.md` updated after each meaningful rewrite slice with built/not-built surfaces, verification evidence, and the next work queue before moving to another major slice.

Reasoning:

- The current app uses Tauri 2 with a Rust backend. The Rust compile surface is large and slow.
- The frontend is already a web app, so a Go + native WebView shell preserves the UI investment.
- Wails supports Go service bindings, generated TypeScript bindings, events, dialogs, windows, and packaging.
- Wails v3 is still alpha, so migration must be side-by-side until the replacement shell proves the critical paths.
- The current codebase is significantly larger than the product needs. The rewrite should reduce feature surface, command count, dependencies, and native integration complexity instead of copying Rust modules into Go.
- Frontend behavior, navigation, screens, interaction patterns, and user-visible workflows should remain equivalent while the backend is simplified.

Do not delete `src-tauri/` during early migration. Keep both shells until the Go shell passes functional parity checks.

## Rewrite Philosophy

This is not a one-to-one port.

The Go rewrite should be based on product functions and domain logic, with frontend experience preserved:

- Keep capabilities that directly support the learning workflow.
- Merge duplicated command families into small service APIs.
- Delete debug-only, test-only, abandoned, and speculative surfaces unless they are still required.
- Prefer a smaller durable data model over preserving every legacy internal abstraction.
- Preserve user data through explicit import/migration paths, not by cloning every old repository layer.
- Treat existing Rust code as behavior reference and schema evidence, not as the architecture to reproduce.
- Do not remove or visibly simplify existing frontend workflows merely to make the backend easier.
- If a backend feature is deleted, the frontend must either no longer expose it because it was not user-facing, or expose an equivalent workflow through a leaner service.

Success is measured by a smaller, faster, more maintainable app with the important workflows intact.

## Current Migration Status

- Wails CLI installed and verified: `wails3 v3.0.0-alpha.98`.
- `wails3 doctor` reports the Windows development environment is ready for Wails development.
- `desktop-go/` contains a minimal Wails shell with embedded placeholder assets.
- `SettingsService`, `SystemService`, `FileService`, `NotesService`, `TodoService`, `DstuService`, `QbankService`, `ChatService`, and `VfsService` are registered as the first Wails services.
- Wails TypeScript bindings generate successfully into `src/runtime/wails-bindings`: 9 services, 146 methods, 97 models.
- Main frontend native runtime facade exists at `src/runtime/native.ts`.
- Wails settings bridge exists at `src/runtime/wailsBridge.ts`.
- Go `SettingsService` now supports both single-key and batch settings reads/writes, plus lightweight attachment root config compatibility.
- Go `SystemService` now covers the first path/log commands: app data directory, log folder opening, debug log directory creation, and frontend log reporting.
- Go `FileService` now covers the first desktop local-file commands needed by attachment and textbook workflows: read bytes, get size, and copy file.
- Go `NotesService` now covers the first Notes surfaces: preference KV used by tabs, sort mode, note annotations, and folder storage hooks; plus note image asset save/list/delete/resolve and base64 preview reads.
- Go `TodoService` now covers the first complete study-data workflow through a lean durable JSON store: inbox/list/item CRUD, favorite toggles, reorder, today/overdue/upcoming/completed/search queries, active todo summary, and Pomodoro focus-session records/stats.
- Go `DstuService` now covers the Notes-facing DSTU basics through a lean JSON-backed notes resource store: list/get/create/update/delete/search/getContent, metadata/favorite writes, and Markdown file import/batch import. Note create/update/import now registers source-stable `note_xxx` resources into the Go hybrid VFS so Learning Hub context injection can resolve current note content without cloning the Rust VFS stack. Existing DSTU commands can also expose Go hybrid VFS-backed file/image/textbook nodes to Learning Hub, including get/content/delete/metadata/favorite operations, DSTU create for file/image/textbook uploads, and `textbooks_add` local file imports into the same hybrid VFS.
- Go `QbankService` now covers the core practice workflow through a lean JSON-backed question store: list/search/get/create/update/delete/batch delete, favorite toggles, answer submission, progress reset, and stats refresh. Question create/update now registers source-stable `q_xxx` resources into the Go hybrid VFS as lightweight `exam` files, returning `resource_id`/`resource_hash` without cloning the old all-virtual VFS. AI grading, sync conflict resolution, source-image cropping, mock exams, generated papers, and temporal analytics remain `replace` work rather than copied Rust subsystems.
- Go `ChatService` now covers the first chat shell workflow through a lean JSON-backed session/group/message/block store: create/get/load/save/list/count/archive/delete/move/branch sessions, create/get/list/update/reorder groups, session tag add/remove/list/batch reads, message summary stats, local send/continue placeholders, stream cancellation state, retry/edit-and-resend state repair, tool approval and ask-user response recording, message deletion, block content updates, and streaming block upserts. Real LLM streaming is still a separate replacement slice rather than a Rust pipeline clone.
- Resource storage direction is now native hybrid VFS, not the old all-virtual VFS. Go should keep real files visible under app-data/library/import roots, with a small resource index for IDs, metadata, hashes, ref counts, tags, source links, search units, and preview/extraction state.
- Go `VfsService` now implements the first native hybrid VFS resource-index, context-ref, lightweight attachment, file CRUD/content/bookmarks/metadata, compact index-status, local search, file-list, legacy-alias, media processing status/control, and existing-preview page-image read slices: create/reuse, get resource, exists, increment/decrement refs, source/resource/hash/original-path alias lookup, source/resource path lookup, ref count reads, hash updates, `vfs_get_resource_refs`, `vfs_resolve_resource_refs`, compatibility no-op `vfs_update_path_cache`, `vfs_upload_attachment`, `vfs_get_attachment`, `vfs_get_attachment_content`, `vfs_upload_file`, `vfs_get_file`, `vfs_delete_file`, `vfs_get_file_content`, `textbooks_update_bookmarks`, compact unit/status queries, text chunk inspection, no-op index maintenance commands, `vfs_rag_search` as title/source/metadata/text fallback search, `vfs_list_files` over file/image/textbook/attachment-like resources, truthful lightweight `vfs_get_pdf_processing_status` / batch / cancel / retry / start compatibility, and `vfs_get_pdf_page_image` reads from already-available previewJson image refs. `DstuService` uses this layer for `textbooks_add`, including same-hash promotion from regular file to textbook. Learning Hub file/textbook/image previews now prefer file-like VFS content reads and only fall back to attachment content for legacy resources. Legacy virtual/resource URIs are parsed only as aliases to existing hybrid VFS records; visible files and `vfs-go.json` remain the primary model. Real OCR, PDF preview/page image generation, extracted text generation, embedding/LanceDB, multimodal indexing, and semantic RAG search remain replacement slices.
- Settings config loading/saving uses batch facade calls where practical, preserving the existing Settings UI while reducing backend command granularity.
- Initial settings paths have moved from direct Tauri calls to the native facade:
  - `src/hooks/useAppInitialization.ts`
  - `src/hooks/useSystemSettings.ts`
  - generic `getSetting` / `saveSetting` / `deleteSetting` in `src/utils/settingsApi.ts`
  - settings read/write batches in `src/features/settings/components/useSettingsConfig.ts`
- Direct frontend invoke references have dropped from 995 to the high 850s; the current inventory is 859 invokes / 643 unique commands after adding Wails-routed system, file, Todo, chat, study-data, and VFS facade commands.
- The native command triage now marks 211 commands as `merge` and 167 as `replace`, including the 20 frontend `todo_*` commands, 5 `pomodoro_*` commands, 2 Notes preference commands, 4 Notes asset commands, `get_image_as_base64`, 11 Notes-facing DSTU/import commands, `textbooks_add`, 14 core `qbank_*` commands, 30 chat session/group/tag/stats/interaction/message/block commands, VFS resource-index/context-ref/attachment/file/compact-index/local-search/list commands, 5 PDF/image processing status/control commands, and existing-preview `vfs_get_pdf_page_image` reads as `merge`, folded into smaller Go services instead of preserving the old Rust VFS/repository/handler stack.
- `npm run go:check` passes Go tests and smoke-starts the migration shell in CLI mode.

Packaging dependencies still missing for release work:

- NSIS.
- Windows SDK packaging/signing tools.

## Current Backend Shape

The current `src-tauri` backend has roughly:

- 690 Rust files.
- 712 `#[tauri::command]` endpoints.
- Large service domains:
  - `chat_v2` - chat/session/block/tool pipeline.
  - `vfs` - virtual file system, resources, textbooks, questions, todos, indexing.
  - `data_governance` - backup, archive, migration, sync, audit.
  - `llm_manager` - model/vendor orchestration.
  - `dstu` - finder/protocol/file semantics.
  - `memory` - memory-as-VFS.
  - `mcp` - native MCP transports.
  - `cloud_storage` - cloud sync.
  - OCR/PDF/document processing modules.

Top command files by endpoint count:

- `src-tauri/src/commands.rs` - 140 commands.
- `src-tauri/src/vfs/handlers.rs` - 88 commands.
- `src-tauri/src/cmd/notes.rs` - 39 commands.
- `src-tauri/src/dstu/handlers.rs` - 31 commands.
- `src-tauri/src/memory/handlers.rs` - 29 commands.
- `src-tauri/src/vfs/todo_handlers.rs` - 25 commands.
- `src-tauri/src/cmd/enhanced_anki.rs` - 22 commands.
- `src-tauri/src/essay_grading/mod.rs` - 20 commands.

This is a full product migration, not a mechanical language conversion.

## Target Architecture

```text
desktop-go/
  cmd/deep-student/
    main.go
  internal/app/
    app.go
    lifecycle.go
    paths.go
    events.go
  internal/bindings/
    services registered for Wails
  internal/store/
    sqlite.go
    migrations.go
  internal/settings/
  internal/chat/
  internal/vfs/
  internal/memory/
  internal/llm/
  internal/mcp/
  internal/pdf/
  internal/ocr/
  internal/backup/
  internal/cloud/
  internal/system/
  frontend/
    adapters for Wails runtime and generated bindings
```

Frontend compatibility layer:

```text
src/runtime/
  native.ts
  tauriRuntime.ts
  wailsRuntime.ts
```

All frontend code should stop importing `@tauri-apps/api` directly. Instead it should call a narrow runtime facade:

- `native.invoke(command, args)`
- `native.listen(event, handler)`
- `native.emit(event, payload)`
- `native.path.appDataDir()`
- `native.dialog.open/save/message/confirm()`
- `native.window.*`
- `native.assetUrl(path)`

This lets the frontend support both Tauri and Wails during migration.

## Migration Order

### Phase 0 - Compatibility Inventory

Goal: list every native call from the frontend, then decide whether to keep, merge, replace, or delete it.

Tasks:

- Generate a command inventory from all frontend `invoke(...)` calls.
- Generate an event inventory from all frontend `listen(...)` / `emit(...)` calls.
- Classify commands by domain: settings, files, chat, vfs, memory, mcp, pdf, backup, system.
- Mark each command as keep, merge, replace, defer, or delete.
- Identify no-op, debug-only, test-only, and low-value commands that should not be migrated.
- Collapse command groups into smaller Go service interfaces before implementation.

Exit criteria:

- Every frontend native call has an owner domain, migration status, and keep/delete rationale.

### Phase 1 - Go Shell And Runtime Facade

Goal: run the existing frontend inside a Go desktop shell.

Tasks:

- Add `desktop-go/` with Wails app scaffold.
- Point Wails dev server to the existing Vite frontend.
- Add `src/runtime/native.ts` and route common native calls through it.
- Keep Tauri as the default shell until Wails opens the app and serves assets correctly.

Exit criteria:

- `go run` / `wails3 dev` opens the app shell.
- Frontend loads without blank screen.
- Basic window, path, logging, and settings calls work through the runtime facade.

### Phase 2 - Core Persistence

Goal: migrate the database foundation before feature services.

Tasks:

- Port app data directory resolution.
- Port SQLite connection management.
- Port settings read/write.
- Port migrations needed for existing local databases.
- Keep database schema compatible unless a migration plan says otherwise.

Exit criteria:

- Existing user data can be opened read-only by the Go backend.
- Settings round-trip through Go.

### Phase 3 - Low-Risk Commands

Goal: reduce frontend dependency on Tauri commands quickly.

Migrate:

- Settings.
- Logs.
- Path helpers.
- File read/write/hash/copy helpers.
- Anki Connect HTTP helpers.
- Todo commands.
- Template commands.

Exit criteria:

- A small user workflow runs fully through Go services.

### Phase 4 - Lean Study Data Core

Goal: rebuild the durable learning data model around the app's real workflows.

Storage model:

- Use a hybrid VFS instead of an all-virtual VFS.
- Store imported files and generated assets as native files in stable app-data subdirectories that users and support tools can inspect.
- Keep a compact resource index for resource IDs, canonical paths, original paths, content hashes, MIME/type, ownership, tags, ref counts, favorite/status fields, source links, preview state, extracted text state, and search/index units.
- Treat virtual URIs as compatibility aliases, not the primary storage model.
- Prefer hard-link/copy-on-import plus hash dedupe where safe; never require a hidden blob database just to open a textbook, note asset, or question attachment.
- Keep notes, textbooks, questions, chat context refs, and attachments able to point at the same indexed resource without losing native path visibility.

Keep or rebuild:

- Notes, resources, textbooks, and question-bank data that users actively need.
- Review plans and spaced repetition.
- Import/export flows needed for user data continuity.

Simplify or delete:

- All-virtual VFS layers that exist only to support old internal abstractions.
- Repository wrappers with no product-facing distinction.
- Memory-as-VFS behavior unless it proves necessary for the simplified learning flow.
- Duplicate indexing/status tables where a smaller model is enough.

Exit criteria:

- Existing notes, textbooks, questions, and resources open in the Go shell.
- CRUD and search smoke tests pass.

### Phase 5 - Lean Chat And LLM

Goal: rebuild chat around the minimum durable interaction model.

Keep or rebuild:

- Chat session/group/block persistence.
- Streaming event pipeline.
- LLM vendor/provider orchestration.
- Tool calling and approvals.
- Usage accounting.

Defer or delete:

- Multi-agent/debug surfaces unless they are core to the product.
- MCP transports that are not required by the target user workflow.
- Block variants or pipeline branches that only exist to support legacy experiments.

Exit criteria:

- A multi-turn chat with streaming, tool output, cancellation, and persisted history works in Go.

### Phase 6 - Heavy Native Processing

Goal: replace Rust-native document/OCR/PDF machinery.

Migrate or wrap:

- Pdfium loading and PDF rendering/text extraction.
- OCR adapters.
- Document parsing for docx/xlsx/pptx/epub/rtf/csv.
- Vector indexing and reranking.
- Backup and cloud sync.

Exit criteria:

- PDF upload, preview, OCR/indexing, and backup workflows pass parity tests.

### Phase 7 - Release Replacement

Goal: retire Tauri only after Wails reaches parity.

Tasks:

- Replace Windows packaging.
- Validate auto-update strategy.
- Validate installer/uninstaller behavior.
- Validate crash/log paths.
- Remove Tauri dependencies from frontend.
- Remove `src-tauri/` only after a tagged parity release.

Exit criteria:

- Go shell is the only release artifact.
- Tauri build scripts are removed.
- Existing local data is preserved.

## Go Package Mapping

| Rust area | Go package |
| --- | --- |
| `commands.rs` general commands | `internal/system`, `internal/settings`, `internal/files` |
| `database` | `internal/store` |
| `vfs` | `internal/vfs` as a native hybrid resource index plus visible file storage; keep `internal/dstu` as the Notes-facing adapter over that model instead of reviving the all-virtual Rust VFS |
| `chat_v2` | `internal/chat` |
| `llm_manager`, `providers`, `vendors` | `internal/llm` |
| `memory` | `internal/memory` |
| `mcp` | `internal/mcp` |
| `pdfium_utils`, `pdf_ocr_service`, `ocr_adapters` | `internal/pdf`, `internal/ocr` |
| `data_governance`, `backup_*` | `internal/backup`, `internal/governance` |
| `cloud_storage` | `internal/cloud` |
| `dstu` | `internal/dstu` |

## Risk Register

- Wails v3 is alpha: expect API changes. Keep migration isolated in `desktop-go/`.
- Tauri command naming uses snake_case strings. Wails generated bindings are service/method oriented. The runtime facade should preserve existing names until the frontend is cleaned up.
- SQLite schema compatibility is critical. Do not rewrite schema casually.
- Pdfium and OCR are the highest native-integration risks.
- Current backend uses event-heavy streaming. Event naming and payload shapes must be locked with tests before migration.
- Auto-updater and installer behavior must be validated separately from app runtime.

## First Implementation Slice

Start with a non-destructive dual-shell slice:

1. Create `desktop-go/`.
2. Add a Wails app shell that loads the existing Vite frontend.
3. Add a Go `SettingsService` with `GetSetting` and `SaveSetting`.
4. Add `src/runtime/native.ts`.
5. Move one frontend area from direct `@tauri-apps/api/core` usage to the runtime facade.
6. Verify both shells still work.
7. Start a keep/delete inventory for native commands before porting more services.

The first frontend target should be settings/app initialization because it is small, visible, and used early at startup.

## Verification Strategy

- Inventory tests: generated command/event lists stay in sync.
- Facade tests: `native.invoke` routes correctly in Tauri and Wails modes.
- Database tests: Go services can read existing SQLite fixtures.
- Parity smoke tests: same frontend workflow succeeds in Tauri shell and Go shell.
- Release smoke tests: Windows installer, app launch, data path, logs, and updater are checked separately.
