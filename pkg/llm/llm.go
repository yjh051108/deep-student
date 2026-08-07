// Package llm 提供统一 LLM Provider 适配（Chat/Embed/Stream）。
package llm

import (
	"context"
	"errors"
	"io"
)

// Role 消息角色。
type Role string

const (
	RoleSystem    Role = "system"
	RoleUser      Role = "user"
	RoleAssistant Role = "assistant"
	RoleTool      Role = "tool"
)

// Message 通用消息。
type Message struct {
	Role       Role             `json:"role"`
	Content    string           `json:"content,omitempty"`
	Name       string           `json:"name,omitempty"`
	ToolCallID string           `json:"tool_call_id,omitempty"`
	ToolCalls  []ToolCall       `json:"tool_calls,omitempty"`
	MultiModal []MultiModalPart `json:"multimodal,omitempty"`
}

// MultiModalPart 多模态分片（图像、PDF 等）。
type MultiModalPart struct {
	Type     string `json:"type"` // image | file | audio
	MIME     string `json:"mime"`
	Data     []byte `json:"data,omitempty"`
	URL      string `json:"url,omitempty"`
	Filename string `json:"filename,omitempty"`
}

// ToolCall 工具调用。
type ToolCall struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	Arguments string `json:"arguments"` // JSON
}

// Tool 工具声明。
type Tool struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	Parameters  any    `json:"parameters"` // JSON Schema
}

// ChatRequest 聊天请求。
type ChatRequest struct {
	Model       string            `json:"model"`
	Messages    []Message         `json:"messages"`
	Tools       []Tool            `json:"tools,omitempty"`
	MaxTokens   int               `json:"max_tokens,omitempty"`
	Temperature float32           `json:"temperature,omitempty"`
	TopP        float32           `json:"top_p,omitempty"`
	Stop        []string          `json:"stop,omitempty"`
	Stream      bool              `json:"stream"`
	Metadata    map[string]string `json:"metadata,omitempty"`
}

// Chunk 流式响应。
type Chunk struct {
	Delta    string    `json:"delta,omitempty"`
	Reason   string    `json:"reason,omitempty"` // 思考链
	Done     bool      `json:"done"`
	Err      error     `json:"-"`
	Usage    Usage     `json:"usage"`
	ToolCall *ToolCall `json:"tool_call,omitempty"`
}

// ChatResponse 非流式响应。
type ChatResponse struct {
	Content   string
	Reasoning string
	ToolCalls []ToolCall
	Usage     Usage
}

// Usage token 计量。
type Usage struct {
	PromptTokens     int `json:"prompt_tokens"`
	CompletionTokens int `json:"completion_tokens"`
	TotalTokens      int `json:"total_tokens"`
}

// EmbedRequest 嵌入请求。
type EmbedRequest struct {
	Model string   `json:"model"`
	Input []string `json:"input"`
}

// EmbedResponse 嵌入响应。
type EmbedResponse struct {
	Embeddings [][]float32 `json:"embeddings"`
	Usage      Usage       `json:"usage"`
}

// Provider LLM 供应商接口。
type Provider interface {
	Name() string
	Chat(ctx context.Context, req ChatRequest) (*ChatResponse, error)
	Stream(ctx context.Context, req ChatRequest) (<-chan Chunk, error)
	Embed(ctx context.Context, req EmbedRequest) (*EmbedResponse, error)
}

// Registry Provider 注册表。
type Registry struct {
	providers map[string]Provider
}

// NewRegistry 创建注册表。
func NewRegistry() *Registry { return &Registry{providers: map[string]Provider{}} }

// Register 注册一个 Provider。
func (r *Registry) Register(p Provider) { r.providers[p.Name()] = p }

// Get 获取 Provider。
func (r *Registry) Get(name string) (Provider, bool) {
	p, ok := r.providers[name]
	return p, ok
}

// Names 列出所有 provider 名称。
func (r *Registry) Names() []string {
	out := make([]string, 0, len(r.providers))
	for k := range r.providers {
		out = append(out, k)
	}
	return out
}

// ErrNotFound 未找到。
var ErrNotFound = errors.New("llm: provider not found")

// ErrUnsupported 不支持。
var ErrUnsupported = errors.New("llm: unsupported")

// NopReader 用于 io.Discard 等价。
type NopReader struct{}

func (NopReader) Read(p []byte) (int, error) { return 0, io.EOF }
