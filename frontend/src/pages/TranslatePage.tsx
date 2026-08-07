// TranslatePage —— 翻译工作台
// ------------------------------------------------------------
// 顶部：源语言 / 目标语言 / 领域预设 / 自定义 Prompt
// 主体双栏：左源文本，右译文
// 底部：翻译文本 / 翻译文档按钮 + 术语表编辑

import { Button } from "@/components/ui/Button";
import { Input } from "@/components/ui/Input";
import { Textarea } from "@/components/ui/Textarea";
import { Badge } from "@/components/ui/Badge";
import { cn } from "@/lib/utils";
import {
  useTranslateStore,
  DOMAINS,
  LANGUAGES,
  type TranslateDomain,
} from "@/state/translate";
import {
  Languages,
  ArrowLeftRight,
  Loader2,
  AlertCircle,
  FileText,
  Plus,
  Trash2,
  Copy,
  CheckCircle2,
} from "lucide-react";

export function TranslatePage() {
  const srcLang = useTranslateStore((s) => s.srcLang);
  const tgtLang = useTranslateStore((s) => s.tgtLang);
  const domain = useTranslateStore((s) => s.domain);
  const customPrompt = useTranslateStore((s) => s.customPrompt);
  const sourceText = useTranslateStore((s) => s.sourceText);
  const targetText = useTranslateStore((s) => s.targetText);
  const documentUri = useTranslateStore((s) => s.documentUri);
  const documentOutput = useTranslateStore((s) => s.documentOutput);
  const glossary = useTranslateStore((s) => s.glossary);
  const translating = useTranslateStore((s) => s.translating);
  const translatingDoc = useTranslateStore((s) => s.translatingDoc);
  const error = useTranslateStore((s) => s.error);
  const notice = useTranslateStore((s) => s.notice);

  const setSrcLang = useTranslateStore((s) => s.setSrcLang);
  const setTgtLang = useTranslateStore((s) => s.setTgtLang);
  const setDomain = useTranslateStore((s) => s.setDomain);
  const setCustomPrompt = useTranslateStore((s) => s.setCustomPrompt);
  const setSourceText = useTranslateStore((s) => s.setSourceText);
  const setDocumentUri = useTranslateStore((s) => s.setDocumentUri);
  const addGlossary = useTranslateStore((s) => s.addGlossary);
  const updateGlossary = useTranslateStore((s) => s.updateGlossary);
  const removeGlossary = useTranslateStore((s) => s.removeGlossary);
  const translateText = useTranslateStore((s) => s.translateText);
  const translateDocument = useTranslateStore((s) => s.translateDocument);
  const swapLangs = useTranslateStore((s) => s.swapLangs);

  return (
    <div className="flex h-full w-full min-h-0 flex-col bg-background">
      {/* —— 顶部配置栏 —— */}
      <header className="shrink-0 space-y-2 border-b border-border bg-card px-4 py-3">
        <div className="flex items-center gap-2 text-sm">
          <Languages size={16} className="text-primary" />
          <h1 className="font-semibold text-foreground">翻译工作台</h1>
          <Badge variant="outline" className="ml-1 text-[10px]">
            {DOMAINS.find((d) => d.key === domain)?.label ?? domain}
          </Badge>
        </div>
        <div className="flex flex-wrap items-end gap-3">
          {/* 源语言 */}
          <Field label="源语言">
            <LangSelect value={srcLang} onChange={setSrcLang} />
          </Field>
          {/* 交换按钮 */}
          <button
            type="button"
            onClick={swapLangs}
            title="交换源 / 目标语言"
            className="mb-1 rounded-md border border-border bg-background p-1.5 text-muted-foreground transition-colors hover:bg-accent hover:text-foreground"
          >
            <ArrowLeftRight size={13} />
          </button>
          {/* 目标语言 */}
          <Field label="目标语言">
            <LangSelect value={tgtLang} onChange={setTgtLang} excludeAuto />
          </Field>
          {/* 领域预设 */}
          <Field label="领域预设">
            <select
              value={domain}
              onChange={(e) => setDomain(e.target.value as TranslateDomain)}
              className="h-[var(--touch-target-size)] rounded-md border border-input bg-transparent px-3 text-sm text-foreground transition-colors focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring"
            >
              {DOMAINS.map((d) => (
                <option key={d.key} value={d.key} className="bg-card">
                  {d.label} — {d.description}
                </option>
              ))}
            </select>
          </Field>
          {/* 自定义 Prompt */}
          <div className="min-w-[16rem] flex-1">
            <Field label="自定义 Prompt（可选）">
              <Input
                value={customPrompt}
                onChange={(e) => setCustomPrompt(e.target.value)}
                placeholder="例如：保留术语大小写、保持学术语气…"
                className="h-[var(--touch-target-size)] text-[13px]"
              />
            </Field>
          </div>
        </div>
      </header>

      {/* —— 错误 / 通知横幅 —— */}
      {error && (
        <div className="shrink-0 flex items-center gap-2 border-b border-destructive/30 bg-destructive/10 px-4 py-2 text-xs text-destructive">
          <AlertCircle size={12} />
          <span className="truncate">{error}</span>
        </div>
      )}
      {notice && (
        <div className="shrink-0 flex items-center gap-2 border-b border-emerald-500/30 bg-emerald-500/10 px-4 py-2 text-xs text-emerald-500">
          <CheckCircle2 size={12} />
          <span className="truncate">{notice}</span>
        </div>
      )}

      {/* —— 主体双栏：左源 / 右译 —— */}
      <div className="flex min-h-0 flex-1 gap-px bg-border">
        {/* 左：源文本 */}
        <section className="flex min-w-0 flex-1 flex-col bg-background">
          <div className="shrink-0 border-b border-border bg-card px-4 py-2 text-[10px] font-semibold uppercase tracking-wider text-muted-foreground/70">
            源文本
            <span className="ml-2 font-normal normal-case text-muted-foreground/50">
              {sourceText.length} 字符
            </span>
          </div>
          <Textarea
            value={sourceText}
            onChange={(e) => setSourceText(e.target.value)}
            placeholder="在此输入或粘贴要翻译的文本…"
            className="min-h-0 flex-1 resize-none rounded-none border-0 bg-transparent font-mono text-[13px] leading-relaxed text-foreground focus-visible:ring-0"
          />
        </section>

        {/* 右：译文 */}
        <section className="flex min-w-0 flex-1 flex-col bg-muted/20">
          <div className="flex shrink-0 items-center justify-between border-b border-border bg-card px-4 py-2 text-[10px] font-semibold uppercase tracking-wider text-muted-foreground/70">
            <span>
              译文
              {translating && (
                <span className="ml-2 inline-flex items-center gap-1 font-normal normal-case text-primary">
                  <Loader2 size={10} className="animate-spin" />
                  翻译中…
                </span>
              )}
            </span>
            {targetText && (
              <button
                type="button"
                onClick={() => void navigator.clipboard.writeText(targetText)}
                className="inline-flex items-center gap-1 rounded px-1.5 py-0.5 text-[10px] font-normal normal-case text-muted-foreground transition-colors hover:bg-accent hover:text-foreground"
                title="复制译文"
              >
                <Copy size={10} />
                复制
              </button>
            )}
          </div>
          <div className="min-h-0 flex-1 overflow-y-auto scrollbar-dark">
            {targetText ? (
              <pre className="whitespace-pre-wrap break-words px-4 py-3 font-mono text-[13px] leading-relaxed text-foreground/90">
                {targetText}
              </pre>
            ) : (
              <EmptyTranslate translating={translating} />
            )}
          </div>
        </section>
      </div>

      {/* —— 底部操作 + 术语表 —— */}
      <footer className="shrink-0 space-y-3 border-t border-border bg-card px-4 py-3">
        {/* 操作按钮 */}
        <div className="flex flex-wrap items-end gap-3">
          <Button
            onClick={() => void translateText()}
            disabled={translating || !sourceText.trim()}
            className="h-9"
          >
            {translating ? (
              <Loader2 size={14} className="animate-spin" />
            ) : (
              <Languages size={14} />
            )}
            {translating ? "翻译中…" : "翻译文本"}
          </Button>

          {/* 翻译文档 */}
          <div className="flex min-w-[24rem] flex-1 items-end gap-2">
            <div className="min-w-0 flex-1">
              <Field label="翻译文档（vfs:// URI）">
                <Input
                  value={documentUri}
                  onChange={(e) => setDocumentUri(e.target.value)}
                  placeholder="vfs://textbook/abc123.md"
                  className="h-[var(--touch-target-size)] font-mono text-[12px]"
                />
              </Field>
            </div>
            <Button
              variant="outline"
              onClick={() => void translateDocument()}
              disabled={translatingDoc || !documentUri.trim()}
              className="h-9"
            >
              {translatingDoc ? (
                <Loader2 size={14} className="animate-spin" />
              ) : (
                <FileText size={14} />
              )}
              {translatingDoc ? "处理中…" : "翻译文档"}
            </Button>
          </div>
        </div>

        {/* 文档翻译输出（如有） */}
        {documentOutput && (
          <div className="rounded-md border border-primary/30 bg-primary/5 p-3">
            <div className="mb-1 flex items-center gap-1 text-[10px] font-semibold uppercase tracking-wider text-primary">
              <FileText size={10} />
              文档翻译输出
            </div>
            <pre className="whitespace-pre-wrap break-words font-mono text-[11px] leading-relaxed text-foreground/90">
              {documentOutput}
            </pre>
          </div>
        )}

        {/* 术语表 */}
        <div>
          <div className="mb-2 flex items-center justify-between">
            <div className="text-[10px] font-semibold uppercase tracking-wider text-muted-foreground/70">
              术语表
              {glossary.length > 0 && (
                <span className="ml-1 font-normal normal-case text-muted-foreground/50">
                  （{glossary.length} 条）
                </span>
              )}
            </div>
            <Button
              variant="ghost"
              size="sm"
              className="h-7"
              onClick={addGlossary}
            >
              <Plus size={12} />
              添加
            </Button>
          </div>
          {glossary.length === 0 ? (
            <div className="rounded-md border border-dashed border-border bg-background px-3 py-2 text-[11px] text-muted-foreground">
              暂无术语表条目 —— 点击"添加"建立 key → value 对照（如：
              attention → 注意力）
            </div>
          ) : (
            <ul className="space-y-1.5">
              {glossary.map((g) => (
                <li key={g.id} className="flex items-center gap-2">
                  <Input
                    value={g.key}
                    onChange={(e) => updateGlossary(g.id, "key", e.target.value)}
                    placeholder="原文术语"
                    className="h-7 font-mono text-[12px]"
                  />
                  <ArrowLeftRight size={11} className="shrink-0 text-muted-foreground/50" />
                  <Input
                    value={g.value}
                    onChange={(e) => updateGlossary(g.id, "value", e.target.value)}
                    placeholder="译文术语"
                    className="h-7 font-mono text-[12px]"
                  />
                  <button
                    type="button"
                    onClick={() => removeGlossary(g.id)}
                    className="shrink-0 rounded p-1 text-muted-foreground/60 transition-colors hover:bg-destructive/15 hover:text-destructive"
                    title="删除"
                  >
                    <Trash2 size={12} />
                  </button>
                </li>
              ))}
            </ul>
          )}
        </div>
      </footer>
    </div>
  );
}

/** 字段包装：标签 + 子内容 */
function Field({
  label,
  children,
}: {
  label: string;
  children: React.ReactNode;
}) {
  return (
    <label className="block">
      <span className="mb-1 block text-[10px] font-semibold uppercase tracking-wider text-muted-foreground/70">
        {label}
      </span>
      {children}
    </label>
  );
}

/** 语言下拉选择 */
function LangSelect({
  value,
  onChange,
  excludeAuto,
}: {
  value: string;
  onChange: (v: string) => void;
  excludeAuto?: boolean;
}) {
  return (
    <select
      value={value}
      onChange={(e) => onChange(e.target.value)}
      className="h-[var(--touch-target-size)] rounded-md border border-input bg-transparent px-3 text-sm text-foreground transition-colors focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring"
    >
      {LANGUAGES.filter((l) => (excludeAuto ? l.code !== "auto" : true)).map(
        (l) => (
          <option key={l.code} value={l.code} className="bg-card">
            {l.label}
          </option>
        )
      )}
    </select>
  );
}

/** 空状态：尚未翻译 */
function EmptyTranslate({ translating }: { translating: boolean }) {
  return (
    <div className="flex h-full flex-col items-center justify-center gap-3 px-6 text-center">
      <div
        className={cn(
          "flex h-12 w-12 items-center justify-center rounded-full bg-muted text-muted-foreground"
        )}
      >
        {translating ? (
          <Loader2 size={20} className="animate-spin" />
        ) : (
          <Languages size={20} />
        )}
      </div>
      <div className="space-y-1">
        <div className="text-sm font-medium text-foreground">
          {translating ? "正在翻译…" : "译文将显示在此处"}
        </div>
        <div className="text-xs text-muted-foreground">
          {translating
            ? "正在调用后端翻译服务"
            : "在左侧输入文本后点击「翻译文本」"}
        </div>
      </div>
    </div>
  );
}
