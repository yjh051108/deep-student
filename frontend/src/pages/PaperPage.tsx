// PaperPage —— 论文检索
// ------------------------------------------------------------
// 布局：
// 1. 顶部：搜索框 + 引擎切换（arXiv / OpenAlex）+ 结果数（5/10/20）
// 2. 结果列表：标题 / 作者 / 年份 / venue / source 标签
// 3. 展开后：摘要 + 操作按钮（下载 PDF / 生成引用 / 解析 DOI）
//
// 对接后端：PaperSearchArXiv / PaperSearchOpenAlex / PaperDownload
//           PaperCite / PaperResolveDOI

import { useState } from "react";
import {
  usePaperStore,
  type PaperSource,
  type CitationFormat,
} from "@/state/paper";
import { Button } from "@/components/ui/Button";
import { Input } from "@/components/ui/Input";
import { Badge } from "@/components/ui/Badge";
import { cn } from "@/lib/utils";
import {
  FileSearch,
  Search,
  Loader2,
  AlertCircle,
  Download,
  Quote,
  ExternalLink,
  ChevronDown,
  Inbox,
  CheckCircle2,
  Copy,
} from "lucide-react";

export function PaperPage() {
  const query = usePaperStore((s) => s.query);
  const setQuery = usePaperStore((s) => s.setQuery);
  const engine = usePaperStore((s) => s.engine);
  const setEngine = usePaperStore((s) => s.setEngine);
  const maxResults = usePaperStore((s) => s.maxResults);
  const setMaxResults = usePaperStore((s) => s.setMaxResults);
  const loading = usePaperStore((s) => s.loading);
  const error = usePaperStore((s) => s.error);
  const search = usePaperStore((s) => s.search);

  return (
    <div className="flex h-full w-full min-h-0 flex-col bg-background">
      {/* —— 顶部搜索栏 —— */}
      <div className="shrink-0 space-y-2 border-b border-border bg-card px-4 py-3">
        <div className="flex items-center gap-2">
          <div className="flex h-7 w-7 shrink-0 items-center justify-center rounded-md bg-primary/15 text-primary">
            <FileSearch size={15} />
          </div>
          <h1 className="shrink-0 text-sm font-semibold text-foreground">
            论文检索
          </h1>
        </div>
        <div className="flex items-center gap-2">
          {/* 引擎切换 */}
          <div className="flex items-center gap-0.5 rounded-md bg-muted p-0.5">
            <EngineToggle
              active={engine === "arxiv"}
              onClick={() => setEngine("arxiv")}
              label="arXiv"
            />
            <EngineToggle
              active={engine === "openalex"}
              onClick={() => setEngine("openalex")}
              label="OpenAlex"
            />
          </div>
          {/* 搜索框 */}
          <div className="relative flex-1">
            <Search
              size={14}
              className="absolute left-2.5 top-1/2 -translate-y-1/2 text-muted-foreground"
            />
            <Input
              value={query}
              onChange={(e) => setQuery(e.target.value)}
              onKeyDown={(e) => {
                if (e.key === "Enter") void search();
              }}
              placeholder="输入关键词…"
              className="h-8 pl-8 text-[13px]"
            />
          </div>
          {/* 结果数 */}
          <select
            value={maxResults}
            onChange={(e) => setMaxResults(Number(e.target.value))}
            className="h-8 rounded-md border border-input bg-transparent px-2 text-[13px] text-foreground focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring"
          >
            <option value={5}>5 条</option>
            <option value={10}>10 条</option>
            <option value={20}>20 条</option>
          </select>
          {/* 搜索按钮 */}
          <Button
            size="sm"
            className="h-8"
            disabled={loading || !query.trim()}
            onClick={() => void search()}
          >
            {loading ? (
              <Loader2 size={13} className="animate-spin" />
            ) : (
              <Search size={13} />
            )}
            {loading ? "搜索中…" : "搜索"}
          </Button>
        </div>
      </div>

      {/* —— 错误横幅 —— */}
      {error && (
        <div className="shrink-0 flex items-center gap-2 border-b border-destructive/30 bg-destructive/10 px-4 py-2 text-xs text-destructive">
          <AlertCircle size={12} />
          <span className="truncate">{error}</span>
        </div>
      )}

      {/* —— 结果列表 —— */}
      <div className="min-h-0 flex-1 overflow-y-auto scrollbar-dark">
        {loading ? (
          <CenteredLoading label="检索中…" />
        ) : (
          <ResultList />
        )}
      </div>
    </div>
  );
}

// —— 结果列表 ——
function ResultList() {
  const results = usePaperStore((s) => s.results);
  if (results.length === 0) {
    return (
      <div className="flex h-full flex-col items-center justify-center gap-3 px-6 text-center">
        <div className="flex h-12 w-12 items-center justify-center rounded-full bg-muted text-muted-foreground">
          <Inbox size={22} />
        </div>
        <div className="space-y-1">
          <div className="text-sm font-medium text-foreground">暂无结果</div>
          <div className="text-xs text-muted-foreground">
            在顶部输入关键词后点击"搜索"
          </div>
        </div>
      </div>
    );
  }
  return (
    <ul className="divide-y divide-border/60">
      {results.map((src, i) => (
        <PaperRow key={src.id || i} src={src} />
      ))}
    </ul>
  );
}

// —— 单条论文 ——
function PaperRow({ src }: { src: PaperSource }) {
  const expandedId = usePaperStore((s) => s.expandedId);
  const toggleExpand = usePaperStore((s) => s.toggleExpand);
  const expanded = expandedId === src.id;

  return (
    <li className="animate-fade-in">
      <button
        type="button"
        onClick={() => toggleExpand(src.id)}
        className="flex w-full items-start gap-3 px-4 py-3 text-left transition-colors hover:bg-accent/40"
      >
        {/* 展开箭头 */}
        <ChevronDown
          size={14}
          className={cn(
            "mt-0.5 shrink-0 text-muted-foreground transition-transform",
            expanded && "rotate-180"
          )}
        />
        <div className="min-w-0 flex-1">
          {/* 标题行 */}
          <h3 className="text-sm font-medium leading-snug text-foreground">
            {src.title || "(无标题)"}
          </h3>
          {/* 元信息行 */}
          <div className="mt-1 flex flex-wrap items-center gap-2 text-[11px] text-muted-foreground">
            {src.authors?.length > 0 && (
              <span className="truncate">
                {src.authors.slice(0, 3).join(", ")}
                {src.authors.length > 3 && " 等"}
              </span>
            )}
            {src.year > 0 && <span>· {src.year}</span>}
            {src.venue && (
              <span className="truncate">· {src.venue}</span>
            )}
            <SourceBadge source={src.source} />
          </div>
        </div>
      </button>
      {/* 展开内容 */}
      {expanded && <PaperDetail src={src} />}
    </li>
  );
}

// —— 展开详情 ——
function PaperDetail({ src }: { src: PaperSource }) {
  const downloadUris = usePaperStore((s) => s.downloadUris);
  const citations = usePaperStore((s) => s.citations);
  const citationFormat = usePaperStore((s) => s.citationFormat);
  const setCitationFormat = usePaperStore((s) => s.setCitationFormat);
  const busyId = usePaperStore((s) => s.busyId);
  const download = usePaperStore((s) => s.download);
  const cite = usePaperStore((s) => s.cite);
  const resolveDOI = usePaperStore((s) => s.resolveDOI);
  const [doiInput, setDoiInput] = useState("");

  const cacheKey = `${src.id}:${citationFormat}`;
  const citationText = citations[cacheKey];
  const downloadUri = downloadUris[src.id];
  const isBusy = busyId === src.id;

  return (
    <div className="space-y-3 border-t border-border/40 bg-background px-4 py-3 animate-fadeSlideUp">
      {/* 摘要 */}
      {src.abstract ? (
        <div>
          <div className="mb-1 text-[10px] font-semibold uppercase tracking-wider text-muted-foreground/70">
            摘要
          </div>
          <p className="text-[12px] leading-relaxed text-foreground/85">
            {src.abstract}
          </p>
        </div>
      ) : (
        <div className="text-[11px] text-muted-foreground">无摘要</div>
      )}

      {/* 链接 */}
      <div className="flex flex-wrap items-center gap-2 text-[10px] text-muted-foreground">
        {src.url && (
          <a
            href={src.url}
            target="_blank"
            rel="noopener noreferrer"
            className="flex items-center gap-1 rounded border border-border px-1.5 py-0.5 hover:bg-accent"
          >
            <ExternalLink size={9} />
            原文链接
          </a>
        )}
        {src.doi && (
          <span className="rounded border border-border px-1.5 py-0.5 font-mono">
            DOI: {src.doi}
          </span>
        )}
      </div>

      {/* 下载区 */}
      <div className="space-y-1.5">
        <div className="flex items-center gap-2">
          <Button
            variant="outline"
            size="sm"
            className="h-7"
            disabled={isBusy || !src.pdf_url}
            onClick={() => void download(src)}
            title={!src.pdf_url ? "无 PDF 链接" : "下载 PDF"}
          >
            {isBusy ? (
              <Loader2 size={12} className="animate-spin" />
            ) : (
              <Download size={12} />
            )}
            下载 PDF
          </Button>
          {downloadUri && (
            <div className="flex items-center gap-1.5 text-[11px] text-emerald-500">
              <CheckCircle2 size={12} />
              <span className="truncate font-mono">{downloadUri}</span>
              <button
                type="button"
                onClick={() => void navigator.clipboard.writeText(downloadUri)}
                className="rounded p-0.5 hover:bg-accent"
                title="复制 URI"
              >
                <Copy size={10} />
              </button>
            </div>
          )}
        </div>
      </div>

      {/* 引用区 */}
      <div className="space-y-1.5">
        <div className="flex items-center gap-2">
          {/* 引用格式选择 */}
          <select
            value={citationFormat}
            onChange={(e) =>
              setCitationFormat(e.target.value as CitationFormat)
            }
            className="h-7 rounded-md border border-input bg-transparent px-2 text-[11px] text-foreground focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring"
          >
            <option value="bibtex">BibTeX</option>
            <option value="gbt7714">GB/T 7714</option>
            <option value="apa">APA</option>
          </select>
          <Button
            variant="outline"
            size="sm"
            className="h-7"
            disabled={isBusy}
            onClick={() => void cite(src)}
          >
            {isBusy ? (
              <Loader2 size={12} className="animate-spin" />
            ) : (
              <Quote size={12} />
            )}
            生成引用
          </Button>
          {citationText && (
            <button
              type="button"
              onClick={() => void navigator.clipboard.writeText(citationText)}
              className="flex items-center gap-1 rounded p-1 text-[10px] text-muted-foreground hover:bg-accent hover:text-foreground"
              title="复制引用"
            >
              <Copy size={10} />
              复制
            </button>
          )}
        </div>
        {citationText && (
          <pre className="whitespace-pre-wrap break-words rounded-md border border-border bg-muted/40 p-2.5 font-mono text-[11px] leading-relaxed text-foreground/90">
            {citationText}
          </pre>
        )}
      </div>

      {/* DOI 解析 */}
      <div className="space-y-1.5 border-t border-border/40 pt-2">
        <div className="text-[10px] font-semibold uppercase tracking-wider text-muted-foreground/70">
          DOI 解析
        </div>
        <div className="flex items-center gap-2">
          <Input
            value={doiInput}
            onChange={(e) => setDoiInput(e.target.value)}
            placeholder="输入 DOI，例如 10.1000/xyz123"
            className="h-7 text-[12px]"
          />
          <Button
            variant="ghost"
            size="sm"
            className="h-7"
            disabled={busyId === "doi" || !doiInput.trim()}
            onClick={() => void resolveDOI(doiInput)}
          >
            {busyId === "doi" ? (
              <Loader2 size={12} className="animate-spin" />
            ) : (
              <ExternalLink size={12} />
            )}
            解析
          </Button>
        </div>
      </div>
    </div>
  );
}

// —— 引擎切换按钮 ——
function EngineToggle({
  active,
  onClick,
  label,
}: {
  active: boolean;
  onClick: () => void;
  label: string;
}) {
  return (
    <button
      type="button"
      onClick={onClick}
      className={cn(
        "rounded px-2.5 py-1 text-[11px] font-medium transition-colors",
        active
          ? "bg-background text-foreground shadow-sm"
          : "text-muted-foreground hover:text-foreground"
      )}
    >
      {label}
    </button>
  );
}

// —— 来源标签 ——
function SourceBadge({ source }: { source: string }) {
  const variant = source === "arxiv" ? "info" : "warning";
  const label = source === "arxiv" ? "arXiv" : "OpenAlex";
  return <Badge variant={variant as "info"}>{label}</Badge>;
}

// —— 居中加载 ——
function CenteredLoading({ label }: { label: string }) {
  return (
    <div className="flex h-full flex-col items-center justify-center gap-3 text-center">
      <div className="flex h-12 w-12 items-center justify-center rounded-full bg-primary/10 text-primary">
        <Loader2 size={22} className="animate-spin" />
      </div>
      <div className="text-sm font-medium text-foreground">{label}</div>
    </div>
  );
}


