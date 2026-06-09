//! 教材库命令模块
//! 从 commands.rs 剥离 (原始行号: 7077-7400)

use crate::commands::AppState;
use crate::document_parser::DocumentParser;
use crate::models::AppError;
use crate::textbooks_db::{ListQuery as TextbooksListQuery, Textbook as TextbookDto, TextbooksDb};
use crate::unified_file_manager;
use crate::vfs::repos::pdf_preview::{render_pdf_preview_with_progress, PdfPreviewConfig};
// ★ 2026-02 移除：VfsIndexService 和 UnitBuildInput 不再需要
// sync_resource_units 调用已移除，由 Pipeline 统一处理
use crate::vfs::{PdfProcessingService, ProcessingStage};
use chrono::Utc;
use serde::{Deserialize, Serialize};
use std::sync::Arc;
use tauri::{Emitter, State, Window};
use tracing::{info, warn};

type Result<T> = std::result::Result<T, AppError>;

fn attach_textbook_to_folder(
    vfs_db: &crate::vfs::VfsDatabase,
    textbook_id: &str,
    folder_id: Option<&str>,
) {
    if let Some(fid) = folder_id {
        let folder_item = crate::vfs::VfsFolderItem::new(
            Some(fid.to_string()),
            "file".to_string(),
            textbook_id.to_string(),
        );
        if let Err(e) = crate::vfs::VfsFolderRepo::add_item_to_folder(vfs_db, &folder_item) {
            warn!(
                "[Textbooks] Failed to attach textbook {} to folder {}: {}",
                textbook_id, fid, e
            );
        }
    }
}

fn emit_textbook_watch_event(window: &Window, textbook_id: &str, event_type: &str) {
    let dstu_path = format!("/{}", textbook_id);
    let watch_event = serde_json::json!({
        "type": event_type,
        "path": dstu_path,
    });

    if let Err(err) = window.emit(&format!("dstu:change:{}", dstu_path), &watch_event) {
        warn!(
            "[Textbooks] Failed to emit dstu:change:{} for {}: {}",
            event_type, textbook_id, err
        );
    }
    if let Err(err) = window.emit("dstu:change", &watch_event) {
        warn!(
            "[Textbooks] Failed to emit global dstu:change:{} for {}: {}",
            event_type, textbook_id, err
        );
    }
}

fn start_textbook_pipeline_if_needed(
    pdf_processing_service: &Arc<PdfProcessingService>,
    textbook_id: &str,
    extension: &str,
) {
    if extension != "pdf" {
        return;
    }

    let textbook_id = textbook_id.to_string();
    let pdf_service = pdf_processing_service.clone();
    tokio::spawn(async move {
        info!(
            "[Textbooks] Starting PDF pipeline for textbook: {}",
            textbook_id
        );
        if let Err(e) = pdf_service
            .start_pipeline(&textbook_id, Some(ProcessingStage::OcrProcessing))
            .await
        {
            warn!(
                "[Textbooks] Failed to start PDF pipeline for textbook {}: {}",
                textbook_id, e
            );
        }
    });
}

// ==================== 教材库（独立数据库）命令 ====================

#[tauri::command]
pub async fn textbooks_list(
    state: State<'_, AppState>,
    query: Option<TextbooksListQuery>,
) -> Result<Vec<TextbookDto>> {
    // ★ 切换到 VFS 版本
    let vfs_db = state
        .vfs_db
        .as_ref()
        .ok_or_else(|| AppError::configuration("VFS database not configured"))?;

    let q = query.unwrap_or(TextbooksListQuery {
        q: None,
        favorite: None,
        status: None,
        limit: Some(500),
        offset: Some(0),
        sort_by: Some("time".into()),
        order: Some("desc".into()),
    });

    let limit = q.limit.unwrap_or(500) as u32;
    let offset = q.offset.unwrap_or(0) as u32;
    // VFS 版本：include_global = true 以包含全局教材
    let vfs_items = TextbooksDb::list_vfs(vfs_db, None, true, limit, offset)?;

    // 转换为旧版 TextbookDto 以保持兼容性
    let items: Vec<TextbookDto> = vfs_items.into_iter().map(|v| v.to_textbook()).collect();
    Ok(items)
}

#[tauri::command]
pub async fn textbooks_remove(
    window: Window,
    state: State<'_, AppState>,
    id: String,
) -> Result<bool> {
    warn!(
        "[Textbooks] textbooks_remove is deprecated; prefer DSTU trash/purge flows for new callers. id={}",
        id
    );
    // ★ 切换到 VFS 版本
    let vfs_db = state
        .vfs_db
        .as_ref()
        .ok_or_else(|| AppError::configuration("VFS database not configured"))?;

    let deleted = TextbooksDb::delete_vfs(vfs_db, &id)?;

    if deleted {
        emit_textbook_watch_event(&window, &id, "purged");
    }

    Ok(deleted)
}

/// 采用已有文件（不复制），直接计算哈希并入库
#[tauri::command]
pub async fn textbooks_adopt(
    window: Window,
    state: State<'_, AppState>,
    pdf_processing_service: State<'_, Arc<PdfProcessingService>>,
    paths: Vec<String>,
    folder_id: Option<String>,
) -> Result<Vec<TextbookDto>> {
    if paths.is_empty() {
        return Ok(vec![]);
    }

    // ★ 切换到 VFS 版本
    let vfs_db = state
        .vfs_db
        .as_ref()
        .ok_or_else(|| AppError::configuration("VFS database not configured"))?;

    let mut out: Vec<TextbookDto> = Vec::new();
    for p in paths {
        let size = unified_file_manager::get_file_size(&window, &p)?;
        if size == 0 {
            continue;
        }
        let sha256 = unified_file_manager::hash_file_sha256(&window, &p)?;
        let (resolved_name, resolved_ext) = unified_file_manager::resolve_file_info(&window, &p);
        // ★ 移动端修复：不透明 document ID → 生成友好文件名
        let uri_raw_name = unified_file_manager::extract_file_name(&p);
        let file_name = if unified_file_manager::is_opaque_document_id(&uri_raw_name) {
            let ext_suffix = resolved_ext
                .as_ref()
                .map(|e| format!(".{}", e))
                .unwrap_or_default();
            format!(
                "导入文档_{}{}",
                Utc::now().format("%Y%m%d_%H%M%S"),
                ext_suffix
            )
        } else {
            resolved_name
        };
        let extension = resolved_ext.unwrap_or_else(|| {
            std::path::Path::new(&file_name)
                .extension()
                .and_then(|ext| ext.to_str())
                .unwrap_or_default()
                .to_lowercase()
        });

        if let Some(tb) = crate::vfs::VfsTextbookRepo::get_by_sha256(vfs_db, &sha256)
            .map_err(|e| AppError::database(format!("VFS 查询教材失败: {}", e)))?
        {
            let mut watch_event_type = "created";
            if tb.status != "active" {
                crate::vfs::VfsTextbookRepo::restore_textbook(vfs_db, &tb.id)
                    .map_err(|e| AppError::database(format!("VFS 恢复教材失败: {}", e)))?;
                watch_event_type = "restored";
            }
            attach_textbook_to_folder(vfs_db, &tb.id, folder_id.as_deref());
            emit_textbook_watch_event(&window, &tb.id, watch_event_type);
            start_textbook_pipeline_if_needed(pdf_processing_service.inner(), &tb.id, &extension);
            out.push(tb.to_textbook());
            continue;
        }

        let conn = vfs_db
            .get_conn_safe()
            .map_err(|e| AppError::database(format!("获取 VFS 连接失败: {}", e)))?;
        let blobs_dir = vfs_db.blobs_dir();

        let (preview_json_str, extracted_text, page_count) = if extension == "pdf" {
            let pdf_bytes = unified_file_manager::read_all_bytes(&window, &p)
                .map_err(|e| AppError::file_system(format!("读取 PDF 文件失败: {}", e)))?;

            match render_pdf_preview_with_progress(
                &conn,
                &blobs_dir,
                &pdf_bytes,
                &PdfPreviewConfig::default(),
                |_current_page, _total_pages| {},
            ) {
                Ok(result) => (
                    result
                        .preview_json
                        .as_ref()
                        .and_then(|preview| serde_json::to_string(preview).ok()),
                    result.extracted_text,
                    Some(result.page_count as i32),
                ),
                Err(e) => {
                    warn!(
                        "[Textbooks] PDF preview failed during adopt, storing without preview: {}",
                        e
                    );
                    (None, None, None)
                }
            }
        } else {
            let parser = DocumentParser::new();
            match parser.extract_text_from_path(&p) {
                Ok(text) => (None, Some(text), Some(1)),
                Err(e) => {
                    warn!(
                        "[Textbooks] Document parsing failed during adopt for {}: {}",
                        file_name, e
                    );
                    (None, None, None)
                }
            }
        };

        let tb = crate::vfs::VfsTextbookRepo::create_textbook_with_preview(
            &conn,
            &sha256,
            &file_name,
            size as i64,
            None,     // blob_hash
            Some(&p), // original_path
            preview_json_str.as_deref(),
            extracted_text.as_deref(),
            page_count,
        )
        .map_err(|e| AppError::database(format!("VFS 创建教材失败: {}", e)))?;

        attach_textbook_to_folder(vfs_db, &tb.id, folder_id.as_deref());

        emit_textbook_watch_event(&window, &tb.id, "created");
        start_textbook_pipeline_if_needed(pdf_processing_service.inner(), &tb.id, &extension);

        out.push(tb.to_textbook());
    }
    Ok(out)
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct PurgeTrashOptions {
    pub delete_files: Option<bool>,
}

/// 恢复回收站中的教材
#[tauri::command]
pub async fn textbooks_recover(state: State<'_, AppState>, id: String) -> Result<bool> {
    // ★ 切换到 VFS 版本
    let vfs_db = state
        .vfs_db
        .as_ref()
        .ok_or_else(|| AppError::configuration("VFS database not configured"))?;
    crate::vfs::VfsTextbookRepo::restore_textbook(vfs_db, &id)
        .map_err(|e| AppError::database(format!("VFS 恢复教材失败: {}", e)))?;
    Ok(true)
}

/// 清空回收站（可选物理删除文件）
#[tauri::command]
pub async fn textbooks_purge_trash(
    _window: Window,
    state: State<'_, AppState>,
    options: Option<PurgeTrashOptions>,
) -> Result<serde_json::Value> {
    // ★ 切换到 VFS 版本
    let vfs_db = state
        .vfs_db
        .as_ref()
        .ok_or_else(|| AppError::configuration("VFS database not configured"))?;

    let delete_files = options.and_then(|o| o.delete_files).unwrap_or(false);
    let mut deleted_files: Vec<String> = Vec::new();

    if delete_files {
        // 先获取所有已删除的教材，删除物理文件
        let trashed = crate::vfs::VfsTextbookRepo::list_deleted_textbooks(vfs_db, 10000, 0)
            .map_err(|e| AppError::database(format!("VFS 列出回收站失败: {}", e)))?;
        for tb in &trashed {
            if let Some(ref path) = tb.original_path {
                // content:// 等虚拟 URI 无法通过 std::fs 操作，跳过物理删除
                if unified_file_manager::is_virtual_uri(path) {
                    continue;
                }
                if std::path::Path::new(path).exists() {
                    if let Err(e) = std::fs::remove_file(path) {
                        warn!("[Textbooks] 删除文件失败: {} ({})", path, e);
                    } else {
                        deleted_files.push(path.clone());
                    }
                }
            }
        }
    }

    let purged = crate::vfs::VfsFileRepo::purge_deleted_files(vfs_db)
        .map_err(|e| AppError::database(format!("VFS 清空回收站失败: {}", e)))?;
    Ok(serde_json::json!({ "purged": purged, "deleted_files": deleted_files }))
}

/// 永久删除单个教材（可选物理删除）
#[tauri::command]
pub async fn textbooks_delete_permanent(
    _window: Window,
    state: State<'_, AppState>,
    id: String,
    delete_file: Option<bool>,
) -> Result<bool> {
    // ★ 切换到 VFS 版本
    let vfs_db = state
        .vfs_db
        .as_ref()
        .ok_or_else(|| AppError::configuration("VFS database not configured"))?;

    // 如果需要删除物理文件，先获取教材信息
    if delete_file.unwrap_or(false) {
        if let Ok(Some(tb)) = crate::vfs::VfsTextbookRepo::get_textbook(vfs_db, &id) {
            if let Some(ref path) = tb.original_path {
                // content:// 等虚拟 URI 无法通过 std::fs 操作，跳过物理删除
                if !unified_file_manager::is_virtual_uri(path) {
                    let p = std::path::Path::new(path);
                    if p.exists() {
                        if let Err(err) = std::fs::remove_file(p) {
                            warn!(
                                "[Textbooks] 永久删除教材时清理文件失败: {} ({})",
                                p.display(),
                                err
                            );
                        }
                    }
                }
            }
        }
    }

    crate::vfs::VfsTextbookRepo::purge_textbook_with_folder_item(vfs_db, &id)
        .map_err(|e| AppError::database(format!("VFS 永久删除教材失败: {}", e)))?;
    Ok(true)
}

/// 更新教材阅读进度（打开时间和页码）
#[tauri::command]
pub async fn textbooks_update_reading_progress(
    state: State<'_, AppState>,
    id: String,
    last_page: Option<i64>,
) -> Result<bool> {
    // ★ 切换到 VFS 版本
    let vfs_db = state
        .vfs_db
        .as_ref()
        .ok_or_else(|| AppError::configuration("VFS database not configured"))?;
    let params = crate::textbooks_db::VfsUpdateTextbookParams {
        last_page: last_page.map(|p| p as i32),
        ..Default::default()
    };
    TextbooksDb::update_vfs(vfs_db, &id, params)?;
    Ok(true)
}

/// 设置教材收藏状态
#[tauri::command]
pub async fn textbooks_set_favorite(
    state: State<'_, AppState>,
    id: String,
    favorite: bool,
) -> Result<bool> {
    // ★ 切换到 VFS 版本
    let vfs_db = state
        .vfs_db
        .as_ref()
        .ok_or_else(|| AppError::configuration("VFS database not configured"))?;
    let params = crate::textbooks_db::VfsUpdateTextbookParams {
        favorite: Some(favorite),
        ..Default::default()
    };
    TextbooksDb::update_vfs(vfs_db, &id, params)?;
    Ok(true)
}

/// 更新教材页数
#[tauri::command]
pub async fn textbooks_update_page_count(
    state: State<'_, AppState>,
    id: String,
    page_count: i64,
) -> Result<bool> {
    // ★ 切换到 VFS 版本
    let vfs_db = state
        .vfs_db
        .as_ref()
        .ok_or_else(|| AppError::configuration("VFS database not configured"))?;
    let params = crate::textbooks_db::VfsUpdateTextbookParams {
        page_count: Some(page_count as i32),
        ..Default::default()
    };
    TextbooksDb::update_vfs(vfs_db, &id, params)?;
    Ok(true)
}
