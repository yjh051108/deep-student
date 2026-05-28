use serde::{Deserialize, Serialize};
use std::sync::Arc;
use tauri::State;
use tracing::{info, warn};

use crate::chat_v2::database::ChatV2Database;
use crate::chat_v2::repo::ChatV2Repo;
use crate::chat_v2::types::PersistStatus;
use crate::llm_manager::LLMManager;
use crate::vfs::database::VfsDatabase;
use crate::vfs::indexing::VfsFullIndexingService;
use crate::vfs::lance_store::VfsLanceStore;

use super::audit_log::{self, MemoryAuditLogItem};
use super::service::{
    MemoryConfigOutput, MemoryListItem, MemorySearchResult, MemoryService, MemoryWriteOutput,
    SmartWriteOutput, WriteMode,
};

#[derive(Debug, Serialize, Deserialize)]
#[serde(rename_all = "camelCase")]
pub struct BatchOperationResult {
    pub total: usize,
    pub succeeded: usize,
    pub failed: usize,
    pub errors: Vec<String>,
}

#[derive(Debug, Serialize, Deserialize)]
#[serde(rename_all = "camelCase")]
pub struct MemoryReadOutput {
    pub note_id: String,
    pub title: String,
    pub content: String,
    pub folder_path: String,
    pub updated_at: String,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
#[serde(rename_all = "camelCase")]
pub struct MemoryBatchWriteItemInput {
    pub title: String,
    pub content: String,
    #[serde(default)]
    pub folder_path: Option<String>,
    #[serde(default)]
    pub scope: Option<String>,
    #[serde(default)]
    pub memory_type: Option<String>,
    #[serde(default)]
    pub memory_purpose: Option<String>,
    #[serde(default)]
    pub idempotency_key: Option<String>,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
#[serde(rename_all = "camelCase")]
pub struct MemoryBatchWriteItemResult {
    pub title: String,
    #[serde(flatten)]
    pub output: SmartWriteOutput,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
#[serde(rename_all = "camelCase")]
pub struct MemoryBatchWriteOutput {
    pub total: usize,
    pub succeeded: usize,
    pub failed: usize,
    pub added: usize,
    pub updated: usize,
    pub skipped: usize,
    pub filtered: usize,
    pub results: Vec<MemoryBatchWriteItemResult>,
}

fn get_memory_service(
    vfs_db: &Arc<VfsDatabase>,
    lance_store: &Arc<VfsLanceStore>,
    llm_manager: &Arc<LLMManager>,
) -> MemoryService {
    MemoryService::new(vfs_db.clone(), lance_store.clone(), llm_manager.clone())
}

fn parse_memory_type(memory_type: Option<&str>) -> Result<super::service::MemoryType, String> {
    match memory_type.map(|s| s.trim().to_lowercase()) {
        Some(s) if s == "fact" => Ok(super::service::MemoryType::Fact),
        Some(s) if s == "study" => Ok(super::service::MemoryType::Study),
        Some(s) if s == "note" => Ok(super::service::MemoryType::Note),
        Some(s) => Err(format!(
            "Invalid memory_type '{}', expected one of: fact, study, note",
            s
        )),
        None => Ok(super::service::MemoryType::Fact),
    }
}

fn parse_memory_purpose(
    memory_purpose: Option<&str>,
) -> Result<Option<super::service::MemoryPurpose>, String> {
    match memory_purpose.map(|s| s.trim().to_lowercase()) {
        Some(s) if s == "internalized" => Ok(Some(super::service::MemoryPurpose::Internalized)),
        Some(s) if s == "memorized" => Ok(Some(super::service::MemoryPurpose::Memorized)),
        Some(s) if s == "supplementary" => Ok(Some(super::service::MemoryPurpose::Supplementary)),
        Some(s) if s == "systemic" => Ok(Some(super::service::MemoryPurpose::Systemic)),
        Some(s) => Err(format!(
            "Invalid memory_purpose '{}', expected one of: internalized, memorized, supplementary, systemic",
            s
        )),
        None => Ok(None),
    }
}

fn scoped_visible_paths(
    chat_db: Option<&ChatV2Database>,
    group_id: Option<&str>,
    group_name: Option<&str>,
    admin_all: Option<bool>,
) -> Result<Option<Vec<String>>, String> {
    if admin_all.unwrap_or(false) {
        return Ok(None);
    }
    if group_id.map(|id| !id.trim().is_empty()).unwrap_or(false)
        || group_name
            .map(|name| !name.trim().is_empty())
            .unwrap_or(false)
    {
        let chat_db = chat_db
            .ok_or_else(|| "memory topic reads require chat database state".to_string())?;
        let (resolved_group_id, resolved_group_name) =
            resolve_active_memory_topic(chat_db, group_id)?;
        Ok(Some(crate::memory::visible_scope_roots(
            Some(resolved_group_id.as_str()),
            Some(resolved_group_name.as_str()),
        )))
    } else {
        Ok(Some(vec![crate::memory::GLOBAL_MEMORY_FOLDER.to_string()]))
    }
}

fn path_in_any_scope(path: &str, roots: &[String]) -> bool {
    roots
        .iter()
        .any(|root| crate::memory::is_folder_path_within_scope(path, root))
}

fn validate_memory_path_visible(path: &str, roots: &[String]) -> Result<(), String> {
    if path_in_any_scope(path, roots) {
        Ok(())
    } else {
        Err(format!(
            "记忆路径 '{}' 不属于当前课题可见范围，已拒绝访问。",
            path
        ))
    }
}

fn scoped_read_paths(
    chat_db: &ChatV2Database,
    folder_path: Option<String>,
    folder_paths: Option<Vec<String>>,
    group_id: Option<&str>,
    group_name: Option<&str>,
    admin_all: Option<bool>,
) -> Result<Option<Vec<String>>, String> {
    let explicit_paths: Vec<String> = folder_paths
        .unwrap_or_default()
        .into_iter()
        .map(|path| path.trim().to_string())
        .filter(|path| !path.is_empty())
        .collect();
    let single_path = folder_path
        .map(|path| path.trim().to_string())
        .filter(|path| !path.is_empty());

    let Some(visible_roots) =
        scoped_visible_paths(Some(chat_db), group_id, group_name, admin_all)?
    else {
        if explicit_paths.is_empty() {
            return Ok(single_path.map(|path| vec![path]));
        }
        return Ok(Some(explicit_paths));
    };

    let requested = if !explicit_paths.is_empty() {
        explicit_paths
    } else if let Some(path) = single_path {
        vec![path]
    } else {
        visible_roots.clone()
    };

    for path in &requested {
        validate_memory_path_visible(path, &visible_roots)?;
    }
    Ok(Some(requested))
}

fn validate_note_visible(
    chat_db: &ChatV2Database,
    service: &MemoryService,
    note_id: &str,
    group_id: Option<&str>,
    group_name: Option<&str>,
    admin_all: Option<bool>,
) -> Result<(), String> {
    let Some(visible_roots) =
        scoped_visible_paths(Some(chat_db), group_id, group_name, admin_all)?
    else {
        return Ok(());
    };
    let folder_path = service
        .get_note_folder_path(note_id)
        .map_err(|e| e.to_string())?;
    validate_memory_path_visible(&folder_path, &visible_roots)
}

fn scoped_mutation_roots(
    chat_db: &ChatV2Database,
    group_id: Option<&str>,
    group_name: Option<&str>,
    admin_all: Option<bool>,
) -> Result<Option<Vec<String>>, String> {
    if admin_all.unwrap_or(false) {
        return Ok(None);
    }
    if group_id.map(|id| !id.trim().is_empty()).unwrap_or(false)
        || group_name
            .map(|name| !name.trim().is_empty())
            .unwrap_or(false)
    {
        let (resolved_group_id, resolved_group_name) =
            resolve_active_memory_topic(chat_db, group_id)?;
        return Ok(Some(crate::memory::visible_scope_roots(
            Some(resolved_group_id.as_str()),
            Some(resolved_group_name.as_str()),
        )));
    }
    Ok(Some(vec![crate::memory::GLOBAL_MEMORY_FOLDER.to_string()]))
}

fn validate_note_mutable(
    chat_db: &ChatV2Database,
    service: &MemoryService,
    note_id: &str,
    group_id: Option<&str>,
    group_name: Option<&str>,
    admin_all: Option<bool>,
) -> Result<(), String> {
    let Some(visible_roots) = scoped_mutation_roots(chat_db, group_id, group_name, admin_all)?
    else {
        return Ok(());
    };
    let folder_path = service
        .get_note_folder_path(note_id)
        .map_err(|e| e.to_string())?;
    validate_memory_path_visible(&folder_path, &visible_roots)
}

fn validate_memory_mutation_path_visible(
    chat_db: &ChatV2Database,
    path: &str,
    group_id: Option<&str>,
    group_name: Option<&str>,
    admin_all: Option<bool>,
) -> Result<(), String> {
    if let Some(visible_roots) = scoped_mutation_roots(chat_db, group_id, group_name, admin_all)? {
        validate_memory_path_visible(path, &visible_roots)?;
    }
    Ok(())
}

fn scoped_maintenance_paths(
    service: &MemoryService,
    note_id: Option<&str>,
    explicit_path: Option<&str>,
    group_id: Option<&str>,
    group_name: Option<&str>,
    admin_all: Option<bool>,
) -> Vec<String> {
    if admin_all.unwrap_or(false) {
        return Vec::new();
    }

    if let Some(path) = explicit_path.map(str::trim).filter(|path| !path.is_empty()) {
        return vec![path.to_string()];
    }

    if let Some(note_id) = note_id {
        if let Ok(path) = service.get_note_folder_path(note_id) {
            return vec![path];
        }
    }

    scoped_visible_paths(None, group_id, group_name, admin_all)
        .ok()
        .flatten()
        .unwrap_or_default()
}

fn resolve_memory_write_folder(
    chat_db: &ChatV2Database,
    folder_path: Option<&str>,
    scope: Option<&str>,
    group_id: Option<&str>,
    group_name: Option<&str>,
    admin_all: Option<bool>,
) -> Result<Option<String>, String> {
    if admin_all.unwrap_or(false) {
        return Ok(folder_path
            .map(str::trim)
            .filter(|path| !path.is_empty())
            .map(String::from)
            .or(Some(crate::memory::GLOBAL_MEMORY_FOLDER.to_string())));
    }

    if group_id.map(|id| id.trim().is_empty()).unwrap_or(true)
        && group_name
            .map(|name| name.trim().is_empty())
            .unwrap_or(true)
    {
        if matches!(
            scope
                .map(|raw| crate::memory::MemoryScope::from_arg(Some(raw)))
                .transpose()?,
            Some(crate::memory::MemoryScope::Topic)
        ) {
            return Err("memory_write requires group context for topic scope".to_string());
        }
        let target = folder_path
            .map(str::trim)
            .filter(|path| !path.is_empty())
            .map(String::from)
            .unwrap_or_else(|| crate::memory::GLOBAL_MEMORY_FOLDER.to_string());
        validate_memory_path_visible(&target, &[crate::memory::GLOBAL_MEMORY_FOLDER.to_string()])?;
        return Ok(Some(target));
    }

    let memory_scope = crate::memory::MemoryScope::from_arg(scope)?;
    let folder = match memory_scope {
        crate::memory::MemoryScope::Global => crate::memory::join_memory_folder_paths(
            crate::memory::GLOBAL_MEMORY_FOLDER,
            folder_path,
        ),
        crate::memory::MemoryScope::Topic => {
            let (group_id, group_name) = resolve_active_memory_topic(chat_db, group_id)?;
            crate::memory::scoped_folder_path(
                Some(group_id.as_str()),
                Some(group_name.as_str()),
                crate::memory::MemoryScope::Topic,
                folder_path,
            )
            .ok_or_else(|| "memory_write requires group context for topic scope".to_string())?
        }
    };
    Ok(Some(folder))
}

fn resolve_active_memory_topic(
    chat_db: &ChatV2Database,
    group_id: Option<&str>,
) -> Result<(String, String), String> {
    let group_id = group_id
        .map(str::trim)
        .filter(|id| !id.is_empty())
        .ok_or_else(|| "memory topic scope requires an active group_id".to_string())?;
    let conn = chat_db.get_conn_safe().map_err(|e| e.to_string())?;
    let group = ChatV2Repo::get_group_with_conn(&conn, group_id)
        .map_err(|e| e.to_string())?
        .ok_or_else(|| format!("memory topic group not found: {}", group_id))?;
    if group.persist_status != PersistStatus::Active {
        return Err(format!(
            "memory topic group is not active: {} ({:?})",
            group_id, group.persist_status
        ));
    }
    Ok((group.id, group.name))
}

/// 写入后触发单资源索引，保证 write-then-search SLA。
/// 索引成功后标记为 indexed，防止批量 worker 重复处理。
fn trigger_immediate_index(
    vfs_db: Arc<VfsDatabase>,
    llm_manager: Arc<LLMManager>,
    lance_store: Arc<VfsLanceStore>,
    resource_id: String,
) {
    crate::background_tasks::BACKGROUND_TASKS.spawn(async move {
        let db_ref = vfs_db.clone();
        let indexing_service = match VfsFullIndexingService::new(vfs_db, llm_manager, lance_store) {
            Ok(svc) => svc,
            Err(e) => {
                warn!(
                    "[Memory] Failed to create indexing service for immediate index of {}: {}",
                    resource_id, e
                );
                return;
            }
        };

        match indexing_service
            .index_resource(&resource_id, None, None)
            .await
        {
            Ok((chunk_count, dim)) => {
                if let Err(e) = crate::vfs::repos::embedding_repo::VfsIndexStateRepo::mark_indexed(
                    &db_ref,
                    &resource_id,
                    &format!("mem_handler_{}", chrono::Utc::now().timestamp_millis()),
                ) {
                    warn!(
                        "[Memory] Failed to mark indexed after immediate indexing: {}",
                        e
                    );
                }
                info!(
                    "[Memory] Immediate index completed for resource {} ({} chunks, dim={})",
                    resource_id, chunk_count, dim
                );
            }
            Err(e) => {
                warn!(
                    "[Memory] Immediate index failed for resource {} (will retry in next batch): {}",
                    resource_id, e
                );
            }
        }
    });
}

#[tauri::command]
pub async fn memory_get_config(
    vfs_db: State<'_, Arc<VfsDatabase>>,
    lance_store: State<'_, Arc<VfsLanceStore>>,
    llm_manager: State<'_, Arc<LLMManager>>,
) -> Result<MemoryConfigOutput, String> {
    let service = get_memory_service(&vfs_db, &lance_store, &llm_manager);
    service.get_config().map_err(|e| e.to_string())
}

#[tauri::command]
pub async fn memory_set_root_folder(
    folder_id: String,
    vfs_db: State<'_, Arc<VfsDatabase>>,
    lance_store: State<'_, Arc<VfsLanceStore>>,
    llm_manager: State<'_, Arc<LLMManager>>,
) -> Result<(), String> {
    let service = get_memory_service(&vfs_db, &lance_store, &llm_manager);
    service
        .set_root_folder(&folder_id)
        .map_err(|e| e.to_string())
}

#[tauri::command]
pub async fn memory_set_privacy_mode(
    enabled: bool,
    vfs_db: State<'_, Arc<VfsDatabase>>,
    lance_store: State<'_, Arc<VfsLanceStore>>,
    llm_manager: State<'_, Arc<LLMManager>>,
) -> Result<(), String> {
    let service = get_memory_service(&vfs_db, &lance_store, &llm_manager);
    service.set_privacy_mode(enabled).map_err(|e| e.to_string())
}

#[tauri::command]
pub async fn memory_create_root_folder(
    title: String,
    vfs_db: State<'_, Arc<VfsDatabase>>,
    lance_store: State<'_, Arc<VfsLanceStore>>,
    llm_manager: State<'_, Arc<LLMManager>>,
) -> Result<String, String> {
    let service = get_memory_service(&vfs_db, &lance_store, &llm_manager);
    service
        .create_root_folder(&title)
        .map_err(|e| e.to_string())
}

#[tauri::command]
pub async fn memory_get_or_create_root_folder(
    vfs_db: State<'_, Arc<VfsDatabase>>,
    lance_store: State<'_, Arc<VfsLanceStore>>,
    llm_manager: State<'_, Arc<LLMManager>>,
) -> Result<String, String> {
    let service = get_memory_service(&vfs_db, &lance_store, &llm_manager);
    service
        .get_or_create_root_folder()
        .map_err(|e| e.to_string())
}

#[tauri::command]
pub async fn memory_search(
    query: String,
    top_k: Option<usize>,
    folder_path: Option<String>,
    folder_paths: Option<Vec<String>>,
    group_id: Option<String>,
    group_name: Option<String>,
    admin_all: Option<bool>,
    vfs_db: State<'_, Arc<VfsDatabase>>,
    lance_store: State<'_, Arc<VfsLanceStore>>,
    llm_manager: State<'_, Arc<LLMManager>>,
    chat_db: State<'_, Arc<ChatV2Database>>,
) -> Result<Vec<MemorySearchResult>, String> {
    let service = get_memory_service(&vfs_db, &lance_store, &llm_manager);
    let k = top_k.unwrap_or(5).clamp(1, 100);
    let scoped_paths = scoped_read_paths(
        chat_db.inner().as_ref(),
        folder_path,
        folder_paths,
        group_id.as_deref(),
        group_name.as_deref(),
        admin_all,
    )?;
    if let Some(paths) = scoped_paths {
        service
            .search_with_rerank_in_folder_paths(&query, k, false, &paths)
            .await
            .map_err(|e| e.to_string())
    } else {
        service
            .search_with_rerank_in_folder_path(&query, k, false, None)
            .await
            .map_err(|e| e.to_string())
    }
}

#[tauri::command]
pub async fn memory_read(
    note_id: String,
    group_id: Option<String>,
    group_name: Option<String>,
    admin_all: Option<bool>,
    vfs_db: State<'_, Arc<VfsDatabase>>,
    lance_store: State<'_, Arc<VfsLanceStore>>,
    llm_manager: State<'_, Arc<LLMManager>>,
    chat_db: State<'_, Arc<ChatV2Database>>,
) -> Result<Option<MemoryReadOutput>, String> {
    let service = get_memory_service(&vfs_db, &lance_store, &llm_manager);
    validate_note_visible(
        chat_db.inner().as_ref(),
        &service,
        &note_id,
        group_id.as_deref(),
        group_name.as_deref(),
        admin_all,
    )?;

    match service.read(&note_id).map_err(|e| e.to_string())? {
        Some((note, content)) => {
            let folder_path = service
                .get_note_folder_path(&note_id)
                .map_err(|e| e.to_string())?;

            Ok(Some(MemoryReadOutput {
                note_id: note.id,
                title: note.title,
                content,
                folder_path,
                updated_at: note.updated_at,
            }))
        }
        None => Ok(None),
    }
}

#[tauri::command]
pub async fn memory_write(
    note_id: Option<String>,
    folder_path: Option<String>,
    scope: Option<String>,
    group_id: Option<String>,
    group_name: Option<String>,
    admin_all: Option<bool>,
    title: String,
    content: String,
    mode: Option<String>,
    vfs_db: State<'_, Arc<VfsDatabase>>,
    lance_store: State<'_, Arc<VfsLanceStore>>,
    llm_manager: State<'_, Arc<LLMManager>>,
    chat_db: State<'_, Arc<ChatV2Database>>,
) -> Result<MemoryWriteOutput, String> {
    let service = get_memory_service(&vfs_db, &lance_store, &llm_manager);
    let write_mode = mode
        .map(|m| WriteMode::from_str(&m))
        .unwrap_or(WriteMode::Create);

    let result = if let Some(target_note_id) = note_id {
        validate_note_mutable(
            chat_db.inner().as_ref(),
            &service,
            &target_note_id,
            group_id.as_deref(),
            group_name.as_deref(),
            admin_all,
        )?;
        match write_mode {
            WriteMode::Append => {
                let current = service
                    .read(&target_note_id)
                    .map_err(|e| e.to_string())?
                    .map(|(_, existing)| existing)
                    .unwrap_or_default();
                let final_content = if current.is_empty() {
                    content.clone()
                } else {
                    format!("{}\n\n{}", current, content)
                };
                service
                    .update_by_id(&target_note_id, Some(&title), Some(&final_content))
                    .map_err(|e| e.to_string())?
            }
            _ => service
                .update_by_id(&target_note_id, Some(&title), Some(&content))
                .map_err(|e| e.to_string())?,
        }
    } else {
        let resolved_folder = resolve_memory_write_folder(
            chat_db.inner().as_ref(),
            folder_path.as_deref(),
            scope.as_deref(),
            group_id.as_deref(),
            group_name.as_deref(),
            admin_all,
        )?;
        service
            .write(resolved_folder.as_deref(), &title, &content, write_mode)
            .map_err(|e| e.to_string())?
    };

    // ★ P2-2 修复：写入后立即触发索引，保证 write-then-search SLA
    trigger_immediate_index(
        Arc::clone(vfs_db.inner()),
        Arc::clone(llm_manager.inner()),
        Arc::clone(lance_store.inner()),
        result.resource_id.clone(),
    );

    service.spawn_post_write_maintenance_for_paths(scoped_maintenance_paths(
        &service,
        Some(&result.note_id),
        None,
        group_id.as_deref(),
        group_name.as_deref(),
        admin_all,
    ));

    Ok(result)
}

#[tauri::command]
pub async fn memory_list(
    folder_path: Option<String>,
    folder_paths: Option<Vec<String>>,
    group_id: Option<String>,
    group_name: Option<String>,
    admin_all: Option<bool>,
    limit: Option<u32>,
    offset: Option<u32>,
    vfs_db: State<'_, Arc<VfsDatabase>>,
    lance_store: State<'_, Arc<VfsLanceStore>>,
    llm_manager: State<'_, Arc<LLMManager>>,
    chat_db: State<'_, Arc<ChatV2Database>>,
) -> Result<Vec<MemoryListItem>, String> {
    let service = get_memory_service(&vfs_db, &lance_store, &llm_manager);
    let safe_limit = limit.unwrap_or(100).clamp(1, 500);
    let scoped_paths = scoped_read_paths(
        chat_db.inner().as_ref(),
        folder_path,
        folder_paths,
        group_id.as_deref(),
        group_name.as_deref(),
        admin_all,
    )?;
    if let Some(paths) = scoped_paths {
        service
            .list_folder_paths(&paths, safe_limit, offset.unwrap_or(0))
            .map_err(|e| e.to_string())
    } else {
        service
            .list(None, safe_limit, offset.unwrap_or(0))
            .map_err(|e| e.to_string())
    }
}

#[tauri::command]
pub async fn memory_get_tree(
    folder_path: Option<String>,
    group_id: Option<String>,
    group_name: Option<String>,
    admin_all: Option<bool>,
    vfs_db: State<'_, Arc<VfsDatabase>>,
    lance_store: State<'_, Arc<VfsLanceStore>>,
    llm_manager: State<'_, Arc<LLMManager>>,
    chat_db: State<'_, Arc<ChatV2Database>>,
) -> Result<Option<crate::vfs::types::FolderTreeNode>, String> {
    let service = get_memory_service(&vfs_db, &lance_store, &llm_manager);
    let scoped_paths = scoped_read_paths(
        chat_db.inner().as_ref(),
        folder_path,
        None,
        group_id.as_deref(),
        group_name.as_deref(),
        admin_all,
    )?;
    let folder_path = scoped_paths
        .as_ref()
        .and_then(|paths| paths.first())
        .map(String::as_str);
    service
        .get_tree_in_folder_path(folder_path)
        .map_err(|e| e.to_string())
}

/// 添加记忆关联（双向）
#[tauri::command]
pub async fn memory_add_relation(
    note_id_a: String,
    note_id_b: String,
    group_id: Option<String>,
    group_name: Option<String>,
    admin_all: Option<bool>,
    vfs_db: State<'_, Arc<VfsDatabase>>,
    lance_store: State<'_, Arc<VfsLanceStore>>,
    llm_manager: State<'_, Arc<LLMManager>>,
    chat_db: State<'_, Arc<ChatV2Database>>,
) -> Result<(), String> {
    let service = get_memory_service(&vfs_db, &lance_store, &llm_manager);
    validate_note_mutable(
        chat_db.inner().as_ref(),
        &service,
        &note_id_a,
        group_id.as_deref(),
        group_name.as_deref(),
        admin_all,
    )?;
    validate_note_mutable(
        chat_db.inner().as_ref(),
        &service,
        &note_id_b,
        group_id.as_deref(),
        group_name.as_deref(),
        admin_all,
    )?;
    service
        .add_relation(&note_id_a, &note_id_b)
        .map_err(|e| e.to_string())
}

/// 移除记忆关联（双向）
#[tauri::command]
pub async fn memory_remove_relation(
    note_id_a: String,
    note_id_b: String,
    group_id: Option<String>,
    group_name: Option<String>,
    admin_all: Option<bool>,
    vfs_db: State<'_, Arc<VfsDatabase>>,
    lance_store: State<'_, Arc<VfsLanceStore>>,
    llm_manager: State<'_, Arc<LLMManager>>,
    chat_db: State<'_, Arc<ChatV2Database>>,
) -> Result<(), String> {
    let service = get_memory_service(&vfs_db, &lance_store, &llm_manager);
    validate_note_mutable(
        chat_db.inner().as_ref(),
        &service,
        &note_id_a,
        group_id.as_deref(),
        group_name.as_deref(),
        admin_all,
    )?;
    validate_note_mutable(
        chat_db.inner().as_ref(),
        &service,
        &note_id_b,
        group_id.as_deref(),
        group_name.as_deref(),
        admin_all,
    )?;
    service
        .remove_relation(&note_id_a, &note_id_b)
        .map_err(|e| e.to_string())
}

/// 获取关联记忆 ID 列表
#[tauri::command]
pub async fn memory_get_related(
    note_id: String,
    group_id: Option<String>,
    group_name: Option<String>,
    admin_all: Option<bool>,
    vfs_db: State<'_, Arc<VfsDatabase>>,
    lance_store: State<'_, Arc<VfsLanceStore>>,
    llm_manager: State<'_, Arc<LLMManager>>,
    chat_db: State<'_, Arc<ChatV2Database>>,
) -> Result<Vec<String>, String> {
    let service = get_memory_service(&vfs_db, &lance_store, &llm_manager);
    validate_note_visible(
        chat_db.inner().as_ref(),
        &service,
        &note_id,
        group_id.as_deref(),
        group_name.as_deref(),
        admin_all,
    )?;
    let related_ids = service
        .get_related_ids(&note_id)
        .map_err(|e| e.to_string())?;
    if admin_all.unwrap_or(false) {
        return Ok(related_ids);
    }
    Ok(related_ids
        .into_iter()
        .filter(|related_id| {
            validate_note_visible(
                chat_db.inner().as_ref(),
                &service,
                related_id,
                group_id.as_deref(),
                group_name.as_deref(),
                admin_all,
            )
            .is_ok()
        })
        .collect())
}

/// 更新记忆标签
#[tauri::command]
pub async fn memory_update_tags(
    note_id: String,
    tags: Vec<String>,
    group_id: Option<String>,
    group_name: Option<String>,
    admin_all: Option<bool>,
    vfs_db: State<'_, Arc<VfsDatabase>>,
    lance_store: State<'_, Arc<VfsLanceStore>>,
    llm_manager: State<'_, Arc<LLMManager>>,
    chat_db: State<'_, Arc<ChatV2Database>>,
) -> Result<(), String> {
    let service = get_memory_service(&vfs_db, &lance_store, &llm_manager);
    validate_note_mutable(
        chat_db.inner().as_ref(),
        &service,
        &note_id,
        group_id.as_deref(),
        group_name.as_deref(),
        admin_all,
    )?;
    service
        .update_tags(&note_id, tags)
        .map_err(|e| e.to_string())
}

/// 获取记忆标签
#[tauri::command]
pub async fn memory_get_tags(
    note_id: String,
    group_id: Option<String>,
    group_name: Option<String>,
    admin_all: Option<bool>,
    vfs_db: State<'_, Arc<VfsDatabase>>,
    lance_store: State<'_, Arc<VfsLanceStore>>,
    llm_manager: State<'_, Arc<LLMManager>>,
    chat_db: State<'_, Arc<ChatV2Database>>,
) -> Result<Vec<String>, String> {
    let service = get_memory_service(&vfs_db, &lance_store, &llm_manager);
    validate_note_visible(
        chat_db.inner().as_ref(),
        &service,
        &note_id,
        group_id.as_deref(),
        group_name.as_deref(),
        admin_all,
    )?;
    service.get_tags(&note_id).map_err(|e| e.to_string())
}

/// 批量删除记忆
#[tauri::command]
pub async fn memory_batch_delete(
    note_ids: Vec<String>,
    group_id: Option<String>,
    group_name: Option<String>,
    admin_all: Option<bool>,
    vfs_db: State<'_, Arc<VfsDatabase>>,
    lance_store: State<'_, Arc<VfsLanceStore>>,
    llm_manager: State<'_, Arc<LLMManager>>,
    chat_db: State<'_, Arc<ChatV2Database>>,
) -> Result<BatchOperationResult, String> {
    let service = get_memory_service(&vfs_db, &lance_store, &llm_manager);
    let total = note_ids.len();
    let mut succeeded = 0usize;
    let mut failed = 0usize;
    let mut errors: Vec<String> = Vec::new();

    for note_id in &note_ids {
        if let Err(e) = validate_note_mutable(
            chat_db.inner().as_ref(),
            &service,
            note_id,
            group_id.as_deref(),
            group_name.as_deref(),
            admin_all,
        ) {
            failed += 1;
            if errors.len() < 5 {
                errors.push(format!("{}: {}", note_id, e));
            }
            continue;
        }
        match service.delete(note_id).await {
            Ok(()) => succeeded += 1,
            Err(e) => {
                failed += 1;
                if errors.len() < 5 {
                    errors.push(format!("{}: {}", note_id, e));
                }
            }
        }
    }

    if succeeded > 0 {
        service.spawn_post_write_maintenance_for_paths(scoped_maintenance_paths(
            &service,
            None,
            None,
            group_id.as_deref(),
            group_name.as_deref(),
            admin_all,
        ));
    }

    Ok(BatchOperationResult {
        total,
        succeeded,
        failed,
        errors,
    })
}

/// 批量移动记忆到指定文件夹
#[tauri::command]
pub async fn memory_batch_move(
    note_ids: Vec<String>,
    target_folder_path: String,
    group_id: Option<String>,
    group_name: Option<String>,
    admin_all: Option<bool>,
    vfs_db: State<'_, Arc<VfsDatabase>>,
    lance_store: State<'_, Arc<VfsLanceStore>>,
    llm_manager: State<'_, Arc<LLMManager>>,
    chat_db: State<'_, Arc<ChatV2Database>>,
) -> Result<BatchOperationResult, String> {
    let service = get_memory_service(&vfs_db, &lance_store, &llm_manager);
    let total = note_ids.len();
    let mut succeeded = 0usize;
    let mut failed = 0usize;
    let mut errors: Vec<String> = Vec::new();
    validate_memory_mutation_path_visible(
        chat_db.inner().as_ref(),
        &target_folder_path,
        group_id.as_deref(),
        group_name.as_deref(),
        admin_all,
    )?;

    for note_id in &note_ids {
        if let Err(e) = validate_note_mutable(
            chat_db.inner().as_ref(),
            &service,
            note_id,
            group_id.as_deref(),
            group_name.as_deref(),
            admin_all,
        ) {
            failed += 1;
            if errors.len() < 5 {
                errors.push(format!("{}: {}", note_id, e));
            }
            continue;
        }
        match service.move_to_folder(note_id, &target_folder_path) {
            Ok(()) => succeeded += 1,
            Err(e) => {
                failed += 1;
                if errors.len() < 5 {
                    errors.push(format!("{}: {}", note_id, e));
                }
            }
        }
    }

    if succeeded > 0 {
        service.spawn_post_write_maintenance_for_paths(scoped_maintenance_paths(
            &service,
            None,
            Some(&target_folder_path),
            group_id.as_deref(),
            group_name.as_deref(),
            admin_all,
        ));
    }

    Ok(BatchOperationResult {
        total,
        succeeded,
        failed,
        errors,
    })
}

/// 移动记忆到指定文件夹路径
#[tauri::command]
pub async fn memory_move_to_folder(
    note_id: String,
    target_folder_path: String,
    group_id: Option<String>,
    group_name: Option<String>,
    admin_all: Option<bool>,
    vfs_db: State<'_, Arc<VfsDatabase>>,
    lance_store: State<'_, Arc<VfsLanceStore>>,
    llm_manager: State<'_, Arc<LLMManager>>,
    chat_db: State<'_, Arc<ChatV2Database>>,
) -> Result<(), String> {
    let service = get_memory_service(&vfs_db, &lance_store, &llm_manager);
    validate_note_mutable(
        chat_db.inner().as_ref(),
        &service,
        &note_id,
        group_id.as_deref(),
        group_name.as_deref(),
        admin_all,
    )?;
    validate_memory_mutation_path_visible(
        chat_db.inner().as_ref(),
        &target_folder_path,
        group_id.as_deref(),
        group_name.as_deref(),
        admin_all,
    )?;
    service
        .move_to_folder(&note_id, &target_folder_path)
        .map_err(|e| e.to_string())?;
    service.spawn_post_write_maintenance_for_paths(scoped_maintenance_paths(
        &service,
        None,
        Some(&target_folder_path),
        group_id.as_deref(),
        group_name.as_deref(),
        admin_all,
    ));
    Ok(())
}

// ★ 修复风险2：按 note_id 更新记忆
#[tauri::command]
pub async fn memory_update_by_id(
    note_id: String,
    title: Option<String>,
    content: Option<String>,
    group_id: Option<String>,
    group_name: Option<String>,
    admin_all: Option<bool>,
    vfs_db: State<'_, Arc<VfsDatabase>>,
    lance_store: State<'_, Arc<VfsLanceStore>>,
    llm_manager: State<'_, Arc<LLMManager>>,
    chat_db: State<'_, Arc<ChatV2Database>>,
) -> Result<MemoryWriteOutput, String> {
    let service = get_memory_service(&vfs_db, &lance_store, &llm_manager);
    validate_note_mutable(
        chat_db.inner().as_ref(),
        &service,
        &note_id,
        group_id.as_deref(),
        group_name.as_deref(),
        admin_all,
    )?;
    let result = service
        .update_by_id(&note_id, title.as_deref(), content.as_deref())
        .map_err(|e| e.to_string())?;

    // ★ P2-2 修复：更新后立即触发索引，保证 write-then-search SLA
    trigger_immediate_index(
        Arc::clone(vfs_db.inner()),
        Arc::clone(llm_manager.inner()),
        Arc::clone(lance_store.inner()),
        result.resource_id.clone(),
    );

    service.spawn_post_write_maintenance_for_paths(scoped_maintenance_paths(
        &service,
        Some(&note_id),
        None,
        group_id.as_deref(),
        group_name.as_deref(),
        admin_all,
    ));

    Ok(result)
}

// ★ 修复风险3：删除记忆
#[tauri::command]
pub async fn memory_delete(
    note_id: String,
    group_id: Option<String>,
    group_name: Option<String>,
    admin_all: Option<bool>,
    vfs_db: State<'_, Arc<VfsDatabase>>,
    lance_store: State<'_, Arc<VfsLanceStore>>,
    llm_manager: State<'_, Arc<LLMManager>>,
    chat_db: State<'_, Arc<ChatV2Database>>,
) -> Result<(), String> {
    let service = get_memory_service(&vfs_db, &lance_store, &llm_manager);
    validate_note_mutable(
        chat_db.inner().as_ref(),
        &service,
        &note_id,
        group_id.as_deref(),
        group_name.as_deref(),
        admin_all,
    )?;
    service.delete(&note_id).await.map_err(|e| e.to_string())?;
    service.spawn_post_write_maintenance_for_paths(scoped_maintenance_paths(
        &service,
        None,
        None,
        group_id.as_deref(),
        group_name.as_deref(),
        admin_all,
    ));
    Ok(())
}

#[tauri::command]
pub async fn memory_set_auto_create_subfolders(
    enabled: bool,
    vfs_db: State<'_, Arc<VfsDatabase>>,
    lance_store: State<'_, Arc<VfsLanceStore>>,
    llm_manager: State<'_, Arc<LLMManager>>,
) -> Result<(), String> {
    let service = get_memory_service(&vfs_db, &lance_store, &llm_manager);
    let cfg = super::config::MemoryConfig::new(service.vfs_db_ref().clone());
    cfg.set_auto_create_subfolders(enabled)
        .map_err(|e| e.to_string())
}

#[tauri::command]
pub async fn memory_set_default_category(
    category: String,
    vfs_db: State<'_, Arc<VfsDatabase>>,
    lance_store: State<'_, Arc<VfsLanceStore>>,
    llm_manager: State<'_, Arc<LLMManager>>,
) -> Result<(), String> {
    let service = get_memory_service(&vfs_db, &lance_store, &llm_manager);
    let cfg = super::config::MemoryConfig::new(service.vfs_db_ref().clone());
    cfg.set_default_category(&category)
        .map_err(|e| e.to_string())
}

#[tauri::command]
pub async fn memory_set_auto_extract_frequency(
    frequency: String,
    vfs_db: State<'_, Arc<VfsDatabase>>,
    lance_store: State<'_, Arc<VfsLanceStore>>,
    llm_manager: State<'_, Arc<LLMManager>>,
) -> Result<(), String> {
    let freq = match frequency.trim().to_lowercase().as_str() {
        "off" => super::config::AutoExtractFrequency::Off,
        "balanced" => super::config::AutoExtractFrequency::Balanced,
        "aggressive" => super::config::AutoExtractFrequency::Aggressive,
        other => {
            return Err(format!(
                "Invalid auto extract frequency '{}', expected one of: off, balanced, aggressive",
                other
            ));
        }
    };
    let service = get_memory_service(&vfs_db, &lance_store, &llm_manager);
    let cfg = super::config::MemoryConfig::new(service.vfs_db_ref().clone());
    cfg.set_auto_extract_frequency(freq)
        .map_err(|e| e.to_string())
}

#[derive(Debug, Serialize)]
#[serde(rename_all = "camelCase")]
pub struct MemoryExportItem {
    pub title: String,
    pub content: String,
    pub folder: String,
    pub updated_at: String,
}

#[tauri::command]
pub async fn memory_export_all(
    folder_paths: Option<Vec<String>>,
    group_id: Option<String>,
    group_name: Option<String>,
    admin_all: Option<bool>,
    vfs_db: State<'_, Arc<VfsDatabase>>,
    lance_store: State<'_, Arc<VfsLanceStore>>,
    llm_manager: State<'_, Arc<LLMManager>>,
    chat_db: State<'_, Arc<ChatV2Database>>,
) -> Result<Vec<MemoryExportItem>, String> {
    let service = get_memory_service(&vfs_db, &lance_store, &llm_manager);
    let scoped_paths = scoped_read_paths(
        chat_db.inner().as_ref(),
        None,
        folder_paths,
        group_id.as_deref(),
        group_name.as_deref(),
        admin_all,
    )?;
    let items = if let Some(paths) = scoped_paths {
        service
            .list_folder_paths(&paths, 500, 0)
            .map_err(|e| e.to_string())?
    } else {
        service.list(None, 500, 0).map_err(|e| e.to_string())?
    };

    let mut results = Vec::with_capacity(items.len());
    for item in &items {
        let content = service
            .read(&item.id)
            .map_err(|e| e.to_string())?
            .map(|(_, c)| c)
            .unwrap_or_default();
        results.push(MemoryExportItem {
            title: item.title.clone(),
            content,
            folder: item.folder_path.clone(),
            updated_at: item.updated_at.clone(),
        });
    }
    Ok(results)
}

#[derive(Debug, Serialize)]
#[serde(rename_all = "camelCase")]
pub struct MemoryProfileSection {
    pub category: String,
    pub content: String,
}

#[tauri::command]
pub async fn memory_get_profile(
    folder_paths: Option<Vec<String>>,
    group_id: Option<String>,
    group_name: Option<String>,
    admin_all: Option<bool>,
    vfs_db: State<'_, Arc<VfsDatabase>>,
    lance_store: State<'_, Arc<VfsLanceStore>>,
    llm_manager: State<'_, Arc<LLMManager>>,
    chat_db: State<'_, Arc<ChatV2Database>>,
) -> Result<Vec<MemoryProfileSection>, String> {
    let service = get_memory_service(&vfs_db, &lance_store, &llm_manager);
    let root_id = match service.get_root_folder_id().map_err(|e| e.to_string())? {
        Some(id) => id,
        None => return Ok(vec![]),
    };

    let cat_mgr = super::category_manager::MemoryCategoryManager::new(
        service.vfs_db_ref().clone(),
        llm_manager.inner().clone(),
    );

    let scoped_paths = scoped_read_paths(
        chat_db.inner().as_ref(),
        None,
        folder_paths,
        group_id.as_deref(),
        group_name.as_deref(),
        admin_all,
    )?;
    let is_all_profile = scoped_paths.is_none();
    let categories = if let Some(scope_paths) = scoped_paths.as_ref() {
        cat_mgr
            .load_category_summaries_for_paths(&root_id, scope_paths)
            .map_err(|e| e.to_string())?
    } else {
        cat_mgr
            .load_all_category_summaries(&root_id)
            .map_err(|e| e.to_string())?
    };

    if !categories.is_empty() {
        return Ok(categories
            .into_iter()
            .map(|(cat, content)| MemoryProfileSection {
                category: cat,
                content,
            })
            .collect());
    }

    if !is_all_profile {
        return Ok(vec![]);
    }

    match service.get_profile_summary().map_err(|e| e.to_string())? {
        Some(profile) => Ok(vec![MemoryProfileSection {
            category: "画像".to_string(),
            content: profile,
        }]),
        None => Ok(vec![]),
    }
}

#[tauri::command]
pub async fn memory_write_smart(
    folder_path: Option<String>,
    scope: Option<String>,
    group_id: Option<String>,
    group_name: Option<String>,
    title: String,
    content: String,
    memory_type: Option<String>,
    memory_purpose: Option<String>,
    idempotency_key: Option<String>,
    vfs_db: State<'_, Arc<VfsDatabase>>,
    lance_store: State<'_, Arc<VfsLanceStore>>,
    llm_manager: State<'_, Arc<LLMManager>>,
    chat_db: State<'_, Arc<ChatV2Database>>,
) -> Result<SmartWriteOutput, String> {
    let _ = group_name.as_deref();
    if title.trim().is_empty() {
        return Err("标题不能为空".to_string());
    }
    if content.trim().is_empty() {
        return Err("内容不能为空".to_string());
    }

    let service = get_memory_service(&vfs_db, &lance_store, &llm_manager);
    let mem_type = parse_memory_type(memory_type.as_deref())?;
    let purpose = parse_memory_purpose(memory_purpose.as_deref())?;
    let memory_scope = scope
        .as_deref()
        .map(|s| crate::memory::MemoryScope::from_arg(Some(s)))
        .transpose()?
        .unwrap_or(crate::memory::MemoryScope::Topic);
    let scoped_folder = match memory_scope {
        crate::memory::MemoryScope::Global => Some(crate::memory::join_memory_folder_paths(
            crate::memory::GLOBAL_MEMORY_FOLDER,
            folder_path.as_deref(),
        )),
        crate::memory::MemoryScope::Topic => {
            let (resolved_group_id, resolved_group_name) =
                resolve_active_memory_topic(chat_db.inner().as_ref(), group_id.as_deref())?;
            Some(
                crate::memory::scoped_folder_path(
                    Some(resolved_group_id.as_str()),
                    Some(resolved_group_name.as_str()),
                    crate::memory::MemoryScope::Topic,
                    folder_path.as_deref(),
                )
                .ok_or_else(|| {
                    "memory_write_smart requires group context for topic scope".to_string()
                })?,
            )
        }
    };
    let result = service
        .write_smart_with_source(
            scoped_folder.as_deref(),
            &title,
            &content,
            super::audit_log::MemoryOpSource::Handler,
            None,
            mem_type,
            purpose,
            idempotency_key.as_deref(),
        )
        .await
        .map_err(|e| e.to_string())?;

    if result.event != "NONE" && result.event != "FILTERED" {
        service.spawn_post_write_maintenance_for_paths(scoped_folder.into_iter().collect());
    }

    Ok(result)
}

#[tauri::command]
pub async fn memory_write_batch(
    items: Vec<MemoryBatchWriteItemInput>,
    default_folder_path: Option<String>,
    default_scope: Option<String>,
    group_id: Option<String>,
    group_name: Option<String>,
    default_memory_type: Option<String>,
    default_memory_purpose: Option<String>,
    vfs_db: State<'_, Arc<VfsDatabase>>,
    lance_store: State<'_, Arc<VfsLanceStore>>,
    llm_manager: State<'_, Arc<LLMManager>>,
    chat_db: State<'_, Arc<ChatV2Database>>,
) -> Result<MemoryBatchWriteOutput, String> {
    let _ = group_name.as_deref();
    if items.is_empty() {
        return Ok(MemoryBatchWriteOutput {
            total: 0,
            succeeded: 0,
            failed: 0,
            added: 0,
            updated: 0,
            skipped: 0,
            filtered: 0,
            results: vec![],
        });
    }

    let service = get_memory_service(&vfs_db, &lance_store, &llm_manager);
    let default_type = default_memory_type
        .as_deref()
        .map(|s| parse_memory_type(Some(s)))
        .transpose()?
        .unwrap_or(super::service::MemoryType::Study);
    let default_purpose = parse_memory_purpose(default_memory_purpose.as_deref())?;
    let parsed_default_scope = default_scope
        .as_deref()
        .map(|s| crate::memory::MemoryScope::from_arg(Some(s)))
        .transpose()?
        .unwrap_or(crate::memory::MemoryScope::Topic);
    let resolved_topic = if parsed_default_scope == crate::memory::MemoryScope::Topic
        || items.iter().any(|item| {
            item.scope
                .as_deref()
                .map(|scope| {
                    crate::memory::MemoryScope::from_arg(Some(scope))
                        .map(|scope| scope == crate::memory::MemoryScope::Topic)
                        .unwrap_or(false)
                })
                .unwrap_or(false)
        }) {
        Some(resolve_active_memory_topic(
            chat_db.inner().as_ref(),
            group_id.as_deref(),
        )?)
    } else {
        None
    };

    let mut results = Vec::with_capacity(items.len());
    let mut added = 0usize;
    let mut updated = 0usize;
    let mut skipped = 0usize;
    let mut filtered = 0usize;
    let mut resource_ids = Vec::new();
    let mut maintenance_paths = Vec::new();

    for item in items {
        let mem_type = item
            .memory_type
            .as_deref()
            .map(|s| parse_memory_type(Some(s)))
            .transpose()?
            .unwrap_or(default_type);
        let purpose = parse_memory_purpose(item.memory_purpose.as_deref())?.or(default_purpose);
        let item_scope = item
            .scope
            .as_deref()
            .map(|scope| crate::memory::MemoryScope::from_arg(Some(scope)))
            .transpose()?
            .unwrap_or(parsed_default_scope);
        let raw_folder = item
            .folder_path
            .as_deref()
            .or(default_folder_path.as_deref());
        let scoped_folder = match item_scope {
            crate::memory::MemoryScope::Global => Some(crate::memory::join_memory_folder_paths(
                crate::memory::GLOBAL_MEMORY_FOLDER,
                raw_folder,
            )),
            crate::memory::MemoryScope::Topic => {
                let (resolved_group_id, resolved_group_name) =
                    resolved_topic.as_ref().ok_or_else(|| {
                        "memory_write_batch requires group context for topic scope".to_string()
                    })?;
                Some(
                    crate::memory::scoped_folder_path(
                        Some(resolved_group_id.as_str()),
                        Some(resolved_group_name.as_str()),
                        crate::memory::MemoryScope::Topic,
                        raw_folder,
                    )
                    .ok_or_else(|| {
                        "memory_write_batch requires group context for topic scope".to_string()
                    })?,
                )
            }
        };
        let output = match mem_type {
            super::service::MemoryType::Fact => service
                .write_smart_with_source(
                    scoped_folder.as_deref(),
                    &item.title,
                    &item.content,
                    super::audit_log::MemoryOpSource::Handler,
                    None,
                    mem_type,
                    purpose,
                    item.idempotency_key.as_deref(),
                )
                .await
                .map_err(|e| e.to_string())?,
            _ => service
                .write_explicit_memory(
                    scoped_folder.as_deref(),
                    &item.title,
                    &item.content,
                    mem_type,
                    purpose,
                )
                .map_err(|e| e.to_string())?,
        };

        match output.event.as_str() {
            "ADD" => added += 1,
            "UPDATE" | "APPEND" | "DELETE" => updated += 1,
            "FILTERED" => filtered += 1,
            _ => skipped += 1,
        }
        if let Some(resource_id) = &output.resource_id {
            resource_ids.push(resource_id.clone());
        }
        if matches!(
            output.event.as_str(),
            "ADD" | "UPDATE" | "APPEND" | "DELETE"
        ) {
            if let Some(path) = scoped_folder.as_ref() {
                if !maintenance_paths.contains(path) {
                    maintenance_paths.push(path.clone());
                }
            }
        }
        results.push(MemoryBatchWriteItemResult {
            title: item.title,
            output,
        });
    }

    if !resource_ids.is_empty() {
        for resource_id in resource_ids {
            trigger_immediate_index(
                Arc::clone(vfs_db.inner()),
                Arc::clone(llm_manager.inner()),
                Arc::clone(lance_store.inner()),
                resource_id,
            );
        }
        service.spawn_post_write_maintenance_for_paths(maintenance_paths);
    }

    let succeeded = added + updated;
    Ok(MemoryBatchWriteOutput {
        total: results.len(),
        succeeded,
        failed: filtered,
        added,
        updated,
        skipped,
        filtered,
        results,
    })
}

#[tauri::command]
pub async fn memory_get_audit_logs(
    limit: Option<u32>,
    offset: Option<u32>,
    source_filter: Option<String>,
    operation_filter: Option<String>,
    success_filter: Option<bool>,
    vfs_db: State<'_, Arc<VfsDatabase>>,
) -> Result<Vec<MemoryAuditLogItem>, String> {
    let limit = limit.unwrap_or(50).clamp(1, 200);
    let offset = offset.unwrap_or(0);
    audit_log::query_audit_logs(
        &vfs_db,
        limit,
        offset,
        source_filter.as_deref(),
        operation_filter.as_deref(),
        success_filter,
    )
    .map_err(|e| e.to_string())
}

/// 将记忆导出为 ChatAnki 卡片格式的文档内容
///
/// 筛选记忆后，格式化为结构化文本，返回给前端。
/// 前端可将此文本传入 chatanki_run 工具触发制卡。
#[tauri::command]
pub async fn memory_to_anki_document(
    folder_path: Option<String>,
    folder_paths: Option<Vec<String>>,
    group_id: Option<String>,
    group_name: Option<String>,
    admin_all: Option<bool>,
    purpose_filter: Option<String>,
    limit: Option<u32>,
    vfs_db: State<'_, Arc<VfsDatabase>>,
    lance_store: State<'_, Arc<VfsLanceStore>>,
    llm_manager: State<'_, Arc<LLMManager>>,
    chat_db: State<'_, Arc<ChatV2Database>>,
) -> Result<MemoryAnkiDocument, String> {
    let service = get_memory_service(&vfs_db, &lance_store, &llm_manager);
    let limit = limit.unwrap_or(200).clamp(1, 1000);
    let scoped_paths = scoped_read_paths(
        chat_db.inner().as_ref(),
        folder_path,
        folder_paths,
        group_id.as_deref(),
        group_name.as_deref(),
        admin_all,
    )?;
    let items = if let Some(paths) = scoped_paths {
        service
            .list_folder_paths(&paths, limit, 0)
            .map_err(|e| e.to_string())?
    } else {
        service.list(None, limit, 0).map_err(|e| e.to_string())?
    };

    let purpose = purpose_filter.as_deref();

    let mut lines = Vec::new();
    let mut count = 0usize;

    for item in &items {
        if item.title.starts_with("__") {
            continue;
        }
        if let Some(p) = purpose {
            if item.memory_purpose != p {
                continue;
            }
        }

        let content =
            crate::vfs::repos::note_repo::VfsNoteRepo::get_note_content(&vfs_db, &item.id)
                .map_err(|e| e.to_string())?
                .unwrap_or_default();

        let text = if content.is_empty() {
            &item.title
        } else {
            &content
        };

        lines.push(format!("## {}\n\n{}\n\n---\n", item.title, text));
        count += 1;
    }

    let document_content = if lines.is_empty() {
        String::new()
    } else {
        format!(
            "# 用户记忆知识卡片\n\n以下是从用户记忆库中提取的 {} 条记忆，请为每条生成对应的 Anki 卡片。\n\n{}",
            count,
            lines.join("\n")
        )
    };

    Ok(MemoryAnkiDocument {
        document_content,
        memory_count: count,
        document_name: format!("记忆卡片_{}", chrono::Local::now().format("%Y%m%d")),
    })
}

#[derive(Debug, Serialize, Deserialize)]
#[serde(rename_all = "camelCase")]
pub struct MemoryAnkiDocument {
    pub document_content: String,
    pub memory_count: usize,
    pub document_name: String,
}
