import { readFileSync } from 'node:fs';
import { resolve } from 'node:path';
import { describe, expect, it } from 'vitest';

describe('VFS attachment delete contract', () => {
  const repoSource = readFileSync(
    resolve(process.cwd(), 'src-tauri/src/vfs/repos/attachment_repo.rs'),
    'utf-8'
  );
  const handlerSource = readFileSync(
    resolve(process.cwd(), 'src-tauri/src/vfs/handlers.rs'),
    'utf-8'
  );

  it('routes attachment soft delete through the unified file repository', () => {
    const deleteBody = repoSource.slice(
      repoSource.indexOf('pub fn delete_attachment_with_conn'),
      repoSource.indexOf('/// 恢复软删除的附件')
    );

    expect(deleteBody).toContain('VfsFileRepo::delete_file_with_conn(conn, id)?;');
    expect(deleteBody).not.toContain("UPDATE files SET status = 'deleted'");
    expect(deleteBody).not.toContain('UPDATE folder_items SET deleted_at');
  });

  it('does not reject file-backed attachments by prefix before hitting the repo', () => {
    const handlerBody = handlerSource.slice(
      handlerSource.indexOf('pub async fn vfs_delete_attachment'),
      handlerSource.indexOf('// ============================================================================', handlerSource.indexOf('pub async fn vfs_delete_attachment'))
    );

    expect(handlerBody).toContain('VfsAttachmentRepo::delete_attachment(&vfs_db, &attachment_id)');
    expect(handlerBody).not.toContain('attachment_id.starts_with("att_")');
  });
});
