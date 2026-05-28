import { readFileSync } from 'node:fs';
import { resolve } from 'node:path';
import { describe, expect, it } from 'vitest';

describe('VFS upload ready modes source contract', () => {
  it('adds image OCR ready mode only when stored OCR text passes the quality gate', () => {
    const source = readFileSync(
      resolve(process.cwd(), 'src-tauri/src/vfs/handlers.rs'),
      'utf-8'
    );

    expect(source).toContain('let has_usable_ocr_text = if let Some(ref resource_id)');
    expect(source).toContain('"SELECT ocr_text FROM resources WHERE id = ?1"');
    expect(source).toContain('VfsChunker::is_text_quality_acceptable(text)');
    expect(source).toContain('if has_usable_ocr_text {');
    expect(source).not.toContain(
      '"SELECT ocr_text IS NOT NULL AND TRIM(ocr_text) != \'\' FROM resources WHERE id = ?1"'
    );
  });

  it('adds PDF text ready mode only when extracted text passes the quality gate', () => {
    const source = readFileSync(
      resolve(process.cwd(), 'src-tauri/src/vfs/handlers.rs'),
      'utf-8'
    );

    const uploadStart = source.indexOf('pub async fn vfs_upload_attachment');
    expect(uploadStart).toBeGreaterThan(-1);
    const uploadEnd = source.indexOf('\n#[derive(Debug, Clone, Serialize)]', uploadStart);
    const uploadBody = source.slice(uploadStart, uploadEnd);

    expect(uploadBody).toContain('let text_ready = result');
    expect(uploadBody).toContain('VfsChunker::is_text_quality_acceptable(t)');
    expect(uploadBody).not.toContain('.map(|t| !t.trim().is_empty())');
  });
});
