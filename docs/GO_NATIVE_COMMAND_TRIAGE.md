# Go Native Command Triage

Purpose: prevent the Go rewrite from becoming a line-by-line clone of the Tauri backend.

Every native command or event should be classified before implementation.

## Status Labels

| Status | Meaning |
| --- | --- |
| keep | Required by the simplified product and should be implemented in Go. |
| merge | Useful behavior, but should be combined into a smaller Go service API. |
| replace | Needed outcome, but old command shape or implementation should be replaced. |
| defer | Potentially useful, but not needed for the first lean release. |
| delete | Debug-only, stale, duplicate, or no longer product-relevant. |

## Initial Domain Priorities

| Priority | Domain | Default Direction |
| --- | --- | --- |
| 1 | app paths, settings, logs | keep/merge |
| 1 | core notes/resources/study data | keep/replace |
| 1 | chat sessions and basic LLM streaming | keep/replace |
| 2 | textbooks, question bank, review plans | keep/replace |
| 2 | import/export and backup | keep/merge |
| 3 | OCR/PDF processing | replace/defer until workflow is clear |
| 3 | MCP | defer unless required by target workflow |
| 4 | debug plugins, test commands, internal audits | delete/defer |
| 4 | legacy multi-variant/experimental branches | delete/defer |

## Inventory Workflow

1. Run `npm run native:inventory -- --json`.
2. Group commands by feature area and calling files.
3. Mark each command with a status label.
4. For `merge` and `replace`, define the new Go service method before writing code.
5. Implement only commands that are `keep`, `merge`, or `replace` for the active release slice.

## Current Go Merge Surface

These frontend command families are currently folded into smaller Go services:

| Go service | Frontend command family |
| --- | --- |
| `SettingsService` | `get_setting`, `get_settings`, `save_setting`, `save_settings`, `delete_setting`, attachment root config compatibility commands |
| `SystemService` | app data/log/debug/frontend log commands |
| `FileService` | local file read/size/copy helpers |
| `NotesService` | Notes preferences and note image asset helpers |
| `TodoService` | `todo_*` plus `pomodoro_*` study workflow commands |
| `DstuService` | Notes-facing `dstu_list`, `dstu_get`, `dstu_create`, `dstu_update`, `dstu_delete`, `dstu_search`, `dstu_get_content`, `dstu_set_metadata`, `dstu_set_favorite`, `notes_import_markdown`, `notes_import_markdown_batch` |
| `QbankService` | Core practice `qbank_list_questions`, `qbank_search_questions`, `qbank_rebuild_fts_index`, `qbank_get_question`, `qbank_create_question`, `qbank_update_question`, `qbank_delete_question`, `qbank_batch_delete_questions`, `qbank_submit_answer`, `qbank_toggle_favorite`, `qbank_get_stats`, `qbank_refresh_stats`, `qbank_reset_progress`, `qbank_reset_questions_progress` |
| `ChatService` | Chat shell/session/tag/stats/interaction/message/block `chat_v2_create_session`, `chat_v2_get_session`, `chat_v2_load_session`, `chat_v2_save_session`, `chat_v2_update_session_settings`, `chat_v2_archive_session`, `chat_v2_delete_session`, `chat_v2_move_session_to_group`, `chat_v2_list_sessions`, `chat_v2_count_sessions`, `chat_v2_branch_session`, `chat_v2_create_group`, `chat_v2_get_group`, `chat_v2_update_group`, `chat_v2_list_groups`, `chat_v2_reorder_groups`, `chat_v2_add_tag`, `chat_v2_remove_tag`, `chat_v2_list_all_tags`, `chat_v2_get_session_tags`, `chat_v2_get_tags_batch`, `chat_v2_get_message_summary`, `chat_v2_send_message`, `chat_v2_continue_message`, `chat_v2_cancel_stream`, `chat_v2_retry_message`, `chat_v2_edit_and_resend`, `chat_v2_tool_approval_respond`, `chat_v2_ask_user_respond`, `chat_v2_delete_message`, `chat_v2_update_block_content`, `chat_v2_upsert_streaming_block` |
| `VfsService` | Native hybrid VFS resource-index/context-ref/attachment/file/compact-index/local-search commands `vfs_create_or_reuse`, `vfs_get_resource`, `vfs_resource_exists`, `vfs_increment_ref`, `vfs_decrement_ref`, `vfs_get_resource_path`, `vfs_get_resource_ref_count`, `vfs_update_resource_hash`, `vfs_get_resource_refs`, `vfs_resolve_resource_refs`, `vfs_update_path_cache`, `vfs_upload_attachment`, `vfs_get_attachment`, `vfs_get_attachment_content`, `vfs_upload_file`, `vfs_get_file`, `vfs_delete_file`, `vfs_get_file_content`, `textbooks_update_bookmarks`, `vfs_unified_index_status`, `vfs_get_resource_units`, `vfs_sync_resource_units`, `vfs_get_all_index_status`, `vfs_reindex_resource`, `vfs_reindex_unit`, `vfs_unified_batch_index`, `vfs_batch_index_pending`, `vfs_delete_resource_index`, `vfs_list_embedding_dims`, `vfs_list_dimensions`, `vfs_get_resource_text_chunks`, `vfs_get_resource_ocr_info`, `vfs_clear_resource_ocr`, `vfs_rag_search`, `vfs_list_files` |

## Review Questions

- Does a normal user need this workflow?
- Does this command represent a product capability or only an implementation detail?
- Can multiple commands become one simpler service method?
- Is this behavior still needed if the data model is simplified?
- Can user data be migrated without preserving this internal abstraction?
