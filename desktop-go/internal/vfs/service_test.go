package vfs

import (
	"encoding/base64"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"
)

func newTestService(t *testing.T) *Service {
	t.Helper()
	service, err := NewService(t.TempDir())
	if err != nil {
		t.Fatalf("NewService() error = %v", err)
	}
	return service
}

func withPdfRasterPreviewRenderer(t *testing.T, renderer pdfRasterPreviewRendererFunc) {
	t.Helper()
	previous := pdfRasterPreviewRenderer
	pdfRasterPreviewRenderer = renderer
	t.Cleanup(func() {
		pdfRasterPreviewRenderer = previous
	})
}

func fakePdfRasterPreviewRenderer(t *testing.T, relativePath string, pageBytes []byte) pdfRasterPreviewRendererFunc {
	t.Helper()
	return func(service *Service, _ Resource, _ map[string]any, now int64) (map[string]any, bool) {
		t.Helper()
		absolutePath, err := service.resolveLibraryPath(relativePath)
		if err != nil {
			t.Fatalf("resolveLibraryPath(fake raster preview) error = %v", err)
		}
		if err := os.MkdirAll(filepath.Dir(absolutePath), 0o700); err != nil {
			t.Fatalf("MkdirAll(fake raster preview) error = %v", err)
		}
		if err := os.WriteFile(absolutePath, pageBytes, 0o600); err != nil {
			t.Fatalf("WriteFile(fake raster preview) error = %v", err)
		}
		previewJSON, err := json.Marshal(map[string]any{
			"pages": []map[string]any{{
				"pageIndex": 0,
				"path":      relativePath,
				"width":     120,
				"height":    160,
				"mimeType":  pdfRasterPreviewMimeType,
				"source":    pdfRasterPreviewSource,
			}},
			"totalPages": 1,
			"source":     pdfRasterPreviewSource,
			"mimeType":   pdfRasterPreviewMimeType,
		})
		if err != nil {
			t.Fatalf("Marshal(fake raster preview) error = %v", err)
		}
		return map[string]any{
			"previewJson":         string(previewJSON),
			"previewSource":       pdfRasterPreviewSource,
			"previewGeneratedAt":  formatMillis(now),
			"previewPageCount":    1,
			"previewMimeType":     pdfRasterPreviewMimeType,
			"pageCount":           1,
			"pageRenderingStatus": "completed",
			"pageRenderingSource": pdfRasterPreviewSource,
			"pageRenderingError":  "",
			"rasterPreviewStatus": "completed",
			"rasterPreviewSource": pdfRasterPreviewSource,
			"rasterPreviewError":  "",
		}, true
	}
}

func assertRasterPreviewJSON(t *testing.T, resource Resource) {
	t.Helper()
	raw := metadataString(resource.Metadata, "previewJson", "")
	if raw == "" {
		t.Fatalf("expected raster previewJson, got metadata %+v", resource.Metadata)
	}
	var preview map[string]any
	if err := json.Unmarshal([]byte(raw), &preview); err != nil {
		t.Fatalf("Unmarshal(raster previewJson) error = %v", err)
	}
	if preview["source"] != pdfRasterPreviewSource || preview["mimeType"] != pdfRasterPreviewMimeType {
		t.Fatalf("unexpected raster preview source/mime: %+v", preview)
	}
	if totalPages, ok := preview["totalPages"].(float64); !ok || totalPages != 1 {
		t.Fatalf("expected raster totalPages=1, got %+v", preview["totalPages"])
	}
	pages, ok := preview["pages"].([]any)
	if !ok || len(pages) != 1 {
		t.Fatalf("expected one raster preview page, got %+v", preview["pages"])
	}
	page, ok := pages[0].(map[string]any)
	if !ok {
		t.Fatalf("unexpected raster page shape: %+v", pages[0])
	}
	if page["source"] != pdfRasterPreviewSource || page["mimeType"] != pdfRasterPreviewMimeType {
		t.Fatalf("unexpected raster page source/mime: %+v", page)
	}
	if pageIndex, ok := page["pageIndex"].(float64); !ok || pageIndex != 0 {
		t.Fatalf("expected raster pageIndex=0, got %+v", page["pageIndex"])
	}
	path, ok := page["path"].(string)
	if !ok || strings.TrimSpace(path) == "" || filepath.IsAbs(path) || strings.Contains(filepath.Clean(path), "..") {
		t.Fatalf("expected safe relative raster preview path, got %+v", page["path"])
	}
	if width, ok := page["width"].(float64); !ok || width <= 0 {
		t.Fatalf("expected positive raster page width, got %+v", page["width"])
	}
	if height, ok := page["height"].(float64); !ok || height <= 0 {
		t.Fatalf("expected positive raster page height, got %+v", page["height"])
	}
}

func TestCreateOrReuseStoresVisibleFileAndHydratesData(t *testing.T) {
	service := newTestService(t)
	sourceID := "note_1"
	result, err := service.CreateOrReuse(CreateResourceInput{
		Type:     "note",
		Data:     "# Hello",
		SourceID: &sourceID,
		Metadata: map[string]any{"name": "hello.md"},
	})
	if err != nil {
		t.Fatalf("CreateOrReuse() error = %v", err)
	}
	if result.ResourceID == "" || result.Hash == "" || !result.IsNew {
		t.Fatalf("unexpected create result: %+v", result)
	}

	resource, err := service.GetResource(result.ResourceID)
	if err != nil {
		t.Fatalf("GetResource() error = %v", err)
	}
	if resource == nil || resource.Data == nil || *resource.Data != "# Hello" || resource.StorageMode != "external" {
		t.Fatalf("unexpected resource: %+v", resource)
	}
	if resource.ExternalPath == nil {
		t.Fatalf("resource missing visible path: %+v", resource)
	}
	absolute, err := service.GetResourcePath(sourceID)
	if err != nil {
		t.Fatalf("GetResourcePath() error = %v", err)
	}
	if absolute == nil {
		t.Fatal("GetResourcePath() returned nil")
	}
	if _, err := os.Stat(*absolute); err != nil {
		t.Fatalf("visible resource file missing: %v", err)
	}
	if !filepath.IsAbs(*absolute) {
		t.Fatalf("resource path is not absolute: %s", *absolute)
	}
}

func TestCreateOrReuseDedupesByTypeAndHash(t *testing.T) {
	service := newTestService(t)
	first, err := service.CreateOrReuse(CreateResourceInput{Type: "retrieval", Data: "same"})
	if err != nil {
		t.Fatalf("CreateOrReuse(first) error = %v", err)
	}
	second, err := service.CreateOrReuse(CreateResourceInput{Type: "retrieval", Data: "same"})
	if err != nil {
		t.Fatalf("CreateOrReuse(second) error = %v", err)
	}
	if second.IsNew || second.ResourceID != first.ResourceID || second.Hash != first.Hash {
		t.Fatalf("resource was not reused: first=%+v second=%+v", first, second)
	}
	image, err := service.CreateOrReuse(CreateResourceInput{Type: "image", Data: "same"})
	if err != nil {
		t.Fatalf("CreateOrReuse(image) error = %v", err)
	}
	if image.ResourceID == first.ResourceID {
		t.Fatalf("different resource type should not reuse the same id: %+v", image)
	}
}

func TestReferenceCountingAndPersistence(t *testing.T) {
	dir := t.TempDir()
	service, err := NewService(dir)
	if err != nil {
		t.Fatalf("NewService() error = %v", err)
	}
	sourceID := "card_1"
	created, err := service.CreateOrReuse(CreateResourceInput{Type: "file", Data: base64.StdEncoding.EncodeToString([]byte("file bytes")), SourceID: &sourceID})
	if err != nil {
		t.Fatalf("CreateOrReuse() error = %v", err)
	}
	if count, err := service.IncrementRef(created.ResourceID); err != nil || count != 1 {
		t.Fatalf("IncrementRef() = %d, %v", count, err)
	}
	if count, err := service.IncrementRef(created.ResourceID); err != nil || count != 2 {
		t.Fatalf("IncrementRef(second) = %d, %v", count, err)
	}
	if count, err := service.DecrementRef(created.ResourceID); err != nil || count != 1 {
		t.Fatalf("DecrementRef() = %d, %v", count, err)
	}
	if count, err := service.GetResourceRefCount(sourceID); err != nil || count != 1 {
		t.Fatalf("GetResourceRefCount(source) = %d, %v", count, err)
	}

	reloaded, err := NewService(dir)
	if err != nil {
		t.Fatalf("NewService(reloaded) error = %v", err)
	}
	exists, err := reloaded.ResourceExists(created.ResourceID)
	if err != nil || !exists {
		t.Fatalf("ResourceExists(reloaded) = %v, %v", exists, err)
	}
	resource, err := reloaded.GetResource(created.ResourceID)
	if err != nil {
		t.Fatalf("GetResource(reloaded) error = %v", err)
	}
	if resource == nil || resource.Data == nil || *resource.Data != base64.StdEncoding.EncodeToString([]byte("file bytes")) {
		t.Fatalf("unexpected reloaded resource: %+v", resource)
	}
}

func TestUpdateResourceHashBySource(t *testing.T) {
	service := newTestService(t)
	sourceID := "note_hash"
	created, err := service.CreateOrReuse(CreateResourceInput{Type: "note", Data: "old", SourceID: &sourceID})
	if err != nil {
		t.Fatalf("CreateOrReuse() error = %v", err)
	}
	if ok, err := service.UpdateResourceHash(sourceID, "new_hash"); err != nil || !ok {
		t.Fatalf("UpdateResourceHash() = %v, %v", ok, err)
	}
	resource, err := service.GetResource(created.ResourceID)
	if err != nil {
		t.Fatalf("GetResource() error = %v", err)
	}
	if resource == nil || resource.Hash != "new_hash" {
		t.Fatalf("hash was not updated: %+v", resource)
	}
}

func TestResourceSyncNoteCreatesSourceStableHybridResource(t *testing.T) {
	service := newTestService(t)
	first, err := service.ResourceSyncNote("note_resource_sync_1")
	if err != nil {
		t.Fatalf("ResourceSyncNote(first) error = %v", err)
	}
	if first.ResourceID == "" || first.Hash == "" || !first.IsNew {
		t.Fatalf("unexpected first sync result: %+v", first)
	}

	second, err := service.ResourceSyncNote("note_resource_sync_1")
	if err != nil {
		t.Fatalf("ResourceSyncNote(second) error = %v", err)
	}
	if second.IsNew || second.ResourceID != first.ResourceID || second.Hash != first.Hash {
		t.Fatalf("expected source-stable note reuse: first=%+v second=%+v", first, second)
	}

	resource, err := service.GetResource("note_resource_sync_1")
	if err != nil {
		t.Fatalf("GetResource(source id) error = %v", err)
	}
	if resource == nil || resource.SourceID == nil || *resource.SourceID != "note_resource_sync_1" || resource.Type != "note" {
		t.Fatalf("unexpected synced note resource: %+v", resource)
	}
	if resource.Data == nil || !strings.Contains(*resource.Data, "Source note: note_resource_sync_1") {
		t.Fatalf("synced note content missing source marker: %+v", resource)
	}
	if resource.Metadata["syncMode"] != "go-hybrid-vfs-source-stable" {
		t.Fatalf("expected sync metadata, got %+v", resource.Metadata)
	}

	path, err := service.GetResourcePath("note_resource_sync_1")
	if err != nil {
		t.Fatalf("GetResourcePath() error = %v", err)
	}
	if path == nil || !filepath.IsAbs(*path) {
		t.Fatalf("expected visible resource path, got %v", path)
	}
}

func TestResourceCheckSyncNeededComparesExistingHash(t *testing.T) {
	service := newTestService(t)
	missing, err := service.ResourceCheckSyncNeeded("note", "missing_source", nil)
	if err != nil {
		t.Fatalf("ResourceCheckSyncNeeded(missing) error = %v", err)
	}
	if !missing.NeedsSync || missing.ExistingResourceID != nil || missing.ExistingHash != nil {
		t.Fatalf("missing source should need sync: %+v", missing)
	}

	synced, err := service.ResourceSyncExam("exam_sync_1")
	if err != nil {
		t.Fatalf("ResourceSyncExam() error = %v", err)
	}
	matching, err := service.ResourceCheckSyncNeeded("exam", "exam_sync_1", &synced.Hash)
	if err != nil {
		t.Fatalf("ResourceCheckSyncNeeded(matching) error = %v", err)
	}
	if matching.NeedsSync || matching.ExistingResourceID == nil || *matching.ExistingResourceID != synced.ResourceID || matching.ExistingHash == nil || *matching.ExistingHash != synced.Hash {
		t.Fatalf("matching hash should be clean: %+v", matching)
	}

	differentHash := "different_hash"
	changed, err := service.ResourceCheckSyncNeeded("exam", "exam_sync_1", &differentHash)
	if err != nil {
		t.Fatalf("ResourceCheckSyncNeeded(changed) error = %v", err)
	}
	if !changed.NeedsSync || changed.ExistingHash == nil || *changed.ExistingHash != synced.Hash {
		t.Fatalf("different hash should need sync: %+v", changed)
	}

	wrongType, err := service.ResourceCheckSyncNeeded("note", "exam_sync_1", &synced.Hash)
	if err != nil {
		t.Fatalf("ResourceCheckSyncNeeded(wrong type) error = %v", err)
	}
	if !wrongType.NeedsSync || wrongType.ExistingResourceID != nil {
		t.Fatalf("wrong type should behave like missing source: %+v", wrongType)
	}
}

func TestResourceSyncTextbookPagesUsesRangeStableSourceID(t *testing.T) {
	service := newTestService(t)
	results, err := service.ResourceSyncTextbookPages("textbook_sync_1", []int{2, 4})
	if err != nil {
		t.Fatalf("ResourceSyncTextbookPages() error = %v", err)
	}
	if len(results) != 1 || !results[0].IsNew {
		t.Fatalf("unexpected textbook sync result: %+v", results)
	}

	resource, err := service.GetResource("textbook_sync_1:pages:2-4")
	if err != nil {
		t.Fatalf("GetResource(page source) error = %v", err)
	}
	if resource == nil || resource.Type != "textbook" || resource.Data == nil || !strings.Contains(*resource.Data, "Page range: 2-4") {
		t.Fatalf("unexpected synced textbook resource: %+v", resource)
	}
	if resource.Metadata["textbookId"] != "textbook_sync_1" {
		t.Fatalf("expected textbook metadata, got %+v", resource.Metadata)
	}

	reused, err := service.ResourceSyncTextbookPages("textbook_sync_1", []int{2, 4})
	if err != nil {
		t.Fatalf("ResourceSyncTextbookPages(reused) error = %v", err)
	}
	if len(reused) != 1 || reused[0].IsNew || reused[0].ResourceID != results[0].ResourceID {
		t.Fatalf("expected page range reuse: first=%+v reused=%+v", results, reused)
	}

	if _, err := service.ResourceSyncTextbookPages("textbook_sync_1", []int{5, 3}); err == nil {
		t.Fatal("invalid page range should return an error")
	}
}

func TestGetResourceRefsReturnsStableRefsAndTruncates(t *testing.T) {
	service := newTestService(t)
	firstSourceID := "note_ref_1"
	secondSourceID := "note_ref_2"
	first, err := service.CreateOrReuse(CreateResourceInput{
		Type:     "note",
		Data:     "first note",
		SourceID: &firstSourceID,
		Metadata: map[string]any{"name": "First Note", "snippet": "first preview"},
	})
	if err != nil {
		t.Fatalf("CreateOrReuse(first) error = %v", err)
	}
	if _, err := service.CreateOrReuse(CreateResourceInput{
		Type:     "note",
		Data:     "second note",
		SourceID: &secondSourceID,
		Metadata: map[string]any{"name": "Second Note"},
	}); err != nil {
		t.Fatalf("CreateOrReuse(second) error = %v", err)
	}

	allRefs, err := service.GetResourceRefs(GetResourceRefsInput{
		SourceIDs: []string{firstSourceID, secondSourceID, "missing"},
		MaxItems:  10,
	})
	if err != nil {
		t.Fatalf("GetResourceRefs(all) error = %v", err)
	}
	if allRefs.TotalCount != 2 || allRefs.Truncated {
		t.Fatalf("unexpected all refs metadata: %+v", allRefs)
	}
	if len(allRefs.Refs) != 2 {
		t.Fatalf("expected 2 refs, got %+v", allRefs.Refs)
	}
	if allRefs.Refs[0].SourceID != firstSourceID || allRefs.Refs[0].ResourceID == nil || *allRefs.Refs[0].ResourceID != first.ResourceID {
		t.Fatalf("unexpected first ref identity: %+v", allRefs.Refs[0])
	}
	if allRefs.Refs[0].ResourceHash != first.Hash || allRefs.Refs[0].Type != "note" || allRefs.Refs[0].Name != "First Note" {
		t.Fatalf("unexpected first ref fields: %+v", allRefs.Refs[0])
	}
	if allRefs.Refs[0].Snippet == nil || *allRefs.Refs[0].Snippet != "first preview" {
		t.Fatalf("expected snippet on first ref: %+v", allRefs.Refs[0])
	}
	byResourceID, err := service.GetResourceRefs(GetResourceRefsInput{
		SourceIDs: []string{first.ResourceID},
		MaxItems:  1,
	})
	if err != nil {
		t.Fatalf("GetResourceRefs(by resource id) error = %v", err)
	}
	if len(byResourceID.Refs) != 1 || byResourceID.Refs[0].SourceID != firstSourceID {
		t.Fatalf("expected resource id lookup to preserve stable source id: %+v", byResourceID)
	}

	limitedRefs, err := service.GetResourceRefs(GetResourceRefsInput{
		SourceIDs: []string{firstSourceID, secondSourceID},
		MaxItems:  1,
	})
	if err != nil {
		t.Fatalf("GetResourceRefs(limited) error = %v", err)
	}
	if len(limitedRefs.Refs) != 1 || limitedRefs.TotalCount != 2 || !limitedRefs.Truncated {
		t.Fatalf("unexpected limited refs: %+v", limitedRefs)
	}
}

func TestResolveResourceRefsHydratesFoundAndMissingResources(t *testing.T) {
	service := newTestService(t)
	sourceID := "note_resolve_1"
	created, err := service.CreateOrReuse(CreateResourceInput{
		Type:     "note",
		Data:     "resolve me",
		SourceID: &sourceID,
		Metadata: map[string]any{"name": "Resolve Me", "size": float64(10)},
	})
	if err != nil {
		t.Fatalf("CreateOrReuse() error = %v", err)
	}
	refData, err := service.GetResourceRefs(GetResourceRefsInput{SourceIDs: []string{sourceID}})
	if err != nil {
		t.Fatalf("GetResourceRefs() error = %v", err)
	}
	if len(refData.Refs) != 1 {
		t.Fatalf("expected 1 ref, got %+v", refData)
	}
	missingID := "missing_resource"
	refs := append(refData.Refs, ResourceRef{
		SourceID:     missingID,
		ResourceHash: "missing_hash",
		Type:         "note",
		Name:         "Missing",
	})

	resolved, err := service.ResolveResourceRefs(refs)
	if err != nil {
		t.Fatalf("ResolveResourceRefs() error = %v", err)
	}
	if len(resolved) != 2 {
		t.Fatalf("expected 2 resolved resources, got %+v", resolved)
	}
	found := resolved[0]
	if !found.Found || found.SourceID != sourceID || found.ResourceHash != created.Hash || found.Type != "note" || found.Name != "Resolve Me" {
		t.Fatalf("unexpected found resource: %+v", found)
	}
	if found.Content == nil || *found.Content != "resolve me" {
		t.Fatalf("expected hydrated content, got %+v", found)
	}
	if found.ByteSize == nil || *found.ByteSize != 10 {
		t.Fatalf("expected metadata byte size, got %+v", found.ByteSize)
	}
	if !filepath.IsAbs(found.Path) || !strings.Contains(filepath.ToSlash(found.Path), "/vfs_resources/") {
		t.Fatalf("expected visible absolute vfs path, got %q", found.Path)
	}
	if _, err := os.Stat(found.Path); err != nil {
		t.Fatalf("resolved path does not exist: %v", err)
	}

	missing := resolved[1]
	if missing.Found || missing.SourceID != missingID || missing.Content != nil || missing.Path != "" {
		t.Fatalf("unexpected missing resource: %+v", missing)
	}
}

func TestUpdatePathCacheIsHybridVfsCompatibilityNoop(t *testing.T) {
	service := newTestService(t)
	count, err := service.UpdatePathCache("fld_any")
	if err != nil {
		t.Fatalf("UpdatePathCache() error = %v", err)
	}
	if count != 0 {
		t.Fatalf("UpdatePathCache() = %d, want 0", count)
	}
}

func TestUploadAttachmentStoresVisibleFileAndReturnsContent(t *testing.T) {
	service := newTestService(t)
	imageBytes := []byte("png bytes")
	content := base64.StdEncoding.EncodeToString(imageBytes)
	attachmentType := "image"
	result, err := service.UploadAttachment(UploadAttachmentInput{
		Name:           "diagram.png",
		MimeType:       "image/png",
		Base64Content:  content,
		AttachmentType: &attachmentType,
	})
	if err != nil {
		t.Fatalf("UploadAttachment() error = %v", err)
	}
	if !strings.HasPrefix(result.SourceID, "att_") || result.ResourceHash == "" || !result.IsNew {
		t.Fatalf("unexpected upload result: %+v", result)
	}
	if result.Attachment.ID != result.SourceID || result.Attachment.ResourceID == nil || result.Attachment.Type != "image" {
		t.Fatalf("unexpected attachment metadata: %+v", result.Attachment)
	}
	if result.Attachment.Name != "diagram.png" || result.Attachment.MimeType != "image/png" || result.Attachment.Size != int64(len(imageBytes)) {
		t.Fatalf("unexpected attachment fields: %+v", result.Attachment)
	}
	if result.ProcessingStatus == nil || *result.ProcessingStatus != "completed" || result.ProcessingPercent == nil || *result.ProcessingPercent != 100 {
		t.Fatalf("unexpected image processing state: %+v", result)
	}
	if len(result.ReadyModes) != 1 || result.ReadyModes[0] != "image" {
		t.Fatalf("expected image ready mode: %+v", result.ReadyModes)
	}

	contentResult, err := service.GetAttachmentContent(result.SourceID)
	if err != nil {
		t.Fatalf("GetAttachmentContent() error = %v", err)
	}
	if !contentResult.Found || contentResult.Content == nil || *contentResult.Content != content {
		t.Fatalf("unexpected attachment content: %+v", contentResult)
	}

	attachment, err := service.GetAttachment(result.SourceID)
	if err != nil {
		t.Fatalf("GetAttachment() error = %v", err)
	}
	if attachment == nil || attachment.ID != result.SourceID || attachment.ContentHash != result.ResourceHash {
		t.Fatalf("unexpected attachment lookup: %+v", attachment)
	}

	path, err := service.GetResourcePath(result.SourceID)
	if err != nil {
		t.Fatalf("GetResourcePath() error = %v", err)
	}
	if path == nil || !filepath.IsAbs(*path) {
		t.Fatalf("expected visible absolute path, got %v", path)
	}
	if bytes, err := os.ReadFile(*path); err != nil || string(bytes) != string(imageBytes) {
		t.Fatalf("visible attachment bytes mismatch: %q, %v", string(bytes), err)
	}
}

func TestUploadAttachmentDedupesOnlyAttachments(t *testing.T) {
	service := newTestService(t)
	content := base64.StdEncoding.EncodeToString([]byte("shared bytes"))
	regularSourceID := "img_regular"
	if _, err := service.CreateOrReuse(CreateResourceInput{
		Type:     "image",
		Data:     content,
		SourceID: &regularSourceID,
		Metadata: map[string]any{"name": "regular.png", "mimeType": "image/png"},
	}); err != nil {
		t.Fatalf("CreateOrReuse() error = %v", err)
	}

	first, err := service.UploadAttachment(UploadAttachmentInput{
		Name:          "upload.png",
		MimeType:      "image/png",
		Base64Content: content,
	})
	if err != nil {
		t.Fatalf("UploadAttachment(first) error = %v", err)
	}
	if !first.IsNew || !strings.HasPrefix(first.SourceID, "att_") {
		t.Fatalf("expected a new attachment source, got %+v", first)
	}

	second, err := service.UploadAttachment(UploadAttachmentInput{
		Name:          "upload-again.png",
		MimeType:      "image/png",
		Base64Content: content,
	})
	if err != nil {
		t.Fatalf("UploadAttachment(second) error = %v", err)
	}
	if second.IsNew || second.SourceID != first.SourceID || second.Attachment.ResourceID == nil || first.Attachment.ResourceID == nil || *second.Attachment.ResourceID != *first.Attachment.ResourceID {
		t.Fatalf("expected attachment dedupe: first=%+v second=%+v", first, second)
	}
}

func TestUploadPdfAttachmentReturnsLightweightProcessingState(t *testing.T) {
	service := newTestService(t)
	result, err := service.UploadAttachment(UploadAttachmentInput{
		Name:          "chapter.pdf",
		MimeType:      "application/pdf",
		Base64Content: base64.StdEncoding.EncodeToString([]byte("%PDF")),
	})
	if err != nil {
		t.Fatalf("UploadAttachment(pdf) error = %v", err)
	}
	if result.ProcessingStatus == nil || *result.ProcessingStatus != "page_compression" {
		t.Fatalf("unexpected pdf status: %+v", result)
	}
	if result.ProcessingPercent == nil || *result.ProcessingPercent != 25 {
		t.Fatalf("unexpected pdf percent: %+v", result.ProcessingPercent)
	}
	if len(result.ReadyModes) != 0 {
		t.Fatalf("expected no ready modes for lightweight pdf upload: %+v", result.ReadyModes)
	}
}

func TestGetAttachmentContentReturnsNotFound(t *testing.T) {
	service := newTestService(t)
	result, err := service.GetAttachmentContent("att_missing")
	if err != nil {
		t.Fatalf("GetAttachmentContent() error = %v", err)
	}
	if result.Found || result.Content != nil || result.Error != nil {
		t.Fatalf("unexpected missing result: %+v", result)
	}
}

func TestCompactIndexStatusReflectsHybridResources(t *testing.T) {
	service := newTestService(t)
	noteSourceID := "note_index_1"
	note, err := service.CreateOrReuse(CreateResourceInput{
		Type:     "note",
		Data:     "indexed note body",
		SourceID: &noteSourceID,
		Metadata: map[string]any{"name": "Indexed Note"},
	})
	if err != nil {
		t.Fatalf("CreateOrReuse(note) error = %v", err)
	}
	if _, err := service.UploadAttachment(UploadAttachmentInput{
		Name:          "scan.png",
		MimeType:      "image/png",
		Base64Content: base64.StdEncoding.EncodeToString([]byte("image bytes")),
	}); err != nil {
		t.Fatalf("UploadAttachment() error = %v", err)
	}
	deletedSourceID := "note_deleted_index_1"
	deleted, err := service.CreateOrReuse(CreateResourceInput{
		Type:     "note",
		Data:     "deleted note body",
		SourceID: &deletedSourceID,
		Metadata: map[string]any{"name": "Deleted Note", "status": "deleted", "deletedAt": "2026-06-08T00:00:00Z"},
	})
	if err != nil {
		t.Fatalf("CreateOrReuse(deleted note) error = %v", err)
	}

	summary, err := service.UnifiedIndexStatus()
	if err != nil {
		t.Fatalf("UnifiedIndexStatus() error = %v", err)
	}
	if summary.TotalUnits != 2 || summary.TextStats.Indexed != 1 || summary.MMStats.Pending != 1 {
		t.Fatalf("unexpected summary: %+v", summary)
	}

	units, err := service.GetResourceUnits(noteSourceID)
	if err != nil {
		t.Fatalf("GetResourceUnits() error = %v", err)
	}
	if len(units) != 1 || units[0].ResourceID != note.ResourceID || units[0].TextState != "indexed" || units[0].TextChunkCount != 1 {
		t.Fatalf("unexpected note units: %+v", units)
	}

	all, err := service.GetAllIndexStatus(GetIndexStatusInput{ResourceType: strPtr("note")})
	if err != nil {
		t.Fatalf("GetAllIndexStatus() error = %v", err)
	}
	if all.TotalResources != 1 || len(all.Resources) != 1 || all.Resources[0].TextIndexState != "indexed" || all.TextIndexedCount != 1 {
		t.Fatalf("unexpected resource index status: %+v", all)
	}
	deletedUnits, err := service.GetResourceUnits(deleted.ResourceID)
	if err != nil {
		t.Fatalf("GetResourceUnits(deleted) error = %v", err)
	}
	if len(deletedUnits) != 0 {
		t.Fatalf("expected deleted resource units to be hidden, got %+v", deletedUnits)
	}

	chunks, err := service.GetResourceTextChunks(noteSourceID)
	if err != nil {
		t.Fatalf("GetResourceTextChunks() error = %v", err)
	}
	if len(chunks) != 1 || chunks[0].TextContent == nil || *chunks[0].TextContent != "indexed note body" {
		t.Fatalf("unexpected text chunks: %+v", chunks)
	}

	ocrInfo, err := service.GetResourceOcrInfo(noteSourceID)
	if err != nil {
		t.Fatalf("GetResourceOcrInfo() error = %v", err)
	}
	if ocrInfo.HasOcr || ocrInfo.ActiveSource != "extracted" || ocrInfo.ExtractedText == nil || *ocrInfo.ExtractedText != "indexed note body" {
		t.Fatalf("unexpected OCR info: %+v", ocrInfo)
	}
}

func TestClearResourceOcrRemovesOcrMetadataAndKeepsExtractedText(t *testing.T) {
	service := newTestService(t)
	sourceID := "file_ocr_clear"
	_, err := service.CreateOrReuse(CreateResourceInput{
		Type:     "file",
		Data:     "indexed fallback body",
		SourceID: &sourceID,
		Metadata: map[string]any{
			"fileRecord":         true,
			"name":               "ocr.pdf",
			"mimeType":           "application/pdf",
			"status":             "active",
			"extractedText":      "native extracted text",
			"ocrText":            "real OCR text",
			"ocrPagesJson":       `[{"pageIndex":0,"text":"real OCR page","charCount":13,"isFailed":false}]`,
			"ocrPagesSource":     "real_ocr",
			"processingStatus":   "completed",
			"processingProgress": map[string]any{"stage": "completed", "readyModes": []string{"text", "image", "ocr"}},
			"indexStatus":        "indexed",
			"textIndexState":     "indexed",
		},
	})
	if err != nil {
		t.Fatalf("CreateOrReuse() error = %v", err)
	}

	before, err := service.GetResourceOcrInfo(sourceID)
	if err != nil {
		t.Fatalf("GetResourceOcrInfo(before) error = %v", err)
	}
	if !before.HasOcr || before.ActiveSource != "ocr" || before.OcrText == nil || len(before.OcrPages) != 1 {
		t.Fatalf("expected real OCR before clear, got %+v", before)
	}

	cleared, err := service.ClearResourceOcr(sourceID)
	if err != nil {
		t.Fatalf("ClearResourceOcr() error = %v", err)
	}
	if !cleared {
		t.Fatal("ClearResourceOcr() should report a changed resource")
	}

	after, err := service.GetResourceOcrInfo(sourceID)
	if err != nil {
		t.Fatalf("GetResourceOcrInfo(after) error = %v", err)
	}
	if after.HasOcr || after.ActiveSource != "extracted" || after.OcrText != nil || len(after.OcrPages) != 0 {
		t.Fatalf("expected OCR cleared with extracted text active, got %+v", after)
	}
	if after.ExtractedText == nil || *after.ExtractedText != "native extracted text" {
		t.Fatalf("ClearResourceOcr() should preserve extracted text, got %+v", after.ExtractedText)
	}

	resource, err := service.GetResource(sourceID)
	if err != nil {
		t.Fatalf("GetResource(after clear) error = %v", err)
	}
	if resource == nil {
		t.Fatal("expected resource after clear")
	}
	for _, key := range []string{"ocrText", "ocrPagesJson", "ocrPagesSource"} {
		if _, ok := resource.Metadata[key]; ok {
			t.Fatalf("expected %s to be removed, metadata=%+v", key, resource.Metadata)
		}
	}
	if metadataString(resource.Metadata, "extractedText", "") != "native extracted text" {
		t.Fatalf("expected extracted text to remain, metadata=%+v", resource.Metadata)
	}
	if metadataString(resource.Metadata, "processingStatus", "") != "ocr_processing" || metadataString(resource.Metadata, "indexStatus", "") != "pending" || metadataString(resource.Metadata, "textIndexState", "") != "pending" {
		t.Fatalf("expected OCR clear to reset processing/index metadata, got %+v", resource.Metadata)
	}
	progress, ok := resource.Metadata["processingProgress"].(map[string]any)
	if !ok || progress["stage"] != "ocr_processing" {
		t.Fatalf("expected OCR processing progress metadata, got %+v", resource.Metadata["processingProgress"])
	}

	if err := service.DeleteFile(sourceID); err != nil {
		t.Fatalf("DeleteFile() error = %v", err)
	}
	deletedInfo, err := service.GetResourceOcrInfo(sourceID)
	if err != nil {
		t.Fatalf("GetResourceOcrInfo(deleted) error = %v", err)
	}
	if deletedInfo.ActiveSource != "none" || deletedInfo.HasOcr || deletedInfo.ExtractedText != nil || len(deletedInfo.OcrPages) != 0 {
		t.Fatalf("deleted resource should not expose OCR info, got %+v", deletedInfo)
	}
	deletedCleared, err := service.ClearResourceOcr(sourceID)
	if err != nil {
		t.Fatalf("ClearResourceOcr(deleted) error = %v", err)
	}
	if deletedCleared {
		t.Fatal("ClearResourceOcr(deleted) should be a no-op")
	}
}

func TestRagSearchFindsHybridResourceTextAndMetadata(t *testing.T) {
	service := newTestService(t)
	noteSourceID := "note_search_1"
	created, err := service.CreateOrReuse(CreateResourceInput{
		Type:     "note",
		Data:     "Trigonometric identities help transform sine and cosine equations.",
		SourceID: &noteSourceID,
		Metadata: map[string]any{
			"name":     "Trigonometry Notebook",
			"tags":     []string{"math", "identity"},
			"folderId": "folder_math",
		},
	})
	if err != nil {
		t.Fatalf("CreateOrReuse(note) error = %v", err)
	}
	if _, err := service.CreateOrReuse(CreateResourceInput{
		Type: "image",
		Data: base64.StdEncoding.EncodeToString([]byte("image")),
		Metadata: map[string]any{
			"name":     "diagram.png",
			"mimeType": "image/png",
		},
	}); err != nil {
		t.Fatalf("CreateOrReuse(image) error = %v", err)
	}

	result, err := service.RagSearch(VfsRagSearchInput{
		Query:         "trigonometric",
		ResourceTypes: []string{"note"},
		FolderIDs:     []string{"folder_math"},
		TopK:          10,
	})
	if err != nil {
		t.Fatalf("RagSearch() error = %v", err)
	}
	if result.Count != 1 || len(result.Results) != 1 {
		t.Fatalf("expected one note result, got %+v", result)
	}
	match := result.Results[0]
	if match.ResourceID != created.ResourceID || match.SourceID == nil || *match.SourceID != noteSourceID {
		t.Fatalf("unexpected result identity: %+v", match)
	}
	if match.ResourceTitle == nil || *match.ResourceTitle != "Trigonometry Notebook" {
		t.Fatalf("unexpected result title: %+v", match)
	}
	if match.ResourceType == nil || *match.ResourceType != "note" {
		t.Fatalf("unexpected result type: %+v", match)
	}
	if !strings.Contains(strings.ToLower(match.ChunkText), "trigonometric") || match.Score <= 0 {
		t.Fatalf("unexpected result content/score: %+v", match)
	}

	filtered, err := service.RagSearch(VfsRagSearchInput{
		Query:         "trigonometric",
		ResourceTypes: []string{"image"},
		TopK:          10,
	})
	if err != nil {
		t.Fatalf("RagSearch(filtered) error = %v", err)
	}
	if filtered.Count != 0 || len(filtered.Results) != 0 {
		t.Fatalf("type filter should exclude note result: %+v", filtered)
	}
}

func TestListFilesReturnsHybridAttachments(t *testing.T) {
	service := newTestService(t)
	imageBytes := []byte("listed image bytes")
	uploaded, err := service.UploadAttachment(UploadAttachmentInput{
		Name:          "listed.png",
		MimeType:      "image/png",
		Base64Content: base64.StdEncoding.EncodeToString(imageBytes),
	})
	if err != nil {
		t.Fatalf("UploadAttachment() error = %v", err)
	}
	if _, err := service.CreateOrReuse(CreateResourceInput{
		Type: "note",
		Data: "not a file",
		Metadata: map[string]any{
			"name": "Plain note",
		},
	}); err != nil {
		t.Fatalf("CreateOrReuse(note) error = %v", err)
	}

	files, err := service.ListFiles(ListFilesInput{FileType: "image", Limit: 10})
	if err != nil {
		t.Fatalf("ListFiles() error = %v", err)
	}
	if len(files) != 1 {
		t.Fatalf("expected one image file, got %+v", files)
	}
	file := files[0]
	if file.ID != uploaded.SourceID || file.ResourceID == nil || *file.ResourceID != *uploaded.Attachment.ResourceID {
		t.Fatalf("unexpected file identity: %+v", file)
	}
	if file.FileName != "listed.png" || file.FileType != "image" || file.MimeType == nil || *file.MimeType != "image/png" {
		t.Fatalf("unexpected file metadata: %+v", file)
	}
	if file.Size != int64(len(imageBytes)) || file.SHA256 != uploaded.ResourceHash || file.BlobHash == nil || *file.BlobHash != uploaded.ResourceHash {
		t.Fatalf("unexpected file hash/size: %+v", file)
	}
	if file.OriginalPath == nil || !filepath.IsAbs(*file.OriginalPath) {
		t.Fatalf("expected visible absolute original path: %+v", file.OriginalPath)
	}

	documents, err := service.ListFiles(ListFilesInput{FileType: "document", Limit: 10})
	if err != nil {
		t.Fatalf("ListFiles(document) error = %v", err)
	}
	if len(documents) != 0 {
		t.Fatalf("document filter should exclude image attachment and note: %+v", documents)
	}
}

func TestUploadFileGetContentBookmarksAndSoftDelete(t *testing.T) {
	service := newTestService(t)
	content := "calculus notes about derivatives"
	fileType := "document"
	folderID := "folder_files"
	uploaded, err := service.UploadFile(UploadFileInput{
		Name:          "calculus.md",
		MimeType:      "text/markdown",
		Base64Content: base64.StdEncoding.EncodeToString([]byte(content)),
		FileType:      &fileType,
		FolderID:      &folderID,
	})
	if err != nil {
		t.Fatalf("UploadFile() error = %v", err)
	}
	if !uploaded.IsNew || !strings.HasPrefix(uploaded.SourceID, "file_") {
		t.Fatalf("unexpected upload result: %+v", uploaded)
	}
	if uploaded.File.ID != uploaded.SourceID || uploaded.File.ResourceID == nil || uploaded.File.FileName != "calculus.md" {
		t.Fatalf("unexpected file identity: %+v", uploaded.File)
	}
	if uploaded.File.FileType != "document" || uploaded.File.MimeType == nil || *uploaded.File.MimeType != "text/markdown" {
		t.Fatalf("unexpected file metadata: %+v", uploaded.File)
	}
	if uploaded.File.ExtractedText == nil || *uploaded.File.ExtractedText != content {
		t.Fatalf("expected extracted text from text upload: %+v", uploaded.File.ExtractedText)
	}
	if uploaded.IndexStatus == nil || !uploaded.IndexStatus.Queued || uploaded.IndexStatus.UnitsCreated != 1 {
		t.Fatalf("expected lightweight index status: %+v", uploaded.IndexStatus)
	}

	file, err := service.GetFile(uploaded.SourceID)
	if err != nil {
		t.Fatalf("GetFile() error = %v", err)
	}
	if file == nil || file.ID != uploaded.SourceID || file.SHA256 != uploaded.ResourceHash {
		t.Fatalf("unexpected GetFile result: %+v", file)
	}

	contentResult, err := service.GetFileContent(uploaded.SourceID)
	if err != nil {
		t.Fatalf("GetFileContent() error = %v", err)
	}
	if !contentResult.Found || contentResult.Content == nil || *contentResult.Content != base64.StdEncoding.EncodeToString([]byte(content)) {
		t.Fatalf("unexpected file content: %+v", contentResult)
	}

	bookmarks := []any{map[string]any{"id": "bm_1", "page": float64(1), "title": "Start", "createdAt": float64(123)}}
	ok, err := service.UpdateBookmarks(uploaded.SourceID, bookmarks)
	if err != nil || !ok {
		t.Fatalf("UpdateBookmarks() = %v, %v", ok, err)
	}
	withBookmarks, err := service.GetFile(uploaded.SourceID)
	if err != nil {
		t.Fatalf("GetFile(with bookmarks) error = %v", err)
	}
	if withBookmarks == nil || len(withBookmarks.Bookmarks) != 1 {
		t.Fatalf("expected bookmark metadata: %+v", withBookmarks)
	}

	if err := service.DeleteFile(uploaded.SourceID); err != nil {
		t.Fatalf("DeleteFile() error = %v", err)
	}
	deletedFile, err := service.GetFile(uploaded.SourceID)
	if err != nil {
		t.Fatalf("GetFile(deleted) error = %v", err)
	}
	if deletedFile != nil {
		t.Fatalf("soft-deleted file should not be active: %+v", deletedFile)
	}
	contentAfterDelete, err := service.GetFileContent(uploaded.SourceID)
	if err != nil {
		t.Fatalf("GetFileContent(deleted) error = %v", err)
	}
	if contentAfterDelete.Found || contentAfterDelete.Content != nil {
		t.Fatalf("deleted file content should be hidden: %+v", contentAfterDelete)
	}
	attachmentAfterDelete, err := service.GetAttachment(uploaded.SourceID)
	if err != nil {
		t.Fatalf("GetAttachment(deleted file) error = %v", err)
	}
	if attachmentAfterDelete != nil {
		t.Fatalf("deleted file should not be visible through attachment compatibility: %+v", attachmentAfterDelete)
	}
	attachmentContentAfterDelete, err := service.GetAttachmentContent(uploaded.SourceID)
	if err != nil {
		t.Fatalf("GetAttachmentContent(deleted file) error = %v", err)
	}
	if attachmentContentAfterDelete.Found || attachmentContentAfterDelete.Content != nil {
		t.Fatalf("deleted file content should not be visible through attachment compatibility: %+v", attachmentContentAfterDelete)
	}
	files, err := service.ListFiles(ListFilesInput{FileType: "document", Limit: 10})
	if err != nil {
		t.Fatalf("ListFiles(after delete) error = %v", err)
	}
	if len(files) != 0 {
		t.Fatalf("deleted file should not be listed: %+v", files)
	}
}

func TestLegacyAliasesResolveHybridVfsResources(t *testing.T) {
	service := newTestService(t)
	content := "legacy alias visible content"
	originalPath := "content://com.android.providers.media.documents/document/primary%3ADownload%2Flegacy.pdf"
	uploaded, err := service.UploadFile(UploadFileInput{
		Name:          "legacy.pdf",
		MimeType:      "application/pdf",
		Base64Content: base64.StdEncoding.EncodeToString([]byte(content)),
		Metadata:      map[string]any{"originalPath": originalPath},
	})
	if err != nil {
		t.Fatalf("UploadFile() error = %v", err)
	}
	if uploaded.File.ResourceID == nil {
		t.Fatalf("uploaded file missing resource id: %+v", uploaded.File)
	}
	resourceID := *uploaded.File.ResourceID

	resource, err := service.GetResource("vfs://" + resourceID)
	if err != nil {
		t.Fatalf("GetResource(legacy resource uri) error = %v", err)
	}
	if resource == nil || resource.Data == nil || *resource.Data != base64.StdEncoding.EncodeToString([]byte(content)) {
		t.Fatalf("legacy resource uri did not hydrate resource: %+v", resource)
	}
	if exists, err := service.ResourceExists("vfs-resource://lookup?resourceId=" + resourceID); err != nil || !exists {
		t.Fatalf("ResourceExists(legacy resource query) = %v, %v", exists, err)
	}
	if count, err := service.IncrementRef("dstu:///folder/" + uploaded.SourceID); err != nil || count != 1 {
		t.Fatalf("IncrementRef(legacy source path) = %d, %v", count, err)
	}
	if count, err := service.GetResourceRefCount("resource://by-hash?hash=" + uploaded.ResourceHash); err != nil || count != 1 {
		t.Fatalf("GetResourceRefCount(legacy hash query) = %d, %v", count, err)
	}

	bySource, err := service.GetFile("dstu:///library/" + uploaded.SourceID)
	if err != nil {
		t.Fatalf("GetFile(legacy source path) error = %v", err)
	}
	if bySource == nil || bySource.ID != uploaded.SourceID {
		t.Fatalf("legacy source path did not resolve file: %+v", bySource)
	}
	byHash, err := service.GetFileContent("resource://blob?hash=" + uploaded.ResourceHash)
	if err != nil {
		t.Fatalf("GetFileContent(legacy hash query) error = %v", err)
	}
	if !byHash.Found || byHash.Content == nil || *byHash.Content != base64.StdEncoding.EncodeToString([]byte(content)) {
		t.Fatalf("legacy hash query did not resolve file content: %+v", byHash)
	}
	byOriginalPath, err := service.GetFileContent(originalPath)
	if err != nil {
		t.Fatalf("GetFileContent(legacy original path) error = %v", err)
	}
	if !byOriginalPath.Found || byOriginalPath.Content == nil || *byOriginalPath.Content != base64.StdEncoding.EncodeToString([]byte(content)) {
		t.Fatalf("legacy original path did not resolve file content: %+v", byOriginalPath)
	}

	refData, err := service.GetResourceRefs(GetResourceRefsInput{SourceIDs: []string{"vfs://" + uploaded.SourceID}})
	if err != nil {
		t.Fatalf("GetResourceRefs(legacy source uri) error = %v", err)
	}
	if len(refData.Refs) != 1 || refData.Refs[0].ResourceID == nil || *refData.Refs[0].ResourceID != resourceID {
		t.Fatalf("legacy source uri did not resolve refs: %+v", refData)
	}
	refData.Refs[0].ResourceID = strPtr("vfs://" + resourceID)
	resolved, err := service.ResolveResourceRefs(refData.Refs)
	if err != nil {
		t.Fatalf("ResolveResourceRefs(legacy resource uri) error = %v", err)
	}
	if len(resolved) != 1 || !resolved[0].Found || resolved[0].Content == nil {
		t.Fatalf("legacy resource uri did not resolve ref content: %+v", resolved)
	}
}

func TestLegacyAliasesRespectSoftDelete(t *testing.T) {
	service := newTestService(t)
	uploaded, err := service.UploadFile(UploadFileInput{
		Name:          "deleted.md",
		MimeType:      "text/markdown",
		Base64Content: base64.StdEncoding.EncodeToString([]byte("deleted content")),
	})
	if err != nil {
		t.Fatalf("UploadFile() error = %v", err)
	}
	if uploaded.File.ResourceID == nil {
		t.Fatalf("uploaded file missing resource id: %+v", uploaded.File)
	}
	resourceURI := "vfs://" + *uploaded.File.ResourceID
	sourceURI := "dstu:///folder/" + uploaded.SourceID
	hashURI := "resource://blob?hash=" + uploaded.ResourceHash

	if err := service.DeleteFile(sourceURI); err != nil {
		t.Fatalf("DeleteFile(legacy source uri) error = %v", err)
	}
	for _, alias := range []string{resourceURI, sourceURI, hashURI} {
		content, err := service.GetFileContent(alias)
		if err != nil {
			t.Fatalf("GetFileContent(%s) error = %v", alias, err)
		}
		if content.Found || content.Content != nil {
			t.Fatalf("deleted content should be hidden through alias %s: %+v", alias, content)
		}
		attachment, err := service.GetAttachmentContent(alias)
		if err != nil {
			t.Fatalf("GetAttachmentContent(%s) error = %v", alias, err)
		}
		if attachment.Found || attachment.Content != nil {
			t.Fatalf("deleted attachment content should be hidden through alias %s: %+v", alias, attachment)
		}
	}
}

func TestPdfProcessingStatusReflectsHybridVfsCapabilities(t *testing.T) {
	service := newTestService(t)

	pdfWithoutText, err := service.UploadFile(UploadFileInput{
		Name:          "lesson.pdf",
		MimeType:      "application/pdf",
		Base64Content: base64.StdEncoding.EncodeToString([]byte("%PDF-1.4\n")),
	})
	if err != nil {
		t.Fatalf("UploadFile(pdf without text) error = %v", err)
	}
	pdfStatus, err := service.GetPdfProcessingStatus(pdfWithoutText.SourceID)
	if err != nil {
		t.Fatalf("GetPdfProcessingStatus(pdf without text) error = %v", err)
	}
	if pdfStatus.Stage != "completed_with_issues" || pdfStatus.Percent != 100 || pdfStatus.MediaType != "pdf" {
		t.Fatalf("unexpected PDF status: %+v", pdfStatus)
	}
	if len(pdfStatus.ReadyModes) != 0 || len(pdfStatus.FailedStages) < 2 {
		t.Fatalf("PDF without extracted text should not report ready modes: %+v", pdfStatus)
	}
	if pdfStatus.Progress.Stage != pdfStatus.Stage || pdfStatus.Progress.Percent != pdfStatus.Percent || pdfStatus.Progress.MediaType != pdfStatus.MediaType {
		t.Fatalf("progress should mirror top-level status: %+v", pdfStatus)
	}

	pdfWithText, err := service.UploadFile(UploadFileInput{
		Name:          "textbook.pdf",
		MimeType:      "application/pdf",
		Base64Content: base64.StdEncoding.EncodeToString([]byte("%PDF-1.4\n")),
		Metadata:      map[string]any{"extractedText": "chapter one summary"},
	})
	if err != nil {
		t.Fatalf("UploadFile(pdf with text) error = %v", err)
	}
	textStatus, err := service.GetPdfProcessingStatus(pdfWithText.SourceID)
	if err != nil {
		t.Fatalf("GetPdfProcessingStatus(pdf with text) error = %v", err)
	}
	if textStatus.Stage != "completed_with_issues" || textStatus.MediaType != "pdf" || !readyModesContain(textStatus.ReadyModes, "text") || readyModesContain(textStatus.ReadyModes, "image") || readyModesContain(textStatus.ReadyModes, "ocr") {
		t.Fatalf("PDF with extracted text should report text readiness without raster image readiness: %+v", textStatus)
	}
	if !failedStagesContain(textStatus.FailedStages, "raster_preview") {
		t.Fatalf("PDF with extracted text should report raster preview issue, got %+v", textStatus.FailedStages)
	}

	pdfWithOcr, err := service.CreateOrReuse(CreateResourceInput{
		Type: "file",
		Data: base64.StdEncoding.EncodeToString([]byte("%PDF-1.4\n% ocr-only\n")),
		Metadata: map[string]any{
			"fileRecord":          true,
			"name":                "ocr-only.pdf",
			"mimeType":            "application/pdf",
			"status":              "active",
			"ocrPagesJson":        `[{"pageIndex":0,"text":"real OCR only","charCount":13,"isFailed":false}]`,
			"ocrPagesSource":      "real_ocr",
			"previewJson":         `{"pages":[{"pageIndex":0,"path":"page.svg","mimeType":"image/svg+xml"}]}`,
			"previewSource":       pdfTextPreviewSource,
			"previewMimeType":     pdfTextPreviewMimeType,
			"rasterPreviewStatus": "unavailable",
		},
	})
	if err != nil {
		t.Fatalf("CreateOrReuse(pdf with OCR) error = %v", err)
	}
	ocrStatus, err := service.GetPdfProcessingStatus(pdfWithOcr.ResourceID)
	if err != nil {
		t.Fatalf("GetPdfProcessingStatus(pdf with OCR) error = %v", err)
	}
	if !readyModesContain(ocrStatus.ReadyModes, "ocr") || readyModesContain(ocrStatus.ReadyModes, "text") || readyModesContain(ocrStatus.ReadyModes, "image") {
		t.Fatalf("PDF with real OCR should report OCR without native text/image readiness: %+v", ocrStatus)
	}

	image, err := service.UploadAttachment(UploadAttachmentInput{
		Name:          "diagram.png",
		MimeType:      "image/png",
		Base64Content: base64.StdEncoding.EncodeToString([]byte("png bytes")),
	})
	if err != nil {
		t.Fatalf("UploadAttachment(image) error = %v", err)
	}
	imageStatus, err := service.GetPdfProcessingStatus(image.SourceID)
	if err != nil {
		t.Fatalf("GetPdfProcessingStatus(image) error = %v", err)
	}
	if imageStatus.Stage != "completed" || imageStatus.MediaType != "image" || imageStatus.Percent != 100 || len(imageStatus.ReadyModes) != 1 || imageStatus.ReadyModes[0] != "image" {
		t.Fatalf("image should be immediately ready for image mode: %+v", imageStatus)
	}
}

func TestPdfProcessingBatchAndControlCommands(t *testing.T) {
	service := newTestService(t)
	uploaded, err := service.UploadFile(UploadFileInput{
		Name:          "batch.pdf",
		MimeType:      "application/pdf",
		Base64Content: base64.StdEncoding.EncodeToString([]byte("%PDF-1.4\n")),
	})
	if err != nil {
		t.Fatalf("UploadFile() error = %v", err)
	}

	statuses, err := service.GetBatchPdfProcessingStatus([]string{uploaded.SourceID, "missing_file", ""})
	if err != nil {
		t.Fatalf("GetBatchPdfProcessingStatus() error = %v", err)
	}
	if len(statuses) != 2 {
		t.Fatalf("expected existing and missing statuses, got %+v", statuses)
	}
	if statuses[uploaded.SourceID].Stage != "completed_with_issues" {
		t.Fatalf("unexpected existing batch status: %+v", statuses[uploaded.SourceID])
	}
	if statuses["missing_file"].Stage != "pending" || statuses["missing_file"].Percent != 0 || len(statuses["missing_file"].ReadyModes) != 0 {
		t.Fatalf("missing file should return pending status: %+v", statuses["missing_file"])
	}

	cancelled, err := service.CancelPdfProcessing(uploaded.SourceID)
	if err != nil || !cancelled {
		t.Fatalf("CancelPdfProcessing(existing) = %v, %v", cancelled, err)
	}
	cancelled, err = service.CancelPdfProcessing("missing_file")
	if err != nil || cancelled {
		t.Fatalf("CancelPdfProcessing(missing) = %v, %v", cancelled, err)
	}

	stage := "ocr_processing"
	if err := service.StartPdfProcessing(uploaded.SourceID, &stage); err != nil {
		t.Fatalf("StartPdfProcessing() error = %v", err)
	}
	resource, err := service.GetResource(uploaded.SourceID)
	if err != nil {
		t.Fatalf("GetResource(after start) error = %v", err)
	}
	if resource == nil || metadataString(resource.Metadata, "processingRequestedAt", "") == "" || metadataString(resource.Metadata, "processingStartStage", "") != "ocr_processing" {
		t.Fatalf("start should record lightweight processing metadata: %+v", resource)
	}

	if err := service.RetryPdfProcessing(uploaded.SourceID); err != nil {
		t.Fatalf("RetryPdfProcessing() error = %v", err)
	}
	if err := service.StartPdfProcessing("missing_file", &stage); err != nil {
		t.Fatalf("StartPdfProcessing(missing) should be a no-op, got %v", err)
	}
}

func TestPdfProcessingControlCommandsEmitProgressEvents(t *testing.T) {
	service := newTestService(t)
	events := []struct {
		name    string
		payload any
	}{}
	service.SetEventEmitter(func(name string, payload any) {
		events = append(events, struct {
			name    string
			payload any
		}{name: name, payload: payload})
	})

	uploaded, err := service.UploadFile(UploadFileInput{
		Name:          "events.pdf",
		MimeType:      "application/pdf",
		Base64Content: base64.StdEncoding.EncodeToString([]byte("%PDF-1.4\n")),
	})
	if err != nil {
		t.Fatalf("UploadFile() error = %v", err)
	}
	if err := service.StartPdfProcessing(uploaded.SourceID, nil); err != nil {
		t.Fatalf("StartPdfProcessing() error = %v", err)
	}

	sawStart := false
	sawUnifiedCompleted := false
	sawLegacyCompleted := false
	for _, event := range events {
		switch event.name {
		case "media-processing-progress":
			payload, ok := event.payload.(mediaProcessingProgressPayload)
			if ok && payload.FileID == uploaded.SourceID && payload.MediaType == "pdf" && payload.Status.Stage == "text_extraction" {
				sawStart = true
			}
		case "media-processing-completed":
			payload, ok := event.payload.(mediaProcessingCompletedPayload)
			if ok && payload.FileID == uploaded.SourceID && payload.MediaType == "pdf" && payload.Stage == "completed_with_issues" {
				sawUnifiedCompleted = true
			}
		case "pdf-processing-completed":
			payload, ok := event.payload.(mediaProcessingCompletedPayload)
			if ok && payload.FileID == uploaded.SourceID && payload.MediaType == "pdf" {
				sawLegacyCompleted = true
			}
		}
	}
	if !sawStart || !sawUnifiedCompleted || !sawLegacyCompleted {
		t.Fatalf("expected PDF processing progress/completion events, got %+v", events)
	}

	eventCountBeforeRetry := len(events)
	if err := service.RetryPdfProcessing(uploaded.SourceID); err != nil {
		t.Fatalf("RetryPdfProcessing() error = %v", err)
	}
	sawRetryProgress := false
	for _, event := range events[eventCountBeforeRetry:] {
		if event.name != "media-processing-progress" {
			continue
		}
		payload, ok := event.payload.(mediaProcessingProgressPayload)
		if ok && payload.FileID == uploaded.SourceID && payload.MediaType == "pdf" && payload.Status.Stage == "text_extraction" {
			sawRetryProgress = true
		}
	}
	if !sawRetryProgress {
		t.Fatalf("expected retry to emit media-processing-progress, got %+v", events[eventCountBeforeRetry:])
	}

	eventCountBeforeCancel := len(events)
	cancelled, err := service.CancelPdfProcessing(uploaded.SourceID)
	if err != nil || !cancelled {
		t.Fatalf("CancelPdfProcessing() = %v, %v", cancelled, err)
	}
	sawCancelError := false
	sawLegacyCancelError := false
	for _, event := range events[eventCountBeforeCancel:] {
		switch event.name {
		case "media-processing-error":
			payload, ok := event.payload.(mediaProcessingErrorPayload)
			if ok && payload.FileID == uploaded.SourceID && payload.MediaType == "pdf" && payload.Error == "processing cancelled" {
				sawCancelError = true
			}
		case "pdf-processing-error":
			payload, ok := event.payload.(mediaProcessingErrorPayload)
			if ok && payload.FileID == uploaded.SourceID && payload.MediaType == "pdf" && payload.Error == "processing cancelled" {
				sawLegacyCancelError = true
			}
		}
	}
	if !sawCancelError || !sawLegacyCancelError {
		t.Fatalf("expected cancel to emit unified and legacy processing errors, got %+v", events[eventCountBeforeCancel:])
	}
}

func TestUploadFileStoresRasterPreviewWhenRendererSucceeds(t *testing.T) {
	service := newTestService(t)
	pngBytes := []byte{0x89, 'P', 'N', 'G', '\r', '\n', 0x1a, '\n', 'r', 'a', 's', 't', 'e', 'r'}
	withPdfRasterPreviewRenderer(t, fakePdfRasterPreviewRenderer(t, "preview-raster/upload/page-0001.png", pngBytes))

	pdfBytes := []byte("%PDF-1.4\n1 0 obj\n<< /Type /Catalog /Pages 2 0 R >>\nendobj\n2 0 obj\n<< /Type /Pages /Count 1 /Kids [3 0 R] >>\nendobj\n3 0 obj\n<< /Type /Page /Parent 2 0 R >>\nendobj\n4 0 obj\n<< /Length 54 >>\nstream\nBT\n(Raster preview keeps searchable text) Tj\nET\nendstream\nendobj\n%%EOF")
	uploaded, err := service.UploadFile(UploadFileInput{
		Name:          "raster-upload.pdf",
		MimeType:      "application/pdf",
		Base64Content: base64.StdEncoding.EncodeToString(pdfBytes),
	})
	if err != nil {
		t.Fatalf("UploadFile(raster pdf) error = %v", err)
	}
	resource, err := service.GetResource(uploaded.SourceID)
	if err != nil {
		t.Fatalf("GetResource(raster upload) error = %v", err)
	}
	if resource == nil || metadataString(resource.Metadata, "previewSource", "") != pdfRasterPreviewSource || metadataString(resource.Metadata, "previewMimeType", "") != pdfRasterPreviewMimeType {
		t.Fatalf("expected raster preview metadata, got %+v", resource)
	}
	if metadataString(resource.Metadata, "rasterPreviewStatus", "") != "completed" || metadataString(resource.Metadata, "pageRenderingStatus", "") != "completed" {
		t.Fatalf("expected completed raster/page rendering metadata, got %+v", resource.Metadata)
	}
	assertRasterPreviewJSON(t, *resource)
	status, err := service.GetPdfProcessingStatus(uploaded.SourceID)
	if err != nil {
		t.Fatalf("GetPdfProcessingStatus(raster upload) error = %v", err)
	}
	if status.Stage != "completed" || !readyModesContain(status.ReadyModes, "text") || !readyModesContain(status.ReadyModes, "image") {
		t.Fatalf("expected text and image readiness from raster upload, got %+v", status)
	}
	image, err := service.GetPdfPageImage(uploaded.SourceID, 0)
	if err != nil {
		t.Fatalf("GetPdfPageImage(raster upload) error = %v", err)
	}
	if image.MimeType != pdfRasterPreviewMimeType || image.Size != int64(len(pngBytes)) {
		t.Fatalf("expected PNG raster preview image, got %+v", image)
	}
	decoded, err := base64.StdEncoding.DecodeString(image.Base64)
	if err != nil {
		t.Fatalf("raster preview base64 decode error: %v", err)
	}
	if string(decoded) != string(pngBytes) {
		t.Fatalf("expected raster preview bytes %q, got %q", pngBytes, decoded)
	}
}

func TestStartPdfProcessingStoresRasterPreviewWhenRendererSucceeds(t *testing.T) {
	service := newTestService(t)
	pngBytes := []byte{0x89, 'P', 'N', 'G', '\r', '\n', 0x1a, '\n', 's', 't', 'a', 'r', 't'}
	called := false
	withPdfRasterPreviewRenderer(t, func(service *Service, resource Resource, effectiveMetadata map[string]any, now int64) (map[string]any, bool) {
		called = true
		return fakePdfRasterPreviewRenderer(t, "preview-raster/start/page-0001.png", pngBytes)(service, resource, effectiveMetadata, now)
	})

	sourceID := "file_raster_processing"
	pdfBytes := []byte("%PDF-1.4\n1 0 obj\n<< /Type /Catalog /Pages 2 0 R >>\nendobj\n2 0 obj\n<< /Type /Pages /Count 1 /Kids [3 0 R] >>\nendobj\n3 0 obj\n<< /Type /Page /Parent 2 0 R >>\nendobj\n4 0 obj\n<< /Length 48 >>\nstream\nBT\n(Start raster processing text) Tj\nET\nendstream\nendobj\n%%EOF")
	_, err := service.CreateOrReuse(CreateResourceInput{
		Type:     "file",
		Data:     base64.StdEncoding.EncodeToString(pdfBytes),
		SourceID: &sourceID,
		Metadata: map[string]any{
			"fileRecord": true,
			"name":       "start-raster.pdf",
			"mimeType":   "application/pdf",
			"status":     "active",
		},
	})
	if err != nil {
		t.Fatalf("CreateOrReuse(raster processing pdf) error = %v", err)
	}
	if err := service.StartPdfProcessing(sourceID, nil); err != nil {
		t.Fatalf("StartPdfProcessing(raster) error = %v", err)
	}
	if !called {
		t.Fatal("expected raster preview renderer to be called")
	}
	resource, err := service.GetResource(sourceID)
	if err != nil {
		t.Fatalf("GetResource(raster processing) error = %v", err)
	}
	if resource == nil || metadataString(resource.Metadata, "previewSource", "") != pdfRasterPreviewSource || metadataString(resource.Metadata, "rasterPreviewStatus", "") != "completed" {
		t.Fatalf("expected raster processing metadata, got %+v", resource)
	}
	assertRasterPreviewJSON(t, *resource)
	image, err := service.GetPdfPageImage(sourceID, 0)
	if err != nil {
		t.Fatalf("GetPdfPageImage(raster processing) error = %v", err)
	}
	if image.MimeType != pdfRasterPreviewMimeType || image.Size != int64(len(pngBytes)) {
		t.Fatalf("expected PNG raster processing preview, got %+v", image)
	}
}

func TestStartPdfProcessingDoesNotHoldVfsLockWhileRenderingRasterPreview(t *testing.T) {
	service := newTestService(t)
	renderStarted := make(chan struct{})
	releaseRenderer := make(chan struct{})
	var once sync.Once
	withPdfRasterPreviewRenderer(t, func(_ *Service, _ Resource, _ map[string]any, _ int64) (map[string]any, bool) {
		once.Do(func() { close(renderStarted) })
		<-releaseRenderer
		return pdfRasterPreviewFailedMetadata("unavailable", nil), false
	})

	sourceID := "file_nonblocking_raster"
	pdfBytes := []byte("%PDF-1.4\n1 0 obj\n<< /Type /Catalog /Pages 2 0 R >>\nendobj\n2 0 obj\n<< /Type /Pages /Count 1 /Kids [3 0 R] >>\nendobj\n3 0 obj\n<< /Type /Page /Parent 2 0 R >>\nendobj\n4 0 obj\n<< /Length 45 >>\nstream\nBT\n(Nonblocking raster text) Tj\nET\nendstream\nendobj\n%%EOF")
	_, err := service.CreateOrReuse(CreateResourceInput{
		Type:     "file",
		Data:     base64.StdEncoding.EncodeToString(pdfBytes),
		SourceID: &sourceID,
		Metadata: map[string]any{
			"fileRecord": true,
			"name":       "nonblocking.pdf",
			"mimeType":   "application/pdf",
			"status":     "active",
		},
	})
	if err != nil {
		t.Fatalf("CreateOrReuse(nonblocking raster pdf) error = %v", err)
	}

	done := make(chan error, 1)
	go func() {
		done <- service.StartPdfProcessing(sourceID, nil)
	}()

	select {
	case <-renderStarted:
	case err := <-done:
		t.Fatalf("StartPdfProcessing finished before renderer blocked: %v", err)
	case <-time.After(2 * time.Second):
		t.Fatal("renderer did not start")
	}

	statusDone := make(chan error, 1)
	go func() {
		_, err := service.GetPdfProcessingStatus(sourceID)
		statusDone <- err
	}()
	select {
	case err := <-statusDone:
		if err != nil {
			t.Fatalf("GetPdfProcessingStatus while rendering error = %v", err)
		}
	case <-time.After(300 * time.Millisecond):
		t.Fatal("GetPdfProcessingStatus blocked while raster renderer was running")
	}

	close(releaseRenderer)
	if err := <-done; err != nil {
		t.Fatalf("StartPdfProcessing(nonblocking raster) error = %v", err)
	}
}

func TestUploadFileFallsBackToTextPreviewWhenRasterRendererFails(t *testing.T) {
	service := newTestService(t)
	withPdfRasterPreviewRenderer(t, func(_ *Service, _ Resource, _ map[string]any, _ int64) (map[string]any, bool) {
		return pdfRasterPreviewFailedMetadata("unavailable", nil), false
	})

	pdfBytes := []byte("%PDF-1.4\n1 0 obj\n<< /Type /Catalog /Pages 2 0 R >>\nendobj\n2 0 obj\n<< /Type /Pages /Count 1 /Kids [3 0 R] >>\nendobj\n3 0 obj\n<< /Type /Page /Parent 2 0 R >>\nendobj\n4 0 obj\n<< /Length 47 >>\nstream\nBT\n(Upload raster fallback text) Tj\nET\nendstream\nendobj\n%%EOF")
	uploaded, err := service.UploadFile(UploadFileInput{
		Name:          "upload-fallback.pdf",
		MimeType:      "application/pdf",
		Base64Content: base64.StdEncoding.EncodeToString(pdfBytes),
	})
	if err != nil {
		t.Fatalf("UploadFile(fallback) error = %v", err)
	}
	resource, err := service.GetResource(uploaded.SourceID)
	if err != nil {
		t.Fatalf("GetResource(upload fallback) error = %v", err)
	}
	if resource == nil || metadataString(resource.Metadata, "rasterPreviewStatus", "") != "unavailable" || metadataString(resource.Metadata, "previewSource", "") != pdfTextPreviewSource {
		t.Fatalf("expected upload SVG fallback metadata, got %+v", resource)
	}
	if metadataString(resource.Metadata, "previewMimeType", "") != pdfTextPreviewMimeType || metadataString(resource.Metadata, "pageRenderingStatus", "") != "completed" {
		t.Fatalf("expected completed upload SVG fallback metadata, got %+v", resource.Metadata)
	}
	status, err := service.GetPdfProcessingStatus(uploaded.SourceID)
	if err != nil {
		t.Fatalf("GetPdfProcessingStatus(upload fallback) error = %v", err)
	}
	if status.Stage != "completed_with_issues" || !readyModesContain(status.ReadyModes, "text") || readyModesContain(status.ReadyModes, "image") || !failedStagesContain(status.FailedStages, "raster_preview") {
		t.Fatalf("expected text readiness and raster issue from upload SVG fallback, got %+v", status)
	}
	image, err := service.GetPdfPageImage(uploaded.SourceID, 0)
	if err != nil {
		t.Fatalf("GetPdfPageImage(upload fallback) error = %v", err)
	}
	if image.MimeType != pdfTextPreviewMimeType {
		t.Fatalf("expected SVG upload fallback image, got %+v", image)
	}
	decoded, err := base64.StdEncoding.DecodeString(image.Base64)
	if err != nil {
		t.Fatalf("upload fallback preview base64 decode error: %v", err)
	}
	if !strings.Contains(string(decoded), "Upload raster fallback text") || !strings.Contains(string(decoded), pdfTextPreviewSource) {
		t.Fatalf("expected upload SVG fallback text and source marker, got %q", decoded)
	}
}

func TestStartPdfProcessingFallsBackToTextPreviewWhenRasterRendererFails(t *testing.T) {
	service := newTestService(t)
	withPdfRasterPreviewRenderer(t, func(_ *Service, _ Resource, _ map[string]any, _ int64) (map[string]any, bool) {
		return pdfRasterPreviewFailedMetadata("unavailable", nil), false
	})

	sourceID := "file_raster_fallback"
	pdfBytes := []byte("%PDF-1.4\n1 0 obj\n<< /Type /Catalog /Pages 2 0 R >>\nendobj\n2 0 obj\n<< /Type /Pages /Count 1 /Kids [3 0 R] >>\nendobj\n3 0 obj\n<< /Type /Page /Parent 2 0 R >>\nendobj\n4 0 obj\n<< /Length 50 >>\nstream\nBT\n(Raster fallback text layer) Tj\nET\nendstream\nendobj\n%%EOF")
	_, err := service.CreateOrReuse(CreateResourceInput{
		Type:     "file",
		Data:     base64.StdEncoding.EncodeToString(pdfBytes),
		SourceID: &sourceID,
		Metadata: map[string]any{
			"fileRecord": true,
			"name":       "fallback.pdf",
			"mimeType":   "application/pdf",
			"status":     "active",
		},
	})
	if err != nil {
		t.Fatalf("CreateOrReuse(fallback pdf) error = %v", err)
	}
	if err := service.StartPdfProcessing(sourceID, nil); err != nil {
		t.Fatalf("StartPdfProcessing(fallback) error = %v", err)
	}
	resource, err := service.GetResource(sourceID)
	if err != nil {
		t.Fatalf("GetResource(fallback) error = %v", err)
	}
	if resource == nil || metadataString(resource.Metadata, "rasterPreviewStatus", "") != "unavailable" || metadataString(resource.Metadata, "previewSource", "") != pdfTextPreviewSource {
		t.Fatalf("expected raster failure metadata with text preview fallback, got %+v", resource)
	}
	if metadataString(resource.Metadata, "previewMimeType", "") != pdfTextPreviewMimeType || metadataString(resource.Metadata, "pageRenderingStatus", "") != "completed" {
		t.Fatalf("expected completed SVG fallback metadata, got %+v", resource.Metadata)
	}
	status, err := service.GetPdfProcessingStatus(sourceID)
	if err != nil {
		t.Fatalf("GetPdfProcessingStatus(fallback) error = %v", err)
	}
	if status.Stage != "completed_with_issues" || !readyModesContain(status.ReadyModes, "text") || readyModesContain(status.ReadyModes, "image") || !failedStagesContain(status.FailedStages, "raster_preview") {
		t.Fatalf("expected text readiness and raster issue from SVG fallback, got %+v", status)
	}
	image, err := service.GetPdfPageImage(sourceID, 0)
	if err != nil {
		t.Fatalf("GetPdfPageImage(fallback) error = %v", err)
	}
	if image.MimeType != pdfTextPreviewMimeType {
		t.Fatalf("expected SVG fallback image, got %+v", image)
	}
	decoded, err := base64.StdEncoding.DecodeString(image.Base64)
	if err != nil {
		t.Fatalf("fallback preview base64 decode error: %v", err)
	}
	if !strings.Contains(string(decoded), "Raster fallback text layer") || !strings.Contains(string(decoded), pdfTextPreviewSource) {
		t.Fatalf("expected SVG fallback text and source marker, got %q", decoded)
	}
}

func TestUploadFilePreservesLegacyPreviewJsonWhenRasterRendererExists(t *testing.T) {
	service := newTestService(t)
	called := false
	withPdfRasterPreviewRenderer(t, func(_ *Service, _ Resource, _ map[string]any, _ int64) (map[string]any, bool) {
		called = true
		return map[string]any{"previewSource": pdfRasterPreviewSource}, true
	})

	legacyBytes := []byte{0x89, 'P', 'N', 'G', '\r', '\n', 0x1a, '\n', 'l', 'e', 'g', 'a', 'c', 'y'}
	previewJSON := `{"pages":[{"pageIndex":0,"dataUrl":"data:image/png;base64,` + base64.StdEncoding.EncodeToString(legacyBytes) + `"}]}`
	uploaded, err := service.UploadFile(UploadFileInput{
		Name:          "legacy-preview.pdf",
		MimeType:      "application/pdf",
		Base64Content: base64.StdEncoding.EncodeToString([]byte("%PDF-1.4\n")),
		Metadata:      map[string]any{"previewJson": previewJSON},
	})
	if err != nil {
		t.Fatalf("UploadFile(legacy preview) error = %v", err)
	}
	if called {
		t.Fatal("raster renderer should not run when legacy previewJson is already present")
	}
	resource, err := service.GetResource(uploaded.SourceID)
	if err != nil {
		t.Fatalf("GetResource(legacy preview) error = %v", err)
	}
	if resource == nil || metadataString(resource.Metadata, "previewJson", "") != previewJSON || metadataString(resource.Metadata, "previewSource", "") != "" {
		t.Fatalf("expected legacy previewJson to remain untouched, got %+v", resource)
	}
	image, err := service.GetPdfPageImage(uploaded.SourceID, 0)
	if err != nil {
		t.Fatalf("GetPdfPageImage(legacy preview) error = %v", err)
	}
	if image.MimeType != pdfRasterPreviewMimeType || image.Size != int64(len(legacyBytes)) {
		t.Fatalf("expected original legacy PNG preview, got %+v", image)
	}
}

func TestUploadFileDedupePreservesUserMetadata(t *testing.T) {
	service := newTestService(t)
	pdfBytes := []byte("%PDF-1.4\n1 0 obj\n<< /Length 44 >>\nstream\nBT\n(User metadata preserved) Tj\nET\nendstream\nendobj\n%%EOF")
	first, err := service.UploadFile(UploadFileInput{
		Name:          "preserve-metadata.pdf",
		MimeType:      "application/pdf",
		Base64Content: base64.StdEncoding.EncodeToString(pdfBytes),
	})
	if err != nil {
		t.Fatalf("UploadFile(first preserve metadata) error = %v", err)
	}
	bookmarks := []any{map[string]any{"id": "bm_keep", "page": float64(3), "title": "Keep"}}
	ok, err := service.UpdateBookmarks(first.SourceID, bookmarks)
	if !ok || err != nil {
		t.Fatalf("UpdateBookmarks(preserve metadata) = %v, %v", ok, err)
	}

	second, err := service.UploadFile(UploadFileInput{
		Name:          "preserve-metadata.pdf",
		MimeType:      "application/pdf",
		Base64Content: base64.StdEncoding.EncodeToString(pdfBytes),
	})
	if err != nil {
		t.Fatalf("UploadFile(second preserve metadata) error = %v", err)
	}
	if second.IsNew {
		t.Fatal("second upload should reuse existing PDF")
	}
	resource, err := service.GetResource(second.SourceID)
	if err != nil {
		t.Fatalf("GetResource(preserve metadata) error = %v", err)
	}
	if resource == nil {
		t.Fatal("expected reused resource")
	}
	kept := metadataAnySlice(resource.Metadata, "bookmarks")
	if len(kept) != 1 {
		t.Fatalf("expected dedupe upload to preserve bookmarks, got %+v", resource.Metadata)
	}
}

func TestUploadFilePreservesProvidedTextSourcePreviewJson(t *testing.T) {
	service := newTestService(t)
	called := false
	withPdfRasterPreviewRenderer(t, func(_ *Service, _ Resource, _ map[string]any, _ int64) (map[string]any, bool) {
		called = true
		return map[string]any{"previewSource": pdfRasterPreviewSource}, true
	})

	legacyBytes := []byte{0x89, 'P', 'N', 'G', '\r', '\n', 0x1a, '\n', 't', 'e', 'x', 't', 's', 'r', 'c'}
	previewJSON := `{"pages":[{"pageIndex":0,"dataUrl":"data:image/png;base64,` + base64.StdEncoding.EncodeToString(legacyBytes) + `"}]}`
	uploaded, err := service.UploadFile(UploadFileInput{
		Name:          "provided-text-preview.pdf",
		MimeType:      "application/pdf",
		Base64Content: base64.StdEncoding.EncodeToString([]byte("%PDF-1.4\n")),
		Metadata: map[string]any{
			"previewJson":   previewJSON,
			"previewSource": pdfTextPreviewSource,
		},
	})
	if err != nil {
		t.Fatalf("UploadFile(provided text preview) error = %v", err)
	}
	if called {
		t.Fatal("raster renderer should not run for user-provided text-source previewJson")
	}
	resource, err := service.GetResource(uploaded.SourceID)
	if err != nil {
		t.Fatalf("GetResource(provided text preview) error = %v", err)
	}
	if resource == nil || metadataString(resource.Metadata, "previewJson", "") != previewJSON || metadataString(resource.Metadata, "previewSource", "") != pdfTextPreviewSource {
		t.Fatalf("expected provided text-source preview metadata to remain untouched, got %+v", resource)
	}
}

func TestStartPdfProcessingPreservesLegacyPreviewJson(t *testing.T) {
	service := newTestService(t)
	called := false
	withPdfRasterPreviewRenderer(t, func(_ *Service, _ Resource, _ map[string]any, _ int64) (map[string]any, bool) {
		called = true
		return map[string]any{"previewSource": pdfRasterPreviewSource}, true
	})

	legacyBytes := []byte{0x89, 'P', 'N', 'G', '\r', '\n', 0x1a, '\n', 's', 't', 'a', 'r', 't', 'l', 'e', 'g'}
	previewJSON := `{"pages":[{"pageIndex":0,"dataUrl":"data:image/png;base64,` + base64.StdEncoding.EncodeToString(legacyBytes) + `"}]}`
	sourceID := "file_legacy_preview_processing"
	pdfBytes := []byte("%PDF-1.4\n1 0 obj\n<< /Type /Catalog /Pages 2 0 R >>\nendobj\n2 0 obj\n<< /Type /Pages /Count 1 /Kids [3 0 R] >>\nendobj\n3 0 obj\n<< /Type /Page /Parent 2 0 R >>\nendobj\n4 0 obj\n<< /Length 47 >>\nstream\nBT\n(Legacy preview processing) Tj\nET\nendstream\nendobj\n%%EOF")
	_, err := service.CreateOrReuse(CreateResourceInput{
		Type:     "file",
		Data:     base64.StdEncoding.EncodeToString(pdfBytes),
		SourceID: &sourceID,
		Metadata: map[string]any{
			"fileRecord":  true,
			"name":        "legacy-processing.pdf",
			"mimeType":    "application/pdf",
			"status":      "active",
			"previewJson": previewJSON,
		},
	})
	if err != nil {
		t.Fatalf("CreateOrReuse(legacy preview processing pdf) error = %v", err)
	}
	if err := service.StartPdfProcessing(sourceID, nil); err != nil {
		t.Fatalf("StartPdfProcessing(legacy preview) error = %v", err)
	}
	if called {
		t.Fatal("raster renderer should not run when processing resource with legacy previewJson")
	}
	resource, err := service.GetResource(sourceID)
	if err != nil {
		t.Fatalf("GetResource(legacy preview processing) error = %v", err)
	}
	if resource == nil || metadataString(resource.Metadata, "previewJson", "") != previewJSON || metadataString(resource.Metadata, "previewSource", "") != "" {
		t.Fatalf("expected processing to preserve legacy previewJson, got %+v", resource)
	}
	if metadataString(resource.Metadata, "rasterPreviewStatus", "") != "" || metadataString(resource.Metadata, "pageRenderingSource", "") != "" {
		t.Fatalf("processing should not add generated preview sidecar metadata, got %+v", resource.Metadata)
	}
	image, err := service.GetPdfPageImage(sourceID, 0)
	if err != nil {
		t.Fatalf("GetPdfPageImage(legacy preview processing) error = %v", err)
	}
	if image.MimeType != pdfRasterPreviewMimeType || image.Size != int64(len(legacyBytes)) {
		t.Fatalf("expected original processing legacy PNG preview, got %+v", image)
	}
}

func TestUploadFileReplacesGeneratedPreviewMetadataOnDedupePreviewJson(t *testing.T) {
	service := newTestService(t)
	renderCalls := 0
	generatedBytes := []byte{0x89, 'P', 'N', 'G', '\r', '\n', 0x1a, '\n', 'g', 'e', 'n'}
	withPdfRasterPreviewRenderer(t, func(service *Service, resource Resource, effectiveMetadata map[string]any, now int64) (map[string]any, bool) {
		renderCalls++
		if renderCalls > 1 {
			t.Fatal("raster renderer should not run for dedupe upload with incoming previewJson")
		}
		return fakePdfRasterPreviewRenderer(t, "preview-raster/dedupe/page-0001.png", generatedBytes)(service, resource, effectiveMetadata, now)
	})

	pdfBytes := []byte("%PDF-1.4\n1 0 obj\n<< /Type /Catalog /Pages 2 0 R >>\nendobj\n2 0 obj\n<< /Type /Pages /Count 1 /Kids [3 0 R] >>\nendobj\n3 0 obj\n<< /Type /Page /Parent 2 0 R >>\nendobj\n4 0 obj\n<< /Length 42 >>\nstream\nBT\n(Dedupe preview text) Tj\nET\nendstream\nendobj\n%%EOF")
	first, err := service.UploadFile(UploadFileInput{
		Name:          "dedupe-preview.pdf",
		MimeType:      "application/pdf",
		Base64Content: base64.StdEncoding.EncodeToString(pdfBytes),
	})
	if err != nil {
		t.Fatalf("UploadFile(first dedupe) error = %v", err)
	}
	firstResource, err := service.GetResource(first.SourceID)
	if err != nil {
		t.Fatalf("GetResource(first dedupe) error = %v", err)
	}
	if firstResource == nil || metadataString(firstResource.Metadata, "previewSource", "") != pdfRasterPreviewSource {
		t.Fatalf("expected first upload to generate raster preview, got %+v", firstResource)
	}

	legacyBytes := []byte{0x89, 'P', 'N', 'G', '\r', '\n', 0x1a, '\n', 'n', 'e', 'w'}
	legacyPreviewJSON := `{"pages":[{"pageIndex":0,"dataUrl":"data:image/png;base64,` + base64.StdEncoding.EncodeToString(legacyBytes) + `"}]}`
	second, err := service.UploadFile(UploadFileInput{
		Name:          "dedupe-preview.pdf",
		MimeType:      "application/pdf",
		Base64Content: base64.StdEncoding.EncodeToString(pdfBytes),
		Metadata:      map[string]any{"previewJson": legacyPreviewJSON},
	})
	if err != nil {
		t.Fatalf("UploadFile(second dedupe) error = %v", err)
	}
	if second.IsNew {
		t.Fatal("second upload with same content should reuse existing resource")
	}
	resource, err := service.GetResource(second.SourceID)
	if err != nil {
		t.Fatalf("GetResource(second dedupe) error = %v", err)
	}
	if resource == nil || metadataString(resource.Metadata, "previewJson", "") != legacyPreviewJSON {
		t.Fatalf("expected dedupe upload to use incoming previewJson, got %+v", resource)
	}
	for _, key := range []string{"previewSource", "previewGeneratedAt", "previewPageCount", "previewMimeType", "pageRenderingStatus", "pageRenderingSource", "pageRenderingError", "rasterPreviewStatus", "rasterPreviewSource", "rasterPreviewError"} {
		if value := metadataString(resource.Metadata, key, ""); value != "" {
			t.Fatalf("expected generated preview sidecar key %s to be cleared, got %q in %+v", key, value, resource.Metadata)
		}
	}
	if len(metadataAnySlice(resource.Metadata, "bookmarks")) != 0 {
		t.Fatalf("expected no synthetic bookmarks side effect, got %+v", resource.Metadata)
	}
	image, err := service.GetPdfPageImage(second.SourceID, 0)
	if err != nil {
		t.Fatalf("GetPdfPageImage(second dedupe) error = %v", err)
	}
	if image.MimeType != pdfRasterPreviewMimeType || image.Size != int64(len(legacyBytes)) {
		t.Fatalf("expected dedupe upload to resolve incoming legacy PNG preview, got %+v", image)
	}
}

func TestStartPdfProcessingDoesNotEmitCompletedWhenFlushFails(t *testing.T) {
	service := newTestService(t)

	uploaded, err := service.UploadFile(UploadFileInput{
		Name:          "flush-failure.pdf",
		MimeType:      "application/pdf",
		Base64Content: base64.StdEncoding.EncodeToString([]byte("%PDF-1.4\n")),
	})
	if err != nil {
		t.Fatalf("UploadFile() error = %v", err)
	}

	events := []struct {
		name    string
		payload any
	}{}
	service.SetEventEmitter(func(name string, payload any) {
		events = append(events, struct {
			name    string
			payload any
		}{name: name, payload: payload})
	})

	nonEmptyDir := filepath.Join(t.TempDir(), "blocked-index")
	if err := os.MkdirAll(nonEmptyDir, 0o700); err != nil {
		t.Fatalf("MkdirAll(blocked index) error = %v", err)
	}
	if err := os.WriteFile(filepath.Join(nonEmptyDir, "keep.txt"), []byte("block remove"), 0o600); err != nil {
		t.Fatalf("WriteFile(blocked child) error = %v", err)
	}
	service.indexPath = nonEmptyDir

	if err := service.StartPdfProcessing(uploaded.SourceID, nil); err == nil {
		t.Fatal("StartPdfProcessing() expected flush error")
	}

	sawError := false
	for _, event := range events {
		switch event.name {
		case "media-processing-completed", "pdf-processing-completed":
			t.Fatalf("flush failure should not emit completed event before error: %+v", events)
		case "media-processing-error":
			payload, ok := event.payload.(mediaProcessingErrorPayload)
			if ok && payload.FileID == uploaded.SourceID && payload.MediaType == "pdf" {
				sawError = true
			}
		}
	}
	if !sawError {
		t.Fatalf("expected flush failure to emit media-processing-error, got %+v", events)
	}
}

func TestIndexCommandsEmitVfsProgressEvents(t *testing.T) {
	service := newTestService(t)
	events := []struct {
		name    string
		payload any
	}{}
	service.SetEventEmitter(func(name string, payload any) {
		events = append(events, struct {
			name    string
			payload any
		}{name: name, payload: payload})
	})

	uploaded, err := service.UploadFile(UploadFileInput{
		Name:          "pending-index.pdf",
		MimeType:      "application/pdf",
		Base64Content: base64.StdEncoding.EncodeToString([]byte("%PDF-1.4\n")),
	})
	if err != nil {
		t.Fatalf("UploadFile() error = %v", err)
	}
	chunks, err := service.ReindexResource(uploaded.SourceID)
	if err != nil {
		t.Fatalf("ReindexResource() error = %v", err)
	}
	if chunks != 1 {
		t.Fatalf("expected one compact Go VFS unit, got %d", chunks)
	}

	result, err := service.BatchIndexPending(10)
	if err != nil {
		t.Fatalf("BatchIndexPending() error = %v", err)
	}
	if result.Total != 1 || result.SuccessCount != 1 || result.FailCount != 0 {
		t.Fatalf("unexpected batch result: %+v", result)
	}

	sawReindexStarted := false
	sawReindexCompleted := false
	sawBatchStarted := false
	sawBatchCompleted := false
	for _, event := range events {
		if event.name != "vfs-index-progress" {
			continue
		}
		payload, ok := event.payload.(map[string]any)
		if !ok {
			t.Fatalf("unexpected index event payload type: %T", event.payload)
		}
		switch payload["type"] {
		case "started":
			if payload["resourceId"] == uploaded.SourceID {
				sawReindexStarted = true
			}
		case "completed":
			if payload["resourceId"] == uploaded.SourceID {
				sawReindexCompleted = true
			}
		case "batch_started":
			sawBatchStarted = true
		case "batch_completed":
			sawBatchCompleted = true
		}
	}
	if !sawReindexStarted || !sawReindexCompleted || !sawBatchStarted || !sawBatchCompleted {
		t.Fatalf("expected reindex and batch index progress events, got %+v", events)
	}
}

func TestPdfProcessingStatusHidesSoftDeletedResources(t *testing.T) {
	service := newTestService(t)
	uploaded, err := service.UploadFile(UploadFileInput{
		Name:          "deleted.pdf",
		MimeType:      "application/pdf",
		Base64Content: base64.StdEncoding.EncodeToString([]byte("%PDF-1.4\n")),
	})
	if err != nil {
		t.Fatalf("UploadFile() error = %v", err)
	}
	if err := service.DeleteFile(uploaded.SourceID); err != nil {
		t.Fatalf("DeleteFile() error = %v", err)
	}
	status, err := service.GetPdfProcessingStatus(uploaded.SourceID)
	if err != nil {
		t.Fatalf("GetPdfProcessingStatus(deleted) error = %v", err)
	}
	if status.Stage != "error" || status.Error == nil || status.MediaType != "pdf" || len(status.ReadyModes) != 0 {
		t.Fatalf("deleted PDF should return an error status without ready modes: %+v", status)
	}
}

func TestGetPdfPageImageReadsInlinePreviewData(t *testing.T) {
	service := newTestService(t)
	pageBytes := []byte{0x89, 'P', 'N', 'G', '\r', '\n', 0x1a, '\n', 'p', 'a', 'g', 'e'}
	previewJSON := `{"pages":[{"pageIndex":0,"dataUrl":"data:image/png;base64,` + base64.StdEncoding.EncodeToString(pageBytes) + `"}]}`
	uploaded, err := service.UploadFile(UploadFileInput{
		Name:          "inline-preview.pdf",
		MimeType:      "application/pdf",
		Base64Content: base64.StdEncoding.EncodeToString([]byte("%PDF-1.4\n")),
		Metadata:      map[string]any{"previewJson": previewJSON},
	})
	if err != nil {
		t.Fatalf("UploadFile() error = %v", err)
	}

	image, err := service.GetPdfPageImage(uploaded.SourceID, 0)
	if err != nil {
		t.Fatalf("GetPdfPageImage(inline) error = %v", err)
	}
	if image.MimeType != "image/png" || image.Size != int64(len(pageBytes)) {
		t.Fatalf("unexpected inline page image metadata: %+v", image)
	}
	decoded, err := base64.StdEncoding.DecodeString(image.Base64)
	if err != nil {
		t.Fatalf("result base64 decode error: %v", err)
	}
	if string(decoded) != string(pageBytes) {
		t.Fatalf("unexpected inline page bytes: %q", decoded)
	}
}

func TestGetPdfPageImageResolvesRegisteredPreviewBlob(t *testing.T) {
	service := newTestService(t)
	pageBytes := []byte{0xff, 0xd8, 0xff, 0xdb, 'j', 'p', 'e', 'g'}
	pageImage, err := service.CreateOrReuse(CreateResourceInput{
		Type:     "image",
		Data:     base64.StdEncoding.EncodeToString(pageBytes),
		Metadata: map[string]any{"name": "page-1.jpg", "mimeType": "image/jpeg"},
	})
	if err != nil {
		t.Fatalf("CreateOrReuse(page image) error = %v", err)
	}
	previewJSON := `{"pages":[{"page_index":1,"blob_hash":"` + pageImage.Hash + `","mime_type":"image/jpeg"}]}`
	uploaded, err := service.UploadFile(UploadFileInput{
		Name:          "registered-preview.pdf",
		MimeType:      "application/pdf",
		Base64Content: base64.StdEncoding.EncodeToString([]byte("%PDF-1.4\n")),
		Metadata:      map[string]any{"previewJson": previewJSON},
	})
	if err != nil {
		t.Fatalf("UploadFile() error = %v", err)
	}

	image, err := service.GetPdfPageImage("vfs://"+uploaded.SourceID, 1)
	if err != nil {
		t.Fatalf("GetPdfPageImage(blob hash) error = %v", err)
	}
	if image.MimeType != "image/jpeg" || image.Size != int64(len(pageBytes)) {
		t.Fatalf("unexpected blob page image metadata: %+v", image)
	}
	decoded, err := base64.StdEncoding.DecodeString(image.Base64)
	if err != nil {
		t.Fatalf("result base64 decode error: %v", err)
	}
	if string(decoded) != string(pageBytes) {
		t.Fatalf("unexpected blob page bytes: %q", decoded)
	}
}

func TestGetPdfPageImageReadsLibraryPreviewPathAndErrorsClearly(t *testing.T) {
	service := newTestService(t)
	pageBytes := []byte("gif89 preview bytes")
	pageImage, err := service.CreateOrReuse(CreateResourceInput{
		Type:     "image",
		Data:     base64.StdEncoding.EncodeToString(pageBytes),
		Metadata: map[string]any{"name": "page.gif", "mimeType": "image/gif"},
	})
	if err != nil {
		t.Fatalf("CreateOrReuse(page image) error = %v", err)
	}
	pageResource, err := service.GetResource(pageImage.ResourceID)
	if err != nil {
		t.Fatalf("GetResource(page image) error = %v", err)
	}
	if pageResource == nil || pageResource.ExternalPath == nil {
		t.Fatalf("page image missing external path: %+v", pageResource)
	}

	previewJSON := `{"pages":[{"pageIndex":2,"path":"` + *pageResource.ExternalPath + `","mimeType":"image/gif"}]}`
	uploaded, err := service.UploadFile(UploadFileInput{
		Name:          "path-preview.pdf",
		MimeType:      "application/pdf",
		Base64Content: base64.StdEncoding.EncodeToString([]byte("%PDF-1.4\n")),
		Metadata:      map[string]any{"previewJson": previewJSON},
	})
	if err != nil {
		t.Fatalf("UploadFile() error = %v", err)
	}

	image, err := service.GetPdfPageImage(uploaded.SourceID, 2)
	if err != nil {
		t.Fatalf("GetPdfPageImage(path) error = %v", err)
	}
	if image.MimeType != "image/gif" || image.Size != int64(len(pageBytes)) {
		t.Fatalf("unexpected path page image metadata: %+v", image)
	}
	if _, err := service.GetPdfPageImage(uploaded.SourceID, 3); err == nil {
		t.Fatal("GetPdfPageImage(missing page) should return an error")
	}
	if err := service.DeleteFile(uploaded.SourceID); err != nil {
		t.Fatalf("DeleteFile() error = %v", err)
	}
	if _, err := service.GetPdfPageImage(uploaded.SourceID, 2); err == nil {
		t.Fatal("GetPdfPageImage(deleted resource) should return an error")
	}
}

func TestGetBlobBase64ReadsHybridVfsResourceByHashAndSourceID(t *testing.T) {
	service := newTestService(t)
	imageBytes := []byte{0x89, 'P', 'N', 'G', '\r', '\n', 0x1a, '\n', 'b', 'l', 'o', 'b'}
	uploaded, err := service.UploadAttachment(UploadAttachmentInput{
		Name:          "blob.png",
		MimeType:      "image/png",
		Base64Content: base64.StdEncoding.EncodeToString(imageBytes),
	})
	if err != nil {
		t.Fatalf("UploadAttachment() error = %v", err)
	}

	byHash, err := service.GetBlobBase64(uploaded.ResourceHash)
	if err != nil {
		t.Fatalf("GetBlobBase64(hash) error = %v", err)
	}
	if byHash.MimeType != "image/png" || byHash.Size != int64(len(imageBytes)) {
		t.Fatalf("unexpected blob metadata by hash: %+v", byHash)
	}
	decoded, err := base64.StdEncoding.DecodeString(byHash.Base64)
	if err != nil {
		t.Fatalf("result base64 decode error: %v", err)
	}
	if string(decoded) != string(imageBytes) {
		t.Fatalf("unexpected blob bytes by hash: %q", decoded)
	}

	bySourceID, err := service.GetBlobBase64(uploaded.SourceID)
	if err != nil {
		t.Fatalf("GetBlobBase64(source id) error = %v", err)
	}
	if bySourceID.Base64 != byHash.Base64 || bySourceID.MimeType != byHash.MimeType || bySourceID.Size != byHash.Size {
		t.Fatalf("source id alias should resolve same blob: hash=%+v source=%+v", byHash, bySourceID)
	}
}

func TestGetBlobBase64RejectsSoftDeletedFileResource(t *testing.T) {
	service := newTestService(t)
	uploaded, err := service.UploadFile(UploadFileInput{
		Name:          "deleted-image.webp",
		MimeType:      "image/webp",
		Base64Content: base64.StdEncoding.EncodeToString([]byte("RIFFxxxxWEBPdeleted")),
	})
	if err != nil {
		t.Fatalf("UploadFile() error = %v", err)
	}
	if err := service.DeleteFile(uploaded.SourceID); err != nil {
		t.Fatalf("DeleteFile() error = %v", err)
	}
	if _, err := service.GetBlobBase64(uploaded.ResourceHash); err == nil {
		t.Fatal("GetBlobBase64(deleted hash) should return an error")
	}
}

func TestSyncResourceUnitsRegistersCompactResource(t *testing.T) {
	service := newTestService(t)
	data := "synced resource content"
	pageCount := 3
	previewJSON := `{"pages":[{"pageIndex":0,"path":"page.png","mimeType":"image/png"}]}`
	ocrPagesJSON := `[{"pageIndex":0,"text":"OCR page","charCount":8,"isFailed":false}]`

	units, err := service.SyncResourceUnits(SyncResourceUnitsInput{
		ResourceID:   "sync_resource_1",
		ResourceType: "note",
		Data:         &data,
		PageCount:    &pageCount,
		PreviewJSON:  &previewJSON,
		OcrPagesJSON: &ocrPagesJSON,
	})
	if err != nil {
		t.Fatalf("SyncResourceUnits() error = %v", err)
	}
	if len(units) != 1 || units[0].TextState != "indexed" {
		t.Fatalf("unexpected synced units: %+v", units)
	}

	refData, err := service.GetResourceRefs(GetResourceRefsInput{SourceIDs: []string{"sync_resource_1"}})
	if err != nil {
		t.Fatalf("GetResourceRefs() error = %v", err)
	}
	if len(refData.Refs) != 1 || refData.Refs[0].Type != "note" {
		t.Fatalf("unexpected synced refs: %+v", refData)
	}

	resolved, err := service.ResolveResourceRefs(refData.Refs)
	if err != nil {
		t.Fatalf("ResolveResourceRefs() error = %v", err)
	}
	if len(resolved) != 1 || resolved[0].Content == nil || *resolved[0].Content != data {
		t.Fatalf("unexpected resolved synced resource: %+v", resolved)
	}

	refData, err = service.GetResourceRefs(GetResourceRefsInput{SourceIDs: []string{"sync_resource_1"}})
	if err != nil {
		t.Fatalf("GetResourceRefs(after preview sync) error = %v", err)
	}
	if len(refData.Refs) != 1 {
		t.Fatalf("unexpected synced refs after preview sync: %+v", refData)
	}
	resolved, err = service.ResolveResourceRefs(refData.Refs)
	if err != nil {
		t.Fatalf("ResolveResourceRefs(after preview sync) error = %v", err)
	}
	if len(resolved) != 1 || resolved[0].Metadata == nil {
		t.Fatalf("unexpected resolved synced resource after preview sync: %+v", resolved)
	}
	if metadataString(resolved[0].Metadata, "previewJson", "") != previewJSON {
		t.Fatalf("expected synced previewJson in resolved resource, got %+v", resolved[0].Metadata)
	}
	if metadataString(resolved[0].Metadata, "ocrPagesJson", "") != ocrPagesJSON {
		t.Fatalf("expected synced ocrPagesJson in resolved resource, got %+v", resolved[0].Metadata)
	}
}

func TestGetFileExposesLegacySnakeCasePreviewMetadata(t *testing.T) {
	service := newTestService(t)
	previewJSON := `{"pages":[{"pageIndex":0,"path":"legacy.png","mimeType":"image/png"}]}`
	ocrPagesJSON := `[{"pageIndex":0,"text":"Legacy OCR","charCount":10,"isFailed":false}]`
	_, err := service.CreateOrUpdateSource(CreateResourceInput{
		Type:     "file",
		Data:     "legacy text",
		SourceID: strPtr("file_legacy_metadata"),
		Metadata: map[string]any{
			"fileRecord":     true,
			"name":           "legacy.pdf",
			"mimeType":       "application/pdf",
			"preview_json":   previewJSON,
			"ocr_pages_json": ocrPagesJSON,
		},
	})
	if err != nil {
		t.Fatalf("CreateOrUpdateSource() error = %v", err)
	}

	file, err := service.GetFile("file_legacy_metadata")
	if err != nil {
		t.Fatalf("GetFile() error = %v", err)
	}
	if file == nil || file.PreviewJSON == nil || *file.PreviewJSON != previewJSON {
		t.Fatalf("expected legacy preview_json to surface as previewJson, got %+v", file)
	}
	if file.OcrPagesJSON == nil || *file.OcrPagesJSON != ocrPagesJSON {
		t.Fatalf("expected legacy ocr_pages_json to surface as ocrPagesJson, got %+v", file.OcrPagesJSON)
	}

	files, err := service.ListFiles(ListFilesInput{FileType: "document", Limit: 10})
	if err != nil {
		t.Fatalf("ListFiles() error = %v", err)
	}
	if len(files) != 1 || files[0].PreviewJSON == nil || *files[0].PreviewJSON != previewJSON || files[0].OcrPagesJSON == nil || *files[0].OcrPagesJSON != ocrPagesJSON {
		t.Fatalf("expected legacy metadata to surface in list API, got %+v", files)
	}
}

func TestLegacySnakeCaseFileMetadataAliases(t *testing.T) {
	service := newTestService(t)
	originalPath := filepath.Join(t.TempDir(), "legacy.pdf")
	if err := os.WriteFile(originalPath, []byte("legacy alias file body"), 0o600); err != nil {
		t.Fatalf("WriteFile(originalPath) error = %v", err)
	}
	bookmarksJSON := `[{"id":"bm_1","page":2,"title":"Legacy bookmark"}]`
	ocrPagesJSON := `[{"pageIndex":0,"text":"Legacy estimated page","charCount":21,"isFailed":false}]`

	_, err := service.CreateOrUpdateSource(CreateResourceInput{
		Type:     "file",
		Data:     "legacy resource data",
		SourceID: strPtr("file_legacy_aliases"),
		Metadata: map[string]any{
			"fileRecord":       true,
			"name":             "legacy-no-extension",
			"mime_type":        "application/pdf",
			"page_count":       "7",
			"extracted_text":   "Legacy extracted text",
			"ocr_text":         "Legacy OCR text",
			"ocr_pages_json":   ocrPagesJSON,
			"ocr_pages_source": "pdf_text_layer_estimated",
			"bookmarks_json":   bookmarksJSON,
			"original_path":    originalPath,
		},
	})
	if err != nil {
		t.Fatalf("CreateOrUpdateSource(legacy aliases) error = %v", err)
	}

	file, err := service.GetFile("file_legacy_aliases")
	if err != nil {
		t.Fatalf("GetFile(legacy aliases) error = %v", err)
	}
	if file == nil || file.MimeType == nil || *file.MimeType != "application/pdf" || file.PageCount == nil || *file.PageCount != 7 {
		t.Fatalf("expected legacy mime_type/page_count to surface, got %+v", file)
	}
	if file.ExtractedText == nil || *file.ExtractedText != "Legacy extracted text" || file.OriginalPath == nil || *file.OriginalPath != originalPath || len(file.Bookmarks) != 1 {
		t.Fatalf("expected legacy text/path/bookmarks to surface, got %+v", file)
	}
	status, err := service.GetPdfProcessingStatus("file_legacy_aliases")
	if err != nil {
		t.Fatalf("GetPdfProcessingStatus(legacy aliases) error = %v", err)
	}
	if status.MediaType != "pdf" || status.TotalPages == nil || *status.TotalPages != 7 || !readyModesContain(status.ReadyModes, "text") || !readyModesContain(status.ReadyModes, "ocr") {
		t.Fatalf("expected legacy aliases to drive PDF text and explicit OCR text status, got %+v", status)
	}
	info, err := service.GetResourceOcrInfo("file_legacy_aliases")
	if err != nil {
		t.Fatalf("GetResourceOcrInfo(legacy aliases) error = %v", err)
	}
	if !info.HasOcr || info.ActiveSource != "ocr" || info.OcrText == nil || *info.OcrText != "Legacy OCR text" || len(info.OcrPages) != 1 {
		t.Fatalf("expected legacy ocr_text to surface as real OCR text while pages remain hydrated, got %+v", info)
	}
	content, err := service.GetFileContent(originalPath)
	if err != nil {
		t.Fatalf("GetFileContent(original_path alias) error = %v", err)
	}
	if !content.Found || content.Content == nil || *content.Content != "legacy resource data" {
		t.Fatalf("expected original_path alias to resolve resource data, got %+v", content)
	}
}

func strPtr(value string) *string {
	return &value
}
