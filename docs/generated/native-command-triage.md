# Native Command Triage

Generated: 2026-06-09T14:09:04.815Z
Unique commands: 606

## By Status

| Status | Count |
| --- | ---: |
| defer | 172 |
| delete | 26 |
| merge | 328 |
| replace | 80 |

## By Domain

| Domain | Count |
| --- | ---: |
| backup-governance | 63 |
| chat | 107 |
| llm | 11 |
| mcp | 4 |
| other | 112 |
| pdf-ocr | 3 |
| settings | 49 |
| study-data | 238 |
| system | 19 |

## Commands

| Domain | Status | Count | Command | Rationale |
| --- | --- | ---: | --- | --- |
| backup-governance | defer | 2 | `data_governance_get_maintenance_status` | Not part of the first lean release slice; revisit after core workflows work. |
| backup-governance | defer | 2 | `data_governance_get_migration_status` | Not part of the first lean release slice; revisit after core workflows work. |
| backup-governance | defer | 1 | `cloud_sync_delete_version` | Not part of the first lean release slice; revisit after core workflows work. |
| backup-governance | defer | 1 | `cloud_sync_download` | Not part of the first lean release slice; revisit after core workflows work. |
| backup-governance | defer | 1 | `cloud_sync_get_status` | Not part of the first lean release slice; revisit after core workflows work. |
| backup-governance | defer | 1 | `cloud_sync_list_versions` | Not part of the first lean release slice; revisit after core workflows work. |
| backup-governance | defer | 1 | `cloud_sync_upload` | Not part of the first lean release slice; revisit after core workflows work. |
| backup-governance | defer | 1 | `data_governance_backup_and_export_zip` | Not part of the first lean release slice; revisit after core workflows work. |
| backup-governance | defer | 1 | `data_governance_backup_tiered` | Not part of the first lean release slice; revisit after core workflows work. |
| backup-governance | defer | 1 | `data_governance_cancel_backup` | Not part of the first lean release slice; revisit after core workflows work. |
| backup-governance | defer | 1 | `data_governance_check_disk_space_for_restore` | Not part of the first lean release slice; revisit after core workflows work. |
| backup-governance | defer | 1 | `data_governance_cleanup_persisted_jobs` | Not part of the first lean release slice; revisit after core workflows work. |
| backup-governance | defer | 1 | `data_governance_delete_backup` | Not part of the first lean release slice; revisit after core workflows work. |
| backup-governance | defer | 1 | `data_governance_detect_conflicts` | Not part of the first lean release slice; revisit after core workflows work. |
| backup-governance | defer | 1 | `data_governance_detect_prune_gap` | Not part of the first lean release slice; revisit after core workflows work. |
| backup-governance | defer | 1 | `data_governance_export_sync_data` | Not part of the first lean release slice; revisit after core workflows work. |
| backup-governance | defer | 1 | `data_governance_export_zip` | Not part of the first lean release slice; revisit after core workflows work. |
| backup-governance | defer | 1 | `data_governance_get_asset_types` | Not part of the first lean release slice; revisit after core workflows work. |
| backup-governance | defer | 1 | `data_governance_get_backup_job` | Not part of the first lean release slice; revisit after core workflows work. |
| backup-governance | defer | 1 | `data_governance_get_backup_list` | Not part of the first lean release slice; revisit after core workflows work. |
| backup-governance | defer | 1 | `data_governance_get_database_status` | Not part of the first lean release slice; revisit after core workflows work. |
| backup-governance | defer | 1 | `data_governance_get_migration_diagnostic_report` | Not part of the first lean release slice; revisit after core workflows work. |
| backup-governance | defer | 1 | `data_governance_get_schema_registry` | Not part of the first lean release slice; revisit after core workflows work. |
| backup-governance | defer | 1 | `data_governance_get_sync_status` | Not part of the first lean release slice; revisit after core workflows work. |
| backup-governance | defer | 1 | `data_governance_import_sync_data` | Not part of the first lean release slice; revisit after core workflows work. |
| backup-governance | defer | 1 | `data_governance_import_zip` | Not part of the first lean release slice; revisit after core workflows work. |
| backup-governance | defer | 1 | `data_governance_list_backup_jobs` | Not part of the first lean release slice; revisit after core workflows work. |
| backup-governance | defer | 1 | `data_governance_list_record_conflicts` | Not part of the first lean release slice; revisit after core workflows work. |
| backup-governance | defer | 1 | `data_governance_list_resumable_jobs` | Not part of the first lean release slice; revisit after core workflows work. |
| backup-governance | defer | 1 | `data_governance_mark_asset_deleted` | Not part of the first lean release slice; revisit after core workflows work. |
| backup-governance | defer | 1 | `data_governance_mark_blob_deleted` | Not part of the first lean release slice; revisit after core workflows work. |
| backup-governance | defer | 1 | `data_governance_purge_resolved_conflicts` | Not part of the first lean release slice; revisit after core workflows work. |
| backup-governance | defer | 1 | `data_governance_resolve_conflicts` | Not part of the first lean release slice; revisit after core workflows work. |
| backup-governance | defer | 1 | `data_governance_resolve_record_conflict` | Not part of the first lean release slice; revisit after core workflows work. |
| backup-governance | defer | 1 | `data_governance_restore_backup` | Not part of the first lean release slice; revisit after core workflows work. |
| backup-governance | defer | 1 | `data_governance_restore_with_assets` | Not part of the first lean release slice; revisit after core workflows work. |
| backup-governance | defer | 1 | `data_governance_resume_backup_job` | Not part of the first lean release slice; revisit after core workflows work. |
| backup-governance | defer | 1 | `data_governance_run_backup` | Not part of the first lean release slice; revisit after core workflows work. |
| backup-governance | defer | 1 | `data_governance_run_sync` | Not part of the first lean release slice; revisit after core workflows work. |
| backup-governance | defer | 1 | `data_governance_run_sync_with_progress` | Not part of the first lean release slice; revisit after core workflows work. |
| backup-governance | defer | 1 | `data_governance_scan_assets` | Not part of the first lean release slice; revisit after core workflows work. |
| backup-governance | defer | 1 | `data_governance_verify_backup` | Not part of the first lean release slice; revisit after core workflows work. |
| backup-governance | defer | 1 | `data_governance_verify_backup_with_assets` | Not part of the first lean release slice; revisit after core workflows work. |
| backup-governance | defer | 1 | `fix_database_schema` | Not part of the first lean release slice; revisit after core workflows work. |
| backup-governance | defer | 1 | `optimize_lance_database` | Not part of the first lean release slice; revisit after core workflows work. |
| backup-governance | defer | 1 | `purge_all_database_files` | Not part of the first lean release slice; revisit after core workflows work. |
| backup-governance | delete | 2 | `clear_test_slot` | Likely debug/test/internal surface; exclude from lean release unless proven product-critical. |
| backup-governance | delete | 1 | `clear_test_slots` | Likely debug/test/internal surface; exclude from lean release unless proven product-critical. |
| backup-governance | delete | 1 | `cloud_sync_get_device_id` | Likely debug/test/internal surface; exclude from lean release unless proven product-critical. |
| backup-governance | delete | 1 | `create_test_database_in_slot` | Likely debug/test/internal surface; exclude from lean release unless proven product-critical. |
| backup-governance | delete | 1 | `create_test_files_in_slot` | Likely debug/test/internal surface; exclude from lean release unless proven product-critical. |
| backup-governance | delete | 1 | `data_governance_auto_verify_latest_backup` | Likely debug/test/internal surface; exclude from lean release unless proven product-critical. |
| backup-governance | delete | 1 | `data_governance_cleanup_audit_logs` | Likely debug/test/internal surface; exclude from lean release unless proven product-critical. |
| backup-governance | delete | 1 | `data_governance_get_audit_logs` | Likely debug/test/internal surface; exclude from lean release unless proven product-critical. |
| backup-governance | delete | 1 | `data_governance_run_health_check` | Likely debug/test/internal surface; exclude from lean release unless proven product-critical. |
| backup-governance | delete | 1 | `data_governance_run_slot_c_empty_db_test` | Likely debug/test/internal surface; exclude from lean release unless proven product-critical. |
| backup-governance | delete | 1 | `data_governance_run_slot_d_clone_db_test` | Likely debug/test/internal surface; exclude from lean release unless proven product-critical. |
| backup-governance | delete | 1 | `debug_get_database_stats` | Likely debug/test/internal surface; exclude from lean release unless proven product-critical. |
| backup-governance | delete | 1 | `debug_get_raw_mistake` | Likely debug/test/internal surface; exclude from lean release unless proven product-critical. |
| backup-governance | delete | 1 | `debug_get_raw_mistakes_batch` | Likely debug/test/internal surface; exclude from lean release unless proven product-critical. |
| backup-governance | delete | 1 | `debug_verify_mistake_integrity` | Likely debug/test/internal surface; exclude from lean release unless proven product-critical. |
| backup-governance | delete | 1 | `get_test_slot_info` | Likely debug/test/internal surface; exclude from lean release unless proven product-critical. |
| backup-governance | delete | 1 | `memory_get_audit_logs` | Likely debug/test/internal surface; exclude from lean release unless proven product-critical. |
| chat | defer | 1 | `chat_v2_cancel_variant` | Not part of the first lean release slice; revisit after core workflows work. |
| chat | defer | 1 | `chat_v2_delete_variant` | Not part of the first lean release slice; revisit after core workflows work. |
| chat | defer | 1 | `chat_v2_retry_variant` | Not part of the first lean release slice; revisit after core workflows work. |
| chat | defer | 1 | `chat_v2_retry_variants` | Not part of the first lean release slice; revisit after core workflows work. |
| chat | defer | 1 | `chat_v2_switch_variant` | Not part of the first lean release slice; revisit after core workflows work. |
| chat | defer | 1 | `export_multi_template_apkg` | Not part of the first lean release slice; revisit after core workflows work. |
| chat | delete | 2 | `chat_v2_update_block_tool_output` | Likely debug/test/internal surface; exclude from lean release unless proven product-critical. |
| chat | delete | 1 | `chat_v2_send` | Likely debug/test/internal surface; exclude from lean release unless proven product-critical. |
| chat | delete | 1 | `vfs_delete_attachment` | Likely debug/test/internal surface; exclude from lean release unless proven product-critical. |
| chat | merge | 10 | `chat_v2_update_session_settings` | Fold into a smaller chat service instead of preserving per-command shape. |
| chat | merge | 9 | `chat_v2_list_sessions` | Fold into a smaller chat service instead of preserving per-command shape. |
| chat | merge | 8 | `chat_v2_delete_session` | Fold into a smaller chat service instead of preserving per-command shape. |
| chat | merge | 7 | `vfs_get_resource` | Fold into a smaller chat service instead of preserving per-command shape. |
| chat | merge | 5 | `chat_v2_list_groups` | Fold into a smaller chat service instead of preserving per-command shape. |
| chat | merge | 5 | `chat_v2_load_session` | Fold into a smaller chat service instead of preserving per-command shape. |
| chat | merge | 5 | `chat_v2_upsert_streaming_block` | Fold into a smaller chat service instead of preserving per-command shape. |
| chat | merge | 4 | `vfs_upload_attachment` | Fold into a smaller chat service instead of preserving per-command shape. |
| chat | merge | 3 | `chat_v2_archive_session` | Fold into a smaller chat service instead of preserving per-command shape. |
| chat | merge | 3 | `chat_v2_count_sessions` | Fold into a smaller chat service instead of preserving per-command shape. |
| chat | merge | 3 | `chat_v2_update_group` | Fold into a smaller chat service instead of preserving per-command shape. |
| chat | merge | 3 | `skill_read_file` | Fold into a smaller chat service instead of preserving per-command shape. |
| chat | merge | 3 | `vfs_resolve_resource_refs` | Fold into a smaller chat service instead of preserving per-command shape. |
| chat | merge | 2 | `chat_v2_ask_user_respond` | Fold into a smaller chat service instead of preserving per-command shape. |
| chat | merge | 2 | `chat_v2_create_session` | Fold into a smaller chat service instead of preserving per-command shape. |
| chat | merge | 2 | `chat_v2_get_group` | Fold into a smaller chat service instead of preserving per-command shape. |
| chat | merge | 2 | `chat_v2_get_session` | Fold into a smaller chat service instead of preserving per-command shape. |
| chat | merge | 2 | `chat_v2_reorder_groups` | Fold into a smaller chat service instead of preserving per-command shape. |
| chat | merge | 2 | `chat_v2_send_message` | Fold into a smaller chat service instead of preserving per-command shape. |
| chat | merge | 2 | `chat_v2_tool_approval_respond` | Fold into a smaller chat service instead of preserving per-command shape. |
| chat | merge | 2 | `skill_list_directories` | Fold into a smaller chat service instead of preserving per-command shape. |
| chat | merge | 2 | `vfs_create_or_reuse` | Fold into a smaller chat service instead of preserving per-command shape. |
| chat | merge | 2 | `vfs_get_resource_path` | Fold into a smaller chat service instead of preserving per-command shape. |
| chat | merge | 1 | `chat_v2_add_tag` | Fold into a smaller chat service instead of preserving per-command shape. |
| chat | merge | 1 | `chat_v2_branch_session` | Fold into a smaller chat service instead of preserving per-command shape. |
| chat | merge | 1 | `chat_v2_cancel_stream` | Fold into a smaller chat service instead of preserving per-command shape. |
| chat | merge | 1 | `chat_v2_clear_approval_history` | Fold into a smaller chat service instead of preserving per-command shape. |
| chat | merge | 1 | `chat_v2_continue_message` | Fold into a smaller chat service instead of preserving per-command shape. |
| chat | merge | 1 | `chat_v2_create_group` | Fold into a smaller chat service instead of preserving per-command shape. |
| chat | merge | 1 | `chat_v2_delete_message` | Fold into a smaller chat service instead of preserving per-command shape. |
| chat | merge | 1 | `chat_v2_edit_and_resend` | Fold into a smaller chat service instead of preserving per-command shape. |
| chat | merge | 1 | `chat_v2_get_message_summary` | Fold into a smaller chat service instead of preserving per-command shape. |
| chat | merge | 1 | `chat_v2_list_all_tags` | Fold into a smaller chat service instead of preserving per-command shape. |
| chat | merge | 1 | `chat_v2_move_session_to_group` | Fold into a smaller chat service instead of preserving per-command shape. |
| chat | merge | 1 | `chat_v2_remove_tag` | Fold into a smaller chat service instead of preserving per-command shape. |
| chat | merge | 1 | `chat_v2_retry_message` | Fold into a smaller chat service instead of preserving per-command shape. |
| chat | merge | 1 | `chat_v2_save_session` | Fold into a smaller chat service instead of preserving per-command shape. |
| chat | merge | 1 | `chat_v2_update_block_content` | Fold into a smaller chat service instead of preserving per-command shape. |
| chat | merge | 1 | `skill_create` | Fold into a smaller chat service instead of preserving per-command shape. |
| chat | merge | 1 | `skill_delete` | Fold into a smaller chat service instead of preserving per-command shape. |
| chat | merge | 1 | `skill_update` | Fold into a smaller chat service instead of preserving per-command shape. |
| chat | merge | 1 | `vfs_decrement_ref` | Fold into a smaller chat service instead of preserving per-command shape. |
| chat | merge | 1 | `vfs_get_resource_ref_count` | Fold into a smaller chat service instead of preserving per-command shape. |
| chat | merge | 1 | `vfs_get_resource_refs` | Fold into a smaller chat service instead of preserving per-command shape. |
| chat | merge | 1 | `vfs_increment_ref` | Fold into a smaller chat service instead of preserving per-command shape. |
| chat | merge | 1 | `vfs_resource_exists` | Fold into a smaller chat service instead of preserving per-command shape. |
| chat | merge | 1 | `vfs_update_path_cache` | Fold into a smaller chat service instead of preserving per-command shape. |
| chat | merge | 1 | `vfs_update_resource_hash` | Fold into a smaller chat service instead of preserving per-command shape. |
| chat | replace | 6 | `cancel_stream` | Keep the product capability, but redesign the Go API and data flow. |
| chat | replace | 3 | `call_llm_for_boundary` | Keep the product capability, but redesign the Go API and data flow. |
| chat | replace | 2 | `delete_anki_card` | Keep the product capability, but redesign the Go API and data flow. |
| chat | replace | 2 | `import_question_bank_stream` | Keep the product capability, but redesign the Go API and data flow. |
| chat | replace | 1 | `chat_v2_anki_cards_result` | Keep the product capability, but redesign the Go API and data flow. |
| chat | replace | 1 | `chat_v2_canvas_edit_result` | Keep the product capability, but redesign the Go API and data flow. |
| chat | replace | 1 | `chat_v2_check_migration_status` | Keep the product capability, but redesign the Go API and data flow. |
| chat | replace | 1 | `chat_v2_get_anki_cards_from_block_by_document_id` | Keep the product capability, but redesign the Go API and data flow. |
| chat | replace | 1 | `chat_v2_migrate_legacy_chat` | Keep the product capability, but redesign the Go API and data flow. |
| chat | replace | 1 | `chat_v2_perform_ocr` | Keep the product capability, but redesign the Go API and data flow. |
| chat | replace | 1 | `chat_v2_rollback_migration` | Keep the product capability, but redesign the Go API and data flow. |
| chat | replace | 1 | `chat_v2_search_content` | Keep the product capability, but redesign the Go API and data flow. |
| chat | replace | 1 | `clear_message_embeddings` | Keep the product capability, but redesign the Go API and data flow. |
| chat | replace | 1 | `continue_unified_chat_stream` | Keep the product capability, but redesign the Go API and data flow. |
| chat | replace | 1 | `essay_grading_create_session` | Keep the product capability, but redesign the Go API and data flow. |
| chat | replace | 1 | `essay_grading_delete_session` | Keep the product capability, but redesign the Go API and data flow. |
| chat | replace | 1 | `essay_grading_get_session` | Keep the product capability, but redesign the Go API and data flow. |
| chat | replace | 1 | `essay_grading_list_sessions` | Keep the product capability, but redesign the Go API and data flow. |
| chat | replace | 1 | `essay_grading_stream` | Keep the product capability, but redesign the Go API and data flow. |
| chat | replace | 1 | `essay_grading_update_session` | Keep the product capability, but redesign the Go API and data flow. |
| chat | replace | 1 | `export_anki_cards` | Keep the product capability, but redesign the Go API and data flow. |
| chat | replace | 1 | `get_chat_index_stats` | Keep the product capability, but redesign the Go API and data flow. |
| chat | replace | 1 | `list_anki_library_cards` | Keep the product capability, but redesign the Go API and data flow. |
| chat | replace | 1 | `list_document_sessions` | Keep the product capability, but redesign the Go API and data flow. |
| chat | replace | 1 | `rebuild_chat_fts` | Keep the product capability, but redesign the Go API and data flow. |
| chat | replace | 1 | `translate_text_stream` | Keep the product capability, but redesign the Go API and data flow. |
| chat | replace | 1 | `unified_generate_tag_hierarchy_preview_stream` | Keep the product capability, but redesign the Go API and data flow. |
| chat | replace | 1 | `unified_get_force_graph_data` | Keep the product capability, but redesign the Go API and data flow. |
| chat | replace | 1 | `unified_import_tag_hierarchy_from_content_stream` | Keep the product capability, but redesign the Go API and data flow. |
| chat | replace | 1 | `unified_search_cards` | Keep the product capability, but redesign the Go API and data flow. |
| chat | replace | 1 | `update_anki_card` | Keep the product capability, but redesign the Go API and data flow. |
| chat | replace | 1 | `vfs_download_paper` | Keep the product capability, but redesign the Go API and data flow. |
| chat | replace | 1 | `workspace_cancel_agent` | Keep the product capability, but redesign the Go API and data flow. |
| chat | replace | 1 | `workspace_cancel_sleep` | Keep the product capability, but redesign the Go API and data flow. |
| chat | replace | 1 | `workspace_close` | Keep the product capability, but redesign the Go API and data flow. |
| chat | replace | 1 | `workspace_create` | Keep the product capability, but redesign the Go API and data flow. |
| chat | replace | 1 | `workspace_create_agent` | Keep the product capability, but redesign the Go API and data flow. |
| chat | replace | 1 | `workspace_delete` | Keep the product capability, but redesign the Go API and data flow. |
| chat | replace | 1 | `workspace_get` | Keep the product capability, but redesign the Go API and data flow. |
| chat | replace | 1 | `workspace_get_context` | Keep the product capability, but redesign the Go API and data flow. |
| chat | replace | 1 | `workspace_get_document` | Keep the product capability, but redesign the Go API and data flow. |
| chat | replace | 1 | `workspace_list_agents` | Keep the product capability, but redesign the Go API and data flow. |
| chat | replace | 1 | `workspace_list_all` | Keep the product capability, but redesign the Go API and data flow. |
| chat | replace | 1 | `workspace_list_documents` | Keep the product capability, but redesign the Go API and data flow. |
| chat | replace | 1 | `workspace_list_messages` | Keep the product capability, but redesign the Go API and data flow. |
| chat | replace | 1 | `workspace_manual_wake` | Keep the product capability, but redesign the Go API and data flow. |
| chat | replace | 1 | `workspace_restore_executions` | Keep the product capability, but redesign the Go API and data flow. |
| chat | replace | 1 | `workspace_run_agent` | Keep the product capability, but redesign the Go API and data flow. |
| chat | replace | 1 | `workspace_send_message` | Keep the product capability, but redesign the Go API and data flow. |
| chat | replace | 1 | `workspace_set_context` | Keep the product capability, but redesign the Go API and data flow. |
| llm | merge | 1 | `llm_usage_by_caller` | Fold into a smaller llm service instead of preserving per-command shape. |
| llm | merge | 1 | `llm_usage_by_model` | Fold into a smaller llm service instead of preserving per-command shape. |
| llm | merge | 1 | `llm_usage_cleanup` | Fold into a smaller llm service instead of preserving per-command shape. |
| llm | merge | 1 | `llm_usage_daily` | Fold into a smaller llm service instead of preserving per-command shape. |
| llm | merge | 1 | `llm_usage_get_trends` | Fold into a smaller llm service instead of preserving per-command shape. |
| llm | merge | 1 | `llm_usage_recent` | Fold into a smaller llm service instead of preserving per-command shape. |
| llm | merge | 1 | `llm_usage_summary` | Fold into a smaller llm service instead of preserving per-command shape. |
| llm | replace | 1 | `essay_grading_get_models` | Keep the product capability, but redesign the Go API and data flow. |
| llm | replace | 1 | `get_anki_model_names` | Keep the product capability, but redesign the Go API and data flow. |
| llm | replace | 1 | `llm_generate_answer_with_context` | Keep the product capability, but redesign the Go API and data flow. |
| llm | replace | 1 | `voice_input_transcribe` | Keep the product capability, but redesign the Go API and data flow. |
| mcp | merge | 4 | `mcp_stdio_close` | Fold into a smaller mcp service instead of preserving per-command shape. |
| mcp | merge | 3 | `mcp_stdio_send` | Fold into a smaller mcp service instead of preserving per-command shape. |
| mcp | merge | 3 | `mcp_stdio_start` | Fold into a smaller mcp service instead of preserving per-command shape. |
| mcp | merge | 1 | `preheat_mcp_tools` | Fold into a smaller mcp service instead of preserving per-command shape. |
| other | defer | 3 | `unified_create_tag` | Not part of the first lean release slice; revisit after core workflows work. |
| other | defer | 3 | `unified_outline_update_tag` | Not part of the first lean release slice; revisit after core workflows work. |
| other | defer | 2 | `dstu_unwatch` | Not part of the first lean release slice; revisit after core workflows work. |
| other | defer | 2 | `memory_get_tree` | Not part of the first lean release slice; revisit after core workflows work. |
| other | defer | 2 | `memory_list` | Not part of the first lean release slice; revisit after core workflows work. |
| other | defer | 2 | `memory_search` | Not part of the first lean release slice; revisit after core workflows work. |
| other | defer | 2 | `unified_fix_tag_hierarchy` | Not part of the first lean release slice; revisit after core workflows work. |
| other | defer | 2 | `unified_get_tags` | Not part of the first lean release slice; revisit after core workflows work. |
| other | defer | 1 | `add_cards_to_anki_connect` | Not part of the first lean release slice; revisit after core workflows work. |
| other | defer | 1 | `auto_install_package_manager` | Not part of the first lean release slice; revisit after core workflows work. |
| other | defer | 1 | `batch_export_cards` | Not part of the first lean release slice; revisit after core workflows work. |
| other | defer | 1 | `check_all_package_managers` | Not part of the first lean release slice; revisit after core workflows work. |
| other | defer | 1 | `check_anki_connect_status` | Not part of the first lean release slice; revisit after core workflows work. |
| other | defer | 1 | `check_package_manager` | Not part of the first lean release slice; revisit after core workflows work. |
| other | defer | 1 | `cloud_storage_check_connection` | Not part of the first lean release slice; revisit after core workflows work. |
| other | defer | 1 | `cloud_storage_delete` | Not part of the first lean release slice; revisit after core workflows work. |
| other | defer | 1 | `cloud_storage_exists` | Not part of the first lean release slice; revisit after core workflows work. |
| other | defer | 1 | `cloud_storage_get` | Not part of the first lean release slice; revisit after core workflows work. |
| other | defer | 1 | `cloud_storage_is_s3_enabled` | Not part of the first lean release slice; revisit after core workflows work. |
| other | defer | 1 | `cloud_storage_list` | Not part of the first lean release slice; revisit after core workflows work. |
| other | defer | 1 | `cloud_storage_put` | Not part of the first lean release slice; revisit after core workflows work. |
| other | defer | 1 | `cloud_storage_stat` | Not part of the first lean release slice; revisit after core workflows work. |
| other | defer | 1 | `delete_memory_internalization_tasks` | Not part of the first lean release slice; revisit after core workflows work. |
| other | defer | 1 | `dismiss_pending_memory_candidates` | Not part of the first lean release slice; revisit after core workflows work. |
| other | defer | 1 | `dstu_copy` | Not part of the first lean release slice; revisit after core workflows work. |
| other | defer | 1 | `dstu_export` | Not part of the first lean release slice; revisit after core workflows work. |
| other | defer | 1 | `dstu_export_formats` | Not part of the first lean release slice; revisit after core workflows work. |
| other | defer | 1 | `dstu_move` | Not part of the first lean release slice; revisit after core workflows work. |
| other | defer | 1 | `dstu_move_many` | Not part of the first lean release slice; revisit after core workflows work. |
| other | defer | 1 | `dstu_rename` | Not part of the first lean release slice; revisit after core workflows work. |
| other | defer | 1 | `dstu_search_in_folder` | Not part of the first lean release slice; revisit after core workflows work. |
| other | defer | 1 | `dstu_watch` | Not part of the first lean release slice; revisit after core workflows work. |
| other | defer | 1 | `essay_grading_create_custom_mode` | Not part of the first lean release slice; revisit after core workflows work. |
| other | defer | 1 | `essay_grading_delete_custom_mode` | Not part of the first lean release slice; revisit after core workflows work. |
| other | defer | 1 | `essay_grading_get_mode` | Not part of the first lean release slice; revisit after core workflows work. |
| other | defer | 1 | `essay_grading_get_modes` | Not part of the first lean release slice; revisit after core workflows work. |
| other | defer | 1 | `essay_grading_get_round` | Not part of the first lean release slice; revisit after core workflows work. |
| other | defer | 1 | `essay_grading_get_rounds` | Not part of the first lean release slice; revisit after core workflows work. |
| other | defer | 1 | `essay_grading_has_builtin_override` | Not part of the first lean release slice; revisit after core workflows work. |
| other | defer | 1 | `essay_grading_list_custom_modes` | Not part of the first lean release slice; revisit after core workflows work. |
| other | defer | 1 | `essay_grading_reset_builtin_mode` | Not part of the first lean release slice; revisit after core workflows work. |
| other | defer | 1 | `essay_grading_toggle_favorite` | Not part of the first lean release slice; revisit after core workflows work. |
| other | defer | 1 | `essay_grading_update_custom_mode` | Not part of the first lean release slice; revisit after core workflows work. |
| other | defer | 1 | `export_cards_as_apkg_with_template` | Not part of the first lean release slice; revisit after core workflows work. |
| other | defer | 1 | `export_knowledge_graph_data` | Not part of the first lean release slice; revisit after core workflows work. |
| other | defer | 1 | `generate_anki_cards_for_segment` | Not part of the first lean release slice; revisit after core workflows work. |
| other | defer | 1 | `get_all_tags` | Not part of the first lean release slice; revisit after core workflows work. |
| other | defer | 1 | `get_anki_deck_names` | Not part of the first lean release slice; revisit after core workflows work. |
| other | defer | 1 | `get_anki_stats` | Not part of the first lean release slice; revisit after core workflows work. |
| other | defer | 1 | `get_ann_status` | Not part of the first lean release slice; revisit after core workflows work. |
| other | defer | 1 | `get_app_version` | Not part of the first lean release slice; revisit after core workflows work. |
| other | defer | 1 | `get_data_space_info` | Not part of the first lean release slice; revisit after core workflows work. |
| other | defer | 1 | `get_detailed_tag_hierarchy` | Not part of the first lean release slice; revisit after core workflows work. |
| other | defer | 1 | `get_learning_heatmap` | Not part of the first lean release slice; revisit after core workflows work. |
| other | defer | 1 | `get_security_status` | Not part of the first lean release slice; revisit after core workflows work. |
| other | defer | 1 | `get_storage_info` | Not part of the first lean release slice; revisit after core workflows work. |
| other | defer | 1 | `get_tag_mapping_history` | Not part of the first lean release slice; revisit after core workflows work. |
| other | defer | 1 | `graph_batch_reorder_tags` | Not part of the first lean release slice; revisit after core workflows work. |
| other | defer | 1 | `graph_reorder_tag` | Not part of the first lean release slice; revisit after core workflows work. |
| other | defer | 1 | `import_knowledge_graph_data` | Not part of the first lean release slice; revisit after core workflows work. |
| other | defer | 1 | `initialize_default_tag_hierarchy` | Not part of the first lean release slice; revisit after core workflows work. |
| other | defer | 1 | `initialize_unified_irec` | Not part of the first lean release slice; revisit after core workflows work. |
| other | defer | 1 | `mark_data_space_pending_switch_to_inactive` | Not part of the first lean release slice; revisit after core workflows work. |
| other | defer | 1 | `mark_pending_memory_candidates_saved` | Not part of the first lean release slice; revisit after core workflows work. |
| other | defer | 1 | `memory_add_relation` | Not part of the first lean release slice; revisit after core workflows work. |
| other | defer | 1 | `memory_batch_delete` | Not part of the first lean release slice; revisit after core workflows work. |
| other | defer | 1 | `memory_batch_move` | Not part of the first lean release slice; revisit after core workflows work. |
| other | defer | 1 | `memory_create_root_folder` | Not part of the first lean release slice; revisit after core workflows work. |
| other | defer | 1 | `memory_delete` | Not part of the first lean release slice; revisit after core workflows work. |
| other | defer | 1 | `memory_export_all` | Not part of the first lean release slice; revisit after core workflows work. |
| other | defer | 1 | `memory_get_related` | Not part of the first lean release slice; revisit after core workflows work. |
| other | defer | 1 | `memory_get_tags` | Not part of the first lean release slice; revisit after core workflows work. |
| other | defer | 1 | `memory_move_to_folder` | Not part of the first lean release slice; revisit after core workflows work. |
| other | defer | 1 | `memory_read` | Not part of the first lean release slice; revisit after core workflows work. |
| other | defer | 1 | `memory_remove_relation` | Not part of the first lean release slice; revisit after core workflows work. |
| other | defer | 1 | `memory_set_auto_create_subfolders` | Not part of the first lean release slice; revisit after core workflows work. |
| other | defer | 1 | `memory_set_auto_extract_frequency` | Not part of the first lean release slice; revisit after core workflows work. |
| other | defer | 1 | `memory_set_default_category` | Not part of the first lean release slice; revisit after core workflows work. |
| other | defer | 1 | `memory_set_privacy_mode` | Not part of the first lean release slice; revisit after core workflows work. |
| other | defer | 1 | `memory_set_root_folder` | Not part of the first lean release slice; revisit after core workflows work. |
| other | defer | 1 | `memory_update_by_id` | Not part of the first lean release slice; revisit after core workflows work. |
| other | defer | 1 | `memory_update_tags` | Not part of the first lean release slice; revisit after core workflows work. |
| other | defer | 1 | `memory_write` | Not part of the first lean release slice; revisit after core workflows work. |
| other | defer | 1 | `memory_write_batch` | Not part of the first lean release slice; revisit after core workflows work. |
| other | defer | 1 | `memory_write_smart` | Not part of the first lean release slice; revisit after core workflows work. |
| other | defer | 1 | `plugin:clipboard-manager|write_text` | Not part of the first lean release slice; revisit after core workflows work. |
| other | defer | 1 | `purge_active_data_dir_now` | Not part of the first lean release slice; revisit after core workflows work. |
| other | defer | 1 | `restart_app` | Not part of the first lean release slice; revisit after core workflows work. |
| other | defer | 1 | `search_existing_tags` | Not part of the first lean release slice; revisit after core workflows work. |
| other | defer | 1 | `secure_delete_cloud_credentials` | Not part of the first lean release slice; revisit after core workflows work. |
| other | defer | 1 | `secure_get_cloud_credentials` | Not part of the first lean release slice; revisit after core workflows work. |
| other | defer | 1 | `secure_store_is_available` | Not part of the first lean release slice; revisit after core workflows work. |
| other | defer | 1 | `tts_check_available` | Not part of the first lean release slice; revisit after core workflows work. |
| other | defer | 1 | `tts_speak` | Not part of the first lean release slice; revisit after core workflows work. |
| other | defer | 1 | `tts_stop` | Not part of the first lean release slice; revisit after core workflows work. |
| other | defer | 1 | `unified_add_card_tag` | Not part of the first lean release slice; revisit after core workflows work. |
| other | defer | 1 | `unified_auto_generate_tag_hierarchy` | Not part of the first lean release slice; revisit after core workflows work. |
| other | defer | 1 | `unified_delete_card` | Not part of the first lean release slice; revisit after core workflows work. |
| other | defer | 1 | `unified_delete_tag` | Not part of the first lean release slice; revisit after core workflows work. |
| other | defer | 1 | `unified_export_tag_hierarchy` | Not part of the first lean release slice; revisit after core workflows work. |
| other | defer | 1 | `unified_generate_missing_tag_vectors` | Not part of the first lean release slice; revisit after core workflows work. |
| other | defer | 1 | `unified_get_card` | Not part of the first lean release slice; revisit after core workflows work. |
| other | defer | 1 | `unified_get_card_stats` | Not part of the first lean release slice; revisit after core workflows work. |
| other | defer | 1 | `unified_get_card_tags` | Not part of the first lean release slice; revisit after core workflows work. |
| other | defer | 1 | `unified_get_tag_hierarchy` | Not part of the first lean release slice; revisit after core workflows work. |
| other | defer | 1 | `unified_graph_recall_sql` | Not part of the first lean release slice; revisit after core workflows work. |
| other | defer | 1 | `unified_import_tag_hierarchy_from_content` | Not part of the first lean release slice; revisit after core workflows work. |
| other | defer | 1 | `unified_outline_move_tag` | Not part of the first lean release slice; revisit after core workflows work. |
| other | defer | 1 | `unified_remove_card_tag` | Not part of the first lean release slice; revisit after core workflows work. |
| other | defer | 1 | `unified_track_card_access` | Not part of the first lean release slice; revisit after core workflows work. |
| other | defer | 1 | `unified_update_card` | Not part of the first lean release slice; revisit after core workflows work. |
| other | defer | 1 | `update_card_content` | Not part of the first lean release slice; revisit after core workflows work. |
| pdf-ocr | defer | 1 | `memory_to_anki_document` | Not part of the first lean release slice; revisit after core workflows work. |
| pdf-ocr | defer | 1 | `ocr_extract_text` | Not part of the first lean release slice; revisit after core workflows work. |
| pdf-ocr | defer | 1 | `parse_document_from_base64` | Not part of the first lean release slice; revisit after core workflows work. |
| settings | merge | 22 | `save_setting` | Fold into a smaller settings service instead of preserving per-command shape. |
| settings | merge | 12 | `get_api_configurations` | Fold into a smaller settings service instead of preserving per-command shape. |
| settings | merge | 12 | `get_model_assignments` | Fold into a smaller settings service instead of preserving per-command shape. |
| settings | merge | 8 | `get_setting` | Fold into a smaller settings service instead of preserving per-command shape. |
| settings | merge | 4 | `get_vendor_configs` | Fold into a smaller settings service instead of preserving per-command shape. |
| settings | merge | 4 | `save_model_assignments` | Fold into a smaller settings service instead of preserving per-command shape. |
| settings | merge | 3 | `update_ocr_engine_priority` | Fold into a smaller settings service instead of preserving per-command shape. |
| settings | merge | 2 | `delete_setting` | Fold into a smaller settings service instead of preserving per-command shape. |
| settings | merge | 2 | `get_available_ocr_models` | Fold into a smaller settings service instead of preserving per-command shape. |
| settings | merge | 2 | `get_model_profiles` | Fold into a smaller settings service instead of preserving per-command shape. |
| settings | merge | 2 | `test_api_connection` | Fold into a smaller settings service instead of preserving per-command shape. |
| settings | merge | 1 | `add_ocr_engine` | Fold into a smaller settings service instead of preserving per-command shape. |
| settings | merge | 1 | `chat_v2_delete_group` | Fold into a smaller settings service instead of preserving per-command shape. |
| settings | merge | 1 | `chat_v2_restore_group` | Fold into a smaller settings service instead of preserving per-command shape. |
| settings | merge | 1 | `chat_v2_restore_session` | Fold into a smaller settings service instead of preserving per-command shape. |
| settings | merge | 1 | `check_api_config_status` | Fold into a smaller settings service instead of preserving per-command shape. |
| settings | merge | 1 | `cleanup_orphaned_images` | Fold into a smaller settings service instead of preserving per-command shape. |
| settings | merge | 1 | `get_backup_config` | Fold into a smaller settings service instead of preserving per-command shape. |
| settings | merge | 1 | `get_cn_whitelist_config` | Fold into a smaller settings service instead of preserving per-command shape. |
| settings | merge | 1 | `get_enhanced_statistics` | Fold into a smaller settings service instead of preserving per-command shape. |
| settings | merge | 1 | `get_mcp_status` | Fold into a smaller settings service instead of preserving per-command shape. |
| settings | merge | 1 | `get_model_adapter_options` | Fold into a smaller settings service instead of preserving per-command shape. |
| settings | merge | 1 | `get_ocr_engine_type` | Fold into a smaller settings service instead of preserving per-command shape. |
| settings | merge | 1 | `get_ocr_engines` | Fold into a smaller settings service instead of preserving per-command shape. |
| settings | merge | 1 | `get_ocr_thinking_enabled` | Fold into a smaller settings service instead of preserving per-command shape. |
| settings | merge | 1 | `get_provider_strategies_config` | Fold into a smaller settings service instead of preserving per-command shape. |
| settings | merge | 1 | `get_settings_by_prefix` | Fold into a smaller settings service instead of preserving per-command shape. |
| settings | merge | 1 | `get_statistics` | Fold into a smaller settings service instead of preserving per-command shape. |
| settings | merge | 1 | `memory_get_config` | Fold into a smaller settings service instead of preserving per-command shape. |
| settings | merge | 1 | `reload_mcp_client` | Fold into a smaller settings service instead of preserving per-command shape. |
| settings | merge | 1 | `remove_ocr_engine` | Fold into a smaller settings service instead of preserving per-command shape. |
| settings | merge | 1 | `restore_default_api_configs` | Fold into a smaller settings service instead of preserving per-command shape. |
| settings | merge | 1 | `save_api_configurations` | Fold into a smaller settings service instead of preserving per-command shape. |
| settings | merge | 1 | `save_available_ocr_models` | Fold into a smaller settings service instead of preserving per-command shape. |
| settings | merge | 1 | `save_image_from_base64_path` | Fold into a smaller settings service instead of preserving per-command shape. |
| settings | merge | 1 | `save_model_profiles` | Fold into a smaller settings service instead of preserving per-command shape. |
| settings | merge | 1 | `save_provider_strategies_config` | Fold into a smaller settings service instead of preserving per-command shape. |
| settings | merge | 1 | `save_settings` | Fold into a smaller settings service instead of preserving per-command shape. |
| settings | merge | 1 | `save_vendor_configs` | Fold into a smaller settings service instead of preserving per-command shape. |
| settings | merge | 1 | `save_webview_settings` | Fold into a smaller settings service instead of preserving per-command shape. |
| settings | merge | 1 | `set_backup_config` | Fold into a smaller settings service instead of preserving per-command shape. |
| settings | merge | 1 | `set_ocr_thinking_enabled` | Fold into a smaller settings service instead of preserving per-command shape. |
| settings | merge | 1 | `test_all_search_engines` | Fold into a smaller settings service instead of preserving per-command shape. |
| settings | merge | 1 | `test_ocr_engine` | Fold into a smaller settings service instead of preserving per-command shape. |
| settings | merge | 1 | `test_search_engine` | Fold into a smaller settings service instead of preserving per-command shape. |
| settings | merge | 1 | `test_web_search_connectivity` | Fold into a smaller settings service instead of preserving per-command shape. |
| settings | merge | 1 | `vfs_create_attachment_root_folder` | Fold into a smaller settings service instead of preserving per-command shape. |
| settings | merge | 1 | `vfs_get_attachment_config` | Fold into a smaller settings service instead of preserving per-command shape. |
| settings | merge | 1 | `vfs_set_attachment_root_folder` | Fold into a smaller settings service instead of preserving per-command shape. |
| study-data | delete | 1 | `vfs_debug_index_status` | Likely debug/test/internal surface; exclude from lean release unless proven product-critical. |
| study-data | merge | 8 | `get_all_custom_templates` | Fold into a smaller study-data service instead of preserving per-command shape. |
| study-data | merge | 6 | `import_builtin_templates` | Fold into a smaller study-data service instead of preserving per-command shape. |
| study-data | merge | 6 | `qbank_list_questions` | Fold into a smaller study-data service instead of preserving per-command shape. |
| study-data | merge | 5 | `notes_get_pref` | Fold into a smaller study-data service instead of preserving per-command shape. |
| study-data | merge | 4 | `delete_document_session` | Fold into a smaller study-data service instead of preserving per-command shape. |
| study-data | merge | 4 | `get_default_template_id` | Fold into a smaller study-data service instead of preserving per-command shape. |
| study-data | merge | 4 | `get_document_tasks` | Fold into a smaller study-data service instead of preserving per-command shape. |
| study-data | merge | 4 | `notes_set_pref` | Fold into a smaller study-data service instead of preserving per-command shape. |
| study-data | merge | 4 | `qbank_cancel_grading` | Fold into a smaller study-data service instead of preserving per-command shape. |
| study-data | merge | 3 | `get_document_cards` | Fold into a smaller study-data service instead of preserving per-command shape. |
| study-data | merge | 3 | `import_custom_templates_bulk` | Fold into a smaller study-data service instead of preserving per-command shape. |
| study-data | merge | 3 | `pause_document_processing` | Fold into a smaller study-data service instead of preserving per-command shape. |
| study-data | merge | 3 | `qbank_refresh_stats` | Fold into a smaller study-data service instead of preserving per-command shape. |
| study-data | merge | 3 | `qbank_toggle_favorite` | Fold into a smaller study-data service instead of preserving per-command shape. |
| study-data | merge | 3 | `qbank_update_question` | Fold into a smaller study-data service instead of preserving per-command shape. |
| study-data | merge | 3 | `resume_document_processing` | Fold into a smaller study-data service instead of preserving per-command shape. |
| study-data | merge | 3 | `trigger_task_processing` | Fold into a smaller study-data service instead of preserving per-command shape. |
| study-data | merge | 3 | `update_custom_template` | Fold into a smaller study-data service instead of preserving per-command shape. |
| study-data | merge | 3 | `vfs_get_attachment_content` | Fold into a smaller study-data service instead of preserving per-command shape. |
| study-data | merge | 2 | `create_custom_template` | Fold into a smaller study-data service instead of preserving per-command shape. |
| study-data | merge | 2 | `export_questions_csv` | Fold into a smaller study-data service instead of preserving per-command shape. |
| study-data | merge | 2 | `export_template` | Fold into a smaller study-data service instead of preserving per-command shape. |
| study-data | merge | 2 | `get_csv_preview` | Fold into a smaller study-data service instead of preserving per-command shape. |
| study-data | merge | 2 | `import_questions_csv` | Fold into a smaller study-data service instead of preserving per-command shape. |
| study-data | merge | 2 | `notes_save_asset` | Fold into a smaller study-data service instead of preserving per-command shape. |
| study-data | merge | 2 | `qbank_delete_question` | Fold into a smaller study-data service instead of preserving per-command shape. |
| study-data | merge | 2 | `qbank_get_question` | Fold into a smaller study-data service instead of preserving per-command shape. |
| study-data | merge | 2 | `qbank_get_stats` | Fold into a smaller study-data service instead of preserving per-command shape. |
| study-data | merge | 2 | `qbank_get_sync_conflicts` | Fold into a smaller study-data service instead of preserving per-command shape. |
| study-data | merge | 2 | `qbank_submit_answer` | Fold into a smaller study-data service instead of preserving per-command shape. |
| study-data | merge | 2 | `set_default_template` | Fold into a smaller study-data service instead of preserving per-command shape. |
| study-data | merge | 2 | `start_enhanced_document_processing` | Fold into a smaller study-data service instead of preserving per-command shape. |
| study-data | merge | 2 | `textbooks_add` | Fold into a smaller study-data service instead of preserving per-command shape. |
| study-data | merge | 2 | `vfs_get_file_content` | Fold into a smaller study-data service instead of preserving per-command shape. |
| study-data | merge | 2 | `vfs_get_mindmap` | Fold into a smaller study-data service instead of preserving per-command shape. |
| study-data | merge | 2 | `vfs_get_mindmap_content` | Fold into a smaller study-data service instead of preserving per-command shape. |
| study-data | merge | 1 | `canvas_note_append` | Fold into a smaller study-data service instead of preserving per-command shape. |
| study-data | merge | 1 | `canvas_note_read` | Fold into a smaller study-data service instead of preserving per-command shape. |
| study-data | merge | 1 | `canvas_note_replace` | Fold into a smaller study-data service instead of preserving per-command shape. |
| study-data | merge | 1 | `canvas_note_set` | Fold into a smaller study-data service instead of preserving per-command shape. |
| study-data | merge | 1 | `delete_custom_template` | Fold into a smaller study-data service instead of preserving per-command shape. |
| study-data | merge | 1 | `dstu_batch_move` | Fold into a smaller study-data service instead of preserving per-command shape. |
| study-data | merge | 1 | `dstu_build_path` | Fold into a smaller study-data service instead of preserving per-command shape. |
| study-data | merge | 1 | `dstu_create` | Fold into a smaller study-data service instead of preserving per-command shape. |
| study-data | merge | 1 | `dstu_delete` | Fold into a smaller study-data service instead of preserving per-command shape. |
| study-data | merge | 1 | `dstu_delete_many` | Fold into a smaller study-data service instead of preserving per-command shape. |
| study-data | merge | 1 | `dstu_empty_trash` | Fold into a smaller study-data service instead of preserving per-command shape. |
| study-data | merge | 1 | `dstu_folder_add_item` | Fold into a smaller study-data service instead of preserving per-command shape. |
| study-data | merge | 1 | `dstu_folder_create` | Fold into a smaller study-data service instead of preserving per-command shape. |
| study-data | merge | 1 | `dstu_folder_delete` | Fold into a smaller study-data service instead of preserving per-command shape. |
| study-data | merge | 1 | `dstu_folder_get` | Fold into a smaller study-data service instead of preserving per-command shape. |
| study-data | merge | 1 | `dstu_folder_get_all_resources` | Fold into a smaller study-data service instead of preserving per-command shape. |
| study-data | merge | 1 | `dstu_folder_get_breadcrumbs` | Fold into a smaller study-data service instead of preserving per-command shape. |
| study-data | merge | 1 | `dstu_folder_get_items` | Fold into a smaller study-data service instead of preserving per-command shape. |
| study-data | merge | 1 | `dstu_folder_get_tree` | Fold into a smaller study-data service instead of preserving per-command shape. |
| study-data | merge | 1 | `dstu_folder_list` | Fold into a smaller study-data service instead of preserving per-command shape. |
| study-data | merge | 1 | `dstu_folder_move` | Fold into a smaller study-data service instead of preserving per-command shape. |
| study-data | merge | 1 | `dstu_folder_move_item` | Fold into a smaller study-data service instead of preserving per-command shape. |
| study-data | merge | 1 | `dstu_folder_remove_item` | Fold into a smaller study-data service instead of preserving per-command shape. |
| study-data | merge | 1 | `dstu_folder_rename` | Fold into a smaller study-data service instead of preserving per-command shape. |
| study-data | merge | 1 | `dstu_folder_reorder` | Fold into a smaller study-data service instead of preserving per-command shape. |
| study-data | merge | 1 | `dstu_folder_reorder_items` | Fold into a smaller study-data service instead of preserving per-command shape. |
| study-data | merge | 1 | `dstu_folder_set_expanded` | Fold into a smaller study-data service instead of preserving per-command shape. |
| study-data | merge | 1 | `dstu_get` | Fold into a smaller study-data service instead of preserving per-command shape. |
| study-data | merge | 1 | `dstu_get_content` | Fold into a smaller study-data service instead of preserving per-command shape. |
| study-data | merge | 1 | `dstu_get_path_by_id` | Fold into a smaller study-data service instead of preserving per-command shape. |
| study-data | merge | 1 | `dstu_get_resource_by_path` | Fold into a smaller study-data service instead of preserving per-command shape. |
| study-data | merge | 1 | `dstu_get_resource_location` | Fold into a smaller study-data service instead of preserving per-command shape. |
| study-data | merge | 1 | `dstu_list` | Fold into a smaller study-data service instead of preserving per-command shape. |
| study-data | merge | 1 | `dstu_list_deleted` | Fold into a smaller study-data service instead of preserving per-command shape. |
| study-data | merge | 1 | `dstu_list_trash` | Fold into a smaller study-data service instead of preserving per-command shape. |
| study-data | merge | 1 | `dstu_move_to_folder` | Fold into a smaller study-data service instead of preserving per-command shape. |
| study-data | merge | 1 | `dstu_parse_path` | Fold into a smaller study-data service instead of preserving per-command shape. |
| study-data | merge | 1 | `dstu_permanently_delete` | Fold into a smaller study-data service instead of preserving per-command shape. |
| study-data | merge | 1 | `dstu_purge` | Fold into a smaller study-data service instead of preserving per-command shape. |
| study-data | merge | 1 | `dstu_purge_all` | Fold into a smaller study-data service instead of preserving per-command shape. |
| study-data | merge | 1 | `dstu_refresh_path_cache` | Fold into a smaller study-data service instead of preserving per-command shape. |
| study-data | merge | 1 | `dstu_restore` | Fold into a smaller study-data service instead of preserving per-command shape. |
| study-data | merge | 1 | `dstu_restore_many` | Fold into a smaller study-data service instead of preserving per-command shape. |
| study-data | merge | 1 | `dstu_search` | Fold into a smaller study-data service instead of preserving per-command shape. |
| study-data | merge | 1 | `dstu_set_favorite` | Fold into a smaller study-data service instead of preserving per-command shape. |
| study-data | merge | 1 | `dstu_set_metadata` | Fold into a smaller study-data service instead of preserving per-command shape. |
| study-data | merge | 1 | `dstu_soft_delete` | Fold into a smaller study-data service instead of preserving per-command shape. |
| study-data | merge | 1 | `dstu_trash_restore` | Fold into a smaller study-data service instead of preserving per-command shape. |
| study-data | merge | 1 | `dstu_update` | Fold into a smaller study-data service instead of preserving per-command shape. |
| study-data | merge | 1 | `get_document_processing_state` | Fold into a smaller study-data service instead of preserving per-command shape. |
| study-data | merge | 1 | `get_document_state` | Fold into a smaller study-data service instead of preserving per-command shape. |
| study-data | merge | 1 | `get_image_as_base64` | Fold into a smaller study-data service instead of preserving per-command shape. |
| study-data | merge | 1 | `notes_assets_bulk_delete` | Fold into a smaller study-data service instead of preserving per-command shape. |
| study-data | merge | 1 | `notes_assets_index_scan` | Fold into a smaller study-data service instead of preserving per-command shape. |
| study-data | merge | 1 | `notes_assets_scan_orphans` | Fold into a smaller study-data service instead of preserving per-command shape. |
| study-data | merge | 1 | `notes_db_stats` | Fold into a smaller study-data service instead of preserving per-command shape. |
| study-data | merge | 1 | `notes_db_vacuum` | Fold into a smaller study-data service instead of preserving per-command shape. |
| study-data | merge | 1 | `notes_delete_asset` | Fold into a smaller study-data service instead of preserving per-command shape. |
| study-data | merge | 1 | `notes_empty_trash` | Fold into a smaller study-data service instead of preserving per-command shape. |
| study-data | merge | 1 | `notes_export` | Fold into a smaller study-data service instead of preserving per-command shape. |
| study-data | merge | 1 | `notes_export_single` | Fold into a smaller study-data service instead of preserving per-command shape. |
| study-data | merge | 1 | `notes_hard_delete` | Fold into a smaller study-data service instead of preserving per-command shape. |
| study-data | merge | 1 | `notes_import` | Fold into a smaller study-data service instead of preserving per-command shape. |
| study-data | merge | 1 | `notes_import_markdown` | Fold into a smaller study-data service instead of preserving per-command shape. |
| study-data | merge | 1 | `notes_import_markdown_batch` | Fold into a smaller study-data service instead of preserving per-command shape. |
| study-data | merge | 1 | `notes_list_assets` | Fold into a smaller study-data service instead of preserving per-command shape. |
| study-data | merge | 1 | `notes_list_deleted` | Fold into a smaller study-data service instead of preserving per-command shape. |
| study-data | merge | 1 | `notes_list_tags` | Fold into a smaller study-data service instead of preserving per-command shape. |
| study-data | merge | 1 | `notes_mentions_search` | Fold into a smaller study-data service instead of preserving per-command shape. |
| study-data | merge | 1 | `notes_resolve_asset_path` | Fold into a smaller study-data service instead of preserving per-command shape. |
| study-data | merge | 1 | `notes_restore` | Fold into a smaller study-data service instead of preserving per-command shape. |
| study-data | merge | 1 | `notes_search` | Fold into a smaller study-data service instead of preserving per-command shape. |
| study-data | merge | 1 | `pomodoro_create_record` | Fold into a smaller study-data service instead of preserving per-command shape. |
| study-data | merge | 1 | `pomodoro_get_record` | Fold into a smaller study-data service instead of preserving per-command shape. |
| study-data | merge | 1 | `pomodoro_list_by_todo` | Fold into a smaller study-data service instead of preserving per-command shape. |
| study-data | merge | 1 | `pomodoro_list_today` | Fold into a smaller study-data service instead of preserving per-command shape. |
| study-data | merge | 1 | `pomodoro_today_stats` | Fold into a smaller study-data service instead of preserving per-command shape. |
| study-data | merge | 1 | `qbank_ai_grade` | Fold into a smaller study-data service instead of preserving per-command shape. |
| study-data | merge | 1 | `qbank_batch_delete_questions` | Fold into a smaller study-data service instead of preserving per-command shape. |
| study-data | merge | 1 | `qbank_batch_resolve_conflicts` | Fold into a smaller study-data service instead of preserving per-command shape. |
| study-data | merge | 1 | `qbank_generate_mock_exam` | Fold into a smaller study-data service instead of preserving per-command shape. |
| study-data | merge | 1 | `qbank_generate_paper` | Fold into a smaller study-data service instead of preserving per-command shape. |
| study-data | merge | 1 | `qbank_get_activity_heatmap` | Fold into a smaller study-data service instead of preserving per-command shape. |
| study-data | merge | 1 | `qbank_get_check_in_calendar` | Fold into a smaller study-data service instead of preserving per-command shape. |
| study-data | merge | 1 | `qbank_get_daily_practice` | Fold into a smaller study-data service instead of preserving per-command shape. |
| study-data | merge | 1 | `qbank_get_history` | Fold into a smaller study-data service instead of preserving per-command shape. |
| study-data | merge | 1 | `qbank_get_knowledge_stats_with_comparison` | Fold into a smaller study-data service instead of preserving per-command shape. |
| study-data | merge | 1 | `qbank_get_learning_trend` | Fold into a smaller study-data service instead of preserving per-command shape. |
| study-data | merge | 1 | `qbank_rebuild_fts_index` | Fold into a smaller study-data service instead of preserving per-command shape. |
| study-data | merge | 1 | `qbank_reset_progress` | Fold into a smaller study-data service instead of preserving per-command shape. |
| study-data | merge | 1 | `qbank_reset_questions_progress` | Fold into a smaller study-data service instead of preserving per-command shape. |
| study-data | merge | 1 | `qbank_resolve_sync_conflict` | Fold into a smaller study-data service instead of preserving per-command shape. |
| study-data | merge | 1 | `qbank_search_questions` | Fold into a smaller study-data service instead of preserving per-command shape. |
| study-data | merge | 1 | `qbank_set_sync_enabled` | Fold into a smaller study-data service instead of preserving per-command shape. |
| study-data | merge | 1 | `qbank_start_timed_practice` | Fold into a smaller study-data service instead of preserving per-command shape. |
| study-data | merge | 1 | `qbank_submit_mock_exam` | Fold into a smaller study-data service instead of preserving per-command shape. |
| study-data | merge | 1 | `qbank_sync_check` | Fold into a smaller study-data service instead of preserving per-command shape. |
| study-data | merge | 1 | `qbank_update_sync_config` | Fold into a smaller study-data service instead of preserving per-command shape. |
| study-data | merge | 1 | `recover_stuck_document_tasks` | Fold into a smaller study-data service instead of preserving per-command shape. |
| study-data | merge | 1 | `resource_check_sync_needed` | Fold into a smaller study-data service instead of preserving per-command shape. |
| study-data | merge | 1 | `resource_sync_exam` | Fold into a smaller study-data service instead of preserving per-command shape. |
| study-data | merge | 1 | `resource_sync_note` | Fold into a smaller study-data service instead of preserving per-command shape. |
| study-data | merge | 1 | `resource_sync_textbook_pages` | Fold into a smaller study-data service instead of preserving per-command shape. |
| study-data | merge | 1 | `review_plan_batch_create` | Fold into a smaller study-data service instead of preserving per-command shape. |
| study-data | merge | 1 | `review_plan_create` | Fold into a smaller study-data service instead of preserving per-command shape. |
| study-data | merge | 1 | `review_plan_create_for_exam` | Fold into a smaller study-data service instead of preserving per-command shape. |
| study-data | merge | 1 | `review_plan_delete` | Fold into a smaller study-data service instead of preserving per-command shape. |
| study-data | merge | 1 | `review_plan_get_by_question` | Fold into a smaller study-data service instead of preserving per-command shape. |
| study-data | merge | 1 | `review_plan_get_calendar_data` | Fold into a smaller study-data service instead of preserving per-command shape. |
| study-data | merge | 1 | `review_plan_get_due` | Fold into a smaller study-data service instead of preserving per-command shape. |
| study-data | merge | 1 | `review_plan_get_due_with_filter` | Fold into a smaller study-data service instead of preserving per-command shape. |
| study-data | merge | 1 | `review_plan_get_history` | Fold into a smaller study-data service instead of preserving per-command shape. |
| study-data | merge | 1 | `review_plan_get_or_create` | Fold into a smaller study-data service instead of preserving per-command shape. |
| study-data | merge | 1 | `review_plan_get_stats` | Fold into a smaller study-data service instead of preserving per-command shape. |
| study-data | merge | 1 | `review_plan_list_by_exam` | Fold into a smaller study-data service instead of preserving per-command shape. |
| study-data | merge | 1 | `review_plan_process` | Fold into a smaller study-data service instead of preserving per-command shape. |
| study-data | merge | 1 | `review_plan_refresh_stats` | Fold into a smaller study-data service instead of preserving per-command shape. |
| study-data | merge | 1 | `review_plan_resume` | Fold into a smaller study-data service instead of preserving per-command shape. |
| study-data | merge | 1 | `review_plan_suspend` | Fold into a smaller study-data service instead of preserving per-command shape. |
| study-data | merge | 1 | `save_anki_cards` | Fold into a smaller study-data service instead of preserving per-command shape. |
| study-data | merge | 1 | `textbooks_update_bookmarks` | Fold into a smaller study-data service instead of preserving per-command shape. |
| study-data | merge | 1 | `todo_create_item` | Fold into a smaller study-data service instead of preserving per-command shape. |
| study-data | merge | 1 | `todo_create_list` | Fold into a smaller study-data service instead of preserving per-command shape. |
| study-data | merge | 1 | `todo_delete_item` | Fold into a smaller study-data service instead of preserving per-command shape. |
| study-data | merge | 1 | `todo_delete_list` | Fold into a smaller study-data service instead of preserving per-command shape. |
| study-data | merge | 1 | `todo_ensure_inbox` | Fold into a smaller study-data service instead of preserving per-command shape. |
| study-data | merge | 1 | `todo_get_active_summary` | Fold into a smaller study-data service instead of preserving per-command shape. |
| study-data | merge | 1 | `todo_get_item` | Fold into a smaller study-data service instead of preserving per-command shape. |
| study-data | merge | 1 | `todo_get_list` | Fold into a smaller study-data service instead of preserving per-command shape. |
| study-data | merge | 1 | `todo_list_completed` | Fold into a smaller study-data service instead of preserving per-command shape. |
| study-data | merge | 1 | `todo_list_items` | Fold into a smaller study-data service instead of preserving per-command shape. |
| study-data | merge | 1 | `todo_list_lists` | Fold into a smaller study-data service instead of preserving per-command shape. |
| study-data | merge | 1 | `todo_list_overdue` | Fold into a smaller study-data service instead of preserving per-command shape. |
| study-data | merge | 1 | `todo_list_today` | Fold into a smaller study-data service instead of preserving per-command shape. |
| study-data | merge | 1 | `todo_list_upcoming` | Fold into a smaller study-data service instead of preserving per-command shape. |
| study-data | merge | 1 | `todo_reorder_items` | Fold into a smaller study-data service instead of preserving per-command shape. |
| study-data | merge | 1 | `todo_search` | Fold into a smaller study-data service instead of preserving per-command shape. |
| study-data | merge | 1 | `todo_toggle_item` | Fold into a smaller study-data service instead of preserving per-command shape. |
| study-data | merge | 1 | `todo_toggle_list_favorite` | Fold into a smaller study-data service instead of preserving per-command shape. |
| study-data | merge | 1 | `todo_update_item` | Fold into a smaller study-data service instead of preserving per-command shape. |
| study-data | merge | 1 | `todo_update_list` | Fold into a smaller study-data service instead of preserving per-command shape. |
| study-data | merge | 1 | `vfs_batch_index_pending` | Fold into a smaller study-data service instead of preserving per-command shape. |
| study-data | merge | 1 | `vfs_cancel_pdf_processing` | Fold into a smaller study-data service instead of preserving per-command shape. |
| study-data | merge | 1 | `vfs_clear_resource_ocr` | Fold into a smaller study-data service instead of preserving per-command shape. |
| study-data | merge | 1 | `vfs_create_mindmap` | Fold into a smaller study-data service instead of preserving per-command shape. |
| study-data | merge | 1 | `vfs_delete_file` | Fold into a smaller study-data service instead of preserving per-command shape. |
| study-data | merge | 1 | `vfs_delete_mindmap` | Fold into a smaller study-data service instead of preserving per-command shape. |
| study-data | merge | 1 | `vfs_delete_resource_index` | Fold into a smaller study-data service instead of preserving per-command shape. |
| study-data | merge | 1 | `vfs_get_all_index_status` | Fold into a smaller study-data service instead of preserving per-command shape. |
| study-data | merge | 1 | `vfs_get_attachment` | Fold into a smaller study-data service instead of preserving per-command shape. |
| study-data | merge | 1 | `vfs_get_batch_pdf_processing_status` | Fold into a smaller study-data service instead of preserving per-command shape. |
| study-data | merge | 1 | `vfs_get_blob_base64` | Fold into a smaller study-data service instead of preserving per-command shape. |
| study-data | merge | 1 | `vfs_get_file` | Fold into a smaller study-data service instead of preserving per-command shape. |
| study-data | merge | 1 | `vfs_get_mindmap_version` | Fold into a smaller study-data service instead of preserving per-command shape. |
| study-data | merge | 1 | `vfs_get_mindmap_version_content` | Fold into a smaller study-data service instead of preserving per-command shape. |
| study-data | merge | 1 | `vfs_get_pdf_page_image` | Fold into a smaller study-data service instead of preserving per-command shape. |
| study-data | merge | 1 | `vfs_get_pdf_processing_status` | Fold into a smaller study-data service instead of preserving per-command shape. |
| study-data | merge | 1 | `vfs_get_resource_ocr_info` | Fold into a smaller study-data service instead of preserving per-command shape. |
| study-data | merge | 1 | `vfs_get_resource_text_chunks` | Fold into a smaller study-data service instead of preserving per-command shape. |
| study-data | merge | 1 | `vfs_get_resource_units` | Fold into a smaller study-data service instead of preserving per-command shape. |
| study-data | merge | 1 | `vfs_list_dimensions` | Fold into a smaller study-data service instead of preserving per-command shape. |
| study-data | merge | 1 | `vfs_list_embedding_dims` | Fold into a smaller study-data service instead of preserving per-command shape. |
| study-data | merge | 1 | `vfs_list_files` | Fold into a smaller study-data service instead of preserving per-command shape. |
| study-data | merge | 1 | `vfs_list_mindmaps` | Fold into a smaller study-data service instead of preserving per-command shape. |
| study-data | merge | 1 | `vfs_rag_search` | Fold into a smaller study-data service instead of preserving per-command shape. |
| study-data | merge | 1 | `vfs_reindex_resource` | Fold into a smaller study-data service instead of preserving per-command shape. |
| study-data | merge | 1 | `vfs_reindex_unit` | Fold into a smaller study-data service instead of preserving per-command shape. |
| study-data | merge | 1 | `vfs_retry_pdf_processing` | Fold into a smaller study-data service instead of preserving per-command shape. |
| study-data | merge | 1 | `vfs_set_mindmap_favorite` | Fold into a smaller study-data service instead of preserving per-command shape. |
| study-data | merge | 1 | `vfs_start_pdf_processing` | Fold into a smaller study-data service instead of preserving per-command shape. |
| study-data | merge | 1 | `vfs_sync_resource_units` | Fold into a smaller study-data service instead of preserving per-command shape. |
| study-data | merge | 1 | `vfs_unified_batch_index` | Fold into a smaller study-data service instead of preserving per-command shape. |
| study-data | merge | 1 | `vfs_unified_index_status` | Fold into a smaller study-data service instead of preserving per-command shape. |
| study-data | merge | 1 | `vfs_update_mindmap` | Fold into a smaller study-data service instead of preserving per-command shape. |
| study-data | merge | 1 | `vfs_upload_file` | Fold into a smaller study-data service instead of preserving per-command shape. |
| study-data | replace | 1 | `qbank_crop_source_image` | Keep the product capability, but redesign the Go API and data flow. |
| study-data | replace | 1 | `qbank_get_source_images` | Keep the product capability, but redesign the Go API and data flow. |
| study-data | replace | 1 | `unified_generate_tag_hierarchy_preview` | Keep the product capability, but redesign the Go API and data flow. |
| study-data | replace | 1 | `update_exam_sheet_cards` | Keep the product capability, but redesign the Go API and data flow. |
| study-data | replace | 1 | `vfs_assign_dimension_model` | Keep the product capability, but redesign the Go API and data flow. |
| study-data | replace | 1 | `vfs_clear_default_embedding_dimension` | Keep the product capability, but redesign the Go API and data flow. |
| study-data | replace | 1 | `vfs_clear_media_cache` | Keep the product capability, but redesign the Go API and data flow. |
| study-data | replace | 1 | `vfs_create_dimension` | Keep the product capability, but redesign the Go API and data flow. |
| study-data | replace | 1 | `vfs_delete_dimension` | Keep the product capability, but redesign the Go API and data flow. |
| study-data | replace | 1 | `vfs_diagnose_lance_schema` | Keep the product capability, but redesign the Go API and data flow. |
| study-data | replace | 1 | `vfs_get_default_embedding_dimension` | Keep the product capability, but redesign the Go API and data flow. |
| study-data | replace | 1 | `vfs_get_dimension_range` | Keep the product capability, but redesign the Go API and data flow. |
| study-data | replace | 1 | `vfs_get_lance_stats` | Keep the product capability, but redesign the Go API and data flow. |
| study-data | replace | 1 | `vfs_get_media_cache_stats` | Keep the product capability, but redesign the Go API and data flow. |
| study-data | replace | 1 | `vfs_get_multimodal_index_capability` | Keep the product capability, but redesign the Go API and data flow. |
| study-data | replace | 1 | `vfs_get_preset_dimensions` | Keep the product capability, but redesign the Go API and data flow. |
| study-data | replace | 1 | `vfs_multimodal_delete` | Keep the product capability, but redesign the Go API and data flow. |
| study-data | replace | 1 | `vfs_multimodal_index` | Keep the product capability, but redesign the Go API and data flow. |
| study-data | replace | 1 | `vfs_multimodal_index_resource` | Keep the product capability, but redesign the Go API and data flow. |
| study-data | replace | 1 | `vfs_multimodal_search` | Keep the product capability, but redesign the Go API and data flow. |
| study-data | replace | 1 | `vfs_multimodal_stats` | Keep the product capability, but redesign the Go API and data flow. |
| study-data | replace | 1 | `vfs_optimize_lance` | Keep the product capability, but redesign the Go API and data flow. |
| study-data | replace | 1 | `vfs_reset_all_index_state` | Keep the product capability, but redesign the Go API and data flow. |
| study-data | replace | 1 | `vfs_reset_disabled_to_pending` | Keep the product capability, but redesign the Go API and data flow. |
| study-data | replace | 1 | `vfs_reset_indexed_without_embeddings` | Keep the product capability, but redesign the Go API and data flow. |
| study-data | replace | 1 | `vfs_set_default_embedding_dimension` | Keep the product capability, but redesign the Go API and data flow. |
| system | defer | 1 | `essay_grading_save_builtin_override` | Not part of the first lean release slice; revisit after core workflows work. |
| system | defer | 1 | `memory_get_profile` | Not part of the first lean release slice; revisit after core workflows work. |
| system | defer | 1 | `plugin:clipboard-manager|read_text` | Not part of the first lean release slice; revisit after core workflows work. |
| system | defer | 1 | `secure_save_cloud_credentials` | Not part of the first lean release slice; revisit after core workflows work. |
| system | defer | 1 | `unified_log_metric_event` | Not part of the first lean release slice; revisit after core workflows work. |
| system | delete | 1 | `create_edge_case_test_files` | Likely debug/test/internal surface; exclude from lean release unless proven product-critical. |
| system | delete | 1 | `create_symlink_test` | Likely debug/test/internal surface; exclude from lean release unless proven product-critical. |
| system | delete | 1 | `essay_grading_get_latest_round_number` | Likely debug/test/internal surface; exclude from lean release unless proven product-critical. |
| system | delete | 1 | `graph_recall_test` | Likely debug/test/internal surface; exclude from lean release unless proven product-critical. |
| system | delete | 1 | `write_debug_logs` | Likely debug/test/internal surface; exclude from lean release unless proven product-critical. |
| system | merge | 2 | `report_frontend_log` | Fold into a smaller system service instead of preserving per-command shape. |
| system | merge | 1 | `copy_file` | Fold into a smaller system service instead of preserving per-command shape. |
| system | merge | 1 | `ensure_debug_log_dir` | Fold into a smaller system service instead of preserving per-command shape. |
| system | merge | 1 | `get_app_data_dir` | Fold into a smaller system service instead of preserving per-command shape. |
| system | merge | 1 | `get_file_size` | Fold into a smaller system service instead of preserving per-command shape. |
| system | merge | 1 | `open_logs_folder` | Fold into a smaller system service instead of preserving per-command shape. |
| system | merge | 1 | `read_file_bytes` | Fold into a smaller system service instead of preserving per-command shape. |
| system | merge | 1 | `read_file_text` | Fold into a smaller system service instead of preserving per-command shape. |
| system | merge | 1 | `save_text_to_file` | Fold into a smaller system service instead of preserving per-command shape. |
