package hub

import (
	"context"
	"path/filepath"
	"strings"
	"testing"

	"github.com/helixnow/deep-student-go/pkg/llm"
	"github.com/helixnow/deep-student-go/pkg/store"
	"github.com/helixnow/deep-student-go/pkg/store/blob"
	"github.com/helixnow/deep-student-go/pkg/vfs"
)

func newSvc(t *testing.T, reg *llm.Registry) *Service {
	t.Helper()
	dir := t.TempDir()
	bs, _ := blob.New(filepath.Join(dir, "b"))
	fs := vfs.NewFS(bs)
	st, err := store.Open(filepath.Join(dir, "x.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = st.Close() })
	if reg == nil {
		reg = llm.NewRegistry()
	}
	return New(fs, st, reg)
}

type fakeProv struct{ deltas []string }

func (f *fakeProv) Name() string { return "openai" }
func (f *fakeProv) Chat(_ context.Context, _ llm.ChatRequest) (*llm.ChatResponse, error) {
	return &llm.ChatResponse{Content: strings.Join(f.deltas, "")}, nil
}
func (f *fakeProv) Stream(_ context.Context, _ llm.ChatRequest) (<-chan llm.Chunk, error) {
	ch := make(chan llm.Chunk, len(f.deltas)+1)
	for _, d := range f.deltas {
		ch <- llm.Chunk{Delta: d}
	}
	ch <- llm.Chunk{Done: true}
	close(ch)
	return ch, nil
}
func (f *fakeProv) Embed(_ context.Context, _ llm.EmbedRequest) (*llm.EmbedResponse, error) {
	return &llm.EmbedResponse{Embeddings: [][]float32{{0}}}, nil
}

func TestImportListGetDelete(t *testing.T) {
	s := newSvc(t, nil)
	uri, err := s.ImportResource(context.Background(), vfs.TypeNote, "My Note", []byte("hello"), []string{"a", "b"})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(uri, "vfs://note/") {
		t.Fatalf("uri=%s", uri)
	}
	xs := s.List(vfs.TypeNote)
	if len(xs) != 1 {
		t.Fatalf("list=%d", len(xs))
	}
	if xs[0].Title != "My Note" {
		t.Fatalf("title=%q", xs[0].Title)
	}
	xs = s.Search(vfs.TypeNote, "a")
	if len(xs) != 1 {
		t.Fatalf("search=%d", len(xs))
	}
	data, _, err := s.Get(uri)
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != "hello" {
		t.Fatalf("data=%q", data)
	}
	if err := s.Delete(uri); err != nil {
		t.Fatal(err)
	}
	if len(s.List(vfs.TypeNote)) != 0 {
		t.Fatal("not deleted")
	}
}

func TestContinueNote(t *testing.T) {
	reg := llm.NewRegistry()
	reg.Register(&fakeProv{deltas: []string{"a", "b", "c"}})
	s := newSvc(t, reg)
	uri, _ := s.ImportResource(context.Background(), vfs.TypeNote, "T", []byte("seed"), nil)
	out, err := s.ContinueNote(context.Background(), uri, "go on")
	if err != nil {
		t.Fatal(err)
	}
	var sb strings.Builder
	for chunk := range out {
		sb.WriteString(chunk)
	}
	if sb.String() != "abc" {
		t.Fatalf("stream=%q", sb.String())
	}
}

func TestContinueNoteNoProvider(t *testing.T) {
	s := newSvc(t, nil)
	uri, _ := s.ImportResource(context.Background(), vfs.TypeNote, "T", []byte("x"), nil)
	if _, err := s.ContinueNote(context.Background(), uri, "go"); err == nil {
		t.Fatal("expected error")
	}
}
