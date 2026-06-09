//! 笔记系统命令模块
//! 从 commands.rs 剥离 (原始行号: 3505-5798)

#![allow(non_snake_case)] // Tauri 命令参数使用 camelCase 与前端保持一致

use crate::commands::AppState;
use crate::models::AppError;
use tauri::{State, Window};

type Result<T> = std::result::Result<T, AppError>;

// ================= Notes: 独立笔记系统（CRUD） =================

#[tauri::command]
pub async fn notes_list(
    _subject: String,
    state: State<'_, AppState>,
) -> Result<Vec<crate::notes_manager::NoteItem>> {
    // 使用 spawn_blocking 避免 Lance 操作导致的死锁
    let notes_manager = state.notes_manager.clone();

    tokio::task::spawn_blocking(move || notes_manager.list_notes_vfs(None, 1000, 0))
        .await
        .map_err(|e| AppError::internal(format!("列出笔记任务失败: {}", e)))?
}

/// 轻量列表：不返回 content_md，用于初次渲染降低载荷
#[tauri::command]
pub async fn notes_list_meta(
    _subject: String,
    state: State<'_, AppState>,
) -> Result<Vec<crate::notes_manager::NoteItem>> {
    // 使用 spawn_blocking 避免 Lance 操作导致的死锁
    let notes_manager = state.notes_manager.clone();
    tokio::task::spawn_blocking(move || notes_manager.list_notes_meta())
        .await
        .map_err(|e| AppError::internal(format!("列出笔记元数据任务失败: {}", e)))?
}

#[derive(Debug, serde::Deserialize)]
pub struct NotesListAdvancedOptions {
    pub tags: Option<Vec<String>>,
    pub date_start: Option<String>,
    pub date_end: Option<String>,
    pub has_assets: Option<bool>,
    pub sort_by: Option<String>,
    pub sort_dir: Option<String>,
    pub page: Option<i64>,
    pub page_size: Option<i64>,
    pub keyword: Option<String>,
    pub include_deleted: Option<bool>,
    pub only_deleted: Option<bool>,
}

#[derive(Debug, serde::Serialize)]
pub struct NotesListAdvancedResponse {
    pub items: Vec<crate::notes_manager::NoteItem>,
    pub total: i64,
    pub page: i64,
    pub page_size: i64,
}

#[tauri::command]
pub async fn notes_list_advanced(
    _subject: String,
    options: NotesListAdvancedOptions,
    state: State<'_, AppState>,
) -> Result<NotesListAdvancedResponse> {
    // 使用 spawn_blocking 避免 Lance 操作导致的死锁
    let notes_manager = state.notes_manager.clone();
    let opt = crate::notes_manager::ListOptions {
        tags: options.tags,
        date_start: options.date_start,
        date_end: options.date_end,
        has_assets: options.has_assets,
        sort_by: options.sort_by,
        sort_dir: options.sort_dir,
        page: options.page.unwrap_or(0),
        page_size: options.page_size.unwrap_or(20),
        keyword: options.keyword,
        include_deleted: options.include_deleted.unwrap_or(false),
        only_deleted: options.only_deleted.unwrap_or(false),
    };
    let page = options.page.unwrap_or(0);
    let page_size = options.page_size.unwrap_or(20);

    let (items, total) =
        tokio::task::spawn_blocking(move || notes_manager.list_notes_advanced(opt))
            .await
            .map_err(|e| AppError::internal(format!("高级列表任务失败: {}", e)))??;

    Ok(NotesListAdvancedResponse {
        items,
        total,
        page,
        page_size,
    })
}

#[derive(Debug, serde::Deserialize)]
pub struct NewNotePayload {
    pub title: String,
    pub content_md: String,
    pub tags: Option<Vec<String>>,
}

#[tauri::command]
pub async fn notes_create(
    _subject: String,
    note: NewNotePayload,
    state: State<'_, AppState>,
    _window: Window,
) -> Result<crate::notes_manager::NoteItem> {
    let tags: Vec<String> = note.tags.unwrap_or_default();

    // 使用 spawn_blocking 避免在异步上下文中阻塞
    let notes_manager = state.notes_manager.clone();
    let title = note.title.clone();
    let content_md = note.content_md.clone();
    let tags_clone = tags.clone();

    let created = tokio::task::spawn_blocking(move || {
        notes_manager.create_note_vfs(&title, &content_md, &tags_clone)
    })
    .await
    .map_err(|e| AppError::internal(format!("创建笔记任务失败: {}", e)))??;

    Ok(created)
}

#[derive(Debug, serde::Deserialize)]
pub struct UpdateNotePayload {
    pub id: String,
    pub title: Option<String>,
    pub content_md: Option<String>,
    pub tags: Option<Vec<String>>,
    pub should_reindex: Option<bool>,
    pub content_hash: Option<String>,
    pub force_reindex: Option<bool>,
    pub expected_updated_at: Option<String>,
}

#[tauri::command]
pub async fn notes_update(
    _subject: String,
    note: UpdateNotePayload,
    state: State<'_, AppState>,
    _window: Window,
) -> Result<crate::notes_manager::NoteItem> {
    // 使用 spawn_blocking 避免在异步上下文中阻塞
    let notes_manager = state.notes_manager.clone();
    let note_id = note.id.clone();
    let title = note.title.clone();
    let content_md = note.content_md.clone();
    let tags = note.tags.clone();
    let expected_updated_at = note.expected_updated_at.clone();

    let updated = tokio::task::spawn_blocking(move || {
        notes_manager.update_note_vfs(
            &note_id,
            title.as_deref(),
            content_md.as_deref(),
            tags.as_deref(),
            expected_updated_at.as_deref(),
        )
    })
    .await
    .map_err(|e| AppError::internal(format!("更新笔记任务失败: {}", e)))??;

    Ok(updated)
}

#[tauri::command]
pub async fn notes_set_favorite(
    subject: String,
    id: String,
    favorite: bool,
    state: State<'_, AppState>,
) -> Result<crate::notes_manager::NoteItem> {
    // 使用 spawn_blocking 避免 Lance 操作导致的死锁
    let notes_manager = state.notes_manager.clone();
    let _subject = subject; // VFS 版本不需要 subject，只需要 note_id
                            // ★ 切换到 VFS 版本
    tokio::task::spawn_blocking(move || notes_manager.set_favorite_vfs(&id, favorite))
        .await
        .map_err(|e| AppError::internal(format!("设置收藏任务失败: {}", e)))?
}

/// 获取单条笔记（包含内容）
#[tauri::command]
pub async fn notes_get(
    subject: String,
    id: String,
    state: State<'_, AppState>,
) -> Result<crate::notes_manager::NoteItem> {
    // 使用 spawn_blocking 避免潜在的死锁
    let notes_manager = state.notes_manager.clone();
    let _subject = subject; // VFS 版本不需要 subject，只需要 note_id
                            // ★ 切换到 VFS 版本
    tokio::task::spawn_blocking(move || notes_manager.get_note_vfs(&id))
        .await
        .map_err(|e| AppError::internal(format!("获取笔记任务失败: {}", e)))?
}

#[tauri::command]
pub async fn notes_delete(subject: String, id: String, state: State<'_, AppState>) -> Result<bool> {
    // 回收站语义：软删除仅标记 deleted_at，不删除 RAG 文档/映射与资产，
    // 以便回收站中仍可通过恢复找回，且检索层已在查询时过滤 deleted_at 笔记。
    // 使用 spawn_blocking 避免 Lance 操作导致的死锁
    let notes_manager = state.notes_manager.clone();
    let _subject = subject; // VFS 版本不需要 subject，只需要 note_id
                            // ★ 切换到 VFS 版本
    tokio::task::spawn_blocking(move || notes_manager.delete_note_vfs(&id))
        .await
        .map_err(|e| AppError::internal(format!("删除笔记任务失败: {}", e)))?
}

// ============== RAG FTS 索引维护 ==============

/// 重建主库（mistakes.db）的 RAG 文档块 FTS 索引
#[tauri::command]
pub async fn rag_rebuild_fts_index(state: State<'_, AppState>) -> Result<usize> {
    let _ = state;
    println!("ℹ️ Lance RAG 检索使用原生 FTS，无需额外重建");
    Ok(0)
}

/// 重建笔记库（notes.db）的 RAG 文档块 FTS 索引
#[tauri::command]
pub async fn notes_rag_rebuild_fts_index(state: State<'_, AppState>) -> Result<usize> {
    let _ = state;
    println!("ℹ️ Notes RAG 已使用 Lance 内置 FTS，无需重建");
    Ok(0)
}

// Notes 专属 RAG 学科参数（每学科 chunk_size/overlap/rerank）
#[derive(Debug, serde::Serialize, serde::Deserialize, Clone)]
pub struct NotesSubjectRagConfig {
    pub chunk_size: i32,
    pub chunk_overlap: i32,
    pub min_chunk_size: i32,
    pub rerank_enabled: bool,
}

#[tauri::command]
pub async fn notes_get_subject_rag_config(
    subject: String,
    state: State<'_, AppState>,
) -> Result<NotesSubjectRagConfig> {
    // 从 notes_database.settings 中读取，没有则使用 rag_configurations 默认
    if let Ok(Some(json)) = state
        .notes_database
        .get_setting(&format!("notes.rag.config.{}", subject))
    {
        if let Ok(cfg) = serde_json::from_str::<NotesSubjectRagConfig>(&json) {
            return Ok(cfg);
        }
    }
    // fallback 默认
    let def = state
        .notes_database
        .get_rag_configuration()
        .map_err(|e| AppError::database(e.to_string()))?;
    Ok(NotesSubjectRagConfig {
        chunk_size: def.as_ref().map(|c| c.chunk_size).unwrap_or(512),
        chunk_overlap: def.as_ref().map(|c| c.chunk_overlap).unwrap_or(50),
        min_chunk_size: def.as_ref().map(|c| c.min_chunk_size).unwrap_or(20),
        rerank_enabled: def
            .as_ref()
            .map(|c| c.default_rerank_enabled)
            .unwrap_or(true),
    })
}

#[tauri::command]
pub async fn notes_update_subject_rag_config(
    subject: String,
    cfg: NotesSubjectRagConfig,
    state: State<'_, AppState>,
) -> Result<bool> {
    // 参数校验（与全局RAG设置保持一致并加上更严格的重叠约束）
    if cfg.chunk_size < 50 || cfg.chunk_size > 2048 {
        return Err(AppError::validation("分块大小必须在50-2048之间"));
    }
    if cfg.min_chunk_size < 10 || cfg.min_chunk_size > cfg.chunk_size {
        return Err(AppError::validation("最小分块大小必须在10和分块大小之间"));
    }
    // 基础约束：重叠 < 分块
    if cfg.chunk_overlap < 0 || cfg.chunk_overlap >= cfg.chunk_size {
        return Err(AppError::validation("重叠大小必须非负且小于分块大小"));
    }
    // 额外安全约束：限制最大重叠比例（避免步长接近1导致爆炸性分块）
    // 要求步长 >= max(64, chunk_size/4)
    let min_stride = std::cmp::max(64, (cfg.chunk_size / 4).max(1));
    let stride = cfg.chunk_size - cfg.chunk_overlap;
    if stride < min_stride {
        return Err(AppError::validation(format!(
            "重叠过大：当前步长{}，需>= {}（重叠<= {}）",
            stride,
            min_stride,
            cfg.chunk_size - min_stride
        )));
    }

    // 保存科目专属配置
    let json = serde_json::to_string(&cfg).map_err(|e| AppError::database(e.to_string()))?;
    state
        .notes_database
        .save_setting(&format!("notes.rag.config.{}", subject), &json)
        .map_err(|e| AppError::database(e.to_string()))?;

    // 同步覆盖 notes 数据库中的默认 rag_configurations，使后续嵌入过程生效
    state
        .notes_database
        .update_rag_configuration(&crate::models::RagConfigRequest {
            chunk_size: cfg.chunk_size,
            chunk_overlap: cfg.chunk_overlap,
            chunking_strategy: "fixed_size".to_string(),
            min_chunk_size: cfg.min_chunk_size,
            default_top_k: 5,
            default_rerank_enabled: cfg.rerank_enabled,
        })
        .map_err(|e| AppError::database(e.to_string()))?;
    Ok(true)
}
