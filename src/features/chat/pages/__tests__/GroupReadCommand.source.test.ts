import { existsSync, readFileSync } from 'node:fs';
import path from 'node:path';
import { describe, expect, it } from 'vitest';

const repoRoot = process.cwd();
const groupHandlersPath = path.join(repoRoot, 'src-tauri/src/chat_v2/handlers/group_handlers.rs');
const wailsBridgePath = path.join(repoRoot, 'src/runtime/wailsBridge.ts');

function readSource(absolutePath: string) {
  return existsSync(absolutePath) ? readFileSync(absolutePath, 'utf8') : '';
}

describe('chat group read command source guards', () => {
  it('retires the Rust command and routes chat_v2_get_group through Go', () => {
    const rustSource = readSource(groupHandlersPath);
    const bridgeSource = readSource(wailsBridgePath);
    const bridgeCommand = bridgeSource.match(/if \(command === 'chat_v2_get_group'\)[\s\S]*?return await ChatService\.GetGroup\(groupId\) as T;/)?.[0] ?? '';

    expect(rustSource).not.toContain('pub async fn chat_v2_get_group');
    expect(bridgeCommand).toContain("command === 'chat_v2_get_group'");
    expect(bridgeCommand).toContain('ChatService.GetGroup(groupId)');
    expect(bridgeCommand).not.toContain('ensure_group_folder');
    expect(bridgeCommand).not.toContain('update_group_with_conn');
  });
});
