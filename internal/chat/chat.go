// Package chat 多模态聊天：会话/分支/分组、多模型并行、子 Agent、深度思考。
package chat

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"

	"github.com/helixnow/deep-student-go/pkg/eventbus"
	"github.com/helixnow/deep-student-go/pkg/llm"
	"github.com/helixnow/deep-student-go/pkg/store"
	"github.com/helixnow/deep-student-go/pkg/vfs"
)

// Session 聊天会话。
type Session struct {
	ID         string    `json:"id"`
	GroupID    string    `json:"group_id"`
	Title      string    `json:"title"`
	BranchOf   string    `json:"branch_of,omitempty"`
	Tags       []string  `json:"tags"`
	Model      string    `json:"model"`
	Provider   string    `json:"provider"`
	Messages   []Message `json:"messages"`
	CreatedAt  time.Time `json:"created_at"`
	UpdatedAt  time.Time `json:"updated_at"`
	SystemHint string    `json:"system_hint,omitempty"`
}

// Message 单条消息。
type Message struct {
	ID        string    `json:"id"`
	Role      string    `json:"role"`
	Content   string    `json:"content"`
	Reasoning string    `json:"reasoning,omitempty"`
	Refs      []string  `json:"refs,omitempty"` // vfs:// 引用
	CreatedAt time.Time `json:"created_at"`
}

// Group 会话分组。
type Group struct {
	ID           string   `json:"id"`
	Name         string   `json:"name"`
	SystemHint   string   `json:"system_hint"`
	DefaultSkill string   `json:"default_skill"`
	Tags         []string `json:"tags"`
}

// Service 聊天服务。
type Service struct {
	vfs      *vfs.FS
	store    *store.Store
	llm      *llm.Registry
	bus      *eventbus.Bus
	mu       sync.RWMutex
	sessions map[string]*Session
	groups   map[string]*Group
}

// New 创建 Service。
func New(fs *vfs.FS, st *store.Store, l *llm.Registry, bus *eventbus.Bus) *Service {
	return &Service{
		vfs: fs, store: st, llm: l, bus: bus,
		sessions: map[string]*Session{},
		groups:   map[string]*Group{},
	}
}

// CreateGroup 创建分组。
func (s *Service) CreateGroup(name, systemHint, defaultSkill string, tags []string) *Group {
	g := &Group{ID: uuid.NewString(), Name: name, SystemHint: systemHint, DefaultSkill: defaultSkill, Tags: tags}
	s.mu.Lock()
	s.groups[g.ID] = g
	s.mu.Unlock()
	return g
}

// CreateSession 创建会话。
func (s *Service) CreateSession(groupID, title, model, provider string) *Session {
	now := time.Now()
	se := &Session{
		ID:        uuid.NewString(),
		GroupID:   groupID,
		Title:     title,
		Model:     model,
		Provider:  provider,
		CreatedAt: now,
		UpdatedAt: now,
		Tags:      []string{},
	}
	s.mu.Lock()
	s.sessions[se.ID] = se
	s.mu.Unlock()
	return se
}

// Branch 从某个消息点创建分支。
func (s *Service) Branch(parentID, atMessageID string) (*Session, error) {
	s.mu.RLock()
	parent, ok := s.sessions[parentID]
	s.mu.RUnlock()
	if !ok {
		return nil, fmt.Errorf("chat: parent not found")
	}
	now := time.Now()
	b := &Session{
		ID:        uuid.NewString(),
		GroupID:   parent.GroupID,
		Title:     parent.Title + " (branch)",
		BranchOf:  parent.ID,
		Model:     parent.Model,
		Provider:  parent.Provider,
		CreatedAt: now,
		UpdatedAt: now,
	}
	// 复制到 atMessageID 为止的消息
	for _, m := range parent.Messages {
		b.Messages = append(b.Messages, m)
		if m.ID == atMessageID {
			break
		}
	}
	s.mu.Lock()
	s.sessions[b.ID] = b
	s.mu.Unlock()
	return b, nil
}

// Send 发送消息并流式返回。
func (s *Service) Send(ctx context.Context, sessionID, content string, refs []string, deepThink bool) (<-chan string, error) {
	s.mu.RLock()
	se, ok := s.sessions[sessionID]
	s.mu.RUnlock()
	if !ok {
		return nil, fmt.Errorf("chat: session not found")
	}
	p, ok := s.llm.Get(se.Provider)
	if !ok {
		return nil, fmt.Errorf("chat: provider %s not registered", se.Provider)
	}

	// 构造多模态（把 refs 内容追加）
	prompt := content
	for _, r := range refs {
		data, _, err := s.vfs.Get(r)
		if err == nil {
			prompt += "\n\n[Reference " + r + "]\n" + string(data)
		}
	}

	userMsg := Message{ID: uuid.NewString(), Role: "user", Content: content, Refs: refs, CreatedAt: time.Now()}
	se.Messages = append(se.Messages, userMsg)
	se.UpdatedAt = time.Now()

	msgs := []llm.Message{}
	if se.SystemHint != "" {
		msgs = append(msgs, llm.Message{Role: llm.RoleSystem, Content: se.SystemHint})
	}
	if deepThink {
		msgs = append(msgs, llm.Message{Role: llm.RoleSystem, Content: "Please think step by step and show your reasoning."})
	}
	for _, m := range se.Messages {
		msgs = append(msgs, llm.Message{Role: llm.Role(m.Role), Content: m.Content})
	}
	req := llm.ChatRequest{Model: se.Model, Messages: msgs, Stream: true}
	ch, err := p.Stream(ctx, req)
	if err != nil {
		return nil, err
	}
	out := make(chan string, 64)
	asst := Message{ID: uuid.NewString(), Role: "assistant", Content: "", CreatedAt: time.Now()}
	go func() {
		defer close(out)
		for c := range ch {
			if c.Err != nil {
				out <- "[error] " + c.Err.Error()
				continue
			}
			if c.Delta != "" {
				asst.Content += c.Delta
				out <- c.Delta
			}
			if c.Reason != "" {
				asst.Reasoning += c.Reason
			}
			if c.Done {
				break
			}
		}
		se.Messages = append(se.Messages, asst)
		se.UpdatedAt = time.Now()
		s.bus.PublishAsync(ctx, "chat.completed", se.ID)
	}()
	return out, nil
}

// Compare 多模型并行对比。
func (s *Service) Compare(ctx context.Context, sessionID, content string, providers []string) map[string]string {
	out := map[string]string{}
	var mu sync.Mutex
	var wg sync.WaitGroup
	for _, pname := range providers {
		wg.Add(1)
		go func(pname string) {
			defer wg.Done()
			prov, ok := s.llm.Get(pname)
			if !ok {
				return
			}
			resp, err := prov.Chat(ctx, llm.ChatRequest{
				Model:    "default",
				Messages: []llm.Message{{Role: llm.RoleUser, Content: content}},
			})
			if err != nil {
				mu.Lock()
				out[pname] = "[error] " + err.Error()
				mu.Unlock()
				return
			}
			mu.Lock()
			out[pname] = resp.Content
			mu.Unlock()
		}(pname)
	}
	wg.Wait()
	return out
}

// EstimateTokens 粗略估算 token 数（每 4 字符 ~ 1 token）。
func (s *Service) EstimateMessages(msgs []Message) int {
	n := 0
	for _, m := range msgs {
		n += len(m.Content) / 4
	}
	return n
}

// Snapshot 导出全部会话。
func (s *Service) Snapshot() ([]byte, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return json.MarshalIndent(struct {
		Sessions []*Session `json:"sessions"`
		Groups   []*Group   `json:"groups"`
	}{Sessions: mapVals(s.sessions), Groups: mapValsG(s.groups)}, "", "  ")
}

func mapVals(m map[string]*Session) []*Session {
	out := make([]*Session, 0, len(m))
	for _, v := range m {
		out = append(out, v)
	}
	return out
}
func mapValsG(m map[string]*Group) []*Group {
	out := make([]*Group, 0, len(m))
	for _, v := range m {
		out = append(out, v)
	}
	return out
}

// IDs 返回所有 session id（用于分页/搜索占位）。
func (s *Service) IDs() []string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]string, 0, len(s.sessions))
	for k := range s.sessions {
		out = append(out, k)
	}
	return out
}

// JoinStrings helper.
func JoinStrings(parts ...string) string { return strings.Join(parts, "") }
