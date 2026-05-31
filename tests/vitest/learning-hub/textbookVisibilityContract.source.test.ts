import { readFileSync } from 'node:fs';
import { resolve } from 'node:path';
import { describe, expect, it } from 'vitest';

describe('Learning Hub textbook visibility contract', () => {
  const sliceFunction = (source: string, name: string, endMarker: string) => {
    const start = source.indexOf(name);
    expect(start).toBeGreaterThanOrEqual(0);
    const end = source.indexOf(endMarker, start + name.length);
    expect(end).toBeGreaterThan(start);
    return source.slice(start, end);
  };

  const textbookRepoSource = readFileSync(
    resolve(process.cwd(), 'src-tauri/src/vfs/repos/textbook_repo.rs'),
    'utf-8'
  );
  const dstuHandlersSource = readFileSync(
    resolve(process.cwd(), 'src-tauri/src/dstu/handlers.rs'),
    'utf-8'
  );
  const crudSource = readFileSync(
    resolve(process.cwd(), 'src-tauri/src/dstu/handler_utils/crud.rs'),
    'utf-8'
  );
  const searchSource = readFileSync(
    resolve(process.cwd(), 'src-tauri/src/dstu/handler_utils/search_helpers.rs'),
    'utf-8'
  );
  const trashHandlersSource = readFileSync(
    resolve(process.cwd(), 'src-tauri/src/dstu/trash_handlers.rs'),
    'utf-8'
  );

  it('keeps deleted file-backed textbooks out of public reads and lists', () => {
    const activeGetBody = sliceFunction(
      textbookRepoSource,
      'pub fn get_active_textbook_with_conn',
      '/// 根据 SHA256 获取教材'
    );
    const listBody = sliceFunction(
      textbookRepoSource,
      'pub fn list_textbooks_with_conn',
      '/// 按关键词列出教材'
    );
    const repoSearchBody = sliceFunction(
      textbookRepoSource,
      'pub fn search_textbooks_with_conn',
      '// ========================================================================'
    );
    const folderListBody = sliceFunction(
      textbookRepoSource,
      'pub fn list_textbooks_by_folder_with_conn',
      'pub fn get_textbook_location'
    );
    const dstuSearchBody = sliceFunction(
      searchSource,
      'pub fn search_textbooks',
      '/// 搜索作文会话'
    );
    const indexSearchBody = sliceFunction(
      searchSource,
      'pub fn search_by_index',
      '/// 根据 source_table'
    );
    const resolveBody = sliceFunction(
      searchSource,
      'fn resolve_source_to_node',
      '_ => {'
    );

    expect(textbookRepoSource).toContain('pub fn get_active_textbook(');
    expect(activeGetBody).toContain("WHERE id = ?1 AND status = 'active' AND deleted_at IS NULL");
    expect(listBody).toContain("WHERE status = 'active'\n              AND deleted_at IS NULL");
    expect(repoSearchBody).toContain("WHERE status = 'active'\n              AND deleted_at IS NULL");
    expect(folderListBody).toContain('AND fi.deleted_at IS NULL');
    expect(folderListBody).toContain('AND t.deleted_at IS NULL');
    expect(dstuHandlersSource).toContain('VfsTextbookRepo::get_active_textbook(&vfs_db, &id)');
    expect(dstuHandlersSource).toContain('VfsTextbookRepo::get_active_textbook(&vfs_db, resource_id)');
    expect(crudSource).toContain('VfsTextbookRepo::get_active_textbook(vfs_db, id)');
    expect(crudSource).toContain('VfsTextbookRepo::get_active_textbook(vfs_db, &item.item_id)');
    expect(crudSource).toContain('VfsTextbookRepo::get_active_textbook(vfs_db, uuid_id)');
    expect(dstuSearchBody).toContain("WHERE status = 'active'\n          AND deleted_at IS NULL");
    expect(indexSearchBody).toContain('JOIN resources r ON r.id = u.resource_id AND r.deleted_at IS NULL');
    expect(resolveBody).toContain('VfsFileRepo::get_active_file(vfs_db, source_id)');
    expect(resolveBody).toContain('VfsTextbookRepo::get_active_textbook(vfs_db, source_id)');
  });

  it('routes textbook soft delete through the unified file delete path', () => {
    const deleteBody = sliceFunction(
      textbookRepoSource,
      'pub fn delete_textbook_with_folder_item_with_conn',
      '/// 永久删除教材'
    );
    const restoreBody = sliceFunction(
      textbookRepoSource,
      'pub fn restore_textbook_with_conn',
      '/// 永久删除教材'
    );

    expect(deleteBody).toContain('VfsFileRepo::delete_file_with_conn(conn, textbook_id)?;');
    expect(deleteBody).not.toContain('UPDATE folder_items SET deleted_at');
    expect(restoreBody).toContain('VfsFileRepo::restore_file_with_conn(conn, textbook_id)?;');
    expect(trashHandlersSource).toContain('"textbook" => VfsFileRepo::delete_file(&db, &id)');
  });
});
