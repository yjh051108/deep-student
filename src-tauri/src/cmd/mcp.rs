//! MCP related commands.
//!
//! This module now keeps only legacy config persistence. Settings diagnostics
//! and stdio process proxying moved to frontend MCP SDK testers plus the
//! Go/Wails stdio proxy.

use crate::commands::AppState;
use crate::models::AppError;
use tauri::State;

type Result<T> = std::result::Result<T, AppError>;

#[tauri::command]
pub async fn save_mcp_config(
    config: serde_json::Value,
    state: State<'_, AppState>,
) -> Result<bool> {
    let db = &state.database;

    if let Some(transport) = config.get("transport") {
        if let Some(transport_type) = transport.get("type").and_then(|v| v.as_str()) {
            db.save_setting("mcp.transport.type", transport_type)?;

            match transport_type {
                "stdio" => {
                    if let Some(command) = transport.get("command").and_then(|v| v.as_str()) {
                        db.save_setting("mcp.transport.command", command)?;
                    }
                    if let Some(args) = transport.get("args").and_then(|v| v.as_array()) {
                        let args_str = args
                            .iter()
                            .filter_map(|v| v.as_str())
                            .collect::<Vec<_>>()
                            .join(",");
                        db.save_setting("mcp.transport.args", &args_str)?;
                    }
                    if let Some(framing) = transport.get("framing").and_then(|v| v.as_str()) {
                        db.save_setting("mcp.transport.framing", framing)?;
                    }
                }
                "websocket" => {
                    if let Some(url) = transport.get("url").and_then(|v| v.as_str()) {
                        db.save_setting("mcp.transport.url", url)?;
                    }
                }
                _ => {}
            }
        }
    }

    if let Some(tools) = config.get("tools") {
        if let Some(cache_ttl_ms) = tools.get("cache_ttl_ms").and_then(|v| v.as_u64()) {
            db.save_setting("mcp.tools.cache_ttl_ms", &cache_ttl_ms.to_string())?;
        }
        if let Some(advertise_all) = tools.get("advertise_all_tools").and_then(|v| v.as_bool()) {
            db.save_setting("mcp.tools.advertise_all_tools", &advertise_all.to_string())?;
        }
        if let Some(whitelist) = tools.get("whitelist").and_then(|v| v.as_array()) {
            let whitelist_str = whitelist
                .iter()
                .filter_map(|v| v.as_str())
                .collect::<Vec<_>>()
                .join(",");
            db.save_setting("mcp.tools.whitelist", &whitelist_str)?;
        }
        if let Some(blacklist) = tools.get("blacklist").and_then(|v| v.as_array()) {
            let blacklist_str = blacklist
                .iter()
                .filter_map(|v| v.as_str())
                .collect::<Vec<_>>()
                .join(",");
            db.save_setting("mcp.tools.blacklist", &blacklist_str)?;
        }
    }

    if let Some(performance) = config.get("performance") {
        if let Some(timeout_ms) = performance.get("timeout_ms").and_then(|v| v.as_u64()) {
            db.save_setting("mcp.performance.timeout_ms", &timeout_ms.to_string())?;
        }
        if let Some(rate_limit) = performance
            .get("rate_limit_per_second")
            .and_then(|v| v.as_u64())
        {
            db.save_setting(
                "mcp.performance.rate_limit_per_second",
                &rate_limit.to_string(),
            )?;
        }
        if let Some(cache_max_size) = performance.get("cache_max_size").and_then(|v| v.as_u64()) {
            db.save_setting(
                "mcp.performance.cache_max_size",
                &cache_max_size.to_string(),
            )?;
        }
        if let Some(cache_ttl_ms) = performance.get("cache_ttl_ms").and_then(|v| v.as_u64()) {
            db.save_setting("mcp.performance.cache_ttl_ms", &cache_ttl_ms.to_string())?;
        }
    }

    Ok(true)
}
