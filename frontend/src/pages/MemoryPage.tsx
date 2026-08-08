// MemoryPage —— 智能记忆
// ------------------------------------------------------------
// 三栏布局：
// 1. 左：操作面板（搜索框、用户画像按钮、隐私模式开关、衰减按钮）
// 2. 中：记忆列表（按 category 分组），每条显示 content / tags / weight / hit_count
// 3. 右：选中记忆详情 + 摄入对话面板
//
// 对接后端：MemoryIngest / MemorySearch / MemoryProfile
//           MemoryPrivacyMode / MemoryDecay

import { useMemo } from "react";
import {
  useMemoryStore,
  MEMORY_CATEGORIES,
  type MemoryItem,
} from "@/state/memory";
import { Button } from "@/components/ui/Button";
import { Input } from "@/components/ui/Input";
import { Textarea } from "@/components/ui/Textarea";
import { Switch } from "@/components/ui/Switch";
import { Badge } from "@/components/ui/Badge";
import { cn } from "@/lib/utils";
import {
  Brain,
  Search,
  UserRound,
  ShieldOff,
  TrendingDown,
  Loader2,
  AlertCircle,
  Inbox,
  MessageSquarePlus,
  Hash,
  CheckCircle2,
  Tag,
} from "lucide-react";

export function MemoryPage() {
  return (
    <div className="flex h-full w-full min-h-0 bg-background">
      {/* —— 左：操作面板 —— */}
      <aside className="w-60 shrink-0 border-r border-[var(--shell-seam)] bg-[var(--shell-inspector-panel)]">
        <OperationPanel />
      </aside>

      {/* —— 中：记忆列表 —— */}
      <section className="flex min-w-0 flex-1 flex-col">
        <MemoryList />
      </section>

      {/* —— 右：详情 + 摄入 —— */}
      <aside className="w-96 shrink-0 border-l border-[var(--shell-seam)] bg-[var(--shell-inspector-panel)]">
        <DetailPanel />
      </aside>
    </div>
  );
}

// —— 左：操作面板 ——
function OperationPanel() {
  const searchQuery = useMemoryStore((s) => s.searchQuery);
  const setSearchQuery = useMemoryStore((s) => s.setSearchQuery);
  const searching = useMemoryStore((s) => s.searching);
  const loading = useMemoryStore((s) => s.loading);
  const privacyMode = useMemoryStore((s) => s.privacyMode);
  const profileLoading = useMemoryStore((s) => s.profileLoading);
  const error = useMemoryStore((s) => s.error);
  const toast = useMemoryStore((s) => s.toast);
  const search = useMemoryStore((s) => s.search);
  const clearSearch = useMemoryStore((s) => s.clearSearch);
  const loadProfile = useMemoryStore((s) => s.loadProfile);
  const togglePrivacy = useMemoryStore((s) => s.togglePrivacy);
  const decay = useMemoryStore((s) => s.decay);

  return (
    <div className="flex h-full w-full flex-col">
      <div className="shrink-0 border-b border-[var(--shell-seam)] px-3 py-2.5">
        <div className="flex items-center gap-2">
          <div className="flex h-6 w-6 items-center justify-center rounded-md bg-primary/15 text-primary">
            <Brain size={13} />
          </div>
          <span className="text-[11px] font-semibold uppercase tracking-wider text-muted-foreground/70">
            记忆操作
          </span>
        </div>
      </div>

      <div className="min-h-0 flex-1 space-y-4 overflow-y-auto scrollbar-dark px-3 py-3">
        {/* 搜索 */}
        <div className="space-y-1.5">
          <div className="text-[10px] font-semibold uppercase tracking-wider text-muted-foreground/70">
            搜索记忆
          </div>
          <div className="relative">
            <Search
              size={12}
              className="absolute left-2 top-1/2 -translate-y-1/2 text-muted-foreground"
            />
            <Input
              value={searchQuery}
              onChange={(e) => setSearchQuery(e.target.value)}
              onKeyDown={(e) => {
                if (e.key === "Enter") void search();
              }}
              placeholder="关键词…"
              className="h-8 pl-7 text-[12px]"
            />
          </div>
          <div className="flex items-center gap-1">
            <Button
              size="sm"
              className="h-7 flex-1"
              disabled={loading || !searchQuery.trim()}
              onClick={() => void search()}
            >
              {loading ? (
                <Loader2 size={12} className="animate-spin" />
              ) : (
                <Search size={12} />
              )}
              搜索
            </Button>
            {searching && (
              <Button
                variant="ghost"
                size="sm"
                className="h-7"
                onClick={clearSearch}
              >
                清除
              </Button>
            )}
          </div>
        </div>

        {/* 分隔线 */}
        <div className="border-t border-[var(--shell-seam)]/60" />

        {/* 用户画像 */}
        <div className="space-y-1.5">
          <div className="text-[10px] font-semibold uppercase tracking-wider text-muted-foreground/70">
            用户画像
          </div>
          <Button
            variant="outline"
            size="sm"
            className="h-7 w-full"
            disabled={profileLoading}
            onClick={() => void loadProfile()}
          >
            {profileLoading ? (
              <Loader2 size={12} className="animate-spin" />
            ) : (
              <UserRound size={12} />
            )}
            {profileLoading ? "加载中…" : "查看画像"}
          </Button>
        </div>

        {/* 分隔线 */}
        <div className="border-t border-[var(--shell-seam)]/60" />

        {/* 隐私模式 */}
        <div className="space-y-1.5">
          <div className="flex items-center justify-between">
            <div className="flex items-center gap-1.5 text-[11px] font-semibold text-foreground">
              <ShieldOff size={11} className="text-primary" />
              隐私模式
            </div>
            <Switch checked={privacyMode} onCheckedChange={() => void togglePrivacy()} />
          </div>
          <p className="text-[10px] leading-relaxed text-muted-foreground">
            开启后阻止所有外部 API 调用
          </p>
        </div>

        {/* 分隔线 */}
        <div className="border-t border-[var(--shell-seam)]/60" />

        {/* 衰减 */}
        <div className="space-y-1.5">
          <div className="text-[10px] font-semibold uppercase tracking-wider text-muted-foreground/70">
            记忆衰减
          </div>
          <Button
            variant="outline"
            size="sm"
            className="h-7 w-full"
            disabled={loading}
            onClick={() => void decay()}
          >
            {loading ? (
              <Loader2 size={12} className="animate-spin" />
            ) : (
              <TrendingDown size={12} />
            )}
            执行衰减
          </Button>
          <p className="text-[10px] leading-relaxed text-muted-foreground">
            90 天未用降权，高频加权
          </p>
        </div>
      </div>

      {/* 底部状态 */}
      <div className="shrink-0 border-t border-[var(--shell-seam)] px-3 py-2">
        {toast ? (
          <div className="flex items-center gap-1.5 text-[10px] text-emerald-500">
            <CheckCircle2 size={10} />
            {toast}
          </div>
        ) : error ? (
          <div className="flex items-center gap-1.5 text-[10px] text-destructive">
            <AlertCircle size={10} />
            <span className="truncate">{error}</span>
          </div>
        ) : (
          <div className="text-[10px] text-muted-foreground/60">
            {privacyMode ? "🔒 隐私模式" : "🟢 正常"}
          </div>
        )}
      </div>
    </div>
  );
}

// —— 中：记忆列表 ——
function MemoryList() {
  const items = useMemoryStore((s) => s.items);
  const searching = useMemoryStore((s) => s.searching);
  const loading = useMemoryStore((s) => s.loading);
  const selectedId = useMemoryStore((s) => s.selectedId);
  const selectItem = useMemoryStore((s) => s.selectItem);
  const ingestedItems = useMemoryStore((s) => s.ingestedItems);

  // 合并搜索结果与新摄入的记忆
  const allItems = useMemo(() => {
    const map = new Map<string, MemoryItem>();
    // 先放搜索结果
    for (const it of items) map.set(it.id, it);
    // 再放摄入的新记忆（不覆盖）
    for (const it of ingestedItems) {
      if (!map.has(it.id)) map.set(it.id, it);
    }
    return Array.from(map.values());
  }, [items, ingestedItems]);

  // 按 category 分组
  const grouped = useMemo(() => {
    const m: Record<string, MemoryItem[]> = {};
    for (const it of allItems) {
      const cat = it.category || "other";
      if (!m[cat]) m[cat] = [];
      m[cat].push(it);
    }
    return m;
  }, [allItems]);

  return (
    <div className="flex h-full w-full min-w-0 flex-col">
      {/* 头部 */}
      <div className="shrink-0 border-b border-[var(--shell-seam)] bg-[var(--shell-inspector-panel)] px-4 py-3">
        <div className="flex items-center justify-between">
          <h1 className="text-sm font-semibold text-foreground">
            {searching ? "搜索结果" : "记忆库"}
          </h1>
          <span className="text-[10px] text-muted-foreground/70">
            共 {allItems.length} 条
          </span>
        </div>
      </div>

      {/* 列表 */}
      <div className="min-h-0 flex-1 overflow-y-auto scrollbar-dark">
        {loading ? (
          <div className="flex h-full items-center justify-center">
            <Loader2 size={20} className="animate-spin text-muted-foreground" />
          </div>
        ) : allItems.length === 0 ? (
          <div className="flex h-full flex-col items-center justify-center gap-3 px-6 text-center">
            <div className="flex h-12 w-12 items-center justify-center rounded-full bg-muted text-muted-foreground">
              <Inbox size={22} />
            </div>
            <div className="space-y-1">
              <div className="text-sm font-medium text-foreground">
                {searching ? "无匹配结果" : "暂无记忆"}
              </div>
              <div className="text-xs text-muted-foreground">
                {searching
                  ? "换个关键词试试"
                  : "在右侧粘贴对话文本进行摄入"}
              </div>
            </div>
          </div>
        ) : (
          <div className="space-y-4 px-4 py-3">
            {MEMORY_CATEGORIES.map((cat) => {
              const list = grouped[cat.key];
              if (!list || list.length === 0) return null;
              return (
                <CategoryGroup
                  key={cat.key}
                  categoryKey={cat.key}
                  categoryLabel={cat.label}
                  items={list}
                  selectedId={selectedId}
                  onSelect={selectItem}
                />
              );
            })}
            {/* 未分类的 category */}
            {Object.entries(grouped)
              .filter(
                ([key]) => !MEMORY_CATEGORIES.some((c) => c.key === key)
              )
              .map(([key, list]) => (
                <CategoryGroup
                  key={key}
                  categoryKey={key}
                  categoryLabel={key}
                  items={list}
                  selectedId={selectedId}
                  onSelect={selectItem}
                />
              ))}
          </div>
        )}
      </div>
    </div>
  );
}

// —— 分类组 ——
function CategoryGroup({
  categoryKey,
  categoryLabel,
  items,
  selectedId,
  onSelect,
}: {
  categoryKey: string;
  categoryLabel: string;
  items: MemoryItem[];
  selectedId: string | null;
  onSelect: (id: string | null) => void;
}) {
  return (
    <div>
      <div className="mb-1.5 flex items-center gap-2">
        <span className="text-[10px] font-semibold uppercase tracking-wider text-muted-foreground/70">
          {categoryLabel}
        </span>
        <span className="rounded-full bg-muted px-1.5 py-0.5 text-[9px] text-muted-foreground">
          {items.length}
        </span>
        <div className="flex-1 border-t border-[var(--shell-seam)]/40" />
      </div>
      <ul className="space-y-1">
        {items.map((item) => (
          <MemoryRow
            key={item.id}
            item={item}
            active={item.id === selectedId}
            onSelect={() => onSelect(item.id)}
          />
        ))}
      </ul>
    </div>
  );
}

// —— 单条记忆 ——
function MemoryRow({
  item,
  active,
  onSelect,
}: {
  item: MemoryItem;
  active: boolean;
  onSelect: () => void;
}) {
  return (
    <li
      onClick={onSelect}
      className={cn(
        "group cursor-pointer rounded-md border px-3 py-2 transition-colors",
        active
          ? "border-primary/40 bg-primary/10"
          : "border-transparent bg-background hover:bg-accent/40"
      )}
    >
      <div className="flex items-start gap-2">
        <div className="min-w-0 flex-1">
          <p className="text-[12px] leading-relaxed text-foreground">
            {item.content}
          </p>
          {/* 标签 */}
          {item.tags?.length > 0 && (
            <div className="mt-1 flex flex-wrap items-center gap-1">
              {item.tags.map((tag) => (
                <span
                  key={tag}
                  className="flex items-center gap-0.5 rounded bg-muted px-1 py-0.5 text-[9px] text-muted-foreground"
                >
                  <Hash size={7} />
                  {tag}
                </span>
              ))}
            </div>
          )}
        </div>
        {/* 权重与命中 */}
        <div className="shrink-0 space-y-0.5 text-right">
          <div className="flex items-center gap-1 text-[9px] text-muted-foreground">
            <span>权重</span>
            <span className="font-mono text-foreground/80">{item.weight}</span>
          </div>
          <div className="flex items-center gap-1 text-[9px] text-muted-foreground">
            <span>命中</span>
            <span className="font-mono text-foreground/80">
              {item.hit_count}
            </span>
          </div>
        </div>
      </div>
    </li>
  );
}

// —— 右：详情 + 摄入 ——
function DetailPanel() {
  const items = useMemoryStore((s) => s.items);
  const ingestedItems = useMemoryStore((s) => s.ingestedItems);
  const selectedId = useMemoryStore((s) => s.selectedId);
  const profile = useMemoryStore((s) => s.profile);

  // 合并查找选中项
  const allItems = useMemo(() => {
    const map = new Map<string, MemoryItem>();
    for (const it of items) map.set(it.id, it);
    for (const it of ingestedItems) if (!map.has(it.id)) map.set(it.id, it);
    return map;
  }, [items, ingestedItems]);

  const selected = selectedId ? allItems.get(selectedId) ?? null : null;

  return (
    <div className="flex h-full w-full flex-col">
      {/* 头部 */}
      <div className="shrink-0 border-b border-[var(--shell-seam)] px-3 py-2.5">
        <div className="text-[10px] font-semibold uppercase tracking-wider text-muted-foreground/70">
          详情与摄入
        </div>
      </div>

      <div className="min-h-0 flex-1 overflow-y-auto scrollbar-dark">
        {/* 选中记忆详情 */}
        {selected ? (
          <MemoryDetail item={selected} />
        ) : profile ? (
          <ProfileView profile={profile} />
        ) : (
          <div className="flex h-full flex-col items-center justify-center gap-3 px-6 text-center">
            <div className="flex h-10 w-10 items-center justify-center rounded-full bg-muted text-muted-foreground">
              <Brain size={18} />
            </div>
            <div className="space-y-1">
              <div className="text-xs font-medium text-foreground">
                未选择记忆
              </div>
              <div className="text-[11px] text-muted-foreground">
                从中间列表选择一项查看详情
              </div>
            </div>
          </div>
        )}
      </div>

      {/* 底部：摄入对话 */}
      <IngestPanel />
    </div>
  );
}

// —— 记忆详情 ——
function MemoryDetail({ item }: { item: MemoryItem }) {
  return (
    <div className="space-y-3 px-3 py-3 animate-fadeSlideUp">
      {/* 内容 */}
      <div>
        <div className="mb-1 text-[10px] font-semibold uppercase tracking-wider text-muted-foreground/70">
          内容
        </div>
        <p className="text-[12px] leading-relaxed text-foreground">
          {item.content}
        </p>
      </div>

      {/* 元信息 */}
      <div className="space-y-1">
        <div className="text-[10px] font-semibold uppercase tracking-wider text-muted-foreground/70">
          元信息
        </div>
        <dl className="space-y-1 text-[10px]">
          <div className="flex items-center gap-1.5">
            <dt className="text-muted-foreground/60">分类：</dt>
            <dd>
              <Badge variant="secondary">{item.category}</Badge>
            </dd>
          </div>
          <div className="flex items-center gap-1.5">
            <dt className="text-muted-foreground/60">权重：</dt>
            <dd className="font-mono text-foreground/80">{item.weight}</dd>
          </div>
          <div className="flex items-center gap-1.5">
            <dt className="text-muted-foreground/60">命中次数：</dt>
            <dd className="font-mono text-foreground/80">{item.hit_count}</dd>
          </div>
          <div className="flex items-center gap-1.5">
            <dt className="text-muted-foreground/60">来源：</dt>
            <dd className="truncate font-mono text-foreground/80">
              {item.source || "-"}
            </dd>
          </div>
          <div className="flex items-center gap-1.5">
            <dt className="text-muted-foreground/60">创建：</dt>
            <dd className="font-mono text-foreground/80">
              {formatDateTime(item.created_at)}
            </dd>
          </div>
          <div className="flex items-center gap-1.5">
            <dt className="text-muted-foreground/60">更新：</dt>
            <dd className="font-mono text-foreground/80">
              {formatDateTime(item.updated_at)}
            </dd>
          </div>
          <div className="flex items-center gap-1.5">
            <dt className="text-muted-foreground/60">最后命中：</dt>
            <dd className="font-mono text-foreground/80">
              {formatDateTime(item.last_hit)}
            </dd>
          </div>
        </dl>
      </div>

      {/* 标签 */}
      {item.tags?.length > 0 && (
        <div>
          <div className="mb-1 flex items-center gap-1 text-[10px] font-semibold uppercase tracking-wider text-muted-foreground/70">
            <Tag size={10} />
            标签
          </div>
          <div className="flex flex-wrap gap-1">
            {item.tags.map((tag) => (
              <span
                key={tag}
                className="flex items-center gap-0.5 rounded bg-muted px-1.5 py-0.5 text-[10px] text-muted-foreground"
              >
                <Hash size={8} />
                {tag}
              </span>
            ))}
          </div>
        </div>
      )}

      {/* 元数据 */}
      {item.metadata && Object.keys(item.metadata).length > 0 && (
        <div>
          <div className="mb-1 text-[10px] font-semibold uppercase tracking-wider text-muted-foreground/70">
            扩展元数据
          </div>
          <dl className="space-y-0.5 text-[10px]">
            {Object.entries(item.metadata).map(([k, v]) => (
              <div key={k} className="flex items-center gap-1.5">
                <dt className="text-muted-foreground/60">{k}：</dt>
                <dd className="font-mono text-foreground/80">{v}</dd>
              </div>
            ))}
          </dl>
        </div>
      )}
    </div>
  );
}

// —— 用户画像视图 ——
function ProfileView({ profile }: { profile: string }) {
  return (
    <div className="space-y-2 px-3 py-3 animate-fadeSlideUp">
      <div className="flex items-center gap-1.5 text-[10px] font-semibold uppercase tracking-wider text-muted-foreground/70">
        <UserRound size={11} className="text-primary" />
        用户画像
      </div>
      <pre className="whitespace-pre-wrap break-words rounded-md border border-[var(--shell-seam)] bg-background p-3 font-mono text-[11px] leading-relaxed text-foreground/90">
        {profile}
      </pre>
    </div>
  );
}

// —— 摄入对话面板 ——
function IngestPanel() {
  const ingestText = useMemoryStore((s) => s.ingestText);
  const setIngestText = useMemoryStore((s) => s.setIngestText);
  const loading = useMemoryStore((s) => s.loading);
  const ingest = useMemoryStore((s) => s.ingest);
  const ingestedItems = useMemoryStore((s) => s.ingestedItems);

  return (
    <div className="shrink-0 border-t border-[var(--shell-seam)] bg-background px-3 py-3">
      <div className="mb-2 flex items-center gap-1.5 text-[10px] font-semibold uppercase tracking-wider text-muted-foreground/70">
        <MessageSquarePlus size={11} className="text-primary" />
        摄入对话
      </div>
      <Textarea
        value={ingestText}
        onChange={(e) => setIngestText(e.target.value)}
        placeholder="粘贴对话文本，AI 将自动抽取记忆…"
        rows={4}
        className="resize-none text-[12px]"
      />
      <div className="mt-2 flex items-center justify-between">
        <span className="text-[10px] text-muted-foreground">
          {ingestedItems.length > 0
            ? `已摄入 ${ingestedItems.length} 条`
            : "从对话中抽取持久事实"}
        </span>
        <Button
          size="sm"
          className="h-7"
          disabled={loading || !ingestText.trim()}
          onClick={() => void ingest()}
        >
          {loading ? (
            <Loader2 size={12} className="animate-spin" />
          ) : (
            <MessageSquarePlus size={12} />
          )}
          {loading ? "摄入中…" : "摄入"}
        </Button>
      </div>
    </div>
  );
}

// —— 工具函数 ——
/** 格式化日期时间 */
function formatDateTime(ts: string): string {
  if (!ts) return "-";
  try {
    const d = new Date(ts);
    if (isNaN(d.getTime())) return ts;
    const y = d.getFullYear();
    const m = String(d.getMonth() + 1).padStart(2, "0");
    const day = String(d.getDate()).padStart(2, "0");
    const hh = String(d.getHours()).padStart(2, "0");
    const mm = String(d.getMinutes()).padStart(2, "0");
    return `${y}-${m}-${day} ${hh}:${mm}`;
  } catch {
    return ts;
  }
}

// 保留未使用图标引用
void AlertCircle;
