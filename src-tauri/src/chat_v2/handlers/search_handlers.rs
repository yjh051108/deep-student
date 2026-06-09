//! 内容搜索与标签管理命令处理器

use std::sync::Arc;
use tauri::State;

use crate::chat_v2::database::ChatV2Database;
use crate::chat_v2::repo::ChatV2Repo;
use crate::chat_v2::types::ContentSearchResult;

/// 搜索消息内容（FTS5 全文搜索）
#[tauri::command]
pub async fn chat_v2_search_content(
    query: String,
    limit: Option<u32>,
    db: State<'_, Arc<ChatV2Database>>,
) -> Result<Vec<ContentSearchResult>, String> {
    let limit = limit.unwrap_or(50).min(200);
    let conn = db.get_conn_safe().map_err(|e| e.to_string())?;
    ChatV2Repo::search_content(&conn, &query, limit).map_err(|e| e.to_string())
}
