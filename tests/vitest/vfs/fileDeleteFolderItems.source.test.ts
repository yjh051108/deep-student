import { readFileSync } from "node:fs";
import { resolve } from "node:path";
import { describe, expect, it } from "vitest";

describe("VfsFileRepo folder item deletion contract", () => {
  const source = readFileSync(
    resolve(process.cwd(), "src-tauri/src/vfs/repos/file_repo.rs"),
    "utf-8",
  );
  const attachmentSource = readFileSync(
    resolve(process.cwd(), "src-tauri/src/vfs/repos/attachment_repo.rs"),
    "utf-8",
  );
  const dstuHandlersSource = readFileSync(
    resolve(process.cwd(), "src-tauri/src/dstu/handlers.rs"),
    "utf-8",
  );
  const dstuCrudSource = readFileSync(
    resolve(process.cwd(), "src-tauri/src/dstu/handler_utils/crud.rs"),
    "utf-8",
  );
  const dstuListSource = readFileSync(
    resolve(process.cwd(), "src-tauri/src/dstu/handler_utils/list_helpers.rs"),
    "utf-8",
  );
  const nodeConvertersSource = readFileSync(
    resolve(
      process.cwd(),
      "src-tauri/src/dstu/handler_utils/node_converters.rs",
    ),
    "utf-8",
  );
  const vfsHandlersSource = readFileSync(
    resolve(process.cwd(), "src-tauri/src/vfs/handlers.rs"),
    "utf-8",
  );
  const folderRepoSource = readFileSync(
    resolve(process.cwd(), "src-tauri/src/vfs/repos/folder_repo.rs"),
    "utf-8",
  );

  it("soft deletes every folder mapping for the file id, including image mappings", () => {
    expect(source).toContain(
      "UPDATE folder_items SET deleted_at = ?1, updated_at = ?2 WHERE item_id = ?3 AND item_type IN ('file', 'image', 'attachment', 'textbook') AND deleted_at IS NULL",
    );
    expect(source).not.toContain(
      "WHERE item_type = 'file' AND item_id = ?3 AND deleted_at IS NULL",
    );
  });

  it("restores every folder mapping for the file id", () => {
    expect(source).toContain(
      "UPDATE folder_items SET deleted_at = NULL, updated_at = ?1 WHERE item_id = ?2 AND item_type IN ('file', 'image', 'attachment', 'textbook') AND deleted_at IS NOT NULL",
    );
    expect(source).not.toContain(
      "WHERE item_type = 'file' AND item_id = ?2 AND deleted_at IS NOT NULL",
    );
  });

  it("keeps attachment delete and restore in sync with visible folder mappings", () => {
    expect(attachmentSource).toContain(
      "VfsFileRepo::delete_file_with_conn(conn, id)?;",
    );
    expect(attachmentSource).toContain(
      "VfsFileRepo::restore_file_with_conn(conn, id)?;",
    );
    expect(attachmentSource).not.toContain(
      "UPDATE folder_items SET deleted_at = NULL",
    );
  });

  it("does not expose soft-deleted files through DSTU get/fetch paths", () => {
    expect(nodeConvertersSource).toContain(
      "pub fn active_file_to_dstu_node(file: &VfsFile) -> Option<DstuNode>",
    );
    expect(nodeConvertersSource).toContain(
      'file.status == "active" && file.deleted_at.is_none()',
    );
    expect(dstuHandlersSource).toContain(
      "Ok(Some(file)) => active_file_to_dstu_node(&file)",
    );
    expect(dstuCrudSource).toContain(
      "Ok(file.and_then(|f| active_file_to_dstu_node(&f)))",
    );
    expect(dstuCrudSource).toContain(
      "Ok(Some(f)) => Ok(active_file_to_dstu_node(&f))",
    );
  });

  it("does not expose resources whose only folder mappings are deleted", () => {
    expect(dstuCrudSource).toContain(
      "pub fn is_hidden_by_deleted_folder_mapping(",
    );
    expect(dstuCrudSource).toContain(
      "COUNT(*)",
    );
    expect(dstuCrudSource).toContain(
      "AND (fi.folder_id IS NULL OR f.deleted_at IS NULL)",
    );
    expect(dstuHandlersSource).toContain(
      "is_hidden_by_deleted_folder_mapping(&vfs_db, &id)?",
    );
    expect(dstuHandlersSource).toContain(
      "VfsFolderRepo::folder_exists(&vfs_db, &id)",
    );
    expect(dstuListSource).toContain(
      "is_hidden_by_deleted_folder_mapping(vfs_db, &file.id)?",
    );
    expect(dstuListSource).toContain(
      "is_hidden_by_deleted_folder_mapping(vfs_db, &note.id)?",
    );
    expect(dstuListSource).toContain("list_unassigned_notes");
    expect(dstuListSource).toContain(
      "is_hidden_by_deleted_folder_mapping(vfs_db, &textbook.id)?",
    );
    expect(dstuHandlersSource).toContain(
      "is_hidden_by_deleted_folder_mapping(&vfs_db, resource_id)?",
    );
  });

  it("keeps deleted folder children hidden from finder root fallback resources", () => {
    expect(folderRepoSource).toContain("LEFT JOIN folders f ON f.id = fi.folder_id");
    expect(folderRepoSource).toContain("WHERE fi.deleted_at IS NULL");
    expect(folderRepoSource).toContain("OR f.deleted_at IS NOT NULL");
  });

  it("uses the VFS resource id for file nodes and keeps content hashes in metadata", () => {
    expect(nodeConvertersSource).toContain("let resource_id = file");
    expect(nodeConvertersSource).toContain(".resource_id");
    expect(nodeConvertersSource).toContain("DstuNode::resource(&file.id, &path, &display_name, node_type, resource_id)");
    expect(nodeConvertersSource).toContain('"sha256": file.sha256');
    expect(nodeConvertersSource).not.toContain("DstuNode::resource(&file.id, &path, &display_name, node_type, &file.sha256)");
  });

  it("resolves resource paths from active folder mappings without reviving deleted resources", () => {
    const start = vfsHandlersSource.indexOf("pub async fn vfs_get_resource_path");
    const end = vfsHandlersSource.indexOf("/// 批量更新路径缓存", start);
    expect(start).toBeGreaterThanOrEqual(0);
    expect(end).toBeGreaterThan(start);
    const resourcePathSource = vfsHandlersSource.slice(start, end);

    expect(resourcePathSource).toContain(
      "WHERE item_id = ?1 AND cached_path IS NOT NULL\n              AND deleted_at IS NULL",
    );
    expect(resourcePathSource).toContain(
      "WHERE item_id = ?1\n              AND deleted_at IS NULL",
    );
    expect(resourcePathSource).toContain(
      "if let Some(title) = get_active_resource_title_with_conn(&conn, &source_id)?",
    );
    expect(resourcePathSource).toContain(
      'return Ok(format!("/{}", source_id));',
    );
    expect(vfsHandlersSource).toContain(
      "SELECT file_name FROM files WHERE id = ?1 AND status = 'active' AND deleted_at IS NULL",
    );
  });

  it("emits delete watch events with real paths and stable resource ids", () => {
    expect(dstuHandlersSource).toContain(
      "DstuWatchEvent::deleted(&path).with_resource(id.clone(), resource_type.clone())",
    );
    expect(dstuHandlersSource).not.toContain(
      'DstuWatchEvent::deleted(format!("/{id}"))',
    );
    expect(nodeConvertersSource).toContain("pub fn emit_watch_event(window: &Window, event: DstuWatchEvent)");
  });
});
