// Package voiceinput 提供语音输入转写（ASR）。
//
// 对齐 Rust 原版 src-tauri/src/voice_input.rs（voice_input_transcribe 命令）：
// 录音（前端 Web Audio）→ base64 → 后端调 ASR 端点 → 返回文本。
// 默认使用 OpenAI 兼容的 audio/transcriptions 端点（SiliconFlow TeleSpeech、
// OpenAI Whisper 等均兼容）；可配置 provider 与 base_url。
package voiceinput

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"strings"
	"time"
)

// Provider 配置。
type Provider struct {
	Name    string `json:"name"`
	BaseURL string `json:"baseURL"` // 兼容端点根，如 https://api.siliconflow.cn/v1
	APIKey  string `json:"apiKey"`
	Model   string `json:"model"`
}

// DefaultProvider 默认 ASR 供应商（SiliconFlow TeleSpeech）。
func DefaultProvider(apiKey string) Provider {
	return Provider{
		Name:    "siliconflow",
		BaseURL: "https://api.siliconflow.cn/v1",
		APIKey:  apiKey,
		Model:   "FunAudioLLM/SenseVoiceSmall",
	}
}

// TranscribeResult 转写结果。
type TranscribeResult struct {
	Text      string `json:"text"`
	Provider  string `json:"provider"`
	Model     string `json:"model"`
	Duration  int64  `json:"durationMs"`
}

// Service 语音输入服务。
type Service struct {
	provider Provider
	client   *http.Client
}

// New 构造服务。默认 SiliconFlow；apiKey 为空时转写返回明确错误。
func New(apiKey string) *Service {
	return &Service{
		provider: DefaultProvider(apiKey),
		client:   &http.Client{Timeout: 60 * time.Second},
	}
}

// NewWithProvider 使用自定义 provider。
func NewWithProvider(p Provider) *Service {
	return &Service{provider: p, client: &http.Client{Timeout: 60 * time.Second}}
}

// SetProvider 运行时切换 provider。
func (s *Service) SetProvider(p Provider) { s.provider = p }

// Provider 返回当前 provider 配置。
func (s *Service) Provider() Provider { return s.provider }

// Transcribe 转写音频。audioData 为原始音频字节（wav/mp3/webm），
// mime 如 "audio/wav"、"audio/mp3"。返回转写文本。
func (s *Service) Transcribe(ctx context.Context, audioData []byte, mime string) (*TranscribeResult, error) {
	if len(audioData) == 0 {
		return nil, fmt.Errorf("voiceinput: empty audio")
	}
	if s.provider.APIKey == "" {
		return nil, fmt.Errorf("voiceinput: no API key configured")
	}
	start := time.Now()
	endpoint := strings.TrimSuffix(s.provider.BaseURL, "/") + "/audio/transcriptions"

	// multipart/form-data：file + model
	var buf bytes.Buffer
	w := multipart.NewWriter(&buf)
	ext := mimeExt(mime)
	fw, err := w.CreateFormFile("file", "recording"+ext)
	if err != nil {
		return nil, err
	}
	if _, err := fw.Write(audioData); err != nil {
		return nil, err
	}
	if err := w.WriteField("model", s.provider.Model); err != nil {
		return nil, err
	}
	_ = w.Close()

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, &buf)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", w.FormDataContentType())
	req.Header.Set("Authorization", "Bearer "+s.provider.APIKey)

	resp, err := s.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("voiceinput: request: %w", err)
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("voiceinput: %s: %s", resp.Status, truncate(string(body), 300))
	}
	var parsed struct {
		Text string `json:"text"`
	}
	if err := json.Unmarshal(body, &parsed); err != nil {
		return nil, fmt.Errorf("voiceinput: parse: %w", err)
	}
	text := strings.TrimSpace(parsed.Text)
	if text == "" {
		return nil, fmt.Errorf("voiceinput: empty transcription")
	}
	return &TranscribeResult{
		Text:     text,
		Provider: s.provider.Name,
		Model:    s.provider.Model,
		Duration: time.Since(start).Milliseconds(),
	}, nil
}

// mimeExt 根据 mime 类型给文件名扩展名。
func mimeExt(mime string) string {
	switch strings.ToLower(mime) {
	case "audio/mpeg", "audio/mp3":
		return ".mp3"
	case "audio/ogg":
		return ".ogg"
	case "audio/webm":
		return ".webm"
	case "audio/x-m4a", "audio/mp4":
		return ".m4a"
	default:
		return ".wav"
	}
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "..."
}
