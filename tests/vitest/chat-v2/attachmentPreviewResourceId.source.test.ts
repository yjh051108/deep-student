import { readFileSync } from 'node:fs';
import { resolve } from 'node:path';
import { describe, expect, it } from 'vitest';

describe('attachment preview resource id contract', () => {
  it('resolves chat resource ids before opening the right-side preview panel', () => {
    const source = readFileSync(
      resolve(process.cwd(), 'src/features/chat/pages/useChatPageEvents.ts'),
      'utf-8'
    );

    expect(source).toContain("if (id?.startsWith('res_'))");
    expect(source).toContain("'vfs_get_resource'");
    expect(source).toContain('id = resource.sourceId;');
    expect(source).toContain('setOpenApp({');
  });
});
