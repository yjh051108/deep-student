// SessionList —— chat_v2 会话列表（左栏）
// ------------------------------------------------------------
// - 顶部：标题 + 新建按钮
// - 视图切换：会话 / 回收站
// - 会话列表：分组（置顶优先）、操作（改名/置顶/删除/恢复/彻底删除）
// - 底部：会话统计

import { useChatStore, type UISession } from "@/state/chat";
import { cn, relativeTime, truncate } from "@/lib/utils";
import { Button } from "@/components/ui/Button";
import {
  Plus,
  Trash2,
  MessageSquare,
  Pin,
  PinOff,
  RotateCcw,
  XCircle,
  Pencil,
  Search,
  Inbox,
  Check,
  X,
} from "lucide-react";
import { useState } from "react";
import { Tooltip, TooltipContent, TooltipTrigger } from "@/components/ui/Tooltip";
import { chatV2Api } from "@/lib/chat";

export function SessionList() {
  const sessions = useChatStore((s) => s.sessions);
  const groups = useChatStore((s) => s.groups);
  const activeSessionId = useChatStore((s) => s.activeSessionId);
  const view = useChatStore((s) => s.view);
  const selectSession = useChatStore((s) => s.selectSession);
  const setView = useChatStore((s) => s.setView);
  const createSession = useChatStore((s) => s.createSession);
  const softDeleteSession = useChatStore((s) => s.softDeleteSession);
  const restoreSession = useChatStore((s) => s.restoreSession);
  const purgeSession = useChatStore((s) => s.purgeSession);
  const togglePin = useChatStore((s) => s.togglePin);
  const renameSession = useChatStore((s) => s.renameSession);
  const [keyword, setKeyword] = useState("");

  const visible = sessions.filter((s) =>
    view === "trash" ? s.isDeleted : !s.isDeleted
  );
  const filtered = keyword.trim()
    ? visible.filter((s) => s.title.toLowerCase().includes(keyword.toLowerCase()))
    : visible;

  const activeGroup = groups.find((g) => g.id === activeGroupIdOf(sessions, activeSessionId));

  return (
    <aside className="flex w-64 shrink-0 flex-col border-r border-border bg-card">
      {/* —— 顶部 —— */}
      <div className="flex h-14 shrink-0 items-center justify-between border-b border-border px-3">
        <div>
          <div className="text-xs font-semibold uppercase tracking-wider text-muted-foreground">
            {view === "trash" ? "回收站" : "会话"}
          </div>
          <div className="text-[10px] text-muted-foreground/70">
            {filtered.length} 条 · {activeGroup ? activeGroup.name : "全部"}
          </div>
        </div>
        <div className="flex items-center gap-1">
          <Tooltip>
            <TooltipTrigger asChild>
              <Button
                variant="ghost"
                size="icon"
                className="h-7 w-7"
                onClick={() => setView(view === "trash" ? "normal" : "trash")}
                aria-label="回收站"
              >
                <Trash2 size={13} />
              </Button>
            </TooltipTrigger>
            <TooltipContent side="bottom">回收站</TooltipContent>
          </Tooltip>
          {view !== "trash" && (
            <Tooltip>
              <TooltipTrigger asChild>
                <Button
                  variant="ghost"
                  size="icon"
                  className="h-7 w-7"
                  onClick={() => void createSession("新会话")}
                  aria-label="新建会话"
                >
                  <Plus size={14} />
                </Button>
              </TooltipTrigger>
              <TooltipContent side="bottom">新建会话</TooltipContent>
            </Tooltip>
          )}
        </div>
      </div>

      {/* —— 搜索 —— */}
      <div className="shrink-0 border-b border-border px-2 py-1.5">
        <div className="flex items-center gap-1.5 rounded-md bg-muted/50 px-2 py-1">
          <Search size={11} className="text-muted-foreground" />
          <input
            value={keyword}
            onChange={(e) => setKeyword(e.target.value)}
            placeholder="搜索会话…"
            className="w-full bg-transparent text-[11px] text-foreground placeholder:text-muted-foreground/60 focus:outline-none"
          />
        </div>
      </div>

      {/* —— 列表 —— */}
      <div className="min-h-0 flex-1 overflow-y-auto scrollbar-dark">
        {filtered.length === 0 ? (
          <div className="flex h-full flex-col items-center justify-center gap-2 px-4 text-center">
            <Inbox size={20} className="text-muted-foreground/40" />
            <p className="text-[11px] text-muted-foreground">
              {view === "trash" ? "回收站是空的" : "还没有会话"}
            </p>
          </div>
        ) : (
          <div className="space-y-0.5 p-1.5">
            {filtered
              .slice()
              .sort((a, b) => Number(b.pinned) - Number(a.pinned))
              .map((s) => (
                <SessionRow
                  key={s.id}
                  session={s}
                  active={s.id === activeSessionId}
                  inTrash={view === "trash"}
                  onSelect={() => selectSession(s.id)}
                  onRename={(t) => void renameSession(s.id, t)}
                  onTogglePin={() => void togglePin(s.id)}
                  onDelete={() => void softDeleteSession(s.id)}
                  onRestore={() => void restoreSession(s.id)}
                  onPurge={() => void purgeSession(s.id)}
                />
              ))}
          </div>
        )}
      </div>

      {/* —— 底部统计 —— */}
      <div className="shrink-0 border-t border-border px-3 py-2 text-[10px] text-muted-foreground/60">
        {sessions.filter((s) => !s.isDeleted).length} 活跃 ·{" "}
        {sessions.filter((s) => s.isDeleted).length} 已删除
      </div>
    </aside>
  );
}

// —— 单条会话 ——
function SessionRow({
  session,
  active,
  inTrash,
  onSelect,
  onRename,
  onTogglePin,
  onDelete,
  onRestore,
  onPurge,
}: {
  session: UISession;
  active: boolean;
  inTrash: boolean;
  onSelect: () => void;
  onRename: (t: string) => void;
  onTogglePin: () => void;
  onDelete: () => void;
  onRestore: () => void;
  onPurge: () => void;
}) {
  const [editing, setEditing] = useState(false);
  const [draft, setDraft] = useState(session.title);

  if (editing) {
    return (
      <div className="flex items-center gap-1 rounded-md border border-primary/40 bg-primary/5 px-2 py-1.5">
        <input
          autoFocus
          value={draft}
          onChange={(e) => setDraft(e.target.value)}
          onKeyDown={(e) => {
            if (e.key === "Enter" && draft.trim()) {
              onRename(draft.trim());
              setEditing(false);
            }
            if (e.key === "Escape") setEditing(false);
          }}
          className="w-full bg-transparent text-[12px] text-foreground focus:outline-none"
        />
        <button onClick={() => { onRename(draft.trim()); setEditing(false); }} className="text-primary">
          <Check size={12} />
        </button>
        <button onClick={() => setEditing(false)} className="text-muted-foreground">
          <X size={12} />
        </button>
      </div>
    );
  }

  return (
    <div
      onClick={onSelect}
      className={cn(
        "group relative flex cursor-pointer items-center gap-2 rounded-md px-2 py-1.5 text-[12px] transition-colors",
        active
          ? "bg-primary/12 font-medium text-primary"
          : "text-muted-foreground hover:bg-accent"
      )}
    >
      <MessageSquare size={12} className="shrink-0 opacity-70" />
      <span className="min-w-0 flex-1 truncate">{session.title}</span>
      {session.pinned && <Pin size={10} className="shrink-0 text-amber-500" />}
      <span className="shrink-0 text-[9px] text-muted-foreground/50">
        {relativeTime(new Date(session.updatedAt).getTime())}
      </span>

      {/* hover 操作 */}
      <div className="absolute right-1 top-1/2 hidden -translate-y-1/2 items-center gap-0.5 rounded bg-card px-0.5 group-hover:flex">
        {!inTrash ? (
          <>
            <button
              title="重命名"
              onClick={(e) => { e.stopPropagation(); setEditing(true); }}
              className="rounded p-0.5 text-muted-foreground hover:text-foreground"
            >
              <Pencil size={11} />
            </button>
            <button
              title={session.pinned ? "取消置顶" : "置顶"}
              onClick={(e) => { e.stopPropagation(); onTogglePin(); }}
              className="rounded p-0.5 text-muted-foreground hover:text-amber-500"
            >
              {session.pinned ? <PinOff size={11} /> : <Pin size={11} />}
            </button>
            <button
              title="删除"
              onClick={(e) => { e.stopPropagation(); onDelete(); }}
              className="rounded p-0.5 text-muted-foreground hover:text-destructive"
            >
              <Trash2 size={11} />
            </button>
          </>
        ) : (
          <>
            <button
              title="恢复"
              onClick={(e) => { e.stopPropagation(); onRestore(); }}
              className="rounded p-0.5 text-muted-foreground hover:text-primary"
            >
              <RotateCcw size={11} />
            </button>
            <button
              title="彻底删除"
              onClick={(e) => { e.stopPropagation(); onPurge(); }}
              className="rounded p-0.5 text-muted-foreground hover:text-destructive"
            >
              <XCircle size={11} />
            </button>
          </>
        )}
      </div>
    </div>
  );
}

function activeGroupIdOf(sessions: UISession[], activeId: string | null): string | null {
  if (!activeId) return null;
  return sessions.find((s) => s.id === activeId)?.groupId ?? null;
}

// 保持 chatV2Api 引用（供未来分组管理扩展）
void chatV2Api;
void truncate;
