//! 会话分组命令处理器
//!
//! 提供会话分组的 CRUD、排序、会话移动等功能。

use crate::vfs::{VfsDatabase, VfsFolder, VfsFolderRepo, MAX_FOLDER_TITLE_LENGTH};

fn topic_folder_title(name: &str) -> String {
    let mut title: String = name
        .trim()
        .chars()
        .map(|ch| {
            if ch.is_control() || matches!(ch, '\0' | '/' | '\\') {
                '_'
            } else {
                ch
            }
        })
        .take(MAX_FOLDER_TITLE_LENGTH)
        .collect();
    title = title.trim_matches('_').trim().to_string();
    if title.is_empty() {
        "未命名课题".to_string()
    } else {
        title
    }
}

fn first_existing_pinned_folder_id(
    conn: &rusqlite::Connection,
    pinned_resource_ids: &[String],
) -> Result<Option<String>, String> {
    for id in pinned_resource_ids {
        let folder_id = id.trim();
        if !folder_id.starts_with("fld_") {
            continue;
        }
        if VfsFolderRepo::get_folder_with_conn(conn, folder_id)
            .map_err(|e| e.to_string())?
            .is_some()
        {
            return Ok(Some(folder_id.to_string()));
        }
    }
    Ok(None)
}

fn prepend_unique_pinned_folder(
    mut pinned_resource_ids: Vec<String>,
    folder_id: String,
) -> Vec<String> {
    pinned_resource_ids.retain(|id| id.trim() != folder_id);
    pinned_resource_ids.insert(0, folder_id);
    pinned_resource_ids
}

pub(crate) fn ensure_group_folder(
    vfs_db: &VfsDatabase,
    group_name: &str,
    pinned_resource_ids: Vec<String>,
) -> Result<Vec<String>, String> {
    let conn = vfs_db.get_conn_safe().map_err(|e| e.to_string())?;
    let folder_title = topic_folder_title(group_name);

    if let Some(folder_id) = first_existing_pinned_folder_id(&conn, &pinned_resource_ids)? {
        if let Some(mut folder) =
            VfsFolderRepo::get_folder_with_conn(&conn, &folder_id).map_err(|e| e.to_string())?
        {
            let unique_title = VfsFolderRepo::generate_unique_folder_title_with_conn(
                &conn,
                &folder_title,
                folder.parent_id.as_deref(),
                Some(&folder_id),
            )
            .map_err(|e| e.to_string())?;
            if folder.title != unique_title {
                folder.title = unique_title;
                VfsFolderRepo::update_folder_with_conn(&conn, &folder)
                    .map_err(|e| e.to_string())?;
            }
            return Ok(prepend_unique_pinned_folder(pinned_resource_ids, folder_id));
        }
    }

    let unique_title =
        VfsFolderRepo::generate_unique_folder_title_with_conn(&conn, &folder_title, None, None)
            .map_err(|e| e.to_string())?;
    let folder = VfsFolder::new(unique_title, None, Some("folder".to_string()), None);
    VfsFolderRepo::create_folder_with_conn(&conn, &folder).map_err(|e| e.to_string())?;
    Ok(prepend_unique_pinned_folder(pinned_resource_ids, folder.id))
}
