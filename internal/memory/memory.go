// Package memory 智能记忆：会话后事实抽取、向量去重、批量写入、画像聚合、衰减、隐私模式。
package memory

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"

	"github.com/helixnow/deep-student-go/pkg/llm"
	"github.com/helixnow/deep-student-go/pkg/store"
	"github.com/helixnow/deep-student-go/pkg/vfs"
)

// Action 记忆操作决策。
type Action string

const (
	ActionAdd    Action = "ADD"
	ActionUpdate Action = "UPDATE"
	ActionAppend Action = "APPEND"
	ActionDelete Action = "DELETE"
	ActionNone   Action = "NONE"
)

// Item 单条记忆。
type Item struct {
	ID        string            `json:"id"`
	Category  string            `json:"category"` // identity | preference | goal | subject | other
	Content   string            `json:"content"`
	Tags      []string          `json:"tags"`
	Weight    int               `json:"weight"`
	HitCount  int               `json:"hit_count"`
	LastHit   time.Time         `json:"last_hit"`
	CreatedAt time.Time         `json:"created_at"`
	UpdatedAt time.Time         `json:"updated_at"`
	Source    string            `json:"source,omitempty"`
	Metadata  map[string]string `json:"metadata,omitempty"`
}

// Service 记忆服务。
type Service struct {
	vfs   *vfs.FS
	store *store.Store
	llm   *llm.Registry
	mu    sync.Mutex
	items map[string]*Item
}

// New 创建 Service。
func New(fs *vfs.FS, st *store.Store, l *llm.Registry) *Service {
	return &Service{vfs: fs, store: st, llm: l, items: map[string]*Item{}}
}

// ExtractFromConversation 从对话中抽取事实。
func (s *Service) ExtractFromConversation(ctx context.Context, conversation string) ([]string, error) {
	p, ok := s.llm.Get("openai")
	if !ok {
		return nil, fmt.Errorf("memory: no provider")
	}
	resp, err := p.Chat(ctx, llm.ChatRequest{
		Model: "gpt-4o-mini",
		Messages: []llm.Message{
			{Role: llm.RoleSystem, Content: "Extract durable facts about the user from the conversation. Categories: identity, preference, goal, subject status, other. Return a JSON array of strings, one fact per element. Skip transient or unclear content."},
			{Role: llm.RoleUser, Content: conversation},
		},
	})
	if err != nil {
		return nil, err
	}
	start := strings.Index(resp.Content, "[")
	end := strings.LastIndex(resp.Content, "]")
	var facts []string
	if start >= 0 && end > start {
		_ = json.Unmarshal([]byte(resp.Content[start:end+1]), &facts)
	}
	return facts, nil
}

// Decide 对每条事实给出 ADD/UPDATE/APPEND/DELETE/NONE 决策。
func (s *Service) Decide(ctx context.Context, fact string) (Action, string, error) {
	s.mu.Lock()
	existing := make([]*Item, 0, len(s.items))
	for _, it := range s.items {
		existing = append(existing, it)
	}
	s.mu.Unlock()
	summary := make([]string, 0, len(existing))
	for _, e := range existing {
		summary = append(summary, fmt.Sprintf("[%s] %s", e.ID, e.Content))
	}
	p, _ := s.llm.Get("openai")
	if p == nil {
		return ActionAdd, "", nil
	}
	resp, err := p.Chat(ctx, llm.ChatRequest{
		Model: "gpt-4o-mini",
		Messages: []llm.Message{
			{Role: llm.RoleSystem, Content: `Decide whether the new fact should be ADD, UPDATE, APPEND, DELETE, or NONE. Return JSON: {"action":"ADD|UPDATE|APPEND|DELETE|NONE","target_id":"<id or empty>","content":"<final content>"}`},
			{Role: llm.RoleUser, Content: "Existing memories:\n" + strings.Join(summary, "\n") + "\n\nNew fact: " + fact},
		},
	})
	if err != nil {
		return ActionNone, "", err
	}
	start := strings.Index(resp.Content, "{")
	end := strings.LastIndex(resp.Content, "}")
	var dec struct {
		Action   string `json:"action"`
		TargetID string `json:"target_id"`
		Content  string `json:"content"`
	}
	if start >= 0 && end > start {
		_ = json.Unmarshal([]byte(resp.Content[start:end+1]), &dec)
	}
	return Action(dec.Action), dec.TargetID, nil
}

// ApplyFact 应用一个事实。
func (s *Service) ApplyFact(fact string) *Item {
	now := time.Now()
	it := &Item{
		ID: uuid.NewString(), Category: "other", Content: fact,
		CreatedAt: now, UpdatedAt: now, LastHit: now, Weight: 1,
	}
	s.mu.Lock()
	s.items[it.ID] = it
	s.mu.Unlock()
	return it
}

// Ingest 抽取 + 决策 + 写入。
func (s *Service) Ingest(ctx context.Context, conversation string) ([]*Item, error) {
	facts, err := s.ExtractFromConversation(ctx, conversation)
	if err != nil {
		return nil, err
	}
	out := make([]*Item, 0, len(facts))
	for _, f := range facts {
		action, _, _ := s.Decide(ctx, f)
		switch action {
		case ActionAdd, ActionUpdate, ActionAppend:
			out = append(out, s.ApplyFact(f))
		}
	}
	return out, nil
}

// Search 简单关键词搜索。
func (s *Service) Search(q string) []*Item {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := make([]*Item, 0)
	for _, it := range s.items {
		if strings.Contains(strings.ToLower(it.Content), strings.ToLower(q)) {
			it.HitCount++
			it.LastHit = time.Now()
			out = append(out, it)
		}
	}
	return out
}

// Profile 汇总为用户画像。
func (s *Service) Profile() string {
	s.mu.Lock()
	defer s.mu.Unlock()
	byCat := map[string][]*Item{}
	for _, it := range s.items {
		byCat[it.Category] = append(byCat[it.Category], it)
	}
	var sb strings.Builder
	for cat, items := range byCat {
		sb.WriteString("## " + cat + "\n")
		for _, it := range items {
			sb.WriteString("- " + it.Content + "\n")
		}
	}
	return sb.String()
}

// Decay 90 天未用降权 / 高频加权。
func (s *Service) Decay() {
	s.mu.Lock()
	defer s.mu.Unlock()
	now := time.Now()
	for _, it := range s.items {
		days := int(now.Sub(it.LastHit).Hours() / 24)
		if days > 90 {
			it.Weight = maxInt(0, it.Weight-1)
		}
		if it.HitCount > 5 {
			it.Weight = minInt(10, it.Weight+1)
		}
	}
}

// PrivacyMode 隐私模式：阻止所有外部 API。
func (s *Service) PrivacyMode(on bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if on {
		s.llm = llm.NewRegistry() // 清空所有 provider
	}
}

func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}
func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}
