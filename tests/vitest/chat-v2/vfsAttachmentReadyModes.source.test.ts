import { readFileSync } from 'node:fs';
import path from 'node:path';
import { describe, expect, it } from 'vitest';

const handlersPath = path.join(process.cwd(), 'src-tauri/src/vfs/handlers.rs');

describe('vfs attachment ready modes source contract', () => {
  it('reports image mode as ready from upload instead of waiting for compression', () => {
    const source = readFileSync(handlersPath, 'utf8');
    const uploadBody = source.slice(
      source.indexOf('pub async fn vfs_upload_attachment'),
      source.indexOf('pub async fn vfs_get_attachment_config')
    );

    expect(uploadBody).toContain('Some(vec!["image".to_string()])');
    expect(uploadBody).toContain('if is_image && processing_status.as_deref() != Some("error")');
    expect(uploadBody).toContain('modes.push("image".to_string())');
    expect(uploadBody).not.toContain('ready_modes = Some(vec![]);');
  });
});
