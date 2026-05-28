import { readFileSync } from 'node:fs';
import { resolve } from 'node:path';
import { describe, expect, it } from 'vitest';

describe('VFS reference OCR quality contract', () => {
  it('does not inject image or PDF OCR text unless it passes the OCR quality gate', () => {
    const source = readFileSync(
      resolve(process.cwd(), 'src-tauri/src/vfs/ref_handlers.rs'),
      'utf-8'
    );

    expect(source).toContain('fn is_usable_ocr_text(text: &str) -> bool');
    expect(source).toContain('VfsChunker::is_text_quality_acceptable(text)');
    expect(source).toContain('Ok(Some(text)) if is_usable_ocr_text(&text)');
    expect(source).toContain('fn filter_usable_ocr_pages');
    expect(source).toContain('fn has_enough_usable_ocr_pages');
    expect(source).toContain('if !has_enough_usable_ocr_pages(&pages)');
    expect(source).toContain('if is_usable_ocr_text(ocr_text)');
  });
});
