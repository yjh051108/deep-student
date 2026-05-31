import { readFileSync } from 'node:fs';
import { resolve } from 'node:path';
import { describe, expect, it } from 'vitest';

describe('memory and resource scope stale group contract', () => {
  const pipelineSource = readFileSync(
    resolve(process.cwd(), 'src-tauri/src/chat_v2/pipeline.rs'),
    'utf-8'
  );

  it('clears missing or non-active group scope before prompt memory and tools run', () => {
    expect(pipelineSource).toContain('fn enrich_group_scope_for_options');
    expect(pipelineSource).toContain('fn enrich_group_scope_options');
    expect(pipelineSource).toContain('clearing stale topic scope');
    expect(pipelineSource).toContain('options.group_id = None;');
    expect(pipelineSource).toContain('options.group_name = None;');
    expect(pipelineSource).toContain('options.group_pinned_resource_ids = None;');
    expect(pipelineSource).not.toContain('continuing with stored scope only');
    expect(pipelineSource).not.toContain('options.group_name.get_or_insert(group.name)');
  });

  it('does not trust frontend-passed topic folder hints without reloading the active group', () => {
    const helperStart = pipelineSource.indexOf('fn enrich_group_scope_for_options');
    expect(helperStart).toBeGreaterThan(-1);
    const helperEnd = pipelineSource.indexOf('\n    fn enrich_group_scope_options', helperStart);
    const helper = pipelineSource.slice(helperStart, helperEnd);

    expect(helper).toContain('ChatV2Repo::get_group_with_conn');
    expect(helper).toContain('let frontend_group_id = options.group_id.clone();');
    expect(helper).toContain('let session_group_id = ChatV2Repo::get_session_with_conn(&conn, session_id)?');
    expect(helper).toContain('Ignoring frontend group scope');
    expect(helper).toContain('group.persist_status != crate::chat_v2::types::PersistStatus::Active');
    expect(helper).not.toContain('let mut group_id = options.group_id.clone();');
    expect(helper).not.toContain('if group_id.is_none()');
    expect(helper).not.toContain('has_topic_folder');
    expect(helper).not.toContain('needs_group_load');
    expect(helper).not.toContain('if !needs_group_load');
    expect(helper).not.toContain('return Ok(());\n        }\n\n        let conn = self.db.get_conn_safe()?;');
  });

  it('revalidates SendOptions before multi-variant and retry prompt memory/tools run', () => {
    const multiVariantSource = readFileSync(
      resolve(process.cwd(), 'src-tauri/src/chat_v2/pipeline/multi_variant.rs'),
      'utf-8'
    );

    for (const fnName of [
      'execute_multi_variant',
      'execute_variants_retry_batch',
      'execute_variant_retry',
    ]) {
      const fnMarker = `fn ${fnName}`;
      const start = multiVariantSource.indexOf(fnMarker);
      expect(start, `${fnName} should exist`).toBeGreaterThan(-1);
      const nextFn = multiVariantSource.indexOf('\n    ///', start + fnMarker.length);
      const body = multiVariantSource.slice(start, nextFn === -1 ? undefined : nextFn);
      expect(body).toMatch(
        /(?:mut options\s*:\s*SendOptions|mut options\s*=\s*(?:request\.options\.clone\(\)\.unwrap_or_default\(\)|variant\.options\.clone\(\)))/
      );
      expect(body).toContain('self.enrich_group_scope_for_options(&session_id, &mut options)?;');
    }
  });

  it('does not fall back to stale resource tool scope when a chat database is available', () => {
    const resourceScopeSource = readFileSync(
      resolve(process.cwd(), 'src-tauri/src/chat_v2/tools/resource_scope.rs'),
      'utf-8'
    );
    const start = resourceScopeSource.indexOf('fn effective_group_scope');
    expect(start).toBeGreaterThan(-1);
    const end = resourceScopeSource.indexOf('\npub fn current_topic_folder_roots', start);
    const helper = resourceScopeSource.slice(start, end);

    expect(helper).toContain('if ctx.chat_v2_db.is_some()');
    expect(helper).toContain('return None;');
    expect(helper.indexOf('return None;')).toBeLessThan(
      helper.indexOf('pinned_resource_ids: ctx.group_pinned_resource_ids.clone()')
    );
    expect(helper).not.toContain('repair_group_scope(ctx, group_id.trim())');
  });

  it('resolves memory tool topic scope from active database state, not raw context hints', () => {
    const memoryExecutorSource = readFileSync(
      resolve(process.cwd(), 'src-tauri/src/chat_v2/tools/memory_executor.rs'),
      'utf-8'
    );

    expect(memoryExecutorSource).toContain('struct EffectiveMemoryScope');
    expect(memoryExecutorSource).toContain('fn load_active_group_scope');
    expect(memoryExecutorSource).toContain('fn load_active_session_scope');
    expect(memoryExecutorSource).toContain('fn effective_topic_scope');
    expect(memoryExecutorSource).toContain('pub(crate) fn visible_scope_roots_for_context');
    expect(memoryExecutorSource).toContain('PersistStatus::Active');
    expect(memoryExecutorSource).toContain('ctx.chat_v2_db.is_some()');

    const scopeStart = memoryExecutorSource.indexOf('fn effective_topic_scope');
    const scopeEnd = memoryExecutorSource.indexOf('\n    fn topic_memory_root', scopeStart);
    const scopeHelper = memoryExecutorSource.slice(scopeStart, scopeEnd);
    expect(scopeHelper).toContain('if ctx.chat_v2_db.is_some()');
    expect(scopeHelper).toContain('return None;');
    expect(scopeHelper).toContain('group_id: group_id.to_string()');
    expect(scopeHelper).not.toContain('Self::load_active_group_scope(ctx, group_id)');

    const visibleStart = memoryExecutorSource.indexOf('fn visible_scope_roots');
    const visibleEnd = memoryExecutorSource.indexOf('\n    fn scoped_folder_path', visibleStart);
    const visibleHelper = memoryExecutorSource.slice(visibleStart, visibleEnd);
    expect(visibleHelper).toContain('let effective_scope = Self::effective_topic_scope(ctx);');
    expect(visibleHelper).not.toContain('ctx.group_id.as_deref()');
    expect(visibleHelper).not.toContain('ctx.group_name.as_deref()');

    const folderStart = memoryExecutorSource.indexOf('fn scoped_folder_path');
    const folderEnd = memoryExecutorSource.indexOf('\n    fn parse_scope', folderStart);
    const folderHelper = memoryExecutorSource.slice(folderStart, folderEnd);
    expect(folderHelper).toContain('let effective_scope = Self::effective_topic_scope(ctx);');
    expect(folderHelper).not.toContain('ctx.group_id.as_deref()');
    expect(folderHelper).not.toContain('ctx.group_name.as_deref()');
  });

  it('keeps topicless memory list folder reads under global memory by default', () => {
    const memoryExecutorSource = readFileSync(
      resolve(process.cwd(), 'src-tauri/src/chat_v2/tools/memory_executor.rs'),
      'utf-8'
    );

    expect(memoryExecutorSource).toContain('let has_explicit_scope = call.arguments.get("scope").is_some();');
    expect(memoryExecutorSource).toContain('&& Self::topic_memory_root(ctx).is_none()');
    expect(memoryExecutorSource).toContain('MemoryScope::Global');
    expect(memoryExecutorSource).toContain('Self::scoped_folder_path(ctx, effective_scope, folder.as_deref())');
  });

  it('validates explicit retrieval filters against current topic resource scope', () => {
    const retrievalSource = readFileSync(
      resolve(process.cwd(), 'src-tauri/src/chat_v2/tools/builtin_retrieval_executor.rs'),
      'utf-8'
    );

    expect(retrievalSource).toContain(') -> Result<(Option<Vec<String>>, Option<Vec<String>>), String>');
    expect(retrievalSource).toContain('if resource_scope::is_topic_scoped(ctx)');
    expect(retrievalSource).toContain('resource_scope::current_topic_folder_roots(ctx)');
    expect(retrievalSource).toContain('resource_scope::folder_is_within_roots(');
    expect(retrievalSource).toContain('resource_scope::ensure_item_in_scope(');
    expect(retrievalSource).toContain('MemoryToolExecutor::visible_scope_roots_for_context(ctx)');
    expect(retrievalSource).toContain('self.scoped_filters(ctx, vfs_db, explicit_folder_ids, explicit_resource_ids)?');
  });
});
