// TodoPage —— 待办管理
// ------------------------------------------------------------
// 布局：
// 1. 左：视图导航（今日/逾期/未来/全部/回收站）+ 列表管理
// 2. 中：条目列表（完成切换 / 优先级 / 标签 / 子任务 / 编辑 / AI 拆解）
// 3. 顶部：搜索 + 新增条目
//
// 对接后端：todoApi（TodoXxx 方法）

import { useEffect, useMemo, useState } from "react";
import { useTodoStore } from "@/state/todo";
import { Button } from "@/components/ui/Button";
import { Input } from "@/components/ui/Input";
import { Textarea } from "@/components/ui/Textarea";
import { Badge } from "@/components/ui/Badge";
import { cn } from "@/lib/utils";
import {
  CheckCircle2,
  Circle,
  ListChecks,
  ListTodo,
  CalendarDays,
  CalendarClock,
  CalendarRange,
  Trash2,
  Search,
  Plus,
  Loader2,
  Sparkles,
  ChevronDown,
  ChevronRight,
  Flag,
  Hash,
  RotateCcw,
} from "lucide-react";
import type { TodoItem, TodoList } from "@/lib/todo";

export function TodoPage() {
  const view = useTodoStore((s) => s.view);
  const currentListId = useTodoStore((s) => s.currentListId);
  const loadAll = useTodoStore((s) => s.loadAll);
  const loadView = useTodoStore((s) => s.loadView);
  const loadItems = useTodoStore((s) => s.loadItems);

  useEffect(() => {
    void loadAll();
    if (view === "today") void loadView("today");
    else if (view === "list" || view === "all") void loadView("all");
    else if (view === "trash") useTodoStore.getState().loadTrash();
    else void loadView(view);
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, []);

  const selectList = (id: string) => {
    useTodoStore.setState({ currentListId: id, view: "list" });
    void loadItems(id);
  };

  return (
    <div className="flex h-full w-full min-h-0 bg-background">
      {/* —— 左：视图 + 列表 —— */}
      <aside className="w-60 shrink-0 border-r border-border bg-card">
        <LeftPanel onSelectList={selectList} currentListId={currentListId} />
      </aside>

      {/* —— 中：条目 —— */}
      <section className="flex min-w-0 flex-1 flex-col">
        <ItemList />
      </section>
    </div>
  );
}

// —— 左：视图与列表 ——
function LeftPanel({
  onSelectList,
  currentListId,
}: {
  onSelectList: (id: string) => void;
  currentListId: string | null;
}) {
  const lists = useTodoStore((s) => s.lists);
  const view = useTodoStore((s) => s.view);
  const summary = useTodoStore((s) => s.summary);
  const loadView = useTodoStore((s) => s.loadView);
  const loadTrash = useTodoStore((s) => s.loadTrash);
  const createList = useTodoStore((s) => s.createList);
  const deleteList = useTodoStore((s) => s.deleteList);
  const toggleListFavorite = useTodoStore((s) => s.updateList);
  const [newListName, setNewListName] = useState("");

  const views = [
    {
      key: "today" as const,
      label: "今日",
      icon: CalendarDays,
      count: summary?.dueTodayCount ?? 0,
    },
    {
      key: "overdue" as const,
      label: "逾期",
      icon: CalendarClock,
      count: summary?.overdueCount ?? 0,
    },
    {
      key: "upcoming" as const,
      label: "未来 7 天",
      icon: CalendarRange,
    },
    { key: "all" as const, label: "全部待办", icon: ListTodo },
  ];

  return (
    <div className="flex h-full w-full flex-col">
      <div className="shrink-0 border-b border-border px-3 py-2.5">
        <div className="flex items-center gap-2">
          <div className="flex h-6 w-6 items-center justify-center rounded-md bg-primary/15 text-primary">
            <ListChecks size={13} />
          </div>
          <span className="text-[11px] font-semibold uppercase tracking-wider text-muted-foreground/70">
            待办
          </span>
        </div>
      </div>

      <div className="min-h-0 flex-1 overflow-y-auto scrollbar-dark px-2 py-2">
        {/* 视图 */}
        <div className="space-y-0.5">
          {views.map((v) => (
            <button
              key={v.key}
              onClick={() => void loadView(v.key)}
              className={cn(
                "flex w-full items-center gap-2 rounded-md px-2 py-1.5 text-left text-[12px] transition-colors",
                view === v.key
                  ? "bg-primary/12 font-medium text-primary"
                  : "text-muted-foreground hover:bg-accent"
              )}
            >
              <v.icon size={13} className="shrink-0" />
              <span className="flex-1 truncate">{v.label}</span>
              {v.count !== undefined && v.count > 0 && (
                <span className="rounded-full bg-primary/15 px-1.5 text-[10px] font-medium text-primary">
                  {v.count}
                </span>
              )}
            </button>
          ))}
          <button
            onClick={() => void loadTrash()}
            className={cn(
              "flex w-full items-center gap-2 rounded-md px-2 py-1.5 text-left text-[12px] transition-colors",
              view === "trash"
                ? "bg-primary/12 font-medium text-primary"
                : "text-muted-foreground hover:bg-accent"
            )}
          >
            <Trash2 size={13} className="shrink-0" />
            <span className="flex-1">回收站</span>
          </button>
        </div>

        <div className="mt-3 border-t border-border/60 pt-3">
          <div className="mb-1.5 px-2 text-[10px] font-semibold uppercase tracking-wider text-muted-foreground/70">
            列表
          </div>
          <div className="space-y-0.5">
            {lists.map((l) => (
              <div key={l.id} className="group relative">
                <button
                  onClick={() => onSelectList(l.id)}
                  className={cn(
                    "flex w-full items-center gap-2 rounded-md px-2 py-1.5 text-left text-[12px] transition-colors",
                    currentListId === l.id
                      ? "bg-primary/12 font-medium text-primary"
                      : "text-muted-foreground hover:bg-accent"
                  )}
                >
                  <span
                    className="h-2 w-2 shrink-0 rounded-full"
                    style={{ background: l.color || "#8b5cf6" }}
                  />
                  <span className="min-w-0 flex-1 truncate">{l.name}</span>
                  {l.isInbox && (
                    <span className="text-[9px] text-muted-foreground/50">收件箱</span>
                  )}
                  {l.pendingCount !== undefined && l.pendingCount > 0 && (
                    <span className="rounded-full bg-muted px-1.5 text-[10px] text-muted-foreground">
                      {l.pendingCount}
                    </span>
                  )}
                </button>
                {!l.isInbox && (
                  <div className="absolute right-1 top-1/2 hidden -translate-y-1/2 items-center gap-0.5 bg-card group-hover:flex">
                    <button
                      className="rounded p-0.5 text-muted-foreground hover:text-amber-500"
                      title={l.isFavorite ? "取消收藏" : "收藏"}
                      onClick={() =>
                        void toggleListFavorite({
                          id: l.id,
                          favorite: !l.isFavorite,
                        })
                      }
                    >
                      ★
                    </button>
                    <button
                      className="rounded p-0.5 text-muted-foreground hover:text-destructive"
                      title="删除"
                      onClick={() => void deleteList(l.id)}
                    >
                      <Trash2 size={11} />
                    </button>
                  </div>
                )}
              </div>
            ))}
          </div>

          {/* 新建列表 */}
          <div className="mt-2 flex gap-1 px-1">
            <Input
              value={newListName}
              onChange={(e) => setNewListName(e.target.value)}
              onKeyDown={(e) => {
                if (e.key === "Enter") {
                  void createList(newListName);
                  setNewListName("");
                }
              }}
              placeholder="新列表…"
              className="h-7 text-[11px]"
            />
            <Button
              size="sm"
              variant="ghost"
              className="h-7 px-2"
              disabled={!newListName.trim()}
              onClick={() => {
                void createList(newListName);
                setNewListName("");
              }}
            >
              <Plus size={13} />
            </Button>
          </div>
        </div>
      </div>
    </div>
  );
}

// —— 中：条目列表 ——
function ItemList() {
  const items = useTodoStore((s) => s.items);
  const view = useTodoStore((s) => s.view);
  const loading = useTodoStore((s) => s.loading);
  const toast = useTodoStore((s) => s.toast);
  const error = useTodoStore((s) => s.error);
  const toggleItem = useTodoStore((s) => s.toggleItem);
  const deleteItem = useTodoStore((s) => s.deleteItem);
  const restoreItem = useTodoStore((s) => s.restoreItem);
  const purgeItem = useTodoStore((s) => s.purgeItem);
  const restoreList = useTodoStore((s) => s.restoreList);
  const purgeList = useTodoStore((s) => s.purgeList);

  const [newTitle, setNewTitle] = useState("");
  const [search, setSearch] = useState("");
  const [aiOpen, setAiOpen] = useState(false);
  const [aiTitle, setAiTitle] = useState("");
  const [aiNotes, setAiNotes] = useState("");
  const [aiBusy, setAiBusy] = useState(false);
  const createItem = useTodoStore((s) => s.createItem);
  const aiBreakdown = useTodoStore((s) => s.aiBreakdown);
  const currentListId = useTodoStore((s) => s.currentListId);

  // 视图标题
  const title = useMemo(() => {
    switch (view) {
      case "today":
        return "今日待办";
      case "overdue":
        return "逾期待办";
      case "upcoming":
        return "未来 7 天";
      case "all":
        return "全部待办";
      case "trash":
        return "回收站";
      case "list":
        return "列表";
      default:
        return "待办";
    }
  }, [view]);

  const filtered = useMemo(() => {
    if (!search.trim()) return items;
    const kw = search.trim().toLowerCase();
    return items.filter(
      (it) =>
        it.title.toLowerCase().includes(kw) ||
        (it.tags ?? []).some((t) => t.toLowerCase().includes(kw))
    );
  }, [items, search]);

  const submit = () => {
    if (!newTitle.trim()) return;
    void createItem({
      listId: currentListId ?? "",
      title: newTitle.trim(),
      priority: 1,
    });
    setNewTitle("");
  };

  const runAi = async () => {
    if (!aiTitle.trim()) return;
    setAiBusy(true);
    const subs = await aiBreakdown(aiTitle.trim(), aiNotes);
    setAiBusy(false);
    setAiOpen(false);
    if (subs.length > 0) {
      for (const s of subs) {
        await createItem({
          listId: currentListId ?? "",
          title: s.title,
          estPomodoros: s.estPomodoros,
          priority: 2,
        });
      }
    }
    setAiTitle("");
    setAiNotes("");
  };

  return (
    <div className="flex h-full w-full min-w-0 flex-col">
      {/* 头部 */}
      <div className="shrink-0 border-b border-border bg-card px-4 py-3">
        <div className="flex items-center justify-between gap-3">
          <h1 className="text-sm font-semibold text-foreground">{title}</h1>
          <div className="flex items-center gap-2">
            <div className="relative">
              <Search
                size={12}
                className="absolute left-2 top-1/2 -translate-y-1/2 text-muted-foreground"
              />
              <Input
                value={search}
                onChange={(e) => setSearch(e.target.value)}
                placeholder="搜索…"
                className="h-7 w-40 pl-7 text-[11px]"
              />
            </div>
            <Button
              variant="outline"
              size="sm"
              className="h-7"
              onClick={() => setAiOpen((v) => !v)}
            >
              <Sparkles size={12} className="mr-1 text-primary" />
              AI 拆解
            </Button>
          </div>
        </div>

        {/* AI 拆解面板 */}
        {aiOpen && (
          <div className="mt-2 rounded-md border border-border bg-background p-2.5">
            <div className="text-[10px] font-semibold uppercase tracking-wider text-muted-foreground/70">
              把大任务拆成小步骤
            </div>
            <div className="mt-1.5 flex gap-2">
              <Input
                value={aiTitle}
                onChange={(e) => setAiTitle(e.target.value)}
                placeholder="任务标题，如：完成毕业论文…"
                className="h-7 flex-1 text-[11px]"
              />
              <Button
                size="sm"
                className="h-7"
                disabled={aiBusy || !aiTitle.trim()}
                onClick={() => void runAi()}
              >
                {aiBusy ? (
                  <Loader2 size={12} className="animate-spin" />
                ) : (
                  <Sparkles size={12} />
                )}
                拆解
              </Button>
            </div>
            <Textarea
              value={aiNotes}
              onChange={(e) => setAiNotes(e.target.value)}
              placeholder="补充说明（可选）…"
              rows={2}
              className="mt-1.5 resize-none text-[11px]"
            />
          </div>
        )}

        {/* 新增条目（回收站视图不显示） */}
        {view !== "trash" && (
          <div className="mt-2 flex gap-2">
            <Input
              value={newTitle}
              onChange={(e) => setNewTitle(e.target.value)}
              onKeyDown={(e) => e.key === "Enter" && submit()}
              placeholder="添加待办，回车确认…"
              className="h-8 flex-1 text-[12px]"
            />
            <Button size="sm" className="h-8" disabled={!newTitle.trim()} onClick={submit}>
              <Plus size={13} className="mr-1" />
              添加
            </Button>
          </div>
        )}
      </div>

      {/* 列表 */}
      <div className="min-h-0 flex-1 overflow-y-auto scrollbar-dark">
        {loading ? (
          <div className="flex h-full items-center justify-center">
            <Loader2 size={20} className="animate-spin text-muted-foreground" />
          </div>
        ) : filtered.length === 0 ? (
          <div className="flex h-full flex-col items-center justify-center gap-3 px-6 text-center">
            <div className="flex h-12 w-12 items-center justify-center rounded-full bg-muted text-muted-foreground">
              <ListTodo size={22} />
            </div>
            <div className="space-y-1">
              <div className="text-sm font-medium text-foreground">
                {view === "trash" ? "回收站是空的" : "暂无待办"}
              </div>
              <div className="text-xs text-muted-foreground">
                {view === "trash" ? "删除的待办会出现在这里" : "在上方输入添加一条"}
              </div>
            </div>
          </div>
        ) : (
          <div className="space-y-0.5 px-3 py-2">
            {filtered.map((it) => (
              <ItemRow
                key={it.id}
                item={it}
                inTrash={view === "trash"}
                onToggle={() => void toggleItem(it.id)}
                onDelete={() => void deleteItem(it.id)}
                onRestore={() => void restoreItem(it.id)}
                onPurge={() => void purgeItem(it.id)}
              />
            ))}
          </div>
        )}
      </div>

      {/* 底部状态 */}
      <div className="shrink-0 border-t border-border px-4 py-1.5">
        {toast ? (
          <span className="text-[11px] text-emerald-500">{toast}</span>
        ) : error ? (
          <span className="text-[11px] text-destructive">{error}</span>
        ) : (
          <span className="text-[11px] text-muted-foreground/60">
            {filtered.length} 项 · 数据存储于本地 SQLite
          </span>
        )}
      </div>

      {/* 回收站批量操作 */}
      {view === "trash" && (
        <div className="flex shrink-0 items-center gap-2 border-t border-border px-4 py-2">
          <Button size="sm" variant="outline" className="h-7" onClick={() => {
            useTodoStore.getState().lists.filter(l => l.isDeleted).forEach(l => void restoreList(l.id));
          }}>
            <RotateCcw size={12} className="mr-1" />
            恢复所有列表
          </Button>
          <Button size="sm" variant="destructive" className="h-7" onClick={() => {
            useTodoStore.getState().lists.filter(l => l.isDeleted).forEach(l => void purgeList(l.id));
          }}>
            <Trash2 size={12} className="mr-1" />
            清空列表回收站
          </Button>
        </div>
      )}
    </div>
  );
}

// —— 单条待办 ——
function ItemRow({
  item,
  inTrash,
  onToggle,
  onDelete,
  onRestore,
  onPurge,
}: {
  item: TodoItem;
  inTrash: boolean;
  onToggle: () => void;
  onDelete: () => void;
  onRestore: () => void;
  onPurge: () => void;
}) {
  const [expanded, setExpanded] = useState(false);
  const subs = useTodoStore((s) => s.items);

  const children = useMemo(
    () => subs.filter((x) => x.parentId === item.id),
    [subs, item.id]
  );

  const priorityColor =
    item.priority >= 3
      ? "text-red-500"
      : item.priority === 2
        ? "text-amber-500"
        : "text-muted-foreground";

  return (
    <div className="group rounded-md border border-transparent px-2 py-1 hover:border-border hover:bg-accent/30">
      <div className="flex items-center gap-2">
        <button
          onClick={onToggle}
          className="shrink-0 text-muted-foreground hover:text-primary"
          title={item.completedAt ? "标记未完成" : "标记完成"}
        >
          {item.completedAt ? (
            <CheckCircle2 size={16} className="text-primary" />
          ) : (
            <Circle size={16} />
          )}
        </button>

        <button
          onClick={() => setExpanded((v) => !v)}
          className="shrink-0 text-muted-foreground"
        >
          {children.length > 0 ? (
            expanded ? (
              <ChevronDown size={13} />
            ) : (
              <ChevronRight size={13} />
            )
          ) : null}
        </button>

        <span
          className={cn(
            "min-w-0 flex-1 truncate text-[12.5px]",
            item.completedAt ? "text-muted-foreground line-through" : "text-foreground"
          )}
        >
          {item.title}
        </span>

        {item.estPomodoros > 0 && (
          <span className="shrink-0 rounded bg-muted px-1.5 py-0.5 text-[10px] text-muted-foreground">
            🍅 {item.donePomodoros}/{item.estPomodoros}
          </span>
        )}

        {item.priority > 0 && (
          <Flag size={11} className={cn("shrink-0", priorityColor)} />
        )}

        <div className="hidden shrink-0 items-center gap-0.5 group-hover:flex">
          {!inTrash ? (
            <button
              className="rounded p-1 text-muted-foreground hover:text-destructive"
              title="删除"
              onClick={onDelete}
            >
              <Trash2 size={12} />
            </button>
          ) : (
            <>
              <button
                className="rounded p-1 text-muted-foreground hover:text-primary"
                title="恢复"
                onClick={onRestore}
              >
                <RotateCcw size={12} />
              </button>
              <button
                className="rounded p-1 text-muted-foreground hover:text-destructive"
                title="彻底删除"
                onClick={onPurge}
              >
                <Trash2 size={12} />
              </button>
            </>
          )}
        </div>
      </div>

      {/* 展开：标签 + 备注 + 子任务 */}
      {expanded && (
        <div className="ml-7 mt-1 space-y-1.5">
          {item.notes && (
            <p className="whitespace-pre-wrap text-[11px] text-muted-foreground">
              {item.notes}
            </p>
          )}
          {item.tags && item.tags.length > 0 && (
            <div className="flex flex-wrap items-center gap-1">
              {item.tags.map((t) => (
                <span
                  key={t}
                  className="flex items-center gap-0.5 rounded bg-muted px-1.5 py-0.5 text-[9px] text-muted-foreground"
                >
                  <Hash size={7} />
                  {t}
                </span>
              ))}
            </div>
          )}
          {item.dueAt && (
            <span className="text-[10px] text-muted-foreground">
              截止：{new Date(item.dueAt).toLocaleString()}
            </span>
          )}
          {children.map((c) => (
            <div key={c.id} className="flex items-center gap-2 pl-2">
              <span className="text-[11px] text-foreground/80">{c.title}</span>
              {c.completedAt && <Badge variant="secondary">完成</Badge>}
            </div>
          ))}
        </div>
      )}
    </div>
  );
}
