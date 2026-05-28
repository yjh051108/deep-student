import { readFileSync } from 'node:fs';
import { resolve } from 'node:path';
import { describe, expect, it } from 'vitest';

describe('OCR status effective text contract', () => {
  it('does not treat empty or low-quality OCR page JSON as usable OCR', () => {
    const source = readFileSync(
      resolve(process.cwd(), 'src-tauri/src/vfs/handlers.rs'),
      'utf-8'
    );

    expect(source).toContain('fn ocr_pages_effective_text_stats');
    expect(source).toContain('fn is_usable_ocr_text(text: &str) -> bool');
    expect(source).toContain('VfsChunker::is_text_quality_acceptable(text)');
    expect(source).toContain('.filter(|page| !page.is_failed)');
    expect(source).toContain('has_effective_ocr_pages');
    expect(source).toContain('has_ocr: has_usable_resource_ocr || has_effective_ocr_pages');
    expect(source).toContain('"image" => (resource_ocr_usable, resource_ocr_len)');
  });
});
