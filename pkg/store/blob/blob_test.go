package blob

import (
	"bytes"
	"path/filepath"
	"testing"
)

func TestPutGet(t *testing.T) {
	dir := t.TempDir()
	s, err := New(filepath.Join(dir, "b"))
	if err != nil {
		t.Fatal(err)
	}
	data := []byte("hello world")
	ref, n, err := s.Put(data)
	if err != nil {
		t.Fatal(err)
	}
	if n != int64(len(data)) {
		t.Fatalf("size=%d", n)
	}
	if !s.Has(ref) {
		t.Fatal("ref not found")
	}
	back, err := s.Get(ref)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(back, data) {
		t.Fatal("data mismatch")
	}
}

func TestDedup(t *testing.T) {
	dir := t.TempDir()
	s, _ := New(dir)
	r1, _, _ := s.Put([]byte("same"))
	r2, _, _ := s.Put([]byte("same"))
	if r1 != r2 {
		t.Fatal("dedup failed")
	}
}
