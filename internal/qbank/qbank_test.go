package qbank

import (
	"path/filepath"
	"testing"

	"github.com/helixnow/deep-student-go/pkg/llm"
	"github.com/helixnow/deep-student-go/pkg/store"
	"github.com/helixnow/deep-student-go/pkg/store/blob"
	"github.com/helixnow/deep-student-go/pkg/vfs"
)

func newSvc(t *testing.T) *Service {
	t.Helper()
	dir := t.TempDir()
	bs, _ := blob.New(filepath.Join(dir, "b"))
	fs := vfs.NewFS(bs)
	st, err := store.Open(filepath.Join(dir, "x.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = st.Close() })
	return New(fs, st, llm.NewRegistry())
}

func TestAttemptLifecycle(t *testing.T) {
	s := newSvc(t)
	set := &Set{ID: "s1", Title: "T", Questions: []Question{
		{ID: "q1", Stem: "1+1?", Options: []string{"1", "2"}, Answer: "2"},
		{ID: "q2", Stem: "2+2?", Options: []string{"3", "4"}, Answer: "4"},
	}}
	s.mu.Lock()
	s.sets[set.ID] = set
	s.mu.Unlock()
	a, err := s.StartAttempt(set.ID)
	if err != nil {
		t.Fatal(err)
	}
	s.Answer(a.ID, "q1", "2")
	s.Answer(a.ID, "q2", "3")
	final, err := s.Submit(a.ID)
	if err != nil {
		t.Fatal(err)
	}
	if final.Score != 1 {
		t.Fatalf("score=%d", final.Score)
	}
}

func TestMastery(t *testing.T) {
	s := newSvc(t)
	set := &Set{ID: "s1", Questions: []Question{{ID: "q1", Answer: "a", Knowledge: []string{"k1"}}}}
	s.mu.Lock()
	s.sets[set.ID] = set
	s.mu.Unlock()
	a, _ := s.StartAttempt("s1")
	s.Answer(a.ID, "q1", "a")
	s.Submit(a.ID)
	m := s.Mastery()
	if m["k1"] < 5 {
		t.Fatalf("mastery=%v", m)
	}
}
