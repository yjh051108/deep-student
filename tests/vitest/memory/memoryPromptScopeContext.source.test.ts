import { readFileSync } from 'node:fs';
import { resolve } from 'node:path';
import { describe, expect, it } from 'vitest';

describe('memory prompt scope context contract', () => {
  it('injects scope instructions even when a topicless session has no memory summary yet', () => {
    const promptPipeline = readFileSync(
      resolve(process.cwd(), 'src-tauri/src/chat_v2/pipeline/prompt.rs'),
      'utf-8'
    );
    const multiVariantPipeline = readFileSync(
      resolve(process.cwd(), 'src-tauri/src/chat_v2/pipeline/multi_variant.rs'),
      'utf-8'
    );
    const promptBuilder = readFileSync(
      resolve(process.cwd(), 'src-tauri/src/chat_v2/prompt_builder.rs'),
      'utf-8'
    );

    expect(promptPipeline).not.toContain(
      'if global_profile.is_none() && topic_profile.is_none() && topic_root.is_none()'
    );
    expect(multiVariantPipeline).not.toContain(
      'if global_profile.is_none() && topic_profile.is_none() && topic_root.is_none()'
    );
    expect(promptBuilder).toContain('通用无课题会话');
    expect(promptBuilder).toContain('默认只能使用全局记忆');
    expect(promptBuilder).toContain('不得把新记忆写入 topic');
  });
});
