// OCRPage —— 图片/PDF 文字识别
// ------------------------------------------------------------
// 对接 ocrApi（OCR* 方法）：图片上传识别 + 引擎切换 + PDF 文本层提取。
// PDF 整卷逐页识别由前端分页调用（此处提供图片识别 + PDF 文本层）。

import { useEffect, useRef, useState } from "react";
import { ocrApi, type OCREngineInfo, type OcrResult } from "@/lib/ocr";
import { Button } from "@/components/ui/Button";
import { Badge } from "@/components/ui/Badge";
import { Textarea } from "@/components/ui/Textarea";
import {
  ScanLine,
  Upload,
  Loader2,
  FileText,
  Copy,
  Check,
  AlertCircle,
  ImageIcon,
} from "lucide-react";

export function OCRPage() {
  const [engines, setEngines] = useState<OCREngineInfo[]>([]);
  const [engine, setEngine] = useState<string>("");
  const [image, setImage] = useState<string | null>(null);
  const [imageBytes, setImageBytes] = useState<number[] | null>(null);
  const [result, setResult] = useState<OcrResult | null>(null);
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [pdfText, setPdfText] = useState("");
  const [pdfBusy, setPdfBusy] = useState(false);
  const fileRef = useRef<HTMLInputElement>(null);

  useEffect(() => {
    void (async () => {
      const es = await ocrApi.listEngines();
      if (es) setEngines(es);
      const cur = await ocrApi.engineType();
      if (cur) setEngine(cur);
    })();
  }, []);

  const onFile = (f: File | undefined | null) => {
    if (!f) return;
    const reader = new FileReader();
    reader.onload = () => {
      const bytes = new Uint8Array(reader.result as ArrayBuffer);
      setImageBytes(Array.from(bytes));
      setImage(URL.createObjectURL(f));
      setResult(null);
      setError(null);
    };
    reader.readAsArrayBuffer(f);
  };

  const recognize = async () => {
    if (!imageBytes) return;
    setBusy(true);
    setError(null);
    try {
      const res = await ocrApi.recognize(imageBytes, "image/png");
      setResult(res);
    } catch (e) {
      setError(e instanceof Error ? e.message : String(e));
    } finally {
      setBusy(false);
    }
  };

  const extractPDF = async (f: File) => {
    setPdfBusy(true);
    setError(null);
    try {
      const buf = await f.arrayBuffer();
      const bytes = Array.from(new Uint8Array(buf));
      const text = await ocrApi.extractPDFText(bytes);
      setPdfText(text ?? "");
    } catch (e) {
      setError(e instanceof Error ? e.message : String(e));
    } finally {
      setPdfBusy(false);
    }
  };

  return (
    <div className="flex h-full w-full min-h-0 bg-background">
      {/* —— 左：图片 OCR —— */}
      <section className="flex min-w-0 flex-1 flex-col border-r border-border">
        <div className="flex shrink-0 items-center gap-2 border-b border-border bg-card px-4 py-2.5">
          <div className="flex h-6 w-6 items-center justify-center rounded-md bg-primary/15 text-primary">
            <ScanLine size={13} />
          </div>
          <span className="text-[11px] font-semibold uppercase tracking-wider text-muted-foreground/70">
            图片识别
          </span>
          <div className="ml-auto flex items-center gap-1.5">
            {engines.map((e) => (
              <button
                key={e.type}
                disabled={!e.available}
                onClick={() => { void ocrApi.setEngine(e.type); setEngine(e.type); }}
                className={`rounded-md px-2 py-0.5 text-[10px] transition-colors ${
                  engine === e.type
                    ? "bg-primary/15 font-medium text-primary"
                    : "text-muted-foreground hover:bg-accent"
                } ${!e.available ? "opacity-40" : ""}`}
                title={e.description}
              >
                {e.name}
              </button>
            ))}
          </div>
        </div>

        <div className="flex min-h-0 flex-1 flex-col gap-3 overflow-y-auto scrollbar-dark p-4">
          {/* 上传区 */}
          <div
            onClick={() => fileRef.current?.click()}
            className="flex h-44 cursor-pointer flex-col items-center justify-center gap-2 rounded-lg border-2 border-dashed border-border text-muted-foreground transition-colors hover:border-primary/40 hover:bg-primary/5"
          >
            {image ? (
              <img src={image} alt="upload" className="max-h-36 max-w-full rounded object-contain" />
            ) : (
              <>
                <Upload size={24} className="opacity-50" />
                <span className="text-[12px]">点击选择图片（PNG/JPG）</span>
              </>
            )}
            <input
              ref={fileRef}
              type="file"
              accept="image/*"
              className="hidden"
              onChange={(e) => onFile(e.target.files?.[0])}
            />
          </div>

          <Button disabled={!imageBytes || busy} onClick={() => void recognize()}>
            {busy ? <Loader2 size={14} className="mr-1 animate-spin" /> : <ScanLine size={14} className="mr-1" />}
            {busy ? "识别中…" : "开始识别"}
          </Button>

          {error && (
            <div className="flex items-center gap-1.5 text-[11px] text-destructive">
              <AlertCircle size={12} />
              {error}
            </div>
          )}

          {/* 结果 */}
          {result && (
            <div className="rounded-md border border-border bg-card p-3">
              <div className="mb-2 flex items-center justify-between">
                <span className="text-[10px] font-semibold uppercase tracking-wider text-muted-foreground/70">
                  识别结果 · {result.engine} · {result.durationMs}ms
                </span>
                <CopyButton text={result.text} />
              </div>
              <pre className="whitespace-pre-wrap text-[12px] leading-relaxed text-foreground">
                {result.text}
              </pre>
            </div>
          )}

          {!result && !error && (
            <div className="flex flex-col items-center justify-center gap-1 py-6 text-muted-foreground">
              <ImageIcon size={20} className="opacity-40" />
              <span className="text-[11px]">选择图片后点「开始识别」</span>
            </div>
          )}
        </div>
      </section>

      {/* —— 右：PDF 文本层 —— */}
      <section className="flex w-[40%] shrink-0 flex-col">
        <div className="flex shrink-0 items-center gap-2 border-b border-border bg-card px-4 py-2.5">
          <div className="flex h-6 w-6 items-center justify-center rounded-md bg-primary/15 text-primary">
            <FileText size={13} />
          </div>
          <span className="text-[11px] font-semibold uppercase tracking-wider text-muted-foreground/70">
            PDF 文本提取
          </span>
          <div className="ml-auto">
            <input
              type="file"
              accept="application/pdf"
              className="hidden"
              id="pdf-input"
              onChange={(e) => { const f = e.target.files?.[0]; if (f) void extractPDF(f); }}
            />
            <label htmlFor="pdf-input">
              <Button size="sm" variant="outline" className="h-7 cursor-pointer" asChild={false}>
                <Upload size={12} className="mr-1" />
                选择 PDF
              </Button>
            </label>
          </div>
        </div>
        <div className="min-h-0 flex-1 p-3">
          {pdfBusy ? (
            <div className="flex h-full items-center justify-center">
              <Loader2 size={18} className="animate-spin text-muted-foreground" />
            </div>
          ) : pdfText ? (
            <div className="relative h-full">
              <div className="absolute right-2 top-1">
                <CopyButton text={pdfText} />
              </div>
              <Textarea
                value={pdfText}
                readOnly
                className="h-full w-full resize-none font-mono text-[11px] leading-relaxed"
              />
            </div>
          ) : (
            <div className="flex h-full flex-col items-center justify-center gap-2 text-muted-foreground">
              <FileText size={22} className="opacity-40" />
              <span className="text-[11px]">选择 PDF 提取其文本层</span>
            </div>
          )}
        </div>
      </section>
    </div>
  );
}

function CopyButton({ text }: { text: string }) {
  const [copied, setCopied] = useState(false);
  return (
    <button
      onClick={() => {
        void navigator.clipboard?.writeText(text);
        setCopied(true);
        setTimeout(() => setCopied(false), 1500);
      }}
      className="rounded p-1 text-muted-foreground hover:text-foreground"
      title="复制"
    >
      {copied ? <Check size={12} className="text-emerald-500" /> : <Copy size={12} />}
    </button>
  );
}
