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
    existing_folder_ids(vfs_db, pinned_ids.iter().cloned())
        .next()
        .is_some()
}

fn existing_folder_ids(
    vfs_db: &Arc<VfsDatabase>,
    ids: impl IntoIterator<Item = String>,
) -> impl Iterator<Item = String> + '_ {
    normalize_folder_ids(ids)
        .into_iter()
        .filter(|folder_id| has_existing_folder(vfs_db, folder_id))
}

fn existing_topic_folder_roots(
    vfs_db: &Arc<VfsDatabase>,
    pinned_ids: impl IntoIterator<Item = String>,
) -> Vec<String> {
    existing_folder_ids(vfs_db, pinned_ids).collect()
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

fn load_group_scope_from_session(ctx: &ExecutionContext) -> Option<GroupScope> {
    let chat_db = ctx.chat_v2_db.as_ref()?;
    let conn = chat_db.get_conn_safe().ok()?;
    let group_id = conn
        .query_row(
            "SELECT g.id
         FROM chat_v2_sessions s
         JOIN chat_v2_session_groups g ON g.id = s.group_id
         WHERE s.id = ?1 AND g.persist_status = 'active'",
            rusqlite::params![ctx.session_id],
            |row| row.get::<_, String>(0),
        )
        .ok()?;
    let group = ChatV2Repo::get_group_with_conn(&conn, &group_id)
        .ok()
        .flatten()?;
    if group.persist_status != PersistStatus::Active {
        return None;
    }
    Some(GroupScope {
        id: group.id,
        pinned_resource_ids: group.pinned_resource_ids,
    })
}

/// Resolve the authoritative topic scope for a tool call.
///
/// Frontend send options may include a group scope snapshot, but the backend
/// session/group relationship is the source of truth. Falling back to the
/// database here prevents resource tools from treating a topic session as a
/// global-root session when the frontend snapshot is missing or stale.
fn effective_group_scope(
    ctx: &ExecutionContext,
    repair_missing_folder: bool,
) -> Option<GroupScope> {
    if let Some(scope) = load_group_scope_from_session(ctx) {
        if repair_missing_folder {
            return repair_group_scope(ctx, &scope.id);
        }
        if let Some(vfs_db) = ctx.vfs_db.as_ref() {
            if !has_existing_topic_folder(vfs_db, &scope.pinned_resource_ids) {
                return repair_group_scope(ctx, &scope.id);
            }
        }
        return Some(scope);
    }

    if ctx.chat_v2_db.is_some() {
        return None;
    }

    if let Some(group_id) = ctx.group_id.as_ref().filter(|id| !id.trim().is_empty()) {
        return Some(GroupScope {
            id: group_id.trim().to_string(),
            pinned_resource_ids: ctx.group_pinned_resource_ids.clone(),
        });
    }

    None
}

pub fn current_topic_folder_roots(ctx: &ExecutionContext) -> Vec<String> {
    let pinned_ids = effective_group_scope(ctx, false)
        .map(|scope| scope.pinned_resource_ids)
        .unwrap_or_default();
    if let Some(vfs_db) = ctx.vfs_db.as_ref() {
        return existing_topic_folder_roots(vfs_db, pinned_ids);
    }
    normalize_folder_ids(pinned_ids)
}

pub fn is_topic_scoped(ctx: &ExecutionContext) -> bool {
    effective_group_scope(ctx, false).is_some()
}

pub fn current_topic_pinned_resource_ids(ctx: &ExecutionContext) -> Vec<String> {
    effective_group_scope(ctx, false)
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
    effective_group_scope(ctx, false).map(|scope| scope.id)
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
    resolve_scoped_folder_id_inner(ctx, vfs_db, explicit_folder_id, tool_name, false)
}

fn resolve_scoped_folder_id_inner(
    ctx: &ExecutionContext,
    vfs_db: &Arc<VfsDatabase>,
    explicit_folder_id: Option<String>,
    tool_name: &str,
    repair_missing_folder: bool,
) -> Result<Option<String>, String> {
    let Some(scope) = effective_group_scope(ctx, repair_missing_folder) else {
        return Ok(explicit_folder_id);
    };

    let topic_folders = existing_topic_folder_roots(vfs_db, scope.pinned_resource_ids);
    if topic_folders.is_empty() {
        log::warn!(
            "[ResourceScope] {}: group {:?} has no usable topic folder; returning empty scoped results",
            tool_name,
            scope.id
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
    let scoped = resolve_scoped_folder_id_inner(ctx, vfs_db, explicit_folder_id, tool_name, true)?;
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

#[cfg(test)]
mod tests {
    use super::*;
    use crate::vfs::{VfsDatabase, VfsFolder};
    use std::sync::Arc;
    use tempfile::TempDir;

    fn setup_test_db() -> (TempDir, Arc<VfsDatabase>) {
        let temp_dir = TempDir::new().expect("create temp dir");
        let db = Arc::new(VfsDatabase::new(temp_dir.path()).expect("create vfs db"));
        (temp_dir, db)
    }

    fn create_folder(db: &Arc<VfsDatabase>, title: &str, parent_id: Option<String>) -> VfsFolder {
        let folder = VfsFolder::new(title.to_string(), parent_id, None, None);
        VfsFolderRepo::create_folder(db, &folder).expect("create folder");
        folder
    }

    #[test]
    fn current_topic_folder_roots_filters_stale_pinned_folders() {
        let (_temp_dir, db) = setup_test_db();
        let valid = create_folder(&db, "Topic", None);

        assert_eq!(
            existing_topic_folder_roots(
                &db,
                vec![
                    "fld_stale_missing".to_string(),
                    valid.id.clone(),
                    "res_pinned".to_string(),
                ],
            ),
            vec![valid.id]
        );
    }

    #[test]
    fn topic_folder_roots_empty_when_all_pinned_folders_are_stale() {
        let (_temp_dir, db) = setup_test_db();

        assert!(existing_topic_folder_roots(
            &db,
            vec!["fld_stale_missing".to_string(), "res_pinned".to_string(),],
        )
        .is_empty());
    }

    #[test]
    fn folder_scope_allows_descendants_and_rejects_siblings() {
        let (_temp_dir, db) = setup_test_db();
        let root = create_folder(&db, "Topic", None);
        let child = create_folder(&db, "Child", Some(root.id.clone()));
        let sibling = create_folder(&db, "Sibling", None);

        assert!(folder_is_within_roots(&db, &child.id, &[root.id.clone()])
            .expect("descendant check should succeed"));
        assert!(!folder_is_within_roots(&db, &sibling.id, &[root.id])
            .expect("sibling check should succeed"));
    }
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
