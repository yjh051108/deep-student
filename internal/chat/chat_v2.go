// chat 包的 chat_v2 扩展：会话管理/标签/搜索/工具循环/多变体并行/子 Agent。

package chat

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/helixnow/deep-student-go/pkg/llm"
)

// RegisterTool 注册 chat_v2 工具。
func (s *Service) RegisterTool(name string, fn ToolFunc) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.tools == nil {
		s.tools = map[string]ToolFunc{}
	}
	s.tools[name] = fn
}

// Tools 列出已注册工具名。
func (s *Service) Tools() []string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]string, 0, len(s.tools))
	for k := range s.tools {
		out = append(out, k)
	}
	return out
}

// ListGroups 列出分组。
func (s *Service) ListGroups(includeDeleted bool) []*Group {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]*Group, 0, len(s.groups))
	for _, g := range s.groups {
		if !includeDeleted && g.IsDeleted {
			continue
		}
		out = append(out, g)
	}
	return out
}

// UpdateGroup 更新分组。
func (s *Service) UpdateGroup(g *Group) error {
	s.mu.Lock()
	g.UpdatedAt = time.Now()
	s.groups[g.ID] = g
	s.mu.Unlock()
	if s.db != nil {
		return s.db.SaveGroup(g)
	}
	return nil
}

// DeleteGroup 软删除分组。
func (s *Service) DeleteGroup(id string) error {
	s.mu.Lock()
	if g, ok := s.groups[id]; ok {
		t := time.Now()
		g.IsDeleted = true
		g.DeletedAt = &t
	}
	s.mu.Unlock()
	if s.db != nil {
		return s.db.DeleteGroup(id)
	}
	return nil
}

// RestoreGroup 恢复分组。
func (s *Service) RestoreGroup(id string) error {
	s.mu.Lock()
	if g, ok := s.groups[id]; ok {
		g.IsDeleted = false
		g.DeletedAt = nil
	}
	s.mu.Unlock()
	if s.db != nil {
		return s.db.RestoreGroup(id)
	}
	return nil
}

// PurgeGroup 彻底删除分组。
func (s *Service) PurgeGroup(id string) error {
	s.mu.Lock()
	delete(s.groups, id)
	s.mu.Unlock()
	if s.db != nil {
		return s.db.PurgeGroup(id)
	}
	return nil
}

// ListSessions 列出会话（支持过滤）。
func (s *Service) ListSessions(filter SessionFilter) []*Session {
	if s.db != nil {
		if out, err := s.db.ListSessions(filter); err == nil {
			return out
		}
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]*Session, 0, len(s.sessions))
	for _, se := range s.sessions {
		if filter.GroupID != "" && se.GroupID != filter.GroupID {
			continue
		}
		if filter.OnlyDeleted && !se.IsDeleted {
			continue
		}
		if !filter.IncludeDeleted && se.IsDeleted {
			continue
		}
		out = append(out, se)
	}
	return out
}

// GetSession 读取会话。
func (s *Service) GetSession(id string) (*Session, error) {
	s.mu.RLock()
	se, ok := s.sessions[id]
	s.mu.RUnlock()
	if ok {
		return se, nil
	}
	if s.db != nil {
		return s.db.GetSession(id)
	}
	return nil, fmt.Errorf("chat: session not found: %s", id)
}

// UpdateSessionTitle 修改会话标题。
func (s *Service) UpdateSessionTitle(id, title string) error {
	s.mu.Lock()
	if se, ok := s.sessions[id]; ok {
		se.Title = title
		se.UpdatedAt = time.Now()
	}
	s.mu.Unlock()
	if s.db != nil {
		return s.db.UpdateSessionTitle(id, title)
	}
	return nil
}

// PinSession 置顶会话。
func (s *Service) PinSession(id string, pinned bool) error {
	s.mu.Lock()
	if se, ok := s.sessions[id]; ok {
		se.Pinned = pinned
		se.UpdatedAt = time.Now()
	}
	s.mu.Unlock()
	if s.db != nil {
		return s.db.PinSession(id, pinned)
	}
	return nil
}

// SoftDeleteSession 软删除会话（回收站）。
func (s *Service) SoftDeleteSession(id string) error {
	s.mu.Lock()
	if se, ok := s.sessions[id]; ok {
		t := time.Now()
		se.IsDeleted = true
		se.DeletedAt = &t
	}
	s.mu.Unlock()
	if s.db != nil {
		return s.db.SoftDeleteSession(id)
	}
	return nil
}

// RestoreSession 恢复会话。
func (s *Service) RestoreSession(id string) error {
	s.mu.Lock()
	if se, ok := s.sessions[id]; ok {
		se.IsDeleted = false
		se.DeletedAt = nil
	}
	s.mu.Unlock()
	if s.db != nil {
		return s.db.RestoreSession(id)
	}
	return nil
}

// PurgeSession 彻底删除会话。
func (s *Service) PurgeSession(id string) error {
	s.mu.Lock()
	delete(s.sessions, id)
	s.mu.Unlock()
	if s.db != nil {
		return s.db.PurgeSession(id)
	}
	return nil
}

// UpdateSessionTags 更新会话标签。
func (s *Service) UpdateSessionTags(id string, tags []string) error {
	s.mu.Lock()
	if se, ok := s.sessions[id]; ok {
		se.Tags = tags
		se.UpdatedAt = time.Now()
	}
	s.mu.Unlock()
	if s.db != nil {
		se, err := s.db.GetSession(id)
		if err != nil {
			return err
		}
		se.Tags = tags
		return s.db.SaveSession(se)
	}
	return nil
}

// SearchContent 搜索会话消息。
func (s *Service) SearchContent(keyword string, limit int) ([]SearchHit, error) {
	if s.db != nil {
		return s.db.SearchMessages(keyword, limit)
	}
	return nil, nil
}

// CountSessions 会话总数。
func (s *Service) CountSessions() (int64, error) {
	if s.db != nil {
		return s.db.CountSessions()
	}
	return int64(len(s.sessions)), nil
}

// DeleteMessage 删除单条消息。
func (s *Service) DeleteMessage(sessionID, messageID string) error {
	s.mu.RLock()
	se, ok := s.sessions[sessionID]
	s.mu.RUnlock()
	if ok {
		for i, m := range se.Messages {
			if m.ID == messageID {
				se.Messages = append(se.Messages[:i], se.Messages[i+1:]...)
				break
			}
		}
	}
	if s.db != nil {
		return s.db.DeleteMessage(sessionID, messageID)
	}
	return nil
}

// SendWithTools 发送消息并允许工具循环（chat_v2）。
// 返回最终助手回复与工具调用记录。onDelta 可选流式回调。
func (s *Service) SendWithTools(ctx context.Context, sessionID, content string, refs []string, onDelta func(string)) (string, []ToolCallRecord, error) {
	se, err := s.GetSession(sessionID)
	if err != nil {
		return "", nil, err
	}
	p, ok := s.llm.Get(se.Provider)
	if !ok {
		if p, ok = s.llm.Get("openai"); !ok {
			return "", nil, fmt.Errorf("chat: no provider")
		}
	}
	// 追加用户消息
	um := Message{ID: uuid.NewString(), SessionID: sessionID, Role: "user", Content: content, Refs: refs, CreatedAt: time.Now()}
	se.Messages = append(se.Messages, um)
	if s.db != nil {
		_ = s.db.AppendMessage(&um)
	}
	s.mu.Lock()
	se.UpdatedAt = time.Now()
	s.mu.Unlock()

	// 工具循环（最多 5 轮）
	var records []ToolCallRecord
	var lastReply string
	for round := 0; round < 5; round++ {
		msgs := s.buildPrompt(se)
		resp, err := p.Chat(ctx, llm.ChatRequest{
			Model:    se.Model,
			Messages: msgs,
			Tools:    s.toolDecls(),
		})
		if err != nil {
			return "", records, err
		}
		if len(resp.ToolCalls) == 0 {
			lastReply = resp.Content
			break
		}
		// 执行工具
		var parts []string
		for _, tc := range resp.ToolCalls {
			fn, ok := s.tool(tc.Name)
			if !ok {
				parts = append(parts, fmt.Sprintf("[工具 %s 不存在]", tc.Name))
				records = append(records, ToolCallRecord{Name: tc.Name, Arguments: tc.Arguments, Error: "tool not found"})
				continue
			}
			out, terr := fn(ctx, tc.Arguments)
			rec := ToolCallRecord{Name: tc.Name, Arguments: tc.Arguments}
			if terr != nil {
				rec.Error = terr.Error()
				parts = append(parts, fmt.Sprintf("[%s 错误: %v]", tc.Name, terr))
			} else {
				rec.Output = fmt.Sprintf("%v", out)
				parts = append(parts, rec.Output)
			}
			records = append(records, rec)
		}
		// 工具结果作为 assistant 消息回灌
		toolMsg := Message{ID: uuid.NewString(), SessionID: sessionID, Role: "assistant", Content: strings.Join(parts, "\n\n"), CreatedAt: time.Now()}
		se.Messages = append(se.Messages, toolMsg)
	}
	// 保存最终回复
	am := Message{ID: uuid.NewString(), SessionID: sessionID, Role: "assistant", Content: lastReply, CreatedAt: time.Now()}
	se.Messages = append(se.Messages, am)
	if s.db != nil {
		_ = s.db.AppendMessage(&am)
	}
	if onDelta != nil {
		onDelta(lastReply)
	}
	return lastReply, records, nil
}

// ToolCallRecord 工具调用记录。
type ToolCallRecord struct {
	Name      string `json:"name"`
	Arguments string `json:"arguments"`
	Output    string `json:"output,omitempty"`
	Error     string `json:"error,omitempty"`
}

// tool 按名取工具。
func (s *Service) tool(name string) (ToolFunc, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	fn, ok := s.tools[name]
	return fn, ok
}

// toolDecls 构造工具声明。
func (s *Service) toolDecls() []llm.Tool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]llm.Tool, 0, len(s.tools))
	for name := range s.tools {
		out = append(out, llm.Tool{
			Name:        name,
			Description: "chat tool: " + name,
			Parameters:  map[string]any{"type": "object", "properties": map[string]any{}},
		})
	}
	return out
}

// buildPrompt 构造消息序列（system + 历史 + 当前）。
func (s *Service) buildPrompt(se *Session) []llm.Message {
	var out []llm.Message
	sys := se.SystemHint
	if sys == "" {
		sys = "You are a helpful AI study assistant."
	}
	out = append(out, llm.Message{Role: llm.RoleSystem, Content: sys})
	// 历史（最多最近 40 条）
	start := 0
	if len(se.Messages) > 40 {
		start = len(se.Messages) - 40
	}
	for _, m := range se.Messages[start:] {
		out = append(out, llm.Message{Role: llm.Role(m.Role), Content: m.Content})
	}
	return out
}

// CompareWithTools 多变体并行比较（chat_v2）。
func (s *Service) CompareWithTools(ctx context.Context, sessionID, content string, providers []string) map[string]string {
	out := map[string]string{}
	for _, prov := range providers {
		p, ok := s.llm.Get(prov)
		if !ok {
			out[prov] = "provider unavailable"
			continue
		}
		resp, err := p.Chat(ctx, llm.ChatRequest{
			Model:    "gpt-4o-mini",
			Messages: []llm.Message{{Role: llm.RoleUser, Content: content}},
		})
		if err != nil {
			out[prov] = "error: " + err.Error()
			continue
		}
		out[prov] = resp.Content
	}
	return out
}

// ExportJSON 导出全部会话（备份/迁移）。
func (s *Service) ExportJSON() ([]byte, error) {
	sessions := s.ListSessions(SessionFilter{IncludeDeleted: true, Limit: 10000})
	groups := s.ListGroups(true)
	return json.Marshal(map[string]any{
		"groups":   groups,
		"sessions": sessions,
		"version":  2,
	})
}
