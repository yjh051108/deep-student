package vfs

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestMindMapCrudStoresVisibleJsonAndListsActiveRecords(t *testing.T) {
	service := newTestService(t)
	description := "Algebra outline"
	view := "outline"
	theme := "minimal"
	folderID := "folder_math"
	content := `{"version":"1.0","root":{"id":"root","text":"Algebra","children":[{"id":"n1","text":"Groups","children":[]}]},"meta":{"createdAt":"2026-01-01T00:00:00Z"}}`

	created, err := service.CreateMindMap(CreateMindMapInput{
		Title:       "Algebra",
		Description: &description,
		Content:     content,
		DefaultView: &view,
		Theme:       &theme,
		FolderID:    &folderID,
	})
	if err != nil {
		t.Fatalf("CreateMindMap() error = %v", err)
	}
	if !strings.HasPrefix(created.ID, "mm_") || created.ResourceID == "" || created.Title != "Algebra" {
		t.Fatalf("unexpected created mindmap: %+v", created)
	}
	if created.Description == nil || *created.Description != description || created.DefaultView != "outline" || created.Theme == nil || *created.Theme != theme {
		t.Fatalf("metadata not preserved: %+v", created)
	}

	stored, err := service.GetMindMap(created.ID)
	if err != nil {
		t.Fatalf("GetMindMap() error = %v", err)
	}
	if stored == nil || stored.ID != created.ID || stored.ResourceID != created.ResourceID {
		t.Fatalf("unexpected GetMindMap result: %+v", stored)
	}
	storedContent, err := service.GetMindMapContent(created.ID)
	if err != nil {
		t.Fatalf("GetMindMapContent() error = %v", err)
	}
	if storedContent == nil || *storedContent != content {
		t.Fatalf("unexpected content: %v", storedContent)
	}

	resource, err := service.GetResource(created.ID)
	if err != nil {
		t.Fatalf("GetResource(mindmap source) error = %v", err)
	}
	if resource == nil || resource.ExternalPath == nil || resource.Type != "mindmap" {
		t.Fatalf("unexpected resource: %+v", resource)
	}
	if !strings.HasSuffix(filepath.ToSlash(*resource.ExternalPath), ".json") {
		t.Fatalf("mindmap should be stored as visible json: %s", *resource.ExternalPath)
	}
	absolute, err := service.GetResourcePath(created.ID)
	if err != nil {
		t.Fatalf("GetResourcePath() error = %v", err)
	}
	if absolute == nil {
		t.Fatal("GetResourcePath() returned nil")
	}
	if _, err := os.Stat(*absolute); err != nil {
		t.Fatalf("visible mindmap json missing: %v", err)
	}

	listed, err := service.ListMindMaps()
	if err != nil {
		t.Fatalf("ListMindMaps() error = %v", err)
	}
	if len(listed) != 1 || listed[0].ID != created.ID {
		t.Fatalf("unexpected list result: %+v", listed)
	}
}

func TestMindMapUpdateCreatesVersionAndDetectsConflict(t *testing.T) {
	service := newTestService(t)
	firstContent := `{"version":"1.0","root":{"id":"root","text":"First","children":[]},"meta":{"createdAt":"2026-01-01T00:00:00Z"}}`
	secondContent := `{"version":"1.0","root":{"id":"root","text":"Second","children":[{"id":"n1","text":"Child","children":[]}]},"meta":{"createdAt":"2026-01-01T00:00:00Z"}}`

	created, err := service.CreateMindMap(CreateMindMapInput{
		Title:   "First",
		Content: firstContent,
	})
	if err != nil {
		t.Fatalf("CreateMindMap() error = %v", err)
	}

	nextTitle := "Second"
	updated, err := service.UpdateMindMap(created.ID, UpdateMindMapInput{
		Title:             &nextTitle,
		Content:           &secondContent,
		ExpectedUpdatedAt: &created.UpdatedAt,
		Settings:          map[string]any{"layoutId": "balanced"},
	})
	if err != nil {
		t.Fatalf("UpdateMindMap() error = %v", err)
	}
	if updated.Title != nextTitle || updated.Settings["layoutId"] != "balanced" || updated.UpdatedAt == created.UpdatedAt {
		t.Fatalf("unexpected updated mindmap: %+v", updated)
	}
	content, err := service.GetMindMapContent(created.ID)
	if err != nil {
		t.Fatalf("GetMindMapContent(updated) error = %v", err)
	}
	if content == nil || *content != secondContent {
		t.Fatalf("updated content not persisted: %v", content)
	}

	versions, err := service.GetMindMapVersions(created.ID)
	if err != nil {
		t.Fatalf("GetMindMapVersions() error = %v", err)
	}
	if len(versions) != 1 || !strings.HasPrefix(versions[0].VersionID, "mv_") || versions[0].MindMapID != created.ID {
		t.Fatalf("unexpected versions: %+v", versions)
	}
	version, err := service.GetMindMapVersion(versions[0].VersionID)
	if err != nil {
		t.Fatalf("GetMindMapVersion() error = %v", err)
	}
	if version == nil || version.VersionID != versions[0].VersionID || version.ResourceID != created.ResourceID {
		t.Fatalf("unexpected version lookup: %+v", version)
	}
	versionContent, err := service.GetMindMapVersionContent(versions[0].VersionID)
	if err != nil {
		t.Fatalf("GetMindMapVersionContent() error = %v", err)
	}
	if versionContent == nil || *versionContent != firstContent {
		t.Fatalf("version should preserve previous content: %v", versionContent)
	}

	staleTitle := "Stale"
	if _, err := service.UpdateMindMap(created.ID, UpdateMindMapInput{
		Title:             &staleTitle,
		ExpectedUpdatedAt: &created.UpdatedAt,
	}); err == nil || !strings.Contains(strings.ToLower(err.Error()), "conflict") {
		t.Fatalf("expected stale update conflict, got %v", err)
	}
}

func TestMindMapFavoriteAndSoftDelete(t *testing.T) {
	service := newTestService(t)
	created, err := service.CreateMindMap(CreateMindMapInput{Title: "Delete Me"})
	if err != nil {
		t.Fatalf("CreateMindMap() error = %v", err)
	}

	if err := service.SetMindMapFavorite(created.ID, true); err != nil {
		t.Fatalf("SetMindMapFavorite() error = %v", err)
	}
	favorite, err := service.GetMindMap(created.ID)
	if err != nil {
		t.Fatalf("GetMindMap(favorite) error = %v", err)
	}
	if favorite == nil || !favorite.IsFavorite {
		t.Fatalf("favorite flag not persisted: %+v", favorite)
	}

	if err := service.DeleteMindMap(created.ID); err != nil {
		t.Fatalf("DeleteMindMap() error = %v", err)
	}
	deleted, err := service.GetMindMap(created.ID)
	if err != nil {
		t.Fatalf("GetMindMap(deleted) error = %v", err)
	}
	if deleted != nil {
		t.Fatalf("deleted mindmap should be hidden: %+v", deleted)
	}
	deletedContent, err := service.GetMindMapContent(created.ID)
	if err != nil {
		t.Fatalf("GetMindMapContent(deleted) error = %v", err)
	}
	if deletedContent != nil {
		t.Fatalf("deleted mindmap content should be hidden: %v", deletedContent)
	}
	listed, err := service.ListMindMaps()
	if err != nil {
		t.Fatalf("ListMindMaps(after delete) error = %v", err)
	}
	if len(listed) != 0 {
		t.Fatalf("deleted mindmap should not be listed: %+v", listed)
	}
}
