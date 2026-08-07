package store

import (
	"path/filepath"
	"testing"
)

func TestOpenAndMigrate(t *testing.T) {
	dir := t.TempDir()
	s, err := Open(filepath.Join(dir, "test.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	rows, err := s.ListResources("")
	if err != nil {
		t.Fatal(err)
	}
	if len(rows) != 0 {
		t.Fatal("expected empty")
	}
}

func TestResourceCRUD(t *testing.T) {
	dir := t.TempDir()
	s, _ := Open(filepath.Join(dir, "test.db"))
	defer s.Close()
	ts := int64(1700000000)
	if err := s.SaveResource("vfs://note/n1", "note", "n1", "title", "a,b", "{}", "ref", 100, ts); err != nil {
		t.Fatal(err)
	}
	rows, _ := s.ListResources("note")
	if len(rows) != 1 {
		t.Fatalf("rows=%d", len(rows))
	}
	if err := s.DeleteResource("vfs://note/n1"); err != nil {
		t.Fatal(err)
	}
	rows, _ = s.ListResources("note")
	if len(rows) != 0 {
		t.Fatalf("after delete rows=%d", len(rows))
	}
}

func TestChunks(t *testing.T) {
	dir := t.TempDir()
	s, _ := Open(filepath.Join(dir, "test.db"))
	defer s.Close()
	uri := "vfs://note/n1"
	if err := s.AppendChunk(uri, 0, "hello", 1, []byte{1, 2, 3}); err != nil {
		t.Fatal(err)
	}
	chunks, err := s.Chunks(uri)
	if err != nil {
		t.Fatal(err)
	}
	if len(chunks) != 1 || chunks[0].Content != "hello" {
		t.Fatal("chunk mismatch")
	}
}

func TestAuditLog(t *testing.T) {
	dir := t.TempDir()
	s, _ := Open(filepath.Join(dir, "test.db"))
	defer s.Close()
	if err := s.LogAudit("u", "test", "ok", 1700000000); err != nil {
		t.Fatal(err)
	}
	rows, _ := s.AuditLogs(10)
	if len(rows) != 1 || rows[0].Action != "test" {
		t.Fatal("audit mismatch")
	}
}
