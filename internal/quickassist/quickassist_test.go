package quickassist

import (
	"context"
	"testing"

	"github.com/helixnow/deep-student-go/pkg/llm"
)

type mockProv struct{}

func (m *mockProv) Name() string { return "openai" }

func (m *mockProv) Chat(_ context.Context, req llm.ChatRequest) (*llm.ChatResponse, error) {
	// 返回最后一条 user 消息的镜像回答
	last := ""
	for _, msg := range req.Messages {
		if msg.Role == llm.RoleUser {
			last = msg.Content
		}
	}
	return &llm.ChatResponse{Content: "回答: " + last}, nil
}

func (m *mockProv) Stream(context.Context, llm.ChatRequest) (<-chan llm.Chunk, error) {
	ch := make(chan llm.Chunk, 1)
	ch <- llm.Chunk{Delta: "x", Done: true}
	close(ch)
	return ch, nil
}

func (m *mockProv) Embed(context.Context, llm.EmbedRequest) (*llm.EmbedResponse, error) {
	return &llm.EmbedResponse{Embeddings: [][]float32{{1}}}, nil
}

func newSvc(t *testing.T) *Service {
	t.Helper()
	reg := llm.NewRegistry()
	reg.Register(&mockProv{})
	return New(reg, nil)
}

func TestAsk(t *testing.T) {
	s := newSvc(t)
	ans, err := s.Ask(context.Background(), "什么是注意力机制？")
	if err != nil {
		t.Fatal(err)
	}
	if ans == "" || !contains(ans, "注意力机制") {
		t.Fatalf("ans=%q", ans)
	}
	// 历史
	hist := s.History(10)
	if len(hist) != 2 {
		t.Fatalf("hist=%d", len(hist))
	}
}

func TestAskEmpty(t *testing.T) {
	s := newSvc(t)
	if _, err := s.Ask(context.Background(), "  "); err == nil {
		t.Fatal("empty question should error")
	}
}

func TestClearAndSummary(t *testing.T) {
	s := newSvc(t)
	s.Ask(context.Background(), "q1")
	s.Clear()
	sum := s.Summary()
	if sum["messageCount"] != 0 {
		t.Fatalf("sum=%+v", sum)
	}
}

func contains(s, sub string) bool {
	return len(s) >= len(sub) && (s == sub || len(s) > 0 && indexOf(s, sub) >= 0)
}

func indexOf(s, sub string) int {
	for i := 0; i+len(sub) <= len(s); i++ {
		if s[i:i+len(sub)] == sub {
			return i
		}
	}
	return -1
}
