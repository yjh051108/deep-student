// ReaderActionPanel —— 阅读器右侧操作面板
// ------------------------------------------------------------
// 1. 页码 + 总结本页按钮 + 总结文本展示
// 2. 选段注入：start/end 页码 + 选段文本 + 生成按钮 + 结果 + 复制

import { useReaderStore } from "@/state/reader";
import { Button } from "@/components/ui/Button";
import { Input } from "@/components/ui/Input";
import { Textarea } from "@/components/ui/Textarea";
import { cn } from "@/lib/utils";
import {
  Sparkles,
  Loader2,
  Inbox,
  Copy,
  CheckCircle2,
  Wand2,
  FileText,
  AlertCircle,
} from "lucide-react";
import { useEffect, useState, type ReactNode } from "react";

export function ReaderActionPanel() {
  const doc = useReaderStore((s) => s.doc);
  const currentPageIdx = useReaderStore((s) => s.currentPageIdx);
  const summary = useReaderStore((s) => s.summary);
  const summarizing = useReaderStore((s) => s.summarizing);
  const injection = useReaderStore((s) => s.injection);
  const injecting = useReaderStore((s) => s.injecting);
  const error = useReaderStore((s) => s.error);
  const summarize = useReaderStore((s) => s.summarize);
  const inject = useReaderStore((s) => s.inject);
  const clearSummary = useReaderStore((s) => s.clearSummary);

  const [start, setStart] = useState(0);
  const [end, setEnd] = useState(0);
  const [sel, setSel] = useState("");
  const [copied, setCopied] = useState(false);
  const [inited, setInited] = useState(false);

  // 文档首次打开时，把 start / end 初始化为当前页
  useEffect(() => {
    if (doc && !inited && doc.pages.length > 0) {
      setStart(currentPageIdx);
      setEnd(currentPageIdx);
      setInited(true);
    }
    if (!doc && inited) {
      setInited(false);
    }
  }, [doc, inited, currentPageIdx]);

  const handleInject = async () => {
    await inject(start, end, sel);
  };

  const handleCopy = async () => {
    if (!injection) return;
    await navigator.clipboard.writeText(injection);
    setCopied(true);
    setTimeout(() => setCopied(false), 1500);
  };

  if (!doc) {
    return <EmptyState />;
  }

  return (
    <div className="flex h-full w-full flex-col">
      {/* 头部 */}
      <div className="shrink-0 border-b border-border bg-card px-4 py-2.5">
        <div className="flex items-center gap-1.5">
          <Wand2 size={13} className="text-primary" />
          <span className="text-[10px] font-semibold uppercase tracking-wider text-muted-foreground/70">
            AI 操作
          </span>
        </div>
      </div>

      {/* 错误横幅 */}
      {error && (
        <div className="shrink-0 flex items-center gap-2 border-b border-destructive/30 bg-destructive/10 px-4 py-2 text-[11px] text-destructive">
          <AlertCircle size={11} />
          <span className="truncate">{error}</span>
        </div>
      )}

      <div className="min-h-0 flex-1 overflow-y-auto scrollbar-dark">
        <div className="space-y-4 px-4 py-3">
          {/* 总结本页 */}
          <Section title="本页总结" icon={Sparkles}>
            <div className="space-y-2">
              <div className="flex items-center gap-2 text-[11px] text-muted-foreground">
                <span>当前页：</span>
                <span className="font-mono font-semibold text-foreground">
                  {currentPageIdx + 1} / {doc.pages.length}
                </span>
              </div>
              <Button
                size="sm"
                className="h-8 w-full"
                onClick={() => void summarize()}
                disabled={summarizing}
              >
                {summarizing ? (
                  <Loader2 size={12} className="animate-spin" />
                ) : (
                  <Sparkles size={12} />
                )}
                {summarizing ? "总结中…" : "总结本页"}
              </Button>
              {summary && (
                <div className="rounded-md border border-primary/30 bg-primary/5 p-3">
                  <div className="mb-1.5 flex items-center justify-between">
                    <div className="flex items-center gap-1 text-[10px] font-semibold uppercase tracking-wider text-primary">
                      <Sparkles size={10} />
                      总结结果
                    </div>
                    <button
                      type="button"
                      onClick={clearSummary}
                      className="text-[10px] text-muted-foreground hover:text-foreground"
                    >
                      清除
                    </button>
                  </div>
                  <pre className="whitespace-pre-wrap break-words font-sans text-[12px] leading-relaxed text-foreground/90">
                    {summary}
                  </pre>
                </div>
              )}
            </div>
          </Section>

          {/* 选段注入 */}
          <Section title="选段注入" icon={Wand2}>
            <div className="space-y-2">
              <div className="grid grid-cols-2 gap-2">
                <div>
                  <label className="mb-1 block text-[10px] text-muted-foreground/70">
                    起始页
                  </label>
                  <Input
                    type="number"
                    min={1}
                    max={doc.pages.length}
                    value={start + 1}
                    onChange={(e) => {
                      const v = Number(e.target.value);
                      if (!Number.isNaN(v)) setStart(Math.max(0, v - 1));
                    }}
                    className="h-8 text-[12px]"
                  />
                </div>
                <div>
                  <label className="mb-1 block text-[10px] text-muted-foreground/70">
                    结束页
                  </label>
                  <Input
                    type="number"
                    min={1}
                    max={doc.pages.length}
                    value={end + 1}
                    onChange={(e) => {
                      const v = Number(e.target.value);
                      if (!Number.isNaN(v)) setEnd(Math.max(0, v - 1));
                    }}
                    className="h-8 text-[12px]"
                  />
                </div>
              </div>
              <div>
                <label className="mb-1 block text-[10px] text-muted-foreground/70">
                  选段文本
                </label>
                <Textarea
                  value={sel}
                  onChange={(e) => setSel(e.target.value)}
                  placeholder="粘贴或输入要注入的选段…"
                  rows={5}
                  className="resize-none text-[12px]"
                />
              </div>
              <Button
                size="sm"
                className="h-8 w-full"
                onClick={handleInject}
                disabled={!sel.trim() || injecting}
              >
                {injecting ? (
                  <Loader2 size={12} className="animate-spin" />
                ) : (
                  <Wand2 size={12} />
                )}
                {injecting ? "生成中…" : "生成注入串"}
              </Button>
              {injection && (
                <div className="rounded-md border border-primary/30 bg-primary/5 p-3">
                  <div className="mb-1.5 flex items-center justify-between">
                    <div className="flex items-center gap-1 text-[10px] font-semibold uppercase tracking-wider text-primary">
                      <FileText size={10} />
                      注入结果
                    </div>
                    <button
                      type="button"
                      onClick={handleCopy}
                      className="flex items-center gap-1 text-[10px] text-muted-foreground hover:text-foreground"
                      title="复制结果"
                    >
                      {copied ? (
                        <CheckCircle2 size={10} className="text-success" />
                      ) : (
                        <Copy size={10} />
                      )}
                      {copied ? "已复制" : "复制"}
                    </button>
                  </div>
                  <pre className="whitespace-pre-wrap break-words font-mono text-[11px] leading-relaxed text-foreground/90">
                    {injection}
                  </pre>
                </div>
              )}
            </div>
          </Section>
        </div>
      </div>
    </div>
  );
}

/** 小节容器 */
function Section({
  title,
  icon: Icon,
  children,
}: {
  title: string;
  icon: typeof Sparkles;
  children: ReactNode;
}) {
  return (
    <div className="space-y-2">
      <div className="flex items-center gap-1.5 text-[10px] font-semibold uppercase tracking-wider text-muted-foreground/70">
        <Icon size={11} />
        {title}
      </div>
      {children}
    </div>
  );
}

function EmptyState() {
  return (
    <div className="flex h-full flex-col items-center justify-center gap-3 px-6 text-center">
      <div className="flex h-12 w-12 items-center justify-center rounded-full bg-muted text-muted-foreground">
        <Inbox size={20} />
      </div>
      <div className="space-y-1">
        <div className="text-sm font-medium text-foreground">未打开文档</div>
        <div className="text-xs text-muted-foreground">
          打开文档后可使用 AI 总结与选段注入
        </div>
      </div>
    </div>
  );
}

// 保留 cn 引用以备未来扩展
void cn;
