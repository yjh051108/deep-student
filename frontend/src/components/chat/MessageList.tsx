// MessageList —— chat_v2 块式消息渲染
// ------------------------------------------------------------
// 每个消息渲染为独立块：
// - 角色徽标 + 时间
// - 工具调用记录折叠块（名称/入参/输出/错误）
// - reasoning（思考链）折叠
// - refs 引用 chips
// - 流式光标 / 错误状态
// 简单 Markdown 行内渲染（加粗/行内代码/列表符号）。

import { useEffect, useRef, useState } from "react";
import { cn } from "@/lib/utils";
import type { UIMessage } from "@/state/chat";
import {
  Bot,
  User,
  Wrench,
  ChevronDown,
  ChevronRight,
  Link2,
  AlertCircle,
  Trash2,
} from "lucide-react";
import { Badge } from "@/components/ui/Badge";

export function MessageList({
  session,
  onDeleteMessage,
}: {
  session: { id: string; messages: UIMessage[]; title: string } | null;
  onDeleteMessage?: (msgId: string) => void;
}) {
  const scrollRef = useRef<HTMLDivElement>(null);

  useEffect(() => {
    if (scrollRef.current) {
      scrollRef.current.scrollTop = scrollRef.current.scrollHeight;
    }
  }, [session?.messages.length]);

  if (!session) {
    return (
      <div className="flex h-full flex-1 items-center justify-center bg-background">
        <div className="text-center text-muted-foreground">
          <Bot size={32} className="mx-auto mb-2 opacity-40" />
          <p className="text-sm">选择一个会话开始对话</p>
        </div>
      </div>
    );
  }

  if (session.messages.length === 0) {
    return (
      <div className="flex h-full flex-1 items-center justify-center bg-background">
        <div className="text-center text-muted-foreground">
          <Bot size={32} className="mx-auto mb-2 opacity-40" />
          <p className="text-sm">「{session.title}」还没有消息</p>
          <p className="mt-1 text-xs opacity-70">在下方输入框开始对话</p>
        </div>
      </div>
    );
  }

  return (
    <div
      ref={scrollRef}
      className="flex min-h-0 flex-1 flex-col gap-2 overflow-y-auto scrollbar-dark bg-background p-4"
    >
      {session.messages.map((m) => (
        <MessageBlock
          key={m.id}
          msg={m}
          onDelete={onDeleteMessage ? () => onDeleteMessage(m.id) : undefined}
        />
      ))}
    </div>
  );
}

// —— 单条消息块 ——
function MessageBlock({
  msg,
  onDelete,
}: {
  msg: UIMessage;
  onDelete?: () => void;
}) {
  const isUser = msg.role === "user";
  const [toolOpen, setToolOpen] = useState(true);
  const [reasonOpen, setReasonOpen] = useState(false);

  return (
    <div
      className={cn(
        "group flex gap-3 rounded-lg border p-3 transition-colors",
        isUser ? "border-primary/20 bg-primary/5" : "border-border bg-card"
      )}
    >
      {/* 角色图标 */}
      <div
        className={cn(
          "flex h-7 w-7 shrink-0 items-center justify-center rounded-md",
          isUser ? "bg-primary/15 text-primary" : "bg-muted text-muted-foreground"
        )}
      >
        {isUser ? <User size={14} /> : <Bot size={14} />}
      </div>

      {/* 内容 */}
      <div className="min-w-0 flex-1 space-y-2">
        {/* 头部：角色 + 时间 + 操作 */}
        <div className="flex items-center gap-2">
          <span className="text-[10px] font-semibold uppercase tracking-wider text-muted-foreground/70">
            {isUser ? "你" : msg.role === "tool" ? "工具" : "AI"}
          </span>
          <span className="text-[9px] text-muted-foreground/50">
            {msg.created_at
              ? new Date(msg.created_at).toLocaleTimeString([], {
                  hour: "2-digit",
                  minute: "2-digit",
                })
              : ""}
          </span>
          {onDelete && (
            <button
              onClick={onDelete}
              className="ml-auto hidden rounded p-0.5 text-muted-foreground/50 hover:text-destructive group-hover:block"
              title="删除该条消息"
            >
              <Trash2 size={11} />
            </button>
          )}
        </div>

        {/* 思考链折叠 */}
        {msg.reasoning && (
          <div className="rounded-md border border-border/60 bg-muted/40">
            <button
              onClick={() => setReasonOpen((v) => !v)}
              className="flex w-full items-center gap-1.5 px-2 py-1 text-[10px] text-muted-foreground"
            >
              {reasonOpen ? <ChevronDown size={11} /> : <ChevronRight size={11} />}
              思考链
            </button>
            {reasonOpen && (
              <pre className="whitespace-pre-wrap px-2 pb-2 text-[11px] text-muted-foreground/80">
                {msg.reasoning}
              </pre>
            )}
          </div>
        )}

        {/* 工具调用记录折叠块 */}
        {msg.toolRecords && msg.toolRecords.length > 0 && (
          <div className="rounded-md border border-primary/20 bg-primary/5">
            <button
              onClick={() => setToolOpen((v) => !v)}
              className="flex w-full items-center gap-1.5 px-2 py-1 text-[10px] font-medium text-primary"
            >
              {toolOpen ? <ChevronDown size={11} /> : <ChevronRight size={11} />}
              <Wrench size={10} />
              工具调用 · {msg.toolRecords.length} 次
            </button>
            {toolOpen && (
              <div className="space-y-1 px-2 pb-2">
                {msg.toolRecords.map((t, i) => (
                  <div
                    key={i}
                    className="rounded border border-border/60 bg-background/60 px-2 py-1.5"
                  >
                    <div className="flex items-center gap-1.5">
                      <Badge variant="secondary" className="text-[9px]">
                        {t.name}
                      </Badge>
                      {t.error ? (
                        <span className="flex items-center gap-1 text-[10px] text-destructive">
                          <AlertCircle size={10} />
                          {t.error}
                        </span>
                      ) : (
                        <span className="text-[9px] text-emerald-500">✓ 完成</span>
                      )}
                    </div>
                    {t.arguments && (
                      <pre className="mt-1 whitespace-pre-wrap break-all text-[10px] text-muted-foreground">
                        {t.arguments}
                      </pre>
                    )}
                    {t.output && (
                      <pre className="mt-0.5 whitespace-pre-wrap break-all text-[10px] text-foreground/80">
                        {t.output}
                      </pre>
                    )}
                  </div>
                ))}
              </div>
            )}
          </div>
        )}

        {/* 引用 chips */}
        {msg.refs && msg.refs.length > 0 && (
          <div className="flex flex-wrap items-center gap-1">
            <Link2 size={10} className="text-muted-foreground" />
            {msg.refs.map((r, i) => (
              <span
                key={i}
                className="rounded bg-muted px-1.5 py-0.5 font-mono text-[9px] text-muted-foreground"
              >
                {r}
              </span>
            ))}
          </div>
        )}

        {/* 正文 */}
        {msg.streaming ? (
          <div className="flex items-start gap-1">
            <RenderText text={msg.content} />
            <span className="mt-0.5 inline-block h-3.5 w-[2px] animate-pulse bg-primary" />
          </div>
        ) : msg.error ? (
          <div className="flex items-center gap-1.5 text-[12px] text-destructive">
            <AlertCircle size={12} />
            {msg.error}
          </div>
        ) : (
          <RenderText text={msg.content} />
        )}
      </div>
    </div>
  );
}

// —— 简易行内渲染 ——
function RenderText({ text }: { text: string }) {
  if (!text) return null;
  const lines = text.split("\n");
  return (
    <div className="space-y-1 text-[13px] leading-relaxed text-foreground">
      {lines.map((line, i) => {
        if (line.startsWith("```")) {
          return <CodeBlock key={i} text={line.replace(/^```.*/, "")} />;
        }
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
      return (
        <strong key={i} className="font-semibold">
          {p.slice(2, -2)}
        </strong>
      );
    }
    if (p.startsWith("`") && p.endsWith("`")) {
      return (
        <code key={i} className="rounded bg-muted px-1 font-mono text-[11px]">
          {p.slice(1, -1)}
        </code>
      );
    }
    return <span key={i}>{p}</span>;
  });
}

function CodeBlock({ text }: { text: string }) {
  const [open, setOpen] = useState(true);
  if (!text.trim()) return null;
  return (
    <div className="rounded-md border border-border/60 bg-muted/50">
      <button
        onClick={() => setOpen((v) => !v)}
        className="flex w-full items-center gap-1 px-2 py-1 text-[9px] text-muted-foreground"
      >
        {open ? <ChevronDown size={10} /> : <ChevronRight size={10} />}
        代码块
      </button>
      {open && (
        <pre className="overflow-x-auto px-2 pb-2 font-mono text-[11px]">{text}</pre>
      )}
    </div>
  );
}
