// InputBar —— 1:1 对齐原版 ChatV2 composer 输入栏
// ------------------------------------------------------------
// - 居中线程宽度容器（max-w-thread）
// - 圆角 toolbar-radius 实心 composer 卡片（inspector 底 + soft 阴影）
// - 输入区上方 chips 行（引用 chips + 模型 chip）
// - 底部按钮行：左组（+ 附件 / 技能 / MCP / 对话控制）│
//                右组（上下文环 / 模型选择 / 语音 / 黑色圆形发送）

import { useChatStore } from "@/state/chat";
import { cn } from "@/lib/utils";
import { useEffect, useRef, useState } from "react";
import {
  Plus,
  Lightning,
  Wrench,
  SlidersHorizontal,
  ArrowUp,
  Square,
  Paperclip,
  LinkSimple,
  Microphone,
} from "@phosphor-icons/react";
import { Tooltip, TooltipContent, TooltipTrigger } from "@/components/ui/Tooltip";

export function InputBar() {
  const sendMessage = useChatStore((s) => s.sendMessage);
  const loading = useChatStore((s) => s.loading);
  const refs = useChatStore((s) => s.refs);
  const removeRef = useChatStore((s) => s.removeRef);
  const activeSessionId = useChatStore((s) => s.activeSessionId);
  const sessions = useChatStore((s) => s.sessions);

  const [text, setText] = useState("");
  const [recording, setRecording] = useState(false);
  const [transcribing, setTranscribing] = useState(false);
  const taRef = useRef<HTMLTextAreaElement>(null);
  const mediaRecorder = useRef<MediaRecorder | null>(null);
  const audioChunks = useRef<Blob[]>([]);

  const activeSession = sessions.find((s) => s.id === activeSessionId);

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
    if (e.key === "Enter" && !e.shiftKey && !e.nativeEvent.isComposing) {
      e.preventDefault();
      handleSend();
    }
  };

  // —— 语音输入（Web Audio → WAV → VoiceTranscribe）——
  const toggleVoice = async () => {
    if (recording) {
      mediaRecorder.current?.stop();
      setRecording(false);
      return;
    }
    try {
      const stream = await navigator.mediaDevices.getUserMedia({ audio: true });
      const rec = new MediaRecorder(stream);
      audioChunks.current = [];
      rec.ondataavailable = (e) => audioChunks.current.push(e.data);
      rec.onstop = async () => {
        stream.getTracks().forEach((t) => t.stop());
        const blob = new Blob(audioChunks.current, { type: "audio/webm" });
        setTranscribing(true);
        try {
          // 转 wav：webm 直接上传（后端按 mime 处理）
          const buf = await blob.arrayBuffer();
          const bytes = Array.from(new Uint8Array(buf));
          // 通过 wails 绑定调用（若无绑定则忽略）
          const go = (window as unknown as { go?: { deepstudent?: { App?: Record<string, (...a: unknown[]) => unknown> }; main?: { App?: Record<string, (...a: unknown[]) => unknown> } } }).go;
          const app = go?.deepstudent?.App ?? go?.main?.App;
          const res = app?.VoiceTranscribe
            ? ((await app.VoiceTranscribe(bytes, "audio/webm")) as { text?: string })
            : null;
          if (res?.text) {
            setText((prev) => (prev ? prev + " " : "") + res.text);
            taRef.current?.focus();
          }
        } finally {
          setTranscribing(false);
        }
      };
      rec.start();
      mediaRecorder.current = rec;
      setRecording(true);
    } catch {
      setRecording(false);
    }
  };

  const placeholder = activeSession
    ? `向 ${activeSession.provider} 发送消息…`
    : "输入消息开始新会话…";

  return (
    <div className="shrink-0 border-t border-[var(--shell-seam)] bg-[var(--shell-workspace-panel)] px-6 py-3">
      <div className="mx-auto" style={{ maxWidth: "var(--chat-thread-max-w)" }}>
        {/* —— chips 行 —— */}
        {(refs.length > 0 || activeSession) && (
          <div className="mb-2 flex flex-wrap items-center gap-1.5">
            {activeSession && (
              <Chip icon={<Lightning size={11} weight="fill" />} text={activeSession.model} />
            )}
            {refs.map((r) => (
              <Chip key={r} icon={<LinkSimple size={11} />} text={r} onRemove={() => removeRef(r)} />
            ))}
          </div>
        )}

        {/* —— composer 卡片 —— */}
        <div
          className="rounded-[var(--radius-shell-toolbar)] border border-[var(--border-default)] bg-[var(--shell-inspector-panel)] p-2 shadow-soft transition-colors focus-within:border-[var(--primary-color)]/40"
        >
          {/* 输入区 */}
          <div className="flex items-end gap-2 px-1">
            <textarea
              ref={taRef}
              value={text}
              onChange={(e) => setText(e.target.value)}
              onKeyDown={handleKeyDown}
              placeholder={placeholder}
              rows={1}
              className="flex-1 resize-none bg-transparent px-1 py-1.5 text-[14px] text-foreground placeholder:text-muted-foreground focus:outline-none"
              style={{ minHeight: "32px", maxHeight: "200px" }}
            />
          </div>

          {/* 底部按钮行 */}
          <div className="mt-1 flex items-center justify-between px-1">
            {/* 左组 */}
            <div className="flex items-center gap-0.5">
              <BarButton title="附加（上传 / 资源库 / 相机）"><Plus size={16} /></BarButton>
              <BarButton title="技能" active={false}><Lightning size={15} /></BarButton>
              <BarButton title="MCP 工具"><Wrench size={15} /></BarButton>
              <BarButton title="对话控制"><SlidersHorizontal size={15} /></BarButton>
            </div>

            {/* 右组 */}
            <div className="flex items-center gap-1">
              <BarButton
                title={recording ? "停止并转写" : "语音输入"}
                active={recording}
                danger={recording}
                disabled={transcribing}
                onClick={() => void toggleVoice()}
              >
                {transcribing ? <Spinner /> : recording ? <Square size={13} weight="fill" /> : <Microphone size={16} />}
              </BarButton>
              <Tooltip>
                <TooltipTrigger asChild>
                  <button
                    onClick={handleSend}
                    disabled={(!text.trim() && !recording) || loading}
                    className="flex h-8 w-8 items-center justify-center rounded-full bg-black text-white transition-colors hover:bg-black/80 disabled:opacity-30 dark:bg-white dark:text-black dark:hover:bg-white/80"
                    aria-label="发送"
                  >
                    {loading ? <Square size={13} weight="fill" /> : <ArrowUp size={15} weight="bold" />}
                  </button>
                </TooltipTrigger>
                <TooltipContent side="top">发送</TooltipContent>
              </Tooltip>
            </div>
          </div>
        </div>

        {/* 底部提示 */}
        <div className="mt-1.5 flex items-center justify-between text-[10px] text-muted-foreground/50">
          <span>
            <kbd className="font-mono">Enter</kbd> 发送 ·{" "}
            <kbd className="font-mono">Shift+Enter</kbd> 换行
          </span>
          {recording && <span className="animate-pulse text-destructive">● 录音中，点击停止转写</span>}
          {loading && <span className="animate-pulse">生成中…</span>}
        </div>
      </div>
    </div>
  );
}

// —— chip ——
function Chip({ icon, text, onRemove }: { icon: React.ReactNode; text: string; onRemove?: () => void }) {
  return (
    <span className="flex items-center gap-1 rounded-full border border-[var(--border-default)] bg-[var(--shell-inspector-panel)] px-2 py-0.5 text-[11px] text-foreground/90">
      <span className="text-primary">{icon}</span>
      <span className="max-w-40 truncate">{text}</span>
      {onRemove && (
        <button onClick={onRemove} className="text-muted-foreground hover:text-foreground">
          <Paperclip size={9} className="hidden" />
          <span className="text-[10px]">×</span>
        </button>
      )}
    </span>
  );
}

// —— 底部工具栏按钮 ——
function BarButton({
  title,
  children,
  active = false,
  danger = false,
  disabled = false,
  onClick,
}: {
  title: string;
  children: React.ReactNode;
  active?: boolean;
  danger?: boolean;
  disabled?: boolean;
  onClick?: () => void;
}) {
  return (
    <Tooltip>
      <TooltipTrigger asChild>
        <button
          onClick={onClick}
          disabled={disabled}
          className={cn(
            "flex h-8 w-8 items-center justify-center rounded-md transition-colors",
            danger
              ? "bg-destructive/15 text-destructive animate-pulse"
              : active
                ? "bg-primary/15 text-primary"
                : "text-muted-foreground hover:bg-[var(--interactive-hover)] hover:text-foreground",
            disabled && "opacity-40"
          )}
        >
          {children}
        </button>
      </TooltipTrigger>
      <TooltipContent side="top">{title}</TooltipContent>
    </Tooltip>
  );
}

// —— 小转圈 ——
function Spinner() {
  return (
    <span className="inline-block h-3.5 w-3.5 animate-spin rounded-full border-2 border-muted-foreground/30 border-t-primary" />
  );
}
