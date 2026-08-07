package essay

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

func TestGradeWithJSON(t *testing.T) {
	reg := llm.NewRegistry()
	reg.Register(&fakeProv{out: `{"polished":"P","dimensions":[{"name":"vocab","score":80,"weight":1,"note":"ok"}],"suggestions":["s1"],"highlights":["h1"]}`})
	s := newSvc(t, reg)
	r, err := s.Grade(context.Background(), "hello", ScenarioIELTS, nil)
	if err != nil {
		t.Fatal(err)
	}
	if r.Polished != "P" {
		t.Fatalf("polished=%q", r.Polished)
	}
	if r.Total != 80 {
		t.Fatalf("total=%v", r.Total)
	}
}

func TestGradeNoProvider(t *testing.T) {
	reg := llm.NewRegistry()
	s := newSvc(t, reg)
	if _, err := s.Grade(context.Background(), "x", ScenarioIELTS, nil); err == nil {
		t.Fatal("expected error")
	}
}

func TestSave(t *testing.T) {
	reg := llm.NewRegistry()
	reg.Register(&fakeProv{out: `{"polished":"","dimensions":[],"suggestions":[],"highlights":[]}`})
	s := newSvc(t, reg)
	r, err := s.Grade(context.Background(), "x", ScenarioGaokao, nil)
	if err != nil {
		t.Fatal(err)
	}
	uri, err := s.Save(r)
	if err != nil {
		t.Fatal(err)
	}
	if uri == "" {
		t.Fatal("empty uri")
	}
	if _, _, err := s.vfs.Get(uri); err != nil {
		t.Fatal(err)
	}
}
