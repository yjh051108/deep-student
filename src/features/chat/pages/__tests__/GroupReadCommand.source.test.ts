import { existsSync, readFileSync } from 'node:fs';
import path from 'node:path';
import { describe, expect, it } from 'vitest';

const repoRoot = process.cwd();
const groupHandlersPath = path.join(repoRoot, 'src-tauri/src/chat_v2/handlers/group_handlers.rs');

function readSource(absolutePath: string) {
  return existsSync(absolutePath) ? readFileSync(absolutePath, 'utf8') : '';
}

describe('chat group read command source guards', () => {
  it('keeps chat_v2_get_group read-only', () => {
    const source = readSource(groupHandlersPath);
    const command = source.match(/pub async fn chat_v2_get_group[\s\S]*?\n}\n\n\/\/\/ 列出分组/)?.[0] ?? '';

    expect(command).toContain('ChatV2Repo::get_group_with_conn');
    expect(command).not.toContain('ensure_group_folder');
    expect(command).not.toContain('update_group_with_conn');
    expect(command).not.toContain('vfs_db');
  });
});
