import { readFileSync } from 'node:fs';
import { resolve } from 'node:path';
import { describe, expect, it } from 'vitest';

describe('InputBar media routing contract', () => {
  const useInputBarSource = readFileSync(
    resolve(process.cwd(), 'src/features/chat/components/input-bar/useInputBarV2.ts'),
    'utf-8'
  );
  const inputBarUiSource = readFileSync(
    resolve(process.cwd(), 'src/features/chat/components/input-bar/InputBarUI.tsx'),
    'utf-8'
  );
  const injectModeSource = readFileSync(
    resolve(process.cwd(), 'src/features/chat/components/input-bar/injectModeUtils.ts'),
    'utf-8'
  );
  const zhLocaleSource = readFileSync(
    resolve(process.cwd(), 'src/locales/zh-CN/chatV2.json'),
    'utf-8'
  );

  it('uses the selected runtime model capability when no model chips are active', () => {
    expect(useInputBarSource).toContain('state.chatParams.model2OverrideId || state.chatParams.modelId');
    expect(useInputBarSource).toContain('shouldDowngradeForTextOnlyTargets');
  });

  it('keeps multimodal image mode sendable without waiting for OCR fallback', () => {
    expect(inputBarUiSource).toContain("const hasImageModeReady = readyModes.includes('image');");
    expect(inputBarUiSource).toContain('const isCompleted = hasImageModeReady');
    expect(inputBarUiSource).toContain('|| stage === \'completed\'');
    expect(inputBarUiSource).toContain('|| stage === \'completed_with_issues\'');
    expect(inputBarUiSource).toContain('const canSendWithAttachments = hasText || hasSendableAttachments || hasMediaAwaitingPreparation;');
    expect(inputBarUiSource).toContain(': !!disabledReason || !canSendWithAttachments || !effectiveCanSubmit || hasUploadingAttachments || queueFull;');
    expect(inputBarUiSource).not.toContain('|| hasProcessingMedia || queueFull');
  });

  it('starts OCR only after downgrading a non-multimodal image target', () => {
    expect(useInputBarSource).toContain('downgradeInjectModesForNonMultimodal(attachment)');
    expect(useInputBarSource).toContain("missingModes.includes('ocr')");
    expect(useInputBarSource).toContain("await startPdfProcessing(attachment.sourceId, 'ocr_processing');");
    expect(useInputBarSource).toContain('return !areAttachmentInjectModesReady(attachment, status);');
    expect(useInputBarSource).not.toContain('return !hasAnySelectedInjectModeReady(attachment, status);');
  });

  it('trusts backend readyModes instead of inferring modes from completed stages', () => {
    expect(injectModeSource).toContain('直接使用后端报告的 readyModes');
    expect(injectModeSource).not.toContain("effectiveStatus?.stage === 'completed' || effectiveStatus?.stage === 'completed_with_issues'");
    expect(injectModeSource).toContain("if (attachment.status === 'ready' && !effectiveStatus)");
  });

  it('labels pending media as parsing instead of not-ready', () => {
    expect(zhLocaleSource).toContain('"attachmentNotReady": "附件解析中：{{name}}（{{modes}}）"');
    expect(zhLocaleSource).toContain('"processingIndicatorPartial": "附件解析中..."');
    expect(zhLocaleSource).toContain('"modesNotReady": "解析中：{{modes}}"');
  });
});
