# Rust Retirement Map

Generated: 2026-06-09T14:14:42.642Z

## Summary

| Metric | Count |
| --- | ---: |
| uniqueCommands | 606 |
| byStatus | defer: 172, delete: 26, merge: 328, replace: 80 |
| mergeCommands | 328 |
| mergeWithGoBridge | 326 |
| mergeRustRegistered | 0 |
| mergeRustDefined | 35 |
| retirementCandidates | 34 |
| directTauriBlockedMerged | 0 |
| directTauriBlockedEdges | 0 |
| directTauriBlockedFiles | 0 |
| replaceRustRegistered | 71 |

## Retirement Batches

| Rust file | Merged Go commands still present in Rust | Commands |
| --- | ---: | --- |
| `src-tauri/src/database/mod.rs` | 12 | `create_custom_template`, `delete_custom_template`, `delete_document_session`, `delete_setting`, `get_all_custom_templates`, `get_model_assignments`, `get_setting`, `recover_stuck_document_tasks`, `save_model_assignments`, `save_setting`, `set_default_template`, `update_custom_template` |
| `src-tauri/src/commands.rs` | 9 | `create_custom_template`, `delete_custom_template`, `export_template`, `get_all_custom_templates`, `get_default_template_id`, `import_builtin_templates`, `import_custom_templates_bulk`, `set_default_template`, `update_custom_template` |
| `src-tauri/src/llm_manager/mod.rs` | 9 | `get_available_ocr_models`, `get_model_assignments`, `get_model_profiles`, `get_ocr_engine_type`, `get_vendor_configs`, `save_api_configurations`, `save_model_assignments`, `save_model_profiles`, `save_vendor_configs` |
| `src-tauri/src/enhanced_anki_service.rs` | 6 | `delete_document_session`, `get_document_state`, `get_document_tasks`, `pause_document_processing`, `resume_document_processing`, `trigger_task_processing` |
| `src-tauri/src/file_manager.rs` | 2 | `get_app_data_dir`, `get_image_as_base64` |
| `src-tauri/src/unified_file_manager.rs` | 2 | `copy_file`, `get_file_size` |
| `src-tauri/src/chat_v2/database.rs` | 1 | `get_statistics` |
| `src-tauri/src/database.debug.rs` | 1 | `update_custom_template` |
| `src-tauri/src/debug_log_service.rs` | 1 | `ensure_debug_log_dir` |
| `src-tauri/src/document_processing_service.rs` | 1 | `get_document_tasks` |
| `src-tauri/src/llm_usage/database.rs` | 1 | `get_statistics` |
| `src-tauri/src/vfs/database.rs` | 1 | `get_statistics` |

## Direct Tauri Call Blockers

Merged commands in this table already have Go/Wails routing, but at least one frontend file still calls the command through direct `@tauri-apps/api/core` imports. Move these callers to `src/runtime/native.ts` before deleting the matching Rust command batch.

| Command | Domain | Direct frontend files |
| --- | --- | --- |

## Replace Commands Still Registered In Rust

| Command | Domain | Rust definitions |
| --- | --- | --- |
| `call_llm_for_boundary` | chat | src-tauri/src/commands.rs:1527 |
| `cancel_stream` | chat | src-tauri/src/chat_v2/state.rs:154, src-tauri/src/commands.rs:307 |
| `chat_v2_anki_cards_result` | chat | src-tauri/src/chat_v2/handlers/block_actions.rs:412 |
| `chat_v2_canvas_edit_result` | chat | src-tauri/src/chat_v2/handlers/canvas_handlers.rs:18 |
| `chat_v2_check_migration_status` | chat | src-tauri/src/chat_v2/handlers/migration.rs:18 |
| `chat_v2_get_anki_cards_from_block_by_document_id` | chat | src-tauri/src/chat_v2/handlers/block_actions.rs:176 |
| `chat_v2_migrate_legacy_chat` | chat | src-tauri/src/chat_v2/handlers/migration.rs:33 |
| `chat_v2_perform_ocr` | chat | src-tauri/src/chat_v2/handlers/ocr.rs:43 |
| `chat_v2_rollback_migration` | chat | src-tauri/src/chat_v2/handlers/migration.rs:48 |
| `chat_v2_search_content` | chat | src-tauri/src/chat_v2/handlers/search_handlers.rs:12 |
| `clear_message_embeddings` | chat | src-tauri/src/commands.rs:1084 |
| `delete_anki_card` | chat | src-tauri/src/cmd/enhanced_anki.rs:63, src-tauri/src/database/mod.rs:4244, src-tauri/src/enhanced_anki_service.rs:602 |
| `essay_grading_create_session` | chat | src-tauri/src/essay_grading/mod.rs:80 |
| `essay_grading_delete_session` | chat | src-tauri/src/essay_grading/mod.rs:150 |
| `essay_grading_get_session` | chat | src-tauri/src/essay_grading/mod.rs:108 |
| `essay_grading_list_sessions` | chat | src-tauri/src/essay_grading/mod.rs:170 |
| `essay_grading_stream` | chat | src-tauri/src/essay_grading/mod.rs:31 |
| `essay_grading_update_session` | chat | src-tauri/src/essay_grading/mod.rs:125 |
| `export_anki_cards` | chat | src-tauri/src/cmd/enhanced_anki.rs:222 |
| `import_question_bank_stream` | chat | src-tauri/src/commands.rs:951 |
| `list_anki_library_cards` | chat | src-tauri/src/cmd/enhanced_anki.rs:133, src-tauri/src/database/mod.rs:5639 |
| `list_document_sessions` | chat | src-tauri/src/cmd/enhanced_anki.rs:160, src-tauri/src/database/mod.rs:5523 |
| `translate_text_stream` | chat | src-tauri/src/translation/mod.rs:34 |
| `update_anki_card` | chat | src-tauri/src/cmd/enhanced_anki.rs:37, src-tauri/src/database/mod.rs:4217, src-tauri/src/enhanced_anki_service.rs:595 |
| `vfs_download_paper` | chat | src-tauri/src/vfs/handlers.rs:3581 |
| `workspace_cancel_agent` | chat | src-tauri/src/chat_v2/handlers/workspace_handlers.rs:1052 |
| `workspace_cancel_sleep` | chat | src-tauri/src/chat_v2/handlers/workspace_handlers.rs:1139 |
| `workspace_close` | chat | src-tauri/src/chat_v2/handlers/workspace_handlers.rs:190 |
| `workspace_create` | chat | src-tauri/src/chat_v2/handlers/workspace_handlers.rs:154 |
| `workspace_create_agent` | chat | src-tauri/src/chat_v2/handlers/workspace_handlers.rs:212 |
| `workspace_delete` | chat | src-tauri/src/chat_v2/handlers/workspace_handlers.rs:201 |
| `workspace_get` | chat | src-tauri/src/chat_v2/handlers/workspace_handlers.rs:170 |
| `workspace_get_context` | chat | src-tauri/src/chat_v2/handlers/workspace_handlers.rs:405 |
| `workspace_get_document` | chat | src-tauri/src/chat_v2/handlers/workspace_handlers.rs:441 |
| `workspace_list_agents` | chat | src-tauri/src/chat_v2/handlers/workspace_handlers.rs:312 |
| `workspace_list_all` | chat | src-tauri/src/chat_v2/handlers/workspace_handlers.rs:454 |
| `workspace_list_documents` | chat | src-tauri/src/chat_v2/handlers/workspace_handlers.rs:418 |
| `workspace_list_messages` | chat | src-tauri/src/chat_v2/handlers/workspace_handlers.rs:368 |
| `workspace_manual_wake` | chat | src-tauri/src/chat_v2/handlers/workspace_handlers.rs:1105 |
| `workspace_restore_executions` | chat | src-tauri/src/chat_v2/handlers/workspace_handlers.rs:1187 |
| `workspace_run_agent` | chat | src-tauri/src/chat_v2/handlers/workspace_handlers.rs:510 |
| `workspace_send_message` | chat | src-tauri/src/chat_v2/handlers/workspace_handlers.rs:335 |
| `workspace_set_context` | chat | src-tauri/src/chat_v2/handlers/workspace_handlers.rs:393 |
| `essay_grading_get_models` | llm | src-tauri/src/essay_grading/mod.rs:553 |
| `get_anki_model_names` | llm | src-tauri/src/cmd/anki_connect.rs:123 |
| `voice_input_transcribe` | llm | src-tauri/src/voice_input.rs:296 |
| `qbank_crop_source_image` | study-data | src-tauri/src/commands.rs:4336 |
| `qbank_get_source_images` | study-data | src-tauri/src/commands.rs:4249 |
| `update_exam_sheet_cards` | study-data | src-tauri/src/commands.rs:819, src-tauri/src/exam_sheet_service.rs:217 |
| `vfs_assign_dimension_model` | study-data | src-tauri/src/vfs/handlers.rs:1180 |
| `vfs_clear_default_embedding_dimension` | study-data | src-tauri/src/vfs/handlers.rs:1537 |
| `vfs_clear_media_cache` | study-data | src-tauri/src/vfs/handlers.rs:3216 |
| `vfs_create_dimension` | study-data | src-tauri/src/vfs/handlers.rs:1246 |
| `vfs_delete_dimension` | study-data | src-tauri/src/vfs/handlers.rs:1280 |
| `vfs_diagnose_lance_schema` | study-data | src-tauri/src/vfs/handlers.rs:2893 |
| `vfs_get_default_embedding_dimension` | study-data | src-tauri/src/vfs/handlers.rs:1462 |
| `vfs_get_dimension_range` | study-data | src-tauri/src/vfs/handlers.rs:1374 |
| `vfs_get_lance_stats` | study-data | src-tauri/src/vfs/handlers.rs:1889 |
| `vfs_get_media_cache_stats` | study-data | src-tauri/src/vfs/handlers.rs:3054 |
| `vfs_get_multimodal_index_capability` | study-data | src-tauri/src/vfs/handlers.rs:1649 |
| `vfs_get_preset_dimensions` | study-data | src-tauri/src/vfs/handlers.rs:1369 |
| `vfs_multimodal_delete` | study-data | src-tauri/src/vfs/handlers.rs:2818 |
| `vfs_multimodal_index` | study-data | src-tauri/src/vfs/handlers.rs:2645 |
| `vfs_multimodal_index_resource` | study-data | src-tauri/src/vfs/handlers.rs:2842 |
| `vfs_multimodal_search` | study-data | src-tauri/src/vfs/handlers.rs:2754 |
| `vfs_multimodal_stats` | study-data | src-tauri/src/vfs/handlers.rs:2795 |
| `vfs_optimize_lance` | study-data | src-tauri/src/vfs/handlers.rs:1908 |
| `vfs_reset_all_index_state` | study-data | src-tauri/src/vfs/handlers.rs:2489 |
| `vfs_reset_disabled_to_pending` | study-data | src-tauri/src/vfs/handlers.rs:2431 |
| `vfs_reset_indexed_without_embeddings` | study-data | src-tauri/src/vfs/handlers.rs:2453 |
| `vfs_set_default_embedding_dimension` | study-data | src-tauri/src/vfs/handlers.rs:1392 |

## Notes

- `retirementCandidate` means the command is marked `merge`, has a Wails bridge route, and still has a Rust registration or definition.
- `directTauriBlocked` means deleting the Rust command could still break a frontend caller that bypasses the native facade.
- This is a generated snapshot. Re-run `node scripts/rust-retirement-map.mjs` after triage or bridge changes.
