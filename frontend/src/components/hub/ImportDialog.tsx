// ImportDialog —— 导入新资源对话框
// ------------------------------------------------------------
// 用户选择：类型 + 标题 + 标签 + 文件内容
// 提交时调用 HubImportResource

import { useHubStore, RESOURCE_TYPES, type ResourceType } from "@/state/hub";
import { Button } from "@/components/ui/Button";
import { Input } from "@/components/ui/Input";
import { Textarea } from "@/components/ui/Textarea";
import { cn, uid } from "@/lib/utils";
import { Upload, X, FileText, Loader2, CheckCircle2 } from "lucide-react";
import { useRef, useState } from "react";

interface ImportDialogProps {
  open: boolean;
  onClose: () => void;
}

export function ImportDialog({ open, onClose }: ImportDialogProps) {
  const importResource = useHubStore((s) => s.importResource);
  const loading = useHubStore((s) => s.loading);
  const error = useHubStore((s) => s.error);

  const [type, setType] = useState<ResourceType>("note");
  const [title, setTitle] = useState("");
  const [tags, setTags] = useState("");
  const [fileBytes, setFileBytes] = useState<Uint8Array | null>(null);
  const [fileName, setFileName] = useState<string>("");
  const [textMode, setTextMode] = useState(true);
  const [text, setText] = useState("");
  const [success, setSuccess] = useState<string | null>(null);
  const fileInputRef = useRef<HTMLInputElement>(null);

  if (!open) return null;

  const reset = () => {
    setType("note");
    setTitle("");
    setTags("");
    setFileBytes(null);
    setFileName("");
    setText("");
    setTextMode(true);
    setSuccess(null);
  };

  const handleClose = () => {
    reset();
    onClose();
  };

  const handleFile = async (file: File) => {
    setFileName(file.name);
    const buf = await file.arrayBuffer();
    setFileBytes(new Uint8Array(buf));
    if (!title) setTitle(file.name.replace(/\.[^.]+$/, ""));
    setTextMode(false);
  };

  const handleSubmit = async () => {
    if (!title.trim()) return;
    // 数据来源：文本模式用 utf-8 编码；文件模式用文件字节
    let bytes: Uint8Array;
    if (textMode) {
      bytes = new TextEncoder().encode(text);
    } else if (fileBytes) {
      bytes = fileBytes;
    } else {
      return;
    }
    const tagList = tags
      .split(/[,，\s]+/)
      .map((t) => t.trim())
      .filter(Boolean);
    const uri = await importResource(type, title.trim(), bytes, tagList);
    if (uri) {
      setSuccess(uri);
      // 1.5s 后自动关闭
      setTimeout(handleClose, 1500);
    }
  };

  return (
    <div
      className="fixed inset-0 z-50 flex items-center justify-center bg-black/60 backdrop-blur-sm animate-fade-in"
      onClick={handleClose}
    >
      <div
        className="relative w-full max-w-lg overflow-hidden rounded-lg border border-border bg-card shadow-floating animate-zoom-in"
        onClick={(e) => e.stopPropagation()}
      >
        {/* 头部 */}
        <div className="flex items-center justify-between border-b border-border px-4 py-3">
          <div className="flex items-center gap-2">
            <Upload size={15} className="text-primary" />
            <h3 className="text-sm font-semibold text-foreground">导入新资源</h3>
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
          {/* 类型选择 */}
          <div>
            <label className="mb-1 block text-[10px] font-semibold uppercase tracking-wider text-muted-foreground/70">
              资源类型
            </label>
            <div className="flex flex-wrap gap-1">
              {RESOURCE_TYPES.map((meta) => (
                <button
                  key={meta.type}
                  type="button"
                  onClick={() => setType(meta.type)}
                  className={cn(
                    "rounded-full border px-2 py-0.5 text-[11px] font-medium transition-colors",
                    type === meta.type
                      ? "border-primary/40 bg-primary/15 text-primary"
                      : "border-border bg-transparent text-muted-foreground hover:bg-accent hover:text-foreground"
                  )}
                >
                  {meta.label}
                </button>
              ))}
            </div>
          </div>

          {/* 标题 */}
          <div>
            <label className="mb-1 block text-[10px] font-semibold uppercase tracking-wider text-muted-foreground/70">
              标题
            </label>
            <Input
              value={title}
              onChange={(e) => setTitle(e.target.value)}
              placeholder="给资源起个名字…"
              className="h-8 text-[13px]"
            />
          </div>

          {/* 标签 */}
          <div>
            <label className="mb-1 block text-[10px] font-semibold uppercase tracking-wider text-muted-foreground/70">
              标签（逗号或空格分隔）
            </label>
            <Input
              value={tags}
              onChange={(e) => setTags(e.target.value)}
              placeholder="math, algebra, 重点"
              className="h-8 text-[13px]"
            />
          </div>

          {/* 数据来源切换 */}
          <div>
            <div className="mb-1.5 flex items-center justify-between">
              <label className="text-[10px] font-semibold uppercase tracking-wider text-muted-foreground/70">
                内容
              </label>
              <div className="flex items-center gap-1 text-[10px]">
                <ToggleBtn
                  active={textMode}
                  onClick={() => setTextMode(true)}
                  label="文本"
                />
                <ToggleBtn
                  active={!textMode}
                  onClick={() => setTextMode(false)}
                  label="文件"
                />
              </div>
            </div>
            {textMode ? (
              <Textarea
                value={text}
                onChange={(e) => setText(e.target.value)}
                placeholder="粘贴文本内容…"
                rows={5}
                className="resize-none text-[12px]"
              />
            ) : (
              <div
                onClick={() => fileInputRef.current?.click()}
                className="flex cursor-pointer flex-col items-center justify-center gap-2 rounded-md border border-dashed border-border bg-background px-4 py-6 text-center transition-colors hover:border-primary/40 hover:bg-accent/30"
              >
                <input
                  ref={fileInputRef}
                  type="file"
                  className="hidden"
                  onChange={(e) => {
                    const f = e.target.files?.[0];
                    if (f) void handleFile(f);
                  }}
                />
                <FileText size={20} className="text-muted-foreground" />
                <div className="text-xs text-foreground">
                  {fileName || "点击选择文件"}
                </div>
                {fileName && (
                  <div className="text-[10px] text-muted-foreground">
                    {fileBytes?.length ?? 0} 字节
                  </div>
                )}
              </div>
            )}
          </div>

          {/* 错误提示 */}
          {error && (
            <div className="rounded-md border border-destructive/30 bg-destructive/10 px-3 py-2 text-[11px] text-destructive">
              {error}
            </div>
          )}

          {/* 成功提示 */}
          {success && (
            <div className="flex items-center gap-2 rounded-md border border-emerald-500/30 bg-emerald-500/10 px-3 py-2 text-[11px] text-emerald-500">
              <CheckCircle2 size={12} />
              导入成功：{success}
            </div>
          )}
        </div>

        {/* 底部按钮 */}
        <div className="flex items-center justify-end gap-2 border-t border-border bg-card px-4 py-3">
          <Button variant="ghost" size="sm" onClick={handleClose}>
            取消
          </Button>
          <Button
            size="sm"
            onClick={handleSubmit}
            disabled={
              loading ||
              !title.trim() ||
              (textMode ? !text.trim() : !fileBytes) ||
              !!success
            }
          >
            {loading ? (
              <Loader2 size={12} className="animate-spin" />
            ) : (
              <Upload size={12} />
            )}
            {loading ? "导入中…" : "导入"}
          </Button>
        </div>
      </div>
    </div>
  );
}

function ToggleBtn({
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
        "rounded px-2 py-0.5 transition-colors",
        active
          ? "bg-primary/15 text-primary"
          : "text-muted-foreground hover:text-foreground"
      )}
    >
      {label}
    </button>
  );
}

// 保留 uid 引用以备未来扩展
void uid;
