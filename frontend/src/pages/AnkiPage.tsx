// AnkiPage —— Anki 卡片工厂
// ------------------------------------------------------------
// 布局：
// 1. 左侧：模板列表 + 添加模板按钮
// 2. 中间：制卡表单（deck / 模板 / 文本 / 批量）+ 生成后 Job 状态与卡片列表
// 3. 底部：保存到 Hub / 导出 .apkg
//
// 对接后端：AnkiTemplates / AnkiAddTemplate / AnkiGenerate
//           AnkiSave / AnkiExport

import { useEffect, useState } from "react";
import { useAnkiStore, type AnkiCard, type AnkiTemplate } from "@/state/anki";
import { Button } from "@/components/ui/Button";
import { Input } from "@/components/ui/Input";
import { Textarea } from "@/components/ui/Textarea";
import { Badge } from "@/components/ui/Badge";
import { cn, uid } from "@/lib/utils";
import {
  Layers,
  Plus,
  Sparkles,
  Loader2,
  AlertCircle,
  Save,
  Download,
  Inbox,
  CheckCircle2,
  RotateCcw,
  X,
  FileCode,
} from "lucide-react";

export function AnkiPage() {
  const loadTemplates = useAnkiStore((s) => s.loadTemplates);
  const setAddTemplateOpen = useAnkiStore((s) => s.setAddTemplateOpen);
  const templates = useAnkiStore((s) => s.templates);
  const job = useAnkiStore((s) => s.job);

  // 挂载时拉取模板
  useEffect(() => {
    void loadTemplates();
  }, [loadTemplates]);

  return (
    <div className="flex h-full w-full min-h-0 flex-col bg-background">
      {/* —— 顶部统计卡（对齐原版 TaskDashboard） —— */}
      <div className="shrink-0 border-b border-[var(--shell-seam)] bg-[var(--shell-workspace-panel)] px-4 py-3">
        <div className="grid grid-cols-4 gap-3">
          <StatCard label="模板数" value={String(templates.length ?? 0)} icon="▦" />
          <StatCard label="当前任务" value={job ? job.deck || "进行中" : "无"} icon="🃏" />
          <StatCard label="已生成卡片" value={String(job?.cards?.length ?? 0)} icon="📇" />
          <StatCard label="任务状态" value={job ? jobStatusCN(job.status) : "空闲"} icon={job?.status === "failed" ? "⚠" : "●"} danger={job?.status === "failed"} />
        </div>
      </div>

      <div className="flex min-h-0 flex-1">
      {/* —— 左：模板列表 —— */}
      <aside className="w-56 shrink-0 border-r border-border bg-card">
        <TemplateList />
      </aside>

      {/* —— 中：制卡表单 + Job / 卡片列表 —— */}
      <section className="flex min-w-0 flex-1 flex-col">
        <CardFactory />
      </section>

      </div>

      {/* 添加模板弹窗 */}
      <AddTemplateDialog />
    </div>
  );
}

// —— 统计卡 ——
function StatCard({ label, value, icon, danger = false }: { label: string; value: string; icon: string; danger?: boolean }) {
  return (
    <div className="rounded-lg border border-[var(--border-default)] bg-[var(--shell-inspector-panel)] px-3.5 py-2.5">
      <div className="flex items-center gap-1.5 text-[10px] font-semibold uppercase tracking-wider text-muted-foreground/70">
        <span className={danger ? "text-destructive" : "text-primary"}>{icon}</span>
        {label}
      </div>
      <div className={cn("mt-1 truncate text-[15px] font-semibold", danger ? "text-destructive" : "text-foreground")}>
        {value}
      </div>
    </div>
  );
}

function jobStatusCN(s: string): string {
  switch (s) {
    case "running": return "生成中";
    case "done": return "已完成";
    case "failed": return "失败";
    default: return "待开始";
  }
}

// —— 模板列表 ——
function TemplateList() {
  const templates = useAnkiStore((s) => s.templates);
  const selectedTemplateId = useAnkiStore((s) => s.selectedTemplateId);
  const setSelectedTemplate = useAnkiStore((s) => s.setSelectedTemplate);
  const loadingTemplates = useAnkiStore((s) => s.loadingTemplates);
  const setAddTemplateOpen = useAnkiStore((s) => s.setAddTemplateOpen);

  return (
    <div className="flex h-full w-full flex-col">
      <div className="shrink-0 border-b border-border px-3 py-2.5">
        <div className="flex items-center justify-between">
          <div className="text-[10px] font-semibold uppercase tracking-wider text-muted-foreground/70">
            模板
          </div>
          <button
            type="button"
            onClick={() => setAddTemplateOpen(true)}
            title="添加模板"
            className="rounded p-1 text-muted-foreground hover:bg-accent hover:text-foreground"
          >
            <Plus size={12} />
          </button>
        </div>
      </div>
      <div className="min-h-0 flex-1 overflow-y-auto scrollbar-dark px-2 py-2">
        {loadingTemplates && templates.length === 0 ? (
          <div className="flex items-center gap-2 py-4 text-[11px] text-muted-foreground">
            <Loader2 size={12} className="animate-spin" />
            加载模板…
          </div>
        ) : templates.length === 0 ? (
          <div className="px-2 py-4 text-center text-[11px] text-muted-foreground">
            暂无模板
          </div>
        ) : (
          <ul className="space-y-0.5">
            {templates.map((t) => (
              <li key={t.id}>
                <button
                  type="button"
                  onClick={() => setSelectedTemplate(t.id)}
                  className={cn(
                    "flex w-full items-center gap-2 rounded-md px-2 py-1.5 text-left text-sm transition-colors",
                    selectedTemplateId === t.id
                      ? "bg-primary/12 text-primary font-medium"
                      : "text-muted-foreground hover:bg-accent hover:text-foreground"
                  )}
                >
                  <FileCode size={13} className="shrink-0" />
                  <span className="min-w-0 flex-1 truncate">{t.name}</span>
                </button>
              </li>
            ))}
          </ul>
        )}
      </div>
    </div>
  );
}

// —— 制卡表单 + Job / 卡片列表 ——
function CardFactory() {
  const deck = useAnkiStore((s) => s.deck);
  const setDeck = useAnkiStore((s) => s.setDeck);
  const text = useAnkiStore((s) => s.text);
  const setText = useAnkiStore((s) => s.setText);
  const batch = useAnkiStore((s) => s.batch);
  const setBatch = useAnkiStore((s) => s.setBatch);
  const templates = useAnkiStore((s) => s.templates);
  const selectedTemplateId = useAnkiStore((s) => s.selectedTemplateId);
  const setSelectedTemplate = useAnkiStore((s) => s.setSelectedTemplate);
  const generating = useAnkiStore((s) => s.generating);
  const job = useAnkiStore((s) => s.job);
  const error = useAnkiStore((s) => s.error);
  const generate = useAnkiStore((s) => s.generate);

  return (
    <div className="flex h-full w-full min-w-0 flex-col">
      {/* 顶部表单 */}
      <div className="shrink-0 space-y-3 border-b border-border bg-card px-4 py-3">
        <div className="flex items-center gap-2">
          <div className="flex h-7 w-7 shrink-0 items-center justify-center rounded-md bg-primary/15 text-primary">
            <Layers size={15} />
          </div>
          <h1 className="shrink-0 text-sm font-semibold text-foreground">
            Anki 卡片工厂
          </h1>
        </div>
        <div className="grid grid-cols-3 gap-3">
          {/* deck 名 */}
          <div>
            <label className="mb-1 block text-[10px] font-semibold uppercase tracking-wider text-muted-foreground/70">
              Deck 名称
            </label>
            <Input
              value={deck}
              onChange={(e) => setDeck(e.target.value)}
              placeholder="例如：生物学基础"
              className="h-8 text-[13px]"
            />
          </div>
          {/* 模板选择 */}
          <div>
            <label className="mb-1 block text-[10px] font-semibold uppercase tracking-wider text-muted-foreground/70">
              模板
            </label>
            <select
              value={selectedTemplateId}
              onChange={(e) => setSelectedTemplate(e.target.value)}
              className="flex h-8 w-full rounded-md border border-input bg-transparent px-2 text-[13px] text-foreground focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring"
            >
              {templates.length === 0 && <option value="">无模板</option>}
              {templates.map((t) => (
                <option key={t.id} value={t.id}>
                  {t.name}
                </option>
              ))}
            </select>
          </div>
          {/* 批量大小 */}
          <div>
            <label className="mb-1 block text-[10px] font-semibold uppercase tracking-wider text-muted-foreground/70">
              每批卡片数
            </label>
            <Input
              type="number"
              min={1}
              max={20}
              value={batch}
              onChange={(e) => setBatch(Number(e.target.value) || 1)}
              className="h-8 text-[13px]"
            />
          </div>
        </div>
        {/* 源文本 */}
        <div>
          <label className="mb-1 block text-[10px] font-semibold uppercase tracking-wider text-muted-foreground/70">
            源文本
          </label>
          <Textarea
            value={text}
            onChange={(e) => setText(e.target.value)}
            placeholder="粘贴需要制卡的文本内容…"
            rows={4}
            className="resize-none text-[12px]"
          />
        </div>
        {/* 生成按钮 */}
        <div className="flex items-center justify-end">
          <Button
            size="sm"
            disabled={generating || !deck.trim() || !text.trim()}
            onClick={() => void generate()}
          >
            {generating ? (
              <Loader2 size={13} className="animate-spin" />
            ) : (
              <Sparkles size={13} />
            )}
            {generating ? "生成中…" : "生成卡片"}
          </Button>
        </div>
      </div>

      {/* 错误横幅 */}
      {error && (
        <div className="shrink-0 flex items-center gap-2 border-b border-destructive/30 bg-destructive/10 px-4 py-2 text-xs text-destructive">
          <AlertCircle size={12} />
          <span className="truncate">{error}</span>
        </div>
      )}

      {/* 主体：Job 状态 + 卡片列表 */}
      <div className="min-h-0 flex-1 overflow-y-auto scrollbar-dark">
        {generating && !job ? (
          <CenteredLoading label="AI 正在生成卡片…" />
        ) : !job ? (
          <CenteredEmpty
            icon={Layers}
            title="尚未生成卡片"
            hint={'填写表单后点击「生成卡片」开始'}
          />
        ) : (
          <JobView />
        )}
      </div>
    </div>
  );
}

// —— Job 状态 + 卡片列表 ——
function JobView() {
  const job = useAnkiStore((s) => s.job);
  const saving = useAnkiStore((s) => s.saving);
  const savedUri = useAnkiStore((s) => s.savedUri);
  const save = useAnkiStore((s) => s.save);
  const exportApkg = useAnkiStore((s) => s.exportApkg);
  if (!job) return null;

  const progress = job.total > 0 ? Math.round((job.done / job.total) * 100) : 0;

  return (
    <div className="space-y-4 px-4 py-4 animate-fade-in">
      {/* Job 状态卡片 */}
      <div className="rounded-lg border border-border bg-card p-4 shadow-[var(--shadow-shell-soft)]">
        <div className="flex items-center justify-between">
          <div className="flex items-center gap-2">
            <h3 className="text-sm font-semibold text-foreground">
              {job.deck || "未命名 Deck"}
            </h3>
            <JobStatusBadge status={job.status} />
          </div>
          <span className="font-mono text-[10px] text-muted-foreground/60">
            {job.id.slice(0, 8)}
          </span>
        </div>
        {/* 进度条 */}
        <div className="mt-3">
          <div className="mb-1 flex items-center justify-between text-[11px] text-muted-foreground">
            <span>进度</span>
            <span className="font-mono">
              {job.done} / {job.total}（{progress}%）
            </span>
          </div>
          <div className="h-2 overflow-hidden rounded-full bg-muted">
            <div
              className="h-full rounded-full bg-primary transition-all duration-300"
              style={{ width: `${progress}%` }}
            />
          </div>
        </div>
      </div>

      {/* 保存成功提示 */}
      {savedUri && (
        <div className="flex items-center gap-2 rounded-md border border-emerald-500/30 bg-emerald-500/10 px-3 py-2 text-[11px] text-emerald-500">
          <CheckCircle2 size={12} />
          已保存：{savedUri}
        </div>
      )}

      {/* 卡片列表 */}
      <div>
        <div className="mb-2 flex items-center justify-between">
          <div className="text-[10px] font-semibold uppercase tracking-wider text-muted-foreground/70">
            卡片列表（{job.cards.length} 张）
          </div>
        </div>
        {job.cards.length === 0 ? (
          <div className="py-8 text-center text-xs text-muted-foreground">
            无卡片
          </div>
        ) : (
          <ul className="grid grid-cols-1 gap-2 md:grid-cols-2">
            {job.cards.map((card, i) => (
              <CardItem key={card.id || i} card={card} index={i} />
            ))}
          </ul>
        )}
      </div>

      {/* 底部操作 */}
      <div className="sticky bottom-0 flex items-center justify-end gap-2 border-t border-border bg-card px-4 py-3">
        <Button
          variant="outline"
          size="sm"
          disabled={saving || job.cards.length === 0}
          onClick={() => void exportApkg()}
        >
          <Download size={13} />
          导出 .apkg
        </Button>
        <Button
          size="sm"
          disabled={saving || job.cards.length === 0}
          onClick={() => void save()}
        >
          {saving ? (
            <Loader2 size={13} className="animate-spin" />
          ) : (
            <Save size={13} />
          )}
          {saving ? "保存中…" : "保存到 Hub"}
        </Button>
      </div>
    </div>
  );
}

// —— 卡片翻转预览 ——
function CardItem({ card, index }: { card: AnkiCard; index: number }) {
  const [flipped, setFlipped] = useState(false);
  return (
    <li
      onClick={() => setFlipped((f) => !f)}
      className="group cursor-pointer rounded-md border border-border bg-card p-3 transition-colors hover:border-primary/40 animate-fadeSlideUp"
      style={{ animationDelay: `${Math.min(index * 30, 300)}ms` }}
    >
      <div className="mb-1.5 flex items-center justify-between">
        <span className="text-[10px] font-medium text-muted-foreground">
          #{index + 1}
        </span>
        <div className="flex items-center gap-1">
          {card.tags?.map((tag) => (
            <span
              key={tag}
              className="rounded bg-muted px-1 py-0.5 text-[9px] text-muted-foreground"
            >
              {tag}
            </span>
          ))}
          <RotateCcw
            size={10}
            className="text-muted-foreground/40 opacity-0 transition-opacity group-hover:opacity-100"
          />
        </div>
      </div>
      <div className="min-h-[3rem]">
        {!flipped ? (
          <div>
            <div className="mb-1 text-[9px] font-semibold uppercase tracking-wider text-primary/70">
              正面
            </div>
            <p className="text-[12px] leading-relaxed text-foreground">
              {card.front}
            </p>
          </div>
        ) : (
          <div>
            <div className="mb-1 text-[9px] font-semibold uppercase tracking-wider text-success">
              背面
            </div>
            <p className="text-[12px] leading-relaxed text-foreground/85">
              {card.back}
            </p>
          </div>
        )}
      </div>
    </li>
  );
}

// —— 添加模板弹窗 ——
function AddTemplateDialog() {
  const open = useAnkiStore((s) => s.addTemplateOpen);
  const setOpen = useAnkiStore((s) => s.setAddTemplateOpen);
  const addTemplate = useAnkiStore((s) => s.addTemplate);
  const [name, setName] = useState("");
  const [front, setFront] = useState("");
  const [back, setBack] = useState("");
  const [css, setCss] = useState("");

  if (!open) return null;

  const handleClose = () => {
    setName("");
    setFront("");
    setBack("");
    setCss("");
    setOpen(false);
  };

  const handleSubmit = async () => {
    if (!name.trim()) return;
    const tpl: AnkiTemplate = {
      id: uid("tpl"),
      name: name.trim(),
      front: front.trim(),
      back: back.trim(),
      style: "",
      css: css.trim(),
    };
    await addTemplate(tpl);
    handleClose();
  };

  return (
    <div
      className="fixed inset-0 z-50 flex items-center justify-center bg-black/60 backdrop-blur-sm animate-fade-in"
      onClick={handleClose}
    >
      <div
        className="relative w-full max-w-lg overflow-hidden rounded-lg border border-border bg-card shadow-[var(--shadow-shell-floating)] animate-zoom-in"
        onClick={(e) => e.stopPropagation()}
      >
        {/* 头部 */}
        <div className="flex items-center justify-between border-b border-border px-4 py-3">
          <div className="flex items-center gap-2">
            <Plus size={15} className="text-primary" />
            <h3 className="text-sm font-semibold text-foreground">添加模板</h3>
          </div>
          <button
            type="button"
            onClick={handleClose}
            className="rounded p-1 text-muted-foreground hover:bg-accent hover:text-foreground"
          >
            <X size={14} />
          </button>
        </div>
        {/* 内容 */}
        <div className="space-y-3 px-4 py-4">
          <div>
            <label className="mb-1 block text-[10px] font-semibold uppercase tracking-wider text-muted-foreground/70">
              模板名称
            </label>
            <Input
              value={name}
              onChange={(e) => setName(e.target.value)}
              placeholder="例如：基础 QA"
              className="h-8 text-[13px]"
            />
          </div>
          <div>
            <label className="mb-1 block text-[10px] font-semibold uppercase tracking-wider text-muted-foreground/70">
              正面模板
            </label>
            <Textarea
              value={front}
              onChange={(e) => setFront(e.target.value)}
              placeholder={'<div class="card">{{Front}}</div>'}
              rows={2}
              className="resize-none font-mono text-[11px]"
            />
          </div>
          <div>
            <label className="mb-1 block text-[10px] font-semibold uppercase tracking-wider text-muted-foreground/70">
              背面模板
            </label>
            <Textarea
              value={back}
              onChange={(e) => setBack(e.target.value)}
              placeholder={'<div class="card">{{Front}}</div><hr><div>{{Back}}</div>'}
              rows={2}
              className="resize-none font-mono text-[11px]"
            />
          </div>
          <div>
            <label className="mb-1 block text-[10px] font-semibold uppercase tracking-wider text-muted-foreground/70">
              CSS（可选）
            </label>
            <Textarea
              value={css}
              onChange={(e) => setCss(e.target.value)}
              placeholder=".card { font-size: 18px; }"
              rows={2}
              className="resize-none font-mono text-[11px]"
            />
          </div>
        </div>
        {/* 底部 */}
        <div className="flex items-center justify-end gap-2 border-t border-border bg-card px-4 py-3">
          <Button variant="ghost" size="sm" onClick={handleClose}>
            取消
          </Button>
          <Button size="sm" disabled={!name.trim()} onClick={handleSubmit}>
            <Plus size={12} />
            添加
          </Button>
        </div>
      </div>
    </div>
  );
}

// —— Job 状态徽章 ——
function JobStatusBadge({ status }: { status: string }) {
  const variant =
    status === "done"
      ? "success"
      : status === "running"
      ? "info"
      : status === "failed"
      ? "destructive"
      : "secondary";
  const label =
    status === "done"
      ? "完成"
      : status === "running"
      ? "运行中"
      : status === "failed"
      ? "失败"
      : "待处理";
  return <Badge variant={variant as "success"}>{label}</Badge>;
}

// —— 居中加载状态 ——
function CenteredLoading({ label }: { label: string }) {
  return (
    <div className="flex h-full flex-col items-center justify-center gap-3 text-center">
      <div className="flex h-12 w-12 items-center justify-center rounded-full bg-primary/10 text-primary">
        <Loader2 size={22} className="animate-spin" />
      </div>
      <div className="space-y-1">
        <div className="text-sm font-medium text-foreground">{label}</div>
      </div>
    </div>
  );
}

// —— 居中空状态 ——
function CenteredEmpty({
  icon: Icon,
  title,
  hint,
}: {
  icon: typeof Inbox;
  title: string;
  hint: string;
}) {
  return (
    <div className="flex h-full flex-col items-center justify-center gap-3 px-6 text-center">
      <div className="flex h-12 w-12 items-center justify-center rounded-full bg-muted text-muted-foreground">
        <Icon size={22} />
      </div>
      <div className="space-y-1">
        <div className="text-sm font-medium text-foreground">{title}</div>
        <div className="text-xs text-muted-foreground">{hint}</div>
      </div>
    </div>
  );
}

// 保留未使用图标引用
void AlertCircle;
void Inbox;
