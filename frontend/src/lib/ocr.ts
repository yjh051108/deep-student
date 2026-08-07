// OCR 前端类型与 Wails 封装
// ------------------------------------------------------------
// 与后端 internal/ocr 对齐。

import { callWails } from "@/lib/wails";

/** OCR 引擎信息 —— 与后端 ocr.EngineInfo 对齐 */
export interface OCREngineInfo {
  type: string;
  name: string;
  available: boolean;
  isDefault: boolean;
  priority: number;
  description: string;
}

/** OCR 结果 —— 与后端 ocr.OcrResult 对齐 */
export interface OcrResult {
  text: string;
  engine: string;
  lines?: { text: string; confidence: number }[];
  durationMs: number;
}

export const ocrApi = {
  recognize: (imageData: number[], mime: string) =>
    callWails<OcrResult>("OCRRecognize", imageData, mime),
  listEngines: () => callWails<OCREngineInfo[]>("OCRListEngines"),
  setEngine: (t: string) => callWails<void>("OCRSetEngineType", t),
  engineType: () => callWails<string>("OCRGetEngineType"),
  startPDFSession: (name: string, pages: number) =>
    callWails<string>("OCRStartPDFSession", name, pages),
  uploadPage: (sessionId: string, pageIndex: number, imageData: number[], mime: string) =>
    callWails<string>("OCRUploadPage", sessionId, pageIndex, imageData, mime),
  cancelPDFSession: (sessionId: string) =>
    callWails<void>("OCRCancelPDFSession", sessionId),
  extractPDFText: (data: number[]) => callWails<string>("OCRExtractTextFromPDF", data),
};
