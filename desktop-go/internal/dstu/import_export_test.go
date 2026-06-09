package dstu

import (
	"deep-student-go/internal/notes"
	"deep-student-go/internal/vfs"
	"testing"
)

func TestImportNoteRecordsWritesDstuAndHybridVfs(t *testing.T) {
	dir := t.TempDir()
	vfsService, err := vfs.NewService(dir)
	if err != nil {
		t.Fatal(err)
	}
	service, err := NewService(dir, vfsService)
	if err != nil {
		t.Fatal(err)
	}

	result, err := service.ImportNoteRecords([]notes.ExportNoteRecord{{
		ID:         "note_imported",
		Title:      "Imported",
		ContentMD:  "# Imported",
		Tags:       []string{"go"},
		CreatedAt:  "2026-01-01T00:00:00Z",
		UpdatedAt:  "2026-01-02T00:00:00Z",
		IsFavorite: true,
	}}, "skip")
	if err != nil {
		t.Fatal(err)
	}
	if result.NoteCount != 1 || result.SkippedCount != 0 || result.OverwrittenCount != 0 {
		t.Fatalf("unexpected import result: %+v", result)
	}
	content, err := service.GetContent("/note_imported")
	if err != nil {
		t.Fatal(err)
	}
	if content != "# Imported" {
		t.Fatalf("unexpected content: %q", content)
	}
	refs, err := vfsService.GetResourceRefs(vfs.GetResourceRefsInput{SourceIDs: []string{"note_imported"}})
	if err != nil {
		t.Fatal(err)
	}
	if len(refs.Refs) != 1 || refs.Refs[0].ResourceHash == "" {
		t.Fatalf("expected imported note VFS ref, got %+v", refs)
	}

	skipped, err := service.ImportNoteRecords([]notes.ExportNoteRecord{{
		ID:        "note_imported",
		Title:     "Imported changed",
		ContentMD: "# Changed",
		UpdatedAt: "2026-01-03T00:00:00Z",
	}}, "skip")
	if err != nil {
		t.Fatal(err)
	}
	if skipped.NoteCount != 0 || skipped.SkippedCount != 1 {
		t.Fatalf("unexpected skip result: %+v", skipped)
	}
	content, err = service.GetContent("/note_imported")
	if err != nil {
		t.Fatal(err)
	}
	if content != "# Imported" {
		t.Fatalf("skip should preserve content, got %q", content)
	}

	overwritten, err := service.ImportNoteRecords([]notes.ExportNoteRecord{{
		ID:        "note_imported",
		Title:     "Imported changed",
		ContentMD: "# Changed",
		UpdatedAt: "2026-01-03T00:00:00Z",
	}}, "merge_keep_newer")
	if err != nil {
		t.Fatal(err)
	}
	if overwritten.NoteCount != 1 || overwritten.OverwrittenCount != 1 {
		t.Fatalf("unexpected overwrite result: %+v", overwritten)
	}
	content, err = service.GetContent("/note_imported")
	if err != nil {
		t.Fatal(err)
	}
	if content != "# Changed" {
		t.Fatalf("newer import should update content, got %q", content)
	}
}

func TestExportNoteRecordHidesDeletedNotes(t *testing.T) {
	service := newTestService(t)
	content := "# Keep"
	node, err := service.Create("/", CreateOptions{
		Type:    "note",
		Name:    "Keep",
		Content: &content,
	})
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := service.ExportNoteRecord(node.ID, false); !ok {
		t.Fatal("expected active note to be exportable")
	}
	if _, err := service.Delete(node.Path); err != nil {
		t.Fatal(err)
	}
	if _, ok := service.ExportNoteRecord(node.ID, false); ok {
		t.Fatal("deleted note should be hidden from normal export")
	}
	if _, ok := service.ExportNoteRecord(node.ID, true); !ok {
		t.Fatal("includeDeleted export should see deleted note")
	}
}
