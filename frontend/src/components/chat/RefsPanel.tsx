// RefsPanel —— 引用面板（右栏）
// ------------------------------------------------------------
// 设计要点：
// - 添加引用：输入 vfs:// URI 后回车
// - 引用列表：可移除单条 / 清空全部
// - 引用预览：仅显示 URI 与索引（真实内容拉取在阶段 4+）

import { useChatStore } from "@/state/chat";
import { Button } from "@/components/ui/Button";
import { X, Plus, FileText, Trash2 } from "lucide-react";
import { useState } from "react";

export function RefsPanel() {
  const refs = useChatStore((s) => s.refs);
  const addRef = useChatStore((s) => s.addRef);
  const removeRef = useChatStore((s) => s.removeRef);
  const clearRefs = useChatStore((s) => s.clearRefs);
  const [input, setInput] = useState("");
  const [error, setError] = useState<string | null>(null);

  const handleAdd = () => {
    const v = input.trim();
    if (!v) return;
    if (!/^[a-zA-Z]+:\/\//.test(v) && !v.startsWith("/")) {
      setError("URI 须为 vfs://xxx 或绝对路径");
      return;
    }
    addRef(v);
    setInput("");
    setError(null);
  };

  const handleKeyDown = (e: React.KeyboardEvent) => {
    if (e.key === "Enter") {
      e.preventDefault();
      handleAdd();
    }
  };

  return (
    <div className="flex min-h-0 flex-1 flex-col">
      {/* —— 添加输入 —— */}
      <div className="shrink-0 border-b border-border p-2.5">
        <div className="flex items-center gap-1.5">
          <input
            value={input}
            onChange={(e) => setInput(e.target.value)}
            onKeyDown={handleKeyDown}
            placeholder="vfs://note/xxx"
            className="flex-1 rounded-md border border-input bg-transparent px-2 py-1 text-xs text-foreground placeholder:text-muted-foreground focus:border-primary focus:outline-none focus:ring-1 focus:ring-ring"
          />
          <Button
            variant="outline"
            size="icon"
            className="h-7 w-7 shrink-0"
            onClick={handleAdd}
            aria-label="添加引用"
          >
            <Plus size={12} />
          </Button>
        </div>
        {error && (
          <div className="mt-1.5 text-[10px] text-destructive">{error}</div>
        )}
      </div>

      {/* —— 引用列表 —— */}
      <div className="min-h-0 flex-1 overflow-y-auto scrollbar-dark">
        {refs.length === 0 ? (
          <div className="flex h-full flex-col items-center justify-center gap-2 px-4 text-center">
            <FileText size={24} className="text-muted-foreground/40" />
            <div className="text-xs text-muted-foreground">
              还没有引用
            </div>
            <div className="text-[10px] text-muted-foreground/70">
              添加 vfs:// 资源作为对话上下文
            </div>
          </div>
        ) : (
          <ul className="space-y-1 p-2">
            {refs.map((uri, idx) => (
              <li
                key={uri + idx}
                className="group flex items-start gap-2 rounded-md border border-border bg-background px-2 py-1.5"
              >
                <FileText size={12} className="mt-0.5 shrink-0 text-primary" />
                <div className="min-w-0 flex-1">
                  <div className="truncate text-xs font-medium text-foreground">
                    {uri.split("/").pop() ?? uri}
                  </div>
                  <div className="truncate text-[10px] text-muted-foreground/70">
                    {uri}
                  </div>
                </div>
                <button
                  type="button"
                  onClick={() => removeRef(uri)}
                  className="shrink-0 rounded p-0.5 text-muted-foreground/60 opacity-0 transition-opacity hover:bg-destructive/15 hover:text-destructive group-hover:opacity-100"
                  aria-label="移除引用"
                >
                  <X size={11} />
                </button>
              </li>
            ))}
          </ul>
        )}
      </div>

      {/* —— 底部：清空 —— */}
      {refs.length > 0 && (
        <div className="shrink-0 border-t border-border p-2">
          <Button
            variant="ghost"
            size="sm"
            onClick={clearRefs}
            className="w-full justify-center text-destructive hover:bg-destructive/10"
          >
            <Trash2 size={11} />
            清空全部引用
          </Button>
        </div>
      )}
    </div>
  );
}
