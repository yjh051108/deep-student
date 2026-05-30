import { existsSync, readFileSync } from 'node:fs';
import path from 'node:path';
import { describe, expect, it } from 'vitest';

const repoRoot = process.cwd();
const inputBarUIPath = path.join(repoRoot, 'src/features/chat/components/input-bar/InputBarUI.tsx');
const vfsRefApiPath = path.join(repoRoot, 'src/features/chat/context/vfsRefApi.ts');

function readSource(absolutePath: string) {
  return existsSync(absolutePath) ? readFileSync(absolutePath, 'utf8') : '';
}

describe('chat attachment topic scope source guards', () => {
  it('keeps chat input attachments scoped by the current session and group', () => {
    const inputBarSource = readSource(inputBarUIPath);
    const uploadCall = inputBarSource.match(/vfsRefApi\.uploadAttachment\(\{[\s\S]*?\}\);/)?.[0] ?? '';

    expect(inputBarSource).toContain('groupId,');
    expect(uploadCall).toContain('sessionId,');
    expect(uploadCall).toContain('groupId,');
  });

  it('forwards attachment scope fields through the Tauri upload command', () => {
    const source = readSource(vfsRefApiPath);
    const invokeBlock = source.match(/invoke<UploadAttachmentResult>\('vfs_upload_attachment'[\s\S]*?\}\s*\);/)?.[0] ?? '';

    expect(source).toContain('sessionId?: string | null;');
    expect(source).toContain('groupId?: string | null;');
    expect(invokeBlock).toContain('sessionId: params.sessionId');
    expect(invokeBlock).toContain('groupId: params.groupId');
  });
});
