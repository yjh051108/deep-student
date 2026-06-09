//! DSTU Tauri 命令处理器
//!
//! 提供 DSTU 访达协议层的所有 Tauri 命令

use std::sync::Arc;

use serde_json::Value;
use tauri::{State, Window};

use super::error::DstuError;
use super::path_parser::build_simple_resource_path;
use super::types::{DstuListOptions, DstuNode, DstuNodeType, DstuWatchEvent};

// 从子模块导入路径工具和节点转换器
use super::handler_utils::{
    emit_watch_event,
    exam_to_dstu_node,
    extract_resource_info,
    fetch_resource_as_dstu_node,
    file_to_dstu_node,
    // CRUD 辅助函数
    get_resource_by_type_and_id,
    item_type_to_dstu_node_type,
    // 列表辅助函数
    list_resources_by_type_with_folder_path,
    list_unassigned_essays,
    list_unassigned_exams,
    list_unassigned_notes,
    list_unassigned_textbooks,
    list_unassigned_translations,
    mindmap_to_dstu_node,
    note_to_dstu_node,
    search_all,
    // 搜索辅助函数
    search_by_index,
    session_to_dstu_node,
    textbook_to_dstu_node,
    translation_to_dstu_node,
};

use crate::vfs::{
    canonical_folder_item_type, repos::VfsMindMapRepo, VfsCreateEssaySessionParams,
    VfsCreateExamSheetParams, VfsCreateMindMapParams, VfsCreateNoteParams, VfsDatabase,
    VfsEssayRepo, VfsExamRepo, VfsFileRepo, VfsFolderItem, VfsFolderRepo, VfsNoteRepo,
    VfsTextbookRepo, VfsTranslationRepo, VfsUpdateMindMapParams, VfsUpdateNoteParams,
};

// ============================================================================
// 记忆系统隐藏名称检测
// ============================================================================

/// 检测名称是否为记忆系统保留名称（以 `__` 开头且以 `__` 结尾）
/// 这些文件夹/笔记是记忆系统内部使用的，不应在 Finder 中展示给用户
fn is_memory_system_hidden_name(name: &str) -> bool {
    let trimmed = name.trim();
    trimmed.len() > 4 && trimmed.starts_with("__") && trimmed.ends_with("__")
}

/// 批量操作的最大数量限制 (防止 DoS 和超时)
const MAX_BATCH_SIZE: usize = 100;

// ============================================================================
// Tauri 命令
// ============================================================================
// ============================================================================
// 资源获取命令
// ============================================================================

/// 获取资源详情
///
/// 获取指定路径的资源节点详情。
///
/// ## 参数
/// - `path`: DSTU 路径（支持完整路径如 `/数学/notes/note_xxx` 或简化路径如 `/note_xxx` 或 `note_xxx`）
/// - `vfs_db`: VFS 数据库实例
///
/// ## 返回
/// 资源节点，不存在时返回 None
/// 创建资源
///
/// 在指定路径下创建新资源。
///
/// ## 参数
/// - `path`: 父目录路径（如 `/数学/notes`）
/// - `options`: 创建选项（类型、名称、内容等）
/// - `vfs_db`: VFS 数据库实例
///
/// ## 返回
/// 新创建的资源节点
/// 更新资源内容
///
/// 更新指定资源的内容。对于笔记等资源，会自动触发版本管理。
///
/// ## 参数
/// - `path`: 资源路径
/// - `content`: 新内容
/// - `vfs_db`: VFS 数据库实例
///
/// ## 返回
/// 更新后的资源节点
/// 删除资源
///
/// 删除指定路径的资源（软删除）。
///
/// ## 参数
/// - `path`: 资源路径（支持完整路径如 `/数学/notes/note_xxx` 或 ID 如 `note_xxx`）
/// - `vfs_db`: VFS 数据库实例
/// 移动/重命名资源
///
/// 将资源从一个路径移动到另一个路径。可用于：
/// - 跨科目移动（更新 subject 字段）
/// - 重命名
///
/// ## 参数
/// - `src`: 源路径
/// - `dst`: 目标路径
/// - `vfs_db`: VFS 数据库实例
///
/// ## 返回
/// 移动后的资源节点
#[tauri::command]
pub async fn dstu_move(
    src: String,
    dst: String,
    window: Window,
    vfs_db: State<'_, Arc<VfsDatabase>>,
) -> Result<DstuNode, String> {
    log::info!("[DSTU::handlers] dstu_move: src={}, dst={}", src, dst);

    // 统一路径解析
    let (src_type, src_id) = match extract_resource_info(&src) {
        Ok((rt, rid)) => (rt, rid),
        Err(e) => {
            log::error!(
                "[DSTU::handlers] dstu_move: FAILED - src={}, error={}",
                src,
                e
            );
            return Err(e.to_string());
        }
    };
    let resource_type = src_type;

    let item_type = match resource_type.as_str() {
        "notes" => "note",
        "textbooks" => "textbook",
        "exams" => "exam",
        "translations" => "translation",
        "essays" => "essay",
        "folders" => "folder",
        "mindmaps" => "mindmap",
        "files" | "images" | "attachments" => "file",
        _ => {
            return Err(DstuError::invalid_node_type(resource_type).to_string());
        }
    };

    let dest_folder_id = if dst.trim().is_empty() || dst.trim() == "/" {
        None
    } else {
        let (dst_type, dst_id) = match extract_resource_info(&dst) {
            Ok((rt, rid)) => (rt, rid),
            Err(e) => {
                log::error!(
                    "[DSTU::handlers] dstu_move: FAILED - dst={}, error={}",
                    dst,
                    e
                );
                return Err(e.to_string());
            }
        };
        if dst_type != "folders" {
            return Err("Destination must be a folder".to_string());
        }
        Some(dst_id)
    };

    if let Err(e) =
        VfsFolderRepo::move_item_to_folder(&vfs_db, item_type, &src_id, dest_folder_id.as_deref())
    {
        log::error!(
            "[DSTU::handlers] dstu_move: FAILED - type={}, id={}, error={}",
            item_type,
            src_id,
            e
        );
        return Err(e.to_string());
    }

    let node = match get_resource_by_type_and_id(&vfs_db, &resource_type, &src_id).await {
        Ok(Some(n)) => n,
        Ok(None) => {
            log::error!(
                "[DSTU::handlers] dstu_move: FAILED - resource not found after move, id={}",
                src_id
            );
            return Err(DstuError::not_found(&src).to_string());
        }
        Err(e) => {
            log::error!(
                "[DSTU::handlers] dstu_move: FAILED - get_resource error, id={}, error={}",
                src_id,
                e
            );
            return Err(e);
        }
    };

    // 发射移动事件
    emit_watch_event(
        &window,
        DstuWatchEvent::moved(&src, &node.path, node.clone()),
    );

    log::info!("[DSTU::handlers] dstu_move: moved {} to {}", src, node.path);
    Ok(node)
}

/// 重命名资源
///
/// 更新资源的显示名称/标题。
///
/// ## 参数
/// - `path`: 资源路径（如 `/数学/notes/note_xxx`）
/// - `new_name`: 新名称
/// - `vfs_db`: VFS 数据库实例
///
/// ## 返回
/// 重命名后的资源节点
#[tauri::command]
pub async fn dstu_rename(
    path: String,
    new_name: String,
    window: Window,
    vfs_db: State<'_, Arc<VfsDatabase>>,
) -> Result<DstuNode, String> {
    log::info!(
        "[DSTU::handlers] dstu_rename: path={}, new_name={}",
        path,
        new_name
    );

    // 统一路径解析
    let (resource_type, id) = match extract_resource_info(&path) {
        Ok((rt, rid)) => (rt, rid),
        Err(e) => {
            log::error!(
                "[DSTU::handlers] dstu_rename: FAILED - path={}, error={}",
                path,
                e
            );
            return Err(e.to_string());
        }
    };

    // 根据类型路由到对应 Repo
    let node = match resource_type.as_str() {
        "notes" => {
            // 更新笔记标题
            let mut updated_note = match VfsNoteRepo::update_note(
                &vfs_db,
                &id,
                VfsUpdateNoteParams {
                    title: Some(new_name.clone()),
                    content: None,
                    tags: None,
                    expected_updated_at: None,
                },
            ) {
                Ok(n) => {
                    log::info!(
                        "[DSTU::handlers] dstu_rename: SUCCESS - type=note, id={}",
                        id
                    );
                    n
                }
                Err(e) => {
                    log::error!(
                        "[DSTU::handlers] dstu_rename: FAILED - type=note, id={}, error={}",
                        id,
                        e
                    );
                    return Err(e.to_string());
                }
            };

            note_to_dstu_node(&updated_note)
        }
        "exams" => {
            // 更新题目集名称
            let updated_exam = match VfsExamRepo::update_exam_name(&vfs_db, &id, &new_name) {
                Ok(e) => {
                    log::info!(
                        "[DSTU::handlers] dstu_rename: SUCCESS - type=exam, id={}",
                        id
                    );
                    e
                }
                Err(e) => {
                    log::error!(
                        "[DSTU::handlers] dstu_rename: FAILED - type=exam, id={}, error={}",
                        id,
                        e
                    );
                    return Err(e.to_string());
                }
            };

            exam_to_dstu_node(&updated_exam)
        }
        "essays" => {
            // 更新作文会话标题（注意：essay_sessions 表，不是 essays 表）
            match VfsEssayRepo::update_session(
                &vfs_db,
                &id,
                Some(&new_name),
                None,
                None,
                None,
                None,
            ) {
                Ok(_) => log::info!(
                    "[DSTU::handlers] dstu_rename: updated essay session, id={}",
                    id
                ),
                Err(e) => {
                    log::error!("[DSTU::handlers] dstu_rename: FAILED - update_session error, id={}, error={}", id, e);
                    return Err(e.to_string());
                }
            }

            // 重新获取会话
            let session = match VfsEssayRepo::get_session(&vfs_db, &id) {
                Ok(Some(s)) => s,
                Ok(None) => {
                    log::error!("[DSTU::handlers] dstu_rename: FAILED - essay not found after rename, id={}", id);
                    return Err(DstuError::not_found(&path).to_string());
                }
                Err(e) => {
                    log::error!(
                        "[DSTU::handlers] dstu_rename: FAILED - get_session error, id={}, error={}",
                        id,
                        e
                    );
                    return Err(e.to_string());
                }
            };

            let essay_path = build_simple_resource_path(&session.id);
            let created_at_str = &session.created_at;
            let created_at = chrono::DateTime::parse_from_rfc3339(created_at_str)
                .map(|dt| dt.timestamp_millis())
                .unwrap_or_else(|e| {
                    log::warn!("[DSTU::handlers] Failed to parse created_at '{}': {}, using epoch fallback", created_at_str, e);
                    0_i64
                });
            let updated_at_str = &session.updated_at;
            let updated_at = chrono::DateTime::parse_from_rfc3339(updated_at_str)
                .map(|dt| dt.timestamp_millis())
                .unwrap_or_else(|e| {
                    log::warn!("[DSTU::handlers] Failed to parse updated_at '{}': {}, using epoch fallback", updated_at_str, e);
                    created_at
                });

            DstuNode::resource(
                &session.id,
                &essay_path,
                &session.title,
                DstuNodeType::Essay,
                &session.id,
            )
            .with_timestamps(created_at, updated_at)
            .with_metadata(serde_json::json!({
                "totalRounds": session.total_rounds,
                "isFavorite": session.is_favorite,
            }))
        }
        "translations" => {
            // 更新翻译标题
            let updated_translation =
                match VfsTranslationRepo::update_title(&vfs_db, &id, &new_name) {
                    Ok(t) => {
                        log::info!(
                            "[DSTU::handlers] dstu_rename: SUCCESS - type=translation, id={}",
                            id
                        );
                        t
                    }
                    Err(e) => {
                        log::error!(
                        "[DSTU::handlers] dstu_rename: FAILED - type=translation, id={}, error={}",
                        id,
                        e
                    );
                        return Err(e.to_string());
                    }
                };

            translation_to_dstu_node(&updated_translation)
        }
        "textbooks" => {
            // 更新教材文件名
            let updated_textbook = match VfsTextbookRepo::update_file_name(&vfs_db, &id, &new_name)
            {
                Ok(t) => {
                    log::info!(
                        "[DSTU::handlers] dstu_rename: SUCCESS - type=textbook, id={}",
                        id
                    );
                    t
                }
                Err(e) => {
                    log::error!(
                        "[DSTU::handlers] dstu_rename: FAILED - type=textbook, id={}, error={}",
                        id,
                        e
                    );
                    return Err(e.to_string());
                }
            };

            textbook_to_dstu_node(&updated_textbook)
        }
        "files" => {
            // 更新文件名
            let updated_file = match VfsFileRepo::update_file_name(&vfs_db, &id, &new_name) {
                Ok(f) => {
                    log::info!(
                        "[DSTU::handlers] dstu_rename: SUCCESS - type=file, id={}",
                        id
                    );
                    f
                }
                Err(e) => {
                    log::error!(
                        "[DSTU::handlers] dstu_rename: FAILED - type=file, id={}, error={}",
                        id,
                        e
                    );
                    return Err(e.to_string());
                }
            };

            file_to_dstu_node(&updated_file)
        }
        "images" => {
            // 图片通过 VfsFileRepo 管理
            let updated_file = match VfsFileRepo::update_file_name(&vfs_db, &id, &new_name) {
                Ok(f) => {
                    log::info!(
                        "[DSTU::handlers] dstu_rename: SUCCESS - type=image, id={}",
                        id
                    );
                    f
                }
                Err(e) => {
                    log::error!(
                        "[DSTU::handlers] dstu_rename: FAILED - type=image, id={}, error={}",
                        id,
                        e
                    );
                    return Err(e.to_string());
                }
            };

            file_to_dstu_node(&updated_file)
        }
        "mindmaps" => {
            // 更新知识导图标题
            let update_params = VfsUpdateMindMapParams {
                title: Some(new_name.clone()),
                description: None,
                content: None,
                default_view: None,
                theme: None,
                settings: None,
                expected_updated_at: None,
                version_source: None,
            };
            let updated_mindmap = match VfsMindMapRepo::update_mindmap(&vfs_db, &id, update_params)
            {
                Ok(m) => {
                    log::info!(
                        "[DSTU::handlers] dstu_rename: SUCCESS - type=mindmap, id={}",
                        id
                    );
                    m
                }
                Err(e) => {
                    log::error!(
                        "[DSTU::handlers] dstu_rename: FAILED - type=mindmap, id={}, error={}",
                        id,
                        e
                    );
                    return Err(e.to_string());
                }
            };

            mindmap_to_dstu_node(&updated_mindmap)
        }
        "folders" => {
            // 获取文件夹
            let mut folder = match VfsFolderRepo::get_folder(&vfs_db, &id) {
                Ok(Some(f)) => f,
                Ok(None) => {
                    log::error!(
                        "[DSTU::handlers] dstu_rename: FAILED - folder not found, id={}",
                        id
                    );
                    return Err(DstuError::not_found(&path).to_string());
                }
                Err(e) => {
                    log::error!(
                        "[DSTU::handlers] dstu_rename: FAILED - get_folder error, id={}, error={}",
                        id,
                        e
                    );
                    return Err(e.to_string());
                }
            };

            // 更新文件夹标题
            folder.title = new_name.clone();

            // 保存更新
            match VfsFolderRepo::update_folder(&vfs_db, &folder) {
                Ok(_) => {
                    log::info!(
                        "[DSTU::handlers] dstu_rename: SUCCESS - type=folder, id={}",
                        id
                    );
                }
                Err(e) => {
                    log::error!("[DSTU::handlers] dstu_rename: FAILED - update_folder error, id={}, error={}", id, e);
                    return Err(e.to_string());
                }
            }

            // 构建 DstuNode
            let folder_path = build_simple_resource_path(&folder.id);
            DstuNode::folder(&folder.id, &folder_path, &folder.title)
                .with_timestamps(folder.created_at, folder.updated_at)
                .with_metadata(serde_json::json!({
                    "isExpanded": folder.is_expanded,
                    "isFavorite": folder.is_favorite,
                    "icon": folder.icon,
                    "color": folder.color,
                }))
        }
        _ => {
            return Err(DstuError::invalid_node_type(resource_type).to_string());
        }
    };

    // 27-DSTU统一虚拟路径架构改造：重命名后清空 cached_path
    // 因为 cached_path 中包含资源标题，重命名后需要重新计算
    if let Err(e) = vfs_db.get_conn_safe().and_then(|conn| {
        let canonical_resource_type = canonical_folder_item_type(&resource_type);
        conn.execute(
            "UPDATE folder_items SET cached_path = NULL WHERE item_id = ?1 AND item_type = ?2 AND deleted_at IS NULL",
            rusqlite::params![id, canonical_resource_type],
        )
        .map_err(|e| crate::vfs::error::VfsError::Database(e.to_string()))
    }) {
        log::warn!(
            "[DSTU::handlers] dstu_rename: failed to clear cached_path for {}: {}",
            id,
            e
        );
    }

    // 发射更新事件
    emit_watch_event(&window, DstuWatchEvent::updated(&path, node.clone()));

    log::info!(
        "[DSTU::handlers] dstu_rename: renamed {} to {} (cached_path cleared)",
        path,
        new_name
    );
    Ok(node)
}

/// 复制资源
///
/// 将资源复制到另一个路径。
///
/// ## 参数
/// - `src`: 源路径
/// - `dst`: 目标路径
/// - `vfs_db`: VFS 数据库实例
///
/// ## 返回
/// 复制后的新资源节点
#[tauri::command]
pub async fn dstu_copy(
    src: String,
    dst: String,
    window: Window,
    vfs_db: State<'_, Arc<VfsDatabase>>,
) -> Result<DstuNode, String> {
    log::info!("[DSTU::handlers] dstu_copy: src={}, dst={}", src, dst);

    // 统一路径解析
    let (src_resource_type, src_id) = match extract_resource_info(&src) {
        Ok((rt, rid)) => (rt, rid),
        Err(e) => {
            log::error!(
                "[DSTU::handlers] dstu_copy: FAILED - src={}, error={}",
                src,
                e
            );
            return Err(e.to_string());
        }
    };

    // 解析目标文件夹 ID（参考 dstu_move 的实现）
    let dest_folder_id: Option<String> = if dst.trim().is_empty() || dst.trim() == "/" {
        None // 根目录
    } else {
        let (dst_type, dst_id) = match extract_resource_info(&dst) {
            Ok((rt, rid)) => (rt, rid),
            Err(e) => {
                log::error!(
                    "[DSTU::handlers] dstu_copy: FAILED - invalid dst path, error={}",
                    e
                );
                return Err(format!("Invalid destination path: {}", e));
            }
        };
        if dst_type != "folders" {
            return Err("Destination must be a folder".to_string());
        }
        Some(dst_id)
    };

    // 复制 = 创建新资源并复制内容
    let node = match src_resource_type.as_str() {
        "notes" => {
            // 获取原笔记
            let note = match VfsNoteRepo::get_note(&vfs_db, &src_id) {
                Ok(Some(n)) => n,
                Ok(None) => {
                    log::error!(
                        "[DSTU::handlers] dstu_copy: FAILED - note not found, id={}",
                        src_id
                    );
                    return Err(DstuError::not_found(&src).to_string());
                }
                Err(e) => {
                    log::error!(
                        "[DSTU::handlers] dstu_copy: FAILED - get_note error, id={}, error={}",
                        src_id,
                        e
                    );
                    return Err(e.to_string());
                }
            };

            let content = match VfsNoteRepo::get_note_content(&vfs_db, &src_id) {
                Ok(Some(c)) => c,
                Ok(None) => String::new(),
                Err(e) => {
                    log::error!("[DSTU::handlers] dstu_copy: FAILED - get_note_content error, id={}, error={}", src_id, e);
                    return Err(e.to_string());
                }
            };

            // 创建新笔记（复制）
            let new_note = match VfsNoteRepo::create_note(
                &vfs_db,
                VfsCreateNoteParams {
                    title: format!("{} (副本)", note.title),
                    content,
                    tags: note.tags.clone(),
                },
            ) {
                Ok(n) => {
                    log::info!(
                        "[DSTU::handlers] dstu_copy: SUCCESS - created copy, id={}",
                        n.id
                    );
                    n
                }
                Err(e) => {
                    log::error!(
                        "[DSTU::handlers] dstu_copy: FAILED - create_note error={}",
                        e
                    );
                    return Err(e.to_string());
                }
            };

            // 如果指定了目标文件夹，将新资源添加到文件夹
            if let Some(ref folder_id) = dest_folder_id {
                let folder_item = VfsFolderItem::new(
                    Some(folder_id.clone()),
                    "note".to_string(),
                    new_note.id.clone(),
                );
                if let Err(e) = VfsFolderRepo::add_item_to_folder(&vfs_db, &folder_item) {
                    log::warn!(
                        "[DSTU::handlers] dstu_copy: failed to add note to folder {}: {}",
                        folder_id,
                        e
                    );
                }
            }

            note_to_dstu_node(&new_note)
        }
        "textbooks" => {
            // 获取原教材
            let textbook = match VfsTextbookRepo::get_textbook(&vfs_db, &src_id) {
                Ok(Some(t)) => t,
                Ok(None) => {
                    log::error!(
                        "[DSTU::handlers] dstu_copy: FAILED - textbook not found, id={}",
                        src_id
                    );
                    return Err(DstuError::not_found(&src).to_string());
                }
                Err(e) => {
                    log::error!(
                        "[DSTU::handlers] dstu_copy: FAILED - get_textbook error, id={}, error={}",
                        src_id,
                        e
                    );
                    return Err(e.to_string());
                }
            };

            // 教材复制需要复制 blob 引用
            // 由于 blob 是内容寻址的（sha256），我们需要生成新的 sha256 或标记为副本
            // 为了简化，我们创建一个新的文件名但指向同一个 blob
            let new_file_name = format!("{} (副本)", textbook.file_name.trim_end_matches(".pdf"));
            let new_file_name = if textbook.file_name.ends_with(".pdf") {
                format!("{}.pdf", new_file_name)
            } else {
                new_file_name
            };

            // 使用新的 sha256（在原 sha256 基础上添加时间戳以确保唯一）
            let new_sha256 = format!(
                "{}_{}",
                textbook.sha256,
                chrono::Utc::now().timestamp_millis()
            );

            let new_textbook = match VfsTextbookRepo::create_textbook(
                &vfs_db,
                &new_sha256,
                &new_file_name,
                textbook.size,
                textbook.blob_hash.as_deref(),
                textbook.original_path.as_deref(),
            ) {
                Ok(t) => {
                    log::info!(
                        "[DSTU::handlers] dstu_copy: SUCCESS - created textbook copy, id={}",
                        t.id
                    );
                    t
                }
                Err(e) => {
                    log::error!(
                        "[DSTU::handlers] dstu_copy: FAILED - create_textbook error={}",
                        e
                    );
                    return Err(e.to_string());
                }
            };

            // 如果指定了目标文件夹，将新资源添加到文件夹
            if let Some(ref folder_id) = dest_folder_id {
                let folder_item = VfsFolderItem::new(
                    Some(folder_id.clone()),
                    "file".to_string(),
                    new_textbook.id.clone(),
                );
                if let Err(e) = VfsFolderRepo::add_item_to_folder(&vfs_db, &folder_item) {
                    log::warn!(
                        "[DSTU::handlers] dstu_copy: failed to add textbook to folder {}: {}",
                        folder_id,
                        e
                    );
                }
            }

            textbook_to_dstu_node(&new_textbook)
        }
        "translations" => {
            // 获取原翻译
            let translation = match VfsTranslationRepo::get_translation(&vfs_db, &src_id) {
                Ok(Some(t)) => t,
                Ok(None) => {
                    log::error!(
                        "[DSTU::handlers] dstu_copy: FAILED - translation not found, id={}",
                        src_id
                    );
                    return Err(DstuError::not_found(&src).to_string());
                }
                Err(e) => {
                    log::error!("[DSTU::handlers] dstu_copy: FAILED - get_translation error, id={}, error={}", src_id, e);
                    return Err(e.to_string());
                }
            };

            // 获取翻译内容
            let content = match VfsTranslationRepo::get_translation_content(&vfs_db, &src_id) {
                Ok(Some(c)) => c,
                Ok(None) => String::from(r#"{"source":"","translated":""}"#),
                Err(e) => {
                    log::error!("[DSTU::handlers] dstu_copy: FAILED - get_translation_content error, id={}, error={}", src_id, e);
                    return Err(e.to_string());
                }
            };

            // 解析内容 JSON
            let content_json: Value = serde_json::from_str(&content)
                .unwrap_or_else(|_| serde_json::json!({"source": "", "translated": ""}));
            let source = content_json
                .get("source")
                .and_then(|v| v.as_str())
                .unwrap_or("")
                .to_string();
            let translated = content_json
                .get("translated")
                .and_then(|v| v.as_str())
                .unwrap_or("")
                .to_string();

            // 创建新翻译
            let new_title = translation.title.map(|t| format!("{} (副本)", t));
            let new_translation = match VfsTranslationRepo::create_translation(
                &vfs_db,
                crate::vfs::types::VfsCreateTranslationParams {
                    title: new_title,
                    source,
                    translated,
                    src_lang: translation.src_lang.clone(),
                    tgt_lang: translation.tgt_lang.clone(),
                    engine: translation.engine.clone(),
                    model: translation.model.clone(),
                },
            ) {
                Ok(t) => {
                    log::info!(
                        "[DSTU::handlers] dstu_copy: SUCCESS - created translation copy, id={}",
                        t.id
                    );
                    t
                }
                Err(e) => {
                    log::error!(
                        "[DSTU::handlers] dstu_copy: FAILED - create_translation error={}",
                        e
                    );
                    return Err(e.to_string());
                }
            };

            // 如果指定了目标文件夹，将新资源添加到文件夹
            if let Some(ref folder_id) = dest_folder_id {
                let folder_item = VfsFolderItem::new(
                    Some(folder_id.clone()),
                    "translation".to_string(),
                    new_translation.id.clone(),
                );
                if let Err(e) = VfsFolderRepo::add_item_to_folder(&vfs_db, &folder_item) {
                    log::warn!(
                        "[DSTU::handlers] dstu_copy: failed to add translation to folder {}: {}",
                        folder_id,
                        e
                    );
                }
            }

            translation_to_dstu_node(&new_translation)
        }
        "exams" => {
            // 获取原题目集
            let exam = match VfsExamRepo::get_exam_sheet(&vfs_db, &src_id) {
                Ok(Some(e)) => e,
                Ok(None) => {
                    log::error!(
                        "[DSTU::handlers] dstu_copy: FAILED - exam not found, id={}",
                        src_id
                    );
                    return Err(DstuError::not_found(&src).to_string());
                }
                Err(e) => {
                    log::error!("[DSTU::handlers] dstu_copy: FAILED - get_exam_sheet error, id={}, error={}", src_id, e);
                    return Err(e.to_string());
                }
            };

            // 创建新题目集
            let new_exam_name = exam.exam_name.map(|n| format!("{} (副本)", n));
            let new_temp_id = format!("copy_{}", nanoid::nanoid!(10));

            let new_exam = match VfsExamRepo::create_exam_sheet(
                &vfs_db,
                VfsCreateExamSheetParams {
                    exam_name: new_exam_name,
                    temp_id: new_temp_id,
                    metadata_json: exam.metadata_json.clone(),
                    preview_json: exam.preview_json.clone(),
                    status: exam.status.clone(),
                    folder_id: dest_folder_id.clone(),
                },
            ) {
                Ok(e) => {
                    log::info!(
                        "[DSTU::handlers] dstu_copy: SUCCESS - created exam copy, id={}",
                        e.id
                    );
                    e
                }
                Err(e) => {
                    log::error!(
                        "[DSTU::handlers] dstu_copy: FAILED - create_exam_sheet error={}",
                        e
                    );
                    return Err(e.to_string());
                }
            };

            // 如果指定了目标文件夹，将新资源添加到文件夹
            if let Some(ref folder_id) = dest_folder_id {
                let folder_item = VfsFolderItem::new(
                    Some(folder_id.clone()),
                    "exam".to_string(),
                    new_exam.id.clone(),
                );
                if let Err(e) = VfsFolderRepo::add_item_to_folder(&vfs_db, &folder_item) {
                    log::warn!(
                        "[DSTU::handlers] dstu_copy: failed to add exam to folder {}: {}",
                        folder_id,
                        e
                    );
                }
            }

            exam_to_dstu_node(&new_exam)
        }
        "essays" => {
            // essays 使用 session 模型
            let session = match VfsEssayRepo::get_session(&vfs_db, &src_id) {
                Ok(Some(s)) => s,
                Ok(None) => {
                    log::error!(
                        "[DSTU::handlers] dstu_copy: FAILED - essay session not found, id={}",
                        src_id
                    );
                    return Err(DstuError::not_found(&src).to_string());
                }
                Err(e) => {
                    log::error!(
                        "[DSTU::handlers] dstu_copy: FAILED - get_session error, id={}, error={}",
                        src_id,
                        e
                    );
                    return Err(e.to_string());
                }
            };

            // 创建新会话（只复制会话元数据，不复制关联的作文轮次）
            let new_session = match VfsEssayRepo::create_session(
                &vfs_db,
                VfsCreateEssaySessionParams {
                    title: format!("{} (副本)", session.title),
                    essay_type: session.essay_type.clone(),
                    grade_level: session.grade_level.clone(),
                    custom_prompt: session.custom_prompt.clone(),
                },
            ) {
                Ok(s) => {
                    log::info!(
                        "[DSTU::handlers] dstu_copy: SUCCESS - created essay session copy, id={}",
                        s.id
                    );
                    s
                }
                Err(e) => {
                    log::error!(
                        "[DSTU::handlers] dstu_copy: FAILED - create_session error={}",
                        e
                    );
                    return Err(e.to_string());
                }
            };

            // 如果指定了目标文件夹，将新资源添加到文件夹
            if let Some(ref folder_id) = dest_folder_id {
                let folder_item = VfsFolderItem::new(
                    Some(folder_id.clone()),
                    "essay".to_string(),
                    new_session.id.clone(),
                );
                if let Err(e) = VfsFolderRepo::add_item_to_folder(&vfs_db, &folder_item) {
                    log::warn!(
                        "[DSTU::handlers] dstu_copy: failed to add essay to folder {}: {}",
                        folder_id,
                        e
                    );
                }
            }

            session_to_dstu_node(&new_session)
        }
        "files" | "images" => {
            // files 和 images 共享 VfsFileRepo
            let file = match VfsFileRepo::get_file(&vfs_db, &src_id) {
                Ok(Some(f)) => f,
                Ok(None) => {
                    log::error!(
                        "[DSTU::handlers] dstu_copy: FAILED - file not found, id={}",
                        src_id
                    );
                    return Err(DstuError::not_found(&src).to_string());
                }
                Err(e) => {
                    log::error!(
                        "[DSTU::handlers] dstu_copy: FAILED - get_file error, id={}, error={}",
                        src_id,
                        e
                    );
                    return Err(e.to_string());
                }
            };

            // 创建新文件记录（指向同一个 blob）
            let new_file_name = format!("{} (副本)", file.file_name);
            // 使用新的 sha256 以确保唯一性
            let new_sha256 = format!("{}_{}", file.sha256, chrono::Utc::now().timestamp_millis());

            let new_file = match VfsFileRepo::create_file(
                &vfs_db,
                &new_sha256,
                &new_file_name,
                file.size,
                &file.file_type,
                file.mime_type.as_deref(),
                file.blob_hash.as_deref(),
                file.original_path.as_deref(),
            ) {
                Ok(f) => {
                    log::info!(
                        "[DSTU::handlers] dstu_copy: SUCCESS - created file copy, id={}",
                        f.id
                    );
                    f
                }
                Err(e) => {
                    log::error!(
                        "[DSTU::handlers] dstu_copy: FAILED - create_file error={}",
                        e
                    );
                    return Err(e.to_string());
                }
            };

            // 如果指定了目标文件夹，将新资源添加到文件夹
            if let Some(ref folder_id) = dest_folder_id {
                let folder_item = VfsFolderItem::new(
                    Some(folder_id.clone()),
                    "file".to_string(),
                    new_file.id.clone(),
                );
                if let Err(e) = VfsFolderRepo::add_item_to_folder(&vfs_db, &folder_item) {
                    log::warn!(
                        "[DSTU::handlers] dstu_copy: failed to add file to folder {}: {}",
                        folder_id,
                        e
                    );
                }
            }

            file_to_dstu_node(&new_file)
        }
        "mindmaps" => {
            // 获取原知识导图
            let mindmap = match VfsMindMapRepo::get_mindmap(&vfs_db, &src_id) {
                Ok(Some(m)) => m,
                Ok(None) => {
                    log::error!(
                        "[DSTU::handlers] dstu_copy: FAILED - mindmap not found, id={}",
                        src_id
                    );
                    return Err(DstuError::not_found(&src).to_string());
                }
                Err(e) => {
                    log::error!(
                        "[DSTU::handlers] dstu_copy: FAILED - get_mindmap error, id={}, error={}",
                        src_id,
                        e
                    );
                    return Err(e.to_string());
                }
            };

            // 获取导图内容
            let content = match VfsMindMapRepo::get_mindmap_content(&vfs_db, &src_id) {
                Ok(Some(c)) => c,
                Ok(None) => {
                    r#"{"version":"1.0","root":{"id":"root","text":"根节点","children":[]}}"#
                        .to_string()
                }
                Err(e) => {
                    log::error!("[DSTU::handlers] dstu_copy: FAILED - get_mindmap_content error, id={}, error={}", src_id, e);
                    return Err(e.to_string());
                }
            };

            // M-078 修复：使用 create_mindmap_in_folder（事务版），确保导图创建和 folder 关联在同一事务中
            let new_mindmap = match VfsMindMapRepo::create_mindmap_in_folder(
                &vfs_db,
                VfsCreateMindMapParams {
                    title: format!("{} (副本)", mindmap.title),
                    description: mindmap.description.clone(),
                    content,
                    default_view: mindmap.default_view.clone(),
                    theme: mindmap.theme.clone(),
                },
                dest_folder_id.as_deref(),
            ) {
                Ok(m) => {
                    log::info!(
                        "[DSTU::handlers] dstu_copy: SUCCESS - created mindmap copy, id={}",
                        m.id
                    );
                    m
                }
                Err(e) => {
                    log::error!(
                        "[DSTU::handlers] dstu_copy: FAILED - create_mindmap error={}",
                        e
                    );
                    return Err(e.to_string());
                }
            };

            mindmap_to_dstu_node(&new_mindmap)
        }
        "folders" => {
            // 检查循环引用：目标文件夹不能是源文件夹或其子文件夹
            if let Some(ref dest_id) = dest_folder_id {
                if is_subfolder_of(&vfs_db, dest_id, &src_id)? {
                    log::error!(
                        "[DSTU::handlers] dstu_copy: FAILED - circular reference detected, src={}, dest={}",
                        src_id, dest_id
                    );
                    return Err("Cannot copy a folder into itself or its subfolder".to_string());
                }
            }
            // 递归复制文件夹
            copy_folder_recursive(&vfs_db, &src_id, dest_folder_id.clone(), 0)?
        }
        _ => {
            return Err(DstuError::invalid_node_type(src_resource_type).to_string());
        }
    };

    // 发射创建事件
    emit_watch_event(&window, DstuWatchEvent::created(&node.path, node.clone()));

    log::info!(
        "[DSTU::handlers] dstu_copy: copied {} to {}",
        src,
        node.path
    );
    Ok(node)
}

/// 检查目标文件夹是否是源文件夹或其子文件夹（循环引用检测）
///
/// ## 参数
/// - `vfs_db`: VFS 数据库实例
/// - `potential_child`: 潜在的子文件夹 ID（目标文件夹）
/// - `potential_parent`: 潜在的父文件夹 ID（源文件夹）
///
/// ## 返回
/// - `Ok(true)`: 目标是源文件夹或其子文件夹
/// - `Ok(false)`: 目标不是源文件夹的子文件夹
fn is_subfolder_of(
    vfs_db: &Arc<VfsDatabase>,
    potential_child: &str,
    potential_parent: &str,
) -> Result<bool, String> {
    // 如果目标和源相同，则是循环引用
    if potential_child == potential_parent {
        return Ok(true);
    }

    // 遍历 potential_child 的所有父文件夹，检查是否包含 potential_parent
    let mut current_id = potential_child.to_string();
    let mut depth = 0;
    const MAX_DEPTH: i32 = 100;

    while depth < MAX_DEPTH {
        // 获取当前文件夹的信息
        let folder = match VfsFolderRepo::get_folder(vfs_db, &current_id) {
            Ok(Some(f)) => f,
            Ok(None) => return Ok(false), // 文件夹不存在，到达终点
            Err(e) => return Err(e.to_string()),
        };

        // 获取父文件夹 ID
        let parent_id = match folder.parent_id {
            Some(pid) => pid,
            None => return Ok(false), // 到达根目录，没有找到循环引用
        };

        // 检查父文件夹是否是 potential_parent
        if parent_id == potential_parent {
            return Ok(true);
        }

        current_id = parent_id;
        depth += 1;
    }

    // 超过最大深度，视为没有循环引用
    Ok(false)
}

/// 递归复制文件夹的最大深度限制（防止无限循环）
const MAX_COPY_DEPTH: usize = 10;

/// 递归复制文件夹
///
/// ## 参数
/// - `vfs_db`: VFS 数据库实例
/// - `src_folder_id`: 源文件夹 ID
/// - `dest_parent_id`: 目标父文件夹 ID（None 表示根目录）
/// - `depth`: 当前递归深度
///
/// ## 返回
/// 新创建的文件夹节点
fn copy_folder_recursive(
    vfs_db: &Arc<VfsDatabase>,
    src_folder_id: &str,
    dest_parent_id: Option<String>,
    depth: usize,
) -> Result<DstuNode, String> {
    // 1. 检查递归深度限制
    if depth >= MAX_COPY_DEPTH {
        log::warn!(
            "[DSTU::handlers] copy_folder_recursive: max depth reached, src_folder_id={}",
            src_folder_id
        );
        return Err(format!(
            "文件夹复制深度超出限制（最大 {} 层）",
            MAX_COPY_DEPTH
        ));
    }

    // 2. 获取原文件夹信息
    let folder = match VfsFolderRepo::get_folder(vfs_db, src_folder_id) {
        Ok(Some(f)) => f,
        Ok(None) => {
            log::error!(
                "[DSTU::handlers] copy_folder_recursive: folder not found, id={}",
                src_folder_id
            );
            return Err(format!("文件夹不存在: {}", src_folder_id));
        }
        Err(e) => {
            log::error!(
                "[DSTU::handlers] copy_folder_recursive: get_folder error, id={}, error={}",
                src_folder_id,
                e
            );
            return Err(e.to_string());
        }
    };

    // 3. 创建新文件夹（标题加 "(副本)" 后缀，仅在顶层）
    let new_title = if depth == 0 {
        format!("{} (副本)", folder.title)
    } else {
        folder.title.clone()
    };

    let new_folder = crate::vfs::VfsFolder::new(
        new_title,
        dest_parent_id.clone(),
        folder.icon.clone(),
        folder.color.clone(),
    );

    if let Err(e) = VfsFolderRepo::create_folder(vfs_db, &new_folder) {
        log::error!(
            "[DSTU::handlers] copy_folder_recursive: create_folder error, error={}",
            e
        );
        return Err(e.to_string());
    }

    log::info!(
        "[DSTU::handlers] copy_folder_recursive: created folder copy, src={}, new_id={}",
        src_folder_id,
        new_folder.id
    );

    // 4. 获取原文件夹下的子文件夹
    let sub_folders = match VfsFolderRepo::list_folders_by_parent(vfs_db, Some(src_folder_id)) {
        Ok(folders) => folders,
        Err(e) => {
            log::warn!(
                "[DSTU::handlers] copy_folder_recursive: list_folders_by_parent error, id={}, error={}",
                src_folder_id,
                e
            );
            Vec::new()
        }
    };

    // 5. 递归复制子文件夹
    for sub_folder in sub_folders {
        if let Err(e) = copy_folder_recursive(
            vfs_db,
            &sub_folder.id,
            Some(new_folder.id.clone()),
            depth + 1,
        ) {
            log::warn!(
                "[DSTU::handlers] copy_folder_recursive: failed to copy subfolder {}: {}",
                sub_folder.id,
                e
            );
            // 继续复制其他子文件夹
        }
    }

    // 6. 获取原文件夹内的资源项
    let items = match VfsFolderRepo::list_items_by_folder(vfs_db, Some(src_folder_id)) {
        Ok(items) => items,
        Err(e) => {
            log::warn!(
                "[DSTU::handlers] copy_folder_recursive: list_items_by_folder error, id={}, error={}",
                src_folder_id,
                e
            );
            Vec::new()
        }
    };

    // 7. 复制每个资源到新文件夹
    for item in items {
        if let Err(e) = copy_resource_to_folder(vfs_db, &item, &new_folder.id) {
            log::warn!(
                "[DSTU::handlers] copy_folder_recursive: failed to copy item {}/{}: {}",
                item.item_type,
                item.item_id,
                e
            );
            // 继续复制其他资源
        }
    }

    // 8. 返回新文件夹节点
    let folder_path = build_simple_resource_path(&new_folder.id);
    Ok(
        DstuNode::folder(&new_folder.id, &folder_path, &new_folder.title)
            .with_timestamps(new_folder.created_at, new_folder.updated_at)
            .with_metadata(serde_json::json!({
                "isExpanded": new_folder.is_expanded,
                "isFavorite": new_folder.is_favorite,
                "icon": new_folder.icon,
                "color": new_folder.color,
            })),
    )
}

/// 复制单个资源到目标文件夹
///
/// ## 参数
/// - `vfs_db`: VFS 数据库实例
/// - `item`: 源文件夹项
/// - `dest_folder_id`: 目标文件夹 ID
fn copy_resource_to_folder(
    vfs_db: &Arc<VfsDatabase>,
    item: &VfsFolderItem,
    dest_folder_id: &str,
) -> Result<(), String> {
    match item.item_type.as_str() {
        "note" => {
            // 复制笔记
            let note = match VfsNoteRepo::get_note(vfs_db, &item.item_id) {
                Ok(Some(n)) => n,
                Ok(None) => return Err(format!("笔记不存在: {}", item.item_id)),
                Err(e) => return Err(e.to_string()),
            };

            let content = match VfsNoteRepo::get_note_content(vfs_db, &item.item_id) {
                Ok(Some(c)) => c,
                Ok(None) => String::new(),
                Err(e) => return Err(e.to_string()),
            };

            let new_note = match VfsNoteRepo::create_note(
                vfs_db,
                VfsCreateNoteParams {
                    title: note.title.clone(),
                    content,
                    tags: note.tags.clone(),
                },
            ) {
                Ok(n) => n,
                Err(e) => return Err(e.to_string()),
            };

            // 添加到目标文件夹
            let folder_item = VfsFolderItem::new(
                Some(dest_folder_id.to_string()),
                "note".to_string(),
                new_note.id.clone(),
            );
            VfsFolderRepo::add_item_to_folder(vfs_db, &folder_item).map_err(|e| e.to_string())?;

            log::debug!(
                "[DSTU::handlers] copy_resource_to_folder: copied note {} -> {}",
                item.item_id,
                new_note.id
            );
        }
        "textbook" => {
            // 复制教材
            let textbook = match VfsTextbookRepo::get_textbook(vfs_db, &item.item_id) {
                Ok(Some(t)) => t,
                Ok(None) => return Err(format!("教材不存在: {}", item.item_id)),
                Err(e) => return Err(e.to_string()),
            };

            let new_sha256 = format!(
                "{}_{}",
                textbook.sha256,
                chrono::Utc::now().timestamp_millis()
            );

            let new_textbook = match VfsTextbookRepo::create_textbook(
                vfs_db,
                &new_sha256,
                &textbook.file_name,
                textbook.size,
                textbook.blob_hash.as_deref(),
                textbook.original_path.as_deref(),
            ) {
                Ok(t) => t,
                Err(e) => return Err(e.to_string()),
            };

            let folder_item = VfsFolderItem::new(
                Some(dest_folder_id.to_string()),
                "file".to_string(),
                new_textbook.id.clone(),
            );
            VfsFolderRepo::add_item_to_folder(vfs_db, &folder_item).map_err(|e| e.to_string())?;

            log::debug!(
                "[DSTU::handlers] copy_resource_to_folder: copied textbook {} -> {}",
                item.item_id,
                new_textbook.id
            );
        }
        "translation" => {
            // 复制翻译
            let translation = match VfsTranslationRepo::get_translation(vfs_db, &item.item_id) {
                Ok(Some(t)) => t,
                Ok(None) => return Err(format!("翻译不存在: {}", item.item_id)),
                Err(e) => return Err(e.to_string()),
            };

            let content = match VfsTranslationRepo::get_translation_content(vfs_db, &item.item_id) {
                Ok(Some(c)) => c,
                Ok(None) => String::from(r#"{"source":"","translated":""}"#),
                Err(e) => return Err(e.to_string()),
            };

            let content_json: Value = serde_json::from_str(&content)
                .unwrap_or_else(|_| serde_json::json!({"source": "", "translated": ""}));
            let source = content_json
                .get("source")
                .and_then(|v| v.as_str())
                .unwrap_or("")
                .to_string();
            let translated = content_json
                .get("translated")
                .and_then(|v| v.as_str())
                .unwrap_or("")
                .to_string();

            let new_translation = match VfsTranslationRepo::create_translation(
                vfs_db,
                crate::vfs::types::VfsCreateTranslationParams {
                    title: translation.title.clone(),
                    source,
                    translated,
                    src_lang: translation.src_lang.clone(),
                    tgt_lang: translation.tgt_lang.clone(),
                    engine: translation.engine.clone(),
                    model: translation.model.clone(),
                },
            ) {
                Ok(t) => t,
                Err(e) => return Err(e.to_string()),
            };

            let folder_item = VfsFolderItem::new(
                Some(dest_folder_id.to_string()),
                "translation".to_string(),
                new_translation.id.clone(),
            );
            VfsFolderRepo::add_item_to_folder(vfs_db, &folder_item).map_err(|e| e.to_string())?;

            log::debug!(
                "[DSTU::handlers] copy_resource_to_folder: copied translation {} -> {}",
                item.item_id,
                new_translation.id
            );
        }
        "exam" => {
            // 复制题目集
            let exam = match VfsExamRepo::get_exam_sheet(vfs_db, &item.item_id) {
                Ok(Some(e)) => e,
                Ok(None) => return Err(format!("题目集不存在: {}", item.item_id)),
                Err(e) => return Err(e.to_string()),
            };

            let new_temp_id = format!("copy_{}", nanoid::nanoid!(10));

            let new_exam = match VfsExamRepo::create_exam_sheet(
                vfs_db,
                VfsCreateExamSheetParams {
                    exam_name: exam.exam_name.clone(),
                    temp_id: new_temp_id,
                    metadata_json: exam.metadata_json.clone(),
                    preview_json: exam.preview_json.clone(),
                    status: exam.status.clone(),
                    folder_id: Some(dest_folder_id.to_string()),
                },
            ) {
                Ok(e) => e,
                Err(e) => return Err(e.to_string()),
            };

            let folder_item = VfsFolderItem::new(
                Some(dest_folder_id.to_string()),
                "exam".to_string(),
                new_exam.id.clone(),
            );
            VfsFolderRepo::add_item_to_folder(vfs_db, &folder_item).map_err(|e| e.to_string())?;

            log::debug!(
                "[DSTU::handlers] copy_resource_to_folder: copied exam {} -> {}",
                item.item_id,
                new_exam.id
            );
        }
        "essay" => {
            // 复制作文会话
            let session = match VfsEssayRepo::get_session(vfs_db, &item.item_id) {
                Ok(Some(s)) => s,
                Ok(None) => return Err(format!("作文会话不存在: {}", item.item_id)),
                Err(e) => return Err(e.to_string()),
            };

            let new_session = match VfsEssayRepo::create_session(
                vfs_db,
                VfsCreateEssaySessionParams {
                    title: session.title.clone(),
                    essay_type: session.essay_type.clone(),
                    grade_level: session.grade_level.clone(),
                    custom_prompt: session.custom_prompt.clone(),
                },
            ) {
                Ok(s) => s,
                Err(e) => return Err(e.to_string()),
            };

            let folder_item = VfsFolderItem::new(
                Some(dest_folder_id.to_string()),
                "essay".to_string(),
                new_session.id.clone(),
            );
            VfsFolderRepo::add_item_to_folder(vfs_db, &folder_item).map_err(|e| e.to_string())?;

            log::debug!(
                "[DSTU::handlers] copy_resource_to_folder: copied essay {} -> {}",
                item.item_id,
                new_session.id
            );
        }
        "file" | "image" => {
            // 复制文件/图片
            let file = match VfsFileRepo::get_file(vfs_db, &item.item_id) {
                Ok(Some(f)) => f,
                Ok(None) => return Err(format!("文件不存在: {}", item.item_id)),
                Err(e) => return Err(e.to_string()),
            };

            let new_sha256 = format!("{}_{}", file.sha256, chrono::Utc::now().timestamp_millis());

            let new_file = match VfsFileRepo::create_file(
                vfs_db,
                &new_sha256,
                &file.file_name,
                file.size,
                &file.file_type,
                file.mime_type.as_deref(),
                file.blob_hash.as_deref(),
                file.original_path.as_deref(),
            ) {
                Ok(f) => f,
                Err(e) => return Err(e.to_string()),
            };

            let folder_item = VfsFolderItem::new(
                Some(dest_folder_id.to_string()),
                "file".to_string(),
                new_file.id.clone(),
            );
            VfsFolderRepo::add_item_to_folder(vfs_db, &folder_item).map_err(|e| e.to_string())?;

            log::debug!(
                "[DSTU::handlers] copy_resource_to_folder: copied file {} -> {}",
                item.item_id,
                new_file.id
            );
        }
        "mindmap" => {
            // 复制知识导图
            let mindmap =
                match crate::vfs::repos::VfsMindMapRepo::get_mindmap(vfs_db, &item.item_id) {
                    Ok(Some(m)) => m,
                    Ok(None) => return Err(format!("知识导图不存在: {}", item.item_id)),
                    Err(e) => return Err(e.to_string()),
                };

            let content =
                match crate::vfs::repos::VfsMindMapRepo::get_mindmap_content(vfs_db, &item.item_id)
                {
                    Ok(Some(c)) => c,
                    Ok(None) => {
                        r#"{"version":"1.0","root":{"id":"root","text":"根节点","children":[]}}"#
                            .to_string()
                    }
                    Err(e) => return Err(e.to_string()),
                };

            let new_mindmap = match crate::vfs::repos::VfsMindMapRepo::create_mindmap(
                vfs_db,
                VfsCreateMindMapParams {
                    title: mindmap.title.clone(),
                    description: mindmap.description.clone(),
                    content,
                    default_view: mindmap.default_view.clone(),
                    theme: mindmap.theme.clone(),
                },
            ) {
                Ok(m) => m,
                Err(e) => return Err(e.to_string()),
            };

            let folder_item = VfsFolderItem::new(
                Some(dest_folder_id.to_string()),
                "mindmap".to_string(),
                new_mindmap.id.clone(),
            );
            VfsFolderRepo::add_item_to_folder(vfs_db, &folder_item).map_err(|e| e.to_string())?;

            log::debug!(
                "[DSTU::handlers] copy_resource_to_folder: copied mindmap {} -> {}",
                item.item_id,
                new_mindmap.id
            );
        }
        _ => {
            log::warn!(
                "[DSTU::handlers] copy_resource_to_folder: unsupported item type: {}",
                item.item_type
            );
            // 跳过不支持的类型
        }
    }

    Ok(())
}

/// 获取资源内容
/// 获取题目集识别内容（支持多模态模式）
///
/// 用于上下文注入时获取题目集识别的格式化内容。
///
/// ## 参数
/// - `exam_id`: 题目集识别 ID（不需要完整路径，直接传 ID）
/// - `is_multimodal`: 是否为多模态模式
///   - `true`: 返回图片 + 文本交替的 ContentBlock[]
///   - `false`: 返回纯 XML 格式文本
///
/// ## 返回
/// - `Vec<ContentBlock>`: 格式化后的内容块列表
#[tauri::command]
pub async fn dstu_get_exam_content(
    exam_id: String,
    is_multimodal: bool,
    vfs_db: State<'_, Arc<VfsDatabase>>,
) -> Result<Vec<crate::chat_v2::resource_types::ContentBlock>, String> {
    log::info!(
        "[DSTU::handlers] dstu_get_exam_content: exam_id={}, is_multimodal={}",
        exam_id,
        is_multimodal
    );

    // 调用 exam_formatter 进行格式化
    super::exam_formatter::format_exam_for_context(&vfs_db.inner().clone(), &exam_id, is_multimodal)
        .await
}

// ============================================================================
// dstu_move_many: 批量移动
// ============================================================================

/// 批量移动资源到指定目录
///
/// ## 参数
/// - `paths`: 源路径列表
/// - `dest_folder`: 目标文件夹路径（如 /数学/notes）
/// - `window`: 窗口实例
/// - `vfs_db`: VFS 数据库实例
///
/// ## 返回
/// 成功移动的数量
#[tauri::command]
pub async fn dstu_move_many(
    paths: Vec<String>,
    dest_folder: String,
    window: Window,
    vfs_db: State<'_, Arc<VfsDatabase>>,
) -> Result<usize, String> {
    log::info!(
        "[DSTU::handlers] dstu_move_many: {} paths to {}",
        paths.len(),
        dest_folder
    );

    // 批量操作数量限制检查
    if paths.len() > MAX_BATCH_SIZE {
        return Err(format!(
            "批量操作数量超出限制：最多允许 {} 个，实际 {} 个",
            MAX_BATCH_SIZE,
            paths.len()
        ));
    }

    // 目标文件夹路径解析
    let dest_folder_id = if dest_folder.trim().is_empty() || dest_folder.trim() == "/" {
        None
    } else {
        let (dst_type, dst_id) = match extract_resource_info(&dest_folder) {
            Ok((rt, rid)) => (rt, rid),
            Err(e) => {
                log::error!(
                    "[DSTU::handlers] dstu_move_many: FAILED - dest={}, error={}",
                    dest_folder,
                    e
                );
                return Err(e.to_string());
            }
        };
        if dst_type != "folders" {
            return Err("Destination must be a folder".to_string());
        }
        Some(dst_id)
    };

    let mut success_count = 0;

    for path in &paths {
        // 统一路径解析
        let (resource_type, id) = match extract_resource_info(path) {
            Ok((rt, rid)) => (rt, rid),
            Err(_) => continue,
        };

        let item_type = match resource_type.as_str() {
            "notes" => "note",
            "textbooks" => "textbook",
            "exams" => "exam",
            "translations" => "translation",
            "essays" => "essay",
            "folders" => "folder",
            "mindmaps" => "mindmap",
            "files" | "images" | "attachments" => "file",
            _ => continue,
        };

        let result =
            VfsFolderRepo::move_item_to_folder(&vfs_db, item_type, &id, dest_folder_id.as_deref());
        if result.is_ok() {
            success_count += 1;

            if let Ok(Some(node)) = get_resource_by_type_and_id(&vfs_db, &resource_type, &id).await
            {
                let new_path = node.path.clone();
                emit_watch_event(&window, DstuWatchEvent::moved(path, &new_path, node));
            }
        } else if let Err(e) = result {
            log::warn!(
                "[DSTU::handlers] dstu_move_many: FAILED - type={}, id={}, error={}",
                item_type,
                id,
                e
            );
        }
    }

    log::info!(
        "[DSTU::handlers] dstu_move_many: moved {} of {} items",
        success_count,
        paths.len()
    );
    Ok(success_count)
}

// ============================================================================
// dstu_watch / dstu_unwatch: 资源变化监听
// ============================================================================

/// 注册资源变化监听（当前实现为前端事件通道占位）
#[tauri::command]
pub async fn dstu_watch(path: String) -> Result<(), String> {
    log::info!("[DSTU::handlers] dstu_watch: path={}", path);
    Ok(())
}

/// 取消资源变化监听（当前实现为前端事件通道占位）
#[tauri::command]
pub async fn dstu_unwatch(path: String) -> Result<(), String> {
    log::info!("[DSTU::handlers] dstu_unwatch: path={}", path);
    Ok(())
}

// ============================================================================
// dstu_search_in_folder: 文件夹内搜索
// ============================================================================

/// 在指定文件夹内搜索资源
///
/// ## 参数
/// - `folder_id`: VFS 文件夹 ID（可选，null 表示根目录）
/// - `query`: 搜索关键词
/// - `options`: 搜索选项
/// - `vfs_db`: VFS 数据库实例
///
/// ## 返回
/// 匹配的资源列表
#[tauri::command]
pub async fn dstu_search_in_folder(
    folder_id: Option<String>,
    query: String,
    options: Option<DstuListOptions>,
    vfs_db: State<'_, Arc<VfsDatabase>>,
) -> Result<Vec<DstuNode>, String> {
    log::info!(
        "[DSTU::handlers] dstu_search_in_folder: folder={:?}, query={}",
        folder_id,
        query
    );

    let options = options.unwrap_or_default();

    // 如果有 folder_id，先获取文件夹内的所有项
    if let Some(ref fid) = folder_id {
        // 需要获取文件夹的 subject
        let _folder = match crate::vfs::VfsFolderRepo::get_folder(&vfs_db, fid) {
            Ok(Some(f)) => f,
            Ok(None) => {
                log::error!(
                    "[DSTU::handlers] dstu_get_nodes_in_folder: FAILED - folder not found, id={}",
                    fid
                );
                return Err(format!("Folder not found: {}", fid));
            }
            Err(e) => {
                log::error!("[DSTU::handlers] dstu_get_nodes_in_folder: FAILED - get_folder error, id={}, error={}", fid, e);
                return Err(e.to_string());
            }
        };
        let items = match crate::vfs::VfsFolderRepo::list_items_by_folder(&vfs_db, Some(fid)) {
            Ok(i) => i,
            Err(e) => {
                log::error!("[DSTU::handlers] dstu_get_nodes_in_folder: FAILED - list_items_by_folder error, folder_id={}, error={}", fid, e);
                return Err(e.to_string());
            }
        };

        // 获取文件夹内所有 item_id 集合（用于索引召回过滤）
        let folder_item_ids: std::collections::HashSet<String> =
            items.iter().map(|item| item.item_id.clone()).collect();

        // 获取每个项的详细信息并按标题/文件名过滤
        let query_lower = query.to_lowercase();
        let mut results = Vec::new();
        for item in items {
            let node = match item.item_type.as_str() {
                "note" => {
                    if let Ok(Some(note)) = VfsNoteRepo::get_note(&vfs_db, &item.item_id) {
                        if note.title.to_lowercase().contains(&query_lower) {
                            Some(note_to_dstu_node(&note))
                        } else {
                            None
                        }
                    } else {
                        None
                    }
                }
                "textbook" => {
                    if let Ok(Some(tb)) =
                        VfsTextbookRepo::get_active_textbook(&vfs_db, &item.item_id)
                    {
                        if tb.file_name.to_lowercase().contains(&query_lower) {
                            Some(textbook_to_dstu_node(&tb))
                        } else {
                            None
                        }
                    } else {
                        None
                    }
                }
                "file" | "image" => {
                    if let Ok(Some(f)) = VfsFileRepo::get_active_file(&vfs_db, &item.item_id) {
                        if f.file_name.to_lowercase().contains(&query_lower) {
                            Some(file_to_dstu_node(&f))
                        } else {
                            None
                        }
                    } else {
                        None
                    }
                }
                "translation" => {
                    if let Ok(Some(t)) = VfsTranslationRepo::get_translation(&vfs_db, &item.item_id)
                    {
                        if t.title
                            .as_deref()
                            .unwrap_or("")
                            .to_lowercase()
                            .contains(&query_lower)
                        {
                            Some(translation_to_dstu_node(&t))
                        } else {
                            None
                        }
                    } else {
                        None
                    }
                }
                "exam" => {
                    if let Ok(Some(e)) = VfsExamRepo::get_exam_sheet(&vfs_db, &item.item_id) {
                        if e.exam_name
                            .as_deref()
                            .unwrap_or("")
                            .to_lowercase()
                            .contains(&query_lower)
                        {
                            Some(exam_to_dstu_node(&e))
                        } else {
                            None
                        }
                    } else {
                        None
                    }
                }
                "mindmap" => {
                    if let Ok(Some(m)) = VfsMindMapRepo::get_mindmap(&vfs_db, &item.item_id) {
                        if m.title.to_lowercase().contains(&query_lower) {
                            Some(mindmap_to_dstu_node(&m))
                        } else {
                            None
                        }
                    } else {
                        None
                    }
                }
                _ => None,
            };

            if let Some(n) = node {
                // ★ 记忆系统改造：搜索结果也需隐藏 __*__ 系统笔记
                if is_memory_system_hidden_name(&n.name) {
                    continue;
                }
                results.push(n);
            }
        }

        // ★ 索引内容召回：追加内容匹配的结果，限定在当前文件夹范围内
        let existing_ids: std::collections::HashSet<String> =
            results.iter().map(|n| n.id.clone()).collect();
        let index_limit = options.limit.unwrap_or(50);
        if let Ok(index_results) = search_by_index(&vfs_db, &query, index_limit, &existing_ids) {
            for node in index_results {
                // 只保留属于当前文件夹的资源
                if folder_item_ids.contains(&node.id) {
                    // ★ 记忆系统改造：搜索结果也需隐藏 __*__ 系统笔记
                    if is_memory_system_hidden_name(&node.name) {
                        continue;
                    }
                    results.push(node);
                }
            }
        }

        // 按更新时间排序
        results.sort_by(|a, b| b.updated_at.cmp(&a.updated_at));

        // 限制结果数量
        if let Some(limit) = options.limit {
            results.truncate(limit as usize);
        }

        return Ok(results);
    }

    // 没有指定文件夹，使用全局搜索
    let mut results = search_all(&vfs_db, &query, &options)?;
    // ★ 记忆系统改造：全局搜索结果也需隐藏 __*__ 系统保留笔记
    results.retain(|node| !is_memory_system_hidden_name(&node.name));
    log::info!(
        "[DSTU::handlers] dstu_search_in_folder: global search found {} results",
        results.len()
    );
    Ok(results)
}

// ============================================================================
// 辅助函数：列出未分配到 folder_items 的资源（向后兼容旧数据）
// ============================================================================

// ============================================================================
// E5: Subject 迁移命令（文档 28 Prompt 6）
// ============================================================================

// ============================================================================
// 单元测试
// ============================================================================

#[cfg(test)]
mod tests {
    use super::*;
    use crate::dstu::handler_utils::{create_type_folder, generate_resource_id};

    #[test]
    fn test_generate_resource_id() {
        let id = generate_resource_id(&DstuNodeType::Note);
        assert!(id.starts_with("note_"));
        assert_eq!(id.len(), 15); // "note_" + 10 chars

        let id = generate_resource_id(&DstuNodeType::Textbook);
        assert!(id.starts_with("tb_"));

        let id = generate_resource_id(&DstuNodeType::Translation);
        assert!(id.starts_with("tr_"));
    }

    #[test]
    fn test_create_type_folder() {
        let folder = create_type_folder(DstuNodeType::Note);
        assert_eq!(folder.node_type, DstuNodeType::Folder);
        assert_eq!(folder.name, "笔记");
        assert_eq!(folder.path, "/notes");

        let folder = create_type_folder(DstuNodeType::Translation);
        assert_eq!(folder.path, "/translations");
    }

    // ============================================================================
    // 路径和路由测试（纯函数，不依赖 VfsDatabase）
    // ============================================================================

    /// 验证简化路径格式
    #[test]
    fn test_simple_path_format() {
        // 验证简化路径格式正确性
        let resource_type = "notes";
        let id = "note_abc123";

        let simple_path = format!("/{}", id);
        assert_eq!(simple_path, "/note_abc123");
    }

    // 这些函数已被 build_simple_resource_path 替代

    /// 验证 build_simple_resource_path 函数
    #[test]
    fn test_build_simple_resource_path() {
        let path = build_simple_resource_path("note_123");
        assert_eq!(path, "/note_123");

        let path2 = build_simple_resource_path("tr_456");
        assert_eq!(path2, "/tr_456");
    }
}
