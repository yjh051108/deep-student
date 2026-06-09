# Go Rewrite Progress

Last updated: 2026-06-09

Purpose: keep the Go/Wails rewrite resumable when thread context is lost. This file records what exists, what is verified, what is still missing, and what should be built next.

## Goal Boundary

Objective: replace the Tauri/Rust backend with a lean Go/Wails backend while preserving the React frontend experience one-to-one.

Progress management is part of the goal, not optional bookkeeping. After every meaningful rewrite slice, update this file with what was built, what remains missing, verification evidence, and the next work queue so future continuations do not lose direction.

Do not start a new implementation slice until the previous slice has a checkpoint here, including verification results and known gaps.

Do not treat the rewrite as complete until the Go shell can replace the Tauri shell for the core user paths:

- App startup, settings, logs, local files.
- Notes and note assets.
- Todo and Pomodoro.
- DSTU/learning resources.
- Question bank practice.
- Chat sessions, messages, blocks, tags, approvals, streaming, and provider calls.
- Resource/file workflows through the native hybrid VFS.
- Import/export or migration from existing user data.
- Packaging/release smoke path.

Current conservative progress:

- Product replacement progress: about 66%. Many command facades and data stores are in Go/Wails, MCP stdio has a Go/Wails process proxy plus Go-level JSONL/content-length protocol smoke coverage and a live Wails/WebView2 UI stdio smoke for spawn/send/close, the old Rust stdio proxy fallback is retired, legacy diagnostics are cleaned up, Skill file operations now have a Go/Wails service plus a live Wails/WebView2 API-facade CRUD smoke, template management now has a lean Go/Wails JSON store for builtin sync, CRUD, default selection, import, export, and one-time old `mistakes.db/custom_anki_templates` migration plus a live Wails/WebView2 smoke that proves both migration and CRUD/import/export, qbank sync-config compatibility now persists through Go/Wails instead of old Rust, `save_anki_cards` now saves generated cards into the Go local Anki document/card store, textbook local import progress now has a Go/Wails compatibility event bridge, PDF resources now have a Windows PDFium raster preview path over Go hybrid VFS `previewJson`, Windows Go-owned PDFium package layouts now have CLI smoke coverage, PDF/OCR readiness semantics now distinguish extracted text, real OCR, and raster image readiness instead of treating text-layer SVG fallback as multimodal image-ready, the Go shell now embeds a real Vite/React build and passes a live Wails/WebView2 embedded-React/native-binding smoke instead of only a static placeholder or static embed proof, and a live Wails/WebView2 PDF/textbook hybrid VFS smoke now proves the public textbook adapter can import a real temp PDF fixture through Go hybrid VFS, receive import progress events, resolve external resources/files/content/status/index entries, and verify extracted text plus batch PDF readiness. Real OCR/scanned-PDF/textbook open-search parity, provider protocol parity, full MCP runtime/cache/tool-list parity, broader old data migration beyond templates, and installer smoke still determine whether the Go app can actually replace the Tauri app.
- Command/facade migration progress is higher than product replacement: 326 of 328 `merge` commands have Wails bridge routes, direct Tauri blockers for merged commands are 0, and 0 merged commands remain registered in old Rust. The 2 merged commands without Wails routes, `cleanup_orphaned_images` and `save_image_from_base64_path`, are not old Rust registrations; bridge, delete, or reclassify them separately.
- Full Tauri/Rust backend replacement progress: about 50-60%. Keep shrinking old Rust, but do not inflate this number until the Go shell can run the core workflows end to end.

Keep the goal active. This is not done.

Execution preference update:

- The user explicitly accepts higher token/time usage to improve throughput, stability, and quality.
- Prefer high-output execution: parallel read-only subagents where available, broader verification matrices, repeated source checks, and immediate progress-document updates.
- If a subagent role/model is unavailable, continue locally and try another bounded role rather than waiting.
- Prefer `npm run go:smoke:live-wails-core` for milestone runtime proof when MCP stdio, Skills, Templates, and VFS all need live Wails evidence. It reuses one Wails/WebView2 launch for base + MCP + Skills + Templates + VFS; keep the individual live smoke commands for failure isolation.

## Architecture Decisions

- Use Wails v3 side-by-side with the existing Tauri shell until parity is proven.
- Keep the existing React/Vite frontend and user-visible workflows.
- Route native calls through `src/runtime/native.ts`; do not add new direct `@tauri-apps/api` usage for migrated paths.
- Preserve command names through the compatibility bridge where helpful, but merge them into smaller Go services.
- Rebuild by product workflow, not by copying Rust modules.
- Use JSON-backed stores for the current migration shell where that reduces complexity and is enough for functional progress.
- Rust/Tauri is reference and retirement inventory only, not the target runtime. For migrated commands, the effective entrypoint must be Wails/Go; old Rust command bodies should not receive new feature work. Delete or quarantine the old Rust backend after core command parity, data migration, and packaging smoke are proven.
- Move study resources toward native hybrid VFS:
  - Real files stay visible under stable app-data/library/import roots.
  - A compact resource index tracks IDs, paths, hashes, metadata, ref counts, tags, source links, preview/extraction state, and search units.
  - Virtual URIs are compatibility aliases, not the primary storage model.
  - Do not recreate the all-virtual Rust VFS stack.

## Rust Retirement Policy

- Go/Wails is the rebuild target. `src-tauri` exists only as reference code until all core product paths have Go replacements and migration coverage.
- Each migrated command should be routed through `src/runtime/wailsBridge.ts`, tracked as `merge` or `delete` in `scripts/native-triage.mjs`, and recorded here with verification evidence.
- Do not add new product behavior to old Rust command bodies unless it is required to unblock a migration check. Prefer Go services and the native facade.
- Start deleting old Rust in batches only when the related Go workflow has tests, frontend routing, command triage, and at least one smoke path. This avoids breaking the still-incomplete rewrite while keeping deletion as the explicit end state.

## Implemented Services

| Area | Go service | Status | Notes |
| --- | --- | --- | --- |
| Shell | `desktop-go` Wails app | Working shell smoke slice | Wails v3 shell and bindings generation work. The Go embed tree carries a real Vite/React production build with static asset integrity smoke, the Windows Go-owned exe+PDFium smoke path syncs those assets before build, `npm run go:smoke:live-wails` proves a live Wails/WebView2 embedded React mount plus basic Go binding roundtrip, and `npm run go:smoke:live-wails-core` reuses one launch for base + MCP + Skills + Templates + VFS runtime proof. Full workflow navigation and installer smoke remain open. |
| Runtime | `src/runtime/native.ts`, `wailsBridge.ts` | Partial | Main facade exists; Wails v3/WebView runtime detection routes through Go instead of browser fallbacks; live smoke proves `get_app_data_dir` plus settings read/write over Wails. Direct Tauri imports and Wails path assumptions still remain in frontend. |
| Settings | `SettingsService` | Working slice | Single and batch settings read/write/delete, lightweight attachment root config compatibility, backup config persistence, LLM vendor/model/API assignment config persistence, provider strategy config persistence, Go-backed API connection testing for OpenAI-compatible chat-completions providers, Go-backed web-search diagnostic probes, and Go-backed OCR engine diagnostic tests for the settings UI. |
| MCP stdio | `McpService` | Working slice | Go/Wails stdio process proxy for `mcp_stdio_start`, `mcp_stdio_send`, and `mcp_stdio_close`, with JSONL/content-length framing, Wails EventBus message/error/closed events, frontend MCP SDK diagnostics through the native transport, Go protocol smoke coverage for initialize/tools/list/tools/call/close, and live Wails/WebView2 UI smoke proof for spawn/send/close plus active-session close rejection. This is not the full backend MCP cache/tool-list runtime. |
| System | `SystemService` | Working slice | App data dir, logs/debug dirs, frontend log reporting, and WebView localStorage settings persistence for backup/export. |
| File | `FileService` | Working slice | Read bytes/text, save text, file size, copy file. |
| Notes | `NotesService` | Working slice | Preferences, image assets, resolve/list/delete/base64 preview, asset maintenance/stats/vacuum, Go zip import/export over DSTU notes and visible `notes_assets`, and legacy trash wrappers routed through Go DSTU soft-delete semantics. |
| Todo/Pomodoro | `TodoService` | Working slice | Lists, items, active summary, focus records, local-day-safe today stats. |
| DSTU | `DstuService` | Working slice | Notes-facing CRUD/search/content/import plus tag search/list, soft-delete/trash lifecycle, and Canvas note read/append/replace/set utilities, plus Learning Hub file/image/textbook nodes backed by Go hybrid VFS; note create/update/import/delete/restore/purge, DSTU file/image/textbook create, `textbooks_add` local imports with legacy `textbook-import-progress` compatibility events through the Wails EventBus, and folder/path CRUD/move/tree/breadcrumb/resource lookup now use a lean JSON-backed folder store plus Go hybrid VFS metadata sync. |
| Qbank | `QbankService` | Working slice | Question CRUD/search/answers/favorites/stats/progress reset, history/submissions, Notes mention search over Go qbank questions, learning trend, activity heatmap, knowledge stats, timed practice, mock exam scoring, daily practice, check-in calendar, paper generation, CSV preview/import/export, provider-backed AI grading/analyze compatibility with local fallback, and local sync/conflict-status/config compatibility; create/update/submit/AI grade and CSV import now register or resync source-stable questions into Go hybrid VFS. |
| Review Plan | `ReviewPlanService` | Working slice | Lean JSON-backed spaced repetition plans over Go Qbank questions, including create/batch/create-for-exam, due filters, SM-2 style review processing, stats, history, suspend/resume/delete, per-question lookup, list-by-exam, and calendar heatmap. |
| Anki document tasks / AnkiConnect metadata | `AnkiService` | Provider-backed worker + local-save + metadata facade slice | JSON-backed document start/status/control compatibility for CardForge and task dashboards: start a document session, split text into tasks, prefer assigned OpenAI-compatible Anki card provider generation, fall back to local cards when unconfigured/unparseable, emit `anki_generation_event`, query tasks/cards/state/counts, pause/resume, retry trigger, delete session, recover stuck in-flight tasks, and save generated chat cards into the Go local Anki document/card store through `save_anki_cards`. The settings-page AnkiConnect status/deck/model metadata checks now route through Go/Wails. This is not yet OCR, robust document extraction/segmentation, AnkiConnect add-card/APKG parity, or old data migration. |
| Templates | `TemplateService` | Working slice + legacy migration + live smoke | Lean JSON-backed Anki template management for builtin sync, list/default, create/update/delete, bulk import, export, and one-time read-only migration from old Rust/Tauri `mistakes.db/custom_anki_templates` plus `settings.default_template_id`. Frontend template surfaces route through the native facade and Wails bridge, and `npm run go:smoke:live-wails-templates` proves legacy SQLite migration, builtin/list/create/update/default/import/export, and `templates-go.json` persistence in a real Wails/WebView2 shell. This is not APKG export/import, Anki add-card parity, or full designer backend parity. |
| Chat | `ChatService` | Partial core | Session/group/tag/message/block store, archive/restore/permanent-delete semantics for the archive governance path, group delete/restore compatibility, branch sessions, Wails event bridge, text-only OpenAI-compatible streaming for send/continue/retry/edit-resend, local fallback streams, cancellable provider requests, approvals, ask-user, block upserts. |
| Skills | `SkillService` | Working slice | Go/Wails filesystem service for `skill_list_directories`, `skill_read_file`, `skill_create`, `skill_update`, and `skill_delete`; supports global/app-data/project skill roots, native `getAppDataDir` project path resolution, generated Wails bindings, frontend native facade routing, stricter backend guards for direct `SKILL.md` files, writable roots, root-delete prevention, symlink/reparse-point rejection, ASCII skill IDs, and 512 KiB content limits, plus a live Wails/WebView2 smoke that drives the public frontend Skill API facade through create/read/update/list/delete and read-after-delete rejection. |
| VFS | `VfsService` | Foundation+refs+attachments+files+mindmaps+compact-index+local-search+legacy-alias+media-status+page-image/blob-read+pdf-text/page-count/page-text/text-preview+resource-sync+ocr-clear+progress-events+pdf-ocr-status-semantics+live-textbook-smoke slice | Native hybrid resource index with visible file storage, create/reuse, get, path lookup, exists, ref counts, hash update, context refs, ref resolve, attachment upload/content, file upload/get/content/delete/bookmarks/metadata, mindmap CRUD/content/favorite/version compatibility, file-like content reads for Learning Hub previews, legacy URI alias resolution, path-cache compatibility, compact index status/units, local text/metadata search, file listing, lightweight PDF/image processing status/control compatibility, PDF/page/index progress events through the Wails EventBus plus legacy event names, PDF page image reads from existing preview data, lightweight PDF text-layer extraction/page-count/page-text compatibility for searchable/structured PDFs, generated text-layer SVG page previews for searchable PDFs, Windows PDFium raster preview plus package smoke, blob/base64 reads from indexed visible resource files, source-stable resource sync compatibility for note/exam/textbook refs including `previewJson`/`ocrPagesJson`/`extractedText`/`ocrText` metadata, legacy snake_case PDF/file metadata aliases, OCR info/clear compatibility that removes OCR metadata while preserving extracted text, injected OCR-runner metadata persistence tests, PDF readiness semantics that keep extracted text, real OCR, and raster image modes distinct, and a live Wails/WebView2 smoke proving PDF textbook import/progress/resource/file/content/status/batch/index routing over Go hybrid VFS with extracted text evidence. |

Current generated Wails bindings:

- 14 services.
- 336 methods.
- 223 models.

Current native inventory:

- 1534 scanned files.
- 931 native references.
- 837 invokes.
- 606 unique invokes.
- 66 listens.
- 47 unique listens.
- 28 emits.
- 24 unique emits.

Current native command triage:

- 606 unique commands.
- `merge`: 328.
- `replace`: 80.
- `defer`: 172.
- `delete`: 26.

Current Rust retirement map:

- 328 merged commands.
- 326 merged commands with Wails bridge routes.
- 0 merged commands still registered in Rust.
- 35 merged Rust definitions.
- 34 Rust retirement candidates.
- 0 direct Tauri blocked merged commands.
- 0 blocker edges.
- 0 blocker files.
- 71 replace commands still registered in Rust.

Verification policy update:

- The rewrite target is Go/Wails, not Tauri/Rust. For Go implementation, frontend facade, hybrid VFS, and Wails parity slices, prefer `go test`, `npm run typecheck`, native triage/inventory, and runtime smoke checks; do not use `cargo check` as the default proof.
- Use Rust checks only as a transitional guard when a slice directly edits/deletes `src-tauri` command registrations or Rust wrappers and a structural reference check is not enough. The success metric is shrinking and eventually quarantining/removing `src-tauri`, not keeping old Rust warning debt healthy.

## Completed Go/Wails Template Service Slice

Replaced the deferred Wails template bridge with a real lean Go service. This follows the rewrite rule of rebuilding by product workflow rather than copying the old Rust DB implementation.

What changed:

- Added `desktop-go/internal/templates` with a JSON-backed `TemplateService` stored as `templates-go.json`.
- Implemented builtin template sync, list/default, create/update/delete, default selection, bulk import, and export.
- Seeded six lean builtin templates (`极简卡片`, `编程代码卡片`, `填空题卡片`, `选择题卡片`, `语言学习卡片`, `法律条文卡片`) instead of copying the full old Rust template corpus.
- Preserved frontend-compatible template JSON fields: `fields` arrays and `field_extraction_rules` objects are normalized on write; legacy `fields_json` and `field_extraction_rules_json` import payloads are accepted but not stored in the old shape.
- Added `expected_version` conflict checks and version bumping for updates.
- Added a lightweight Go boundary defang for script blocks, event-handler attributes, `javascript:` URLs, and high-risk CSS tokens. This complements the existing frontend sanitizer; it is not a full HTML sanitization library.
- Added `desktop-go/internal/bindings/template_service.go`, wired `Templates *templates.Service` into `app.New`, and registered `bindings.NewTemplateService(applicationState)` in Wails.
- Generated Wails bindings for `TemplateService`.
- Updated `src/runtime/wailsBridge.ts` so `import_builtin_templates`, `get_all_custom_templates`, `get_default_template_id`, `create_custom_template`, `update_custom_template`, `delete_custom_template`, `set_default_template`, `import_custom_templates_bulk`, and `export_template` call the real Go binding instead of returning placeholders or throwing the old deferred error.
- Updated `src/components/TemplateManager.tsx`, `src/components/TemplateManagementPage.tsx`, and `src/utils/forceImportTemplates.ts` to import `invoke` from `@/runtime/native` instead of `@tauri-apps/api/core`.
- Updated `scripts/native-triage.mjs` so the nine implemented template commands are tracked as `study-data` / `merge`.
- Updated `tests/vitest/runtime/ankiTemplatesNativeFacade.source.test.ts` to require real TemplateService routing and generated bindings.

GitNexus / impact notes:

- `npx gitnexus impact --repo "Deep Student" --direction upstream New`: LOW, direct caller `desktop-go/cmd/deep-student-go/main.go`.
- `npx gitnexus impact --repo "Deep Student" --direction upstream import_builtin_templates`: LOW, 0 impacted upstream callers.
- `npx gitnexus impact --repo "Deep Student" --direction upstream export_template`: LOW, 0 impacted upstream callers.
- `npx gitnexus impact --repo "Deep Student" --direction upstream TemplateManager`: LOW, broader frontend import fanout noted; this slice preserved public command names and object parameter shapes.
- `npx gitnexus impact --repo "Deep Student" --direction upstream TemplateManagementPage`: LOW, 0 impacted upstream callers.
- `npx gitnexus impact --repo "Deep Student" --direction upstream TaskDashboardPage`: LOW, 0 impacted upstream callers.
- `forceImportTemplates`, `implementedCommandOverrides`, and `native-triage` were not indexed by GitNexus.

Metrics after this slice:

- Generated Wails bindings: 14 services / 333 methods / 219 models.
- Native inventory: 1534 scanned files / 920 native references / 826 invokes / 606 unique invokes / 66 listens / 47 unique listens / 28 emits / 24 unique emits.
- Native triage: 606 unique commands; `merge` 328, `replace` 80, `defer` 172, `delete` 26.
- Rust retirement map: 328 merged commands, 323 with Wails bridge routes, 15 merged Rust registrations, 39 merged Rust definitions, 35 retirement candidates, 0 direct Tauri blockers, 0 blocker edges, 0 blocker files, 71 replace commands still registered in Rust.

Verification:

- `npm run go:bindings`: pass, generated 14 services / 333 methods / 219 models.
- `go test ./internal/templates ./internal/bindings -count=1` from `desktop-go`: pass.
- `go test ./... -count=1` from `desktop-go`: pass.
- `npm run go:check`: pass.
- `npm run test -- tests/vitest/runtime/ankiTemplatesNativeFacade.source.test.ts`: pass, 1 file / 2 tests.
- `npm run typecheck -- --pretty false`: pass.
- `npm run native:inventory -- --summary`: pass, metrics above.
- `npm run native:triage`: pass, regenerated `docs/generated/native-command-triage.{json,md}`.
- `node scripts/rust-retirement-map.mjs`: pass, regenerated `docs/generated/rust-retirement-map.{json,md}`.
- `git diff --check -- <touched template/go/runtime/docs files>`: pass.
- Full `git diff --check -- desktop-go src tests docs scripts package.json package-lock.json` is blocked by pre-existing unrelated trailing whitespace in `src/hooks/useSystemSettings.ts`.
- `cargo check --manifest-path src-tauri/Cargo.toml`: intentionally not used; this slice did not edit old Rust/Tauri code.

Known gaps / do not count as complete:

- Migration from the old Rust SQLite/custom template table is now covered by the Completed Legacy SQLite Template Migration Slice below.
- It does not implement APKG export/import or direct Anki add-card parity.
- The Go-side defang is a migration safety boundary, not a complete sanitizer replacement.
- Template designer/editor business logic remains frontend-owned; only persistence/native command parity moved to Go.
- Old Rust template commands remain registered/defined and are now counted as merged Rust retirement candidates; delete them only after data migration and live template UI smoke are added.

Next queue:

- Live Wails/WebView2 template smoke is now covered by the Completed Live Wails Template Service Smoke Slice below.
- Old Rust template DB migration strategy is now implemented as one-time read-only SQLite import into `templates-go.json` in the Completed Legacy SQLite Template Migration Slice below.
- Continue direct Tauri invoke cleanup for remaining product paths that now have Go/Wails routes.

## Completed Live Wails Template Service Smoke Slice

Closed the runtime proof gap for the Go/Wails template service. The smoke launches the real Wails/WebView2 shell with embedded React assets, then drives the compatibility command names through `src/runtime/native.ts` and `src/runtime/wailsBridge.ts` rather than importing generated bindings directly.

What changed:

- Added `npm run go:smoke:live-wails-templates`, implemented as `node scripts/go-live-wails-smoke.mjs --templates`.
- Extended `npm run go:smoke:live-wails-core` to include `--templates` alongside MCP stdio, Skills, and VFS.
- Added `runTemplateWailsSmoke` in `src/main.tsx` to exercise `import_builtin_templates`, `get_all_custom_templates`, `create_custom_template`, `set_default_template`, `get_default_template_id`, `update_custom_template`, `export_template`, and `import_custom_templates_bulk` through the frontend native facade.
- The smoke verifies one-time old `mistakes.db/custom_anki_templates` migration, six lean builtins, custom create/update/default/export, legacy `fields_json` / `field_extraction_rules_json` bulk import normalization, and `is_built_in: false` for custom imports.
- The Node runner now writes a temporary old `mistakes.db` fixture before Wails launch, then asserts `templates-go.json` exists in the temporary Wails data dir, contains the migrated, created, and imported templates, persists migration metadata, persists the updated description, normalizes away `fields_json`, and persists `defaultTemplateId`.
- Extended `tests/vitest/system/goLiveWailsSmoke.source.test.ts` so source contracts require the template smoke script, frontend hook, route evidence, and core-smoke inclusion.

GitNexus / impact notes:

- `npx gitnexus impact --repo "Deep Student" --direction upstream runVfsWailsSmoke`: target not found.
- `npx gitnexus impact --repo "Deep Student" --direction upstream runSkillWailsSmoke`: target not found.
- `npx gitnexus impact --repo "Deep Student" --direction upstream installGoWailsSmokeHook`: target not found.
- `npx gitnexus impact --repo "Deep Student" --direction upstream goLiveWailsSmoke`: target not found.
- These smoke helper symbols are not indexed; coverage is from source contracts, TypeScript typecheck, and the live Wails/WebView2 run.

Metrics after this slice:

- Native inventory: 1534 scanned files / 931 native references / 837 invokes / 606 unique invokes / 66 listens / 47 unique listens / 28 emits / 24 unique emits.
- Native triage: 606 unique commands; `merge` 328, `replace` 80, `defer` 172, `delete` 26.
- Rust retirement map: 328 merged commands, 323 with Wails bridge routes, 15 merged Rust registrations, 39 merged Rust definitions, 35 retirement candidates, 0 direct Tauri blockers, 0 blocker edges, 0 blocker files, 71 replace commands still registered in Rust.

Verification:

- `npm run test -- tests/vitest/system/goLiveWailsSmoke.source.test.ts tests/vitest/runtime/ankiTemplatesNativeFacade.source.test.ts`: pass, 2 files / 14 tests.
- `npm run typecheck -- --pretty false`: pass.
- `npm run go:smoke:live-wails-templates`: pass. Evidence included Wails runtime `isWails: true`, `templates.ok: true`, `builtinCount: 6`, migrated legacy template `Legacy Wails Smoke Template`, `legacyMigration.imported: 1`, created template update/export parity, bulk import normalization, route evidence for all template commands, and persisted `templates-go.json`.
- `npm run native:inventory -- --summary`: pass, metrics above.
- `npm run native:triage`: pass, regenerated `docs/generated/native-command-triage.{json,md}`.
- `node scripts/rust-retirement-map.mjs`: pass, regenerated `docs/generated/rust-retirement-map.{json,md}`.
- `git diff --check -- package.json src/main.tsx scripts/go-live-wails-smoke.mjs tests/vitest/system/goLiveWailsSmoke.source.test.ts docs/GO_REWRITE_PROGRESS.md docs/generated`: pass.
- `cargo check --manifest-path src-tauri/Cargo.toml`: intentionally not used; this slice added Go/Wails runtime proof and did not edit old Rust command code.

Known gaps / do not count as complete:

- This smoke is a runtime command/facade proof, not a visual/manual test of the full template management UI.
- It uses a temporary old SQLite fixture and proves the migration path in a real Wails launch; it does not prove every historical user data-space layout or APKG/card insertion parity.
- It does not cover APKG export/import or direct Anki card insertion.

Next queue:

- Retire the old Rust template commands in a deletion batch after one more source-level command registration review.
- Keep extending live Wails smoke coverage to the next high-value replacement path, preferably provider-backed Chat or old-data migration.

## Completed Legacy SQLite Template Migration Slice

Implemented the first concrete old-data migration path for the Go/Wails rewrite: old Rust/Tauri Anki templates now migrate from `mistakes.db` into the lean Go `templates-go.json` store without keeping SQLite as the active storage model.

What changed:

- Added read-only migration from old `mistakes.db/custom_anki_templates` into Go `TemplateService`.
- Migrated old `settings.default_template_id` when the referenced migrated template exists and is active.
- Skips old built-in template rows by default so the lean Go built-in set remains the source of truth.
- Preserves existing Go `templates-go.json` entries: old rows with duplicate ID or duplicate name are skipped rather than overwriting newer Go data.
- Persists `legacyMigration` metadata in `templates-go.json` with source path, size, mtime, imported/skipped/failed counts, migrated time, migrated default ID, and last error when startup import fails.
- Added old data root discovery for production app startup through `templates.NewServiceWithLegacyRoots(dataDir, LegacyDataDirCandidates(dataDir))`.
  - Explicit `DEEP_STUDENT_DATA_DIR` stays isolated and does not scan user directories.
  - Normal startup checks old Tauri roots such as `com.deepstudent.app`, `DeepStudent`, and `slots/slotA|slotB|*/mistakes.db`.
- Added `modernc.org/sqlite` as the pure-Go SQLite reader for this one-time import path.
- Extended the live Wails template smoke to create a temporary old `mistakes.db` fixture before launch, then prove the migrated template is visible through the frontend native facade and persisted in `templates-go.json`.
- Extended source contracts so the migration hook, path isolation, live fixture, and `legacyMigration` smoke evidence remain guarded.

GitNexus / impact notes:

- `npx gitnexus impact --repo "Deep Student" --direction upstream TemplateService`: LOW; frontend import fanout remains unchanged because command names and payload shapes are preserved.
- `npx gitnexus impact --repo "Deep Student" --direction upstream ResolveDataDir`: LOW; direct caller `desktop-go/internal/app/app.go:New`.
- `npx gitnexus impact --repo "Deep Student" --direction upstream New`: LOW; direct caller `desktop-go/cmd/deep-student-go/main.go:main`.
- `ImportCustomTemplatesBulk`, `GetAllCustomTemplates`, `templates.NewService`, and `Function:desktop-go/internal/templates/service.go:NewService` were not found in the current GitNexus index.
- `npx gitnexus detect_changes --repo "Deep Student"` remains unavailable in this GitNexus CLI build (`unknown command 'detect_changes'`), so scope was checked with tests, native inventory/triage, rust-retirement-map, and focused diff checks.

Metrics after this slice:

- Generated Wails bindings: unchanged, 14 services / 333 methods / 219 models.
- Native inventory: 1534 scanned files / 931 native references / 837 invokes / 606 unique invokes / 66 listens / 47 unique listens / 28 emits / 24 unique emits.
- Native triage: 606 unique commands; `merge` 328, `replace` 80, `defer` 172, `delete` 26.
- Rust retirement map: 328 merged commands, 323 with Wails bridge routes, 15 merged Rust registrations, 39 merged Rust definitions, 35 retirement candidates, 0 direct Tauri blockers, 0 blocker edges, 0 blocker files, 71 replace commands still registered in Rust.

Verification:

- `go test ./internal/templates ./internal/app -count=1` from `desktop-go`: pass.
- `go test ./... -count=1` from `desktop-go`: pass.
- `npm run go:check`: pass before the final live-smoke extension; later full `go test ./... -count=1` passed after app path-discovery changes.
- `npm run test -- tests/vitest/runtime/ankiTemplatesNativeFacade.source.test.ts tests/vitest/system/goLiveWailsSmoke.source.test.ts`: pass, 2 files / 15 tests.
- `npm run go:smoke:live-wails-templates`: pass. Evidence included real Wails/WebView2 `isWails: true`, migrated legacy template `Legacy Wails Smoke Template`, normalized migrated fields `[Front, Back]`, `legacyMigration.imported: 1`, builtin count 6, create/update/default/export parity, custom bulk import normalization, route evidence for all template commands, and persisted `templates-go.json`.
- `npm run native:inventory -- --summary`: pass, metrics above.
- `npm run native:triage`: pass, regenerated `docs/generated/native-command-triage.{json,md}`.
- `node scripts/rust-retirement-map.mjs`: pass, regenerated `docs/generated/rust-retirement-map.{json,md}` with metrics above.
- `cargo check --manifest-path src-tauri/Cargo.toml`: intentionally not used; this slice added Go migration and Wails smoke coverage and did not edit old Rust/Tauri command code.

Known gaps / do not count as complete:

- This migrates template records only. It does not migrate old Anki generated cards, APKG artifacts, document task rows, qbank records, or broader `mistakes.db` content.
- It skips old built-in template rows intentionally; users keep the lean Go built-ins. If a user modified an old built-in row in place, that row will not be imported unless a future explicit recovery/import path is added.
- It does not implement APKG export/import or direct Anki add-card parity.
- It does not delete old Rust template commands yet.

Next queue:

- Run a focused old Rust template command retirement review and delete the now-Go-backed template wrappers/registrations if no non-template Rust caller still needs them.
- Continue broader old-data migration planning for the next product-owned dataset, likely Anki document tasks/cards or chat/provider state.

## Completed Unused Tauri Invoke Import Cleanup Slice

Removed unused direct `@tauri-apps/api/core` `invoke` imports from files that no longer make direct invoke calls. This is not command parity work; it is surface-area reduction so future scans focus on real native blockers instead of stale imports.

What changed:

- Removed unused `invoke` imports from `src/components/BatchOperationToolbar/index.tsx`.
- Removed unused `invoke` imports from `src/components/shared/UnifiedDragDropZone.tsx`.
- Removed unused `invoke` imports from `src/features/chat/adapters/contextHelper.ts`.
- Removed unused `invoke` imports from `src/features/chat/components/input-bar/InputBarV2.tsx`.
- Removed unused `invoke` imports from `src/features/chat/plugins/blocks/subagentEmbed.tsx`.
- Added `tests/vitest/runtime/unusedTauriInvokeImports.source.test.ts` to keep these files from regressing to unused direct Tauri core invoke imports.

GitNexus / impact notes:

- `npx gitnexus impact --repo "Deep Student" --direction upstream BatchOperationToolbar`: LOW, 0 impacted upstream callers.
- `npx gitnexus impact --repo "Deep Student" --direction upstream UnifiedDragDropZone`: LOW, 0 impacted upstream callers.
- `npx gitnexus impact --repo "Deep Student" --direction upstream InputBarV2`: target not found.
- `npx gitnexus impact --repo "Deep Student" --direction upstream SubagentEmbedBlockComponent`: target not found.
- `npx gitnexus impact --repo "Deep Student" --direction upstream contextHelper`: target not found.
- The edit is import-only; no function bodies, command names, event names, or runtime semantics changed.

Metrics after this slice:

- Native inventory: 1531 scanned files / 920 native references / 826 invokes / 606 unique invokes / 66 listens / 47 unique listens / 28 emits / 24 unique emits.
- Inventory command counts are unchanged because this slice removed unused imports, not command calls.
- Native triage: 606 unique commands; `merge` 319, `replace` 80, `defer` 181, `delete` 26.
- Rust retirement map: 319 merged commands, 314 with Wails bridge routes, 6 merged Rust registrations, 30 merged Rust definitions, 26 retirement candidates, 0 direct Tauri blockers, 0 blocker edges, 0 blocker files, 71 replace commands still registered in Rust.

Verification:

- `rg -n "import \{ invoke \} from '@tauri-apps/api/core'" src/components/BatchOperationToolbar/index.tsx src/components/shared/UnifiedDragDropZone.tsx src/features/chat/adapters/contextHelper.ts src/features/chat/components/input-bar/InputBarV2.tsx src/features/chat/plugins/blocks/subagentEmbed.tsx tests/vitest/runtime/unusedTauriInvokeImports.source.test.ts`: only the new source test assertion string remains.
- `npm run test -- tests/vitest/runtime/unusedTauriInvokeImports.source.test.ts`: pass, 1 file / 1 test.
- `npm run typecheck -- --pretty false`: pass.
- `npm run native:inventory -- --summary`: pass, metrics above.
- `npm run native:triage`: pass, regenerated `docs/generated/native-command-triage.{json,md}`.
- `node scripts/rust-retirement-map.mjs`: pass, regenerated `docs/generated/rust-retirement-map.{json,md}` with metrics above.
- `cargo check --manifest-path src-tauri/Cargo.toml`: intentionally not used; this slice did not edit old Rust/Tauri code.

Known gaps / do not count as complete:

- This does not migrate dynamic invoke wrappers, event listeners, `convertFileSrc`, WebView APIs, clipboard plugins, or commands without Go/Wails parity.
- `forceImportTemplates.ts` was intentionally not changed in this historical slice because `import_builtin_templates` was deferred at that time. This gap is superseded by the Completed Go/Wails Template Service Slice above.

Next queue:

- Continue with real direct-invoke cleanup only where the command has an active Go/Wails implementation, not just a deferred placeholder.
- Treat remaining `@tauri-apps/api/core` imports with no literal commands as separate categories: asset URL compatibility, dynamic wrappers, tests, or unused imports.

## Completed AnkiConnect Status Native Facade Cleanup Slice

Closed one more frontend business-level direct-call gap for `check_anki_connect_status`. The chat Anki card block refresh path now uses the shared AnkiConnect client facade instead of calling a legacy command string directly from the component.

What changed:

- Updated `src/features/chat/plugins/blocks/ankiCardsBlock.tsx` to import `invoke` from `src/runtime/native.ts` instead of `@tauri-apps/api/core`.
- Added `ankiConnectClient` usage in the same block and changed the refresh-status path from `invoke<boolean>('check_anki_connect_status')` to `ankiConnectClient.check()`.
- Kept the remaining block commands on the runtime native `invoke` path; `get_document_tasks`, `get_document_cards`, and `chat_v2_update_block_tool_output` are not part of this small cleanup slice.
- Extended `tests/vitest/anki/ankiConnectNativeFacade.source.test.ts` so the source contract verifies `ankiCardsBlock.tsx` uses `ankiConnectClient.check()` and no longer directly invokes `check_anki_connect_status`.

GitNexus / impact notes:

- `npx gitnexus impact --repo "Deep Student" --direction upstream check_anki_connect_status`: LOW, 0 impacted upstream callers.
- This slice intentionally did not delete old Rust. `check_anki_connect_status` remains listed as a replace command still registered in Rust, and broader AnkiConnect operations still need a product-level Go replacement decision.

Metrics after this slice:

- Native inventory: 1531 scanned files / 920 native references / 826 invokes / 606 unique invokes / 66 listens / 47 unique listens / 28 emits / 24 unique emits.
- This is a reduction from the previous 921 native references / 827 invokes because the Anki card block status refresh moved behind the shared AnkiConnect facade.
- Native triage: 606 unique commands; `merge` 319, `replace` 80, `defer` 181, `delete` 26.
- Rust retirement map: 319 merged commands, 314 with Wails bridge routes, 6 merged Rust registrations, 30 merged Rust definitions, 26 retirement candidates, 0 direct Tauri blockers, 0 blocker edges, 0 blocker files, 72 replace commands still registered in Rust.

Verification:

- `rg -n "check_anki_connect_status" src tests/vitest docs/generated -g "*.ts" -g "*.tsx" -g "*.md" -g "*.json"`: the command string remains in the AnkiConnect facade, runtime bridge, tests, and generated inventory docs; `ankiCardsBlock.tsx` no longer contains it.
- `npm run test -- tests/vitest/anki/ankiConnectNativeFacade.source.test.ts tests/vitest/runtime/wailsBridgeSettingsSystem.behavior.test.ts`: pass, 2 files / 3 tests.
- `npm run typecheck -- --pretty false`: pass.
- `go test ./internal/anki ./internal/bindings -count=1` from `desktop-go`: pass.
- `npm run native:inventory -- --summary`: pass, metrics above.
- `npm run native:triage`: pass, regenerated `docs/generated/native-command-triage.{json,md}`.
- `node scripts/rust-retirement-map.mjs`: pass, regenerated `docs/generated/rust-retirement-map.{json,md}`.
- `cargo check --manifest-path src-tauri/Cargo.toml`: intentionally not used; this slice did not edit old Rust/Tauri code.

Known gaps / do not count as complete:

- This only cleans the status refresh path. It does not implement AnkiConnect add-card/import/deck-write parity in Go.
- `create_anki_deck`, `import_anki_package`, and `add_cards_to_anki_connect` remain outside this slice.
- `check_anki_connect_status` remains a replace-class old Rust registration until the broader AnkiConnect product path is redesigned or removed.

Next queue:

- Continue direct native-call cleanup on product paths that already have Go/Wails routes, prioritizing small business components over broad old Rust helper deletion.
- For AnkiConnect, decide whether deck/card write operations are in the lean release or deferred; implement them as Go product APIs instead of copying the old Rust command shape.

## Completed Image Base64 Native Facade Cleanup Slice

Closed the remaining frontend direct-call gap for `get_image_as_base64` after the earlier Settings/System/Image Utility Rust Command Retirement slice. The old Rust command wrapper was already removed; this slice moves the last business-level reads behind the shared native facade so Wails/Go owns the active route and Tauri fallback stays centralized in `src/runtime/native.ts`.

What changed:

- Added `getImageAsBase64(relativePath)` to `src/runtime/native.ts` and exposed it as `native.files.getImageAsBase64`.
- Kept the Wails route in `src/runtime/wailsBridge.ts`: `get_image_as_base64` dispatches to Go `NotesService.GetImageAsBase64`.
- Updated `src/utils/configApi.ts` to call the native facade instead of trying camelCase invoke, snake_case invoke, then a dynamic `@tauri-apps/api/core` `convertFileSrc` fallback.
- Updated `src/components/crepe/features/imageUpload.ts` so the image proxy path imports `getImageAsBase64` from the native facade instead of calling `invoke('get_image_as_base64')` directly. Its separate `notes_save_asset` call remains on the normal native invoke path.
- Added source-contract coverage that business files do not directly invoke `get_image_as_base64` and that the facade/Wails bridge route remains present.

GitNexus / impact notes:

- `npx gitnexus impact --repo "Deep Student" --direction upstream get_image_as_base64`: LOW, 0 impacted callers for the indexed old Rust `FileManager::get_image_as_base64` helper.
- `npx gitnexus impact --repo "Deep Student" --direction upstream get_file_size`: LOW, 0 impacted callers for the indexed old Rust helper.
- `npx gitnexus impact --repo "Deep Student" --direction upstream ensure_debug_log_dir`: LOW, 0 impacted callers for the indexed old Rust helper.
- `npx gitnexus impact --repo "Deep Student" --direction upstream get_app_data_dir`: CRITICAL, 144 impacted; this is a broad old Rust helper and was intentionally not deleted.
- `npx gitnexus impact --repo "Deep Student" --direction upstream copy_file`: CRITICAL, 8 impacted; this is still used by old Rust internal import/data-governance paths and was intentionally not deleted.
- The actual edit was frontend facade cleanup, not Rust helper deletion. The old Rust command wrappers for `get_image_as_base64`, `get_app_data_dir`, and `ensure_debug_log_dir` were already retired in the earlier Settings/System/Image Utility Rust Command Retirement slice.

Metrics after this slice:

- Native inventory: 1531 scanned files / 921 native references / 827 invokes / 606 unique invokes / 66 listens / 47 unique listens / 28 emits / 24 unique emits.
- This is a reduction from the previous 923 native references / 829 invokes because two business-level `get_image_as_base64` direct calls moved behind the native facade.
- Native triage: 606 unique commands; `merge` 319, `replace` 81, `defer` 180, `delete` 26.
- Rust retirement map: 319 merged commands, 314 with Wails bridge routes, 6 merged Rust registrations, 30 merged Rust definitions, 26 retirement candidates, 0 direct Tauri blockers, 0 blocker edges, 0 blocker files, 72 replace commands still registered in Rust.

Verification:

- `rg -n "invoke<[^>]+>\\('get_image_as_base64'|invoke\\('get_image_as_base64'|get_image_as_base64" src tests/vitest -g "*.ts" -g "*.tsx"`: only `src/runtime/native.ts` and `src/runtime/wailsBridge.ts` contain the command string.
- `npm run test -- tests/vitest/runtime/nativeAppDataDirWailsContract.source.test.ts tests/vitest/runtime/wailsBridgeSettingsSystem.behavior.test.ts`: pass, 2 files / 5 tests.
- `npm run typecheck -- --pretty false`: pass.
- `go test ./internal/notes ./internal/bindings ./internal/system ./internal/files -count=1` from `desktop-go`: pass.
- `npm run native:inventory -- --summary`: pass, metrics above.
- `npm run native:triage`: pass, regenerated `docs/generated/native-command-triage.{json,md}`.
- `node scripts/rust-retirement-map.mjs`: pass, regenerated `docs/generated/rust-retirement-map.{json,md}`.
- `cargo check --manifest-path src-tauri/Cargo.toml`: intentionally not used; this slice did not edit old Rust/Tauri code and the relevant active route is Go/Wails.

Known gaps / do not count as complete:

- This cleans one image-base64 facade path. It does not remove the broader direct `@tauri-apps/api` surface for dialogs, paths, drag/drop, window APIs, asset URLs, or un-migrated product commands.
- `convertFileSrc` still exists in `imageUpload.ts` for asset URL display/proxy behavior; this slice only moved `get_image_as_base64` data reads.
- The generated retirement map still lists `get_app_data_dir`, `copy_file`, `get_file_size`, `get_image_as_base64`, and `ensure_debug_log_dir` helper symbols because they are Rust helper definitions, not necessarily live Tauri command wrappers. Do not delete CRITICAL helper symbols until their old Rust internal callers are retired or replaced.

Next queue:

- Continue frontend native facade cleanup for direct `@tauri-apps/api/core`, path, dialog, fs, event, and window imports, prioritizing product paths already backed by Wails bridge routes.
- Continue low-risk Rust retirement only where the remaining symbol is a command wrapper or module boundary, not a still-used helper.

## Completed Live Wails PDF/Textbook Hybrid VFS Smoke Slice

Closed the live runtime proof gap for PDF textbook import over the Go hybrid VFS. The smoke now launches the real Wails/WebView2 shell, creates a temp PDF fixture from the embedded React runtime, imports it through the public `textbookDstuAdapter.addTextbooks` path, receives `textbook-import-progress`, then verifies Go VFS resource/file/content/PDF status/batch/index routes over the Wails bridge.

What changed:

- Added `npm run go:smoke:live-wails-vfs`, implemented as `node scripts/go-live-wails-smoke.mjs --vfs`.
- Added `npm run go:smoke:live-wails-core`, implemented as `node scripts/go-live-wails-smoke.mjs --mcp --skills --vfs`, so milestone proof can cover base shell + MCP stdio + Skills + VFS in one Wails/WebView2 launch instead of three repeated build/launch cycles.
- Extended the live smoke hook in `src/main.tsx` with `runVfsWailsSmoke`.
- The frontend proof creates a unique PDF under the temporary `appDataDir/smoke-fixtures`, calls the public textbook adapter, and asserts the imported textbook exposes a Go hybrid VFS `fileId`, `resourceId`, and `resourceHash`.
- The proof verifies `vfs_get_resource`, `vfs_get_resource_path`, `vfs_get_file`, `vfs_get_file_content`, `vfs_get_pdf_processing_status`, `vfs_get_batch_pdf_processing_status`, `getAllIndexStatus`, and `textbook-import-progress -> Wails EventBus` route evidence.
- Tightened the text proof so raw PDF byte retrieval and extracted text-layer evidence are separate fields: `rawContentContainsSentinel` proves retrievable imported bytes, while `extractedTextContainsSentinel` must come from `file.extractedText` or resource metadata text and cannot reuse decoded raw bytes.
- Strengthened batch PDF status proof: batch status must be terminal, report page count 1, and include `text` in `readyModes`.
- Added source-contract coverage for the VFS live smoke, the new core aggregate script, extracted-text source separation, and batch status assertions.

GitNexus / impact notes:

- `npx gitnexus impact --repo "Deep Student" --direction upstream runVfsWailsSmoke`: target not found; this smoke-hook helper is not indexed yet.
- `npx gitnexus impact --repo "Deep Student" --direction upstream assertVfsSmokeResult`: target not found; this script-local helper is not indexed yet.
- `npx gitnexus impact --repo "Deep Student" --direction upstream goLiveWailsSmoke`: target not found; the source-contract test name is not indexed yet.
- Earlier impact for the public `addTextbooks` path was LOW, with direct Learning Hub callers; this slice kept the product adapter API and only added live smoke proof around it.
- `npx gitnexus detect_changes --repo "Deep Student"` was attempted but the installed GitNexus CLI reports `unknown command 'detect_changes'`; current scope was checked with `git diff`, native inventory/triage, and the Rust retirement map instead.

Metrics after this slice:

- Native inventory: 1531 scanned files / 923 native references / 829 invokes / 606 unique invokes / 66 listens / 47 unique listens / 28 emits / 24 unique emits.
- Native triage: 606 unique commands; `merge` 319, `replace` 81, `defer` 180, `delete` 26.
- Rust retirement map: 319 merged commands, 314 with Wails bridge routes, 6 merged Rust registrations, 30 merged Rust definitions, 26 retirement candidates, 0 direct Tauri blockers, 0 blocker edges, 0 blocker files, 72 replace commands still registered in Rust.

Verification:

- `node --check scripts/go-live-wails-smoke.mjs`: pass.
- `npm run test -- tests/vitest/system/goLiveWailsSmoke.source.test.ts`: pass, 10 tests.
- `npm run typecheck -- --pretty false`: pass.
- `go test ./internal/vfs ./internal/dstu ./internal/bindings -run "Test(UploadFileExtractsPdfTextLayer|UploadFileDetectsPdfPageCount|UploadFileBuildsPdfTextLayerPageInfo|StartPdfProcessingExtractsTextForExistingPdfResource|DstuCreateFileImageAndTextbookUseHybridVfs|AddTextbooksImportsLocalFilesIntoHybridVfs|AddTextbooksPromotesExistingFileResource|NewWiresDstuTextbookImportProgressToEventBus)" -count=1` from `desktop-go`: pass.
- `npm run go:smoke:live-wails-core`: pass. The run includes `npm run build`, frontend dist sync, static frontend embed smoke, Go exe build, live Wails/WebView2 launch, CDP attach, base settings roundtrip, MCP stdio smoke, Skill CRUD smoke, VFS PDF textbook smoke, strict console/page/request checks, and temp-data cleanup.
- `npm run native:inventory -- --summary`: pass, metrics above.
- `npm run native:triage`: pass, regenerated `docs/generated/native-command-triage.{json,md}`.
- `node scripts/rust-retirement-map.mjs`: pass, regenerated `docs/generated/rust-retirement-map.{json,md}`.
- `cargo check --manifest-path src-tauri/Cargo.toml`: intentionally not used; this slice did not edit old Rust/Tauri delivery code and the proof target is the Go/Wails runtime.

Live smoke proof excerpt:

```json
{
  "isWails": true,
  "mcpStdio": { "ok": true, "toolName": "smoke_echo", "tauriFallbackRejected": true },
  "skills": { "ok": true, "deleteSucceeded": true, "readAfterDeleteRejected": true },
  "vfs": {
    "ok": true,
    "resourceType": "textbook",
    "resourceStorageMode": "external",
    "pdfStage": "completed_with_issues",
    "pdfReadyText": true,
    "pdfPageCount": 1,
    "batchPdfStage": "completed_with_issues",
    "batchPdfReadyText": true,
    "batchPdfPageCount": 1,
    "rawContentContainsSentinel": true,
    "extractedTextContainsSentinel": true,
    "progressStages": ["hashing", "copying", "saving", "done"]
  }
}
```

Known gaps / do not count as complete:

- This proves one live Wails/WebView2 PDF textbook import path for a searchable one-page fixture. It does not prove scanned-PDF OCR, robust complex/font-encoded PDF extraction, full textbook open/search UI parity, restart persistence, old Rust data migration, or installer/package parity.
- `vfs_get_pdf_page_image` remains outside this live smoke because the raster/PDFium preview path depends on package/runtime layout and should keep separate `go:smoke:pdfium` and future UI preview proof.
- `completed_with_issues` is accepted because the fixture proves text readiness while other heavy stages such as OCR/raster can remain intentionally incomplete in the lean path.
- The aggregate `go:smoke:live-wails-core` is the preferred milestone gate, but keep the single-purpose MCP/Skills/Templates/VFS smoke commands for debugging failures quickly.

Next queue:

- Use `go:smoke:live-wails-core` as the higher-throughput runtime gate when touching Wails bridge, MCP stdio, Skills, Templates, or hybrid VFS smoke infrastructure.
- Low-risk Rust retirement queue from read-only subagent review: AnkiConnect metadata probes, Settings/System/File utilities, Todo/Pomodoro, then Notes asset/import-export helpers; defer Anki document internals and VFS ref/path semantics until coverage is stronger.
- If staying on study resources, deepen native raster preview/OCR and rich textbook open/search state over the same hybrid VFS rather than adding another textbook store.

## Completed Live Wails UI Binding Smoke Slice

Closed the static-only shell proof gap by adding and passing a live Windows Wails/WebView2 smoke for the embedded React app and basic frontend-to-Go binding dispatch. This is a shell/runtime proof slice: it proves the Go app can start the real embedded frontend in a real Wails window and perform a low-risk native roundtrip, but it does not prove full product workflow parity.

What changed:

- Strengthened `scripts/go-live-wails-smoke.mjs` so it runs a fresh frontend build, syncs embedded assets, checks the embedded dist contains the live smoke hook, builds a temporary Go exe, launches Wails with a temporary `DEEP_STUDENT_DATA_DIR`, attaches over WebView2 CDP, and rejects Vite dev-server targets/resources.
- Added response-level diagnostics for HTTP failures so opaque browser console messages such as `Failed to load resource` include status, method, and URL.
- Kept HTTP failure handling strict, with only one narrow framework exception: Wails v3 runtime's optional `HEAD http://wails.localhost/wails/custom.js` probe. Wails source marks `custom.js` as server-mode-only/optional, so this is not an app asset gap.
- Strengthened the smoke hook in `src/main.tsx` so it installs only on `?go-wails-smoke=true`, only after Wails flags are present, and reports Wails/runtime markers, React mount state, top-level error-boundary visibility, early runtime errors, temp app data dir, and a `save_setting` / `get_setting` roundtrip.
- Decoupled React rendering from Sentry/native-setting initialization so optional startup telemetry cannot leave the smoke page with an empty `#root`.
- Routed Sentry consent and frontend log reporting through the native facade instead of direct `@tauri-apps/api/core` for this startup path.
- Routed Anki template startup reads through the native facade and, at that time, added deferred Wails/browser fallbacks for read-only template commands, removing a boot-time Tauri invoke error from the Wails path. The deferred Wails part is superseded by the Completed Go/Wails Template Service Slice above.
- Suppressed the startup auto-update network check only for the live smoke URL, avoiding dependency on release-server availability while preserving normal startup and manual update behavior.
- Recorded known Tauri HTTP noise into the smoke early-error buffer instead of hiding it during Go/Wails smoke, so lingering Tauri plugin paths fail the smoke instead of being silently ignored.
- Added/strengthened source-contract tests for the live smoke runner, the frontend smoke hook, the native Anki template facade, and smoke-only startup update suppression.

GitNexus / impact notes:

- `npx gitnexus status`: pass, index up to date at commit `1812ec7`.
- `npx gitnexus impact initSentryIfConfigured --repo "Deep Student" --direction upstream --depth 3 --include-tests`: LOW risk; one direct file-level caller in `src/main.tsx`.
- `npx gitnexus impact useAppUpdater --repo "Deep Student" --direction upstream --depth 3 --include-tests`: LOW risk from the sidecar check; direct callers are `App` and `AboutTab`.
- `npx gitnexus impact loadTemplates --repo "Deep Student" --direction upstream --depth 3 --include-tests`: CRITICAL, 47 impacted. The actual change was intentionally narrow: replace a Tauri-only startup import with the existing native facade and defer only read-style template commands in Wails/browser fallback. The later Go/Wails Template Service Slice removed that Wails deferral for template management.
- New/untracked smoke helper symbols such as `attachPageDiagnostics`, `assertSmokeResult`, `installGoWailsSmokeHook`, and script-local helpers are not currently indexed by GitNexus and returned `Target not found`; coverage for those is from source tests, live smoke, typecheck, and direct review.
- `npx gitnexus detect_changes()` is still unavailable in this installed GitNexus CLI build, so change scope was checked with native inventory, native triage, rust-retirement map, targeted diffs, and tests.

Metrics after this slice:

- Native inventory: 1531 scanned files / 913 native references / 820 invokes / 606 unique invokes / 65 listens / 47 unique listens / 28 emits / 24 unique emits.
- Native triage: 606 unique commands; `merge` 319, `replace` 81, `defer` 180, `delete` 26.
- Rust retirement map: 319 merged commands, 314 with Wails bridge routes, 9 merged Rust registrations, 33 merged Rust definitions, 29 retirement candidates, 0 direct Tauri blockers, 72 replace commands still registered in Rust.
- Wails bindings: unchanged from the previous generated snapshot, 13 services, 324 methods, 217 models.

Verification:

- `node --check scripts/go-live-wails-smoke.mjs`: pass.
- `npm run test -- tests/vitest/system/goLiveWailsSmoke.source.test.ts`: pass, 1 file / 3 tests.
- `npm run test -- tests/vitest/system/goLiveWailsSmoke.source.test.ts tests/vitest/appUpdateNotificationSource.test.ts`: pass, 2 files / 5 tests.
- `npm run test -- tests/vitest/system/goLiveWailsSmoke.source.test.ts tests/vitest/runtime/ankiTemplatesNativeFacade.source.test.ts`: pass, 2 files / 5 tests.
- `npm run typecheck -- --pretty false`: pass.
- `npm run go:smoke:live-wails`: pass; includes `npm run build`, frontend dist sync, static frontend embed smoke, temporary Go exe build, live Wails/WebView2 launch, CDP attach, embedded asset proof, React mount/sentinel proof, Wails flag/runtime proof, temp app data dir proof, settings persistence to `settings-go.json`, console/page/request failure checks, and clean temp-data cleanup.
- `npm run go:check`: pass; full Go tests plus Go app `--smoke` using a temporary data dir.
- `npm run native:inventory -- --summary`: pass, metrics above.
- `npm run native:triage`: pass, regenerated `docs/generated/native-command-triage.{json,md}`.
- `node scripts/rust-retirement-map.mjs`: pass, metrics above.
- `cargo check --manifest-path src-tauri/Cargo.toml`: intentionally not used; this slice did not edit old Rust/Tauri code as the delivery surface.

Known gaps / do not count as complete:

- This proves only the live shell and a basic settings/system binding roundtrip. It does not prove full workflow parity for notes, todo/pomodoro, qbank, learning resources, chat, provider streaming, MCP tool runtime/cache, skills UI flows, OCR, scanned PDFs, textbook open/search, old data migration, or installer packaging.
- The smoke is Windows/WebView2-oriented because it depends on Wails Windows browser args and CDP attach.
- Only `get_app_data_dir`, `save_setting`, and `get_setting` are proven through this live binding hook; other migrated commands still need workflow-specific runtime smokes.
- Wails still performs an optional `/wails/custom.js` server-mode probe; the smoke allows only that exact framework HEAD 404.
- The startup updater suppression is smoke-URL-only. Normal update behavior and manual update checks still rely on the existing updater implementation, including remaining Tauri updater assumptions outside this smoke path.
- Direct Tauri imports and old Rust registrations remain elsewhere; they are retirement targets, not proof that the Go shell is complete.

Next queue:

- Keep `go:smoke:live-wails`, `go:smoke:live-wails-mcp`, and `go:smoke:live-wails-skills` in the packaging/release smoke queue so future Go shell changes cannot regress embedded React boot, basic native binding dispatch, Wails stdio IPC, or the public Skill API facade path.
- Wire settings-backed OCR recognition into the injected Go VFS OCR runner and add scanned-PDF smoke coverage through PDFium preview pages, OCR metadata, and chat/textbook consumption.
- Continue old-data migration/backfill from Rust `files.preview_json` / `ocr_pages_json` / `extracted_text` / `page_count` into Go hybrid VFS metadata.
- Add the next live Wails product smoke for PDF/textbook preview over Go hybrid VFS or a chat provider/tool path.

## Completed Live Wails MCP Stdio UI Smoke Slice

Closed the live UI proof gap for the Go/Wails MCP stdio process proxy. The smoke now launches the real Wails/WebView2 app, invokes `mcp_stdio_start`, `mcp_stdio_send`, and `mcp_stdio_close` from the embedded React frontend, receives per-session Wails message/closed events, performs a content-length MCP initialize/tools/list/tools/call/shutdown roundtrip against a dedicated Go smoke child, and proves active-session close removes the session by asserting a later send is rejected.

What changed:

- Added `npm run go:smoke:live-wails-mcp`, implemented as `node scripts/go-live-wails-smoke.mjs --mcp`.
- Added `--mcp-stdio-smoke-child` to `desktop-go/cmd/deep-student-go/main.go`. The child is a small content-length JSON-RPC MCP server that answers `initialize`, `tools/list`, `tools/call`, and `shutdown`.
- Extended the live Wails smoke hook in `src/main.tsx` with `runMcpStdioWailsSmoke`.
- The frontend smoke proof starts a Go/Wails stdio session, registers `mcp-stdio-{session}-message/error/closed` listeners, sends MCP requests over `mcp_stdio_send`, verifies `dstu-mcp-smoke` and `smoke_echo`, waits for natural child exit/closed event after `shutdown`, then calls `mcp_stdio_close`.
- Added a second active-close proof session: start an MCP child, call `mcp_stdio_close` while it is still active, then assert a later `mcp_stdio_send` is rejected.
- Hardened `scripts/go-live-wails-smoke.mjs` so a pre-existing CDP endpoint cannot be hidden by the CDP-closed check's catch block.
- Kept live smoke HTTP/console failure handling strict while allowing only two narrow Wails framework/runtime cases:
  - optional `HEAD http://wails.localhost/wails/custom.js` 404;
  - the expected `422 POST http://wails.localhost/wails/runtime` produced by the deliberate active-close send-rejection assertion.
- Hardened the smoke hook so closed-event waits clear both timeout and interval on either success or timeout, and invalid JSON messages fail with explicit MCP smoke diagnostics instead of a later timeout.
- Fixed `src/runtime/nativeEvents.ts` Wails payload normalization so only a real Wails event envelope with `name === event` is unwrapped. Business payloads that legitimately contain a top-level `data` field are preserved.
- Removed duplicate Wails/raw payload unwrapping from the MCP live smoke hook and added behavior coverage for Wails envelope vs raw `{ data }` payloads.
- Updated source/behavior contracts around the live smoke runner, Wails bridge stdio routing, and native event payload normalization.

GitNexus / impact notes:

- `npx gitnexus impact TauriStdioClientTransport --repo "Deep Student" --direction upstream --depth 3 --include-tests`: HIGH risk, 10 impacted. Direct caller is `connectServer`; indirect callers include `connectAll`, `connectServerById`, `ensureConnected`, MCP status UI, settings health checks, and tool/prompt/resource fetches. The slice did not edit `TauriStdioClientTransport`; it proved the Go/Wails command/event path through the live smoke hook.
- `npx gitnexus impact assertCDPClosed --repo "Deep Student" --direction upstream --depth 3 --include-tests`: target not found; script-local helper is not indexed.
- `npx gitnexus impact runMcpStdioWailsSmoke --repo "Deep Student" --direction upstream --depth 3 --include-tests`: target not found; smoke-hook helper is not indexed.
- `npx gitnexus impact listen --repo "Deep Student" --direction upstream --depth 3 --include-tests`: LOW but hit `tests/ct/mocks/tauri-event-mock.ts:listen`, not `src/runtime/nativeEvents.ts`; treat this as a GitNexus coverage gap for the native event facade.
- Background review found and this slice fixed the risky Wails `{ data }` unwrap behavior for `nativeEvents`. Injected-runtime event support remains a separate facade gap: injected native commands exist, but `nativeEvents.listen` still has no injected event bridge and returns a no-op outside Wails/Tauri.

Metrics after this slice:

- Native inventory: 1531 scanned files / 920 native references / 827 invokes / 606 unique invokes / 65 listens / 47 unique listens / 28 emits / 24 unique emits.
- Native triage: 606 unique commands; `merge` 319, `replace` 81, `defer` 180, `delete` 26.
- Rust retirement map: 319 merged commands, 314 with Wails bridge routes, 9 merged Rust registrations, 33 merged Rust definitions, 29 retirement candidates, 0 direct Tauri blockers, 72 replace commands still registered in Rust.

Verification:

- `node --check scripts/go-live-wails-smoke.mjs`: pass.
- `npm run test -- tests/vitest/runtime/nativeEventsWailsPayload.behavior.test.ts tests/vitest/runtime/wailsBridgeMcpStdio.behavior.test.ts tests/vitest/runtime/nativeMcpTauriFallback.behavior.test.ts tests/vitest/mcp/mcpStdioWailsContract.source.test.ts tests/vitest/system/goLiveWailsSmoke.source.test.ts`: pass, 5 files / 18 tests.
- `npm run test -- tests/vitest/system/goLiveWailsSmoke.source.test.ts tests/vitest/runtime/nativeEventsWailsPayload.behavior.test.ts tests/vitest/mcp/mcpStdioWailsContract.source.test.ts`: pass, 3 files / 13 tests after the final smoke-script filtering change.
- `npm run typecheck -- --pretty false`: pass.
- `go test ./cmd/deep-student-go ./internal/mcp` from `desktop-go`: pass.
- `go test ./internal/mcp -run "Test(StartStdioSession|SendStdioMessage)" -count=1` from `desktop-go`: pass.
- `npm run go:smoke:live-wails-mcp`: pass. The run includes `npm run build`, frontend dist sync, static embed smoke, Go exe build, live Wails/WebView2 launch, CDP attach, embedded React mount proof, settings roundtrip proof, MCP stdio content-length initialize/tools/list/tools/call/shutdown proof, message/closed Wails event proof, active-close rejection proof, strict console/page/request checks, and temp-data cleanup.
- `npm run go:check`: pass; full Go test suite plus Go app `--smoke` using a temporary data dir.
- `npm run native:inventory -- --summary`: pass, metrics above.
- `npm run native:triage`: pass, regenerated `docs/generated/native-command-triage.{json,md}`.
- `node scripts/rust-retirement-map.mjs`: pass, regenerated `docs/generated/rust-retirement-map.{json,md}` with metrics above.
- `cargo check --manifest-path src-tauri/Cargo.toml`: intentionally not used; this slice did not edit old Rust/Tauri code as the delivery surface.

Live smoke proof excerpt:

```json
{
  "isWails": true,
  "mcpStdio": {
    "browserFallbackRejected": true,
    "activeCloseCommandSucceeded": true,
    "activeCloseSendRejected": true,
    "activeCloseSessionStarted": true,
    "closeCommandSucceeded": true,
    "closedEventReceived": true,
    "commandStarted": true,
    "errorEventReceived": false,
    "framing": "content_length",
    "initializeServerName": "dstu-mcp-smoke",
    "invalidMessageReceived": false,
    "messageEventReceived": true,
    "ok": true,
    "tauriFallbackRejected": true,
    "toolCallText": "echo: wails",
    "toolName": "smoke_echo"
  }
}
```

Known gaps / do not count as complete:

- This proves the stdio process/IPC path in a real Wails app, not the full backend MCP runtime. Frontend `McpService` and the MCP SDK still own initialize, tools/list, tool execution, prompts/resources, cache state, and runtime status behavior.
- Remote MCP HTTP/SSE/WebSocket diagnostics were not expanded in this slice.
- Old Rust `mcp_stdio_start/send/close` and `stdio_proxy` have been retired in the later "Completed MCP Stdio Rust Retirement Slice". If Tauri side-by-side stdio support is needed again, route it through the shared native facade instead of restoring the old Rust proxy.
- The smoke uses the built Go exe as its child server, so it proves process spawn/framing/events/close without adding node/cmd/powershell variability. It does not prove arbitrary third-party MCP server behavior.
- Injected native event support remains incomplete; `nativeEvents.listen` still only handles Wails and Tauri.

Next queue:

- Continue broader MCP runtime parity only where product value requires it; the old Rust stdio fallback is now retired.
- Build broader MCP runtime parity only if needed: cache/status/tool-list behavior, prompts/resources, tool execution surfaces, and settings/runtime status UI.
- Add a live Wails smoke for a real product workflow beyond settings/system/MCP stdio, ideally Skill file operations or PDF/textbook preview over the Go hybrid VFS.
- Continue product-critical OCR/PDF/textbook and chat/provider parity; do not count MCP stdio as full MCP completion.

## Completed MCP Stdio Rust Retirement Slice

Retired the old Tauri/Rust stdio proxy now that the Go/Wails MCP stdio path has both Go-level protocol tests and a live Wails/WebView2 UI smoke for `mcp_stdio_start`, `mcp_stdio_send`, and `mcp_stdio_close`.

What changed:

- Removed old Tauri command registrations for `mcp_stdio_start`, `mcp_stdio_send`, and `mcp_stdio_close` from `src-tauri/src/lib.rs`.
- Removed the old Rust command wrappers from `src-tauri/src/cmd/mcp.rs`; that module now keeps legacy MCP config persistence only.
- Removed `pub mod stdio_proxy` from `src-tauri/src/mcp/mod.rs`.
- Deleted `src-tauri/src/mcp/stdio_proxy.rs`.
- Left the frontend command names unchanged; in the target runtime they route through `src/runtime/native.ts` -> `src/runtime/wailsBridge.ts` -> Go `McpService`.

GitNexus / impact notes:

- `npx gitnexus impact mcp_stdio_start --repo "Deep Student" --direction upstream --depth 3 --include-tests`: LOW, 0 impacted.
- `npx gitnexus impact mcp_stdio_send --repo "Deep Student" --direction upstream --depth 3 --include-tests`: LOW, 0 impacted.
- `npx gitnexus impact mcp_stdio_close --repo "Deep Student" --direction upstream --depth 3 --include-tests`: LOW, 0 impacted.
- The broader frontend `TauriStdioClientTransport` class remains HIGH impact, but it was not edited in this retirement slice; its native calls continue to route to Go/Wails in the target runtime.

Metrics after this slice:

- Native inventory: 1531 scanned files / 920 native references / 827 invokes / 606 unique invokes / 65 listens / 47 unique listens / 28 emits / 24 unique emits.
- Native triage: 606 unique commands; `merge` 319, `replace` 81, `defer` 180, `delete` 26.
- Rust retirement map: 319 merged commands, 314 with Wails bridge routes, 6 merged Rust registrations, 30 merged Rust definitions, 26 retirement candidates, 0 direct Tauri blockers, 72 replace commands still registered in Rust.

Verification:

- `rg -n "mcp_stdio_(start|send|close)|stdio_proxy" src-tauri/src -g "*.rs"`: pass; no old Rust stdio fallback definitions, registrations, or module exports remain.
- `npm run test -- tests/vitest/runtime/wailsBridgeMcpStdio.behavior.test.ts tests/vitest/runtime/nativeMcpTauriFallback.behavior.test.ts tests/vitest/mcp/mcpStdioWailsContract.source.test.ts tests/vitest/settings/mcpLegacyDiagnosticsRetirement.source.test.ts tests/vitest/settings/mcpStdioDiagnosticsNativeFacade.source.test.ts tests/vitest/system/goLiveWailsSmoke.source.test.ts`: pass, 6 files / 22 tests.
- `go test ./internal/mcp ./internal/bindings` from `desktop-go`: pass.
- `npm run typecheck -- --pretty false`: pass.
- `npm run native:inventory -- --summary`: pass, metrics above.
- `npm run native:triage`: pass, regenerated `docs/generated/native-command-triage.{json,md}`.
- `node scripts/rust-retirement-map.mjs`: pass, regenerated `docs/generated/rust-retirement-map.{json,md}` with Rust registered merged commands reduced from 9 to 6.
- `git diff --check` for the touched Rust/docs/generated paths: pass.
- `cargo check --manifest-path src-tauri/Cargo.toml`: intentionally not used as the default proof; this slice's acceptance evidence is removal from old Rust source/registration, Wails/Go stdio tests, TypeScript checks, native inventory/triage, and the reduced Rust retirement map. Run `cargo check` only if a future Tauri-side structural edit needs transitional Rust validation.

Known gaps / do not count as complete:

- This retires only the old Rust stdio process proxy. It does not retire legacy MCP config persistence or the old Rust global MCP client/config modules.
- Full backend MCP runtime parity is still not complete; frontend MCP SDK still owns initialize, tools/list, prompts/resources, execution, and cache state.
- If someone runs the old Tauri shell on this branch and expects Tauri-side stdio MCP transport, that fallback is gone by design. The rewrite target is Go/Wails.

Next queue:

- Prefer a product-value slice next: live Wails Skill workflow smoke, PDF/textbook preview over Go hybrid VFS, OCR runner wiring, or chat provider/tool parity.
- Continue low-risk Rust retirement only when a Go/Wails workflow has tests and at least one smoke or strong source/behavior gate.

## Completed Live Wails Skill File Workflow Smoke Slice

Closed the live product-workflow proof gap for the Go/Wails Skill filesystem service. The smoke now launches the real Wails/WebView2 app, invokes the public frontend Skill API facade from the embedded React runtime, routes through `src/runtime/native.ts` and `src/runtime/wailsBridge.ts`, reaches Go `SkillService`, and proves create/read/update/list/delete plus read-after-delete rejection under a temporary app data skill root.

What changed:

- Added `npm run go:smoke:live-wails-skills`, implemented as `node scripts/go-live-wails-smoke.mjs --skills`.
- Extended `scripts/go-live-wails-smoke.mjs` with Skill result assertions, Windows-safe path normalization, directory-boundary inside checks, and a narrow expected Wails runtime `422` allowance only when the frontend proof has already shown read-after-delete rejection.
- Extended the live smoke hook in `src/main.tsx` with `runSkillWailsSmoke`.
- The smoke hook dynamically imports `./features/chat/skills/api` and calls `createSkill`, `readSkillFile`, `updateSkill`, `listSkillDirectories`, and `deleteSkill`; it no longer calls raw `nativeInvoke('skill_*')` inside the Skill workflow.
- The frontend proof creates a generated `wails-smoke-*` skill under the temporary `appDataDir/skills`, reads and updates `SKILL.md`, lists before and after deletion, deletes the generated skill directory, then asserts a post-delete read is rejected.
- Tightened frontend and runner path checks so a sibling such as `skills2` cannot satisfy the proof by simple prefix matching.
- Added source-contract coverage that the live Skill smoke uses the public API facade, forbids raw `nativeInvoke('skill_*')` inside the Skill smoke body, locks path normalization helpers, and records the expected read-after-delete runtime rejection.

GitNexus / impact notes:

- `npx gitnexus context createSkill --repo "Deep Student" --file src/features/chat/skills/api.ts`: found `src/features/chat/skills/api.ts:createSkill`.
- `npx gitnexus impact createSkill --repo "Deep Student" --direction upstream --depth 2 --include-tests`: LOW, 0 impacted in the current index.
- New smoke-hook/script helpers such as `assertSkillSmokeResult`, `runSkillWailsSmoke`, `normalizePathForSmoke`, and `filterExpectedWailsRuntimeHTTPFailures` are not currently indexed by GitNexus and returned `Target not found`; coverage is from source contracts, live Wails smoke, typecheck, Go tests, and code review.
- Two attempted subagent roles using unavailable `gpt-5.3-codex-spark` / `gpt-5.4-mini` model routes failed with 403; subsequent `gpt-5.5` review agents completed.

Metrics after this slice:

- Native inventory: 1531 scanned files / 920 native references / 827 invokes / 606 unique invokes / 65 listens / 47 unique listens / 28 emits / 24 unique emits.
- Native triage: 606 unique commands; `merge` 319, `replace` 81, `defer` 180, `delete` 26.
- Rust retirement map: 319 merged commands, 314 with Wails bridge routes, 6 merged Rust registrations, 30 merged Rust definitions, 26 retirement candidates, 0 direct Tauri blockers, 72 replace commands still registered in Rust.

Verification:

- `node --check scripts/go-live-wails-smoke.mjs`: pass.
- `npm run test -- tests/vitest/system/goLiveWailsSmoke.source.test.ts tests/vitest/runtime/wailsBridgeSkillService.behavior.test.ts tests/vitest/runtime/wailsBridgeSkillServiceContract.source.test.ts src/features/chat/skills/__tests__/skillApiNativeFacade.test.ts src/features/chat/skills/__tests__/spssProjectSkillLoader.test.ts`: pass, 5 files / 21 tests.
- `npm run typecheck -- --pretty false`: pass.
- `go test ./internal/skills ./internal/bindings` from `desktop-go`: pass.
- `npm run go:smoke:live-wails-skills`: pass. The run includes `npm run build`, frontend dist sync, static embed smoke, Go exe build, live Wails/WebView2 launch, CDP attach, embedded React mount proof, settings roundtrip proof, public Skill API facade CRUD proof, strict console/page/request checks with only the expected read-after-delete Wails runtime 422 consumed, and temp-data cleanup.
- `npm run go:check`: pass; full Go test suite plus Go app `--smoke` using a temporary data dir.
- `npm run native:inventory -- --summary`: pass, metrics above.
- `npm run native:triage`: pass, regenerated `docs/generated/native-command-triage.{json,md}`.
- `node scripts/rust-retirement-map.mjs`: pass, regenerated `docs/generated/rust-retirement-map.{json,md}` with metrics above.

Live smoke proof excerpt:

```json
{
  "isWails": true,
  "skills": {
    "createdContentIncludesName": true,
    "deleteSucceeded": true,
    "listAfterDeleteIncludesSkill": false,
    "listBeforeIncludesSkill": true,
    "ok": true,
    "readAfterDeleteRejected": true,
    "readContentIncludesName": true,
    "routeCreate": "skill_create -> SkillService.Create",
    "routeDelete": "skill_delete -> SkillService.Delete",
    "routeList": "skill_list_directories -> SkillService.ListDirectories",
    "routeRead": "skill_read_file -> SkillService.ReadFile",
    "routeUpdate": "skill_update -> SkillService.Update",
    "updatedContentIncludesName": true
  }
}
```

Known gaps / do not count as complete:

- This proves the Skill filesystem command workflow, not the full Skills management UI interaction flow.
- It does not implement or retire `workspace_*`, chat search/content, translation/essay streams, APKG/Anki import/export, or broader provider/tool parity.
- The smoke is Windows/WebView2-oriented because it depends on the current live Wails CDP smoke harness.

Next queue:

- Use this smoke as the regression gate for future Skill facade or `SkillService` changes.
- Prefer the next live product smoke in PDF/textbook preview over Go hybrid VFS or a chat provider/tool path.
- Continue low-risk Rust retirement from the generated map; old Rust Skill filesystem commands were already retired in the earlier Skill Rust retirement slice.

## Completed Go Frontend Embed Sync And Static Smoke Slice

Synced the real Vite/React frontend build into the Go/Wails embedded asset tree and added a static smoke gate that rejects the old placeholder shell. This is a frontend embed/static proof slice: it proves the Go binary can carry current production frontend assets and initialize cleanly, but it does not prove live Wails WebView rendering or frontend-to-Go binding dispatch.

What changed:

- Added `scripts/go-sync-frontend-dist.mjs` and `npm run go:sync:frontend-dist`.
- The sync script copies root `dist` into `desktop-go/cmd/deep-student-go/frontend/dist`, guards that source and target stay in the expected repo paths, and refuses to delete/copy outside the Go embed target.
- The sync script requires runtime build outputs: `index.html`, `assets`, `cmaps`, `icons`, `standard_fonts`, and `wasm`.
- The sync script excludes recursive/package-only output such as `dist/desktop-go/**` and `bundle-report.html`, so previous Go package binaries are not embedded back into the app.
- Added `scripts/go-frontend-embed-smoke.mjs` and `npm run go:smoke:frontend-embed`.
- The embed smoke rejects placeholder HTML, requires the React root, verifies referenced Vite JS/CSS assets exist, requires PDF/static assets including `pdf.worker.wrapper.mjs` and `pdf.worker.min.mjs`, rejects recursive `desktop-go`/bundle-report output, and runs `go run ./cmd/deep-student-go --smoke` with a temporary data dir.
- Updated `scripts/go-package-windows.mjs` so `npm run go:package:windows` syncs the frontend dist and runs the frontend embed smoke before `go build`, preventing stale placeholder or old assets from being compiled into the Windows Go exe.
- Added source-contract coverage for the new package script and for `get_app_data_dir` Wails routing through `SystemService.AppDataDir` before fallback paths.
- Ran the sync after a fresh `npm run build`; `desktop-go/cmd/deep-student-go/frontend/dist/index.html` now contains the Vite/React `<div id="root"></div>` and no longer contains `Deep Student Go shell` / `Wails migration shell`.

GitNexus / impact notes:

- `npx gitnexus status`: pass, index up to date at commit `1812ec7`.
- `npx gitnexus impact --repo "Deep Student" main --direction upstream`: LOW risk for the old indexed Rust build-script `main`; GitNexus does not currently provide useful coverage for new script entrypoints or the untracked Go shell files in this branch.
- `npx gitnexus detect_changes --repo "Deep Student"` remains unavailable in this CLI build, so scope was checked with generated native inventory/triage/retirement scripts plus targeted diffs/tests.

Metrics after this slice:

- Native inventory: 1531 scanned files / 911 native references / 818 invokes / 606 unique invokes / 65 listens / 47 unique listens / 28 emits / 24 unique emits.
- Native triage: 606 unique commands; `merge` 319, `replace` 81, `defer` 180, `delete` 26.
- Rust retirement map: 319 merged commands, 314 with Wails bridge routes, 9 merged Rust registrations, 33 merged Rust definitions, 29 retirement candidates, 0 direct Tauri blockers, 72 replace commands still registered in Rust.
- Wails bindings: unchanged, 13 services, 324 methods, 217 models.
- Frontend embed sync copied 1131 files from root `dist` to `desktop-go/cmd/deep-student-go/frontend/dist`.

Verification:

- `node --check scripts/go-sync-frontend-dist.mjs`: pass.
- `node --check scripts/go-frontend-embed-smoke.mjs`: pass.
- `node --check scripts/go-package-windows.mjs`: pass.
- `npm run test -- tests/vitest/system/goSyncFrontendDist.source.test.ts`: pass, 1 file / 2 tests.
- `npm run test -- tests/vitest/runtime/wailsBridgeSettingsSystem.behavior.test.ts tests/vitest/system/nativeSystemFacade.source.test.ts`: pass, 2 files / 3 tests.
- `npm run build`: pass; includes `npm run version:generate`, `npm run typecheck`, and Vite production build. Existing Vite CSS/chunk warnings remain warnings, not failures.
- `npm run go:sync:frontend-dist`: pass; copied 1131 files.
- `npm run go:smoke:frontend-embed`: pass; validates real Vite/React embed assets and Go `--smoke` with temporary data dir.
- `npm run go:check`: pass; full Go tests plus Go app `--smoke` using a temporary data dir.
- `npm run native:inventory -- --summary`: pass, metrics above.
- `npm run native:triage`: pass, regenerated `docs/generated/native-command-triage.{json,md}`.
- `node scripts/rust-retirement-map.mjs`: pass, metrics above.
- `npm run test -- tests/vitest/runtime/nativeAppDataDirWailsContract.source.test.ts tests/vitest/system/goSyncFrontendDist.source.test.ts tests/vitest/runtime/wailsBridgeSettingsSystem.behavior.test.ts tests/vitest/system/nativeSystemFacade.source.test.ts`: pass, 4 files / 7 tests.
- After strengthening the script safety guards, `npm run test -- tests/vitest/system/goSyncFrontendDist.source.test.ts tests/vitest/runtime/nativeAppDataDirWailsContract.source.test.ts`: pass, 2 files / 7 tests.
- After strengthening the script safety guards, `npm run go:sync:frontend-dist`: pass; copied 1131 files.
- After strengthening the script safety guards, `npm run go:smoke:frontend-embed`: pass; validates the synced embed tree with guarded deletion in place.
- `Select-String -Path desktop-go/cmd/deep-student-go/frontend/dist/index.html -Pattern '<div id="root"></div>','Deep Student Go shell','Wails migration shell'`: only the React root matched.
- `npm run go:package:windows`: pass; syncs frontend assets, runs `go-frontend-embed-smoke`, builds `dist/desktop-go/windows/Deep Student.exe`, copies root `pdfium.dll`, runs app smoke, then renders the bundled PDFium minimal-PDF smoke.
- `cargo check --manifest-path src-tauri/Cargo.toml`: intentionally not used; this slice did not modify old Rust/Tauri code and the rewrite target is Go/Wails.

Known gaps / do not count as complete:

- This does not prove that the native Wails WebView opens and renders the embedded React app.
- This does not prove frontend JavaScript boot, lazy-route loading, browser console health, native event delivery, or frontend-to-Go binding dispatch in a real Wails window.
- Static source-contract tests prove `get_app_data_dir` should route through `SystemService.AppDataDir` in Wails before fallback paths, but they do not replace a live WebView binding smoke.
- `go:package:windows` still builds a simple Go exe layout, not a full Wails v3 installer/MSI/NSIS/signed release bundle.
- Runtime-critical real OCR/scanned-PDF/textbook open-search parity, provider protocol parity, full MCP runtime parity, old data migration, and installer smoke remain incomplete.

Next queue:

- Add an automated live Wails UI binding smoke. On Windows, likely path is a test-only Wails launch mode with WebView2 remote debugging enabled, then Playwright `chromium.connectOverCDP` to assert React root render, no fatal console/resource errors, and a real `get_app_data_dir` / settings read-write binding roundtrip through Go.
- As a fallback until CDP attach is implemented, run a visible dev-server Wails smoke with `FRONTEND_DEVSERVER_URL=http://localhost:1422` and `DEEP_STUDENT_DATA_DIR` pointing at a temp dir, but do not count manual observation as parity.
- Wire settings-backed OCR recognition into the injected Go VFS OCR runner.
- Add a scanned-PDF smoke with PDFium preview pages feeding OCR and `GetResourceOcrInfo` / chat PDF OCR injection consuming the result.
- Continue old-data migration/backfill from Rust `files.preview_json` / `ocr_pages_json` / `extracted_text` / `page_count` into Go hybrid VFS metadata.

## Completed Textbook Import Progress Event Bridge Slice

Restored the legacy textbook import progress event contract on the Go/Wails `DstuService.AddTextbooks` path. This is an import-phase compatibility slice only; it does not implement native PDF raster rendering, OCR, semantic indexing, or full textbook open/search parity.

What changed:

- Added a Go `DstuService` event emitter and wired it into the shared Wails `EventBus` in `desktop-go/internal/app/app.go`.
- `AddTextbooks` now emits `textbook-import-progress` events for local textbook imports using the frontend's existing snake_case payload contract: `file_name`, `stage`, `progress`, `error`, plus ignored-but-useful `import_id`, `index`, `total`, `textbook_id`, and `resource_id`.
- The emitted stages are limited to the synchronous local import work Go can currently prove: `hashing`, `copying`, `saving`, `done`, and `error`. It intentionally does not emit `rendering`, because this slice does not run real PDF raster rendering or OCR.
- Added DSTU unit coverage for successful single-file import events, multi-file per-source `done` events, failure events for missing files, and no partial textbook creation on failed import.
- Added app-level proof that `App.New()` forwards DSTU `textbook-import-progress` events through the same EventBus that the Wails shell emits to the frontend.
- Fixed the existing Go `VfsService.StartPdfProcessing` event ordering on persistence failure so a failed `flushLocked()` emits error events without first emitting `media-processing-completed` / `pdf-processing-completed`.

GitNexus / impact notes:

- `desktop-go/internal/app/app.go:New`: LOW risk; one direct upstream caller, `desktop-go/cmd/deep-student-go/main.go:main`.
- `AddTextbooks`, `addTextbookFile`, `SetEventEmitter`, and `StartPdfProcessing`: target-not-found in the current GitNexus CLI index, so this slice is guarded by direct source inspection, Go tests, app/EventBus tests, typecheck, native inventory, and triage/retirement scripts instead of graph impact for those new Go symbols.
- `npx gitnexus detect_changes --repo "Deep Student"` is still unavailable in this GitNexus CLI build; the available CLI has `status`, `query`, `context`, `impact`, and `cypher`, but not `detect_changes`.

Metrics after this slice:

- Wails bindings: unchanged, 13 services, 324 methods, 217 models.
- Native inventory: 1531 scanned files / 911 native references / 818 invokes / 606 unique invokes / 65 listens / 47 unique listens / 28 emits / 24 unique emits.
- Native triage: 606 unique commands; `merge` 319, `replace` 81, `defer` 180, `delete` 26.
- Rust retirement map: 319 merged commands, 314 with Wails bridge routes, 9 merged Rust registrations, 33 merged Rust definitions, 29 retirement candidates, 0 direct Tauri blockers.
- Test fixtures used: inline/local minimal PDF byte files for successful import, multi-file import, missing-file failure, and app EventBus propagation.

Verification:

- `go test ./internal/dstu -run "TestAddTextbooks" -count=1` from `desktop-go`: pass.
- `go test ./internal/app -run "TestNewWires.*EventBus" -count=1` from `desktop-go`: pass.
- `go test ./internal/vfs -run "TestPdfProcessingControlCommandsEmitProgressEvents|TestStartPdfProcessingDoesNotEmitCompletedWhenFlushFails" -count=1` from `desktop-go`: pass.
- `go test ./internal/dstu ./internal/app ./internal/vfs -count=1` from `desktop-go`: pass.
- `go test ./... -count=1` from `desktop-go`: pass.
- `npm run go:check`: pass; full Go tests plus `go run ./cmd/deep-student-go --smoke`.
- `npm run test -- tests/vitest/chat-v2/vfsPdfProcessingApi.test.ts tests/vitest/learning-hub/textbook-preview-resolution.test.ts tests/vitest/learning-hub/file-preview-resolution.test.ts tests/vitest/learning-hub/preview-types.test.ts`: pass, 4 files / 12 tests.
- `npm run typecheck -- --pretty false`: pass.
- `npm run native:inventory -- --summary`: pass, metrics above.
- `npm run native:triage`: pass, generated 606-command triage.
- `node scripts/rust-retirement-map.mjs`: pass, metrics above.
- `npx gitnexus status`: pass, index up to date at commit `1812ec7`.
- `npx gitnexus impact New --repo "Deep Student" --direction upstream --depth 3 --include-tests`: pass, LOW risk for `desktop-go/internal/app/app.go:New`.
- `npx gitnexus impact AddTextbooks --repo "Deep Student" --direction upstream --depth 3 --include-tests`, `addTextbookFile`, `SetEventEmitter`, and `StartPdfProcessing`: target-not-found; recorded as a coverage gap.
- `cargo check --manifest-path src-tauri/Cargo.toml`: not used; no Rust files were edited in this slice.

Known gaps / do not count as complete:

- This does not implement real PDF raster rendering, real OCR, scanned-PDF OCR, semantic embeddings, or robust PDF extraction for complex/font-encoded documents.
- `textbook-import-progress` is a legacy compatibility UI event for local `AddTextbooks` import phases, not an authoritative PDF processing, OCR, or indexing progress signal.
- The event is global and the current frontend listeners still do not filter by `import_id`; the backend now includes `import_id/index/total`, but frontend filtering remains a future cleanup if concurrent imports matter.
- No live Wails desktop UI smoke or packaged-app smoke was run for the import modal.
- Existing Tauri/Rust textbook/OCR/PDF session data migration remains unimplemented.

Next queue:

- Move from import-event compatibility to real product replacement: native PDF raster preview generation over the existing `previewJson` path, then OCR/page-text persistence into `ocrPagesJson`, then rich textbook open/search state over the Go hybrid VFS.
- Add frontend filtering for `textbook-import-progress.import_id` if concurrent textbook import entry points can overlap.
- Keep old Rust PDF/OCR/document commands registered until real Go OCR/raster/document parity has tests and migration coverage.

## Completed Windows PDFium Package Smoke And Legacy Preview Metadata Slice

Closed the immediate Windows PDFium packaging proof gap for the Go/Wails migration shell and fixed a small legacy metadata compatibility leak in the Go hybrid VFS. This is still not a full Wails installer/live UI smoke; it proves the Go-owned executable layout can carry and load `pdfium.dll` without relying on Tauri resources or environment-variable shortcuts.

What changed:

- Added `desktop-go/internal/vfs/pdfium_smoke.go` with `SmokePDFiumRasterPreview`, a reusable minimal-PDF render smoke that asserts a real PNG page is produced.
- Added `--smoke-pdfium` to `desktop-go/cmd/deep-student-go/main.go` before app initialization, so PDFium runtime smoke no longer creates or reads the user's normal app data directory.
- Added `scripts/go-pdfium-smoke.mjs` and `npm run go:smoke:pdfium` to build a temporary Go exe, clear `DEEP_STUDENT_PDFIUM_PATH` / `DEEP_STUDENT_ENABLE_DEV_PDFIUM_PATHS`, and prove both production lookup layouts:
  - exe-adjacent `pdfium.dll`;
  - `..\Resources\pdfium.dll`.
- Added `scripts/go-package-windows.mjs` and `npm run go:package:windows` to build `dist/desktop-go/windows/Deep Student.exe`, copy the Go-owned root `pdfium.dll` next to the exe, then run both `--smoke` and `--smoke-pdfium` against the packaged layout using a temporary `DEEP_STUDENT_DATA_DIR`.
- Added `scripts/go-check.mjs` and changed `npm run go:check` so app smoke uses a temporary data directory instead of `%LOCALAPPDATA%\Deep Student`.
- Refactored `pdfiumCandidatePaths` into a testable helper and added Windows unit coverage for exe-adjacent, `..\Resources`, rejected relative `DEEP_STUDENT_PDFIUM_PATH`, and dev repo candidate gating.
- Fixed `VfsFile.PreviewJSON` / `OcrPagesJSON` exposure for legacy/imported `preview_json` and `ocr_pages_json` metadata.
- Fixed `SyncResourceUnits` to persist incoming `ocrPagesJson`, so source-stable resource sync can round-trip OCR page metadata instead of only `previewJson`.

GitNexus / impact notes:

- `npx gitnexus context "Function:desktop-go/cmd/deep-student-go/main.go:main" --repo "Deep Student"`: found Go `main`, no incoming callers, outgoing calls to app/settings/bindings initialization; LOW practical blast radius for adding an early smoke branch.
- `npx gitnexus impact pdfiumCandidatePaths`, `loadPdfium`, `renderPdfPagesWithPdfium`, `resourceToFile`, and `SyncResourceUnits`: target-not-found in the current CLI index, so this slice is guarded by source inspection, focused Go tests, package smoke, vet, frontend contract tests, generated native inventory/triage, and rust-retirement-map output.
- `npx gitnexus impact main --repo "Deep Student" --direction upstream --depth 3 --include-tests` resolves an old Rust `build.rs:main`, so it is not valid evidence for the Go entrypoint.
- `npx gitnexus status`: pass, index up to date at commit `1812ec7`.
- `npx gitnexus detect_changes --repo "Deep Student"` remains unavailable in this CLI build.

Metrics after this slice:

- Native inventory: 1531 scanned files / 911 native references / 818 invokes / 606 unique invokes / 65 listens / 47 unique listens / 28 emits / 24 unique emits.
- Native triage: 606 unique commands; `merge` 319, `replace` 81, `defer` 180, `delete` 26.
- Rust retirement map: 319 merged commands, 314 with Wails bridge routes, 9 merged Rust registrations, 33 merged Rust definitions, 29 retirement candidates, 0 direct Tauri blockers, 72 replace commands still registered in Rust.
- Packaged smoke output: `dist/desktop-go/windows/Deep Student.exe` plus `dist/desktop-go/windows/pdfium.dll`; `--smoke-pdfium` rendered one 816x1056 PNG page from the minimal PDF.

Verification:

- `go test ./internal/vfs -run "TestSyncResourceUnitsRegistersCompactResource|TestGetFileExposesLegacySnakeCasePreviewMetadata|TestPdfiumCandidatePaths|TestPdfiumIntegrationRendersMinimalPdfWhenConfigured" -count=1 -v`: pass; the env-gated PDFium integration test correctly skips by default.
- `go test ./internal/vfs -count=1`: pass.
- `go test ./cmd/deep-student-go ./internal/app -count=1`: pass.
- `go vet ./internal/vfs ./cmd/deep-student-go`: pass.
- `node --check scripts/go-check.mjs`, `scripts/go-pdfium-smoke.mjs`, and `scripts/go-package-windows.mjs`: pass.
- `go test ./... -count=1` from `desktop-go`: pass.
- `npm run go:check`: pass; app smoke now reports a temporary `_tmp/deep-student-go-check-*` data directory.
- `npm run go:smoke:pdfium`: pass; verifies exe-adjacent and `..\Resources` PDFium layouts without PDFium env vars or dev-path gate.
- `npm run go:package:windows`: pass; builds `dist/desktop-go/windows/Deep Student.exe`, copies root `pdfium.dll`, runs app smoke with `dist/desktop-go/windows/.smoke-data`, then renders PDFium PNG through the bundled DLL.
- `npm run test -- tests/vitest/chat-v2/vfsPdfProcessingApi.test.ts tests/vitest/learning-hub/textbook-preview-resolution.test.ts tests/vitest/learning-hub/file-preview-resolution.test.ts tests/vitest/learning-hub/preview-types.test.ts`: pass, 4 files / 12 tests.
- `npm run native:inventory -- --summary`: pass, metrics above.
- `npm run native:triage`: pass, regenerated `docs/generated/native-command-triage.{json,md}`.
- `node scripts/rust-retirement-map.mjs`: pass, regenerated `docs/generated/rust-retirement-map.{json,md}`.
- `cargo check --manifest-path src-tauri/Cargo.toml`: intentionally not used; this slice did not edit old Rust/Tauri code.

Known gaps / do not count as complete:

- This does not prove a real Wails installer/MSI/NSIS bundle, signed updater, or live desktop UI flow. It proves the Go-owned Windows executable layout and PDFium loader candidates.
- The later Go Frontend Embed Sync slice replaced the placeholder embedded assets with a real Vite/React build. Live Wails WebView/native-binding smoke remains a separate blocker.
- `go:package:windows` currently builds a simple exe layout, not a full Wails v3 release bundle.
- Non-Windows PDFium raster remains unavailable.
- Real OCR/scanned-PDF handling, robust PDF text extraction, semantic textbook search/open-state parity, old data migration, and full MCP runtime parity remain incomplete.
- Text-layer SVG fallback status semantics were later hardened by the Go VFS OCR/PDF readiness slice below; this package-smoke slice itself did not include that semantic fix.

Next queue:

- Run or enable a real Wails React UI/native-binding smoke now that static Go frontend embed assets are real.
- Implement real OCR/page-text persistence into `ocrPagesJson` by wiring the injected OCR runner to settings/provider OCR.
- Status semantics for text-layer SVG fallback were hardened in the following slice; remaining status work is truncated raster page availability.
- Continue toward old data migration and textbook open/search parity over the Go hybrid VFS.
- Keep old Rust PDF/OCR/document commands until Go raster + OCR/text + migration + UI smoke coverage is proven end to end.

## Completed Go VFS OCR/PDF Readiness Semantics Hardening Slice

Closed the immediate semantic contract gap between Go VFS PDF processing status, OCR metadata, frontend injection-mode readiness, and legacy metadata aliases. This is still not full scanned-PDF OCR product parity: the real OCR engine is only an injectable runner in this slice, and live Wails UI/textbook smoke remains open.

What changed:

- Added Go VFS OCR runner injection semantics for `StartPdfProcessing(..., "ocr_processing")`.
- A successful injected OCR run now persists real OCR metadata:
  - `ocrText`;
  - `ocrPagesJson`;
  - `ocrPagesSource`;
  - `ocrStatus`;
  - `ocrError`;
  - `ocrUpdatedAt`;
  - `ocrCompletedAt`;
  - `ocrPageCount`;
  - `ocrFailedPages`;
  - `textIndexState=indexed`;
  - `indexStatus=indexed`.
- The default OCR runner records `ocrStatus=unavailable` and does not claim OCR readiness.
- `GetResourceOcrInfo` now distinguishes extracted/native text from real OCR:
  - extracted PDF text remains `ActiveSource=extracted`;
  - explicit `ocrText` / `ocr_text` is real OCR;
  - `ocrPagesJson` is real OCR only when `ocrPagesSource` / `ocr_pages_source` is not `pdf_text_layer_estimated`;
  - estimated PDF text-layer pages can hydrate page info without making `HasOcr=true`.
- `ResolveResourceRefs` now emits a text `MultimodalBlock` from real OCR content so chat PDF OCR injection can consume OCR text via the existing resolved-resource contract.
- PDF processing readiness now keeps the three modes separate:
  - `text` means extracted/native searchable text;
  - `ocr` means real OCR text/pages;
  - `image` means raster image readiness, not merely a text-layer SVG fallback.
- Text-layer SVG previews remain available through `GetPdfPageImage`, but no longer advertise `readyModes: ["image"]`. They produce `completed_with_issues` with a non-empty `raster_preview` or `page_compression` failed stage when raster preview is unavailable.
- Legacy unannotated `previewJson` remains image-ready for compatibility with old PNG/JPEG preview data; annotated `previewSource=pdf_text_layer_svg` / `previewMimeType=image/svg+xml` is not image-ready.
- Expanded legacy snake_case metadata alias support in Go VFS read paths:
  - `page_count`;
  - `extracted_text`;
  - `ocr_text`;
  - `mime_type`;
  - `bookmarks_json`;
  - `original_path`;
  - `preview_json`;
  - `ocr_pages_json`;
  - `ocr_pages_source`.
- `SyncResourceUnits` now preserves incoming `extractedText` and `ocrText`, not only generic data and `ocrPagesJson`.
- `AttachmentInjectModeSelector` now follows the same backend-readyModes truth source as the send-blocking path; it no longer invents image readiness for processing image attachments.
- `IndexStatusView` direct command invocation moved from `@tauri-apps/api/core` to the native runtime facade, and behavior/source tests now mock/check the facade.

GitNexus / impact notes:

- `npx gitnexus impact pdfProcessingStatusForResource --repo "Deep Student" --direction upstream --depth 3 --include-tests`: target-not-found.
- `npx gitnexus impact mediaTypeForResource --repo "Deep Student" --direction upstream --depth 3 --include-tests`: target-not-found.
- `npx gitnexus impact GetResourceOcrInfo --repo "Deep Student" --direction upstream --depth 3 --include-tests`: target-not-found.
- `npx gitnexus impact StartPdfProcessing --repo "Deep Student" --direction upstream --depth 3 --include-tests`: target-not-found.
- `npx gitnexus impact resourceToFile --repo "Deep Student" --direction upstream --depth 3 --include-tests`: target-not-found.
- `npx gitnexus impact SyncResourceUnits --repo "Deep Student" --direction upstream --depth 3 --include-tests`: target-not-found.
- `npx gitnexus impact AttachmentInjectModeSelector --repo "Deep Student" --direction upstream --depth 3 --include-tests`: target-not-found.
- `IndexStatusView` upstream impact was checked by the frontend worker and reported LOW risk with zero direct impact in the current index.
- `npx gitnexus status`: pass, index up to date at commit `1812ec7`.
- `npx gitnexus detect_changes --repo "Deep Student"` remains unavailable in this CLI build.

Metrics after this slice:

- Native inventory: 1531 scanned files / 911 native references / 818 invokes / 606 unique invokes / 65 listens / 47 unique listens / 28 emits / 24 unique emits.
- Native triage: 606 unique commands; `merge` 319, `replace` 81, `defer` 180, `delete` 26.
- Rust retirement map: 319 merged commands, 314 with Wails bridge routes, 9 merged Rust registrations, 33 merged Rust definitions, 29 retirement candidates, 0 direct Tauri blockers, 72 replace commands still registered in Rust.
- Wails bindings: unchanged, 13 services, 324 methods, 217 models.

Verification:

- `go test ./internal/vfs -count=1` from `desktop-go`: pass.
- `go test ./... -count=1` from `desktop-go`: pass.
- `npm run go:check`: pass; full Go tests plus Go app `--smoke` using a temporary data dir.
- `go vet ./internal/vfs ./cmd/deep-student-go` from `desktop-go`: pass.
- `npm run typecheck -- --pretty false`: pass.
- `npm run test -- tests/vitest/chat-v2/vfsPdfProcessingApi.test.ts tests/vitest/chat-v2/inputBarMediaRoutingContract.source.test.ts tests/vitest/chat-v2/injectModeUtils.test.ts tests/vitest/chat-v2/context/fileDefinitionPdf.test.ts tests/vitest/learning-hub/IndexStatusView.behavior.test.tsx tests/vitest/learning-hub/indexStatusStateSource.source.test.ts tests/vitest/learning-hub/indexStatusProgress.source.test.ts tests/vitest/learning-hub/indexStatusVisibleRows.source.test.ts tests/vitest/learning-hub/textbook-preview-resolution.test.ts tests/vitest/learning-hub/file-preview-resolution.test.ts tests/vitest/learning-hub/preview-types.test.ts tests/vitest/settings/ocrEngineTestNativeFacade.source.test.ts`: pass, 12 files / 55 tests.
- `npm run native:inventory -- --summary`: pass, metrics above.
- `npm run native:triage`: pass, regenerated `docs/generated/native-command-triage.{json,md}`.
- `node scripts/rust-retirement-map.mjs`: pass, regenerated `docs/generated/rust-retirement-map.{json,md}`.
- `npx gitnexus status`: pass.
- `cargo check --manifest-path src-tauri/Cargo.toml`: intentionally not used; this slice did not modify old Rust/Tauri code and the rewrite target is Go/Wails.

Known gaps / do not count as complete:

- The real OCR engine is not yet wired from settings/provider code into the Go VFS pipeline; OCR is proven through injection tests only.
- Scanned PDFs still need a real end-to-end path: PDFium raster pages -> OCR provider/engine -> persisted OCR pages/text -> search/chat/textbook UI.
- Text-layer SVG fallback is a display fallback, not multimodal image readiness.
- Raster preview truncation still needs richer frontend/status semantics for page ranges beyond rendered pages.
- Static Go frontend embed assets were later replaced with a real Vite/React build, but live Wails WebView/native-binding smoke remains unproven.
- Old Rust/Tauri data migration for existing VFS/PDF/OCR rows remains incomplete.
- Full textbook open/search parity, full MCP runtime parity, chat provider/tool parity, and installer smoke remain open.

Next queue:

- Build the next live React-in-Wails smoke path over the now-real Go embed assets: enable a test-only WebView2 remote-debugging launch, attach Playwright over CDP, and prove React render plus a low-risk Go binding roundtrip.
- Wire settings-backed OCR recognition into the injected Go VFS OCR runner.
- Add a scanned-PDF smoke with PDFium preview pages feeding OCR and `GetResourceOcrInfo` / chat PDF OCR injection consuming the result.
- Add status semantics for truncated raster previews so `image` readiness can describe which pages are actually available.
- Continue old-data migration/backfill from Rust `files.preview_json` / `ocr_pages_json` / `extracted_text` / `page_count` into Go hybrid VFS metadata.

## Completed PDFium Raster Preview Slice

Added the first real Go/Wails PDF raster preview path over the hybrid VFS `previewJson` contract. This is a preview-rendering slice only; it does not implement OCR, scanned-PDF text recognition, robust PDF text extraction, semantic textbook search, or full textbook open/search parity.

What changed:

- Added Windows PDFium raster rendering in `desktop-go/internal/vfs/pdfium_windows.go` and a non-Windows stub in `pdfium_stub.go`.
- Added `desktop-go/internal/vfs/pdf_raster_preview.go` to render PDF pages as PNG files under VFS library preview paths and reference them from existing `previewJson` page refs.
- `UploadFile` and `StartPdfProcessing` now use the same preview contract: raster preview is attempted first, then the existing text-layer SVG fallback is used when PDFium is unavailable or fails and searchable text/estimated OCR pages exist.
- Raster preview metadata records `previewJson`, `previewSource=pdfium_raster`, `previewMimeType=image/png`, rendered page count, truncation state, `pageRendering*`, and `rasterPreview*` status/error fields.
- Legacy or caller-provided `previewJson` is preserved and does not trigger raster regeneration, including legacy `previewSource=pdf_text_layer_svg` metadata.
- Existing-hash reuploads preserve user metadata such as `bookmarks`; if a reupload provides a new preview JSON, generated preview sidecar metadata is cleared so old raster/text status does not describe the new preview.
- PDFium work now runs outside the VFS global mutex: the service snapshots resource metadata, unlocks while rendering/writing PNGs, then briefly relocks to verify hash/preview state and commit metadata. A regression test proves `GetPdfProcessingStatus` is not blocked while the fake renderer is waiting.
- PDFium loading is constrained on Windows:
  - production candidates are exe-adjacent `pdfium.dll` and packaged `../Resources/pdfium.dll`;
  - external `DEEP_STUDENT_PDFIUM_PATH` must be an absolute `pdfium.dll` path;
  - repo-root development candidates are gated by `DEEP_STUDENT_ENABLE_DEV_PDFIUM_PATHS=1`;
  - the loader uses `LoadLibraryEx` with DLL-directory/System32 search flags instead of ordinary CWD/PATH lookup.
- Reworked bitmap rendering to use `FPDFBitmap_CreateEx` with a Go-owned BGRA buffer, avoiding `uintptr` to Go pointer conversion and making `go vet` clean.
- Added an env-gated Windows PDFium integration smoke: by default it skips; with `DEEP_STUDENT_PDFIUM_PATH` it renders a minimal valid PDF and asserts PNG bytes.

GitNexus / impact notes:

- `fileMetadata`, `pdfiumCandidatePaths`, `UploadFile`, `StartPdfProcessing`, and newer Go VFS helpers are target-not-found in the current GitNexus CLI index, so this slice is guarded by direct source inspection, Go tests, race/vet checks, cross-platform compile, frontend tests, and generated native inventory/triage/retirement artifacts.
- `npx gitnexus status`: pass, index up to date at commit `1812ec7`.
- `npx gitnexus detect_changes --repo "Deep Student"` remains unavailable in this CLI build.

Metrics after this slice:

- Native inventory: 1531 scanned files / 911 native references / 818 invokes / 606 unique invokes / 65 listens / 47 unique listens / 28 emits / 24 unique emits.
- Native triage: 606 unique commands; `merge` 319, `replace` 81, `defer` 180, `delete` 26.
- Rust retirement map: 319 merged commands, 314 with Wails bridge routes, 9 merged Rust registrations, 33 merged Rust definitions, 29 retirement candidates, 0 direct Tauri blockers, 72 replace commands still registered in Rust.
- Raster preview cap: up to 12 rendered PDF pages per resource for this first preview slice.

Verification:

- `go test ./internal/vfs -run "TestUploadFileStoresRasterPreviewWhenRendererSucceeds|TestStartPdfProcessingStoresRasterPreviewWhenRendererSucceeds|TestStartPdfProcessingDoesNotHoldVfsLockWhileRenderingRasterPreview|TestUploadFileFallsBackToTextPreviewWhenRasterRendererFails|TestStartPdfProcessingFallsBackToTextPreviewWhenRasterRendererFails|TestUploadFilePreservesLegacyPreviewJsonWhenRasterRendererExists|TestUploadFilePreservesProvidedTextSourcePreviewJson|TestStartPdfProcessingPreservesLegacyPreviewJson|TestUploadFileDedupePreservesUserMetadata|TestUploadFileReplacesGeneratedPreviewMetadataOnDedupePreviewJson" -count=1`: pass.
- `go test ./internal/vfs -count=1`: pass.
- `go test ./... -count=1` from `desktop-go`: pass.
- `go vet ./...` from `desktop-go`: pass.
- `go test -race ./internal/vfs -run "TestStartPdfProcessingDoesNotHoldVfsLockWhileRenderingRasterPreview|TestUploadFileStoresRasterPreviewWhenRendererSucceeds" -count=1`: pass.
- Linux cross-compile check for the non-Windows stub: `GOOS=linux GOARCH=amd64 go test ./internal/vfs -c`: pass.
- `DEEP_STUDENT_PDFIUM_PATH=..\pdfium.dll go test ./internal/vfs -run TestPdfiumIntegrationRendersMinimalPdfWhenConfigured -count=1 -v`: pass; rendered a minimal PDF through the real local `pdfium.dll`.
- Default integration-test behavior without env: `go test ./internal/vfs -run TestPdfiumIntegrationRendersMinimalPdfWhenConfigured -count=1 -v`: pass with skip.
- `npm run go:check`: pass; full Go tests plus `go run ./cmd/deep-student-go --smoke`.
- `npm run test -- tests/vitest/chat-v2/vfsPdfProcessingApi.test.ts tests/vitest/learning-hub/textbook-preview-resolution.test.ts tests/vitest/learning-hub/file-preview-resolution.test.ts tests/vitest/learning-hub/preview-types.test.ts`: pass, 4 files / 12 tests.
- `npm run typecheck -- --pretty false`: pass.
- `npm run native:inventory -- --summary`: pass, metrics above.
- `npm run native:triage`: pass, generated 606-command triage.
- `node scripts/rust-retirement-map.mjs`: pass, metrics above.
- `cargo check --manifest-path src-tauri/Cargo.toml`: intentionally not used; this slice is Go/Wails/VFS work and did not edit Rust/Tauri registrations.

Known gaps / do not count as complete:

- No live Wails desktop UI smoke was run for PDF preview rendering.
- Windows Go-owned exe-adjacent and `..\Resources\pdfium.dll` package layout smoke is covered by the later Windows PDFium Package Smoke slice above; full Wails installer/release bundle smoke remains open.
- Non-Windows builds intentionally report PDFium raster unavailable until platform-specific libraries/loaders are added.
- Raster preview is capped and preview-only; it is not full document rendering parity, OCR, or searchable page text generation.
- Scanned PDF OCR, complex/font-encoded PDF text extraction, `ocrPagesJson` persistence from real OCR, semantic indexing/ranking, and full textbook open/search parity remain incomplete.
- The preview PNG files are not yet backed by a storage GC for stale sidecar files after metadata replacement; current correctness relies on metadata no longer pointing at stale files.

Next queue:

- Wire and smoke live Wails textbook/PDF preview UX against `previewSource=pdfium_raster`.
- Add real OCR/page-text persistence into `ocrPagesJson` as a separate slice, then connect textbook search/open-state flows over the Go hybrid VFS.
- Move beyond CLI/package-layout smoke into a real Wails React UI smoke for PDF preview consumption.
- Consider a helper-process renderer for untrusted/bulk PDFs if PDFium crash/hang isolation becomes a product stability requirement.
- Keep old Rust PDF/OCR/document commands until Go raster + OCR/text + migration + UI smoke coverage is proven end to end.

## Completed MCP Stdio Protocol Smoke Hardening Slice

Hardened the Go/Wails MCP stdio process proxy with a real protocol smoke inside the Go test harness. This slice deliberately did not retire the old Rust stdio fallback because `isTauriStdioSupported` currently has CRITICAL blast radius while the branch still keeps a side-by-side Tauri runtime.

What changed:

- Extended `desktop-go/internal/mcp/stdio_test.go` beyond plain JSONL echo coverage.
- Added content-length echo coverage, matching the frontend default `testMcpStdioFrontend(... framing: 'content_length')` path.
- Added a minimal MCP protocol child process inside the Go test binary. It handles:
  - `initialize`
  - `notifications/initialized`
  - `tools/list`
  - `tools/call` for a `smoke_echo` tool
- The protocol smoke runs through the real Go `StartStdioSession`, `SendStdioMessage`, EventBus-style emitted `mcp-stdio-{session}-message` events, and `CloseStdioSession`, for both `jsonl` and `content_length` framing.
- Verified close behavior by asserting send-after-close is rejected.

GitNexus / impact notes:

- Old Rust `mcp_stdio_start`: LOW risk, 0 upstream callers in the current index.
- Old Rust `mcp_stdio_send`: LOW risk, 0 upstream callers in the current index.
- Old Rust `mcp_stdio_close`: LOW risk, 0 upstream callers in the current index.
- `isTauriStdioSupported`: CRITICAL risk. Direct upstream callers include the settings MCP editor, `TauriStdioClientTransport.start`, and `McpService.connectServer`; indirect callers include settings rendering, `connectAll`, `connectServerById`, `ensureConnected`, server tool/prompt/resource fetches, MCP status UI, and bootstrap flows. Because of that blast radius, this slice did not change Tauri stdio runtime detection or delete the Rust fallback commands.

Verification:

- `go test ./internal/mcp -count=1` from `desktop-go`: pass.
- `go test ./... -count=1` from `desktop-go`: pass.
- `npm run test -- tests/vitest/runtime/wailsBridgeMcpStdio.behavior.test.ts tests/vitest/mcp/mcpStdioWailsContract.source.test.ts tests/vitest/settings/mcpStdioDiagnosticsNativeFacade.source.test.ts tests/vitest/settings/mcpLegacyDiagnosticsRetirement.source.test.ts`: pass, 4 files / 12 tests.
- `npm run typecheck -- --pretty false`: pass.
- `npm run native:inventory -- --summary`: pass, 1531 scanned files / 911 native references / 818 invokes / 606 unique invokes / 65 listens / 47 unique listens / 28 emits / 24 unique emits.

Known gaps / do not count as complete:

- This is a Go process/protocol smoke, not a live Wails desktop UI smoke or packaged-app smoke.
- Superseded by the later live Wails MCP stdio smoke and MCP stdio Rust retirement slices: old Rust `mcp_stdio_start/send/close` registrations and `stdio_proxy` are now removed.
- This still does not complete full backend MCP runtime parity. Frontend `McpService` and the MCP SDK still own initialize, tools/list, tool execution, prompts, resources, cache state, and runtime status behavior.
- HTTP/SSE/WebSocket MCP diagnostics and long-running MCP tool flows were not expanded in this slice.

Next queue:

- Move main effort back to PDF/Textbook/OCR or Chat provider/tool parity; MCP stdio is now better guarded, but it is not the dominant product replacement blocker.
- Future MCP stdio work should keep using the Go/Wails `McpService` plus native facade. Do not restore the old Rust stdio proxy unless the branch explicitly reintroduces Tauri-side stdio support with a new compatibility design.

## Completed Skill File API Go/Wails Replacement Slice

Migrated the Chat Skills filesystem command family from direct Tauri-only calls to a Go/Wails service:

- `skill_list_directories`
- `skill_read_file`
- `skill_create`
- `skill_update`
- `skill_delete`

What changed:

- Added `desktop-go/internal/skills.Service`, a lean filesystem service for listing skill directories, reading `SKILL.md`, creating skills, updating skills, and deleting skill directories.
- Added `desktop-go/internal/bindings.SkillService`, wired it into `app.App`, registered it in the Wails service list, and regenerated Wails bindings. Generated bindings now report 13 services, 324 methods, and 217 models.
- Routed all five `skill_*` commands through `src/runtime/wailsBridge.ts` to Go `SkillService`.
- Moved `src/features/chat/skills/api.ts` and `src/features/chat/skills/loader.ts` to `@/runtime/native`; `loader.ts` now uses `getAppDataDir()` from the native facade instead of direct `@tauri-apps/api/path`.
- Added backend guards beyond the old broad allowlist: only direct child skill directories under configured roots are accepted; read/update targets must be `SKILL.md`; create/update/delete require writable roots; Cursor skills are read-only; root deletion is rejected; symlink/reparse-point paths are rejected using platform file identity checks; skill IDs are limited to ASCII letters/numbers/hyphen/underscore with reserved Windows names rejected; create/update content is capped at 512 KiB.
- Updated `scripts/native-triage.mjs` so the five skill commands are tracked as `chat/merge` instead of `chat/replace`.

GitNexus / impact notes:

- `loadSkillsFromDirectory`: LOW risk; direct upstream `loadSkillsFromFileSystem`, indirect `reloadSkills`.
- `loadSkillsFromFileSystem`: LOW risk; direct upstream `reloadSkills`.
- `loadSingleSkill`: LOW risk; 0 upstream callers in the current index.
- Newly added Go `SkillService` symbols are not in the current GitNexus index yet; coverage is guarded with Go unit tests, Wails binding generation, Wails bridge behavior tests, source contracts, typecheck, inventory, and generated triage/retirement evidence.

Metrics after this slice:

- Wails bindings: 13 services, 324 methods, 217 models.
- Native inventory: 1531 scanned files / 911 native references / 818 invokes / 606 unique invokes / 65 listens / 47 unique listens / 28 emits / 24 unique emits.
- Native triage: 606 unique commands; `merge` 319, `replace` 81, `defer` 180, `delete` 26.
- Rust retirement map: 319 merged commands, 314 with Wails bridge routes, 14 merged commands still registered in Rust, 38 merged Rust definitions, 34 retirement candidates, 0 direct Tauri blockers, 72 replace commands still registered in Rust.

Verification:

- `go test ./internal/skills ./internal/app ./internal/bindings -count=1` from `desktop-go`: pass.
- `go test ./... -count=1` from `desktop-go`: pass.
- `go vet ./internal/skills` from `desktop-go`: pass.
- `go run ./cmd/deep-student-go --smoke` from `desktop-go`: pass; app state initialized and reported the local Deep Student data directory.
- `npm run go:bindings`: pass; Wails generated 13 services / 324 methods / 217 models.
- `npm run test -- src/features/chat/skills/__tests__/spssProjectSkillLoader.test.ts src/features/chat/skills/__tests__/skillApiNativeFacade.test.ts tests/vitest/runtime/wailsBridgeSkillService.behavior.test.ts tests/vitest/runtime/wailsBridgeSkillServiceContract.source.test.ts tests/vitest/runtime/wailsBridgeMcpStdio.behavior.test.ts tests/vitest/runtime/wailsBridgeSettingsSystem.behavior.test.ts`: pass, 6 files / 18 tests.
- `npm run typecheck -- --pretty false`: pass.
- `npm run native:inventory -- --summary`: pass.
- `node scripts/native-triage.mjs && node scripts/rust-retirement-map.mjs`: pass; generated docs updated with the metrics above.

Known gaps / do not count as complete:

- Old Rust `skill_*` handlers and registrations remain as Tauri fallback/reference code. They are now retirement candidates, but deletion should wait for a live Wails smoke or a broader chat/skills runtime smoke.
- This slice does not implement `workspace_*`, `chat_v2_search_content`, APKG/Anki import/export, translation/essay streaming, or broader chat provider/tool parity.
- No live Wails desktop UI smoke was run for list/read/create/update/delete; current proof is Go shell init smoke plus unit/source/bridge/typecheck/generated inventory evidence.
- `cargo check` was intentionally not used because this slice did not edit old Rust. If the old Rust `skill_*` registrations are deleted later, run `cargo check --manifest-path src-tauri/Cargo.toml`.

Next queue:

- Run a live Wails skill filesystem UI smoke when practical; old Rust `skill_*` registrations and `src-tauri/src/chat_v2/skills.rs` were retired in the follow-up slice below.
- Continue with the product-critical slices: PDF/Textbook/OCR mainline over Go hybrid VFS, Chat provider/tool parity, MCP live stdio smoke and backend runtime parity, old data migration, and Wails packaging smoke.

## Completed Skill File Rust Retirement Slice

Retired the old Tauri/Rust implementation after the Go/Wails `SkillService` replacement passed unit, bridge, source-contract, typecheck, inventory, and Go shell smoke gates.

What changed:

- Removed old Tauri command registrations for `skill_list_directories`, `skill_read_file`, `skill_create`, `skill_update`, and `skill_delete` from `src-tauri/src/lib.rs`.
- Removed `pub mod skills` and stale skill command re-exports from `src-tauri/src/chat_v2/mod.rs`.
- Deleted the old Rust-only `src-tauri/src/chat_v2/skills.rs` module.
- Kept unrelated Rust `workspace::skills` and `tools::skills_executor` modules in place; those are workspace/tool execution surfaces and are not the retired filesystem command module.

GitNexus / impact notes:

- `skill_list_directories`: LOW risk, 0 upstream callers.
- `skill_read_file`: LOW risk, 0 upstream callers.
- `skill_create`: LOW risk, 0 upstream callers.
- `skill_update`: LOW risk, 0 upstream callers.
- `skill_delete`: LOW risk, 0 upstream callers.

Metrics after this slice:

- Native triage remains 606 unique commands; `merge` 319, `replace` 81, `defer` 180, `delete` 26.
- Rust retirement map: 319 merged commands, 314 with Wails bridge routes, 9 merged commands still registered in Rust, 33 merged Rust definitions, 29 retirement candidates, 0 direct Tauri blockers, 72 replace commands still registered in Rust.

Verification:

- Source search over `src-tauri/src`: no old `chat_v2::skills` command registration/module path and no old `skill_list_directories`, `skill_read_file`, `skill_create`, `skill_update`, or `skill_delete` Rust definitions remain. Matches for `workspace::skills` and `skills_executor` are unrelated and intentionally preserved.
- `cargo check --manifest-path src-tauri/Cargo.toml`: pass; old warnings remain.
- `node scripts/native-triage.mjs && node scripts/rust-retirement-map.mjs`: pass; generated retirement map updated with the metrics above.

Known gaps / do not count as complete:

- This retires only the old Rust skill file command module. It does not implement or retire `workspace_*`, chat search/content, translation/essay streams, APKG/Anki import/export, or broader provider/tool parity.
- No live Wails desktop UI smoke was run for the skill editor/listing flow.

## Completed MCP Stdio Go/Wails Migration Slice

Migrated the runtime MCP stdio process/IPC path from Tauri-only frontend calls to a Go/Wails native process proxy:

- `mcp_stdio_start`
- `mcp_stdio_send`
- `mcp_stdio_close`

What changed:

- Added `desktop-go/internal/mcp.Service`, a lean stdio process proxy with command/args/env/cwd support, JSONL and content-length framing, session IDs, explicit close, app shutdown cleanup, and Wails EventBus `mcp-stdio-{session}-message/error/closed` events.
- Added `desktop-go/internal/bindings.McpService`, wired it into `app.App`, registered it in the Wails service list, and regenerated Wails bindings. Generated bindings now report 12 services, 319 methods, and 215 models.
- Routed legacy `mcp_stdio_start`, `mcp_stdio_send`, and `mcp_stdio_close` through `src/runtime/wailsBridge.ts` to Go `McpService`.
- Reworked `src/mcp/tauriStdioTransport.ts` to use `@/runtime/native` invoke and `@/runtime/nativeEvents` listen instead of direct `@tauri-apps/api/core` and `@tauri-apps/api/event`; the same frontend MCP SDK transport now works under Wails, Tauri fallback, and injected native runtimes.
- Moved settings-page stdio diagnostics in `McpEditorSection.tsx` off direct `tauriInvoke('test_mcp_connection')`; stdio tests now use `testMcpStdioFrontend`, which drives the same native stdio transport and frontend MCP SDK `Client` path as real MCP runtime connections.
- Kept stderr as a drained backend log channel rather than surfacing every MCP server stderr line as a frontend transport error, matching the old Rust proxy boundary.
- Redacted stdio debug events by default. `mcp-stdio-send` and `mcp-stdio-recv` now emit byte length, JSON-RPC id/method, and top-level param/result keys; raw payloads require explicit local opt-in via `localStorage.deep_student_mcp_stdio_debug_raw = '1'`.
- Updated `scripts/native-triage.mjs` so the three stdio commands are tracked as `mcp/merge`.
- Kept old Rust `mcp_stdio_start/send/close` registered for now as side-by-side Tauri fallback/reference code. The legacy MCP diagnostics wrappers were cleaned in the later cleanup slice below; do not retire stdio fallback until live Wails smoke proves spawn/send/close equivalence.

GitNexus / impact notes:

- `TauriStdioClientTransport`: HIGH risk. Direct caller is `McpService.connectServer`; indirect impact includes `connectAll`, `connectServerById`, `ensureConnected`, settings health checks, tool/prompt/resource fetches, and MCP status UI. The edit was limited to its native IPC/event boundary and left MCP SDK protocol logic intact.
- `mcpFrontendTester.runClient` and existing remote tester functions report CRITICAL because they feed the user-visible settings test flows. This slice added `testMcpStdioFrontend` without changing existing SSE/HTTP/WebSocket tester behavior.
- `App` and `app.New`: LOW risk; direct impact is Wails app construction and indirect impact is `main`.
- New Go `McpService` and `NewMcpService` were not present in the current GitNexus index yet; coverage is guarded with Go unit tests, binding/app tests, Wails bridge behavior tests, and source contracts.

Metrics after this slice:

- Wails bindings: 12 services, 319 methods, 215 models.
- Native triage: 611 unique commands; `merge` 314, `replace` 86, `defer` 180, `delete` 31.
- Rust retirement map: 314 merged commands, 309 with Wails bridge routes, 9 merged commands still registered in Rust, 33 merged Rust definitions, 29 retirement candidates, 0 direct Tauri blockers, 77 replace commands still registered in Rust.

Verification:

- `go test ./internal/mcp -count=1` from `desktop-go`: pass.
- `go test ./internal/mcp ./internal/app ./internal/bindings -count=1` from `desktop-go`: pass.
- `npm run go:bindings`: pass; Wails generated 12 services / 319 methods / 215 models.
- `npm run test -- tests/vitest/runtime/wailsBridgeMcpStdio.behavior.test.ts tests/vitest/mcp/mcpStdioWailsContract.source.test.ts tests/vitest/settings/mcpStdioDiagnosticsNativeFacade.source.test.ts tests/vitest/runtime/wailsBridgeSettingsSystem.behavior.test.ts`: pass, 8 tests.
- `go test ./... -count=1` from `desktop-go`: pass.
- `npm run typecheck -- --pretty false`: pass.
- `npm run native:inventory -- --summary`: pass, 1527 scanned files / 916 native references / 823 invokes / 611 unique invokes / 65 listens / 47 unique listens / 28 emits / 24 unique emits.
- `node scripts/native-triage.mjs && node scripts/rust-retirement-map.mjs`: pass; generated docs updated with the metrics above.
- Source search over `src/mcp/tauriStdioTransport.ts`, `src/features/settings/components/McpEditorSection.tsx`, and `src/runtime/wailsBridge.ts`: no direct Tauri import remains in the stdio transport/settings diagnostics path; `mcp_stdio_start/send/close` route through the native/Wails bridge.
- Scoped `git diff --check` for touched Go/TS/generated files: pass.

Known gaps / do not count as complete:

- This completes the MCP stdio process/IPC path only. It does not make Go a full backend MCP cache/status/tool-list runtime; frontend `McpService` and the MCP SDK still own initialize, tools/list, tool execution, prompts, resources, and cache state.
- Old Rust `mcp_stdio_start/send/close` remains registered as a side-by-side Tauri fallback/reference path. It should be retired only after live Wails smoke covers spawn/send/close and settings diagnostics.
- The legacy direct Tauri wrappers for `test_mcp_connection`, `test_mcp_http`, `test_mcp_sse`, `test_mcp_websocket`, and `test_mcp_modelscope` were removed in the later MCP legacy diagnostics cleanup slice below.
- This is not broader chat tool-calling/provider-tool parity.
- `cargo check` was intentionally not used because this slice did not edit old Rust; verification used Go/Wails tests, TS checks, source searches, and generated triage/retirement evidence.

Next queue:

- Run a live Wails stdio smoke if practical, then retire old Rust `mcp_stdio_start/send/close` registrations and `stdio_proxy` only after the Go path proves equivalent.
- Continue backend MCP runtime parity separately: cache/status/tool-list behavior, tool execution, prompts/resources, and provider/tool approval integration.
- Continue OCR/PDF/textbook mainline and provider/chat parity separately; do not count this stdio slice as full MCP runtime completion.

## Completed OCR Engine Diagnostic Go Migration Slice

Migrated the settings-page OCR engine test command from the old Rust/Tauri command surface to Go/Wails:

- `test_ocr_engine`

What changed:

- Added Go `SettingsService.TestOCREngine` with the old request/response envelope preserved: `{ request: { imageBase64, engineType, configId } }` and `engineType`, `engineName`, `text`, `regions`, `elapsedMs`, `success`, `error` response fields.
- The Go diagnostic validates data URL or raw base64 image input, resolves `configId` before `engineType`, uses OCR model metadata for display names, resolves API key/base URL/vendor headers from existing settings config, and sends an OpenAI-compatible non-streaming multimodal `/chat/completions` request.
- Provider/config/network/response failures return `success:false` diagnostics instead of rejecting the whole settings-page comparison batch; invalid image base64 still rejects the command, matching the old command's hard input-error semantics.
- Added lightweight Paddle JSON block and DeepSeek `<|ref|>/<|det|>` region parsing for diagnostic display; generic providers fall back to one text region.
- Routed `src/features/settings/components/OcrEngineTestPanel.tsx` through `@/runtime/native` instead of direct `@tauri-apps/api/core`.
- Added `test_ocr_engine` to `src/runtime/wailsBridge.ts`, `src/runtime/native.ts`, generated Wails bindings, and `scripts/native-triage.mjs` as `settings/merge`.
- Retired the old Rust `test_ocr_engine` command registration, DTOs, image parsing helpers, and the now-exclusive `LLMManager::test_ocr_with_engine` helper. OCR adapter code, model migration helpers, Chat OCR, and the real OCR/PDF pipeline remain in place.

GitNexus / impact notes:

- Old Rust `test_ocr_engine`: LOW risk, 0 upstream callers.
- Go binding `SettingsService`: LOW risk; direct caller `NewSettingsService`, indirect `main`.
- `OcrEngineTestPanel`: LOW risk, 0 upstream callers.
- Old Rust `test_ocr_with_engine`, `parse_base64_image`, and `infer_extension_from_data_url`: LOW risk; their direct upstream was only the retired Rust `test_ocr_engine` path.

Metrics after this slice:

- Wails bindings: 11 services, 316 methods, 215 models.
- Native triage: 611 unique commands; `merge` 311, `replace` 86, `defer` 183, `delete` 31.
- Rust retirement map: 311 merged commands, 306 with Wails bridge routes, 7 merged commands still registered in Rust, 31 merged Rust definitions, 27 retirement candidates, 0 direct Tauri blockers, 77 replace commands still registered in Rust.

Verification:

- `go test ./internal/settings ./internal/bindings -count=1` from `desktop-go`: pass.
- `npm run go:bindings`: pass; Wails generated 316 methods / 215 models.
- `npm run test -- tests/vitest/runtime/wailsBridgeSettingsSystem.behavior.test.ts tests/vitest/settings/ocrEngineTestNativeFacade.source.test.ts`: pass.
- `go test ./... -count=1` from `desktop-go`: pass.
- `npm run typecheck -- --pretty false`: pass.
- `npm run native:inventory -- --summary`: pass, 1526 scanned files / 916 native references / 823 invokes / 611 unique invokes.
- `node scripts/native-triage.mjs && node scripts/rust-retirement-map.mjs`: pass; generated docs updated.
- Source search over `src-tauri/src/cmd/ocr.rs`, `src-tauri/src/llm_manager/mod.rs`, and `src-tauri/src/lib.rs`: no old Rust `test_ocr_engine`, `test_ocr_with_engine`, `OcrTest*`, `parse_base64_image`, or `infer_extension_from_data_url` remains.
- Scoped `git diff --check` for the touched Go/TS/Rust/generated files: pass.

Known gaps / do not count as complete:

- This is only the settings-page OCR diagnostic seam. It does not complete real OCR extraction for scanned PDFs/images, textbook OCR/open/search, Anki OCR orchestration, or a shared full provider adapter layer.
- Go `system_ocr` diagnostic currently returns `success:false` with an explicit unsupported message instead of silently passing; native system OCR parity remains part of the broader OCR/PDF mainline.
- `rustfmt --edition 2021` was attempted for touched Rust files but was blocked by pre-existing trailing whitespace in unrelated `src-tauri/src/translation/pipeline.rs`; no unrelated cleanup was made.

Next queue:

- Continue OCR/PDF mainline separately: scanned image OCR, robust PDF extraction, textbook import/open/search, and Anki document OCR orchestration.
- Visible MCP legacy diagnostics cleanup is now covered by the later MCP cleanup slice; remaining MCP work is live stdio smoke and backend runtime parity.
- Continue retiring small Go-backed Rust command wrappers, prioritizing commands with Wails routes and low GitNexus impact.

## Completed Chat V2 Send/Replay Rust Retirement Slice

Retired the Go/Wails-backed Chat V2 send/replay command batch from the old Rust/Tauri command surface:

- `chat_v2_send_message`
- `chat_v2_cancel_stream`
- `chat_v2_retry_message`
- `chat_v2_edit_and_resend`
- `chat_v2_continue_message`

What changed:

- Removed the five old Tauri command registrations from `src-tauri/src/lib.rs`.
- Removed the five old command re-exports from `src-tauri/src/chat_v2/handlers/mod.rs` and `src-tauri/src/chat_v2/mod.rs`.
- Reduced `src-tauri/src/chat_v2/handlers/send_message.rs` from the legacy Rust send/retry/edit/continue pipeline command module to a tiny transitional replay helper module.
- Preserved `apply_original_skill_snapshot_overrides` because old Rust variant retry commands still call it until multi-variant parity is retired or moved to Go.
- Kept the legacy command names active through `src/runtime/wailsBridge.ts`, where they route to Go `ChatService`.

GitNexus / impact notes:

- `chat_v2_send_message`, `chat_v2_cancel_stream`, `chat_v2_retry_message`, `chat_v2_edit_and_resend`, and `chat_v2_continue_message`: LOW risk, 0 impacted callers for the old Rust command wrappers.
- Several private helpers in the old `send_message.rs` reported CRITICAL only because they were called by the retired Rust command wrappers. They were removed together with that old wrapper chain.
- `apply_original_skill_snapshot_overrides`: CRITICAL before edit because old Rust `retry_variant_impl` and `retry_variants_impl` still call it. It was intentionally preserved with the same replay snapshot merge behavior.

Verification:

- `rg "chat_v2_(send_message|cancel_stream|retry_message|edit_and_resend|continue_message)" src-tauri/src/chat_v2 src-tauri/src/lib.rs -S`: no old Rust command definition, registration, or re-export remains; only a prose comment in `workspace_handlers.rs` still mentions `chat_v2_continue_message`.
- `rg "chat_v2_(send_message|cancel_stream|retry_message|edit_and_resend|continue_message)" src -g "*.ts" -g "*.tsx" -S`: production calls remain in `TauriAdapter.ts`, routed through `@/runtime/native`, plus Wails bridge compatibility routes.
- `go test ./internal/chat ./internal/bindings -count=1` from `desktop-go`: pass.
- `go test ./... -count=1` from `desktop-go`: pass.
- `npm run typecheck`: pass.
- `npm run native:triage`: pass, 643 commands.
- `npm run native:inventory -- --summary`: pass, 1526 scanned files / 951 native references / 858 invokes / 643 unique invokes.
- `node scripts/rust-retirement-map.mjs`: pass, merged Rust registrations reduced from 71 to 66; merged Rust definitions reduced from 93 to 88; retirement candidates reduced from 77 to 72; direct Tauri blockers remain 33 commands / 33 edges / 6 files.
- `npx gitnexus detect_changes --repo "Deep Student"` was attempted and still fails because this GitNexus CLI build does not provide `detect_changes`.
- `cargo check` was intentionally not used as this slice's verification gate. The acceptance evidence is Go/Wails parity plus frontend/native command routing and the reduced Rust retirement map.

Next retirement candidates identified by subagent scan:

- Mindmap/file/ref/index VFS batches in `src-tauri/src/vfs/handlers.rs`.
- Avoid Anki document internals such as `get_document_tasks` and `recover_stuck_document_tasks` for now because existing notes show CRITICAL/HIGH old Rust risk.

## Completed Attachment Config Rust Retirement Slice

Retired the Go/Wails-backed attachment-root settings command trio from the old Rust/Tauri command surface:

- `vfs_get_attachment_config`
- `vfs_set_attachment_root_folder`
- `vfs_create_attachment_root_folder`

What changed:

- Removed the three old Tauri command registrations from `src-tauri/src/lib.rs`.
- Removed the three old Rust command wrappers and their `AttachmentConfigOutput` DTO from `src-tauri/src/vfs/handlers.rs`.
- Kept `vfs_get_or_create_attachment_root_folder`, attachment upload/content commands, and the old Rust `AttachmentConfig` helper because still-live Rust attachment flows use them internally.
- Kept the legacy command names active through `src/runtime/wailsBridge.ts`, where they route to Go `SettingsService`.

GitNexus / impact notes:

- `vfs_get_attachment_config`, `vfs_set_attachment_root_folder`, and `vfs_create_attachment_root_folder`: LOW risk, 0 impacted callers for the old Rust command wrappers.
- `AttachmentConfig` is still used by other old Rust VFS attachment flows and was not edited or deleted.

Verification:

- `rg "crate::vfs::handlers::(vfs_get_attachment_config|vfs_set_attachment_root_folder|vfs_create_attachment_root_folder)|pub async fn (...)" src-tauri/src -g "*.rs"`: no old Rust command definitions or Tauri registrations remain.
- `go test ./internal/settings ./internal/bindings -count=1` from `desktop-go`: pass.
- `go test ./... -count=1` from `desktop-go`: pass.
- `npm run typecheck`: pass.
- `npm run native:triage`: pass, 643 commands.
- `npm run native:inventory -- --summary`: pass, 1526 scanned files / 951 native references / 858 invokes / 643 unique invokes.
- `node scripts/rust-retirement-map.mjs`: pass, merged Rust registrations reduced from 57 to 54; merged Rust definitions reduced from 81 to 78; retirement candidates reduced from 65 to 62; `src-tauri/src/vfs/handlers.rs` merged Go commands still present in Rust reduced from 41 to 38.
- `git diff --check` on this slice's touched files and regenerated native docs: pass.
- `npx gitnexus detect_changes --repo "Deep Student"` was attempted and still fails because this GitNexus CLI build does not provide `detect_changes`.
- `cargo check` was intentionally not used as this slice's verification gate. The acceptance evidence is Go/Wails settings parity plus frontend/native command routing and the reduced Rust retirement map.

## Completed VFS Mindmap Rust Retirement Slice

Retired the Go/Wails-backed mindmap command batch from the old Rust/Tauri VFS command surface:

- `vfs_create_mindmap`
- `vfs_get_mindmap`
- `vfs_get_mindmap_content`
- `vfs_get_mindmap_versions`
- `vfs_get_mindmap_version`
- `vfs_get_mindmap_version_content`
- `vfs_update_mindmap`
- `vfs_delete_mindmap`
- `vfs_list_mindmaps`
- `vfs_set_mindmap_favorite`

What changed:

- Removed the old Tauri command registrations from `src-tauri/src/lib.rs`.
- Removed the old Rust command wrappers from `src-tauri/src/vfs/handlers.rs`.
- Removed the now-unused `VfsMindMapRepo` import from `src-tauri/src/vfs/handlers.rs`.
- Kept `VfsMindMapRepo` itself because old Rust folder-purge internals still reference it; only the public command surface was retired.
- Kept the legacy command names active through `src/runtime/wailsBridge.ts`, where they route to Go `VfsService`.

GitNexus / impact notes:

- `vfs_create_mindmap`, `vfs_get_mindmap`, `vfs_get_mindmap_content`, `vfs_get_mindmap_versions`, `vfs_get_mindmap_version`, `vfs_get_mindmap_version_content`, `vfs_update_mindmap`, `vfs_delete_mindmap`, `vfs_list_mindmaps`, and `vfs_set_mindmap_favorite`: LOW risk, 0 impacted callers for the old Rust command wrappers.
- `VfsMindMapRepo` was not deleted because it is still used by old Rust folder deletion/purge paths until those paths are fully retired or migrated.

Verification:

- `rg "vfs_(create|get|update|delete|list|set)_mindmap|vfs_get_mindmap|vfs_list_mindmaps|vfs_set_mindmap_favorite" src-tauri/src/vfs/handlers.rs src-tauri/src/lib.rs src-tauri/src/vfs/mod.rs src-tauri/src/vfs/types.rs`: no old Rust command definitions or Tauri registrations remain.
- `go test ./internal/vfs ./internal/bindings -count=1` from `desktop-go`: pass.
- `go test ./... -count=1` from `desktop-go`: pass.
- `npm run typecheck`: pass.
- `npm run native:triage`: pass, 643 commands.
- `npm run native:inventory -- --summary`: pass, 1526 scanned files / 951 native references / 858 invokes / 643 unique invokes.
- `node scripts/rust-retirement-map.mjs`: pass, merged Rust registrations reduced from 54 to 45; merged Rust definitions reduced from 78 to 69; retirement candidates reduced from 62 to 53; direct Tauri blockers remain 33 commands / 33 edges / 6 files.
- `git diff --check -- src-tauri/src/vfs/handlers.rs src-tauri/src/lib.rs docs/generated/rust-retirement-map.md docs/generated/rust-retirement-map.json`: pass.
- `npx gitnexus detect_changes --repo "Deep Student"` was attempted and still fails because this GitNexus CLI build does not provide `detect_changes`.
- `cargo check` was intentionally not used as this slice's verification gate. The acceptance evidence is Go/Wails VFS mindmap parity plus frontend/native command routing and the reduced Rust retirement map.

## Completed VFS Resource Ref/Path Rust Retirement Slice

Retired the Go/Wails-backed resource ref/path compatibility command batch from the old Rust/Tauri VFS command surface:

- `vfs_create_or_reuse`
- `vfs_resource_exists`
- `vfs_increment_ref`
- `vfs_decrement_ref`
- `vfs_get_resource_path`
- `vfs_update_path_cache`

What changed:

- Removed the six old Tauri command registrations from `src-tauri/src/lib.rs`.
- Removed the six old Rust command wrappers from `src-tauri/src/vfs/handlers.rs`.
- Removed the private Rust path-cache helper chain that only served the deleted `vfs_get_resource_path` and `vfs_update_path_cache` wrappers.
- Kept the legacy command names active through `src/runtime/wailsBridge.ts`, where they route to Go `VfsService`.

GitNexus / impact notes:

- `vfs_create_or_reuse`, `vfs_resource_exists`, `vfs_increment_ref`, `vfs_decrement_ref`, `vfs_get_resource_path`, and `vfs_update_path_cache`: LOW risk, 0 impacted callers for the old Rust command wrappers.
- No HIGH or CRITICAL GitNexus warnings were returned for this batch.

Verification:

- `rg "vfs_create_or_reuse|vfs_resource_exists|vfs_increment_ref|vfs_decrement_ref|vfs_get_resource_path|vfs_update_path_cache" src-tauri/src/vfs/handlers.rs src-tauri/src/lib.rs src/runtime/wailsBridge.ts desktop-go/internal/vfs desktop-go/internal/bindings`: only Wails bridge compatibility routes remain for those legacy command names.
- `rg "update_path_cache_internal|compute_path_with_conn|get_resource_title_with_conn|get_active_resource_title_with_conn" src-tauri/src/vfs/handlers.rs`: no removed private helper remains.
- `go test ./internal/vfs ./internal/bindings -count=1` from `desktop-go`: pass.
- `go test ./... -count=1` from `desktop-go`: pass.
- `npm run typecheck`: pass.
- `npm run native:triage`: pass, 643 commands.
- `npm run native:inventory -- --summary`: pass, 1526 scanned files / 951 native references / 858 invokes / 643 unique invokes.
- `node scripts/rust-retirement-map.mjs`: pass, merged Rust registrations reduced from 45 to 39; merged Rust definitions reduced from 69 to 63; retirement candidates reduced from 53 to 47; `src-tauri/src/vfs/handlers.rs` merged Go commands still present in Rust reduced from 29 to 23.
- `git diff --check -- src-tauri/src/vfs/handlers.rs src-tauri/src/lib.rs docs/generated/rust-retirement-map.md docs/generated/rust-retirement-map.json docs/generated/native-command-triage.md docs/generated/native-command-triage.json`: pass.
- `npx gitnexus detect_changes --repo "Deep Student"` was attempted and still fails because this GitNexus CLI build does not provide `detect_changes`.
- `cargo check` was intentionally not used as this slice's verification gate. The acceptance evidence is Go/Wails VFS parity plus frontend/native command routing and the reduced Rust retirement map.

## Completed VFS Blob Read Rust Retirement Slice

Retired the Go/Wails-backed blob base64 read command from the old Rust/Tauri VFS command surface:

- `vfs_get_blob_base64`

What changed:

- Removed the old Tauri command registration from `src-tauri/src/lib.rs`.
- Removed the old Rust command wrapper from `src-tauri/src/vfs/handlers.rs`.
- Kept the shared `VfsBlobBase64Result` DTO because the old Rust PDF page-image command still returns it until that higher-coupling PDF surface is retired.
- Kept the legacy command name active through `src/runtime/wailsBridge.ts`, where it routes to Go `VfsService.GetBlobBase64`.

GitNexus / impact notes:

- `vfs_get_blob_base64`: LOW risk, 0 impacted callers for the old Rust command wrapper.
- Existing Go tests already cover hash/source-id blob reads and soft-deleted resource rejection in the hybrid VFS.

Verification:

- `rg "vfs_get_blob_base64" src-tauri/src/vfs/handlers.rs src-tauri/src/lib.rs src/runtime/wailsBridge.ts src/features/chat/context/blobApi.ts`: no old Rust command definition or Tauri registration remains; frontend/native facade calls and Wails bridge compatibility remain.
- `go test ./internal/vfs ./internal/bindings -count=1` from `desktop-go`: pass.
- `go test ./... -count=1` from `desktop-go`: pass.
- `npm run typecheck`: pass.
- `npm run native:triage`: pass, 643 commands.
- `npm run native:inventory -- --summary`: pass, 1526 scanned files / 951 native references / 858 invokes / 643 unique invokes.
- `node scripts/rust-retirement-map.mjs`: pass, merged Rust registrations reduced from 39 to 38; merged Rust definitions reduced from 63 to 62; retirement candidates reduced from 47 to 46; `src-tauri/src/vfs/handlers.rs` merged Go commands still present in Rust reduced from 23 to 22.
- `git diff --check -- src-tauri/src/vfs/handlers.rs src-tauri/src/lib.rs docs/generated/rust-retirement-map.md docs/generated/rust-retirement-map.json docs/generated/native-command-triage.md docs/generated/native-command-triage.json`: pass.
- `npx gitnexus detect_changes --repo "Deep Student"` was attempted and still fails because this GitNexus CLI build does not provide `detect_changes`.
- `cargo check` was intentionally not used as this slice's verification gate. The acceptance evidence is Go/Wails VFS blob-read parity plus frontend/native command routing and the reduced Rust retirement map.

## Completed VFS File CRUD/Content Rust Retirement Slice

Retired the Go/Wails-backed file CRUD/content command batch from the old Rust/Tauri VFS command surface:

- `vfs_upload_file`
- `vfs_get_file`
- `vfs_list_files`
- `vfs_delete_file`
- `vfs_get_file_content`

What changed:

- Removed the five old Tauri command registrations from `src-tauri/src/lib.rs`.
- Removed the five old Rust command wrappers and command-only file DTOs from `src-tauri/src/vfs/handlers.rs`.
- Kept `OcrStrategyConfig`, `file_delete_watch_targets`, attachment delete, attachment upload/read, and PDF processing helpers because still-live old Rust attachment/PDF surfaces use them until those surfaces are retired separately.
- Kept the legacy command names active through `src/runtime/wailsBridge.ts`, where they route to Go `VfsService`.
- Updated two source-contract tests so they no longer require retired Rust file/resource-path wrappers to exist; they now lock Go hybrid VFS service behavior for file path resolution, upload dedupe/restore, soft delete, and active-only content reads.

GitNexus / impact notes:

- `vfs_upload_file`, `vfs_get_file`, `vfs_list_files`, `vfs_delete_file`, and `vfs_get_file_content`: LOW risk, 0 impacted callers for the old Rust command wrappers.
- No HIGH or CRITICAL GitNexus warnings were returned for this batch.

Verification:

- `rg "vfs_upload_file|vfs_get_file|vfs_list_files|vfs_delete_file|vfs_get_file_content|vfs_get_resource_path|vfs_update_path_cache|VfsUploadFileParams|VfsUploadFileResult|VfsFileContentResult|pub struct IndexStatus\\b|pub struct OcrStatus\\b" src-tauri/src/vfs/handlers.rs src-tauri/src/lib.rs tests/vitest/learning-hub/vfsAttachmentDeleteContract.source.test.ts tests/vitest/vfs/fileDeleteFolderItems.source.test.ts src/runtime/wailsBridge.ts src/api/vfsFileApi.ts`: old Rust command definitions/registrations and stale source-test anchors are gone; frontend/native facade calls and Wails bridge compatibility remain.
- `go test ./internal/vfs ./internal/bindings -count=1` from `desktop-go`: pass.
- `go test ./... -count=1` from `desktop-go`: pass.
- `npm run typecheck`: pass.
- `npm run native:triage`: pass, 643 commands.
- `npm run native:inventory -- --summary`: pass, 1526 scanned files / 951 native references / 858 invokes / 643 unique invokes.
- `node scripts/rust-retirement-map.mjs`: pass, merged Rust registrations reduced from 38 to 33; merged Rust definitions reduced from 62 to 57; retirement candidates reduced from 46 to 41; `src-tauri/src/vfs/handlers.rs` merged Go commands still present in Rust reduced from 22 to 17.
- `git diff --check -- src-tauri/src/vfs/handlers.rs src-tauri/src/lib.rs tests/vitest/learning-hub/vfsAttachmentDeleteContract.source.test.ts tests/vitest/vfs/fileDeleteFolderItems.source.test.ts docs/generated/rust-retirement-map.md docs/generated/rust-retirement-map.json docs/generated/native-command-triage.md docs/generated/native-command-triage.json`: pass.
- `npm run test -- tests/vitest/learning-hub/vfsAttachmentDeleteContract.source.test.ts tests/vitest/vfs/fileDeleteFolderItems.source.test.ts` was attempted but did not reach test bodies because the install at that time was missing `@testing-library/dom`.
- `npx gitnexus detect_changes --repo "Deep Student"` was attempted and still fails because this GitNexus CLI build does not provide `detect_changes`.
- `cargo check` was intentionally not used as this slice's verification gate. The acceptance evidence is Go/Wails VFS file CRUD/content parity plus frontend/native command routing and the reduced Rust retirement map.

## Completed VFS Attachment Upload/Read Rust Retirement Slice

Retired the Go/Wails-backed attachment upload/read command batch from the old Rust/Tauri VFS command surface:

- `vfs_upload_attachment`
- `vfs_get_attachment`
- `vfs_get_attachment_content`

What changed:

- Removed the three old Tauri command registrations from `src-tauri/src/lib.rs`.
- Removed the three old Rust command wrappers, upload-only DTOs, attachment-content DTO, topic-folder resolver helpers, and now-stale helper tests from `src-tauri/src/vfs/handlers.rs`.
- Kept `vfs_get_or_create_attachment_root_folder` and `vfs_delete_attachment` in old Rust because they still have separate live transitional responsibilities.
- Kept the legacy upload/read command names active through `src/runtime/wailsBridge.ts`, where they route to Go `VfsService`.
- Updated the attachment ready-mode source contract so it now locks Go hybrid VFS `UploadAttachment` and `processingStateForAttachment` behavior instead of requiring the retired Rust upload wrapper to exist.
- Added the missing `@testing-library/dom` dev dependency required by the existing `@testing-library/react` test setup, allowing Vitest source-contract tests to collect and run again.

GitNexus / impact notes:

- `vfs_upload_attachment`, `vfs_get_attachment`, and `vfs_get_attachment_content`: LOW risk, 0 impacted callers for the old Rust command wrappers.
- `resolve_upload_target_folder_id`: HIGH in the current GitNexus index because its only direct caller is the retired old Rust `vfs_upload_attachment` wrapper and related old upload processes. The current worktree no longer has that wrapper; this slice removed only the stale Rust tests still referencing the deleted helper.
- Go `UploadAttachment` is not represented in the current GitNexus symbol index yet.

Verification:

- `rg "crate::vfs::handlers::(vfs_upload_attachment|vfs_get_attachment|vfs_get_attachment_content)|pub async fn (vfs_upload_attachment|vfs_get_attachment|vfs_get_attachment_content)|VfsUploadAttachmentParamsExt|VfsAttachmentContentResult|resolve_upload_target_folder_id|resolve_upload_topic_folder_id|folder_is_within_upload_root" src-tauri/src/vfs/handlers.rs src-tauri/src/lib.rs -S`: no old Rust command definitions, registrations, upload DTOs, or resolver helper references remain.
- `go test ./internal/vfs ./internal/bindings -count=1` from `desktop-go`: pass.
- `go test ./... -count=1` from `desktop-go`: pass.
- `npm run typecheck`: pass.
- `npm run native:triage`: pass, 643 commands.
- `npm run native:inventory -- --summary`: pass, 1526 scanned files / 951 native references / 858 invokes / 643 unique invokes.
- `node scripts/rust-retirement-map.mjs`: pass, merged Rust registrations reduced from 33 to 30; merged Rust definitions reduced from 57 to 54; retirement candidates reduced from 41 to 38; `src-tauri/src/vfs/handlers.rs` merged Go commands still present in Rust reduced from 17 to 14.
- `git diff --check -- src-tauri/src/vfs/handlers.rs tests/vitest/chat-v2/vfsAttachmentReadyModes.source.test.ts tests/vitest/learning-hub/vfsAttachmentDeleteContract.source.test.ts docs/GO_REWRITE_PROGRESS.md docs/generated/rust-retirement-map.md docs/generated/rust-retirement-map.json`: pass.
- `npm run test -- tests/vitest/chat-v2/vfsAttachmentReadyModes.source.test.ts tests/vitest/learning-hub/vfsAttachmentDeleteContract.source.test.ts`: pass, 2 files / 12 tests. This first required `npm install -D @testing-library/dom@^10.0.0 --legacy-peer-deps`; plain install failed on the repo's existing peer-resolution conflict around `@lobehub/icons` / React.
- `npm ls @testing-library/dom`: pass, `@testing-library/dom@10.4.1` installed and deduped under `@testing-library/react` / `@testing-library/user-event`.
- `npx gitnexus detect_changes --repo "Deep Student"` was attempted and still fails because this GitNexus CLI build does not provide `detect_changes`.
- `cargo check` was intentionally not used as this slice's verification gate. The acceptance evidence is Go/Wails VFS attachment upload/read parity plus frontend/native command routing and the reduced Rust retirement map.

## Completed VFS Search/Dimension/Text Chunk Rust Retirement Slice

Retired a low-coupling Go/Wails-backed VFS query command batch from the old Rust/Tauri VFS command surface:

- `vfs_list_dimensions`
- `vfs_get_resource_text_chunks`
- `vfs_rag_search`

What changed:

- Removed the three old Tauri command registrations from `src-tauri/src/lib.rs`.
- Removed the three old Rust command wrappers from `src-tauri/src/vfs/handlers.rs`.
- Removed command-only Rust DTO/default helpers for the retired RAG/text-chunk wrappers: `VfsRagSearchInput`, `VfsRagSearchOutput`, `TextChunkInfo`, `default_rag_top_k`, `default_enable_reranking`, `default_enable_cross_dimension`, and the local `default_modality` helper.
- Kept all still-live Rust PDF/OCR/indexing commands, index services, Lance helpers, dimension management mutation commands, and diagnostic commands intact.
- Kept the legacy command names active through `src/runtime/wailsBridge.ts`, where they route to Go `VfsService`.

GitNexus / impact notes:

- `vfs_list_dimensions`, `vfs_get_resource_text_chunks`, and `vfs_rag_search`: LOW risk, 0 impacted callers for the old Rust command wrappers.
- The attempted Codex subagent for this candidate scan failed with model-access 403, so this slice used local inspection plus GitNexus CLI impact.

Verification:

- `rg "crate::vfs::handlers::(vfs_list_dimensions|vfs_get_resource_text_chunks|vfs_rag_search)|pub async fn (vfs_list_dimensions|vfs_get_resource_text_chunks|vfs_rag_search)|pub struct (VfsRagSearchInput|VfsRagSearchOutput|TextChunkInfo)|default_rag_top_k|default_enable_cross_dimension|default_modality\\(" src-tauri/src/vfs/handlers.rs src-tauri/src/lib.rs -S`: no old Rust command definitions, registrations, or command-only DTO/helper anchors remain.
- `go test ./internal/vfs ./internal/bindings -count=1` from `desktop-go`: pass.
- `go test ./... -count=1` from `desktop-go`: pass.
- `npm run typecheck`: pass.
- `npm run native:triage`: pass, 643 commands.
- `npm run native:inventory -- --summary`: pass, 1526 scanned files / 951 native references / 858 invokes / 643 unique invokes.
- `npm run test -- tests/vitest/learning-hub/indexStatusStateSource.source.test.ts tests/vitest/learning-hub/indexStatusProgress.source.test.ts`: pass, 2 files / 19 tests.
- `npm run test -- tests/vitest/learning-hub/IndexStatusView.behavior.test.tsx tests/vitest/learning-hub/indexStatusStateSource.source.test.ts tests/vitest/learning-hub/indexStatusProgress.source.test.ts` was attempted after `@testing-library/dom` was installed. The two source tests passed, but one pre-existing UI behavior test failed because `IndexStatusView.behavior` rendered `indexStatus.progress.textIndexProgress undefined/undefined` instead of the test's expected `7/9`; this slice did not edit that UI or API mock path.
- `node scripts/rust-retirement-map.mjs`: pass, merged Rust registrations reduced from 30 to 27; merged Rust definitions reduced from 54 to 51; retirement candidates reduced from 38 to 35; `src-tauri/src/vfs/handlers.rs` merged Go commands still present in Rust reduced from 14 to 11.
- `git diff --check -- src-tauri/src/vfs/handlers.rs src-tauri/src/lib.rs docs/generated/rust-retirement-map.md docs/generated/rust-retirement-map.json docs/GO_REWRITE_PROGRESS.md`: pass.
- `npx gitnexus detect_changes --repo "Deep Student"` was attempted and still fails because this GitNexus CLI build does not provide `detect_changes`.
- `cargo check` was intentionally not used as this slice's verification gate. The acceptance evidence is Go/Wails VFS query parity plus frontend/native command routing and the reduced Rust retirement map.

## Completed OCR Settings Rust Retirement Slice

Retired the Go/Wails-backed OCR settings command batch from the old Rust/Tauri command surface:

- `get_available_ocr_models`
- `get_ocr_engines`
- `get_ocr_engine_type`
- `get_ocr_thinking_enabled`
- `set_ocr_thinking_enabled`
- `save_available_ocr_models`
- `update_ocr_engine_priority`
- `add_ocr_engine`
- `remove_ocr_engine`

What changed:

- Removed the nine old OCR settings Tauri command registrations from `src-tauri/src/lib.rs`.
- Removed the nine old OCR settings command wrappers and their command-only DTOs from `src-tauri/src/cmd/ocr.rs`.
- Kept `set_ocr_engine_type`, `infer_ocr_engine_from_model`, `validate_ocr_model`, `get_ocr_prompt_template`, and `test_ocr_engine` because they belong to the still-unmigrated OCR execution/test surface.
- Kept old Rust `llm_manager` OCR helpers such as `get_available_ocr_models` and `get_ocr_engine_type` because `chat_v2_perform_ocr`, PDF OCR, VLM grounding, and old OCR pipelines still call them.
- Kept the legacy settings command names active through `src/runtime/wailsBridge.ts`, where they route to Go `SettingsService`.

GitNexus / impact notes:

- `get_ocr_engines`, `get_ocr_thinking_enabled`, `set_ocr_thinking_enabled`, `save_available_ocr_models`, `update_ocr_engine_priority`, `add_ocr_engine`, and `remove_ocr_engine`: LOW risk, 0 impacted callers for the old `src-tauri/src/cmd/ocr.rs` command wrappers.
- `get_ocr_engine_type` and `get_available_ocr_models` matched CRITICAL old Rust `src-tauri/src/llm_manager/mod.rs` symbols in GitNexus. Those internal OCR execution helpers were not edited or deleted.
- Source search after deletion confirms only the Wails bridge compatibility routes and old `llm_manager` internal helpers still use those names.

Verification:

- `rg "crate::commands::(get_ocr_engines|get_ocr_engine_type|get_ocr_thinking_enabled|set_ocr_thinking_enabled|get_available_ocr_models|save_available_ocr_models|update_ocr_engine_priority|add_ocr_engine|remove_ocr_engine)|pub async fn (...)" src-tauri/src -g "*.rs"`: no old `cmd/ocr.rs` command definitions or Tauri registrations remain; only old `llm_manager` internal OCR helpers remain.
- `go test ./internal/settings ./internal/bindings -count=1` from `desktop-go`: pass.
- `go test ./... -count=1` from `desktop-go`: pass.
- `npm run typecheck`: pass.
- `npm run native:triage`: pass, 643 commands.
- `npm run native:inventory -- --summary`: pass, 1526 scanned files / 951 native references / 858 invokes / 643 unique invokes.
- `node scripts/rust-retirement-map.mjs`: pass, merged Rust registrations reduced from 66 to 57; merged Rust definitions reduced from 88 to 81; retirement candidates reduced from 72 to 65; direct Tauri blockers remain 33 commands / 33 edges / 6 files.
- `git diff --check` on this slice's touched files and regenerated native docs: pass.
- `npx gitnexus detect_changes --repo "Deep Student"` was attempted and still fails because this GitNexus CLI build does not provide `detect_changes`.
- `cargo check` was intentionally not used as this slice's verification gate. The acceptance evidence is Go/Wails settings parity plus frontend/native command routing and the reduced Rust retirement map.

## Completed OCR Settings Go Parity Facade Slice

Added Go/Wails compatibility for the OCR configuration command surface and moved the remaining settings-page OCR command callers off direct Tauri invoke:

- `get_available_ocr_models`
- `get_ocr_engines`
- `get_ocr_engine_type`
- `get_ocr_thinking_enabled`
- `set_ocr_thinking_enabled`
- `save_available_ocr_models`
- `update_ocr_engine_priority`
- `add_ocr_engine`
- `remove_ocr_engine`

What changed:

- Added lean Go `SettingsService` OCR configuration methods over `settings-go.json` for engine metadata, available OCR model persistence, priority/enabled updates, add/remove, engine type, and OCR thinking toggle.
- Added compatibility DTOs and regression tests in `desktop-go/internal/settings` instead of copying the old Rust OCR adapter/service stack.
- Exposed the new methods through Wails `SettingsService` bindings and routed the legacy command names through `src/runtime/wailsBridge.ts`.
- Added non-native/local fallback support in `src/runtime/native.ts`.
- Moved `OcrEngineCard.tsx`, `SiliconFlowSection.tsx`, and `ModelsTab.tsx` OCR command calls from `@tauri-apps/api/core` to `@/runtime/native`.

Known parity notes:

- This slice is configuration parity only. It does not implement the real Go OCR execution pipeline, PDF raster rendering, scanned-PDF OCR, or old Rust OCR adapter behavior.
- The generated retirement map now marks the old Rust OCR command wrappers as retirement candidates, but do not delete them until the OCR execution/migration risk is reviewed separately or the Rust runtime is quarantined as a whole.

GitNexus / impact notes:

- `OcrEngineCard`, `SiliconFlowSection`, `ModelsTab`, `SettingsService`, and Go settings `Service`: LOW risk in the current index.
- `invokeWails` and `fallbackInvoke` are not represented in the current GitNexus symbol index.

Verification:

- `go test ./internal/settings -count=1` from `desktop-go`: pass.
- `go test ./internal/bindings -count=1` from `desktop-go`: pass.
- `go test ./... -count=1` from `desktop-go`: pass.
- `npm run go:bindings`: pass, Wails bindings regenerated with 11 services, 298 methods, and 196 models.
- `npm run typecheck`: pass.
- `npm run native:triage`: pass, 643 commands.
- `npm run native:inventory -- --summary`: pass, 1526 scanned files / 951 native references / 858 invokes / 643 unique invokes.
- `node scripts/rust-retirement-map.mjs`: pass, merged commands with Wails bridge routes increased to 291; direct Tauri blockers reduced to 33 commands / 33 edges / 6 files; merged Rust registrations remain 71; merged Rust definitions remain 93.
- `git diff --check`: still reports only pre-existing trailing whitespace in `src/hooks/useSystemSettings.ts` lines 52 and 130, plus Git line-ending normalization notice for `src-tauri/src/vfs/ref_handlers.rs`.
- `npx gitnexus detect_changes --repo "Deep Student"` was attempted and still fails because this GitNexus CLI build does not provide `detect_changes`.

## Completed DSTU Search Wrapper Rust Retirement Slice

Retired the final Go-backed DSTU search command wrapper from the old Rust/Tauri command surface:

- `dstu_search`

What changed:

- Removed the old `dstu_search` Tauri command body from `src-tauri/src/dstu/handlers.rs`.
- Removed the matching Tauri registration from `src-tauri/src/lib.rs`.
- Removed the old re-export from `src-tauri/src/dstu/mod.rs`.
- Rewired the remaining Rust `dstu_search_in_folder` no-folder branch to call the internal `search_all` helper directly and keep the existing `__*__` hidden-system-note filtering.
- Active command compatibility remains through `src/runtime/wailsBridge.ts`, which routes the legacy `dstu_search` command name to Go `DstuService`.

GitNexus / impact notes:

- `dstu_search`: CRITICAL before the edit because `dstu_search_in_folder` was its single direct Rust caller.
- `dstu_search_in_folder`: LOW risk, 0 impacted callers before changing the internal no-folder branch.

Verification:

- `rg "\bdstu_search\b|\bdstu_search_in_folder\b" src-tauri/src -g "*.rs"`: no `dstu_search` Rust command definition, registration, or re-export remains; only `dstu_search_in_folder` remains intentionally.
- `go test ./internal/dstu -count=1` from `desktop-go`: pass.
- `go test ./... -count=1` from `desktop-go`: pass.
- `npm run typecheck`: pass.
- `npm run native:triage`: pass, 643 commands.
- `npm run native:inventory -- --summary`: pass, 1526 scanned files / 951 native references / 858 invokes / 643 unique invokes.
- `node scripts/rust-retirement-map.mjs`: pass, merged Rust registrations reduced to 71; merged Rust definitions reduced to 93; retirement candidates reduced to 68; direct Tauri blockers remain 42 commands / 43 edges / 9 files.
- `git diff --check`: still reports only pre-existing trailing whitespace in `src/hooks/useSystemSettings.ts` lines 52 and 130, plus Git line-ending normalization notice for `src-tauri/src/vfs/ref_handlers.rs`.
- `npx gitnexus detect_changes --repo "Deep Student"` was attempted and still fails because this GitNexus CLI build does not provide `detect_changes`.

## Completed DSTU Main CRUD Rust Retirement Slice

Retired the Go-backed DSTU main CRUD command batch from the old Rust/Tauri command surface:

- `dstu_create`
- `dstu_delete`
- `dstu_delete_many`
- `dstu_get`
- `dstu_get_content`
- `dstu_list`
- `dstu_list_deleted`
- `dstu_purge`
- `dstu_purge_all`
- `dstu_restore`
- `dstu_restore_many`
- `dstu_set_favorite`
- `dstu_set_metadata`
- `dstu_update`

What changed:

- Removed the old command bodies from `src-tauri/src/dstu/handlers.rs`.
- Removed the matching Tauri registrations from `src-tauri/src/lib.rs`.
- Removed the old command re-exports from `src-tauri/src/dstu/mod.rs`.
- Deleted now-dead `src-tauri/src/dstu/folder_handlers.rs` and `src-tauri/src/dstu/trash_handlers.rs` after the remaining command wrappers that depended on their helpers were retired.
- Kept `dstu_search` intentionally because `dstu_search_in_folder` still calls it directly; retire this only after splitting `dstu_search_in_folder` onto the shared `search_all` path.
- Active command compatibility remains through `src/runtime/wailsBridge.ts`, which routes these legacy command names to Go `DstuService`.

GitNexus / impact notes:

- `dstu_create`, `dstu_update`, `dstu_delete`, `dstu_get`, `dstu_list`, `dstu_delete_many`, `dstu_get_content`, `dstu_list_deleted`, `dstu_purge`, `dstu_purge_all`, `dstu_restore`, `dstu_restore_many`, `dstu_set_favorite`, and `dstu_set_metadata`: LOW risk, 0 impacted callers for the old Rust command wrappers.
- `dstu_search`: CRITICAL because `dstu_search_in_folder` is a direct Rust caller; it was not edited or deleted in this slice.
- `dstu_list_folder_first`: LOW risk and only retained by the retired `dstu_list` wrapper, so it was removed with the wrapper.
- `log_and_skip_err` impact matched a different same-name symbol in `question_sync_service.rs`; the local DSTU helper was dead after the wrapper removal.

Verification:

- `rg "\bdstu_(create|delete_many|delete|get_content|get|list_deleted|list|purge_all|purge|restore_many|restore|set_favorite|set_metadata|update)\b|folder_handlers|trash_handlers" src-tauri/src -g "*.rs"`: no live Rust command definitions/registrations remain; only comments and old command-name prose references remain.
- `go test ./internal/dstu -count=1` from `desktop-go`: pass.
- `go test ./... -count=1` from `desktop-go`: pass.
- `npm run typecheck`: pass.
- `cargo check` from `src-tauri`: pass with existing warnings.
- `npm run native:triage`: pass, 643 commands.
- `npm run native:inventory -- --summary`: pass, 1526 scanned files / 951 native references / 858 invokes / 643 unique invokes.
- `node scripts/rust-retirement-map.mjs`: pass, merged Rust registrations reduced to 72; merged Rust definitions reduced to 94; retirement candidates reduced to 69; direct Tauri blockers remain 42 commands / 43 edges / 9 files.
- `git diff --check`: still reports only pre-existing trailing whitespace in `src/hooks/useSystemSettings.ts` lines 52 and 130, plus Git line-ending normalization notice for `src-tauri/src/vfs/ref_handlers.rs`.
- `npx gitnexus detect_changes --repo "Deep Student"` was attempted and still fails because this GitNexus CLI build does not provide `detect_changes`.

## Completed Qbank/CSV Rust Command Retirement Slice

Retired the old Rust/Tauri qbank practice and CSV command wrappers now that Go `QbankService` owns these native routes:

- `qbank_list_questions`
- `qbank_search_questions`
- `qbank_rebuild_fts_index`
- `qbank_get_question`
- `qbank_update_question`
- `qbank_delete_question`
- `qbank_batch_delete_questions`
- `qbank_submit_answer`
- `qbank_toggle_favorite`
- `qbank_get_stats`
- `qbank_refresh_stats`
- `qbank_get_history`
- `qbank_reset_progress`
- `qbank_reset_questions_progress`
- `qbank_get_learning_trend`
- `qbank_get_activity_heatmap`
- `qbank_get_knowledge_stats_with_comparison`
- `qbank_start_timed_practice`
- `qbank_generate_mock_exam`
- `qbank_submit_mock_exam`
- `qbank_get_daily_practice`
- `qbank_generate_paper`
- `qbank_get_check_in_calendar`
- `get_csv_preview`
- `import_questions_csv`
- `export_questions_csv`
- `get_csv_exportable_fields`

What changed:

- Removed the old Rust command bodies from `src-tauri/src/commands.rs`.
- Removed the matching Tauri handler registrations from `src-tauri/src/lib.rs`.
- Removed the now-unused Rust CSV command request DTOs and qbank helper/request fragments that only supported the retired wrappers.
- Kept adjacent old Rust qbank entrances that were not part of this proven Go retirement batch: `qbank_get_question_by_card_id`, `qbank_create_question`, `qbank_batch_create_questions`, `qbank_batch_update_questions`, `qbank_get_submissions`, and `qbank_get_knowledge_stats`.
- Kept the old Rust `QuestionBankService` internals because unfinished Rust chat/qbank tooling still uses that service directly; this slice retires the external Tauri command surface, not every old internal implementation yet.
- Active command compatibility remains through `src/runtime/wailsBridge.ts`, which routes these command names to Go `QbankService`.

Known parity notes:

- Current qbank/CSV UI callers use `@/runtime/native`; a targeted search found no direct `@tauri-apps/api/core` imports in the inspected qbank/CSV caller set.
- Go CSV import/export is local-file oriented. The old Rust CSV wrapper had extra virtual-URI materialization/staging behavior. Current UI paths provide local file paths, so this is acceptable for the retired command surface, but virtual CSV export compatibility remains a known Go-side gap if a future caller sends a virtual URI.

GitNexus / impact notes:

- `get_csv_preview`, `import_questions_csv`, `export_questions_csv`, and `get_csv_exportable_fields`: LOW risk, 0 impacted callers for the old Rust command wrappers.
- `qbank_list_questions`, `qbank_search_questions`, `qbank_rebuild_fts_index`, `qbank_get_question`, `qbank_update_question`, `qbank_delete_question`, `qbank_batch_delete_questions`, and `qbank_submit_answer`: LOW risk, 0 impacted callers for the old Rust command wrappers.
- `qbank_toggle_favorite`, `qbank_get_stats`, `qbank_refresh_stats`, `qbank_get_history`, `qbank_reset_progress`, `qbank_reset_questions_progress`, `qbank_get_learning_trend`, and `qbank_get_activity_heatmap`: LOW risk, 0 impacted callers for the old Rust command wrappers.
- `qbank_get_knowledge_stats_with_comparison`, `qbank_start_timed_practice`, `qbank_generate_mock_exam`, `qbank_submit_mock_exam`, `qbank_get_daily_practice`, `qbank_generate_paper`, and `qbank_get_check_in_calendar`: LOW risk, 0 impacted callers for the old Rust command wrappers.
- The private Rust helpers `default_page` and `default_page_size` also returned LOW risk with 0 impacted callers before removal.

Verification:

- `rg` over `src-tauri/src/commands.rs` and `src-tauri/src/lib.rs`: no retired qbank/CSV command definitions or Tauri registrations remain; the retained adjacent Rust `qbank_get_question_by_card_id` entry still exists intentionally.
- `rg` over `src/runtime/wailsBridge.ts`, `desktop-go/internal/bindings/qbank_service.go`, and `desktop-go/internal/qbank`: Wails/Go bridge routes remain for the retired command names.
- `go test ./internal/qbank -count=1` from `desktop-go`: pass.
- `go test ./... -count=1` from `desktop-go`: pass.
- `npm run typecheck`: pass.
- `cargo check` from `src-tauri`: pass in 1m09s with 199 existing warnings after removing the new unused import from this slice.
- `npm run native:triage`: pass, 643 commands.
- `npm run native:inventory -- --summary`: pass, 1526 scanned files / 951 native references / 858 invokes / 643 unique invokes.
- `node scripts/rust-retirement-map.mjs`: pass, merged Rust registrations reduced from 121 to 95; merged Rust definitions reduced from 143 to 117; retirement candidates reduced from 118 to 92; direct Tauri blockers remain 42 commands / 43 edges / 9 files.
- `git diff --check`: still reports only pre-existing trailing whitespace in `src/hooks/useSystemSettings.ts` lines 52 and 130, plus the existing line-ending warning for `src-tauri/src/vfs/ref_handlers.rs`.
- `npx gitnexus detect_changes --repo "Deep Student"` was attempted and still fails because this GitNexus CLI build does not provide `detect_changes`.

## Completed Model Config Rust Command Retirement Slice

Retired the old Rust/Tauri model/API configuration command wrappers now that Go `SettingsService` owns these native routes:

- `get_api_configurations`
- `save_api_configurations`
- `get_vendor_configs`
- `save_vendor_configs`
- `get_model_profiles`
- `save_model_profiles`
- `get_model_assignments`
- `save_model_assignments`

What changed:

- Removed the eight old Rust command bodies from `src-tauri/src/commands.rs`.
- Removed the matching Tauri handler registrations from `src-tauri/src/lib.rs`.
- Kept the old Rust `llm_manager` methods with the same names because unfinished Rust LLM/chat internals still call them directly; this slice only retires the external Tauri command surface.
- Moved residual settings-page model/API config reads through `src/runtime/native.ts` / `nativeInvoke` so Wails/Go remains the effective path for the migrated command names.
- Active command compatibility remains through `src/runtime/wailsBridge.ts`, which routes these command names to Go `SettingsService`.

GitNexus / impact notes:

- `npx gitnexus impact --repo "Deep Student" --direction upstream useSettingsConfig`: LOW risk, direct caller `Settings`.
- `npx gitnexus impact --repo "Deep Student" --direction upstream Settings` matched an indexed Go app `Settings` field instead of the React component; an exact function-id target for `src/features/settings/components/Settings.tsx:Settings` was not found in the current index.
- `npx gitnexus impact --repo "Deep Student" --direction upstream get_api_configurations`: LOW risk, 0 impacted callers for the old Rust command wrapper.
- `save_api_configurations`, `get_vendor_configs`, `save_vendor_configs`, `get_model_profiles`, `save_model_profiles`, `get_model_assignments`, and `save_model_assignments` each returned LOW risk with 0 impacted callers for the old Rust command wrappers.

Verification:

- `rg "pub async fn (get_api_configurations|save_api_configurations|get_vendor_configs|save_vendor_configs|get_model_profiles|save_model_profiles|get_model_assignments|save_model_assignments)|crate::commands::(get_api_configurations|save_api_configurations|get_vendor_configs|save_vendor_configs|get_model_profiles|save_model_profiles|get_model_assignments|save_model_assignments)" src-tauri/src/commands.rs src-tauri/src/lib.rs`: no matches.
- `rg "command === '(get_api_configurations|save_api_configurations|get_vendor_configs|save_vendor_configs|get_model_profiles|save_model_profiles|get_model_assignments|save_model_assignments)'|SettingsService\\.(GetAPIConfigurations|SaveAPIConfigurations|GetVendorConfigs|SaveVendorConfigs|GetModelProfiles|SaveModelProfiles|GetModelAssignments|SaveModelAssignments)" src/runtime/wailsBridge.ts desktop-go/internal/settings desktop-go/internal/bindings/settings_service.go`: Wails/Go bridge routes remain.
- `go test ./internal/settings -count=1` from `desktop-go`: pass.
- `go test ./... -count=1` from `desktop-go`: pass.
- `npm run typecheck`: pass.
- `cargo check` from `src-tauri`: pass in 1m15s with existing warnings.
- `npm run native:triage`: pass, 643 commands.
- `npm run native:inventory -- --summary`: pass, 1526 scanned files / 951 native references / 858 invokes / 643 unique invokes.
- `node scripts/rust-retirement-map.mjs`: pass, merged Rust registrations reduced from 129 to 121; merged Rust definitions reduced from 144 to 143; retirement candidates reduced from 119 to 118; direct Tauri blockers remain 42 commands / 43 edges / 9 files.
- `git diff --check`: still reports only pre-existing trailing whitespace in `src/hooks/useSystemSettings.ts` lines 52 and 130, plus the existing line-ending warning for `src-tauri/src/vfs/ref_handlers.rs`.
- `npx gitnexus detect_changes --repo "Deep Student"` was attempted and still fails because this GitNexus CLI build does not provide `detect_changes`.

## Completed Enhanced Anki Rust Command Retirement Slice

Retired the old Rust/Tauri document-task command wrappers now that the Go `AnkiService` owns these native routes:

- `start_enhanced_document_processing`
- `pause_document_processing`
- `resume_document_processing`
- `get_document_processing_state`
- `get_document_task_counts`
- `trigger_task_processing`
- `get_document_tasks`
- `delete_document_session`
- `get_document_cards`
- `recover_stuck_document_tasks`

What changed:

- Removed the old Rust command bodies from `src-tauri/src/cmd/enhanced_anki.rs`.
- Removed the matching Tauri handler registrations from `src-tauri/src/lib.rs`.
- Kept the still-live Rust Anki card/library/export commands in `src-tauri/src/cmd/enhanced_anki.rs`: `get_task_cards`, `update_anki_card`, `delete_anki_card`, `delete_document_task`, `export_apkg_for_selection`, `list_anki_library_cards`, `export_anki_cards`, `list_document_sessions`, and `get_anki_stats`.
- Kept `src-tauri/src/enhanced_anki_service.rs` because old Rust chat Anki tooling still directly uses `EnhancedAnkiService`; that service must be deleted only after chat Anki itself is rebuilt in Go.
- Active command compatibility remains through `src/runtime/wailsBridge.ts`: the retired document start/status/control/card/recovery command names call Go `AnkiService`.

GitNexus / impact notes:

- `npx gitnexus impact --repo "Deep Student" --direction upstream start_enhanced_document_processing`: LOW risk, 0 impacted callers for the old command wrapper.
- `npx gitnexus impact --repo "Deep Student" --direction upstream get_document_processing_state`: LOW risk, 0 impacted callers for the old command wrapper.
- `npx gitnexus impact --repo "Deep Student" --direction upstream get_document_cards`: LOW risk, 0 impacted callers for the old command wrapper.
- `pause_document_processing`, `resume_document_processing`, `get_document_task_counts`, `trigger_task_processing`, and `delete_document_session` matched the old Rust service methods rather than only the command wrappers; the returned impact was LOW and showed the command wrappers as direct callers.
- `get_document_tasks` matched `src-tauri/src/enhanced_anki_service.rs:get_document_tasks` and returned CRITICAL risk because old Rust service/chat paths still use it. This slice did not edit that service symbol; it only removed the Go-backed Tauri command wrapper.
- `recover_stuck_document_tasks` matched the old database helper and returned HIGH risk. This slice did not edit the database helper; it only removed the Go-backed Tauri command wrapper.

Verification:

- `rg "start_enhanced_document_processing|pause_document_processing|resume_document_processing|get_document_processing_state|get_document_task_counts|trigger_task_processing|get_document_tasks|delete_document_session|get_document_cards|recover_stuck_document_tasks" src-tauri/src/cmd/enhanced_anki.rs src-tauri/src/lib.rs src-tauri/src/commands.rs`: no command-wrapper/registration matches remain; only the separate startup database recovery helper in `src-tauri/src/lib.rs` still contains `recover_stuck_document_tasks`.
- `go test ./internal/anki -count=1` from `desktop-go`: pass.
- `go test ./... -count=1` from `desktop-go`: pass.
- `npm run typecheck`: pass.
- `cargo check` from `src-tauri`: pass in 2m01s with existing warnings.
- `npm run native:triage`: pass, 643 commands.
- `npm run native:inventory -- --summary`: pass, 1526 scanned files / 951 native references / 858 invokes / 643 unique invokes.
- `node scripts/rust-retirement-map.mjs`: pass, merged Rust registrations reduced from 137 to 129; merged Rust definitions reduced from 147 to 144; retirement candidates reduced from 122 to 119; direct Tauri blockers remain 42 commands / 43 edges / 9 files.
- `cargo fmt --check` is still blocked by pre-existing formatting issues in `src-tauri/src/vfs/ref_handlers.rs` and trailing whitespace in `src-tauri/src/translation/pipeline.rs`.
- `git diff --check`: still reports only pre-existing trailing whitespace in `src/hooks/useSystemSettings.ts` lines 52 and 130, plus the existing line-ending warning for `src-tauri/src/vfs/ref_handlers.rs`.
- `npx gitnexus detect_changes --repo "Deep Student"` was attempted and still fails because this GitNexus CLI build does not provide `detect_changes`.

## Completed File/System Utility Rust Command Retirement Slice

Retired 4 additional Go-backed utility command wrappers from the old Rust/Tauri command surface:

- `get_file_size`
- `copy_file`
- `open_logs_folder`
- `report_frontend_log`

What changed:

- Removed old Rust `get_file_size` and `copy_file` command bodies from `src-tauri/src/commands.rs`.
- Removed old Rust `open_logs_folder` and `report_frontend_log` command bodies from `src-tauri/src/commands.rs`.
- Removed the now-unused old Rust `FrontendLogPayload` DTO and helper implementation from `src-tauri/src/commands.rs`.
- Removed the matching command registrations from `src-tauri/src/lib.rs`.
- Kept `read_file_bytes` registered and defined in old Rust for now because GitNexus shows still-live Rust textbook adoption paths directly call it.
- Active command compatibility remains through `src/runtime/wailsBridge.ts`: file-size/copy commands call Go `FileService`, and logs/frontend-log commands call Go `SystemService`.

GitNexus / impact notes:

- `npx gitnexus impact --repo "Deep Student" --direction upstream open_logs_folder`: LOW risk, 0 impacted callers.
- `npx gitnexus impact --repo "Deep Student" --direction upstream report_frontend_log`: LOW risk, 0 impacted callers.
- `npx gitnexus impact --repo "Deep Student" --direction upstream FrontendLogPayload`: LOW risk, 0 impacted callers.
- `npx gitnexus impact --repo "Deep Student" --direction upstream read_file_bytes`: CRITICAL, direct old Rust callers include `textbooks_add`/`textbooks_adopt`; `read_file_bytes` was intentionally left in place for this slice.
- `npx gitnexus context --repo "Deep Student" get_file_size` and `copy_file` reported ambiguous symbols between command wrappers and `unified_file_manager` helpers. The plain `impact` command matched the helper symbols, so source search and GitNexus Cypher by `filePath = 'src-tauri/src/commands.rs'` were used to scope the command wrappers.
- `rg "(read_file_bytes|copy_file|get_file_size)\\(" src-tauri/src -g "*.rs"` confirmed old Rust internal callers use `unified_file_manager::{copy_file,get_file_size}` directly, while `read_file_bytes` is still called by the Rust textbook path.

Verification:

- `rg "crate::commands::(get_file_size|copy_file|open_logs_folder|report_frontend_log)|pub async fn (get_file_size|copy_file|open_logs_folder|report_frontend_log)|FrontendLogPayload" src-tauri/src -g "*.rs"`: no matches.
- `rg "command === '(get_file_size|copy_file|open_logs_folder|report_frontend_log)'|FileService\\.(GetFileSize|CopyFile)|SystemService\\.(OpenLogsFolder|ReportFrontendLog)" src/runtime/wailsBridge.ts src/runtime/native.ts`: Wails/Go bridge routes remain.
- `go test ./internal/files ./internal/system -count=1` from `desktop-go`: pass.
- `go test ./... -count=1` from `desktop-go`: pass.
- `npm run typecheck`: pass.
- `cargo check` from `src-tauri`: pass in 2m40s with 200 existing warnings.
- `npm run native:triage`: pass, 643 commands.
- `npm run native:inventory -- --summary`: pass, 1526 scanned files / 951 native references / 858 invokes / 643 unique invokes.
- `node scripts/rust-retirement-map.mjs`: pass, merged Rust registrations reduced from 141 to 137; merged Rust definitions reduced from 149 to 147; retirement candidates reduced from 124 to 122; direct Tauri blockers remain 42 commands / 43 edges / 9 files.
- `git diff --check`: still reports only pre-existing trailing whitespace in `src/hooks/useSystemSettings.ts` lines 52 and 130, plus Git line-ending normalization notice for `src-tauri/src/vfs/ref_handlers.rs`.
- `npx gitnexus detect_changes --repo "Deep Student"` was attempted and still fails because this GitNexus CLI build does not provide `detect_changes`.

## Completed Qbank AI Grading Rust Module Retirement Slice

Retired the old Rust/Tauri Qbank AI grading module now that the Go `QbankService` owns the native route:

- `qbank_ai_grade`
- `qbank_cancel_grading`

What changed:

- Removed `pub mod qbank_grading` from `src-tauri/src/lib.rs`.
- Removed the old `crate::qbank_grading::qbank_ai_grade` and `crate::qbank_grading::qbank_cancel_grading` Tauri handler registrations from `src-tauri/src/lib.rs`.
- Deleted the old Rust module files under `src-tauri/src/qbank_grading/`: `mod.rs`, `events.rs`, `pipeline.rs`, and `types.rs`.
- Active command compatibility remains through `src/runtime/wailsBridge.ts`: `qbank_ai_grade` calls Go `QbankService.AIGrade`, and `qbank_cancel_grading` calls Go `QbankService.CancelGrading`.
- The old Rust chat qbank tool still contains `qbank_ai_grade` as a tool-name string, but it only returns a UI hint and does not call the removed Rust grading pipeline.

GitNexus / impact notes:

- `npx gitnexus impact --repo "Deep Student" --direction upstream qbank_ai_grade`: LOW risk, 0 impacted callers.
- `npx gitnexus impact --repo "Deep Student" --direction upstream qbank_cancel_grading`: LOW risk, 0 impacted callers.
- `npx gitnexus impact --repo "Deep Student" --direction upstream run_qbank_grading`: LOW risk, only direct caller was the removed `qbank_ai_grade` wrapper.
- `npx gitnexus impact --repo "Deep Student" --direction upstream QbankGradingEmitter`: LOW risk, 0 impacted callers.
- `npx gitnexus impact --repo "Deep Student" --direction upstream QbankGradingRequest`: LOW risk, 0 impacted callers.
- `npx gitnexus impact --repo "Deep Student" --direction upstream QbankGradingResponse`: LOW risk, only intra-module callers through the removed pipeline/wrapper.

Verification:

- `rg "qbank_grading|qbank_ai_grade|qbank_cancel_grading|QbankGrading" src-tauri/src src/runtime/wailsBridge.ts src/runtime/native.ts desktop-go -g "*.rs" -g "*.ts" -g "*.go"`: no old Rust `qbank_grading` module or handler registration remains; Wails bridge and Go `QbankService` routes remain; Rust chat tool only has the non-calling tool-name hint.
- `go test ./internal/qbank -count=1` from `desktop-go`: pass.
- `go test ./... -count=1` from `desktop-go`: pass.
- `npm run typecheck`: pass.
- `cargo check` from `src-tauri`: pass in 2m39s with 200 existing warnings.
- `npm run native:triage`: pass, 643 commands.
- `npm run native:inventory -- --summary`: pass, 1526 scanned files / 951 native references / 858 invokes / 643 unique invokes.
- `node scripts/rust-retirement-map.mjs`: pass, merged Rust registrations reduced from 143 to 141; merged Rust definitions reduced from 151 to 149; retirement candidates reduced from 126 to 124; direct Tauri blockers remain 42 commands / 43 edges / 9 files.
- `git diff --check`: still reports only pre-existing trailing whitespace in `src/hooks/useSystemSettings.ts` lines 52 and 130, plus Git line-ending normalization notice for `src-tauri/src/vfs/ref_handlers.rs`.
- `npx gitnexus detect_changes --repo "Deep Student"` was attempted and still fails because this GitNexus CLI build does not provide `detect_changes`.

## Completed Settings/System/Image Utility Rust Command Retirement Slice

Retired 6 additional Go-backed command wrappers from the old Rust/Tauri command surface:

- `save_setting`
- `get_setting`
- `delete_setting`
- `get_app_data_dir`
- `ensure_debug_log_dir`
- `get_image_as_base64`

What changed:

- Removed old Rust `save_setting`, `get_setting`, and `delete_setting` command bodies from `src-tauri/src/cmd/web_search.rs`.
- Removed old Rust `get_app_data_dir`, `ensure_debug_log_dir`, and `get_image_as_base64` command bodies from `src-tauri/src/commands.rs`.
- Removed the matching command registrations from `src-tauri/src/lib.rs`.
- Kept the still-needed Rust internals such as `Database::{save_setting,get_setting,delete_setting}`, `FileManager::{get_app_data_dir,get_image_as_base64}`, and `debug_log_service::ensure_debug_log_dir`; those are not command entrypoints and are still used by old Rust modules that have not been fully retired yet.
- Active command compatibility remains through `src/runtime/wailsBridge.ts`: settings commands call Go `SettingsService`; app-data/debug-dir commands call Go `SystemService`; image base64 reads call Go `NotesService`.

GitNexus / impact notes:

- `npx gitnexus context --repo "Deep Student" save_setting|get_setting|delete_setting` reported ambiguous symbols; exact command wrappers existed in `src-tauri/src/cmd/web_search.rs`, while high-risk database methods with the same names remain in `src-tauri/src/database/mod.rs`.
- `npx gitnexus impact` in this CLI build does not accept the UID strings reported by `context`; it falls back to same-name database symbols for `save_setting`/`get_setting`, so those CRITICAL impact results are not the edited wrapper symbols.
- Used GitNexus Cypher fallback to scope the edited wrappers by `filePath`; the `web_search.rs` setting wrappers had only same-file `DEFINES` inbound edges and database outbound calls.
- Used GitNexus context/Cypher plus source search for `get_app_data_dir`, `ensure_debug_log_dir`, and `get_image_as_base64`; the command wrappers were distinguished from still-live Rust internal helpers with the same names.
- `docs/generated/rust-retirement-map.json` now shows `rustRegistered: false` for `save_setting`, `delete_setting`, `get_app_data_dir`, `ensure_debug_log_dir`, and `get_image_as_base64`. It still marks `get_setting` as `rustRegistered: true`, but that is a false positive from non-command database reads in `src-tauri/src/lib.rs`; `rg` confirms no `crate::commands::get_setting` handler registration remains.

Verification:

- `rg "crate::commands::(save_setting|get_setting|delete_setting|get_app_data_dir|ensure_debug_log_dir|get_image_as_base64)|pub async fn (save_setting|get_setting|delete_setting|get_app_data_dir|ensure_debug_log_dir|get_image_as_base64)" src-tauri/src -g "*.rs"`: no old Rust command registrations or command definitions remain; only the non-command `FileManager::get_image_as_base64` method remains.
- `rg "command === '(save_setting|get_setting|delete_setting|get_app_data_dir|ensure_debug_log_dir|get_image_as_base64)'|SettingsService\\.(SaveSetting|GetSetting|DeleteSetting)|SystemService\\.(AppDataDir|EnsureDebugLogDir)|NotesService\\.GetImageAsBase64" src/runtime/wailsBridge.ts src/runtime/native.ts`: Wails/Go bridge routes remain.
- `go test ./internal/system ./internal/notes ./internal/settings -count=1` from `desktop-go`: pass.
- `go test ./... -count=1` from `desktop-go`: pass.
- `npm run typecheck`: pass.
- `cargo check` from `src-tauri`: pass in 2m06s with 200 existing warnings.
- `npm run native:triage`: pass, 643 commands.
- `npm run native:inventory -- --summary`: pass, 1526 scanned files / 951 native references / 858 invokes / 643 unique invokes.
- `node scripts/rust-retirement-map.mjs`: pass, merged Rust registrations reduced from 148 to 143; merged Rust definitions remain 151; retirement candidates remain 126; direct Tauri blockers remain 42 commands / 43 edges / 9 files. Note: the map still counts some same-name internal Rust helpers as retirement candidates because they are not command wrappers.
- `git diff --check`: still reports only pre-existing trailing whitespace in `src/hooks/useSystemSettings.ts` lines 52 and 130, plus Git line-ending normalization notice for `src-tauri/src/vfs/ref_handlers.rs`.
- `npx gitnexus detect_changes --repo "Deep Student"` was attempted and still fails because this GitNexus CLI build does not provide `detect_changes`.

## Completed Notes Utility/Assets Rust Retirement Slice

Retired the Go-backed Notes utility command batch from old Rust:

- `canvas_note_append`
- `canvas_note_read`
- `canvas_note_replace`
- `canvas_note_set`
- `notes_assets_bulk_delete`
- `notes_assets_index_scan`
- `notes_assets_scan_orphans`
- `notes_db_stats`
- `notes_db_vacuum`
- `notes_delete_asset`
- `notes_empty_trash`
- `notes_export`
- `notes_export_single`
- `notes_get_pref`
- `notes_hard_delete`
- `notes_import`
- `notes_import_markdown`
- `notes_import_markdown_batch`
- `notes_list_assets`
- `notes_list_deleted`
- `notes_list_tags`
- `notes_mentions_search`
- `notes_resolve_asset_path`
- `notes_restore`
- `notes_save_asset`
- `notes_search`
- `notes_set_pref`

What changed:

- Removed the 27 old Notes utility command registrations from `src-tauri/src/lib.rs`.
- Rewrote `src-tauri/src/cmd/notes.rs` down to the still-live old Rust CRUD/RAG-config surface: `notes_list`, `notes_list_meta`, `notes_create`, `notes_update`, `notes_set_favorite`, `notes_get`, `notes_delete`, `notes_list_advanced`, `rag_rebuild_fts_index`, `notes_rag_rebuild_fts_index`, `notes_get_subject_rag_config`, and `notes_update_subject_rag_config`.
- Deleted old Rust helper code and DTOs that were only needed by the retired assets, trash, import/export, preferences, Canvas, search, and mention-search commands.
- Active command compatibility remains through `src/runtime/wailsBridge.ts`: Notes asset/pref/import/export/DB commands call Go `NotesService`; search/tag/trash/Canvas/import-markdown commands call Go `DstuService`/`QbankService` as appropriate.

Verification:

- GitNexus impact was run for all 27 retired command symbols; every command returned LOW risk with 0 impacted callers.
- `rg "pub async fn (canvas_note_append|canvas_note_read|canvas_note_replace|canvas_note_set|notes_assets_bulk_delete|notes_assets_index_scan|notes_assets_scan_orphans|notes_db_stats|notes_db_vacuum|notes_delete_asset|notes_empty_trash|notes_export|notes_export_single|notes_get_pref|notes_hard_delete|notes_import|notes_import_markdown|notes_import_markdown_batch|notes_list_assets|notes_list_deleted|notes_list_tags|notes_mentions_search|notes_resolve_asset_path|notes_restore|notes_save_asset|notes_search|notes_set_pref)" src-tauri/src -g "*.rs"`: no matches.
- `go test ./internal/notes -count=1` from `desktop-go`: pass.
- `go test ./internal/dstu ./internal/qbank -count=1` from `desktop-go`: pass.
- `go test ./... -count=1` from `desktop-go`: pass.
- `npm run typecheck`: pass.
- `cargo check` from `src-tauri`: pass in 2m18s with 200 existing warnings.
- `npm run native:triage`: pass, 643 commands.
- `npm run native:inventory -- --summary`: pass, 1526 scanned files / 951 native references / 858 invokes / 643 unique invokes.
- `node scripts/rust-retirement-map.mjs`: pass, merged Rust registrations reduced from 175 to 148; merged Rust definitions reduced from 178 to 151; retirement candidates reduced from 153 to 126; direct Tauri blockers unchanged at 42 commands / 43 edges / 9 files.
- `git diff --check`: still reports only pre-existing trailing whitespace in `src/hooks/useSystemSettings.ts` lines 52 and 130, plus Git line-ending normalization notice for `src-tauri/src/vfs/ref_handlers.rs`.
- `npx gitnexus detect_changes --repo "Deep Student"` was attempted and still fails because this GitNexus CLI build does not provide `detect_changes`.

## Completed Textbook Add/Bookmarks Rust Retirement Slice

Retired the Go-backed textbook import/bookmark command pair from old Rust:

- `textbooks_add`
- `textbooks_update_bookmarks`

What changed:

- Removed both command registrations from `src-tauri/src/lib.rs`.
- Deleted the old Rust `textbooks_add` command body, the old Rust `textbooks_update_bookmarks` command body, and the now-unused `TextbookImportProgress` DTO from `src-tauri/src/cmd/textbooks.rs`.
- Preserved the remaining Rust textbook commands and shared helpers in `src-tauri/src/cmd/textbooks.rs` because commands such as list/remove/adopt/recover/purge/progress/favorite/page-count still need separate Go parity or retirement proof before deleting the whole file.
- Active command compatibility remains through `src/runtime/wailsBridge.ts`: `textbooks_add` calls Go `DstuService.AddTextbooks`, and `textbooks_update_bookmarks` calls Go `VfsService.UpdateBookmarks`.

Verification:

- `npx gitnexus impact --repo "Deep Student" --direction upstream textbooks_add`: LOW risk, 0 impacted callers.
- `npx gitnexus impact --repo "Deep Student" --direction upstream textbooks_update_bookmarks`: LOW risk, 0 impacted callers.
- `rg "textbooks_add|textbooks_update_bookmarks|TextbookImportProgress" src-tauri/src -g "*.rs"`: no matches.
- `go test ./internal/dstu -count=1` from `desktop-go`: pass.
- `go test ./... -count=1` from `desktop-go`: pass.
- `npm run typecheck`: pass.
- `cargo check` from `src-tauri`: pass in 2m25s with 201 existing warnings.
- `npm run native:triage`: pass, 643 commands.
- `npm run native:inventory -- --summary`: pass, 1526 scanned files / 951 native references / 858 invokes / 643 unique invokes.
- `node scripts/rust-retirement-map.mjs`: pass, merged Rust registrations reduced from 177 to 175; merged Rust definitions reduced from 180 to 178; retirement candidates reduced from 155 to 153; direct Tauri blockers unchanged at 42 commands / 43 edges / 9 files.

## Completed VFS Index Rust Dead Module Retirement Slice

Retired the Go-backed VFS compact/unified index command module from old Rust:

- `vfs_unified_index_status`
- `vfs_get_resource_units`
- `vfs_reindex_unit`
- `vfs_unified_batch_index`
- `vfs_sync_resource_units`
- `vfs_delete_resource_index`
- `vfs_list_embedding_dims`

What changed:

- Deleted `src-tauri/src/vfs/index_handlers.rs`, which contained old Tauri command definitions but was no longer registered in `src-tauri/src/lib.rs`.
- Removed `pub mod index_handlers` from `src-tauri/src/vfs/mod.rs`.
- Active compatibility routes remain in `src/runtime/native.ts` and `src/runtime/wailsBridge.ts`, dispatching to Go `VfsService` methods such as `UnifiedIndexStatus`, `GetResourceUnits`, `SyncResourceUnits`, `ReindexUnit`, `DeleteResourceIndex`, and `ListEmbeddingDims`.

Verification:

- GitNexus impact was run for all 7 retired command symbols; every command returned LOW risk with 0 impacted callers.
- GitNexus impact was also run for `BatchIndexResult`, `DeleteIndexResult`, and `EmbeddingDimInfo`; `BatchIndexResult` and `EmbeddingDimInfo` were LOW with only same-file command callers. `DeleteIndexResult` matched the still-live `src-tauri/src/vfs/index_service.rs` symbol rather than the deleted same-name command DTO, so `index_service.rs` was not edited.
- `rg "index_handlers|vfs_(delete_resource_index|get_resource_units|list_embedding_dims|reindex_unit|sync_resource_units|unified_batch_index|unified_index_status)" src-tauri/src -g "*.rs"`: no matches.
- `go test ./internal/vfs -count=1` from `desktop-go`: pass.
- `go test ./...` from `desktop-go`: pass.
- `npm run native:triage`: pass, 643 commands.
- `npm run native:inventory -- --summary`: pass, 1526 scanned files / 951 native references / 858 invokes / 643 unique invokes.
- `node scripts/rust-retirement-map.mjs`: pass, merged Rust registrations unchanged at 177 because this module was already unregistered; merged Rust definitions reduced from 187 to 180; retirement candidates reduced from 162 to 155; direct Tauri blockers unchanged at 42 commands / 43 edges / 9 files.
- `npm run typecheck`: pass.
- `cargo check` from `src-tauri`: pass in 2m22s with 201 existing warnings.
- `git diff --check`: still reports only pre-existing trailing whitespace in `src/hooks/useSystemSettings.ts` lines 52 and 130, plus Git line-ending normalization notice for `src-tauri/src/vfs/ref_handlers.rs`.
- `npx gitnexus detect_changes --repo "Deep Student"` was attempted and still fails because this GitNexus CLI build does not provide `detect_changes`.

## Completed DSTU Folder Rust Retirement Slice

Retired the Go-backed DSTU folder command batch from the old Rust/Tauri command surface:

- `dstu_folder_create`
- `dstu_folder_get`
- `dstu_folder_rename`
- `dstu_folder_delete`
- `dstu_folder_move`
- `dstu_folder_set_expanded`
- `dstu_folder_add_item`
- `dstu_folder_remove_item`
- `dstu_folder_move_item`
- `dstu_folder_list`
- `dstu_folder_get_tree`
- `dstu_folder_get_items`
- `dstu_folder_get_all_resources`
- `dstu_folder_reorder`
- `dstu_folder_reorder_items`
- `dstu_folder_get_breadcrumbs`

What changed:

- Removed the 16 old Rust `crate::dstu::folder_handlers::*` Tauri command registrations from `src-tauri/src/lib.rs`.
- Removed the old DSTU folder command re-exports and `BreadcrumbItem` export from `src-tauri/src/dstu/mod.rs`.
- Reduced `src-tauri/src/dstu/folder_handlers.rs` from the old 1100+ line command module to the shared `collect_folder_delete_watch_targets_with_conn` helper still used by remaining Rust DSTU delete paths.
- Active compatibility routes remain in `src/runtime/native.ts` and `src/runtime/wailsBridge.ts`, dispatching to Go `DstuService` folder/path methods.

Verification:

- GitNexus impact was run for all 16 retired command symbols plus command-only helpers/DTOs: `BreadcrumbItem`, `validate_string_input`, `contains_invalid_chars`, `contains_unicode_bypass_chars`, `is_valid_color`, and `collect_folder_delete_watch_targets`; every result was LOW risk.
- `rg "dstu_folder_(create|get|rename|delete|move|set_expanded|add_item|remove_item|move_item|list|get_tree|get_items|get_all_resources|reorder|reorder_items|get_breadcrumbs)|BreadcrumbItem" src-tauri/src -g "*.rs"`: no matches.
- `rg "collect_folder_delete_watch_targets|folder_handlers" src-tauri/src/dstu -g "*.rs"` confirms only the retained helper and its two remaining Rust DSTU delete call sites.
- `go test ./internal/dstu -count=1` from `desktop-go`: pass.
- `go test ./...` from `desktop-go`: pass.
- `npm run native:triage`: pass, 643 commands.
- `npm run native:inventory -- --summary`: pass, 1526 scanned files / 951 native references / 858 invokes / 643 unique invokes.
- `node scripts/rust-retirement-map.mjs`: pass, merged Rust registrations reduced from 193 to 177; merged Rust definitions from 203 to 187; retirement candidates from 178 to 162; direct Tauri blockers unchanged at 42 commands / 43 edges / 9 files.
- `npm run typecheck`: pass.
- `cargo check` from `src-tauri`: pass in 2m22s with 201 existing warnings.
- `git diff --check`: still reports only pre-existing trailing whitespace in `src/hooks/useSystemSettings.ts` lines 52 and 130, plus Git line-ending normalization notice for `src-tauri/src/vfs/ref_handlers.rs`.
- `npx gitnexus detect_changes --repo "Deep Student"` was attempted and still fails because this GitNexus CLI build does not provide `detect_changes`.

## Completed DSTU Path/Move Rust Retirement Slice

Retired the Go-backed DSTU path parsing, resource lookup, move, batch-move, and path-cache command batch from the old Rust/Tauri command surface:

- `dstu_parse_path`
- `dstu_build_path`
- `dstu_get_resource_location`
- `dstu_get_resource_by_path`
- `dstu_move_to_folder`
- `dstu_batch_move`
- `dstu_refresh_path_cache`
- `dstu_get_path_by_id`

What changed:

- Removed the old E1-E4 command bodies from `src-tauri/src/dstu/handlers.rs`.
- Removed the matching Tauri registrations from `src-tauri/src/lib.rs`.
- Removed the obsolete DSTU command DTO re-exports from `src-tauri/src/dstu/mod.rs`.
- Active compatibility routes remain in `src/runtime/wailsBridge.ts`, dispatching to Go `DstuService` path/folder/resource methods over the lean JSON-backed folder store plus Go hybrid VFS metadata.

GitNexus / impact notes:

- `dstu_parse_path`: LOW risk, one direct caller in old Rust (`dstu_get_resource_by_path`), which was retired in the same batch.
- `dstu_build_path`, `dstu_get_resource_location`, `dstu_get_resource_by_path`, `dstu_move_to_folder`, `dstu_batch_move`, `dstu_refresh_path_cache`, and `dstu_get_path_by_id`: LOW risk with 0 impacted callers for the old Rust command wrappers.

Verification:

- `rg "dstu_parse_path|dstu_build_path|dstu_get_resource_location|dstu_get_resource_by_path|dstu_move_to_folder|dstu_batch_move|dstu_refresh_path_cache|dstu_get_path_by_id" src-tauri/src`: no matches.
- `go test ./internal/dstu -count=1` from `desktop-go`: pass.
- `go test ./... -count=1` from `desktop-go`: pass.
- `npm run typecheck`: pass.
- `cargo check` from `src-tauri`: pass with 201 existing warnings.
- `npm run native:triage`: pass, 643 commands.
- `npm run native:inventory -- --summary`: pass, 1526 scanned files / 951 native references / 858 invokes / 643 unique invokes.
- `node scripts/rust-retirement-map.mjs`: pass, merged Rust registrations reduced from 95 to 87; merged Rust definitions reduced from 117 to 109; retirement candidates reduced from 92 to 84; direct Tauri blockers remain 42 commands / 43 edges / 9 files.
- `git diff --check`: still reports only pre-existing trailing whitespace in `src/hooks/useSystemSettings.ts` lines 52 and 130, plus Git line-ending normalization notice for `src-tauri/src/vfs/ref_handlers.rs`.
- `npx gitnexus detect_changes --repo "Deep Student"` was attempted and still fails because this GitNexus CLI build does not provide `detect_changes`.

## Completed File Bytes Rust Command Retirement Slice

Retired the old Rust/Tauri `read_file_bytes` command wrapper now that Go `FileService.ReadFileBytes` owns the active native route.

What changed:

- Replaced the remaining Rust internal caller in `src-tauri/src/cmd/textbooks.rs` with `unified_file_manager::read_all_bytes`, preserving old Rust textbook preview behavior without keeping the command wrapper as an internal helper.
- Removed the `read_file_bytes` Tauri command body from `src-tauri/src/commands.rs`.
- Removed the matching Tauri registration from `src-tauri/src/lib.rs`.
- Active compatibility remains in `src/runtime/native.ts` and `src/runtime/wailsBridge.ts`, dispatching `read_file_bytes` to Go `FileService.ReadFileBytes`.

Known parity notes:

- The same File/System candidate group still includes `get_app_data_dir`, `get_image_as_base64`, `copy_file`, `get_file_size`, and `ensure_debug_log_dir` in the generated retirement map, but these are Rust helper functions rather than independent external command wrappers in the current source. They should not be deleted until their remaining Rust internal dependencies are gone or each external wrapper boundary is separated cleanly.
- Superseded by the later "Completed Image Base64 Native Facade Cleanup Slice": frontend `get_image_as_base64` business reads now go through `src/runtime/native.ts`, and `rg` confirms the command string remains only in the native facade and Wails bridge.

GitNexus / impact notes:

- `read_file_bytes`: CRITICAL because the symbol had two old Rust internal callers, `textbooks_add` and `textbooks_adopt`. Both are old Rust textbook command paths, and the remaining call was replaced with the shared unified-file helper before deleting the wrapper.
- `textbooks_add` and `textbooks_adopt`: LOW risk with 0 impacted callers in the current index before the helper substitution.
- `get_file_size`, `get_image_as_base64`, and `ensure_debug_log_dir`: LOW for the checked old Rust symbols.
- `get_app_data_dir` and `copy_file`: CRITICAL as broad Rust helper symbols; they were intentionally not deleted.

Verification:

- `rg "read_file_bytes|crate::commands::read_file_bytes" src-tauri/src -g "*.rs"`: no matches.
- `go test ./internal/files ./internal/system -count=1` from `desktop-go`: pass.
- `go test ./... -count=1` from `desktop-go`: pass.
- `cargo check` from `src-tauri`: pass with 201 existing warnings.
- `npm run typecheck`: pass.
- `npm run native:triage`: pass, 643 commands.
- `npm run native:inventory -- --summary`: pass, 1526 scanned files / 951 native references / 858 invokes / 643 unique invokes.
- `node scripts/rust-retirement-map.mjs`: pass, merged Rust registrations reduced from 87 to 86; merged Rust definitions reduced from 109 to 108; retirement candidates reduced from 84 to 83; direct Tauri blockers remain 42 commands / 43 edges / 9 files.
- `git diff --check`: still reports only pre-existing trailing whitespace in `src/hooks/useSystemSettings.ts` lines 52 and 130, plus Git line-ending normalization notice for `src-tauri/src/vfs/ref_handlers.rs`.
- `npx gitnexus detect_changes --repo "Deep Student"` was attempted and still fails because this GitNexus CLI build does not provide `detect_changes`.

## Completed Review Plan Rust Retirement Slice

Retired the Go-backed spaced-repetition/review-plan command batch from the old Rust/Tauri command surface:

- `review_plan_create`
- `review_plan_process`
- `review_plan_get_due`
- `review_plan_get_due_with_filter`
- `review_plan_get_stats`
- `review_plan_refresh_stats`
- `review_plan_get_by_question`
- `review_plan_get`
- `review_plan_suspend`
- `review_plan_resume`
- `review_plan_delete`
- `review_plan_get_history`
- `review_plan_batch_create`
- `review_plan_create_for_exam`
- `review_plan_list_by_exam`
- `review_plan_get_or_create`
- `review_plan_get_calendar_data`

What changed:

- Removed the old Rust `crate::review_plan_service::*` Tauri command registrations from `src-tauri/src/lib.rs`.
- Deleted `src-tauri/src/review_plan_service.rs`, which was no longer referenced outside its Tauri registrations.
- Deleted the now-orphaned Rust VFS review-plan repo layer `src-tauri/src/vfs/repos/review_plan_repo.rs`.
- Removed `review_plan_repo` exports from `src-tauri/src/vfs/repos/mod.rs`.
- Active compatibility routes remain in `src/runtime/native.ts` and `src/runtime/wailsBridge.ts`, dispatching to Go `ReviewPlanService` in `desktop-go/internal/reviewplan` through Wails bindings.

Verification:

- GitNexus impact was run for all 17 retired command symbols plus `ReviewPlanService`; every result was LOW risk with 0 impacted callers.
- GitNexus impact was run for the old Rust repo symbols before deleting `review_plan_repo.rs`. `ReviewHistory` returned HIGH because the stale index still traced same-file repo helpers and the just-deleted `review_plan_service.rs`; no active execution process was reported. Other checked repo symbols were LOW.
- `rg "review_plan_service|review_plan_repo|review_plan_(create|process|get_due|get_due_with_filter|get_stats|refresh_stats|get_by_question|get\\b|suspend|resume|delete|get_history|batch_create|create_for_exam|list_by_exam|get_or_create|get_calendar_data)|VfsReviewPlanRepo|ReviewPlanStatus" src-tauri/src -g "*.rs"`: no matches.
- `go test ./internal/reviewplan -count=1` from `desktop-go`: pass.
- `go test ./internal/bindings -count=1` from `desktop-go`: pass.
- `go test ./...` from `desktop-go`: pass.
- `npm run typecheck`: pass.
- `npm run native:triage`: pass, 643 commands.
- `npm run native:inventory -- --summary`: pass, 1526 scanned files / 951 native references / 858 invokes / 643 unique invokes.
- `node scripts/rust-retirement-map.mjs`: pass, merged Rust registrations reduced from 209 to 193; merged Rust definitions from 219 to 203; retirement candidates from 194 to 178; direct Tauri blockers unchanged at 42 commands / 43 edges / 9 files.
- `cargo check` from `src-tauri`: pass in 6m30s with 201 existing warnings.
- `git diff --check`: still reports only pre-existing trailing whitespace in `src/hooks/useSystemSettings.ts` lines 52 and 130, plus Git line-ending normalization notice for `src-tauri/src/vfs/ref_handlers.rs`.
- `npx gitnexus detect_changes --repo "Deep Student"` was attempted and still fails because this GitNexus CLI build does not provide `detect_changes`.

## Completed VFS Ref Rust Retirement Slice

Retired the Go-backed VFS resource-ref command batch from the old Rust/Tauri command surface:

- `vfs_get_resource_refs`
- `vfs_resolve_resource_refs`
- `vfs_get_resource_ref_count`

What changed:

- Removed the 3 old Rust command wrappers from `src-tauri/src/vfs/ref_handlers.rs`.
- Removed the matching `crate::vfs::ref_handlers::*` registrations from `src-tauri/src/lib.rs`.
- Deleted the now-dead private Rust ref-resolution helper tree that only served the retired wrappers, including legacy ref construction, single-ref resolution, old multimodal block assembly, and retired schema tests.
- Kept the Rust helpers still used by remaining DSTU paths: `extract_file_text_with_strategy`, `get_image_ocr_text_with_conn`, `get_resource_path_internal`, OCR/extracted-text helpers, and source-id typing.
- Updated `src-tauri/src/dstu/handler_utils/content_helpers.rs` comments so they no longer point at the retired Rust command.
- Active compatibility routes remain in `src/runtime/wailsBridge.ts` and dispatch to Go `VfsService`.

Verification:

- GitNexus impact was run for all 3 retired command symbols; each returned LOW risk with 0 impacted callers.
- GitNexus impact was also run for the deleted private helper tree. The only MEDIUM result was `get_resource_ref_with_conn`, whose direct impacts were the retired wrapper, same-file helper chain, and now-deleted helper tests; no active process was affected and there were no HIGH/CRITICAL results.
- `rg "\b(vfs_get_resource_refs|vfs_resolve_resource_refs|vfs_get_resource_ref_count|get_resource_ref_with_conn|resolve_single_ref_with_conn|get_exam_multimodal_blocks_with_conn|get_file_multimodal_blocks_with_conn)\b" src-tauri/src -g "*.rs"`: no matches.
- `go test ./internal/vfs -count=1` from `desktop-go`: pass.
- `cargo check` from `src-tauri`: pass with 201 existing warnings after the dead helper cleanup. A later cold rebuild after `cargo clean` was blocked by disk capacity, not by source errors.
- `cargo test vfs::ref_handlers --lib` was attempted after freeing build temp/cache space, but the cold Rust dependency rebuild filled `C:\gh-runners\deep-student\_target` again and failed with `os error 112` before reaching these tests.
- `go test ./...` from `desktop-go`: pass after clearing stale build temp files.
- `npm run typecheck`: pass.
- `npm run native:triage`: pass, 643 commands.
- `npm run native:inventory -- --summary`: pass, 1526 scanned files / 951 native references / 858 invokes / 643 unique invokes.
- `node scripts/rust-retirement-map.mjs`: pass, merged Rust registrations reduced from 212 to 209; merged Rust definitions from 222 to 219; retirement candidates from 197 to 194; direct Tauri blockers unchanged at 42 commands / 43 edges / 9 files.
- `git diff --check`: still reports only pre-existing trailing whitespace in `src/hooks/useSystemSettings.ts` lines 52 and 130, plus Git line-ending normalization notice for `src-tauri/src/vfs/ref_handlers.rs`.
- `npx gitnexus detect_changes --repo "Deep Student"` was attempted and still fails because this GitNexus CLI build does not provide `detect_changes`.

## Completed DSTU Trash Rust Retirement Slice

Retired the Go-backed DSTU trash command batch from the old Rust/Tauri command surface:

- `dstu_soft_delete`
- `dstu_trash_restore`
- `dstu_list_trash`
- `dstu_empty_trash`
- `dstu_permanently_delete`

What changed:

- Reduced `src-tauri/src/dstu/trash_handlers.rs` from the old Tauri trash command implementation to the single `is_resource_in_trash` helper still used by remaining Rust DSTU purge safety checks.
- Removed the 5 `crate::dstu::trash_handlers::*` registrations from `src-tauri/src/lib.rs`.
- Active compatibility routes remain in `src/runtime/wailsBridge.ts` and dispatch to Go `DstuService`.

Verification:

- GitNexus impact was run for all 5 retired command symbols; each returned LOW risk with 0 impacted callers.
- `rg "\b(dstu_soft_delete|dstu_trash_restore|dstu_list_trash|dstu_empty_trash|dstu_permanently_delete)\b" src-tauri/src -g "*.rs"`: no matches.
- `go test ./internal/dstu -count=1` from `desktop-go`: pass.
- `cargo check` from `src-tauri`: pass in 1m18s with 201 existing warnings.
- `go test ./...` from `desktop-go`: pass.
- `npm run typecheck`: pass.
- `npm run native:triage`: pass, 643 commands.
- `npm run native:inventory -- --summary`: pass, 1526 scanned files / 951 native references / 858 invokes / 643 unique invokes.
- `node scripts/rust-retirement-map.mjs`: pass, merged Rust registrations reduced from 217 to 212; merged Rust definitions from 227 to 222; retirement candidates from 202 to 197; direct Tauri blockers unchanged at 42 commands / 43 edges / 9 files.
- `git diff --check`: still reports only pre-existing trailing whitespace in `src/hooks/useSystemSettings.ts` lines 52 and 130.
- `npx gitnexus detect_changes --repo "Deep Student"` was attempted and still fails because this GitNexus CLI build does not provide `detect_changes`.

## Completed Qbank Sync Rust Retirement Slice

Retired the Go-backed Qbank sync compatibility batch from the old Rust/Tauri command surface:

- `qbank_sync_check`
- `qbank_get_sync_conflicts`
- `qbank_resolve_sync_conflict`
- `qbank_batch_resolve_conflicts`
- `qbank_set_sync_enabled`

What changed:

- Removed the 5 old Rust command bodies from `src-tauri/src/question_sync_service.rs`.
- Removed the matching `invoke_handler` registrations from `src-tauri/src/lib.rs`.
- At the time of this historical slice, `qbank_update_sync_config` and the broader old Rust `QuestionSyncService` internals were kept because that command was still registered. This is superseded by the Completed Qbank Sync Config Native Facade And Rust Retirement Slice below.
- Active compatibility routes remain in `src/runtime/wailsBridge.ts` and dispatch to Go `QbankService`.

Verification:

- GitNexus impact was run for all 5 retired command symbols; each returned LOW risk with 0 impacted callers.
- `rg "\b(qbank_sync_check|qbank_get_sync_conflicts|qbank_resolve_sync_conflict|qbank_batch_resolve_conflicts|qbank_set_sync_enabled)\b" src-tauri/src -g "*.rs"`: no matches.
- `go test ./internal/qbank -count=1` from `desktop-go`: pass.
- `cargo check` from `src-tauri`: pass in 41.10s with 201 existing warnings.
- `go test ./...` from `desktop-go`: pass.
- `npm run typecheck`: pass.
- `npm run native:triage`: pass, 643 commands.
- `npm run native:inventory -- --summary`: pass, 1526 scanned files / 951 native references / 858 invokes / 643 unique invokes.
- `node scripts/rust-retirement-map.mjs`: pass, merged Rust registrations reduced from 222 to 217; merged Rust definitions from 232 to 227; retirement candidates from 207 to 202; direct Tauri blockers unchanged at 42 commands / 43 edges / 9 files.
- `git diff --check`: still reports only pre-existing trailing whitespace in `src/hooks/useSystemSettings.ts` lines 52 and 130.
- `npx gitnexus detect_changes --repo "Deep Student"` was attempted and still fails because this GitNexus CLI build does not provide `detect_changes`.

## Completed Todo/Pomodoro Rust Retirement Slice

Retired the Go-backed Todo and Pomodoro command batch from the old Rust/Tauri command surface:

- Todo list commands: `todo_create_list`, `todo_get_list`, `todo_list_lists`, `todo_update_list`, `todo_delete_list`, `todo_toggle_list_favorite`, `todo_ensure_inbox`.
- Todo item/query commands: `todo_create_item`, `todo_get_item`, `todo_list_items`, `todo_update_item`, `todo_toggle_item`, `todo_delete_item`, `todo_reorder_items`, `todo_list_today`, `todo_list_overdue`, `todo_list_upcoming`, `todo_list_completed`, `todo_search`, `todo_get_active_summary`.
- Pomodoro commands: `pomodoro_create_record`, `pomodoro_get_record`, `pomodoro_list_by_todo`, `pomodoro_today_stats`, `pomodoro_list_today`.

What changed:

- Deleted `src-tauri/src/vfs/todo_handlers.rs`, which only contained old Tauri command handlers for migrated Todo/Pomodoro UI APIs.
- Removed `pub mod todo_handlers` from `src-tauri/src/vfs/mod.rs`.
- Removed all 25 `crate::vfs::todo_handlers::*` registrations from `src-tauri/src/lib.rs`.
- Kept old Rust VFS todo/pomodoro repos and types because other still-live Rust paths can read VFS todo state during the broader backend retirement.
- Active compatibility routes remain in `src/runtime/wailsBridge.ts` and dispatch to Go `TodoService`.

Verification:

- GitNexus impact was run for all 25 retired command symbols; each returned LOW risk with 0 impacted callers.
- `rg "\b(todo_create_list|todo_get_list|todo_list_lists|todo_update_list|todo_delete_list|todo_toggle_list_favorite|todo_ensure_inbox|todo_create_item|todo_get_item|todo_list_items|todo_update_item|todo_toggle_item|todo_delete_item|todo_reorder_items|todo_list_today|todo_list_overdue|todo_list_upcoming|todo_list_completed|todo_search|todo_get_active_summary|pomodoro_create_record|pomodoro_get_record|pomodoro_list_by_todo|pomodoro_today_stats|pomodoro_list_today)\b" src-tauri/src -g "*.rs"`: no matches.
- `go test ./internal/todo -count=1` from `desktop-go`: pass.
- `cargo check` from `src-tauri`: pass in 2m25s with 201 existing warnings.
- `go test ./...` from `desktop-go`: pass.
- `npm run typecheck`: pass.
- `npm run native:triage`: pass, 643 commands.
- `npm run native:inventory -- --summary`: pass, 1526 scanned files / 951 native references / 858 invokes / 643 unique invokes.
- `node scripts/rust-retirement-map.mjs`: pass, merged Rust registrations reduced from 247 to 222; merged Rust definitions from 257 to 232; retirement candidates from 232 to 207; direct Tauri blockers unchanged at 42 commands / 43 edges / 9 files.
- `git diff --check`: still reports only pre-existing trailing whitespace in `src/hooks/useSystemSettings.ts` lines 52 and 130.
- `npx gitnexus detect_changes --repo "Deep Student"` was attempted and still fails because this GitNexus CLI build does not provide `detect_changes`.

## Completed LLM Usage Rust Retirement Slice

Retired the Go-backed LLM usage query command batch from the old Rust/Tauri command surface:

- `llm_usage_get_trends`
- `llm_usage_by_model`
- `llm_usage_by_caller`
- `llm_usage_summary`
- `llm_usage_recent`
- `llm_usage_daily`
- `llm_usage_cleanup`

What changed:

- Deleted `src-tauri/src/llm_usage/handlers.rs`, which only contained old Rust Tauri query handlers for the migrated usage-stat command surface.
- Removed `pub mod handlers` from `src-tauri/src/llm_usage/mod.rs`.
- Removed the 7 `crate::llm_usage::handlers::*` registrations from `src-tauri/src/lib.rs`.
- Kept the old Rust `llm_usage` collector/database/repo/types modules because remaining Rust LLM paths can still record usage while the broader provider pipeline is being retired.
- Active compatibility routes remain in `src/runtime/wailsBridge.ts` and dispatch to Go `ChatService.LLMUsage*` methods.

Verification:

- GitNexus impact was run for all 7 retired command symbols; each returned LOW risk with 0 impacted callers.
- `rg "llm_usage_(get_trends|by_model|by_caller|summary|recent|cleanup)|llm_usage::handlers|pub async fn llm_usage_daily|handlers::llm_usage_daily" src-tauri/src -g "*.rs"`: no matches.
- `go test ./internal/chat -count=1` from `desktop-go`: pass.
- `cargo check` from `src-tauri`: pass in 1m22s with 201 existing warnings.
- `go test ./...` from `desktop-go`: pass.
- `npm run typecheck`: pass.
- `npm run native:triage`: pass, 643 commands.
- `npm run native:inventory -- --summary`: pass, 1526 scanned files / 951 native references / 858 invokes / 643 unique invokes.
- `node scripts/rust-retirement-map.mjs`: pass, merged Rust registrations reduced from 254 to 247; merged Rust definitions from 264 to 257; retirement candidates from 239 to 232; direct Tauri blockers unchanged at 42 commands / 43 edges / 9 files.
- `git diff --check`: still reports only pre-existing trailing whitespace in `src/hooks/useSystemSettings.ts` lines 52 and 130.
- `npx gitnexus detect_changes --repo "Deep Student"` was attempted and still fails because this GitNexus CLI build does not provide `detect_changes`.

## Completed Chat Tags/Approval/AskUser Rust Retirement Slice

Retired the next Go-backed chat interaction batch from the old Rust/Tauri command surface:

- Tag commands: `chat_v2_get_session_tags`, `chat_v2_get_tags_batch`, `chat_v2_add_tag`, `chat_v2_remove_tag`, `chat_v2_list_all_tags`.
- Approval commands: `chat_v2_tool_approval_respond`, `chat_v2_clear_approval_history`.
- Ask-user command: `chat_v2_ask_user_respond`.

What changed:

- Removed the old Rust command bodies for tag management from `src-tauri/src/chat_v2/handlers/search_handlers.rs`, leaving only the still-live Rust `chat_v2_search_content` command.
- Removed the old Rust tool-approval response and approval-history clearing command bodies from `src-tauri/src/chat_v2/handlers/approval_handlers.rs`, leaving only `chat_v2_tool_approval_cancel`.
- Deleted `src-tauri/src/chat_v2/handlers/ask_user_handlers.rs` after its only command moved to the native chat runtime.
- Removed matching Rust handler module exports and `invoke_handler` registrations from `src-tauri/src/chat_v2/handlers/mod.rs` and `src-tauri/src/lib.rs`.
- Updated the stale ask-user executor comment so it no longer claims responses arrive through the retired Tauri command.
- Active compatibility routes remain in `src/runtime/wailsBridge.ts` and dispatch to Go `ChatService` methods.

Verification:

- GitNexus impact was run for all 8 retired command symbols; each returned LOW risk with 0 impacted callers.
- `rg "chat_v2_get_session_tags|chat_v2_get_tags_batch|chat_v2_add_tag|chat_v2_remove_tag|chat_v2_list_all_tags|chat_v2_tool_approval_respond|chat_v2_clear_approval_history|chat_v2_ask_user_respond|ask_user_handlers" src-tauri/src -g "*.rs"`: no matches.
- `go test ./internal/chat -count=1` from `desktop-go`: pass.
- `cargo check` from `src-tauri`: pass in 1m22s with 201 existing warnings.
- `go test ./...` from `desktop-go`: pass.
- `npm run typecheck`: pass.
- `npm run native:triage`: pass, 643 commands.
- `npm run native:inventory -- --summary`: pass, 1526 scanned files / 951 native references / 858 invokes / 643 unique invokes.
- `node scripts/rust-retirement-map.mjs`: pass, merged Rust registrations reduced from 260 to 254; merged Rust definitions from 270 to 264; retirement candidates from 245 to 239; direct Tauri blockers unchanged at 42 commands / 43 edges / 9 files.
- `git diff --check`: still reports only pre-existing trailing whitespace in `src/hooks/useSystemSettings.ts` lines 52 and 130.
- `npx gitnexus detect_changes --repo "Deep Student"` was attempted and still fails because this GitNexus CLI build does not provide `detect_changes`.

## Completed Core Chat Session/Group/Block Rust Retirement Slice

Retired another Go-backed core chat batch from the old Rust/Tauri command surface:

- Session commands: `chat_v2_create_session`, `chat_v2_archive_session`, `chat_v2_save_session`, `chat_v2_list_sessions`, `chat_v2_count_sessions`, `chat_v2_branch_session`.
- Group commands: `chat_v2_create_group`, `chat_v2_update_group`, `chat_v2_list_groups`, `chat_v2_reorder_groups`, `chat_v2_move_session_to_group`.
- Block commands: `chat_v2_delete_message`, `chat_v2_update_block_content`.

What changed:

- Removed the old Rust command bodies, exports, and `invoke_handler` registrations from `src-tauri/src/chat_v2/handlers/manage_session.rs`, `src-tauri/src/chat_v2/handlers/group_handlers.rs`, `src-tauri/src/chat_v2/handlers/block_actions.rs`, `src-tauri/src/chat_v2/handlers/mod.rs`, `src-tauri/src/chat_v2/mod.rs`, and `src-tauri/src/lib.rs`.
- Removed old Rust-only helper logic for the retired session branch/save/create/archive paths and the retired message delete/block content update paths.
- Kept `ensure_group_folder` in Rust because it is still shared by remaining Rust pipeline/resource-scope/VFS code; it is not a callable group command.
- The active command compatibility routes remain in `src/runtime/wailsBridge.ts` and dispatch to Go `ChatService`.

Verification:

- GitNexus impact was run for all 13 retired command symbols; each returned LOW risk with 0 impacted callers.
- `rg "chat_v2_archive_session|chat_v2_branch_session|chat_v2_count_sessions|chat_v2_create_session|chat_v2_list_sessions|chat_v2_save_session|chat_v2_create_group|chat_v2_list_groups|chat_v2_move_session_to_group|chat_v2_reorder_groups|chat_v2_update_group|chat_v2_delete_message|chat_v2_update_block_content" src-tauri/src -g "*.rs"`: no matches.
- `cargo check` from `src-tauri`: pass in 1m39s with 201 existing warnings.
- `npm run typecheck`: pass.
- `go test ./...` from `desktop-go`: pass.
- `npm run native:triage`: pass, 643 commands.
- `npm run native:inventory -- --summary`: pass, 1526 scanned files / 951 native references / 858 invokes / 643 unique invokes.
- `node scripts/rust-retirement-map.mjs`: pass, merged Rust registrations reduced from 273 to 260; merged Rust definitions from 283 to 270; retirement candidates from 258 to 245.
- `cargo fmt` was attempted on the touched Rust files, but this repo's current Rust formatter pass is blocked by pre-existing trailing whitespace in `src-tauri/src/translation/pipeline.rs:169`.

## Completed Chat Load/Streaming Block Rust Retirement Slice

Moved remaining core/debug frontend callers for already-merged chat load and streaming-block commands behind the Go/Wails native facade, then deleted the matching Rust command entries:

- `chat_v2_load_session`
- `chat_v2_upsert_streaming_block`

What changed:

- `src/features/chat/workspace/events.ts`, `src/features/chat/workspace/components/SubagentContainer.tsx`, and `src/features/chat/plugins/events/toolCall.ts` now use `@/runtime/native` for `chat_v2_load_session` / `chat_v2_upsert_streaming_block` instead of direct Tauri core imports.
- Chat debug/test helpers in `src/features/chat/debug/*` and `src/debug-panel/plugins/ThinkingBlockDebugPlugin.tsx` were moved to `@/runtime/native` for their load/delete session checks, so debug-only direct Tauri calls no longer block Rust retirement.
- Removed the dead Rust `clear_session_sequence_counter` helper after `chat_v2_delete_session` had already moved to Go.
- Removed the old Rust `chat_v2_load_session` command file/module/export/registration.
- Removed the old Rust `chat_v2_upsert_streaming_block` command body/export/registration and its exclusive placeholder-message helper while keeping shared block upsert helpers still used by remaining Rust commands.
- Updated the stale subagent executor comment so it no longer claims `SubagentContainer` depends on the retired Rust command.

Verification:

- `npx gitnexus impact --repo "Deep Student" --direction upstream clear_session_sequence_counter`: LOW; only direct caller was the retired Rust `chat_v2_delete_session`.
- `npx gitnexus impact --repo "Deep Student" --direction upstream initWorkspaceEventListeners`: LOW/0.
- `npx gitnexus impact --repo "Deep Student" --direction upstream SubagentContainer`: LOW/0.
- `npx gitnexus impact --repo "Deep Student" --direction upstream updateWorkspaceStatusBlockSnapshot`: LOW; direct impact stays within the tool-call event file/import chain.
- `npx gitnexus impact --repo "Deep Student" --direction upstream chat_v2_load_session`: LOW/0.
- `npx gitnexus impact --repo "Deep Student" --direction upstream chat_v2_upsert_streaming_block`: LOW/0.
- `npx gitnexus impact --repo "Deep Student" --direction upstream SubagentExecutor`: LOW/0.
- `rg "chat_v2_load_session|chat_v2_upsert_streaming_block|handlers::load_session|pub mod load_session" src-tauri/src`: no Rust command/module matches remain.
- `npm run typecheck`: pass.
- `cargo check` from `src-tauri`: pass in 2m14s with 201 existing warnings.
- `npm run native:triage`: pass, 643 commands.
- `npm run native:inventory -- --summary`: pass, 1526 scanned files / 951 native references / 858 invokes / 643 unique invokes.
- `node scripts/rust-retirement-map.mjs`: pass, merged Rust registrations reduced from 275 to 273; merged Rust definitions from 285 to 283; direct Tauri blocked merged commands reduced from 46 to 42; blocker edges from 53 to 43; blocker files from 16 to 9.

## Completed Core Chat/Learning Hub Native Facade Retirement Slice

Moved core frontend call sites through the Go/Wails native facade and retired the matching Rust command entries:

- `chat_v2_get_session`
- `chat_v2_update_session_settings`
- `chat_v2_get_group`
- `vfs_get_resource`

What changed:

- `src/features/chat/pages/useChatPageEvents.ts` now imports `invoke` from `@/runtime/native` for session navigation, bookmark settings updates, and VFS resource resolution.
- `src/features/learning-hub/LearningHubPage.tsx` now resolves `res_*` VFS IDs through `@/runtime/native` instead of dynamically importing Tauri core.
- `src/features/learning-hub/LearningHubSidebar.tsx` now resolves topic group roots through `@/runtime/native`; dialog/event/asset helpers remain Tauri-specific until the runtime facade grows those surfaces.
- `src/features/chat/pages/__tests__/GroupReadCommand.source.test.ts` now asserts the old Rust group read command is retired and the Wails bridge routes `chat_v2_get_group` to Go `ChatService.GetGroup`.
- Old Rust command bodies/exports/registrations for `chat_v2_get_session`, `chat_v2_update_session_settings`, `chat_v2_get_group`, and `vfs_get_resource` were removed from `src-tauri/src/chat_v2/handlers/*`, `src-tauri/src/vfs/handlers.rs`, `src-tauri/src/chat_v2/mod.rs`, and `src-tauri/src/lib.rs`.
- The old Rust `decrement_vfs_refs_for_session` helper and `clear_session_sequence_counter` import were removed after the hard-delete Rust command retirement left them unused.

Verification:

- `npx gitnexus impact --repo "Deep Student" --direction upstream useChatPageEvents`: LOW; direct impact to `ChatV2Page`.
- `npx gitnexus impact --repo "Deep Student" --direction upstream LearningHubPage`: LOW/0.
- `npx gitnexus impact --repo "Deep Student" --direction upstream LearningHubSidebar`: LOW/0.
- `npx gitnexus impact --repo "Deep Student" --direction upstream chat_v2_get_session`: LOW/0.
- `npx gitnexus impact --repo "Deep Student" --direction upstream chat_v2_update_session_settings`: LOW/0.
- `npx gitnexus impact --repo "Deep Student" --direction upstream chat_v2_get_group`: LOW/0.
- `npx gitnexus impact --repo "Deep Student" --direction upstream vfs_get_resource`: LOW/0.
- `npx gitnexus impact --repo "Deep Student" --direction upstream decrement_vfs_refs_for_session`: LOW; direct caller was the retired Rust `chat_v2_delete_session`.
- `npm run typecheck`: pass.
- `npm run native:triage`: pass, 643 commands.
- `npm run native:inventory -- --summary`: pass, 1526 scanned files / 951 native references / 858 invokes / 643 unique invokes.
- `node scripts/rust-retirement-map.mjs`: pass, direct Tauri blocked merged commands reduced from 50 to 46; blocker edges from 58 to 53; blocker files from 19 to 16; merged Rust registrations from 279 to 275; merged Rust definitions from 289 to 285.
- `cargo check` from `src-tauri`: pass. The first run took 6m32s and produced 203 existing warnings plus a new unused import/helper warning from this slice; after removing the helper/import, the second run passed in 1m24s with 201 existing warnings.
- `npm run test -- src/features/chat/pages/__tests__/GroupReadCommand.source.test.ts` was attempted but did not reach tests because the install at that time was missing `@testing-library/dom`.
- `git diff --check`: still reports only the pre-existing trailing whitespace in `src/hooks/useSystemSettings.ts` lines 52 and 130.
- `npx gitnexus detect_changes --repo "Deep Student"` was attempted and failed because this GitNexus CLI build does not provide that command.

## Completed Chat Archive Retirement Slice

Merged into Go `ChatService` and retired from Rust/Tauri command paths:

- `chat_v2_restore_session`
- `chat_v2_delete_session`
- `chat_v2_delete_group`
- Added Go/Wails-only compatibility for `chat_v2_restore_group`, which the archive UI already expected but Rust did not provide as a registered command.

What changed:

- `desktop-go/internal/chat/service.go` now restores archived/deleted sessions to active, permanently deletes sessions by removing session state/messages/blocks, blocks hard delete while a message stream is active, soft-deletes groups to `deleted`, ungroups sessions when a group is deleted, and restores groups to active.
- `desktop-go/internal/chat/service_test.go` covers session restore + permanent delete cascade and group delete/restore behavior.
- `desktop-go/internal/bindings/chat_service.go` and `src/runtime/wailsBridge.ts` expose the new Go methods through Wails.
- `src/features/settings/components/data-governance/ChatSessionArchiveTab.tsx` now imports `invoke` from `@/runtime/native` instead of direct `@tauri-apps/api/core`.
- Old Rust command bodies/exports/registrations for `chat_v2_restore_session`, `chat_v2_delete_session`, and `chat_v2_delete_group` were removed from `src-tauri/src/chat_v2/handlers/*`, `src-tauri/src/chat_v2/mod.rs`, `src-tauri/src/chat_v2/repo.rs`, and `src-tauri/src/lib.rs`.

Verification:

- `npx gitnexus impact --repo "Deep Student" --direction upstream chat_v2_delete_session`: LOW, 0 impacted callers.
- `npx gitnexus impact --repo "Deep Student" --direction upstream chat_v2_restore_session`: LOW, 0 impacted callers.
- `npx gitnexus impact --repo "Deep Student" --direction upstream chat_v2_delete_group`: LOW, 0 impacted callers.
- `npx gitnexus impact --repo "Deep Student" --direction upstream ChatSessionArchiveTab`: LOW, 0 impacted callers.
- `npx gitnexus impact --repo "Deep Student" --direction upstream soft_delete_group_with_conn`: LOW, one direct caller, the retired Rust command.
- New Go symbols such as `RestoreSession`, `DeleteGroup`, and `RestoreGroup` are not represented in the current GitNexus index yet because `desktop-go` is still untracked relative to indexed commit `1812ec7`.
- `go test ./...` from `desktop-go`: pass.
- `npm run go:bindings`: pass, 11 services / 289 methods / 192 models.
- `npm run typecheck`: pass.
- `npm run native:triage`: pass, 643 commands.
- `npm run native:inventory -- --summary`: pass, 1526 scanned files / 951 native references / 858 invokes / 643 unique invokes.
- `node scripts/rust-retirement-map.mjs`: pass, direct Tauri blocked merged commands reduced from 55 to 50; blocker edges from 64 to 58; blocker files from 20 to 19.
- `rg "chat_v2_delete_session|chat_v2_restore_session|chat_v2_delete_group|restore_session_in_db|soft_delete_group_with_conn" src-tauri/src/chat_v2 src-tauri/src/lib.rs`: no matches.
- `git diff --check`: still reports only the pre-existing trailing whitespace in `src/hooks/useSystemSettings.ts` lines 52 and 130.
- `npx gitnexus detect_changes --repo "Deep Student"` was attempted and failed because this GitNexus CLI build does not provide that command.

## Completed Anki Document Task Commands

Merged into Go `AnkiService`:

- `start_enhanced_document_processing`
- `get_document_tasks`
- `pause_document_processing`
- `resume_document_processing`
- `get_document_processing_state`
- `get_document_state`
- `get_document_task_counts`
- `trigger_task_processing`
- `delete_document_session`
- `get_document_cards`
- `recover_stuck_document_tasks`

The Anki document task slice now has a lean start worker over `anki-go.json`. `start_enhanced_document_processing` validates text input, creates a document session, splits plain text into segment tasks, asynchronously emits the existing `anki_generation_event` shapes (`DocumentProcessingStarted`, `TaskStatusUpdate`, `NewCard`, `TaskCompleted`, `DocumentProcessingPaused`, `TaskProcessingError`, `DocumentProcessingCompleted`), stores generated cards, and lets the existing CardForge collector receive cards in Wails through `src/runtime/nativeEvents.ts`. The worker now loads Settings API configs and `anki_card_model_config_id`, prefers an explicitly supplied model/config option, calls OpenAI-compatible `/chat/completions` for JSON card generation, normalizes provider cards into legacy CardForge fields, and falls back to deterministic local cards if no usable provider is configured or the provider output cannot be parsed. Pause/resume/retry/delete/status/card reads are still handled by the same Go service. This intentionally does not yet run OCR, render PDFs, implement robust document/PDF segmentation, stream partial provider cards, migrate old Rust document-task/card rows, or reproduce the heavy old EnhancedAnkiService pipeline.

## Completed Settings And LLM Config Commands

Merged into Go `SettingsService`:

- `get_setting`
- `get_settings`
- `get_settings_by_prefix`
- `save_setting`
- `save_settings`
- `delete_setting`
- `vfs_get_attachment_config`
- `vfs_set_attachment_root_folder`
- `vfs_create_attachment_root_folder`
- `get_api_configurations`
- `save_api_configurations`
- `get_vendor_configs`
- `save_vendor_configs`
- `get_model_profiles`
- `save_model_profiles`
- `get_model_assignments`
- `save_model_assignments`
- `get_cn_whitelist_config`
- `preheat_mcp_tools`
- `get_mcp_status`
- `get_mcp_tools`
- `reload_mcp_client`

The LLM configuration slice intentionally stores compact JSON values in `settings-go.json` and keeps frontend-compatible DTO shapes for API configs, vendor configs, model profiles, and model assignments. This replaces the command surface needed by settings/model picker UI without copying the old Rust LLM manager, provider bootstrap, repair logic, or secret-store complexity.

Important limitation: API keys are currently persisted as plain JSON inside the Go settings store. This is acceptable for the migration shell but not the final secure storage story. Minimal chat send/retry provider calls now exist in Go for text-only OpenAI-compatible `/chat/completions` SSE. Model list fetching, provider health checks, secure key storage, richer protocol selection, and non-chat consumers are still not implemented in Go.

## Completed Chat Commands

Merged into Go `ChatService`:

- `chat_v2_create_session`
- `chat_v2_get_session`
- `chat_v2_load_session`
- `chat_v2_save_session`
- `chat_v2_update_session_settings`
- `chat_v2_archive_session`
- `chat_v2_restore_session`
- `chat_v2_delete_session`
- `chat_v2_move_session_to_group`
- `chat_v2_list_sessions`
- `chat_v2_count_sessions`
- `chat_v2_branch_session`
- `chat_v2_create_group`
- `chat_v2_get_group`
- `chat_v2_update_group`
- `chat_v2_delete_group`
- `chat_v2_restore_group`
- `chat_v2_list_groups`
- `chat_v2_reorder_groups`
- `chat_v2_add_tag`
- `chat_v2_remove_tag`
- `chat_v2_list_all_tags`
- `chat_v2_get_session_tags`
- `chat_v2_get_tags_batch`
- `chat_v2_get_message_summary`
- `chat_v2_send_message`
- `chat_v2_continue_message`
- `chat_v2_cancel_stream`
- `chat_v2_retry_message`
- `chat_v2_edit_and_resend`
- `chat_v2_tool_approval_respond`
- `chat_v2_clear_approval_history`
- `chat_v2_ask_user_respond`
- `chat_v2_delete_message`
- `chat_v2_update_block_content`
- `chat_v2_upsert_streaming_block`
- `llm_usage_get_trends`
- `llm_usage_by_model`
- `llm_usage_by_caller`
- `llm_usage_summary`
- `llm_usage_recent`
- `llm_usage_daily`
- `llm_usage_cleanup`

Current provider-streaming status: Go `ChatService.SendMessage`, `ContinueMessage`, `RetryMessage`, and `EditAndResend` can load enabled API configs from `SettingsService`, call text-only OpenAI-compatible `/chat/completions` with `stream: true`, emit frontend-compatible `stream_start`, `content` block start/chunk/end, and `stream_complete` events through the Go runtime event bus and Wails `app.Event.Emit`, and persist the final assistant block plus actual model config ID. If no usable config exists, these paths keep the local placeholder stream fallback so the frontend still exercises the Wails event path. `chat_v2_cancel_stream` now cancels active provider HTTP requests through per-message Go contexts, persists cancelled blocks, and emits `stream_cancelled` for the existing frontend abort path. The edit-resend path now uses a frontend-preallocated replacement assistant message ID so Wails stream events, frontend stream expectation, and persisted Go message state all target the same assistant message. OpenAI-compatible provider streaming now requests final stream usage chunks with `stream_options.include_usage`, normalizes `promptTokens`, `completionTokens`, `totalTokens`, optional `reasoningTokens`/`cachedTokens`, and `lastRoundPromptTokens`, then persists usage in assistant `_meta.usage` and emits it on `stream_complete` for the existing frontend token usage UI.

Important limitation: this is intentionally a lean first provider slice, not a clone of the old Rust LLM manager. It only supports simple text messages over Chat Completions SSE. Message-level API usage is captured when the provider supplies the final stream usage chunk, and the old `llm_usage_*` command surface now has a Go-derived compatibility path over persisted Chat assistant `_meta.usage`. This is chat-derived usage only: non-chat usage capture, provider cost tables, exact pricing, and old standalone usage storage are not implemented. Tool loops, tool approval continuation, multimodal content, Responses API, provider-specific reasoning/thinking protocols, model list fetching, provider health checks, encrypted key storage, and multi-variant parity are still pending.

## Completed DSTU Commands

Merged into Go `DstuService`:

- `dstu_list`
- `dstu_get`
- `dstu_create`
- `dstu_update`
- `dstu_delete`
- `dstu_delete_many`
- `dstu_restore`
- `dstu_restore_many`
- `dstu_purge`
- `dstu_purge_all`
- `dstu_list_deleted`
- `dstu_soft_delete`
- `dstu_trash_restore`
- `dstu_list_trash`
- `dstu_empty_trash`
- `dstu_permanently_delete`
- `dstu_search`
- `dstu_get_content`
- `dstu_set_metadata`
- `dstu_set_favorite`
- `dstu_folder_create`
- `dstu_folder_get`
- `dstu_folder_rename`
- `dstu_folder_delete`
- `dstu_folder_move`
- `dstu_folder_set_expanded`
- `dstu_folder_add_item`
- `dstu_folder_remove_item`
- `dstu_folder_move_item`
- `dstu_folder_list`
- `dstu_folder_get_tree`
- `dstu_folder_get_items`
- `notes_import_markdown`
- `notes_import_markdown_batch`
- `textbooks_add`
- `dstu_folder_get_all_resources`
- `dstu_folder_reorder`
- `dstu_folder_reorder_items`
- `dstu_folder_get_breadcrumbs`
- `dstu_get_resource_by_path`
- `dstu_get_resource_location`
- `dstu_parse_path`
- `dstu_build_path`
- `dstu_move_to_folder`
- `dstu_batch_move`
- `dstu_refresh_path_cache`
- `dstu_get_path_by_id`
- `notes_search`
- `notes_list_tags`
- `canvas_note_read`
- `canvas_note_append`
- `canvas_note_replace`
- `canvas_note_set`

DSTU folder/path compatibility now uses a lean JSON-backed `folders` / `folderItems` store inside `dstu-go.json` instead of recreating the old Rust SQLite folder repository stack. Folder CRUD, tree reads, item add/remove/move, reorder, breadcrumbs, path parse/build, move-to-folder, batch move, path cache refresh, and path-by-ID are implemented in Go. `folderItems` is the preferred source of resource position; note metadata and Go hybrid VFS file metadata are synchronized for compatibility, and older metadata-only resources remain a fallback for location/resource aggregation.

Important limitation: this does not migrate old Rust `folders` / `folder_items` rows yet, does not recreate old soft-delete/trash semantics for folders, and does not resolve non-DSTU resource bodies such as old exam/translation/essay records unless those resources are already represented by Go notes or Go hybrid VFS records. Deleting a Go folder currently moves contained items to root rather than deleting resource bodies, which is intentional for a migration-safe lean shell.

Notes utility compatibility now routes `notes_search`, `notes_list_tags`, and `canvas_note_*` through the current Go `DstuService` note store instead of resurrecting the old Rust Notes database. Search supports simple terms plus `tag:<name>` filters over Go DSTU notes, tag listing is frequency-sorted with a 50-tag cap, and Canvas helpers read full notes or Markdown sections, append into section boundaries, perform literal/regex replacements, and overwrite note content while keeping hybrid VFS note refs synchronized.

Go DSTU note deletion is now migration-safe soft delete. `dstu_delete` marks notes as `status: deleted` with `deletedAt`, synchronizes that state into the Go hybrid VFS note resource, and active `dstu_get`, `dstu_list`, `notes_search`, `notes_list_tags`, `GetContent`, and VFS ref reads hide deleted notes. Restore and purge commands share the same store path so old DSTU trash command names and new wrapper aliases behave consistently. File/image/textbook deletion remains VFS soft delete, while folder deletion still moves contained items to root.

Important limitation: this notes utility slice intentionally does not rebuild old Notes version/mentions/link databases. Search covers Go DSTU notes currently in `dstu-go.json`; old Tauri/Rust notes still need a migration or compatibility path before full replacement can be claimed. Go note trash and Go zip import/export exist for migrated/Go DSTU notes, but old Rust notes are not automatically imported into that store yet.

## Completed Notes Commands

Merged into Go `NotesService` or Go `DstuService` wrappers:

- `notes_get_pref`
- `notes_set_pref`
- `notes_save_asset`
- `notes_list_assets`
- `notes_delete_asset`
- `notes_resolve_asset_path`
- `get_image_as_base64`
- `notes_assets_index_scan`
- `notes_assets_scan_orphans`
- `notes_assets_bulk_delete`
- `notes_db_stats`
- `notes_db_vacuum`
- `notes_export`
- `notes_export_single`
- `notes_import`
- `notes_list_deleted`
- `notes_empty_trash`
- `notes_hard_delete`
- `notes_restore`

Notes asset maintenance is intentionally filesystem/index based: asset scan commands walk the Go notes assets directory and current metadata instead of recreating the old Rust Notes database. Orphan detection includes deleted Go DSTU notes when checking content references, so recoverable note assets are not incorrectly deleted. Database stats/vacuum are lightweight JSON-store compatibility reports rather than SQLite maintenance.

Notes import/export is now a lean Go zip format over the current Go DSTU note store and visible `notes_assets` files. `notes_export` and `notes_export_single` write readable Markdown note files, `_notes.json` metadata, `manifest.json`, and referenced/listed `notes_assets/...` files. `notes_import` reads that zip format, restores assets into the Go notes assets tree, and upserts note records into `dstu-go.json` with `skip`, `overwrite`, or `merge_keep_newer` conflict behavior while resyncing note resources into the Go hybrid VFS. This replaces the product capability without copying the old Rust exporter/importer internals; the old Rust exporter is now only a reference until the Rust/Tauri tree is retired.

Important limitation: Go Notes import/export does not yet export old version history, preferences, note links, mentions, or old Rust SQLite note rows. It handles local zip file paths in Wails; mobile/content URI materialization and progress events are still part of the broader runtime/decommission work.

## Completed Qbank Commands

Merged into Go `QbankService`:

- `qbank_list_questions`
- `qbank_search_questions`
- `qbank_rebuild_fts_index`
- `qbank_get_question`
- `qbank_create_question`
- `qbank_update_question`
- `qbank_delete_question`
- `qbank_batch_delete_questions`
- `qbank_submit_answer`
- `qbank_toggle_favorite`
- `qbank_get_stats`
- `qbank_refresh_stats`
- `qbank_reset_progress`
- `qbank_reset_questions_progress`
- `qbank_get_history`
- `qbank_get_submissions`
- `qbank_get_learning_trend`
- `qbank_get_activity_heatmap`
- `qbank_get_knowledge_stats`
- `qbank_get_knowledge_stats_with_comparison`
- `qbank_start_timed_practice`
- `qbank_generate_mock_exam`
- `qbank_submit_mock_exam`
- `qbank_get_daily_practice`
- `qbank_generate_paper`
- `qbank_get_check_in_calendar`
- `qbank_ai_grade`
- `qbank_cancel_grading`
- `qbank_sync_check`
- `qbank_get_sync_conflicts`
- `qbank_resolve_sync_conflict`
- `qbank_batch_resolve_conflicts`
- `qbank_set_sync_enabled`
- `get_csv_preview`
- `import_questions_csv`
- `export_questions_csv`
- `get_csv_exportable_fields`

Learning-loop compatibility now uses the lean Go JSON qbank store rather than rebuilding the old Rust SQLite repository stack. It derives trend/heatmap/check-in data from `AnswerSubmission` rows, computes knowledge stats from question tags/status/attempt counts, records compact question history for updates and answers, supports history/submission reads, and provides stable local selection for timed practice, daily practice, mock exam sessions, mock exam scoring, and generated paper previews. `qbank_submit_answer` now also resyncs the hybrid VFS `exam` resource content/hash after progress changes.

Important limitation: CSV preview/import/export now works over local file paths with UTF-8, UTF-8 BOM, GBK, and GB18030 read support, plus UTF-8, UTF-8 BOM, and GBK export support. CSV import uses Go `QbankService` state and syncs imported questions into the hybrid VFS as `exam` resources. It does not yet materialize mobile/content virtual URIs into local temp files in the Wails bridge, and Wails does not emit incremental `csv_import_progress` events yet; the UI updates to final progress from the returned result. Existing Tauri/Rust qbank data migration is still pending.

Important limitation: the Go learning-loop slice is intentionally real-but-lean. It does not yet migrate old Rust qbank records/submissions/history, preserve rich AI grading state, persist mock exam sessions as first-class records, implement old sync/conflict flows, or reproduce SQL random sampling exactly. Practice selection is stable and deterministic for now, which keeps tests reproducible while preserving the frontend workflow contract.

Qbank AI grading compatibility now routes through Go `QbankService` and the native event facade. It validates that the submitted answer row belongs to the requested question, emits the existing `qbank_grading_stream_{sessionId}` `data` and `complete` event shapes through the Go/Wails event bus, loads Settings API configs plus `qbank_ai_grading_model_config_id`, calls OpenAI-compatible `/chat/completions` for JSON grading/analyze feedback when a usable provider is configured, falls back to deterministic local grading when no usable provider result is available, persists `ai_feedback`, `ai_score`, `ai_graded_at`, and grade-mode `is_correct`, updates the associated submission correctness, and resyncs the question into the hybrid VFS so resource refs include the latest AI feedback.

Important limitation: provider-backed grading is intentionally lean. It supports OpenAI-compatible non-streaming JSON responses and model assignment selection, but it does not yet stream token chunks, cancel an in-flight provider HTTP request, implement provider-specific Responses/reasoning protocols, use rubric libraries, or migrate richer old Rust grading state. The deterministic local scorer remains as the no-config/no-parse fallback.

Qbank sync/conflict compatibility now reports local Go qbank state without claiming cloud sync exists. `qbank_sync_check` returns the current Go question count as synced/local-clean state plus a persisted compact sync config; `qbank_get_sync_conflicts` returns an empty list because the Go JSON store currently has no remote conflict source; `qbank_batch_resolve_conflicts` returns an empty question list; `qbank_set_sync_enabled` persists `auto_sync` per exam in `qbank-go.json`; and `qbank_resolve_sync_conflict` fails clearly when asked for a non-existent local conflict.

Important limitation: this is not old Rust/cloud sync parity. It does not compare remote versions, materialize conflict records, push/pull remote qbank data, or run background sync. It only preserves the frontend sync-status/conflict-dialog contract for the local Go migration shell.

## Completed ReviewPlan Commands

Merged into Go `ReviewPlanService`:

- `review_plan_create`
- `review_plan_process`
- `review_plan_get_due`
- `review_plan_get_due_with_filter`
- `review_plan_get_stats`
- `review_plan_refresh_stats`
- `review_plan_get_by_question`
- `review_plan_get`
- `review_plan_suspend`
- `review_plan_resume`
- `review_plan_delete`
- `review_plan_get_history`
- `review_plan_batch_create`
- `review_plan_create_for_exam`
- `review_plan_list_by_exam`
- `review_plan_get_or_create`
- `review_plan_get_calendar_data`

Review plans are intentionally implemented as a compact Go JSON store in `review-plan-go.json`, with Qbank question/exam validation through the existing Go `QbankService`. The review algorithm uses an SM-2 style update path with local deterministic state: quality scores update ease factor, interval, repetitions, due date, difficult flags, review history, and aggregate stats without recreating the old Rust repository/database stack.

The frontend `ReviewPlanStore` now uses `src/runtime/native.ts`, so Wails, Tauri, injected native runtimes, and browser fallback all pass through the same facade. The Wails bridge maps the existing `review_plan_*` command names to `ReviewPlanService`, preserving frontend behavior while keeping the Go service smaller and workflow-oriented.

Important limitation: this slice does not migrate old Rust review-plan data, sync/conflict metadata, or any future cloud review state. It also does not attach full question payloads to due items; the current store keeps plan state only and expects existing Qbank/question UI paths to resolve question details when needed.

## Completed VFS Commands

Merged into Go `VfsService`:

- `vfs_create_or_reuse`
- `vfs_get_resource`
- `vfs_resource_exists`
- `vfs_increment_ref`
- `vfs_decrement_ref`
- `vfs_get_resource_path`
- `vfs_get_resource_ref_count`
- `vfs_update_resource_hash`
- `vfs_get_resource_refs`
- `vfs_resolve_resource_refs`
- `vfs_update_path_cache`
- `vfs_upload_attachment`
- `vfs_get_attachment`
- `vfs_get_attachment_content`
- `vfs_upload_file`
- `vfs_get_file`
- `vfs_delete_file`
- `vfs_get_file_content`
- `textbooks_update_bookmarks`
- `textbooks_add`
- `vfs_unified_index_status`
- `vfs_get_resource_units`
- `vfs_sync_resource_units`
- `vfs_get_all_index_status`
- `vfs_reindex_resource`
- `vfs_reindex_unit`
- `vfs_unified_batch_index`
- `vfs_batch_index_pending`
- `vfs_delete_resource_index`
- `vfs_list_embedding_dims`
- `vfs_list_dimensions`
- `vfs_get_resource_text_chunks`
- `vfs_get_resource_ocr_info`
- `vfs_clear_resource_ocr`
- `vfs_rag_search`
- `vfs_list_files`
- `vfs_create_mindmap`
- `vfs_get_mindmap`
- `vfs_get_mindmap_content`
- `vfs_update_mindmap`
- `vfs_delete_mindmap`
- `vfs_list_mindmaps`
- `vfs_set_mindmap_favorite`
- `vfs_get_mindmap_versions`
- `vfs_get_mindmap_version`
- `vfs_get_mindmap_version_content`
- `vfs_get_pdf_processing_status`
- `vfs_get_batch_pdf_processing_status`
- `vfs_cancel_pdf_processing`
- `vfs_retry_pdf_processing`
- `vfs_start_pdf_processing`
- `vfs_get_pdf_page_image`
- `vfs_get_blob_base64`
- `resource_sync_note`
- `resource_sync_exam`
- `resource_sync_textbook_pages`
- `resource_check_sync_needed`

The current Go VFS is a native hybrid resource index. Resource contents are stored as visible files under `vfs_resources`, while `vfs-go.json` stores IDs, hashes, source links, metadata, paths, and ref counts. This is the intended direction; do not replace it with the old all-virtual VFS blob stack.

Resource sync compatibility now creates or updates source-stable Go hybrid VFS records for `resource_sync_note`, `resource_sync_exam`, and `resource_sync_textbook_pages`, and `resource_check_sync_needed` compares source-linked hashes against the compact resource index. This is local compatibility only: it does not read old notes/exam/textbook database contents, backfill legacy `resources.db`, or recreate old table writeback semantics.

Important limitation: context refs currently resolve only resources already registered in the Go hybrid VFS index. Attachment upload/content is lightweight: it stores visible files, returns base64 content, and reports minimal image/PDF readiness. File upload/get/content/delete/bookmark/metadata commands now work over the same hybrid VFS; delete is a migration-safe soft delete that hides resources from active reads/search/listing without physically removing visible bytes yet. Mindmaps are now stored as real visible JSON files under `vfs_resources/mindmap/...` with `mm_` source IDs, metadata-backed favorites/settings/default view, soft delete, and `mv_` version snapshots created when content changes; this intentionally avoids recreating the old Rust `mindmaps`/`mindmap_versions` tables. DSTU can now expose VFS file/image/textbook resources to Learning Hub through the existing `dstu_*` commands, and `textbooks_add` now imports local files into the same Go hybrid VFS instead of a separate textbook store. Learning Hub file/textbook/image previews now prefer file-like VFS content reads and keep attachment content only as a compatibility fallback. Legacy virtual/resource URI compatibility now exists as alias parsing only: wrapped `resourceId`, `sourceId`, hash, DSTU path-tail IDs, and registered original paths can resolve to existing hybrid VFS records, but no virtual URI is stored as the primary file model. PDF/image processing status is now truthful but lightweight: images report `completed` with `image` ready, searchable PDFs report `text` plus generated `image` readiness when text-layer SVG previews exist, and PDFs without extracted text/preview evidence return terminal `completed_with_issues` without fake ready modes. PDF uploads now also try a lean built-in text-layer extraction pass for simple searchable PDFs, including uncompressed streams and `/FlateDecode` streams with literal or hex text strings, plus a lightweight page-count pass that detects `/Type /Page` objects or falls back to `/Pages /Count`. Extracted text flows into `extractedText`, resource units, OCR-info compatibility, local RAG search, estimated per-page `ocrPagesJson` entries marked with `ocrPagesSource: pdf_text_layer_estimated`, and generated visible SVG page preview files under `vfs_resources/pdf_previews/...` referenced by `previewJson` with `previewSource: pdf_text_layer_svg`; detected page counts flow into `pageCount`, upload OCR-status totals, and `vfs_get_pdf_processing_status` / progress `totalPages` and `currentPage`. `vfs_start_pdf_processing` and `vfs_retry_pdf_processing` now synchronously retry the same lightweight PDF text-layer/page-count/page-text parser and generate text-layer SVG previews for already indexed visible files that do not yet have preview metadata; `vfs_cancel_pdf_processing` remains an existence/status compatibility control because there is no background worker yet. `vfs_get_pdf_page_image` can read existing PDF preview page images from `previewJson` using inline base64/data URLs, registered page-image resources, or VFS-library paths, and now also reads generated text-layer SVG preview files, which supports RAG citation image rendering for simple searchable PDFs without reintroducing the old virtual blob table. `vfs_get_blob_base64` now reads indexed resource bytes by hash/source/resource aliases from the same visible `vfs_resources` library and hides soft-deleted records; it does not recreate the old Rust virtual blob table. Real PDF raster rendering, real OCR, robust font/CMap-aware PDF extraction, scanned-PDF handling, and richer textbook processing still need dedicated replacement work. Compact index status reports local text/file/mindmap units from the Go resource index. `vfs_rag_search` is currently an honest local fallback over titles, source IDs, metadata, tags, and text content; it is not semantic embedding/LanceDB search. The Go VFS still does not implement OCR, LanceDB, embedding search, multimodal indexing, folder expansion, or full semantic RAG yet.

## Recent Checkpoint

Latest implementation checkpoint:

- Added Go `VfsService` and `desktop-go/internal/vfs`.
- Registered `VfsService` in the Wails app.
- Routed the first 8 VFS resource-index commands through `src/runtime/wailsBridge.ts`.
- Switched `src/features/chat/resources/api.ts` from direct Tauri invoke to the native runtime facade.
- Switched `src/features/chat/resources/index.ts` to use the real resource API under Tauri, Wails, and injected native runtimes.
- Exported `isWailsRuntime` from `src/runtime/native.ts`.
- Updated command triage overrides for the VFS resource-index slice.
- Added Go tests for visible file storage, hydration, hash dedupe, ref counts, persistence, and hash updates.
- Added Go context-ref support for `vfs_get_resource_refs`, `vfs_resolve_resource_refs`, and compatibility no-op `vfs_update_path_cache`.
- Routed chat VFS ref APIs through `src/runtime/native.ts` instead of direct Tauri invoke while preserving frontend return shapes.
- Added Go tests for stable refs, resourceId fallback lookup, max-item truncation, resolved visible paths/content, missing resources, and path-cache no-op behavior.
- Added lightweight Go attachment flow for `vfs_upload_attachment`, `vfs_get_attachment`, and `vfs_get_attachment_content`.
- Attachment bytes are stored as visible files through the same VFS resource index, with `att_xxx` stable source IDs, SHA-256 content hashes, metadata, and base64 content reads.
- Moved attachment preview/content callers in learning hub, question bank, and PDF loading paths from direct Tauri invoke to `src/runtime/native.ts`.
- Added Go tests for attachment upload metadata, base64 content retrieval, visible file bytes, attachment dedupe, PDF readiness metadata, and not-found content reads.
- Added `VfsService.CreateOrUpdateSource` as an internal source-stable upsert path for product resources.
- Wired Go `DstuService` to the hybrid VFS so note create/update/Markdown import writes visible VFS resource files and stores the VFS `resourceId`/`resourceHash` on returned nodes.
- Added Go tests proving a DSTU note can be resolved through `vfs_get_resource_refs` / `vfs_resolve_resource_refs`, and that note updates keep the same VFS resource ID while changing hash/content.
- Wired Go `QbankService` to the same hybrid VFS so question create/update writes visible `exam` resource files and stores `resource_id`/`resource_hash` on returned questions.
- Added Go tests proving a Qbank question can be resolved through `vfs_get_resource_refs` / `vfs_resolve_resource_refs`, and that question updates keep the same VFS resource ID while changing hash/content.
- Merged lightweight attachment root config commands into Go `SettingsService`: `vfs_get_attachment_config`, `vfs_set_attachment_root_folder`, and `vfs_create_attachment_root_folder`.
- Routed `src/api/attachmentConfigApi.ts` through `src/runtime/native.ts`, with Wails bridge compatibility and a localStorage fallback for non-native development contexts.
- Added compact Go VFS index status/unit DTOs and Wails methods for `vfs_unified_index_status`, `vfs_get_resource_units`, `vfs_sync_resource_units`, `vfs_get_all_index_status`, no-op reindex/batch/delete, dimension list stubs, text chunk reads, and OCR-info stubs over the hybrid VFS index.
- Routed `src/api/vfsUnifiedIndexApi.ts` through `src/runtime/native.ts`, with Wails bridge compatibility and local fallback responses for non-native development contexts.
- Updated `scripts/native-triage.mjs` so compact index commands are tracked as merged Go surface instead of remaining in `replace`.
- Added Go VFS local text/metadata search for `vfs_rag_search`, ranked by title/source/type/metadata/text hits and filtered by resource type and folder metadata.
- Added Go VFS file listing for `vfs_list_files`, returning file/image/textbook/attachment-like resources with visible paths, hashes, sizes, MIME type, tags, bookmarks, favorite/status fields, and timestamps.
- Routed `src/api/vfsRagApi.ts` and `src/api/vfsFileApi.ts` through `src/runtime/native.ts` instead of direct Tauri invoke for the migrated search/list paths.
- Added Wails bridge and fallback responses for `vfs_rag_search` and `vfs_list_files`.
- Added Go VFS tests for local text/metadata search, type/folder filtering, and image attachment file listing.
- Added Go VFS file commands for `vfs_upload_file`, `vfs_get_file`, `vfs_get_file_content`, `vfs_delete_file`, and `textbooks_update_bookmarks`.
- File uploads now create `file_xxx` source IDs, store visible bytes under `vfs_resources`, return frontend-compatible `VfsFile` and upload status shapes, extract text for simple text/markdown/json/csv/xml files, and dedupe only file records rather than mixing with chat attachments.
- File delete is soft-delete metadata so migration recovery remains possible while active list/get/content/search paths hide deleted files.
- Routed `src/features/notes/PreviewPanel.tsx` file-content fallback through `src/runtime/native.ts` instead of direct Tauri invoke.
- Updated `scripts/native-triage.mjs` so file CRUD/content/bookmark commands are tracked as merged Go surface.
- Added Go VFS tests for text file upload, content retrieval, lightweight index status, bookmark update, and soft-delete hiding from get/content/list.
- Added metadata pass-through to Go `vfs.UploadFileInput` and `vfs.VfsFile`, plus `VfsService.UpdateFileMetadata` for file/image/textbook metadata and favorite/title/tag updates.
- Extended Go `DstuService` so existing `dstu_list`, `dstu_get`, `dstu_create`, `dstu_delete`, `dstu_get_content`, `dstu_set_metadata`, and `dstu_set_favorite` can operate on hybrid VFS-backed file/image/textbook nodes, not just notes.
- DSTU now maps VFS files into Learning Hub-compatible `DstuNode` shapes with `resourceId`, `resourceHash`, visible `filePath`/`originalPath`, MIME type, size, tags, favorite state, bookmarks, extracted text, and preview type.
- DSTU `create` for `file`, `image`, and `textbook` now writes visible bytes into the Go hybrid VFS through `vfs_upload_file` internals instead of creating a separate DSTU blob store.
- Go VFS file IDs remain source-stable `file_xxx` for uploads; textbook/image identity is carried by resource metadata and DSTU node type, avoiding a return to the old all-virtual VFS design.
- Added Go DSTU regression tests proving existing VFS uploads and image attachments are visible through DSTU, searchable by metadata/text, updatable via metadata/favorite writes, and hidden after soft delete.
- Added Go DSTU regression tests proving `dstu_create` for file/image/textbook stores resources in the Go hybrid VFS and keeps textbook resources out of the regular file filter.
- Added Go `DstuService.AddTextbooks` for the legacy `textbooks_add` command shape, importing local file paths as textbook resources through the Go hybrid VFS with source path, folder, resource ID/hash, MIME, status, and preview metadata.
- Routed `textbooks_add` through Wails `DstuService.AddTextbooks` in `src/runtime/wailsBridge.ts`, added a native-runtime fallback error, and moved `src/dstu/adapters/textbookDstuAdapter.ts` plus the legacy `src/utils/chatApi.ts` textbook shim away from direct Tauri invoke for this command.
- Updated generated Wails bindings for the new `AddTextbooksRequest` and `TextbookRecord` models.
- Updated `scripts/native-triage.mjs` so `textbooks_add` is tracked as a merged Go study-data surface.
- Added Go DSTU regression tests proving local textbook imports land in the hybrid VFS, list through DSTU as textbooks, keep folder metadata, expose visible content, and promote an existing same-hash regular file resource into a textbook without duplicating it.
- Added `vfsFileApi.getFileLikeContent` so Learning Hub previews first read `vfs_get_file_content` for file/image/textbook resources and fall back to `vfs_get_attachment_content` only for legacy attachment-shaped resources.
- Routed `usePdfLoader`, `TextbookContentView`, `FileContentView`, and `ImageContentView` through the file-like content facade, reducing direct invoke usage and making imported Go VFS textbooks/files open from the same hybrid VFS content path.
- Tightened Go `VfsService.GetAttachment` / `GetAttachmentContent` so soft-deleted file-like resources are not exposed through the old attachment compatibility commands.
- Added Go VFS regression coverage proving soft-deleted file content stays hidden from both `vfs_get_file_content` and old attachment compatibility reads.
- Added lightweight legacy URI alias parsing to Go `VfsService` lookup paths. `vfs://...`, `vfs-resource://...?resourceId=...`, `dstu:///folder/{sourceId}`, `resource://...?hash=...`, and registered original paths such as `content://...` now resolve to existing hybrid VFS resources when those resources are already indexed.
- Resource get/exists/ref-count/ref-resolve/file-content/attachment-content/bookmark/metadata/delete paths now share alias-aware lookup, while file/attachment reads are restricted to file-like resources and still hide soft-deleted records.
- Added Go VFS regression coverage proving legacy aliases resolve by resource ID, source ID, hash, and original path, and that soft-deleted file-like resources stay hidden through all those alias forms.
- Added lightweight Go/Wails compatibility for PDF/image processing status and controls: `vfs_get_pdf_processing_status`, `vfs_get_batch_pdf_processing_status`, `vfs_cancel_pdf_processing`, `vfs_retry_pdf_processing`, and `vfs_start_pdf_processing`.
- PDF/image status is derived from current hybrid VFS resource evidence instead of pretending the old Rust OCR/rendering pipeline exists. Images are terminal with `image` ready. PDFs are terminal `completed_with_issues`, expose `text` only when extracted text is already indexed, and leave `image`/`ocr` unavailable until real processing is built.
- `StartPdfProcessing` / `RetryPdfProcessing` record lightweight processing request metadata (`processingRequestedAt`, optional `processingStartStage`) without claiming background OCR or PDF rendering ran. `CancelPdfProcessing` reports whether an indexed file-like resource exists.
- Routed `src/api/vfsPdfProcessingApi.ts` through `src/runtime/native.ts`, added Wails bridge and local fallback coverage, and updated the Vitest mock to use the native facade instead of direct Tauri invoke.
- Regenerated Wails bindings so `PdfProcessingStatus`, `PdfProcessingProgress`, and the five processing methods are available to the frontend bridge.
- Updated `scripts/native-triage.mjs` so the PDF processing status/control commands are tracked as merged Go study-data surface. The batch command was also made scan-visible by extracting a shorter `BackendBatchStatusShape` alias in the frontend API.
- Added Go `VfsService.GetPdfPageImage` for the legacy `vfs_get_pdf_page_image` command shape, reading already-available preview page images from hybrid VFS metadata rather than invoking PDFium or background rendering.
- `GetPdfPageImage` supports existing `previewJson` page formats with camelCase/snake_case page indexes, `compressedBlobHash`/`blobHash`, inline base64/data URLs, registered image/blob resources, and paths inside the Go VFS library. Soft-deleted resources and missing pages return explicit errors.
- Routed `vfs_get_pdf_page_image` through Wails `VfsService.GetPdfPageImage`, added a native-runtime fallback error for non-native contexts, regenerated bindings for `PdfPageImageResult`, and updated command triage so this command is tracked as merged Go study-data surface.
- Added Go VFS regression tests proving page image reads from inline preview data, registered preview image resources, VFS-library preview paths, missing-page errors, and soft-delete hiding.
- Added Go `VfsService.GetBlobBase64` for the legacy `vfs_get_blob_base64` command shape, reading bytes from indexed hybrid VFS resources by hash/source/resource aliases.
- Blob reads now use the visible `vfs_resources` file library plus compact metadata for MIME/size/hash, and hide soft-deleted resources instead of reintroducing the old Rust virtual blob table.
- Routed `vfs_get_blob_base64` through Wails `VfsService.GetBlobBase64`, added a native-runtime fallback error for non-native contexts, moved `src/features/chat/context/blobApi.ts` from direct Tauri invoke to `src/runtime/native.ts`, regenerated bindings for `VfsBlobBase64Result`, and updated command triage so this command is tracked as merged Go study-data surface.
- Added Go VFS regression tests proving blob reads by hash and source ID return the same visible-file bytes and that soft-deleted file resources cannot be read through the blob compatibility API.
- Added lightweight PDF text-layer page compatibility: searchable PDF uploads and `vfs_start_pdf_processing` / `vfs_retry_pdf_processing` now derive estimated per-page `ocrPagesJson` from extracted text and detected page count, mark the metadata source as `pdf_text_layer_estimated`, and expose those page snippets through `GetResourceOcrInfo` without claiming real OCR.
- Updated Go VFS OCR-info behavior so real `ocrText` still wins as active source, extracted PDF text remains `activeSource: extracted`, and text-layer page snippets are visible in the existing page-result UI while `HasOcr` remains false for estimated text-layer pages.
- Added Go VFS regression coverage proving new PDF uploads expose text-layer page JSON, `GetResourceOcrInfo` returns page snippets without claiming OCR, and processing an existing indexed PDF backfills text-layer page info.
- Added generated PDF text-layer SVG previews in Go VFS. Searchable PDF uploads and `vfs_start_pdf_processing` / `vfs_retry_pdf_processing` now write visible SVG page preview files under `vfs_resources/pdf_previews/...`, store them in `previewJson`, mark `previewSource: pdf_text_layer_svg`, and report `image` ready when those previews exist.
- Updated `vfs_get_pdf_page_image` coverage so generated SVG preview paths are read back through the existing page-image command shape as `image/svg+xml`, preserving the hybrid VFS direction without adding PDFium or a virtual blob table.
- The SVG preview generator can also use existing `ocrPagesJson` page text when migrated data has page text but no `extractedText`, so `vfs_start_pdf_processing` can backfill image previews without falsely reporting `text` readiness.
- Added Go `DstuService.GetResourceLocation`, `GetResourceByPath`, and `GetFolderAllResources` for the legacy `dstu_get_resource_location`, `dstu_get_resource_by_path`, and `dstu_folder_get_all_resources` command shapes.
- DSTU folder/path lookup now derives resource location from note metadata and hybrid VFS file metadata, including `folderId`, `folderIds`, and `folderPathIds`. It builds stable compatibility paths as `/{folderId}/{resourceId}` without introducing the old folder database layer.
- Folder resource aggregation now returns `FolderResourcesResult` with `FolderResourceInfo` records for notes and Go VFS-backed file/image/textbook resources, optionally including content via the existing DSTU content path.
- Routed the three DSTU folder/path commands through Wails `DstuService`, moved only those three frontend calls in `src/dstu/api/folderApi.ts` and `src/dstu/api/pathApi.ts` to `src/runtime/native.ts`, and left unrelated folder CRUD/path move commands on their current backend path until they have Go implementations.
- Updated `scripts/native-triage.mjs` and `scripts/native-inventory.mjs` to count `nativeInvoke(...)` calls as native invoke references, so migrated facade calls do not disappear from command inventory.
- Regenerated Wails bindings for `ResourceLocation`, `FolderResourceInfo`, and `FolderResourcesResult`, bringing DstuService to 15 Wails methods.
- Added Go DSTU regression tests proving folder metadata location lookup, path lookup, folder resource aggregation with content, and recursive metadata-subfolder inclusion.
- Added Go Qbank CSV support in `desktop-go/internal/qbank/csv.go` for `get_csv_preview`, `import_questions_csv`, `export_questions_csv`, and `get_csv_exportable_fields`.
- CSV preview/import reads UTF-8, UTF-8 BOM, GBK, and GB18030 local files; CSV export writes UTF-8, UTF-8 BOM, or GBK files with spreadsheet formula-prefix neutralization.
- CSV import supports `skip`, `overwrite`, and `merge` duplicate strategies based on normalized content hash, maps the existing frontend field names into `Question` records, and syncs imported questions into the Go hybrid VFS as `exam` resources.
- Routed the four CSV commands through Wails `QbankService` in `src/runtime/wailsBridge.ts`, moved `CsvImportDialog` and `QuestionBankExportDialog` from direct Tauri invoke to `src/runtime/native.ts`, and kept Tauri-only `csv_import_progress` listening so Wails uses final-result progress without crashing on event setup.
- Added local fallback behavior in `src/runtime/native.ts`: CSV file-path commands report a native-runtime requirement outside native shells, while `get_csv_exportable_fields` returns the static field list for UI rendering.
- Declared `golang.org/x/text` as a direct Go dependency for GBK/GB18030 CSV compatibility.
- Updated `scripts/native-triage.mjs` so the CSV commands are tracked as merged Go study-data surface instead of remaining in `replace`.
- Regenerated Wails bindings for `CsvPreviewResult`, `CsvImportRequest`, `CsvImportResult`, `CsvExportRequest`, `CsvExportResult`, and `CsvDuplicateStrategy`, bringing QbankService to 18 Wails methods.
- Added Go Qbank regression tests proving CSV preview/import/VFS sync, duplicate skip/overwrite/merge behavior, exportable fields, UTF-8 BOM export, and CSV formula neutralization.
- Added Go mindmap support in `desktop-go/internal/vfs/mindmap.go` for `vfs_create_mindmap`, `vfs_get_mindmap`, `vfs_get_mindmap_content`, `vfs_update_mindmap`, `vfs_delete_mindmap`, `vfs_list_mindmaps`, `vfs_set_mindmap_favorite`, `vfs_get_mindmap_versions`, `vfs_get_mindmap_version`, and `vfs_get_mindmap_version_content`.
- Mindmaps now use the Go hybrid VFS directly: current documents are visible JSON files under `vfs_resources/mindmap/...`, `vfs-go.json` carries the compact metadata/index state, `mm_` source IDs remain stable, and legacy URI/source/hash alias lookup can resolve them through the existing resource lookup path.
- Mindmap updates support `expectedUpdatedAt` optimistic conflict checks, metadata updates (`title`, `description`, `defaultView`, `theme`, `settings`), favorites, and soft delete. Content updates create `mv_` version resources that preserve the prior document JSON without introducing a separate old-style table.
- Routed the mindmap commands through Wails `VfsService` in `src/runtime/wailsBridge.ts`, added non-native fallback behavior in `src/runtime/native.ts`, and moved `src/features/mindmap/api/mindmapApi.ts` plus `MindMapEmbed.tsx` from direct Tauri invoke to the native facade.
- Updated `scripts/native-triage.mjs` so the mindmap commands are tracked as merged Go study-data surface instead of remaining in `replace`.
- Regenerated Wails bindings for `CreateMindMapInput`, `UpdateMindMapInput`, `VfsMindMap`, `VfsMindMapVersion`, and the 10 mindmap VFS methods, bringing VfsService to 58 Wails methods and the app total to 164 methods / 111 models.
- Added Go VFS regression tests proving mindmap visible JSON storage, content reads, active listing, update-generated versions, version metadata/content lookup, optimistic conflict detection, favorite persistence, and soft-delete hiding.
- Added compact Go `SettingsService` support for LLM configuration commands: API configurations, vendor configs, model profiles, and model assignments.
- The settings LLM config store preserves frontend JSON shapes such as `baseUrl`, `websiteUrl`, `translation_display_mode`, and all current model assignment keys including `review_analysis_model_config_id`.
- Model config normalization now trims stable IDs/models/base URLs, fills conservative defaults for model adapter, max output tokens, temperature, Gemini API version, provider type, profile status, and clears empty optional assignment IDs.
- Routed the eight LLM settings commands through Wails `SettingsService` in `src/runtime/wailsBridge.ts`, added localStorage fallback behavior in `src/runtime/native.ts`, and moved model/settings UI call sites from direct Tauri invoke to the native facade.
- Regenerated Wails bindings for `ApiConfig`, `VendorConfig`, `ModelProfile`, `ModelAssignments`, and the eight settings methods, bringing the app total to 172 methods / 115 models.
- Added Go settings regression tests proving API config, vendor config, model profile, and assignment persistence across service reloads plus normalization/default behavior.
- Added a Wails-native event facade in `src/runtime/nativeEvents.ts` that maps `@wailsio/runtime` `Events.On` payloads into the existing Tauri-style `{ event, payload }` shape, while preserving Tauri event support.
- Switched `ChatV2TauriAdapter` event setup and save-session runtime guard from Tauri-only detection to native event runtime detection, so Wails can register `chat_v2_event_{sessionId}` and `chat_v2_session_{sessionId}` listeners.
- Extended the Go runtime `EventBus` with wildcard listeners and wired `desktop-go/cmd/deep-student-go/main.go` to forward all internal events to Wails via `wailsApp.Event.Emit`.
- Connected Go `ChatService` to the runtime event bus and made local send/retry placeholder responses emit a frontend-compatible stream sequence after persistence: session `stream_start`, content block `start/chunk/end`, and session `stream_complete`.
- Added Go regression tests for wildcard event dispatch and `SendMessage` local stream event emission, proving channels, payload types, block IDs, chunks, and terminal events are emitted.
- Added a minimal Go OpenAI-compatible chat provider path. `ChatService` now receives API configs from `SettingsService` through a loader callback, resolves the runtime `model2OverrideId` / `modelId`, sends text-only `/chat/completions` SSE requests with configured API key/base URL/model/headers plus temperature/top_p/max_tokens, parses delta content chunks, emits them over the Wails/Tauri-compatible stream event bridge, and persists the final assistant block.
- Reused the same provider stream path for `chat_v2_retry_message` by replaying the previous user message content into the provider request, while keeping the local placeholder fallback when no enabled provider config is available.
- Earlier provider-slice note: `chat_v2_edit_and_resend` initially stayed on a local placeholder because the frontend established the new assistant message stream expectation after invoke returned; that temporary limitation is superseded by the later edit-resend preallocation bullets in this checkpoint.
- Changed Go chat stream `sequenceId` values to start at 1 because JSON `omitempty` can drop zero-valued sequence IDs and confuse the frontend ordered event bridge.
- Hardened the Go chat JSON store flush on Windows by removing the old `chat-go.json` before renaming the temp file; this fixed an `Access is denied` failure observed during chat tests. The same atomic-write pattern exists in other Go JSON stores and should be reviewed separately rather than expanded inside the provider slice.
- Added per-message active stream tracking in Go `ChatService`. Provider send/retry now run under cancellable contexts, `chat_v2_cancel_stream` invokes the active cancel function before marking streaming blocks cancelled, and provider cancellation is persisted as block status `cancelled` while emitting frontend-supported `stream_cancelled`.
- Added Go regression coverage proving a blocked OpenAI-compatible provider request receives context cancellation, `SendMessage` returns after cancellation, the assistant block is persisted as `cancelled`, and `stream_cancelled` is emitted.
- Extended `chat_v2_continue_message` to accept frontend send options through the Wails bridge and bindings, append a streaming continuation block to the existing assistant message, replay the previous user prompt through the same OpenAI-compatible provider stream path, and persist/emits chunks for the continuation block. This path is safe for synchronous backend streaming because the frontend establishes the stream expectation for the same assistant message ID before invoking the backend.
- Added Go regression coverage proving continue makes a second provider request with the original user prompt, appends rather than replaces assistant blocks, persists the continuation block content/status, and emits the continued chunk event.
- Replaced the `chat_v2_edit_and_resend` local placeholder with real provider streaming. The frontend adapter now preallocates the replacement assistant message ID before invoking Wails, begins the stream expectation immediately, and passes that ID to Go as `assistantMessageId`.
- Go `ChatService.EditAndResend` now uses the frontend-provided assistant ID when present, deletes old messages after the edited user message, updates the user block/context snapshot, creates a new streaming markdown block, releases the chat store lock, then reuses `runAssistantStream` for OpenAI-compatible provider streaming or local fallback streaming.
- Hardened the frontend edit-resend store merge so a `stream_start` placeholder that arrives before the invoke returns is preserved rather than overwritten with an empty `blockIds` array. The post-callback state update only marks `currentStreamingMessageId` when the session is still streaming, avoiding stale streaming state after synchronous Wails completion.
- Regenerated Wails bindings so `EditAndResendRequest` includes `assistantMessageId`.
- Added Go regression coverage proving edit-resend makes a second provider request with the edited prompt, deletes the old assistant, uses the preallocated new assistant ID, persists the provider output/status/model metadata, updates the edited user block, and emits stream start/chunk events for the replacement assistant.
- Added message-level token usage capture for the lean Go OpenAI-compatible provider stream. Requests now include `stream_options.include_usage`, the SSE reader preserves final usage-only chunks before `[DONE]`, and Go normalizes OpenAI snake_case fields into the frontend `TokenUsage` shape: `promptTokens`, `completionTokens`, `totalTokens`, `source: api`, `lastRoundPromptTokens`, optional `reasoningTokens`, and optional `cachedTokens`.
- Persisted provider usage to assistant message `_meta.usage` and emitted the same usage payload on `stream_complete`, letting existing frontend `handleStreamComplete`, `TokenUsageDisplay`, and context-window usage UI work over Go chat streams without adding the old global usage subsystem.
- Added Go regression coverage proving the provider request asks for usage, the assistant message stores normalized usage, and the final session event emits the same normalized usage.
- Added Go `ChatService.ClearApprovalHistory` plus Wails binding/bridge support for legacy `chat_v2_clear_approval_history`, so the settings page can clear remembered tool approvals through the Go shell instead of the old Tauri command.
- Updated native command triage overrides so `chat_v2_clear_approval_history` is tracked as merged Go chat surface rather than a generic settings merge item.
- Extended the interaction regression test to prove remembered approval choices are counted and cleared.
- Added Go-derived LLM usage statistics over persisted Chat assistant `_meta.usage`, intentionally avoiding a standalone old-style usage database. `ChatService` now exposes `LLMUsageGetTrends`, `LLMUsageByModel`, `LLMUsageByCaller`, `LLMUsageSummary`, `LLMUsageRecent`, `LLMUsageDaily`, and `LLMUsageCleanup`.
- Routed legacy `llm_usage_get_trends`, `llm_usage_by_model`, `llm_usage_by_caller`, `llm_usage_summary`, `llm_usage_recent`, `llm_usage_daily`, and `llm_usage_cleanup` through the Wails bridge to the Go Chat service, and moved `src/api/llmUsageApi.ts` from direct Tauri invoke to the native facade.
- The usage compatibility path aggregates request counts, prompt/completion/total tokens, reasoning/cached token totals when present, model summaries, caller summaries, daily rows, trends, and recent calls from Chat message metadata. Cleanup removes old usage metadata from assistant messages before the requested date without deleting chat messages.
- Updated native triage overrides so the seven `llm_usage_*` commands move from `replace` to `merge`.
- Added Go regression coverage proving summary/model/caller/recent/daily/trend aggregation and cleanup behavior over Chat message usage metadata.
- Added lightweight Go PDF text-layer extraction in `desktop-go/internal/vfs/pdf_text.go`, used by `vfs_upload_file`/`UploadFile` when the uploaded file is a PDF.
- The extractor handles simple searchable PDF content streams, including uncompressed and `/FlateDecode` streams, literal strings, hex strings, UTF-8-ish byte strings, and UTF-16BE/LE PDF strings. This is intentionally a lean extraction path, not OCR or a full PDF engine.
- PDF uploads with extractable text now populate `extractedText`, report `text` as a ready mode in `vfs_get_pdf_processing_status`, expose the text through `vfs_get_resource_ocr_info`/resource units/text chunks, and become searchable through the current local `vfs_rag_search` fallback.
- Added Go regression coverage proving PDF text-layer extraction flows through upload metadata, index status, processing status, OCR-info compatibility, RAG search, and Flate/UTF-16BE text streams.
- Extended `vfs_start_pdf_processing` / `vfs_retry_pdf_processing` so already indexed PDF files without `extractedText` retry the same lightweight text-layer parser against the visible hybrid VFS file bytes. Successful retries record `textExtractionStatus: completed`, `textExtractionSource: pdf_text_layer`, and populate `extractedText`; missing text layers record `no_text_layer` without faking OCR success.
- Added lightweight Go PDF page-count detection in `desktop-go/internal/vfs/pdf_text.go`, counting `/Type /Page` objects first and falling back to page-tree `/Count` values when page objects are not visible in the byte stream.
- PDF upload now writes detected `pageCount` / `pageCountStatus` metadata, exposes total pages in the upload OCR-status compatibility shape, and mirrors `totalPages` / `currentPage` into `vfs_get_pdf_processing_status` and its nested progress object.
- Extended `vfs_start_pdf_processing` / `vfs_retry_pdf_processing` so already indexed PDFs missing `pageCount` retry page-count detection from the visible hybrid VFS bytes alongside the text-layer parser, recording `pageCountStatus` as `completed`, `unknown`, or `failed` without pretending rendering/OCR occurred.
- Added Go regression coverage proving PDF upload page-count metadata/status, page-tree fallback count detection, and start/retry page-count backfill for an existing indexed PDF resource.
- Added `desktop-go/internal/storage.WriteJSONAtomic` and moved Chat, DSTU, Notes, Qbank, Settings, Todo, and VFS JSON stores onto the shared atomic writer. The helper first tries normal atomic rename, then falls back to delete-and-rename on platforms such as Windows where replacing an existing destination can return `Access is denied`.
- The shared JSON writer fixes the intermittent Windows test/runtime failure previously seen in DSTU/Qbank flush paths (`rename *.tmp -> *.json: Access is denied`) and removes duplicated store-specific temp-file write logic.
- Added Go `DstuService.NotesSearch`, `ListTags`, `CanvasReadContent`, `CanvasAppendContent`, `CanvasReplaceContent`, and `CanvasSetContent` for the legacy notes utility command shapes.
- Notes search/list-tags now use the current Go DSTU note store: `notes_search` supports simple all-term matching plus `tag:<name>` filters and snippets, while `notes_list_tags` returns frequency-sorted tags capped at 50.
- Canvas note helpers now read full note content or Markdown sections, append before the next same-or-higher Markdown heading, perform literal and regex replacements with replacement counts, and overwrite note content through the normal DSTU update path so hybrid VFS note refs stay synchronized.
- Routed `notes_search`, `notes_list_tags`, `canvas_note_read`, `canvas_note_append`, `canvas_note_replace`, and `canvas_note_set` through Wails `DstuService` in `src/runtime/wailsBridge.ts`, with local non-native fallbacks returning empty search/tag lists and explicit runtime errors for Canvas read/write commands.
- Regenerated Wails bindings for `NotesSearchHit` and the six DstuService notes/canvas methods, bringing the app total to 186 methods / 122 models.
- Updated `scripts/native-triage.mjs` so these six commands are tracked as merged Go study-data surface.
- Added Go DSTU regression tests proving tag frequency/order, tag-filtered and term search with snippets, Markdown section extraction, section append boundaries, literal/regex replacement counts, full-content overwrite, and post-mutation hybrid VFS ref content synchronization.
- Added Go `QbankService` learning-loop support for `qbank_get_history`, `qbank_get_submissions`, `qbank_get_learning_trend`, `qbank_get_activity_heatmap`, `qbank_get_knowledge_stats`, `qbank_get_knowledge_stats_with_comparison`, `qbank_start_timed_practice`, `qbank_generate_mock_exam`, `qbank_submit_mock_exam`, `qbank_get_daily_practice`, `qbank_generate_paper`, and `qbank_get_check_in_calendar`.
- Qbank history is now recorded for user updates and answer/progress changes in the Go JSON store, and `qbank_submit_answer` resyncs the hybrid VFS question resource after changing answer/status/attempt metadata.
- Learning trend, activity heatmap, check-in calendar, and daily completion counts are derived from Go `AnswerSubmission` records; knowledge stats are derived from tags plus current status/attempt aggregates.
- Timed practice, daily practice, mock exam generation, mock exam scoring, and paper generation now provide Wails-compatible local workflows using stable deterministic selection over the Go question store instead of recreating the old Rust SQLite query layer.
- Routed the twelve qbank learning-loop commands through `QbankService` in `src/runtime/wailsBridge.ts`, added local non-native fallbacks for read-only chart/history/calendar shapes, and changed `QuestionHistoryView` to use `src/runtime/native.ts` instead of direct Tauri invoke.
- Regenerated Wails bindings for the qbank learning-loop DTOs and methods, bringing the app total to 198 methods / 149 models.
- Updated `scripts/native-triage.mjs` so these twelve qbank commands are tracked as merged Go study-data surface; `qbank_generate_mock_exam` and `qbank_submit_mock_exam` moved out of `delete`.
- Added Go qbank regression tests proving zero-filled learning trends, heatmap levels, knowledge stats, comparison shape, timed practice, daily practice source distribution/completion counts, generated paper filtering and answer/explanation stripping, check-in calendar aggregation, update/answer history, submissions readback, mock exam scoring breakdowns, and VFS hash sync after answer submission.
- Added Go `ReviewPlanService` and `desktop-go/internal/reviewplan` for the `review_plan_*` spaced-repetition command surface.
- Review plans are stored in compact `review-plan-go.json` state with plan rows and review history rows, validating question/exam ownership through the current Go Qbank service instead of copying the old Rust persistence stack.
- Added SM-2 style review processing for quality scores 0-5, including ease factor, interval, repetitions, next review date, difficult flags, pass/fail history, aggregate stats, due filters, suspend/resume/delete, list-by-exam, batch create, create-for-exam, get-or-create, per-question lookup, and calendar heatmap data.
- Registered `ReviewPlanService` in the Go app and Wails service list, routed the existing `review_plan_*` commands through `src/runtime/wailsBridge.ts`, moved `src/stores/reviewPlanStore.ts` from direct Tauri invoke to `src/runtime/native.ts`, and added non-native fallback shapes for read-only review views.
- Regenerated Wails bindings for `ReviewPlanService`, bringing the app total to 10 services, 215 methods, and 157 models.
- Updated `scripts/native-triage.mjs` so the seventeen `review_plan_*` commands are tracked as merged Go study-data surface, moving triage counts to merge 266 and replace 114.
- Added Go reviewplan regression tests proving create validation, duplicate prevention, batch/create-for-exam behavior, due filters, SM-2 review processing/history/stats, suspend/resume/delete, get-or-create, list-by-exam, and calendar heatmap aggregation.
- Added lean Go `QbankService.AIGrade` / `CancelGrading` for the legacy `qbank_ai_grade` and `qbank_cancel_grading` command shapes.
- Qbank AI grading now uses the existing Go runtime event bus to emit `qbank_grading_stream_{sessionId}` `data`, `complete`, `error`, and `cancelled` payload shapes, and `useQbankAiGrading` now listens through `src/runtime/nativeEvents.ts` instead of direct Tauri events.
- The Go grader validates `submission_id` ownership before writing, persists `ai_feedback`, `ai_score`, `ai_graded_at`, grade-mode `is_correct`, and submission correctness, then resyncs the question into the hybrid VFS so AI feedback is available to resource refs/search.
- The current Go grading body is deterministic local compatibility: normalized exact match yields full credit, otherwise reference-token coverage determines `correct` / `partial` / `incorrect`, and analyze mode returns a local Markdown analysis from reference answer/explanation/tags.
- Regenerated Wails bindings for `QbankGradingRequest` and `QbankGradingResponse`, bringing the app total to 10 services, 217 methods, and 159 models.
- Updated `scripts/native-triage.mjs` so `qbank_ai_grade` and `qbank_cancel_grading` are tracked as merged Go study-data surface, moving triage counts to merge 268 and replace 112.
- Added Go qbank regression coverage proving AI grading emits stream events, persists AI fields and correctness, updates the submission, and syncs AI feedback into the hybrid VFS resource.
- Added provider-backed Go qbank grading: `QbankService` now loads Settings API configs plus `qbank_ai_grading_model_config_id`, prefers a request `model_config_id` when supplied, calls OpenAI-compatible non-streaming `/chat/completions`, parses JSON `verdict` / `score` / `feedback`, and keeps the deterministic local scorer as the no-config/no-parse fallback.
- Wired `desktop-go/internal/app.New` so `QbankService` receives the saved API configuration surface and qbank model assignment without changing frontend command parameters or Wails method count.
- Refactored `AIGrade` so provider calls happen outside the qbank store mutex, then reacquire the store lock only for validation, persistence, VFS resync, and event emission.
- Added Go qbank regression coverage with an httptest provider proving `qbank_ai_grade` uses the assigned qbank model config, sends question/user/reference context to `/chat/completions`, persists provider feedback and score, updates submission correctness, emits the normal data/complete events, and syncs provider feedback into the hybrid VFS resource.
- Added local Go qbank sync/conflict compatibility for `qbank_sync_check`, `qbank_get_sync_conflicts`, `qbank_resolve_sync_conflict`, `qbank_batch_resolve_conflicts`, and `qbank_set_sync_enabled`.
- The sync compatibility path is intentionally local and honest: Go qbank has no remote source yet, so status reports all current local Go questions as clean/synced, conflict reads return empty lists, batch resolve returns an empty list, and single-conflict resolve errors if a stale/non-existent conflict ID is passed.
- `qbank_set_sync_enabled` persists compact per-exam sync config in `qbank-go.json` so the frontend setting survives reloads without adding the old cloud sync stack.
- Regenerated Wails bindings for `SyncConfig`, `SyncStatusResult`, `QuestionVersion`, and `SyncConflict`, bringing the app total to 10 services, 222 methods, and 163 models.
- Updated `scripts/native-triage.mjs` so the five qbank sync/conflict commands are tracked as merged Go study-data surface, moving triage counts to merge 273 and replace 107.
- Added Go qbank regression coverage proving local sync status counts current questions, conflicts are empty, batch resolve is empty, and `setSyncEnabled` persists across service reloads.
- Added Go `VfsService.ResourceSyncNote`, `ResourceSyncExam`, `ResourceSyncTextbookPages`, and `ResourceCheckSyncNeeded` for the legacy `resource_sync_*` and `resource_check_sync_needed` command shapes.
- Resource sync now writes source-stable note/exam/textbook compatibility resources into the Go hybrid VFS using `CreateOrUpdateSource`, visible files under `vfs_resources`, compact metadata, and hash comparison through the resource index.
- Routed the four resource sync commands through Wails `VfsService` in `src/runtime/wailsBridge.ts`, added local non-native fallback records in `src/runtime/native.ts`, and moved `src/services/resourceSyncService.ts` from direct Tauri invoke to the native facade while keeping existing exported service names for frontend compatibility.
- Regenerated Wails bindings for `ResourceSyncResult`, `CheckSyncNeededResponse`, and the four resource sync methods, bringing the app total to 10 services, 226 methods, and 165 models.
- Updated `scripts/native-triage.mjs` so the four resource sync commands are tracked as merged Go study-data surface, moving triage counts to merge 277 and replace 103.
- Fixed Go `TodoService` local-day stats so UTC `CreatedAt` / `CompletedAt` timestamps are parsed and compared as local dates for active summary and Pomodoro today stats. This was surfaced by the broader `go test ./...` verification and is independent of the resource sync slice.
- Added Go VFS regression coverage proving resource sync creates/reuses note refs by source ID, compares matching and mismatched hashes, creates range-stable textbook page resources, and rejects invalid page ranges. Added Go Todo regression coverage through existing tests for local-day active summary and Pomodoro today stats.
- Added a lean Go DSTU folder store inside `dstu-go.json`: `folders` and `folderItems` now replace the old folder CRUD/path command surface without recreating the Rust SQLite repository layer.
- Implemented Go folder CRUD/tree/item/path commands for `dstu_folder_create`, `dstu_folder_get`, `dstu_folder_rename`, `dstu_folder_delete`, `dstu_folder_move`, `dstu_folder_set_expanded`, `dstu_folder_add_item`, `dstu_folder_remove_item`, `dstu_folder_move_item`, `dstu_folder_list`, `dstu_folder_get_tree`, `dstu_folder_get_items`, `dstu_folder_reorder`, `dstu_folder_reorder_items`, `dstu_folder_get_breadcrumbs`, `dstu_parse_path`, `dstu_build_path`, `dstu_move_to_folder`, `dstu_batch_move`, `dstu_refresh_path_cache`, and `dstu_get_path_by_id`.
- `folderItems` is now the preferred position source for DSTU resources. Moving notes updates note metadata and resyncs the note resource into the Go hybrid VFS. Moving file/image/textbook/mindmap-style VFS resources updates Go hybrid VFS metadata through `UpdateFileMetadata`.
- DSTU resource location and folder aggregation now prefer folder item records and fall back to legacy metadata-only placement, so old Go metadata slices and new folder store entries can coexist during migration.
- Routed all implemented folder/path commands through Wails `DstuService`, switched `src/dstu/api/folderApi.ts` and `src/dstu/api/pathApi.ts` to `src/runtime/native.ts`, and added a small localStorage fallback for non-native development contexts.
- Updated `scripts/native-triage.mjs` so the new folder/path command surface is tracked as merged study-data work, moving triage counts to merge 298 and defer 204.
- Added Go DSTU regression coverage for folder CRUD/tree/breadcrumb persistence, note move/path parsing/building, cycle rejection, VFS file metadata sync, folder resource aggregation, batch move failures, item reorder, path-cache refresh, and delete-folder moving items safely to root.
- Added Go `AnkiService` and `desktop-go/internal/anki` as a lean document task/status/control compatibility shell over `anki-go.json`.
- Registered `AnkiService` in the Go app and Wails service list, and routed `get_document_tasks`, `pause_document_processing`, `resume_document_processing`, `get_document_processing_state`, `get_document_state`, `get_document_task_counts`, `trigger_task_processing`, `delete_document_session`, `get_document_cards`, and `recover_stuck_document_tasks` through `src/runtime/wailsBridge.ts`.
- Switched the Anki task dashboard, CardForge controller/agent command calls, Anki API adapter cleanup, Anki cards block reads, and the ChatAnki debug persistence read from direct Tauri invoke to `src/runtime/native.ts` for this status/control surface.
- Added non-native fallback shapes in `src/runtime/native.ts` for read-only Anki document task/card/state/count commands and no-op control compatibility, without adding a fake document processing worker.
- Updated `scripts/native-triage.mjs` so the ten Anki task/status/control commands are tracked as merged study-data surface while `start_enhanced_document_processing` remains deferred.
- Added Go regression coverage proving missing documents return empty/pending shapes, pause/resume creates and clears paused state, retry trigger resets a matched failed task, delete removes a session, stuck processing/streaming tasks recover to pending, and mutating controls reject blank document IDs.
- Extended Go `AnkiService` with `StartEnhancedDocumentProcessing` for the legacy `start_enhanced_document_processing` command shape.
- The lean start worker validates plain text, creates a source-local document session, splits content into bounded text segments, creates pending tasks, and asynchronously processes pending tasks so the frontend can set the returned `documentId` before card events arrive.
- The worker emits existing CardForge-compatible `anki_generation_event` payloads through the Go runtime event bus: `DocumentProcessingStarted`, `TaskStatusUpdate`, `NewCard`, `TaskCompleted`, `TaskProcessingError`, `DocumentProcessingPaused`, and `DocumentProcessingCompleted`.
- Generated cards are local deterministic learning cards derived from text units in the segment and persisted in `anki-go.json`; they include legacy snake_case fields such as `task_id`, `front`, `back`, `tags`, `images`, `is_error_card`, `extra_fields`, `template_id`, `created_at`, and `updated_at`.
- Resume and retry now restart runnable pending tasks when task content exists, while empty compatibility sessions remain synchronous no-op controls to avoid background writer races.
- Routed `start_enhanced_document_processing` through Wails `AnkiService.StartEnhancedDocumentProcessing`, added Wails bridge argument aliases for camelCase and snake_case frontend calls, and changed the local non-native fallback to throw a clear native-runtime error.
- Extended `src/runtime/nativeEvents.ts` with a Wails/Tauri-compatible `emit` and switched CardAgent plus the Anki API adapter's temporary card collector from direct Tauri event listening to the native event facade for Anki generation events.
- Regenerated Wails bindings, bringing the app total to 11 services, 258 methods, and 176 models.
- Updated `scripts/native-triage.mjs` so `start_enhanced_document_processing` is tracked as merged study-data surface, moving triage counts to merge 308 and defer 196.
- Added Go regression coverage proving start creates tasks, emits completion, persists generated cards, and reaches completed document state; blank document content is rejected.
- Added provider-backed generation inside Go `AnkiService`: the worker can load Settings API configs plus `anki_card_model_config_id`, prefer command-supplied model/config IDs, call OpenAI-compatible non-streaming `/chat/completions`, parse JSON `{cards:[...]}` output, normalize cards into CardForge-compatible fields, and keep the deterministic local generator as a no-config/no-parse fallback.
- Wired `desktop-go/internal/app.New` so `AnkiService` receives the same saved API configuration surface as Go chat, without adding new frontend command parameters or Wails methods.
- Added Go regression coverage with an httptest provider proving `start_enhanced_document_processing` uses the assigned Anki model config, sends a chat-completions request, persists provider cards with provider metadata, preserves tags, and emits the normal completion flow.
- Added Go DSTU note soft-delete/trash lifecycle over `dstu-go.json`: active note reads/search/list/tag scans hide deleted notes, deleted metadata synchronizes into Go hybrid VFS note resources, restore clears deleted state, and purge removes note bodies plus associated active folder items.
- Added batch and alias trash command coverage for `dstu_delete_many`, `dstu_restore`, `dstu_restore_many`, `dstu_purge`, `dstu_purge_all`, `dstu_list_deleted`, `dstu_soft_delete`, `dstu_trash_restore`, `dstu_list_trash`, `dstu_empty_trash`, and `dstu_permanently_delete`.
- Routed legacy Notes trash wrappers (`notes_list_deleted`, `notes_empty_trash`, `notes_hard_delete`, `notes_restore`) through the same Go DSTU trash semantics instead of maintaining a separate note-deletion model.
- Added Go Notes asset maintenance and store utility commands: `notes_assets_index_scan`, `notes_assets_scan_orphans`, `notes_assets_bulk_delete`, `notes_db_stats`, and `notes_db_vacuum`.
- `notes_assets_scan_orphans` scans current DSTU note content references, including deleted Go notes, before declaring assets orphaned so trash recovery does not lose still-referenced assets.
- Moved `src/dstu/api/trashApi.ts` from direct `@tauri-apps/api/core` invoke to the native facade and routed the new command shapes through `src/runtime/wailsBridge.ts`.
- Fixed the DSTU restore path to release the store write lock before applying stored folder metadata to the VFS-backed node, avoiding a read/write self-deadlock.
- Regenerated Wails bindings, bringing the app total to 11 services, 274 methods, and 177 models.
- Updated `scripts/native-triage.mjs` so the DSTU trash and Notes asset-maintenance commands are tracked as merged study-data surface, moving triage counts to merge 328, replace 93, defer 185, delete 37.
- Added Go regression coverage proving soft delete hides notes from normal DSTU/Notes/VFS paths, restore re-exposes them, purge removes them, Notes wrappers share the same lifecycle, asset scans protect referenced assets, bulk asset delete works, and stats/vacuum report the JSON-backed store state.
- Added lean Go Notes zip import/export under `desktop-go/internal/notes/import_export.go` and `desktop-go/internal/dstu/import_export.go`, replacing `notes_export`, `notes_export_single`, and `notes_import` without copying the old Rust exporter/importer internals.
- Export now snapshots active Go DSTU notes into readable Markdown files plus `_notes.json`, writes `manifest.json` and `README.md`, and includes visible `notes_assets/...` files referenced by content or listed under each exported note ID.
- Import now reads the Go zip format, restores `notes_assets/...` files into the visible Go notes assets tree, upserts imported notes into `dstu-go.json`, supports `skip`, `overwrite`, and `merge_keep_newer`, and resyncs imported/overwritten note resources into the Go hybrid VFS.
- Routed `notes_export`, `notes_export_single`, and `notes_import` through Wails `NotesService` in `src/runtime/wailsBridge.ts`.
- Regenerated Wails bindings, bringing the app total to 11 services, 277 methods, and 182 models.
- Updated `scripts/native-triage.mjs` so Notes zip import/export is tracked as merged study-data surface, moving triage counts to merge 331, replace 90, defer 185, delete 37.
- Added Go regression coverage proving zip export contains manifest/metadata/Markdown/assets, import archive reads metadata and restores assets, DSTU import writes notes into `dstu-go.json`, skip preserves existing content, `merge_keep_newer` overwrites newer records, VFS refs are produced, and deleted notes are hidden from normal export unless explicitly requested.
- Added Go `QbankService.NotesMentionsSearch` for the legacy `notes_mentions_search` command, replacing the old Rust Notes/mistakes lookup with lean search over current Go qbank questions.
- The Go mention result preserves the existing `mistakes` and `irec_cards` response shape, supports optional subject/exam filtering, ranks question label/content/user note/explanation/answer/tags, returns stable card IDs when present, and links hits back to the backing question ID as `mistake_id`.
- Routed `notes_mentions_search` through Wails `QbankService.NotesMentionsSearch` in `src/runtime/wailsBridge.ts`; old Rust `src-tauri/src/cmd/notes.rs:notes_mentions_search` is now reference-only for this path.
- Regenerated Wails bindings, bringing the app total to 11 services, 278 methods, and 185 models.
- Updated `scripts/native-triage.mjs` so `notes_mentions_search` is tracked as merged study-data surface, moving triage counts to merge 332, replace 89, defer 185, delete 37.
- Added Go regression coverage proving all-scope mention search, subject-filtered search, card-ID preservation, backing question IDs, keyword snippets, and blank-query empty results.
- Added `scripts/rust-retirement-map.mjs` to generate `docs/generated/rust-retirement-map.json` and `.md`, cross-checking native triage, Wails bridge routing, old Rust registrations/definitions, and frontend direct Tauri invoke blockers.
- Current retirement map shows 332 merged commands, 274 merged commands with explicit Wails bridge routes, 265 Rust retirement candidates still present in old Rust, 61 merged commands still blocked by at least one direct Tauri frontend caller, 71 direct Tauri blocker edges across 22 frontend files, and 76 replace commands still registered in Rust.
- Moved `src/voice-input/runtimeConfig.ts` from direct `@tauri-apps/api/core` invoke to `@/runtime/native` and changed runtime detection from Tauri-only to injected/Tauri/Wails native runtime, so Wails can load Go `get_model_assignments` and `get_api_configurations` for voice input model selection.
- The voice-input slice removes `src/voice-input/runtimeConfig.ts` from the Rust retirement map's direct Tauri blocker list for `get_model_assignments` and `get_api_configurations`; those commands remain blocked by other frontend files and need more facade cleanup before the settings Rust batch can retire.
- Moved `src/features/chat/readiness/readinessGate.ts` from dynamic direct Tauri import to `@/runtime/native` for `get_model_assignments`, preserving its injectable `getAssignments` test seam and current failure-open readiness behavior.
- The chat readiness slice removes `src/features/chat/readiness/readinessGate.ts` from the Rust retirement map's direct Tauri blocker list, reducing direct Tauri blocker edges from 86 to 85 and blocker files from 28 to 27.
- Moved the `chat_v2_clear_approval_history` call in `src/features/settings/components/McpToolsSection.tsx` to `@/runtime/native` via a `nativeInvoke` alias, while leaving other still-blocked settings calls on the existing Tauri invoke path for later migration.
- Moved the `get_model_profiles` call in `src/features/chat/components/input-bar/InputBarV2.tsx` to `@/runtime/native` via a `nativeInvoke` alias, while leaving unrelated direct Tauri calls in the same file untouched.
- The latest facade cleanup removes both `chat_v2_clear_approval_history` and `get_model_profiles` from the Rust retirement map's direct Tauri blocker list, reducing blocked merged commands from 69 to 67, blocker edges from 85 to 83, and blocker files from 27 to 26.
- Added Go `SettingsService.GetSettingsByPrefix` plus Wails binding/bridge support for the legacy `get_settings_by_prefix` command, returning the old three-column `[key, value, updated_at]` shape over the compact Go settings store.
- Moved the remaining migrated tool-permission settings calls in `src/features/settings/components/McpToolsSection.tsx` (`get_settings_by_prefix`, `save_setting`, and `delete_setting`) from direct Tauri invoke to `@/runtime/native`.
- This MCP tools settings facade slice removes `get_settings_by_prefix` and the `McpToolsSection.tsx` edges for `save_setting`/`delete_setting` from the Rust retirement map's direct Tauri blocker list, reducing blocked merged commands from 67 to 65, blocker edges from 83 to 80, and blocker files from 26 to 25.
- Deleted the old Rust/Tauri `get_settings_by_prefix` registration, command handler, and database helper after confirming no direct Tauri blocker remains. `delete_settings_by_prefix` stays in old Rust for now because the old `chat_v2_clear_approval_history` handler still calls it.
- After the old Rust deletion, the retirement map drops merged Rust registrations from 288 to 287, merged Rust definitions from 299 to 298, and Rust retirement candidates from 266 to 265.
- Added Go `SettingsService.GetCNWhitelistConfig` plus Wails bridge and local fallback support for the legacy `get_cn_whitelist_config` shape used by the web-search advanced settings UI. The Go implementation reads the same compact settings keys, supports JSON-array and comma-separated custom-site formats, and carries the old trusted-site default list without importing the old Rust search stack.
- Moved `src/components/NoTagTreeShadPanel.tsx`, `src/components/WebSearchAdvancedConfig.tsx`, and the settings-loading path in `src/features/chat/components/TranslationPopover.tsx` from direct Tauri settings invokes to `@/runtime/native` for `get_api_configurations`, `get_model_assignments`, `get_setting`, `save_setting`, and `get_cn_whitelist_config`.
- Deleted the old Rust/Tauri `get_cn_whitelist_config` registration and command handler after confirming the frontend now reaches the Go/native facade. The old `CN_TRUSTED_SITES` Rust constant remains because other old Rust web-search internals may still use it.
- This settings facade/retirement slice removes `get_api_configurations`, `get_model_assignments`, `get_cn_whitelist_config`, and the WebSearch page's `get_setting`/`save_setting` edges from the direct Tauri blocker list, reducing blocked merged commands from 65 to 61, blocker edges from 80 to 71, and blocker files from 25 to 22. It also drops merged Rust registrations from 287 to 286 and merged Rust definitions from 298 to 297.
- Added Go `SettingsService.PreheatMCPTools` plus generated Wails binding, Wails bridge route, and local fallback support for the legacy `preheat_mcp_tools` response shape `{ ok, count }`.
- Moved `src/mcp/mcpService.ts` settings reads (`mcp.tools.list`, `mcp.performance.cache_ttl_ms`, `mcp.tools.cache_ttl_ms`) from dynamic Tauri core imports to `@/runtime/native`.
- Reworked MCP preheat so the real cache refresh happens in the frontend MCP SDK via `McpService.listTools()`, while the Go command is only a compatibility/status entrypoint. This avoids rebuilding the old Rust backend MCP cache architecture inside Go.
- Moved `src/features/settings/components/McpEditorSection.tsx` reconnect/retry preheat calls to `bootstrapMcpFromSettings({ force: true, preheat: true })` and routed its migrated settings `save_setting` calls through the native facade; Tauri core remains only for the old stdio connection test path.
- Deleted the old Rust/Tauri `preheat_mcp_tools` handler from `src-tauri/src/cmd/mcp.rs`. It was already not registered in `src-tauri/src/lib.rs`, so keeping it only preserved stale Rust runtime surface.
- This MCP preheat retirement slice moves `preheat_mcp_tools` into the merged Go/Wails bridge surface, increases merged commands with Wails bridge routes from 274 to 275, reduces direct Tauri blocked merged commands from 61 to 59, blocker edges from 71 to 69, blocker files from 22 to 21, and merged Rust definitions from 297 to 296.
- Deleted the now-unreferenced old Rust `LLMManager::preheat_mcp_tools_public` method after GitNexus showed its only upstream caller was the deleted old `preheat_mcp_tools` command.
- Added Go `SettingsService.GetMCPStatus`, `GetMCPTools`, and `ReloadMCPClient` plus generated Wails binding, Wails bridge routes, and local fallback support for the old backend-disabled MCP status/tool/reload compatibility commands.
- Moved `src/utils/settingsApi.ts` MCP status/reload/tools helpers from direct Tauri invoke to `@/runtime/native`, while leaving unrelated research/search helpers untouched for later slices.
- Updated `src/features/settings/components/McpToolsManager.tsx` so the MCP tools manager loads under Wails and injected native runtimes, not only Tauri.
- Deleted old Rust/Tauri registrations and handlers for `get_mcp_status`, `get_mcp_tools`, and `reload_mcp_client` from `src-tauri/src/lib.rs` and `src-tauri/src/cmd/mcp.rs`.
- This MCP backend-disabled compatibility slice increases merged commands with Wails bridge routes from 275 to 277, reduces merged Rust registrations from 286 to 284, merged Rust definitions from 296 to 294, direct Tauri blocked merged commands from 59 to 57, and blocker edges from 69 to 67.
- Added Go `FileService.ReadFileText` and `SaveTextToFile` plus Wails binding, Wails bridge routes, native facade helpers, and local fallback errors for non-native runtimes.
- Moved `src/utils/settingsApi.ts` and `src/utils/chatApi.ts` text-file read/write helpers from direct Tauri invoke to `@/runtime/native`, preserving public helper signatures used by export/download flows.
- Deleted old Rust/Tauri registrations and handlers for `read_file_text` and `save_text_to_file` from `src-tauri/src/lib.rs` and `src-tauri/src/commands.rs`; other file commands such as `read_file_bytes`, `get_file_size`, `copy_file`, and `hash_file` remain for later slices.
- This file text read/write retirement slice increases merged commands with Wails bridge routes from 277 to 279, reduces merged Rust registrations from 284 to 282, merged Rust definitions from 294 to 292, direct Tauri blocked merged commands from 57 to 55, blocker edges from 67 to 64, and blocker files from 21 to 20.

## Verified

Fresh checks from this checkpoint:

- `npx gitnexus impact --repo "Deep Student" --direction upstream notes_mentions_search` before migrating the command; output was LOW risk with 0 impacted callers/processes/modules for the old Rust command symbol.
- `cd desktop-go; go test ./internal/qbank ./internal/bindings -count=1` after adding Go `NotesMentionsSearch` and Wails binding exposure.
- `npm run go:bindings` after adding `QbankService.NotesMentionsSearch`; output was 11 services, 278 methods, 185 models.
- `npm run typecheck` after routing `notes_mentions_search` through the Wails bridge.
- `npm run native:triage` after marking `notes_mentions_search` merged; output was 643 unique commands, merge 332, replace 89, defer 185, delete 37.
- `npm run native:inventory -- --summary` after the mention-search slice; output was 1526 files scanned, 954 native references, 861 invokes / 643 unique, 65 listens / 47 unique, 28 emits / 24 unique.
- `cd desktop-go; go test ./...` after the mention-search slice.
- `$env:HTTP_PROXY="http://127.0.0.1:7892"; $env:HTTPS_PROXY="http://127.0.0.1:7892"; npm run go:check` after the mention-search slice; Go tests and smoke passed, data dir `C:\Users\Eldwen\AppData\Local\Deep Student`.
- `npx gitnexus status` after the mention-search slice; index is up-to-date at commit `1812ec7`.
- `npx gitnexus detect_changes --scope all` after the mention-search slice was attempted and failed because this GitNexus CLI build does not provide that command.
- `git diff --check` after the mention-search slice still reports only pre-existing trailing whitespace in `src/hooks/useSystemSettings.ts` lines 52 and 130.
- `node scripts/rust-retirement-map.mjs` after adding the Rust retirement map generator; output was 332 merged commands, 265 retirement candidates, 69 direct Tauri blocked merged commands, 86 blocker edges, and 28 blocker files.
- `npx gitnexus impact --repo "Deep Student" --direction upstream loadVoiceInputRuntimeConfig` before editing voice input runtime config; output was HIGH risk through `syncRuntimeConfig`, `useVoiceInputIntegration`, and `InputBarUI` progress/error/completed flows. The slice kept fallback behavior and only expanded native runtime detection plus the invoke facade.
- `npx gitnexus impact --repo "Deep Student" --direction upstream get_api_configurations` before editing the voice input caller; output was LOW/0 callers for the old Rust command symbol.
- `npm run typecheck` after moving `src/voice-input/runtimeConfig.ts` to `@/runtime/native`.
- `node scripts/rust-retirement-map.mjs` after the voice input facade cleanup; `src/voice-input/runtimeConfig.ts` no longer appears in any direct Tauri blocker list.
- `npm run native:triage` after the retirement-map and voice-input slice; output remained 643 unique commands, merge 332, replace 89, defer 185, delete 37.
- `npm run native:inventory -- --summary` after the retirement-map and voice-input slice; output was 1526 files scanned, 954 native references, 861 invokes / 643 unique, 65 listens / 47 unique, 28 emits / 24 unique.
- `npx gitnexus detect_changes --scope all` after the retirement-map and voice-input slice was attempted and failed because this GitNexus CLI build does not provide that command.
- `git diff --check` after the retirement-map and voice-input slice still reports only pre-existing trailing whitespace in `src/hooks/useSystemSettings.ts` lines 52 and 130.
- `npx gitnexus impact --repo "Deep Student" --direction upstream resolveChatReadiness` before editing chat readiness; output was LOW/0 impacted callers.
- `npx gitnexus impact --repo "Deep Student" --direction upstream checkChatReadiness` before editing chat readiness; output was LOW with direct impact only to `resolveChatReadiness`.
- `npx gitnexus impact --repo "Deep Student" --direction upstream get_model_assignments` before editing the chat readiness caller; output was LOW/0 callers for the old Rust command symbol.
- `npm run typecheck` after moving `src/features/chat/readiness/readinessGate.ts` to `@/runtime/native`.
- `node scripts/rust-retirement-map.mjs` after the chat readiness facade cleanup; output was 332 merged commands, 265 retirement candidates, 69 direct Tauri blocked merged commands, 85 blocker edges, and 27 blocker files. `src/features/chat/readiness/readinessGate.ts` no longer appears in any direct Tauri blocker list.
- `npm run native:triage` after the chat readiness facade cleanup; output remained 643 unique commands, merge 332, replace 89, defer 185, delete 37.
- `npm run native:inventory -- --summary` after the chat readiness facade cleanup; output was 1526 files scanned, 954 native references, 861 invokes / 643 unique, 65 listens / 47 unique, 28 emits / 24 unique.
- `npx gitnexus detect_changes --scope all` after the chat readiness facade cleanup was attempted and failed because this GitNexus CLI build does not provide that command.
- `git diff --check` after the chat readiness facade cleanup still reports only pre-existing trailing whitespace in `src/hooks/useSystemSettings.ts` lines 52 and 130.
- `npx gitnexus impact --repo "Deep Student" --direction upstream chat_v2_clear_approval_history` before moving the MCP tools approval-history caller; output was LOW/0 callers for the old Rust command symbol. `handleClearHistory` was not found in the current GitNexus index.
- `npx gitnexus impact --repo "Deep Student" --direction upstream get_model_profiles` before moving the InputBar model-profile caller; output was LOW/0 callers for the old Rust command symbol. `InputBarV2` was not found in the current GitNexus index.
- `npm run typecheck` after moving the MCP tools approval-history and InputBar model-profile calls to `@/runtime/native`.
- `node scripts/rust-retirement-map.mjs` after the two-call facade cleanup; output was 332 merged commands, 265 retirement candidates, 67 direct Tauri blocked merged commands, 83 blocker edges, and 26 blocker files. `chat_v2_clear_approval_history` and `get_model_profiles` no longer appear with direct Tauri files.
- `npm run native:triage` after the two-call facade cleanup; output remained 643 unique commands, merge 332, replace 89, defer 185, delete 37.
- `npm run native:inventory -- --summary` after the two-call facade cleanup; output was 1526 files scanned, 954 native references, 861 invokes / 643 unique, 65 listens / 47 unique, 28 emits / 24 unique.
- `npx gitnexus status` after the two-call facade cleanup; index is up-to-date at commit `1812ec7`.
- `npx gitnexus detect_changes --scope all` after the two-call facade cleanup was attempted and failed because this GitNexus CLI build does not provide that command.
- `git diff --check` after the two-call facade cleanup still reports only pre-existing trailing whitespace in `src/hooks/useSystemSettings.ts` lines 52 and 130.
- `npx gitnexus impact --repo "Deep Student" --direction upstream ToolPermissionsSection` before moving the MCP tools settings calls; output was LOW/0 impacted callers/processes/modules.
- `npx gitnexus impact --repo "Deep Student" --direction upstream SaveSetting` before extending the Go settings service; output was LOW/0 impacted callers/processes/modules. `invokeWails`, `fallbackInvoke`, and `GetSettings` were not found in the current GitNexus index; `SettingsService` returned LOW with direct impact only to `NewSettingsService` and indirect impact to `main`.
- `cd desktop-go; go test ./internal/settings ./internal/bindings -count=1` after adding `GetSettingsByPrefix` and its regression test.
- `npm run go:bindings` after adding `SettingsService.GetSettingsByPrefix`; output was 11 services, 279 methods, 185 models.
- `cd desktop-go; go test ./...` after the MCP tools settings facade slice.
- `npm run typecheck` after routing the MCP tools settings calls through `@/runtime/native`.
- `node scripts/rust-retirement-map.mjs` after the MCP tools settings facade slice; output was 332 merged commands, 273 commands with Wails bridge routes, 266 retirement candidates, 65 direct Tauri blocked merged commands, 80 blocker edges, and 25 blocker files. `get_settings_by_prefix` no longer appears in the direct Tauri blocker list.
- `npm run native:triage` after the MCP tools settings facade slice; output remained 643 unique commands, merge 332, replace 89, defer 185, delete 37.
- `npm run native:inventory -- --summary` after the MCP tools settings facade slice; output was 1526 files scanned, 954 native references, 861 invokes / 643 unique, 65 listens / 47 unique, 28 emits / 24 unique.
- `npx gitnexus status` after the MCP tools settings facade slice; index is up-to-date at commit `1812ec7`.
- `git diff --check` after the MCP tools settings facade slice still reports only pre-existing trailing whitespace in `src/hooks/useSystemSettings.ts` lines 52 and 130.
- `npx gitnexus impact --repo "Deep Student" --direction upstream get_settings_by_prefix` before deleting the old Rust implementation; output was LOW/0 impacted callers/processes/modules.
- `npx gitnexus impact --repo "Deep Student" --direction upstream delete_settings_by_prefix` before deciding whether to delete the adjacent old Rust helper; output was LOW with one direct caller, old Rust `chat_v2_clear_approval_history`, so that helper was deliberately left in place.
- `node scripts/rust-retirement-map.mjs` after deleting old Rust `get_settings_by_prefix`; output was 332 merged commands, 273 commands with Wails bridge routes, 287 merged Rust registrations, 298 merged Rust definitions, 265 retirement candidates, 65 direct Tauri blocked merged commands, 80 blocker edges, and 25 blocker files.
- `rg -n "get_settings_by_prefix" src-tauri docs/generated/rust-retirement-map.md` after the Rust deletion returned no matches, confirming the legacy command is no longer present in old Rust or the generated retirement map.
- `cargo fmt` / `rustfmt --edition 2021` were attempted after the Rust deletion but failed on unrelated pre-existing trailing whitespace in `src-tauri/src/translation/pipeline.rs:169`; the touched Rust files were small deletion-only edits.
- `npx gitnexus impact --repo "Deep Student" --direction upstream NoTagTreeShadPanel` before moving its settings calls; output was LOW/0 impacted callers/processes/modules. `loadModelOptions` was not found in the current GitNexus index.
- `npx gitnexus impact --repo "Deep Student" --direction upstream TranslationPopover` and `loadTranslationSettings` before moving its settings calls; both were LOW, with `loadTranslationSettings` directly impacting only `TranslationPopover`.
- `npx gitnexus impact --repo "Deep Student" --direction upstream WebSearchAdvancedConfigInner` and `loadConfigs` before moving its settings calls; both were LOW, with `loadConfigs` directly impacting only `WebSearchAdvancedConfigInner`.
- `npx gitnexus impact --repo "Deep Student" --direction upstream get_api_configurations`, `get_model_assignments`, and `get_cn_whitelist_config` before moving/deleting old settings command paths; all returned LOW/0 for old Rust command symbols.
- `npx gitnexus impact --repo "Deep Student" --direction upstream SettingsService` and `NewService` before adding Go `GetCNWhitelistConfig`; both returned LOW. `SettingsService` directly impacts `NewSettingsService` and indirectly `main`; settings `NewService` directly impacts Go app construction and `main`.
- `cd desktop-go; go test ./internal/settings ./internal/bindings -count=1` after adding `GetCNWhitelistConfig` and its regression test.
- `npm run go:bindings` after adding `SettingsService.GetCNWhitelistConfig`; output was 11 services, 280 methods, 187 models.
- `cd desktop-go; go test ./...` after the settings facade/retirement slice.
- `npm run typecheck` after moving the three frontend settings call sites to `@/runtime/native`.
- `npm run native:triage` after the settings facade/retirement slice; output remained 643 unique commands, merge 332, replace 89, defer 185, delete 37.
- `npm run native:inventory -- --summary` after the settings facade/retirement slice; output was 1526 files scanned, 954 native references, 861 invokes / 643 unique, 65 listens / 47 unique, 28 emits / 24 unique.
- `node scripts/rust-retirement-map.mjs` after the settings facade/retirement slice; output was 332 merged commands, 274 commands with Wails bridge routes, 286 merged Rust registrations, 297 merged Rust definitions, 265 retirement candidates, 61 direct Tauri blocked merged commands, 71 blocker edges, and 22 blocker files. `get_api_configurations`, `get_model_assignments`, and `get_cn_whitelist_config` no longer appear in the direct Tauri blocker list.
- `rg -n "get_cn_whitelist_config" src-tauri docs/generated/rust-retirement-map.md` after the Rust deletion returned no matches, confirming the legacy command is no longer present in old Rust or the generated retirement map.
- `git diff --check` after the settings facade/retirement slice still reports only pre-existing trailing whitespace in `src/hooks/useSystemSettings.ts` lines 52 and 130.
- `npx gitnexus impact --repo "Deep Student" --direction upstream preheat_mcp_tools` before deleting the old Rust handler; output was LOW/0 impacted callers/processes/modules.
- `npx gitnexus impact --repo "Deep Student" --direction upstream loadServersFromSettings`, `loadCacheTtlFromSettings`, and `bootstrapMcpFromSettings` before moving MCP settings/preheat paths; all returned LOW, affecting only MCP bootstrap, settings reconnect, and main settings-change refresh paths.
- `npx gitnexus impact --repo "Deep Student" --direction upstream handleReconnectClient` before moving settings-page preheat calls; output was LOW/0 impacted callers/processes/modules. `McpEditorSection` was not found in the current GitNexus index.
- `cd desktop-go; go test ./...` after adding `SettingsService.PreheatMCPTools`; all Go packages passed, including the new settings regression test.
- `npm run go:bindings` after adding `SettingsService.PreheatMCPTools`; output was 11 services, 281 methods, 188 models.
- `npm run typecheck` after moving MCP settings/preheat paths through the native facade; passed.
- `npm run native:triage` after the MCP preheat retirement slice; output was 643 unique commands, merge 332, replace 89, defer 185, delete 37. `preheat_mcp_tools` is now tracked as `merge`.
- `npm run native:inventory -- --summary` after the MCP preheat retirement slice; output was 1526 files scanned, 952 native references, 859 invokes / 643 unique, 65 listens / 47 unique, 28 emits / 24 unique.
- `npx gitnexus impact --repo "Deep Student" --direction upstream preheat_mcp_tools_public` before deleting the old Rust LLM manager helper; output was LOW, with one direct caller: the already-deleted old `src-tauri/src/cmd/mcp.rs:preheat_mcp_tools`.
- `node scripts/rust-retirement-map.mjs` after the MCP preheat retirement slice; output was 332 merged commands, 275 commands with Wails bridge routes, 286 merged Rust registrations, 296 merged Rust definitions, 265 retirement candidates, 59 direct Tauri blocked merged commands, 69 blocker edges, 21 blocker files, and 76 replace commands still registered in Rust.
- `rg -n "preheat_mcp_tools" src-tauri/src/cmd/mcp.rs src-tauri/src/lib.rs` after deleting the Rust handler returned no matches for old Rust command registration/handler.
- `npx gitnexus detect_changes --scope all` after the MCP preheat retirement slice was attempted and failed because this GitNexus CLI build does not provide that command.
- `git diff --check` after the MCP preheat retirement slice still reports only pre-existing trailing whitespace in `src/hooks/useSystemSettings.ts` lines 52 and 130.
- `npx gitnexus impact --repo "Deep Student" --direction upstream get_mcp_status`, `get_mcp_tools`, and `reload_mcp_client` before deleting old Rust handlers; all returned LOW/0 impacted callers/processes/modules for the old Rust command symbols.
- `npx gitnexus impact --repo "Deep Student" --direction upstream getMcpStatus`, `getMcpTools`, and `reloadMcpClient` before moving frontend helpers; all returned LOW. `getMcpStatus` impacts `McpStatusIndicator` and `McpToolsManager`; `getMcpTools` impacts `McpToolsManager`; `reloadMcpClient` impacts `McpToolsManager`.
- `npx gitnexus impact --repo "Deep Student" --direction upstream McpToolsManager` before expanding its runtime gate to Wails/injected native; output was LOW/0 impacted callers/processes/modules.
- `cd desktop-go; go test ./...` after adding MCP status/tool/reload compatibility methods; all Go packages passed, including new settings regression tests.
- `npm run go:bindings` after adding the three MCP compatibility methods; output was 11 services, 284 methods, 192 models.
- `npm run typecheck` after moving MCP status/tool/reload helpers through the native facade; passed.
- `npm run native:triage` after the MCP status/tool/reload slice; output remained 643 unique commands, merge 332, replace 89, defer 185, delete 37.
- `npm run native:inventory -- --summary` after the MCP status/tool/reload slice; output was 1526 files scanned, 952 native references, 859 invokes / 643 unique, 65 listens / 47 unique, 28 emits / 24 unique.
- `node scripts/rust-retirement-map.mjs` after the MCP status/tool/reload slice; output was 332 merged commands, 277 commands with Wails bridge routes, 284 merged Rust registrations, 294 merged Rust definitions, 265 retirement candidates, 57 direct Tauri blocked merged commands, 67 blocker edges, 21 blocker files, and 76 replace commands still registered in Rust.
- `rg -n "get_mcp_status|get_mcp_tools|reload_mcp_client" src-tauri/src/cmd/mcp.rs src-tauri/src/lib.rs` after deleting the old Rust handlers returned no matches for old Rust command registration/handler.
- `npx gitnexus detect_changes --scope all` after the MCP status/tool/reload slice was attempted and failed because this GitNexus CLI build does not provide that command.
- `git diff --check` after the MCP status/tool/reload slice still reports only pre-existing trailing whitespace in `src/hooks/useSystemSettings.ts` lines 52 and 130.
- `npx gitnexus impact --repo "Deep Student" --direction upstream read_file_text` and `save_text_to_file` before deleting old Rust handlers; both returned LOW/0 impacted callers/processes/modules for old Rust command symbols.
- `npx gitnexus impact --repo "Deep Student" --direction upstream readFileText` before moving the settings helper returned LOW/0; `readFileAsText` returned LOW with one direct `fileManager.readTextFile` caller and indirect MindMap use.
- `npx gitnexus impact --repo "Deep Student" --direction upstream saveTextToFile` before moving the settings helper returned CRITICAL because it is used by the shared `fileManager.saveTextFile` export/download helper across many UI flows. The edit preserved the public helper signature and command result semantics, replacing only the backend transport with the native facade.
- `cd desktop-go; go test ./...` after adding Go text read/write methods; all Go packages passed, including new file-service regression tests.
- `npm run go:bindings` after adding text read/write methods; output was 11 services, 286 methods, 192 models.
- `npm run typecheck` after moving text read/write helpers through the native facade; passed.
- `npm run native:triage` after the file text read/write slice; output remained 643 unique commands, merge 332, replace 89, defer 185, delete 37.
- `npm run native:inventory -- --summary` after the file text read/write slice; output was 1526 files scanned, 951 native references, 858 invokes / 643 unique, 65 listens / 47 unique, 28 emits / 24 unique.
- `node scripts/rust-retirement-map.mjs` after the file text read/write slice; output was 332 merged commands, 279 commands with Wails bridge routes, 282 merged Rust registrations, 292 merged Rust definitions, 265 retirement candidates, 55 direct Tauri blocked merged commands, 64 blocker edges, 20 blocker files, and 76 replace commands still registered in Rust.
- `rg -n "read_file_text|save_text_to_file" src-tauri/src/commands.rs src-tauri/src/lib.rs` after deleting the old Rust handlers returned no matches for old Rust command registration/handler.
- `npx gitnexus detect_changes --scope all` after the file text read/write slice was attempted and failed because this GitNexus CLI build does not provide that command.
- `git diff --check` after the file text read/write slice still reports only pre-existing trailing whitespace in `src/hooks/useSystemSettings.ts` lines 52 and 130.
- `cd desktop-go; go test ./internal/notes ./internal/dstu ./internal/bindings -count=1` after adding Go Notes zip import/export and DSTU import/export adapters.
- `cd desktop-go; go test ./...` after adding Notes import/export bindings and bridge routing.
- `npm run go:bindings` after adding Notes import/export commands; output was 11 services, 277 methods, 182 models.
- `npm run typecheck` after routing `notes_export`, `notes_export_single`, and `notes_import` through Wails `NotesService`.
- `npm run native:triage` after marking Notes zip import/export merged; output was 643 unique commands, merge 331, replace 90, defer 185, delete 37.
- `npm run native:inventory -- --summary` after the Notes import/export slice; output was 1526 files scanned, 954 native references, 861 invokes / 643 unique, 65 listens / 47 unique, 28 emits / 24 unique.
- `$env:HTTP_PROXY="http://127.0.0.1:7892"; $env:HTTPS_PROXY="http://127.0.0.1:7892"; npm run go:check` after the Notes import/export slice; Go tests and smoke passed, data dir `C:\Users\Eldwen\AppData\Local\Deep Student`.
- `cd desktop-go; go test ./internal/dstu -run TestNotesTrashLifecycleSyncsHybridVfs -count=1 -timeout 20s -v` after adding DSTU/Notes trash lifecycle and fixing the restore deadlock.
- `cd desktop-go; go test ./internal/dstu ./internal/vfs ./internal/bindings -count=1` after wiring DSTU trash command bindings.
- `cd desktop-go; go test ./internal/notes ./internal/dstu ./internal/vfs ./internal/bindings -count=1` after adding Notes asset maintenance and trash wrappers.
- `npm run go:bindings` after adding DSTU/Notes trash and asset-maintenance commands; output was 11 services, 274 methods, 177 models.
- `cd desktop-go; go test ./...` after adding DSTU/Notes trash and asset-maintenance commands.
- `npm run typecheck` after moving `src/dstu/api/trashApi.ts` to the native facade and regenerating Wails bindings.
- `npm run native:triage` after marking DSTU/Notes trash and asset-maintenance commands merged; output was 643 unique commands, merge 328, replace 93, defer 185, delete 37.
- `npm run native:inventory -- --summary` after the DSTU/Notes slice; output was 1526 files scanned, 954 native references, 861 invokes / 643 unique, 65 listens / 47 unique, 28 emits / 24 unique.
- `$env:HTTP_PROXY="http://127.0.0.1:7892"; $env:HTTPS_PROXY="http://127.0.0.1:7892"; npm run go:check` after DSTU/Notes trash and asset-maintenance commands; Go tests and smoke passed.
- `cd desktop-go; go test ./internal/anki -count=1` after adding Anki document task/status/control compatibility.
- `cd desktop-go; go test ./internal/app ./internal/bindings ./internal/anki` after registering Anki service and Wails bindings.
- `cd desktop-go; go test ./...` after adding Anki service, frontend native-facade routing, and regenerated bindings.
- `cd desktop-go; go test ./internal/anki -count=1` after adding the lean `start_enhanced_document_processing` worker, event emission, and card persistence.
- `cd desktop-go; go test ./internal/app ./internal/bindings ./internal/anki` after wiring `StartEnhancedDocumentProcessing`.
- `cd desktop-go; go test ./...` after routing Anki start/events through Wails/native facades.
- `cd desktop-go; go test ./internal/anki -count=1` after adding provider-backed Anki card generation and assigned-model config selection.
- `cd desktop-go; go test ./internal/app ./internal/bindings ./internal/anki` after wiring Anki provider config loading from Settings.
- `cd desktop-go; go test ./...` after adding the provider-backed Anki generation fallback path.
- `cd desktop-go; go test ./internal/dstu ./internal/vfs`
- `cd desktop-go; go test ./internal/dstu ./internal/vfs ./internal/app ./internal/bindings`
- `cd desktop-go; go test ./internal/qbank ./internal/vfs ./internal/app ./internal/bindings`
- `cd desktop-go; go test ./internal/settings ./internal/bindings`
- `cd desktop-go; go test ./internal/vfs ./internal/bindings`
- `cd desktop-go; go test ./internal/vfs`
- `cd desktop-go; go test ./internal/vfs ./internal/dstu ./internal/app ./internal/bindings`
- `cd desktop-go; go test ./internal/vfs` after adding `vfs_get_blob_base64`
- `cd desktop-go; go test ./internal/vfs ./internal/dstu ./internal/app ./internal/bindings` after adding `vfs_get_blob_base64`
- `cd desktop-go; go test ./internal/dstu ./internal/vfs ./internal/app ./internal/bindings` after adding DSTU folder/path resource lookup
- `cd desktop-go; go test ./internal/qbank ./internal/bindings` after adding Qbank CSV preview/import/export
- `cd desktop-go; go test ./internal/vfs ./internal/bindings` after adding Mindmap hybrid VFS support
- `cd desktop-go; go test ./internal/settings ./internal/bindings` after adding LLM config persistence.
- `cd desktop-go; go test ./internal/runtime ./internal/chat ./internal/app ./cmd/deep-student-go` after adding Wails chat event bridge.
- `cd desktop-go; go test ./internal/chat ./internal/app ./internal/runtime ./internal/bindings` after adding minimal OpenAI-compatible provider streaming and retry replay.
- `cd desktop-go; go test ./internal/chat -count=1` after wiring provider request cancellation.
- `cd desktop-go; go test ./internal/chat ./internal/app ./internal/runtime ./internal/bindings` after wiring provider request cancellation.
- `cd desktop-go; go test ./internal/chat -count=1` after wiring provider streaming for `chat_v2_continue_message`.
- `cd desktop-go; go test ./internal/chat ./internal/app ./internal/runtime ./internal/bindings` after wiring provider streaming for `chat_v2_continue_message`.
- `cd desktop-go; go test ./internal/chat -count=1` after wiring provider streaming for `chat_v2_edit_and_resend`.
- `cd desktop-go; go test ./internal/chat ./internal/app ./internal/runtime ./internal/bindings` after wiring provider streaming for `chat_v2_edit_and_resend`.
- `cd desktop-go; go test ./internal/chat -count=1` after adding message-level provider usage capture.
- `cd desktop-go; go test ./internal/chat ./internal/app ./internal/runtime ./internal/bindings` after adding message-level provider usage capture.
- `cd desktop-go; go test ./internal/chat -count=1` after adding `chat_v2_clear_approval_history`.
- `cd desktop-go; go test ./internal/chat ./internal/app ./internal/runtime ./internal/bindings` after adding `chat_v2_clear_approval_history`.
- `cd desktop-go; go test ./internal/chat -count=1` after adding Go-derived `llm_usage_*` compatibility.
- `cd desktop-go; go test ./internal/chat ./internal/app ./internal/runtime ./internal/bindings` after adding Go-derived `llm_usage_*` compatibility.
- `cd desktop-go; go test ./internal/vfs -count=1` after adding lightweight PDF text-layer extraction.
- `cd desktop-go; go test ./internal/storage ./internal/dstu ./internal/qbank ./internal/vfs -count=1` after introducing the shared atomic JSON writer and PDF start/retry text extraction.
- `cd desktop-go; go test ./...`
- `cd desktop-go; go test ./...` after adding lightweight PDF text-layer extraction.
- `cd desktop-go; go test ./...` after moving Go JSON stores to the shared atomic writer.
- `cd desktop-go; go test ./internal/qbank -count=1` after one Windows temp-file rename failure in the broader smoke; the isolated rerun passed.
- `cd desktop-go; go test ./internal/dstu -count=1` after adding notes search/tag and Canvas helpers.
- `cd desktop-go; go test ./...` after adding notes search/tag and Canvas helpers.
- `cd desktop-go; go test ./internal/qbank -count=1` after adding qbank learning-loop commands, history, mock exam scoring, generated paper, check-in calendar, and answer VFS resync.
- `cd desktop-go; go test ./...` after adding qbank learning-loop commands.
- `cd desktop-go; go test ./internal/reviewplan -count=1` after adding review-plan service and tests.
- `cd desktop-go; go test ./internal/reviewplan ./internal/app ./internal/bindings` after registering review-plan service and bindings.
- `cd desktop-go; go test ./...` after adding review-plan service.
- `cd desktop-go; go test ./internal/qbank -count=1` after adding lean Qbank AI grading/analyze compatibility.
- `cd desktop-go; go test ./internal/qbank ./internal/app ./internal/bindings ./internal/runtime` after wiring Qbank grading events and bindings.
- `cd desktop-go; go test ./...` after adding Qbank AI grading compatibility.
- `cd desktop-go; go test ./internal/qbank -count=1` after adding provider-backed qbank grading and assigned-model config selection.
- `cd desktop-go; go test ./internal/qbank ./internal/app ./internal/bindings ./internal/runtime` after wiring qbank provider config loading from Settings.
- `cd desktop-go; go test ./...` after adding the provider-backed qbank grading fallback path.
- `cd desktop-go; go test ./internal/qbank -count=1` after adding local qbank sync/conflict compatibility.
- `cd desktop-go; go test ./internal/qbank ./internal/app ./internal/bindings` after wiring qbank sync/conflict bindings.
- `cd desktop-go; go test ./...` after adding qbank sync/conflict compatibility.
- `cd desktop-go; go test ./internal/vfs -count=1` after adding resource sync compatibility.
- `cd desktop-go; go test ./internal/vfs ./internal/app ./internal/bindings` after wiring resource sync bindings.
- `cd desktop-go; go test ./internal/todo -count=1` after fixing local-day Todo/Pomodoro stats.
- `cd desktop-go; go test ./...` after adding resource sync compatibility and the Todo local-day stats fix.
- `cd desktop-go; go test ./internal/dstu -count=1` after adding the lean DSTU folder store and folder/path commands.
- `cd desktop-go; go test ./internal/dstu ./internal/vfs ./internal/app ./internal/bindings` after wiring folder/path bindings.
- `cd desktop-go; go test ./...` after adding DSTU folder/path CRUD, move, tree, breadcrumbs, path cache, and folder item metadata sync.
- `cd desktop-go; go test ./internal/vfs -count=1` after adding lightweight PDF page-count detection and PDF processing-status page fields.
- `cd desktop-go; go test ./...` after adding lightweight PDF page-count detection and PDF processing-status page fields.
- `cd desktop-go; go test ./internal/vfs -count=1` after adding PDF text-layer page JSON and OCR-info page snippets.
- `cd desktop-go; go test ./internal/vfs ./internal/dstu ./internal/app ./internal/bindings` after adding PDF text-layer page JSON compatibility.
- `cd desktop-go; go test ./...` after adding PDF text-layer page JSON compatibility.
- `cd desktop-go; go test ./internal/vfs -count=1` after adding generated PDF text-layer SVG previews and page-image readback coverage.
- `cd desktop-go; go test ./...` after adding generated PDF text-layer SVG previews.
- `npm run go:bindings`
- `npm run go:bindings` after adding qbank learning-loop commands; output was 9 services, 198 methods, 149 models.
- `npm run go:bindings` after adding review-plan commands; output was 10 services, 215 methods, 157 models.
- `npm run go:bindings` after adding qbank AI grading commands; output was 10 services, 217 methods, 159 models.
- `npm run go:bindings` after adding provider-backed qbank grading; output remained 11 services, 258 methods, 176 models.
- `npm run go:bindings` after adding qbank sync/conflict commands; output was 10 services, 222 methods, 163 models.
- `npm run go:bindings` after adding resource sync commands; output was 10 services, 226 methods, 165 models.
- `npm run go:bindings` after adding DSTU folder/path commands; output was 10 services, 247 methods, 173 models.
- `npm run go:bindings` after adding Anki document task/status/control compatibility; output was 11 services, 257 methods, 176 models.
- `npm run go:bindings` after adding the lean Anki start worker; output was 11 services, 258 methods, 176 models.
- `npm run go:bindings` after adding provider-backed Anki generation; output remained 11 services, 258 methods, 176 models.
- `npm run go:bindings` after adding PDF text-layer page JSON compatibility; output remained 11 services, 258 methods, 176 models.
- `npm run go:bindings` after adding generated PDF text-layer SVG previews; output remained 11 services, 258 methods, 176 models.
- `npm run native:triage`
- `npm run native:triage` after marking qbank learning-loop commands merged; output was 643 unique commands, merge 250, replace 130, defer 225, delete 38.
- `npm run native:triage` after marking review-plan commands merged; output was 643 unique commands, merge 266, replace 114, defer 225, delete 38.
- `npm run native:triage` after marking qbank AI grading commands merged; output was 643 unique commands, merge 268, replace 112, defer 225, delete 38.
- `npm run native:triage` after adding provider-backed qbank grading; output remained 643 unique commands, merge 308, replace 102, defer 196, delete 37.
- `npm run native:triage` after marking qbank sync/conflict commands merged; output was 643 unique commands, merge 273, replace 107, defer 225, delete 38.
- `npm run native:triage` after marking resource sync commands merged; output was 643 unique commands, merge 277, replace 103, defer 225, delete 38.
- `npm run native:triage` after marking DSTU folder/path commands merged; output was 643 unique commands, merge 298, replace 103, defer 204, delete 38.
- `npm run native:triage` after adding lightweight PDF page-count detection; output remained 643 unique commands, merge 298, replace 103, defer 204, delete 38 because no command surface changed.
- `npm run native:triage` after marking Anki document task/status/control commands merged; output was 643 unique commands, merge 307, replace 102, defer 197, delete 37. `start_enhanced_document_processing` remains defer.
- `npm run native:triage` after marking `start_enhanced_document_processing` merged; output was 643 unique commands, merge 308, replace 102, defer 196, delete 37.
- `npm run native:triage` after adding provider-backed Anki generation; output remained 643 unique commands, merge 308, replace 102, defer 196, delete 37.
- `npm run native:triage` after adding PDF text-layer page JSON compatibility; output remained 643 unique commands, merge 308, replace 102, defer 196, delete 37.
- `npm run native:triage` after adding generated PDF text-layer SVG previews; output remained 643 unique commands.
- `$env:HTTP_PROXY="http://127.0.0.1:7892"; $env:HTTPS_PROXY="http://127.0.0.1:7892"; npm run go:check`
- `$env:HTTP_PROXY="http://127.0.0.1:7892"; $env:HTTPS_PROXY="http://127.0.0.1:7892"; npm run go:check` after adding lightweight PDF page-count detection and PDF processing-status page fields.
- `$env:HTTP_PROXY="http://127.0.0.1:7892"; $env:HTTPS_PROXY="http://127.0.0.1:7892"; npm run go:check` after adding Anki document task/status/control compatibility; Go tests and smoke passed, data dir `C:\Users\Eldwen\AppData\Local\Deep Student`.
- `$env:HTTP_PROXY="http://127.0.0.1:7892"; $env:HTTPS_PROXY="http://127.0.0.1:7892"; npm run go:check` after adding the lean Anki start worker; Go tests and smoke passed, data dir `C:\Users\Eldwen\AppData\Local\Deep Student`.
- `$env:HTTP_PROXY="http://127.0.0.1:7892"; $env:HTTPS_PROXY="http://127.0.0.1:7892"; npm run go:check` after adding provider-backed Anki generation; Go tests and smoke passed, data dir `C:\Users\Eldwen\AppData\Local\Deep Student`.
- `$env:HTTP_PROXY="http://127.0.0.1:7892"; $env:HTTPS_PROXY="http://127.0.0.1:7892"; npm run go:check` after adding provider-backed qbank grading; Go tests and smoke passed, data dir `C:\Users\Eldwen\AppData\Local\Deep Student`.
- `$env:HTTP_PROXY="http://127.0.0.1:7892"; $env:HTTPS_PROXY="http://127.0.0.1:7892"; npm run go:check` after adding PDF text-layer page JSON compatibility; Go tests and smoke passed, data dir `C:\Users\Eldwen\AppData\Local\Deep Student`.
- `$env:HTTP_PROXY="http://127.0.0.1:7892"; $env:HTTPS_PROXY="http://127.0.0.1:7892"; npm run go:check` after adding generated PDF text-layer SVG previews; Go tests and smoke passed, data dir `C:\Users\Eldwen\AppData\Local\Deep Student`.
- `npm run typecheck`
- `npm run typecheck` after switching Anki task/status/control callers to `src/runtime/native.ts`.
- `npm run typecheck` after routing Anki start/events through Wails/native facades.
- `npm run typecheck` after adding provider-backed Anki generation.
- `npm run typecheck` after adding provider-backed qbank grading.
- `npm run typecheck` after adding PDF text-layer page JSON compatibility.
- `npm run typecheck` after adding generated PDF text-layer SVG previews.
- `npm run native:inventory -- --summary`
- `npm run native:inventory -- --summary` after Anki compatibility: 1526 files scanned, 953 native references, 861 invokes / 643 unique, 64 listens / 47 unique, 28 emits / 24 unique.
- `npm run native:inventory -- --summary` after Anki start/event routing: 1526 files scanned, 954 native references, 861 invokes / 643 unique, 65 listens / 47 unique, 28 emits / 24 unique.
- `npm run native:inventory -- --summary` after provider-backed Anki generation: 1526 files scanned, 954 native references, 861 invokes / 643 unique, 65 listens / 47 unique, 28 emits / 24 unique.
- `npm run native:inventory -- --summary` after provider-backed qbank grading: 1526 files scanned, 954 native references, 861 invokes / 643 unique, 65 listens / 47 unique, 28 emits / 24 unique.
- `npm run native:inventory -- --summary` after PDF text-layer page JSON compatibility: 1526 files scanned, 954 native references, 861 invokes / 643 unique, 65 listens / 47 unique, 28 emits / 24 unique.
- `npm run native:inventory -- --summary` after generated PDF text-layer SVG previews: 1526 files scanned, 954 native references, 861 invokes / 643 unique, 65 listens / 47 unique, 28 emits / 24 unique.
- `npx gitnexus status`
- `npx gitnexus status` after Anki compatibility reported index up to date at commit `1812ec7`.
- `npx gitnexus impact -r "Deep Student" -d upstream executeEditAndResend` returned LOW risk with one direct caller, adapter `setup`.
- `npx gitnexus impact -r "Deep Student" -d upstream editAndResend` returned HIGH risk through `MessageItemInner`; the edit-resend slice kept the UI entrypoint unchanged and limited changes to the injected callback/adapter/store merge path.
- `npx gitnexus impact -r "Deep Student" -d upstream EditAndResend`, `EditAndResendRequest`, and `invokeWails` were attempted and returned target not found because the new Go/Wails symbols are not represented in the current GitNexus symbol index yet.
- `npx gitnexus impact --repo "Deep Student" --direction upstream extractedTextForUpload`, `UploadFile`, and `fileMetadata` were attempted before the PDF text-layer slice and returned target not found because these Go symbols are not represented in the current GitNexus symbol index yet.
- `npx gitnexus impact --repo "Deep Student" --direction upstream StartPdfProcessing`, `RetryPdfProcessing`, and `readResourceBytesLocked` were attempted before the PDF retry-extraction slice and returned target not found because these Go symbols are not represented in the current GitNexus symbol index yet.
- `npx gitnexus impact --repo "Deep Student" --direction upstream UploadFile`, `StartPdfProcessing`, `GetPdfProcessingStatus`, `GetPdfPageImage`, `extractPdfTextLayer`, `fileMetadata`, `pdfProcessingStatus`, and `detectPdfPageCount` were attempted before/during the PDF page-count slice and returned target not found because these Go symbols are not represented in the current GitNexus symbol index yet.
- `npx gitnexus impact --repo "Deep Student" --direction upstream get_document_tasks` returned CRITICAL risk on the old Rust `src-tauri/src/enhanced_anki_service.rs:get_document_tasks`, with 15 impacted references and direct callers including old pause/resume/delete/task-count wrappers. The Anki compatibility slice did not edit that old Rust executor.
- `npx gitnexus impact --repo "Deep Student" --direction upstream recover_stuck_document_tasks` returned HIGH risk on the old Rust database recovery path. The Go slice added a separate lightweight Wails compatibility command instead of modifying that old database symbol.
- `npx gitnexus impact --repo "Deep Student" --direction upstream pause_document_processing`, `resume_document_processing`, `delete_document_session`, `trigger_task_processing`, `get_document_cards`, `get_document_processing_state`, `get_document_state`, and `get_document_task_counts` returned LOW risk for the old Rust command/service symbols.
- `npx gitnexus impact --repo "Deep Student" --direction upstream TaskDashboardPage`, `TaskController`, and `CardAgent` returned LOW risk before switching their Anki document task/control calls to `src/runtime/native.ts`. `AnkiApiAdapter`, `AnkiCardsBlock`, `chatAnkiIntegrationTestPlugin`, `errDocumentIDRequired`, and `RecoverStuckDocumentTasks` returned target not found in the current GitNexus index.
- `npx gitnexus impact --repo "Deep Student" --direction upstream start_enhanced_document_processing` returned LOW/0 callers for the old Rust command wrapper before routing the command through Go/Wails. `CardAgent` returned LOW impact before switching Anki event usage to `src/runtime/nativeEvents.ts`. `New` in `desktop-go/internal/app/app.go` returned LOW direct impact to `main`. New Go `StartEnhancedDocumentProcessing`, `nativeEvents`, and Wails `invokeWails` are not represented in the current GitNexus index yet; `emit` matched an unrelated cache manager symbol with LOW/0 callers.
- `npx gitnexus impact --repo "Deep Student" --direction upstream flushLocked` matched the indexed Settings `flushLocked` method and returned LOW risk with one direct caller; same-named DSTU/Qbank/other Go flush methods and the new `storage.WriteJSONAtomic` helper are not accurately represented in the current GitNexus symbol index yet.
- `git diff --check` still reports only the pre-existing trailing whitespace in `src/hooks/useSystemSettings.ts` lines 52 and 130.
- `npx gitnexus impact --repo "Deep Student" get_csv_preview`
- `npx gitnexus impact --repo "Deep Student" import_questions_csv`
- `npx gitnexus impact --repo "Deep Student" export_questions_csv`
- `npx gitnexus impact --repo "Deep Student" get_csv_exportable_fields`
- `npx gitnexus impact --repo "Deep Student" CsvImportDialog`
- `npx gitnexus impact --repo "Deep Student" QuestionBankExportDialog`
- `npx gitnexus impact --repo "Deep Student" QbankService` was attempted and returned `Target 'QbankService' not found`.
- `npx gitnexus impact --repo "Deep Student" --direction upstream SubmitAnswer` and `UpdateQuestion` were attempted before editing the qbank learning-loop slice and returned target not found because the Go qbank methods are not represented in the current GitNexus symbol index yet.
- `npx gitnexus impact --repo "Deep Student" --direction upstream NewService` matched the indexed Settings `NewService` method, not Qbank, and returned LOW risk with two direct callers.
- `npx gitnexus impact --repo "Deep Student" --direction upstream load` matched the indexed Vite config `load` symbol, not Qbank, and returned LOW/0 callers.
- `npx gitnexus impact --repo "Deep Student" --direction upstream qbank_get_history`, `qbank_generate_mock_exam`, and `qbank_submit_mock_exam` returned LOW/0 callers for the legacy Rust command wrappers before the Go bridge migration.
- `npx gitnexus impact --repo "Deep Student" --direction upstream QuestionHistoryView` returned LOW/0 callers before switching it to the native facade.
- `npx gitnexus impact --repo "Deep Student" --direction upstream review_plan_create`, `review_plan_process`, `review_plan_get_due`, `review_plan_get_stats`, `review_plan_get_due_with_filter`, `review_plan_batch_create`, and `review_plan_get_calendar_data` returned LOW/0 callers for the legacy command names before the Go bridge migration.
- `npx gitnexus impact --repo "Deep Student" --direction upstream useReviewPlanStore`, `invokeWails`, `fallbackInvoke`, and `implementedCommandOverrides` were attempted before the review-plan bridge edits and returned target not found because those symbols are not represented in the current GitNexus symbol index.
- `npx gitnexus impact --repo "Deep Student" --direction upstream qbank_ai_grade` and `qbank_cancel_grading` returned LOW/0 callers for the legacy Rust command wrappers before the Go bridge migration.
- `npx gitnexus impact --repo "Deep Student" --direction upstream useQbankAiGrading` returned LOW risk with one direct caller, `QuestionBankEditor`.
- `npx gitnexus impact --repo "Deep Student" --direction upstream SubmitAnswer` and `QbankService` were attempted before the qbank AI grading slice and returned target not found because the new Go qbank symbols are not represented in the current GitNexus symbol index.
- `npx gitnexus impact --repo "Deep Student" --direction upstream qbank_sync_check`, `qbank_get_sync_conflicts`, `qbank_resolve_sync_conflict`, `qbank_batch_resolve_conflicts`, and `qbank_set_sync_enabled` returned LOW/0 callers for the legacy Rust command wrappers before the Go bridge migration.
- `npx gitnexus impact --repo "Deep Student" vfs_create_mindmap`
- `npx gitnexus impact --repo "Deep Student" vfs_get_mindmap`
- `npx gitnexus impact --repo "Deep Student" vfs_get_mindmap_content`
- `npx gitnexus impact --repo "Deep Student" vfs_update_mindmap`
- `npx gitnexus impact --repo "Deep Student" vfs_delete_mindmap`
- `npx gitnexus impact --repo "Deep Student" vfs_list_mindmaps`
- `npx gitnexus impact --repo "Deep Student" vfs_set_mindmap_favorite`
- `npx gitnexus impact --repo "Deep Student" vfs_get_mindmap_versions`
- `npx gitnexus impact --repo "Deep Student" vfs_get_mindmap_version`
- `npx gitnexus impact --repo "Deep Student" vfs_get_mindmap_version_content`
- `npx gitnexus impact --repo "Deep Student" createMindMap`
- `npx gitnexus impact --repo "Deep Student" getMindMap`
- `npx gitnexus impact --repo "Deep Student" getMindMapContent`
- `npx gitnexus impact --repo "Deep Student" updateMindMap`
- `npx gitnexus impact --repo "Deep Student" deleteMindMap`
- `npx gitnexus impact --repo "Deep Student" listMindMaps`
- `npx gitnexus impact --repo "Deep Student" setMindMapFavorite`
- `npx gitnexus impact --repo "Deep Student" MindMapEmbed`
- `npx gitnexus impact get_api_configurations --repo "Deep Student" --direction upstream`
- `npx gitnexus impact save_api_configurations --repo "Deep Student" --direction upstream`
- `npx gitnexus impact get_vendor_configs --repo "Deep Student" --direction upstream`
- `npx gitnexus impact save_vendor_configs --repo "Deep Student" --direction upstream`
- `npx gitnexus impact get_model_profiles --repo "Deep Student" --direction upstream`
- `npx gitnexus impact save_model_profiles --repo "Deep Student" --direction upstream`
- `npx gitnexus impact get_model_assignments --repo "Deep Student" --direction upstream`
- `npx gitnexus impact save_model_assignments --repo "Deep Student" --direction upstream`
- `npx gitnexus impact normalizeModelAssignments --repo "Deep Student"` was attempted and returned `Target 'normalizeModelAssignments' not found` because the new Go helper is not represented in the current GitNexus symbol index yet.
- `npx gitnexus impact GetAPIConfigurations --repo "Deep Student"` was attempted and returned `Target 'GetAPIConfigurations' not found` because the new Go method is not represented in the current GitNexus symbol index yet.
- `npx gitnexus impact ChatV2TauriAdapter --repo "Deep Student" --direction upstream`
- `npx gitnexus impact EventBus --repo "Deep Student" --direction upstream`
- `npx gitnexus impact New --repo "Deep Student" --direction upstream`
- `npx gitnexus impact NewService --repo "Deep Student" --direction upstream`
- `npx gitnexus impact ApiConfig --repo "Deep Student" --direction upstream` returned CRITICAL for the old Rust `src-tauri/src/llm_manager/mod.rs:ApiConfig` with 198 impacted references. This provider slice did not edit that old Rust symbol; it added/used a lean Go `chat.ApiConfig` DTO and the already-migrated Go settings config.
- `npx gitnexus impact SendMessage --repo "Deep Student" --direction upstream` was attempted and returned `Target 'SendMessage' not found` because the new Go method is not represented in the current GitNexus symbol index yet.
- `npx gitnexus impact RetryMessage --repo "Deep Student" --direction upstream` was attempted and returned `Target 'RetryMessage' not found` because the new Go method is not represented in the current GitNexus symbol index yet.
- `npx gitnexus impact CancelStream --repo "Deep Student" --direction upstream` was attempted and returned `Target 'CancelStream' not found` because the new Go method is not represented in the current GitNexus symbol index yet.
- `npx gitnexus impact ContinueMessage --repo "Deep Student" --direction upstream` was attempted and returned `Target 'ContinueMessage' not found` because the new Go method is not represented in the current GitNexus symbol index yet.
- `npx gitnexus impact runAssistantStream --repo "Deep Student" --direction upstream` was attempted and returned `Target 'runAssistantStream' not found` because the new Go helper is not represented in the current GitNexus symbol index yet.
- `npx gitnexus impact streamOpenAICompatible --repo "Deep Student" --direction upstream` was attempted and returned `Target 'streamOpenAICompatible' not found` because the new Go helper is not represented in the current GitNexus symbol index yet.
- `npx gitnexus impact persistAssistantBlock --repo "Deep Student" --direction upstream` was attempted and returned `Target 'persistAssistantBlock' not found` because the new Go helper is not represented in the current GitNexus symbol index yet.
- `npx gitnexus impact previousUserContentBeforeLocked --repo "Deep Student" --direction upstream` was attempted and returned `Target 'previousUserContentBeforeLocked' not found` because the new Go helper is not represented in the current GitNexus symbol index yet.
- `npx gitnexus impact flushLocked --repo "Deep Student" --direction upstream` matched the indexed settings `flushLocked` symbol rather than the new chat method, returning LOW impact with one direct caller; the new chat method is not accurately represented in the current GitNexus symbol index yet.
- `npx gitnexus impact --repo "Deep Student" invokeWails` was attempted again and returned `Target 'invokeWails' not found`.
- `npx gitnexus impact --repo "Deep Student" fallbackInvoke` was attempted again and returned `Target 'fallbackInvoke' not found`.
- `npx gitnexus impact --repo "Deep Student" VfsService` was attempted again and returned `Target 'VfsService' not found`.
- `npx gitnexus impact --repo "Deep Student" extensionForResource` was attempted and returned `Target 'extensionForResource' not found`.
- `npx gitnexus impact --repo "Deep Student" dstu_folder_get_all_resources`
- `npx gitnexus impact --repo "Deep Student" dstu_get_resource_by_path`
- `npx gitnexus impact --repo "Deep Student" dstu_get_resource_location`
- `npx gitnexus impact --repo "Deep Student" getFolderAllResources`
- `npx gitnexus impact --repo "Deep Student" getResourceByPath`
- `npx gitnexus impact --repo "Deep Student" getResourceLocation`
- `npx gitnexus impact --repo "Deep Student" DstuService` was attempted and returned `Target 'DstuService' not found`.
- `npx gitnexus impact --repo "Deep Student" native-inventory` was attempted and returned `Target 'native-inventory' not found`.
- `npx gitnexus impact --repo "Deep Student" vfs_get_blob_base64`
- `npx gitnexus impact --repo "Deep Student" getBlobBase64`
- `npx gitnexus context --repo "Deep Student" getBlobBase64`
- `npx gitnexus impact --repo "Deep Student" invokeWails` was attempted and returned `Target 'invokeWails' not found` because the Wails facade is not represented in the current GitNexus symbol index.
- `npx gitnexus impact --repo "Deep Student" fallbackInvoke` was attempted and returned `Target 'fallbackInvoke' not found`.
- `npx gitnexus impact --repo "Deep Student" VfsService` was attempted and returned `Target 'VfsService' not found`.
- `npx gitnexus impact --repo "Deep Student" GetBlobBase64` was attempted and returned `Target 'GetBlobBase64' not found`.
- `npx gitnexus impact --repo "Deep Student" readResourceBytesLocked` was attempted and returned `Target 'readResourceBytesLocked' not found`.
- `npx gitnexus impact --repo "Deep Student" --direction upstream findResourceByAnyIDLocked` was attempted and returned `Target 'findResourceByAnyIDLocked' not found` because the new Go helper is not represented in the current GitNexus symbol index.
- `npx gitnexus context --repo "Deep Student" findResourceByAnyIDLocked` was attempted and returned `Symbol 'findResourceByAnyIDLocked' not found`.
- `npx gitnexus query --repo "Deep Student" --limit 8 "legacy virtual uri vfs resource source hash alias"` found old DSTU/VFS resource-path processes, including `parse_real_path`, and resource repository definitions.
- `npx gitnexus impact --repo "Deep Student" addTextbooks`
- `npx gitnexus impact --repo "Deep Student" textbooksAdd`
- `npx gitnexus impact --repo "Deep Student" usePdfLoader`
- `npx gitnexus impact --repo "Deep Student" TextbookContentViewInner`
- `npx gitnexus impact --repo "Deep Student" FileContentViewInner`
- `npx gitnexus impact --repo "Deep Student" ImageContentView`
- `npx gitnexus impact --repo "Deep Student" cancelPdfProcessing`
- `npx gitnexus impact --repo "Deep Student" retryPdfProcessing`
- `npx gitnexus impact --repo "Deep Student" startPdfProcessing`
- `npx gitnexus impact --repo "Deep Student" getPdfProcessingStatus`
- `npx gitnexus impact --repo "Deep Student" getBatchPdfProcessingStatus`
- `npx gitnexus impact --repo "Deep Student" getPdfPageImage`
- `npx gitnexus impact --repo "Deep Student" vfs_get_pdf_page_image`
- `npx gitnexus detect_changes --repo "Deep Student"` was attempted and failed because this GitNexus CLI build does not provide that command.
- `git diff --check` was attempted and still reports the pre-existing trailing whitespace in `src/hooks/useSystemSettings.ts` lines 52 and 130.
- `npm run test -- src/features/chat/components/input-bar/__tests__/InputBarAttachmentScope.source.test.ts` was attempted but did not reach tests because the install at that time was missing `@testing-library/dom`.
- `npm run test -- tests/vitest/chat-v2/vfsPdfProcessingApi.test.ts` was attempted but did not reach tests because the install at that time was missing `@testing-library/dom`.

Latest observed outputs:

- Wails bindings: 11 services, 289 methods, 192 models.
- Native triage: 643 unique commands, merge 332, replace 89, defer 185, delete 37.
- Native inventory: 1526 files scanned, 951 native references, 858 invokes / 643 unique, 65 listens / 47 unique, 28 emits / 24 unique.
- Rust retirement map: 332 merged commands, 291 merged commands with Wails bridge routes, 33 merged commands still registered in Rust, 57 merged Rust definitions, 41 Rust retirement candidates, 33 direct Tauri blocked merged commands, 33 blocker edges, 6 blocker files, and 76 replace commands still registered in Rust.
- Go smoke data dir: `C:\Users\Eldwen\AppData\Local\Deep Student`.
- Latest Go smoke and regression set, including provider-backed Anki document start/status/control/card/event compatibility with local fallback, DSTU folder/path CRUD/tree/items/reorder/breadcrumbs/path-cache/move/batch-move plus resource lookup, DSTU/Notes soft-delete/trash lifecycle, Notes asset maintenance/stats/vacuum, Notes zip import/export over Go DSTU notes and visible `notes_assets`, notes search/tag/Canvas helpers, Notes mention search over Go qbank questions, VFS blob/page-image/PDF-text/PDF-page-count/PDF-text-page/generated-SVG-preview/resource-sync coverage, Qbank CRUD/CSV/learning-loop/history/practice/mock-exam/paper/check-in/provider-backed AI-grading/sync-status compatibility flows, ReviewPlan spaced-repetition create/due/process/stats/history/calendar flows, Todo/Pomodoro local-day stats, Mindmap hybrid VFS coverage, LLM config settings persistence, Wails chat event bridge coverage, minimal OpenAI-compatible chat provider streaming for send/continue/retry/edit-resend, message-level provider usage capture, derived `llm_usage_*` stats, approval-history clearing, and provider request cancellation: proxied `npm run go:check` passes.

GitNexus notes:

- `npx gitnexus status` reports current index at commit `1812ec7`.
- `npx gitnexus detect_changes` / `detect-changes` is not supported by the installed CLI; it returns `unknown command`.
- Latest impact checks for the VFS file CRUD/content retirement slice returned LOW/0 for old Rust `vfs_upload_file`, `vfs_get_file`, `vfs_list_files`, `vfs_delete_file`, and `vfs_get_file_content`. The slice deleted the old Rust registrations/bodies and updated stale source-contract tests to assert Go hybrid VFS behavior instead of retired Rust wrappers.
- Latest impact check for the VFS blob-read retirement slice returned LOW/0 for old Rust `vfs_get_blob_base64`. The slice deleted the old Rust registration/body, while preserving `VfsBlobBase64Result` because the still-live Rust PDF page-image command returns it.
- Latest impact checks for the VFS resource ref/path retirement slice returned LOW/0 for old Rust `vfs_create_or_reuse`, `vfs_resource_exists`, `vfs_increment_ref`, `vfs_decrement_ref`, `vfs_get_resource_path`, and `vfs_update_path_cache`. The slice deleted the old Rust registrations/bodies and the private path-cache helper chain that only served the retired public wrappers.
- Latest impact checks for the Notes utility/assets retirement slice returned LOW/0 for all 27 old Rust command symbols: `canvas_note_append`, `canvas_note_read`, `canvas_note_replace`, `canvas_note_set`, `notes_assets_bulk_delete`, `notes_assets_index_scan`, `notes_assets_scan_orphans`, `notes_db_stats`, `notes_db_vacuum`, `notes_delete_asset`, `notes_empty_trash`, `notes_export`, `notes_export_single`, `notes_get_pref`, `notes_hard_delete`, `notes_import`, `notes_import_markdown`, `notes_import_markdown_batch`, `notes_list_assets`, `notes_list_deleted`, `notes_list_tags`, `notes_mentions_search`, `notes_resolve_asset_path`, `notes_restore`, `notes_save_asset`, `notes_search`, and `notes_set_pref`. The slice deleted the old Rust registrations/bodies/helpers and left only the still-live Notes CRUD/RAG config commands in `src-tauri/src/cmd/notes.rs`.
- Latest impact checks for the textbook add/bookmarks retirement slice returned LOW/0 for old Rust `textbooks_add` and LOW/0 for old Rust `textbooks_update_bookmarks`. The slice deleted the two old Rust command registrations/bodies and the unused `TextbookImportProgress` DTO, while keeping the rest of `src-tauri/src/cmd/textbooks.rs` because those commands still need separate Go parity or retirement proof.
- Latest impact checks for the core chat/Learning Hub native-facade retirement slice returned LOW for `useChatPageEvents` with direct impact to `ChatV2Page`; LOW/0 for `LearningHubPage`, `LearningHubSidebar`, old Rust `chat_v2_get_session`, `chat_v2_update_session_settings`, `chat_v2_get_group`, and `vfs_get_resource`; and LOW with one direct caller for old Rust `decrement_vfs_refs_for_session`, whose only caller was the retired Rust hard-delete command. The slice moved the remaining core chat/Learning Hub command calls through `@/runtime/native`, updated the group-read source guard to assert Go/Wails routing, deleted four old Rust command registrations/handlers, and verified `cargo check` still passes with only existing Rust warnings.
- Latest impact checks for the chat archive retirement slice returned LOW/0 for old Rust `chat_v2_delete_session`, `chat_v2_restore_session`, and `chat_v2_delete_group`; LOW/0 for `ChatSessionArchiveTab`; and LOW with one direct caller for `soft_delete_group_with_conn`, whose only caller was the retired Rust command. Go `RestoreSession`, `DeleteGroup`, and `RestoreGroup` are not represented in the current GitNexus index yet. The slice added Go/Wails archive restore, permanent-delete cascade, group delete/restore compatibility, moved the archive UI through `@/runtime/native`, and deleted the old Rust command registrations/handlers/helpers.
- Latest impact checks for the file text read/write slice returned LOW/0 for old Rust `read_file_text` and `save_text_to_file`; LOW/0 for settings `readFileText`; LOW for chat `readFileAsText`; and CRITICAL for settings `saveTextToFile` because it flows through the shared `fileManager.saveTextFile` export/download helper across many UI areas. The slice preserved the public helper signatures and command shapes, added Go `ReadFileText`/`SaveTextToFile`, moved callers through `@/runtime/native`, and deleted the old Rust command registrations/handlers.
- Latest impact checks for the MCP status/tool/reload slice returned LOW/0 for old Rust `get_mcp_status`, `get_mcp_tools`, and `reload_mcp_client`; LOW for frontend `getMcpStatus`, `getMcpTools`, and `reloadMcpClient`; and LOW/0 for `McpToolsManager`. The slice added Go `GetMCPStatus`, `GetMCPTools`, and `ReloadMCPClient`, moved the corresponding frontend helpers through `@/runtime/native`, expanded the MCP tools manager runtime gate to Wails/injected native, and deleted the old Rust command registrations/handlers.
- Latest impact check for the voice input runtime facade slice returned HIGH for `loadVoiceInputRuntimeConfig` because it flows through InputBar voice input states. The edit did not change command shapes or fallback output; it replaced direct Tauri invoke with the native facade and made Wails count as a valid native runtime. Old Rust `get_api_configurations` returned LOW/0 callers for this slice.
- Latest impact checks for the chat readiness facade slice returned LOW/0 impacted callers for `resolveChatReadiness`, LOW direct impact from `checkChatReadiness` to `resolveChatReadiness`, and LOW/0 callers for old Rust `get_model_assignments`. The edit did not change readiness semantics; it only moved the default assignment fetch from dynamic Tauri import to the native facade.
- Latest impact checks for the MCP tools/InputBar facade slice returned LOW/0 callers for old Rust `chat_v2_clear_approval_history` and `get_model_profiles`; `handleClearHistory` and `InputBarV2` were not found in the current GitNexus index. The edit was limited to two already-merged command calls and preserved all other direct Tauri calls in those files.
- Latest impact checks for the MCP tools settings facade slice returned LOW/0 for `ToolPermissionsSection` and Go `SaveSetting`; `SettingsService` returned LOW with direct impact only to `NewSettingsService` and indirect impact to `main`; `invokeWails`, `fallbackInvoke`, and `GetSettings` were not found in the current GitNexus index. The edit added Go `GetSettingsByPrefix` and moved only already-merged settings commands in `McpToolsSection.tsx` to the native facade.
- Latest impact check for deleting old Rust `get_settings_by_prefix` returned LOW/0. Adjacent `delete_settings_by_prefix` returned LOW with direct impact to old Rust `chat_v2_clear_approval_history`, so it was not deleted in this slice.
- Latest impact checks for the MCP preheat retirement slice returned LOW/0 for old Rust `preheat_mcp_tools` and settings-page `handleReconnectClient`; LOW for `loadServersFromSettings`, `loadCacheTtlFromSettings`, and `bootstrapMcpFromSettings`, affecting only MCP bootstrap/settings reconnect/main settings-change refresh; LOW for old Rust `preheat_mcp_tools_public`, whose only caller was the deleted old command. `McpEditorSection` was not found in the current GitNexus index. The slice added Go `PreheatMCPTools`, moved MCP settings/preheat paths through `@/runtime/native`, updated triage to mark it merged, and deleted the old Rust `preheat_mcp_tools` handler plus stale LLM manager helper.
- Previous impact checks for the settings facade/retirement slice returned LOW/0 for `NoTagTreeShadPanel`, old Rust `get_api_configurations`, `get_model_assignments`, and `get_cn_whitelist_config`; LOW for `TranslationPopover`, `loadTranslationSettings`, `WebSearchAdvancedConfigInner`, `loadConfigs`, `SettingsService`, and settings `NewService`. `loadModelOptions` was not found in the current GitNexus index. The slice added Go `GetCNWhitelistConfig`, routed settings call sites through `@/runtime/native`, and deleted only the old Rust `get_cn_whitelist_config` command registration/handler.
- Latest impact check for the Notes mention-search slice returned LOW/0 impacted callers for old Rust `notes_mentions_search`. The slice routes the legacy command name to Go `QbankService.NotesMentionsSearch`; new Go symbols and Wails bridge helpers are not represented in the current GitNexus symbol index yet.
- Latest impact checks for the Notes zip import/export slice returned LOW/0 callers for old Rust command symbols `notes_export`, `notes_export_single`, and `notes_import`. The slice added separate Go import/export code and Wails routing instead of editing or relying on the old Rust exporter/importer.
- Latest impact checks for the DSTU/Notes trash and asset-maintenance slice returned LOW/0 callers for old command symbols `dstu_delete`, `dstu_list_deleted`, `notes_assets_scan_orphans`, `notes_export`, `notes_db_stats`, `notes_assets_index_scan`, `notes_assets_bulk_delete`, and `softDelete`. New Go methods and many Wails bridge helpers are not represented in the current GitNexus symbol index yet.
- Latest impact checks for the provider-backed Anki generation slice returned LOW/0 callers for old Rust `start_enhanced_document_processing`, LOW for frontend `CardAgent`, and LOW direct impact from Go `desktop-go/internal/app/app.go:New` to `main`. The slice did not edit old Rust Anki executor symbols. New Go Anki provider helpers are not represented in the current GitNexus symbol index yet.
- Latest impact checks for the provider-backed qbank grading slice returned LOW/0 callers for old Rust `qbank_ai_grade`, LOW direct impact from `useQbankAiGrading` to `QuestionBankEditor`, and LOW direct impact from Go `desktop-go/internal/app/app.go:New` to `main`. New Go `AIGrade`, `buildLeanGradingFeedback`, `buildLeanGradeFeedback`, `buildLeanAnalysisFeedback`, and qbank provider helpers are not represented in the current GitNexus symbol index yet.
- Latest impact checks for the PDF text-layer page compatibility slice returned LOW for frontend `getPdfPageImage` and `startPdfProcessing`; HIGH for frontend `retryPdfProcessing` because of input-bar progress/error/completed flows, but this slice did not edit those frontend symbols or change command shapes. Go `GetResourceOcrInfo`, `UploadFile`, `fileMetadata`, and the new PDF text-page helpers are not represented in the current GitNexus symbol index yet.
- Latest impact checks for the generated PDF text-layer SVG preview slice returned LOW for frontend `startPdfProcessing` and `getPdfPageImage`; HIGH for frontend `retryPdfProcessing` because of input-bar progress/error/completed flows, but this slice did not edit frontend call sites or command shapes. Go `StartPdfProcessing`, `GetPdfPageImage`, `UploadFile`, and `fileMetadata` are not represented in the current GitNexus symbol index yet.
- Latest impact checks for the Anki document start/status/control slice returned LOW/0 callers for old Rust `start_enhanced_document_processing`; CRITICAL for old Rust `get_document_tasks`; and HIGH for old Rust/database `recover_stuck_document_tasks`. These old Rust symbols were not edited. Old Rust `pause_document_processing`, `resume_document_processing`, `delete_document_session`, `trigger_task_processing`, `get_document_cards`, `get_document_processing_state`, `get_document_state`, and `get_document_task_counts` returned LOW. Frontend `TaskDashboardPage`, `TaskController`, and `CardAgent` returned LOW before moving migrated Anki command/event calls through `src/runtime/native.ts` / `src/runtime/nativeEvents.ts`; `AnkiApiAdapter`, `AnkiCardsBlock`, `chatAnkiIntegrationTestPlugin`, `StartEnhancedDocumentProcessing`, `nativeEvents`, `invokeWails`, and other new Go/Wails symbols are not represented in the current index yet.
- Latest impact checks for the DSTU folder/path store slice returned LOW for legacy command wrappers `dstu_folder_create`, `dstu_folder_delete`, `dstu_folder_get_breadcrumbs`, and `dstu_move_to_folder`; LOW for frontend/mock-matched `createFolder`, `deleteFolder`, `renameFolder`, `moveFolder`, `moveItem`, `listFolders`, `getFolderItems`, `getFolderTree`, `getBreadcrumbs`, `parsePath`, `moveToFolder`, and `batchMove`. `moveFolder` directly affects `LearningHubSidebar`; `moveItem` directly affects Notes/Learning Hub sidebars; `getBreadcrumbs` affects Learning Hub/finder breadcrumb paths. New Go symbols such as `DstuService`, `GetFolderAllResources`, `GetResourceLocation`, `nodeResourceLocation`, `folderResourceInfoFromNode`, `matchesListOptions`, `invokeWails`, and `fallbackInvoke` are not represented in the current GitNexus symbol index yet or matched unrelated same-name symbols.
- Latest impact checks for the resource sync slice: `TauriResourceSyncService` returned LOW risk with one direct file and indirect Notes/chat UI callers; old command names `resource_sync_note`, `resource_sync_exam`, `resource_sync_textbook_pages`, and `resource_check_sync_needed` returned target-not-found in the current GitNexus index; Go `CreateOrUpdateSource`, `VfsService`, `invokeWails`, `fallbackInvoke`, `resourceSyncService`, `todayDate`, `nowISO`, `Service.ActiveSummary`, `Service.PomodoroTodayStats`, and `Service.ListTodayPomodoros` are not represented as indexed GitNexus symbols yet. `CreateResource` and `PomodoroTodayStats` matched unrelated/frontend symbols with LOW/0 callers.
- Latest impact checks for the qbank sync/conflict slice returned LOW/0 callers for old Rust `qbank_sync_check`, `qbank_get_sync_conflicts`, `qbank_resolve_sync_conflict`, `qbank_batch_resolve_conflicts`, and `qbank_set_sync_enabled`.
- Latest impact checks for the qbank AI grading slice returned LOW/0 callers for old Rust `qbank_ai_grade` and `qbank_cancel_grading`; LOW direct impact from `useQbankAiGrading` to `QuestionBankEditor`; and target-not-found for new Go `SubmitAnswer` and `QbankService`.
- Latest impact checks for the review-plan slice returned LOW/0 callers for old command names `review_plan_create`, `review_plan_process`, `review_plan_get_due`, `review_plan_get_stats`, `review_plan_get_due_with_filter`, `review_plan_batch_create`, and `review_plan_get_calendar_data`. `useReviewPlanStore`, `invokeWails`, `fallbackInvoke`, and `implementedCommandOverrides` are not represented in the current GitNexus symbol index yet.
- Latest impact checks for the notes utility slice returned LOW/0 callers for old Rust `notes_search`, `notes_list_tags`, `canvas_note_read`, `canvas_note_append`, `canvas_note_replace`, and `canvas_note_set`. New Go `NotesSearch`, `CanvasAppendContent`, `DstuService`, `invokeWails`, `fallbackInvoke`, and `implementedCommandOverrides` are not represented in the current GitNexus symbol index yet.
- Latest impact checks for the chat provider/settings/usage slices: old Rust `ApiConfig` returned CRITICAL because the legacy Rust LLM manager has a large blast radius, but that symbol was not edited; `llm_usage_summary` and `llm_usage_get_trends` returned LOW/0 callers, `LlmUsageStatsSection` returned LOW/0 callers, and new Go `LLMUsageSummary` was not represented in the current GitNexus symbol index yet. `chat_v2_clear_approval_history` returned LOW/0 callers, `McpToolsSection` returned LOW/0 callers, and new Go `ClearApprovalHistory` was not represented in the current GitNexus symbol index yet. Go-side `SendMessage`, `ContinueMessage`, `RetryMessage`, `CancelStream`, `runAssistantStream`, `streamOpenAICompatible`, `readOpenAIStream`, `parseOpenAIStreamEvent`, `persistAssistantBlock`, and `previousUserContentBeforeLocked` are not represented in the current GitNexus symbol index yet; `flushLocked` matched the indexed settings symbol instead of chat and returned LOW. Earlier impact checks for the Wails event bridge returned LOW/0 callers for frontend `ChatV2TauriAdapter`, LOW impact for Go `EventBus`, LOW impact for `desktop-go/internal/app.New`, and LOW impact for the indexed `NewService` symbol. Earlier impact checks for LLM config returned LOW/0 callers for old Rust `get_api_configurations`, `save_api_configurations`, `get_vendor_configs`, `save_vendor_configs`, `get_model_profiles`, `save_model_profiles`, `get_model_assignments`, and `save_model_assignments`; `normalizeModelAssignments` and `GetAPIConfigurations` are not represented in the current GitNexus symbol index yet. Earlier impact checks for the Mindmap slice returned LOW/0 callers for old Rust `vfs_create_mindmap`, `vfs_get_mindmap`, `vfs_get_mindmap_content`, `vfs_update_mindmap`, `vfs_delete_mindmap`, `vfs_list_mindmaps`, `vfs_set_mindmap_favorite`, `vfs_get_mindmap_versions`, `vfs_get_mindmap_version`, and `vfs_get_mindmap_version_content`; LOW impact for frontend `createMindMap`, `getMindMap`, `getMindMapContent`, and `updateMindMap` through `mindmapStore.ts`; LOW/0 callers for frontend `deleteMindMap`, `listMindMaps`, `setMindMapFavorite`, and `MindMapEmbed`; and `VfsService`, `invokeWails`, `fallbackInvoke`, plus `extensionForResource` are not represented in the current GitNexus symbol index yet. Earlier impact checks for Qbank CSV returned LOW/0 callers for old Rust `get_csv_preview`, `import_questions_csv`, `export_questions_csv`, and `get_csv_exportable_fields`; LOW/0 callers for `CsvImportDialog` and `QuestionBankExportDialog`; and `QbankService` is not represented in the current GitNexus symbol index yet. Earlier impact checks returned LOW/0 callers for old Rust `dstu_folder_get_all_resources`, `dstu_get_resource_by_path`, and `dstu_get_resource_location`; LOW/0 callers for `getFolderAllResources` in the mock API symbol; LOW direct impact from `getResourceByPath` to `pathExists`; and LOW direct impact from `getResourceLocation` to `resourceExists`. Earlier impact checks returned LOW/0 callers for old Rust `vfs_get_blob_base64`, LOW/0 callers for frontend `getBlobBase64`, and a `getBlobBase64` context showing only local cache helper dependencies. Earlier impact checks returned LOW for `getPdfPageImage`, with direct impact on `getPdfPageImageDataUrl` and indirect impact on Markdown citation image rendering; LOW/0 callers for old Rust `vfs_get_pdf_page_image`; HIGH for `cancelPdfProcessing` because it directly affects `InputBarUI` and the InputBar progress/error/completed processes; HIGH for `retryPdfProcessing` because it directly affects `InputBarUI` and `ensureChatImageOcrProcessing`; LOW for `startPdfProcessing`; LOW/0 callers for `getBatchPdfProcessingStatus`; and a misleading LOW hit for a different same-named `getPdfProcessingStatus` symbol in `src/hooks/usePdfProcessingProgress.ts`. Earlier impact checks returned LOW for `addTextbooks` in `src/dstu/adapters/textbookDstuAdapter.ts` with two direct Learning Hub callers, LOW/0 callers for `textbooksAdd` in `src/utils/chatApi.ts`, LOW for `usePdfLoader` with direct callers in `TextbookContentViewInner` and `FileContentViewInner`, and LOW/0 callers for `TextbookContentViewInner`, `FileContentViewInner`, and `ImageContentView`. `findResourceByAnyIDLocked`, `GetResource`, `GetAttachmentContent`, `GetFileContent`, `AddTextbooks`, `GetPdfPageImage`, `GetBlobBase64`, `normalizeModelAssignments`, `GetAPIConfigurations`, `SendMessage`, `ContinueMessage`, `RetryMessage`, `CancelStream`, `runAssistantStream`, `streamOpenAICompatible`, `readOpenAIStream`, `parseOpenAIStreamEvent`, `persistAssistantBlock`, `previousUserContentBeforeLocked`, `ClearApprovalHistory`, Go `LLMUsage*` methods, `DstuService`, `QbankService`, `VfsService`, `invokeWails`, `fallbackInvoke`, `extensionForResource`, `native-triage`, `native-inventory`, and `implementedCommandOverrides` are not represented as indexed GitNexus symbols yet.
- `git diff --check` still reports only pre-existing trailing whitespace in `src/hooks/useSystemSettings.ts` lines 52 and 130.
- `git diff --check` after generated PDF text-layer SVG previews still reports only pre-existing trailing whitespace in `src/hooks/useSystemSettings.ts` lines 52 and 130.
- Run `npx gitnexus impact <symbol> --repo "Deep Student" --direction upstream` before editing code symbols.

## Completed VFS OCR Info/Clear Rust Retirement Slice

Retired the Go/Wails-backed OCR inspection/clear command pair from the old Rust/Tauri VFS command surface:

- `vfs_get_resource_ocr_info`
- `vfs_clear_resource_ocr`

What changed:

- Implemented Go `VfsService.ClearResourceOcr` in `desktop-go/internal/vfs/service.go`.
- Go OCR clear removes `ocrText` / `ocr_text`, `ocrPagesJson` / `ocr_pages_json`, OCR source/status/error timestamps, preserves `extractedText`, and writes lightweight `processingStatus = ocr_processing`, `processingProgress`, `indexStatus = pending`, and `textIndexState = pending` metadata for the future Go OCR pipeline.
- Go `GetResourceOcrInfo` now treats soft-deleted resources like missing resources and returns `activeSource: none` without exposing OCR or extracted text.
- Removed old Tauri registrations from `src-tauri/src/lib.rs`.
- Removed old Rust `vfs_get_resource_ocr_info`, `vfs_clear_resource_ocr`, `ResourceOcrInfo`, and the clear-only `reset_file_ocr_processing_progress` helper from `src-tauri/src/vfs/handlers.rs`.
- Kept Rust `OcrPageInfo` / `parse_ocr_pages_for_display` / `ocr_pages_effective_text_stats` because old Rust index-status code still calls them until that batch is retired.
- Updated `tests/vitest/learning-hub/vfsAttachmentDeleteContract.source.test.ts` to assert Go hybrid VFS deleted-resource filtering and old Rust OCR wrapper retirement.

GitNexus / impact notes:

- `npx gitnexus impact --repo "Deep Student" --direction upstream vfs_clear_resource_ocr`: LOW risk, 0 impacted callers.
- `npx gitnexus impact --repo "Deep Student" --direction upstream vfs_get_resource_ocr_info`: LOW risk, 0 impacted callers.
- `npx gitnexus impact --repo "Deep Student" --direction upstream reset_file_ocr_processing_progress`: LOW risk; one direct caller, the deleted old Rust `vfs_clear_resource_ocr` wrapper.
- `npx gitnexus impact --repo "Deep Student" --direction upstream clearResourceOcr` and `getResourceOcrInfo`: LOW risk, direct frontend impact limited to `IndexStatusView`.
- `npx gitnexus impact --repo "Deep Student" --direction upstream ClearResourceOcr` and `GetResourceOcrInfo` returned target not found because the new Go symbols are not represented in the current GitNexus index.

Verification:

- `rg "vfs_get_resource_ocr_info|vfs_clear_resource_ocr|ResourceOcrInfo|reset_file_ocr_processing_progress" src-tauri/src -S`: no old Rust command definition, registration, DTO, or clear-only helper remains.
- `go test ./internal/vfs -run "TestClearResourceOcr|TestCompactIndexStatusReflectsHybridResources|TestUploadFileBuildsPdfTextLayerPageInfo" -count=1` from `desktop-go`: pass.
- `go test ./internal/vfs ./internal/bindings -count=1` from `desktop-go`: pass.
- `go test ./... -count=1` from `desktop-go`: pass.
- `npm run test -- tests/vitest/learning-hub/vfsAttachmentDeleteContract.source.test.ts`: pass, 11 tests.
- `npm run typecheck`: pass.
- `npm run native:triage`: pass, 643 commands.
- `npm run native:inventory -- --summary`: pass, 1526 scanned files / 951 native references / 858 invokes / 643 unique invokes / 65 listens / 47 unique listens / 28 emits / 24 unique emits.
- `node scripts/rust-retirement-map.mjs`: pass; merged Rust registrations reduced from 27 to 25, merged Rust definitions from 51 to 49, retirement candidates from 35 to 33, and `src-tauri/src/vfs/handlers.rs` merged Go commands still present in Rust from 11 to 9.
- `git diff --check` on this slice's touched Go/Rust/test/generated-doc files: pass.
- `npx gitnexus detect_changes --repo "Deep Student"` was attempted and still fails because this GitNexus CLI build does not provide `detect_changes`.
- `npm run test -- tests/vitest/learning-hub/IndexStatusView.behavior.test.tsx` still has the known pre-existing failure: the progress label renders `indexStatus.progress.textIndexProgress undefined/undefined` instead of `7/9`; the other 4 tests in that file pass.
- `cargo check` was intentionally not used as this slice's verification gate. The acceptance evidence is Go OCR clear parity, Wails bridge routing, source-contract coverage, and the reduced Rust retirement map.

Known gap:

- This slice does not implement a real Go OCR engine or raster PDF OCR pipeline. `ClearResourceOcr` now clears OCR metadata and marks processing/index metadata pending, but OCR reprocessing still needs the future native OCR/PDF pipeline listed below.

## Completed VFS PDF Read-Only Status/Page Rust Retirement Slice

Retired three Go/Wails-backed PDF read-only command wrappers from the old Rust/Tauri VFS command surface:

- `vfs_get_pdf_processing_status`
- `vfs_get_batch_pdf_processing_status`
- `vfs_get_pdf_page_image`

What changed:

- Removed the three old Tauri command registrations from `src-tauri/src/lib.rs`.
- Removed old Rust `vfs_get_pdf_processing_status`, `vfs_get_batch_pdf_processing_status`, `vfs_get_pdf_page_image`, and the now-unused Rust-only `VfsBlobBase64Result` DTO from `src-tauri/src/vfs/handlers.rs`.
- Kept old Rust `vfs_cancel_pdf_processing`, `vfs_retry_pdf_processing`, `vfs_start_pdf_processing`, and `vfs_list_pending_pdf_processing` for now because their progress/error/event flows need a separate parity review before deletion.
- Kept legacy command names active through `src/runtime/wailsBridge.ts`, where they route to Go `VfsService.GetPdfProcessingStatus`, `GetBatchPdfProcessingStatus`, and `GetPdfPageImage`.
- Updated `tests/vitest/learning-hub/vfsAttachmentDeleteContract.source.test.ts` so deleted-file page preview protection is asserted against Go hybrid VFS rather than the retired Rust SQL wrapper.

GitNexus / impact notes:

- `npx gitnexus impact --repo "Deep Student" --direction upstream vfs_get_pdf_processing_status`: LOW risk, 0 impacted callers.
- `npx gitnexus impact --repo "Deep Student" --direction upstream vfs_get_batch_pdf_processing_status`: LOW risk, 0 impacted callers.
- `npx gitnexus impact --repo "Deep Student" --direction upstream vfs_get_pdf_page_image`: LOW risk, 0 impacted callers.
- `npx gitnexus impact --repo "Deep Student" --direction upstream getBatchPdfProcessingStatus`: LOW risk, 0 impacted callers.
- `npx gitnexus impact --repo "Deep Student" --direction upstream getPdfPageImage`: LOW risk; direct impact to `getPdfPageImageDataUrl`, indirect impact to Markdown citation image rendering.
- `npx gitnexus impact --repo "Deep Student" --direction upstream getPdfProcessingStatus` matched a same-named hook symbol in `src/hooks/usePdfProcessingProgress.ts` with LOW/0 rather than the API function; the old Rust command impact and Wails bridge routing were used as the stronger evidence for this deletion.
- `npx gitnexus impact --repo "Deep Student" --direction upstream GetPdfProcessingStatus`, `GetBatchPdfProcessingStatus`, and `GetPdfPageImage` returned target not found because the new Go symbols are not represented in the current GitNexus index.

Verification:

- `rg "pub async fn vfs_get_pdf_page_image|pub async fn vfs_get_pdf_processing_status|pub async fn vfs_get_batch_pdf_processing_status|VfsBlobBase64Result" src-tauri/src -S`: no old Rust command definition, registration, or Rust-only DTO remains.
- `go test ./internal/vfs ./internal/bindings -count=1` from `desktop-go`: pass.
- `go test ./... -count=1` from `desktop-go`: pass.
- `npm run test -- tests/vitest/learning-hub/vfsAttachmentDeleteContract.source.test.ts tests/vitest/chat-v2/vfsPdfProcessingApi.test.ts`: pass, 13 tests.
- `npm run typecheck`: pass.
- `npm run native:triage`: pass, 643 commands.
- `npm run native:inventory -- --summary`: pass, 1526 scanned files / 951 native references / 858 invokes / 643 unique invokes / 65 listens / 47 unique listens / 28 emits / 24 unique emits.
- `node scripts/rust-retirement-map.mjs`: pass; merged Rust registrations reduced from 25 to 22, merged Rust definitions from 49 to 46, retirement candidates from 33 to 30, and `src-tauri/src/vfs/handlers.rs` merged Go commands still present in Rust from 9 to 6.
- `git diff --check` on this slice's touched Rust/test/generated-doc files: pass.
- `npx gitnexus detect_changes --repo "Deep Student"` was attempted and still fails because this GitNexus CLI build does not provide `detect_changes`.
- `cargo check` was intentionally not used as this slice's verification gate. The acceptance evidence is Go/Wails read-only PDF parity, source-contract coverage, and the reduced Rust retirement map.

Known gap:

- This slice does not upgrade Go PDF processing to real native raster/OCR. It only removes old Rust read-only status/page-image wrappers already covered by Go/Wails. Start/retry/cancel PDF processing and richer progress events remain separate migration work.

## Completed Index Status Summary Parity Slice

Resolved the Learning Hub index status progress mismatch introduced by the Go/Wails compact index summary shape.

What changed:

- Added a small frontend normalization layer in `src/features/learning-hub/views/IndexStatusView.tsx` for resource index counters.
- Text progress now falls back from Go text-specific counters to legacy `totalResources` / `indexedCount` and display counters when older mocks or transitional backends omit `textTotalResources` / `textIndexedCount`.
- Display progress and the one-click text indexing queue count now use the same normalized count source, avoiding split behavior between the progress UI and action scope.
- Image/multimodal progress defaults to `0/0` when multimodal counters are absent instead of rendering `undefined/undefined`.

GitNexus / impact notes:

- `npx gitnexus impact --repo "Deep Student" --direction upstream IndexStatusView`: LOW risk, 0 impacted callers.
- `npx gitnexus detect_changes --repo "Deep Student"` was attempted and still fails because this GitNexus CLI build does not provide `detect_changes`.
- The earlier subagent scan attempt failed with model access 403 for the `explore` role; no child-agent edits were made.

Verification:

- `npm run test -- tests/vitest/learning-hub/IndexStatusView.behavior.test.tsx`: pass, 5 tests. This clears the previous `indexStatus.progress.textIndexProgress undefined/undefined` failure.
- `npm run typecheck`: pass.
- `git diff --check -- src/features/learning-hub/views/IndexStatusView.tsx`: pass.
- `npm run native:inventory -- --summary`: pass, 1526 scanned files / 951 native references / 858 invokes / 643 unique invokes / 65 listens / 47 unique listens / 28 emits / 24 unique emits.
- `node scripts/rust-retirement-map.mjs`: pass; merged Rust registrations remain 22, merged Rust definitions remain 46, retirement candidates remain 30, direct Tauri blockers remain 33 commands / 33 edges / 6 files.
- `cargo check` was intentionally not used because this was a Go/Wails/frontend parity slice and did not edit old Rust.

Known gap:

- This slice only fixes summary counter compatibility. It does not retire additional Rust VFS commands or add real native PDF/OCR indexing.

## Completed VFS Index/PDF Control Rust Retirement Slice

Retired the final six Go/Wails-backed command wrappers from the old Rust/Tauri VFS handler retirement batch:

- `vfs_batch_index_pending`
- `vfs_cancel_pdf_processing`
- `vfs_get_all_index_status`
- `vfs_reindex_resource`
- `vfs_retry_pdf_processing`
- `vfs_start_pdf_processing`

What changed:

- Removed the six old Tauri command registrations from `src-tauri/src/lib.rs`.
- Removed the old Rust command wrappers from `src-tauri/src/vfs/handlers.rs`.
- Removed now-stale Rust command-only DTO/helper code tied to the deleted resource index wrappers: `BatchIndexResult`, `ResourceIndexStatus`, `IndexStatusSummary`, `requeue_embedding_dimension_config_failures`, and `reconcile_completed_text_indexing_resources`.
- Kept legacy command names active through `src/runtime/wailsBridge.ts`, where they route to Go `VfsService`.
- Fixed Go `VfsService.UnifiedIndexStatus`, `GetResourceUnits`, and `GetAllIndexStatus` to filter soft-deleted resources before exposing units/list rows/summary counts.
- Updated `tests/vitest/learning-hub/indexStatusStateSource.source.test.ts` so it protects the Go/Wails command surface and Rust retirement state instead of requiring the deleted Rust SQL wrapper to exist.

GitNexus / impact notes:

- `vfs_get_all_index_status`, `vfs_reindex_resource`, `vfs_batch_index_pending`, `vfs_cancel_pdf_processing`, `vfs_retry_pdf_processing`, and `vfs_start_pdf_processing`: all LOW risk, 0 impacted callers for the old Rust command wrappers.
- `IndexStatusView`: LOW risk, 0 impacted callers.
- `GetAllIndexStatus`, `resourceIsDeleted`, and `indexStatusStateSource` are not represented in the current GitNexus symbol index yet.
- A read-only default subagent confirmed all six legacy names are Go/Wails bridged and reported the remaining event-parity risk; no child-agent edits were made.
- `npx gitnexus detect_changes --repo "Deep Student"` was attempted and still fails because this GitNexus CLI build does not provide `detect_changes`.

Verification:

- `rg "vfs_batch_index_pending|vfs_cancel_pdf_processing|vfs_get_all_index_status|vfs_reindex_resource|vfs_retry_pdf_processing|vfs_start_pdf_processing|BatchIndexResult|ResourceIndexStatus" src-tauri/src/vfs/handlers.rs src-tauri/src/lib.rs -S`: no old Rust command definition, Tauri registration, or stale command DTO remains.
- `go test ./internal/vfs ./internal/bindings -count=1` from `desktop-go`: pass.
- `go test ./... -count=1` from `desktop-go`: pass.
- `npm run test -- tests/vitest/learning-hub/indexStatusStateSource.source.test.ts tests/vitest/learning-hub/IndexStatusView.behavior.test.tsx tests/vitest/chat-v2/vfsPdfProcessingApi.test.ts tests/vitest/chat-v2/inputBarMediaRoutingContract.source.test.ts`: pass, 25 tests.
- `npm run typecheck`: pass.
- `npm run native:triage`: pass, 643 commands.
- `npm run native:inventory -- --summary`: pass, 1526 scanned files / 951 native references / 858 invokes / 643 unique invokes / 65 listens / 47 unique listens / 28 emits / 24 unique emits.
- `node scripts/rust-retirement-map.mjs`: pass; merged Rust registrations reduced from 22 to 16, merged Rust definitions from 46 to 40, retirement candidates from 30 to 24, and `src-tauri/src/vfs/handlers.rs` no longer appears in the merged-Go-command retirement batch list.
- `git diff --check` on this slice's touched Go/Rust/test/progress files: pass.
- `cargo check` was intentionally not used as this slice's proof. The target runtime is Go/Wails, and this slice's acceptance evidence is Go service behavior, Wails legacy routing, frontend/source contracts, native inventory/triage, and the reduced Rust retirement map.

Known gap:

- Go `BatchIndexPending`, `ReindexResource`, and PDF start/retry/cancel controls are still lean compatibility paths. A follow-up slice added the UI-required progress/completion/error events, but these commands still do not implement real native PDF raster/OCR processing or semantic embedding work.

## Completed Go VFS Progress Event Bridge Slice

Added the lean Go/Wails event path that the previous retirement slice intentionally left open, without copying the old Rust all-virtual VFS processing stack.

What changed:

- Added an event emitter to Go `VfsService` and wired it into the shared Wails `EventBus` in `desktop-go/internal/app/app.go`.
- `StartPdfProcessing` and `RetryPdfProcessing` now emit unified `media-processing-progress` / `media-processing-completed` events and legacy `pdf-processing-progress` / `pdf-processing-completed` events for PDF resources.
- `CancelPdfProcessing` now emits unified `media-processing-error` and legacy `pdf-processing-error` for active PDF resources.
- `ReindexResource` now emits `vfs-index-progress` single-resource `started` / `completed` / `failed` events.
- `BatchIndexPending` now emits `vfs-index-progress` `batch_started`, per-resource `resource_completed`, and `batch_completed` events over compact Go hybrid VFS index states.
- Added app-level proof that VFS events emitted by `App.New()` reach `app.Events`, which is forwarded to Wails in `cmd/deep-student-go/main.go`.

GitNexus / impact notes:

- `StartPdfProcessing`, `RetryPdfProcessing`, `CancelPdfProcessing`, `ReindexResource`, and `BatchIndexPending` are not represented in the current GitNexus symbol index yet; `npx gitnexus impact` returned target-not-found for these new Go symbols.
- `npx gitnexus detect_changes --repo "Deep Student"` was attempted again and still fails because this GitNexus CLI build does not provide `detect_changes`.
- A verifier subagent reviewed the slice read-only and confirmed the implementation satisfies VFS EventBus wiring, PDF/unified events for PDF resources, VFS index progress events, and the non-copying lean architecture direction. It requested stronger proof for retry, legacy cancel errors, and app-level event propagation; those tests were added.

Verification:

- `go test ./internal/vfs -run "TestPdfProcessingControlCommandsEmitProgressEvents|TestIndexCommandsEmitVfsProgressEvents" -count=1` from `desktop-go`: pass.
- `go test ./internal/app -run TestNewWiresVfsEventsToEventBus -count=1` from `desktop-go`: pass.
- `go test ./... -count=1` from `desktop-go`: pass.
- `npm run test -- tests/vitest/learning-hub/indexStatusStateSource.source.test.ts tests/vitest/learning-hub/IndexStatusView.behavior.test.tsx tests/vitest/chat-v2/vfsPdfProcessingApi.test.ts tests/vitest/chat-v2/inputBarMediaRoutingContract.source.test.ts`: pass, 25 tests.
- `npm run typecheck`: pass.
- `npm run native:triage`: pass, 643 commands.
- `npm run native:inventory -- --summary`: pass, 1526 scanned files / 951 native references / 858 invokes / 643 unique invokes / 65 listens / 47 unique listens / 28 emits / 24 unique emits.
- `node scripts/rust-retirement-map.mjs`: pass; counts remain 16 merged Rust registrations, 40 merged Rust definitions, 24 retirement candidates, 33 direct Tauri blocked merged commands.
- `cargo check` was intentionally not used as the proof. This slice only changed Go/Wails runtime behavior and tests, and the target is retiring the old Rust runtime rather than maintaining it.

Known gap:

- These events are compatibility/status events over the current lean Go processing path. Real PDF raster rendering, OCR, semantic embeddings, and provider-quality indexing still remain separate implementation slices.

## Completed Frontend VFS Native Event Facade Slice

Closed the downstream listener gap for the new Go VFS event bridge. The Go backend now emits Wails events, and the frontend VFS/media listeners now listen through the existing native event facade instead of directly importing Tauri events.

What changed:

- Migrated `src/hooks/usePdfProcessingProgress.ts` from direct `@tauri-apps/api/event` listening to `@/runtime/nativeEvents`.
- Migrated `src/features/learning-hub/views/IndexStatusView.tsx` `vfs-index-progress` / `mm_index_progress` listeners to `@/runtime/nativeEvents`.
- Migrated `src/debug-panel/plugins/MediaProcessingDebugPlugin.tsx` media/PDF debug event listeners to `@/runtime/nativeEvents`.
- Migrated `src/features/chat/debug/attachmentPipelineTestPlugin.ts` media event capture to `@/runtime/nativeEvents`.
- Updated `tests/vitest/learning-hub/IndexStatusView.behavior.test.tsx` to mock the native event facade.
- Added a source-contract assertion in `tests/vitest/chat-v2/inputBarMediaRoutingContract.source.test.ts` requiring the VFS/media listener files to use `@/runtime/nativeEvents` and not direct `@tauri-apps/api/event`.

GitNexus / impact notes:

- `usePdfProcessingProgress`: HIGH risk because it directly affects `InputBarUI` and the attachment progress/error/completed flows. The edit was intentionally limited to changing the listener facade; state updates and payload handling were left unchanged.
- `IndexStatusView`: LOW risk, 0 impacted callers.
- `MediaProcessingDebugPlugin`: LOW risk, 0 impacted callers.
- `createMediaProcessingCapture`: LOW risk, direct debug-only impact through `runSingleTestCase`.
- `npx gitnexus detect_changes --repo "Deep Student"` was attempted again and still fails because this GitNexus CLI build does not provide `detect_changes`.

Verification:

- `npm run test -- tests/vitest/learning-hub/IndexStatusView.behavior.test.tsx tests/vitest/chat-v2/inputBarMediaRoutingContract.source.test.ts`: pass, 10 tests.
- `npm run test -- tests/vitest/learning-hub/indexStatusStateSource.source.test.ts tests/vitest/learning-hub/IndexStatusView.behavior.test.tsx tests/vitest/chat-v2/vfsPdfProcessingApi.test.ts tests/vitest/chat-v2/inputBarMediaRoutingContract.source.test.ts`: pass, 26 tests.
- `npm run typecheck`: pass.
- `go test ./... -count=1` from `desktop-go`: pass.
- `npm run native:triage`: pass, 643 commands.
- `npm run native:inventory -- --summary`: pass, 1526 scanned files / 951 native references / 858 invokes / 643 unique invokes / 65 listens / 47 unique listens / 28 emits / 24 unique emits.
- `node scripts/rust-retirement-map.mjs`: pass; counts remain 16 merged Rust registrations, 40 merged Rust definitions, 24 retirement candidates, 33 direct Tauri blocked merged commands.

Known gap:

- Many unrelated product areas still directly import `@tauri-apps/api/event`; this slice only covered VFS/media/index progress listeners needed by the new Go VFS event bridge. Continue migrating event listeners by workflow as their Go/Wails backend event paths are implemented.

## Completed Anki Native Event Facade Slice

Closed the Anki/CardForge listener gap for the Go/Wails `anki_generation_event` path. The Go Anki service already emits through the shared Wails `EventBus`; this slice makes the core frontend Anki listeners and tests consume that event through the native facade instead of direct Tauri event APIs.

What changed:

- Migrated `src/components/anki/cardforge/engines/CardEngine.ts` from direct `@tauri-apps/api/event` listening to `@/runtime/nativeEvents`.
- Confirmed `src/components/anki/cardforge/engines/CardAgent.ts` uses `@/runtime/nativeEvents` and `@/runtime/native`.
- Migrated `src/debug-panel/plugins/ChatAnkiWorkflowDebugPlugin.tsx` and `src/features/chat/debug/chatAnkiIntegrationTestPlugin.ts` to `@/runtime/nativeEvents`; the debug integration plugin also now invokes through `@/runtime/native`.
- Updated `tests/vitest/anki/cardforge/CardAgent.test.ts` to mock `@/runtime/native` and `@/runtime/nativeEvents`, matching production imports instead of stale direct Tauri modules.
- Added `tests/vitest/anki/cardforge/ankiEventFacade.source.test.ts` as a source-contract guard for the main `anki_generation_event` listener files.
- Fixed `CardAgent`'s pre-backend-call card collector to queue early document-scoped card/complete/pause events until the returned `documentId` is known, then flush them with the same document filtering. Early events without a `documentId` are dropped instead of replayed against the new generation, which avoids premature completion or card contamination while preserving the existing race-prevention design.

GitNexus / impact notes:

- `CardEngine`: LOW risk, 0 impacted callers.
- `ChatAnkiWorkflowDebugPlugin`: LOW risk, 0 impacted callers.
- `createAnkiEventCapture`: LOW risk, direct debug-only impact through `runAllChatAnkiTests`.
- `runChatAnkiIntegrationTest`: target not found in the current GitNexus index.
- `createCardCollector`: LOW risk; one direct caller, `generateCards`, and one indirect test-file impact.
- `npx gitnexus detect_changes --repo "Deep Student"` was attempted again and still fails because this GitNexus CLI build does not provide `detect_changes`.

Verification:

- `npm run test -- tests/vitest/anki/cardforge/ankiEventFacade.source.test.ts tests/vitest/anki/cardforge/CardAgent.test.ts`: pass, 5 tests.
- `npm run test -- tests/vitest/learning-hub/indexStatusStateSource.source.test.ts tests/vitest/learning-hub/IndexStatusView.behavior.test.tsx tests/vitest/chat-v2/vfsPdfProcessingApi.test.ts tests/vitest/chat-v2/inputBarMediaRoutingContract.source.test.ts tests/vitest/anki/cardforge/ankiEventFacade.source.test.ts tests/vitest/anki/cardforge/CardAgent.test.ts`: pass, 31 tests.
- `npm run typecheck`: pass.
- `go test ./... -count=1` from `desktop-go`: pass.
- `npm run native:triage`: pass, 643 commands.
- `npm run native:inventory -- --summary`: pass, 1526 scanned files / 951 native references / 858 invokes / 643 unique invokes / 65 listens / 47 unique listens / 28 emits / 24 unique emits.
- `node scripts/rust-retirement-map.mjs`: pass; counts remain 16 merged Rust registrations, 40 merged Rust definitions, 24 retirement candidates, 33 direct Tauri blocked merged commands.
- `cargo check` was intentionally not used as this slice's proof. This was a Go/Wails frontend event-facade and TS/Go compatibility slice, and the target runtime is not Tauri/Rust.

Known gap:

- This only covers the main Anki/CardForge event listener path. The Go Anki document worker is still a lean provider-backed text worker; OCR/text extraction orchestration, robust LLM-aware segmentation, richer progress semantics, old Rust session/card migration, and any remaining non-event direct Tauri imports in broader Anki/debug surfaces remain separate work.

## Completed System WebView Settings Native Facade Slice

Reduced one direct Tauri blocker by moving the WebView localStorage backup path onto the Go/Wails system service and native facade.

What changed:

- Added `SystemService.SaveWebviewSettings` in Go, writing `webview_settings.json` under the Go app data directory and returning `{ success, path, size }` compatible with the old Rust command shape.
- Exposed the method through `desktop-go/internal/bindings/SystemService` and regenerated Wails bindings with `npm run go:bindings`.
- Added the legacy `save_webview_settings` route to `src/runtime/wailsBridge.ts`.
- Added a fallback branch in `src/runtime/native.ts` so non-native tests/web contexts still get a harmless success result.
- Migrated `src/utils/systemApi.ts` `saveWebviewSettings` from direct Tauri invoke to `nativeInvoke`.
- Added `tests/vitest/system/nativeSystemFacade.source.test.ts` to lock the bridge/facade route.
- Widened the Go payload from `map[string]string` to `map[string]any` after review, preserving the old Rust `serde_json::Value` behavior for non-string/nested JSON values.

GitNexus / impact notes:

- `saveWebviewSettings`: LOW risk; direct callers are `App.persistWebviewSettings` and `DataImportExport.handleExport`, with indirect `App` impact.
- `SaveWebviewSettings` and `invokeWails`: target not found in the current GitNexus index, because the new Go method/generated binding route is not indexed yet.
- A read-only reviewer warned not to migrate broad `settingsApi.ts` research/statistics commands yet because most do not have Wails bridge routes. This slice intentionally stayed on the single bridged system command.
- A second read-only reviewer found the initial `map[string]string` payload narrowed the old Rust contract; the payload was widened to `map[string]any`, and the Go test now covers string, number, boolean, and nested JSON values.
- `npx gitnexus detect_changes --repo "Deep Student"` was attempted again and still fails because this GitNexus CLI build does not provide `detect_changes`.

Verification:

- `npm run go:bindings`: pass; generated bindings now report 11 services, 301 methods, 1 enum, and 198 models after the follow-up backup config slice.
- `go test ./internal/system ./internal/bindings -count=1` from `desktop-go`: pass.
- `go test ./internal/system ./internal/settings ./internal/bindings -count=1` from `desktop-go`: pass after widening the WebView settings payload.
- `go test ./... -count=1` from `desktop-go`: pass.
- `npm run test -- tests/vitest/system/nativeSystemFacade.source.test.ts`: pass, 1 test.
- `npm run test -- tests/vitest/system/nativeSystemFacade.source.test.ts tests/vitest/anki/cardforge/ankiEventFacade.source.test.ts tests/vitest/anki/cardforge/CardAgent.test.ts tests/vitest/learning-hub/indexStatusStateSource.source.test.ts tests/vitest/learning-hub/IndexStatusView.behavior.test.tsx tests/vitest/chat-v2/vfsPdfProcessingApi.test.ts tests/vitest/chat-v2/inputBarMediaRoutingContract.source.test.ts`: pass, 32 tests.
- `npm run typecheck`: pass.
- `npm run native:triage`: pass, 643 commands.
- `npm run native:inventory -- --summary`: pass, 1526 scanned files / 951 native references / 858 invokes / 643 unique invokes / 65 listens / 47 unique listens / 28 emits / 24 unique emits.
- `node scripts/rust-retirement-map.mjs`: pass; merged commands with Wails bridge routes increased from 291 to 292, direct Tauri blocked merged commands dropped from 33 to 32, blocker edges from 33 to 32, and blocker files from 6 to 5.
- `cargo check` was intentionally not used as this slice's proof. This was a Go/Wails system service and frontend native-facade slice; the old Rust command remains reference code until broader data-governance/backup replacement is complete.

Known gap:

- `load_webview_settings` is still old Rust-only and is not yet bridged to Go. The broader data-governance backup/export surface still has direct Tauri callers and should be migrated in small command-backed batches, not by blindly replacing `@tauri-apps/api/core` imports in `settingsApi.ts`.

## Completed Backup Config Native Facade Slice

Moved the lightweight backup configuration read/write pair onto Go/Wails without copying the old Rust backup scheduler or archive engine.

What changed:

- Added Go `SettingsService.GetBackupConfig` / `SetBackupConfig` over the existing settings store key `backup.config`.
- Preserved the old Rust `BackupConfig` camelCase shape and defaults: `backupDirectory: null`, `autoBackupEnabled: false`, `autoBackupIntervalHours: 24`, `maxBackupCount: 5`, `slimBackup: false`, with optional `backupTiers`.
- Exposed the methods through `desktop-go/internal/bindings/SettingsService` and regenerated Wails bindings.
- Added `get_backup_config` / `set_backup_config` legacy routes to `src/runtime/wailsBridge.ts`.
- Added native fallback behavior in `src/runtime/native.ts`.
- Migrated `src/api/dataGovernance.ts` `getBackupConfig` / `setBackupConfig` from direct Tauri invoke to `nativeInvoke`.
- Updated the data-governance contract test so only these migrated config commands assert native facade usage, while the broader old data-governance command surface remains on direct Tauri until each command has a Go/Wails bridge.
- Added `tests/vitest/data-governance/nativeDataGovernanceFacade.source.test.ts` as a source-contract guard.

GitNexus / impact notes:

- `getBackupConfig`: LOW risk; only direct indexed impact is the data-governance API contract test.
- `setBackupConfig`: LOW risk; only direct indexed impact is the data-governance API contract test.
- `fallbackInvoke`, `SaveWebviewSettings`, and `invokeWails`: target not found in the current GitNexus index.
- `npx gitnexus detect_changes --repo "Deep Student"` was attempted again and still fails because this GitNexus CLI build does not provide `detect_changes`.

Verification:

- `npm run go:bindings`: pass; generated bindings now report 11 services, 301 methods, 1 enum, and 198 models.
- `go test ./internal/system ./internal/settings ./internal/bindings -count=1` from `desktop-go`: pass.
- `go test ./... -count=1` from `desktop-go`: pass.
- `npm run test -- tests/vitest/data-governance/dataGovernance.api-contract.test.ts -t "Backup Config"`: pass, 7 tests.
- `npm run test -- tests/vitest/system/nativeSystemFacade.source.test.ts tests/vitest/data-governance/nativeDataGovernanceFacade.source.test.ts`: pass, 2 tests.
- `npm run typecheck`: pass.
- `npm run native:triage`: pass, 643 commands.
- `npm run native:inventory -- --summary`: pass, 1526 scanned files / 951 native references / 858 invokes / 643 unique invokes / 65 listens / 47 unique listens / 28 emits / 24 unique emits.
- `node scripts/rust-retirement-map.mjs`: pass; merged commands with Wails bridge routes increased from 292 to 294, direct Tauri blocked merged commands dropped from 32 to 30, blocker edges from 32 to 30, and blocker files from 5 to 4.
- Full `tests/vitest/data-governance/dataGovernance.api-contract.test.ts` still has 26 pre-existing non-backup failures where other data-governance commands pass camelCase params while the old contract expects snake_case. The backup config tests pass and this slice did not change those unrelated commands.
- `cargo check` was intentionally not used as this slice's proof. This was a Go/Wails settings/data-governance facade slice, and old Rust backup execution remains reference/runtime only until broader data-governance replacement is built.

Known gap:

- This only migrates backup configuration read/write. Actual backup job execution, restore, audit, schema, asset scan, ZIP import/export, cloud sync, and `load_webview_settings` remain old Tauri/Rust or direct Tauri-call surfaces until separate Go/Wails slices are implemented.

## Completed Statistics Native Facade Slice

Moved the visible data/statistics API pair onto Go/Wails without copying the old Rust mistake database analytics.

What changed:

- Added Go `settings.BasicStatistics`, `ImageStatistics`, and `EnhancedStatistics` DTOs.
- Added `SettingsService.GetStatistics` as a lean zero/default basic-statistics compatibility response.
- Added `SettingsService.GetEnhancedStatistics` through the Wails binding layer, combining the basic stats with image file counts and bytes estimated from the Go hybrid VFS file index.
- Regenerated Wails bindings; generated output now reports 11 services, 305 methods, 1 enum, and 202 models.
- Added `get_statistics` / `get_enhanced_statistics` legacy routes to `src/runtime/wailsBridge.ts`.
- Added browser/native fallback responses in `src/runtime/native.ts`.
- Migrated `src/utils/settingsApi.ts` `getStatistics` / `getEnhancedStatistics` from direct Tauri invoke to `nativeInvoke`.
- Added Go and TS contract tests:
  - `desktop-go/internal/settings/service_test.go` covers default and enhanced statistics shape.
  - `desktop-go/internal/bindings/settings_service_test.go` covers VFS image count/size aggregation.
  - `tests/vitest/runtime/wailsBridgeSettingsSystem.behavior.test.ts` covers Wails bridge forwarding.
  - `tests/vitest/settings/nativeStatisticsFacade.source.test.ts` guards frontend routing.

GitNexus / impact notes:

- `getEnhancedStatistics`: HIGH risk because it directly feeds `useEnhancedStatistics`, `useReviewStatistics`, `useAllStatistics`, `SOTADashboard`, and `DataImportExport`. The edit was intentionally constrained to native routing and compatible response shape; UI logic was not changed.
- `SettingsService`: LOW risk; direct indexed impact is `NewSettingsService` and Wails app startup registration.
- `VfsService`, `GetStatistics`, and the new Go methods are not fully represented in the current GitNexus symbol index yet.
- `npx gitnexus detect_changes --repo "Deep Student"` is still unavailable in this GitNexus CLI build; `rust-retirement-map` plus focused tests are the substitute scope check.

Verification:

- `npm run go:bindings`: pass; generated bindings report 11 services, 305 methods, 1 enum, 202 models.
- `go test ./... -count=1` from `desktop-go`: pass.
- `npm run test -- tests/vitest/runtime/wailsBridgeSettingsSystem.behavior.test.ts tests/vitest/settings/nativeStatisticsFacade.source.test.ts tests/vitest/system/nativeSystemFacade.source.test.ts`: pass, 3 tests.
- `npm run typecheck -- --pretty false`: pass.
- `npm run native:triage`: pass, 643 commands.
- `npm run native:inventory -- --summary`: pass, 1526 scanned files / 951 native references / 858 invokes / 643 unique invokes / 65 listens / 47 unique listens / 28 emits / 24 unique emits.
- `node scripts/rust-retirement-map.mjs`: pass; merged commands with Wails bridge routes increased from 294 to 298 across the latest WebView/backup/statistics slices, direct Tauri blocked merged commands dropped from 30 to 26, blocker edges from 30 to 26, and blocker files from 4 to 3.
- `cargo check` was intentionally not used. This was a Go/Wails statistics/facade slice and did not edit old Rust runtime code.

Known gap:

- This is a lean compatibility statistics path. It does not yet migrate old mistake/review analytics, monthly trend computation, quality scoring, or legacy Rust image folder scanning. The Go hybrid VFS image count/size is enough for the current visible statistics cards and export payload shape, but deeper analytics should be rebuilt from Go Qbank/Chat/VFS state when those product paths are the next focus.

## Completed AnkiConnect Metadata Native Facade Slice

Moved the settings-page AnkiConnect status/deck/model metadata path onto Go/Wails without expanding into the full Anki import/add-card workflow.

What changed:

- Added Go `AnkiService.CheckConnectStatus`, `ListDeckNames`, and `ListModelNames`.
- Added injectable AnkiConnect URL support for tests while keeping `http://127.0.0.1:8765` as the runtime default.
- Added shared Go AnkiConnect JSON-RPC request/response parsing for `version`, `deckNames`, and `modelNames`.
- Exposed the three methods through `desktop-go/internal/bindings/anki_service.go`.
- Regenerated Wails bindings; generated output now reports 11 services, 308 methods, 1 enum, and 202 models.
- Added legacy Wails bridge routes for `check_anki_connect_status`, `get_anki_deck_names`, `anki_get_deck_names`, and `get_anki_model_names`.
- Added browser/native fallbacks in `src/runtime/native.ts`: unavailable status by default, plus lightweight `Default`, `Basic`, and `Cloze` metadata lists for non-native development.
- Migrated `src/services/ankiConnectClient.ts` `check`, `listDecks`, and `listModels` to `nativeInvoke`; write/import commands remain on old Tauri until the broader Anki import slice is rebuilt in Go.
- Migrated `src/features/settings/components/AnkiConnectSettingsSection.tsx` test-connection metadata reads from dynamic `@tauri-apps/api/core` imports to `ankiConnectClient.listDecks()` / `listModels()`.
- Fixed `scripts/rust-retirement-map.mjs` so direct Tauri blocker detection also catches `(await import('@tauri-apps/api/core')).invoke(...)` property-access dynamic imports.
- Added Go and TS contract tests:
  - `desktop-go/internal/anki/service_test.go` covers AnkiConnect metadata query success and AnkiConnect error handling.
  - `tests/vitest/runtime/wailsBridgeSettingsSystem.behavior.test.ts` covers Wails bridge forwarding for AnkiConnect metadata commands.
  - `tests/vitest/anki/ankiConnectNativeFacade.source.test.ts` guards the frontend metadata facade and prevents reintroducing dynamic Tauri imports in the settings test path.

GitNexus / impact notes:

- `listModels`: LOW risk, 0 indexed upstream callers.
- old Rust `get_anki_model_names`: LOW risk, 0 indexed upstream callers.
- `check` and `listDecks`: LOW risk, 0 indexed upstream callers.
- old Rust `check_anki_connect_status`: LOW risk, 0 indexed upstream callers.
- old Rust `get_anki_deck_names`: LOW risk, one direct old Rust caller (`anki_get_deck_names` alias).
- old Rust `anki_get_deck_names`: LOW risk, 0 indexed upstream callers.
- `testConnection`: LOW risk, 0 indexed upstream callers.
- `tauriInvokeNames` was not found in the current GitNexus index; the script change was kept to one detection regex and verified through regenerated retirement-map output.

Verification:

- `npm run go:bindings`: pass; generated bindings report 11 services, 308 methods, 1 enum, 202 models.
- `go test ./... -count=1` from `desktop-go`: pass.
- `npm run test -- tests/vitest/runtime/wailsBridgeSettingsSystem.behavior.test.ts tests/vitest/anki/ankiConnectNativeFacade.source.test.ts tests/vitest/settings/nativeStatisticsFacade.source.test.ts tests/vitest/system/nativeSystemFacade.source.test.ts`: pass, 4 tests.
- `npm run typecheck -- --pretty false`: pass.
- `npm run native:inventory -- --summary`: pass, 1526 scanned files / 946 native references / 853 invokes / 640 unique invokes / 65 listens / 47 unique listens / 28 emits / 24 unique emits.
- `node scripts/native-triage.mjs && node scripts/rust-retirement-map.mjs`: pass; native triage now has 640 commands, and the retirement map reports 330 merged commands, 298 with Wails bridge routes, 31 retirement candidates, 25 direct Tauri blocked merged commands, 25 blocker edges, and 2 blocker files.
- `cargo check` was intentionally not used. This was a Go/Wails AnkiConnect metadata/facade slice and did not edit old Rust runtime code.

Known gap:

- This only covers read-only AnkiConnect metadata needed by settings/test-connection. `create_anki_deck`, `import_anki_package`, `add_cards_to_anki_connect`, Anki library/card CRUD/export, and old Anki session/card migration remain separate Go rebuild slices.
- `check_anki_connect_status` and `get_anki_model_names` can still appear under "Replace Commands Still Registered In Rust"; `get_anki_deck_names` is currently generated as `defer`. All three have Go bridge routes for the settings/test-connection metadata path, and they are no longer direct Tauri blockers in `docs/generated/rust-retirement-map.md`.

## Completed Memory Config Native Facade Slice

Moved the remaining isolated non-research direct Tauri blocker, `memory_get_config`, onto Go/Wails as a read-only settings compatibility route.

What changed:

- Added Go `settings.MemoryConfig` with camelCase output fields matching the old Rust `MemoryConfigOutput`.
- Added `SettingsService.GetMemoryConfig` with lean defaults: no root folder, `autoCreateSubfolders=true`, `defaultCategory=通用`, `privacyMode=false`, and `autoExtractFrequency=balanced`.
- Reads existing lightweight settings-store keys for memory root folder id/title, auto-subfolder behavior, default category, privacy mode, and extraction frequency.
- Normalizes unknown auto-extract frequency values back to `balanced`, matching old Rust lossy parsing behavior.
- Exposed `GetMemoryConfig` through Wails bindings and regenerated bindings; generated output now reports 11 services, 309 methods, 1 enum, and 203 models.
- Added `memory_get_config` route in `src/runtime/wailsBridge.ts` and browser/native fallback shape in `src/runtime/native.ts`.
- Migrated only `src/api/memoryApi.ts` `getMemoryConfig()` to `nativeInvoke`; the broader memory search/write/tree/profile commands remain direct Tauri/deferred until the memory product strategy is rebuilt.
- Added Go and TS tests:
  - `desktop-go/internal/settings/service_test.go` covers memory config defaults, stored values, stale root-title suppression, and frequency fallback.
  - `tests/vitest/runtime/wailsBridgeSettingsSystem.behavior.test.ts` covers Wails bridge forwarding.
  - `tests/vitest/memory/memoryApi.test.ts` now asserts `getMemoryConfig` uses the native facade while other memory commands keep their existing command contracts.
  - `tests/vitest/memory/memoryConfigNativeFacade.source.test.ts` guards the source-level native facade route.

GitNexus / impact notes:

- old Rust `memory_get_config`: LOW risk, 0 indexed upstream callers.
- frontend `getMemoryConfig`: HIGH risk because it directly affects `LearningHubSidebar`, `MemorySettingsSection`, `MemoryView`, and `MemoryFolderBanner`, plus related Learning Hub processes. The edit was intentionally limited to read-only native routing and same-shape defaults; UI behavior and memory mutation/search commands were not changed.
- `npx gitnexus detect_changes -r "Deep Student"` is unavailable in this CLI build (`unknown command 'detect_changes'`); generated retirement map plus focused tests are the substitute scope check.
- `npx gitnexus status` reports the index is up-to-date at commit `1812ec7`.

Verification:

- `npm run go:bindings`: pass; generated bindings report 11 services, 309 methods, 1 enum, 203 models.
- `go test ./... -count=1` from `desktop-go`: pass.
- `npm run test -- tests/vitest/runtime/wailsBridgeSettingsSystem.behavior.test.ts tests/vitest/anki/ankiConnectNativeFacade.source.test.ts tests/vitest/memory/memoryApi.test.ts tests/vitest/memory/memoryConfigNativeFacade.source.test.ts tests/vitest/settings/nativeStatisticsFacade.source.test.ts tests/vitest/system/nativeSystemFacade.source.test.ts`: pass, 22 tests.
- `npm run typecheck -- --pretty false`: pass.
- `npm run native:inventory -- --summary`: pass, 1526 scanned files / 946 native references / 853 invokes / 640 unique invokes / 65 listens / 47 unique listens / 28 emits / 24 unique emits.
- `node scripts/native-triage.mjs && node scripts/rust-retirement-map.mjs`: pass; native triage remains 640 commands, and the retirement map now reports 330 merged commands, 299 with Wails bridge routes, 32 retirement candidates, 24 direct Tauri blocked merged commands, 24 blocker edges, and 1 blocker file.
- `cargo check` was intentionally not used. This was a Go/Wails memory-config facade slice and did not edit old Rust runtime code.

Known gap:

- `memory_get_config` is now Go/Wails-backed but the old Rust command remains registered and appears in the retirement batch list under `src-tauri/src/memory/handlers.rs`; it can be deleted only as part of a deliberate Rust command retirement batch.
- The memory product surface itself is still not rebuilt in Go. `memory_search`, `memory_read`, `memory_write`, folder tree, relations, tags, profile/export/audit, smart/batch write, and memory-to-Anki remain old/deferred until the memory-as-hybrid-VFS strategy is chosen.

## Completed Retired Research Facades And Wails Runtime Detection Slice

Resolved the last direct Tauri blocker group in the merged-command retirement map by deleting dead Deep Research wrappers from `src/utils/settingsApi.ts`, removed the matching unused Deep Research report wrappers from `src/utils/chatApi.ts`, then fixed a review-found Wails v3 runtime detection gap that could have sent real Wails sessions into browser fallback data.

What changed:

- Confirmed the 26 `settingsApi.ts` `research*` wrapper exports had no indexed upstream callers and no exact production/test references outside their own definitions.
- Removed the retired Deep Research wrappers from `src/utils/settingsApi.ts` instead of mechanically rebuilding 24 stale `research_*` commands in Go.
- Confirmed `researchListReports`, `researchGetReport`, `researchDeleteReport`, and `researchExportAllReportsZip` in `src/utils/chatApi.ts` had no indexed upstream callers and no exact production/test references outside their own definitions.
- Removed the retired Deep Research report wrappers from `src/utils/chatApi.ts`; native triage no longer tracks `research_list_reports`, `research_get_report`, `research_delete_report`, or `research_export_all_reports_zip`.
- Added `tests/vitest/settings/retiredResearchSettingsFacade.source.test.ts` to prevent reintroducing direct Tauri `research_*` invokes in `settingsApi.ts`.
- Added `tests/vitest/chat-v2/retiredResearchReports.source.test.ts` to prevent reintroducing the retired Deep Research report commands in `chatApi.ts`.
- Extended `isWailsRuntime()` to recognize Wails v3/WebView markers used by `@wailsio/runtime`: `_wails.environment`, `_wails.flags`, Android `window.wails.invoke`, Windows `window.chrome.webview.postMessage`, macOS/iOS `window.webkit.messageHandlers.external.postMessage`, plus an explicit `__DEEP_STUDENT_WAILS__` app marker.
- Added a runtime behavior test proving Wails marker detection makes `native.invoke('get_anki_model_names')` and `native.invoke('memory_get_config')` call Wails bindings instead of browser fallback values.
- A background code-review agent found the Wails detection issue and the two medium research/direct-helper risks. The high-severity finding and the Deep Research report finding are fixed in this slice; Tauri-only MCP/search helpers in `settingsApi.ts` remain queued.

GitNexus / impact notes:

- `researchGetRound`, `researchGetRoundVisualSummary`, `researchDeleteRound`, `researchGenerateRoundReport`, `researchSetRoundNote`, `researchGetRoundNote`, `researchGetRoundNotes`, `researchGenerateSessionReport`, `researchGetChunkText`, `researchGetChunkContext`, `researchUpdateSessionOptions`, `researchDeleteSession`, `researchRunUntil`, `researchRunMacroRound`, `researchRunToFullCoverage`, `researchAuditUserQuestions`, `researchFindSimilarQuestions`, `researchGetFullChatHistory`, `researchDeepReadByDocs`, `researchDeepReadByTag`, `researchCountTokensPrecise`, `researchGetFullContentPrecise`, `researchGetSetting`, `researchSetSetting`, `researchDeleteSetting`, and `researchListArtifacts`: all LOW risk with 0 indexed upstream callers/processes/modules.
- `researchListReports`, `researchGetReport`, `researchDeleteReport`, and `researchExportAllReportsZip`: all LOW risk with 0 indexed upstream callers/processes/modules.
- `isWailsRuntime` is not present in the current GitNexus index; the edit was verified through source inspection of `@wailsio/runtime` and targeted runtime tests.

Verification:

- `npm run test -- tests/vitest/settings/retiredResearchSettingsFacade.source.test.ts tests/vitest/settings/nativeStatisticsFacade.source.test.ts`: pass, 2 tests.
- `npm run test -- tests/vitest/runtime/wailsBridgeSettingsSystem.behavior.test.ts tests/vitest/settings/retiredResearchSettingsFacade.source.test.ts`: pass, 3 tests.
- `npm run test -- tests/vitest/settings/retiredResearchSettingsFacade.source.test.ts tests/vitest/chat-v2/retiredResearchReports.source.test.ts tests/vitest/runtime/wailsBridgeSettingsSystem.behavior.test.ts`: pass, 4 tests.
- `npm run typecheck -- --pretty false`: pass.
- `npm run native:inventory -- --summary`: pass, 1526 scanned files / 917 native references / 824 invokes / 611 unique invokes / 65 listens / 47 unique listens / 28 emits / 24 unique emits.
- `node scripts/native-triage.mjs && node scripts/rust-retirement-map.mjs`: pass; native triage now has 611 unique commands, and the retirement map reports 306 merged commands, 299 with Wails bridge routes, 14 merged Rust registrations, 38 merged Rust definitions, 32 retirement candidates, 0 direct Tauri blocked merged commands, 0 blocker edges, 0 blocker files, and 77 replace commands still registered in Rust.
- `cargo check` was intentionally not used. This slice removed dead frontend wrappers and fixed runtime detection without editing old Rust runtime code.

Known gap:

- `settingsApi.ts` still contains direct Tauri-only helper commands for MCP transport tests, web-search checks, and search-engine health. These are no longer merged-command direct blockers, but they remain Wails replacement/quarantine work.
- The old Deep Research product surface is not rebuilt in Go. Current evidence says the removed wrappers were dead; if a research workflow is revived, design a lean Go product API instead of restoring the old direct Tauri command list.

## Completed API Connection Native Facade Slice

Moved the settings-page API connection test from Tauri/Rust to the Go/Wails settings service without broadening provider support beyond the Go chat runtime's current OpenAI-compatible path.

What changed:

- Added Go `SettingsService.TestAPIConnection` over `desktop-go/internal/settings.Service`.
- The Go test sends a minimal non-streaming OpenAI-compatible `/chat/completions` request with the selected model, `max_tokens: 1`, and `stream: false`.
- Intentionally does not use `/responses` yet because the current Go `ChatService` only supports chat completions. This prevents the settings test from passing a protocol path the app cannot actually use.
- Resolves non-masked explicit API keys first, then saved Go `VendorConfig` keys by `vendorId`, then matching `ApiConfig` keys, then the legacy `siliconflow.api_key` setting fallback.
- Rejects masked keys (`***` or all asterisks) instead of sending `Bearer ***`.
- Reuses saved vendor headers when a `vendorId` is supplied.
- Avoids duplicating `/chat/completions` when the base URL already points to that endpoint.
- Exposed `SettingsService.TestAPIConnection` through Go bindings and regenerated Wails bindings; generated output now reports 11 services, 310 methods, 1 enum, and 203 models.
- Routed legacy `test_api_connection` through `src/runtime/wailsBridge.ts`, accepting both snake_case and camelCase args from the current settings UI.
- Migrated `src/features/settings/components/useSettingsVendorState.tsx` and `src/utils/settingsApi.ts` from direct Tauri invoke to `nativeInvoke` for `test_api_connection`.
- Updated `scripts/native-triage.mjs` so `test_api_connection` is tracked as a merged settings command instead of a delete/debug command.
- Added `tests/vitest/settings/nativeApiConnectionFacade.source.test.ts` and extended `tests/vitest/runtime/wailsBridgeSettingsSystem.behavior.test.ts`.

GitNexus / impact notes:

- `useSettingsVendorState`: LOW risk; one direct caller (`Settings`) in the settings process.
- frontend `testApiConnection` wrapper: LOW risk, 0 indexed upstream callers.
- old Rust `test_api_connection`: LOW risk, 0 indexed upstream callers.
- binding `SettingsService`: LOW risk with direct impact to `NewSettingsService` and indirect impact to `main`.

Verification:

- `go test ./internal/settings ./internal/bindings -count=1`: pass.
- `npm run go:bindings`: pass; generated bindings report 11 services, 310 methods, 1 enum, 203 models.
- `npm run test -- tests/vitest/runtime/wailsBridgeSettingsSystem.behavior.test.ts tests/vitest/settings/nativeApiConnectionFacade.source.test.ts tests/vitest/settings/retiredResearchSettingsFacade.source.test.ts tests/vitest/chat-v2/retiredResearchReports.source.test.ts`: pass, 5 tests.
- `npm run typecheck -- --pretty false`: pass.
- `go test ./... -count=1` from `desktop-go`: pass.
- `npm run native:inventory -- --summary`: pass, 1526 scanned files / 917 native references / 824 invokes / 611 unique invokes / 65 listens / 47 unique listens / 28 emits / 24 unique emits.
- `node scripts/native-triage.mjs && node scripts/rust-retirement-map.mjs`: pass; native triage now has 611 unique commands, and the retirement map reports 307 merged commands, 300 with Wails bridge routes, 15 merged Rust registrations, 39 merged Rust definitions, 33 retirement candidates, 0 direct Tauri blocked merged commands, 0 blocker edges, 0 blocker files, and 77 replace commands still registered in Rust.
- `cargo check` was intentionally not used. This slice added the Go/Wails replacement and did not edit old Rust runtime code.

Known gap:

- The old Rust `test_api_connection` and missing `/responses` parity from this initial facade slice have now been resolved in the follow-up API Connection Responses Parity + Rust Retirement slice below.
- Other direct Tauri test/helper surfaces remain outside this slice: MCP stdio transport test and MCP HTTP/SSE/WebSocket/ModelScope checks. Provider-strategy config and `test_ocr_engine` have since been migrated/retired in later slices.

## Completed API Connection Responses Parity + Rust Retirement Slice

Completed the protocol parity gap for the Go/Wails settings API connection diagnostic and retired the old Rust/Tauri command surface:

- `test_api_connection`

What changed:

- Updated Go `SettingsService.TestAPIConnection` so `apiProtocol` and `supportsOpenAIResponses` are no longer ignored.
- Added Go protocol helpers for `openai_chat_completions` vs `openai_responses`, including endpoint switching between `/chat/completions` and `/responses` without duplicate suffixes.
- Preserved old Rust-compatible defaulting: explicit chat always uses chat completions; official `api.openai.com` and declared `supportsOpenAIResponses=true` use `/responses`; third-party explicit `openai_responses` without support declaration downgrades to chat completions.
- Split probe request bodies correctly: chat uses `messages`, `max_tokens: 1`, `stream:false`; responses uses `input:"Hi"`, `max_output_tokens: 1`, `stream:false`.
- Extended `src/utils/settingsApi.ts` so its compatibility helper can pass optional `apiProtocol`, `supportsOpenAIResponses`, and `vendorId` while keeping old callers compatible.
- Extended Wails bridge behavior/source contracts to lock snake_case and camelCase protocol fields, positional binding order, and the Go implementation not ignoring protocol hints.
- Removed old Rust `test_api_connection`, its local `resolve_test_api_protocol` helper, its Rust-only tests, and the Tauri registration. Kept Rust `should_use_openai_responses_for_config` because it is still CRITICAL shared LLM runtime logic.

GitNexus / impact notes:

- Old Rust `test_api_connection`: LOW risk, 0 indexed upstream callers.
- Old Rust `resolve_test_api_protocol`: LOW risk, direct upstream only `test_api_connection`.
- Go `Service`: LOW risk, direct caller `NewService`, indirect `app.New` and `main`.
- `should_use_openai_responses_for_config`: reviewed as out of scope because subagent impact found CRITICAL shared usage across LLM runtime/protocol normalization.

Metrics after this slice:

- Wails bindings: 11 services, 316 methods, 215 models.
- Native triage: 611 unique commands; `merge` 311, `replace` 86, `defer` 183, `delete` 31.
- Rust retirement map: 311 merged commands, 306 with Wails bridge routes, 6 merged commands still registered in Rust, 30 merged Rust definitions, 26 retirement candidates, 0 direct Tauri blockers, 77 replace commands still registered in Rust.

Verification:

- `go test ./internal/settings -run TestAPIConnection -count=1` from `desktop-go`: pass.
- `go test ./internal/settings ./internal/bindings -count=1` from `desktop-go`: pass.
- `npm run go:bindings`: pass; Wails generated 316 methods / 215 models.
- `npm run test -- tests/vitest/runtime/wailsBridgeSettingsSystem.behavior.test.ts tests/vitest/settings/nativeApiConnectionFacade.source.test.ts tests/vitest/settings/goApiConnectionResponses.source.test.ts`: pass.
- `go test ./... -count=1` from `desktop-go`: pass.
- `npm run typecheck -- --pretty false`: pass.
- `npm run native:inventory -- --summary`: pass, 1526 scanned files / 916 native references / 823 invokes / 611 unique invokes.
- `node scripts/native-triage.mjs && node scripts/rust-retirement-map.mjs`: pass; generated docs updated.
- Source search over `src-tauri/src/commands.rs` and `src-tauri/src/lib.rs`: no old Rust `test_api_connection`, `crate::commands::test_api_connection`, or `resolve_test_api_protocol` remains.
- Source search over `src` for `test_api_connection`: production calls remain only in `src/runtime/wailsBridge.ts`, `src/utils/settingsApi.ts`, and `src/features/settings/components/useSettingsVendorState.tsx`, all through the native/Wails route.
- Scoped `git diff --check` for touched Go/TS/Rust/generated files: pass.

Known gaps:

- This is a settings diagnostic parity slice, not full chat provider parity. Go `ChatService` still lacks broader Responses API/provider-specific reasoning/multimodal parity.
- `settingsApi.ts` still imports Tauri for other un-migrated helper commands in that file, but its `test_api_connection` helper now uses `nativeInvoke`.
- `rustfmt --edition 2021` was attempted for touched Rust files but is still blocked by pre-existing trailing whitespace in unrelated `src-tauri/src/translation/pipeline.rs`; no unrelated cleanup was made.
- `cargo check` was intentionally not used as this rewrite slice's default proof path; verification used Go tests, TS checks, source searches, and generated retirement-map evidence.

Next queue:

- Visible MCP legacy diagnostics cleanup is now covered by the later MCP cleanup slice; remaining MCP work is live stdio smoke, then possible retirement of old Rust `mcp_stdio_start/send/close`.
- Continue provider/runtime parity separately for chat: Responses API, reasoning/thinking, multimodal attachments, usage/cost metadata, and provider health/model list flows.

## Completed Web Search Diagnostics Native Facade Slice

Moved the user-visible web-search diagnostic buttons from direct Tauri/Rust calls onto the Go/Wails settings service.

What changed:

- Added Go `SettingsService.TestSearchEngine`, `TestWebSearchConnectivity`, and `TestAllSearchEngines`.
- The Go implementation reads the existing `web_search.*` settings and provider key environment fallbacks, then performs a minimal provider probe for `google_cse`, `serpapi`, `tavily`, `brave`, `searxng`, `zhipu`, and `bocha`.
- The response shapes preserve the existing UI contracts: `test_search_engine` returns `{ ok, message, response_time, test_query, error_details?, results_count? }`, and `test_all_search_engines` returns `{ results, summary, timestamp }`.
- Missing configuration returns a failed/not-configured result instead of throwing, so settings UI can show diagnostics without treating provider misconfiguration as a native bridge crash.
- Exposed the methods through `desktop-go/internal/bindings/SettingsService` and regenerated Wails bindings; generated output now reports 11 services, 313 methods, 1 enum, and 208 models.
- Routed legacy `test_search_engine`, `test_web_search_connectivity`, and `test_all_search_engines` through `src/runtime/wailsBridge.ts`.
- Added `testSearchEngine` / web-search health facades in `src/utils/settingsApi.ts` and migrated `src/features/settings/components/EngineSettingsSection.tsx` plus `src/components/SearchEngineStatus.tsx` off direct Tauri `test_search_engine`.
- Updated `scripts/native-triage.mjs` so the three web-search diagnostic commands are tracked as merged settings commands instead of delete/debug rows.
- Added `tests/vitest/settings/nativeWebSearchDiagnosticsFacade.source.test.ts` and extended `tests/vitest/runtime/wailsBridgeSettingsSystem.behavior.test.ts`.

GitNexus / impact notes:

- Old Rust `test_search_engine`, `test_web_search_connectivity`, and `test_all_search_engines`: LOW risk, 0 indexed upstream callers.
- `testWebSearchConnectivity`: LOW risk, 0 indexed upstream callers.
- `testAllSearchEngines`: LOW risk, one direct settings component caller.
- `SearchEngineStatus.testEngine`, `SearchEngineStatus`, `EngineSettingsSection`, and `SearchEngineStatus.testAllEngines`: LOW risk in the current index.
- `invokeWails` and new Go search diagnostic methods are not represented as indexed GitNexus symbols yet, so bridge coverage is guarded with Wails behavior/source tests.

Verification:

- `go test ./internal/settings ./internal/bindings -count=1`: pass.
- `go test ./... -count=1` from `desktop-go`: pass.
- `npm run go:bindings`: pass; generated bindings report 11 services, 313 methods, 1 enum, 208 models.
- `npm run test -- tests/vitest/runtime/wailsBridgeSettingsSystem.behavior.test.ts tests/vitest/settings/nativeWebSearchDiagnosticsFacade.source.test.ts tests/vitest/settings/nativeApiConnectionFacade.source.test.ts tests/vitest/settings/retiredResearchSettingsFacade.source.test.ts tests/vitest/chat-v2/retiredResearchReports.source.test.ts`: pass, 6 tests.
- `npm run typecheck -- --pretty false`: pass.
- `npm run native:inventory -- --summary`: pass, 1526 scanned files / 916 native references / 823 invokes / 611 unique invokes / 65 listens / 47 unique listens / 28 emits / 24 unique emits.
- `node scripts/native-triage.mjs && node scripts/rust-retirement-map.mjs`: pass; native triage has 611 unique commands, and the retirement map reports 310 merged commands, 303 with Wails bridge routes, 17 merged Rust registrations, 42 merged Rust definitions, 36 retirement candidates, 0 direct Tauri blocked merged commands, 0 blocker edges, 0 blocker files, and 77 replace commands still registered in Rust.
- `cargo check` was intentionally not used. This slice added Go/Wails replacements and did not edit old Rust runtime code.

Known gap:

- Old Rust `test_search_engine`, `test_all_search_engines`, and `test_web_search_connectivity` have now been retired in the follow-up Rust retirement slice below.
- This is a settings diagnostic slice, not the full Go rebuild of chat-time `web_search` tool execution/reranking/injection.
- MCP stdio/remote test helpers and `test_ocr_engine` remain separate higher-scope slices.

## Completed Web Search Diagnostics Rust Retirement Slice

Deleted the old Tauri/Rust command surface for the web-search diagnostics that are now served by Go/Wails:

- Removed `test_web_search_connectivity` and `test_all_search_engines` from `src-tauri/src/cmd/web_search.rs`.
- Removed `test_search_engine` from `src-tauri/src/cmd/web_search.rs`.
- Removed `test_search_engine` and `test_all_search_engines` from the Tauri `invoke_handler` registration list in `src-tauri/src/lib.rs`.
- Kept the legacy command names available through `src/runtime/wailsBridge.ts`, where they dispatch to Go `SettingsService`.

GitNexus / impact notes:

- Old Rust `test_search_engine`: LOW risk, 0 indexed upstream callers.
- Old Rust `test_all_search_engines`: LOW risk, 0 indexed upstream callers.
- Old Rust `test_web_search_connectivity`: LOW risk, 0 indexed upstream callers.

Verification:

- `go test ./... -count=1` from `desktop-go`: pass.
- `npm run test -- tests/vitest/runtime/wailsBridgeSettingsSystem.behavior.test.ts tests/vitest/settings/nativeWebSearchDiagnosticsFacade.source.test.ts`: pass, 3 tests.
- `npm run typecheck -- --pretty false`: pass.
- `npm run native:inventory -- --summary`: pass, 1526 scanned files / 916 native references / 823 invokes / 611 unique invokes / 65 listens / 47 unique listens / 28 emits / 24 unique emits.
- `git diff --check -- src-tauri/src/cmd/web_search.rs src-tauri/src/lib.rs docs/GO_REWRITE_PROGRESS.md docs/generated/native-command-triage.md docs/generated/native-command-triage.json docs/generated/rust-retirement-map.md docs/generated/rust-retirement-map.json`: pass.
- `rg "test_web_search_connectivity|test_all_search_engines|test_search_engine" src-tauri/src src/runtime src/utils docs/GO_REWRITE_PROGRESS.md`: confirms no `src-tauri` definitions/registrations remain; only Go/Wails/frontend compatibility routes and progress notes reference the command names.
- `node scripts/native-triage.mjs && node scripts/rust-retirement-map.mjs`: pass; at this checkpoint the retirement map reported 310 merged commands, 303 with Wails bridge routes, 15 merged Rust registrations, 39 merged Rust definitions, 33 retirement candidates, 0 direct Tauri blocked merged commands, 0 blocker edges, 0 blocker files, and 77 replace commands still registered in Rust. Later retirement batches reduced these counts further; use the top-level summary for the current map.
- `cargo check` was intentionally not used. This slice removed already-retired Rust commands and used source search plus generated retirement-map evidence as the proof.

## Completed Provider Strategies Native Facade + Rust Retirement Slice

Moved the visible web-search provider strategy config path onto Go/Wails and removed the matching old Tauri/Rust commands.

What changed:

- Added Go settings DTOs and persistence for `web_search.provider_strategies`: `ProviderSpecialHandling`, `ProviderStrategy`, `ProviderStrategies`, and `ProviderStrategiesConfigResult`.
- Added `SettingsService.GetProviderStrategiesConfig` and `SettingsService.SaveProviderStrategiesConfig`, including default strategy hydration and legacy JSON round-trip support.
- Exposed both methods through `desktop-go/internal/bindings/SettingsService` and regenerated Wails bindings; generated output now reports 11 services, 315 methods, 1 enum, and 212 models.
- Routed legacy `get_provider_strategies_config` and `save_provider_strategies_config` through `src/runtime/wailsBridge.ts`.
- Added browser/localStorage fallback support in `src/runtime/native.ts` with typed provider-strategy defaults for dev/browser mode.
- Migrated `src/features/settings/components/EngineSettingsSection.tsx` from direct Tauri invoke to `nativeInvoke` for provider strategy load/save.
- Extended the web-search source contract and Wails bridge behavior tests to cover provider strategies.
- Removed the old Rust handlers from `src-tauri/src/cmd/web_search.rs`.
- Removed the old Tauri registrations from `src-tauri/src/lib.rs`.

GitNexus / impact notes:

- Old Rust `get_provider_strategies_config`: LOW risk, 0 indexed upstream callers/processes/modules.
- Old Rust `save_provider_strategies_config`: LOW risk, 0 indexed upstream callers/processes/modules.
- `EngineSettingsSection`: previous impact check was LOW risk, 0 indexed upstream callers in the current index.
- Go `SettingsService`: previous impact check was LOW risk with direct impact to `NewSettingsService` and indirect impact to `main`.
- `fallbackInvoke` is not present in the current GitNexus index; the browser fallback type fix was verified by TypeScript compile and source tests. A generic `invoke` impact query matched an unrelated old Rust tool symbol and was not part of this edit.

Verification:

- `go test ./internal/settings ./internal/bindings -count=1`: pass.
- `npm run go:bindings`: pass; generated bindings report 11 services, 315 methods, 1 enum, 212 models.
- `npm run test -- tests/vitest/runtime/wailsBridgeSettingsSystem.behavior.test.ts tests/vitest/settings/nativeWebSearchDiagnosticsFacade.source.test.ts`: pass, 3 tests.
- `go test ./... -count=1` from `desktop-go`: pass.
- `npm run typecheck -- --pretty false`: pass.
- `npm run native:inventory -- --summary`: pass, 1526 scanned files / 916 native references / 823 invokes / 611 unique invokes / 65 listens / 47 unique listens / 28 emits / 24 unique emits.
- `node scripts/native-triage.mjs && node scripts/rust-retirement-map.mjs`: pass; at this checkpoint the retirement map reported 310 merged commands, 305 with Wails bridge routes, 13 merged Rust registrations, 37 merged Rust definitions, 33 retirement candidates, 0 direct Tauri blocked merged commands, 0 blocker edges, 0 blocker files, and 77 replace commands still registered in Rust. Later retirement batches reduced these counts further; use the top-level summary for the current map.
- `rg "get_provider_strategies_config|save_provider_strategies_config" src-tauri/src src/runtime src/features tests/vitest docs/GO_REWRITE_PROGRESS.md`: confirms no `src-tauri` definitions/registrations remain; only Go/Wails/frontend compatibility routes, tests, and progress notes reference the command names.
- `cargo check` was intentionally not used. This slice removed already-Go-backed Rust commands and used Go tests, TS checks, source search, and generated retirement-map evidence as proof.

## Completed Backup And Config Recovery Rust Retirement Slice

Deleted another low-risk batch of old Tauri/Rust command wrappers that are already served by Go/Wails.

What changed:

- Removed old Rust `get_backup_config` and `set_backup_config` command wrappers from `src-tauri/src/backup_config.rs`.
- Kept `BackupConfig::load/save` and the remaining backup directory commands because old Rust backup scheduler and folder picker flows still depend on that module.
- Removed old Rust `restore_default_api_configs` and `check_api_config_status` command wrappers from `src-tauri/src/config_recovery.rs`.
- Deleted the now-empty `src-tauri/src/config_recovery.rs` module and removed `pub mod config_recovery` from `src-tauri/src/lib.rs`.
- Removed all four registrations from the Tauri `invoke_handler` list in `src-tauri/src/lib.rs`.
- Kept the legacy command names available through `src/runtime/wailsBridge.ts`, where they dispatch to Go `SettingsService` / `SystemService` routes as appropriate.

GitNexus / impact notes:

- Old Rust `get_backup_config`: LOW risk, 0 indexed upstream callers/processes/modules.
- Old Rust `set_backup_config`: LOW risk, 0 indexed upstream callers/processes/modules.
- Old Rust `check_api_config_status`: LOW risk, 0 indexed upstream callers/processes/modules.
- Old Rust `restore_default_api_configs`: LOW risk, 0 indexed upstream callers/processes/modules.
- A background review agent independently confirmed `backup_config.rs` should not be deleted wholesale because directory commands and auto-backup still rely on shared config helpers.

Verification:

- `rg "pub async fn (get_backup_config|set_backup_config|check_api_config_status|restore_default_api_configs)|crate::(backup_config|config_recovery)::(get_backup_config|set_backup_config|check_api_config_status|restore_default_api_configs)|commands::(get_backup_config|set_backup_config|check_api_config_status|restore_default_api_configs)" src-tauri/src src/runtime src/features src/utils tests/vitest`: pass; no old Rust definitions/registrations remain.
- `go test ./internal/settings ./internal/bindings ./internal/system -count=1`: pass.
- `go test ./... -count=1` from `desktop-go`: pass.
- `npm run test -- tests/vitest/runtime/wailsBridgeSettingsSystem.behavior.test.ts tests/vitest/settings/apiConfigRecoveryNativeFacade.source.test.ts tests/vitest/data-governance/nativeDataGovernanceFacade.source.test.ts`: pass, 4 tests.
- `npm run typecheck -- --pretty false`: pass.
- `npm run native:inventory -- --summary`: pass, 1526 scanned files / 916 native references / 823 invokes / 611 unique invokes / 65 listens / 47 unique listens / 28 emits / 24 unique emits.
- `node scripts/native-triage.mjs && node scripts/rust-retirement-map.mjs`: pass; at this checkpoint the retirement map reported 310 merged commands, 305 with Wails bridge routes, 9 merged Rust registrations, 33 merged Rust definitions, 29 retirement candidates, 0 direct Tauri blocked merged commands, 0 blocker edges, 0 blocker files, and 77 replace commands still registered in Rust. Later retirement batches reduced these counts further; use the top-level summary for the current map.
- `cargo check` was intentionally not used. This slice removed already-Go-backed Rust command wrappers and used Go tests, TS checks, source search, and generated retirement-map evidence as proof.

## Completed Statistics And WebView Save Rust Retirement Slice

Deleted two more already-Go-backed old Rust command entrypoints from `src-tauri/src/commands.rs`.

What changed:

- Removed old Rust `get_enhanced_statistics` command wrapper from `src-tauri/src/commands.rs`.
- Removed old Rust `save_webview_settings` command wrapper from `src-tauri/src/commands.rs`.
- Removed both registrations from the Tauri `invoke_handler` list in `src-tauri/src/lib.rs`.
- Kept `load_webview_settings` and `WEBVIEW_SETTINGS_FILE` in Rust because load is still Rust-only and still registered.
- `test_api_connection` was still kept in Rust at that time because Go `/responses` probing was missing; this has now been resolved and retired in the later API Connection Responses Parity + Rust Retirement slice above.

GitNexus / impact notes:

- Old Rust `get_enhanced_statistics`: LOW risk, 0 indexed upstream callers/processes/modules.
- Old Rust `save_webview_settings`: LOW risk, 0 indexed upstream callers/processes/modules.
- Background review confirmed no Rust internal callers remained for either command, and confirmed the WebView settings file constant must stay for `load_webview_settings`.

Verification:

- `rg "pub async fn (get_enhanced_statistics|save_webview_settings)|crate::commands::(get_enhanced_statistics|save_webview_settings)" src-tauri/src -g "*.rs"`: pass; no old Rust definitions/registrations remain.
- `go test ./internal/settings ./internal/system ./internal/bindings -count=1`: pass.
- `go test ./... -count=1` from `desktop-go`: pass.
- `npm run test -- tests/vitest/runtime/wailsBridgeSettingsSystem.behavior.test.ts tests/vitest/settings/nativeStatisticsFacade.source.test.ts tests/vitest/system/nativeSystemFacade.source.test.ts`: pass, 4 tests.
- `npm run typecheck -- --pretty false`: pass.
- `npm run native:inventory -- --summary`: pass, 1526 scanned files / 916 native references / 823 invokes / 611 unique invokes / 65 listens / 47 unique listens / 28 emits / 24 unique emits.
- `node scripts/native-triage.mjs && node scripts/rust-retirement-map.mjs`: pass; retirement map now reports 310 merged commands, 305 with Wails bridge routes, 7 merged Rust registrations, 31 merged Rust definitions, 27 retirement candidates, 0 direct Tauri blocked merged commands, 0 blocker edges, 0 blocker files, and 77 replace commands still registered in Rust.
- `cargo check` was intentionally not used. This slice removed already-Go-backed Rust command wrappers and used Go tests, TS checks, source search, and generated retirement-map evidence as proof.

## Completed MCP Legacy Diagnostics Cleanup Slice

Cleaned up the legacy MCP diagnostics command surface now that visible settings diagnostics use the frontend MCP SDK and Go/Wails stdio proxy.

What changed:

- Removed direct legacy MCP diagnostic wrappers from `src/utils/settingsApi.ts`: `test_mcp_connection`, `test_mcp_http`, `test_mcp_sse`, `test_mcp_websocket`, and `test_mcp_modelscope`.
- Kept settings-page MCP diagnostics routed through frontend testers: stdio via `testMcpStdioFrontend`, streamable HTTP via `testMcpHttpFrontend`, SSE via `testMcpSseFrontend`, and websocket via `testMcpWebsocketFrontend`.
- Removed old Rust/Tauri MCP diagnostic handlers and registrations for `test_mcp_connection`, `test_mcp_http`, `test_mcp_sse`, and `test_mcp_websocket`; `src-tauri/src/cmd/mcp.rs` now keeps only the remaining stdio fallback and legacy MCP config persistence.
- Deleted the unregistered old Rust RMCP diagnostic helper module `src-tauri/src/mcp/rmcp.rs` and removed `pub mod rmcp`.
- Added a small Tauri-runtime compatibility fallback in `src/runtime/native.ts` for `preheat_mcp_tools`, `get_mcp_status`, `get_mcp_tools`, and `reload_mcp_client`, so these retired backend MCP commands do not become command-not-found failures while the frontend MCP SDK remains the active runtime.
- Restored multi-step stdio diagnostics progress through `testMcpStdioFrontend(..., { onProgress })`: `spawn_process`, `connecting`, `initializing`, `listing_tools`, `disconnecting`, and `done`.
- Added `tests/vitest/settings/mcpLegacyDiagnosticsRetirement.source.test.ts` to lock the retirement contract.
- Added `tests/vitest/runtime/nativeMcpTauriFallback.behavior.test.ts` to lock the Tauri compatibility fallback for retired MCP status/tool/reload/preheat commands.

Metrics after this slice:

- Wails bindings: 12 services, 319 methods, 215 models.
- Native inventory: 1527 scanned files, 911 native references, 818 invokes, 606 unique invokes, 65 listens, 47 unique listens, 28 emits, 24 unique emits.
- Native triage: 606 unique commands; `merge` 314, `replace` 86, `defer` 180, `delete` 26.
- Rust retirement map: 314 merged commands, 309 with Wails bridge routes, 9 merged commands still registered in Rust, 33 merged Rust definitions, 29 retirement candidates, 0 direct Tauri blockers, 77 replace commands still registered in Rust.

Verification:

- `npm run test -- tests/vitest/runtime/nativeMcpTauriFallback.behavior.test.ts tests/vitest/settings/mcpLegacyDiagnosticsRetirement.source.test.ts tests/vitest/settings/mcpStdioDiagnosticsNativeFacade.source.test.ts tests/vitest/mcp/mcpStdioWailsContract.source.test.ts tests/vitest/runtime/wailsBridgeMcpStdio.behavior.test.ts tests/vitest/runtime/wailsBridgeSettingsSystem.behavior.test.ts`: pass, 16 tests.
- `go test ./... -count=1` from `desktop-go`: pass.
- `npm run typecheck -- --pretty false`: pass.
- `cargo check --manifest-path src-tauri/Cargo.toml`: pass with pre-existing warnings; used here only because this slice edited/deleted Rust command/module structure.
- `npm run native:inventory -- --summary`: pass, metrics above.
- `node scripts/native-triage.mjs && node scripts/rust-retirement-map.mjs`: pass, generated docs updated with the metrics above.
- Source gate over `src` and `src-tauri/src`: no retired legacy backend command names, `test_rmcp_streamable_http`, `mcp-test-progress`, or `rmcp.rs` product source remains; frontend tester names remain intentionally as `testMcp*Frontend`.
- Positive source gate at the time confirmed `mcp_stdio_start`, `mcp_stdio_send`, and `mcp_stdio_close` routed through `src/runtime/wailsBridge.ts`, `src/mcp/tauriStdioTransport.ts`, Go `desktop-go/internal/mcp`, and the then-present old Rust fallback. The old Rust fallback was removed in the later MCP stdio Rust retirement slice.
- Side-review regression checks: retired backend MCP status/tool/reload/preheat commands no longer call Tauri `invoke` in Tauri runtime, and stdio settings diagnostics now emit multi-step UI progress again.
- Scoped `git diff --check` for touched MCP/Go/Rust/docs/test paths: pass.

Known gaps / do not count as complete:

- Superseded for stdio by the later "Completed Live Wails MCP Stdio UI Smoke Slice": live Wails stdio spawn/send/close smoke has now passed. Remote HTTP/SSE/WebSocket settings diagnostics were still not expanded in this legacy cleanup slice.
- Full backend MCP runtime parity is still not complete; frontend MCP SDK still owns initialize, tools/list, prompts/resources, execution, and cache state.
- Superseded by the later "Completed MCP Stdio Rust Retirement Slice": old Rust `mcp_stdio_start/send/close` and `stdio_proxy` no longer remain.
- Legacy MCP config persistence/import/export paths remain separate from this diagnostics cleanup.

Next queue:

- Continue broader MCP runtime parity only where product value requires it; the old Rust stdio fallback is already retired.
- Continue low-risk Rust retirement batches from the generated map, or return to OCR/PDF/textbook and chat/provider parity.

## Completed Memory Config Rust Retirement And Map Accuracy Slice

Fixed the Rust retirement map so it reports real Tauri command registrations instead of any same-named symbol in `src-tauri/src/lib.rs`, then retired the already-Go-backed `memory_get_config` Rust command surface.

What changed:

- Updated `scripts/rust-retirement-map.mjs` to scan only `tauri::generate_handler![...]` bodies for command registrations. This prevents ordinary setup/startup calls such as `database.get_setting(...)` and `anki_database.recover_stuck_document_tasks()` from being counted as Tauri command registrations.
- Removed `crate::memory::handlers::memory_get_config` from the old Tauri `invoke_handler` list.
- Deleted the old Rust `memory_get_config` handler from `src-tauri/src/memory/handlers.rs`; the Go/Wails path remains `src/api/memoryApi.ts` -> `src/runtime/native.ts` -> `src/runtime/wailsBridge.ts` -> `SettingsService.GetMemoryConfig`.
- Added `tests/vitest/runtime/rustRetirementMap.source.test.ts` to lock both the generate-handler-only map rule and the retired `memory_get_config` contract.
- Regenerated `docs/generated/rust-retirement-map.{json,md}` and `docs/generated/native-command-triage.{json,md}`.

GitNexus / impact notes:

- `memory_get_config`: LOW risk, 0 upstream callers/processes/modules.
- `recover_stuck_document_tasks`: HIGH risk because it is still used by Tauri startup recovery in `build_app_state`; it is intentionally not deleted in this slice and is no longer misreported as a command registration.
- `get_setting` remains widely used as a Rust database method in old internals, but ordinary `database.get_setting(...)` calls are no longer misreported as command registrations.

Metrics after this slice:

- Native inventory: 1534 scanned files / 931 native references / 837 invokes / 606 unique invokes / 66 listens / 47 unique listens / 28 emits / 24 unique emits.
- Native triage: 606 unique commands; `merge` 328, `replace` 80, `defer` 172, `delete` 26.
- Rust retirement map: 328 merged commands, 323 with Wails bridge routes, 3 merged Rust registrations, 38 merged Rust definitions, 34 retirement candidates, 0 direct Tauri blocked merged commands, 0 blocker edges, 0 blocker files, and 71 replace commands still registered in Rust.

Verification:

- `npm run test -- tests/vitest/runtime/rustRetirementMap.source.test.ts tests/vitest/memory/memoryConfigNativeFacade.source.test.ts tests/vitest/runtime/wailsBridgeSettingsSystem.behavior.test.ts`: pass, 3 files / 5 tests.
- `go test ./internal/settings ./internal/bindings -count=1` from `desktop-go`: pass.
- `npm run native:inventory -- --summary`: pass, metrics above.
- `npm run native:triage`: pass, regenerated `docs/generated/native-command-triage.{json,md}`.
- `node scripts/rust-retirement-map.mjs`: pass, metrics above.
- `rg "memory_get_config" src-tauri/src src/runtime src/api tests/vitest -n`: confirms no `src-tauri/src` references remain; only Go/Wails/frontend facade and tests reference the command name.
- `git diff --check -- scripts/rust-retirement-map.mjs src-tauri/src/lib.rs src-tauri/src/memory/handlers.rs tests/vitest/runtime/rustRetirementMap.source.test.ts docs/generated/rust-retirement-map.md docs/generated/rust-retirement-map.json docs/generated/native-command-triage.md docs/generated/native-command-triage.json`: pass.
- `cargo check` was intentionally not used. This slice removes an already-Go-backed Rust command and proves the path with Go tests, Vitest source/behavior gates, generated retirement-map evidence, and source search.

Known gaps / next queue:

- Superseded by later slices: `get_model_adapter_options` and `qbank_update_sync_config` are now Go/Wails-routed and retired from old Rust. The current remaining `merge` command still registered in old Rust is `save_anki_cards`.
- `memory_get_config` no longer appears in the retirement batch list. Broader Memory-as-VFS Rust commands remain old runtime code and should not be copied into Go unless a current product workflow requires them.
- Many merged Rust definitions remain intentionally because old Rust internals still call database/helper methods. Continue deleting command wrappers first, then migrate old-data readers or quarantine helper modules when product parity is proven.

## Completed Model Adapter Options Native Facade And Rust Retirement Slice

Moved `get_model_adapter_options` off direct Tauri/Rust and onto the Go/Wails settings facade.

What changed:

- Added `ModelAdapterOption`, `defaultModelAdapterOptions`, and `GetModelAdapterOptions()` in `desktop-go/internal/settings/service.go`.
- Exposed `GetModelAdapterOptions()` through `desktop-go/internal/bindings/settings_service.go` and regenerated Wails bindings.
- Routed the legacy `get_model_adapter_options` command through `src/runtime/wailsBridge.ts` to `SettingsService.GetModelAdapterOptions`.
- Added fallback adapter options in `src/runtime/native.ts` and included the command in the fallback-before-Tauri set, so the old Tauri registration is no longer required even in transitional Tauri runtime.
- Migrated `src/features/settings/components/ShadApiEditModal.tsx` from direct `@tauri-apps/api/core` invoke to `@/runtime/native`, while preserving localized fallback labels.
- Removed the old Rust command registration from `src-tauri/src/lib.rs` and deleted the old Rust handler from `src-tauri/src/cmd/anki_cards.rs`.
- Added Go unit coverage, Wails bridge behavior coverage, and a source gate for the native facade migration.

GitNexus / impact notes:

- `get_model_adapter_options`: LOW risk, 0 upstream callers/processes/modules.
- `ShadApiEditModal`: LOW risk.
- Go `SettingsService` / settings `Service`: LOW risk for this additive settings facade method.

Metrics after this slice:

- Wails bindings: 14 services / 334 methods / 220 models.
- Native inventory: 1534 scanned files / 931 native references / 837 invokes / 606 unique invokes / 66 listens / 47 unique listens / 28 emits / 24 unique emits.
- Native triage: 606 unique commands; `merge` 328, `replace` 80, `defer` 172, `delete` 26.
- Rust retirement map: 328 merged commands, 324 with Wails bridge routes, 2 merged Rust registrations, 37 merged Rust definitions, 34 retirement candidates, 0 direct Tauri blocked merged commands, 0 blocker edges, 0 blocker files, and 71 replace commands still registered in Rust.

Verification:

- `npm run go:bindings`: pass, generated 14 services / 334 methods / 220 models.
- `go test ./internal/settings ./internal/bindings -count=1` from `desktop-go`: pass.
- `npm run test -- tests/vitest/settings/modelAdapterOptionsNativeFacade.source.test.ts tests/vitest/runtime/wailsBridgeSettingsSystem.behavior.test.ts`: pass, 2 files / 3 tests.
- `npm run native:triage`: pass, metrics above.
- `node scripts/rust-retirement-map.mjs`: pass, metrics above.
- `npm run native:inventory -- --summary`: pass, metrics above.
- `rg "get_model_adapter_options" src-tauri/src src/runtime src/features/settings/components/ShadApiEditModal.tsx docs/generated -n`: confirms no `src-tauri/src` references remain; only Go/Wails/frontend facade, generated docs, and tests reference the command name.
- `cargo check` was intentionally not used. This was a Go/Wails replacement plus old command-wrapper retirement slice; acceptance evidence is Go tests, Vitest source/behavior gates, generated retirement-map evidence, source search, and native inventory/triage.

Known gaps / next queue:

- The only remaining `merge` command still registered in old Rust is `save_anki_cards`.
- `qbank_update_sync_config` is now covered by the Completed Qbank Sync Config Native Facade And Rust Retirement Slice below.
- `save_anki_cards` needs a minimal Go/Wails AnkiConnect save-card path or reclassification as `replace` until Anki add-card/APKG parity is intentionally rebuilt.

## Completed Qbank Sync Config Native Facade And Rust Retirement Slice

Closed the last Qbank sync-config command gap by implementing a lean Go/Wails compatibility route over the current Go Qbank JSON store. This is local sync-config persistence and status compatibility; it is not full cloud sync.

What changed:

- Added `QbankService.UpdateSyncConfig(examID, config)` in `desktop-go/internal/qbank/service.go`, normalizing empty strategy and invalid intervals before persisting per-exam config to `qbank-go.json`.
- Exposed `UpdateSyncConfig` through `desktop-go/internal/bindings/qbank_service.go` and regenerated Wails bindings.
- Routed the legacy `qbank_update_sync_config` command through `src/runtime/wailsBridge.ts` to `QbankService.UpdateSyncConfig`.
- Kept ordinary browser fallback compatibility in `src/runtime/native.ts`, but removed `qbank_update_sync_config` from the Tauri-before-fallback set so old Tauri runtime cannot silently write only localStorage after the Rust command is retired.
- Tightened browser fallback validation for `qbank_set_sync_enabled` and `qbank_update_sync_config` so blank `examId`, non-boolean `enabled`, and non-object `config` fail like Go/Wails.
- Removed the old Rust `qbank_update_sync_config` registration/body from `src-tauri/src/lib.rs` and `src-tauri/src/question_sync_service.rs`.
- Added source/behavior gates for the qbank native facade, native triage classification, Rust retirement map, generated binding, and no-Tauri-before-fallback behavior.

GitNexus / impact notes:

- `qbank_update_sync_config`: LOW risk, 0 upstream callers/processes/modules.
- `UpdateSyncConfig`, `shouldUseFallbackBeforeTauri`, and `fallbackInvoke` are not indexed by the current GitNexus graph; verification uses Go tests, Vitest behavior/source gates, generated bindings, generated triage/retirement maps, and source search.

Metrics after this slice:

- Wails bindings: 14 services / 335 methods / 220 models.
- Native inventory: 1534 scanned files / 931 native references / 837 invokes / 606 unique invokes / 66 listens / 47 unique listens / 28 emits / 24 unique emits.
- Native triage: 606 unique commands; `merge` 328, `replace` 80, `defer` 172, `delete` 26.
- Rust retirement map: 328 merged commands, 325 with Wails bridge routes, 1 merged Rust registration, 36 merged Rust definitions, 34 retirement candidates, 0 direct Tauri blocked merged commands, 0 blocker edges, 0 blocker files, and 71 replace commands still registered in Rust.

Verification:

- `npm run go:bindings`: pass, generated 14 services / 335 methods / 220 models.
- `go test ./internal/qbank ./internal/bindings -count=1` from `desktop-go`: pass.
- `npm run test -- tests/vitest/runtime/qbankSyncConfigNativeFacade.source.test.ts tests/vitest/runtime/nativeTriageQbankSync.source.test.ts tests/vitest/runtime/rustRetirementMap.source.test.ts tests/vitest/runtime/nativeMcpTauriFallback.behavior.test.ts`: pass, 4 files / 8 tests.
- `npm run native:triage`: pass, regenerated `docs/generated/native-command-triage.{json,md}` with `qbank_update_sync_config` as `study-data` / `merge`.
- `node scripts/rust-retirement-map.mjs`: pass, metrics above; the only merged Rust registration left is `save_anki_cards`.
- `npm run native:inventory -- --summary`: pass, metrics above.
- `rg "qbank_update_sync_config" src-tauri/src src/runtime src/stores desktop-go tests/vitest docs/generated docs/GO_REWRITE_PROGRESS.md -n`: confirms no `src-tauri/src` references remain; only Go/Wails/frontend facade, generated docs, tests, and progress notes reference the command name.
- `cargo check` was intentionally not used. This slice is a Go/Wails replacement plus old command-wrapper retirement; acceptance evidence is Go tests, Vitest source/behavior gates, generated retirement-map evidence, source search, and native inventory/triage.

Known gaps / next queue:

- The only remaining `merge` command still registered in old Rust is `save_anki_cards`.
- This qbank slice only preserves local sync-config/status compatibility; it does not implement a real remote qbank sync engine.
- Next action is to implement or reclassify `save_anki_cards`, then continue higher-value Anki document/card parity or OCR/PDF/textbook/chat provider slices.

## Not Built Yet

High-priority missing slices:

1. Native hybrid VFS expansion.
   - Full textbook search and rich open-state flows over the same resource index.
   - Existing Tauri/Rust mindmap data migration into the Go hybrid VFS.
   - Wire attachment-root configuration into actual Go DSTU folders if product flows need the attachment root to create or select real folders instead of only storing a config ID.
   - Native PDF raster rendering, OCR pipeline replacement, and robust PDF text extraction for complex/font-encoded/scanned textbooks beyond the current lightweight text-layer parser and generated SVG text previews.
   - Real semantic resource search over text chunks, metadata, embeddings, and ranking once the lean indexing path is chosen.

2. Chat provider and streaming parity beyond the lean first slice.
   - Tool loop execution and approval continuation.
   - Non-chat usage capture, provider cost/pricing tables, and exact cost aggregation beyond the current chat-derived usage compatibility.
   - Multimodal messages, attachments, and context-resource injection.
   - Responses API and provider-specific reasoning/thinking protocols.
   - Provider health/model-list commands and encrypted API key storage.
   - Multi-variant parity.

3. Textbook, PDF, OCR, and document processing.
   - Live Wails UI and packaged-app smoke for the new Windows PDFium raster preview path, plus non-Windows PDFium loaders if cross-platform desktop preview parity is required.
   - Real OCR, robust PDF extraction for complex encodings/scanned files, OCR/text persistence into `ocrPagesJson`, and processing events beyond the current lightweight status/control/text-layer/import-progress/raster-preview compatibility.
   - Anki document generation beyond the current provider-backed text-segment worker: OCR/text extraction orchestration, robust LLM-aware segmentation, richer streaming/progress events, task persistence migration, and old Rust session/card data migration.
   - Full textbook open/search, rich open-state flows, concurrent import filtering, and import processing beyond the current local import plus legacy progress-event bridge.
   - Avoid copying old heavy Rust processing modules unless needed.

4. Data migration and import/export.
   - Read existing Tauri/Rust data.
   - Migrate settings, notes, resources, qbank, todos, chat where possible.
   - Backup/export compatibility.
   - Go Notes zip import/export exists for current Go DSTU notes and visible assets, but legacy Rust SQLite note migration, version-history export, preferences, links, mentions, mobile/content URI materialization, and import progress events are still pending.
   - Qbank CSV import/export exists in Go for local file paths, but old database migration and virtual/mobile URI materialization are still pending.

5. Frontend native facade cleanup.
   - Continue replacing direct `@tauri-apps/api/core` imports.
   - MCP legacy diagnostics are clean; remaining facade work is broader dialogs, events, windows, paths, asset URLs, and un-migrated product commands.
   - Add facade support for dialogs, events, windows, paths, and asset URLs.
   - Keep frontend UX unchanged.

6. Packaging/release.
   - NSIS.
   - Windows SDK signing/packaging path.
   - Installer smoke.

7. Rust/Tauri retirement.
   - Keep using the generated command-by-command retirement map from current `merge`/`delete` triage rows to old Rust modules.
   - Direct Tauri blockers for merged Go commands are now at 0; keep deleting low-risk Go-backed Rust command batches after verification.
   - After Go parity and migration coverage, remove or quarantine superseded `src-tauri` command modules instead of keeping them as live backend code.
   - Keep old Rust data readers only where needed for one-time migration; do not preserve old Rust service architecture as a runtime dependency.

## Next Work Queue

Recommended next turn:

1. Run `git status --short --branch` and `npx gitnexus status`.
2. Handle the remaining 1 real merged Rust registration:
   - `save_anki_cards`: design the minimal Go/Wails AnkiConnect save-card path or reclassify it as `replace` until Anki add-card/APKG parity is intentionally rebuilt.
3. Choose the next highest-value replacement slice:
   - deepen the Go Anki document worker with OCR/text extraction orchestration, robust segmentation, richer progress/task events, and old session/card migration; or
   - upgrade PDF/textbook previews from generated text-layer SVG files to native raster rendering, OCR, and richer textbook open/search state over the Go hybrid VFS; or
   - expand Go chat from minimal text-only provider streaming to parity: tools, cost/pricing metadata, multimodal/context injection, provider protocols, and multi-variant parity; or
   - extend provider-backed Qbank grading with streamed chunks, active cancellation, rubric support, provider-specific protocols, and old grading-state migration if answer evaluation becomes the next focus.
4. If staying on study resources, move next into native PDF raster rendering or OCR over the existing `previewJson`/`ocrPagesJson` metadata path, then rich textbook open/search state over the Go hybrid VFS rather than a separate textbook store. Treat the current `textbook-import-progress` bridge as UI compatibility only, not processing parity.
5. If staying on chat, add tool approval continuation, multimodal/context injection, cost/pricing metadata, and provider-specific protocol coverage over the existing Go `ChatService` store. Keep provider support lean and verify each path with provider-stream tests before adding more protocol branches.

## Do Not Rebuild Blindly

Avoid copying these Rust structures one-to-one:

- SQLite repository layers whose only purpose is old internal separation.
- All-virtual VFS blob/resource indirection.
- Debug/test commands.
- Memory-as-VFS unless a current workflow proves it is necessary.
- Multi-agent and variant pipeline experiments before core single-chat streaming works.
- Cloud sync/governance surfaces before local core workflows work.

## Exit Criteria For Completion

Only mark the goal complete when current evidence proves:

- Wails shell opens the real frontend.
- Core frontend workflows use Go services without Tauri backend dependency.
- Notes, todo/pomodoro, qbank, study resources, and chat survive restart.
- Real LLM streaming works in Go.
- Native hybrid VFS handles user-visible resource/file paths.
- Existing data has a migration or compatibility path.
- Typecheck, Go tests, command triage, inventory, and at least one runtime smoke pass.
- Packaging path is known and documented.
