//! 删除操作辅助函数
//!
//! 包含软删除、恢复、清除等操作的辅助函数

use std::sync::Arc;

use rusqlite::Connection;

use crate::dstu::error::DstuError;
use crate::vfs::{
    repos::VfsMindMapRepo, VfsDatabase, VfsEssayRepo, VfsExamRepo, VfsFileRepo, VfsFolderRepo,
    VfsNoteRepo, VfsTextbookRepo, VfsTranslationRepo,
};

fn helper_error(action: &str, resource_type: &str, id: &str, error: impl ToString) -> String {
    DstuError::vfs_error(format!(
        "{} failed (type={}, id={}): {}",
        action,
        resource_type,
        id,
        error.to_string()
    ))
    .to_string()
}

fn invalid_type_error(resource_type: &str, id: &str) -> String {
    DstuError::invalid_node_type(format!("{} (id={})", resource_type, id)).to_string()
}

fn resource_table_to_delete_type(
    source_table: Option<&str>,
    resource_type: Option<&str>,
) -> Option<&'static str> {
    let table_or_type = [source_table, resource_type]
        .into_iter()
        .flatten()
        .map(str::trim)
        .find(|value| !value.is_empty());

    match table_or_type {
        Some("notes") | Some("note") => Some("notes"),
        Some("textbooks") | Some("textbook") => Some("textbooks"),
        Some("translations") | Some("translation") => Some("translations"),
        Some("exam_sheets") | Some("exams") | Some("exam") => Some("exams"),
        Some("essays") | Some("essay_sessions") | Some("essay") => Some("essays"),
        Some("files") | Some("images") | Some("attachments") | Some("file") | Some("image")
        | Some("attachment") => Some("files"),
        Some("mindmaps") | Some("mindmap") => Some("mindmaps"),
        _ => None,
    }
}

pub(crate) fn resolve_resource_delete_target_with_conn(
    conn: &Connection,
    resource_id: &str,
) -> Result<(String, String), String> {
    let target = conn
        .query_row(
            "SELECT source_table, source_id, type FROM resources WHERE id = ?1",
            rusqlite::params![resource_id],
            |row| {
                Ok((
                    row.get::<_, Option<String>>(0)?,
                    row.get::<_, Option<String>>(1)?,
                    row.get::<_, Option<String>>(2)?,
                ))
            },
        )
        .map_err(|e| {
            helper_error(
                "resolve_resource_delete_target",
                "resources",
                resource_id,
                e,
            )
        })?;

    let (source_table, source_id, resource_type) = target;
    let source_id = source_id
        .filter(|id| !id.trim().is_empty())
        .ok_or_else(|| {
            helper_error(
                "resolve_resource_delete_target",
                "resources",
                resource_id,
                "missing source_id",
            )
        })?;
    let delete_type =
        resource_table_to_delete_type(source_table.as_deref(), resource_type.as_deref())
            .ok_or_else(|| {
                helper_error(
                    "resolve_resource_delete_target",
                    "resources",
                    resource_id,
                    "unsupported source_table",
                )
            })?;

    Ok((delete_type.to_string(), source_id))
}

pub fn resolve_delete_target_with_conn(
    conn: &Connection,
    resource_type: &str,
    id: &str,
) -> Result<(String, String), String> {
    match resource_type {
        "resources" | "resource" => resolve_resource_delete_target_with_conn(conn, id),
        _ => Ok((resource_type.to_string(), id.to_string())),
    }
}

pub fn lookup_resource_id_for_delete_target_with_conn(
    conn: &Connection,
    resource_type: &str,
    id: &str,
) -> Option<String> {
    let (target_type, target_id) = resolve_delete_target_with_conn(conn, resource_type, id).ok()?;
    let sql = match target_type.as_str() {
        "notes" | "note" => Some("SELECT resource_id FROM notes WHERE id = ?1"),
        "textbooks" | "textbook" | "images" | "image" | "files" | "file" | "attachments"
        | "attachment" => Some("SELECT resource_id FROM files WHERE id = ?1"),
        "exams" | "exam" => Some("SELECT resource_id FROM exam_sheets WHERE id = ?1"),
        "essays" | "essay" => Some("SELECT resource_id FROM essays WHERE id = ?1"),
        "translations" | "translation" => {
            Some("SELECT resource_id FROM translations WHERE id = ?1")
        }
        "mindmaps" | "mindmap" => Some("SELECT resource_id FROM mindmaps WHERE id = ?1"),
        _ => None,
    }?;

    conn.query_row(sql, rusqlite::params![target_id], |row| {
        row.get::<_, Option<String>>(0)
    })
    .ok()
    .flatten()
}

/// 根据资源类型执行软删除
pub fn delete_resource_by_type(
    vfs_db: &Arc<VfsDatabase>,
    resource_type: &str,
    id: &str,
) -> Result<(), String> {
    match resource_type {
        "notes" | "note" => {
            VfsNoteRepo::delete_note_with_folder_item(vfs_db, id)
                .map_err(|e| helper_error("delete", resource_type, id, e))?;
            log::info!(
                "[DSTU::delete_helpers] delete_resource_by_type: SUCCESS - type=note, id={}",
                id
            );
        }
        "textbooks" | "textbook" => {
            VfsTextbookRepo::delete_textbook_with_folder_item(vfs_db, id)
                .map_err(|e| helper_error("delete", resource_type, id, e))?;
            log::info!(
                "[DSTU::delete_helpers] delete_resource_by_type: SUCCESS - type=textbook, id={}",
                id
            );
        }
        "translations" | "translation" => {
            VfsTranslationRepo::delete_translation_with_folder_item(vfs_db, id)
                .map_err(|e| helper_error("delete", resource_type, id, e))?;
            log::info!(
                "[DSTU::delete_helpers] delete_resource_by_type: SUCCESS - type=translation, id={}",
                id
            );
        }
        "exams" | "exam" => {
            VfsExamRepo::delete_exam_sheet_with_folder_item(vfs_db, id)
                .map_err(|e| helper_error("delete", resource_type, id, e))?;
            log::info!(
                "[DSTU::delete_helpers] delete_resource_by_type: SUCCESS - type=exam, id={}",
                id
            );
        }
        "essays" | "essay" => {
            if id.starts_with("essay_session_") {
                VfsEssayRepo::delete_session_with_folder_item(vfs_db, id)
                    .map_err(|e| helper_error("delete", resource_type, id, e))?;
                log::info!(
                    "[DSTU::delete_helpers] delete_resource_by_type: SUCCESS - type=essay_session, id={}",
                    id
                );
            } else {
                VfsEssayRepo::delete_essay_with_folder_item(vfs_db, id)
                    .map_err(|e| helper_error("delete", resource_type, id, e))?;
                log::info!(
                    "[DSTU::delete_helpers] delete_resource_by_type: SUCCESS - type=essay, id={}",
                    id
                );
            }
        }
        "folders" | "folder" => {
            VfsFolderRepo::delete_folder(vfs_db, id)
                .map_err(|e| helper_error("delete", resource_type, id, e))?;
            log::info!(
                "[DSTU::delete_helpers] delete_resource_by_type: SUCCESS - type=folder, id={}",
                id
            );
        }
        "images" | "files" | "attachments" | "image" | "file" | "attachment" => {
            // P0-FIX: 使用软删除而非硬删除，支持回收站恢复
            VfsFileRepo::delete_file(vfs_db, id)
                .map_err(|e| helper_error("delete", resource_type, id, e))?;
            log::info!(
                "[DSTU::delete_helpers] delete_resource_by_type: SUCCESS - type=file, id={}",
                id
            );
        }
        "resources" | "resource" => {
            let conn = vfs_db
                .get_conn_safe()
                .map_err(|e| helper_error("delete", resource_type, id, e))?;
            let (target_type, target_id) =
                resolve_delete_target_with_conn(&conn, resource_type, id)?;
            drop(conn);
            delete_resource_by_type(vfs_db, &target_type, &target_id)?;
            log::info!(
                "[DSTU::delete_helpers] delete_resource_by_type: SUCCESS - type=resource, id={}, target_type={}, target_id={}",
                id,
                target_type,
                target_id
            );
        }
        "mindmaps" | "mindmap" => {
            VfsMindMapRepo::delete_mindmap(vfs_db, id)
                .map_err(|e| helper_error("delete", resource_type, id, e))?;
            log::info!(
                "[DSTU::delete_helpers] delete_resource_by_type: SUCCESS - type=mindmap, id={}",
                id
            );
        }
        _ => {
            return Err(invalid_type_error(resource_type, id));
        }
    }
    Ok(())
}

/// 根据资源类型执行软删除（使用现有连接，支持外部事务）
///
/// ★ CONC-08 修复：供批量删除使用，支持在事务中调用
pub fn delete_resource_by_type_with_conn(
    conn: &Connection,
    resource_type: &str,
    id: &str,
) -> Result<(), String> {
    match resource_type {
        "notes" | "note" => {
            VfsNoteRepo::delete_note_with_folder_item_with_conn(conn, id)
                .map_err(|e| helper_error("delete_with_conn", resource_type, id, e))?;
            log::info!(
                "[DSTU::delete_helpers] delete_resource_by_type_with_conn: SUCCESS - type=note, id={}",
                id
            );
        }
        "textbooks" | "textbook" => {
            VfsTextbookRepo::delete_textbook_with_folder_item_with_conn(conn, id)
                .map_err(|e| helper_error("delete_with_conn", resource_type, id, e))?;
            log::info!(
                "[DSTU::delete_helpers] delete_resource_by_type_with_conn: SUCCESS - type=textbook, id={}",
                id
            );
        }
        "translations" | "translation" => {
            VfsTranslationRepo::delete_translation_with_folder_item_with_conn(conn, id)
                .map_err(|e| e.to_string())?;
            log::info!(
                "[DSTU::delete_helpers] delete_resource_by_type_with_conn: SUCCESS - type=translation, id={}",
                id
            );
        }
        "exams" | "exam" => {
            VfsExamRepo::delete_exam_sheet_with_folder_item_with_conn(conn, id)
                .map_err(|e| e.to_string())?;
            log::info!(
                "[DSTU::delete_helpers] delete_resource_by_type_with_conn: SUCCESS - type=exam, id={}",
                id
            );
        }
        "essays" | "essay" => {
            if id.starts_with("essay_session_") {
                VfsEssayRepo::delete_session_with_folder_item_with_conn(conn, id)
                    .map_err(|e| e.to_string())?;
                log::info!(
                    "[DSTU::delete_helpers] delete_resource_by_type_with_conn: SUCCESS - type=essay_session, id={}",
                    id
                );
            } else {
                VfsEssayRepo::delete_essay_with_folder_item_with_conn(conn, id)
                    .map_err(|e| e.to_string())?;
                log::info!(
                    "[DSTU::delete_helpers] delete_resource_by_type_with_conn: SUCCESS - type=essay, id={}",
                    id
                );
            }
        }
        "folders" | "folder" => {
            VfsFolderRepo::delete_folder_with_conn(conn, id).map_err(|e| e.to_string())?;
            log::info!(
                "[DSTU::delete_helpers] delete_resource_by_type_with_conn: SUCCESS - type=folder, id={}",
                id
            );
        }
        "images" | "files" | "attachments" | "image" | "file" | "attachment" => {
            // P0-FIX: 使用软删除而非硬删除，支持回收站恢复
            VfsFileRepo::delete_file_with_conn(conn, id).map_err(|e| e.to_string())?;
            log::info!(
                "[DSTU::delete_helpers] delete_resource_by_type_with_conn: SUCCESS - type=file, id={}",
                id
            );
        }
        "resources" | "resource" => {
            let (target_type, target_id) = resolve_resource_delete_target_with_conn(conn, id)?;
            delete_resource_by_type_with_conn(conn, &target_type, &target_id)?;
            log::info!(
                "[DSTU::delete_helpers] delete_resource_by_type_with_conn: SUCCESS - type=resource, id={}, target_type={}, target_id={}",
                id,
                target_type,
                target_id
            );
        }
        "mindmaps" | "mindmap" => {
            VfsMindMapRepo::delete_mindmap_with_conn(conn, id).map_err(|e| e.to_string())?;
            log::info!(
                "[DSTU::delete_helpers] delete_resource_by_type_with_conn: SUCCESS - type=mindmap, id={}",
                id
            );
        }
        _ => {
            return Err(invalid_type_error(resource_type, id));
        }
    }
    Ok(())
}

/// 根据资源类型执行永久删除
pub fn purge_resource_by_type(
    vfs_db: &Arc<VfsDatabase>,
    resource_type: &str,
    id: &str,
) -> Result<(), String> {
    match resource_type {
        "notes" | "note" => {
            VfsNoteRepo::purge_note(vfs_db, id)
                .map_err(|e| helper_error("purge", resource_type, id, e))?;
        }
        "textbooks" | "textbook" => {
            VfsTextbookRepo::purge_textbook(vfs_db, id)
                .map_err(|e| helper_error("purge", resource_type, id, e))?;
        }
        "translations" | "translation" => {
            VfsTranslationRepo::purge_translation(vfs_db, id).map_err(|e| e.to_string())?;
        }
        "exams" | "exam" => {
            VfsExamRepo::purge_exam_sheet(vfs_db, id).map_err(|e| e.to_string())?;
        }
        "essays" | "essay" => {
            if id.starts_with("essay_session_") {
                // 会话没有软删除依赖，直接永久删除（同时删除其所有轮次）
                let _ = VfsEssayRepo::purge_session(vfs_db, id)
                    .map_err(|e| helper_error("purge", resource_type, id, e))?;
                // 兜底清理 folder_items（如果存在）
                let _ = VfsFolderRepo::remove_item_by_item_id(vfs_db, "essay", id);
            } else {
                VfsEssayRepo::purge_essay(vfs_db, id)
                    .map_err(|e| helper_error("purge", resource_type, id, e))?;
            }
        }
        "folders" | "folder" => {
            VfsFolderRepo::purge_folder(vfs_db, id)
                .map_err(|e| helper_error("purge", resource_type, id, e))?;
        }
        "images" | "files" | "attachments" | "image" | "file" | "attachment" => {
            VfsFileRepo::purge_file(vfs_db, id)
                .map_err(|e| helper_error("purge", resource_type, id, e))?;
        }
        "resources" | "resource" => {
            let conn = vfs_db
                .get_conn_safe()
                .map_err(|e| helper_error("purge", resource_type, id, e))?;
            let (target_type, target_id) =
                resolve_delete_target_with_conn(&conn, resource_type, id)?;
            drop(conn);
            purge_resource_by_type(vfs_db, &target_type, &target_id)?;
        }
        "mindmaps" | "mindmap" => {
            VfsMindMapRepo::purge_mindmap(vfs_db, id)
                .map_err(|e| helper_error("purge", resource_type, id, e))?;
        }
        _ => {
            return Err(invalid_type_error(resource_type, id));
        }
    }
    log::info!(
        "[DSTU::delete_helpers] purge_resource_by_type: SUCCESS - type={}, id={}",
        resource_type,
        id
    );
    Ok(())
}

/// 根据资源类型执行恢复
pub fn restore_resource_by_type(
    vfs_db: &Arc<VfsDatabase>,
    resource_type: &str,
    id: &str,
) -> Result<(), String> {
    match resource_type {
        "notes" | "note" => {
            VfsNoteRepo::restore_note(vfs_db, id)
                .map_err(|e| helper_error("restore", resource_type, id, e))?;
        }
        "textbooks" | "textbook" => {
            VfsTextbookRepo::restore_textbook(vfs_db, id)
                .map_err(|e| helper_error("restore", resource_type, id, e))?;
        }
        "translations" | "translation" => {
            VfsTranslationRepo::restore_translation(vfs_db, id).map_err(|e| e.to_string())?;
        }
        "exams" | "exam" => {
            VfsExamRepo::restore_exam(vfs_db, id).map_err(|e| e.to_string())?;
        }
        "essays" | "essay" => {
            if id.starts_with("essay_session_") {
                VfsEssayRepo::restore_session(vfs_db, id).map_err(|e| e.to_string())?;
            } else {
                VfsEssayRepo::restore_essay(vfs_db, id).map_err(|e| e.to_string())?;
            }
        }
        "folders" | "folder" => {
            VfsFolderRepo::restore_folder(vfs_db, id).map_err(|e| e.to_string())?;
        }
        "images" | "files" | "attachments" | "image" | "file" | "attachment" => {
            // P0-FIX: 支持从回收站恢复文件
            VfsFileRepo::restore_file(vfs_db, id).map_err(|e| e.to_string())?;
        }
        "resources" | "resource" => {
            let conn = vfs_db
                .get_conn_safe()
                .map_err(|e| helper_error("restore", resource_type, id, e))?;
            let (target_type, target_id) =
                resolve_delete_target_with_conn(&conn, resource_type, id)?;
            drop(conn);
            restore_resource_by_type(vfs_db, &target_type, &target_id)?;
        }
        "mindmaps" | "mindmap" => {
            VfsMindMapRepo::restore_mindmap(vfs_db, id).map_err(|e| e.to_string())?;
        }
        _ => {
            return Err(invalid_type_error(resource_type, id));
        }
    }
    log::info!(
        "[DSTU::delete_helpers] restore_resource_by_type: SUCCESS - type={}, id={}",
        resource_type,
        id
    );
    Ok(())
}

/// 根据资源类型执行恢复（使用现有连接，用于事务批量操作）
///
/// ★ CONC-09 修复：支持在事务中批量恢复资源
pub fn restore_resource_by_type_with_conn(
    conn: &Connection,
    resource_type: &str,
    id: &str,
) -> Result<(), String> {
    match resource_type {
        "notes" | "note" => {
            VfsNoteRepo::restore_note_with_conn(conn, id)
                .map_err(|e| helper_error("restore_with_conn", resource_type, id, e))?;
        }
        "textbooks" | "textbook" => {
            VfsTextbookRepo::restore_textbook_with_conn(conn, id)
                .map_err(|e| helper_error("restore_with_conn", resource_type, id, e))?;
        }
        "translations" | "translation" => {
            VfsTranslationRepo::restore_translation_with_conn(conn, id)
                .map_err(|e| e.to_string())?;
        }
        "exams" | "exam" => {
            VfsExamRepo::restore_exam_with_conn(conn, id).map_err(|e| e.to_string())?;
        }
        "essays" | "essay" => {
            if id.starts_with("essay_session_") {
                VfsEssayRepo::restore_session_with_conn(conn, id).map_err(|e| e.to_string())?;
            } else {
                VfsEssayRepo::restore_essay_with_conn(conn, id).map_err(|e| e.to_string())?;
            }
        }
        "folders" | "folder" => {
            VfsFolderRepo::restore_folder_with_conn(conn, id).map_err(|e| e.to_string())?;
        }
        "images" | "files" | "attachments" | "image" | "file" | "attachment" => {
            VfsFileRepo::restore_file_with_conn(conn, id).map_err(|e| e.to_string())?;
        }
        "resources" | "resource" => {
            let (target_type, target_id) =
                resolve_delete_target_with_conn(conn, resource_type, id)?;
            restore_resource_by_type_with_conn(conn, &target_type, &target_id)?;
        }
        "mindmaps" | "mindmap" => {
            let _ =
                VfsMindMapRepo::restore_mindmap_with_conn(conn, id).map_err(|e| e.to_string())?;
        }
        _ => {
            return Err(invalid_type_error(resource_type, id));
        }
    }
    log::info!(
        "[DSTU::delete_helpers] restore_resource_by_type_with_conn: SUCCESS - type={}, id={}",
        resource_type,
        id
    );
    Ok(())
}

#[cfg(test)]
mod tests {
    use super::*;

    fn resource_conn() -> Connection {
        let conn = Connection::open_in_memory().expect("open memory db");
        conn.execute(
            "CREATE TABLE resources (
                id TEXT PRIMARY KEY,
                source_table TEXT,
                source_id TEXT,
                type TEXT,
                deleted_at TEXT
            )",
            [],
        )
        .expect("create resources table");
        conn.execute(
            "CREATE TABLE notes (
                id TEXT PRIMARY KEY,
                resource_id TEXT
            )",
            [],
        )
        .expect("create notes table");
        conn.execute(
            "CREATE TABLE essays (
                id TEXT PRIMARY KEY,
                resource_id TEXT
            )",
            [],
        )
        .expect("create essays table");
        conn
    }

    #[test]
    fn resolves_resource_delete_target_from_source_table() {
        let conn = resource_conn();
        conn.execute(
            "INSERT INTO resources (id, source_table, source_id, type, deleted_at)
             VALUES ('res_file', 'files', 'file_1', 'image', NULL)",
            [],
        )
        .expect("insert resource");

        assert_eq!(
            resolve_resource_delete_target_with_conn(&conn, "res_file").expect("resolve target"),
            ("files".to_string(), "file_1".to_string())
        );
    }

    #[test]
    fn resolves_resource_delete_target_from_resource_type_when_table_is_missing() {
        let conn = resource_conn();
        conn.execute(
            "INSERT INTO resources (id, source_table, source_id, type, deleted_at)
             VALUES ('res_note', NULL, 'note_1', 'note', NULL)",
            [],
        )
        .expect("insert resource");

        assert_eq!(
            resolve_resource_delete_target_with_conn(&conn, "res_note").expect("resolve target"),
            ("notes".to_string(), "note_1".to_string())
        );
    }

    #[test]
    fn resolves_resource_delete_target_from_resource_type_when_table_is_blank() {
        let conn = resource_conn();
        conn.execute(
            "INSERT INTO resources (id, source_table, source_id, type, deleted_at)
             VALUES ('res_image', ' ', 'image_1', 'image', NULL)",
            [],
        )
        .expect("insert resource");

        assert_eq!(
            resolve_resource_delete_target_with_conn(&conn, "res_image").expect("resolve target"),
            ("files".to_string(), "image_1".to_string())
        );
    }

    #[test]
    fn looks_up_resource_id_after_resolving_resource_row() {
        let conn = resource_conn();
        conn.execute(
            "INSERT INTO resources (id, source_table, source_id, type, deleted_at)
             VALUES ('res_note', 'notes', 'note_1', 'note', NULL)",
            [],
        )
        .expect("insert resource");
        conn.execute(
            "INSERT INTO notes (id, resource_id) VALUES ('note_1', 'res_note')",
            [],
        )
        .expect("insert note");

        assert_eq!(
            lookup_resource_id_for_delete_target_with_conn(&conn, "resources", "res_note"),
            Some("res_note".to_string())
        );
    }

    #[test]
    fn resolves_deleted_resource_rows_for_restore_and_purge() {
        let conn = resource_conn();
        conn.execute(
            "INSERT INTO resources (id, source_table, source_id, type, deleted_at)
             VALUES ('res_deleted_note', 'notes', 'note_1', 'note', '2026-01-01T00:00:00Z')",
            [],
        )
        .expect("insert deleted resource");

        assert_eq!(
            resolve_resource_delete_target_with_conn(&conn, "res_deleted_note")
                .expect("resolve deleted resource"),
            ("notes".to_string(), "note_1".to_string())
        );
    }

    #[test]
    fn looks_up_essay_resource_id_after_resolving_resource_row() {
        let conn = resource_conn();
        conn.execute(
            "INSERT INTO resources (id, source_table, source_id, type, deleted_at)
             VALUES ('res_essay', 'essays', 'essay_1', 'essay', NULL)",
            [],
        )
        .expect("insert resource");
        conn.execute(
            "INSERT INTO essays (id, resource_id) VALUES ('essay_1', 'res_essay')",
            [],
        )
        .expect("insert essay");

        assert_eq!(
            lookup_resource_id_for_delete_target_with_conn(&conn, "resources", "res_essay"),
            Some("res_essay".to_string())
        );
    }
}
