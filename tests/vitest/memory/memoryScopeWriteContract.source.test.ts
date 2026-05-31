import { readFileSync } from 'node:fs';
import { resolve } from 'node:path';
import { describe, expect, it } from 'vitest';

describe('memory write scope contract', () => {
  it('does not allow no-topic writes to target arbitrary topic folders', () => {
    const source = readFileSync(
      resolve(process.cwd(), 'src-tauri/src/memory/handlers.rs'),
      'utf-8'
    );

    expect(source).toContain('&& group_name');
    expect(source).toContain('.map(|name| name.trim().is_empty())');
    expect(source).toContain('.unwrap_or(true)');
    expect(source).toContain('validate_memory_path_visible(');
    expect(source).toContain('&[crate::memory::GLOBAL_MEMORY_FOLDER.to_string()]');
    expect(source).toContain('.map(|raw| crate::memory::MemoryScope::from_arg(Some(raw)))');
    expect(source).toContain('.transpose()?');
    expect(source).toContain('Some(crate::memory::MemoryScope::Topic)');
    expect(source).toContain('memory_write requires group context for topic scope');
  });

  it('validates topic write handlers against active chat groups instead of trusting client names', () => {
    const source = readFileSync(
      resolve(process.cwd(), 'src-tauri/src/memory/handlers.rs'),
      'utf-8'
    );

    expect(source).toContain('fn resolve_active_memory_topic(');
    expect(source).toContain('ChatV2Repo::get_group_with_conn');
    expect(source).toContain('group.persist_status != PersistStatus::Active');
    expect(source).toContain('chat_db: State<');
    expect(source).toContain('resolve_memory_write_folder(');
    expect(source).toContain('chat_db.inner().as_ref()');
    expect(source).toContain('resolve_active_memory_topic(chat_db.inner().as_ref(), group_id.as_deref())');
    expect(source).toContain('Some(resolved_group_id.as_str())');
    expect(source).toContain('Some(resolved_group_name.as_str())');
  });

  it('validates topic read handlers against active chat groups instead of trusting client names', () => {
    const source = readFileSync(
      resolve(process.cwd(), 'src-tauri/src/memory/handlers.rs'),
      'utf-8'
    );

    expect(source).toContain('fn scoped_visible_paths(');
    expect(source).toContain('memory topic reads require chat database state');
    expect(source).toContain('resolve_active_memory_topic(chat_db, group_id)');

    for (const fnName of [
      'memory_search',
      'memory_read',
      'memory_list',
      'memory_get_tree',
      'memory_get_related',
      'memory_get_tags',
      'memory_export_all',
      'memory_get_profile',
      'memory_to_anki_document',
    ]) {
      const start = source.indexOf(`pub async fn ${fnName}`);
      expect(start, `${fnName} should exist`).toBeGreaterThan(-1);
      const next = source.indexOf('\n#[tauri::command]', start + 1);
      const body = source.slice(start, next === -1 ? undefined : next);
      expect(body, `${fnName} must receive ChatV2Database state`).toContain('chat_db: State<');
      expect(body, `${fnName} must use active-group read validation`).toContain(
        'chat_db.inner().as_ref()'
      );
    }
  });

  it('uses active group validation for note-id mutation handlers', () => {
    const source = readFileSync(
      resolve(process.cwd(), 'src-tauri/src/memory/handlers.rs'),
      'utf-8'
    );

    expect(source).toContain('fn scoped_mutation_roots(');
    expect(source).toContain('fn validate_note_mutable(');
    expect(source).toContain('fn validate_memory_mutation_path_visible(');
    expect(source).toContain('resolve_active_memory_topic(chat_db, group_id)');

    const mutationFns = [
      'memory_write',
      'memory_add_relation',
      'memory_remove_relation',
      'memory_update_tags',
      'memory_batch_delete',
      'memory_batch_move',
      'memory_move_to_folder',
      'memory_update_by_id',
      'memory_delete',
    ];

    for (const fnName of mutationFns) {
      const start = source.indexOf(`pub async fn ${fnName}`);
      expect(start, `${fnName} should exist`).toBeGreaterThan(-1);
      const next = source.indexOf('\n#[tauri::command]', start + 1);
      const body = source.slice(start, next === -1 ? undefined : next);
      expect(body, `${fnName} must receive ChatV2Database state`).toContain('chat_db: State<');
      expect(body, `${fnName} must not use read-only visibility validation`).not.toContain('validate_note_visible(');
      expect(body, `${fnName} must use active-group mutation validation`).toContain('validate_note_mutable(');
    }
  });

  it('keeps admin-all memory access read-only for mutation handlers', () => {
    const source = readFileSync(
      resolve(process.cwd(), 'src-tauri/src/memory/handlers.rs'),
      'utf-8'
    );

    expect(source).toContain('admin_all memory access is read-only');
    const mutationStart = source.indexOf('fn scoped_mutation_roots(');
    const mutationEnd = source.indexOf('\nfn validate_note_mutable(', mutationStart);
    const mutationHelper = source.slice(mutationStart, mutationEnd);
    expect(mutationHelper).not.toContain('return Ok(None);');
    const writeFolderStart = source.indexOf('fn resolve_memory_write_folder(');
    const writeFolderEnd = source.indexOf('\nfn resolve_active_memory_topic(', writeFolderStart);
    const writeFolderHelper = source.slice(writeFolderStart, writeFolderEnd);
    expect(writeFolderHelper).not.toContain('or(Some(crate::memory::GLOBAL_MEMORY_FOLDER.to_string()))');
    expect(source).toContain('fn reject_admin_all_mutation(');
    expect(source).toContain('admin_all: Option<bool>,\n    title: String,');
    expect(source).toContain('admin_all: Option<bool>,\n    default_memory_type: Option<String>,');
    expect(source).toContain('reject_admin_all_mutation(admin_all)?;');
  });

  it('defaults topicless smart and batch writes to global scope', () => {
    const source = readFileSync(
      resolve(process.cwd(), 'src-tauri/src/memory/handlers.rs'),
      'utf-8'
    );
    const memoryExecutor = readFileSync(
      resolve(process.cwd(), 'src-tauri/src/chat_v2/tools/memory_executor.rs'),
      'utf-8'
    );

    expect(source).toContain('fn default_write_scope(');
    expect(source).toContain('None => Ok(crate::memory::MemoryScope::Global)');
    expect(source).toContain('default_write_scope(scope.as_deref(), group_id.as_deref())?');
    expect(source).toContain('default_write_scope(default_scope.as_deref(), group_id.as_deref())?');
    expect(memoryExecutor).toContain('fn parse_write_scope(');
    expect(memoryExecutor).toContain('Ok(MemoryScope::Global)');
    expect(memoryExecutor).toContain('Self::parse_write_scope(call, ctx)?');
    const smartStart = memoryExecutor.indexOf('async fn execute_write_smart');
    const smartEnd = memoryExecutor.indexOf('\n    async fn execute_write_batch', smartStart);
    const smartBody = memoryExecutor.slice(smartStart, smartEnd);
    expect(smartBody).toContain('Self::parse_write_scope(call, ctx)?');
    expect(smartBody).not.toContain('let scope = Self::parse_scope(call)?;');
  });

  it('uses the verified source folder when delete and move trigger memory maintenance', () => {
    const source = readFileSync(
      resolve(process.cwd(), 'src-tauri/src/memory/handlers.rs'),
      'utf-8'
    );

    expect(source).toContain('fn push_unique_path(');
    expect(source).toContain('let source_folder = validate_note_mutable(');
    expect(source).toContain('service.spawn_post_write_maintenance_for_paths(vec![source_folder]);');
    expect(source).toContain('push_unique_path(&mut maintenance_paths, source_folder);');
    expect(source).not.toContain('None,\n        None,\n        group_id.as_deref(),\n        group_name.as_deref(),\n        admin_all,\n    ));\n    Ok(())');
  });

  it('keeps admin-all profile reads on the all-memory path instead of falling back to global only', () => {
    const source = readFileSync(
      resolve(process.cwd(), 'src-tauri/src/memory/handlers.rs'),
      'utf-8'
    );

    expect(source).toContain('let is_all_profile = scoped_paths.is_none();');
    expect(source).toContain('load_all_category_summaries(&root_id)');
    expect(source).not.toContain(
      'scoped_paths.unwrap_or_else(|| vec![super::scope::GLOBAL_MEMORY_FOLDER.to_string()])'
    );
  });

  it('classifies global memory even when VFS returns a path under the configured memory root', () => {
    const source = readFileSync(
      resolve(process.cwd(), 'src-tauri/src/memory/scope.rs'),
      'utf-8'
    );

    expect(source).toContain('fn folder_path_scope_candidates(');
    expect(source).toContain('for marker in [GLOBAL_MEMORY_FOLDER, TOPIC_MEMORY_PREFIX]');
    expect(source).toContain('if segment == marker');
    expect(source).toContain('classify_folder_scope("长期记忆/全局"');
    expect(source).toContain('classify_folder_scope("长期记忆/课题/g1/经历"');
  });
});
