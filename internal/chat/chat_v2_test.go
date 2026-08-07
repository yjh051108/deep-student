// chat_v2 扩展测试：会话管理/标签/工具循环/持久化。

package chat

import (
	"context"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/helixnow/deep-student-go/pkg/eventbus"
	"github.com/helixnow/deep-student-go/pkg/llm"
	"github.com/helixnow/deep-student-go/pkg/store"
	"github.com/helixnow/deep-student-go/pkg/store/blob"
	"github.com/helixnow/deep-student-go/pkg/vfs"
)

// newSvc2 构造带持久化的 chat 服务。
func newSvc2(t *testing.T) *Service {
	t.Helper()
	dir := t.TempDir()
	bs, _ := blob.New(filepath.Join(dir, "b"))
	fs := vfs.NewFS(bs)
	st, err := store.Open(filepath.Join(dir, "x.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = st.Close() })
	reg := llm.NewRegistry()
	reg.Register(&v2MockProv{})
	return New(fs, st, reg, eventbus.New())
}

// v2MockProv 支持工具调用的 mock。
type v2MockProv struct{}

func (m *v2MockProv) Name() string { return "openai" }

func (m *v2MockProv) Chat(_ context.Context, req llm.ChatRequest) (*llm.ChatResponse, error) {
	// 工具已执行过（存在 assistant 消息）→ 直接给最终答案
	hasAssistant := false
	lastUser := ""
	for _, msg := range req.Messages {
		if msg.Role == llm.RoleAssistant {
			hasAssistant = true
		}
		if msg.Role == llm.RoleUser {
			lastUser = msg.Content
		}
	}
	if !hasAssistant && strings.Contains(lastUser, "请计算") {
		return &llm.ChatResponse{
			ToolCalls: []llm.ToolCall{{ID: "tc1", Name: "calc", Arguments: `{"expr":"1+1"}`}},
		}, nil
	}
	return &llm.ChatResponse{Content: "mock-reply"}, nil
}

func (m *v2MockProv) Stream(context.Context, llm.ChatRequest) (<-chan llm.Chunk, error) {
	ch := make(chan llm.Chunk, 1)
	ch <- llm.Chunk{Delta: "mock", Done: true}
	close(ch)
	return ch, nil
}

func (m *v2MockProv) Embed(context.Context, llm.EmbedRequest) (*llm.EmbedResponse, error) {
	return &llm.EmbedResponse{Embeddings: [][]float32{{1}}}, nil
}

func TestChatV2SessionManagement(t *testing.T) {
	s := newSvc2(t)
	g := s.CreateGroup("g", "tutor", "", nil)
	se := s.CreateSession(g.ID, "会话", "gpt-4o-mini", "openai")

	// 更新标题/置顶/标签
	if err := s.UpdateSessionTitle(se.ID, "新标题"); err != nil {
		t.Fatal(err)
	}
	if err := s.PinSession(se.ID, true); err != nil {
		t.Fatal(err)
	}
	if err := s.UpdateSessionTags(se.ID, []string{"数学", "重要"}); err != nil {
		t.Fatal(err)
	}
	got, err := s.GetSession(se.ID)
	if err != nil {
		t.Fatal(err)
	}
	if got.Title != "新标题" || !got.Pinned || len(got.Tags) != 2 {
		t.Fatalf("session=%+v", got)
	}

	// 回收站
	if err := s.SoftDeleteSession(se.ID); err != nil {
		t.Fatal(err)
	}
	if err := s.RestoreSession(se.ID); err != nil {
		t.Fatal(err)
	}
	// 分组回收站
	if err := s.DeleteGroup(g.ID); err != nil {
		t.Fatal(err)
	}
	if err := s.RestoreGroup(g.ID); err != nil {
		t.Fatal(err)
	}
	// 列表
	sessions := s.ListSessions(SessionFilter{})
	if len(sessions) != 1 {
		t.Fatalf("sessions=%d", len(sessions))
	}
}

func TestChatV2ToolLoop(t *testing.T) {
	s := newSvc2(t)
	g := s.CreateGroup("g", "assistant", "", nil)
	se := s.CreateSession(g.ID, "工具", "gpt-4o-mini", "openai")

	s.RegisterTool("calc", func(_ context.Context, args string) (any, error) {
		return map[string]any{"result": 2}, nil
	})

	var deltas []string
	reply, records, err := s.SendWithTools(context.Background(), se.ID, "请计算 1+1", nil, func(d string) {
		deltas = append(deltas, d)
	})
	if err != nil {
		t.Fatal(err)
	}
	if reply == "" {
		t.Fatal("empty reply")
	}
	if len(records) == 0 {
		t.Fatal("expected tool call")
	}
	if records[0].Name != "calc" || !strings.Contains(records[0].Output, "2") {
		t.Fatalf("records=%+v", records[0])
	}
	if len(deltas) == 0 {
		t.Fatal("no delta")
	}
	// 消息已落库
	se2, _ := s.GetSession(se.ID)
	if len(se2.Messages) < 2 {
		t.Fatalf("messages=%d", len(se2.Messages))
	}
}

func TestChatV2ToolNotFound(t *testing.T) {
	s := newSvc2(t)
	g := s.CreateGroup("g", "", "", nil)
	se := s.CreateSession(g.ID, "x", "gpt-4o-mini", "openai")
	_, records, err := s.SendWithTools(context.Background(), se.ID, "请计算 1+1", nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(records) == 0 || records[0].Error == "" {
		t.Fatalf("records=%+v", records)
	}
}

func TestChatV2Persistence(t *testing.T) {
	dir := t.TempDir()
	bs, _ := blob.New(filepath.Join(dir, "b"))
	fs := vfs.NewFS(bs)
	st, err := store.Open(filepath.Join(dir, "x.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = st.Close() })
	reg := llm.NewRegistry()
	reg.Register(&v2MockProv{})

	s := New(fs, st, reg, eventbus.New())
	g := s.CreateGroup("g", "", "", nil)
	se := s.CreateSession(g.ID, "持久", "gpt-4o-mini", "openai")
	// 发消息落库
	se.Messages = append(se.Messages, Message{ID: "m1", SessionID: se.ID, Role: "user", Content: "hi", CreatedAt: time.Now()})
	_ = s.db.AppendMessage(&se.Messages[0])

	// 重启：新 Service 从库加载
	s2 := New(fs, st, reg, eventbus.New())
	got, err := s2.GetSession(se.ID)
	if err != nil {
		t.Fatal(err)
	}
	if got.Title != "持久" {
		t.Fatalf("title=%s", got.Title)
	}
	if len(got.Messages) != 1 || got.Messages[0].Content != "hi" {
		t.Fatalf("messages=%+v", got.Messages)
	}
}

