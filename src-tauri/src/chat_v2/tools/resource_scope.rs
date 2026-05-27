use std::collections::HashSet;
use std::sync::Arc;

use serde_json::Value;
use tauri::Emitter;

use super::executor::ExecutionContext;
use crate::chat_v2::repo::ChatV2Repo;
use crate::chat_v2::types::PersistStatus;
use crate::vfs::{VfsDatabase, VfsFolderRepo, VfsResourceRepo};

pub const NO_TOPIC_RESOURCE_FOLDER_SENTINEL: &str = "__no_topic_resource_folder__";
const SESSION_MGMT_EVENT: &str = "session_management_change";

#[derive(Debug, Clone)]
struct GroupScope {
    id: String,
    pinned_resource_ids: Vec<String>,
}

fn normalize_folder_ids(ids: impl IntoIterator<Item = String>) -> Vec<String> {
    ids.into_iter()
        .filter_map(|id| {
            let trimmed = id.trim();
            if trimmed.starts_with("fld_") {
                Some(trimmed.to_string())
            } else {
                None
            }
        })
        .collect()
}

fn has_existing_folder(vfs_db: &Arc<VfsDatabase>, folder_id: &str) -> bool {
    VfsFolderRepo::get_folder(vfs_db, folder_id)
        .ok()
        .flatten()
        .is_some()
}

fn has_existing_topic_folder(vfs_db: &Arc<VfsDatabase>, pinned_ids: &[String]) -> bool {
    normalize_folder_ids(pinned_ids.iter().cloned())
        .iter()
        .any(|folder_id| has_existing_folder(vfs_db, folder_id))
}

fn emit_scope_repaired(ctx: &ExecutionContext, group_id: &str) {
    if let Err(e) = ctx.window.emit(
        SESSION_MGMT_EVENT,
        serde_json::json!({
            "type": "group_resource_scope_repaired",
            "groupId": group_id,
            "sessionId": ctx.session_id,
        }),
    ) {
        log::debug!(
            "[ResourceScope] Failed to emit {} after repairing group {}: {}",
            SESSION_MGMT_EVENT,
            group_id,
            e
        );
    }
}

fn repair_group_scope(ctx: &ExecutionContext, group_id: &str) -> Option<GroupScope> {
    let chat_db = ctx.chat_v2_db.as_ref()?;
    let conn = chat_db.get_conn_safe().ok()?;
    let mut group = ChatV2Repo::get_group_with_conn(&conn, group_id)
        .ok()
        .flatten()?;
    if group.persist_status != PersistStatus::Active {
        return None;
    }

    if let Some(vfs_db) = ctx.vfs_db.as_ref() {
        match crate::chat_v2::handlers::group_handlers::ensure_group_folder(
            vfs_db.as_ref(),
            &group.name,
            group.pinned_resource_ids.clone(),
        ) {
            Ok(next_pinned) => {
                let root_was_missing =
                    !has_existing_topic_folder(vfs_db, &group.pinned_resource_ids);
                if next_pinned != group.pinned_resource_ids || root_was_missing {
                    group.pinned_resource_ids = next_pinned;
                    group.updated_at = chrono::Utc::now();
                    if let Err(e) = ChatV2Repo::update_group_with_conn(&conn, &group) {
                        log::warn!(
                            "[ResourceScope] Failed to persist repaired resource scope for group {}: {}",
                            group.id,
                            e
                        );
                    } else {
                        emit_scope_repaired(ctx, &group.id);
                        log::info!(
                            "[ResourceScope] Repaired topic resource root for group {} ({})",
                            group.id,
                            group.name
                        );
                    }
                }
            }
            Err(e) => {
                log::warn!(
                    "[ResourceScope] Failed to repair topic resource root for group {}: {}",
                    group_id,
                    e
                );
            }
        }
    }

    Some(GroupScope {
        id: group.id,
        pinned_resource_ids: group.pinned_resource_ids,
    })
}

fn load_group_id_from_session(ctx: &ExecutionContext) -> Option<String> {
    let chat_db = ctx.chat_v2_db.as_ref()?;
    let conn = chat_db.get_conn_safe().ok()?;
    conn.query_row(
        "SELECT g.id
         FROM chat_v2_sessions s
         JOIN chat_v2_session_groups g ON g.id = s.group_id
         WHERE s.id = ?1 AND g.persist_status = 'active'",
        rusqlite::params![ctx.session_id],
        |row| row.get::<_, String>(0),
    )
    .ok()
}

/// Resolve the authoritative topic scope for a tool call.
///
/// Frontend send options may include a group scope snapshot, but the backend
/// session/group relationship is the source of truth. Falling back to the
/// database here prevents resource tools from treating a topic session as a
/// global-root session when the frontend snapshot is missing or stale.
fn effective_group_scope(ctx: &ExecutionContext) -> Option<GroupScope> {
    if let Some(group_id) = load_group_id_from_session(ctx) {
        return repair_group_scope(ctx, &group_id);
    }

    if let Some(group_id) = ctx.group_id.as_ref().filter(|id| !id.trim().is_empty()) {
        if let Some(scope) = repair_group_scope(ctx, group_id.trim()) {
            return Some(scope);
        }
        return Some(GroupScope {
            id: group_id.trim().to_string(),
            pinned_resource_ids: ctx.group_pinned_resource_ids.clone(),
        });
    }

    None
}

pub fn current_topic_folder_roots(ctx: &ExecutionContext) -> Vec<String> {
    let pinned_ids = effective_group_scope(ctx)
        .map(|scope| scope.pinned_resource_ids)
        .unwrap_or_default();
    normalize_folder_ids(pinned_ids)
}

pub fn is_topic_scoped(ctx: &ExecutionContext) -> bool {
    effective_group_scope(ctx).is_some()
}

pub fn current_topic_pinned_resource_ids(ctx: &ExecutionContext) -> Vec<String> {
    effective_group_scope(ctx)
        .map(|scope| {
            scope
                .pinned_resource_ids
                .into_iter()
                .filter_map(|id| {
                    let trimmed = id.trim();
                    if trimmed.is_empty() {
                        None
                    } else {
                        Some(trimmed.to_string())
                    }
                })
                .collect()
        })
        .unwrap_or_default()
}

pub fn current_topic_group_id(ctx: &ExecutionContext) -> Option<String> {
    effective_group_scope(ctx).map(|scope| scope.id)
}

pub fn current_context_folder_roots(ctx: &ExecutionContext) -> Vec<String> {
    normalize_folder_ids(ctx.group_pinned_resource_ids.iter().cloned())
}

pub fn normalize_folder_arg(value: Option<&Value>) -> Option<String> {
    value.and_then(|v| v.as_str()).and_then(|raw| {
        let trimmed = raw.trim();
        if trimmed.is_empty() || trimmed == "root" {
            None
        } else {
            Some(trimmed.to_string())
        }
    })
}

pub fn folder_is_within_roots(
    vfs_db: &Arc<VfsDatabase>,
    folder_id: &str,
    roots: &[String],
) -> Result<bool, String> {
    if roots.iter().any(|root| root == folder_id) {
        return Ok(true);
    }

    for root in roots {
        let descendants = VfsFolderRepo::get_folder_ids_recursive(vfs_db, root)
            .map_err(|e| format!("Failed to resolve topic folder scope: {}", e))?;
        if descendants.iter().any(|id| id == folder_id) {
            return Ok(true);
        }
    }

    Ok(false)
}

pub fn resolve_scoped_folder_id(
    ctx: &ExecutionContext,
    vfs_db: &Arc<VfsDatabase>,
    explicit_folder_id: Option<String>,
    tool_name: &str,
) -> Result<Option<String>, String> {
    if !is_topic_scoped(ctx) {
        return Ok(explicit_folder_id);
    }

    let topic_folders = current_topic_folder_roots(ctx);
    if topic_folders.is_empty() {
        log::warn!(
            "[ResourceScope] {}: group {:?} has no usable topic folder after repair attempt; returning empty scoped results",
            tool_name,
            current_topic_group_id(ctx)
        );
        return Ok(Some(NO_TOPIC_RESOURCE_FOLDER_SENTINEL.to_string()));
    }

    let folder_id = explicit_folder_id.unwrap_or_else(|| topic_folders[0].clone());
    if folder_is_within_roots(vfs_db, &folder_id, &topic_folders)? {
        Ok(Some(folder_id))
    } else {
        Err(format!(
            "当前课题只能访问自己的资源文件夹。请求的 folder_id '{}' 不属于当前课题范围。",
            folder_id
        ))
    }
}

pub fn resolve_scoped_folder_id_for_write(
    ctx: &ExecutionContext,
    vfs_db: &Arc<VfsDatabase>,
    explicit_folder_id: Option<String>,
    tool_name: &str,
) -> Result<Option<String>, String> {
    let scoped = resolve_scoped_folder_id(ctx, vfs_db, explicit_folder_id, tool_name)?;
    if scoped.as_deref() == Some(NO_TOPIC_RESOURCE_FOLDER_SENTINEL) {
        return Err("当前课题没有绑定资源文件夹，已拒绝把新资源写入全局根目录。".to_string());
    }
    Ok(scoped)
}

pub fn item_ids_in_topic_scope(
    ctx: &ExecutionContext,
    vfs_db: &Arc<VfsDatabase>,
) -> Result<HashSet<String>, String> {
    let mut allowed = HashSet::new();
    for root in current_topic_folder_roots(ctx) {
        let folder_ids = VfsFolderRepo::get_folder_ids_recursive(vfs_db, &root)
            .map_err(|e| format!("Failed to resolve topic folder scope: {}", e))?;
        for folder_id in folder_ids {
            let items = VfsFolderRepo::list_items_by_folder(vfs_db, Some(&folder_id))
                .map_err(|e| format!("Failed to list topic folder items: {}", e))?;
            allowed.extend(items.into_iter().map(|item| item.item_id));
        }
    }
    Ok(allowed)
}

pub fn pinned_resource_matches(
    vfs_db: &Arc<VfsDatabase>,
    pinned_id: &str,
    requested_id: &str,
    read_id: &str,
) -> bool {
    let pinned_id = pinned_id.trim();
    if pinned_id.is_empty() || pinned_id.starts_with("fld_") {
        return false;
    }
    if pinned_id == requested_id || pinned_id == read_id {
        return true;
    }
    if pinned_id.starts_with("res_") {
        if let Ok(Some(resource)) = VfsResourceRepo::get_resource(vfs_db, pinned_id) {
            if resource.id == requested_id || resource.id == read_id {
                return true;
            }
            if resource.source_id.as_deref().map_or(false, |source_id| {
                source_id == requested_id || source_id == read_id
            }) {
                return true;
            }
        }
    }
    false
}

pub fn ensure_item_in_scope(
    ctx: &ExecutionContext,
    vfs_db: &Arc<VfsDatabase>,
    requested_id: &str,
    read_id: &str,
) -> Result<(), String> {
    if !is_topic_scoped(ctx) {
        return Ok(());
    }

    if current_topic_pinned_resource_ids(ctx)
        .iter()
        .any(|pinned_id| pinned_resource_matches(vfs_db, pinned_id, requested_id, read_id))
    {
        return Ok(());
    }

    let allowed_item_ids = item_ids_in_topic_scope(ctx, vfs_db)?;
    if allowed_item_ids.contains(read_id) {
        return Ok(());
    }

    Err(format!(
        "资源 '{}' 不属于当前课题资源范围，已拒绝访问。",
        requested_id
    ))
}
