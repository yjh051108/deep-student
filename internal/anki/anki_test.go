package anki

import (
	"path/filepath"
	"testing"

	"github.com/helixnow/deep-student-go/pkg/llm"
	"github.com/helixnow/deep-student-go/pkg/store/blob"
	"github.com/helixnow/deep-student-go/pkg/vfs"
)

func newSvc(t *testing.T) *Service {
	t.Helper()
	dir := t.TempDir()
	bs, _ := blob.New(filepath.Join(dir, "b"))
	fs := vfs.NewFS(bs)
	return New(fs, llm.NewRegistry(), nil)
}

func TestTemplates(t *testing.T) {
	s := newSvc(t)
	tpls := s.Templates()
	if len(tpls) == 0 {
		t.Fatal("no default template")
	}
}

func TestExportAPKG(t *testing.T) {
	s := newSvc(t)
	j := &Job{ID: "j1", Deck: "d", Cards: []Card{{ID: "c1", Front: "F", Back: "B"}}}
	b, err := s.ExportAPKG(j)
	if err != nil {
		t.Fatal(err)
	}
	if len(b) == 0 {
		t.Fatal("empty")
	}
}
