package research

import (
	"context"
	"strings"
	"testing"

	"github.com/helixnow/deep-student-go/pkg/eventbus"
	"github.com/helixnow/deep-student-go/pkg/llm"
)

type fakeProv struct{ out string }

func (f *fakeProv) Name() string { return "openai" }
func (f *fakeProv) Chat(_ context.Context, _ llm.ChatRequest) (*llm.ChatResponse, error) {
	return &llm.ChatResponse{Content: f.out}, nil
}
func (f *fakeProv) Stream(_ context.Context, _ llm.ChatRequest) (<-chan llm.Chunk, error) {
	ch := make(chan llm.Chunk, 1)
	ch <- llm.Chunk{Delta: f.out, Done: true}
	close(ch)
	return ch, nil
}
func (f *fakeProv) Embed(_ context.Context, _ llm.EmbedRequest) (*llm.EmbedResponse, error) {
	return &llm.EmbedResponse{Embeddings: [][]float32{{0}}}, nil
}

func newSvc(t *testing.T, reg *llm.Registry) *Service {
	t.Helper()
	return New(nil, reg, eventbus.New()) // vfs nil; we only test methods that don't touch vfs
}

func TestEngines(t *testing.T) {
	if len(Engines()) != 7 {
		t.Fatalf("expected 7 engines, got %d", len(Engines()))
	}
}

func TestConfirmAndPlanWithJSON(t *testing.T) {
	reg := llm.NewRegistry()
	reg.Register(&fakeProv{out: `{"steps":["step1","step2","step3"]}`})
	s := newSvc(t, reg)
	plan, err := s.ConfirmAndPlan(context.Background(), "topic", "deep", "markdown")
	if err != nil {
		t.Fatal(err)
	}
	if len(plan.Steps) != 3 {
		t.Fatalf("steps=%d", len(plan.Steps))
	}
}

func TestConfirmAndPlanFallback(t *testing.T) {
	reg := llm.NewRegistry()
	reg.Register(&fakeProv{out: "no json here"})
	s := newSvc(t, reg)
	plan, err := s.ConfirmAndPlan(context.Background(), "topic", "deep", "markdown")
	if err != nil {
		t.Fatal(err)
	}
	if len(plan.Steps) < 4 {
		t.Fatalf("fallback steps=%d", len(plan.Steps))
	}
}

func TestRenderMarkdown(t *testing.T) {
	r := &Report{
		Topic: "Topic",
		Sections: []ReportSection{
			{Title: "S1", Content: "C1"},
		},
		Sources: []SearchResult{{Engine: EngineGoogle, Title: "T", URL: "u"}},
	}
	md := renderMarkdown(r)
	if !strings.Contains(md, "# Topic") || !strings.Contains(md, "## S1") || !strings.Contains(md, "## Sources") {
		t.Fatalf("md missing parts: %q", md)
	}
}
