// TemplateManagerPage —— Anki 模板管理（字段管理 / 插入栏 / 校验 / 预览）
// ------------------------------------------------------------
// 对接后端 templatemgr（Template* 方法）：
// - 模板列表（内置/自定义）+ 默认模板设置
// - 编辑器：front / back / style / css
// - 插入栏：{{Front}} {{Back}} 等字段快捷插入
// - 校验：字段缺失提示 + 前端预览

import { useEffect, useMemo, useState } from "react";
import { templateApi, type Template } from "@/lib/template";
import { Button } from "@/components/ui/Button";
import { Input } from "@/components/ui/Input";
import { Badge } from "@/components/ui/Badge";
import { Textarea } from "@/components/ui/Textarea";
import {
  LayoutTemplate,
  Plus,
  Trash2,
  Star,
  Copy,
  Check,
  Download,
  Upload,
  AlertCircle,
  Loader2,
} from "lucide-react";
import { cn } from "@/lib/utils";

function replaceAll(s: string, search: string, replace: string): string {
  return s.split(search).join(replace);
}

const FIELD_TAGS = [
  "{{Front}}",
  "{{Back}}",
  "{{cloze:Text}}",
  "{{Extra}}",
  "{{BackExtra}}",
  "{{Tags}}",
];

export function TemplateManagerPage() {
  const [templates, setTemplates] = useState<Template[]>([]);
  const [selectedId, setSelectedId] = useState<string | null>(null);
  const [defaultId, setDefaultId] = useState<string>("");
  const [draft, setDraft] = useState<Template | null>(null);
  const [busy, setBusy] = useState(false);
  const [toast, setToast] = useState<string | null>(null);
  const [error, setError] = useState<string | null>(null);
  const [previewFront, setPreviewFront] = useState("光合作用");

  const load = async () => {
    const [ts, def] = await Promise.all([templateApi.list(), templateApi.getDefaultID()]);
    if (ts) {
      setTemplates(ts);
      if (!selectedId && ts.length > 0) {
        setSelectedId(ts[0].id);
        setDraft({ ...ts[0] });
      }
    }
    if (def) setDefaultId(def);
  };

  useEffect(() => {
    void load();
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, []);

  const selected = useMemo(() => templates.find((t) => t.id === selectedId) ?? null, [templates, selectedId]);

  const select = (t: Template) => {
    setSelectedId(t.id);
    setDraft({ ...t });
    setError(null);
  };

  const save = async () => {
    if (!draft) return;
    if (!draft.name.trim() || !draft.front.trim() || !draft.back.trim()) {
      setError("名称 / 正面 / 背面模板不能为空");
      return;
    }
    setBusy(true);
    const saved = await templateApi.update({
      id: draft.id,
      name: draft.name,
      front: draft.front,
      back: draft.back,
      style: draft.style,
      css: draft.css,
    });
    setBusy(false);
    if (saved) {
      setTemplates((ts) => ts.map((t) => (t.id === saved.id ? saved : t)));
      setToast("已保存");
      setTimeout(() => setToast(null), 1500);
    }
  };

  const create = async () => {
    const t = await templateApi.create({
      name: "新模板",
      front: '<div class="card">{{Front}}</div>',
      back: '<div class="card">{{Front}}</div><hr id=answer>{{Back}}</div>',
      css: ".card { font-family: sans-serif; font-size: 18px; padding: 12px; }",
    });
    if (t) {
      setTemplates((ts) => [...ts, t]);
      select(t);
    }
  };

  const remove = async (id: string) => {
    await templateApi.remove(id);
    setTemplates((ts) => ts.filter((t) => t.id !== id));
    if (selectedId === id) {
      const rest = templates.filter((t) => t.id !== id);
      if (rest.length > 0) select(rest[0]);
      else setSelectedId(null);
    }
  };

  const insertTag = (tag: string, field: "front" | "back") => {
    if (!draft) return;
    setDraft({ ...draft, [field]: draft[field] + tag });
  };

  const exportOne = async () => {
    if (!draft) return;
    const data = await templateApi.export(draft.id);
    if (data) {
      const blob = new Blob([new Uint8Array(data)], { type: "application/json" });
      const url = URL.createObjectURL(blob);
      const a = document.createElement("a");
      a.href = url;
      a.download = `${draft.name}.template.json`;
      a.click();
      URL.revokeObjectURL(url);
    }
  };

  // 校验：检查必备字段
  const missingFields = useMemo(() => {
    if (!draft) return [];
    const missing: string[] = [];
    if (!draft.front.includes("{{Front}}") && !draft.front.includes("{{cloze:")) missing.push("{{Front}} 或 {{cloze:Text}}");
    if (!draft.back.includes("{{Back}}") && !draft.back.includes("{{cloze:Text}}")) missing.push("{{Back}} 或 {{cloze:Text}}");
    return missing;
  }, [draft]);

  // 预览渲染（简化：替换字段）
  const previewHTML = useMemo(() => {
    if (!draft) return "";
    return replaceAll(replaceAll(replaceAll(replaceAll(replaceAll(replaceAll(
      draft.back,
      "{{Front}}", previewFront),
      "{{cloze:Text}}", `<span class="cloze">${previewFront}</span>`),
      "{{Back}}", "答案是：叶绿体"),
      "{{Extra}}", ""),
      "{{BackExtra}}", "中文释义"),
      "{{Tags}}", "science")
      + (draft.style ? `<style>${draft.style}</style>` : "");
  }, [draft, previewFront]);

  return (
    <div className="flex h-full w-full min-h-0 bg-background">
      {/* —— 左：模板列表 —— */}
      <aside className="w-56 shrink-0 border-r border-border bg-card">
        <div className="flex h-full flex-col">
          <div className="shrink-0 border-b border-border px-3 py-2.5">
            <div className="flex items-center gap-2">
              <div className="flex h-6 w-6 items-center justify-center rounded-md bg-primary/15 text-primary">
                <LayoutTemplate size={13} />
              </div>
              <span className="text-[11px] font-semibold uppercase tracking-wider text-muted-foreground/70">
                模板
              </span>
              <Button size="sm" variant="ghost" className="ml-auto h-6 w-6 p-0" onClick={() => void create()}>
                <Plus size={13} />
              </Button>
            </div>
          </div>
          <div className="min-h-0 flex-1 overflow-y-auto scrollbar-dark p-1.5">
            {templates.map((t) => (
              <div
                key={t.id}
                onClick={() => select(t)}
                className={cn(
                  "group flex cursor-pointer items-center gap-1.5 rounded-md px-2 py-1.5 text-[12px]",
                  selectedId === t.id ? "bg-primary/12 font-medium text-primary" : "text-muted-foreground hover:bg-accent"
                )}
              >
                {defaultId === t.id && <Star size={10} className="shrink-0 text-amber-500" />}
                <span className="min-w-0 flex-1 truncate">{t.name}</span>
                {t.isBuiltin ? (
                  <Badge variant="secondary" className="text-[8px]">内置</Badge>
                ) : (
                  <button
                    onClick={(e) => { e.stopPropagation(); void remove(t.id); }}
                    className="hidden rounded p-0.5 text-muted-foreground hover:text-destructive group-hover:block"
                  >
                    <Trash2 size={10} />
                  </button>
                )}
              </div>
            ))}
          </div>
        </div>
      </aside>

      {/* —— 中：编辑器 —— */}
      <section className="flex min-w-0 flex-1 flex-col">
        <div className="flex shrink-0 items-center gap-2 border-b border-border bg-card px-4 py-2.5">
          <h1 className="text-sm font-semibold text-foreground">模板编辑器</h1>
          {selected?.isBuiltin && <Badge variant="secondary" className="text-[9px]">内置模板（只读）</Badge>}
          <div className="ml-auto flex items-center gap-2">
            <Button size="sm" variant="outline" className="h-7" onClick={() => void exportOne()} disabled={!draft}>
              <Download size={12} className="mr-1" />
              导出
            </Button>
            <Button size="sm" className="h-7" onClick={() => void save()} disabled={busy || !draft || draft.isBuiltin}>
              {busy ? <Loader2 size={12} className="mr-1 animate-spin" /> : <Check size={12} className="mr-1" />}
              保存
            </Button>
          </div>
        </div>

        {!draft ? (
          <div className="flex flex-1 items-center justify-center text-muted-foreground">
            <LayoutTemplate size={28} className="mb-2 opacity-40" />
            <p className="text-sm">选择一个模板或新建</p>
          </div>
        ) : (
          <div className="min-h-0 flex-1 space-y-3 overflow-y-auto scrollbar-dark p-4">
            {error && (
              <div className="flex items-center gap-1.5 text-[11px] text-destructive">
                <AlertCircle size={12} />
                {error}
              </div>
            )}
            {/* 名称 */}
            <div>
              <label className="mb-1 block text-[10px] font-semibold uppercase tracking-wider text-muted-foreground/70">模板名称</label>
              <Input
                value={draft.name}
                disabled={draft.isBuiltin}
                onChange={(e) => setDraft({ ...draft, name: e.target.value })}
                className="h-8 text-[12px]"
              />
            </div>

            {/* 校验警告 */}
            {missingFields.length > 0 && (
              <div className="rounded-md border border-amber-500/30 bg-amber-500/10 px-3 py-2 text-[11px] text-amber-600">
                提示：模板缺少 {missingFields.join("、")} 字段
              </div>
            )}

            {/* Front / Back 编辑器 */}
            {(["front", "back"] as const).map((field) => (
              <div key={field}>
                <div className="mb-1 flex items-center justify-between">
                  <label className="text-[10px] font-semibold uppercase tracking-wider text-muted-foreground/70">
                    {field === "front" ? "正面 Front" : "背面 Back"}
                  </label>
                  {/* 插入栏 */}
                  <div className="flex items-center gap-0.5">
                    {FIELD_TAGS.map((tag) => (
                      <button
                        key={tag}
                        onClick={() => insertTag(tag, field)}
                        disabled={draft.isBuiltin}
                        className="rounded bg-muted px-1.5 py-0.5 font-mono text-[9px] text-muted-foreground hover:bg-primary/15 hover:text-primary disabled:opacity-40"
                      >
                        {tag}
                      </button>
                    ))}
                  </div>
                </div>
                <Textarea
                  value={draft[field]}
                  disabled={draft.isBuiltin}
                  onChange={(e) => setDraft({ ...draft, [field]: e.target.value })}
                  rows={field === "front" ? 4 : 5}
                  spellCheck={false}
                  className="resize-y font-mono text-[11px] leading-relaxed"
                />
              </div>
            ))}

            {/* CSS */}
            <div>
              <label className="mb-1 block text-[10px] font-semibold uppercase tracking-wider text-muted-foreground/70">样式 CSS</label>
              <Textarea
                value={draft.css ?? ""}
                disabled={draft.isBuiltin}
                onChange={(e) => setDraft({ ...draft, css: e.target.value })}
                rows={3}
                spellCheck={false}
                className="resize-y font-mono text-[11px]"
              />
            </div>
          </div>
        )}
      </section>

      {/* —— 右：预览 —— */}
      <aside className="w-72 shrink-0 border-l border-border bg-card">
        <div className="flex h-full flex-col">
          <div className="shrink-0 border-b border-border px-3 py-2.5">
            <div className="text-[10px] font-semibold uppercase tracking-wider text-muted-foreground/70">预览</div>
          </div>
          <div className="shrink-0 border-b border-border px-3 py-2">
            <Input
              value={previewFront}
              onChange={(e) => setPreviewFront(e.target.value)}
              placeholder="正面内容预览…"
              className="h-7 text-[11px]"
            />
          </div>
          <div className="min-h-0 flex-1 overflow-y-auto scrollbar-dark p-3">
            {previewHTML ? (
              <div className="rounded-md border border-border bg-white p-4" dangerouslySetInnerHTML={{ __html: previewHTML }} />
            ) : (
              <p className="text-center text-[11px] text-muted-foreground">选择模板后预览</p>
            )}
          </div>
        </div>
      </aside>
    </div>
  );
}
