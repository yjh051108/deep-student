//! 工具审批 Tauri 命令处理器
//!
//! 保留尚未退役的工具审批取消 Tauri 命令。
//!
//! ## 设计文档
//! 参考：`src/chat-v2/docs/29-ChatV2-Agent能力增强改造方案.md` 第 4.7 节

use std::sync::Arc;
use tauri::State;

use crate::chat_v2::approval_manager::ApprovalManager;

// ============================================================================
// Tauri 命令
// ============================================================================

/// 取消工具审批请求
///
/// 当用户切换会话或关闭对话框时调用，清理未响应的审批请求。
///
/// ## 参数
/// - `tool_call_id`: 工具调用 ID
#[tauri::command]
pub async fn chat_v2_tool_approval_cancel(
    approval_manager: State<'_, Arc<ApprovalManager>>,
    tool_call_id: String,
) -> Result<(), String> {
    log::info!(
        "[ChatV2::approval] Cancelling approval request: tool_call_id={}",
        tool_call_id
    );

    approval_manager.cancel(&tool_call_id);
    Ok(())
}
