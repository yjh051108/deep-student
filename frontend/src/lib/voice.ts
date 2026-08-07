// Voice 语音输入前端封装
// ------------------------------------------------------------
// 对接后端 voiceinput（Voice* 方法）。录音用 Web Audio API，
// 输出 wav → 转 base64 数组 → VoiceTranscribe。

import { callWails } from "@/lib/wails";

/** 转写结果 —— 与后端 voiceinput.TranscribeResult 对齐 */
export interface VoiceResult {
  text: string;
  provider: string;
  model: string;
  durationMs: number;
}

export const voiceApi = {
  transcribe: (audioData: number[], mime: string) =>
    callWails<VoiceResult>("VoiceTranscribe", audioData, mime),
};

// —— 录音工具（Web Audio → WAV） ——

/** WAV 编码（16-bit PCM mono） */
export function encodeWAV(samples: Float32Array, sampleRate: number): ArrayBuffer {
  const buffer = new ArrayBuffer(44 + samples.length * 2);
  const view = new DataView(buffer);

  const writeStr = (offset: number, s: string) => {
    for (let i = 0; i < s.length; i++) view.setUint8(offset + i, s.charCodeAt(i));
  };

  writeStr(0, "RIFF");
  view.setUint32(4, 36 + samples.length * 2, true);
  writeStr(8, "WAVE");
  writeStr(12, "fmt ");
  view.setUint32(16, 16, true);
  view.setUint16(20, 1, true); // PCM
  view.setUint16(22, 1, true); // mono
  view.setUint32(24, sampleRate, true);
  view.setUint32(28, sampleRate * 2, true);
  view.setUint16(32, 2, true);
  view.setUint16(34, 16, true);
  writeStr(36, "data");
  view.setUint32(40, samples.length * 2, true);

  let offset = 44;
  for (let i = 0; i < samples.length; i++) {
    const s = Math.max(-1, Math.min(1, samples[i]));
    view.setInt16(offset, s < 0 ? s * 0x8000 : s * 0x7fff, true);
    offset += 2;
  }
  return buffer;
}

/** 录音器：开始/停止返回 WAV ArrayBuffer */
export class VoiceRecorder {
  private stream: MediaStream | null = null;
  private ctx: AudioContext | null = null;
  private chunks: Float32Array[] = [];

  async start(): Promise<void> {
    this.stream = await navigator.mediaDevices.getUserMedia({ audio: true });
    this.ctx = new AudioContext();
    const source = this.ctx.createMediaStreamSource(this.stream);
    const processor = this.ctx.createScriptProcessor(4096, 1, 1);
    processor.onaudioprocess = (e) => {
      this.chunks.push(new Float32Array(e.inputBuffer.getChannelData(0)));
    };
    source.connect(processor);
    processor.connect(this.ctx.destination);
  }

  stop(): ArrayBuffer | null {
    this.stream?.getTracks().forEach((t) => t.stop());
    void this.ctx?.close();
    this.ctx = null;
    this.stream = null;
    if (this.chunks.length === 0) return null;
    // 拼接
    let total = 0;
    for (const c of this.chunks) total += c.length;
    const merged = new Float32Array(total);
    let offset = 0;
    for (const c of this.chunks) {
      merged.set(c, offset);
      offset += c.length;
    }
    this.chunks = [];
    return encodeWAV(merged, 16000);
  }
}
