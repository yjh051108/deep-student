// Package quickassist 提供轻量问答快速助手（Quick Assistant）。
//
// 对齐 Rust 原版 src-tauri/src/quick_assistant.rs：独立小窗口、轻量模型、
// 会话不落库（或保留最近会话）。支持常见学习问答与工具触发。

package quickassist

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/helixnow/deep-student-go/pkg/llm"
	"github.com/helixnow/deep-student-go/pkg/store"
)

// Session 快速助手会话（轻量，保留最近 N 条）。
type Session struct {
	ID        string    `json:"id"`
	Messages  []Message `json:"messages"`
	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
}

// Message 快速助手消息。
type Message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
	At      time.Time `json:"at"`
}

// Service 快速助手服务。
type Service struct {
	llm     *llm.Registry
	store   *store.Store
	mu      sync.Mutex
	session *Session
}

// New 构造服务。
func New(l *llm.Registry, st *store.Store) *Service {
	return &Service{
		llm:     l,
		store:   st,
		session: &Session{ID: "quick", CreatedAt: time.Now()},
	}
}

// Ask 提问并返回回答（单轮，保留最近 8 条上下文）。
func (s *Service) Ask(ctx context.Context, question string) (string, error) {
	if strings.TrimSpace(question) == "" {
		return "", errors.New("quickassist: empty question")
	}
	p, ok := s.llm.Get("openai")
	if !ok {
		return "", errors.New("quickassist: no provider")
	}
	s.mu.Lock()
	// 追加问题
	s.session.Messages = append(s.session.Messages, Message{Role: "user", Content: question, At: time.Now()})
	// 构造上下文（最近 8 条）
	msgs := []llm.Message{{Role: llm.RoleSystem, Content: "You are a quick study assistant. Answer concisely."}}
	start := 0
	if len(s.session.Messages) > 8 {
		start = len(s.session.Messages) - 8
	}
	for _, m := range s.session.Messages[start:] {
		msgs = append(msgs, llm.Message{Role: llm.Role(m.Role), Content: m.Content})
	}
	s.mu.Unlock()

	resp, err := p.Chat(ctx, llm.ChatRequest{
		Model:       "gpt-4o-mini",
		Messages:    msgs,
		Temperature: 0.4,
		MaxTokens:   600,
	})
	if err != nil {
		return "", err
	}
	answer := resp.Content
	s.mu.Lock()
	s.session.Messages = append(s.session.Messages, Message{Role: "assistant", Content: answer, At: time.Now()})
	s.session.UpdatedAt = time.Now()
	s.mu.Unlock()
	return answer, nil
}

// History 返回最近会话（脱敏后全部）。
func (s *Service) History(limit int) []Message {
	s.mu.Lock()
	defer s.mu.Unlock()
	if limit <= 0 || limit >= len(s.session.Messages) {
		return s.session.Messages
	}
	return s.session.Messages[len(s.session.Messages)-limit:]
}

// Clear 清空会话。
func (s *Service) Clear() {
	s.mu.Lock()
	s.session.Messages = nil
	s.session.UpdatedAt = time.Now()
	s.mu.Unlock()
}

// Summary 会话摘要。
func (s *Service) Summary() map[string]any {
	s.mu.Lock()
	defer s.mu.Unlock()
	return map[string]any{
		"messageCount": len(s.session.Messages),
		"updatedAt":    s.session.UpdatedAt,
	}
}

var _ = fmt.Sprintf
