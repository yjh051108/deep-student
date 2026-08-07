// InputBar —— 聊天输入栏（中栏下）
// ------------------------------------------------------------
// 设计要点：
// - textarea 自适应高度（1-8 行）
// - Enter 发送 / Shift+Enter 换行
// - 深度思考开关（紫色 chip）
// - 引用计数 chip（点击聚焦右侧面板）
// - 发送按钮：loading 状态切换为停止（暂未实现中断）

import { useChatStore } from "@/state/chat";
import { cn } from "@/lib/utils";
import { Button } from "@/components/ui/Button";
import { Send, Brain, Paperclip } from "lucide-react";
import { useEffect, useRef, useState } from "react";

export function InputBar() {
  const sendMessage = useChatStore((s) => s.sendMessage);
  const loading = useChatStore((s) => s.loading);
  const deepThink = useChatStore((s) => s.deepThink);
  const toggleDeepThink = useChatStore((s) => s.toggleDeepThink);
  const refs = useChatStore((s) => s.refs);
  const activeSessionId = useChatStore((s) => s.activeSessionId);
  const sessions = useChatStore((s) => s.sessions);

  const [text, setText] = useState("");
  const taRef = useRef<HTMLTextAreaElement>(null);

  const activeSession = sessions.find((s) => s.id === activeSessionId);

  // 自适应高度
  useEffect(() => {
    const ta = taRef.current;
    if (!ta) return;
    ta.style.height = "auto";
    ta.style.height = `${Math.min(ta.scrollHeight, 200)}px`;
  }, [text]);

  const handleSend = () => {
    const trimmed = text.trim();
    if (!trimmed || loading) return;
    setText("");
    void sendMessage(trimmed);
  };

  const handleKeyDown = (e: React.KeyboardEvent<HTMLTextAreaElement>) => {
    // Enter 发送，Shift+Enter 换行
    if (e.key === "Enter" && !e.shiftKey && !e.nativeEvent.isComposing) {
      e.preventDefault();
      handleSend();
    }
  };

  const placeholder = activeSession
    ? `向 ${activeSession.provider} 发送消息…`
    : "输入消息开始新会话…";

  return (
    <div className="shrink-0 border-t border-border bg-card px-6 py-3">
      <div className="mx-auto max-w-thread">
        {/* —— 工具栏（左对齐） —— */}
        <div className="mb-2 flex items-center gap-1.5">
          {/* 深度思考开关 */}
          <button
            type="button"
            onClick={toggleDeepThink}
            className={cn(
              "flex items-center gap-1.5 rounded-full border px-2.5 py-0.5 text-[11px] font-medium transition-colors",
              deepThink
                ? "border-primary/40 bg-primary/15 text-primary"
                : "border-border bg-transparent text-muted-foreground hover:bg-accent hover:text-foreground"
            )}
            aria-pressed={deepThink}
          >
            <Brain size={11} />
            <span>深度思考</span>
          </button>

          {/* 引用计数 */}
          <div className="flex items-center gap-1.5 rounded-full border border-border bg-transparent px-2.5 py-0.5 text-[11px] text-muted-foreground">
            <Paperclip size={11} />
            <span>引用 {refs.length}</span>
          </div>

          {activeSession && (
            <span className="ml-auto text-[10px] text-muted-foreground/60">
              {activeSession.provider} · {activeSession.model}
            </span>
          )}
        </div>

        {/* —— 输入区 —— */}
        <div className="flex items-end gap-2 rounded-row border border-border bg-background p-2 shadow-soft focus-within:border-primary/40 focus-within:ring-2 focus-within:ring-ring/20 transition-colors">
          <textarea
            ref={taRef}
            value={text}
            onChange={(e) => setText(e.target.value)}
            onKeyDown={handleKeyDown}
            placeholder={placeholder}
            rows={1}
            className="flex-1 resize-none bg-transparent px-2 py-1.5 text-sm text-foreground placeholder:text-muted-foreground focus:outline-none"
            style={{ minHeight: "32px", maxHeight: "200px" }}
          />
          <Button
            onClick={handleSend}
            disabled={!text.trim() || loading}
            size="icon"
            aria-label="发送消息"
          >
            <Send size={14} />
          </Button>
        </div>

        {/* —— 底部提示 —— */}
        <div className="mt-1.5 flex items-center justify-between text-[10px] text-muted-foreground/60">
          <span>
            <kbd className="font-mono">Enter</kbd> 发送 ·{" "}
            <kbd className="font-mono">Shift+Enter</kbd> 换行
          </span>
          {loading && <span className="animate-pulse">生成中…</span>}
        </div>
      </div>
    </div>
  );
}
