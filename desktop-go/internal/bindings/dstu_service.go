package bindings

import (
	"deep-student-go/internal/app"
	"deep-student-go/internal/dstu"
)

type DstuService struct {
	app *app.App
}

func NewDstuService(app *app.App) *DstuService {
	return &DstuService{app: app}
}

func (s *DstuService) List(path string, options *dstu.ListOptions) ([]dstu.Node, error) {
	return s.app.Dstu.List(path, options)
}

func (s *DstuService) Get(path string) (*dstu.Node, error) {
	return s.app.Dstu.Get(path)
}

func (s *DstuService) Create(path string, options dstu.CreateOptions) (dstu.Node, error) {
	return s.app.Dstu.Create(path, options)
}

func (s *DstuService) Update(path string, content string, resourceType string) (dstu.Node, error) {
	return s.app.Dstu.Update(path, content, resourceType)
}

func (s *DstuService) Delete(path string) (bool, error) {
	return s.app.Dstu.Delete(path)
}

func (s *DstuService) DeleteMany(paths []string) (int, error) {
	return s.app.Dstu.DeleteMany(paths)
}

func (s *DstuService) Search(query string, options *dstu.ListOptions) ([]dstu.Node, error) {
	return s.app.Dstu.Search(query, options)
}

func (s *DstuService) GetContent(path string) (string, error) {
	return s.app.Dstu.GetContent(path)
}

func (s *DstuService) NotesSearch(keyword string, limit int) ([]dstu.NotesSearchHit, error) {
	return s.app.Dstu.NotesSearch(keyword, limit)
}

func (s *DstuService) ListTags() ([]string, error) {
	return s.app.Dstu.ListTags()
}

func (s *DstuService) CanvasReadContent(noteID string, section *string) (string, error) {
	return s.app.Dstu.CanvasReadContent(noteID, section)
}

func (s *DstuService) CanvasAppendContent(noteID string, content string, section *string) error {
	return s.app.Dstu.CanvasAppendContent(noteID, content, section)
}

func (s *DstuService) CanvasReplaceContent(noteID string, search string, replace string, isRegex bool) (int, error) {
	return s.app.Dstu.CanvasReplaceContent(noteID, search, replace, isRegex)
}

func (s *DstuService) CanvasSetContent(noteID string, content string) error {
	return s.app.Dstu.CanvasSetContent(noteID, content)
}

func (s *DstuService) SetMetadata(path string, metadata map[string]any) (bool, error) {
	return s.app.Dstu.SetMetadata(path, metadata)
}

func (s *DstuService) SetFavorite(path string, favorite bool) (bool, error) {
	return s.app.Dstu.SetFavorite(path, favorite)
}

func (s *DstuService) ImportMarkdown(request dstu.ImportMarkdownRequest) (dstu.Node, error) {
	return s.app.Dstu.ImportMarkdown(request)
}

func (s *DstuService) ImportMarkdownBatch(request dstu.ImportMarkdownBatchRequest) (dstu.ImportMarkdownBatchResponse, error) {
	return s.app.Dstu.ImportMarkdownBatch(request)
}

func (s *DstuService) AddTextbooks(request dstu.AddTextbooksRequest) ([]dstu.TextbookRecord, error) {
	return s.app.Dstu.AddTextbooks(request)
}

func (s *DstuService) CreateFolder(title string, parentID *string, icon *string, color *string) (dstu.VfsFolder, error) {
	return s.app.Dstu.CreateFolder(title, parentID, icon, color)
}

func (s *DstuService) GetFolder(folderID string) (*dstu.VfsFolder, error) {
	return s.app.Dstu.GetFolder(folderID)
}

func (s *DstuService) RenameFolder(folderID string, title string) error {
	return s.app.Dstu.RenameFolder(folderID, title)
}

func (s *DstuService) DeleteFolder(folderID string) error {
	return s.app.Dstu.DeleteFolder(folderID)
}

func (s *DstuService) MoveFolder(folderID string, newParentID *string) error {
	return s.app.Dstu.MoveFolder(folderID, newParentID)
}

func (s *DstuService) SetFolderExpanded(folderID string, isExpanded bool) error {
	return s.app.Dstu.SetFolderExpanded(folderID, isExpanded)
}

func (s *DstuService) AddFolderItem(folderID *string, itemType string, itemID string) (dstu.VfsFolderItem, error) {
	return s.app.Dstu.AddFolderItem(folderID, itemType, itemID)
}

func (s *DstuService) RemoveFolderItem(itemType string, itemID string) error {
	return s.app.Dstu.RemoveFolderItem(itemType, itemID)
}

func (s *DstuService) MoveFolderItem(itemType string, itemID string, newFolderID *string) error {
	return s.app.Dstu.MoveFolderItem(itemType, itemID, newFolderID)
}

func (s *DstuService) ListFolders() ([]dstu.VfsFolder, error) {
	return s.app.Dstu.ListFolders()
}

func (s *DstuService) GetFolderTree() ([]dstu.FolderTreeNode, error) {
	return s.app.Dstu.GetFolderTree()
}

func (s *DstuService) GetFolderItems(folderID *string) ([]dstu.VfsFolderItem, error) {
	return s.app.Dstu.GetFolderItems(folderID)
}

func (s *DstuService) ReorderFolders(folderIDs []string) error {
	return s.app.Dstu.ReorderFolders(folderIDs)
}

func (s *DstuService) ReorderFolderItems(folderID *string, itemIDs []string) error {
	return s.app.Dstu.ReorderFolderItems(folderID, itemIDs)
}

func (s *DstuService) GetFolderBreadcrumbs(folderID string) ([]dstu.BreadcrumbItem, error) {
	return s.app.Dstu.GetFolderBreadcrumbs(folderID)
}

func (s *DstuService) ParsePath(path string) (dstu.ParsedPath, error) {
	return s.app.Dstu.ParsePath(path)
}

func (s *DstuService) BuildPath(folderID *string, resourceID string) (string, error) {
	return s.app.Dstu.BuildPath(folderID, resourceID)
}

func (s *DstuService) MoveToFolder(resourceID string, targetFolderID *string) (dstu.ResourceLocation, error) {
	return s.app.Dstu.MoveToFolder(resourceID, targetFolderID)
}

func (s *DstuService) BatchMove(request dstu.BatchMoveRequest) (dstu.BatchMoveResult, error) {
	return s.app.Dstu.BatchMove(request)
}

func (s *DstuService) RefreshPathCache(resourceID *string) (int, error) {
	return s.app.Dstu.RefreshPathCache(resourceID)
}

func (s *DstuService) GetPathByID(resourceID string) (string, error) {
	return s.app.Dstu.GetPathByID(resourceID)
}

func (s *DstuService) GetResourceByPath(path string) (*dstu.Node, error) {
	return s.app.Dstu.GetResourceByPath(path)
}

func (s *DstuService) GetResourceLocation(resourceID string) (dstu.ResourceLocation, error) {
	return s.app.Dstu.GetResourceLocation(resourceID)
}

func (s *DstuService) GetFolderAllResources(folderID string, includeSubfolders bool, includeContent bool) (dstu.FolderResourcesResult, error) {
	return s.app.Dstu.GetFolderAllResources(folderID, includeSubfolders, includeContent)
}

func (s *DstuService) Restore(path string) (dstu.Node, error) {
	return s.app.Dstu.Restore(path)
}

func (s *DstuService) RestoreMany(paths []string) (int, error) {
	return s.app.Dstu.RestoreMany(paths)
}

func (s *DstuService) Purge(path string) error {
	return s.app.Dstu.Purge(path)
}

func (s *DstuService) PurgeAll(resourceType string) (int, error) {
	return s.app.Dstu.PurgeAll(resourceType)
}

func (s *DstuService) ListDeleted(resourceType string, limit *int, offset *int) ([]dstu.Node, error) {
	return s.app.Dstu.ListDeleted(resourceType, limit, offset)
}

func (s *DstuService) SoftDelete(id string, itemType string) error {
	return s.app.Dstu.SoftDelete(id, itemType)
}

func (s *DstuService) TrashRestore(id string, itemType string) error {
	return s.app.Dstu.TrashRestore(id, itemType)
}

func (s *DstuService) ListTrash(limit *int, offset *int) ([]dstu.Node, error) {
	return s.app.Dstu.ListTrash(limit, offset)
}

func (s *DstuService) EmptyTrash() (int, error) {
	return s.app.Dstu.EmptyTrash()
}

func (s *DstuService) PermanentlyDelete(id string, itemType string) error {
	return s.app.Dstu.PermanentlyDelete(id, itemType)
}
