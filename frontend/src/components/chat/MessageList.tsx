// MessageList —— 消息流（中栏上）
// ------------------------------------------------------------
// 设计要点：
// - 滚动容器：auto-scroll 到底部
// - 消息气泡：user（右对齐 primary）/ assistant（左对齐 muted）
// - 空态：欢迎引导
// - Markdown 渲染：基础排版（标题 / 列表 / 代码块 / 链接）

import { ChatSession, ChatMessage } from "@/state/chat";
import { cn, formatTime } from "@/lib/utils";
import { useEffect, useRef } from "react";
import { Sparkles, User, AlertCircle } from "lucide-react";

interface MessageListProps {
  session: ChatSession | null;
}

export function MessageList({ session }: MessageListProps) {
  const scrollRef = useRef<HTMLDivElement>(null);

  // 消息变化时自动滚动到底
  useEffect(() => {
    const el = scrollRef.current;
    if (el) {
      el.scrollTop = el.scrollHeight;
    }
  }, [session?.messages]);

  if (!session) {
    return <EmptyState />;
  }

  if (session.messages.length === 0) {
    return <EmptyState sessionTitle={session.title} />;
  }

  return (
    <div
      ref={scrollRef}
      className="min-h-0 flex-1 overflow-y-auto scrollbar-dark"
    >
      <div className="mx-auto flex max-w-thread flex-col gap-6 px-6 py-8">
        {session.messages.map((m) => (
          <MessageBubble key={m.id} message={m} />
        ))}
      </div>
    </div>
  );
}

// —— 消息气泡 ——
function MessageBubble({ message }: { message: ChatMessage }) {
  const isUser = message.role === "user";
  const isAssistant = message.role === "assistant";

  return (
    <div
      className={cn(
        "flex w-full gap-3 animate-fadeSlideUp",
        isUser && "flex-row-reverse"
      )}
    >
      {/* 头像 */}
      <div
        className={cn(
          "flex h-7 w-7 shrink-0 items-center justify-center rounded-full text-xs font-semibold",
          isUser
            ? "bg-primary text-primary-foreground"
            : "bg-gradient-to-br from-primary to-brand-primary-dark text-primary-foreground"
        )}
      >
        {isUser ? <User size={14} /> : <Sparkles size={14} />}
      </div>

      {/* 消息体 */}
      <div
        className={cn(
          "flex min-w-0 max-w-[calc(100%-2.5rem)] flex-col gap-1",
          isUser && "items-end"
        )}
      >
        <div className="flex items-center gap-2 text-[10px] text-muted-foreground/70">
          <span className="font-medium">
            {isUser ? "你" : isAssistant ? "Assistant" : "System"}
          </span>
          <span>{formatTime(message.createdAt)}</span>
          {message.refs && message.refs.length > 0 && (
            <span className="rounded bg-muted px-1 py-0.5 text-[10px]">
              refs: {message.refs.length}
            </span>
          )}
        </div>

        {/* 气泡内容 */}
        <div
          className={cn(
            "rounded-row px-3.5 py-2.5 text-sm leading-relaxed",
            isUser
              ? "bg-primary text-primary-foreground"
              : "bg-card border border-border text-foreground shadow-soft"
          )}
        >
          {message.error ? (
            <div className="flex items-center gap-2 text-destructive">
              <AlertCircle size={14} />
              <span>{message.error}</span>
            </div>
          ) : message.content ? (
            <MarkdownLite text={message.content} />
          ) : message.streaming ? (
            <span className="inline-flex items-center gap-1 text-muted-foreground">
              <span className="animate-blink">●</span>
              <span>思考中…</span>
            </span>
          ) : (
            <span className="text-muted-foreground">(空)</span>
          )}
          {message.streaming && message.content && (
            <span className="ml-0.5 inline-block h-3.5 w-1 animate-blink bg-foreground align-middle" />
          )}
        </div>
      </div>
    </div>
  );
}

// —— 极简 Markdown 渲染 ——
// 支持：代码块、内联代码、标题、列表、链接、加粗、斜体
// 不引入 react-markdown 等重型依赖
function MarkdownLite({ text }: { text: string }) {
  const blocks = parseBlocks(text);
  return (
    <div className="space-y-2">
      {blocks.map((b, i) => {
        if (b.type === "code") {
          return (
            <pre
              key={i}
              className="overflow-x-auto rounded-md bg-muted/80 p-2.5 text-xs text-foreground"
            >
              <code className="font-mono">{b.content}</code>
            </pre>
          );
        }
        if (b.type === "heading") {
          const level = b.level ?? 1;
          const Tag = (`h${Math.min(level, 4)}` as "h1" | "h2" | "h3" | "h4");
          return (
            <Tag
              key={i}
              className={cn(
                "font-semibold text-foreground",
                level === 1 && "text-base",
                level === 2 && "text-sm",
                (level === 3 || level === 4) && "text-xs"
              )}
            >
              {renderInline(b.content)}
            </Tag>
          );
        }
        if (b.type === "list") {
          return (
            <ul key={i} className="list-disc space-y-0.5 pl-4 text-xs">
              {b.items?.map((item, j) => (
                <li key={j}>{renderInline(item)}</li>
              ))}
            </ul>
          );
        }
        // paragraph
        return (
          <p key={i} className="whitespace-pre-wrap text-sm">
            {renderInline(b.content)}
          </p>
        );
      })}
    </div>
  );
}

type Block =
  | { type: "code"; content: string }
  | { type: "heading"; content: string; level: number }
  | { type: "list"; items: string[] }
  | { type: "paragraph"; content: string };

function parseBlocks(text: string): Block[] {
  const lines = text.split("\n");
  const out: Block[] = [];
  let i = 0;
  while (i < lines.length) {
    const line = lines[i];
    // 代码块
    if (line.startsWith("```")) {
      const buf: string[] = [];
      i++;
      while (i < lines.length && !lines[i].startsWith("```")) {
        buf.push(lines[i]);
        i++;
      }
      i++; // skip closing ```
      out.push({ type: "code", content: buf.join("\n") });
      continue;
    }
    // 标题
    const hMatch = /^(#{1,4})\s+(.*)$/.exec(line);
    if (hMatch) {
      out.push({
        type: "heading",
        content: hMatch[2],
        level: hMatch[1].length,
      });
      i++;
      continue;
    }
    // 列表
    if (/^[-*]\s+/.test(line)) {
      const items: string[] = [];
      while (i < lines.length && /^[-*]\s+/.test(lines[i])) {
        items.push(lines[i].replace(/^[-*]\s+/, ""));
        i++;
      }
      out.push({ type: "list", items });
      continue;
    }
    // 段落（连续非空行合并）
    if (line.trim() === "") {
      i++;
      continue;
    }
    const buf: string[] = [];
    while (i < lines.length && lines[i].trim() !== "" && !/^```/.test(lines[i]) && !/^#{1,4}\s/.test(lines[i]) && !/^[-*]\s+/.test(lines[i])) {
      buf.push(lines[i]);
      i++;
    }
    out.push({ type: "paragraph", content: buf.join("\n") });
  }
  return out;
}

function renderInline(text: string): React.ReactNode {
  // 内联代码 + 加粗 + 斜体 + 链接
  const parts: React.ReactNode[] = [];
  const regex = /(`[^`]+`|\*\*[^*]+\*\*|\*[^*]+\*|\[[^\]]+\]\([^)]+\))/g;
  let lastIdx = 0;
  let match: RegExpExecArray | null;
  let key = 0;
  while ((match = regex.exec(text)) !== null) {
    if (match.index > lastIdx) {
      parts.push(text.slice(lastIdx, match.index));
    }
    const tok = match[0];
    if (tok.startsWith("`")) {
      parts.push(
        <code key={key++} className="rounded bg-muted/80 px-1 py-0.5 font-mono text-[0.85em]">
          {tok.slice(1, -1)}
        </code>
      );
    } else if (tok.startsWith("**")) {
      parts.push(
        <strong key={key++} className="font-semibold">
          {tok.slice(2, -2)}
        </strong>
      );
    } else if (tok.startsWith("*")) {
      parts.push(
        <em key={key++} className="italic">
          {tok.slice(1, -1)}
        </em>
      );
    } else if (tok.startsWith("[")) {
      const m = /\[([^\]]+)\]\(([^)]+)\)/.exec(tok);
      if (m) {
        parts.push(
          <a
            key={key++}
            href={m[2]}
            target="_blank"
            rel="noopener noreferrer"
            className="text-primary underline hover:text-primary/80"
          >
            {m[1]}
          </a>
        );
      }
    }
    lastIdx = match.index + tok.length;
  }
  if (lastIdx < text.length) {
    parts.push(text.slice(lastIdx));
  }
  return <>{parts}</>;
}

// —— 空态 ——
function EmptyState({ sessionTitle }: { sessionTitle?: string }) {
  return (
    <div className="flex min-h-0 flex-1 items-center justify-center overflow-y-auto scrollbar-dark">
      <div className="mx-auto flex max-w-md flex-col items-center gap-4 px-6 text-center animate-fadeSlideUp">
        <div className="flex h-14 w-14 items-center justify-center rounded-2xl bg-gradient-to-br from-primary to-brand-primary-dark text-primary-foreground shadow-floating">
          <Sparkles size={28} />
        </div>
        <div>
          <h2 className="text-lg font-semibold text-foreground">
            {sessionTitle ? `会话：${sessionTitle}` : "开始你的第一次对话"}
          </h2>
          <p className="mt-1 text-sm text-muted-foreground">
            输入消息发送给 LLM，支持 refs 引用与深度思考。
          </p>
        </div>
        <div className="mt-2 grid grid-cols-2 gap-2 text-xs text-muted-foreground">
          <div className="rounded-md border border-border bg-card px-3 py-2 text-left">
            <div className="mb-0.5 font-medium text-foreground">↩ 发送</div>
            <div>Enter 发送，Shift+Enter 换行</div>
          </div>
          <div className="rounded-md border border-border bg-card px-3 py-2 text-left">
            <div className="mb-0.5 font-medium text-foreground">⌘ 深度思考</div>
            <div>切换 deepThink 模式让 LLM 展示推理</div>
          </div>
          <div className="rounded-md border border-border bg-card px-3 py-2 text-left">
            <div className="mb-0.5 font-medium text-foreground">📎 引用</div>
            <div>右侧引用面板添加 vfs:// 资源</div>
          </div>
          <div className="rounded-md border border-border bg-card px-3 py-2 text-left">
            <div className="mb-0.5 font-medium text-foreground">⚡ 多模型</div>
            <div>顶栏切换 Provider 与 A/B 槽</div>
          </div>
        </div>
      </div>
    </div>
  );
}
