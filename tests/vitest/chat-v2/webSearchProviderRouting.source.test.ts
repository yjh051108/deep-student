import { readFileSync } from 'node:fs';
import { resolve } from 'node:path';
import { describe, expect, it } from 'vitest';

describe('web search provider routing contract', () => {
  const toolsSource = readFileSync(
    resolve(process.cwd(), 'src-tauri/src/tools/mod.rs'),
    'utf-8'
  );
  const webSearchSource = readFileSync(
    resolve(process.cwd(), 'src-tauri/src/tools/web_search.rs'),
    'utf-8'
  );
  const adapterSource = readFileSync(
    resolve(process.cwd(), 'src/features/chat/adapters/TauriAdapter.ts'),
    'utf-8'
  );
  const builtinRetrievalSource = readFileSync(
    resolve(process.cwd(), 'src-tauri/src/chat_v2/tools/builtin_retrieval_executor.rs'),
    'utf-8'
  );

  it('lets configured provider keys route search without requiring a UI engine toggle', () => {
    expect(toolsSource).not.toContain('请在输入栏选择搜索引擎以启用外部搜索功能');
    expect(toolsSource).toContain('cfg.apply_db_overrides');
    expect(webSearchSource).toContain('cfg.resolve_engine');
    expect(webSearchSource).toContain('first_configured_engine');
  });

  it('does not let normal chat calls force an unconfigured provider', () => {
    expect(toolsSource.replace(/\r\n/g, '\n')).toContain('if !is_test_mode {\n            input.force_engine = None;');
  });

  it('constrains model-visible web search engines to the active user selection', () => {
    expect(adapterSource).toContain('getActiveSearchEngines');
    expect(adapterSource).toContain('enum: activeSearchEngines');
    expect(toolsSource).toContain('LLM 指定引擎');
    expect(toolsSource).toContain('改用用户选择的引擎');
    expect(builtinRetrievalSource).toContain('requested_is_allowed');
    expect(builtinRetrievalSource).toContain('selected_engines[0].clone()');
  });
});
