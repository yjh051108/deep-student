// ResourceList —— 中间资源列表
// ------------------------------------------------------------
// 顶部：搜索框 + 标签过滤 + 导入按钮
// 列表：标题 / 标签 / 大小 / 更新时间
// 行操作：选中 / 删除

import { useHubStore, type VFSEntry } from "@/state/hub";
import { Button } from "@/components/ui/Button";
import { Input } from "@/components/ui/Input";
import { cn, formatTime, relativeTime, truncate } from "@/lib/utils";
import {
  Search,
  Plus,
  Trash2,
  FileText,
  RefreshCw,
  AlertCircle,
  Inbox,
} from "lucide-react";
import { useMemo, useState } from "react";

interface ResourceListProps {
  onOpenImport: () => void;
}

export function ResourceList({ onOpenImport }: ResourceListProps) {
  const entries = useHubStore((s) => s.entries);
  const activeUri = useHubStore((s) => s.activeUri);
  const activeType = useHubStore((s) => s.activeType);
  const keyword = useHubStore((s) => s.keyword);
  const tagFilter = useHubStore((s) => s.tagFilter);
  const loading = useHubStore((s) => s.loading);
  const error = useHubStore((s) => s.error);
  const setKeyword = useHubStore((s) => s.setKeyword);
  const setTagFilter = useHubStore((s) => s.setTagFilter);
  const selectResource = useHubStore((s) => s.selectResource);
  const removeResource = useHubStore((s) => s.removeResource);
  const refresh = useHubStore((s) => s.refresh);
  const [confirmingUri, setConfirmingUri] = useState<string | null>(null);

  // 本地关键字过滤（后端 HubSearch 走 tag，关键字过滤在前端做）
  const filtered = useMemo(() => {
    if (!keyword.trim()) return entries;
    const kw = keyword.toLowerCase();
    return entries.filter(
      (e) =>
        e.title?.toLowerCase().includes(kw) ||
        e.tags?.some((t) => t.toLowerCase().includes(kw))
    );
  }, [entries, keyword]);

  return (
    <div className="flex h-full w-full min-w-0 flex-col">
      {/* 顶部工具条 */}
      <div className="shrink-0 space-y-2 border-b border-border bg-card px-4 py-3">
        <div className="flex items-center gap-2">
          <div className="relative flex-1">
            <Search
              size={14}
              className="absolute left-2.5 top-1/2 -translate-y-1/2 text-muted-foreground"
            />
            <Input
              value={keyword}
              onChange={(e) => setKeyword(e.target.value)}
              placeholder={`搜索 ${activeType || "全部"} 资源标题或标签…`}
              className="h-8 pl-8 text-[13px]"
            />
          </div>
          <Button
            variant="outline"
            size="icon"
            className="h-8 w-8"
            onClick={() => void refresh()}
            disabled={loading}
            title="刷新"
          >
            <RefreshCw size={13} className={cn(loading && "animate-spin")} />
          </Button>
          <Button
            size="sm"
            className="h-8"
            onClick={onOpenImport}
            title="导入新资源"
          >
            <Plus size={13} />
            导入
          </Button>
        </div>
        {/* 标签过滤 */}
        <div className="flex items-center gap-1.5">
          <span className="text-[10px] text-muted-foreground/70">标签：</span>
          <TagChip
            label="全部"
            active={tagFilter === ""}
            onClick={() => setTagFilter("")}
          />
          {collectTags(entries).map((tag) => (
            <TagChip
              key={tag}
              label={tag}
              active={tagFilter === tag}
              onClick={() => setTagFilter(tag)}
            />
          ))}
        </div>
      </div>

      {/* 错误横幅 */}
      {error && (
        <div className="shrink-0 flex items-center gap-2 border-b border-destructive/30 bg-destructive/10 px-4 py-2 text-xs text-destructive">
          <AlertCircle size={12} />
          <span className="truncate">{error}</span>
        </div>
      )}

      {/* 列表 */}
      <div className="min-h-0 flex-1 overflow-y-auto scrollbar-dark">
        {filtered.length === 0 ? (
          <EmptyState loading={loading} />
        ) : (
          <ul className="divide-y divide-border/60">
            {filtered.map((entry) => (
              <ResourceRow
                key={entry.uri}
                entry={entry}
                active={entry.uri === activeUri}
                confirming={entry.uri === confirmingUri}
                onSelect={() => selectResource(entry.uri)}
                onAskRemove={() => setConfirmingUri(entry.uri)}
                onCancelRemove={() => setConfirmingUri(null)}
                onConfirmRemove={async () => {
                  setConfirmingUri(null);
                  await removeResource(entry.uri);
                }}
              />
            ))}
          </ul>
        )}
      </div>

      {/* 底部统计 */}
      <div className="shrink-0 border-t border-border bg-card px-4 py-1.5 text-[10px] text-muted-foreground/70">
        共 {filtered.length} 条 / 总计 {entries.length} 条
      </div>
    </div>
  );
}

function TagChip({
  label,
  active,
  onClick,
}: {
  label: string;
  active: boolean;
  onClick: () => void;
}) {
  return (
    <button
      type="button"
      onClick={onClick}
      className={cn(
        "rounded-full border px-2 py-0.5 text-[10px] font-medium transition-colors",
        active
          ? "border-primary/40 bg-primary/15 text-primary"
          : "border-border bg-transparent text-muted-foreground hover:bg-accent hover:text-foreground"
      )}
    >
      {label}
    </button>
  );
}

interface ResourceRowProps {
  entry: VFSEntry;
  active: boolean;
  confirming: boolean;
  onSelect: () => void;
  onAskRemove: () => void;
  onCancelRemove: () => void;
  onConfirmRemove: () => void;
}

function ResourceRow({
  entry,
  active,
  confirming,
  onSelect,
  onAskRemove,
  onCancelRemove,
  onConfirmRemove,
}: ResourceRowProps) {
  return (
    <li
      onClick={onSelect}
      className={cn(
        "group relative flex cursor-pointer items-start gap-3 px-4 py-2.5 transition-colors",
        active ? "bg-primary/10" : "hover:bg-accent/50"
      )}
    >
      {active && (
        <span className="absolute left-0 top-1/2 h-6 w-0.5 -translate-y-1/2 rounded-r-full bg-primary" />
      )}
      <div
        className={cn(
          "flex h-7 w-7 shrink-0 items-center justify-center rounded-md",
          active
            ? "bg-primary/20 text-primary"
            : "bg-muted text-muted-foreground"
        )}
      >
        <FileText size={14} />
      </div>
      <div className="min-w-0 flex-1">
        <div className="flex items-center justify-between gap-2">
          <div
            className={cn(
              "truncate text-sm font-medium",
              active ? "text-foreground" : "text-foreground/90"
            )}
            title={entry.title}
          >
            {entry.title || entry.id}
          </div>
          <span className="shrink-0 text-[10px] text-muted-foreground/60">
            {relativeTime(entry.updated_at || entry.created_at)}
          </span>
        </div>
        <div className="mt-0.5 flex items-center gap-2 text-[10px] text-muted-foreground/70">
          <span className="rounded bg-muted px-1 py-0.5 font-mono">
            {entry.type}
          </span>
          <span>{formatBytes(entry.size)}</span>
          <span>·</span>
          <span>{formatTime(entry.updated_at || entry.created_at)}</span>
          {entry.tags?.length > 0 && (
            <span className="truncate text-muted-foreground/60">
              # {entry.tags.join(" # ")}
            </span>
          )}
        </div>
      </div>
      <div className="shrink-0" onClick={(e) => e.stopPropagation()}>
        {confirming ? (
          <div className="flex items-center gap-1">
            <Button
              variant="destructive"
              size="sm"
              className="h-6 px-2 text-[10px]"
              onClick={onConfirmRemove}
            >
              确认
            </Button>
            <Button
              variant="ghost"
              size="sm"
              className="h-6 px-2 text-[10px]"
              onClick={onCancelRemove}
            >
              取消
            </Button>
          </div>
        ) : (
          <button
            type="button"
            onClick={onAskRemove}
            title="删除"
            className="rounded p-1 text-muted-foreground/60 opacity-0 transition-opacity hover:bg-destructive/15 hover:text-destructive group-hover:opacity-100"
          >
            <Trash2 size={12} />
          </button>
        )}
      </div>
    </li>
  );
}

function EmptyState({ loading }: { loading: boolean }) {
  return (
    <div className="flex h-full flex-col items-center justify-center gap-3 px-6 text-center">
      <div className="flex h-12 w-12 items-center justify-center rounded-full bg-muted text-muted-foreground">
        {loading ? (
          <RefreshCw size={20} className="animate-spin" />
        ) : (
          <Inbox size={20} />
        )}
      </div>
      <div className="space-y-1">
        <div className="text-sm font-medium text-foreground">
          {loading ? "加载中…" : "暂无资源"}
        </div>
        <div className="text-xs text-muted-foreground">
          {loading
            ? "正在从后端拉取资源列表"
            : "点击右上角“导入”按钮添加新资源"}
        </div>
      </div>
    </div>
  );
}

/** 收集列表中所有 tag（去重） */
function collectTags(entries: VFSEntry[]): string[] {
  const set = new Set<string>();
  for (const e of entries) {
    e.tags?.forEach((t) => set.add(t));
  }
  return Array.from(set).slice(0, 12);
}

/** 格式化字节数 */
function formatBytes(bytes: number): string {
  if (!bytes) return "0 B";
  const units = ["B", "KB", "MB", "GB"];
  const i = Math.min(units.length - 1, Math.floor(Math.log(bytes) / Math.log(1024)));
  return `${(bytes / Math.pow(1024, i)).toFixed(i === 0 ? 0 : 1)} ${units[i]}`;
}

// 保留 truncate 引用以备未来扩展
void truncate;
