// Package ocr 提供多引擎 OCR 能力：DeepSeek-VL（API）/ 系统 OCR（Windows）/
// 占位 PaddleOCR。支持引擎优先级管理与 PDF 整卷识别（前端逐页上传）。
//
// 对齐 Rust 原版 src-tauri/src/ocr_adapters/ 与 cmd/ocr.rs 的接口面。

package ocr

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"strings"
	"time"
)

// EngineType OCR 引擎类型。
type EngineType string

const (
	EngineDeepSeek   EngineType = "deepseek_vl"
	EnginePaddle     EngineType = "paddle"
	EngineSystem     EngineType = "system"
)

// EngineInfo 引擎信息。
type EngineInfo struct {
	Type        EngineType `json:"type"`
	Name        string     `json:"name"`
	Available   bool       `json:"available"`
	IsDefault   bool       `json:"isDefault"`
	Priority    int        `json:"priority"`
	Description string     `json:"description"`
}

// Service OCR 服务。
type Service struct {
	engineType EngineType
	// DeepSeek-VL 配置（OpenAI 兼容 chat/completions，多模态 image_url）
	APIKey   string
	BaseURL  string
	Model    string
	Thinking bool
	client   *http.Client
	sessions []string
}

// New 构造 OCR 服务。默认 DeepSeek-VL；apiKey 为空时可用系统 OCR。
func New(apiKey string) *Service {
	return &Service{
		engineType: EngineDeepSeek,
		APIKey:     apiKey,
		BaseURL:    "https://api.deepseek.com",
		Model:      "deepseek-vl2",
		Thinking:   false,
		client:     &http.Client{Timeout: 120 * time.Second},
	}
}

// SetEngineType 切换引擎。
func (s *Service) SetEngineType(t EngineType) error {
	switch t {
	case EngineDeepSeek, EnginePaddle, EngineSystem:
		s.engineType = t
		return nil
	}
	return fmt.Errorf("ocr: unknown engine %q", t)
}

// EngineType 返回当前引擎。
func (s *Service) EngineType() EngineType { return s.engineType }

// SetThinking 开关 VL 推理模式。
func (s *Service) SetThinking(on bool) { s.Thinking = on }

// SetConfig 配置 VL API。
func (s *Service) SetConfig(apiKey, baseURL, model string) {
	if apiKey != "" {
		s.APIKey = apiKey
	}
	if baseURL != "" {
		s.BaseURL = strings.TrimSuffix(baseURL, "/")
	}
	if model != "" {
		s.Model = model
	}
}

// ListEngines 返回引擎列表及可用性。
func (s *Service) ListEngines() []EngineInfo {
	return []EngineInfo{
		{Type: EngineDeepSeek, Name: "DeepSeek-VL", Available: s.APIKey != "", IsDefault: s.engineType == EngineDeepSeek, Priority: 1, Description: "视觉语言模型 OCR（API）"},
		{Type: EngineSystem, Name: "系统 OCR (Windows)", Available: true, IsDefault: s.engineType == EngineSystem, Priority: 2, Description: "Windows.Media.Ocr 引擎（离线）"},
		{Type: EnginePaddle, Name: "PaddleOCR", Available: false, IsDefault: s.engineType == EnginePaddle, Priority: 3, Description: "本地 PaddleOCR（需安装，占位）"},
	}
}

// OcrResult OCR 结果。
type OcrResult struct {
	Text     string     `json:"text"`
	Engine   EngineType `json:"engine"`
	Lines    []OcrLine  `json:"lines,omitempty"`
	Duration int64      `json:"durationMs"`
}

// OcrLine 单行识别结果（含置信度与框）。
type OcrLine struct {
	Text       string    `json:"text"`
	Confidence float64   `json:"confidence"`
	Box        [4][2]int `json:"box,omitempty"`
}

// Recognize 识别图片（imageData 为图片字节，mime 如 image/png）。
func (s *Service) Recognize(ctx context.Context, imageData []byte, mime string) (*OcrResult, error) {
	switch s.engineType {
	case EngineDeepSeek:
		return s.recognizeVL(ctx, imageData, mime)
	case EngineSystem:
		return s.recognizeSystem(ctx, imageData, mime)
	case EnginePaddle:
		return nil, errors.New("ocr: paddle engine not installed")
	}
	return nil, errors.New("ocr: unknown engine")
}

// recognizeVL 调用 DeepSeek-VL（OpenAI 兼容多模态）。
func (s *Service) recognizeVL(ctx context.Context, imageData []byte, mime string) (*OcrResult, error) {
	if s.APIKey == "" {
		return nil, errors.New("ocr: deepseek api key not configured")
	}
	start := time.Now()
	b64 := base64.StdEncoding.EncodeToString(imageData)
	dataURL := "data:" + mime + ";base64," + b64
	system := "You are an OCR engine. Extract ALL text from the image verbatim. " +
		"Return plain text only, preserving reading order and line breaks. " +
		"Do not add commentary."
	msgs := []map[string]any{
		{"role": "system", "content": system},
		{"role": "user", "content": []map[string]any{
			{"type": "image_url", "image_url": map[string]string{"url": dataURL}},
			{"type": "text", "text": "请识别图中所有文字，按阅读顺序输出，只输出文本。"},
		}},
	}
	body, _ := json.Marshal(map[string]any{
		"model": s.Model, "messages": msgs, "temperature": 0.1,
	})
	endpoint := s.BaseURL + "/chat/completions"
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+s.APIKey)
	resp, err := s.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("ocr: vl request: %w", err)
	}
	defer resp.Body.Close()
	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("ocr: vl %s: %s", resp.Status, truncate(string(data), 300))
	}
	var parsed struct {
		Choices []struct {
			Message struct {
				Content string `json:"content"`
			} `json:"message"`
		} `json:"choices"`
	}
	if err := json.Unmarshal(data, &parsed); err != nil {
		return nil, fmt.Errorf("ocr: vl parse: %w", err)
	}
	if len(parsed.Choices) == 0 {
		return nil, errors.New("ocr: empty vl response")
	}
	return &OcrResult{
		Text:     strings.TrimSpace(parsed.Choices[0].Message.Content),
		Engine:   EngineDeepSeek,
		Duration: time.Since(start).Milliseconds(),
	}, nil
}

// recognizeSystem 系统 OCR（Windows.Media.Ocr，通过 powershell 调用）。
// 说明：Windows.Media.Ocr 在 PowerShell 中不可直接调用（需 UWP/COM 互操作），
// 此实现为接口占位 + 明确错误提示；可扩展为 winrt 绑定或调用外部工具。
func (s *Service) recognizeSystem(ctx context.Context, imageData []byte, mime string) (*OcrResult, error) {
	// TODO(扩展): 通过 winrt (go-winrt) 绑定 Windows.Media.Ocr，或用
	// Windows 10+ 自带 PowerShell OCR (WinRT) 桥接。
	// 当前返回结构化错误，保持接口兼容。
	_ = imageData
	_ = mime
	return nil, errors.New("ocr: system engine not yet wired (use DeepSeek-VL)")
}

// PDF 整卷识别：前端逐页上传（OpenAI 兼容 OCR 需要逐页图片）。

// StartPDFSession 启动一次 PDF OCR 会话。
func (s *Service) StartPDFSession(pdfName string, pageCount int) (string, error) {
	sessionID := fmt.Sprintf("pdf-ocr-%d", time.Now().UnixNano())
	s.sessions = append(s.sessions, sessionID)
	// 会话状态存内存（简化）；大文件场景可落盘。
	_ = pdfName
	_ = pageCount
	return sessionID, nil
}

// UploadPage 上传一页图片并识别（流式会话）。
func (s *Service) UploadPage(ctx context.Context, sessionID string, pageIndex int, imageData []byte, mime string) (string, error) {
	res, err := s.Recognize(ctx, imageData, mime)
	if err != nil {
		return "", err
	}
	_ = sessionID
	_ = pageIndex
	return res.Text, nil
}

// CancelPDFSession 取消会话。
func (s *Service) CancelPDFSession(sessionID string) {
	for i, id := range s.sessions {
		if id == sessionID {
			s.sessions = append(s.sessions[:i], s.sessions[i+1:]...)
			return
		}
	}
}

// ExtractTextFromPDFBytes 便捷方法：PDF 字节 → 若内部含文本层直接提取。
// （完整 PDF 整卷 OCR 由前端逐页图片化后调用 UploadPage。）
func (s *Service) ExtractTextFromPDFBytes(data []byte) (string, error) {
	// 复用 reader.ParsePDF 的文本层提取
	return extractPDFText(data)
}

// multipartField 保留 multipart 依赖（供扩展）。
var _ = multipart.NewWriter

// extractPDFText 提取 PDF 文本层（复用 reader 包解析能力）。
func extractPDFText(data []byte) (string, error) {
	pages, err := parsePDFPages(data)
	if err != nil {
		return "", err
	}
	var sb strings.Builder
	for i, p := range pages {
		if i > 0 {
			sb.WriteString("\n")
		}
		sb.WriteString(p)
	}
	return sb.String(), nil
}

// truncate 截断长字符串。
func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "..."
}
