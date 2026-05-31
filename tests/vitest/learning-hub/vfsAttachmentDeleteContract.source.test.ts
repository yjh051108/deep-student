import { readFileSync } from 'node:fs';
import { resolve } from 'node:path';
import { describe, expect, it } from 'vitest';

describe('VFS attachment delete contract', () => {
  const attachmentRepoSource = readFileSync(
    resolve(process.cwd(), 'src-tauri/src/vfs/repos/attachment_repo.rs'),
    'utf-8'
  );
  const fileRepoSource = readFileSync(
    resolve(process.cwd(), 'src-tauri/src/vfs/repos/file_repo.rs'),
    'utf-8'
  );
  const handlerSource = readFileSync(
    resolve(process.cwd(), 'src-tauri/src/vfs/handlers.rs'),
    'utf-8'
  );
  const contentHelpersSource = readFileSync(
    resolve(process.cwd(), 'src-tauri/src/dstu/handler_utils/content_helpers.rs'),
    'utf-8'
  );
  const dstuHandlersSource = readFileSync(
    resolve(process.cwd(), 'src-tauri/src/dstu/handlers.rs'),
    'utf-8'
  );
  const refHandlersSource = readFileSync(
    resolve(process.cwd(), 'src-tauri/src/vfs/ref_handlers.rs'),
    'utf-8'
  );

  it('routes attachment soft delete through the unified file repository', () => {
    const handlerBody = handlerSource.slice(
      handlerSource.indexOf('pub async fn vfs_delete_attachment'),
      handlerSource.indexOf('// ============================================================================', handlerSource.indexOf('pub async fn vfs_delete_attachment'))
    );

    expect(handlerBody).toContain('VfsFileRepo::delete_file_with_index_cleanup(');
    expect(handlerBody).toContain('lance_store.as_ref()');
    expect(handlerBody).toContain('file_delete_watch_targets(&vfs_db, &attachment_id)');
    expect(handlerBody).toContain('for (event_path, id, item_type) in event_targets');
    expect(handlerBody).not.toContain('VfsAttachmentRepo::delete_attachment(&vfs_db, &attachment_id)');
  });

  it('does not reject file-backed attachments by prefix before hitting the repo', () => {
    const handlerBody = handlerSource.slice(
      handlerSource.indexOf('pub async fn vfs_delete_attachment'),
      handlerSource.indexOf('// ============================================================================', handlerSource.indexOf('pub async fn vfs_delete_attachment'))
    );

    expect(handlerBody).toContain('VfsFileRepo::delete_file_with_index_cleanup(');
    expect(handlerBody).not.toContain('attachment_id.starts_with("att_")');
  });

  it('emits DSTU delete events from native file delete commands', () => {
    const fileDeleteBody = handlerSource.slice(
      handlerSource.indexOf('pub async fn vfs_delete_file'),
      handlerSource.indexOf('#[derive(Debug, Clone, Serialize)]', handlerSource.indexOf('pub async fn vfs_delete_file'))
    );

    expect(handlerSource).toContain('fn file_delete_watch_targets(');
    expect(handlerSource).toContain('SELECT f.file_name, f.resource_id, r.source_id');
    expect(handlerSource).toContain('SELECT item_type, item_id, folder_id');
    expect(handlerSource).toContain('VfsFolderRepo::build_folder_path_with_conn(&conn, id)');
    expect(fileDeleteBody).toContain('file_delete_watch_targets(&vfs_db, &file_id)');
    expect(fileDeleteBody).toContain('emit_watch_event(');
    expect(fileDeleteBody).toContain('DstuWatchEvent::deleted(event_path).with_resource(id, item_type)');
  });

  it('handles legacy file-backed folder item types when deleting or restoring files', () => {
    expect(fileRepoSource).toContain('fn folder_item_ids_for_file_with_conn(');
    expect(fileRepoSource).toContain('SELECT f.resource_id, r.source_id');
    expect(fileRepoSource).toContain('LEFT JOIN resources r ON r.id = f.resource_id');
    expect(fileRepoSource).toContain("item_type IN ('file', 'image', 'attachment', 'textbook') AND deleted_at IS NULL");
    expect(fileRepoSource).toContain("item_type IN ('file', 'image', 'attachment', 'textbook') AND deleted_at IS NOT NULL");
    expect(fileRepoSource).toContain('for item_id in &folder_item_ids');
    expect(attachmentRepoSource).toContain('VfsFileRepo::restore_file_with_conn(conn, id)?;');
    expect(attachmentRepoSource).not.toContain("UPDATE folder_items SET deleted_at = NULL");
  });

  it('uses active-only reads for DSTU folder search to avoid ghost files', () => {
    const searchBody = dstuHandlersSource.slice(
      dstuHandlersSource.indexOf('pub async fn dstu_search_in_folder'),
      dstuHandlersSource.indexOf('// ============================================================================', dstuHandlersSource.indexOf('pub async fn dstu_search_in_folder') + 1)
    );

    expect(searchBody).toContain('VfsTextbookRepo::get_active_textbook(&vfs_db, &item.item_id)');
    expect(searchBody).toContain('"file" | "image"');
    expect(searchBody).toContain('VfsFileRepo::get_active_file(&vfs_db, &item.item_id)');
    expect(searchBody).not.toContain('VfsTextbookRepo::get_textbook(&vfs_db, &item.item_id)');
    expect(searchBody).not.toContain('VfsFileRepo::get_file(&vfs_db, &item.item_id)');
  });

  it('keeps deleted attachments out of public metadata and content reads', () => {
    expect(attachmentRepoSource).toContain('pub fn get_active_by_id(');
    expect(attachmentRepoSource).toContain('pub fn get_active_by_id_with_conn(');
    expect(attachmentRepoSource).toContain("WHERE id = ?1 AND status = 'active' AND deleted_at IS NULL");
    expect(attachmentRepoSource).toContain('Self::get_active_by_id_with_conn(conn, id)?');
    expect(handlerSource).toContain('VfsAttachmentRepo::get_active_by_id(&vfs_db, &attachment_id)');
  });

  it('keeps deleted files out of public metadata and content reads', () => {
    const fileContentBody = fileRepoSource.slice(
      fileRepoSource.indexOf('pub fn get_content_with_conn'),
      fileRepoSource.indexOf('// ========================================================================', fileRepoSource.indexOf('pub fn get_content_with_conn') + 1)
    );

    expect(fileRepoSource).toContain('pub fn get_active_file(');
    expect(fileRepoSource).toContain('pub fn get_active_file_with_conn(');
    expect(fileRepoSource).toContain("WHERE id = ?1 AND status = 'active' AND deleted_at IS NULL");
    expect(fileContentBody).toContain('Self::get_active_file_with_conn(conn, file_id)?');
    expect(fileContentBody).not.toContain('Self::get_file_with_conn(conn, file_id)?');
    expect(fileRepoSource).toContain("WHERE status = 'deleted' OR deleted_at IS NOT NULL");
    expect(fileRepoSource).toContain("SELECT id FROM files WHERE status = 'deleted' OR deleted_at IS NOT NULL");
    expect(handlerSource).toContain('VfsFileRepo::get_active_file(&vfs_db, &file_id)');
    expect(handlerSource).toMatch(/VfsFileRepo::get_active_file_with_conn\(\s*&conn,\s*&file_id\s*\)/);
  });

  it('keeps deleted files out of pending processing and DSTU content reads', () => {
    const pendingPdfBody = handlerSource.slice(
      handlerSource.indexOf('pub async fn vfs_list_pending_pdf_processing'),
      handlerSource.indexOf('// ============================================================================', handlerSource.indexOf('pub async fn vfs_list_pending_pdf_processing'))
    );
    const resolveBody = contentHelpersSource.slice(
      contentHelpersSource.indexOf('fn resolve_file_id_for_read'),
      contentHelpersSource.indexOf('fn format_exam_preview_for_read')
    );
    const pagedBody = contentHelpersSource.slice(
      contentHelpersSource.indexOf('pub fn get_content_by_type_paged'),
      contentHelpersSource.indexOf('pub fn update_content_by_type')
    );
    const ocrInfoBody = handlerSource.slice(
      handlerSource.indexOf('pub async fn vfs_get_resource_ocr_info'),
      handlerSource.indexOf('// ============================================================================', handlerSource.indexOf('pub async fn vfs_get_resource_ocr_info'))
    );

    expect(pendingPdfBody).toContain("AND status = 'active'");
    expect(pendingPdfBody).toContain('AND deleted_at IS NULL');
    expect(resolveBody).toContain("status = 'active' AND deleted_at IS NULL");
    expect(resolveBody).not.toContain('unwrap_or_else(|| id.to_string())');
    expect(pagedBody).toContain("status = 'active' AND deleted_at IS NULL");
    expect(pagedBody).not.toContain('unwrap_or_else(|| id.to_string())');
    expect(ocrInfoBody).toContain('WHERE id = ?1 AND deleted_at IS NULL');
    expect(ocrInfoBody).toContain("status = 'active' AND deleted_at IS NULL");
  });

  it('keeps deleted files out of PDF preview image reads', () => {
    const pdfPageImageBody = handlerSource.slice(
      handlerSource.indexOf('pub async fn vfs_get_pdf_page_image'),
      handlerSource.indexOf('// ============================================================================', handlerSource.indexOf('pub async fn vfs_get_pdf_page_image') + 1)
    );

    expect(pdfPageImageBody).toContain('SELECT EXISTS(SELECT 1 FROM resources WHERE id = ?1 AND deleted_at IS NULL)');
    expect(pdfPageImageBody).toContain("SELECT preview_json FROM files WHERE resource_id = ?1 AND status = 'active' AND deleted_at IS NULL");
  });

  it('keeps deleted files out of reference-resolution content reads', () => {
    const existsBody = refHandlersSource.slice(
      refHandlersSource.indexOf('// 查询资源是否存在'),
      refHandlersSource.indexOf('// 获取资源路径（通过 folder_items 表查找）')
    );
    const attachmentTypeBody = refHandlersSource.slice(
      refHandlersSource.indexOf('fn get_attachment_type_with_conn'),
      refHandlersSource.indexOf('fn get_exam_multimodal_blocks_with_conn')
    );
    expect(existsBody).toContain("SELECT 1 FROM files WHERE id = ?1 AND status = 'active' AND deleted_at IS NULL");
    expect(attachmentTypeBody).toContain("SELECT type FROM files WHERE id = ?1 AND status = 'active' AND deleted_at IS NULL");
    expect(refHandlersSource).toContain("fn get_file_multimodal_blocks_with_conn");
    expect(refHandlersSource).toContain("WHERE (id = ?1 OR resource_id = ?1)\n          AND status = 'active' AND deleted_at IS NULL");
    expect(refHandlersSource).toContain("WHERE (a.id = ?1 OR a.resource_id = ?1)\n          AND a.status = 'active' AND a.deleted_at IS NULL AND r.deleted_at IS NULL");
  });
});
