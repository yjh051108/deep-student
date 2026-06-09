//! 会话管理命令处理器
//!
//! 包含创建、更新设置、归档、保存、列表、删除会话等命令。

use std::sync::Arc;

use tauri::State;

use crate::chat_v2::database::ChatV2Database;
use crate::chat_v2::error::ChatV2Error;
use crate::chat_v2::repo::ChatV2Repo;
use crate::chat_v2::state::ChatV2State;
use crate::chat_v2::types::{ChatSession, PersistStatus};
use crate::vfs::database::VfsDatabase;
use crate::vfs::repos::VfsResourceRepo;

fn session_has_running_anki_blocks(db: &ChatV2Database, session_id: &str) -> Result<bool, String> {
    let conn = db.get_conn_safe().map_err(|e| e.to_string())?;
    let count: i64 = conn
        .query_row(
            r#"
            SELECT COUNT(*)
            FROM chat_v2_blocks b
            INNER JOIN chat_v2_messages m ON m.id = b.message_id
            WHERE m.session_id = ?1
              AND b.block_type = 'anki_cards'
              AND b.status IN ('pending', 'running')
            "#,
            rusqlite::params![session_id],
            |row| row.get(0),
        )
        .map_err(|e| e.to_string())?;
    Ok(count > 0)
}

/// 🆕 2026-01-20: 列出 Agent 会话（Worker 会话）
///
/// 列出指定工作区的 Agent 会话，用于工作区面板显示。
///
/// ## 参数
/// - `workspace_id`: 可选的工作区 ID 过滤
/// - `limit`: 数量限制，默认 50
/// - `db`: Chat V2 独立数据库
///
/// ## 返回
/// - `Ok(Vec<ChatSession>)`: Agent 会话列表
/// - `Err(String)`: 查询失败
#[tauri::command]
pub async fn chat_v2_list_agent_sessions(
    workspace_id: Option<String>,
    limit: Option<u32>,
    db: State<'_, Arc<ChatV2Database>>,
) -> Result<Vec<ChatSession>, String> {
    log::info!(
        "[ChatV2::handlers] chat_v2_list_agent_sessions: workspace_id={:?}, limit={:?}",
        workspace_id,
        limit
    );

    let limit = limit.unwrap_or(50);

    let sessions = ChatV2Repo::list_agent_sessions_v2(&db, workspace_id.as_deref(), limit)
        .map_err(|e| e.to_string())?;

    log::info!(
        "[ChatV2::handlers] Listed {} agent sessions",
        sessions.len()
    );

    Ok(sessions)
}

/// P1-23: 软删除会话（移动到回收站）
///
/// 将会话标记为已删除状态，但不永久删除数据。可以恢复。
///
/// ## 参数
/// - `session_id`: 会话 ID
/// - `db`: Chat V2 独立数据库
///
/// ## 返回
/// - `Ok(())`: 软删除成功
/// - `Err(String)`: 软删除失败
#[tauri::command]
pub async fn chat_v2_soft_delete_session(
    session_id: String,
    db: State<'_, Arc<ChatV2Database>>,
    chat_v2_state: State<'_, Arc<ChatV2State>>,
) -> Result<(), String> {
    log::info!(
        "[ChatV2::handlers] chat_v2_soft_delete_session: session_id={}",
        session_id
    );

    // 验证会话 ID 格式
    if !session_id.starts_with("sess_")
        && !session_id.starts_with("agent_")
        && !session_id.starts_with("subagent_")
    {
        return Err(
            ChatV2Error::Validation(format!("Invalid session ID format: {}", session_id)).into(),
        );
    }

    // P0 修复：检查会话是否有活跃流，防止流式中删除导致 save_results 写入失败
    if chat_v2_state.has_active_stream(&session_id) {
        return Err(ChatV2Error::Other(
            "Cannot delete session while streaming. Please wait for completion or cancel first."
                .to_string(),
        )
        .into());
    }

    if session_has_running_anki_blocks(&db, &session_id)? {
        return Err(ChatV2Error::Other(
            "Cannot delete session while ChatAnki generation is still running. Please wait for completion or cancel first."
                .to_string(),
        )
        .into());
    }

    // 软删除会话
    soft_delete_session_in_db(&session_id, &db)?;

    log::info!("[ChatV2::handlers] Soft deleted session: id={}", session_id);

    Ok(())
}

/// P1-3: 清空回收站（永久删除所有已删除会话）
///
/// 一次性删除所有 persist_status = 'deleted' 的会话，
/// 解决前端逐个删除只能处理前 100 条的问题。
///
/// ★ 2026-02 修复：删除前先递减所有待删除会话中消息的 VFS 资源引用计数，
/// 防止 CASCADE DELETE 后引用计数永远无法归零导致资源孤儿。
///
/// ## 参数
/// - `db`: Chat V2 独立数据库
/// - `vfs_db`: VFS 数据库（用于资源引用计数递减）
///
/// ## 返回
/// - `Ok(u32)`: 被删除的会话数量
/// - `Err(String)`: 删除失败
#[tauri::command]
pub async fn chat_v2_empty_deleted_sessions(
    db: State<'_, Arc<ChatV2Database>>,
    vfs_db: State<'_, Arc<VfsDatabase>>,
) -> Result<u32, String> {
    log::info!("[ChatV2::handlers] chat_v2_empty_deleted_sessions");

    // ★ 先查出所有待删除的会话 ID，逐个收集资源引用并批量递减
    let deleted_ids = ChatV2Repo::list_deleted_session_ids(&db).map_err(|e| e.to_string())?;

    if !deleted_ids.is_empty() {
        // 收集所有待删除会话中消息引用的资源 ID（不去重，与递增时对称）
        let mut all_resource_ids: Vec<String> = Vec::new();
        for sid in &deleted_ids {
            if let Ok(messages) = ChatV2Repo::get_session_messages_v2(&db, sid) {
                for msg in &messages {
                    if let Some(ref meta) = msg.meta {
                        if let Some(ref context_snapshot) = meta.context_snapshot {
                            let ids = context_snapshot.all_resource_ids();
                            all_resource_ids.extend(ids.into_iter().map(|s| s.to_string()));
                        }
                    }
                }
            }
        }

        // 批量递减 VFS 资源引用计数（失败仅告警，不阻塞删除）
        if !all_resource_ids.is_empty() {
            match vfs_db.get_conn_safe() {
                Ok(vfs_conn) => {
                    if let Err(e) =
                        VfsResourceRepo::decrement_refs_with_conn(&vfs_conn, &all_resource_ids)
                    {
                        log::warn!(
                            "[ChatV2::handlers] Failed to decrement refs during trash empty: {}",
                            e
                        );
                    } else {
                        log::debug!(
                            "[ChatV2::handlers] Decremented refs for {} resource references before emptying trash ({} sessions)",
                            all_resource_ids.len(),
                            deleted_ids.len()
                        );
                    }
                }
                Err(e) => {
                    log::warn!(
                        "[ChatV2::handlers] Failed to get vfs.db conn for trash empty ref decrement: {}",
                        e
                    );
                }
            }
        }
    }

    // 执行批量硬删除
    let count = ChatV2Repo::purge_deleted_sessions(&db).map_err(|e| e.to_string())?;
    log::info!(
        "[ChatV2::handlers] Emptied trash: {} sessions permanently deleted",
        count
    );
    Ok(count)
}

/// 获取指定会话的消息数量
///
/// 轻量级查询，用于前端判断会话是否为空（无消息）。
///
/// ## 参数
/// - `session_id`: 会话 ID
///
/// ## 返回
/// - `Ok(u32)`: 消息数量
/// - `Err(String)`: 查询失败
#[tauri::command]
pub async fn chat_v2_session_message_count(
    session_id: String,
    db: State<'_, Arc<ChatV2Database>>,
) -> Result<u32, String> {
    let conn = db.get_conn_safe().map_err(|e| e.to_string())?;
    let count: u32 = conn
        .query_row(
            "SELECT COUNT(*) FROM chat_v2_messages WHERE session_id = ?1",
            [&session_id],
            |row| row.get(0),
        )
        .map_err(|e| format!("Failed to count messages for session {}: {}", session_id, e))?;
    Ok(count)
}

// ============================================================================
// 内部辅助函数（调用 ChatV2Repo 实现）
// ============================================================================

/// P1-23: 软删除会话
fn soft_delete_session_in_db(session_id: &str, db: &ChatV2Database) -> Result<(), ChatV2Error> {
    // 先获取现有会话
    let existing = ChatV2Repo::get_session_v2(db, session_id)?
        .ok_or_else(|| ChatV2Error::SessionNotFound(session_id.to_string()))?;

    let now = chrono::Utc::now();

    // 构建软删除后的会话
    let deleted_session = ChatSession {
        id: existing.id,
        mode: existing.mode,
        title: existing.title,
        description: existing.description,
        summary_hash: existing.summary_hash,
        title_locked: existing.title_locked,
        persist_status: PersistStatus::Deleted,
        created_at: existing.created_at,
        updated_at: now,
        metadata: existing.metadata,
        group_id: existing.group_id,
        tags_hash: existing.tags_hash,
        tags: None,
    };

    // 更新数据库
    ChatV2Repo::update_session_v2(db, &deleted_session)?;

    Ok(())
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn test_valid_modes() {
        let valid_modes = [
            "chat", // 前端标准聊天模式
            "analysis",
            "review",
            "textbook",
            "bridge",
            "general_chat",
        ];

        for mode in valid_modes.iter() {
            assert!(valid_modes.contains(mode));
        }

        assert!(!valid_modes.contains(&"invalid_mode"));
    }

    #[test]
    fn test_session_id_generation() {
        let id1 = ChatSession::generate_id();
        let id2 = ChatSession::generate_id();

        assert!(id1.starts_with("sess_"));
        assert!(id2.starts_with("sess_"));
        assert_ne!(id1, id2);
    }

    #[test]
    fn test_session_id_format_validation() {
        // 有效的会话 ID
        assert!("sess_12345".starts_with("sess_"));
        assert!("sess_a1b2c3d4-e5f6-7890-abcd-ef1234567890".starts_with("sess_"));

        // 无效的会话 ID
        assert!(!"session_12345".starts_with("sess_"));
        assert!(!"invalid".starts_with("sess_"));
    }
}
