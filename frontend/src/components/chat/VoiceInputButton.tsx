// VoiceInputButton —— 语音输入按钮（麦克风）
// ------------------------------------------------------------
// 点击开始录音（Web Audio）→ 再点停止 → 上传转写 → 回调文本。
// 对接 voiceApi（VoiceTranscribe）。

import { useRef, useState } from "react";
import { voiceApi, VoiceRecorder } from "@/lib/voice";
import { Mic, MicOff, Loader2 } from "lucide-react";
import { cn } from "@/lib/utils";
import { Tooltip, TooltipContent, TooltipTrigger } from "@/components/ui/Tooltip";

export function VoiceInputButton({
  onText,
  disabled,
}: {
  onText: (text: string) => void;
  disabled?: boolean;
}) {
  const [recording, setRecording] = useState(false);
  const [busy, setBusy] = useState(false);
  const recorderRef = useRef<VoiceRecorder | null>(null);
  const supported = typeof navigator !== "undefined" && !!navigator.mediaDevices?.getUserMedia;

  const toggle = async () => {
    if (recording) {
      // 停止并转写
      const wav = recorderRef.current?.stop();
      setRecording(false);
      if (!wav) return;
      setBusy(true);
      try {
        const bytes = Array.from(new Uint8Array(wav));
        const res = await voiceApi.transcribe(bytes, "audio/wav");
        if (res?.text) onText(res.text);
      } finally {
        setBusy(false);
      }
    } else {
      try {
        const r = new VoiceRecorder();
        await r.start();
        recorderRef.current = r;
        setRecording(true);
      } catch {
        // 无麦克风权限等
        setRecording(false);
      }
    }
  };

  if (!supported) return null;

  return (
    <Tooltip>
      <TooltipTrigger asChild>
        <button
          type="button"
          onClick={() => void toggle()}
          disabled={disabled || busy}
          className={cn(
            "flex h-8 w-8 shrink-0 items-center justify-center rounded-md transition-colors",
            recording
              ? "animate-pulse bg-destructive/20 text-destructive"
              : "text-muted-foreground hover:bg-accent hover:text-foreground"
          )}
          title={recording ? "点击停止并转写" : "语音输入"}
        >
          {busy ? (
            <Loader2 size={15} className="animate-spin" />
          ) : recording ? (
            <MicOff size={15} />
          ) : (
            <Mic size={15} />
          )}
        </button>
      </TooltipTrigger>
      <TooltipContent side="top">
        {recording ? "停止并转写" : "语音输入（需要麦克风）"}
      </TooltipContent>
    </Tooltip>
  );
}
