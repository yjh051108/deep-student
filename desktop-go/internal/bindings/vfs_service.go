package bindings

import (
	"deep-student-go/internal/app"
	"deep-student-go/internal/vfs"
)

type VfsService struct {
	app *app.App
}

func NewVfsService(app *app.App) *VfsService {
	return &VfsService{app: app}
}

func (s *VfsService) CreateOrReuse(params vfs.CreateResourceInput) (vfs.CreateResourceResult, error) {
	return s.app.Vfs.CreateOrReuse(params)
}

func (s *VfsService) ResourceSyncNote(noteID string) (vfs.ResourceSyncResult, error) {
	return s.app.Vfs.ResourceSyncNote(noteID)
}

func (s *VfsService) ResourceSyncExam(sessionID string) (vfs.ResourceSyncResult, error) {
	return s.app.Vfs.ResourceSyncExam(sessionID)
}

func (s *VfsService) ResourceSyncTextbookPages(textbookID string, pageRange []int) ([]vfs.ResourceSyncResult, error) {
	return s.app.Vfs.ResourceSyncTextbookPages(textbookID, pageRange)
}

func (s *VfsService) ResourceCheckSyncNeeded(resourceType string, sourceID string, currentHash *string) (vfs.CheckSyncNeededResponse, error) {
	return s.app.Vfs.ResourceCheckSyncNeeded(resourceType, sourceID, currentHash)
}

func (s *VfsService) GetResource(resourceID string) (*vfs.Resource, error) {
	return s.app.Vfs.GetResource(resourceID)
}

func (s *VfsService) ResourceExists(resourceID string) (bool, error) {
	return s.app.Vfs.ResourceExists(resourceID)
}

func (s *VfsService) IncrementRef(resourceID string) (int, error) {
	return s.app.Vfs.IncrementRef(resourceID)
}

func (s *VfsService) DecrementRef(resourceID string) (int, error) {
	return s.app.Vfs.DecrementRef(resourceID)
}

func (s *VfsService) GetResourcePath(sourceID string) (*string, error) {
	return s.app.Vfs.GetResourcePath(sourceID)
}

func (s *VfsService) GetResourceRefCount(sourceID string) (int, error) {
	return s.app.Vfs.GetResourceRefCount(sourceID)
}

func (s *VfsService) UpdateResourceHash(sourceID string, newHash string) (bool, error) {
	return s.app.Vfs.UpdateResourceHash(sourceID, newHash)
}

func (s *VfsService) UploadAttachment(params vfs.UploadAttachmentInput) (vfs.UploadAttachmentResult, error) {
	return s.app.Vfs.UploadAttachment(params)
}

func (s *VfsService) GetAttachment(attachmentID string) (*vfs.Attachment, error) {
	return s.app.Vfs.GetAttachment(attachmentID)
}

func (s *VfsService) GetAttachmentContent(attachmentID string) (vfs.AttachmentContentResult, error) {
	return s.app.Vfs.GetAttachmentContent(attachmentID)
}

func (s *VfsService) UploadFile(params vfs.UploadFileInput) (vfs.UploadFileResult, error) {
	return s.app.Vfs.UploadFile(params)
}

func (s *VfsService) GetFile(fileID string) (*vfs.VfsFile, error) {
	return s.app.Vfs.GetFile(fileID)
}

func (s *VfsService) DeleteFile(fileID string) error {
	return s.app.Vfs.DeleteFile(fileID)
}

func (s *VfsService) GetFileContent(fileID string) (vfs.AttachmentContentResult, error) {
	return s.app.Vfs.GetFileContent(fileID)
}

func (s *VfsService) UpdateBookmarks(fileID string, bookmarks []any) (bool, error) {
	return s.app.Vfs.UpdateBookmarks(fileID, bookmarks)
}

func (s *VfsService) GetResourceRefs(input vfs.GetResourceRefsInput) (vfs.ContextRefData, error) {
	return s.app.Vfs.GetResourceRefs(input)
}

func (s *VfsService) ResolveResourceRefs(refs []vfs.ResourceRef) ([]vfs.ResolvedResource, error) {
	return s.app.Vfs.ResolveResourceRefs(refs)
}

func (s *VfsService) UpdatePathCache(folderID string) (int, error) {
	return s.app.Vfs.UpdatePathCache(folderID)
}

func (s *VfsService) GetPdfProcessingStatus(fileID string) (vfs.PdfProcessingStatus, error) {
	return s.app.Vfs.GetPdfProcessingStatus(fileID)
}

func (s *VfsService) GetBatchPdfProcessingStatus(fileIDs []string) (map[string]vfs.PdfProcessingStatus, error) {
	return s.app.Vfs.GetBatchPdfProcessingStatus(fileIDs)
}

func (s *VfsService) CancelPdfProcessing(fileID string) (bool, error) {
	return s.app.Vfs.CancelPdfProcessing(fileID)
}

func (s *VfsService) RetryPdfProcessing(fileID string) error {
	return s.app.Vfs.RetryPdfProcessing(fileID)
}

func (s *VfsService) StartPdfProcessing(fileID string, startFromStage *string) error {
	return s.app.Vfs.StartPdfProcessing(fileID, startFromStage)
}

func (s *VfsService) GetPdfPageImage(resourceID string, pageIndex int) (vfs.PdfPageImageResult, error) {
	return s.app.Vfs.GetPdfPageImage(resourceID, pageIndex)
}

func (s *VfsService) GetBlobBase64(blobHash string) (vfs.VfsBlobBase64Result, error) {
	return s.app.Vfs.GetBlobBase64(blobHash)
}

func (s *VfsService) UnifiedIndexStatus() (vfs.IndexStatusSummary, error) {
	return s.app.Vfs.UnifiedIndexStatus()
}

func (s *VfsService) GetResourceUnits(resourceID string) ([]vfs.UnitIndexStatus, error) {
	return s.app.Vfs.GetResourceUnits(resourceID)
}

func (s *VfsService) SyncResourceUnits(input vfs.SyncResourceUnitsInput) ([]vfs.UnitIndexStatus, error) {
	return s.app.Vfs.SyncResourceUnits(input)
}

func (s *VfsService) GetAllIndexStatus(input vfs.GetIndexStatusInput) (vfs.ResourceIndexStatusSummary, error) {
	return s.app.Vfs.GetAllIndexStatus(input)
}

func (s *VfsService) ReindexResource(resourceID string) (int, error) {
	return s.app.Vfs.ReindexResource(resourceID)
}

func (s *VfsService) ReindexUnit(unitID string, mode string) (bool, error) {
	return s.app.Vfs.ReindexUnit(unitID, mode)
}

func (s *VfsService) BatchIndexPending(batchSize int) (vfs.BatchIndexResult, error) {
	return s.app.Vfs.BatchIndexPending(batchSize)
}

func (s *VfsService) DeleteResourceIndex(resourceID string) (vfs.DeleteIndexResult, error) {
	return s.app.Vfs.DeleteResourceIndex(resourceID)
}

func (s *VfsService) ListEmbeddingDims() ([]vfs.EmbeddingDimInfo, error) {
	return s.app.Vfs.ListEmbeddingDims()
}

func (s *VfsService) ListDimensions() ([]vfs.VfsEmbeddingDimension, error) {
	return s.app.Vfs.ListDimensions()
}

func (s *VfsService) GetResourceTextChunks(resourceID string) ([]vfs.TextChunkInfo, error) {
	return s.app.Vfs.GetResourceTextChunks(resourceID)
}

func (s *VfsService) GetResourceOcrInfo(resourceID string) (vfs.ResourceOcrInfo, error) {
	return s.app.Vfs.GetResourceOcrInfo(resourceID)
}

func (s *VfsService) ClearResourceOcr(resourceID string) (bool, error) {
	return s.app.Vfs.ClearResourceOcr(resourceID)
}

func (s *VfsService) RagSearch(input vfs.VfsRagSearchInput) (vfs.VfsRagSearchOutput, error) {
	return s.app.Vfs.RagSearch(input)
}

func (s *VfsService) ListFiles(input vfs.ListFilesInput) ([]vfs.VfsFile, error) {
	return s.app.Vfs.ListFiles(input)
}

func (s *VfsService) CreateMindMap(params vfs.CreateMindMapInput) (vfs.VfsMindMap, error) {
	return s.app.Vfs.CreateMindMap(params)
}

func (s *VfsService) GetMindMap(mindmapID string) (*vfs.VfsMindMap, error) {
	return s.app.Vfs.GetMindMap(mindmapID)
}

func (s *VfsService) GetMindMapContent(mindmapID string) (*string, error) {
	return s.app.Vfs.GetMindMapContent(mindmapID)
}

func (s *VfsService) UpdateMindMap(mindmapID string, params vfs.UpdateMindMapInput) (vfs.VfsMindMap, error) {
	return s.app.Vfs.UpdateMindMap(mindmapID, params)
}

func (s *VfsService) DeleteMindMap(mindmapID string) error {
	return s.app.Vfs.DeleteMindMap(mindmapID)
}

func (s *VfsService) ListMindMaps() ([]vfs.VfsMindMap, error) {
	return s.app.Vfs.ListMindMaps()
}

func (s *VfsService) SetMindMapFavorite(mindmapID string, isFavorite bool) error {
	return s.app.Vfs.SetMindMapFavorite(mindmapID, isFavorite)
}

func (s *VfsService) GetMindMapVersions(mindmapID string) ([]vfs.VfsMindMapVersion, error) {
	return s.app.Vfs.GetMindMapVersions(mindmapID)
}

func (s *VfsService) GetMindMapVersion(versionID string) (*vfs.VfsMindMapVersion, error) {
	return s.app.Vfs.GetMindMapVersion(versionID)
}

func (s *VfsService) GetMindMapVersionContent(versionID string) (*string, error) {
	return s.app.Vfs.GetMindMapVersionContent(versionID)
}
