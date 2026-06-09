package files

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"
)

func TestServiceReadSizeAndCopy(t *testing.T) {
	dir := t.TempDir()
	source := filepath.Join(dir, "source.txt")
	dest := filepath.Join(dir, "nested", "dest.txt")
	content := []byte("deep student")

	if err := os.WriteFile(source, content, 0o600); err != nil {
		t.Fatal(err)
	}

	service := NewService()
	size, err := service.GetFileSize(source)
	if err != nil {
		t.Fatal(err)
	}
	if size != int64(len(content)) {
		t.Fatalf("expected size %d, got %d", len(content), size)
	}

	bytesRead, err := service.ReadFileBytes(source)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(bytesRead, content) {
		t.Fatalf("unexpected bytes %q", string(bytesRead))
	}

	if err := service.CopyFile(source, dest); err != nil {
		t.Fatal(err)
	}
	copied, err := os.ReadFile(dest)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(copied, content) {
		t.Fatalf("unexpected copied bytes %q", string(copied))
	}
}

func TestServiceReadAndSaveText(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "nested", "note.md")
	content := "# Deep Student\n\nGo native text file.\n"

	service := NewService()
	if err := service.SaveTextToFile(path, content); err != nil {
		t.Fatal(err)
	}

	text, err := service.ReadFileText(path)
	if err != nil {
		t.Fatal(err)
	}
	if text != content {
		t.Fatalf("unexpected text %q", text)
	}

	bytesRead, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if string(bytesRead) != content {
		t.Fatalf("unexpected saved bytes %q", string(bytesRead))
	}
}

func TestServiceRejectsVirtualURI(t *testing.T) {
	service := NewService()
	if _, err := service.GetFileSize("content://downloads/file.pdf"); err == nil {
		t.Fatal("expected virtual URI to be rejected")
	}
	if _, err := service.ReadFileText("content://downloads/file.pdf"); err == nil {
		t.Fatal("expected virtual URI text read to be rejected")
	}
	if err := service.SaveTextToFile("content://downloads/file.txt", "text"); err == nil {
		t.Fatal("expected virtual URI text write to be rejected")
	}
}
