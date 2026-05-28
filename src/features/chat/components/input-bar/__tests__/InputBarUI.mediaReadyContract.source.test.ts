import { readFileSync } from 'node:fs';
import { resolve } from 'node:path';
import { describe, expect, it } from 'vitest';

describe('InputBarUI media ready contract', () => {
  it('does not treat terminal attachments as sendable unless selected modes are ready', () => {
    const source = readFileSync(
      resolve(process.cwd(), 'src/features/chat/components/input-bar/InputBarUI.tsx'),
      'utf-8'
    );
    const hookSource = readFileSync(
      resolve(process.cwd(), 'src/features/chat/components/input-bar/useInputBarV2.ts'),
      'utf-8'
    );

    expect(source).not.toContain('if (isTerminalMediaStage(status) && readyModes?.length)');
    expect(source).toContain('const missingModes = getMissingModes(selectedModes, readyModes);');
    expect(source).toContain('return areSelectedModesReady(selectedModes, readyModes);');
    expect(source).toContain('return !areSelectedModesReady(selectedModes, readyModes);');
    expect(source).toContain('if (missingModes.length > 0)');
    expect(hookSource).toContain('return !areAttachmentInjectModesReady(attachment, status);');
    expect(hookSource).toContain('return areAttachmentInjectModesReady(attachment, status);');
    expect(hookSource).not.toContain('hasAnySelectedInjectModeReady');
  });

  it('does not silently shrink user-selected inject modes when processing completes', () => {
    const source = readFileSync(
      resolve(process.cwd(), 'src/features/chat/components/input-bar/InputBarUI.tsx'),
      'utf-8'
    );

    expect(source).not.toContain('function getTerminalReadyInjectModes');
    expect(source).not.toContain('alignedInjectModes');
    expect(source).not.toContain('const fallbackModes = usableSelected.length > 0 ? usableSelected : readyModes;');
  });

  it('does not drop completed_with_issues details between media events and the attachment chip', () => {
    const hookSource = readFileSync(
      resolve(process.cwd(), 'src/hooks/usePdfProcessingProgress.ts'),
      'utf-8'
    );
    const storeSource = readFileSync(
      resolve(process.cwd(), 'src/features/pdf/stores/pdfProcessingStore.ts'),
      'utf-8'
    );
    const inputSource = readFileSync(
      resolve(process.cwd(), 'src/features/chat/components/input-bar/InputBarUI.tsx'),
      'utf-8'
    );

    expect(hookSource).toContain('failedStages: status.failedStages');
    expect(hookSource).toContain('setCompleted(fileId, readyModes, stage, failedStages)');
    expect(storeSource).toContain("const nextFailedStages = stage === 'completed_with_issues'");
    expect(storeSource).toContain('failedStages: nextFailedStages');
    expect(inputSource).toContain('failedStages: status.failedStages');
    expect(inputSource).toContain('const hasProcessingIssue = displayStatus?.stage === \'completed_with_issues\'');
  });

  it('does not turn already usable media into an error when polling times out', () => {
    const inputSource = readFileSync(
      resolve(process.cwd(), 'src/features/chat/components/input-bar/InputBarUI.tsx'),
      'utf-8'
    );

    expect(inputSource).toContain('if (hasAnyReadyMode(selectedModes, readyModes))');
    expect(inputSource).toContain("stage: 'completed_with_issues'");
    expect(inputSource).toContain('usePdfProcessingStore.getState().setCompleted(');
    expect(inputSource).toContain('continue;');
    expect(inputSource).toContain("status: 'error'");
  });

  it('does not optimistically mark image attachments ready without backend ready modes', () => {
    const utilsSource = readFileSync(
      resolve(process.cwd(), 'src/features/chat/components/input-bar/injectModeUtils.ts'),
      'utf-8'
    );
    const selectorSource = readFileSync(
      resolve(process.cwd(), 'src/features/chat/components/input-bar/AttachmentInjectModeSelector.tsx'),
      'utf-8'
    );

    expect(utilsSource).toContain('完成内容验证后才将 image 加入 readyModes');
    expect(utilsSource).not.toContain("if (effectiveStatus?.stage === 'completed')");
    expect(utilsSource).not.toContain("mediaType === 'image' && (attachment.status === 'processing' || attachment.status === 'ready')");
    expect(utilsSource).not.toContain("return ['image'];\n  }\n\n  return undefined;");
    expect(selectorSource).not.toContain("if (mode === 'image') return true;");
    expect(selectorSource).toContain('return readyModes.has(mode);');
    expect(selectorSource).toContain('readyModes: [],');
  });

  it('only animates not-ready inject modes that the user actually selected', () => {
    const inputSource = readFileSync(
      resolve(process.cwd(), 'src/features/chat/components/input-bar/InputBarUI.tsx'),
      'utf-8'
    );
    const selectorSource = readFileSync(
      resolve(process.cwd(), 'src/features/chat/components/input-bar/AttachmentInjectModeSelector.tsx'),
      'utf-8'
    );

    expect(selectorSource).toContain('return selectedModes.includes(mode) && isProcessing && !isModeReady(mode);');
    expect(selectorSource).toContain("isProcessing={isSelectedModeProcessing('image')}");
    expect(selectorSource).toContain("isProcessing={isSelectedModeProcessing('ocr')}");
    expect(inputSource).toContain('const selectedModesReady = missingModes.length === 0;');
    expect(inputSource).toContain('isMediaProcessing && selectedModesReady');
  });

  it('allows retry for completed_with_issues media only when selected modes are still missing', () => {
    const inputSource = readFileSync(
      resolve(process.cwd(), 'src/features/chat/components/input-bar/InputBarUI.tsx'),
      'utf-8'
    );

    expect(inputSource).toContain('const canRetryMediaProcessing = Boolean(attachment.sourceId)');
    expect(inputSource).toContain("attachment.status === 'error'");
    expect(inputSource).toContain("attachment.status === 'ready'");
    expect(inputSource).toContain('hasProcessingIssue');
    expect(inputSource).toContain('missingModes.length > 0');
    expect(inputSource).toContain('processingIssue?.retriable !== false');
    expect(inputSource).toContain('{canRetryMediaProcessing && (');
  });

  it('keeps completed media events self-contained with failed stage details', () => {
    const hookSource = readFileSync(
      resolve(process.cwd(), 'src/hooks/usePdfProcessingProgress.ts'),
      'utf-8'
    );
    const backendSource = readFileSync(
      resolve(process.cwd(), 'src-tauri/src/vfs/pdf_processing_service.rs'),
      'utf-8'
    );

    expect(backendSource).toContain('pub failed_stages: Option<Vec<ProcessingIssue>>');
    expect(backendSource).toContain('progress.failed_stages.clone()');
    expect(backendSource).toContain('failed_stages,');
    expect(hookSource).toContain('const failedStages = payload.failedStages ?? payload.failed_stages');
  });

  it('bounds single-image OCR so the attachment can complete with issues instead of spinning forever', () => {
    const backendSource = readFileSync(
      resolve(process.cwd(), 'src-tauri/src/vfs/pdf_processing_service.rs'),
      'utf-8'
    );

    expect(backendSource).toContain('const IMAGE_OCR_TIMEOUT_SECS: u64 = 90;');
    expect(backendSource).toContain('call_ocr_free_text_with_fallback(&image_path)');
    expect(backendSource).toContain('OCR fallback chain TIMED OUT');
    expect(backendSource).toContain('persist_image_ocr_text(&conn, file_id, &ocr_text)');
    expect(backendSource).toContain('System OCR timed out after');
    expect(backendSource).toContain('OCR API call TIMED OUT');
    expect(backendSource).toContain('OCR API call timed out after');
  });
});
