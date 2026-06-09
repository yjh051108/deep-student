package notes

import (
	"archive/zip"
	"path/filepath"
	"strings"
	"testing"
)

func TestExportAndReadImportArchive(t *testing.T) {
	dir := t.TempDir()
	service, err := NewService(dir)
	if err != nil {
		t.Fatal(err)
	}
	asset, err := service.SaveAsset("_global", "note_export", "aW1hZ2U=", "png")
	if err != nil {
		t.Fatal(err)
	}
	outputPath := filepath.Join(dir, "exports", "notes.zip")
	result, err := service.Export([]ExportNoteRecord{{
		ID:         "note_export",
		Title:      "Export Me",
		ContentMD:  "# Export\n\n![img](" + asset.RelativePath + ")",
		Tags:       []string{"math", "math", "algebra"},
		CreatedAt:  "2026-01-01T00:00:00Z",
		UpdatedAt:  "2026-01-02T00:00:00Z",
		IsFavorite: true,
	}}, ExportRequest{OutputPath: &outputPath})
	if err != nil {
		t.Fatal(err)
	}
	if result.OutputPath != outputPath || result.NoteCount != 1 || result.AttachmentCount != 1 {
		t.Fatalf("unexpected export result: %+v", result)
	}

	entries := zipEntryNames(t, outputPath)
	for _, want := range []string{"manifest.json", "_notes.json", "README.md", asset.RelativePath} {
		if !entries[want] {
			t.Fatalf("expected archive entry %q in %+v", want, entries)
		}
	}
	foundMarkdown := false
	for name := range entries {
		if strings.HasPrefix(name, "notes/") && strings.HasSuffix(name, ".md") {
			foundMarkdown = true
		}
	}
	if !foundMarkdown {
		t.Fatalf("expected a markdown note entry in %+v", entries)
	}

	importDir := t.TempDir()
	importService, err := NewService(importDir)
	if err != nil {
		t.Fatal(err)
	}
	archive, err := importService.ReadImportArchive(ImportRequest{FilePath: outputPath})
	if err != nil {
		t.Fatal(err)
	}
	if archive.AttachmentCount != 1 || len(archive.Notes) != 1 {
		t.Fatalf("unexpected import archive: %+v", archive)
	}
	if archive.Notes[0].ID != "note_export" || archive.Notes[0].Title != "Export Me" || len(archive.Notes[0].Tags) != 2 {
		t.Fatalf("unexpected imported note metadata: %+v", archive.Notes[0])
	}
	if _, err := importService.ResolveAssetPath(asset.RelativePath); err != nil {
		t.Fatalf("expected imported asset to be resolvable: %v", err)
	}
}

func zipEntryNames(t *testing.T, path string) map[string]bool {
	t.Helper()
	reader, err := zip.OpenReader(path)
	if err != nil {
		t.Fatal(err)
	}
	defer reader.Close()
	out := map[string]bool{}
	for _, file := range reader.File {
		out[file.Name] = true
	}
	return out
}
