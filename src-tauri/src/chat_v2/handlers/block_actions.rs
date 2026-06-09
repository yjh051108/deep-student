//! 块操作命令处理器
//!
//! 包含删除消息和复制块内容等命令。

use std::sync::Arc;

use serde::{Deserialize, Serialize};
use tauri::{AppHandle, State};

use crate::chat_v2::database::ChatV2Database;
use crate::chat_v2::error::ChatV2Error;
use crate::chat_v2::events::{event_phase, event_types, next_session_sequence_id};
use crate::chat_v2::repo::ChatV2Repo;

/// 复制块内容响应
#[derive(Debug, Clone, Serialize, Deserialize)]
#[serde(rename_all = "camelCase")]
pub struct CopyBlockContentResponse {
    /// 复制的内容
    pub content: String,
    /// 内容类型（text/markdown/json）
    pub content_type: String,
}

/// 复制块内容
///
/// 获取指定块的内容，用于复制到剪贴板。
/// 根据块类型返回不同格式的内容。
///
/// ## 参数
/// - `block_id`: 块 ID
/// - `format`: 可选的输出格式（text/markdown/json），默认为 text
/// - `db`: Chat V2 独立数据库
///
/// ## 返回
/// - `Ok(CopyBlockContentResponse)`: 块内容和格式
/// - `Err(String)`: 块不存在或读取失败
///
/// ## 格式说明
/// - `text`: 纯文本格式，适合粘贴到普通文本框
/// - `markdown`: Markdown 格式，保留格式信息
/// - `json`: JSON 格式，包含完整块数据
#[tauri::command]
pub async fn chat_v2_copy_block_content(
    block_id: String,
    format: Option<String>,
    db: State<'_, Arc<ChatV2Database>>,
) -> Result<CopyBlockContentResponse, String> {
    log::info!(
        "[ChatV2::handlers] chat_v2_copy_block_content: block_id={}, format={:?}",
        block_id,
        format
    );

    // 验证块 ID 格式
    if !block_id.starts_with("blk_") {
        return Err(
            ChatV2Error::Validation(format!("Invalid block ID format: {}", block_id)).into(),
        );
    }

    let output_format = format.unwrap_or_else(|| "text".to_string());

    // 获取块内容
    let response = get_block_content_from_db(&block_id, &output_format, &db)?;

    log::info!(
        "[ChatV2::handlers] Copied block content: block_id={}, content_type={}, len={}",
        block_id,
        response.content_type,
        response.content.len()
    );

    Ok(response)
}

// ============================================================================
// 内部辅助函数（调用 ChatV2Repo 实现）
// ============================================================================

/// 从数据库获取块内容
fn get_block_content_from_db(
    block_id: &str,
    format: &str,
    db: &ChatV2Database,
) -> Result<CopyBlockContentResponse, ChatV2Error> {
    // 从数据库获取块
    let block = ChatV2Repo::get_block_v2(db, block_id)?
        .ok_or_else(|| ChatV2Error::BlockNotFound(block_id.to_string()))?;

    // 获取块内容（如果为空则使用默认值）
    let block_content = block.content.unwrap_or_default();

    // 根据格式生成输出
    let (content, content_type) = match format {
        "markdown" => {
            // 返回 Markdown 格式
            (block_content, "markdown".to_string())
        }
        "json" => {
            // 返回 JSON 格式（包含完整块数据）
            let json = serde_json::json!({
                "id": block.id,
                "type": block.block_type,
                "status": block.status,
                "content": block_content,
                "toolName": block.tool_name,
                "toolInput": block.tool_input,
                "toolOutput": block.tool_output,
                "citations": block.citations,
                "error": block.error,
                "startedAt": block.started_at,
                "endedAt": block.ended_at,
            });
            (
                serde_json::to_string_pretty(&json).unwrap_or_default(),
                "json".to_string(),
            )
        }
        _ => {
            // 默认返回纯文本
            (block_content, "text".to_string())
        }
    };

    Ok(CopyBlockContentResponse {
        content,
        content_type,
    })
}

/// 更新块的 tool_output（用于前端编辑 anki_cards 卡片后持久化）
///
/// 🔧 修复场景8：前端编辑卡片后调用此命令持久化到数据库，
/// 防止后续 pipeline 重保存消息时丢失用户编辑。
#[tauri::command]
pub async fn chat_v2_update_block_tool_output(
    block_id: String,
    tool_output_json: String,
    db: State<'_, Arc<ChatV2Database>>,
) -> Result<(), String> {
    log::info!(
        "[ChatV2::handlers] chat_v2_update_block_tool_output: block_id={}, len={}",
        block_id,
        tool_output_json.len()
    );

    if !block_id.starts_with("blk_") {
        return Err(
            ChatV2Error::Validation(format!("Invalid block ID format: {}", block_id)).into(),
        );
    }

    // 验证 JSON 合法性
    let _: serde_json::Value = serde_json::from_str(&tool_output_json)
        .map_err(|e| format!("Invalid tool_output_json: {}", e))?;

    let conn = db.get_conn_safe().map_err(|e| e.to_string())?;
    conn.execute(
        "UPDATE chat_v2_blocks SET tool_output_json = ?1 WHERE id = ?2",
        rusqlite::params![tool_output_json, block_id],
    )
    .map_err(|e| format!("Failed to update block tool_output: {}", e))?;

    log::info!(
        "[ChatV2::handlers] Block tool_output updated: block_id={}",
        block_id
    );

    Ok(())
}

/// 根据 document_id 获取聊天块中持久化的 anki_cards（优先返回前端编辑后的版本）
#[tauri::command]
#[allow(non_snake_case)]
pub async fn chat_v2_get_anki_cards_from_block_by_document_id(
    documentId: String,
    db: State<'_, Arc<ChatV2Database>>,
) -> Result<Vec<crate::models::AnkiCard>, String> {
    let doc_id = documentId.trim();
    if doc_id.is_empty() {
        return Err("documentId is required".to_string());
    }

    let conn = db.get_conn_safe().map_err(|e| e.to_string())?;
    let mut stmt = conn
        .prepare(
            r#"
            SELECT tool_output_json
            FROM chat_v2_blocks
            WHERE block_type = 'anki_cards' AND tool_output_json IS NOT NULL
            ORDER BY rowid DESC
            "#,
        )
        .map_err(|e| format!("Failed to prepare query: {}", e))?;

    let rows = stmt
        .query_map([], |row| row.get::<_, String>(0))
        .map_err(|e| format!("Failed to query blocks: {}", e))?;

    for row in rows {
        let tool_output_json = row.map_err(|e| format!("Failed to read row: {}", e))?;
        let parsed: serde_json::Value = match serde_json::from_str(&tool_output_json) {
            Ok(value) => value,
            Err(_) => continue,
        };

        let block_doc_id = parsed
            .get("documentId")
            .and_then(|value| value.as_str())
            .unwrap_or_default();
        if block_doc_id != doc_id {
            continue;
        }

        let cards = parsed
            .get("cards")
            .and_then(|value| value.as_array())
            .cloned()
            .unwrap_or_default();

        let converted = cards
            .into_iter()
            .filter_map(|value| serde_json::from_value::<crate::models::AnkiCard>(value).ok())
            .collect::<Vec<_>>();

        return Ok(converted);
    }

    Ok(Vec::new())
}

/// 🔧 P35 批判性修复：追加块 ID 到消息的 block_ids_json
///
/// 如果消息存在，追加 block_id；如果消息不存在，忽略（流式块场景）
fn append_block_id_to_message(
    conn: &rusqlite::Connection,
    message_id: &str,
    block_id: &str,
) -> Result<(), ChatV2Error> {
    // 尝试读取现有的 block_ids
    let existing_block_ids: Result<Option<String>, _> = conn.query_row(
        "SELECT block_ids_json FROM chat_v2_messages WHERE id = ?1",
        rusqlite::params![message_id],
        |row| row.get(0),
    );

    match existing_block_ids {
        Ok(block_ids_json) => {
            // 消息存在，追加 block_id
            let mut block_ids: Vec<String> = block_ids_json
                .and_then(|s| serde_json::from_str(&s).ok())
                .unwrap_or_default();

            // 避免重复添加
            if !block_ids.contains(&block_id.to_string()) {
                block_ids.push(block_id.to_string());

                let block_ids_json = serde_json::to_string(&block_ids)?;

                conn.execute(
                    "UPDATE chat_v2_messages SET block_ids_json = ?1 WHERE id = ?2",
                    rusqlite::params![block_ids_json, message_id],
                )?;

                log::info!(
                    "[ChatV2::handlers] ✅ Appended block_id {} to message {}, new_block_ids={}",
                    block_id,
                    message_id,
                    block_ids_json
                );
            }
        }
        Err(rusqlite::Error::QueryReturnedNoRows) => {
            // 消息不存在，忽略（流式块场景，消息稍后会创建）
            log::warn!(
                "[ChatV2::handlers] ⚠️ Message {} not found, skipping block_ids update for block {}",
                message_id, block_id
            );
        }
        Err(e) => {
            log::warn!(
                "[ChatV2::handlers] Failed to read message {}: {}",
                message_id,
                e
            );
        }
    }

    Ok(())
}

/// 在数据库中 UPSERT 块（防闪退保存专用）
///
/// 🔧 关键设计：临时禁用外键约束
///
/// 流式过程中，助手消息还未保存到数据库，但我们需要先保存块内容以防闪退。
/// 正常流式结束后，`save_results` 会保存完整的消息和块，覆盖这里的临时数据。
///
/// 如果闪退：
/// - 块数据已保存，可恢复部分内容
/// - 消息数据缺失，需要在恢复时处理孤儿块
fn upsert_block_in_db(
    block: &crate::chat_v2::types::MessageBlock,
    db: &ChatV2Database,
) -> Result<(), ChatV2Error> {
    let conn = db.get_conn_safe()?;

    let tool_input_json = block
        .tool_input
        .as_ref()
        .map(|v| serde_json::to_string(v))
        .transpose()?;
    let tool_output_json = block
        .tool_output
        .as_ref()
        .map(|v| serde_json::to_string(v))
        .transpose()?;
    let citations_json = block
        .citations
        .as_ref()
        .map(|v| serde_json::to_string(v))
        .transpose()?;

    conn.execute(
        r#"
        INSERT INTO chat_v2_blocks
        (id, message_id, block_type, status, block_index, content, tool_name, tool_input_json, tool_output_json, citations_json, error, started_at, ended_at, first_chunk_at)
        VALUES (?1, ?2, ?3, ?4, ?5, ?6, ?7, ?8, ?9, ?10, ?11, ?12, ?13, ?14)
        ON CONFLICT(id) DO UPDATE SET
            message_id = excluded.message_id,
            block_type = excluded.block_type,
            status = excluded.status,
            block_index = excluded.block_index,
            content = excluded.content,
            tool_name = excluded.tool_name,
            tool_input_json = excluded.tool_input_json,
            tool_output_json = excluded.tool_output_json,
            citations_json = excluded.citations_json,
            error = excluded.error,
            started_at = excluded.started_at,
            ended_at = excluded.ended_at,
            first_chunk_at = excluded.first_chunk_at
        "#,
        rusqlite::params![
            block.id,
            block.message_id,
            block.block_type,
            block.status,
            block.block_index,
            block.content,
            block.tool_name,
            tool_input_json,
            tool_output_json,
            citations_json,
            block.error,
            block.started_at,
            block.ended_at,
            block.first_chunk_at,
        ],
    )?;

    // 🔧 P35 批判性修复：更新消息的 block_ids_json，确保块被正确关联
    // 如果不更新，刷新后加载消息时 block_ids_json 中没有这个块 ID，块不会被渲染
    append_block_id_to_message(&conn, &block.message_id, &block.id)?;

    Ok(())
}

// ============================================================================
// Anki 卡片结果处理（CardAgent 回调）
// ============================================================================

/// Anki 卡片结果请求
#[derive(Debug, Clone, Serialize, Deserialize)]
#[serde(rename_all = "camelCase")]
pub struct AnkiCardsResultRequest {
    /// 会话 ID
    pub session_id: String,
    /// 消息 ID（来自工具调用时传递的 messageId）
    pub message_id: String,
    /// 块 ID（来自工具调用时传递的 blockId，将被替换为新的 anki_cards 块）
    pub tool_block_id: String,
    /// 生成的卡片列表
    pub cards: Vec<serde_json::Value>,
    /// 文档 ID（用于后续查询进度）
    pub document_id: Option<String>,
    /// 模板 ID
    pub template_id: Option<String>,
    /// 是否成功
    pub success: bool,
    /// 错误信息（失败时）
    pub error: Option<String>,
}

/// 接收 Anki 卡片生成结果
///
/// 由前端 CardAgent 在完成卡片生成后调用，用于：
/// 1. 创建 anki_cards 块显示在聊天中
/// 2. 持久化卡片数据到数据库
/// 3. 发射事件通知前端 UI 更新
///
/// ## 参数
/// - `request`: Anki 卡片结果请求
/// - `db`: Chat V2 独立数据库
/// - `app`: Tauri AppHandle（用于发射事件）
///
/// ## 返回
/// - `Ok(String)`: 创建的 anki_cards 块 ID
/// - `Err(String)`: 创建失败
#[tauri::command]
pub async fn chat_v2_anki_cards_result(
    request: AnkiCardsResultRequest,
    db: State<'_, Arc<ChatV2Database>>,
    app: AppHandle,
) -> Result<String, String> {
    use tauri::Emitter;

    log::info!(
        "[ChatV2::handlers] chat_v2_anki_cards_result: session_id={}, message_id={}, cards_count={}, success={}",
        request.session_id,
        request.message_id,
        request.cards.len(),
        request.success
    );

    // 验证消息 ID 格式
    if !request.message_id.starts_with("msg_") {
        return Err(ChatV2Error::Validation(format!(
            "Invalid message ID format: {}",
            request.message_id
        ))
        .into());
    }

    // 生成新的 anki_cards 块 ID
    let block_id = format!("blk_{}", uuid::Uuid::new_v4());
    let now_ms = chrono::Utc::now().timestamp_millis();

    // 构建 toolOutput（与前端 AnkiCardsBlockData 兼容）
    let tool_output = serde_json::json!({
        "cards": request.cards,
        "documentId": request.document_id,
        "templateId": request.template_id,
        "syncStatus": "pending",
        "businessSessionId": request.session_id,
        "messageStableId": request.message_id,
    });

    // 确定块状态
    let status = if request.success {
        crate::chat_v2::types::block_status::SUCCESS.to_string()
    } else {
        crate::chat_v2::types::block_status::ERROR.to_string()
    };

    // 构建 anki_cards 块
    let block = crate::chat_v2::types::MessageBlock {
        id: block_id.clone(),
        message_id: request.message_id.clone(),
        block_type: crate::chat_v2::types::block_types::ANKI_CARDS.to_string(),
        status: status.clone(),
        content: None,
        tool_name: Some("anki_generate_cards".to_string()),
        tool_input: None,
        tool_output: Some(tool_output.clone()),
        citations: None,
        error: request.error.clone(),
        started_at: Some(now_ms),
        ended_at: Some(now_ms),
        first_chunk_at: Some(now_ms),
        block_index: 1, // 放在 mcp_tool 块之后
    };

    // 保存到数据库
    upsert_block_in_db(&block, &db).map_err(|e| e.to_string())?;

    // 🆕 2026-01: 发射 anki_cards 事件到前端，通知 UI 更新
    // 使用会话特定的事件通道
    let event_channel = format!("chat_v2_event_{}", request.session_id);

    let start_sequence_id = next_session_sequence_id(&request.session_id);
    // 发射 start 事件
    let start_event = serde_json::json!({
        "sequenceId": start_sequence_id,
        "type": event_types::ANKI_CARDS,
        "phase": event_phase::START,
        "messageId": request.message_id,
        "blockId": block_id,
        "payload": {
            "templateId": request.template_id,
        },
    });
    if let Err(e) = app.emit(&event_channel, &start_event) {
        log::warn!(
            "[ChatV2::handlers] Failed to emit anki_cards start event: {}",
            e
        );
    }

    let end_sequence_id = next_session_sequence_id(&request.session_id);
    // 发射 end 事件（带完整卡片数据）
    let end_event = serde_json::json!({
        "sequenceId": end_sequence_id,
        "type": event_types::ANKI_CARDS,
        "phase": event_phase::END,
        "blockId": block_id,
        "result": tool_output,
        "status": status,
        "error": request.error,
    });
    if let Err(e) = app.emit(&event_channel, &end_event) {
        log::warn!(
            "[ChatV2::handlers] Failed to emit anki_cards end event: {}",
            e
        );
    }

    log::info!(
        "[ChatV2::handlers] Anki cards block created and event emitted: block_id={}, cards_count={}",
        block_id,
        request.cards.len()
    );

    Ok(block_id)
}

#[cfg(test)]
mod tests {
    #[test]
    fn test_block_id_validation() {
        assert!("blk_12345".starts_with("blk_"));
        assert!("blk_a1b2c3d4-e5f6-7890-abcd-ef1234567890".starts_with("blk_"));
        assert!(!"block_12345".starts_with("blk_"));
        assert!(!"invalid".starts_with("blk_"));
    }

    #[test]
    fn test_message_id_validation() {
        assert!("msg_12345".starts_with("msg_"));
        assert!("msg_a1b2c3d4-e5f6-7890-abcd-ef1234567890".starts_with("msg_"));
        assert!(!"message_12345".starts_with("msg_"));
        assert!(!"invalid".starts_with("msg_"));
    }
}
