package translate

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/helixnow/deep-student-go/pkg/llm"
	"github.com/helixnow/deep-student-go/pkg/store/blob"
	"github.com/helixnow/deep-student-go/pkg/vfs"
)

func newSvc(t *testing.T, reg *llm.Registry) *Service {
	t.Helper()
	dir := t.TempDir()
	bs, _ := blob.New(filepath.Join(dir, "b"))
	fs := vfs.NewFS(bs)
	return New(fs, reg)
}

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

func TestTranslateUsesProvider(t *testing.T) {
	reg := llm.NewRegistry()
	reg.Register(&fakeProv{out: "Bonjour"})
	s := newSvc(t, reg)
	r, err := s.Translate(context.Background(), Request{
		Text: "Hello", Source: "en", Target: "fr", Domain: DomainGeneral,
	})
	if err != nil {
		t.Fatal(err)
	}
	if r.Text != "Bonjour" {
		t.Fatalf("text=%q", r.Text)
	}
}

func TestDomainPrompts(t *testing.T) {
	cases := []Domain{DomainAcademic, DomainTech, DomainLiterary, DomainLegal, DomainMedical, DomainBusiness, DomainGeneral}
	if len(cases) != 7 {
		t.Fatalf("expected 7 domains, got %d", len(Domains()))
	}
	for _, d := range cases {
		p := domainPrompt(d)
		if p == "" {
			t.Fatalf("empty prompt for %s", d)
		}
	}
}

func TestTranslateDocument(t *testing.T) {
	reg := llm.NewRegistry()
	reg.Register(&fakeProv{out: "hola"})
	s := newSvc(t, reg)
	// 准备源文档
	if _, err := s.vfs.Put("vfs://note/a", []byte("hello world"), nil); err != nil {
		t.Fatal(err)
	}
	out, err := s.TranslateDocument(context.Background(), "vfs://note/a", "en", "es", DomainGeneral)
	if err != nil {
		t.Fatal(err)
	}
	if out != "vfs://translation/note/a" {
		t.Fatalf("out=%s", out)
	}
	data, _, err := s.vfs.Get(out)
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != "hola" {
		t.Fatalf("doc=%q", string(data))
	}
}

func TestMissingProvider(t *testing.T) {
	reg := llm.NewRegistry()
	s := newSvc(t, reg)
	_, err := s.Translate(context.Background(), Request{Text: "x", Source: "en", Target: "zh", Domain: DomainGeneral})
	if err == nil {
		t.Fatal("expected error")
	}
}
