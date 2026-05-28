import { readFileSync } from 'node:fs';
import { resolve } from 'node:path';
import { describe, expect, it } from 'vitest';

describe('PDF processing service source contract', () => {
  it('records a completed_with_issues OCR problem when OCR produces no usable ready mode', () => {
    const source = readFileSync(
      resolve(process.cwd(), 'src-tauri/src/vfs/pdf_processing_service.rs'),
      'utf-8'
    );

    expect(source).toContain('OCR completed but did not produce enough usable text');
    expect(source).toContain('stage: ProcessingStage::OcrProcessing.as_str().to_string()');
    expect(source).toContain('retriable: true');
  });

  it('uses extracted text quality, not only length, when deciding whether PDF OCR should run', () => {
    const source = readFileSync(
      resolve(process.cwd(), 'src-tauri/src/vfs/pdf_processing_service.rs'),
      'utf-8'
    );

    expect(source).toContain('VfsChunker::is_text_quality_acceptable(text)');
    expect(source).toContain('let has_usable_extracted_text = extracted_text');
    expect(source).toContain('|| !has_usable_extracted_text');
  });

  it('reconciles OCR ready mode only when stored OCR text passes the quality gate', () => {
    const source = readFileSync(
      resolve(process.cwd(), 'src-tauri/src/vfs/pdf_processing_service.rs'),
      'utf-8'
    );

    expect(source).toContain('let has_usable_ocr_text = conn');
    expect(source).toContain('SELECT r.ocr_text');
    expect(source).toContain('VfsChunker::is_text_quality_acceptable(text)');
    expect(source).toContain('if has_usable_ocr_text && !progress.ready_modes.contains(&"ocr".to_string())');
    expect(source).not.toContain("SELECT r.ocr_text IS NOT NULL AND TRIM(r.ocr_text) != ''");
  });

  it('does not treat non-empty image OCR as ready unless it passes the quality gate', () => {
    const source = readFileSync(
      resolve(process.cwd(), 'src-tauri/src/vfs/pdf_processing_service.rs'),
      'utf-8'
    );

    expect(source).toContain('let has_usable_ocr = ocr_text');
    expect(source).toContain('SELECT f.blob_hash, f.resource_id, f.size, r.ocr_text, f.mime_type');
    expect(source).toContain('if has_usable_ocr {');
    expect(source).toContain('VfsChunker::is_text_quality_acceptable(&ocr_text)');
    expect(source).toContain('OCR returned unusable text');
    expect(source).not.toContain('SELECT f.blob_hash, f.resource_id, f.size, r.ocr_text IS NOT NULL, f.mime_type');
  });

  it('routes blob-backed image OCR through the fallback engine chain before marking OCR ready', () => {
    const source = readFileSync(
      resolve(process.cwd(), 'src-tauri/src/vfs/pdf_processing_service.rs'),
      'utf-8'
    );

    expect(source).toContain('call_ocr_free_text_with_fallback(&image_path)');
    expect(source).toContain('OCR fallback chain TIMED OUT');
    expect(source).toContain('OCR fallback chain FAILED');
    expect(source).toContain('persist_image_ocr_text(&conn, file_id, &ocr_text)');
    expect(source).toContain('此时必须返回错误，避免调用方把 \'ocr\' 虚假加入 ready_modes。');
  });

  it('counts a PDF OCR page as successful only when the page text is usable', () => {
    const source = readFileSync(
      resolve(process.cwd(), 'src-tauri/src/vfs/pdf_processing_service.rs'),
      'utf-8'
    );

    expect(source).toContain('let page_text = blocks');
    expect(source).toContain('!VfsChunker::is_text_quality_acceptable(&page_text)');
    expect(source).toContain('"OCR returned unusable text".to_string()');
    expect(source).toContain('let success_count = results.len();');
  });

  it('rebuilds PDF OCR ready state from usable page coverage, not json presence', () => {
    const source = readFileSync(
      resolve(process.cwd(), 'src-tauri/src/vfs/pdf_processing_service.rs'),
      'utf-8'
    );

    expect(source).toContain('fn pdf_ocr_pages_are_usable');
    expect(source).toContain('parse_ocr_pages_json(ocr_pages_json)');
    expect(source).toContain('usable_count * 2 >= denominator');
    expect(source).toContain('let has_usable_ocr_pages =');
    expect(source).not.toContain('ocr_pages_json IS NOT NULL');
  });

  it('emits completed_with_issues with failed stage details in the completed event', () => {
    const source = readFileSync(
      resolve(process.cwd(), 'src-tauri/src/vfs/pdf_processing_service.rs'),
      'utf-8'
    );

    expect(source).toContain('pub failed_stages: Option<Vec<ProcessingIssue>>');
    expect(source).toContain('fn emit_completed(');
    expect(source).toContain('failed_stages: Option<Vec<ProcessingIssue>>');
    expect(source).toContain('progress.failed_stages.clone()');
    expect(source).toContain('failed_stages,');
  });

  it('does not mark image mode ready before image content is verified', () => {
    const source = readFileSync(
      resolve(process.cwd(), 'src-tauri/src/vfs/pdf_processing_service.rs'),
      'utf-8'
    );
    const imagePipelineStart = source.indexOf('图片流水线内部执行');
    const initStart = source.indexOf('let mut ready_modes: Vec<String> = vec![];', imagePipelineStart);
    const contentCheck = source.indexOf('let has_original_blob = blob_hash', imagePipelineStart);
    const firstImageReady = source.indexOf('ready_modes.push("image".to_string())', initStart);

    expect(imagePipelineStart).toBeGreaterThanOrEqual(0);
    expect(initStart).toBeGreaterThan(imagePipelineStart);
    expect(contentCheck).toBeGreaterThanOrEqual(0);
    expect(firstImageReady).toBeGreaterThan(contentCheck);
    expect(source).toContain('VfsBlobRepo::get_blob_path_with_conn(&conn, &blobs_dir, h)');
    expect(source).toContain('VfsFileRepo::get_content_with_conn(&conn, &blobs_dir, file_id)?.is_some()');
    expect(source).toContain('message: "Image content is unavailable".to_string()');
    expect(source).toContain('retriable: false');
  });

  it('marks PDF image ready only when page images are actually available', () => {
    const source = readFileSync(
      resolve(process.cwd(), 'src-tauri/src/vfs/pdf_processing_service.rs'),
      'utf-8'
    );

    expect(source).toContain('VfsResult<(usize, usize)>');
    expect(source).toContain('self.check_pdf_pages_need_compression(pj, total_pages)');
    expect(source).toContain('preview.pages.len() != total_pages');
    expect(source).toContain('let mut failed_page_count = total_pages.saturating_sub(preview.pages.len())');
    expect(source).toContain('let mut usable_page_count = 0usize;');
    expect(source).toContain('usable_page_count == total_pages');
    expect(source).toContain('PDF page images unavailable');
    expect(source).toContain('stage: ProcessingStage::PageCompression.as_str().to_string()');
    expect(source).not.toContain('Ok(()) => {\n                            // ★ P0 改造：压缩完成后，image 模式才就绪');
  });

  it('uses PDF text quality in page compression progress ready modes', () => {
    const source = readFileSync(
      resolve(process.cwd(), 'src-tauri/src/vfs/pdf_processing_service.rs'),
      'utf-8'
    );

    expect(source).toContain('SELECT extracted_text FROM files WHERE id = ?1');
    expect(source).toContain('VfsChunker::is_text_quality_acceptable(&text)');
    expect(source).not.toContain('SELECT extracted_text IS NOT NULL FROM files WHERE id = ?1');
  });
});
