//! Web Search 连通性测试命令
//!
//! 从 commands.rs 拆分：搜索引擎连接测试

use crate::commands::AppState;
use crate::models::AppError;
use crate::tools::ToolConflict;
use tauri::State;

type Result<T> = std::result::Result<T, AppError>;

/// 检查安全存储状态（缓存版本，避免频繁的钥匙串访问）
#[tauri::command]
pub async fn get_security_status(_state: State<'_, AppState>) -> Result<serde_json::Value> {
    let migration_completed = true;

    // 🚨 钥匙串功能已彻底禁用，移除所有相关代码

    // 🚨 钥匙串功能已彻底禁用，直接设置为false
    let keychain_available = false;

    Ok(serde_json::json!({
        "keychain_available": keychain_available,
        "migration_completed": migration_completed,
        "sensitive_keys_count": 0, // 可以在这里添加计数逻辑
        "last_migration_time": null, // 可以添加上次迁移时间
        "warnings": vec!["🚨 钥匙串功能已彻底禁用以避免密码弹窗，敏感数据使用加密数据库存储"],
        "sensitive_key_patterns": [
            "web_search.api_key.*",
            "web_search.searxng.api_key",
            "api_configs",
            "mcp.transport.*"
        ],
        "timestamp": chrono::Utc::now().to_rfc3339()
    }))
}
/// 检测工具名冲突
#[tauri::command]
pub async fn detect_tool_conflicts(_state: State<'_, AppState>) -> Result<Vec<ToolConflict>> {
    // 后端 MCP 已禁用，暂不检测冲突（由前端SDK命名空间解决）
    Ok(vec![])
}

/// 获取工具命名空间配置
#[tauri::command]
pub async fn get_tools_namespace_config(state: State<'_, AppState>) -> Result<serde_json::Value> {
    let db = &state.database;

    let namespace_prefix = db.get_setting("mcp.tools.namespace_prefix").unwrap_or(None);

    let conflict_resolution = db
        .get_setting("mcp.tools.conflict_resolution")
        .unwrap_or(None)
        .unwrap_or_else(|| "use_local".to_string());

    Ok(serde_json::json!({
        "namespace_prefix": namespace_prefix,
        "conflict_resolution": conflict_resolution,
        "available_resolutions": [
            {"value": "use_local", "label": "优先使用本地工具"},
            {"value": "use_mcp", "label": "优先使用MCP工具"},
            {"value": "use_namespace", "label": "使用命名空间前缀"}
        ],
        "config_keys": {
            "namespace_prefix": "mcp.tools.namespace_prefix",
            "conflict_resolution": "mcp.tools.conflict_resolution"
        }
    }))
}

/// 获取功能开关配置
#[tauri::command]
pub async fn get_feature_flags(state: State<'_, AppState>) -> Result<serde_json::Value> {
    use crate::feature_flags::FeatureFlagManager;

    // 获取应用版本（可以从配置或环境变量中读取）
    let app_version = env!("CARGO_PKG_VERSION").to_string();

    // 创建功能开关管理器并从数据库加载
    let manager = FeatureFlagManager::new(app_version)
        .load_from_database(&state.database)
        .await
        .map_err(|e| format!("加载功能开关失败: {}", e))?;

    let all_flags = manager.list_all_flags();
    let flags_by_category: std::collections::HashMap<
        String,
        Vec<&crate::feature_flags::FeatureFlag>,
    > = {
        let mut map = std::collections::HashMap::new();
        for flag in &all_flags {
            map.entry(flag.category.clone())
                .or_insert_with(Vec::new)
                .push(*flag);
        }
        map
    };

    Ok(serde_json::json!({
        "flags": all_flags,
        "flags_by_category": flags_by_category,
        "total_count": all_flags.len()
    }))
}
/// 更新功能开关状态
#[tauri::command]
pub async fn update_feature_flag(
    state: State<'_, AppState>,
    feature_name: String,
    action: String,
    value: Option<serde_json::Value>,
) -> Result<serde_json::Value> {
    use crate::feature_flags::FeatureFlagManager;

    let app_version = env!("CARGO_PKG_VERSION").to_string();
    let mut manager = FeatureFlagManager::new(app_version)
        .load_from_database(&state.database)
        .await
        .map_err(|e| format!("加载功能开关失败: {}", e))?;

    match action.as_str() {
        "enable" => {
            manager
                .enable_feature(&feature_name)
                .map_err(|e| format!("启用功能失败: {}", e))?;
        }
        "disable" => {
            manager
                .disable_feature(&feature_name)
                .map_err(|e| format!("禁用功能失败: {}", e))?;
        }
        "set_gradual" => {
            let percentage = value
                .and_then(|v| v.as_f64())
                .ok_or("渐进发布需要提供百分比参数")? as f32;
            manager
                .set_gradual_rollout(&feature_name, percentage)
                .map_err(|e| format!("设置渐进发布失败: {}", e))?;
        }
        _ => {
            return Err(format!("不支持的操作: {}", action).into());
        }
    }

    // 保存更新后的配置
    manager
        .save_to_database(&state.database)
        .await
        .map_err(|e| format!("保存功能开关失败: {}", e))?;

    Ok(serde_json::json!({
        "success": true,
        "message": format!("功能 '{}' 已成功{}", feature_name, match action.as_str() {
            "enable" => "启用",
            "disable" => "禁用",
            "set_gradual" => "设置渐进发布",
            _ => "更新"
        })
    }))
}

/// 检查功能是否启用
#[tauri::command]
pub async fn is_feature_enabled(
    state: State<'_, AppState>,
    feature_name: String,
    user_id: Option<String>,
) -> Result<bool> {
    use crate::feature_flags::FeatureFlagManager;

    let app_version = env!("CARGO_PKG_VERSION").to_string();
    let mut manager = FeatureFlagManager::new(app_version);

    if let Some(uid) = user_id {
        manager = manager.with_user_id(uid);
    }

    let manager = manager
        .load_from_database(&state.database)
        .await
        .map_err(|e| format!("加载功能开关失败: {}", e))?;

    Ok(manager.is_feature_enabled(&feature_name))
}

/// 按前缀批量删除设置
#[tauri::command]
pub async fn delete_settings_by_prefix(
    prefix: String,
    state: State<'_, AppState>,
) -> Result<usize> {
    let db = &state.database;
    db.delete_settings_by_prefix(&prefix)
        .map_err(|e| AppError::database(format!("按前缀批量删除设置失败: {}", e)))
}
