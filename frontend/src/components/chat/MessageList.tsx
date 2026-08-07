// MessageList —— 1:1 对齐原版 ChatV2 消息区（块式渲染）
// ------------------------------------------------------------
// - 居中线程宽度容器（max-w-thread）
// - 用户消息：右对齐气泡（圆角 16px，160px 截断）
// - 助手消息：块式渲染（思考 → 文本 → 工具记录折叠 → 引用）
// - 消息操作：复制 / 重试 / 分支
// - 底部来源汇总面板（SourcePanel 概念，简化）

import { useEffect, useRef, useState } from "react";
import { cn } from "@/lib/utils";
import type { UIMessage } from "@/state/chat";
import {
  CaretDown,
  CaretRight,
  Copy,
  ArrowCounterClockwise,
  GitBranch,
  LinkSimple,
  Warning,
  Check,
} from "@phosphor-icons/react";
import { Badge } from "@/components/ui/Badge";

export function MessageList({
  session,
  onDeleteMessage,
  onRetry,
  onBranch,
}: {
  session: { id: string; messages: UIMessage[]; title: string } | null;
  onDeleteMessage?: (msgId: string) => void;
  onRetry?: (msgId: string) => void;
  onBranch?: (msgId: string) => void;
}) {
  const scrollRef = useRef<HTMLDivElement>(null);

  useEffect(() => {
    if (scrollRef.current) {
      scrollRef.current.scrollTop = scrollRef.current.scrollHeight;
    }
  }, [session?.messages.length]);

  if (!session) {
    return (
      <div className="flex h-full flex-1 items-center justify-center bg-[var(--shell-workspace-panel)]">
        <div className="text-center text-muted-foreground">
          <ChatIcon />
          <p className="mt-2 text-[13px]">选择一个会话开始对话</p>
        </div>
      </div>
    );
  }

  if (session.messages.length === 0) {
    return (
      <div className="flex h-full flex-1 items-center justify-center bg-[var(--shell-workspace-panel)]">
        <div className="text-center text-muted-foreground">
          <ChatIcon />
          <p className="mt-2 text-[13px]">「{session.title}」还没有消息</p>
          <p className="mt-0.5 text-[11px] opacity-70">在下方输入框开始对话</p>
        </div>
      </div>
    );
  }

  return (
    <div
      ref={scrollRef}
      className="flex min-h-0 flex-1 flex-col overflow-y-auto scrollbar-dark bg-[var(--shell-workspace-panel)]"
    >
      <div className="mx-auto w-full" style={{ maxWidth: "var(--chat-thread-max-w)" }}>
        <div className="flex flex-col gap-4 px-4 py-5">
          {session.messages.map((m) => (
            <MessageBlock
              key={m.id}
              msg={m}
              onDelete={onDeleteMessage ? () => onDeleteMessage(m.id) : undefined}
              onRetry={onRetry ? () => onRetry(m.id) : undefined}
              onBranch={onBranch ? () => onBranch(m.id) : undefined}
            />
          ))}
        </div>
      </div>
    </div>
  );
}

// —— 消息块 ——
function MessageBlock({
  msg,
  onDelete,
  onRetry,
  onBranch,
}: {
  msg: UIMessage;
  onDelete?: () => void;
  onRetry?: () => void;
  onBranch?: () => void;
}) {
  const isUser = msg.role === "user";
  const [toolOpen, setToolOpen] = useState(true);
  const [copied, setCopied] = useState(false);

  if (isUser) {
    return (
      <div className="group flex justify-end">
        <div className="max-w-[85%]">
          {/* 引用 chips */}
          {msg.refs && msg.refs.length > 0 && (
            <div className="mb-1 flex flex-wrap justify-end gap-1">
              {msg.refs.map((r, i) => (
                <span key={i} className="flex items-center gap-0.5 rounded bg-muted px-1.5 py-0.5 font-mono text-[9px] text-muted-foreground">
                  <LinkSimple size={8} />
                  {r}
                </span>
              ))}
            </div>
          )}
          {/* 气泡：右对齐 16px 圆角 */}
          <div
            className={cn(
              "rounded-2xl px-4 py-2.5 text-[13.5px] leading-relaxed",
              msg.error
                ? "border border-destructive/30 bg-destructive/10 text-destructive"
                : "bg-[var(--interactive-selected)] text-foreground"
            )}
          >
            {msg.streaming ? (
              <span className="flex items-center gap-1">
                {msg.content}
                <span className="inline-block h-3.5 w-[2px] animate-pulse bg-primary" />
              </span>
            ) : (
              <RenderText text={msg.content} align="right" />
            )}
          </div>
        </div>
      </div>
    );
  }

  // 助手消息（块式）
  return (
    <div className="group flex gap-3">
      {/* 角色标识 */}
      <div className="flex h-7 w-7 shrink-0 items-center justify-center rounded-md bg-muted text-muted-foreground">
        <AiIcon />
      </div>

      <div className="min-w-0 flex-1 space-y-2">
        {/* 头部：时间 + 操作 */}
        <div className="flex items-center gap-2">
          <span className="text-[10px] text-muted-foreground/50">
            {msg.created_at
              ? new Date(msg.created_at).toLocaleTimeString([], { hour: "2-digit", minute: "2-digit" })
              : ""}
          </span>
          <div className="ml-auto hidden items-center gap-0.5 rounded-lg bg-[var(--shell-inspector-panel)] px-1 py-0.5 shadow-soft group-hover:flex">
            <ActionBtn title="复制" onClick={() => { void navigator.clipboard?.writeText(msg.content); setCopied(true); setTimeout(() => setCopied(false), 1200); }}>
              {copied ? <Check size={11} className="text-emerald-500" /> : <Copy size={11} />}
            </ActionBtn>
            {onRetry && <ActionBtn title="重试" onClick={onRetry}><ArrowCounterClockwise size={11} /></ActionBtn>}
            {onBranch && <ActionBtn title="分支" onClick={onBranch}><GitBranch size={11} /></ActionBtn>}
            {onDelete && <ActionBtn title="删除" onClick={onDelete} danger><Warning size={11} /></ActionBtn>}
          </div>
        </div>

        {/* 工具调用折叠块 */}
        {msg.toolRecords && msg.toolRecords.length > 0 && (
          <div className="rounded-md border border-[var(--border-default)] bg-muted/30">
            <button
              onClick={() => setToolOpen((v) => !v)}
              className="flex w-full items-center gap-1.5 px-2.5 py-1.5 text-[10px] font-medium text-primary"
            >
              {toolOpen ? <CaretDown size={10} /> : <CaretRight size={10} />}
              工具调用 · {msg.toolRecords.length} 次
            </button>
            {toolOpen && (
              <div className="space-y-1 px-2.5 pb-2">
                {msg.toolRecords.map((t, i) => (
                  <div key={i} className="rounded-md border border-[var(--border-default)] bg-background/60 px-2 py-1.5">
                    <div className="flex items-center gap-1.5">
                      <Badge variant="secondary" className="text-[9px]">{t.name}</Badge>
                      {t.error ? (
                        <span className="flex items-center gap-1 text-[10px] text-destructive">
                          <Warning size={10} />
                          {t.error}
                        </span>
                      ) : (
                        <span className="text-[9px] text-emerald-500">✓ 完成</span>
                      )}
                    </div>
                    {t.arguments && (
                      <pre className="mt-1 whitespace-pre-wrap break-all font-mono text-[10px] text-muted-foreground">{t.arguments}</pre>
                    )}
                    {t.output && (
                      <pre className="mt-0.5 whitespace-pre-wrap break-all font-mono text-[10px] text-foreground/80">{t.output}</pre>
                    )}
                  </div>
                ))}
              </div>
            )}
          </div>
        )}

        {/* 正文（块式文本） */}
        {msg.streaming ? (
          <div className="flex items-start gap-1">
            <RenderText text={msg.content} />
            <span className="mt-1 inline-block h-3.5 w-[2px] animate-pulse bg-primary" />
          </div>
        ) : msg.error ? (
          <div className="flex items-center gap-1.5 text-[12px] text-destructive">
            <Warning size={12} />
            {msg.error}
          </div>
        ) : (
          <RenderText text={msg.content} />
        )}
      </div>
    </div>
  );
}

// —— 文本渲染（标题/列表/代码/行内）——
function RenderText({ text, align = "left" }: { text: string; align?: "left" | "right" }) {
  if (!text) return null;
  const lines = text.split("\n");
  return (
    <div className={cn("space-y-1 text-[13.5px] leading-relaxed text-foreground", align === "right" && "text-right")}>
      {lines.map((line, i) => {
        if (/^```/.test(line)) return <CodeBlock key={i} text={line.replace(/^```.*/, "")} />;
        if (/^#{1,6}\s/.test(line)) {
          return (
            <div key={i} className="pt-1 text-[14px] font-semibold">
              {inline(line.replace(/^#{1,6}\s/, ""))}
            </div>
          );
        }
        if (/^[-*]\s/.test(line)) {
          return (
            <div key={i} className="flex gap-1.5 pl-1">
              <span className="text-primary">•</span>
              <span>{inline(line.replace(/^[-*]\s/, ""))}</span>
            </div>
          );
        }
        if (/^\d+\.\s/.test(line)) {
          return (
            <div key={i} className="flex gap-1.5 pl-1">
              <span className="text-primary">{line.match(/^\d+/)?.[0]}.</span>
              <span>{inline(line.replace(/^\d+\.\s/, ""))}</span>
            </div>
          );
        }
        return <div key={i}>{inline(line) || "\u00A0"}</div>;
      })}
    </div>
  );
}

function inline(text: string) {
  const parts = text.split(/(\*\*[^*]+\*\*|`[^`]+`)/g);
  return parts.map((p, i) => {
    if (p.startsWith("**") && p.endsWith("**")) {
      return <strong key={i} className="font-semibold">{p.slice(2, -2)}</strong>;
    }
    if (p.startsWith("`") && p.endsWith("`")) {
      return <code key={i} className="rounded bg-muted px-1 font-mono text-[11px]">{p.slice(1, -1)}</code>;
    }
    return <span key={i}>{p}</span>;
  });
}

function CodeBlock({ text }: { text: string }) {
  const [open, setOpen] = useState(true);
  if (!text.trim()) return null;
  return (
    <div className="rounded-md border border-[var(--border-default)] bg-muted/40">
      <button
        onClick={() => setOpen((v) => !v)}
        className="flex w-full items-center gap-1 px-2 py-1 text-[9px] text-muted-foreground"
      >
        {open ? <CaretDown size={10} /> : <CaretRight size={10} />}
        代码块
      </button>
      {open && <pre className="overflow-x-auto px-2 pb-2 font-mono text-[11px]">{text}</pre>}
    </div>
  );
}

function ActionBtn({ title, onClick, children, danger = false }: { title: string; onClick: () => void; children: React.ReactNode; danger?: boolean }) {
  return (
    <button
      onClick={onClick}
      className={cn("rounded p-0.5 text-muted-foreground hover:text-foreground", danger && "hover:text-destructive")}
      title={title}
    >
      {children}
    </button>
  );
}

function ChatIcon() {
  return (
    <div className="mx-auto flex h-12 w-12 items-center justify-center rounded-full bg-muted text-muted-foreground">
      <svg width="20" height="20" viewBox="0 0 256 256" fill="currentColor">
        <path d="M216,48H40A16,16,0,0,0,24,64V176a16,16,0,0,0,16,16H72v24a8,8,0,0,0,13.66,5.66L112,200h64a16,16,0,0,0,16-16V160h24a16,16,0,0,0,16-16V64A16,16,0,0,0,216,48ZM72,120a8,8,0,1,1,8-8A8,8,0,0,1,72,120Zm56,0a8,8,0,1,1,8-8A8,8,0,0,1,128,120Zm56,0a8,8,0,1,1,8-8A8,8,0,0,1,184,120Z" />
      </svg>
    </div>
  );
}

function AiIcon() {
  return (
    <svg width="15" height="15" viewBox="0 0 256 256" fill="currentColor">
      <path d="M216.42,79.52,190.14,52.46l-2.05-31.74a8,8,0,0,0-13.84-5.06L146.35,46.3,115.46,36.11a8,8,0,0,0-9.47,10.69l12.46,29.29L87.56,104.44a8,8,0,0,0,3.58,13.49l30.35,7.93,7.93,30.35a8,8,0,0,0,13.49,3.58l28.35-28.89,29.29,12.46a8,8,0,0,0,10.69-9.47l-10.19-30.89,27.06-27.06A8,8,0,0,0,216.42,79.52ZM160,120a8,8,0,1,1,8-8A8,8,0,0,1,160,120Z" />
      <path d="M104,160a8,8,0,0,0-8,8v16H80a8,8,0,0,0,0,16h16v16a8,8,0,0,0,16,0V200h16a8,8,0,0,0,0-16H112V168A8,8,0,0,0,104,160Z" />
    </svg>
  );
}
