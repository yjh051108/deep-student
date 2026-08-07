// ResourcePreview —— 右侧资源预览面板
// ------------------------------------------------------------
// 显示选中资源的元数据 + 内容预览
// 笔记类型额外提供 AI 续写功能

import { useHubStore, type VFSEntry } from "@/state/hub";
import { Button } from "@/components/ui/Button";
import { Textarea } from "@/components/ui/Textarea";
import { cn, formatTime, formatDate } from "@/lib/utils";
import {
  FileText,
  Loader2,
  Sparkles,
  Copy,
  Trash2,
  RefreshCw,
  AlertCircle,
} from "lucide-react";
import { useState } from "react";

export function ResourcePreview() {
  const activeUri = useHubStore((s) => s.activeUri);
  const entries = useHubStore((s) => s.entries);
  const previewContent = useHubStore((s) => s.previewContent);
  const previewLoading = useHubStore((s) => s.previewLoading);
  const continuation = useHubStore((s) => s.continuation);
  const continuing = useHubStore((s) => s.continuing);
  const continueNote = useHubStore((s) => s.continueNote);
  const clearContinuation = useHubStore((s) => s.clearContinuation);

  const [prompt, setPrompt] = useState("");

  const entry = activeUri
    ? entries.find((e) => e.uri === activeUri) ?? null
    : null;

  if (!entry) {
    return <EmptyPreview />;
  }

  const isNote = entry.type === "note";

  return (
    <div className="flex h-full w-full flex-col">
      {/* 头部：元数据 */}
      <div className="shrink-0 space-y-2 border-b border-border bg-card px-4 py-3">
        <div className="flex items-start gap-2">
          <div className="flex h-8 w-8 shrink-0 items-center justify-center rounded-md bg-primary/15 text-primary">
            <FileText size={16} />
          </div>
          <div className="min-w-0 flex-1">
            <h2
              className="truncate text-sm font-semibold text-foreground"
              title={entry.title}
            >
              {entry.title || entry.id}
            </h2>
            <div className="mt-0.5 truncate font-mono text-[10px] text-muted-foreground/70">
              {entry.uri}
            </div>
          </div>
        </div>
        <MetaGrid entry={entry} />
      </div>

      {/* 内容预览 */}
      <div className="min-h-0 flex-1 overflow-y-auto scrollbar-dark">
        <div className="space-y-3 px-4 py-3">
          <div className="flex items-center justify-between">
            <div className="text-[10px] font-semibold uppercase tracking-wider text-muted-foreground/70">
              内容预览
            </div>
            <div className="flex items-center gap-1">
              <Button
                variant="ghost"
                size="icon"
                className="h-6 w-6"
                title="复制内容"
                onClick={() => void navigator.clipboard.writeText(previewContent)}
                disabled={!previewContent || previewLoading}
              >
                <Copy size={11} />
              </Button>
            </div>
          </div>
          {previewLoading ? (
            <div className="flex items-center gap-2 py-8 text-xs text-muted-foreground">
              <Loader2 size={14} className="animate-spin" />
              加载中…
            </div>
          ) : previewContent ? (
            <pre className="whitespace-pre-wrap break-words rounded-md border border-border bg-background p-3 font-mono text-[11px] leading-relaxed text-foreground/90">
              {previewContent}
            </pre>
          ) : (
            <div className="py-8 text-center text-xs text-muted-foreground">
              无内容
            </div>
          )}

          {/* AI 续写区（仅笔记类型） */}
          {isNote && (
            <div className="space-y-2 border-t border-border pt-3">
              <div className="flex items-center gap-1.5 text-[10px] font-semibold uppercase tracking-wider text-muted-foreground/70">
                <Sparkles size={11} className="text-primary" />
                AI 续写
              </div>
              <Textarea
                value={prompt}
                onChange={(e) => setPrompt(e.target.value)}
                placeholder="告诉 AI 接下来写什么…  例如：续写关于贝叶斯推断的下一节"
                rows={2}
                className="resize-none text-[12px]"
              />
              <div className="flex items-center gap-2">
                <Button
                  size="sm"
                  className="h-7"
                  disabled={!prompt.trim() || continuing}
                  onClick={() => void continueNote(entry.uri, prompt.trim())}
                >
                  {continuing ? (
                    <Loader2 size={12} className="animate-spin" />
                  ) : (
                    <Sparkles size={12} />
                  )}
                  {continuing ? "生成中…" : "续写"}
                </Button>
                {continuation && (
                  <Button
                    variant="ghost"
                    size="sm"
                    className="h-7"
                    onClick={() => {
                      clearContinuation();
                      setPrompt("");
                    }}
                  >
                    清除
                  </Button>
                )}
              </div>
              {continuation && (
                <div className="rounded-md border border-primary/30 bg-primary/5 p-3">
                  <div className="mb-1 flex items-center gap-1 text-[10px] font-semibold uppercase tracking-wider text-primary">
                    <Sparkles size={10} />
                    续写结果
                  </div>
                  <pre className="whitespace-pre-wrap break-words font-mono text-[11px] leading-relaxed text-foreground/90">
                    {continuation}
                  </pre>
                </div>
              )}
            </div>
          )}
        </div>
      </div>
    </div>
  );
}

function MetaGrid({ entry }: { entry: VFSEntry }) {
  return (
    <dl className="grid grid-cols-2 gap-x-3 gap-y-1 text-[10px]">
      <MetaItem label="类型" value={entry.type} />
      <MetaItem label="大小" value={formatBytes(entry.size)} />
      <MetaItem label="创建" value={formatDate(entry.created_at)} />
      <MetaItem label="更新" value={formatTime(entry.updated_at)} />
      {entry.tags?.length > 0 && (
        <div className="col-span-2 flex flex-wrap items-center gap-1">
          <dt className="text-muted-foreground/60">标签：</dt>
          {entry.tags.map((t) => (
            <dd
              key={t}
              className="rounded bg-muted px-1.5 py-0.5 font-mono text-[10px] text-muted-foreground"
            >
              {t}
            </dd>
          ))}
        </div>
      )}
      <MetaItem
        label="Blob"
        value={entry.blob_ref ? entry.blob_ref.slice(0, 12) + "…" : "-"}
        full
      />
    </dl>
  );
}

function MetaItem({
  label,
  value,
  full,
}: {
  label: string;
  value: string;
  full?: boolean;
}) {
  return (
    <div className={cn("flex items-center gap-1.5", full && "col-span-2")}>
      <dt className="text-muted-foreground/60">{label}：</dt>
      <dd className="truncate font-mono text-foreground/80" title={value}>
        {value}
      </dd>
    </div>
  );
}

function EmptyPreview() {
  return (
    <div className="flex h-full flex-col items-center justify-center gap-3 px-6 text-center">
      <div className="flex h-12 w-12 items-center justify-center rounded-full bg-muted text-muted-foreground">
        <FileText size={20} />
      </div>
      <div className="space-y-1">
        <div className="text-sm font-medium text-foreground">未选择资源</div>
        <div className="text-xs text-muted-foreground">
          从左侧列表选择一项以查看预览
        </div>
      </div>
    </div>
  );
}

function formatBytes(bytes: number): string {
  if (!bytes) return "0 B";
  const units = ["B", "KB", "MB", "GB"];
  const i = Math.min(units.length - 1, Math.floor(Math.log(bytes) / Math.log(1024)));
  return `${(bytes / Math.pow(1024, i)).toFixed(i === 0 ? 0 : 1)} ${units[i]}`;
}

// 保留未使用图标的引用以备未来扩展
void RefreshCw;
void AlertCircle;
void Trash2;
