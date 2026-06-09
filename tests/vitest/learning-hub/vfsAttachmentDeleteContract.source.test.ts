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
  const goVfsServiceSource = readFileSync(
    resolve(process.cwd(), 'desktop-go/internal/vfs/service.go'),
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

  it('keeps Go file delete as a soft delete over hybrid VFS metadata', () => {
    const fileDeleteBody = goVfsServiceSource.slice(
      goVfsServiceSource.indexOf('func (s *Service) DeleteFile('),
      goVfsServiceSource.indexOf('func (s *Service) GetFileContent', goVfsServiceSource.indexOf('func (s *Service) DeleteFile('))
    );

    expect(fileDeleteBody).toContain('findFileLikeResourceIndexByAnyIDLocked(fileID)');
    expect(fileDeleteBody).toContain('"status":    "deleted"');
    expect(fileDeleteBody).toContain('"deletedAt": formatMillis(now)');
    expect(fileDeleteBody).toContain('return s.flushLocked()');
    expect(goVfsServiceSource).toContain('if !ok || resourceIsDeleted(resource)');
    expect(goVfsServiceSource).toContain('return AttachmentContentResult{Found: false}, nil');
  });

  it('handles legacy file-backed folder item types when deleting or restoring files', () => {
    expect(fileRepoSource).toContain('fn folder_item_ids_for_file_with_conn(');
    expect(fileRepoSource).toContain('SELECT f.resource_id, r.source_id');
    expect(fileRepoSource).toContain('LEFT JOIN resources r ON r.id = f.resource_id');
    expect(fileRepoSource).toContain("item_type IN ('file', 'image', 'attachment', 'textbook') AND deleted_at IS NULL");
    expect(fileRepoSource).toContain("item_type IN ('file', 'image', 'attachment', 'textbook') AND deleted_at IS NOT NULL");
    expect(fileRepoSource).toContain('for item_id in &folder_item_ids');
    expect(attachmentRepoSource).toContain('VfsFileRepo::restore_file_with_conn(conn, id)?;');
    expect(attachmentRepoSource).toContain('UPDATE files SET name = ?1, file_name = ?1, updated_at = ?2 WHERE id = ?3');
    expect(attachmentRepoSource).not.toContain("UPDATE folder_items SET deleted_at = NULL");
  });

  it('normalizes restored or deduped attachments into the requested upload folder', () => {
    const uploadWithFolderBody = attachmentRepoSource.slice(
      attachmentRepoSource.indexOf('pub fn upload_with_folder_conn'),
      attachmentRepoSource.indexOf('fn store_inline', attachmentRepoSource.indexOf('pub fn upload_with_folder_conn'))
    );

    expect(uploadWithFolderBody).toContain("item_type IN ('file', 'image', 'attachment', 'textbook')");
    expect(uploadWithFolderBody).toContain('result.attachment.resource_id.as_deref()');
    expect(uploadWithFolderBody).toContain('VfsFolderRepo::add_item_to_folder_with_conn(conn, &folder_item)');
    expect(uploadWithFolderBody).toContain('folder_id');
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
    const getAttachmentBody = goVfsServiceSource.slice(
      goVfsServiceSource.indexOf('func (s *Service) GetAttachment(attachmentID string)'),
      goVfsServiceSource.indexOf('func (s *Service) GetAttachmentContent(attachmentID string)')
    );
    const getAttachmentContentBody = goVfsServiceSource.slice(
      goVfsServiceSource.indexOf('func (s *Service) GetAttachmentContent(attachmentID string)'),
      goVfsServiceSource.indexOf('func (s *Service) UploadFile(', goVfsServiceSource.indexOf('func (s *Service) GetAttachmentContent(attachmentID string)'))
    );

    expect(attachmentRepoSource).toContain('pub fn get_active_by_id(');
    expect(attachmentRepoSource).toContain('pub fn get_active_by_id_with_conn(');
    expect(attachmentRepoSource).toContain("WHERE id = ?1 AND status = 'active' AND deleted_at IS NULL");
    expect(attachmentRepoSource).toContain('Self::get_active_by_id_with_conn(conn, id)?');
    expect(getAttachmentBody).toContain('hasAttachmentLikeAliasLocked(attachmentID)');
    expect(getAttachmentBody).toContain('findFileLikeResourceByAnyIDLocked(attachmentID)');
    expect(getAttachmentBody).toContain('if !ok || resourceIsDeleted(resource)');
    expect(getAttachmentContentBody).toContain('hasAttachmentLikeAliasLocked(attachmentID)');
    expect(getAttachmentContentBody).toContain('findFileLikeResourceByAnyIDLocked(attachmentID)');
    expect(getAttachmentContentBody).toContain('if !ok || resourceIsDeleted(resource)');
    expect(getAttachmentContentBody).toContain('return AttachmentContentResult{Found: false}, nil');
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
    expect(goVfsServiceSource).toContain('func (s *Service) GetFile(fileID string)');
    expect(goVfsServiceSource).toContain('func (s *Service) GetFileContent(fileID string)');
    expect(goVfsServiceSource).toContain('if !ok || resourceIsDeleted(resource)');
    expect(goVfsServiceSource).toContain('return AttachmentContentResult{Found: false}, nil');
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
    const goOcrInfoBody = goVfsServiceSource.slice(
      goVfsServiceSource.indexOf('func (s *Service) GetResourceOcrInfo(resourceID string)'),
      goVfsServiceSource.indexOf('func (s *Service) ClearResourceOcr(resourceID string)')
    );
    const goClearOcrBody = goVfsServiceSource.slice(
      goVfsServiceSource.indexOf('func (s *Service) ClearResourceOcr(resourceID string)'),
      goVfsServiceSource.indexOf('func (s *Service) RagSearch(input VfsRagSearchInput)')
    );

    expect(pendingPdfBody).toContain("AND status = 'active'");
    expect(pendingPdfBody).toContain('AND deleted_at IS NULL');
    expect(resolveBody).toContain("status = 'active' AND deleted_at IS NULL");
    expect(resolveBody).not.toContain('unwrap_or_else(|| id.to_string())');
    expect(pagedBody).toContain("status = 'active' AND deleted_at IS NULL");
    expect(pagedBody).not.toContain('unwrap_or_else(|| id.to_string())');
    expect(handlerSource).not.toContain('pub async fn vfs_get_resource_ocr_info');
    expect(handlerSource).not.toContain('pub async fn vfs_clear_resource_ocr');
    expect(goOcrInfoBody).toContain('if !ok || resourceIsDeleted(resource)');
    expect(goOcrInfoBody).toContain('ActiveSource: "none"');
    expect(goClearOcrBody).toContain('if !ok || resourceIsDeleted(s.state.Resources[index])');
  });

  it('keeps deleted files out of PDF preview image reads', () => {
    const goPdfPageImageBody = goVfsServiceSource.slice(
      goVfsServiceSource.indexOf('func (s *Service) GetPdfPageImage(resourceID string, pageIndex int)'),
      goVfsServiceSource.indexOf('func (s *Service) GetBlobBase64(blobHash string)')
    );

    expect(handlerSource).not.toContain('pub async fn vfs_get_pdf_page_image');
    expect(goPdfPageImageBody).toContain('if !ok || resourceIsDeleted(resource)');
    expect(goPdfPageImageBody).toContain('resource has no PDF page preview data');
  });

  it('keeps deleted files out of reference-resolution content reads', () => {
    const imageOcrBody = refHandlersSource.slice(
      refHandlersSource.indexOf('pub fn get_image_ocr_text_with_conn'),
      refHandlersSource.indexOf('pub fn get_extracted_text_with_conn')
    );
    const existsBody = refHandlersSource.slice(
      refHandlersSource.indexOf('pub fn get_extracted_text_with_conn'),
      refHandlersSource.indexOf('fn get_source_id_type')
    );
    expect(imageOcrBody).toContain("WHERE (a.id = ?1 OR a.resource_id = ?1)");
    expect(imageOcrBody).toContain("AND a.status = 'active' AND a.deleted_at IS NULL");
    expect(imageOcrBody).toContain("AND a.status = 'active' AND a.deleted_at IS NULL AND r.deleted_at IS NULL");
    expect(existsBody).toContain("WHERE (id = ?1 OR resource_id = ?1)");
    expect(existsBody).toContain("AND status = 'active' AND deleted_at IS NULL");
  });
});
