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

  it('routes attachment soft delete through the unified file repository', () => {
    const handlerBody = handlerSource.slice(
      handlerSource.indexOf('pub async fn vfs_delete_attachment'),
      handlerSource.indexOf('// ============================================================================', handlerSource.indexOf('pub async fn vfs_delete_attachment'))
    );

    expect(handlerBody).toContain('VfsFileRepo::delete_file_with_index_cleanup(');
    expect(handlerBody).toContain('lance_store.as_ref()');
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

  it('handles legacy file-backed folder item types when deleting or restoring files', () => {
    expect(fileRepoSource).toContain("item_type IN ('file', 'image', 'attachment', 'textbook') AND deleted_at IS NULL");
    expect(fileRepoSource).toContain("item_type IN ('file', 'image', 'attachment', 'textbook') AND deleted_at IS NOT NULL");
    expect(attachmentRepoSource).toContain("item_type IN ('file', 'image', 'attachment', 'textbook') AND deleted_at IS NOT NULL");
  });
});
