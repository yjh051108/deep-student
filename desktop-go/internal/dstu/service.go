package dstu

import (
	"crypto/rand"
	"crypto/sha256"
	"deep-student-go/internal/storage"
	"deep-student-go/internal/vfs"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"math/big"
	"mime"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"sync"
	"time"
	"unicode/utf16"
)

var noteAssetReferencePattern = regexp.MustCompile(`notes_assets/[^\s\])"'<>]+`)

type Service struct {
	mu      sync.RWMutex
	eventMu sync.RWMutex
	path    string
	state   store
	vfs     *vfs.Service
	emit    func(name string, payload any)
}

type store struct {
	Notes       []NoteRecord    `json:"notes"`
	Folders     []VfsFolder     `json:"folders"`
	FolderItems []VfsFolderItem `json:"folderItems"`
}

type NoteRecord struct {
	ID           string         `json:"id"`
	Name         string         `json:"name"`
	Content      string         `json:"content"`
	Metadata     map[string]any `json:"metadata"`
	ResourceID   string         `json:"resourceId,omitempty"`
	ResourceHash string         `json:"resourceHash,omitempty"`
	CreatedAt    int64          `json:"createdAt"`
	UpdatedAt    int64          `json:"updatedAt"`
}

type Node struct {
	ID           string         `json:"id"`
	Path         string         `json:"path"`
	Name         string         `json:"name"`
	Type         string         `json:"type"`
	Size         int            `json:"size,omitempty"`
	CreatedAt    int64          `json:"createdAt"`
	UpdatedAt    int64          `json:"updatedAt"`
	ResourceID   string         `json:"resourceId,omitempty"`
	SourceID     string         `json:"sourceId"`
	ResourceHash string         `json:"resourceHash,omitempty"`
	PreviewType  string         `json:"previewType,omitempty"`
	Metadata     map[string]any `json:"metadata,omitempty"`
}

type ListOptions struct {
	FolderID   *string  `json:"folderId,omitempty"`
	TypeFilter *string  `json:"typeFilter,omitempty"`
	IsFavorite *bool    `json:"isFavorite,omitempty"`
	Recursive  *bool    `json:"recursive,omitempty"`
	Types      []string `json:"types,omitempty"`
	Search     *string  `json:"search,omitempty"`
	Tags       []string `json:"tags,omitempty"`
	SortBy     *string  `json:"sortBy,omitempty"`
	SortOrder  *string  `json:"sortOrder,omitempty"`
	Limit      *int     `json:"limit,omitempty"`
	Offset     *int     `json:"offset,omitempty"`
}

type VfsFolder struct {
	ID          string  `json:"id"`
	ParentID    *string `json:"parentId"`
	Title       string  `json:"title"`
	Icon        *string `json:"icon,omitempty"`
	Color       *string `json:"color,omitempty"`
	IsExpanded  bool    `json:"isExpanded"`
	SortOrder   int     `json:"sortOrder"`
	IsBuiltin   bool    `json:"isBuiltin,omitempty"`
	BuiltinType *string `json:"builtinType,omitempty"`
	CreatedAt   int64   `json:"createdAt"`
	UpdatedAt   int64   `json:"updatedAt"`
}

type VfsFolderItem struct {
	ID         string  `json:"id"`
	FolderID   *string `json:"folderId"`
	ItemType   string  `json:"itemType"`
	ItemID     string  `json:"itemId"`
	SortOrder  int     `json:"sortOrder"`
	CreatedAt  int64   `json:"createdAt"`
	UpdatedAt  int64   `json:"updatedAt,omitempty"`
	CachedPath *string `json:"cachedPath,omitempty"`
}

type FolderTreeNode struct {
	Folder   VfsFolder        `json:"folder"`
	Children []FolderTreeNode `json:"children"`
	Items    []VfsFolderItem  `json:"items"`
}

type BreadcrumbItem struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

type ParsedPath struct {
	FullPath     string  `json:"fullPath"`
	FolderPath   *string `json:"folderPath"`
	ResourceID   *string `json:"resourceId"`
	ID           *string `json:"id"`
	ResourceType *string `json:"resourceType"`
	IsRoot       bool    `json:"isRoot"`
	IsVirtual    bool    `json:"isVirtual"`
	VirtualType  *string `json:"virtualType,omitempty"`
}

type BatchMoveRequest struct {
	ItemIDs        []string `json:"itemIds"`
	TargetFolderID *string  `json:"targetFolderId"`
}

type FailedMoveItem struct {
	ItemID string `json:"itemId"`
	Error  string `json:"error"`
}

type BatchMoveResult struct {
	Successes   []ResourceLocation `json:"successes"`
	FailedItems []FailedMoveItem   `json:"failedItems"`
	TotalCount  int                `json:"totalCount"`
}

type CreateOptions struct {
	Type     string         `json:"type"`
	Name     string         `json:"name"`
	Content  *string        `json:"content,omitempty"`
	FileBase *string        `json:"fileBase64,omitempty"`
	Metadata map[string]any `json:"metadata,omitempty"`
}

type ImportMarkdownRequest struct {
	FilePath  string  `json:"filePath"`
	TitleHint *string `json:"titleHint,omitempty"`
	FolderID  *string `json:"folderId,omitempty"`
}

type ImportMarkdownBatchItem struct {
	FilePath  string  `json:"filePath"`
	TitleHint *string `json:"titleHint,omitempty"`
}

type ImportMarkdownBatchRequest struct {
	Items    []ImportMarkdownBatchItem `json:"items"`
	FolderID *string                   `json:"folderId,omitempty"`
}

type ImportMarkdownBatchFailure struct {
	FilePath string `json:"file_path"`
	Message  string `json:"message"`
}

type ImportMarkdownBatchResponse struct {
	Imported []Node                       `json:"imported"`
	Failed   []ImportMarkdownBatchFailure `json:"failed"`
}

type AddTextbooksRequest struct {
	Sources  []string `json:"sources"`
	FolderID *string  `json:"folderId,omitempty"`
}

type textbookImportProgressPayload struct {
	FileName    string  `json:"file_name"`
	Stage       string  `json:"stage"`
	CurrentPage *int    `json:"current_page,omitempty"`
	TotalPages  *int    `json:"total_pages,omitempty"`
	Progress    int     `json:"progress"`
	Error       *string `json:"error,omitempty"`
	Source      string  `json:"source,omitempty"`
	ImportID    string  `json:"import_id,omitempty"`
	Index       int     `json:"index"`
	Total       int     `json:"total"`
	TextbookID  string  `json:"textbook_id,omitempty"`
	ResourceID  string  `json:"resource_id,omitempty"`
}

type TextbookRecord struct {
	ID            string         `json:"id"`
	SHA256        string         `json:"sha256"`
	FileName      string         `json:"file_name"`
	FilePath      string         `json:"file_path"`
	Size          int64          `json:"size"`
	PageCount     *int           `json:"page_count,omitempty"`
	TagsJSON      string         `json:"tags_json"`
	Favorite      int            `json:"favorite"`
	LastOpenedAt  *string        `json:"last_opened_at,omitempty"`
	LastPage      *int           `json:"last_page,omitempty"`
	BookmarksJSON string         `json:"bookmarks_json"`
	CoverKey      *string        `json:"cover_key,omitempty"`
	OriginJSON    *string        `json:"origin_json,omitempty"`
	Status        string         `json:"status"`
	CreatedAt     string         `json:"created_at"`
	UpdatedAt     string         `json:"updated_at"`
	ResourceID    string         `json:"resource_id,omitempty"`
	ResourceHash  string         `json:"resource_hash,omitempty"`
	PreviewType   string         `json:"preview_type,omitempty"`
	Metadata      map[string]any `json:"metadata,omitempty"`
}

type ResourceLocation struct {
	ID           string  `json:"id"`
	ResourceType string  `json:"resourceType"`
	FolderID     *string `json:"folderId"`
	FolderPath   string  `json:"folderPath"`
	FullPath     string  `json:"fullPath"`
	Hash         *string `json:"hash,omitempty"`
}

type FolderResourceInfo struct {
	ItemType   string  `json:"itemType"`
	ItemID     string  `json:"itemId"`
	ResourceID *string `json:"resourceId,omitempty"`
	Title      string  `json:"title"`
	Path       string  `json:"path"`
	Content    *string `json:"content,omitempty"`
}

type FolderResourcesResult struct {
	FolderID    string               `json:"folderId"`
	FolderTitle string               `json:"folderTitle"`
	Path        string               `json:"path"`
	TotalCount  int                  `json:"totalCount"`
	Resources   []FolderResourceInfo `json:"resources"`
}

type NotesSearchHit struct {
	ID      string  `json:"id"`
	Title   string  `json:"title"`
	Snippet *string `json:"snippet,omitempty"`
}

func NewService(dataDir string, vfsServices ...*vfs.Service) (*Service, error) {
	service := &Service{
		path: filepath.Join(dataDir, "dstu-go.json"),
		state: store{
			Notes:       []NoteRecord{},
			Folders:     []VfsFolder{},
			FolderItems: []VfsFolderItem{},
		},
	}
	if len(vfsServices) > 0 {
		service.vfs = vfsServices[0]
	}
	if err := service.load(); err != nil {
		return nil, err
	}
	return service, nil
}

func (s *Service) SetEventEmitter(emit func(name string, payload any)) {
	s.eventMu.Lock()
	defer s.eventMu.Unlock()
	s.emit = emit
}

func (s *Service) currentEmitter() func(name string, payload any) {
	s.eventMu.RLock()
	defer s.eventMu.RUnlock()
	return s.emit
}

func (s *Service) emitEvent(name string, payload any) {
	if emit := s.currentEmitter(); emit != nil {
		emit(name, payload)
	}
}

func (s *Service) List(path string, options *ListOptions) ([]Node, error) {
	if !isRootPath(path) {
		return []Node{}, nil
	}

	s.mu.RLock()
	notes := append([]NoteRecord(nil), s.state.Notes...)
	folders := append([]VfsFolder(nil), s.state.Folders...)
	folderItems := append([]VfsFolderItem(nil), s.state.FolderItems...)
	s.mu.RUnlock()

	itemLookup := folderItemLookup(folderItems)
	folderPathIDs := folderPathIDLookup(folders)
	folderPaths := folderTitlePathLookup(folders)
	nodes := make([]Node, 0, len(notes))
	for _, note := range notes {
		if !matchesListOptions(note, options) {
			continue
		}
		node := noteToNode(note)
		applyFolderItemToNode(&node, itemLookup, folderPathIDs, folderPaths)
		if !matchesNodeFolderOption(node, options) {
			continue
		}
		nodes = append(nodes, node)
	}
	if s.vfs != nil && listOptionsMayIncludeVfs(options) {
		files, err := s.listVfsFiles()
		if err != nil {
			return nil, err
		}
		for _, file := range files {
			node := vfsFileToNode(file)
			applyFolderItemToNode(&node, itemLookup, folderPathIDs, folderPaths)
			if !matchesNodeListOptions(node, options) {
				continue
			}
			nodes = append(nodes, node)
		}
	}
	sortNodes(nodes, options)
	return paginate(nodes, options), nil
}

func (s *Service) Get(path string) (*Node, error) {
	id := idFromPath(path)
	if id == "" {
		return nil, nil
	}

	s.mu.RLock()
	note, ok := s.findNoteLocked(id)
	s.mu.RUnlock()
	if ok {
		if noteIsDeleted(note) {
			return nil, nil
		}
		node := noteToNode(note)
		s.applyStoredFolderItemToNode(&node)
		return &node, nil
	}

	if s.vfs == nil {
		return nil, nil
	}
	file, err := s.vfs.GetFile(id)
	if err != nil {
		return nil, err
	}
	if file == nil {
		return nil, nil
	}
	node := vfsFileToNode(*file)
	s.applyStoredFolderItemToNode(&node)
	return &node, nil
}

func (s *Service) Create(path string, options CreateOptions) (Node, error) {
	name := strings.TrimSpace(options.Name)
	if name == "" {
		name = "Untitled"
	}
	if options.Type != "note" {
		return s.createVfsBackedNode(path, name, options)
	}
	content := ""
	if options.Content != nil {
		content = *options.Content
	}

	now := nowMillis()
	note := NoteRecord{
		ID:        "note_" + randomToken(16),
		Name:      name,
		Content:   content,
		Metadata:  normalizeMetadata(options.Metadata),
		CreatedAt: now,
		UpdatedAt: now,
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	s.state.Notes = append(s.state.Notes, note)
	index := len(s.state.Notes) - 1
	if err := s.syncNoteResourceLocked(index); err != nil {
		s.state.Notes = s.state.Notes[:index]
		return Node{}, err
	}
	if err := s.flushLocked(); err != nil {
		return Node{}, err
	}
	return noteToNode(s.state.Notes[index]), nil
}

func (s *Service) Update(path string, content string, resourceType string) (Node, error) {
	if resourceType != "" && resourceType != "note" {
		return Node{}, fmt.Errorf("lean Go DSTU currently supports note updates, got %q", resourceType)
	}
	id := idFromPath(path)
	if id == "" {
		return Node{}, errors.New("dstu path does not contain a resource id")
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	index, ok := s.findNoteIndexLocked(id)
	if !ok {
		return Node{}, fmt.Errorf("resource not found: %s", path)
	}
	previous := s.state.Notes[index]
	s.state.Notes[index].Content = content
	s.state.Notes[index].UpdatedAt = nowMillis()
	if err := s.syncNoteResourceLocked(index); err != nil {
		s.state.Notes[index] = previous
		return Node{}, err
	}
	if err := s.flushLocked(); err != nil {
		return Node{}, err
	}
	return noteToNode(s.state.Notes[index]), nil
}

func (s *Service) Delete(path string) (bool, error) {
	id := idFromPath(path)
	if id == "" {
		return false, errors.New("dstu path does not contain a resource id")
	}

	s.mu.Lock()
	index, ok := s.findNoteIndexLocked(id)
	if !ok {
		s.mu.Unlock()
		if s.vfs == nil {
			return false, nil
		}
		file, err := s.vfs.GetFile(id)
		if err != nil {
			return false, err
		}
		if file == nil {
			return false, nil
		}
		if err := s.vfs.DeleteFile(id); err != nil {
			return false, err
		}
		return true, nil
	}
	previous := s.state.Notes[index]
	metadata := normalizeMetadata(previous.Metadata)
	metadata["status"] = "deleted"
	metadata["deletedAt"] = formatMillis(nowMillis())
	s.state.Notes[index].Metadata = metadata
	s.state.Notes[index].UpdatedAt = nowMillis()
	if err := s.syncNoteResourceLocked(index); err != nil {
		s.state.Notes[index] = previous
		s.mu.Unlock()
		return false, err
	}
	err := s.flushLocked()
	s.mu.Unlock()
	return true, err
}

func (s *Service) DeleteMany(paths []string) (int, error) {
	count := 0
	for _, path := range paths {
		deleted, err := s.Delete(path)
		if err != nil {
			return count, err
		}
		if deleted {
			count++
		}
	}
	return count, nil
}

func (s *Service) ListDeleted(resourceType string, limit *int, offset *int) ([]Node, error) {
	if !resourceTypeAllowsNotes(resourceType) {
		return []Node{}, nil
	}
	s.mu.RLock()
	notes := append([]NoteRecord(nil), s.state.Notes...)
	folderItems := append([]VfsFolderItem(nil), s.state.FolderItems...)
	folders := append([]VfsFolder(nil), s.state.Folders...)
	s.mu.RUnlock()

	itemLookup := folderItemLookup(folderItems)
	folderPathIDs := folderPathIDLookup(folders)
	folderPaths := folderTitlePathLookup(folders)
	nodes := make([]Node, 0, len(notes))
	for _, note := range notes {
		if !noteIsDeleted(note) {
			continue
		}
		node := noteToNode(note)
		applyFolderItemToNode(&node, itemLookup, folderPathIDs, folderPaths)
		nodes = append(nodes, node)
	}
	sort.Slice(nodes, func(left, right int) bool {
		return nodes[left].UpdatedAt > nodes[right].UpdatedAt
	})
	return paginateNodes(nodes, limit, offset), nil
}

func (s *Service) Restore(path string) (Node, error) {
	id := idFromPath(path)
	if id == "" {
		return Node{}, errors.New("dstu path does not contain a resource id")
	}

	s.mu.Lock()
	index, ok := s.findNoteIndexLocked(id)
	if !ok {
		s.mu.Unlock()
		return Node{}, fmt.Errorf("resource not found: %s", path)
	}
	previous := s.state.Notes[index]
	metadata := normalizeMetadata(previous.Metadata)
	metadata["status"] = "active"
	metadata["deletedAt"] = ""
	metadata["restoredAt"] = formatMillis(nowMillis())
	s.state.Notes[index].Metadata = metadata
	s.state.Notes[index].UpdatedAt = nowMillis()
	if err := s.syncNoteResourceLocked(index); err != nil {
		s.state.Notes[index] = previous
		s.mu.Unlock()
		return Node{}, err
	}
	if err := s.flushLocked(); err != nil {
		s.mu.Unlock()
		return Node{}, err
	}
	node := noteToNode(s.state.Notes[index])
	s.mu.Unlock()
	s.applyStoredFolderItemToNode(&node)
	return node, nil
}

func (s *Service) RestoreMany(paths []string) (int, error) {
	count := 0
	for _, path := range paths {
		if _, err := s.Restore(path); err != nil {
			return count, err
		}
		count++
	}
	return count, nil
}

func (s *Service) Purge(path string) error {
	id := idFromPath(path)
	if id == "" {
		return errors.New("dstu path does not contain a resource id")
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	index, ok := s.findNoteIndexLocked(id)
	if !ok {
		return nil
	}
	s.state.Notes = append(s.state.Notes[:index], s.state.Notes[index+1:]...)
	s.removeFolderItemLocked("note", id)
	return s.flushLocked()
}

func (s *Service) PurgeAll(resourceType string) (int, error) {
	if !resourceTypeAllowsNotes(resourceType) {
		return 0, nil
	}
	s.mu.Lock()
	defer s.mu.Unlock()

	kept := make([]NoteRecord, 0, len(s.state.Notes))
	deletedIDs := map[string]bool{}
	for _, note := range s.state.Notes {
		if noteIsDeleted(note) {
			deletedIDs[note.ID] = true
			continue
		}
		kept = append(kept, note)
	}
	if len(deletedIDs) == 0 {
		return 0, nil
	}
	s.state.Notes = kept
	s.state.FolderItems = filterFolderItems(s.state.FolderItems, func(item VfsFolderItem) bool {
		return !(item.ItemType == "note" && deletedIDs[item.ItemID])
	})
	return len(deletedIDs), s.flushLocked()
}

func (s *Service) SoftDelete(id string, itemType string) error {
	itemType = normalizeItemType(itemType, id)
	if itemType != "note" {
		return fmt.Errorf("lean Go DSTU trash currently supports notes, got %q", itemType)
	}
	_, err := s.Delete("/" + idFromPath(id))
	return err
}

func (s *Service) TrashRestore(id string, itemType string) error {
	itemType = normalizeItemType(itemType, id)
	if itemType != "note" {
		return fmt.Errorf("lean Go DSTU trash currently supports notes, got %q", itemType)
	}
	_, err := s.Restore("/" + idFromPath(id))
	return err
}

func (s *Service) ListTrash(limit *int, offset *int) ([]Node, error) {
	return s.ListDeleted("notes", limit, offset)
}

func (s *Service) EmptyTrash() (int, error) {
	return s.PurgeAll("notes")
}

func (s *Service) PermanentlyDelete(id string, itemType string) error {
	itemType = normalizeItemType(itemType, id)
	if itemType != "note" {
		return fmt.Errorf("lean Go DSTU trash currently supports notes, got %q", itemType)
	}
	return s.Purge("/" + idFromPath(id))
}

func (s *Service) Search(query string, options *ListOptions) ([]Node, error) {
	normalizedQuery := strings.TrimSpace(query)
	if options == nil {
		options = &ListOptions{}
	}
	options.Search = &normalizedQuery
	return s.List("/", options)
}

func (s *Service) NotesSearch(keyword string, limit int) ([]NotesSearchHit, error) {
	keyword = strings.TrimSpace(keyword)
	if keyword == "" {
		return []NotesSearchHit{}, nil
	}
	if limit <= 0 {
		limit = 50
	}
	if limit > 200 {
		limit = 200
	}

	tagFilters, terms := parseNotesSearchKeyword(keyword)
	s.mu.RLock()
	notes := append([]NoteRecord(nil), s.state.Notes...)
	s.mu.RUnlock()

	hits := make([]NotesSearchHit, 0, len(notes))
	for _, note := range notes {
		if noteIsDeleted(note) {
			continue
		}
		if len(tagFilters) > 0 && !hasAllTagsFold(metadataTags(note.Metadata), tagFilters) {
			continue
		}
		if len(terms) > 0 && !noteMatchesTerms(note, terms) {
			continue
		}
		snippet := (*string)(nil)
		if len(terms) > 0 {
			snippet = snippetForQuery(note.Content, strings.Join(terms, " "), 160)
		}
		hits = append(hits, NotesSearchHit{
			ID:      note.ID,
			Title:   note.Name,
			Snippet: snippet,
		})
		if len(hits) >= limit {
			break
		}
	}
	return hits, nil
}

func (s *Service) ListTags() ([]string, error) {
	s.mu.RLock()
	notes := append([]NoteRecord(nil), s.state.Notes...)
	s.mu.RUnlock()

	counts := map[string]int{}
	display := map[string]string{}
	for _, note := range notes {
		if noteIsDeleted(note) {
			continue
		}
		for _, tag := range metadataTags(note.Metadata) {
			key := strings.ToLower(strings.TrimSpace(tag))
			if key == "" {
				continue
			}
			counts[key]++
			if _, ok := display[key]; !ok {
				display[key] = strings.TrimSpace(tag)
			}
		}
	}
	keys := make([]string, 0, len(counts))
	for key := range counts {
		keys = append(keys, key)
	}
	sort.Slice(keys, func(a, b int) bool {
		left := keys[a]
		right := keys[b]
		if counts[left] != counts[right] {
			return counts[left] > counts[right]
		}
		return display[left] < display[right]
	})
	if len(keys) > 50 {
		keys = keys[:50]
	}
	tags := make([]string, 0, len(keys))
	for _, key := range keys {
		tags = append(tags, display[key])
	}
	return tags, nil
}

func (s *Service) CountNotes(includeDeleted bool) int {
	s.mu.RLock()
	defer s.mu.RUnlock()

	count := 0
	for _, note := range s.state.Notes {
		if !includeDeleted && noteIsDeleted(note) {
			continue
		}
		count++
	}
	return count
}

func (s *Service) NoteAssetReferences(includeDeleted bool) []string {
	s.mu.RLock()
	notes := append([]NoteRecord(nil), s.state.Notes...)
	s.mu.RUnlock()

	seen := map[string]bool{}
	refs := []string{}
	for _, note := range notes {
		if !includeDeleted && noteIsDeleted(note) {
			continue
		}
		for _, match := range noteAssetReferencePattern.FindAllString(note.Content, -1) {
			trimmed := strings.TrimSpace(strings.Trim(match, `"'()[]<>`))
			if trimmed == "" || seen[trimmed] {
				continue
			}
			seen[trimmed] = true
			refs = append(refs, trimmed)
		}
	}
	sort.Strings(refs)
	return refs
}

func (s *Service) GetContent(path string) (string, error) {
	id := idFromPath(path)
	if id == "" {
		return "", errors.New("dstu path does not contain a resource id")
	}

	s.mu.RLock()
	note, ok := s.findNoteLocked(id)
	s.mu.RUnlock()
	if ok {
		if noteIsDeleted(note) {
			return "", fmt.Errorf("resource not found: %s", path)
		}
		return note.Content, nil
	}

	if s.vfs == nil {
		return "", fmt.Errorf("resource not found: %s", path)
	}
	result, err := s.vfs.GetFileContent(id)
	if err != nil {
		return "", err
	}
	if !result.Found || result.Content == nil {
		return "", fmt.Errorf("resource not found: %s", path)
	}
	return *result.Content, nil
}

func (s *Service) CanvasReadContent(noteID string, section *string) (string, error) {
	content, err := s.noteContentByID(noteID)
	if err != nil {
		return "", err
	}
	if section == nil || strings.TrimSpace(*section) == "" {
		return content, nil
	}
	extracted, ok := extractMarkdownSection(content, *section)
	if !ok {
		return "", fmt.Errorf("section not found: %s", *section)
	}
	return extracted, nil
}

func (s *Service) CanvasAppendContent(noteID string, content string, section *string) error {
	current, err := s.noteContentByID(noteID)
	if err != nil {
		return err
	}
	var next string
	if section != nil && strings.TrimSpace(*section) != "" {
		updated, ok := appendToMarkdownSection(current, *section, content)
		if !ok {
			return fmt.Errorf("section not found: %s", *section)
		}
		next = updated
	} else if strings.TrimSpace(current) == "" {
		next = content
	} else {
		next = strings.TrimRight(current, "\r\n") + "\n\n" + content
	}
	_, err = s.Update("/"+idFromPath(noteID), next, "note")
	return err
}

func (s *Service) CanvasReplaceContent(noteID string, search string, replace string, isRegex bool) (int, error) {
	current, err := s.noteContentByID(noteID)
	if err != nil {
		return 0, err
	}
	if search == "" {
		return 0, nil
	}
	next := current
	count := 0
	if isRegex {
		re, err := regexp.Compile(search)
		if err != nil {
			return 0, fmt.Errorf("invalid regex: %w", err)
		}
		count = len(re.FindAllStringIndex(current, -1))
		if count > 0 {
			next = re.ReplaceAllString(current, replace)
		}
	} else {
		count = strings.Count(current, search)
		if count > 0 {
			next = strings.ReplaceAll(current, search, replace)
		}
	}
	if count == 0 {
		return 0, nil
	}
	_, err = s.Update("/"+idFromPath(noteID), next, "note")
	if err != nil {
		return 0, err
	}
	return count, nil
}

func (s *Service) CanvasSetContent(noteID string, content string) error {
	_, err := s.Update("/"+idFromPath(noteID), content, "note")
	return err
}

func (s *Service) SetMetadata(path string, metadata map[string]any) (bool, error) {
	id := idFromPath(path)
	if id == "" {
		return false, errors.New("dstu path does not contain a resource id")
	}

	s.mu.Lock()
	index, ok := s.findNoteIndexLocked(id)
	if !ok {
		s.mu.Unlock()
		if s.vfs == nil {
			return false, fmt.Errorf("resource not found: %s", path)
		}
		updated, err := s.vfs.UpdateFileMetadata(id, metadata)
		if err != nil {
			return false, err
		}
		if !updated {
			return false, fmt.Errorf("resource not found: %s", path)
		}
		return true, nil
	}
	current := normalizeMetadata(s.state.Notes[index].Metadata)
	for key, value := range metadata {
		current[key] = value
	}
	if title, ok := metadata["title"].(string); ok && strings.TrimSpace(title) != "" {
		s.state.Notes[index].Name = strings.TrimSpace(title)
	}
	s.state.Notes[index].Metadata = current
	s.state.Notes[index].UpdatedAt = nowMillis()
	err := s.flushLocked()
	s.mu.Unlock()
	return true, err
}

func (s *Service) SetFavorite(path string, favorite bool) (bool, error) {
	return s.SetMetadata(path, map[string]any{"isFavorite": favorite})
}

func (s *Service) ImportMarkdown(request ImportMarkdownRequest) (Node, error) {
	content, title, metadata, err := readMarkdownImport(request.FilePath, request.TitleHint, request.FolderID)
	if err != nil {
		return Node{}, err
	}
	return s.Create("/", CreateOptions{
		Type:     "note",
		Name:     title,
		Content:  &content,
		Metadata: metadata,
	})
}

func (s *Service) ImportMarkdownBatch(request ImportMarkdownBatchRequest) (ImportMarkdownBatchResponse, error) {
	response := ImportMarkdownBatchResponse{
		Imported: []Node{},
		Failed:   []ImportMarkdownBatchFailure{},
	}
	for _, item := range request.Items {
		node, err := s.ImportMarkdown(ImportMarkdownRequest{
			FilePath:  item.FilePath,
			TitleHint: item.TitleHint,
			FolderID:  request.FolderID,
		})
		if err != nil {
			response.Failed = append(response.Failed, ImportMarkdownBatchFailure{
				FilePath: item.FilePath,
				Message:  err.Error(),
			})
			continue
		}
		response.Imported = append(response.Imported, node)
	}
	return response, nil
}

func (s *Service) AddTextbooks(request AddTextbooksRequest) ([]TextbookRecord, error) {
	if len(request.Sources) == 0 {
		return []TextbookRecord{}, nil
	}
	importID := "textbook_import_" + randomToken(10)
	if s.vfs == nil {
		err := errors.New("lean Go DSTU has no VFS service for textbook imports")
		s.emitTextbookImportProgress(importID, request.Sources[0], 0, len(request.Sources), "error", 100, nil, err)
		return nil, err
	}
	records := make([]TextbookRecord, 0, len(request.Sources))
	for index, source := range request.Sources {
		s.emitTextbookImportProgress(importID, source, index, len(request.Sources), "hashing", 5, nil, nil)
		s.emitTextbookImportProgress(importID, source, index, len(request.Sources), "copying", 30, nil, nil)
		record, err := s.addTextbookFile(source, request.FolderID)
		if err != nil {
			s.emitTextbookImportProgress(importID, source, index, len(request.Sources), "error", 100, nil, err)
			return nil, err
		}
		s.emitTextbookImportProgress(importID, source, index, len(request.Sources), "saving", 85, &record, nil)
		records = append(records, record)
		s.emitTextbookImportProgress(importID, source, index, len(request.Sources), "done", 100, &record, nil)
	}
	return records, nil
}

func (s *Service) emitTextbookImportProgress(importID string, source string, index int, total int, stage string, progress int, record *TextbookRecord, err error) {
	if total <= 0 {
		total = 1
	}
	if index < 0 {
		index = 0
	}
	fileName := textbookProgressFileName(source)
	payload := textbookImportProgressPayload{
		FileName: fileName,
		Stage:    normalizeTextbookImportStage(stage),
		Progress: clampProgress(progress),
		Source:   strings.TrimSpace(source),
		ImportID: strings.TrimSpace(importID),
		Index:    index,
		Total:    total,
	}
	if record != nil {
		if strings.TrimSpace(record.FileName) != "" {
			payload.FileName = strings.TrimSpace(record.FileName)
		}
		if strings.TrimSpace(record.FilePath) != "" {
			payload.Source = strings.TrimSpace(record.FilePath)
		}
		payload.TextbookID = strings.TrimSpace(record.ID)
		payload.ResourceID = strings.TrimSpace(record.ResourceID)
	}
	if payload.FileName == "" {
		payload.FileName = "textbook"
	}
	if err != nil {
		message := err.Error()
		payload.Error = &message
		payload.Stage = "error"
	}
	s.emitEvent("textbook-import-progress", payload)
}

func textbookProgressFileName(source string) string {
	trimmed := strings.TrimSpace(source)
	if trimmed == "" {
		return "textbook"
	}
	cleaned := filepath.Clean(trimmed)
	name := filepath.Base(cleaned)
	if strings.TrimSpace(name) == "." || strings.TrimSpace(name) == string(filepath.Separator) || strings.TrimSpace(name) == "" {
		return trimmed
	}
	return name
}

func normalizeTextbookImportStage(stage string) string {
	switch strings.ToLower(strings.TrimSpace(stage)) {
	case "hashing", "copying", "rendering", "saving", "done", "error":
		return strings.ToLower(strings.TrimSpace(stage))
	default:
		return "copying"
	}
}

func clampProgress(progress int) int {
	if progress < 0 {
		return 0
	}
	if progress > 100 {
		return 100
	}
	return progress
}

func (s *Service) CreateFolder(title string, parentID *string, icon *string, color *string) (VfsFolder, error) {
	title = strings.TrimSpace(title)
	if title == "" {
		return VfsFolder{}, errors.New("folder title cannot be empty")
	}
	if len([]rune(title)) > 100 {
		return VfsFolder{}, errors.New("folder title is too long")
	}
	parentID = normalizeFolderID(parentID)
	now := nowMillis()

	s.mu.Lock()
	defer s.mu.Unlock()

	if len(s.state.Folders) >= 500 {
		return VfsFolder{}, errors.New("folder count limit exceeded")
	}
	if parentID != nil {
		if !s.folderExistsLocked(*parentID) {
			return VfsFolder{}, fmt.Errorf("parent folder not found: %s", *parentID)
		}
		if s.folderDepthLocked(*parentID) >= 9 {
			return VfsFolder{}, errors.New("folder depth limit exceeded")
		}
	}

	folder := VfsFolder{
		ID:         "fld_" + randomToken(10),
		ParentID:   cloneStringPtr(parentID),
		Title:      title,
		Icon:       normalizeOptionalStringPtr(icon),
		Color:      normalizeOptionalStringPtr(color),
		IsExpanded: true,
		SortOrder:  s.nextFolderSortOrderLocked(parentID),
		CreatedAt:  now,
		UpdatedAt:  now,
	}
	s.state.Folders = append(s.state.Folders, folder)
	if err := s.flushLocked(); err != nil {
		return VfsFolder{}, err
	}
	return folder, nil
}

func (s *Service) GetFolder(folderID string) (*VfsFolder, error) {
	folderID = strings.TrimSpace(folderID)
	if folderID == "" {
		return nil, errors.New("folderId is required")
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	folder, ok := s.findFolderLocked(folderID)
	if !ok {
		return nil, nil
	}
	return &folder, nil
}

func (s *Service) RenameFolder(folderID string, title string) error {
	folderID = strings.TrimSpace(folderID)
	title = strings.TrimSpace(title)
	if folderID == "" {
		return errors.New("folderId is required")
	}
	if title == "" {
		return errors.New("folder title cannot be empty")
	}
	if len([]rune(title)) > 100 {
		return errors.New("folder title is too long")
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	index, ok := s.findFolderIndexLocked(folderID)
	if !ok {
		return fmt.Errorf("folder not found: %s", folderID)
	}
	s.state.Folders[index].Title = title
	s.state.Folders[index].UpdatedAt = nowMillis()
	if err := s.syncFolderItemsForFoldersLocked(s.folderIDSetWithDescendantsLocked(folderID, true)); err != nil {
		return err
	}
	return s.flushLocked()
}

func (s *Service) DeleteFolder(folderID string) error {
	folderID = strings.TrimSpace(folderID)
	if folderID == "" {
		return errors.New("folderId is required")
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	if !s.folderExistsLocked(folderID) {
		return fmt.Errorf("folder not found: %s", folderID)
	}
	removed := s.folderIDSetWithDescendantsLocked(folderID, true)
	remaining := make([]VfsFolder, 0, len(s.state.Folders))
	for _, folder := range s.state.Folders {
		if _, drop := removed[folder.ID]; drop {
			continue
		}
		remaining = append(remaining, folder)
	}
	s.state.Folders = remaining

	now := nowMillis()
	for index := range s.state.FolderItems {
		item := &s.state.FolderItems[index]
		if item.FolderID == nil {
			continue
		}
		if _, affected := removed[*item.FolderID]; !affected {
			continue
		}
		item.FolderID = nil
		item.UpdatedAt = now
		cachedPath := "/" + item.ItemID
		item.CachedPath = &cachedPath
		if err := s.syncFolderMetadataForItemLocked(*item); err != nil {
			return err
		}
	}
	return s.flushLocked()
}

func (s *Service) MoveFolder(folderID string, newParentID *string) error {
	folderID = strings.TrimSpace(folderID)
	if folderID == "" {
		return errors.New("folderId is required")
	}
	newParentID = normalizeFolderID(newParentID)
	if newParentID != nil && *newParentID == folderID {
		return errors.New("cannot move a folder into itself")
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	index, ok := s.findFolderIndexLocked(folderID)
	if !ok {
		return fmt.Errorf("folder not found: %s", folderID)
	}
	if newParentID != nil {
		if !s.folderExistsLocked(*newParentID) {
			return fmt.Errorf("parent folder not found: %s", *newParentID)
		}
		if _, cycle := s.folderIDSetWithDescendantsLocked(folderID, true)[*newParentID]; cycle {
			return errors.New("cannot move a folder into its child")
		}
		if s.folderDepthLocked(*newParentID) >= 9 {
			return errors.New("folder depth limit exceeded")
		}
	}
	s.state.Folders[index].ParentID = cloneStringPtr(newParentID)
	s.state.Folders[index].SortOrder = s.nextFolderSortOrderLocked(newParentID)
	s.state.Folders[index].UpdatedAt = nowMillis()
	if err := s.syncFolderItemsForFoldersLocked(s.folderIDSetWithDescendantsLocked(folderID, true)); err != nil {
		return err
	}
	return s.flushLocked()
}

func (s *Service) SetFolderExpanded(folderID string, isExpanded bool) error {
	folderID = strings.TrimSpace(folderID)
	if folderID == "" {
		return errors.New("folderId is required")
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	index, ok := s.findFolderIndexLocked(folderID)
	if !ok {
		return fmt.Errorf("folder not found: %s", folderID)
	}
	s.state.Folders[index].IsExpanded = isExpanded
	s.state.Folders[index].UpdatedAt = nowMillis()
	return s.flushLocked()
}

func (s *Service) AddFolderItem(folderID *string, itemType string, itemID string) (VfsFolderItem, error) {
	return s.upsertFolderItem(folderID, itemType, itemID)
}

func (s *Service) RemoveFolderItem(itemType string, itemID string) error {
	itemType = normalizeItemType(itemType, itemID)
	itemID = idFromPath(itemID)
	if itemID == "" {
		return errors.New("itemId is required")
	}
	if !isValidFolderItemType(itemType) {
		return fmt.Errorf("invalid itemType: %s", itemType)
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	remaining := make([]VfsFolderItem, 0, len(s.state.FolderItems))
	removed := false
	for _, item := range s.state.FolderItems {
		if item.ItemType == itemType && item.ItemID == itemID {
			removed = true
			rooted := item
			rooted.FolderID = nil
			rooted.UpdatedAt = nowMillis()
			rootPath := "/" + rooted.ItemID
			rooted.CachedPath = &rootPath
			if err := s.syncFolderMetadataForItemLocked(rooted); err != nil {
				return err
			}
			continue
		}
		remaining = append(remaining, item)
	}
	if !removed {
		return nil
	}
	s.state.FolderItems = remaining
	return s.flushLocked()
}

func (s *Service) removeFolderItemLocked(itemType string, itemID string) {
	remaining := make([]VfsFolderItem, 0, len(s.state.FolderItems))
	for _, item := range s.state.FolderItems {
		if item.ItemType == itemType && item.ItemID == itemID {
			continue
		}
		remaining = append(remaining, item)
	}
	s.state.FolderItems = remaining
}

func (s *Service) MoveFolderItem(itemType string, itemID string, newFolderID *string) error {
	_, err := s.upsertFolderItem(newFolderID, itemType, itemID)
	return err
}

func (s *Service) ListFolders() ([]VfsFolder, error) {
	s.mu.RLock()
	folders := append([]VfsFolder(nil), s.state.Folders...)
	s.mu.RUnlock()
	sortFolders(folders)
	return folders, nil
}

func (s *Service) GetFolderTree() ([]FolderTreeNode, error) {
	s.mu.RLock()
	folders := append([]VfsFolder(nil), s.state.Folders...)
	items := append([]VfsFolderItem(nil), s.state.FolderItems...)
	s.mu.RUnlock()
	sortFolders(folders)
	sortFolderItems(items)
	return buildFolderTree(nil, folders, items), nil
}

func (s *Service) GetFolderItems(folderID *string) ([]VfsFolderItem, error) {
	folderID = normalizeFolderID(folderID)
	s.mu.RLock()
	defer s.mu.RUnlock()
	if folderID != nil && !s.folderExistsLocked(*folderID) {
		return nil, fmt.Errorf("folder not found: %s", *folderID)
	}
	items := make([]VfsFolderItem, 0)
	for _, item := range s.state.FolderItems {
		if sameOptionalString(item.FolderID, folderID) {
			items = append(items, item)
		}
	}
	sortFolderItems(items)
	return items, nil
}

func (s *Service) ReorderFolders(folderIDs []string) error {
	order := map[string]int{}
	for index, folderID := range folderIDs {
		if trimmed := strings.TrimSpace(folderID); trimmed != "" {
			order[trimmed] = index
		}
	}
	if len(order) == 0 {
		return nil
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	now := nowMillis()
	for index := range s.state.Folders {
		if sortOrder, ok := order[s.state.Folders[index].ID]; ok {
			s.state.Folders[index].SortOrder = sortOrder
			s.state.Folders[index].UpdatedAt = now
		}
	}
	return s.flushLocked()
}

func (s *Service) ReorderFolderItems(folderID *string, itemIDs []string) error {
	folderID = normalizeFolderID(folderID)
	order := map[string]int{}
	for index, itemID := range itemIDs {
		if trimmed := idFromPath(itemID); trimmed != "" {
			order[trimmed] = index
		}
	}
	if len(order) == 0 {
		return nil
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	now := nowMillis()
	for index := range s.state.FolderItems {
		item := &s.state.FolderItems[index]
		if !sameOptionalString(item.FolderID, folderID) {
			continue
		}
		if sortOrder, ok := order[item.ItemID]; ok {
			item.SortOrder = sortOrder
			item.UpdatedAt = now
		} else if sortOrder, ok := order[item.ID]; ok {
			item.SortOrder = sortOrder
			item.UpdatedAt = now
		}
	}
	return s.flushLocked()
}

func (s *Service) GetFolderBreadcrumbs(folderID string) ([]BreadcrumbItem, error) {
	folderID = strings.TrimSpace(folderID)
	if folderID == "" {
		return []BreadcrumbItem{}, nil
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	chain, ok := s.folderChainLocked(folderID)
	if !ok {
		return nil, fmt.Errorf("folder not found: %s", folderID)
	}
	breadcrumbs := make([]BreadcrumbItem, 0, len(chain))
	for index := len(chain) - 1; index >= 0; index-- {
		folder := chain[index]
		breadcrumbs = append(breadcrumbs, BreadcrumbItem{ID: folder.ID, Name: folder.Title})
	}
	return breadcrumbs, nil
}

func (s *Service) ParsePath(path string) (ParsedPath, error) {
	fullPath := normalizeDstuPath(path)
	if fullPath == "/" {
		return ParsedPath{FullPath: "/", IsRoot: true, IsVirtual: false}, nil
	}
	trimmed := strings.Trim(fullPath, "/")
	parts := strings.Split(trimmed, "/")
	last := parts[len(parts)-1]
	isVirtual := strings.HasPrefix(trimmed, "@")
	virtualType := (*string)(nil)
	if isVirtual {
		first := strings.TrimPrefix(parts[0], "@")
		if first != "" {
			virtualType = &first
		}
	}
	resourceID := (*string)(nil)
	idAlias := (*string)(nil)
	resourceType := (*string)(nil)
	if inferred := inferResourceType(last); inferred != "" {
		resourceID = &last
		idAlias = &last
		resourceType = &inferred
	}
	folderPath := (*string)(nil)
	if len(parts) > 1 {
		value := "/" + strings.Join(parts[:len(parts)-1], "/")
		folderPath = &value
	} else if resourceID != nil {
		value := "/"
		folderPath = &value
	}
	return ParsedPath{
		FullPath:     fullPath,
		FolderPath:   folderPath,
		ResourceID:   resourceID,
		ID:           idAlias,
		ResourceType: resourceType,
		IsRoot:       false,
		IsVirtual:    isVirtual,
		VirtualType:  virtualType,
	}, nil
}

func (s *Service) BuildPath(folderID *string, resourceID string) (string, error) {
	resourceID = idFromPath(resourceID)
	if resourceID == "" {
		return "", errors.New("resourceId is required")
	}
	folderID = normalizeFolderID(folderID)
	s.mu.RLock()
	defer s.mu.RUnlock()
	if folderID != nil && !s.folderExistsLocked(*folderID) {
		return "", fmt.Errorf("folder not found: %s", *folderID)
	}
	return s.buildResourcePathLocked(folderID, resourceID), nil
}

func (s *Service) MoveToFolder(resourceID string, targetFolderID *string) (ResourceLocation, error) {
	resourceID = idFromPath(resourceID)
	if resourceID == "" {
		return ResourceLocation{}, errors.New("resourceId is required")
	}
	itemType := inferResourceType(resourceID)
	if itemType == "" {
		itemType = "file"
	}
	if _, err := s.upsertFolderItem(targetFolderID, itemType, resourceID); err != nil {
		return ResourceLocation{}, err
	}
	return s.GetResourceLocation(resourceID)
}

func (s *Service) BatchMove(request BatchMoveRequest) (BatchMoveResult, error) {
	result := BatchMoveResult{
		Successes:   []ResourceLocation{},
		FailedItems: []FailedMoveItem{},
		TotalCount:  len(request.ItemIDs),
	}
	for _, itemID := range request.ItemIDs {
		location, err := s.MoveToFolder(itemID, request.TargetFolderID)
		if err != nil {
			result.FailedItems = append(result.FailedItems, FailedMoveItem{
				ItemID: idFromPath(itemID),
				Error:  err.Error(),
			})
			continue
		}
		result.Successes = append(result.Successes, location)
	}
	return result, nil
}

func (s *Service) RefreshPathCache(resourceID *string) (int, error) {
	targetID := ""
	if resourceID != nil {
		targetID = idFromPath(*resourceID)
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	updated := 0
	for index := range s.state.FolderItems {
		if targetID != "" && s.state.FolderItems[index].ItemID != targetID {
			continue
		}
		path := s.buildResourcePathLocked(s.state.FolderItems[index].FolderID, s.state.FolderItems[index].ItemID)
		s.state.FolderItems[index].CachedPath = &path
		s.state.FolderItems[index].UpdatedAt = nowMillis()
		if err := s.syncFolderMetadataForItemLocked(s.state.FolderItems[index]); err != nil {
			return updated, err
		}
		updated++
	}
	if updated == 0 {
		return 0, nil
	}
	return updated, s.flushLocked()
}

func (s *Service) GetPathByID(resourceID string) (string, error) {
	resourceID = strings.TrimSpace(resourceID)
	if resourceID == "" {
		return "", errors.New("resourceId is required")
	}
	s.mu.RLock()
	if s.folderExistsLocked(resourceID) {
		path := s.folderPathLocked(resourceID)
		s.mu.RUnlock()
		return path, nil
	}
	s.mu.RUnlock()
	location, err := s.GetResourceLocation(resourceID)
	if err != nil {
		return "", err
	}
	return location.FullPath, nil
}

func (s *Service) GetResourceByPath(path string) (*Node, error) {
	return s.Get(path)
}

func (s *Service) GetResourceLocation(resourceID string) (ResourceLocation, error) {
	id := idFromPath(resourceID)
	if id == "" {
		id = strings.TrimSpace(resourceID)
	}
	if id == "" {
		return ResourceLocation{}, errors.New("resourceId is required")
	}
	node, err := s.Get("/" + id)
	if err != nil {
		return ResourceLocation{}, err
	}
	if node == nil {
		s.mu.RLock()
		item, ok := s.findFolderItemByItemIDLocked(id)
		s.mu.RUnlock()
		if ok {
			return s.folderItemResourceLocation(item, nil), nil
		}
		return ResourceLocation{}, fmt.Errorf("resource not found: %s", resourceID)
	}
	return s.nodeResourceLocation(*node), nil
}

func (s *Service) GetFolderAllResources(folderID string, includeSubfolders bool, includeContent bool) (FolderResourcesResult, error) {
	folderID = strings.TrimSpace(folderID)
	if folderID == "" {
		return FolderResourcesResult{}, errors.New("folderId is required")
	}
	nodes, err := s.List("/", nil)
	if err != nil {
		return FolderResourcesResult{}, err
	}
	s.mu.RLock()
	folders := append([]VfsFolder(nil), s.state.Folders...)
	folderItems := append([]VfsFolderItem(nil), s.state.FolderItems...)
	s.mu.RUnlock()

	nodeByID := make(map[string]Node, len(nodes))
	for _, node := range nodes {
		nodeByID[node.ID] = node
		nodeByID[folderItemKey(node.Type, node.ID)] = node
	}
	folderIDs := map[string]struct{}{folderID: {}}
	if includeSubfolders {
		folderIDs = folderIDSetWithDescendants(folders, folderID, true)
	}
	resources := make([]FolderResourceInfo, 0, len(nodes))
	seen := map[string]struct{}{}
	for _, item := range folderItems {
		if item.FolderID == nil {
			continue
		}
		if _, ok := folderIDs[*item.FolderID]; !ok {
			continue
		}
		node, nodeOK := nodeByID[folderItemKey(item.ItemType, item.ItemID)]
		if !nodeOK {
			node, nodeOK = nodeByID[item.ItemID]
		}
		info := folderResourceInfoFromItem(item, folders)
		if nodeOK {
			info = s.folderResourceInfoFromNode(node)
			if includeContent {
				content, err := s.GetContent(node.Path)
				if err == nil {
					info.Content = &content
				}
			}
		}
		seen[folderItemKey(item.ItemType, item.ItemID)] = struct{}{}
		seen[item.ItemID] = struct{}{}
		resources = append(resources, info)
	}
	for _, node := range nodes {
		if _, ok := seen[folderItemKey(node.Type, node.ID)]; ok {
			continue
		}
		if _, ok := seen[node.ID]; ok {
			continue
		}
		if !nodeInFolderScope(node, folderID, includeSubfolders) {
			continue
		}
		info := s.folderResourceInfoFromNode(node)
		if includeContent {
			content, err := s.GetContent(node.Path)
			if err == nil {
				info.Content = &content
			}
		}
		resources = append(resources, info)
	}
	s.mu.RLock()
	folderPath := s.folderPathLocked(folderID)
	folderTitle := s.folderTitleLocked(folderID)
	s.mu.RUnlock()
	return FolderResourcesResult{
		FolderID:    folderID,
		FolderTitle: folderTitle,
		Path:        folderPath,
		TotalCount:  len(resources),
		Resources:   resources,
	}, nil
}

func (s *Service) addTextbookFile(source string, folderID *string) (TextbookRecord, error) {
	sourcePath := strings.TrimSpace(source)
	if sourcePath == "" {
		return TextbookRecord{}, errors.New("textbook source path cannot be empty")
	}
	if strings.Contains(sourcePath, "://") {
		return TextbookRecord{}, fmt.Errorf("textbook import currently supports local file paths only: %s", sourcePath)
	}
	cleaned := filepath.Clean(sourcePath)
	bytes, err := os.ReadFile(cleaned)
	if err != nil {
		return TextbookRecord{}, fmt.Errorf("read textbook file %q: %w", sourcePath, err)
	}
	info, err := os.Stat(cleaned)
	if err != nil {
		return TextbookRecord{}, fmt.Errorf("stat textbook file %q: %w", sourcePath, err)
	}
	if info.IsDir() {
		return TextbookRecord{}, fmt.Errorf("textbook source is a directory: %s", sourcePath)
	}
	absolutePath, err := filepath.Abs(cleaned)
	if err != nil {
		absolutePath = cleaned
	}
	name := filepath.Base(cleaned)
	mimeType := mimeTypeForName(name)
	fileType := "document"
	metadata := map[string]any{
		"name":         name,
		"title":        name,
		"mimeType":     mimeType,
		"sourceDb":     "dstu",
		"sourceType":   "textbook",
		"resourceType": "textbook",
		"type":         "textbook",
		"previewType":  previewTypeForDstuType("textbook", mimeType, name, fileType),
		"sourcePath":   absolutePath,
		"importPath":   absolutePath,
	}
	if folderID != nil && strings.TrimSpace(*folderID) != "" {
		metadata["folderId"] = strings.TrimSpace(*folderID)
	}
	result, err := s.vfs.UploadFile(vfs.UploadFileInput{
		Name:          name,
		MimeType:      mimeType,
		Base64Content: base64.StdEncoding.EncodeToString(bytes),
		FileType:      &fileType,
		FolderID:      folderID,
		Metadata:      metadata,
	})
	if err != nil {
		return TextbookRecord{}, err
	}
	return textbookRecordFromVfsFile(result.File), nil
}

func (s *Service) findNoteLocked(id string) (NoteRecord, bool) {
	for _, note := range s.state.Notes {
		if note.ID == id {
			return note, true
		}
	}
	return NoteRecord{}, false
}

func (s *Service) findNoteIndexLocked(id string) (int, bool) {
	for index, note := range s.state.Notes {
		if note.ID == id {
			return index, true
		}
	}
	return -1, false
}

func (s *Service) upsertFolderItem(folderID *string, itemType string, itemID string) (VfsFolderItem, error) {
	folderID = normalizeFolderID(folderID)
	itemID = idFromPath(itemID)
	if itemID == "" {
		return VfsFolderItem{}, errors.New("itemId is required")
	}
	itemType = normalizeItemType(itemType, itemID)
	if !isValidFolderItemType(itemType) {
		return VfsFolderItem{}, fmt.Errorf("invalid itemType: %s", itemType)
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	if folderID != nil && !s.folderExistsLocked(*folderID) {
		return VfsFolderItem{}, fmt.Errorf("folder not found: %s", *folderID)
	}
	now := nowMillis()
	cachedPath := s.buildResourcePathLocked(folderID, itemID)
	if index, ok := s.findFolderItemIndexLocked(itemType, itemID); ok {
		s.state.FolderItems[index].FolderID = cloneStringPtr(folderID)
		s.state.FolderItems[index].UpdatedAt = now
		s.state.FolderItems[index].CachedPath = &cachedPath
		if err := s.syncFolderMetadataForItemLocked(s.state.FolderItems[index]); err != nil {
			return VfsFolderItem{}, err
		}
		if err := s.flushLocked(); err != nil {
			return VfsFolderItem{}, err
		}
		return s.state.FolderItems[index], nil
	}

	item := VfsFolderItem{
		ID:         "fi_" + randomToken(10),
		FolderID:   cloneStringPtr(folderID),
		ItemType:   itemType,
		ItemID:     itemID,
		SortOrder:  s.nextItemSortOrderLocked(folderID),
		CreatedAt:  now,
		UpdatedAt:  now,
		CachedPath: &cachedPath,
	}
	s.state.FolderItems = append(s.state.FolderItems, item)
	index := len(s.state.FolderItems) - 1
	if err := s.syncFolderMetadataForItemLocked(s.state.FolderItems[index]); err != nil {
		s.state.FolderItems = s.state.FolderItems[:index]
		return VfsFolderItem{}, err
	}
	if err := s.flushLocked(); err != nil {
		return VfsFolderItem{}, err
	}
	return s.state.FolderItems[index], nil
}

func (s *Service) findFolderLocked(folderID string) (VfsFolder, bool) {
	for _, folder := range s.state.Folders {
		if folder.ID == folderID {
			return folder, true
		}
	}
	return VfsFolder{}, false
}

func (s *Service) findFolderIndexLocked(folderID string) (int, bool) {
	for index, folder := range s.state.Folders {
		if folder.ID == folderID {
			return index, true
		}
	}
	return -1, false
}

func (s *Service) folderExistsLocked(folderID string) bool {
	_, ok := s.findFolderLocked(folderID)
	return ok
}

func (s *Service) findFolderItemIndexLocked(itemType string, itemID string) (int, bool) {
	for index, item := range s.state.FolderItems {
		if item.ItemType == itemType && item.ItemID == itemID {
			return index, true
		}
	}
	return -1, false
}

func (s *Service) findFolderItemByItemIDLocked(itemID string) (VfsFolderItem, bool) {
	for _, item := range s.state.FolderItems {
		if item.ItemID == itemID {
			return item, true
		}
	}
	return VfsFolderItem{}, false
}

func (s *Service) nextFolderSortOrderLocked(parentID *string) int {
	next := 0
	for _, folder := range s.state.Folders {
		if sameOptionalString(folder.ParentID, parentID) && folder.SortOrder >= next {
			next = folder.SortOrder + 1
		}
	}
	return next
}

func (s *Service) nextItemSortOrderLocked(folderID *string) int {
	next := 0
	for _, item := range s.state.FolderItems {
		if sameOptionalString(item.FolderID, folderID) && item.SortOrder >= next {
			next = item.SortOrder + 1
		}
	}
	return next
}

func (s *Service) folderDepthLocked(folderID string) int {
	chain, ok := s.folderChainLocked(folderID)
	if !ok || len(chain) == 0 {
		return 0
	}
	return len(chain) - 1
}

func (s *Service) folderChainLocked(folderID string) ([]VfsFolder, bool) {
	currentID := strings.TrimSpace(folderID)
	if currentID == "" {
		return []VfsFolder{}, false
	}
	chain := []VfsFolder{}
	seen := map[string]struct{}{}
	for currentID != "" {
		if _, ok := seen[currentID]; ok {
			return chain, false
		}
		seen[currentID] = struct{}{}
		folder, ok := s.findFolderLocked(currentID)
		if !ok {
			return chain, false
		}
		chain = append(chain, folder)
		if folder.ParentID == nil || strings.TrimSpace(*folder.ParentID) == "" {
			break
		}
		currentID = strings.TrimSpace(*folder.ParentID)
	}
	return chain, true
}

func (s *Service) folderIDSetWithDescendantsLocked(folderID string, includeSelf bool) map[string]struct{} {
	return folderIDSetWithDescendants(s.state.Folders, folderID, includeSelf)
}

func (s *Service) syncFolderItemsForFoldersLocked(folderIDs map[string]struct{}) error {
	if len(folderIDs) == 0 {
		return nil
	}
	for index := range s.state.FolderItems {
		item := &s.state.FolderItems[index]
		if item.FolderID == nil {
			continue
		}
		if _, ok := folderIDs[*item.FolderID]; !ok {
			continue
		}
		path := s.buildResourcePathLocked(item.FolderID, item.ItemID)
		item.CachedPath = &path
		item.UpdatedAt = nowMillis()
		if err := s.syncFolderMetadataForItemLocked(*item); err != nil {
			return err
		}
	}
	return nil
}

func (s *Service) syncFolderMetadataForItemLocked(item VfsFolderItem) error {
	metadata := s.folderMetadataForItemLocked(item)
	if item.ItemType == "note" {
		index, ok := s.findNoteIndexLocked(item.ItemID)
		if !ok {
			return nil
		}
		current := normalizeMetadata(s.state.Notes[index].Metadata)
		for key, value := range metadata {
			current[key] = value
		}
		s.state.Notes[index].Metadata = current
		s.state.Notes[index].UpdatedAt = nowMillis()
		return s.syncNoteResourceLocked(index)
	}
	if s.vfs == nil || !isVfsFolderMetadataType(item.ItemType) {
		return nil
	}
	_, err := s.vfs.UpdateFileMetadata(item.ItemID, metadata)
	return err
}

func (s *Service) folderMetadataForItemLocked(item VfsFolderItem) map[string]any {
	fullPath := s.buildResourcePathLocked(item.FolderID, item.ItemID)
	folderPath := "/"
	folderPathIDs := []string{}
	folderID := ""
	if item.FolderID != nil && strings.TrimSpace(*item.FolderID) != "" {
		folderID = strings.TrimSpace(*item.FolderID)
		folderPath = s.folderPathLocked(folderID)
		folderPathIDs = s.folderPathIDsLocked(folderID)
	}
	return map[string]any{
		"folderId":      folderID,
		"folderPath":    folderPath,
		"folderPathIds": folderPathIDs,
		"parentPath":    folderPath,
		"path":          fullPath,
		"cachedPath":    fullPath,
	}
}

func (s *Service) folderPathIDsLocked(folderID string) []string {
	chain, ok := s.folderChainLocked(folderID)
	if !ok {
		return []string{strings.TrimSpace(folderID)}
	}
	ids := make([]string, 0, len(chain))
	for index := len(chain) - 1; index >= 0; index-- {
		ids = append(ids, chain[index].ID)
	}
	return ids
}

func (s *Service) folderPathLocked(folderID string) string {
	chain, ok := s.folderChainLocked(folderID)
	if !ok || len(chain) == 0 {
		return folderPathForID(folderID)
	}
	parts := make([]string, 0, len(chain))
	for index := len(chain) - 1; index >= 0; index-- {
		title := strings.Trim(strings.TrimSpace(chain[index].Title), "/\\")
		if title == "" {
			title = chain[index].ID
		}
		parts = append(parts, title)
	}
	return "/" + strings.Join(parts, "/")
}

func (s *Service) folderTitleLocked(folderID string) string {
	if folder, ok := s.findFolderLocked(folderID); ok {
		return folder.Title
	}
	return folderTitleForID(folderID)
}

func (s *Service) buildResourcePathLocked(folderID *string, resourceID string) string {
	resourceID = idFromPath(resourceID)
	if resourceID == "" {
		return "/"
	}
	if folderID == nil || strings.TrimSpace(*folderID) == "" {
		return "/" + resourceID
	}
	return strings.TrimRight(s.folderPathLocked(*folderID), "/") + "/" + resourceID
}

func (s *Service) applyStoredFolderItemToNode(node *Node) {
	s.mu.RLock()
	folders := append([]VfsFolder(nil), s.state.Folders...)
	items := append([]VfsFolderItem(nil), s.state.FolderItems...)
	s.mu.RUnlock()
	applyFolderItemToNode(node, folderItemLookup(items), folderPathIDLookup(folders), folderTitlePathLookup(folders))
}

func (s *Service) nodeResourceLocation(node Node) ResourceLocation {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.nodeResourceLocationLocked(node)
}

func (s *Service) nodeResourceLocationLocked(node Node) ResourceLocation {
	folderID := nodeFolderID(node)
	folderPath := "/"
	if folderID != nil {
		folderPath = s.folderPathLocked(*folderID)
		if value := metadataString(node.Metadata, "folderPath", ""); value != "" && !s.folderExistsLocked(*folderID) {
			folderPath = value
		}
	}
	hash := (*string)(nil)
	if strings.TrimSpace(node.ResourceHash) != "" {
		value := strings.TrimSpace(node.ResourceHash)
		hash = &value
	}
	fullPath := strings.TrimSpace(node.Path)
	if folderID != nil {
		fullPath = strings.TrimRight(folderPath, "/") + "/" + node.ID
	} else if fullPath == "" || idFromPath(fullPath) != node.ID {
		fullPath = strings.TrimRight(folderPath, "/") + "/" + node.ID
	}
	if !strings.HasPrefix(fullPath, "/") {
		fullPath = "/" + fullPath
	}
	return ResourceLocation{
		ID:           node.ID,
		ResourceType: node.Type,
		FolderID:     folderID,
		FolderPath:   folderPath,
		FullPath:     fullPath,
		Hash:         hash,
	}
}

func (s *Service) folderItemResourceLocation(item VfsFolderItem, node *Node) ResourceLocation {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if node != nil {
		return s.nodeResourceLocationLocked(*node)
	}
	folderPath := "/"
	if item.FolderID != nil {
		folderPath = s.folderPathLocked(*item.FolderID)
	}
	fullPath := s.buildResourcePathLocked(item.FolderID, item.ItemID)
	return ResourceLocation{
		ID:           item.ItemID,
		ResourceType: item.ItemType,
		FolderID:     cloneStringPtr(item.FolderID),
		FolderPath:   folderPath,
		FullPath:     fullPath,
	}
}

func (s *Service) folderResourceInfoFromNode(node Node) FolderResourceInfo {
	info := folderResourceInfoFromNode(node)
	info.Path = s.nodeResourceLocation(node).FullPath
	return info
}

func (s *Service) syncNoteResourceLocked(index int) error {
	if s.vfs == nil {
		return nil
	}
	note := s.state.Notes[index]
	result, err := s.vfs.CreateOrUpdateSource(vfs.CreateResourceInput{
		Type:     "note",
		Data:     note.Content,
		SourceID: &note.ID,
		Metadata: noteResourceMetadata(note),
	})
	if err != nil {
		return err
	}
	s.state.Notes[index].ResourceID = result.ResourceID
	s.state.Notes[index].ResourceHash = result.Hash
	return nil
}

func (s *Service) createVfsBackedNode(path string, name string, options CreateOptions) (Node, error) {
	if s.vfs == nil {
		return Node{}, fmt.Errorf("lean Go DSTU has no VFS service for %q resources", options.Type)
	}
	if options.FileBase == nil || strings.TrimSpace(*options.FileBase) == "" {
		return Node{}, fmt.Errorf("DSTU %q resources require fileBase64 content", options.Type)
	}
	metadata := normalizeMetadata(options.Metadata)
	metadata["name"] = name
	metadata["title"] = name
	metadata["sourceDb"] = "dstu"
	metadata["sourceType"] = options.Type
	metadata["resourceType"] = options.Type
	metadata["type"] = options.Type
	metadata["previewType"] = previewTypeForDstuType(options.Type, metadataString(metadata, "mimeType", ""), name, "")
	if !isRootPath(path) {
		metadata["parentPath"] = path
	}
	mimeType := metadataString(metadata, "mimeType", "application/octet-stream")
	fileType := fileTypeForDstuCreate(options.Type, mimeType)
	var folderID *string
	if value := metadataString(metadata, "folderId", ""); value != "" {
		folderID = &value
	}
	result, err := s.vfs.UploadFile(vfs.UploadFileInput{
		Name:          name,
		MimeType:      mimeType,
		Base64Content: *options.FileBase,
		FileType:      &fileType,
		FolderID:      folderID,
		Metadata:      metadata,
	})
	if err != nil {
		return Node{}, err
	}
	return vfsFileToNode(result.File), nil
}

func (s *Service) listVfsFiles() ([]vfs.VfsFile, error) {
	files := []vfs.VfsFile{}
	const batchSize = 500
	for offset := 0; ; offset += batchSize {
		batch, err := s.vfs.ListFiles(vfs.ListFilesInput{Limit: batchSize, Offset: offset})
		if err != nil {
			return nil, err
		}
		files = append(files, batch...)
		if len(batch) < batchSize {
			break
		}
	}
	return files, nil
}

func (s *Service) load() error {
	bytes, err := os.ReadFile(s.path)
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	if err != nil {
		return err
	}
	if len(bytes) == 0 {
		return nil
	}
	if err := json.Unmarshal(bytes, &s.state); err != nil {
		return err
	}
	if s.state.Notes == nil {
		s.state.Notes = []NoteRecord{}
	}
	if s.state.Folders == nil {
		s.state.Folders = []VfsFolder{}
	}
	if s.state.FolderItems == nil {
		s.state.FolderItems = []VfsFolderItem{}
	}
	return nil
}

func (s *Service) flushLocked() error {
	return storage.WriteJSONAtomic(s.path, s.state)
}

func noteToNode(note NoteRecord) Node {
	resourceID := note.ResourceID
	if resourceID == "" {
		resourceID = note.ID
	}
	resourceHash := note.ResourceHash
	if resourceHash == "" {
		resourceHash = noteHash(note)
	}
	return Node{
		ID:           note.ID,
		Path:         "/" + note.ID,
		Name:         note.Name,
		Type:         "note",
		Size:         len([]byte(note.Content)),
		CreatedAt:    note.CreatedAt,
		UpdatedAt:    note.UpdatedAt,
		ResourceID:   resourceID,
		SourceID:     note.ID,
		ResourceHash: resourceHash,
		PreviewType:  "markdown",
		Metadata:     normalizeMetadata(note.Metadata),
	}
}

func noteResourceMetadata(note NoteRecord) map[string]any {
	metadata := normalizeMetadata(note.Metadata)
	metadata["name"] = note.Name
	metadata["title"] = note.Name
	metadata["path"] = "/" + note.ID
	metadata["sourceId"] = note.ID
	metadata["sourceDb"] = "dstu"
	metadata["sourceType"] = "note"
	metadata["previewType"] = "markdown"
	metadata["size"] = len([]byte(note.Content))
	metadata["updatedAt"] = note.UpdatedAt
	return metadata
}

func vfsFileToNode(file vfs.VfsFile) Node {
	metadata := normalizeMetadata(file.Metadata)
	nodeType := nodeTypeForVfsFile(file, metadata)
	previewType := previewTypeForDstuType(nodeType, metadataString(metadata, "mimeType", derefString(file.MimeType)), file.FileName, file.FileType)
	resourceID := ""
	if file.ResourceID != nil {
		resourceID = *file.ResourceID
		metadata["resourceId"] = *file.ResourceID
	}
	if file.BlobHash != nil {
		metadata["blobHash"] = *file.BlobHash
	}
	if file.MimeType != nil {
		metadata["mimeType"] = *file.MimeType
	}
	if file.OriginalPath != nil {
		metadata["filePath"] = *file.OriginalPath
		metadata["originalPath"] = *file.OriginalPath
	}
	if file.PageCount != nil {
		metadata["pageCount"] = *file.PageCount
	}
	if file.LastPage != nil {
		metadata["lastPage"] = *file.LastPage
	}
	if file.LastOpenedAt != nil {
		metadata["lastOpenedAt"] = *file.LastOpenedAt
	}
	if file.ExtractedText != nil {
		metadata["extractedText"] = *file.ExtractedText
	}
	if file.PreviewJSON != nil {
		metadata["previewJson"] = *file.PreviewJSON
	}
	if file.OcrPagesJSON != nil {
		metadata["ocrPagesJson"] = *file.OcrPagesJSON
	}
	if file.Description != nil {
		metadata["description"] = *file.Description
	}
	metadata["fileSize"] = file.Size
	metadata["size"] = file.Size
	metadata["fileType"] = file.FileType
	metadata["isFavorite"] = file.IsFavorite
	metadata["tags"] = file.Tags
	metadata["bookmarks"] = file.Bookmarks
	metadata["sourceId"] = file.ID
	metadata["resourceHash"] = file.SHA256
	metadata["sha256"] = file.SHA256
	metadata["contentHash"] = file.SHA256
	metadata["status"] = file.Status
	metadata["previewType"] = previewType

	return Node{
		ID:           file.ID,
		Path:         "/" + file.ID,
		Name:         file.FileName,
		Type:         nodeType,
		Size:         int(file.Size),
		CreatedAt:    parseVfsTimestampMillis(file.CreatedAt),
		UpdatedAt:    parseVfsTimestampMillis(file.UpdatedAt),
		ResourceID:   resourceID,
		SourceID:     file.ID,
		ResourceHash: file.SHA256,
		PreviewType:  previewType,
		Metadata:     metadata,
	}
}

func textbookRecordFromVfsFile(file vfs.VfsFile) TextbookRecord {
	metadata := normalizeMetadata(file.Metadata)
	filePath := derefString(file.OriginalPath)
	if filePath == "" {
		filePath = metadataString(metadata, "filePath", metadataString(metadata, "originalPath", ""))
	}
	tags := metadataTags(metadata)
	bookmarks := file.Bookmarks
	favorite := 0
	if file.IsFavorite {
		favorite = 1
	}
	resourceID := ""
	if file.ResourceID != nil {
		resourceID = *file.ResourceID
	}
	metadata["filePath"] = filePath
	metadata["sourceId"] = file.ID
	metadata["resourceHash"] = file.SHA256
	return TextbookRecord{
		ID:            file.ID,
		SHA256:        file.SHA256,
		FileName:      file.FileName,
		FilePath:      filePath,
		Size:          file.Size,
		PageCount:     file.PageCount,
		TagsJSON:      jsonString(tags),
		Favorite:      favorite,
		LastOpenedAt:  file.LastOpenedAt,
		LastPage:      file.LastPage,
		BookmarksJSON: jsonString(bookmarks),
		CoverKey:      file.CoverKey,
		Status:        file.Status,
		CreatedAt:     file.CreatedAt,
		UpdatedAt:     file.UpdatedAt,
		ResourceID:    resourceID,
		ResourceHash:  file.SHA256,
		PreviewType:   previewTypeForDstuType("textbook", metadataString(metadata, "mimeType", derefString(file.MimeType)), file.FileName, file.FileType),
		Metadata:      metadata,
	}
}

func normalizeMetadata(metadata map[string]any) map[string]any {
	out := map[string]any{
		"tags":       []string{},
		"isFavorite": false,
	}
	for key, value := range metadata {
		out[key] = value
	}
	return out
}

func normalizeFolderID(value *string) *string {
	if value == nil {
		return nil
	}
	trimmed := strings.TrimSpace(*value)
	if trimmed == "" || strings.EqualFold(trimmed, "root") {
		return nil
	}
	return &trimmed
}

func normalizeOptionalStringPtr(value *string) *string {
	if value == nil {
		return nil
	}
	trimmed := strings.TrimSpace(*value)
	if trimmed == "" {
		return nil
	}
	return &trimmed
}

func cloneStringPtr(value *string) *string {
	if value == nil {
		return nil
	}
	cloned := strings.TrimSpace(*value)
	if cloned == "" {
		return nil
	}
	return &cloned
}

func sameOptionalString(left *string, right *string) bool {
	leftValue := ""
	rightValue := ""
	if left != nil {
		leftValue = strings.TrimSpace(*left)
	}
	if right != nil {
		rightValue = strings.TrimSpace(*right)
	}
	return leftValue == rightValue
}

func normalizeItemType(itemType string, itemID string) string {
	normalized := strings.ToLower(strings.TrimSpace(itemType))
	if normalized == "" || normalized == "unknown" {
		normalized = inferResourceType(itemID)
	}
	if normalized == "attachment" {
		return "file"
	}
	return normalized
}

func inferResourceType(resourceID string) string {
	id := strings.ToLower(idFromPath(resourceID))
	switch {
	case strings.HasPrefix(id, "note_"):
		return "note"
	case strings.HasPrefix(id, "tb_"):
		return "textbook"
	case strings.HasPrefix(id, "exam_"):
		return "exam"
	case strings.HasPrefix(id, "tr_"):
		return "translation"
	case strings.HasPrefix(id, "essay_"):
		return "essay"
	case strings.HasPrefix(id, "img_"):
		return "image"
	case strings.HasPrefix(id, "file_"), strings.HasPrefix(id, "att_"):
		return "file"
	case strings.HasPrefix(id, "mm_"):
		return "mindmap"
	case strings.HasPrefix(id, "fld_"):
		return "folder"
	default:
		return ""
	}
}

func isValidFolderItemType(value string) bool {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "note", "textbook", "exam", "translation", "essay", "image", "file", "mindmap":
		return true
	default:
		return false
	}
}

func isVfsFolderMetadataType(value string) bool {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "file", "image", "textbook", "mindmap":
		return true
	default:
		return false
	}
}

func folderItemKey(itemType string, itemID string) string {
	return strings.ToLower(strings.TrimSpace(itemType)) + "\x00" + idFromPath(itemID)
}

func folderItemLookup(items []VfsFolderItem) map[string]VfsFolderItem {
	lookup := make(map[string]VfsFolderItem, len(items)*2)
	for _, item := range items {
		if item.ItemID == "" {
			continue
		}
		lookup[folderItemKey(item.ItemType, item.ItemID)] = item
		if _, exists := lookup[folderItemKey("", item.ItemID)]; !exists {
			lookup[folderItemKey("", item.ItemID)] = item
		}
	}
	return lookup
}

func applyFolderItemToNode(node *Node, lookup map[string]VfsFolderItem, folderPathIDs map[string][]string, folderPaths map[string]string) {
	if node == nil {
		return
	}
	item, ok := lookup[folderItemKey(node.Type, node.ID)]
	if !ok {
		item, ok = lookup[folderItemKey("", node.ID)]
	}
	if !ok {
		return
	}
	metadata := normalizeMetadata(node.Metadata)
	folderPath := "/"
	folderIDs := []string{}
	if item.FolderID != nil && strings.TrimSpace(*item.FolderID) != "" {
		folderID := strings.TrimSpace(*item.FolderID)
		metadata["folderId"] = folderID
		if values, ok := folderPathIDs[folderID]; ok {
			folderIDs = append([]string(nil), values...)
		} else {
			folderIDs = []string{folderID}
		}
		if value, ok := folderPaths[folderID]; ok && value != "" {
			folderPath = value
		} else {
			folderPath = folderPathForID(folderID)
		}
	} else {
		metadata["folderId"] = ""
	}
	fullPath := strings.TrimSpace(derefString(item.CachedPath))
	if fullPath == "" {
		fullPath = strings.TrimRight(folderPath, "/") + "/" + node.ID
	}
	if !strings.HasPrefix(fullPath, "/") {
		fullPath = "/" + fullPath
	}
	metadata["folderPath"] = folderPath
	metadata["folderPathIds"] = folderIDs
	metadata["parentPath"] = folderPath
	metadata["cachedPath"] = fullPath
	metadata["path"] = fullPath
	node.Path = fullPath
	node.Metadata = metadata
}

func folderPathIDLookup(folders []VfsFolder) map[string][]string {
	out := map[string][]string{}
	for _, folder := range folders {
		out[folder.ID] = folderPathIDsFromSlice(folders, folder.ID)
	}
	return out
}

func folderTitlePathLookup(folders []VfsFolder) map[string]string {
	out := map[string]string{}
	for _, folder := range folders {
		out[folder.ID] = folderTitlePathFromSlice(folders, folder.ID)
	}
	return out
}

func folderPathIDsFromSlice(folders []VfsFolder, folderID string) []string {
	chain := folderChainFromSlice(folders, folderID)
	if len(chain) == 0 {
		return []string{strings.TrimSpace(folderID)}
	}
	ids := make([]string, 0, len(chain))
	for index := len(chain) - 1; index >= 0; index-- {
		ids = append(ids, chain[index].ID)
	}
	return ids
}

func folderTitlePathFromSlice(folders []VfsFolder, folderID string) string {
	chain := folderChainFromSlice(folders, folderID)
	if len(chain) == 0 {
		return folderPathForID(folderID)
	}
	parts := make([]string, 0, len(chain))
	for index := len(chain) - 1; index >= 0; index-- {
		title := strings.Trim(strings.TrimSpace(chain[index].Title), "/\\")
		if title == "" {
			title = chain[index].ID
		}
		parts = append(parts, title)
	}
	return "/" + strings.Join(parts, "/")
}

func folderChainFromSlice(folders []VfsFolder, folderID string) []VfsFolder {
	byID := make(map[string]VfsFolder, len(folders))
	for _, folder := range folders {
		byID[folder.ID] = folder
	}
	currentID := strings.TrimSpace(folderID)
	seen := map[string]struct{}{}
	chain := []VfsFolder{}
	for currentID != "" {
		if _, ok := seen[currentID]; ok {
			return chain
		}
		seen[currentID] = struct{}{}
		folder, ok := byID[currentID]
		if !ok {
			return chain
		}
		chain = append(chain, folder)
		if folder.ParentID == nil || strings.TrimSpace(*folder.ParentID) == "" {
			break
		}
		currentID = strings.TrimSpace(*folder.ParentID)
	}
	return chain
}

func folderIDSetWithDescendants(folders []VfsFolder, folderID string, includeSelf bool) map[string]struct{} {
	start := strings.TrimSpace(folderID)
	out := map[string]struct{}{}
	if start == "" {
		return out
	}
	if includeSelf {
		out[start] = struct{}{}
	}
	changed := true
	for changed {
		changed = false
		for _, folder := range folders {
			if folder.ParentID == nil {
				continue
			}
			parentID := strings.TrimSpace(*folder.ParentID)
			if parentID != start {
				if _, ok := out[parentID]; !ok {
					continue
				}
			}
			if _, exists := out[folder.ID]; exists {
				continue
			}
			out[folder.ID] = struct{}{}
			changed = true
		}
	}
	return out
}

func sortFolders(folders []VfsFolder) {
	sort.SliceStable(folders, func(left int, right int) bool {
		if !sameOptionalString(folders[left].ParentID, folders[right].ParentID) {
			return derefString(folders[left].ParentID) < derefString(folders[right].ParentID)
		}
		if folders[left].SortOrder != folders[right].SortOrder {
			return folders[left].SortOrder < folders[right].SortOrder
		}
		return strings.ToLower(folders[left].Title) < strings.ToLower(folders[right].Title)
	})
}

func sortFolderItems(items []VfsFolderItem) {
	sort.SliceStable(items, func(left int, right int) bool {
		if !sameOptionalString(items[left].FolderID, items[right].FolderID) {
			return derefString(items[left].FolderID) < derefString(items[right].FolderID)
		}
		if items[left].SortOrder != items[right].SortOrder {
			return items[left].SortOrder < items[right].SortOrder
		}
		return items[left].ItemID < items[right].ItemID
	})
}

func buildFolderTree(parentID *string, folders []VfsFolder, items []VfsFolderItem) []FolderTreeNode {
	nodes := []FolderTreeNode{}
	for _, folder := range folders {
		if !sameOptionalString(folder.ParentID, parentID) {
			continue
		}
		folderID := folder.ID
		nodeItems := []VfsFolderItem{}
		for _, item := range items {
			if item.FolderID != nil && *item.FolderID == folderID {
				nodeItems = append(nodeItems, item)
			}
		}
		nodes = append(nodes, FolderTreeNode{
			Folder:   folder,
			Children: buildFolderTree(&folderID, folders, items),
			Items:    nodeItems,
		})
	}
	return nodes
}

func matchesNodeFolderOption(node Node, options *ListOptions) bool {
	if options == nil || options.FolderID == nil || strings.TrimSpace(*options.FolderID) == "" {
		return true
	}
	recursive := options.Recursive != nil && *options.Recursive
	if recursive {
		return nodeInFolderScope(node, *options.FolderID, true)
	}
	return nodeMatchesFolder(node, *options.FolderID)
}

func folderResourceInfoFromItem(item VfsFolderItem, folders []VfsFolder) FolderResourceInfo {
	path := derefString(item.CachedPath)
	if path == "" {
		path = resourcePathFromFolders(folders, item.FolderID, item.ItemID)
	}
	return FolderResourceInfo{
		ItemType: item.ItemType,
		ItemID:   item.ItemID,
		Title:    item.ItemID,
		Path:     path,
	}
}

func resourcePathFromFolders(folders []VfsFolder, folderID *string, resourceID string) string {
	if folderID == nil || strings.TrimSpace(*folderID) == "" {
		return "/" + idFromPath(resourceID)
	}
	return strings.TrimRight(folderTitlePathFromSlice(folders, *folderID), "/") + "/" + idFromPath(resourceID)
}

func normalizeDstuPath(path string) string {
	trimmed := strings.TrimSpace(strings.ReplaceAll(path, "\\", "/"))
	if trimmed == "" {
		return "/"
	}
	parts := []string{}
	for _, part := range strings.Split(trimmed, "/") {
		part = strings.TrimSpace(part)
		if part != "" {
			parts = append(parts, part)
		}
	}
	if len(parts) == 0 {
		return "/"
	}
	return "/" + strings.Join(parts, "/")
}

func listOptionsMayIncludeVfs(options *ListOptions) bool {
	if options == nil {
		return true
	}
	if options.TypeFilter != nil && *options.TypeFilter != "" && !isVfsDstuType(*options.TypeFilter) {
		return false
	}
	if len(options.Types) == 0 {
		return true
	}
	for _, resourceType := range options.Types {
		if isVfsDstuType(resourceType) {
			return true
		}
	}
	return false
}

func isVfsDstuType(value string) bool {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "file", "image", "textbook":
		return true
	default:
		return false
	}
}

func matchesListOptions(note NoteRecord, options *ListOptions) bool {
	if noteIsDeleted(note) {
		return false
	}
	if options == nil {
		return true
	}
	if options.TypeFilter != nil && *options.TypeFilter != "" && *options.TypeFilter != "note" {
		return false
	}
	if len(options.Types) > 0 && !contains(options.Types, "note") {
		return false
	}
	if options.IsFavorite != nil && metadataFavorite(note.Metadata) != *options.IsFavorite {
		return false
	}
	if len(options.Tags) > 0 && !hasAllTags(metadataTags(note.Metadata), options.Tags) {
		return false
	}
	if options.Search != nil && strings.TrimSpace(*options.Search) != "" {
		query := strings.ToLower(strings.TrimSpace(*options.Search))
		haystack := strings.ToLower(note.Name + "\n" + note.Content + "\n" + strings.Join(metadataTags(note.Metadata), "\n"))
		if !strings.Contains(haystack, query) {
			return false
		}
	}
	return true
}

func matchesNodeListOptions(node Node, options *ListOptions) bool {
	if options == nil {
		return true
	}
	if options.TypeFilter != nil && *options.TypeFilter != "" && node.Type != *options.TypeFilter {
		return false
	}
	if len(options.Types) > 0 && !contains(options.Types, node.Type) {
		return false
	}
	if !matchesNodeFolderOption(node, options) {
		return false
	}
	if options.IsFavorite != nil && metadataFavorite(node.Metadata) != *options.IsFavorite {
		return false
	}
	if len(options.Tags) > 0 && !hasAllTags(metadataTags(node.Metadata), options.Tags) {
		return false
	}
	if options.Search != nil && strings.TrimSpace(*options.Search) != "" {
		query := strings.ToLower(strings.TrimSpace(*options.Search))
		if !strings.Contains(strings.ToLower(nodeSearchText(node)), query) {
			return false
		}
	}
	return true
}

func nodeTypeForVfsFile(file vfs.VfsFile, metadata map[string]any) string {
	for _, key := range []string{"sourceType", "resourceType", "type"} {
		if strings.ToLower(metadataString(metadata, key, "")) == "textbook" {
			return "textbook"
		}
	}
	if strings.HasPrefix(file.ID, "tb_") {
		return "textbook"
	}
	if file.FileType == "image" || strings.HasPrefix(strings.ToLower(derefString(file.MimeType)), "image/") {
		return "image"
	}
	return "file"
}

func previewTypeForDstuType(nodeType string, mimeType string, name string, fileType string) string {
	lowerMime := strings.ToLower(strings.TrimSpace(mimeType))
	lowerName := strings.ToLower(strings.TrimSpace(name))
	if nodeType == "textbook" || lowerMime == "application/pdf" || strings.HasSuffix(lowerName, ".pdf") {
		return "pdf"
	}
	if nodeType == "image" || fileType == "image" || strings.HasPrefix(lowerMime, "image/") {
		return "image"
	}
	if strings.HasPrefix(lowerMime, "audio/") || fileType == "audio" {
		return "audio"
	}
	if strings.HasPrefix(lowerMime, "video/") || fileType == "video" {
		return "video"
	}
	if strings.HasPrefix(lowerMime, "text/") ||
		lowerMime == "application/json" ||
		lowerMime == "application/xml" ||
		strings.HasSuffix(lowerName, ".txt") ||
		strings.HasSuffix(lowerName, ".md") ||
		strings.HasSuffix(lowerName, ".markdown") ||
		strings.HasSuffix(lowerName, ".csv") ||
		strings.HasSuffix(lowerName, ".json") ||
		strings.HasSuffix(lowerName, ".xml") {
		return "text"
	}
	return "none"
}

func fileTypeForDstuCreate(nodeType string, mimeType string) string {
	lowerMime := strings.ToLower(strings.TrimSpace(mimeType))
	if nodeType == "image" || strings.HasPrefix(lowerMime, "image/") {
		return "image"
	}
	if strings.HasPrefix(lowerMime, "audio/") {
		return "audio"
	}
	if strings.HasPrefix(lowerMime, "video/") {
		return "video"
	}
	return "document"
}

func mimeTypeForName(name string) string {
	extension := strings.ToLower(filepath.Ext(strings.TrimSpace(name)))
	if extension != "" {
		if detected := mime.TypeByExtension(extension); detected != "" {
			return strings.Split(detected, ";")[0]
		}
	}
	switch extension {
	case ".md", ".markdown":
		return "text/markdown"
	case ".txt":
		return "text/plain"
	case ".csv":
		return "text/csv"
	case ".json":
		return "application/json"
	case ".xml":
		return "application/xml"
	case ".pdf":
		return "application/pdf"
	case ".docx":
		return "application/vnd.openxmlformats-officedocument.wordprocessingml.document"
	case ".xlsx":
		return "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet"
	case ".pptx":
		return "application/vnd.openxmlformats-officedocument.presentationml.presentation"
	default:
		return "application/octet-stream"
	}
}

func nodeMatchesFolder(node Node, folderID string) bool {
	want := strings.ToLower(strings.TrimSpace(folderID))
	if want == "" {
		return true
	}
	for _, key := range []string{"folderId", "folderID", "parentId", "attachmentRootFolderId"} {
		if strings.ToLower(metadataString(node.Metadata, key, "")) == want {
			return true
		}
	}
	for _, value := range metadataTagsKey(node.Metadata, "folderIds") {
		if strings.ToLower(strings.TrimSpace(value)) == want {
			return true
		}
	}
	return false
}

func nodeInFolderScope(node Node, folderID string, includeSubfolders bool) bool {
	if nodeMatchesFolder(node, folderID) {
		return true
	}
	if !includeSubfolders {
		return false
	}
	want := strings.ToLower(strings.TrimSpace(folderID))
	if want == "" {
		return true
	}
	for _, key := range []string{"folderPath", "parentPath", "path"} {
		value := strings.ToLower(strings.TrimSpace(metadataString(node.Metadata, key, "")))
		if value == "" {
			continue
		}
		if strings.Contains(value, "/"+want+"/") ||
			strings.HasSuffix(value, "/"+want) ||
			strings.HasPrefix(value, want+"/") {
			return true
		}
	}
	for _, value := range metadataTagsKey(node.Metadata, "folderPathIds") {
		normalized := strings.ToLower(strings.TrimSpace(value))
		if normalized == want {
			return true
		}
	}
	return false
}

func nodeFolderID(node Node) *string {
	for _, key := range []string{"folderId", "folderID", "parentId", "attachmentRootFolderId"} {
		if value := metadataString(node.Metadata, key, ""); value != "" {
			return &value
		}
	}
	return nil
}

func folderPathForID(folderID string) string {
	trimmed := strings.Trim(strings.TrimSpace(folderID), "/")
	if trimmed == "" {
		return "/"
	}
	return "/" + trimmed
}

func folderTitleForID(folderID string) string {
	trimmed := strings.Trim(strings.TrimSpace(folderID), "/")
	if trimmed == "" {
		return "Root"
	}
	parts := strings.FieldsFunc(trimmed, func(r rune) bool {
		return r == '/' || r == '\\'
	})
	if len(parts) == 0 {
		return trimmed
	}
	return parts[len(parts)-1]
}

func nodeResourceLocation(node Node) ResourceLocation {
	folderID := nodeFolderID(node)
	folderPath := "/"
	if folderID != nil {
		folderPath = folderPathForID(*folderID)
	}
	hash := (*string)(nil)
	if strings.TrimSpace(node.ResourceHash) != "" {
		value := strings.TrimSpace(node.ResourceHash)
		hash = &value
	}
	return ResourceLocation{
		ID:           node.ID,
		ResourceType: node.Type,
		FolderID:     folderID,
		FolderPath:   folderPath,
		FullPath:     strings.TrimRight(folderPath, "/") + "/" + node.ID,
		Hash:         hash,
	}
}

func folderResourceInfoFromNode(node Node) FolderResourceInfo {
	resourceID := (*string)(nil)
	if strings.TrimSpace(node.ResourceID) != "" {
		value := strings.TrimSpace(node.ResourceID)
		resourceID = &value
	}
	return FolderResourceInfo{
		ItemType:   node.Type,
		ItemID:     node.ID,
		ResourceID: resourceID,
		Title:      node.Name,
		Path:       nodeResourceLocation(node).FullPath,
	}
}

func nodeSearchText(node Node) string {
	parts := []string{
		node.ID,
		node.Name,
		node.Type,
		node.SourceID,
		node.ResourceID,
		node.ResourceHash,
		strings.Join(metadataTags(node.Metadata), "\n"),
	}
	for _, key := range []string{"mimeType", "fileType", "description", "sourceType", "resourceType", "filePath", "originalPath", "extractedText"} {
		if value := metadataString(node.Metadata, key, ""); value != "" {
			parts = append(parts, value)
		}
	}
	return strings.Join(parts, "\n")
}

func sortNodes(nodes []Node, options *ListOptions) {
	sortBy := "updatedAt"
	sortOrder := "desc"
	if options != nil {
		if options.SortBy != nil && *options.SortBy != "" {
			sortBy = *options.SortBy
		}
		if options.SortOrder != nil && *options.SortOrder != "" {
			sortOrder = *options.SortOrder
		}
	}
	desc := sortOrder != "asc"
	sort.SliceStable(nodes, func(a, b int) bool {
		left := nodes[a]
		right := nodes[b]
		switch sortBy {
		case "name":
			if desc {
				return strings.ToLower(left.Name) > strings.ToLower(right.Name)
			}
			return strings.ToLower(left.Name) < strings.ToLower(right.Name)
		case "createdAt":
			if desc {
				return left.CreatedAt > right.CreatedAt
			}
			return left.CreatedAt < right.CreatedAt
		default:
			if desc {
				return left.UpdatedAt > right.UpdatedAt
			}
			return left.UpdatedAt < right.UpdatedAt
		}
	})
}

func paginate(nodes []Node, options *ListOptions) []Node {
	if options == nil {
		return nodes
	}
	offset := 0
	limit := len(nodes)
	if options.Offset != nil && *options.Offset > 0 {
		offset = *options.Offset
	}
	if offset >= len(nodes) {
		return []Node{}
	}
	if options.Limit != nil && *options.Limit >= 0 {
		limit = *options.Limit
	}
	end := offset + limit
	if end > len(nodes) {
		end = len(nodes)
	}
	return nodes[offset:end]
}

func paginateNodes(nodes []Node, limit *int, offset *int) []Node {
	start := 0
	if offset != nil && *offset > 0 {
		start = *offset
	}
	if start >= len(nodes) {
		return []Node{}
	}
	end := len(nodes)
	if limit != nil && *limit >= 0 {
		end = start + *limit
	}
	if end > len(nodes) {
		end = len(nodes)
	}
	return nodes[start:end]
}

func filterFolderItems(items []VfsFolderItem, keep func(VfsFolderItem) bool) []VfsFolderItem {
	out := make([]VfsFolderItem, 0, len(items))
	for _, item := range items {
		if keep(item) {
			out = append(out, item)
		}
	}
	return out
}

func noteIsDeleted(note NoteRecord) bool {
	status := strings.ToLower(metadataString(note.Metadata, "status", ""))
	return status == "deleted" || status == "trash" || metadataString(note.Metadata, "deletedAt", "") != ""
}

func resourceTypeAllowsNotes(resourceType string) bool {
	switch strings.ToLower(strings.TrimSpace(resourceType)) {
	case "", "note", "notes":
		return true
	default:
		return false
	}
}

func formatMillis(value int64) string {
	if value <= 0 {
		value = nowMillis()
	}
	return time.UnixMilli(value).UTC().Format(time.RFC3339Nano)
}

func metadataFavorite(metadata map[string]any) bool {
	value, ok := metadata["isFavorite"]
	if !ok {
		return false
	}
	if favorite, ok := value.(bool); ok {
		return favorite
	}
	return false
}

func metadataTags(metadata map[string]any) []string {
	return metadataStringValues(metadata, "tags")
}

func metadataTagsKey(metadata map[string]any, key string) []string {
	return metadataStringValues(metadata, key)
}

func metadataStringValues(metadata map[string]any, key string) []string {
	value, ok := metadata["tags"]
	if key != "tags" {
		value, ok = metadata[key]
	}
	if !ok {
		return []string{}
	}
	switch tags := value.(type) {
	case []string:
		out := make([]string, 0, len(tags))
		for _, tag := range tags {
			if trimmed := strings.TrimSpace(tag); trimmed != "" {
				out = append(out, trimmed)
			}
		}
		return out
	case []any:
		out := make([]string, 0, len(tags))
		for _, tag := range tags {
			if text, ok := tag.(string); ok {
				if trimmed := strings.TrimSpace(text); trimmed != "" {
					out = append(out, trimmed)
				}
			}
		}
		return out
	case string:
		if strings.TrimSpace(tags) == "" {
			return []string{}
		}
		parts := strings.Split(tags, ",")
		out := make([]string, 0, len(parts))
		for _, part := range parts {
			if trimmed := strings.TrimSpace(part); trimmed != "" {
				out = append(out, trimmed)
			}
		}
		return out
	default:
		return []string{}
	}
}

func metadataString(metadata map[string]any, key string, fallback string) string {
	if value, ok := metadata[key].(string); ok && strings.TrimSpace(value) != "" {
		return strings.TrimSpace(value)
	}
	return fallback
}

func jsonString(value any) string {
	bytes, err := json.Marshal(value)
	if err != nil {
		return "[]"
	}
	return string(bytes)
}

func derefString(value *string) string {
	if value == nil {
		return ""
	}
	return strings.TrimSpace(*value)
}

func parseVfsTimestampMillis(value string) int64 {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return nowMillis()
	}
	if parsed, err := time.Parse(time.RFC3339Nano, trimmed); err == nil {
		return parsed.UnixMilli()
	}
	if parsed, err := time.Parse(time.RFC3339, trimmed); err == nil {
		return parsed.UnixMilli()
	}
	return nowMillis()
}

func hasAllTags(actual []string, expected []string) bool {
	for _, want := range expected {
		if !contains(actual, want) {
			return false
		}
	}
	return true
}

func parseNotesSearchKeyword(keyword string) ([]string, []string) {
	parts := strings.Fields(strings.TrimSpace(keyword))
	tags := make([]string, 0, len(parts))
	terms := make([]string, 0, len(parts))
	for _, part := range parts {
		if tag, ok := strings.CutPrefix(part, "tag:"); ok {
			tag = strings.TrimSpace(tag)
			if tag != "" {
				tags = append(tags, tag)
			}
			continue
		}
		if part != "" {
			terms = append(terms, part)
		}
	}
	return tags, terms
}

func hasAllTagsFold(actual []string, expected []string) bool {
	seen := make(map[string]struct{}, len(actual))
	for _, tag := range actual {
		if normalized := strings.ToLower(strings.TrimSpace(tag)); normalized != "" {
			seen[normalized] = struct{}{}
		}
	}
	for _, want := range expected {
		normalized := strings.ToLower(strings.TrimSpace(want))
		if normalized == "" {
			continue
		}
		if _, ok := seen[normalized]; !ok {
			return false
		}
	}
	return true
}

func noteMatchesTerms(note NoteRecord, terms []string) bool {
	haystack := strings.ToLower(strings.Join([]string{
		note.Name,
		note.Content,
		strings.Join(metadataTags(note.Metadata), "\n"),
	}, "\n"))
	for _, term := range terms {
		normalized := strings.ToLower(strings.TrimSpace(term))
		if normalized == "" {
			continue
		}
		if !strings.Contains(haystack, normalized) {
			return false
		}
	}
	return true
}

func snippetForQuery(text string, query string, maxLength int) *string {
	term := strings.TrimSpace(query)
	if term == "" || maxLength <= 0 {
		return nil
	}
	lowerText := strings.ToLower(text)
	lowerTerm := strings.ToLower(term)
	byteIndex := strings.Index(lowerText, lowerTerm)
	if byteIndex < 0 {
		for _, part := range strings.Fields(lowerTerm) {
			if part == "" {
				continue
			}
			if index := strings.Index(lowerText, part); index >= 0 {
				byteIndex = index
				break
			}
		}
	}
	if byteIndex < 0 {
		return nil
	}

	charIndex := len([]rune(text[:byteIndex]))
	runes := []rune(text)
	if len(runes) <= maxLength {
		snippet := strings.TrimSpace(text)
		if snippet == "" {
			return nil
		}
		return &snippet
	}
	half := maxLength / 2
	start := charIndex - half
	if start < 0 {
		start = 0
	}
	end := start + maxLength
	if end > len(runes) {
		end = len(runes)
		start = end - maxLength
		if start < 0 {
			start = 0
		}
	}
	snippet := strings.TrimSpace(string(runes[start:end]))
	if snippet == "" {
		return nil
	}
	if start > 0 {
		snippet = "..." + snippet
	}
	if end < len(runes) {
		snippet += "..."
	}
	return &snippet
}

func (s *Service) noteContentByID(noteID string) (string, error) {
	id := idFromPath(noteID)
	if id == "" {
		return "", errors.New("note id cannot be empty")
	}
	s.mu.RLock()
	note, ok := s.findNoteLocked(id)
	s.mu.RUnlock()
	if !ok {
		return "", fmt.Errorf("note not found: %s", noteID)
	}
	return note.Content, nil
}

type markdownHeading struct {
	index int
	level int
	line  string
	text  string
}

func extractMarkdownSection(content string, sectionTitle string) (string, bool) {
	lines := strings.Split(content, "\n")
	heading, ok := findMarkdownHeading(lines, sectionTitle)
	if !ok {
		return "", false
	}
	end := findMarkdownSectionEnd(lines, heading.index, heading.level)
	return strings.TrimSpace(strings.Join(lines[heading.index+1:end], "\n")), true
}

func appendToMarkdownSection(content string, sectionTitle string, appendContent string) (string, bool) {
	lines := strings.Split(content, "\n")
	heading, ok := findMarkdownHeading(lines, sectionTitle)
	if !ok {
		return "", false
	}
	end := findMarkdownSectionEnd(lines, heading.index, heading.level)
	out := make([]string, 0, len(lines)+2)
	out = append(out, lines[:end]...)
	out = append(out, "", appendContent)
	out = append(out, lines[end:]...)
	return strings.Join(out, "\n"), true
}

func findMarkdownHeading(lines []string, sectionTitle string) (markdownHeading, bool) {
	section := strings.ToLower(strings.TrimSpace(sectionTitle))
	if section == "" {
		return markdownHeading{}, false
	}
	for index, line := range lines {
		heading, ok := parseMarkdownHeading(index, line)
		if !ok {
			continue
		}
		if strings.ToLower(heading.text) == section || strings.ToLower(heading.line) == section {
			return heading, true
		}
	}
	return markdownHeading{}, false
}

func parseMarkdownHeading(index int, line string) (markdownHeading, bool) {
	trimmed := strings.TrimSpace(line)
	if !strings.HasPrefix(trimmed, "#") {
		return markdownHeading{}, false
	}
	level := 0
	for _, char := range trimmed {
		if char != '#' {
			break
		}
		level++
	}
	if level == 0 || level > 6 {
		return markdownHeading{}, false
	}
	rest := trimmed[level:]
	if rest != "" && !strings.HasPrefix(rest, " ") {
		return markdownHeading{}, false
	}
	return markdownHeading{
		index: index,
		level: level,
		line:  trimmed,
		text:  strings.TrimSpace(rest),
	}, true
}

func findMarkdownSectionEnd(lines []string, start int, level int) int {
	for index := start + 1; index < len(lines); index++ {
		heading, ok := parseMarkdownHeading(index, lines[index])
		if ok && heading.level <= level {
			return index
		}
	}
	return len(lines)
}

func contains(values []string, needle string) bool {
	for _, value := range values {
		if value == needle {
			return true
		}
	}
	return false
}

func isRootPath(path string) bool {
	trimmed := strings.TrimSpace(path)
	return trimmed == "" || trimmed == "/"
}

func idFromPath(path string) string {
	trimmed := strings.Trim(strings.TrimSpace(path), "/")
	if trimmed == "" {
		return ""
	}
	parts := strings.Split(trimmed, "/")
	return parts[len(parts)-1]
}

func noteHash(note NoteRecord) string {
	hash := sha256.Sum256([]byte(fmt.Sprintf("%s\x00%s\x00%d", note.Name, note.Content, note.UpdatedAt)))
	return hex.EncodeToString(hash[:])
}

func nowMillis() int64 {
	return time.Now().UnixMilli()
}

func randomToken(length int) string {
	const alphabet = "0123456789abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ"
	out := make([]byte, length)
	max := big.NewInt(int64(len(alphabet)))
	for i := range out {
		n, err := rand.Int(rand.Reader, max)
		if err != nil {
			out[i] = alphabet[int(time.Now().UnixNano())%len(alphabet)]
			continue
		}
		out[i] = alphabet[n.Int64()]
	}
	return string(out)
}

func readMarkdownImport(filePath string, titleHint *string, folderID *string) (string, string, map[string]any, error) {
	path := strings.TrimSpace(filePath)
	if path == "" {
		return "", "", nil, errors.New("Markdown file path cannot be empty")
	}
	bytes, err := os.ReadFile(path)
	if err != nil {
		return "", "", nil, fmt.Errorf("read Markdown file: %w", err)
	}
	content := decodeMarkdownBytes(bytes)
	titleSource := path
	if titleHint != nil && strings.TrimSpace(*titleHint) != "" {
		titleSource = *titleHint
	}
	title := deriveMarkdownTitle(titleSource)
	if isGenericTitle(title) {
		if heading := extractFirstHeading(content); heading != "" {
			title = heading
		} else {
			title = fmt.Sprintf("导入笔记_%s", time.Now().Format("20060102_150405"))
		}
	}
	metadata := map[string]any{}
	if folderID != nil && strings.TrimSpace(*folderID) != "" {
		metadata["folderId"] = strings.TrimSpace(*folderID)
	}
	return content, title, metadata, nil
}

func decodeMarkdownBytes(bytes []byte) string {
	if len(bytes) >= 3 && bytes[0] == 0xEF && bytes[1] == 0xBB && bytes[2] == 0xBF {
		return strings.TrimPrefix(string(bytes[3:]), "\ufeff")
	}
	if len(bytes) >= 2 && bytes[0] == 0xFF && bytes[1] == 0xFE {
		return strings.TrimPrefix(utf16BytesToString(bytes[2:], false), "\ufeff")
	}
	if len(bytes) >= 2 && bytes[0] == 0xFE && bytes[1] == 0xFF {
		return strings.TrimPrefix(utf16BytesToString(bytes[2:], true), "\ufeff")
	}
	return strings.TrimPrefix(string(bytes), "\ufeff")
}

func utf16BytesToString(bytes []byte, bigEndian bool) string {
	words := make([]uint16, 0, len(bytes)/2)
	for index := 0; index+1 < len(bytes); index += 2 {
		if bigEndian {
			words = append(words, uint16(bytes[index])<<8|uint16(bytes[index+1]))
		} else {
			words = append(words, uint16(bytes[index+1])<<8|uint16(bytes[index]))
		}
	}
	return string(utf16.Decode(words))
}

func deriveMarkdownTitle(rawPath string) string {
	base := filepath.Base(strings.TrimSpace(rawPath))
	ext := filepath.Ext(base)
	title := strings.TrimSpace(strings.TrimSuffix(base, ext))
	if title == "" || title == "." || title == string(filepath.Separator) {
		return fmt.Sprintf("导入笔记_%s", time.Now().Format("20060102_150405"))
	}
	return title
}

func extractFirstHeading(content string) string {
	for _, line := range strings.Split(content, "\n") {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "# ") {
			heading := strings.TrimSpace(strings.TrimPrefix(trimmed, "# "))
			if heading != "" {
				return heading
			}
		}
	}
	return ""
}

func isGenericTitle(title string) bool {
	trimmed := strings.TrimSpace(title)
	if trimmed == "" || trimmed == "文件" {
		return true
	}
	allDigits := true
	for _, char := range trimmed {
		if char < '0' || char > '9' {
			allDigits = false
			break
		}
	}
	if allDigits {
		return true
	}
	if colon := strings.LastIndex(trimmed, ":"); colon >= 0 && colon < len(trimmed)-1 {
		for _, char := range trimmed[colon+1:] {
			if char < '0' || char > '9' {
				return false
			}
		}
		return true
	}
	return false
}
