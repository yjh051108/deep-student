import { promises as fs } from 'node:fs';
import path from 'node:path';
import process from 'node:process';

const root = process.cwd();
const srcDir = path.join(root, 'src');
const outDir = path.join(root, 'docs', 'generated');
const exts = new Set(['.ts', '.tsx', '.js', '.jsx']);

const domainRules = [
  ['settings', /setting|config|theme|appearance|font|palette|agreement|consent/i],
  ['chat', /^chat_|chat|session|message|stream|variant|block|group/i],
  ['study-data', /vfs|textbook|qbank|question|review|spaced|todo|pomodoro|note|resource|attachment/i],
  ['llm', /model|provider|vendor|api_config|token|llm|ocr_model/i],
  ['mcp', /^mcp_|mcp|preheat_mcp/i],
  ['pdf-ocr', /pdf|ocr|crop|image|document|exam|raster/i],
  ['backup-governance', /backup|governance|archive|migration|sync|slot|database|audit/i],
  ['system', /file|path|log|window|debug|test|health|dependency|open_|read_|save_|copy_|hash_/i],
];

const implementedCommandOverrides = new Map([
  ['chat_v2_create_session', { domain: 'chat', status: 'merge' }],
  ['chat_v2_get_session', { domain: 'chat', status: 'merge' }],
  ['chat_v2_load_session', { domain: 'chat', status: 'merge' }],
  ['chat_v2_save_session', { domain: 'chat', status: 'merge' }],
  ['chat_v2_update_session_settings', { domain: 'chat', status: 'merge' }],
  ['chat_v2_archive_session', { domain: 'chat', status: 'merge' }],
  ['chat_v2_delete_session', { domain: 'chat', status: 'merge' }],
  ['chat_v2_move_session_to_group', { domain: 'chat', status: 'merge' }],
  ['chat_v2_list_sessions', { domain: 'chat', status: 'merge' }],
  ['chat_v2_count_sessions', { domain: 'chat', status: 'merge' }],
  ['chat_v2_create_group', { domain: 'chat', status: 'merge' }],
  ['chat_v2_get_group', { domain: 'chat', status: 'merge' }],
  ['chat_v2_update_group', { domain: 'chat', status: 'merge' }],
  ['chat_v2_list_groups', { domain: 'chat', status: 'merge' }],
  ['chat_v2_reorder_groups', { domain: 'chat', status: 'merge' }],
  ['chat_v2_add_tag', { domain: 'chat', status: 'merge' }],
  ['chat_v2_remove_tag', { domain: 'chat', status: 'merge' }],
  ['chat_v2_list_all_tags', { domain: 'chat', status: 'merge' }],
  ['chat_v2_get_message_summary', { domain: 'chat', status: 'merge' }],
  ['chat_v2_branch_session', { domain: 'chat', status: 'merge' }],
  ['chat_v2_send_message', { domain: 'chat', status: 'merge' }],
  ['chat_v2_continue_message', { domain: 'chat', status: 'merge' }],
  ['chat_v2_cancel_stream', { domain: 'chat', status: 'merge' }],
  ['chat_v2_retry_message', { domain: 'chat', status: 'merge' }],
  ['chat_v2_edit_and_resend', { domain: 'chat', status: 'merge' }],
  ['chat_v2_tool_approval_respond', { domain: 'chat', status: 'merge' }],
  ['chat_v2_clear_approval_history', { domain: 'chat', status: 'merge' }],
  ['chat_v2_ask_user_respond', { domain: 'chat', status: 'merge' }],
  ['chat_v2_delete_message', { domain: 'chat', status: 'merge' }],
  ['chat_v2_update_block_content', { domain: 'chat', status: 'merge' }],
  ['chat_v2_upsert_streaming_block', { domain: 'chat', status: 'merge' }],
  ['skill_list_directories', { domain: 'chat', status: 'merge' }],
  ['skill_read_file', { domain: 'chat', status: 'merge' }],
  ['skill_create', { domain: 'chat', status: 'merge' }],
  ['skill_update', { domain: 'chat', status: 'merge' }],
  ['skill_delete', { domain: 'chat', status: 'merge' }],
  ['vfs_create_or_reuse', { domain: 'chat', status: 'merge' }],
  ['vfs_get_resource', { domain: 'chat', status: 'merge' }],
  ['vfs_resource_exists', { domain: 'chat', status: 'merge' }],
  ['vfs_increment_ref', { domain: 'chat', status: 'merge' }],
  ['vfs_decrement_ref', { domain: 'chat', status: 'merge' }],
  ['vfs_get_resource_path', { domain: 'chat', status: 'merge' }],
  ['vfs_get_resource_ref_count', { domain: 'chat', status: 'merge' }],
  ['vfs_update_resource_hash', { domain: 'chat', status: 'merge' }],
  ['vfs_get_resource_refs', { domain: 'chat', status: 'merge' }],
  ['vfs_resolve_resource_refs', { domain: 'chat', status: 'merge' }],
  ['vfs_update_path_cache', { domain: 'chat', status: 'merge' }],
  ['resource_sync_note', { domain: 'study-data', status: 'merge' }],
  ['resource_sync_exam', { domain: 'study-data', status: 'merge' }],
  ['resource_sync_textbook_pages', { domain: 'study-data', status: 'merge' }],
  ['resource_check_sync_needed', { domain: 'study-data', status: 'merge' }],
  ['vfs_get_pdf_processing_status', { domain: 'study-data', status: 'merge' }],
  ['vfs_get_batch_pdf_processing_status', { domain: 'study-data', status: 'merge' }],
  ['vfs_cancel_pdf_processing', { domain: 'study-data', status: 'merge' }],
  ['vfs_retry_pdf_processing', { domain: 'study-data', status: 'merge' }],
  ['vfs_start_pdf_processing', { domain: 'study-data', status: 'merge' }],
  ['vfs_get_pdf_page_image', { domain: 'study-data', status: 'merge' }],
  ['vfs_get_blob_base64', { domain: 'study-data', status: 'merge' }],
  ['start_enhanced_document_processing', { domain: 'study-data', status: 'merge' }],
  ['get_document_tasks', { domain: 'study-data', status: 'merge' }],
  ['pause_document_processing', { domain: 'study-data', status: 'merge' }],
  ['resume_document_processing', { domain: 'study-data', status: 'merge' }],
  ['get_document_processing_state', { domain: 'study-data', status: 'merge' }],
  ['get_document_state', { domain: 'study-data', status: 'merge' }],
  ['get_document_task_counts', { domain: 'study-data', status: 'merge' }],
  ['trigger_task_processing', { domain: 'study-data', status: 'merge' }],
  ['delete_document_session', { domain: 'study-data', status: 'merge' }],
  ['get_document_cards', { domain: 'study-data', status: 'merge' }],
  ['recover_stuck_document_tasks', { domain: 'study-data', status: 'merge' }],
  ['save_anki_cards', { domain: 'study-data', status: 'merge' }],
  ['import_builtin_templates', { domain: 'study-data', status: 'merge' }],
  ['get_all_custom_templates', { domain: 'study-data', status: 'merge' }],
  ['get_default_template_id', { domain: 'study-data', status: 'merge' }],
  ['create_custom_template', { domain: 'study-data', status: 'merge' }],
  ['update_custom_template', { domain: 'study-data', status: 'merge' }],
  ['delete_custom_template', { domain: 'study-data', status: 'merge' }],
  ['set_default_template', { domain: 'study-data', status: 'merge' }],
  ['import_custom_templates_bulk', { domain: 'study-data', status: 'merge' }],
  ['export_template', { domain: 'study-data', status: 'merge' }],
  ['vfs_upload_attachment', { domain: 'chat', status: 'merge' }],
  ['vfs_get_attachment', { domain: 'study-data', status: 'merge' }],
  ['vfs_get_attachment_content', { domain: 'study-data', status: 'merge' }],
  ['vfs_upload_file', { domain: 'study-data', status: 'merge' }],
  ['vfs_get_file', { domain: 'study-data', status: 'merge' }],
  ['vfs_delete_file', { domain: 'study-data', status: 'merge' }],
  ['vfs_get_file_content', { domain: 'study-data', status: 'merge' }],
  ['textbooks_update_bookmarks', { domain: 'study-data', status: 'merge' }],
  ['textbooks_add', { domain: 'study-data', status: 'merge' }],
  ['vfs_unified_index_status', { domain: 'study-data', status: 'merge' }],
  ['vfs_get_resource_units', { domain: 'study-data', status: 'merge' }],
  ['vfs_sync_resource_units', { domain: 'study-data', status: 'merge' }],
  ['vfs_get_all_index_status', { domain: 'study-data', status: 'merge' }],
  ['vfs_reindex_resource', { domain: 'study-data', status: 'merge' }],
  ['vfs_reindex_unit', { domain: 'study-data', status: 'merge' }],
  ['vfs_unified_batch_index', { domain: 'study-data', status: 'merge' }],
  ['vfs_batch_index_pending', { domain: 'study-data', status: 'merge' }],
  ['vfs_delete_resource_index', { domain: 'study-data', status: 'merge' }],
  ['vfs_list_embedding_dims', { domain: 'study-data', status: 'merge' }],
  ['vfs_list_dimensions', { domain: 'study-data', status: 'merge' }],
  ['vfs_get_resource_text_chunks', { domain: 'study-data', status: 'merge' }],
  ['vfs_get_resource_ocr_info', { domain: 'study-data', status: 'merge' }],
  ['vfs_clear_resource_ocr', { domain: 'study-data', status: 'merge' }],
  ['vfs_rag_search', { domain: 'study-data', status: 'merge' }],
  ['vfs_list_files', { domain: 'study-data', status: 'merge' }],
  ['vfs_create_mindmap', { domain: 'study-data', status: 'merge' }],
  ['vfs_get_mindmap', { domain: 'study-data', status: 'merge' }],
  ['vfs_get_mindmap_content', { domain: 'study-data', status: 'merge' }],
  ['vfs_update_mindmap', { domain: 'study-data', status: 'merge' }],
  ['vfs_delete_mindmap', { domain: 'study-data', status: 'merge' }],
  ['vfs_list_mindmaps', { domain: 'study-data', status: 'merge' }],
  ['vfs_set_mindmap_favorite', { domain: 'study-data', status: 'merge' }],
  ['vfs_get_mindmap_versions', { domain: 'study-data', status: 'merge' }],
  ['vfs_get_mindmap_version', { domain: 'study-data', status: 'merge' }],
  ['vfs_get_mindmap_version_content', { domain: 'study-data', status: 'merge' }],
  ['dstu_list', { domain: 'study-data', status: 'merge' }],
  ['dstu_get', { domain: 'study-data', status: 'merge' }],
  ['dstu_create', { domain: 'study-data', status: 'merge' }],
  ['dstu_update', { domain: 'study-data', status: 'merge' }],
  ['dstu_delete', { domain: 'study-data', status: 'merge' }],
  ['dstu_delete_many', { domain: 'study-data', status: 'merge' }],
  ['dstu_restore', { domain: 'study-data', status: 'merge' }],
  ['dstu_restore_many', { domain: 'study-data', status: 'merge' }],
  ['dstu_purge', { domain: 'study-data', status: 'merge' }],
  ['dstu_purge_all', { domain: 'study-data', status: 'merge' }],
  ['dstu_list_deleted', { domain: 'study-data', status: 'merge' }],
  ['dstu_soft_delete', { domain: 'study-data', status: 'merge' }],
  ['dstu_trash_restore', { domain: 'study-data', status: 'merge' }],
  ['dstu_list_trash', { domain: 'study-data', status: 'merge' }],
  ['dstu_empty_trash', { domain: 'study-data', status: 'merge' }],
  ['dstu_permanently_delete', { domain: 'study-data', status: 'merge' }],
  ['dstu_search', { domain: 'study-data', status: 'merge' }],
  ['dstu_get_content', { domain: 'study-data', status: 'merge' }],
  ['dstu_set_metadata', { domain: 'study-data', status: 'merge' }],
  ['dstu_set_favorite', { domain: 'study-data', status: 'merge' }],
  ['dstu_folder_create', { domain: 'study-data', status: 'merge' }],
  ['dstu_folder_get', { domain: 'study-data', status: 'merge' }],
  ['dstu_folder_rename', { domain: 'study-data', status: 'merge' }],
  ['dstu_folder_delete', { domain: 'study-data', status: 'merge' }],
  ['dstu_folder_move', { domain: 'study-data', status: 'merge' }],
  ['dstu_folder_set_expanded', { domain: 'study-data', status: 'merge' }],
  ['dstu_folder_add_item', { domain: 'study-data', status: 'merge' }],
  ['dstu_folder_remove_item', { domain: 'study-data', status: 'merge' }],
  ['dstu_folder_move_item', { domain: 'study-data', status: 'merge' }],
  ['dstu_folder_list', { domain: 'study-data', status: 'merge' }],
  ['dstu_folder_get_tree', { domain: 'study-data', status: 'merge' }],
  ['dstu_folder_get_items', { domain: 'study-data', status: 'merge' }],
  ['dstu_folder_get_all_resources', { domain: 'study-data', status: 'merge' }],
  ['dstu_folder_reorder', { domain: 'study-data', status: 'merge' }],
  ['dstu_folder_reorder_items', { domain: 'study-data', status: 'merge' }],
  ['dstu_folder_get_breadcrumbs', { domain: 'study-data', status: 'merge' }],
  ['dstu_get_resource_by_path', { domain: 'study-data', status: 'merge' }],
  ['dstu_get_resource_location', { domain: 'study-data', status: 'merge' }],
  ['dstu_parse_path', { domain: 'study-data', status: 'merge' }],
  ['dstu_build_path', { domain: 'study-data', status: 'merge' }],
  ['dstu_move_to_folder', { domain: 'study-data', status: 'merge' }],
  ['dstu_batch_move', { domain: 'study-data', status: 'merge' }],
  ['dstu_refresh_path_cache', { domain: 'study-data', status: 'merge' }],
  ['dstu_get_path_by_id', { domain: 'study-data', status: 'merge' }],
  ['notes_import_markdown', { domain: 'study-data', status: 'merge' }],
  ['notes_import_markdown_batch', { domain: 'study-data', status: 'merge' }],
  ['notes_search', { domain: 'study-data', status: 'merge' }],
  ['notes_mentions_search', { domain: 'study-data', status: 'merge' }],
  ['notes_list_tags', { domain: 'study-data', status: 'merge' }],
  ['notes_list_deleted', { domain: 'study-data', status: 'merge' }],
  ['notes_empty_trash', { domain: 'study-data', status: 'merge' }],
  ['notes_hard_delete', { domain: 'study-data', status: 'merge' }],
  ['notes_restore', { domain: 'study-data', status: 'merge' }],
  ['notes_assets_index_scan', { domain: 'study-data', status: 'merge' }],
  ['notes_assets_scan_orphans', { domain: 'study-data', status: 'merge' }],
  ['notes_assets_bulk_delete', { domain: 'study-data', status: 'merge' }],
  ['notes_db_stats', { domain: 'study-data', status: 'merge' }],
  ['notes_db_vacuum', { domain: 'study-data', status: 'merge' }],
  ['notes_export', { domain: 'study-data', status: 'merge' }],
  ['notes_export_single', { domain: 'study-data', status: 'merge' }],
  ['notes_import', { domain: 'study-data', status: 'merge' }],
  ['canvas_note_read', { domain: 'study-data', status: 'merge' }],
  ['canvas_note_append', { domain: 'study-data', status: 'merge' }],
  ['canvas_note_replace', { domain: 'study-data', status: 'merge' }],
  ['canvas_note_set', { domain: 'study-data', status: 'merge' }],
  ['qbank_list_questions', { domain: 'study-data', status: 'merge' }],
  ['qbank_search_questions', { domain: 'study-data', status: 'merge' }],
  ['qbank_rebuild_fts_index', { domain: 'study-data', status: 'merge' }],
  ['qbank_get_question', { domain: 'study-data', status: 'merge' }],
  ['qbank_create_question', { domain: 'study-data', status: 'merge' }],
  ['qbank_update_question', { domain: 'study-data', status: 'merge' }],
  ['qbank_delete_question', { domain: 'study-data', status: 'merge' }],
  ['qbank_batch_delete_questions', { domain: 'study-data', status: 'merge' }],
  ['qbank_submit_answer', { domain: 'study-data', status: 'merge' }],
  ['qbank_ai_grade', { domain: 'study-data', status: 'merge' }],
  ['qbank_cancel_grading', { domain: 'study-data', status: 'merge' }],
  ['qbank_sync_check', { domain: 'study-data', status: 'merge' }],
  ['qbank_get_sync_conflicts', { domain: 'study-data', status: 'merge' }],
  ['qbank_resolve_sync_conflict', { domain: 'study-data', status: 'merge' }],
  ['qbank_batch_resolve_conflicts', { domain: 'study-data', status: 'merge' }],
  ['qbank_set_sync_enabled', { domain: 'study-data', status: 'merge' }],
  ['qbank_update_sync_config', { domain: 'study-data', status: 'merge' }],
  ['qbank_toggle_favorite', { domain: 'study-data', status: 'merge' }],
  ['qbank_get_stats', { domain: 'study-data', status: 'merge' }],
  ['qbank_refresh_stats', { domain: 'study-data', status: 'merge' }],
  ['qbank_reset_progress', { domain: 'study-data', status: 'merge' }],
  ['qbank_reset_questions_progress', { domain: 'study-data', status: 'merge' }],
  ['qbank_get_history', { domain: 'study-data', status: 'merge' }],
  ['qbank_get_submissions', { domain: 'study-data', status: 'merge' }],
  ['qbank_get_learning_trend', { domain: 'study-data', status: 'merge' }],
  ['qbank_get_activity_heatmap', { domain: 'study-data', status: 'merge' }],
  ['qbank_get_knowledge_stats', { domain: 'study-data', status: 'merge' }],
  ['qbank_get_knowledge_stats_with_comparison', { domain: 'study-data', status: 'merge' }],
  ['qbank_start_timed_practice', { domain: 'study-data', status: 'merge' }],
  ['qbank_generate_mock_exam', { domain: 'study-data', status: 'merge' }],
  ['qbank_submit_mock_exam', { domain: 'study-data', status: 'merge' }],
  ['qbank_get_daily_practice', { domain: 'study-data', status: 'merge' }],
  ['qbank_generate_paper', { domain: 'study-data', status: 'merge' }],
  ['qbank_get_check_in_calendar', { domain: 'study-data', status: 'merge' }],
  ['get_csv_preview', { domain: 'study-data', status: 'merge' }],
  ['import_questions_csv', { domain: 'study-data', status: 'merge' }],
  ['export_questions_csv', { domain: 'study-data', status: 'merge' }],
  ['get_csv_exportable_fields', { domain: 'study-data', status: 'merge' }],
  ['review_plan_create', { domain: 'study-data', status: 'merge' }],
  ['review_plan_process', { domain: 'study-data', status: 'merge' }],
  ['review_plan_get_due', { domain: 'study-data', status: 'merge' }],
  ['review_plan_get_due_with_filter', { domain: 'study-data', status: 'merge' }],
  ['review_plan_get_stats', { domain: 'study-data', status: 'merge' }],
  ['review_plan_refresh_stats', { domain: 'study-data', status: 'merge' }],
  ['review_plan_get_by_question', { domain: 'study-data', status: 'merge' }],
  ['review_plan_get', { domain: 'study-data', status: 'merge' }],
  ['review_plan_suspend', { domain: 'study-data', status: 'merge' }],
  ['review_plan_resume', { domain: 'study-data', status: 'merge' }],
  ['review_plan_delete', { domain: 'study-data', status: 'merge' }],
  ['review_plan_get_history', { domain: 'study-data', status: 'merge' }],
  ['review_plan_batch_create', { domain: 'study-data', status: 'merge' }],
  ['review_plan_create_for_exam', { domain: 'study-data', status: 'merge' }],
  ['review_plan_list_by_exam', { domain: 'study-data', status: 'merge' }],
  ['review_plan_get_or_create', { domain: 'study-data', status: 'merge' }],
  ['review_plan_get_calendar_data', { domain: 'study-data', status: 'merge' }],
  ['get_app_data_dir', { domain: 'system', status: 'merge' }],
  ['ensure_debug_log_dir', { domain: 'system', status: 'merge' }],
  ['open_logs_folder', { domain: 'system', status: 'merge' }],
  ['report_frontend_log', { domain: 'system', status: 'merge' }],
  ['read_file_bytes', { domain: 'system', status: 'merge' }],
  ['get_file_size', { domain: 'system', status: 'merge' }],
  ['copy_file', { domain: 'system', status: 'merge' }],
  ['notes_get_pref', { domain: 'study-data', status: 'merge' }],
  ['notes_set_pref', { domain: 'study-data', status: 'merge' }],
  ['notes_save_asset', { domain: 'study-data', status: 'merge' }],
  ['notes_list_assets', { domain: 'study-data', status: 'merge' }],
  ['notes_delete_asset', { domain: 'study-data', status: 'merge' }],
  ['notes_resolve_asset_path', { domain: 'study-data', status: 'merge' }],
  ['get_image_as_base64', { domain: 'study-data', status: 'merge' }],
  ['todo_ensure_inbox', { domain: 'study-data', status: 'merge' }],
  ['todo_create_list', { domain: 'study-data', status: 'merge' }],
  ['todo_get_list', { domain: 'study-data', status: 'merge' }],
  ['todo_list_lists', { domain: 'study-data', status: 'merge' }],
  ['todo_update_list', { domain: 'study-data', status: 'merge' }],
  ['todo_delete_list', { domain: 'study-data', status: 'merge' }],
  ['todo_toggle_list_favorite', { domain: 'study-data', status: 'merge' }],
  ['todo_create_item', { domain: 'study-data', status: 'merge' }],
  ['todo_get_item', { domain: 'study-data', status: 'merge' }],
  ['todo_list_items', { domain: 'study-data', status: 'merge' }],
  ['todo_update_item', { domain: 'study-data', status: 'merge' }],
  ['todo_toggle_item', { domain: 'study-data', status: 'merge' }],
  ['todo_delete_item', { domain: 'study-data', status: 'merge' }],
  ['todo_reorder_items', { domain: 'study-data', status: 'merge' }],
  ['todo_list_today', { domain: 'study-data', status: 'merge' }],
  ['todo_list_overdue', { domain: 'study-data', status: 'merge' }],
  ['todo_list_upcoming', { domain: 'study-data', status: 'merge' }],
  ['todo_list_completed', { domain: 'study-data', status: 'merge' }],
  ['todo_search', { domain: 'study-data', status: 'merge' }],
  ['todo_get_active_summary', { domain: 'study-data', status: 'merge' }],
  ['pomodoro_create_record', { domain: 'study-data', status: 'merge' }],
  ['pomodoro_get_record', { domain: 'study-data', status: 'merge' }],
  ['pomodoro_list_by_todo', { domain: 'study-data', status: 'merge' }],
  ['pomodoro_today_stats', { domain: 'study-data', status: 'merge' }],
  ['pomodoro_list_today', { domain: 'study-data', status: 'merge' }],
  ['llm_usage_get_trends', { domain: 'llm', status: 'merge' }],
  ['llm_usage_by_model', { domain: 'llm', status: 'merge' }],
  ['llm_usage_by_caller', { domain: 'llm', status: 'merge' }],
  ['llm_usage_summary', { domain: 'llm', status: 'merge' }],
  ['llm_usage_recent', { domain: 'llm', status: 'merge' }],
  ['llm_usage_daily', { domain: 'llm', status: 'merge' }],
  ['llm_usage_cleanup', { domain: 'llm', status: 'merge' }],
  ['test_api_connection', { domain: 'settings', status: 'merge' }],
  ['test_ocr_engine', { domain: 'settings', status: 'merge' }],
  ['test_search_engine', { domain: 'settings', status: 'merge' }],
  ['test_web_search_connectivity', { domain: 'settings', status: 'merge' }],
  ['test_all_search_engines', { domain: 'settings', status: 'merge' }],
  ['get_mcp_tools', { domain: 'mcp', status: 'merge' }],
  ['mcp_stdio_close', { domain: 'mcp', status: 'merge' }],
  ['mcp_stdio_send', { domain: 'mcp', status: 'merge' }],
  ['mcp_stdio_start', { domain: 'mcp', status: 'merge' }],
  ['preheat_mcp_tools', { domain: 'mcp', status: 'merge' }],
]);

function classifyDomain(name, files) {
  const override = implementedCommandOverrides.get(name);
  if (override) return override.domain;

  const haystack = `${name} ${files.join(' ')}`;
  return domainRules.find(([, pattern]) => pattern.test(haystack))?.[0] ?? 'other';
}

function classifyStatus(name, domain, files) {
  const override = implementedCommandOverrides.get(name);
  if (override) return override.status;

  const haystack = `${name} ${files.join(' ')}`;
  if (/debug|test_|_test|health|snapshot|audit|dev|mock|raw_request/i.test(haystack)) return 'delete';
  if (domain === 'mcp') return 'defer';
  if (domain === 'settings') return 'merge';
  if (domain === 'system') return /^(read|save|copy|hash|get_file|open_)/i.test(name) ? 'merge' : 'defer';
  if (domain === 'chat') return /variant|multi|subagent|debug/i.test(name) ? 'defer' : 'replace';
  if (domain === 'study-data') return 'replace';
  if (domain === 'pdf-ocr') return 'defer';
  if (domain === 'backup-governance') return 'defer';
  if (domain === 'llm') return 'replace';
  return 'defer';
}

function rationale(status, domain) {
  switch (status) {
    case 'merge':
      return `Fold into a smaller ${domain} service instead of preserving per-command shape.`;
    case 'replace':
      return `Keep the product capability, but redesign the Go API and data flow.`;
    case 'delete':
      return `Likely debug/test/internal surface; exclude from lean release unless proven product-critical.`;
    case 'defer':
      return `Not part of the first lean release slice; revisit after core workflows work.`;
    default:
      return `Required as-is for the lean product surface.`;
  }
}

async function walk(dir) {
  const entries = await fs.readdir(dir, { withFileTypes: true });
  const files = [];
  for (const entry of entries) {
    if (entry.name === 'node_modules' || entry.name === 'dist') continue;
    const fullPath = path.join(dir, entry.name);
    if (entry.isDirectory()) {
      files.push(...await walk(fullPath));
    } else if (entry.isFile() && exts.has(path.extname(entry.name))) {
      files.push(fullPath);
    }
  }
  return files;
}

function rel(file) {
  return path.relative(root, file).replaceAll(path.sep, '/');
}

const calls = new Map();
for (const file of await walk(srcDir)) {
  const text = await fs.readFile(file, 'utf8');
  for (const match of text.matchAll(/\b(?:invoke|nativeInvoke)(?:<[^>]+>)?\(\s*['"`]([^'"`]+)['"`]/g)) {
    const name = match[1];
    const current = calls.get(name) ?? { name, count: 0, files: new Set() };
    current.count += 1;
    current.files.add(rel(file));
    calls.set(name, current);
  }
}

const items = [...calls.values()].map(item => {
  const files = [...item.files].sort();
  const domain = classifyDomain(item.name, files);
  const status = classifyStatus(item.name, domain, files);
  return {
    name: item.name,
    count: item.count,
    domain,
    status,
    rationale: rationale(status, domain),
    files,
  };
}).sort((a, b) => a.domain.localeCompare(b.domain) || a.status.localeCompare(b.status) || b.count - a.count || a.name.localeCompare(b.name));

const totals = items.reduce((acc, item) => {
  acc.byDomain[item.domain] = (acc.byDomain[item.domain] ?? 0) + 1;
  acc.byStatus[item.status] = (acc.byStatus[item.status] ?? 0) + 1;
  return acc;
}, { byDomain: {}, byStatus: {} });

await fs.mkdir(outDir, { recursive: true });
await fs.writeFile(path.join(outDir, 'native-command-triage.json'), JSON.stringify({
  generatedAt: new Date().toISOString(),
  uniqueCommands: items.length,
  ...totals,
  items,
}, null, 2));

const lines = [
  '# Native Command Triage',
  '',
  `Generated: ${new Date().toISOString()}`,
  `Unique commands: ${items.length}`,
  '',
  '## By Status',
  '',
  '| Status | Count |',
  '| --- | ---: |',
  ...Object.entries(totals.byStatus).sort().map(([status, count]) => `| ${status} | ${count} |`),
  '',
  '## By Domain',
  '',
  '| Domain | Count |',
  '| --- | ---: |',
  ...Object.entries(totals.byDomain).sort().map(([domain, count]) => `| ${domain} | ${count} |`),
  '',
  '## Commands',
  '',
  '| Domain | Status | Count | Command | Rationale |',
  '| --- | --- | ---: | --- | --- |',
  ...items.map(item => `| ${item.domain} | ${item.status} | ${item.count} | \`${item.name}\` | ${item.rationale} |`),
  '',
];

await fs.writeFile(path.join(outDir, 'native-command-triage.md'), lines.join('\n'));

console.log(`Wrote ${items.length} commands to docs/generated/native-command-triage.{json,md}`);
