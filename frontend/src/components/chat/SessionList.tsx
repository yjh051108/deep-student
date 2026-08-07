// SessionList —— 会话列表（左栏）
// ------------------------------------------------------------
// 设计要点：
// - 顶部：标题 + 新建按钮
// - 中间：会话列表（按 updatedAt 倒序）
// - 底部：会话统计

import { useChatStore, ChatSession } from "@/state/chat";
import { cn, relativeTime, truncate } from "@/lib/utils";
import { Button } from "@/components/ui/Button";
import { Plus, Trash2, MessageSquare } from "lucide-react";
import { Tooltip, TooltipContent, TooltipTrigger } from "@/components/ui/Tooltip";

export function SessionList() {
  const sessions = useChatStore((s) => s.sessions);
  const activeSessionId = useChatStore((s) => s.activeSessionId);
  const selectSession = useChatStore((s) => s.selectSession);
  const createSession = useChatStore((s) => s.createSession);
  const removeSession = useChatStore((s) => s.removeSession);

  return (
    <aside className="flex w-64 shrink-0 flex-col border-r border-border bg-card">
      {/* —— 顶部：标题 + 新建按钮 —— */}
      <div className="flex h-14 shrink-0 items-center justify-between border-b border-border px-3">
        <div>
          <div className="text-xs font-semibold uppercase tracking-wider text-muted-foreground">
            会话
          </div>
          <div className="text-[10px] text-muted-foreground/70">
            共 {sessions.length} 条
          </div>
        </div>
        <Tooltip>
          <TooltipTrigger asChild>
            <Button
              variant="ghost"
              size="icon"
              className="h-7 w-7"
              onClick={() => createSession("新会话")}
              aria-label="新建会话"
            >
              <Plus size={14} />
            </Button>
          </TooltipTrigger>
          <TooltipContent side="bottom">新建会话 (Ctrl+N)</TooltipContent>
        </Tooltip>
      </div>

      {/* —— 中间：会话列表 —— */}
      <div className="min-h-0 flex-1 overflow-y-auto scrollbar-dark">
        {sessions.length === 0 ? (
          <div className="flex h-full flex-col items-center justify-center gap-2 px-4 text-center">
            <MessageSquare size={28} className="text-muted-foreground/40" />
            <div className="text-xs text-muted-foreground">
              还没有会话
            </div>
            <Button
              variant="outline"
              size="sm"
              onClick={() => createSession("新会话")}
              className="mt-1"
            >
              <Plus size={12} />
              新建会话
            </Button>
          </div>
        ) : (
          <ul className="py-1">
            {sessions.map((s) => (
              <li key={s.id}>
                <SessionRow
                  session={s}
                  active={s.id === activeSessionId}
                  onSelect={() => selectSession(s.id)}
                  onRemove={() => removeSession(s.id)}
                />
              </li>
            ))}
          </ul>
        )}
      </div>
    </aside>
  );
}

function SessionRow({
  session,
  active,
  onSelect,
  onRemove,
}: {
  session: ChatSession;
  active: boolean;
  onSelect: () => void;
  onRemove: () => void;
}) {
  const lastMsg = session.messages[session.messages.length - 1];
  const preview = lastMsg
    ? truncate(lastMsg.content.replace(/\n/g, " "), 32)
    : "空会话";

  return (
    <div
      className={cn(
        "group relative flex cursor-pointer items-start gap-2 px-3 py-2 transition-colors",
        active
          ? "bg-primary/10 text-foreground"
          : "text-muted-foreground hover:bg-accent hover:text-foreground"
      )}
      onClick={onSelect}
    >
      {/* 激活态左侧指示条 */}
      {active && (
        <span className="absolute left-0 top-1/2 h-6 w-0.5 -translate-y-1/2 rounded-r-full bg-primary" />
      )}
      <MessageSquare
        size={14}
        className={cn(
          "mt-0.5 shrink-0",
          active ? "text-primary" : "text-muted-foreground/70"
        )}
      />
      <div className="min-w-0 flex-1">
        <div className="flex items-center justify-between gap-2">
          <div className="truncate text-sm font-medium">
            {session.title || "未命名会话"}
          </div>
          <span className="shrink-0 text-[10px] text-muted-foreground/60">
            {relativeTime(session.updatedAt)}
          </span>
        </div>
        <div className="truncate text-[11px] text-muted-foreground/80">
          {preview}
        </div>
        <div className="mt-0.5 flex items-center gap-2 text-[10px] text-muted-foreground/60">
          <span className="rounded bg-muted px-1 py-0.5">{session.provider}</span>
          <span>{session.messages.length} 条</span>
        </div>
      </div>
      {/* 删除按钮（hover 显示） */}
      <button
        type="button"
        onClick={(e) => {
          e.stopPropagation();
          onRemove();
        }}
        className="shrink-0 rounded p-1 text-muted-foreground/60 opacity-0 transition-opacity hover:bg-destructive/15 hover:text-destructive group-hover:opacity-100"
        aria-label="删除会话"
      >
        <Trash2 size={12} />
      </button>
    </div>
  );
}
