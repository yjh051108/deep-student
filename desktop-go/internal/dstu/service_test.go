package dstu

import (
	"deep-student-go/internal/vfs"
	"encoding/base64"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func newTestService(t *testing.T) *Service {
	t.Helper()
	service, err := NewService(t.TempDir())
	if err != nil {
		t.Fatalf("NewService() error = %v", err)
	}
	return service
}

func TestCreateGetUpdateAndContentRoundTrip(t *testing.T) {
	service := newTestService(t)
	content := "# Hello"

	node, err := service.Create("/", CreateOptions{
		Type:    "note",
		Name:    "Hello",
		Content: &content,
		Metadata: map[string]any{
			"tags": []string{"math"},
		},
	})
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}
	if node.ID == "" || node.Path != "/"+node.ID || node.PreviewType != "markdown" || node.ResourceHash == "" {
		t.Fatalf("unexpected node: %+v", node)
	}

	got, err := service.Get(node.Path)
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}
	if got == nil || got.ID != node.ID || got.Name != "Hello" {
		t.Fatalf("unexpected get result: %+v", got)
	}

	updated, err := service.Update(node.Path, "updated", "note")
	if err != nil {
		t.Fatalf("Update() error = %v", err)
	}
	if updated.Size != len("updated") || updated.ResourceHash == node.ResourceHash {
		t.Fatalf("unexpected update result: %+v", updated)
	}

	loadedContent, err := service.GetContent(node.Path)
	if err != nil {
		t.Fatalf("GetContent() error = %v", err)
	}
	if loadedContent != "updated" {
		t.Fatalf("GetContent() = %q", loadedContent)
	}
}

func TestCreateAndUpdateSyncsNoteIntoHybridVfs(t *testing.T) {
	dir := t.TempDir()
	vfsService, err := vfs.NewService(dir)
	if err != nil {
		t.Fatalf("vfs.NewService() error = %v", err)
	}
	service, err := NewService(dir, vfsService)
	if err != nil {
		t.Fatalf("NewService() error = %v", err)
	}
	content := "# VFS Note"
	node, err := service.Create("/", CreateOptions{
		Type:    "note",
		Name:    "VFS Note",
		Content: &content,
	})
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}
	if node.ResourceID == "" || node.ResourceID == node.ID || node.ResourceHash == "" {
		t.Fatalf("expected VFS-backed node, got %+v", node)
	}

	refData, err := vfsService.GetResourceRefs(vfs.GetResourceRefsInput{SourceIDs: []string{node.SourceID}})
	if err != nil {
		t.Fatalf("GetResourceRefs() error = %v", err)
	}
	if len(refData.Refs) != 1 || refData.Refs[0].ResourceHash != node.ResourceHash || refData.Refs[0].ResourceID == nil || *refData.Refs[0].ResourceID != node.ResourceID {
		t.Fatalf("unexpected VFS refs: node=%+v refs=%+v", node, refData)
	}
	resolved, err := vfsService.ResolveResourceRefs(refData.Refs)
	if err != nil {
		t.Fatalf("ResolveResourceRefs() error = %v", err)
	}
	if len(resolved) != 1 || !resolved[0].Found || resolved[0].Content == nil || *resolved[0].Content != content {
		t.Fatalf("unexpected resolved note: %+v", resolved)
	}

	updated, err := service.Update(node.Path, "updated vfs note", "note")
	if err != nil {
		t.Fatalf("Update() error = %v", err)
	}
	if updated.ResourceID != node.ResourceID || updated.ResourceHash == node.ResourceHash {
		t.Fatalf("expected stable resource id and changed hash: before=%+v after=%+v", node, updated)
	}
	updatedRefs, err := vfsService.GetResourceRefs(vfs.GetResourceRefsInput{SourceIDs: []string{node.SourceID}})
	if err != nil {
		t.Fatalf("GetResourceRefs(updated) error = %v", err)
	}
	if len(updatedRefs.Refs) != 1 || updatedRefs.Refs[0].ResourceHash != updated.ResourceHash {
		t.Fatalf("unexpected updated refs: node=%+v refs=%+v", updated, updatedRefs)
	}
	updatedResolved, err := vfsService.ResolveResourceRefs(updatedRefs.Refs)
	if err != nil {
		t.Fatalf("ResolveResourceRefs(updated) error = %v", err)
	}
	if len(updatedResolved) != 1 || updatedResolved[0].Content == nil || *updatedResolved[0].Content != "updated vfs note" {
		t.Fatalf("unexpected updated resolved note: %+v", updatedResolved)
	}
}

func TestNotesTrashLifecycleSyncsHybridVfs(t *testing.T) {
	dir := t.TempDir()
	vfsService, err := vfs.NewService(dir)
	if err != nil {
		t.Fatalf("vfs.NewService() error = %v", err)
	}
	service, err := NewService(dir, vfsService)
	if err != nil {
		t.Fatalf("NewService() error = %v", err)
	}

	content := "# Trash me"
	node, err := service.Create("/", CreateOptions{
		Type:    "note",
		Name:    "Trash me",
		Content: &content,
		Metadata: map[string]any{
			"tags": []string{"keep"},
		},
	})
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	deleted, err := service.Delete(node.Path)
	if err != nil {
		t.Fatalf("Delete(note) error = %v", err)
	}
	if !deleted {
		t.Fatal("Delete(note) = false, want true")
	}
	if got, err := service.Get(node.Path); err != nil || got != nil {
		t.Fatalf("deleted note should be hidden from Get: got=%+v err=%v", got, err)
	}
	listed, err := service.List("/", &ListOptions{TypeFilter: strPtr("note")})
	if err != nil {
		t.Fatalf("List(active notes) error = %v", err)
	}
	if len(listed) != 0 {
		t.Fatalf("deleted note should be hidden from active list: %+v", listed)
	}
	tags, err := service.ListTags()
	if err != nil {
		t.Fatalf("ListTags() error = %v", err)
	}
	if len(tags) != 0 {
		t.Fatalf("deleted note tags should be hidden, got %+v", tags)
	}
	refs, err := vfsService.GetResourceRefs(vfs.GetResourceRefsInput{SourceIDs: []string{node.SourceID}})
	if err != nil {
		t.Fatalf("GetResourceRefs(deleted note) error = %v", err)
	}
	if len(refs.Refs) != 0 {
		t.Fatalf("deleted note VFS resource should be hidden from refs: %+v", refs)
	}

	deletedList, err := service.ListDeleted("notes", intPtr(10), intPtr(0))
	if err != nil {
		t.Fatalf("ListDeleted() error = %v", err)
	}
	if len(deletedList) != 1 || deletedList[0].ID != node.ID {
		t.Fatalf("unexpected deleted list: %+v", deletedList)
	}

	restored, err := service.Restore(node.Path)
	if err != nil {
		t.Fatalf("Restore(note) error = %v", err)
	}
	if restored.ID != node.ID || metadataString(restored.Metadata, "status", "") != "active" {
		t.Fatalf("unexpected restored note: %+v", restored)
	}
	refs, err = vfsService.GetResourceRefs(vfs.GetResourceRefsInput{SourceIDs: []string{node.SourceID}})
	if err != nil {
		t.Fatalf("GetResourceRefs(restored note) error = %v", err)
	}
	if len(refs.Refs) != 1 {
		t.Fatalf("restored note VFS ref should be visible, got %+v", refs)
	}

	if err := service.SoftDelete(node.ID, "note"); err != nil {
		t.Fatalf("SoftDelete(note) error = %v", err)
	}
	count, err := service.PurgeAll("notes")
	if err != nil {
		t.Fatalf("PurgeAll(notes) error = %v", err)
	}
	if count != 1 {
		t.Fatalf("PurgeAll(notes) count = %d, want 1", count)
	}
	deletedList, err = service.ListDeleted("notes", nil, nil)
	if err != nil {
		t.Fatalf("ListDeleted(after purge) error = %v", err)
	}
	if len(deletedList) != 0 {
		t.Fatalf("purged notes should leave empty trash, got %+v", deletedList)
	}
}

func TestListGetMetadataFavoriteAndDeleteHybridVfsFiles(t *testing.T) {
	dir := t.TempDir()
	vfsService, err := vfs.NewService(dir)
	if err != nil {
		t.Fatalf("vfs.NewService() error = %v", err)
	}
	service, err := NewService(dir, vfsService)
	if err != nil {
		t.Fatalf("NewService() error = %v", err)
	}

	fileType := "document"
	folderID := "folder_docs"
	content := "calculus derivatives"
	uploaded, err := vfsService.UploadFile(vfs.UploadFileInput{
		Name:          "calculus.md",
		MimeType:      "text/markdown",
		Base64Content: base64.StdEncoding.EncodeToString([]byte(content)),
		FileType:      &fileType,
		FolderID:      &folderID,
		Metadata: map[string]any{
			"tags":        []string{"math"},
			"description": "lecture notes",
		},
	})
	if err != nil {
		t.Fatalf("UploadFile() error = %v", err)
	}

	imageBytes := []byte{0x89, 'P', 'N', 'G'}
	imageUpload, err := vfsService.UploadAttachment(vfs.UploadAttachmentInput{
		Name:          "diagram.png",
		MimeType:      "image/png",
		Base64Content: base64.StdEncoding.EncodeToString(imageBytes),
	})
	if err != nil {
		t.Fatalf("UploadAttachment() error = %v", err)
	}

	files, err := service.List("/", &ListOptions{TypeFilter: strPtr("file")})
	if err != nil {
		t.Fatalf("List(file) error = %v", err)
	}
	if len(files) != 1 || files[0].ID != uploaded.SourceID || files[0].Type != "file" || files[0].PreviewType != "text" {
		t.Fatalf("unexpected file nodes: %+v", files)
	}
	if files[0].ResourceID == "" || files[0].ResourceHash != uploaded.ResourceHash || files[0].Metadata["mimeType"] != "text/markdown" {
		t.Fatalf("unexpected VFS-backed file metadata: %+v", files[0])
	}
	if files[0].Metadata["filePath"] == "" || len(metadataTags(files[0].Metadata)) != 1 {
		t.Fatalf("expected visible path and tags metadata: %+v", files[0].Metadata)
	}

	folderFiles, err := service.List("/", &ListOptions{TypeFilter: strPtr("file"), FolderID: &folderID})
	if err != nil {
		t.Fatalf("List(file by folder) error = %v", err)
	}
	if len(folderFiles) != 1 || folderFiles[0].ID != uploaded.SourceID {
		t.Fatalf("unexpected folder-filtered files: %+v", folderFiles)
	}

	images, err := service.List("/", &ListOptions{TypeFilter: strPtr("image")})
	if err != nil {
		t.Fatalf("List(image) error = %v", err)
	}
	if len(images) != 1 || images[0].ID != imageUpload.SourceID || images[0].Type != "image" || images[0].PreviewType != "image" {
		t.Fatalf("unexpected image nodes: %+v", images)
	}

	got, err := service.Get("/" + uploaded.SourceID)
	if err != nil {
		t.Fatalf("Get(file) error = %v", err)
	}
	if got == nil || got.ID != uploaded.SourceID || got.Name != "calculus.md" {
		t.Fatalf("unexpected Get(file): %+v", got)
	}
	gotImage, err := service.Get("/" + imageUpload.SourceID)
	if err != nil {
		t.Fatalf("Get(image attachment) error = %v", err)
	}
	if gotImage == nil || gotImage.Type != "image" {
		t.Fatalf("unexpected Get(image attachment): %+v", gotImage)
	}

	fileContent, err := service.GetContent("/" + uploaded.SourceID)
	if err != nil {
		t.Fatalf("GetContent(file) error = %v", err)
	}
	if fileContent != base64.StdEncoding.EncodeToString([]byte(content)) {
		t.Fatalf("unexpected file content: %q", fileContent)
	}
	search, err := service.Search("derivatives", &ListOptions{TypeFilter: strPtr("file")})
	if err != nil {
		t.Fatalf("Search(file) error = %v", err)
	}
	if len(search) != 1 || search[0].ID != uploaded.SourceID {
		t.Fatalf("unexpected file search: %+v", search)
	}

	if _, err := service.SetMetadata("/"+uploaded.SourceID, map[string]any{"title": "Renamed notes", "tags": []string{"math", "saved"}}); err != nil {
		t.Fatalf("SetMetadata(file) error = %v", err)
	}
	if _, err := service.SetFavorite("/"+uploaded.SourceID, true); err != nil {
		t.Fatalf("SetFavorite(file) error = %v", err)
	}
	favoriteFiles, err := service.List("/", &ListOptions{TypeFilter: strPtr("file"), IsFavorite: boolPtr(true)})
	if err != nil {
		t.Fatalf("List(favorite file) error = %v", err)
	}
	if len(favoriteFiles) != 1 || favoriteFiles[0].Name != "Renamed notes" || len(metadataTags(favoriteFiles[0].Metadata)) != 2 {
		t.Fatalf("unexpected favorite file metadata: %+v", favoriteFiles)
	}

	deleted, err := service.Delete("/" + uploaded.SourceID)
	if err != nil {
		t.Fatalf("Delete(file) error = %v", err)
	}
	if !deleted {
		t.Fatal("Delete(file) = false, want true")
	}
	missing, err := service.Get("/" + uploaded.SourceID)
	if err != nil {
		t.Fatalf("Get(deleted file) error = %v", err)
	}
	if missing != nil {
		t.Fatalf("deleted VFS file should be hidden from DSTU: %+v", missing)
	}
	filesAfterDelete, err := service.List("/", &ListOptions{TypeFilter: strPtr("file")})
	if err != nil {
		t.Fatalf("List(file after delete) error = %v", err)
	}
	if len(filesAfterDelete) != 0 {
		t.Fatalf("deleted VFS file should not be listed: %+v", filesAfterDelete)
	}
}

func TestFolderResourceLocationAndPathCompatibilityUseHybridMetadata(t *testing.T) {
	dir := t.TempDir()
	vfsService, err := vfs.NewService(dir)
	if err != nil {
		t.Fatalf("vfs.NewService() error = %v", err)
	}
	service, err := NewService(dir, vfsService)
	if err != nil {
		t.Fatalf("NewService() error = %v", err)
	}

	folderID := "folder_math"
	noteContent := "# Limits"
	note, err := service.Create("/", CreateOptions{
		Type:    "note",
		Name:    "Limits",
		Content: &noteContent,
		Metadata: map[string]any{
			"folderId": folderID,
		},
	})
	if err != nil {
		t.Fatalf("Create(note) error = %v", err)
	}

	fileType := "document"
	uploaded, err := vfsService.UploadFile(vfs.UploadFileInput{
		Name:          "derivatives.txt",
		MimeType:      "text/plain",
		Base64Content: base64.StdEncoding.EncodeToString([]byte("derivative rules")),
		FileType:      &fileType,
		FolderID:      &folderID,
	})
	if err != nil {
		t.Fatalf("UploadFile() error = %v", err)
	}

	location, err := service.GetResourceLocation(note.ID)
	if err != nil {
		t.Fatalf("GetResourceLocation(note) error = %v", err)
	}
	if location.ID != note.ID || location.ResourceType != "note" || location.FolderID == nil || *location.FolderID != folderID || location.FullPath != "/"+folderID+"/"+note.ID || location.Hash == nil || *location.Hash != note.ResourceHash {
		t.Fatalf("unexpected note location: %+v", location)
	}

	fileLocation, err := service.GetResourceLocation(uploaded.SourceID)
	if err != nil {
		t.Fatalf("GetResourceLocation(file) error = %v", err)
	}
	if fileLocation.ID != uploaded.SourceID || fileLocation.ResourceType != "file" || fileLocation.FolderID == nil || *fileLocation.FolderID != folderID || fileLocation.Hash == nil || *fileLocation.Hash != uploaded.ResourceHash {
		t.Fatalf("unexpected file location: %+v", fileLocation)
	}

	byPath, err := service.GetResourceByPath("/" + folderID + "/" + uploaded.SourceID)
	if err != nil {
		t.Fatalf("GetResourceByPath(file) error = %v", err)
	}
	if byPath == nil || byPath.ID != uploaded.SourceID || byPath.Type != "file" {
		t.Fatalf("unexpected resource by path: %+v", byPath)
	}

	resources, err := service.GetFolderAllResources(folderID, false, true)
	if err != nil {
		t.Fatalf("GetFolderAllResources() error = %v", err)
	}
	if resources.FolderID != folderID || resources.FolderTitle != folderID || resources.Path != "/"+folderID || resources.TotalCount != 2 || len(resources.Resources) != 2 {
		t.Fatalf("unexpected folder resources result: %+v", resources)
	}
	seen := map[string]FolderResourceInfo{}
	for _, resource := range resources.Resources {
		seen[resource.ItemID] = resource
	}
	if seen[note.ID].Content == nil || *seen[note.ID].Content != noteContent || seen[note.ID].Path != "/"+folderID+"/"+note.ID {
		t.Fatalf("unexpected note folder resource: %+v", seen[note.ID])
	}
	if seen[uploaded.SourceID].Content == nil || *seen[uploaded.SourceID].Content != base64.StdEncoding.EncodeToString([]byte("derivative rules")) {
		t.Fatalf("unexpected file folder resource: %+v", seen[uploaded.SourceID])
	}
}

func TestFolderAllResourcesCanIncludeMetadataSubfolders(t *testing.T) {
	dir := t.TempDir()
	vfsService, err := vfs.NewService(dir)
	if err != nil {
		t.Fatalf("vfs.NewService() error = %v", err)
	}
	service, err := NewService(dir, vfsService)
	if err != nil {
		t.Fatalf("NewService() error = %v", err)
	}

	parentID := "folder_parent"
	childID := "folder_child"
	content := "nested note"
	node, err := service.Create("/", CreateOptions{
		Type:    "note",
		Name:    "Nested",
		Content: &content,
		Metadata: map[string]any{
			"folderId":      childID,
			"folderPathIds": []string{parentID, childID},
		},
	})
	if err != nil {
		t.Fatalf("Create(nested note) error = %v", err)
	}

	direct, err := service.GetFolderAllResources(parentID, false, false)
	if err != nil {
		t.Fatalf("GetFolderAllResources(direct) error = %v", err)
	}
	if direct.TotalCount != 0 || len(direct.Resources) != 0 {
		t.Fatalf("parent direct lookup should not include child resource: %+v", direct)
	}

	recursive, err := service.GetFolderAllResources(parentID, true, false)
	if err != nil {
		t.Fatalf("GetFolderAllResources(recursive) error = %v", err)
	}
	if recursive.TotalCount != 1 || len(recursive.Resources) != 1 || recursive.Resources[0].ItemID != node.ID || recursive.Resources[0].Content != nil {
		t.Fatalf("unexpected recursive folder resources: %+v", recursive)
	}
}

func TestFolderCrudTreeBreadcrumbsAndPersistence(t *testing.T) {
	dir := t.TempDir()
	service, err := NewService(dir)
	if err != nil {
		t.Fatalf("NewService() error = %v", err)
	}

	math, err := service.CreateFolder("Math", nil, strPtr("BookOpen"), strPtr("#336699"))
	if err != nil {
		t.Fatalf("CreateFolder(parent) error = %v", err)
	}
	english, err := service.CreateFolder("English", nil, nil, nil)
	if err != nil {
		t.Fatalf("CreateFolder(sibling) error = %v", err)
	}
	algebra, err := service.CreateFolder("Algebra", &math.ID, nil, nil)
	if err != nil {
		t.Fatalf("CreateFolder(child) error = %v", err)
	}
	if err := service.RenameFolder(algebra.ID, "Linear Algebra"); err != nil {
		t.Fatalf("RenameFolder() error = %v", err)
	}
	if err := service.SetFolderExpanded(math.ID, false); err != nil {
		t.Fatalf("SetFolderExpanded() error = %v", err)
	}
	if err := service.ReorderFolders([]string{english.ID, math.ID}); err != nil {
		t.Fatalf("ReorderFolders() error = %v", err)
	}

	tree, err := service.GetFolderTree()
	if err != nil {
		t.Fatalf("GetFolderTree() error = %v", err)
	}
	if len(tree) != 2 || tree[0].Folder.ID != english.ID || tree[1].Folder.ID != math.ID || tree[1].Folder.IsExpanded {
		t.Fatalf("unexpected folder tree: %+v", tree)
	}
	if len(tree[1].Children) != 1 || tree[1].Children[0].Folder.Title != "Linear Algebra" {
		t.Fatalf("unexpected child tree: %+v", tree[1].Children)
	}
	breadcrumbs, err := service.GetFolderBreadcrumbs(algebra.ID)
	if err != nil {
		t.Fatalf("GetFolderBreadcrumbs() error = %v", err)
	}
	if len(breadcrumbs) != 2 || breadcrumbs[0].Name != "Math" || breadcrumbs[1].Name != "Linear Algebra" {
		t.Fatalf("unexpected breadcrumbs: %+v", breadcrumbs)
	}

	reloaded, err := NewService(dir)
	if err != nil {
		t.Fatalf("NewService(reloaded) error = %v", err)
	}
	got, err := reloaded.GetFolder(algebra.ID)
	if err != nil {
		t.Fatalf("GetFolder(reloaded) error = %v", err)
	}
	if got == nil || got.Title != "Linear Algebra" || got.ParentID == nil || *got.ParentID != math.ID {
		t.Fatalf("unexpected reloaded folder: %+v", got)
	}
}

func TestFolderItemsMoveNoteAndPathCommands(t *testing.T) {
	dir := t.TempDir()
	vfsService, err := vfs.NewService(dir)
	if err != nil {
		t.Fatalf("vfs.NewService() error = %v", err)
	}
	service, err := NewService(dir, vfsService)
	if err != nil {
		t.Fatalf("NewService() error = %v", err)
	}

	math, err := service.CreateFolder("Math", nil, nil, nil)
	if err != nil {
		t.Fatalf("CreateFolder(Math) error = %v", err)
	}
	algebra, err := service.CreateFolder("Algebra", &math.ID, nil, nil)
	if err != nil {
		t.Fatalf("CreateFolder(Algebra) error = %v", err)
	}
	content := "# Linear systems"
	note, err := service.Create("/", CreateOptions{Type: "note", Name: "Systems", Content: &content})
	if err != nil {
		t.Fatalf("Create(note) error = %v", err)
	}

	location, err := service.MoveToFolder(note.ID, &algebra.ID)
	if err != nil {
		t.Fatalf("MoveToFolder(note) error = %v", err)
	}
	if location.FolderID == nil || *location.FolderID != algebra.ID || location.FolderPath != "/Math/Algebra" || location.FullPath != "/Math/Algebra/"+note.ID {
		t.Fatalf("unexpected note location: %+v", location)
	}
	got, err := service.Get(note.ID)
	if err != nil {
		t.Fatalf("Get(note) error = %v", err)
	}
	if got == nil || got.Path != location.FullPath || got.Metadata["folderId"] != algebra.ID {
		t.Fatalf("unexpected folder-backed note node: %+v", got)
	}
	builtPath, err := service.BuildPath(&algebra.ID, note.ID)
	if err != nil {
		t.Fatalf("BuildPath() error = %v", err)
	}
	if builtPath != location.FullPath {
		t.Fatalf("BuildPath() = %q, want %q", builtPath, location.FullPath)
	}
	pathByID, err := service.GetPathByID(note.ID)
	if err != nil {
		t.Fatalf("GetPathByID() error = %v", err)
	}
	if pathByID != location.FullPath {
		t.Fatalf("GetPathByID() = %q, want %q", pathByID, location.FullPath)
	}
	parsed, err := service.ParsePath(location.FullPath)
	if err != nil {
		t.Fatalf("ParsePath() error = %v", err)
	}
	if parsed.ResourceID == nil || *parsed.ResourceID != note.ID || parsed.FolderPath == nil || *parsed.FolderPath != "/Math/Algebra" {
		t.Fatalf("unexpected parsed path: %+v", parsed)
	}
	if err := service.MoveFolder(math.ID, &algebra.ID); err == nil {
		t.Fatal("MoveFolder(parent into child) should reject cycles")
	}
}

func TestFolderItemsMoveVfsFileAndFolderResources(t *testing.T) {
	dir := t.TempDir()
	vfsService, err := vfs.NewService(dir)
	if err != nil {
		t.Fatalf("vfs.NewService() error = %v", err)
	}
	service, err := NewService(dir, vfsService)
	if err != nil {
		t.Fatalf("NewService() error = %v", err)
	}

	folder, err := service.CreateFolder("Resources", nil, nil, nil)
	if err != nil {
		t.Fatalf("CreateFolder() error = %v", err)
	}
	fileType := "document"
	content := "folder file content"
	uploaded, err := vfsService.UploadFile(vfs.UploadFileInput{
		Name:          "resource.txt",
		MimeType:      "text/plain",
		Base64Content: base64.StdEncoding.EncodeToString([]byte(content)),
		FileType:      &fileType,
	})
	if err != nil {
		t.Fatalf("UploadFile() error = %v", err)
	}
	if _, err := service.AddFolderItem(&folder.ID, "file", uploaded.SourceID); err != nil {
		t.Fatalf("AddFolderItem(file) error = %v", err)
	}
	file, err := vfsService.GetFile(uploaded.SourceID)
	if err != nil {
		t.Fatalf("GetFile() error = %v", err)
	}
	if file == nil || file.Metadata["folderId"] != folder.ID || file.Metadata["folderPath"] != "/Resources" {
		t.Fatalf("expected VFS file metadata to follow folder item: %+v", file)
	}
	location, err := service.GetResourceLocation(uploaded.SourceID)
	if err != nil {
		t.Fatalf("GetResourceLocation(file) error = %v", err)
	}
	if location.FullPath != "/Resources/"+uploaded.SourceID || location.Hash == nil || *location.Hash != uploaded.ResourceHash {
		t.Fatalf("unexpected file location: %+v", location)
	}
	resources, err := service.GetFolderAllResources(folder.ID, false, true)
	if err != nil {
		t.Fatalf("GetFolderAllResources() error = %v", err)
	}
	if resources.TotalCount != 1 || len(resources.Resources) != 1 || resources.Resources[0].ItemID != uploaded.SourceID || resources.Resources[0].Content == nil || *resources.Resources[0].Content != base64.StdEncoding.EncodeToString([]byte(content)) {
		t.Fatalf("unexpected folder resources: %+v", resources)
	}
	if err := service.RemoveFolderItem("file", uploaded.SourceID); err != nil {
		t.Fatalf("RemoveFolderItem() error = %v", err)
	}
	rootLocation, err := service.GetResourceLocation(uploaded.SourceID)
	if err != nil {
		t.Fatalf("GetResourceLocation(rooted file) error = %v", err)
	}
	if rootLocation.FolderID != nil || rootLocation.FullPath != "/"+uploaded.SourceID {
		t.Fatalf("expected file to move back to root: %+v", rootLocation)
	}
}

func TestBatchMoveReorderAndDeleteFolderMovesItemsToRoot(t *testing.T) {
	service := newTestService(t)
	folder, err := service.CreateFolder("Batch", nil, nil, nil)
	if err != nil {
		t.Fatalf("CreateFolder() error = %v", err)
	}
	firstContent := "first"
	secondContent := "second"
	first, err := service.Create("/", CreateOptions{Type: "note", Name: "First", Content: &firstContent})
	if err != nil {
		t.Fatalf("Create(first) error = %v", err)
	}
	second, err := service.Create("/", CreateOptions{Type: "note", Name: "Second", Content: &secondContent})
	if err != nil {
		t.Fatalf("Create(second) error = %v", err)
	}

	result, err := service.BatchMove(BatchMoveRequest{ItemIDs: []string{first.ID, "", second.ID}, TargetFolderID: &folder.ID})
	if err != nil {
		t.Fatalf("BatchMove() error = %v", err)
	}
	if result.TotalCount != 3 || len(result.Successes) != 2 || len(result.FailedItems) != 1 {
		t.Fatalf("unexpected batch result: %+v", result)
	}
	if err := service.ReorderFolderItems(&folder.ID, []string{second.ID, first.ID}); err != nil {
		t.Fatalf("ReorderFolderItems() error = %v", err)
	}
	items, err := service.GetFolderItems(&folder.ID)
	if err != nil {
		t.Fatalf("GetFolderItems() error = %v", err)
	}
	if len(items) != 2 || items[0].ItemID != second.ID || items[1].ItemID != first.ID {
		t.Fatalf("unexpected reordered items: %+v", items)
	}
	refreshed, err := service.RefreshPathCache(nil)
	if err != nil {
		t.Fatalf("RefreshPathCache() error = %v", err)
	}
	if refreshed != 2 {
		t.Fatalf("RefreshPathCache() = %d, want 2", refreshed)
	}
	if err := service.DeleteFolder(folder.ID); err != nil {
		t.Fatalf("DeleteFolder() error = %v", err)
	}
	rootItems, err := service.GetFolderItems(nil)
	if err != nil {
		t.Fatalf("GetFolderItems(root) error = %v", err)
	}
	if len(rootItems) != 2 {
		t.Fatalf("expected deleted folder items at root: %+v", rootItems)
	}
}

func TestDstuCreateFileImageAndTextbookUseHybridVfs(t *testing.T) {
	dir := t.TempDir()
	vfsService, err := vfs.NewService(dir)
	if err != nil {
		t.Fatalf("vfs.NewService() error = %v", err)
	}
	service, err := NewService(dir, vfsService)
	if err != nil {
		t.Fatalf("NewService() error = %v", err)
	}

	textBase := base64.StdEncoding.EncodeToString([]byte("plain file"))
	fileNode, err := service.Create("/", CreateOptions{
		Type:     "file",
		Name:     "plain.txt",
		FileBase: &textBase,
		Metadata: map[string]any{
			"mimeType": "text/plain",
		},
	})
	if err != nil {
		t.Fatalf("Create(file) error = %v", err)
	}
	if fileNode.Type != "file" || fileNode.PreviewType != "text" || fileNode.ResourceID == "" || fileNode.ResourceHash == "" {
		t.Fatalf("unexpected DSTU-created file node: %+v", fileNode)
	}
	if file, err := vfsService.GetFile(fileNode.ID); err != nil || file == nil || file.Metadata["sourceType"] != "file" {
		t.Fatalf("expected DSTU-created file in VFS, file=%+v err=%v", file, err)
	}

	imageBase := base64.StdEncoding.EncodeToString([]byte{0x89, 'P', 'N', 'G'})
	imageNode, err := service.Create("/", CreateOptions{
		Type:     "image",
		Name:     "diagram.png",
		FileBase: &imageBase,
		Metadata: map[string]any{
			"mimeType": "image/png",
		},
	})
	if err != nil {
		t.Fatalf("Create(image) error = %v", err)
	}
	if imageNode.Type != "image" || imageNode.PreviewType != "image" {
		t.Fatalf("unexpected DSTU-created image node: %+v", imageNode)
	}

	pdfBase := base64.StdEncoding.EncodeToString([]byte("%PDF-1.7 fake"))
	textbookNode, err := service.Create("/", CreateOptions{
		Type:     "textbook",
		Name:     "algebra.pdf",
		FileBase: &pdfBase,
		Metadata: map[string]any{
			"mimeType":  "application/pdf",
			"pageCount": 12,
		},
	})
	if err != nil {
		t.Fatalf("Create(textbook) error = %v", err)
	}
	if textbookNode.Type != "textbook" || textbookNode.PreviewType != "pdf" || textbookNode.Metadata["pageCount"] != 12 {
		t.Fatalf("unexpected DSTU-created textbook node: %+v", textbookNode)
	}
	resource, err := vfsService.GetResource(textbookNode.ResourceID)
	if err != nil {
		t.Fatalf("GetResource(textbook) error = %v", err)
	}
	if resource == nil || resource.Type != "textbook" || resource.Metadata["sourceType"] != "textbook" {
		t.Fatalf("expected textbook resource in hybrid VFS: %+v", resource)
	}

	textbooks, err := service.List("/", &ListOptions{TypeFilter: strPtr("textbook")})
	if err != nil {
		t.Fatalf("List(textbook) error = %v", err)
	}
	if len(textbooks) != 1 || textbooks[0].ID != textbookNode.ID {
		t.Fatalf("unexpected textbook list: %+v", textbooks)
	}
	regularFiles, err := service.List("/", &ListOptions{TypeFilter: strPtr("file")})
	if err != nil {
		t.Fatalf("List(file) error = %v", err)
	}
	if len(regularFiles) != 1 || regularFiles[0].ID != fileNode.ID {
		t.Fatalf("textbook should not be listed as regular file: %+v", regularFiles)
	}
}

func TestAddTextbooksImportsLocalFilesIntoHybridVfs(t *testing.T) {
	dir := t.TempDir()
	vfsService, err := vfs.NewService(dir)
	if err != nil {
		t.Fatalf("vfs.NewService() error = %v", err)
	}
	service, err := NewService(dir, vfsService)
	if err != nil {
		t.Fatalf("NewService() error = %v", err)
	}

	sourceDir := t.TempDir()
	sourcePath := filepath.Join(sourceDir, "algebra.pdf")
	sourceBytes := []byte("%PDF-1.7\nlean textbook")
	if err := os.WriteFile(sourcePath, sourceBytes, 0o600); err != nil {
		t.Fatalf("WriteFile(textbook) error = %v", err)
	}
	folderID := "folder_textbooks"

	records, err := service.AddTextbooks(AddTextbooksRequest{
		Sources:  []string{sourcePath},
		FolderID: &folderID,
	})
	if err != nil {
		t.Fatalf("AddTextbooks() error = %v", err)
	}
	if len(records) != 1 {
		t.Fatalf("AddTextbooks() returned %d records, want 1: %+v", len(records), records)
	}
	record := records[0]
	if record.ID == "" || record.FileName != "algebra.pdf" || record.FilePath == "" || record.SHA256 == "" || record.ResourceID == "" || record.ResourceHash == "" {
		t.Fatalf("unexpected textbook record: %+v", record)
	}
	if record.PreviewType != "pdf" || record.Metadata["sourceType"] != "textbook" || record.Metadata["folderId"] != folderID {
		t.Fatalf("unexpected textbook metadata: %+v", record)
	}

	textbooks, err := service.List("/", &ListOptions{TypeFilter: strPtr("textbook")})
	if err != nil {
		t.Fatalf("List(textbook) error = %v", err)
	}
	if len(textbooks) != 1 || textbooks[0].ID != record.ID || textbooks[0].ResourceID != record.ResourceID || textbooks[0].ResourceHash != record.SHA256 {
		t.Fatalf("unexpected textbook list: records=%+v nodes=%+v", records, textbooks)
	}
	folderTextbooks, err := service.List("/", &ListOptions{TypeFilter: strPtr("textbook"), FolderID: &folderID})
	if err != nil {
		t.Fatalf("List(textbook by folder) error = %v", err)
	}
	if len(folderTextbooks) != 1 || folderTextbooks[0].ID != record.ID {
		t.Fatalf("unexpected folder-filtered textbooks: %+v", folderTextbooks)
	}

	resource, err := vfsService.GetResource(record.ResourceID)
	if err != nil {
		t.Fatalf("GetResource(textbook) error = %v", err)
	}
	if resource == nil || resource.Type != "textbook" || resource.Metadata["sourceType"] != "textbook" || resource.Metadata["sourcePath"] == "" {
		t.Fatalf("expected imported textbook resource in hybrid VFS: %+v", resource)
	}

	content, err := service.GetContent("/" + record.ID)
	if err != nil {
		t.Fatalf("GetContent(textbook) error = %v", err)
	}
	if content != base64.StdEncoding.EncodeToString(sourceBytes) {
		t.Fatalf("unexpected textbook content: %q", content)
	}
}

func TestAddTextbooksEmitsProgressForSuccessfulImport(t *testing.T) {
	dir := t.TempDir()
	vfsService, err := vfs.NewService(dir)
	if err != nil {
		t.Fatalf("vfs.NewService() error = %v", err)
	}
	service, err := NewService(dir, vfsService)
	if err != nil {
		t.Fatalf("NewService() error = %v", err)
	}

	events := []dstuTestEvent{}
	service.SetEventEmitter(func(name string, payload any) {
		events = append(events, dstuTestEvent{name: name, payload: payload})
	})

	sourceDir := t.TempDir()
	sourcePath := filepath.Join(sourceDir, "events.pdf")
	if err := os.WriteFile(sourcePath, []byte("%PDF-1.7\nprogress textbook"), 0o600); err != nil {
		t.Fatalf("WriteFile(textbook) error = %v", err)
	}

	records, err := service.AddTextbooks(AddTextbooksRequest{Sources: []string{sourcePath}})
	if err != nil {
		t.Fatalf("AddTextbooks() error = %v", err)
	}
	if len(records) != 1 {
		t.Fatalf("AddTextbooks() records = %+v", records)
	}

	stages := textbookImportStages(events)
	wantStages := []string{"hashing", "copying", "saving", "done"}
	if strings.Join(stages, ",") != strings.Join(wantStages, ",") {
		t.Fatalf("unexpected progress stages: got=%+v want=%+v events=%+v", stages, wantStages, events)
	}
	importID := ""
	for _, event := range events {
		if event.name != "textbook-import-progress" {
			t.Fatalf("unexpected event name: %+v", event)
		}
		payload, ok := event.payload.(textbookImportProgressPayload)
		if !ok {
			t.Fatalf("unexpected payload type: %T", event.payload)
		}
		if payload.FileName != "events.pdf" || payload.Index != 0 || payload.Total != 1 || payload.Progress < 0 || payload.Progress > 100 {
			t.Fatalf("unexpected progress payload: %+v", payload)
		}
		if strings.TrimSpace(payload.ImportID) == "" {
			t.Fatalf("expected import id in payload: %+v", payload)
		}
		if importID == "" {
			importID = payload.ImportID
		}
		if payload.ImportID != importID {
			t.Fatalf("all events should share one import id: first=%s payload=%+v", importID, payload)
		}
		if payload.Stage == "done" && (payload.TextbookID != records[0].ID || payload.ResourceID != records[0].ResourceID) {
			t.Fatalf("done payload should include imported resource identity: payload=%+v record=%+v", payload, records[0])
		}
	}
}

func TestAddTextbooksEmitsProgressForEachSource(t *testing.T) {
	dir := t.TempDir()
	vfsService, err := vfs.NewService(dir)
	if err != nil {
		t.Fatalf("vfs.NewService() error = %v", err)
	}
	service, err := NewService(dir, vfsService)
	if err != nil {
		t.Fatalf("NewService() error = %v", err)
	}

	events := []dstuTestEvent{}
	service.SetEventEmitter(func(name string, payload any) {
		events = append(events, dstuTestEvent{name: name, payload: payload})
	})

	sourceDir := t.TempDir()
	firstPath := filepath.Join(sourceDir, "first.pdf")
	secondPath := filepath.Join(sourceDir, "second.pdf")
	if err := os.WriteFile(firstPath, []byte("%PDF-1.7\nfirst"), 0o600); err != nil {
		t.Fatalf("WriteFile(first) error = %v", err)
	}
	if err := os.WriteFile(secondPath, []byte("%PDF-1.7\nsecond"), 0o600); err != nil {
		t.Fatalf("WriteFile(second) error = %v", err)
	}

	if _, err := service.AddTextbooks(AddTextbooksRequest{Sources: []string{firstPath, secondPath}}); err != nil {
		t.Fatalf("AddTextbooks() error = %v", err)
	}

	doneByName := map[string]textbookImportProgressPayload{}
	for _, event := range events {
		payload, ok := event.payload.(textbookImportProgressPayload)
		if !ok {
			t.Fatalf("unexpected payload type: %T", event.payload)
		}
		if payload.Stage == "done" {
			doneByName[payload.FileName] = payload
		}
	}
	if len(doneByName) != 2 {
		t.Fatalf("expected done event per source, got %+v from events %+v", doneByName, events)
	}
	if doneByName["first.pdf"].Index != 0 || doneByName["first.pdf"].Total != 2 {
		t.Fatalf("unexpected first done payload: %+v", doneByName["first.pdf"])
	}
	if doneByName["second.pdf"].Index != 1 || doneByName["second.pdf"].Total != 2 {
		t.Fatalf("unexpected second done payload: %+v", doneByName["second.pdf"])
	}
}

func TestAddTextbooksEmitsErrorProgressWhenSourceReadFails(t *testing.T) {
	dir := t.TempDir()
	vfsService, err := vfs.NewService(dir)
	if err != nil {
		t.Fatalf("vfs.NewService() error = %v", err)
	}
	service, err := NewService(dir, vfsService)
	if err != nil {
		t.Fatalf("NewService() error = %v", err)
	}

	events := []dstuTestEvent{}
	service.SetEventEmitter(func(name string, payload any) {
		events = append(events, dstuTestEvent{name: name, payload: payload})
	})

	missingPath := filepath.Join(t.TempDir(), "missing.pdf")
	if _, err := service.AddTextbooks(AddTextbooksRequest{Sources: []string{missingPath}}); err == nil {
		t.Fatal("AddTextbooks(missing) expected error")
	}

	stages := textbookImportStages(events)
	wantStages := []string{"hashing", "copying", "error"}
	if strings.Join(stages, ",") != strings.Join(wantStages, ",") {
		t.Fatalf("unexpected progress stages: got=%+v want=%+v events=%+v", stages, wantStages, events)
	}
	lastPayload, ok := events[len(events)-1].payload.(textbookImportProgressPayload)
	if !ok {
		t.Fatalf("unexpected payload type: %T", events[len(events)-1].payload)
	}
	if lastPayload.Error == nil || !strings.Contains(*lastPayload.Error, "read textbook file") || lastPayload.Stage != "error" || lastPayload.FileName != "missing.pdf" {
		t.Fatalf("unexpected error payload: %+v", lastPayload)
	}
	textbooks, err := service.List("/", &ListOptions{TypeFilter: strPtr("textbook")})
	if err != nil {
		t.Fatalf("List(textbook) error = %v", err)
	}
	if len(textbooks) != 0 {
		t.Fatalf("failed import should not create textbook nodes: %+v", textbooks)
	}
}

func TestAddTextbooksPromotesExistingFileResource(t *testing.T) {
	dir := t.TempDir()
	vfsService, err := vfs.NewService(dir)
	if err != nil {
		t.Fatalf("vfs.NewService() error = %v", err)
	}
	service, err := NewService(dir, vfsService)
	if err != nil {
		t.Fatalf("NewService() error = %v", err)
	}

	sourceDir := t.TempDir()
	sourcePath := filepath.Join(sourceDir, "shared.pdf")
	sourceBytes := []byte("%PDF-1.7\nshared bytes")
	if err := os.WriteFile(sourcePath, sourceBytes, 0o600); err != nil {
		t.Fatalf("WriteFile(shared) error = %v", err)
	}
	fileType := "document"
	uploaded, err := vfsService.UploadFile(vfs.UploadFileInput{
		Name:          "shared.pdf",
		MimeType:      "application/pdf",
		Base64Content: base64.StdEncoding.EncodeToString(sourceBytes),
		FileType:      &fileType,
	})
	if err != nil {
		t.Fatalf("UploadFile(file) error = %v", err)
	}
	filesBefore, err := service.List("/", &ListOptions{TypeFilter: strPtr("file")})
	if err != nil {
		t.Fatalf("List(file before promote) error = %v", err)
	}
	if len(filesBefore) != 1 || filesBefore[0].ID != uploaded.SourceID {
		t.Fatalf("expected regular file before promote: %+v", filesBefore)
	}

	records, err := service.AddTextbooks(AddTextbooksRequest{Sources: []string{sourcePath}})
	if err != nil {
		t.Fatalf("AddTextbooks(promote) error = %v", err)
	}
	if len(records) != 1 || records[0].ID != uploaded.SourceID || records[0].ResourceID == "" {
		t.Fatalf("expected existing VFS file to be promoted: uploaded=%+v records=%+v", uploaded, records)
	}
	resource, err := vfsService.GetResource(records[0].ResourceID)
	if err != nil {
		t.Fatalf("GetResource(promoted) error = %v", err)
	}
	if resource == nil || resource.Type != "textbook" || resource.Metadata["sourceType"] != "textbook" {
		t.Fatalf("expected promoted textbook resource: %+v", resource)
	}
	textbooks, err := service.List("/", &ListOptions{TypeFilter: strPtr("textbook")})
	if err != nil {
		t.Fatalf("List(textbook after promote) error = %v", err)
	}
	if len(textbooks) != 1 || textbooks[0].ID != uploaded.SourceID {
		t.Fatalf("expected promoted resource in textbook list: %+v", textbooks)
	}
	filesAfter, err := service.List("/", &ListOptions{TypeFilter: strPtr("file")})
	if err != nil {
		t.Fatalf("List(file after promote) error = %v", err)
	}
	if len(filesAfter) != 0 {
		t.Fatalf("promoted textbook should no longer appear as regular file: %+v", filesAfter)
	}
}

func TestListSearchTagsFavoriteAndPagination(t *testing.T) {
	service := newTestService(t)
	firstContent := "calculus limits"
	secondContent := "grammar"
	first, err := service.Create("/", CreateOptions{
		Type:    "note",
		Name:    "Math",
		Content: &firstContent,
		Metadata: map[string]any{
			"tags":       []string{"exam", "math"},
			"isFavorite": true,
		},
	})
	if err != nil {
		t.Fatalf("Create(first) error = %v", err)
	}
	if _, err := service.Create("/", CreateOptions{
		Type:    "note",
		Name:    "English",
		Content: &secondContent,
		Metadata: map[string]any{
			"tags": []string{"exam"},
		},
	}); err != nil {
		t.Fatalf("Create(second) error = %v", err)
	}

	favorites, err := service.List("/", &ListOptions{IsFavorite: boolPtr(true)})
	if err != nil {
		t.Fatalf("List(favorite) error = %v", err)
	}
	if len(favorites) != 1 || favorites[0].ID != first.ID {
		t.Fatalf("unexpected favorite list: %+v", favorites)
	}

	tagged, err := service.List("/", &ListOptions{Tags: []string{"math"}})
	if err != nil {
		t.Fatalf("List(tags) error = %v", err)
	}
	if len(tagged) != 1 || tagged[0].ID != first.ID {
		t.Fatalf("unexpected tagged list: %+v", tagged)
	}

	search, err := service.Search("grammar", nil)
	if err != nil {
		t.Fatalf("Search() error = %v", err)
	}
	if len(search) != 1 || search[0].Name != "English" {
		t.Fatalf("unexpected search results: %+v", search)
	}

	paged, err := service.List("/", &ListOptions{SortBy: strPtr("name"), SortOrder: strPtr("asc"), Limit: intPtr(1)})
	if err != nil {
		t.Fatalf("List(paged) error = %v", err)
	}
	if len(paged) != 1 || paged[0].Name != "English" {
		t.Fatalf("unexpected paged list: %+v", paged)
	}
}

func TestNotesSearchAndListTagsUseDstuNotes(t *testing.T) {
	service := newTestService(t)
	limitsContent := "calculus limits theorem and derivative notes"
	geometryContent := "geometry proof practice"
	writingContent := "essay outline"

	if _, err := service.Create("/", CreateOptions{
		Type:    "note",
		Name:    "Limits",
		Content: &limitsContent,
		Metadata: map[string]any{
			"tags": []string{"exam", "math"},
		},
	}); err != nil {
		t.Fatalf("Create(limits) error = %v", err)
	}
	if _, err := service.Create("/", CreateOptions{
		Type:    "note",
		Name:    "Geometry",
		Content: &geometryContent,
		Metadata: map[string]any{
			"tags": []string{"Exam", "math"},
		},
	}); err != nil {
		t.Fatalf("Create(geometry) error = %v", err)
	}
	if _, err := service.Create("/", CreateOptions{
		Type:    "note",
		Name:    "Writing",
		Content: &writingContent,
		Metadata: map[string]any{
			"tags": []string{"language"},
		},
	}); err != nil {
		t.Fatalf("Create(writing) error = %v", err)
	}

	tags, err := service.ListTags()
	if err != nil {
		t.Fatalf("ListTags() error = %v", err)
	}
	if len(tags) != 3 || tags[0] != "exam" || tags[1] != "math" || tags[2] != "language" {
		t.Fatalf("unexpected tags: %+v", tags)
	}

	mathHits, err := service.NotesSearch("tag:math", 50)
	if err != nil {
		t.Fatalf("NotesSearch(tag) error = %v", err)
	}
	if len(mathHits) != 2 || mathHits[0].Title != "Limits" || mathHits[1].Title != "Geometry" {
		t.Fatalf("unexpected math hits: %+v", mathHits)
	}

	limitedHits, err := service.NotesSearch("limits", 1)
	if err != nil {
		t.Fatalf("NotesSearch(limits) error = %v", err)
	}
	if len(limitedHits) != 1 || limitedHits[0].Title != "Limits" || limitedHits[0].Snippet == nil || !strings.Contains(strings.ToLower(*limitedHits[0].Snippet), "limits") {
		t.Fatalf("unexpected limited search hit: %+v", limitedHits)
	}

	comboHits, err := service.NotesSearch("tag:math proof", 50)
	if err != nil {
		t.Fatalf("NotesSearch(combo) error = %v", err)
	}
	if len(comboHits) != 1 || comboHits[0].Title != "Geometry" {
		t.Fatalf("unexpected combo hits: %+v", comboHits)
	}
}

func TestSetMetadataFavoriteDeleteAndPersistence(t *testing.T) {
	dir := t.TempDir()
	service, err := NewService(dir)
	if err != nil {
		t.Fatalf("NewService() error = %v", err)
	}
	content := "body"
	node, err := service.Create("/", CreateOptions{Type: "note", Name: "Persist", Content: &content})
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}
	if _, err := service.SetMetadata(node.Path, map[string]any{"title": "Renamed", "tags": []string{"saved"}}); err != nil {
		t.Fatalf("SetMetadata() error = %v", err)
	}
	if _, err := service.SetFavorite(node.Path, true); err != nil {
		t.Fatalf("SetFavorite() error = %v", err)
	}

	reloaded, err := NewService(dir)
	if err != nil {
		t.Fatalf("NewService(reloaded) error = %v", err)
	}
	got, err := reloaded.Get(node.Path)
	if err != nil {
		t.Fatalf("Get(reloaded) error = %v", err)
	}
	if got == nil || got.Name != "Renamed" || !metadataFavorite(got.Metadata) || len(metadataTags(got.Metadata)) != 1 {
		t.Fatalf("unexpected reloaded metadata: %+v", got)
	}

	deleted, err := reloaded.Delete(node.Path)
	if err != nil {
		t.Fatalf("Delete() error = %v", err)
	}
	if !deleted {
		t.Fatal("Delete() = false, want true")
	}
	missing, err := reloaded.Get(node.Path)
	if err != nil {
		t.Fatalf("Get(deleted) error = %v", err)
	}
	if missing != nil {
		t.Fatalf("Get(deleted) = %+v, want nil", missing)
	}
}

func TestCanvasNoteContentMutationsSyncHybridVfs(t *testing.T) {
	dir := t.TempDir()
	vfsService, err := vfs.NewService(dir)
	if err != nil {
		t.Fatalf("vfs.NewService() error = %v", err)
	}
	service, err := NewService(dir, vfsService)
	if err != nil {
		t.Fatalf("NewService() error = %v", err)
	}

	content := "# Title\n\nIntro\n## Code\nold line\n### Detail\nold detail\n## Summary\nold summary"
	node, err := service.Create("/", CreateOptions{
		Type:    "note",
		Name:    "Canvas",
		Content: &content,
	})
	if err != nil {
		t.Fatalf("Create(canvas note) error = %v", err)
	}

	section, err := service.CanvasReadContent(node.ID, strPtr("Code"))
	if err != nil {
		t.Fatalf("CanvasReadContent(section) error = %v", err)
	}
	if !strings.Contains(section, "old line") || !strings.Contains(section, "### Detail") || strings.Contains(section, "Summary") {
		t.Fatalf("unexpected section content: %q", section)
	}

	if err := service.CanvasAppendContent(node.ID, "added line", strPtr("## Code")); err != nil {
		t.Fatalf("CanvasAppendContent(section) error = %v", err)
	}
	appended, err := service.GetContent(node.Path)
	if err != nil {
		t.Fatalf("GetContent(appended) error = %v", err)
	}
	newLineIndex := strings.Index(appended, "added line")
	summaryIndex := strings.Index(appended, "## Summary")
	if newLineIndex < 0 || summaryIndex < 0 || newLineIndex > summaryIndex {
		t.Fatalf("append should insert before next same-level heading, got: %q", appended)
	}

	count, err := service.CanvasReplaceContent(node.ID, "old", "new", false)
	if err != nil {
		t.Fatalf("CanvasReplaceContent(literal) error = %v", err)
	}
	if count != 3 {
		t.Fatalf("literal replacement count = %d, want 3", count)
	}
	count, err = service.CanvasReplaceContent(node.ID, `new (line|detail)`, "fresh $1", true)
	if err != nil {
		t.Fatalf("CanvasReplaceContent(regex) error = %v", err)
	}
	if count != 2 {
		t.Fatalf("regex replacement count = %d, want 2", count)
	}
	replaced, err := service.GetContent(node.Path)
	if err != nil {
		t.Fatalf("GetContent(replaced) error = %v", err)
	}
	if !strings.Contains(replaced, "fresh line") || !strings.Contains(replaced, "fresh detail") || !strings.Contains(replaced, "new summary") {
		t.Fatalf("unexpected replaced content: %q", replaced)
	}

	if err := service.CanvasSetContent(node.ID, "final content"); err != nil {
		t.Fatalf("CanvasSetContent() error = %v", err)
	}
	finalContent, err := service.GetContent(node.Path)
	if err != nil {
		t.Fatalf("GetContent(final) error = %v", err)
	}
	if finalContent != "final content" {
		t.Fatalf("final content = %q", finalContent)
	}

	refData, err := vfsService.GetResourceRefs(vfs.GetResourceRefsInput{SourceIDs: []string{node.SourceID}})
	if err != nil {
		t.Fatalf("GetResourceRefs() error = %v", err)
	}
	resolved, err := vfsService.ResolveResourceRefs(refData.Refs)
	if err != nil {
		t.Fatalf("ResolveResourceRefs() error = %v", err)
	}
	if len(resolved) != 1 || !resolved[0].Found || resolved[0].Content == nil || *resolved[0].Content != "final content" {
		t.Fatalf("unexpected resolved VFS content: %+v", resolved)
	}
}

func TestImportMarkdownUsesHintAndFolderMetadata(t *testing.T) {
	service := newTestService(t)
	path := filepath.Join(t.TempDir(), "ignored.md")
	if err := os.WriteFile(path, []byte("\xef\xbb\xbf# Heading\n\nbody"), 0o600); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}
	titleHint := "Linear Algebra.md"
	folderID := "folder_123"

	node, err := service.ImportMarkdown(ImportMarkdownRequest{
		FilePath:  path,
		TitleHint: &titleHint,
		FolderID:  &folderID,
	})
	if err != nil {
		t.Fatalf("ImportMarkdown() error = %v", err)
	}
	if node.Name != "Linear Algebra" || node.Metadata["folderId"] != folderID {
		t.Fatalf("unexpected imported node: %+v", node)
	}
	content, err := service.GetContent(node.Path)
	if err != nil {
		t.Fatalf("GetContent() error = %v", err)
	}
	if content != "# Heading\n\nbody" {
		t.Fatalf("content = %q", content)
	}
}

func TestImportMarkdownFallsBackToHeadingForGenericTitle(t *testing.T) {
	service := newTestService(t)
	path := filepath.Join(t.TempDir(), "446.md")
	if err := os.WriteFile(path, []byte("# Real Title\n\nbody"), 0o600); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	node, err := service.ImportMarkdown(ImportMarkdownRequest{FilePath: path})
	if err != nil {
		t.Fatalf("ImportMarkdown() error = %v", err)
	}
	if node.Name != "Real Title" {
		t.Fatalf("node.Name = %q", node.Name)
	}
}

func TestImportMarkdownBatchCollectsFailures(t *testing.T) {
	service := newTestService(t)
	dir := t.TempDir()
	okPath := filepath.Join(dir, "ok.md")
	missingPath := filepath.Join(dir, "missing.md")
	if err := os.WriteFile(okPath, []byte("body"), 0o600); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	response, err := service.ImportMarkdownBatch(ImportMarkdownBatchRequest{
		Items: []ImportMarkdownBatchItem{
			{FilePath: okPath},
			{FilePath: missingPath},
		},
	})
	if err != nil {
		t.Fatalf("ImportMarkdownBatch() error = %v", err)
	}
	if len(response.Imported) != 1 || len(response.Failed) != 1 || response.Failed[0].FilePath != missingPath {
		t.Fatalf("unexpected batch response: %+v", response)
	}
}

func boolPtr(value bool) *bool {
	return &value
}

func strPtr(value string) *string {
	return &value
}

func intPtr(value int) *int {
	return &value
}

type dstuTestEvent struct {
	name    string
	payload any
}

func textbookImportStages(events []dstuTestEvent) []string {
	stages := []string{}
	for _, event := range events {
		if event.name != "textbook-import-progress" {
			continue
		}
		payload, ok := event.payload.(textbookImportProgressPayload)
		if !ok {
			continue
		}
		stages = append(stages, payload.Stage)
	}
	return stages
}
