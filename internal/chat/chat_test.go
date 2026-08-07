package chat

import (
	"context"
	"path/filepath"
	"strings"
	"testing"

	"github.com/helixnow/deep-student-go/pkg/eventbus"
	"github.com/helixnow/deep-student-go/pkg/llm"
	"github.com/helixnow/deep-student-go/pkg/store"
	"github.com/helixnow/deep-student-go/pkg/store/blob"
	"github.com/helixnow/deep-student-go/pkg/vfs"
)

func newChatSvc(t *testing.T) (*Service, *llm.Registry) {
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
	return New(fs, st, reg, eventbus.New()), reg
}

// fakeProvider 实现 llm.Provider，用于测试流式 / 非流式响应。
type fakeProvider struct {
	name    string
	content string
	chunks  []llm.Chunk
}

func (f *fakeProvider) Name() string { return f.name }
func (f *fakeProvider) Chat(_ context.Context, _ llm.ChatRequest) (*llm.ChatResponse, error) {
	return &llm.ChatResponse{Content: f.content, Usage: llm.Usage{TotalTokens: 1}}, nil
}
func (f *fakeProvider) Stream(_ context.Context, _ llm.ChatRequest) (<-chan llm.Chunk, error) {
	ch := make(chan llm.Chunk, len(f.chunks)+1)
	for _, c := range f.chunks {
		ch <- c
	}
	ch <- llm.Chunk{Done: true}
	close(ch)
	return ch, nil
}
func (f *fakeProvider) Embed(_ context.Context, _ llm.EmbedRequest) (*llm.EmbedResponse, error) {
	return &llm.EmbedResponse{Embeddings: [][]float32{{0.1, 0.2}}}, nil
}

func TestCreateGroupAndSession(t *testing.T) {
	s, _ := newChatSvc(t)
	g := s.CreateGroup("math", "you are a tutor", "tutor", []string{"math"})
	if g.ID == "" {
		t.Fatal("group id empty")
	}
	se := s.CreateSession(g.ID, "test", "m", "fake")
	if se.GroupID != g.ID {
		t.Fatal("session not bound to group")
	}
	if len(s.IDs()) != 1 {
		t.Fatalf("ids=%d", len(s.IDs()))
	}
}

func TestBranch(t *testing.T) {
	s, reg := newChatSvc(t)
	reg.Register(&fakeProvider{name: "fake", content: "x"})
	g := s.CreateGroup("g", "", "", nil)
	se := s.CreateSession(g.ID, "t", "m", "fake")
	se.Messages = append(se.Messages, Message{ID: "m1", Role: "user", Content: "hi"})
	b, err := s.Branch(se.ID, "m1")
	if err != nil {
		t.Fatal(err)
	}
	if b.BranchOf != se.ID {
		t.Fatal("branch parent")
	}
	if len(b.Messages) != 1 {
		t.Fatalf("branch msgs=%d", len(b.Messages))
	}
}

func TestSendStream(t *testing.T) {
	s, reg := newChatSvc(t)
	reg.Register(&fakeProvider{
		name: "fake",
		chunks: []llm.Chunk{
			{Delta: "hello "},
			{Delta: "world"},
		},
	})
	g := s.CreateGroup("g", "", "", nil)
	se := s.CreateSession(g.ID, "t", "m", "fake")
	out, err := s.Send(context.Background(), se.ID, "hi", nil, false)
	if err != nil {
		t.Fatal(err)
	}
	var got strings.Builder
	for chunk := range out {
		got.WriteString(chunk)
	}
	if got.String() != "hello world" {
		t.Fatalf("stream got=%q", got.String())
	}
	if len(se.Messages) != 2 {
		t.Fatalf("msgs=%d", len(se.Messages))
	}
}

func TestCompare(t *testing.T) {
	s, reg := newChatSvc(t)
	reg.Register(&fakeProvider{name: "a", content: "from a"})
	reg.Register(&fakeProvider{name: "b", content: "from b"})
	g := s.CreateGroup("g", "", "", nil)
	se := s.CreateSession(g.ID, "t", "m", "a")
	res := s.Compare(context.Background(), se.ID, "q", []string{"a", "b"})
	if res["a"] != "from a" || res["b"] != "from b" {
		t.Fatalf("compare=%v", res)
	}
}

func TestEstimateTokens(t *testing.T) {
	s, _ := newChatSvc(t)
	toks := s.EstimateMessages([]Message{{Content: "0123456789"}})
	if toks != 2 {
		t.Fatalf("tokens=%d", toks)
	}
}

func TestSnapshot(t *testing.T) {
	s, _ := newChatSvc(t)
	g := s.CreateGroup("g", "", "", nil)
	s.CreateSession(g.ID, "t", "m", "fake")
	b, err := s.Snapshot()
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(b), `"sessions"`) {
		t.Fatal("snapshot missing sessions")
	}
}
