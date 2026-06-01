import { readFileSync } from 'node:fs';
import { resolve } from 'node:path';
import { describe, expect, it } from 'vitest';

describe('learning hub delete state contract', () => {
  const sidebarSource = readFileSync(
    resolve(process.cwd(), 'src/features/learning-hub/LearningHubSidebar.tsx'),
    'utf-8'
  );
  const recentStoreSource = readFileSync(
    resolve(process.cwd(), 'src/features/learning-hub/stores/recentStore.ts'),
    'utf-8'
  );
  const folderHandlersSource = readFileSync(
    resolve(process.cwd(), 'src-tauri/src/dstu/folder_handlers.rs'),
    'utf-8'
  );
  const dstuApiSource = readFileSync(
    resolve(process.cwd(), 'src/dstu/api.ts'),
    'utf-8'
  );
  const dstuHandlersSource = readFileSync(
    resolve(process.cwd(), 'src-tauri/src/dstu/handlers.rs'),
    'utf-8'
  );

  it('removes deleted resources from visible finder state and recent state by id or path', () => {
    expect(recentStoreSource).toContain('removeRecentByIdentity');
    expect(recentStoreSource).toContain('resourceId?: string;');
    expect(recentStoreSource).toContain('const pathId = path?.split');
    expect(recentStoreSource).toContain('i.resourceId !== id');
    expect(recentStoreSource).toContain('i.resourceId !== pathId');
    expect(sidebarSource).toContain('const pruneFinderResource = (resourceId: string, path?: string | null)');
    expect(sidebarSource).toContain('item.resourceId !== resourceId');
    expect(sidebarSource).toContain('(!path || item.path !== path)');
    expect(sidebarSource).toContain('resourceId: item.resourceId');
    expect(sidebarSource).toContain('removeRecentByIdentity(resourceId, event.path)');
    expect(sidebarSource).toContain('pruneFinderResource(resourceId, event.path)');
    expect(sidebarSource).toContain('const visiblePath = resourcePath ?? deletePath');
    expect(sidebarSource).toContain('removeRecentByIdentity(resource.id, visiblePath)');
    expect(sidebarSource).toContain('pruneFinderResource(resource.id, visiblePath)');
  });

  it('deletes resources by canonical path before falling back to stable id', () => {
    expect(sidebarSource).toContain('const resourcePath = resource.path ?? items.find');
    expect(sidebarSource).toContain('const deletePath = resourcePath ?? `/${resource.id}`');
    expect(sidebarSource).toContain('const deleteResult = await dstu.delete(deletePath)');
    expect(sidebarSource).not.toContain('let deletePath = resource.path');
  });

  it('emits delete events for folder cascades instead of leaving recent ghosts', () => {
    const folderDeleteBody = folderHandlersSource.slice(
      folderHandlersSource.indexOf('pub async fn dstu_folder_delete'),
      folderHandlersSource.indexOf('/// 移动文件夹', folderHandlersSource.indexOf('pub async fn dstu_folder_delete'))
    );

    expect(folderHandlersSource).toContain('fn collect_folder_delete_watch_targets(');
    expect(folderHandlersSource).toContain('SELECT id\n            FROM all_folders');
    expect(folderHandlersSource).toContain('VfsFolderRepo::build_folder_path_with_conn(&conn, &id)');
    expect(folderHandlersSource).toContain('VfsFolderRepo::build_resource_path_with_conn(&conn, &folder_item)');
    expect(folderHandlersSource).toContain('fn canonical_file_id_for_folder_item(');
    expect(folderHandlersSource).toContain('f.id = ?1 OR f.resource_id = ?1 OR r.source_id = ?1');
    expect(folderDeleteBody).toContain('collect_folder_delete_watch_targets(&vfs_db, &folder_id)');
    expect(folderDeleteBody).toContain('DstuWatchEvent::deleted(target.path).with_resource(target.id, target.item_type)');
    expect(dstuHandlersSource).toContain('collect_folder_delete_watch_targets_with_conn(&conn, id)');
    expect(dstuHandlersSource).toContain('DstuWatchEvent::deleted(target.path).with_resource(target.id, target.item_type)');
    expect(dstuHandlersSource).toContain('let success_count = parsed_items.len();');
    expect(dstuHandlersSource).toContain('let mut emitted_events = HashSet::new();');
  });

  it('collects batch delete cache keys before the backend removes nodes', () => {
    const deleteManyBody = dstuApiSource.slice(
      dstuApiSource.indexOf('export async function deleteMany'),
      dstuApiSource.indexOf('/**\n * 批量恢复已删除的资源', dstuApiSource.indexOf('export async function deleteMany'))
    );

    expect(deleteManyBody.indexOf('const nodeIds = await collectNodeIdsForInvalidation(paths);'))
      .toBeLessThan(deleteManyBody.indexOf("invoke<number>('dstu_delete_many'"));
  });

  it('resolves /res_* paths before dstu_get enters type-specific repos', () => {
    const getBody = dstuHandlersSource.slice(
      dstuHandlersSource.indexOf('pub async fn dstu_get'),
      dstuHandlersSource.indexOf('/// 列出指定路径下的资源', dstuHandlersSource.indexOf('pub async fn dstu_get'))
    );

    expect(getBody).toContain('matches!(resource_type.as_str(), "resources" | "resource")');
    expect(getBody).toContain('resolve_delete_target_with_conn(&conn, &resource_type, &id)?');
    expect(getBody.indexOf('resolve_delete_target_with_conn(&conn, &resource_type, &id)?'))
      .toBeLessThan(getBody.indexOf('is_hidden_by_deleted_folder_mapping'));
  });
});
