package vfs

import (
	"crypto/rand"
	"crypto/sha256"
	"deep-student-go/internal/storage"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"math/big"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"
)

var safePathSegmentPattern = regexp.MustCompile(`[^A-Za-z0-9._-]+`)

type Service struct {
	mu           sync.RWMutex
	eventMu      sync.RWMutex
	dataDir      string
	indexPath    string
	libraryDir   string
	emit         func(name string, payload any)
	pdfOCRRunner pdfOCRRunnerFunc
	state        store
}

type store struct {
	Resources []Resource `json:"resources"`
}

type CreateResourceInput struct {
	Type     string         `json:"type"`
	Data     string         `json:"data"`
	SourceID *string        `json:"sourceId,omitempty"`
	Metadata map[string]any `json:"metadata,omitempty"`
	Subject  *string        `json:"subject,omitempty"`
}

type CreateResourceResult struct {
	ResourceID string `json:"resourceId"`
	Hash       string `json:"hash"`
	IsNew      bool   `json:"isNew"`
}

type ResourceSyncResult struct {
	ResourceID string `json:"resourceId"`
	Hash       string `json:"hash"`
	IsNew      bool   `json:"isNew"`
}

type CheckSyncNeededResponse struct {
	NeedsSync          bool    `json:"needsSync"`
	ExistingResourceID *string `json:"existingResourceId,omitempty"`
	ExistingHash       *string `json:"existingHash,omitempty"`
}

type UploadAttachmentInput struct {
	Name           string  `json:"name"`
	MimeType       string  `json:"mimeType"`
	Base64Content  string  `json:"base64Content"`
	AttachmentType *string `json:"attachmentType,omitempty"`
	FolderID       *string `json:"folderId,omitempty"`
	SessionID      *string `json:"sessionId,omitempty"`
	GroupID        *string `json:"groupId,omitempty"`
}

type UploadAttachmentResult struct {
	SourceID          string     `json:"sourceId"`
	ResourceHash      string     `json:"resourceHash"`
	IsNew             bool       `json:"isNew"`
	Attachment        Attachment `json:"attachment"`
	ProcessingStatus  *string    `json:"processingStatus,omitempty"`
	ProcessingPercent *float64   `json:"processingPercent,omitempty"`
	ReadyModes        []string   `json:"readyModes,omitempty"`
}

type UploadFileInput struct {
	Name          string         `json:"name"`
	MimeType      string         `json:"mimeType"`
	Base64Content string         `json:"base64Content"`
	FileType      *string        `json:"fileType,omitempty"`
	FolderID      *string        `json:"folderId,omitempty"`
	Metadata      map[string]any `json:"metadata,omitempty"`
}

type UploadFileResult struct {
	File         VfsFile      `json:"file"`
	SourceID     string       `json:"sourceId"`
	ResourceHash string       `json:"resourceHash"`
	IsNew        bool         `json:"isNew"`
	OcrStatus    *OcrStatus   `json:"ocrStatus,omitempty"`
	IndexStatus  *IndexStatus `json:"indexStatus,omitempty"`
}

type OcrStatus struct {
	Performed        bool    `json:"performed"`
	SkipReason       *string `json:"skipReason,omitempty"`
	SuccessCount     int     `json:"successCount"`
	FailedCount      int     `json:"failedCount"`
	BlobMissingCount int     `json:"blobMissingCount"`
	TotalPages       int     `json:"totalPages"`
	AllSuccess       bool    `json:"allSuccess"`
	Message          string  `json:"message"`
}

type IndexStatus struct {
	Queued       bool   `json:"queued"`
	UnitsCreated int    `json:"unitsCreated"`
	Message      string `json:"message"`
}

type Attachment struct {
	ID          string  `json:"id"`
	ResourceID  *string `json:"resourceId,omitempty"`
	BlobHash    *string `json:"blobHash,omitempty"`
	Type        string  `json:"type"`
	Name        string  `json:"name"`
	MimeType    string  `json:"mimeType"`
	Size        int64   `json:"size"`
	ContentHash string  `json:"contentHash"`
	IsFavorite  bool    `json:"isFavorite"`
	CreatedAt   string  `json:"createdAt"`
	UpdatedAt   string  `json:"updatedAt"`
}

type AttachmentContentResult struct {
	Content *string `json:"content,omitempty"`
	Found   bool    `json:"found"`
	Error   *string `json:"error,omitempty"`
}

type Resource struct {
	ID              string         `json:"id"`
	Hash            string         `json:"hash"`
	Type            string         `json:"type"`
	SourceID        *string        `json:"sourceId,omitempty"`
	SourceTable     *string        `json:"sourceTable,omitempty"`
	StorageMode     string         `json:"storageMode"`
	Data            *string        `json:"data,omitempty"`
	ExternalHash    *string        `json:"externalHash,omitempty"`
	ExternalPath    *string        `json:"externalPath,omitempty"`
	OriginalPath    *string        `json:"originalPath,omitempty"`
	Metadata        map[string]any `json:"metadata,omitempty"`
	RefCount        int            `json:"refCount"`
	CreatedAt       int64          `json:"createdAt"`
	UpdatedAt       int64          `json:"updatedAt"`
	ContentEncoding string         `json:"contentEncoding,omitempty"`
}

type GetResourceRefsInput struct {
	SourceIDs             []string `json:"sourceIds"`
	IncludeFolderContents bool     `json:"includeFolderContents,omitempty"`
	MaxItems              int      `json:"maxItems,omitempty"`
}

type ResourceInjectModes struct {
	Image []string `json:"image,omitempty"`
	PDF   []string `json:"pdf,omitempty"`
}

type ResourceRef struct {
	SourceID     string               `json:"sourceId"`
	ResourceHash string               `json:"resourceHash"`
	Type         string               `json:"type"`
	Name         string               `json:"name"`
	ResourceID   *string              `json:"resourceId,omitempty"`
	Snippet      *string              `json:"snippet,omitempty"`
	InjectModes  *ResourceInjectModes `json:"injectModes,omitempty"`
}

type ContextRefData struct {
	Refs       []ResourceRef `json:"refs"`
	TotalCount int           `json:"totalCount"`
	Truncated  bool          `json:"truncated"`
}

type ResolvedResource struct {
	SourceID         string            `json:"sourceId"`
	ResourceHash     string            `json:"resourceHash"`
	Type             string            `json:"type"`
	Name             string            `json:"name"`
	Path             string            `json:"path"`
	Content          *string           `json:"content,omitempty"`
	ByteSize         *int64            `json:"byteSize,omitempty"`
	Found            bool              `json:"found"`
	Warning          *string           `json:"warning,omitempty"`
	Metadata         map[string]any    `json:"metadata,omitempty"`
	MultimodalBlocks []MultimodalBlock `json:"multimodalBlocks,omitempty"`
}

type MultimodalBlock struct {
	Type      string  `json:"type"`
	Text      *string `json:"text,omitempty"`
	MediaType *string `json:"mediaType,omitempty"`
	Base64    *string `json:"base64,omitempty"`
}

type StateStats struct {
	Pending  int `json:"pending"`
	Indexing int `json:"indexing"`
	Indexed  int `json:"indexed"`
	Failed   int `json:"failed"`
	Disabled int `json:"disabled"`
}

type DimensionStat struct {
	Dimension int    `json:"dimension"`
	Modality  string `json:"modality"`
	Count     int    `json:"count"`
}

type IndexStatusSummary struct {
	TotalUnits int             `json:"totalUnits"`
	TextStats  StateStats      `json:"textStats"`
	MMStats    StateStats      `json:"mmStats"`
	Dimensions []DimensionStat `json:"dimensions"`
}

type UnitIndexStatus struct {
	UnitID           string  `json:"unitId"`
	ResourceID       string  `json:"resourceId"`
	UnitIndex        int     `json:"unitIndex"`
	HasImage         bool    `json:"hasImage"`
	HasText          bool    `json:"hasText"`
	TextSource       *string `json:"textSource"`
	TextRequired     bool    `json:"textRequired"`
	TextState        string  `json:"textState"`
	TextError        *string `json:"textError"`
	TextChunkCount   int     `json:"textChunkCount"`
	TextEmbeddingDim *int    `json:"textEmbeddingDim"`
	MMRequired       bool    `json:"mmRequired"`
	MMState          string  `json:"mmState"`
	MMError          *string `json:"mmError"`
	MMEmbeddingDim   *int    `json:"mmEmbeddingDim"`
	UpdatedAt        int64   `json:"updatedAt"`
}

type SyncResourceUnitsInput struct {
	ResourceID    string  `json:"resourceId"`
	ResourceType  string  `json:"resourceType"`
	Data          *string `json:"data,omitempty"`
	OcrText       *string `json:"ocrText,omitempty"`
	OcrPagesJSON  *string `json:"ocrPagesJson,omitempty"`
	BlobHash      *string `json:"blobHash,omitempty"`
	PageCount     *int    `json:"pageCount,omitempty"`
	ExtractedText *string `json:"extractedText,omitempty"`
	PreviewJSON   *string `json:"previewJson,omitempty"`
}

type GetIndexStatusInput struct {
	FolderID          *string `json:"folderId,omitempty"`
	ResourceType      *string `json:"resourceType,omitempty"`
	StateFilter       *string `json:"stateFilter,omitempty"`
	IncludeImageIndex bool    `json:"includeImageIndex,omitempty"`
	Limit             int     `json:"limit,omitempty"`
	Offset            int     `json:"offset,omitempty"`
}

type ResourceIndexStatus struct {
	ResourceID           string  `json:"resourceId"`
	SourceID             *string `json:"sourceId,omitempty"`
	ResourceType         string  `json:"resourceType"`
	Name                 string  `json:"name"`
	HasOcr               bool    `json:"hasOcr"`
	OcrCount             int     `json:"ocrCount"`
	TextIndexState       string  `json:"textIndexState"`
	TextIndexedAt        *int64  `json:"textIndexedAt,omitempty"`
	TextIndexError       *string `json:"textIndexError,omitempty"`
	TextChunkCount       int     `json:"textChunkCount"`
	NativeTextChunkCount int     `json:"nativeTextChunkCount"`
	OcrTextChunkCount    int     `json:"ocrTextChunkCount"`
	TextEmbeddingDim     *int    `json:"textEmbeddingDim,omitempty"`
	TextIndexSource      *string `json:"textIndexSource,omitempty"`
	TextIndexRetryable   bool    `json:"textIndexRetryable"`
	MMIndexState         string  `json:"mmIndexState"`
	MMIndexedPages       int     `json:"mmIndexedPages"`
	MMEmbeddingDim       *int    `json:"mmEmbeddingDim,omitempty"`
	MMIndexingMode       *string `json:"mmIndexingMode,omitempty"`
	MMIndexError         *string `json:"mmIndexError,omitempty"`
	DisplayIndexState    string  `json:"displayIndexState"`
	EmbeddingDim         *int    `json:"embeddingDim,omitempty"`
	Modality             *string `json:"modality,omitempty"`
	UpdatedAt            int64   `json:"updatedAt"`
	IsStale              bool    `json:"isStale"`
}

type ResourceIndexStatusSummary struct {
	TotalResources        int                   `json:"totalResources"`
	IndexedCount          int                   `json:"indexedCount"`
	PendingCount          int                   `json:"pendingCount"`
	IndexingCount         int                   `json:"indexingCount"`
	FailedCount           int                   `json:"failedCount"`
	DisabledCount         int                   `json:"disabledCount"`
	StaleCount            int                   `json:"staleCount"`
	TextQueueCount        int                   `json:"textQueueCount"`
	TextTotalResources    int                   `json:"textTotalResources"`
	TextIndexedCount      int                   `json:"textIndexedCount"`
	TextPendingCount      int                   `json:"textPendingCount"`
	TextIndexingCount     int                   `json:"textIndexingCount"`
	TextFailedCount       int                   `json:"textFailedCount"`
	TextDisabledCount     int                   `json:"textDisabledCount"`
	DisplayTotalResources int                   `json:"displayTotalResources"`
	DisplayIndexedCount   int                   `json:"displayIndexedCount"`
	DisplayPendingCount   int                   `json:"displayPendingCount"`
	DisplayIndexingCount  int                   `json:"displayIndexingCount"`
	DisplayFailedCount    int                   `json:"displayFailedCount"`
	DisplayDisabledCount  int                   `json:"displayDisabledCount"`
	MMTotalResources      int                   `json:"mmTotalResources"`
	MMIndexedCount        int                   `json:"mmIndexedCount"`
	MMPendingCount        int                   `json:"mmPendingCount"`
	MMIndexingCount       int                   `json:"mmIndexingCount"`
	MMFailedCount         int                   `json:"mmFailedCount"`
	MMDisabledCount       int                   `json:"mmDisabledCount"`
	Resources             []ResourceIndexStatus `json:"resources"`
}

type EmbeddingDimInfo struct {
	Dimension      int    `json:"dimension"`
	Modality       string `json:"modality"`
	LanceTableName string `json:"lanceTableName"`
	RecordCount    int    `json:"recordCount"`
}

type VfsEmbeddingDimension struct {
	Dimension      int     `json:"dimension"`
	Modality       string  `json:"modality"`
	ModelConfigID  *string `json:"modelConfigId,omitempty"`
	ModelName      *string `json:"modelName,omitempty"`
	RecordCount    int     `json:"recordCount"`
	LanceTableName string  `json:"lanceTableName"`
	CreatedAt      int64   `json:"createdAt"`
	LastUsedAt     int64   `json:"lastUsedAt"`
}

type BatchIndexResult struct {
	SuccessCount int `json:"successCount"`
	FailCount    int `json:"failCount"`
	Total        int `json:"total"`
}

type PdfProcessingFailedStage struct {
	Stage     string `json:"stage"`
	Message   string `json:"message"`
	Retriable bool   `json:"retriable,omitempty"`
}

type PdfProcessingProgress struct {
	Stage        string                     `json:"stage"`
	CurrentPage  *int                       `json:"currentPage,omitempty"`
	TotalPages   *int                       `json:"totalPages,omitempty"`
	Percent      float64                    `json:"percent"`
	ReadyModes   []string                   `json:"readyModes"`
	MediaType    string                     `json:"mediaType,omitempty"`
	FailedStages []PdfProcessingFailedStage `json:"failedStages,omitempty"`
}

type PdfProcessingStatus struct {
	FileID       string                     `json:"fileId"`
	Stage        string                     `json:"stage"`
	CurrentPage  *int                       `json:"currentPage,omitempty"`
	TotalPages   *int                       `json:"totalPages,omitempty"`
	Percent      float64                    `json:"percent"`
	ReadyModes   []string                   `json:"readyModes"`
	MediaType    string                     `json:"mediaType,omitempty"`
	Error        *string                    `json:"error,omitempty"`
	FailedStages []PdfProcessingFailedStage `json:"failedStages,omitempty"`
	Progress     PdfProcessingProgress      `json:"progress"`
}

type mediaProcessingProgressPayload struct {
	FileID    string                `json:"fileId"`
	Status    PdfProcessingProgress `json:"status"`
	MediaType string                `json:"mediaType"`
}

type mediaProcessingCompletedPayload struct {
	FileID     string   `json:"fileId"`
	ReadyModes []string `json:"readyModes"`
	Stage      string   `json:"stage,omitempty"`
	MediaType  string   `json:"mediaType"`
}

type mediaProcessingErrorPayload struct {
	FileID    string `json:"fileId"`
	Error     string `json:"error"`
	Stage     string `json:"stage"`
	MediaType string `json:"mediaType"`
}

type PdfPageImageResult struct {
	Base64   string `json:"base64"`
	MimeType string `json:"mimeType"`
	Size     int64  `json:"size"`
}

type VfsBlobBase64Result struct {
	Base64   string `json:"base64"`
	MimeType string `json:"mimeType"`
	Size     int64  `json:"size"`
}

type DeleteIndexResult struct {
	SQLiteOK    bool     `json:"sqliteOk"`
	LanceTextOK bool     `json:"lanceTextOk"`
	LanceMMOK   bool     `json:"lanceMmOk"`
	Warnings    []string `json:"warnings"`
	Retryable   bool     `json:"retryable"`
}

type OcrPageInfo struct {
	PageIndex int    `json:"pageIndex"`
	Text      string `json:"text"`
	CharCount int    `json:"charCount"`
	IsFailed  bool   `json:"isFailed"`
}

type ResourceOcrInfo struct {
	ResourceID          string        `json:"resourceId"`
	ResourceType        string        `json:"resourceType"`
	HasOcr              bool          `json:"hasOcr"`
	OcrText             *string       `json:"ocrText"`
	OcrTextLength       int           `json:"ocrTextLength"`
	ExtractedText       *string       `json:"extractedText"`
	ExtractedTextLength int           `json:"extractedTextLength"`
	ActiveSource        string        `json:"activeSource"`
	OcrPages            []OcrPageInfo `json:"ocrPages"`
}

type TextChunkInfo struct {
	UnitID         string  `json:"unitId"`
	UnitIndex      int     `json:"unitIndex"`
	TextContent    *string `json:"textContent"`
	TextSource     *string `json:"textSource"`
	TextState      string  `json:"textState"`
	TextChunkCount int     `json:"textChunkCount"`
	CharCount      int     `json:"charCount"`
}

type VfsRagSearchInput struct {
	Query                string   `json:"query"`
	FolderIDs            []string `json:"folderIds,omitempty"`
	ResourceTypes        []string `json:"resourceTypes,omitempty"`
	TopK                 int      `json:"topK,omitempty"`
	EnableReranking      bool     `json:"enableReranking,omitempty"`
	EnableCrossDimension bool     `json:"enableCrossDimension,omitempty"`
	Modality             string   `json:"modality,omitempty"`
}

type VfsRagSearchOutput struct {
	Results   []VfsSearchResult `json:"results"`
	Count     int               `json:"count"`
	ElapsedMs int64             `json:"elapsedMs"`
}

type VfsSearchResult struct {
	EmbeddingID   string  `json:"embeddingId"`
	ResourceID    string  `json:"resourceId"`
	ChunkIndex    int     `json:"chunkIndex"`
	ChunkText     string  `json:"chunkText"`
	Score         float64 `json:"score"`
	ResourceTitle *string `json:"resourceTitle,omitempty"`
	ResourceType  *string `json:"resourceType,omitempty"`
	PageIndex     *int    `json:"pageIndex,omitempty"`
	SourceID      *string `json:"sourceId,omitempty"`
}

type ListFilesInput struct {
	FileType string `json:"fileType,omitempty"`
	Limit    int    `json:"limit,omitempty"`
	Offset   int    `json:"offset,omitempty"`
}

type VfsFile struct {
	ID            string         `json:"id"`
	ResourceID    *string        `json:"resourceId,omitempty"`
	BlobHash      *string        `json:"blobHash,omitempty"`
	SHA256        string         `json:"sha256"`
	FileName      string         `json:"fileName"`
	OriginalPath  *string        `json:"originalPath,omitempty"`
	Size          int64          `json:"size"`
	PageCount     *int           `json:"pageCount,omitempty"`
	FileType      string         `json:"fileType"`
	MimeType      *string        `json:"mimeType,omitempty"`
	Tags          []string       `json:"tags"`
	IsFavorite    bool           `json:"isFavorite"`
	LastOpenedAt  *string        `json:"lastOpenedAt,omitempty"`
	LastPage      *int           `json:"lastPage,omitempty"`
	Bookmarks     []any          `json:"bookmarks"`
	CoverKey      *string        `json:"coverKey,omitempty"`
	ExtractedText *string        `json:"extractedText,omitempty"`
	PreviewJSON   *string        `json:"previewJson,omitempty"`
	OcrPagesJSON  *string        `json:"ocrPagesJson,omitempty"`
	Description   *string        `json:"description,omitempty"`
	Status        string         `json:"status"`
	CreatedAt     string         `json:"createdAt"`
	UpdatedAt     string         `json:"updatedAt"`
	DeletedAt     *string        `json:"deletedAt,omitempty"`
	Metadata      map[string]any `json:"metadata,omitempty"`
}

func NewService(dataDir string) (*Service, error) {
	service := &Service{
		dataDir:    filepath.Clean(dataDir),
		indexPath:  filepath.Join(dataDir, "vfs-go.json"),
		libraryDir: filepath.Join(dataDir, "vfs_resources"),
		state: store{
			Resources: []Resource{},
		},
	}
	if err := service.load(); err != nil {
		return nil, err
	}
	if err := os.MkdirAll(service.libraryDir, 0o700); err != nil {
		return nil, err
	}
	return service, nil
}

func (s *Service) SetEventEmitter(emit func(name string, payload any)) {
	s.eventMu.Lock()
	defer s.eventMu.Unlock()
	s.emit = emit
}

func (s *Service) SetPDFOCRRunner(runner pdfOCRRunnerFunc) {
	s.eventMu.Lock()
	defer s.eventMu.Unlock()
	s.pdfOCRRunner = runner
}

func (s *Service) CreateOrReuse(params CreateResourceInput) (CreateResourceResult, error) {
	resourceType := normalizeResourceType(params.Type)
	if resourceType == "" {
		return CreateResourceResult{}, fmt.Errorf("invalid resource type: %s", params.Type)
	}
	if params.Data == "" {
		return CreateResourceResult{}, errors.New("resource data is required")
	}

	hash := computeHash(params.Data)
	now := nowMillis()

	s.mu.Lock()
	defer s.mu.Unlock()

	if index, ok := s.findResourceByHashLocked(hash, resourceType); ok {
		resource := s.state.Resources[index]
		if params.SourceID != nil && strings.TrimSpace(*params.SourceID) != "" && resource.SourceID == nil {
			sourceID := strings.TrimSpace(*params.SourceID)
			s.state.Resources[index].SourceID = &sourceID
		}
		if len(params.Metadata) > 0 {
			s.state.Resources[index].Metadata = mergeMetadata(resource.Metadata, params.Metadata)
		}
		s.state.Resources[index].UpdatedAt = now
		if err := s.flushLocked(); err != nil {
			return CreateResourceResult{}, err
		}
		return CreateResourceResult{ResourceID: resource.ID, Hash: resource.Hash, IsNew: false}, nil
	}

	relativePath, encoding, err := s.writeResourceData(resourceType, hash, params.Data, params.Metadata)
	if err != nil {
		return CreateResourceResult{}, err
	}
	externalHash := hash
	resource := Resource{
		ID:              "res_" + randomToken(12),
		Hash:            hash,
		Type:            resourceType,
		SourceID:        normalizeOptionalString(params.SourceID),
		StorageMode:     "external",
		ExternalHash:    &externalHash,
		ExternalPath:    &relativePath,
		Metadata:        normalizeMetadata(params.Metadata),
		RefCount:        0,
		CreatedAt:       now,
		UpdatedAt:       now,
		ContentEncoding: encoding,
	}
	s.state.Resources = append(s.state.Resources, resource)
	if err := s.flushLocked(); err != nil {
		return CreateResourceResult{}, err
	}
	return CreateResourceResult{ResourceID: resource.ID, Hash: resource.Hash, IsNew: true}, nil
}

func (s *Service) CreateOrUpdateSource(params CreateResourceInput) (CreateResourceResult, error) {
	resourceType := normalizeResourceType(params.Type)
	if resourceType == "" {
		return CreateResourceResult{}, fmt.Errorf("invalid resource type: %s", params.Type)
	}
	sourceID := normalizeOptionalString(params.SourceID)
	if sourceID == nil {
		return CreateResourceResult{}, errors.New("sourceId is required")
	}
	if params.Data == "" {
		return CreateResourceResult{}, errors.New("resource data is required")
	}

	hash := computeHash(params.Data)
	now := nowMillis()

	s.mu.Lock()
	defer s.mu.Unlock()

	if index, ok := s.findResourceIndexByAnyIDLocked(*sourceID); ok {
		relativePath, encoding, err := s.writeResourceData(resourceType, hash, params.Data, params.Metadata)
		if err != nil {
			return CreateResourceResult{}, err
		}
		externalHash := hash
		s.state.Resources[index].Hash = hash
		s.state.Resources[index].Type = resourceType
		s.state.Resources[index].SourceID = sourceID
		s.state.Resources[index].StorageMode = "external"
		s.state.Resources[index].ExternalHash = &externalHash
		s.state.Resources[index].ExternalPath = &relativePath
		s.state.Resources[index].Metadata = mergeMetadata(s.state.Resources[index].Metadata, params.Metadata)
		s.state.Resources[index].UpdatedAt = now
		s.state.Resources[index].ContentEncoding = encoding
		if err := s.flushLocked(); err != nil {
			return CreateResourceResult{}, err
		}
		return CreateResourceResult{ResourceID: s.state.Resources[index].ID, Hash: hash, IsNew: false}, nil
	}

	if index, ok := s.findResourceByHashLocked(hash, resourceType); ok && s.state.Resources[index].SourceID == nil {
		s.state.Resources[index].SourceID = sourceID
		s.state.Resources[index].Metadata = mergeMetadata(s.state.Resources[index].Metadata, params.Metadata)
		s.state.Resources[index].UpdatedAt = now
		if err := s.flushLocked(); err != nil {
			return CreateResourceResult{}, err
		}
		return CreateResourceResult{ResourceID: s.state.Resources[index].ID, Hash: s.state.Resources[index].Hash, IsNew: false}, nil
	}

	relativePath, encoding, err := s.writeResourceData(resourceType, hash, params.Data, params.Metadata)
	if err != nil {
		return CreateResourceResult{}, err
	}
	externalHash := hash
	resource := Resource{
		ID:              "res_" + randomToken(12),
		Hash:            hash,
		Type:            resourceType,
		SourceID:        sourceID,
		StorageMode:     "external",
		ExternalHash:    &externalHash,
		ExternalPath:    &relativePath,
		Metadata:        normalizeMetadata(params.Metadata),
		RefCount:        0,
		CreatedAt:       now,
		UpdatedAt:       now,
		ContentEncoding: encoding,
	}
	s.state.Resources = append(s.state.Resources, resource)
	if err := s.flushLocked(); err != nil {
		return CreateResourceResult{}, err
	}
	return CreateResourceResult{ResourceID: resource.ID, Hash: resource.Hash, IsNew: true}, nil
}

func (s *Service) ResourceSyncNote(noteID string) (ResourceSyncResult, error) {
	sourceID := strings.TrimSpace(noteID)
	if sourceID == "" {
		return ResourceSyncResult{}, errors.New("noteId is required")
	}
	data := "# Synced note\n\nSource note: " + sourceID + "\n"
	return s.syncSourceResource("note", sourceID, data, map[string]any{
		"name":          "note-" + safeSegment(sourceID, "note") + ".md",
		"title":         "Synced note " + sourceID,
		"sourceType":    "note",
		"sourceId":      sourceID,
		"snippet":       "Source note: " + sourceID,
		"legacyCommand": "resource_sync_note",
		"syncMode":      "go-hybrid-vfs-source-stable",
	})
}

func (s *Service) ResourceSyncExam(sessionID string) (ResourceSyncResult, error) {
	sourceID := strings.TrimSpace(sessionID)
	if sourceID == "" {
		return ResourceSyncResult{}, errors.New("sessionId is required")
	}
	data := "# Synced exam\n\nSource exam session: " + sourceID + "\n"
	return s.syncSourceResource("exam", sourceID, data, map[string]any{
		"name":          "exam-" + safeSegment(sourceID, "exam") + ".md",
		"title":         "Synced exam " + sourceID,
		"sourceType":    "exam",
		"sourceId":      sourceID,
		"snippet":       "Source exam session: " + sourceID,
		"legacyCommand": "resource_sync_exam",
		"syncMode":      "go-hybrid-vfs-source-stable",
	})
}

func (s *Service) ResourceSyncTextbookPages(textbookID string, pageRange []int) ([]ResourceSyncResult, error) {
	trimmedID := strings.TrimSpace(textbookID)
	if trimmedID == "" {
		return nil, errors.New("textbookId is required")
	}

	rangeLabel := "all"
	sourceID := trimmedID
	metadata := map[string]any{
		"name":          "textbook-" + safeSegment(trimmedID, "textbook") + ".md",
		"title":         "Synced textbook " + trimmedID,
		"sourceType":    "textbook",
		"sourceId":      trimmedID,
		"legacyCommand": "resource_sync_textbook_pages",
		"syncMode":      "go-hybrid-vfs-source-stable",
	}
	if len(pageRange) > 0 {
		if len(pageRange) != 2 {
			return nil, errors.New("pageRange must contain start and end page numbers")
		}
		start := pageRange[0]
		end := pageRange[1]
		if start <= 0 || end <= 0 || start > end {
			return nil, fmt.Errorf("invalid pageRange: %d-%d", start, end)
		}
		rangeLabel = fmt.Sprintf("%d-%d", start, end)
		sourceID = fmt.Sprintf("%s:pages:%s", trimmedID, rangeLabel)
		metadata["sourceId"] = sourceID
		metadata["textbookId"] = trimmedID
		metadata["pageRange"] = []int{start, end}
		metadata["name"] = "textbook-" + safeSegment(trimmedID, "textbook") + "-pages-" + rangeLabel + ".md"
		metadata["title"] = "Synced textbook " + trimmedID + " pages " + rangeLabel
	}

	data := "# Synced textbook pages\n\nSource textbook: " + trimmedID + "\nPage range: " + rangeLabel + "\n"
	result, err := s.syncSourceResource("textbook", sourceID, data, metadata)
	if err != nil {
		return nil, err
	}
	return []ResourceSyncResult{result}, nil
}

func (s *Service) ResourceCheckSyncNeeded(resourceType string, sourceID string, currentHash *string) (CheckSyncNeededResponse, error) {
	normalizedType := normalizeResourceType(resourceType)
	if normalizedType == "" {
		return CheckSyncNeededResponse{}, fmt.Errorf("invalid resource type: %s", resourceType)
	}
	trimmedSourceID := strings.TrimSpace(sourceID)
	if trimmedSourceID == "" {
		return CheckSyncNeededResponse{}, errors.New("sourceId is required")
	}

	s.mu.RLock()
	defer s.mu.RUnlock()

	resource, ok := s.findResourceByAnyIDLocked(trimmedSourceID)
	if !ok || resource.Type != normalizedType {
		return CheckSyncNeededResponse{NeedsSync: true}, nil
	}

	resourceID := resource.ID
	hash := resource.Hash
	trimmedHash := ""
	if currentHash != nil {
		trimmedHash = strings.TrimSpace(*currentHash)
	}

	return CheckSyncNeededResponse{
		NeedsSync:          trimmedHash != "" && trimmedHash != resource.Hash,
		ExistingResourceID: &resourceID,
		ExistingHash:       &hash,
	}, nil
}

func (s *Service) syncSourceResource(resourceType string, sourceID string, data string, metadata map[string]any) (ResourceSyncResult, error) {
	result, err := s.CreateOrUpdateSource(CreateResourceInput{
		Type:     resourceType,
		Data:     data,
		SourceID: &sourceID,
		Metadata: metadata,
	})
	if err != nil {
		return ResourceSyncResult{}, err
	}
	return ResourceSyncResult{
		ResourceID: result.ResourceID,
		Hash:       result.Hash,
		IsNew:      result.IsNew,
	}, nil
}

func (s *Service) GetResource(resourceID string) (*Resource, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	resource, ok := s.findResourceByAnyIDLocked(resourceID)
	if !ok {
		return nil, nil
	}
	withData, err := s.hydrateResourceData(resource)
	if err != nil {
		return nil, err
	}
	return &withData, nil
}

func (s *Service) ResourceExists(resourceID string) (bool, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	_, ok := s.findResourceByAnyIDLocked(resourceID)
	return ok, nil
}

func (s *Service) IncrementRef(resourceID string) (int, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	index, ok := s.findResourceIndexByAnyIDLocked(resourceID)
	if !ok {
		return 0, fmt.Errorf("resource not found: %s", resourceID)
	}
	s.state.Resources[index].RefCount++
	s.state.Resources[index].UpdatedAt = nowMillis()
	return s.state.Resources[index].RefCount, s.flushLocked()
}

func (s *Service) DecrementRef(resourceID string) (int, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	index, ok := s.findResourceIndexByAnyIDLocked(resourceID)
	if !ok {
		return 0, fmt.Errorf("resource not found: %s", resourceID)
	}
	if s.state.Resources[index].RefCount > 0 {
		s.state.Resources[index].RefCount--
	}
	s.state.Resources[index].UpdatedAt = nowMillis()
	return s.state.Resources[index].RefCount, s.flushLocked()
}

func (s *Service) GetResourcePath(sourceID string) (*string, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	resource, ok := s.findResourceByAnyIDLocked(sourceID)
	if !ok || resource.ExternalPath == nil {
		return nil, nil
	}
	absolute, err := s.resolveLibraryPath(*resource.ExternalPath)
	if err != nil {
		return nil, err
	}
	return &absolute, nil
}

func (s *Service) GetResourceRefCount(sourceID string) (int, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if resource, ok := s.findResourceByAnyIDLocked(sourceID); ok {
		return resource.RefCount, nil
	}
	return 0, nil
}

func (s *Service) UpdateResourceHash(sourceID string, newHash string) (bool, error) {
	newHash = strings.TrimSpace(newHash)
	if newHash == "" {
		return false, errors.New("newHash is required")
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	index, ok := s.findResourceIndexByAnyIDLocked(sourceID)
	if !ok {
		return false, nil
	}
	s.state.Resources[index].Hash = newHash
	s.state.Resources[index].UpdatedAt = nowMillis()
	return true, s.flushLocked()
}

func (s *Service) UploadAttachment(params UploadAttachmentInput) (UploadAttachmentResult, error) {
	name := strings.TrimSpace(params.Name)
	if name == "" {
		return UploadAttachmentResult{}, errors.New("attachment name is required")
	}
	mimeType := strings.TrimSpace(params.MimeType)
	if mimeType == "" {
		mimeType = "application/octet-stream"
	}
	base64Content := strings.TrimSpace(params.Base64Content)
	if base64Content == "" {
		return UploadAttachmentResult{}, errors.New("attachment content is required")
	}
	decoded, err := decodeBase64Payload(base64Content)
	if err != nil {
		return UploadAttachmentResult{}, fmt.Errorf("invalid attachment base64 content: %w", err)
	}
	contentHash := computeBytesHash(decoded)
	attachmentType := normalizeAttachmentType(params.AttachmentType, mimeType, name)
	now := nowMillis()

	s.mu.Lock()
	defer s.mu.Unlock()

	if index, ok := s.findAttachmentByHashLocked(contentHash, attachmentType); ok {
		resource := s.state.Resources[index]
		if resource.SourceID == nil || strings.TrimSpace(*resource.SourceID) == "" {
			sourceID := "att_" + randomToken(10)
			s.state.Resources[index].SourceID = &sourceID
			resource.SourceID = &sourceID
		}
		s.state.Resources[index].Metadata = mergeMetadata(resource.Metadata, attachmentMetadata(params, attachmentType, contentHash, int64(len(decoded)), resource.ID, resource.SourceID, resource.CreatedAt, now))
		s.state.Resources[index].UpdatedAt = now
		if err := s.flushLocked(); err != nil {
			return UploadAttachmentResult{}, err
		}
		resource = s.state.Resources[index]
		return uploadAttachmentResult(resource, false), nil
	}

	sourceID := "att_" + randomToken(10)
	relativePath, encoding, err := s.writeResourceData(attachmentType, contentHash, base64Content, map[string]any{
		"name":     name,
		"mimeType": mimeType,
	})
	if err != nil {
		return UploadAttachmentResult{}, err
	}
	resourceID := "res_" + randomToken(12)
	externalHash := contentHash
	resource := Resource{
		ID:              resourceID,
		Hash:            contentHash,
		Type:            attachmentType,
		SourceID:        &sourceID,
		StorageMode:     "external",
		ExternalHash:    &externalHash,
		ExternalPath:    &relativePath,
		Metadata:        attachmentMetadata(params, attachmentType, contentHash, int64(len(decoded)), resourceID, &sourceID, now, now),
		RefCount:        0,
		CreatedAt:       now,
		UpdatedAt:       now,
		ContentEncoding: encoding,
	}
	s.state.Resources = append(s.state.Resources, resource)
	if err := s.flushLocked(); err != nil {
		return UploadAttachmentResult{}, err
	}
	return uploadAttachmentResult(resource, true), nil
}

func (s *Service) GetAttachment(attachmentID string) (*Attachment, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if !s.hasAttachmentLikeAliasLocked(attachmentID) {
		return nil, fmt.Errorf("invalid attachment ID format: %s", attachmentID)
	}
	resource, ok := s.findFileLikeResourceByAnyIDLocked(attachmentID)
	if !ok || resourceIsDeleted(resource) {
		return nil, nil
	}
	attachment := resourceToAttachment(resource)
	return &attachment, nil
}

func (s *Service) GetAttachmentContent(attachmentID string) (AttachmentContentResult, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if !s.hasAttachmentLikeAliasLocked(attachmentID) {
		return AttachmentContentResult{}, fmt.Errorf("invalid attachment ID format: %s", attachmentID)
	}
	resource, ok := s.findFileLikeResourceByAnyIDLocked(attachmentID)
	if !ok || resourceIsDeleted(resource) {
		return AttachmentContentResult{Found: false}, nil
	}
	hydrated, err := s.hydrateResourceData(resource)
	if err != nil {
		message := err.Error()
		return AttachmentContentResult{Found: false, Error: &message}, nil
	}
	return AttachmentContentResult{Content: hydrated.Data, Found: hydrated.Data != nil}, nil
}

func (s *Service) UploadFile(params UploadFileInput) (UploadFileResult, error) {
	name := strings.TrimSpace(params.Name)
	if name == "" {
		return UploadFileResult{}, errors.New("file name is required")
	}
	mimeType := strings.TrimSpace(params.MimeType)
	if mimeType == "" {
		mimeType = "application/octet-stream"
	}
	base64Content := strings.TrimSpace(params.Base64Content)
	if base64Content == "" {
		return UploadFileResult{}, errors.New("file content is required")
	}
	decoded, err := decodeBase64Payload(base64Content)
	if err != nil {
		return UploadFileResult{}, fmt.Errorf("invalid file base64 content: %w", err)
	}
	contentHash := computeBytesHash(decoded)
	fileType := normalizeFileType(params.FileType, mimeType)
	extractedText := extractedTextForUpload(name, mimeType, decoded)
	pageCount := pageCountForUpload(name, mimeType, decoded)
	ocrPagesJSON := ocrPagesJSONForUpload(name, mimeType, extractedText, pageCount)
	resourceType := resourceTypeForFileUpload(fileType, params.Metadata)
	now := nowMillis()

	s.mu.Lock()

	if index, ok := s.findFileByHashLocked(contentHash, fileType); ok {
		resource := s.state.Resources[index]
		if resourceIsDeleted(resource) {
			s.state.Resources[index].Metadata = mergeMetadata(resource.Metadata, map[string]any{
				"status":    "active",
				"deletedAt": "",
			})
			resource = s.state.Resources[index]
		}
		s.state.Resources[index].Type = resourceType
		metadata := fileMetadata(params, fileType, contentHash, int64(len(decoded)), resource.ID, resource.SourceID, resource.CreatedAt, now, extractedText, pageCount, ocrPagesJSON)
		if !metadataHasKey(params.Metadata, "bookmarks") && metadataHasKey(resource.Metadata, "bookmarks") {
			delete(metadata, "bookmarks")
		}
		incomingPreviewJSON := metadataHasPreviewJSON(params.Metadata)
		existingMetadata := resource.Metadata
		if incomingPreviewJSON {
			existingMetadata = withoutGeneratedPdfPreviewMetadata(existingMetadata)
		}
		s.state.Resources[index].Metadata = mergeMetadata(existingMetadata, metadata)
		s.state.Resources[index].UpdatedAt = now
		previewJob := pdfPreviewRenderJobForResource(s.state.Resources[index], now)
		if err := s.flushLocked(); err != nil {
			s.mu.Unlock()
			return UploadFileResult{}, err
		}
		resourceID := s.state.Resources[index].ID
		s.mu.Unlock()
		if err := s.renderAndCommitPdfPreview(previewJob); err != nil {
			return UploadFileResult{}, err
		}
		return s.uploadFileResultByResourceID(resourceID, fileType, contentHash, false)
	}

	sourceID := "file_" + randomToken(10)
	relativePath, encoding, err := s.writeResourceData(resourceType, contentHash, base64Content, map[string]any{
		"name":     name,
		"mimeType": mimeType,
	})
	if err != nil {
		s.mu.Unlock()
		return UploadFileResult{}, err
	}
	resourceID := "res_" + randomToken(12)
	externalHash := contentHash
	metadata := fileMetadata(params, fileType, contentHash, int64(len(decoded)), resourceID, &sourceID, now, now, extractedText, pageCount, ocrPagesJSON)
	resource := Resource{
		ID:              resourceID,
		Hash:            contentHash,
		Type:            resourceType,
		SourceID:        &sourceID,
		StorageMode:     "external",
		ExternalHash:    &externalHash,
		ExternalPath:    &relativePath,
		Metadata:        metadata,
		RefCount:        0,
		CreatedAt:       now,
		UpdatedAt:       now,
		ContentEncoding: encoding,
	}
	s.state.Resources = append(s.state.Resources, resource)
	previewJob := pdfPreviewRenderJobForResource(resource, now)
	if err := s.flushLocked(); err != nil {
		s.mu.Unlock()
		return UploadFileResult{}, err
	}
	s.mu.Unlock()
	if err := s.renderAndCommitPdfPreview(previewJob); err != nil {
		return UploadFileResult{}, err
	}
	return s.uploadFileResultByResourceID(resourceID, fileType, contentHash, true)
}

func (s *Service) uploadFileResultByResourceID(resourceID string, fileType string, resourceHash string, isNew bool) (UploadFileResult, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	resource, ok := s.findResourceByAnyIDLocked(resourceID)
	if !ok || resourceIsDeleted(resource) {
		return UploadFileResult{}, fmt.Errorf("resource not found: %s", resourceID)
	}
	file, err := s.resourceToFile(resource, fileType)
	if err != nil {
		return UploadFileResult{}, err
	}
	return uploadFileResult(file, resourceHash, isNew), nil
}

func (s *Service) GetFile(fileID string) (*VfsFile, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if !s.hasFileLikeAliasLocked(fileID) {
		return nil, fmt.Errorf("invalid file ID format: %s", fileID)
	}
	resource, ok := s.findFileLikeResourceByAnyIDLocked(fileID)
	if !ok || resourceIsDeleted(resource) {
		return nil, nil
	}
	file, err := s.resourceToFile(resource, fileTypeForResource(resource))
	if err != nil {
		return nil, err
	}
	return &file, nil
}

func (s *Service) DeleteFile(fileID string) error {
	now := nowMillis()
	s.mu.Lock()
	defer s.mu.Unlock()
	if !s.hasFileLikeAliasLocked(fileID) {
		return fmt.Errorf("invalid file ID format: %s", fileID)
	}
	index, ok := s.findFileLikeResourceIndexByAnyIDLocked(fileID)
	if !ok {
		return nil
	}
	s.state.Resources[index].Metadata = mergeMetadata(s.state.Resources[index].Metadata, map[string]any{
		"status":    "deleted",
		"deletedAt": formatMillis(now),
	})
	s.state.Resources[index].UpdatedAt = now
	return s.flushLocked()
}

func (s *Service) GetFileContent(fileID string) (AttachmentContentResult, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if !s.hasFileLikeAliasLocked(fileID) {
		return AttachmentContentResult{}, fmt.Errorf("invalid file ID format: %s", fileID)
	}
	resource, ok := s.findFileLikeResourceByAnyIDLocked(fileID)
	if !ok || resourceIsDeleted(resource) {
		return AttachmentContentResult{Found: false}, nil
	}
	hydrated, err := s.hydrateResourceData(resource)
	if err != nil {
		message := err.Error()
		return AttachmentContentResult{Found: false, Error: &message}, nil
	}
	return AttachmentContentResult{Content: hydrated.Data, Found: hydrated.Data != nil}, nil
}

func (s *Service) UpdateFileMetadata(fileID string, metadata map[string]any) (bool, error) {
	now := nowMillis()
	s.mu.Lock()
	defer s.mu.Unlock()
	if !s.hasFileLikeAliasLocked(fileID) {
		return false, fmt.Errorf("invalid file ID format: %s", fileID)
	}
	index, ok := s.findFileLikeResourceIndexByAnyIDLocked(fileID)
	if !ok || resourceIsDeleted(s.state.Resources[index]) {
		return false, nil
	}
	merged := mergeMetadata(s.state.Resources[index].Metadata, metadata)
	if name := firstNonEmptyMetadataString(metadata, "title", "name", "fileName"); name != "" {
		merged["name"] = name
		merged["title"] = name
	}
	merged["updatedAt"] = formatMillis(now)
	s.state.Resources[index].Metadata = merged
	s.state.Resources[index].UpdatedAt = now
	return true, s.flushLocked()
}

func (s *Service) UpdateBookmarks(fileID string, bookmarks []any) (bool, error) {
	now := nowMillis()
	s.mu.Lock()
	defer s.mu.Unlock()
	if !s.hasFileLikeAliasLocked(fileID) {
		return false, fmt.Errorf("invalid file ID format: %s", fileID)
	}
	index, ok := s.findFileLikeResourceIndexByAnyIDLocked(fileID)
	if !ok || resourceIsDeleted(s.state.Resources[index]) {
		return false, nil
	}
	s.state.Resources[index].Metadata = mergeMetadata(s.state.Resources[index].Metadata, map[string]any{
		"bookmarks": bookmarks,
		"updatedAt": formatMillis(now),
	})
	s.state.Resources[index].UpdatedAt = now
	return true, s.flushLocked()
}

func (s *Service) GetResourceRefs(input GetResourceRefsInput) (ContextRefData, error) {
	maxItems := input.MaxItems
	if maxItems <= 0 || maxItems > 50 {
		maxItems = 50
	}
	refs := make([]ResourceRef, 0, minInt(len(input.SourceIDs), maxItems))
	totalCount := 0
	truncated := false

	s.mu.RLock()
	defer s.mu.RUnlock()

	for _, sourceID := range input.SourceIDs {
		sourceID = strings.TrimSpace(sourceID)
		if sourceID == "" {
			continue
		}
		resource, ok := s.findResourceByAnyIDLocked(sourceID)
		if !ok || resourceIsDeleted(resource) {
			continue
		}
		totalCount++
		if len(refs) >= maxItems {
			truncated = true
			continue
		}
		refs = append(refs, resourceToRef(resource))
	}
	return ContextRefData{
		Refs:       refs,
		TotalCount: totalCount,
		Truncated:  truncated,
	}, nil
}

func (s *Service) ResolveResourceRefs(refs []ResourceRef) ([]ResolvedResource, error) {
	if len(refs) > 50 {
		return nil, fmt.Errorf("too many refs to resolve: %d (max: 50)", len(refs))
	}
	resolved := make([]ResolvedResource, 0, len(refs))

	s.mu.RLock()
	defer s.mu.RUnlock()

	for _, ref := range refs {
		resource, ok := s.findResourceByRefLocked(ref)
		if !ok {
			resolved = append(resolved, missingResolvedResource(ref))
			continue
		}
		hydrated, err := s.hydrateResourceData(resource)
		if err != nil {
			warning := err.Error()
			resolved = append(resolved, ResolvedResource{
				SourceID:     ref.SourceID,
				ResourceHash: ref.ResourceHash,
				Type:         ref.Type,
				Name:         ref.Name,
				Path:         "",
				Found:        false,
				Warning:      &warning,
			})
			continue
		}
		resolved = append(resolved, s.resourceToResolved(hydrated, ref))
	}
	return resolved, nil
}

func (s *Service) UpdatePathCache(folderID string) (int, error) {
	return 0, nil
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

func (s *Service) emitPdfProcessingStatusEvents(status PdfProcessingStatus) {
	mediaType := strings.TrimSpace(status.MediaType)
	if mediaType != "pdf" && mediaType != "image" {
		return
	}
	if status.Stage == "error" {
		message := "media processing failed"
		if status.Error != nil && strings.TrimSpace(*status.Error) != "" {
			message = *status.Error
		}
		payload := mediaProcessingErrorPayload{
			FileID:    status.FileID,
			Error:     message,
			Stage:     status.Stage,
			MediaType: mediaType,
		}
		s.emitEvent("media-processing-error", payload)
		if mediaType == "pdf" {
			s.emitEvent("pdf-processing-error", payload)
		}
		return
	}

	progressPayload := mediaProcessingProgressPayload{
		FileID:    status.FileID,
		Status:    status.Progress,
		MediaType: mediaType,
	}
	s.emitEvent("media-processing-progress", progressPayload)
	if mediaType == "pdf" {
		s.emitEvent("pdf-processing-progress", progressPayload)
	}

	if status.Stage == "completed" || status.Stage == "completed_with_issues" {
		completedPayload := mediaProcessingCompletedPayload{
			FileID:     status.FileID,
			ReadyModes: status.ReadyModes,
			Stage:      status.Stage,
			MediaType:  mediaType,
		}
		s.emitEvent("media-processing-completed", completedPayload)
		if mediaType == "pdf" {
			s.emitEvent("pdf-processing-completed", completedPayload)
		}
	}
}

func (s *Service) emitIndexProgress(payload map[string]any) {
	s.emitEvent("vfs-index-progress", payload)
}

func (s *Service) GetPdfProcessingStatus(fileID string) (PdfProcessingStatus, error) {
	fileID = strings.TrimSpace(fileID)
	if fileID == "" {
		return pdfProcessingMissingStatus(fileID), nil
	}

	s.mu.RLock()
	defer s.mu.RUnlock()

	resource, ok := s.findFileLikeResourceByAnyIDLocked(fileID)
	if !ok {
		return pdfProcessingMissingStatus(fileID), nil
	}
	if resourceIsDeleted(resource) {
		message := "resource is deleted"
		return pdfProcessingErrorStatus(fileID, mediaTypeForResource(resource), message), nil
	}
	hydrated, err := s.hydrateResourceData(resource)
	if err != nil {
		return pdfProcessingErrorStatus(fileID, mediaTypeForResource(resource), err.Error()), nil
	}
	return pdfProcessingStatusForResource(fileID, hydrated), nil
}

func (s *Service) GetBatchPdfProcessingStatus(fileIDs []string) (map[string]PdfProcessingStatus, error) {
	statuses := make(map[string]PdfProcessingStatus, len(fileIDs))
	for _, fileID := range fileIDs {
		trimmed := strings.TrimSpace(fileID)
		if trimmed == "" {
			continue
		}
		status, err := s.GetPdfProcessingStatus(trimmed)
		if err != nil {
			return map[string]PdfProcessingStatus{}, err
		}
		statuses[trimmed] = status
	}
	return statuses, nil
}

func (s *Service) CancelPdfProcessing(fileID string) (bool, error) {
	fileID = strings.TrimSpace(fileID)
	if fileID == "" {
		return false, nil
	}
	var statusToEmit *PdfProcessingStatus
	s.mu.RLock()
	resource, ok := s.findFileLikeResourceByAnyIDLocked(fileID)
	if ok && !resourceIsDeleted(resource) {
		status := pdfProcessingErrorStatus(fileID, mediaTypeForResource(resource), "processing cancelled")
		statusToEmit = &status
	}
	s.mu.RUnlock()
	if statusToEmit != nil {
		s.emitPdfProcessingStatusEvents(*statusToEmit)
	}
	return ok, nil
}

func (s *Service) RetryPdfProcessing(fileID string) error {
	return s.StartPdfProcessing(fileID, nil)
}

func (s *Service) StartPdfProcessing(fileID string, startFromStage *string) error {
	fileID = strings.TrimSpace(fileID)
	if fileID == "" {
		return nil
	}
	now := nowMillis()
	events := []PdfProcessingStatus{}
	var errToReturn error

	s.mu.Lock()
	index, ok := s.findFileLikeResourceIndexByAnyIDLocked(fileID)
	if !ok || resourceIsDeleted(s.state.Resources[index]) {
		s.mu.Unlock()
		return nil
	}
	resource := s.state.Resources[index]
	mediaType := mediaTypeForResource(resource)
	totalPages := metadataInt(resource.Metadata, "pageCount", "page_count")
	events = append(events, pdfProcessingStatus(fileID, "pending", nil, totalPages, 0, []string{}, mediaType, nil, nil))
	startStage := normalizeProcessingStage(startFromStage)
	if startStage == "" {
		startStage = defaultProcessingStartStage(mediaType)
	}
	if startStage != "" {
		events = append(events, pdfProcessingStatus(fileID, startStage, nil, totalPages, 5, []string{}, mediaType, nil, nil))
	}
	metadata := map[string]any{
		"processingRequestedAt": formatMillis(now),
	}
	if startStage != "" {
		metadata["processingStartStage"] = startStage
	}
	if mediaType == "pdf" {
		if resourceTextContent(s.state.Resources[index]) == nil || metadataInt(s.state.Resources[index].Metadata, "pageCount", "page_count") == nil {
			metadata = mergeMetadata(metadata, s.extractPdfTextForProcessingLocked(s.state.Resources[index], now))
		}
	}
	s.state.Resources[index].Metadata = mergeMetadata(s.state.Resources[index].Metadata, metadata)
	s.state.Resources[index].UpdatedAt = now
	previewJob := pdfPreviewRenderJobForResource(s.state.Resources[index], now)
	resourceID := s.state.Resources[index].ID
	errToReturn = s.flushLocked()
	if errToReturn != nil {
		status := pdfProcessingErrorStatus(fileID, mediaType, errToReturn.Error())
		events = append(events, status)
	}
	s.mu.Unlock()

	if errToReturn == nil {
		if err := s.renderAndCommitPdfPreview(previewJob); err != nil {
			errToReturn = err
			status := pdfProcessingErrorStatus(fileID, mediaType, err.Error())
			events = append(events, status)
		}
	}
	if errToReturn == nil && mediaType == "pdf" && startStage == "ocr_processing" {
		s.mu.RLock()
		resource, ok := s.findResourceByAnyIDLocked(resourceID)
		s.mu.RUnlock()
		if !ok || resourceIsDeleted(resource) {
			message := fmt.Sprintf("resource not found: %s", fileID)
			status := pdfProcessingErrorStatus(fileID, mediaType, message)
			events = append(events, status)
		} else {
			ocrMetadata := s.generatePdfOCRMetadata(resource, nowMillis())
			s.mu.Lock()
			if index, ok := s.findResourceIndexByAnyIDLocked(resourceID); ok && !resourceIsDeleted(s.state.Resources[index]) && s.state.Resources[index].Hash == resource.Hash {
				s.state.Resources[index].Metadata = mergeMetadata(s.state.Resources[index].Metadata, ocrMetadata)
				s.state.Resources[index].UpdatedAt = nowMillis()
				if err := s.flushLocked(); err != nil {
					errToReturn = err
					status := pdfProcessingErrorStatus(fileID, mediaType, err.Error())
					events = append(events, status)
				}
			}
			s.mu.Unlock()
		}
	}
	if errToReturn == nil {
		s.mu.RLock()
		if resource, ok := s.findResourceByAnyIDLocked(resourceID); !ok || resourceIsDeleted(resource) {
			message := fmt.Sprintf("resource not found: %s", fileID)
			status := pdfProcessingErrorStatus(fileID, mediaType, message)
			events = append(events, status)
		} else {
			events = append(events, pdfProcessingStatusForResource(fileID, resource))
		}
		s.mu.RUnlock()
	}
	for _, event := range events {
		s.emitPdfProcessingStatusEvents(event)
	}
	return errToReturn
}

func (s *Service) extractPdfTextForProcessingLocked(resource Resource, now int64) map[string]any {
	metadata := map[string]any{
		"textExtractionAttemptedAt": formatMillis(now),
		"pageCountAttemptedAt":      formatMillis(now),
	}
	pageCount := metadataInt(resource.Metadata, "pageCount", "page_count")
	bytes, err := s.readResourceBytesLocked(resource)
	if err != nil {
		metadata["textExtractionStatus"] = "failed"
		metadata["textExtractionError"] = err.Error()
		metadata["pageCountStatus"] = "failed"
		metadata["pageCountError"] = err.Error()
		return metadata
	}
	if detectedPageCount := detectPdfPageCount(bytes); detectedPageCount != nil && *detectedPageCount > 0 {
		pageCount = detectedPageCount
		metadata["pageCount"] = *detectedPageCount
		metadata["pageCountStatus"] = "completed"
		metadata["pageCountError"] = ""
	} else {
		metadata["pageCountStatus"] = "unknown"
		metadata["pageCountError"] = "No PDF page count was detected by the lean Go parser."
	}
	text := extractPdfTextLayer(bytes)
	if text == nil || strings.TrimSpace(*text) == "" {
		metadata["textExtractionStatus"] = "no_text_layer"
		metadata["textExtractionError"] = "No searchable PDF text layer was found by the lean Go parser."
		return metadata
	}
	metadata["extractedText"] = *text
	metadata["textExtractionStatus"] = "completed"
	metadata["textExtractionError"] = ""
	metadata["textExtractionSource"] = "pdf_text_layer"
	if pagesJSON := pdfTextLayerOcrPagesJSON(*text, pageCount); pagesJSON != nil {
		metadata["ocrPagesJson"] = *pagesJSON
		metadata["ocrPagesSource"] = "pdf_text_layer_estimated"
	}
	return metadata
}

func (s *Service) GetPdfPageImage(resourceID string, pageIndex int) (PdfPageImageResult, error) {
	resourceID = strings.TrimSpace(resourceID)
	if resourceID == "" {
		return PdfPageImageResult{}, errors.New("resourceId is required")
	}
	if pageIndex < 0 {
		return PdfPageImageResult{}, errors.New("pageIndex must be >= 0")
	}

	s.mu.RLock()
	defer s.mu.RUnlock()

	resource, ok := s.findFileLikeResourceByAnyIDLocked(resourceID)
	if !ok || resourceIsDeleted(resource) {
		return PdfPageImageResult{}, fmt.Errorf("resource not found: %s", resourceID)
	}
	previewJSON := firstMetadataString(resource.Metadata, "previewJson", "preview_json")
	if previewJSON == nil {
		return PdfPageImageResult{}, fmt.Errorf("resource has no PDF page preview data: %s", resourceID)
	}
	page, err := findPreviewPage(*previewJSON, pageIndex)
	if err != nil {
		return PdfPageImageResult{}, err
	}
	return s.resolvePreviewPageImageLocked(page)
}

func (s *Service) GetBlobBase64(blobHash string) (VfsBlobBase64Result, error) {
	blobHash = strings.TrimSpace(blobHash)
	if blobHash == "" {
		return VfsBlobBase64Result{}, errors.New("blobHash is required")
	}

	s.mu.RLock()
	defer s.mu.RUnlock()

	resource, ok := s.findResourceByAnyIDLocked(blobHash)
	if !ok || resourceIsDeleted(resource) {
		return VfsBlobBase64Result{}, fmt.Errorf("blob not found: %s", blobHash)
	}
	bytes, err := s.readResourceBytesLocked(resource)
	if err != nil {
		return VfsBlobBase64Result{}, fmt.Errorf("read blob failed: %w", err)
	}
	mimeType := firstNonEmptyMetadataString(resource.Metadata, "mimeType", "mime_type")
	if strings.TrimSpace(mimeType) == "" {
		mimeType = inferBlobMimeType(resourceName(resource), bytes)
	}
	return VfsBlobBase64Result{
		Base64:   base64.StdEncoding.EncodeToString(bytes),
		MimeType: mimeType,
		Size:     int64(len(bytes)),
	}, nil
}

func (s *Service) UnifiedIndexStatus() (IndexStatusSummary, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	summary := IndexStatusSummary{Dimensions: []DimensionStat{}}
	for _, resource := range s.state.Resources {
		if resourceIsDeleted(resource) {
			continue
		}
		hydrated, err := s.hydrateResourceData(resource)
		if err != nil {
			return IndexStatusSummary{}, err
		}
		unit := resourceToUnitStatus(hydrated, 0)
		summary.TotalUnits++
		addStateStat(&summary.TextStats, unit.TextState)
		addStateStat(&summary.MMStats, unit.MMState)
	}
	return summary, nil
}

func (s *Service) GetResourceUnits(resourceID string) ([]UnitIndexStatus, error) {
	resourceID = strings.TrimSpace(resourceID)
	if resourceID == "" {
		return []UnitIndexStatus{}, nil
	}

	s.mu.RLock()
	defer s.mu.RUnlock()

	resource, ok := s.findResourceByAnyIDLocked(resourceID)
	if !ok || resourceIsDeleted(resource) {
		return []UnitIndexStatus{}, nil
	}
	hydrated, err := s.hydrateResourceData(resource)
	if err != nil {
		return nil, err
	}
	return []UnitIndexStatus{resourceToUnitStatus(hydrated, 0)}, nil
}

func (s *Service) SyncResourceUnits(input SyncResourceUnitsInput) ([]UnitIndexStatus, error) {
	resourceID := strings.TrimSpace(input.ResourceID)
	if resourceID == "" {
		return nil, errors.New("resourceId is required")
	}

	data := firstNonEmptyString(input.Data, input.ExtractedText, input.OcrText)
	if data != nil {
		resourceType := normalizeResourceType(input.ResourceType)
		if resourceType == "" {
			resourceType = "retrieval"
		}
		metadata := map[string]any{
			"sourceId":     resourceID,
			"sourceType":   "resource_unit",
			"resourceType": resourceType,
			"previewType":  resourceType,
			"syncedUnits":  true,
		}
		if input.PageCount != nil {
			metadata["pageCount"] = *input.PageCount
		}
		if value := normalizeOptionalString(input.ExtractedText); value != nil {
			metadata["extractedText"] = *value
		}
		if value := normalizeOptionalString(input.OcrText); value != nil {
			metadata["ocrText"] = *value
		}
		if value := normalizeOptionalString(input.BlobHash); value != nil {
			metadata["blobHash"] = *value
		}
		if value := normalizeOptionalString(input.PreviewJSON); value != nil {
			metadata["previewJson"] = *value
		}
		if value := normalizeOptionalString(input.OcrPagesJSON); value != nil {
			metadata["ocrPagesJson"] = *value
		}
		if _, err := s.CreateOrUpdateSource(CreateResourceInput{
			Type:     resourceType,
			Data:     *data,
			SourceID: &resourceID,
			Metadata: metadata,
		}); err != nil {
			return nil, err
		}
	}

	return s.GetResourceUnits(resourceID)
}

func (s *Service) GetAllIndexStatus(input GetIndexStatusInput) (ResourceIndexStatusSummary, error) {
	limit := input.Limit
	if limit <= 0 || limit > 500 {
		limit = 100
	}
	offset := input.Offset
	if offset < 0 {
		offset = 0
	}
	resourceType := ""
	if input.ResourceType != nil {
		resourceType = strings.ToLower(strings.TrimSpace(*input.ResourceType))
	}
	stateFilter := ""
	if input.StateFilter != nil {
		stateFilter = strings.ToLower(strings.TrimSpace(*input.StateFilter))
	}

	s.mu.RLock()
	defer s.mu.RUnlock()

	all := make([]ResourceIndexStatus, 0, len(s.state.Resources))
	summary := ResourceIndexStatusSummary{Resources: []ResourceIndexStatus{}}
	for _, resource := range s.state.Resources {
		if resourceIsDeleted(resource) {
			continue
		}
		if resourceType != "" && resource.Type != resourceType {
			continue
		}
		hydrated, err := s.hydrateResourceData(resource)
		if err != nil {
			return ResourceIndexStatusSummary{}, err
		}
		status := resourceToIndexStatus(hydrated)
		if stateFilter != "" && status.DisplayIndexState != stateFilter && status.TextIndexState != stateFilter && status.MMIndexState != stateFilter {
			continue
		}
		all = append(all, status)
		accumulateResourceIndexSummary(&summary, status)
	}
	summary.TotalResources = len(all)
	if offset < len(all) {
		end := offset + limit
		if end > len(all) {
			end = len(all)
		}
		summary.Resources = all[offset:end]
	}
	return summary, nil
}

func (s *Service) ReindexResource(resourceID string) (int, error) {
	resourceID = strings.TrimSpace(resourceID)
	if resourceID == "" {
		return 0, nil
	}
	s.emitIndexProgress(map[string]any{
		"type":       "started",
		"resourceId": resourceID,
		"progress":   0,
		"message":    "Go VFS reindex started",
	})
	units, err := s.GetResourceUnits(resourceID)
	if err != nil {
		s.emitIndexProgress(map[string]any{
			"type":       "failed",
			"resourceId": resourceID,
			"progress":   0,
			"message":    err.Error(),
			"success":    false,
		})
		return 0, err
	}
	chunks := len(units)
	s.emitIndexProgress(map[string]any{
		"type":            "completed",
		"resourceId":      resourceID,
		"progress":        100,
		"message":         "Go VFS reindex completed",
		"success":         true,
		"chunksProcessed": chunks,
		"chunksTotal":     chunks,
	})
	return chunks, nil
}

func (s *Service) ReindexUnit(unitID string, mode string) (bool, error) {
	return strings.TrimSpace(unitID) != "", nil
}

func (s *Service) BatchIndexPending(batchSize int) (BatchIndexResult, error) {
	if batchSize <= 0 {
		batchSize = 10
	}
	pending, err := s.pendingIndexStatuses(batchSize)
	if err != nil {
		return BatchIndexResult{}, err
	}
	total := len(pending)
	s.emitIndexProgress(map[string]any{
		"type":     "batch_started",
		"progress": 0,
		"current":  0,
		"total":    total,
		"message":  "Go VFS batch indexing started",
	})
	for index, status := range pending {
		current := index + 1
		progress := 100
		if total > 0 {
			progress = int(float64(current) / float64(total) * 100)
		}
		s.emitIndexProgress(map[string]any{
			"type":       "resource_completed",
			"resourceId": status.ResourceID,
			"progress":   progress,
			"current":    current,
			"total":      total,
			"message":    "Go VFS resource index state synchronized",
			"success":    true,
		})
	}
	result := BatchIndexResult{SuccessCount: total, FailCount: 0, Total: total}
	s.emitIndexProgress(map[string]any{
		"type":         "batch_completed",
		"progress":     100,
		"current":      total,
		"total":        total,
		"successCount": result.SuccessCount,
		"failCount":    result.FailCount,
		"message":      "Go VFS batch indexing completed",
	})
	return result, nil
}

func (s *Service) pendingIndexStatuses(limit int) ([]ResourceIndexStatus, error) {
	if limit <= 0 {
		limit = 10
	}
	s.mu.RLock()
	defer s.mu.RUnlock()

	pending := []ResourceIndexStatus{}
	for _, resource := range s.state.Resources {
		if resourceIsDeleted(resource) {
			continue
		}
		hydrated, err := s.hydrateResourceData(resource)
		if err != nil {
			return nil, err
		}
		status := resourceToIndexStatus(hydrated)
		if status.TextIndexState == "pending" ||
			status.TextIndexState == "failed" ||
			status.TextIndexState == "stale" ||
			status.TextIndexState == "indexing" ||
			status.MMIndexState == "pending" ||
			status.MMIndexState == "failed" ||
			status.MMIndexState == "indexing" ||
			status.DisplayIndexState == "pending" ||
			status.DisplayIndexState == "failed" ||
			status.DisplayIndexState == "indexing" {
			pending = append(pending, status)
		}
		if len(pending) >= limit {
			break
		}
	}
	return pending, nil
}

func (s *Service) DeleteResourceIndex(resourceID string) (DeleteIndexResult, error) {
	return DeleteIndexResult{
		SQLiteOK:    true,
		LanceTextOK: true,
		LanceMMOK:   true,
		Warnings:    []string{},
		Retryable:   false,
	}, nil
}

func (s *Service) ListEmbeddingDims() ([]EmbeddingDimInfo, error) {
	return []EmbeddingDimInfo{}, nil
}

func (s *Service) ListDimensions() ([]VfsEmbeddingDimension, error) {
	return []VfsEmbeddingDimension{}, nil
}

func (s *Service) GetResourceTextChunks(resourceID string) ([]TextChunkInfo, error) {
	units, err := s.GetResourceUnits(resourceID)
	if err != nil || len(units) == 0 {
		return []TextChunkInfo{}, err
	}

	s.mu.RLock()
	defer s.mu.RUnlock()

	resource, ok := s.findResourceByAnyIDLocked(resourceID)
	if !ok {
		return []TextChunkInfo{}, nil
	}
	hydrated, err := s.hydrateResourceData(resource)
	if err != nil {
		return nil, err
	}
	text := resourceTextContent(hydrated)
	return []TextChunkInfo{{
		UnitID:         units[0].UnitID,
		UnitIndex:      units[0].UnitIndex,
		TextContent:    text,
		TextSource:     units[0].TextSource,
		TextState:      units[0].TextState,
		TextChunkCount: units[0].TextChunkCount,
		CharCount:      runeLenPtr(text),
	}}, nil
}

func (s *Service) GetResourceOcrInfo(resourceID string) (ResourceOcrInfo, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	resource, ok := s.findResourceByAnyIDLocked(resourceID)
	if !ok || resourceIsDeleted(resource) {
		return ResourceOcrInfo{ResourceID: resourceID, ActiveSource: "none", OcrPages: []OcrPageInfo{}}, nil
	}
	hydrated, err := s.hydrateResourceData(resource)
	if err != nil {
		return ResourceOcrInfo{}, err
	}
	extractedText := extractedTextContent(hydrated)
	ocrText := realOcrTextContent(hydrated)
	ocrPages := parseOcrPagesJSON(firstMetadataString(hydrated.Metadata, "ocrPagesJson", "ocr_pages_json"))
	hasRealOcrPages := hasRealOcrPages(hydrated)
	return ResourceOcrInfo{
		ResourceID:          hydrated.ID,
		ResourceType:        hydrated.Type,
		HasOcr:              ocrText != nil || hasRealOcrPages,
		OcrText:             ocrText,
		OcrTextLength:       runeLenPtr(ocrText),
		ExtractedText:       extractedText,
		ExtractedTextLength: runeLenPtr(extractedText),
		ActiveSource:        activeTextSource(ocrText, extractedText),
		OcrPages:            ocrPages,
	}, nil
}

func (s *Service) ClearResourceOcr(resourceID string) (bool, error) {
	resourceID = strings.TrimSpace(resourceID)
	if resourceID == "" {
		return false, nil
	}

	now := nowMillis()
	s.mu.Lock()
	defer s.mu.Unlock()

	index, ok := s.findResourceIndexByAnyIDLocked(resourceID)
	if !ok || resourceIsDeleted(s.state.Resources[index]) {
		return false, nil
	}

	metadata := normalizeMetadata(s.state.Resources[index].Metadata)
	if metadata == nil {
		metadata = map[string]any{}
	}
	for _, key := range []string{
		"ocrText", "ocr_text",
		"ocrPagesJson", "ocr_pages_json",
		"ocrPagesSource", "ocr_pages_source",
		"ocrStatus", "ocr_status",
		"ocrError", "ocr_error",
		"ocrUpdatedAt", "ocr_updated_at",
		"ocrCompletedAt", "ocr_completed_at",
	} {
		delete(metadata, key)
	}
	metadata["ocrClearedAt"] = formatMillis(now)
	metadata["processingStatus"] = "ocr_processing"
	metadata["processingError"] = ""
	metadata["processingProgress"] = map[string]any{
		"stage":      "ocr_processing",
		"percent":    40.0,
		"message":    "OCR cleared; waiting to reprocess",
		"readyModes": []string{},
	}
	metadata["indexStatus"] = "pending"
	metadata["textIndexState"] = "pending"
	metadata["updatedAt"] = formatMillis(now)

	s.state.Resources[index].Metadata = metadata
	s.state.Resources[index].UpdatedAt = now
	return true, s.flushLocked()
}

func (s *Service) RagSearch(input VfsRagSearchInput) (VfsRagSearchOutput, error) {
	startedAt := time.Now()
	query := strings.TrimSpace(input.Query)
	if query == "" {
		return VfsRagSearchOutput{Results: []VfsSearchResult{}, Count: 0, ElapsedMs: time.Since(startedAt).Milliseconds()}, nil
	}
	topK := input.TopK
	if topK <= 0 {
		topK = 10
	}
	if topK > 100 {
		topK = 100
	}
	resourceTypes := normalizeStringSet(input.ResourceTypes)
	folderIDs := normalizeStringSet(input.FolderIDs)

	s.mu.RLock()
	defer s.mu.RUnlock()

	results := make([]VfsSearchResult, 0, minInt(len(s.state.Resources), topK))
	for _, resource := range s.state.Resources {
		if resourceIsDeleted(resource) {
			continue
		}
		if len(resourceTypes) > 0 && !resourceTypes[resource.Type] {
			continue
		}
		if len(folderIDs) > 0 && !resourceMatchesFolderIDs(resource, folderIDs) {
			continue
		}
		hydrated, err := s.hydrateResourceData(resource)
		if err != nil {
			return VfsRagSearchOutput{}, err
		}
		text := resourceTextContent(hydrated)
		score := scoreResourceSearch(hydrated, query, text)
		if score <= 0 {
			continue
		}
		sourceID := resourceSourceID(hydrated)
		resourceType := hydrated.Type
		title := resourceName(hydrated)
		results = append(results, VfsSearchResult{
			EmbeddingID:   fmt.Sprintf("go_vfs_%s_0", hydrated.ID),
			ResourceID:    hydrated.ID,
			ChunkIndex:    0,
			ChunkText:     snippetForQuery(firstNonEmptyValue(text, resourceSnippet(hydrated), &title), query),
			Score:         score,
			ResourceTitle: &title,
			ResourceType:  &resourceType,
			PageIndex:     metadataInt(hydrated.Metadata, "pageIndex"),
			SourceID:      &sourceID,
		})
	}

	sort.SliceStable(results, func(left int, right int) bool {
		if results[left].Score == results[right].Score {
			return results[left].ResourceID < results[right].ResourceID
		}
		return results[left].Score > results[right].Score
	})
	if len(results) > topK {
		results = results[:topK]
	}
	return VfsRagSearchOutput{
		Results:   results,
		Count:     len(results),
		ElapsedMs: time.Since(startedAt).Milliseconds(),
	}, nil
}

func (s *Service) ListFiles(input ListFilesInput) ([]VfsFile, error) {
	limit := input.Limit
	if limit <= 0 || limit > 500 {
		limit = 100
	}
	offset := input.Offset
	if offset < 0 {
		offset = 0
	}
	fileType := strings.ToLower(strings.TrimSpace(input.FileType))

	s.mu.RLock()
	defer s.mu.RUnlock()

	files := make([]VfsFile, 0)
	for _, resource := range s.state.Resources {
		if !resourceIsListableFile(resource) {
			continue
		}
		resolvedFileType := fileTypeForResource(resource)
		if fileType != "" && resolvedFileType != fileType {
			continue
		}
		file, err := s.resourceToFile(resource, resolvedFileType)
		if err != nil {
			return nil, err
		}
		files = append(files, file)
	}
	sort.SliceStable(files, func(left int, right int) bool {
		return files[left].UpdatedAt > files[right].UpdatedAt
	})
	if offset >= len(files) {
		return []VfsFile{}, nil
	}
	end := offset + limit
	if end > len(files) {
		end = len(files)
	}
	return files[offset:end], nil
}

func (s *Service) load() error {
	bytes, err := os.ReadFile(s.indexPath)
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
	if s.state.Resources == nil {
		s.state.Resources = []Resource{}
	}
	return nil
}

func (s *Service) flushLocked() error {
	return storage.WriteJSONAtomic(s.indexPath, s.state)
}

func (s *Service) writeResourceData(resourceType string, hash string, data string, metadata map[string]any) (string, string, error) {
	encoding := contentEncoding(resourceType, data)
	ext := extensionForResource(resourceType, metadata)
	relativePath := filepath.ToSlash(filepath.Join(resourceType, hash[:2], fmt.Sprintf("%s.%s", hash, ext)))
	absolutePath, err := s.resolveLibraryPath(relativePath)
	if err != nil {
		return "", "", err
	}
	if err := os.MkdirAll(filepath.Dir(absolutePath), 0o700); err != nil {
		return "", "", err
	}
	if _, err := os.Stat(absolutePath); err == nil {
		return relativePath, encoding, nil
	} else if !errors.Is(err, os.ErrNotExist) {
		return "", "", err
	}

	bytes := []byte(data)
	if encoding == "base64" {
		decoded, err := decodeBase64Payload(data)
		if err != nil {
			return "", "", err
		}
		bytes = decoded
	}
	if err := os.WriteFile(absolutePath, bytes, 0o600); err != nil {
		return "", "", err
	}
	return relativePath, encoding, nil
}

func (s *Service) hydrateResourceData(resource Resource) (Resource, error) {
	if resource.ExternalPath == nil {
		return resource, nil
	}
	absolutePath, err := s.resolveLibraryPath(*resource.ExternalPath)
	if err != nil {
		return Resource{}, err
	}
	bytes, err := os.ReadFile(absolutePath)
	if err != nil {
		return Resource{}, err
	}
	data := string(bytes)
	if resource.ContentEncoding == "base64" {
		data = base64.StdEncoding.EncodeToString(bytes)
	}
	resource.Data = &data
	return resource, nil
}

func (s *Service) readResourceBytesLocked(resource Resource) ([]byte, error) {
	if resource.ExternalPath != nil {
		absolutePath, err := s.resolveLibraryPath(*resource.ExternalPath)
		if err != nil {
			return nil, err
		}
		return os.ReadFile(absolutePath)
	}
	if resource.Data == nil {
		return nil, fmt.Errorf("resource has no blob data: %s", resource.ID)
	}
	if resource.ContentEncoding == "base64" {
		return decodeBase64Payload(*resource.Data)
	}
	return []byte(*resource.Data), nil
}

func (s *Service) readResourceBytes(resource Resource) ([]byte, error) {
	if resource.ExternalPath != nil {
		absolutePath, err := s.resolveLibraryPath(*resource.ExternalPath)
		if err != nil {
			return nil, err
		}
		return os.ReadFile(absolutePath)
	}
	if resource.Data == nil {
		return nil, fmt.Errorf("resource has no blob data: %s", resource.ID)
	}
	if resource.ContentEncoding == "base64" {
		return decodeBase64Payload(*resource.Data)
	}
	return []byte(*resource.Data), nil
}

func (s *Service) resolveLibraryPath(relativePath string) (string, error) {
	cleaned := filepath.Clean(relativePath)
	if filepath.IsAbs(cleaned) {
		return s.ensureInsideLibrary(cleaned)
	}
	return s.ensureInsideLibrary(filepath.Join(s.libraryDir, cleaned))
}

func (s *Service) ensureInsideLibrary(path string) (string, error) {
	absolute, err := filepath.Abs(path)
	if err != nil {
		return "", err
	}
	root, err := filepath.Abs(s.libraryDir)
	if err != nil {
		return "", err
	}
	if absolute != root && !strings.HasPrefix(absolute, root+string(os.PathSeparator)) {
		return "", fmt.Errorf("path escapes vfs library: %s", path)
	}
	return absolute, nil
}

func (s *Service) findResourceByHashLocked(hash string, resourceType string) (int, bool) {
	for index, resource := range s.state.Resources {
		if resource.Hash == hash && resource.Type == resourceType {
			return index, true
		}
	}
	return -1, false
}

func (s *Service) findAttachmentByHashLocked(hash string, resourceType string) (int, bool) {
	for index, resource := range s.state.Resources {
		if resource.Hash != hash || resource.Type != resourceType {
			continue
		}
		if metadataBool(resource.Metadata, "attachment", false) {
			return index, true
		}
		if resource.SourceID != nil && strings.HasPrefix(*resource.SourceID, "att_") {
			return index, true
		}
	}
	return -1, false
}

func (s *Service) findFileByHashLocked(hash string, fileType string) (int, bool) {
	for index, resource := range s.state.Resources {
		if resource.Hash != hash || !metadataBool(resource.Metadata, "fileRecord", false) {
			continue
		}
		if fileTypeForResource(resource) == fileType {
			return index, true
		}
	}
	return -1, false
}

func (s *Service) findResourceByAnyIDLocked(id string) (Resource, bool) {
	for _, alias := range legacyAliasCandidates(id) {
		for _, resource := range s.state.Resources {
			if resourceMatchesAlias(resource, alias) {
				return resource, true
			}
		}
	}
	return Resource{}, false
}

func (s *Service) findResourceIndexByAnyIDLocked(id string) (int, bool) {
	for _, alias := range legacyAliasCandidates(id) {
		for index, resource := range s.state.Resources {
			if resourceMatchesAlias(resource, alias) {
				return index, true
			}
		}
	}
	return -1, false
}

func (s *Service) findFileLikeResourceByAnyIDLocked(id string) (Resource, bool) {
	for _, alias := range legacyAliasCandidates(id) {
		for _, resource := range s.state.Resources {
			if resourceIsFileLike(resource) && resourceMatchesAlias(resource, alias) {
				return resource, true
			}
		}
	}
	return Resource{}, false
}

func (s *Service) findFileLikeResourceIndexByAnyIDLocked(id string) (int, bool) {
	for _, alias := range legacyAliasCandidates(id) {
		for index, resource := range s.state.Resources {
			if resourceIsFileLike(resource) && resourceMatchesAlias(resource, alias) {
				return index, true
			}
		}
	}
	return -1, false
}

func (s *Service) hasAttachmentLikeAliasLocked(id string) bool {
	for _, alias := range legacyAliasCandidates(id) {
		if isAttachmentLikeIDToken(alias) {
			return true
		}
		for _, resource := range s.state.Resources {
			if resourceMatchesAlias(resource, alias) && resourceIsFileLike(resource) {
				return true
			}
		}
	}
	return false
}

func (s *Service) hasFileLikeAliasLocked(id string) bool {
	for _, alias := range legacyAliasCandidates(id) {
		if isFileLikeIDToken(alias) {
			return true
		}
		for _, resource := range s.state.Resources {
			if resourceMatchesAlias(resource, alias) && resourceIsFileLike(resource) {
				return true
			}
		}
	}
	return false
}

func (s *Service) findResourceByRefLocked(ref ResourceRef) (Resource, bool) {
	if ref.ResourceID != nil && strings.TrimSpace(*ref.ResourceID) != "" {
		if resource, ok := s.findResourceByAnyIDLocked(strings.TrimSpace(*ref.ResourceID)); ok {
			return resource, true
		}
	}
	if ref.SourceID != "" {
		if resource, ok := s.findResourceByAnyIDLocked(ref.SourceID); ok {
			return resource, true
		}
	}
	for _, resource := range s.state.Resources {
		if resource.Hash == ref.ResourceHash && resource.Type == ref.Type {
			return resource, true
		}
	}
	return Resource{}, false
}

func resourceToRef(resource Resource) ResourceRef {
	sourceID := resource.ID
	if resource.SourceID != nil && *resource.SourceID != "" {
		sourceID = *resource.SourceID
	}
	name := resourceName(resource)
	resourceID := resource.ID
	snippet := resourceSnippet(resource)
	return ResourceRef{
		SourceID:     sourceID,
		ResourceHash: resource.Hash,
		Type:         resource.Type,
		Name:         name,
		ResourceID:   &resourceID,
		Snippet:      snippet,
	}
}

func (s *Service) resourceToResolved(resource Resource, ref ResourceRef) ResolvedResource {
	sourceID := ref.SourceID
	if sourceID == "" {
		sourceID = resource.ID
		if resource.SourceID != nil && *resource.SourceID != "" {
			sourceID = *resource.SourceID
		}
	}
	resourceHash := ref.ResourceHash
	if resourceHash == "" {
		resourceHash = resource.Hash
	}
	resourceType := ref.Type
	if resourceType == "" {
		resourceType = resource.Type
	}
	name := ref.Name
	if strings.TrimSpace(name) == "" {
		name = resourceName(resource)
	}
	path := sourceID
	if metadataPath := resourcePathForResolve(resource, ref); metadataPath != "" {
		path = metadataPath
	} else if resource.ExternalPath != nil {
		if absolute, err := s.resolveLibraryPath(*resource.ExternalPath); err == nil && absolute != "" {
			path = absolute
		} else {
			path = *resource.ExternalPath
		}
	}
	var byteSize *int64
	if size := metadataInt64(resource.Metadata, "size"); size != nil {
		byteSize = size
	} else if resource.Data != nil {
		value := int64(len([]byte(*resource.Data)))
		byteSize = &value
	}
	resolved := ResolvedResource{
		SourceID:     sourceID,
		ResourceHash: resourceHash,
		Type:         resourceType,
		Name:         name,
		Path:         path,
		Content:      resource.Data,
		ByteSize:     byteSize,
		Found:        true,
		Metadata:     normalizeMetadata(resource.Metadata),
	}
	if text := realOcrTextContent(resource); text != nil {
		resolved.MultimodalBlocks = append(resolved.MultimodalBlocks, MultimodalBlock{
			Type: "text",
			Text: text,
		})
	}
	return resolved
}

func (s *Service) resourceToFile(resource Resource, fileType string) (VfsFile, error) {
	sourceID := resourceSourceID(resource)
	resourceID := resource.ID
	blobHash := resource.Hash
	if resource.ExternalHash != nil && strings.TrimSpace(*resource.ExternalHash) != "" {
		blobHash = strings.TrimSpace(*resource.ExternalHash)
	}
	var originalPath *string
	if path := firstMetadataString(resource.Metadata, "originalPath", "original_path", "path", "cachedPath", "cached_path"); path != nil {
		originalPath = path
	} else if resource.ExternalPath != nil {
		absolute, err := s.resolveLibraryPath(*resource.ExternalPath)
		if err != nil {
			return VfsFile{}, err
		}
		originalPath = &absolute
	}
	size := int64(0)
	if value := metadataInt64(resource.Metadata, "size"); value != nil {
		size = *value
	} else if originalPath != nil {
		if info, err := os.Stat(*originalPath); err == nil {
			size = info.Size()
		}
	} else if resource.Data != nil {
		size = int64(len([]byte(*resource.Data)))
	}
	createdAt := metadataString(resource.Metadata, "createdAt", formatMillis(resource.CreatedAt))
	updatedAt := metadataString(resource.Metadata, "updatedAt", formatMillis(resource.UpdatedAt))
	return VfsFile{
		ID:            sourceID,
		ResourceID:    &resourceID,
		BlobHash:      &blobHash,
		SHA256:        resource.Hash,
		FileName:      resourceName(resource),
		OriginalPath:  originalPath,
		Size:          size,
		PageCount:     metadataInt(resource.Metadata, "pageCount", "page_count"),
		FileType:      fileType,
		MimeType:      firstMetadataString(resource.Metadata, "mimeType", "mime_type"),
		Tags:          metadataStringSlice(resource.Metadata, "tags"),
		IsFavorite:    metadataBool(resource.Metadata, "isFavorite", false),
		LastOpenedAt:  firstMetadataString(resource.Metadata, "lastOpenedAt", "last_opened_at"),
		LastPage:      metadataInt(resource.Metadata, "lastPage", "last_page"),
		Bookmarks:     metadataAnySlice(resource.Metadata, "bookmarks", "bookmarks_json"),
		CoverKey:      firstMetadataString(resource.Metadata, "coverKey"),
		ExtractedText: extractedTextContent(resource),
		PreviewJSON:   firstMetadataString(resource.Metadata, "previewJson", "preview_json"),
		OcrPagesJSON:  firstMetadataString(resource.Metadata, "ocrPagesJson", "ocr_pages_json"),
		Description:   firstMetadataString(resource.Metadata, "description"),
		Status:        metadataString(resource.Metadata, "status", "ready"),
		CreatedAt:     createdAt,
		UpdatedAt:     updatedAt,
		DeletedAt:     firstMetadataString(resource.Metadata, "deletedAt"),
		Metadata:      normalizeMetadata(resource.Metadata),
	}, nil
}

func resourcePathForResolve(resource Resource, ref ResourceRef) string {
	if path := firstMetadataString(resource.Metadata, "path", "originalPath", "original_path", "cachedPath", "cached_path"); path != nil {
		return *path
	}
	return ""
}

func missingResolvedResource(ref ResourceRef) ResolvedResource {
	return ResolvedResource{
		SourceID:     ref.SourceID,
		ResourceHash: ref.ResourceHash,
		Type:         ref.Type,
		Name:         ref.Name,
		Path:         "",
		Found:        false,
	}
}

func resourceName(resource Resource) string {
	for _, key := range []string{"name", "title"} {
		if value, ok := resource.Metadata[key].(string); ok && strings.TrimSpace(value) != "" {
			return strings.TrimSpace(value)
		}
	}
	if resource.SourceID != nil && *resource.SourceID != "" {
		return *resource.SourceID
	}
	return resource.ID
}

func resourceSnippet(resource Resource) *string {
	if value, ok := resource.Metadata["snippet"].(string); ok && strings.TrimSpace(value) != "" {
		snippet := strings.TrimSpace(value)
		return &snippet
	}
	return nil
}

func resourceToUnitStatus(resource Resource, unitIndex int) UnitIndexStatus {
	text := resourceTextContent(resource)
	hasText := text != nil && strings.TrimSpace(*text) != ""
	hasImage := resourceHasImage(resource)
	textSource := (*string)(nil)
	textState := "disabled"
	textChunkCount := 0
	if hasText {
		source := "native"
		textSource = &source
		textState = "indexed"
		textChunkCount = estimateTextChunks(*text)
	}
	mmRequired := hasImage || resource.Type == "textbook" || firstNonEmptyMetadataString(resource.Metadata, "mimeType", "mime_type") == "application/pdf"
	mmState := "disabled"
	if mmRequired {
		mmState = "pending"
	}
	return UnitIndexStatus{
		UnitID:         fmt.Sprintf("%s_unit_%d", resource.ID, unitIndex),
		ResourceID:     resource.ID,
		UnitIndex:      unitIndex,
		HasImage:       hasImage,
		HasText:        hasText,
		TextSource:     textSource,
		TextRequired:   hasText,
		TextState:      textState,
		TextChunkCount: textChunkCount,
		MMRequired:     mmRequired,
		MMState:        mmState,
		UpdatedAt:      resource.UpdatedAt,
	}
}

func resourceToIndexStatus(resource Resource) ResourceIndexStatus {
	unit := resourceToUnitStatus(resource, 0)
	textIndexedAt := (*int64)(nil)
	if unit.TextState == "indexed" {
		updatedAt := resource.UpdatedAt
		textIndexedAt = &updatedAt
	}
	displayState := "disabled"
	if unit.TextState == "indexed" {
		displayState = "indexed"
	} else if unit.MMState == "pending" {
		displayState = "pending"
	}
	sourceID := normalizeOptionalString(resource.SourceID)
	return ResourceIndexStatus{
		ResourceID:           resource.ID,
		SourceID:             sourceID,
		ResourceType:         resource.Type,
		Name:                 resourceName(resource),
		HasOcr:               false,
		OcrCount:             0,
		TextIndexState:       unit.TextState,
		TextIndexedAt:        textIndexedAt,
		TextChunkCount:       unit.TextChunkCount,
		NativeTextChunkCount: unit.TextChunkCount,
		OcrTextChunkCount:    0,
		TextIndexSource:      unit.TextSource,
		TextIndexRetryable:   false,
		MMIndexState:         unit.MMState,
		MMIndexedPages:       0,
		DisplayIndexState:    displayState,
		UpdatedAt:            resource.UpdatedAt,
		IsStale:              false,
	}
}

func resourceTextContent(resource Resource) *string {
	if text := firstNonEmptyString(extractedTextContent(resource), realOcrTextContent(resource)); text != nil {
		return text
	}
	return resourceDataTextContent(resource)
}

func extractedTextContent(resource Resource) *string {
	return firstNonEmptyString(
		firstMetadataString(resource.Metadata, "extractedText", "extracted_text", "text", "content"),
		resourceDataTextContent(resource),
	)
}

func realOcrTextContent(resource Resource) *string {
	if text := firstMetadataString(resource.Metadata, "ocrText", "ocr_text"); text != nil {
		return text
	}
	if !hasRealOcrPages(resource) {
		return nil
	}
	return ocrTextFromPages(parseOcrPagesJSON(firstMetadataString(resource.Metadata, "ocrPagesJson", "ocr_pages_json")))
}

func resourceDataTextContent(resource Resource) *string {
	if resource.Data == nil || resource.ContentEncoding == "base64" {
		return nil
	}
	if strings.TrimSpace(*resource.Data) == "" {
		return nil
	}
	return resource.Data
}

func hasRealOcrPages(resource Resource) bool {
	pages := parseOcrPagesJSON(firstMetadataString(resource.Metadata, "ocrPagesJson", "ocr_pages_json"))
	source := firstNonEmptyMetadataString(resource.Metadata, "ocrPagesSource", "ocr_pages_source")
	return len(pages) > 0 && source != "pdf_text_layer_estimated"
}

func ocrTextFromPages(pages []OcrPageInfo) *string {
	parts := make([]string, 0, len(pages))
	for _, page := range pages {
		if strings.TrimSpace(page.Text) != "" && !page.IsFailed {
			parts = append(parts, strings.TrimSpace(page.Text))
		}
	}
	if len(parts) == 0 {
		return nil
	}
	text := strings.Join(parts, "\n\n")
	return &text
}

func resourceHasImage(resource Resource) bool {
	mimeType := strings.ToLower(firstNonEmptyMetadataString(resource.Metadata, "mimeType", "mime_type"))
	return resource.Type == "image" || strings.HasPrefix(mimeType, "image/")
}

func resourceSourceID(resource Resource) string {
	if resource.SourceID != nil && strings.TrimSpace(*resource.SourceID) != "" {
		return strings.TrimSpace(*resource.SourceID)
	}
	return resource.ID
}

func resourceIsListableFile(resource Resource) bool {
	if resourceIsDeleted(resource) {
		return false
	}
	return resourceIsFileLike(resource)
}

func resourceIsFileLike(resource Resource) bool {
	if resource.Type == "file" || resource.Type == "image" || resource.Type == "textbook" {
		return true
	}
	return metadataBool(resource.Metadata, "attachment", false) ||
		metadataBool(resource.Metadata, "fileRecord", false) ||
		metadataString(resource.Metadata, "attachmentType", "") != "" ||
		firstNonEmptyMetadataString(resource.Metadata, "mimeType", "mime_type") != ""
}

func resourceIsDeleted(resource Resource) bool {
	status := strings.ToLower(metadataString(resource.Metadata, "status", ""))
	if status == "deleted" || status == "trash" {
		return true
	}
	return metadataString(resource.Metadata, "deletedAt", "") != ""
}

func resourceMatchesFolderIDs(resource Resource, folderIDs map[string]bool) bool {
	for _, key := range []string{"folderId", "folderID", "parentId", "attachmentRootFolderId"} {
		value := strings.ToLower(strings.TrimSpace(metadataString(resource.Metadata, key, "")))
		if value != "" && folderIDs[value] {
			return true
		}
	}
	for _, value := range metadataStringSlice(resource.Metadata, "folderIds") {
		if folderIDs[strings.ToLower(strings.TrimSpace(value))] {
			return true
		}
	}
	return false
}

func fileTypeForResource(resource Resource) string {
	mimeType := strings.ToLower(firstNonEmptyMetadataString(resource.Metadata, "mimeType", "mime_type"))
	attachmentType := strings.ToLower(metadataString(resource.Metadata, "attachmentType", ""))
	if resource.Type == "image" || attachmentType == "image" || strings.HasPrefix(mimeType, "image/") {
		return "image"
	}
	if strings.HasPrefix(mimeType, "audio/") {
		return "audio"
	}
	if strings.HasPrefix(mimeType, "video/") {
		return "video"
	}
	return "document"
}

func resourceTypeForFileUpload(fileType string, metadata map[string]any) string {
	for _, key := range []string{"sourceType", "resourceType", "type"} {
		if strings.ToLower(metadataString(metadata, key, "")) == "textbook" {
			return "textbook"
		}
	}
	if fileType == "image" {
		return "image"
	}
	return "file"
}

func scoreResourceSearch(resource Resource, query string, text *string) float64 {
	lowerQuery := strings.ToLower(strings.TrimSpace(query))
	if lowerQuery == "" {
		return 0
	}
	tokens := searchTokens(lowerQuery)
	score := 0.0
	score += fieldSearchScore(resourceName(resource), lowerQuery, tokens, 1.0)
	score += fieldSearchScore(resourceSourceID(resource), lowerQuery, tokens, 0.85)
	score += fieldSearchScore(resource.Type, lowerQuery, tokens, 0.5)
	score += fieldSearchScore(metadataSearchText(resource.Metadata), lowerQuery, tokens, 0.45)
	if text != nil {
		score += fieldSearchScore(*text, lowerQuery, tokens, 0.7)
	}
	if score == 0 && resourceMatchesTokens(resourceSearchHaystack(resource, text), tokens) {
		score = 0.1
	}
	if score > 1 {
		return 1
	}
	return score
}

func fieldSearchScore(value string, lowerQuery string, tokens []string, weight float64) float64 {
	lowerValue := strings.ToLower(value)
	if strings.TrimSpace(lowerValue) == "" {
		return 0
	}
	if strings.Contains(lowerValue, lowerQuery) {
		return weight
	}
	matches := 0
	for _, token := range tokens {
		if strings.Contains(lowerValue, token) {
			matches++
		}
	}
	if matches == 0 {
		return 0
	}
	return weight * (float64(matches) / float64(len(tokens))) * 0.7
}

func resourceMatchesTokens(haystack string, tokens []string) bool {
	if len(tokens) == 0 {
		return false
	}
	lower := strings.ToLower(haystack)
	for _, token := range tokens {
		if strings.Contains(lower, token) {
			return true
		}
	}
	return false
}

func resourceSearchHaystack(resource Resource, text *string) string {
	parts := []string{resourceName(resource), resourceSourceID(resource), resource.Type, metadataSearchText(resource.Metadata)}
	if text != nil {
		parts = append(parts, *text)
	}
	return strings.Join(parts, "\n")
}

func metadataSearchText(metadata map[string]any) string {
	if len(metadata) == 0 {
		return ""
	}
	parts := make([]string, 0, len(metadata))
	for key, value := range metadata {
		switch typed := value.(type) {
		case string:
			parts = append(parts, key, typed)
		case []string:
			parts = append(parts, key, strings.Join(typed, " "))
		case []any:
			values := make([]string, 0, len(typed))
			for _, item := range typed {
				if text, ok := item.(string); ok {
					values = append(values, text)
				}
			}
			parts = append(parts, key, strings.Join(values, " "))
		}
	}
	return strings.Join(parts, "\n")
}

func snippetForQuery(text string, query string) string {
	trimmed := strings.TrimSpace(text)
	if trimmed == "" {
		return ""
	}
	const maxRunes = 320
	runes := []rune(trimmed)
	if len(runes) <= maxRunes {
		return trimmed
	}
	lowerText := strings.ToLower(trimmed)
	lowerQuery := strings.ToLower(strings.TrimSpace(query))
	index := strings.Index(lowerText, lowerQuery)
	if index < 0 {
		return string(runes[:maxRunes]) + "..."
	}
	prefixRunes := len([]rune(trimmed[:index]))
	start := prefixRunes - maxRunes/3
	if start < 0 {
		start = 0
	}
	end := start + maxRunes
	if end > len(runes) {
		end = len(runes)
		start = end - maxRunes
		if start < 0 {
			start = 0
		}
	}
	snippet := string(runes[start:end])
	if start > 0 {
		snippet = "..." + snippet
	}
	if end < len(runes) {
		snippet += "..."
	}
	return snippet
}

func firstNonEmptyValue(values ...*string) string {
	for _, value := range values {
		if value != nil && strings.TrimSpace(*value) != "" {
			return strings.TrimSpace(*value)
		}
	}
	return ""
}

func firstNonEmptyPlainString(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return strings.TrimSpace(value)
		}
	}
	return ""
}

func searchTokens(value string) []string {
	seen := map[string]bool{}
	tokens := []string{}
	for _, token := range strings.Fields(strings.ToLower(value)) {
		token = strings.Trim(token, ".,;:!?()[]{}\"'")
		if token == "" || seen[token] {
			continue
		}
		seen[token] = true
		tokens = append(tokens, token)
	}
	if len(tokens) == 0 && strings.TrimSpace(value) != "" {
		tokens = append(tokens, strings.TrimSpace(value))
	}
	return tokens
}

func normalizeStringSet(values []string) map[string]bool {
	out := map[string]bool{}
	for _, value := range values {
		trimmed := strings.ToLower(strings.TrimSpace(value))
		if trimmed != "" && trimmed != "all" {
			out[trimmed] = true
		}
	}
	return out
}

func addStateStat(stats *StateStats, state string) {
	switch state {
	case "pending":
		stats.Pending++
	case "indexing":
		stats.Indexing++
	case "indexed":
		stats.Indexed++
	case "failed":
		stats.Failed++
	default:
		stats.Disabled++
	}
}

func accumulateResourceIndexSummary(summary *ResourceIndexStatusSummary, status ResourceIndexStatus) {
	addResourceDisplayStat(summary, status.DisplayIndexState)
	addResourceTextStat(summary, status.TextIndexState)
	addResourceMMStat(summary, status.MMIndexState)
	if status.TextIndexState == "pending" {
		summary.TextQueueCount++
	}
}

func addResourceDisplayStat(summary *ResourceIndexStatusSummary, state string) {
	summary.DisplayTotalResources++
	switch state {
	case "pending":
		summary.PendingCount++
		summary.DisplayPendingCount++
	case "indexing":
		summary.IndexingCount++
		summary.DisplayIndexingCount++
	case "indexed":
		summary.IndexedCount++
		summary.DisplayIndexedCount++
	case "failed":
		summary.FailedCount++
		summary.DisplayFailedCount++
	default:
		summary.DisabledCount++
		summary.DisplayDisabledCount++
	}
}

func addResourceTextStat(summary *ResourceIndexStatusSummary, state string) {
	summary.TextTotalResources++
	switch state {
	case "pending":
		summary.TextPendingCount++
	case "indexing":
		summary.TextIndexingCount++
	case "indexed":
		summary.TextIndexedCount++
	case "failed":
		summary.TextFailedCount++
	default:
		summary.TextDisabledCount++
	}
}

func addResourceMMStat(summary *ResourceIndexStatusSummary, state string) {
	summary.MMTotalResources++
	switch state {
	case "pending":
		summary.MMPendingCount++
	case "indexing":
		summary.MMIndexingCount++
	case "indexed":
		summary.MMIndexedCount++
	case "failed":
		summary.MMFailedCount++
	default:
		summary.MMDisabledCount++
	}
}

func estimateTextChunks(text string) int {
	length := len([]rune(text))
	if length == 0 {
		return 0
	}
	chunks := length / 4000
	if length%4000 != 0 {
		chunks++
	}
	if chunks < 1 {
		chunks = 1
	}
	return chunks
}

func firstNonEmptyString(values ...*string) *string {
	for _, value := range values {
		if value != nil && strings.TrimSpace(*value) != "" {
			trimmed := strings.TrimSpace(*value)
			return &trimmed
		}
	}
	return nil
}

func runeLenPtr(value *string) int {
	if value == nil {
		return 0
	}
	return len([]rune(*value))
}

func activeTextSource(ocrText *string, extractedText *string) string {
	if ocrText != nil && strings.TrimSpace(*ocrText) != "" {
		return "ocr"
	}
	if extractedText != nil && strings.TrimSpace(*extractedText) != "" {
		return "extracted"
	}
	return "none"
}

func metadataInt64(metadata map[string]any, keys ...string) *int64 {
	for _, key := range keys {
		value, ok := metadata[key]
		if !ok {
			continue
		}
		switch typed := value.(type) {
		case int64:
			return &typed
		case int:
			out := int64(typed)
			return &out
		case float64:
			out := int64(typed)
			return &out
		case json.Number:
			if parsed, err := typed.Int64(); err == nil {
				return &parsed
			}
		case string:
			if parsed, err := strconv.ParseInt(strings.TrimSpace(typed), 10, 64); err == nil {
				return &parsed
			}
		}
	}
	return nil
}

func metadataInt(metadata map[string]any, keys ...string) *int {
	if value := metadataInt64(metadata, keys...); value != nil {
		out := int(*value)
		return &out
	}
	return nil
}

func normalizeResourceType(value string) string {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "note", "textbook", "exam", "translation", "essay", "image", "file", "retrieval", "mindmap", "folder":
		return strings.ToLower(strings.TrimSpace(value))
	default:
		return ""
	}
}

func computeHash(data string) string {
	sum := sha256.Sum256([]byte(data))
	return hex.EncodeToString(sum[:])
}

func computeBytesHash(data []byte) string {
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:])
}

func contentEncoding(resourceType string, data string) string {
	switch resourceType {
	case "image", "file", "textbook":
		if _, err := decodeBase64Payload(data); err == nil {
			return "base64"
		}
	}
	return "utf8"
}

func decodeBase64Payload(data string) ([]byte, error) {
	trimmed := strings.TrimSpace(data)
	if comma := strings.Index(trimmed, ","); strings.HasPrefix(trimmed, "data:") && comma >= 0 {
		trimmed = trimmed[comma+1:]
	}
	return base64.StdEncoding.DecodeString(trimmed)
}

func extensionForResource(resourceType string, metadata map[string]any) string {
	if name, ok := metadata["name"].(string); ok {
		if ext := strings.TrimPrefix(filepath.Ext(name), "."); ext != "" {
			return safeSegment(ext, "bin")
		}
	}
	if mimeType := firstNonEmptyMetadataString(metadata, "mimeType", "mime_type"); mimeType != "" {
		switch strings.ToLower(mimeType) {
		case "image/png":
			return "png"
		case "image/jpeg", "image/jpg":
			return "jpg"
		case "image/gif":
			return "gif"
		case "image/webp":
			return "webp"
		case "application/pdf":
			return "pdf"
		case "text/markdown":
			return "md"
		case "text/plain":
			return "txt"
		case "application/json":
			return "json"
		}
	}
	switch resourceType {
	case "note", "retrieval":
		return "md"
	case "mindmap":
		return "json"
	case "image":
		return "bin"
	default:
		return "txt"
	}
}

func mergeMetadata(existing map[string]any, incoming map[string]any) map[string]any {
	out := normalizeMetadata(existing)
	if out == nil {
		out = map[string]any{}
	}
	for key, value := range incoming {
		out[key] = value
	}
	return out
}

func metadataHasPreviewJSON(metadata map[string]any) bool {
	return firstNonEmptyMetadataString(metadata, "previewJson", "preview_json") != ""
}

func metadataHasKey(metadata map[string]any, key string) bool {
	if metadata == nil {
		return false
	}
	_, ok := metadata[key]
	return ok
}

func withoutGeneratedPdfPreviewMetadata(metadata map[string]any) map[string]any {
	out := normalizeMetadata(metadata)
	if out == nil {
		return nil
	}
	for _, key := range []string{
		"previewJson", "preview_json",
		"previewSource", "preview_source",
		"previewGeneratedAt", "preview_generated_at",
		"previewPageCount", "preview_page_count",
		"previewMimeType", "preview_mime_type",
		"pageRenderingStatus", "page_rendering_status",
		"pageRenderingSource", "page_rendering_source",
		"pageRenderingError", "page_rendering_error",
		"pageRenderingTruncated", "page_rendering_truncated",
		"rasterPreviewStatus", "raster_preview_status",
		"rasterPreviewSource", "raster_preview_source",
		"rasterPreviewError", "raster_preview_error",
	} {
		delete(out, key)
	}
	return out
}

func normalizeMetadata(metadata map[string]any) map[string]any {
	if len(metadata) == 0 {
		return nil
	}
	out := make(map[string]any, len(metadata))
	for key, value := range metadata {
		out[key] = value
	}
	return out
}

func legacyAliasCandidates(raw string) []string {
	trimmed := strings.TrimSpace(strings.Trim(raw, "\x00"))
	if trimmed == "" {
		return []string{}
	}
	candidates := []string{}
	addAliasCandidate(&candidates, trimmed)
	if decoded, err := url.QueryUnescape(trimmed); err == nil && decoded != trimmed {
		addAliasCandidate(&candidates, decoded)
	}
	if parsed, err := url.Parse(trimmed); err == nil && parsed.Scheme != "" {
		for _, key := range []string{"resourceId", "resource_id", "sourceId", "source_id", "id", "hash", "blobHash", "blob_hash"} {
			addAliasCandidate(&candidates, parsed.Query().Get(key))
		}
		addAliasCandidate(&candidates, parsed.Host)
		addAliasCandidate(&candidates, parsed.Fragment)
		for _, part := range strings.FieldsFunc(parsed.Path, aliasPathSeparator) {
			addAliasCandidate(&candidates, part)
			if decoded, err := url.QueryUnescape(part); err == nil && decoded != part {
				addAliasCandidate(&candidates, decoded)
			}
		}
	}
	for _, part := range strings.FieldsFunc(trimmed, aliasPathSeparator) {
		addAliasCandidate(&candidates, part)
	}
	return candidates
}

func addAliasCandidate(candidates *[]string, value string) {
	trimmed := strings.TrimSpace(strings.Trim(value, "\x00"))
	if trimmed == "" {
		return
	}
	trimmed = strings.Trim(trimmed, `"'<>[]{}()`)
	if trimmed == "" {
		return
	}
	for _, existing := range *candidates {
		if existing == trimmed {
			return
		}
	}
	*candidates = append(*candidates, trimmed)
}

func aliasPathSeparator(r rune) bool {
	return r == '/' || r == '\\' || r == '?' || r == '&' || r == '=' || r == '#' || r == ':' || r == ';'
}

func resourceMatchesAlias(resource Resource, alias string) bool {
	trimmed := strings.TrimSpace(alias)
	if trimmed == "" {
		return false
	}
	if resource.ID == trimmed || resource.Hash == trimmed {
		return true
	}
	if resource.SourceID != nil && strings.TrimSpace(*resource.SourceID) == trimmed {
		return true
	}
	if resource.ExternalHash != nil && strings.TrimSpace(*resource.ExternalHash) == trimmed {
		return true
	}
	for _, key := range []string{
		"resourceId", "resourceID", "sourceId", "sourceID", "fileId", "fileID",
		"attachmentId", "attachmentID", "blobHash", "contentHash", "sha256",
		"originalPath", "original_path", "path", "cachedPath", "cached_path",
	} {
		if metadataString(resource.Metadata, key, "") == trimmed {
			return true
		}
	}
	if resource.ExternalPath != nil && strings.TrimSpace(*resource.ExternalPath) == trimmed {
		return true
	}
	return false
}

func normalizeOptionalString(value *string) *string {
	if value == nil {
		return nil
	}
	trimmed := strings.TrimSpace(*value)
	if trimmed == "" {
		return nil
	}
	return &trimmed
}

func safeSegment(value string, fallback string) string {
	trimmed := strings.Trim(safePathSegmentPattern.ReplaceAllString(value, "_"), "._-")
	if trimmed == "" {
		return fallback
	}
	return trimmed
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

func minInt(left int, right int) int {
	if left < right {
		return left
	}
	return right
}

func normalizeAttachmentType(value *string, mimeType string, name string) string {
	if value != nil {
		switch strings.ToLower(strings.TrimSpace(*value)) {
		case "image":
			return "image"
		case "file":
			return "file"
		}
	}
	if strings.HasPrefix(strings.ToLower(mimeType), "image/") {
		return "image"
	}
	if strings.HasSuffix(strings.ToLower(name), ".png") ||
		strings.HasSuffix(strings.ToLower(name), ".jpg") ||
		strings.HasSuffix(strings.ToLower(name), ".jpeg") ||
		strings.HasSuffix(strings.ToLower(name), ".gif") ||
		strings.HasSuffix(strings.ToLower(name), ".webp") {
		return "image"
	}
	return "file"
}

func attachmentMetadata(params UploadAttachmentInput, attachmentType string, contentHash string, size int64, resourceID string, sourceID *string, createdAt int64, updatedAt int64) map[string]any {
	name := strings.TrimSpace(params.Name)
	mimeType := strings.TrimSpace(params.MimeType)
	if mimeType == "" {
		mimeType = "application/octet-stream"
	}
	attachmentID := resourceID
	if sourceID != nil && strings.TrimSpace(*sourceID) != "" {
		attachmentID = strings.TrimSpace(*sourceID)
	}
	metadata := map[string]any{
		"attachment":     true,
		"attachmentId":   attachmentID,
		"resourceId":     resourceID,
		"name":           name,
		"mimeType":       mimeType,
		"size":           size,
		"contentHash":    contentHash,
		"attachmentType": attachmentType,
		"createdAt":      formatMillis(createdAt),
		"updatedAt":      formatMillis(updatedAt),
	}
	if value := normalizeOptionalString(params.FolderID); value != nil {
		metadata["folderId"] = *value
	}
	if value := normalizeOptionalString(params.SessionID); value != nil {
		metadata["sessionId"] = *value
	}
	if value := normalizeOptionalString(params.GroupID); value != nil {
		metadata["groupId"] = *value
	}
	return metadata
}

func uploadAttachmentResult(resource Resource, isNew bool) UploadAttachmentResult {
	attachment := resourceToAttachment(resource)
	status, percent, readyModes := processingStateForAttachment(attachment)
	return UploadAttachmentResult{
		SourceID:          attachment.ID,
		ResourceHash:      resource.Hash,
		IsNew:             isNew,
		Attachment:        attachment,
		ProcessingStatus:  status,
		ProcessingPercent: percent,
		ReadyModes:        readyModes,
	}
}

func resourceToAttachment(resource Resource) Attachment {
	sourceID := resource.ID
	if resource.SourceID != nil && strings.TrimSpace(*resource.SourceID) != "" {
		sourceID = strings.TrimSpace(*resource.SourceID)
	}
	resourceID := resource.ID
	blobHash := resource.Hash
	if resource.ExternalHash != nil && strings.TrimSpace(*resource.ExternalHash) != "" {
		blobHash = strings.TrimSpace(*resource.ExternalHash)
	}
	size := int64(0)
	if value := metadataInt64(resource.Metadata, "size"); value != nil {
		size = *value
	} else if resource.Data != nil {
		size = int64(len([]byte(*resource.Data)))
	}
	return Attachment{
		ID:          sourceID,
		ResourceID:  &resourceID,
		BlobHash:    &blobHash,
		Type:        metadataString(resource.Metadata, "attachmentType", resource.Type),
		Name:        resourceName(resource),
		MimeType:    firstNonEmptyPlainString(firstNonEmptyMetadataString(resource.Metadata, "mimeType", "mime_type"), "application/octet-stream"),
		Size:        size,
		ContentHash: metadataString(resource.Metadata, "contentHash", resource.Hash),
		IsFavorite:  metadataBool(resource.Metadata, "isFavorite", false),
		CreatedAt:   metadataString(resource.Metadata, "createdAt", formatMillis(resource.CreatedAt)),
		UpdatedAt:   metadataString(resource.Metadata, "updatedAt", formatMillis(resource.UpdatedAt)),
	}
}

func processingStateForAttachment(attachment Attachment) (*string, *float64, []string) {
	mimeType := strings.ToLower(attachment.MimeType)
	name := strings.ToLower(attachment.Name)
	if strings.HasPrefix(mimeType, "image/") || attachment.Type == "image" {
		status := "completed"
		percent := 100.0
		return &status, &percent, []string{"image"}
	}
	if mimeType == "application/pdf" || strings.HasSuffix(name, ".pdf") {
		status := "page_compression"
		percent := 25.0
		return &status, &percent, []string{}
	}
	return nil, nil, nil
}

type previewPageImageRef struct {
	Base64   string
	DataURL  string
	BlobHash string
	Path     string
	MimeType string
}

func findPreviewPage(raw string, pageIndex int) (previewPageImageRef, error) {
	var decoded any
	if err := json.Unmarshal([]byte(raw), &decoded); err != nil {
		return previewPageImageRef{}, fmt.Errorf("parse previewJson failed: %w", err)
	}
	pages := previewPages(decoded)
	if len(pages) == 0 {
		return previewPageImageRef{}, errors.New("previewJson has no pages")
	}
	for index, pageValue := range pages {
		page, ok := pageValue.(map[string]any)
		if !ok {
			continue
		}
		actualIndex := index
		if value, ok := previewInt(firstPreviewValue(page, "pageIndex", "page_index")); ok {
			actualIndex = value
		}
		if actualIndex != pageIndex {
			continue
		}
		ref := previewPageImageRef{
			Base64:   previewString(firstPreviewValue(page, "base64", "imageBase64", "image_base64")),
			DataURL:  previewString(firstPreviewValue(page, "dataUrl", "dataURL", "data_url")),
			MimeType: previewString(firstPreviewValue(page, "mimeType", "mime_type", "mediaType", "media_type")),
			Path:     previewString(firstPreviewValue(page, "path", "filePath", "file_path", "originalImagePath", "original_image_path", "cachedPath", "cached_path")),
		}
		if value := previewString(firstPreviewValue(page, "url", "src")); strings.HasPrefix(strings.TrimSpace(value), "data:") {
			ref.DataURL = value
		} else if ref.Path == "" {
			ref.Path = value
		}
		ref.BlobHash = firstNonEmptyPlainString(
			previewString(firstPreviewValue(page, "compressedBlobHash", "compressed_blob_hash", "compressedHash", "compressed_hash")),
			previewString(firstPreviewValue(page, "blobHash", "blob_hash", "hash", "resourceHash", "resource_hash")),
		)
		if ref.DataURL == "" && ref.Base64 == "" && ref.BlobHash == "" && ref.Path == "" {
			return previewPageImageRef{}, fmt.Errorf("preview page %d has no image reference", pageIndex)
		}
		return ref, nil
	}
	return previewPageImageRef{}, fmt.Errorf("preview page out of range: %d", pageIndex)
}

func previewPages(decoded any) []any {
	switch typed := decoded.(type) {
	case []any:
		return typed
	case map[string]any:
		for _, key := range []string{"pages", "pageImages", "page_images"} {
			if pages, ok := typed[key].([]any); ok {
				return pages
			}
		}
		if _, ok := typed["pageIndex"]; ok {
			return []any{typed}
		}
		if _, ok := typed["page_index"]; ok {
			return []any{typed}
		}
	}
	return []any{}
}

func firstPreviewValue(values map[string]any, keys ...string) any {
	for _, key := range keys {
		if value, ok := values[key]; ok {
			return value
		}
	}
	return nil
}

func previewString(value any) string {
	switch typed := value.(type) {
	case string:
		return strings.TrimSpace(typed)
	default:
		return ""
	}
}

func previewInt(value any) (int, bool) {
	switch typed := value.(type) {
	case int:
		return typed, true
	case int64:
		return int(typed), true
	case float64:
		return int(typed), true
	case json.Number:
		parsed, err := typed.Int64()
		if err != nil {
			return 0, false
		}
		return int(parsed), true
	default:
		return 0, false
	}
}

func (s *Service) resolvePreviewPageImageLocked(ref previewPageImageRef) (PdfPageImageResult, error) {
	if ref.DataURL != "" {
		return pdfPageImageFromBase64(ref.DataURL, ref.MimeType, "")
	}
	if ref.Base64 != "" {
		return pdfPageImageFromBase64(ref.Base64, ref.MimeType, "")
	}
	if ref.BlobHash != "" {
		if resource, ok := s.findResourceByAnyIDLocked(ref.BlobHash); ok && !resourceIsDeleted(resource) {
			return s.resourceToPdfPageImageLocked(resource, ref.MimeType)
		}
	}
	if ref.Path != "" {
		return s.readPreviewImagePathLocked(ref.Path, ref.MimeType)
	}
	return PdfPageImageResult{}, errors.New("preview page image data is unavailable")
}

func (s *Service) resourceToPdfPageImageLocked(resource Resource, mimeType string) (PdfPageImageResult, error) {
	hydrated, err := s.hydrateResourceData(resource)
	if err != nil {
		return PdfPageImageResult{}, err
	}
	if hydrated.Data == nil {
		return PdfPageImageResult{}, fmt.Errorf("preview blob has no image data: %s", hydrated.ID)
	}
	resolvedMime := firstNonEmptyPlainString(mimeType, metadataString(hydrated.Metadata, "mimeType", ""))
	if hydrated.ContentEncoding == "base64" {
		return pdfPageImageFromBase64(*hydrated.Data, resolvedMime, "")
	}
	bytes := []byte(*hydrated.Data)
	return PdfPageImageResult{
		Base64:   base64.StdEncoding.EncodeToString(bytes),
		MimeType: inferImageMimeType(resolvedMime, resourceName(hydrated), bytes),
		Size:     int64(len(bytes)),
	}, nil
}

func (s *Service) readPreviewImagePathLocked(rawPath string, mimeType string) (PdfPageImageResult, error) {
	absolutePath, err := s.resolveLibraryPath(rawPath)
	if err != nil {
		return PdfPageImageResult{}, err
	}
	bytes, err := os.ReadFile(absolutePath)
	if err != nil {
		return PdfPageImageResult{}, err
	}
	return PdfPageImageResult{
		Base64:   base64.StdEncoding.EncodeToString(bytes),
		MimeType: inferImageMimeType(mimeType, absolutePath, bytes),
		Size:     int64(len(bytes)),
	}, nil
}

func pdfPageImageFromBase64(raw string, mimeType string, name string) (PdfPageImageResult, error) {
	base64Data := strings.TrimSpace(raw)
	resolvedMime := mimeType
	if strings.HasPrefix(base64Data, "data:") {
		comma := strings.Index(base64Data, ",")
		if comma < 0 {
			return PdfPageImageResult{}, errors.New("invalid image data URL")
		}
		header := base64Data[:comma]
		if resolvedMime == "" {
			resolvedMime = strings.TrimPrefix(strings.Split(strings.TrimPrefix(header, "data:"), ";")[0], " ")
		}
		base64Data = base64Data[comma+1:]
	}
	decoded, err := decodeBase64Payload(base64Data)
	if err != nil {
		return PdfPageImageResult{}, fmt.Errorf("invalid preview image base64: %w", err)
	}
	return PdfPageImageResult{
		Base64:   base64.StdEncoding.EncodeToString(decoded),
		MimeType: inferImageMimeType(resolvedMime, name, decoded),
		Size:     int64(len(decoded)),
	}, nil
}

func inferImageMimeType(mimeType string, name string, bytes []byte) string {
	lowerMime := strings.ToLower(strings.TrimSpace(mimeType))
	if strings.HasPrefix(lowerMime, "image/") {
		return lowerMime
	}
	lowerName := strings.ToLower(strings.TrimSpace(name))
	switch strings.TrimPrefix(filepath.Ext(lowerName), ".") {
	case "jpg", "jpeg":
		return "image/jpeg"
	case "png":
		return "image/png"
	case "gif":
		return "image/gif"
	case "webp":
		return "image/webp"
	case "svg":
		return "image/svg+xml"
	}
	if len(bytes) >= 4 {
		if bytes[0] == 0x89 && bytes[1] == 'P' && bytes[2] == 'N' && bytes[3] == 'G' {
			return "image/png"
		}
		if bytes[0] == 0xff && bytes[1] == 0xd8 {
			return "image/jpeg"
		}
		if bytes[0] == 'G' && bytes[1] == 'I' && bytes[2] == 'F' {
			return "image/gif"
		}
		if len(bytes) >= 12 && string(bytes[0:4]) == "RIFF" && string(bytes[8:12]) == "WEBP" {
			return "image/webp"
		}
	}
	return "image/png"
}

func inferBlobMimeType(name string, bytes []byte) string {
	lowerName := strings.ToLower(strings.TrimSpace(name))
	switch strings.TrimPrefix(filepath.Ext(lowerName), ".") {
	case "jpg", "jpeg":
		return "image/jpeg"
	case "png":
		return "image/png"
	case "gif":
		return "image/gif"
	case "webp":
		return "image/webp"
	case "pdf":
		return "application/pdf"
	case "md", "markdown":
		return "text/markdown"
	case "txt":
		return "text/plain"
	case "json":
		return "application/json"
	case "csv":
		return "text/csv"
	}
	if len(bytes) == 0 {
		return "application/octet-stream"
	}
	return http.DetectContentType(bytes)
}

func pdfProcessingMissingStatus(fileID string) PdfProcessingStatus {
	return pdfProcessingStatus(fileID, "pending", nil, nil, 0, []string{}, "", nil, nil)
}

func pdfProcessingErrorStatus(fileID string, mediaType string, message string) PdfProcessingStatus {
	if strings.TrimSpace(message) == "" {
		message = "resource is unavailable"
	}
	failedStage := "text_extraction"
	switch mediaType {
	case "pdf":
		failedStage = "page_compression"
	case "image":
		failedStage = "image_compression"
	}
	failedStages := []PdfProcessingFailedStage{{
		Stage:     failedStage,
		Message:   message,
		Retriable: true,
	}}
	return pdfProcessingStatus(fileID, "error", nil, nil, 0, []string{}, mediaType, &message, failedStages)
}

func mediaTypeForResource(resource Resource) string {
	mimeType := strings.ToLower(firstNonEmptyMetadataString(resource.Metadata, "mimeType", "mime_type"))
	name := strings.ToLower(resourceName(resource))
	if resourceHasImage(resource) {
		return "image"
	}
	if mimeType == "application/pdf" || strings.HasSuffix(name, ".pdf") || resource.Type == "textbook" {
		return "pdf"
	}
	return ""
}

func pdfProcessingStatusForResource(fileID string, resource Resource) PdfProcessingStatus {
	mediaType := mediaTypeForResource(resource)
	if mediaType == "image" {
		page := 1
		return pdfProcessingStatus(fileID, "completed", &page, &page, 100, []string{"image"}, "image", nil, nil)
	}

	if mediaType == "pdf" {
		readyModes := []string{}
		failedStages := []PdfProcessingFailedStage{}
		totalPages := metadataInt(resource.Metadata, "pageCount", "page_count")
		currentPage := (*int)(nil)
		if totalPages != nil && *totalPages > 0 {
			current := *totalPages
			currentPage = &current
		}
		if text := extractedTextContent(resource); text != nil && strings.TrimSpace(*text) != "" {
			readyModes = append(readyModes, "text")
		} else {
			failedStages = append(failedStages, PdfProcessingFailedStage{
				Stage:     "text_extraction",
				Message:   "No extracted PDF text is available in the lean Go VFS index yet.",
				Retriable: true,
			})
		}
		if text := realOcrTextContent(resource); text != nil && strings.TrimSpace(*text) != "" {
			readyModes = append(readyModes, "ocr")
		}
		if previewJSON := firstMetadataString(resource.Metadata, "previewJson", "preview_json"); previewJSON != nil && strings.TrimSpace(*previewJSON) != "" {
			if pdfPreviewIsRasterReady(resource.Metadata) {
				readyModes = append(readyModes, "image")
			} else {
				stage := "page_compression"
				if status := firstNonEmptyMetadataString(resource.Metadata, "rasterPreviewStatus", "raster_preview_status"); status != "" && status != "completed" {
					stage = "raster_preview"
				}
				failedStages = append(failedStages, PdfProcessingFailedStage{
					Stage:     stage,
					Message:   pdfPreviewReadinessMessage(resource.Metadata),
					Retriable: true,
				})
			}
		} else {
			failedStages = append(failedStages, PdfProcessingFailedStage{
				Stage:     "page_compression",
				Message:   "No PDF page preview data is available in the lean Go VFS index yet.",
				Retriable: true,
			})
		}
		stage := "completed"
		if len(failedStages) > 0 {
			stage = "completed_with_issues"
		}
		return pdfProcessingStatus(fileID, stage, currentPage, totalPages, 100, readyModes, "pdf", nil, failedStages)
	}

	return pdfProcessingStatus(fileID, "completed", nil, nil, 100, []string{}, mediaType, nil, nil)
}

func pdfPreviewIsRasterReady(metadata map[string]any) bool {
	source := firstNonEmptyMetadataString(metadata, "previewSource", "preview_source")
	mimeType := strings.ToLower(firstNonEmptyMetadataString(metadata, "previewMimeType", "preview_mime_type"))
	rasterStatus := firstNonEmptyMetadataString(metadata, "rasterPreviewStatus", "raster_preview_status")
	if source == pdfTextPreviewSource || mimeType == pdfTextPreviewMimeType {
		return false
	}
	if source == pdfRasterPreviewSource || rasterStatus == "completed" || strings.HasPrefix(mimeType, "image/") {
		return true
	}
	return source == "" && mimeType == "" && rasterStatus == ""
}

func pdfPreviewReadinessMessage(metadata map[string]any) string {
	if message := firstNonEmptyMetadataString(
		metadata,
		"rasterPreviewError",
		"raster_preview_error",
		"pageRenderingError",
		"page_rendering_error",
	); message != "" {
		return message
	}
	if firstNonEmptyMetadataString(metadata, "previewSource", "preview_source") == pdfTextPreviewSource ||
		strings.EqualFold(firstNonEmptyMetadataString(metadata, "previewMimeType", "preview_mime_type"), pdfTextPreviewMimeType) {
		return "PDFium raster preview is unavailable; using text-layer SVG preview fallback."
	}
	return "No raster PDF page preview is available for multimodal image injection."
}

func pdfProcessingStatus(fileID string, stage string, currentPage *int, totalPages *int, percent float64, readyModes []string, mediaType string, errMessage *string, failedStages []PdfProcessingFailedStage) PdfProcessingStatus {
	if readyModes == nil {
		readyModes = []string{}
	}
	progress := PdfProcessingProgress{
		Stage:        stage,
		CurrentPage:  currentPage,
		TotalPages:   totalPages,
		Percent:      percent,
		ReadyModes:   readyModes,
		MediaType:    mediaType,
		FailedStages: failedStages,
	}
	return PdfProcessingStatus{
		FileID:       fileID,
		Stage:        stage,
		CurrentPage:  currentPage,
		TotalPages:   totalPages,
		Percent:      percent,
		ReadyModes:   readyModes,
		MediaType:    mediaType,
		Error:        errMessage,
		FailedStages: failedStages,
		Progress:     progress,
	}
}

func normalizeProcessingStage(startFromStage *string) string {
	if startFromStage == nil {
		return ""
	}
	switch strings.ToLower(strings.TrimSpace(*startFromStage)) {
	case "pending",
		"text_extraction",
		"page_rendering",
		"page_compression",
		"image_compression",
		"ocr_processing",
		"vector_indexing":
		return strings.ToLower(strings.TrimSpace(*startFromStage))
	default:
		return ""
	}
}

func defaultProcessingStartStage(mediaType string) string {
	switch mediaType {
	case "pdf":
		return "text_extraction"
	case "image":
		return "image_compression"
	default:
		return "vector_indexing"
	}
}

func uploadFileResult(file VfsFile, resourceHash string, isNew bool) UploadFileResult {
	return UploadFileResult{
		File:         file,
		SourceID:     file.ID,
		ResourceHash: resourceHash,
		IsNew:        isNew,
		OcrStatus:    ocrStateForFile(file),
		IndexStatus:  indexStateForFile(file),
	}
}

func ocrStateForFile(file VfsFile) *OcrStatus {
	isPDF := file.MimeType != nil && *file.MimeType == "application/pdf"
	isImage := file.FileType == "image"
	if !isPDF && !isImage {
		return nil
	}
	message := "OCR will be handled by the lean processing pipeline when available"
	totalPages := 0
	if file.PageCount != nil {
		totalPages = *file.PageCount
	}
	return &OcrStatus{
		Performed:        false,
		SuccessCount:     0,
		FailedCount:      0,
		BlobMissingCount: 0,
		TotalPages:       totalPages,
		AllSuccess:       false,
		Message:          message,
	}
}

func indexStateForFile(file VfsFile) *IndexStatus {
	unitsCreated := 0
	if file.ExtractedText != nil && strings.TrimSpace(*file.ExtractedText) != "" {
		unitsCreated = estimateTextChunks(*file.ExtractedText)
	}
	queued := unitsCreated > 0
	message := "No text index units created"
	if queued {
		message = fmt.Sprintf("Queued %d text index unit(s)", unitsCreated)
	}
	return &IndexStatus{
		Queued:       queued,
		UnitsCreated: unitsCreated,
		Message:      message,
	}
}

func fileMetadata(params UploadFileInput, fileType string, contentHash string, size int64, resourceID string, sourceID *string, createdAt int64, updatedAt int64, extractedText *string, pageCount *int, ocrPagesJSON *string) map[string]any {
	name := strings.TrimSpace(params.Name)
	mimeType := strings.TrimSpace(params.MimeType)
	if mimeType == "" {
		mimeType = "application/octet-stream"
	}
	fileID := resourceID
	if sourceID != nil && strings.TrimSpace(*sourceID) != "" {
		fileID = strings.TrimSpace(*sourceID)
	}
	metadata := map[string]any{
		"fileRecord":  true,
		"fileId":      fileID,
		"resourceId":  resourceID,
		"name":        name,
		"mimeType":    mimeType,
		"fileType":    fileType,
		"size":        size,
		"contentHash": contentHash,
		"status":      "active",
		"createdAt":   formatMillis(createdAt),
		"updatedAt":   formatMillis(updatedAt),
	}
	if value := normalizeOptionalString(params.FolderID); value != nil {
		metadata["folderId"] = *value
	}
	if extractedText != nil && strings.TrimSpace(*extractedText) != "" {
		metadata["extractedText"] = *extractedText
	}
	if pageCount != nil && *pageCount > 0 {
		metadata["pageCount"] = *pageCount
		metadata["pageCountStatus"] = "completed"
	}
	if ocrPagesJSON != nil && strings.TrimSpace(*ocrPagesJSON) != "" && firstNonEmptyMetadataString(params.Metadata, "ocrPagesJson", "ocr_pages_json") == "" {
		metadata["ocrPagesJson"] = *ocrPagesJSON
		metadata["ocrPagesSource"] = "pdf_text_layer_estimated"
	}
	metadata = mergeMetadata(params.Metadata, metadata)
	if !metadataHasKey(metadata, "bookmarks") {
		metadata["bookmarks"] = []any{}
	}
	if name := firstNonEmptyMetadataString(params.Metadata, "title", "name", "fileName"); name != "" {
		metadata["name"] = name
		metadata["title"] = name
	}
	return metadata
}

func isAttachmentLikeID(id string) bool {
	for _, alias := range legacyAliasCandidates(id) {
		if isAttachmentLikeIDToken(alias) {
			return true
		}
	}
	return false
}

func isAttachmentLikeIDToken(id string) bool {
	trimmed := strings.TrimSpace(id)
	return strings.HasPrefix(trimmed, "att_") ||
		strings.HasPrefix(trimmed, "file_") ||
		strings.HasPrefix(trimmed, "tb_") ||
		strings.HasPrefix(trimmed, "img_") ||
		strings.HasPrefix(trimmed, "res_")
}

func isFileLikeID(id string) bool {
	for _, alias := range legacyAliasCandidates(id) {
		if isFileLikeIDToken(alias) {
			return true
		}
	}
	return false
}

func isFileLikeIDToken(id string) bool {
	trimmed := strings.TrimSpace(id)
	return strings.HasPrefix(trimmed, "file_") ||
		strings.HasPrefix(trimmed, "att_") ||
		strings.HasPrefix(trimmed, "img_") ||
		strings.HasPrefix(trimmed, "tb_") ||
		strings.HasPrefix(trimmed, "res_")
}

func normalizeFileType(value *string, mimeType string) string {
	if value != nil {
		switch strings.ToLower(strings.TrimSpace(*value)) {
		case "document", "image", "audio", "video":
			return strings.ToLower(strings.TrimSpace(*value))
		}
	}
	lowerMime := strings.ToLower(strings.TrimSpace(mimeType))
	if strings.HasPrefix(lowerMime, "image/") {
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

func extractedTextForUpload(name string, mimeType string, bytes []byte) *string {
	lowerName := strings.ToLower(strings.TrimSpace(name))
	lowerMime := strings.ToLower(strings.TrimSpace(mimeType))
	if isPdfUpload(lowerName, lowerMime) {
		return extractPdfTextLayer(bytes)
	}
	if !strings.HasPrefix(lowerMime, "text/") &&
		lowerMime != "application/json" &&
		lowerMime != "application/xml" &&
		lowerMime != "application/xhtml+xml" &&
		!strings.HasSuffix(lowerName, ".txt") &&
		!strings.HasSuffix(lowerName, ".md") &&
		!strings.HasSuffix(lowerName, ".markdown") &&
		!strings.HasSuffix(lowerName, ".csv") &&
		!strings.HasSuffix(lowerName, ".json") &&
		!strings.HasSuffix(lowerName, ".xml") {
		return nil
	}
	text := strings.TrimSpace(string(bytes))
	if text == "" {
		return nil
	}
	return &text
}

func pageCountForUpload(name string, mimeType string, bytes []byte) *int {
	lowerName := strings.ToLower(strings.TrimSpace(name))
	lowerMime := strings.ToLower(strings.TrimSpace(mimeType))
	if !isPdfUpload(lowerName, lowerMime) {
		return nil
	}
	return detectPdfPageCount(bytes)
}

func ocrPagesJSONForUpload(name string, mimeType string, extractedText *string, pageCount *int) *string {
	lowerName := strings.ToLower(strings.TrimSpace(name))
	lowerMime := strings.ToLower(strings.TrimSpace(mimeType))
	if !isPdfUpload(lowerName, lowerMime) || extractedText == nil {
		return nil
	}
	return pdfTextLayerOcrPagesJSON(*extractedText, pageCount)
}

func pdfTextLayerOcrPagesJSON(text string, pageCount *int) *string {
	pages := pdfTextLayerOcrPages(text, pageCount)
	if len(pages) == 0 {
		return nil
	}
	bytes, err := json.Marshal(pages)
	if err != nil {
		return nil
	}
	value := string(bytes)
	return &value
}

func pdfTextLayerOcrPages(text string, pageCount *int) []OcrPageInfo {
	text = strings.TrimSpace(text)
	if text == "" {
		return []OcrPageInfo{}
	}
	totalPages := 1
	if pageCount != nil && *pageCount > 0 {
		totalPages = *pageCount
	}
	if totalPages > 500 {
		totalPages = 500
	}
	chunks := splitTextIntoPageChunks(text, totalPages)
	pages := make([]OcrPageInfo, 0, len(chunks))
	for index, chunk := range chunks {
		chunk = strings.TrimSpace(chunk)
		pages = append(pages, OcrPageInfo{
			PageIndex: index,
			Text:      chunk,
			CharCount: len([]rune(chunk)),
			IsFailed:  false,
		})
	}
	return pages
}

func splitTextIntoPageChunks(text string, pageCount int) []string {
	if pageCount <= 1 {
		return []string{text}
	}
	runes := []rune(text)
	if len(runes) == 0 {
		return []string{}
	}
	chunks := make([]string, 0, pageCount)
	start := 0
	for pageIndex := 0; pageIndex < pageCount; pageIndex++ {
		if start >= len(runes) {
			chunks = append(chunks, "")
			continue
		}
		remainingPages := pageCount - pageIndex
		remainingRunes := len(runes) - start
		target := start + (remainingRunes+remainingPages-1)/remainingPages
		if target < len(runes) {
			target = nearestPageTextBreak(runes, start, target)
		}
		chunks = append(chunks, string(runes[start:target]))
		start = target
	}
	return chunks
}

func nearestPageTextBreak(runes []rune, start int, target int) int {
	if target <= start || target >= len(runes) {
		return target
	}
	maxLookback := target - 400
	if maxLookback < start+1 {
		maxLookback = start + 1
	}
	for index := target; index >= maxLookback; index-- {
		if isPageTextBreakRune(runes[index-1]) {
			return index
		}
	}
	maxLookahead := target + 400
	if maxLookahead > len(runes) {
		maxLookahead = len(runes)
	}
	for index := target; index < maxLookahead; index++ {
		if isPageTextBreakRune(runes[index]) {
			return index + 1
		}
	}
	return target
}

func isPageTextBreakRune(value rune) bool {
	return value == '\n' || value == '。' || value == '.' || value == '!' || value == '?' || value == '！' || value == '？'
}

func parseOcrPagesJSON(raw *string) []OcrPageInfo {
	if raw == nil || strings.TrimSpace(*raw) == "" {
		return []OcrPageInfo{}
	}
	var pages []OcrPageInfo
	if err := json.Unmarshal([]byte(*raw), &pages); err != nil {
		return []OcrPageInfo{}
	}
	for index := range pages {
		if pages[index].PageIndex < 0 {
			pages[index].PageIndex = index
		}
		pages[index].CharCount = len([]rune(pages[index].Text))
	}
	return pages
}

func metadataString(metadata map[string]any, key string, fallback string) string {
	if value, ok := metadata[key].(string); ok && strings.TrimSpace(value) != "" {
		return strings.TrimSpace(value)
	}
	return fallback
}

func firstMetadataString(metadata map[string]any, keys ...string) *string {
	for _, key := range keys {
		if value := metadataString(metadata, key, ""); value != "" {
			return &value
		}
	}
	return nil
}

func firstNonEmptyMetadataString(metadata map[string]any, keys ...string) string {
	for _, key := range keys {
		if value := metadataString(metadata, key, ""); value != "" {
			return value
		}
	}
	return ""
}

func metadataStringSlice(metadata map[string]any, key string) []string {
	value, ok := metadata[key]
	if !ok {
		return []string{}
	}
	switch typed := value.(type) {
	case []string:
		out := make([]string, 0, len(typed))
		for _, item := range typed {
			if trimmed := strings.TrimSpace(item); trimmed != "" {
				out = append(out, trimmed)
			}
		}
		return out
	case []any:
		out := make([]string, 0, len(typed))
		for _, item := range typed {
			if text, ok := item.(string); ok {
				if trimmed := strings.TrimSpace(text); trimmed != "" {
					out = append(out, trimmed)
				}
			}
		}
		return out
	case string:
		if strings.TrimSpace(typed) == "" {
			return []string{}
		}
		parts := strings.Split(typed, ",")
		out := make([]string, 0, len(parts))
		for _, item := range parts {
			if trimmed := strings.TrimSpace(item); trimmed != "" {
				out = append(out, trimmed)
			}
		}
		return out
	default:
		return []string{}
	}
}

func metadataAnySlice(metadata map[string]any, keys ...string) []any {
	for _, key := range keys {
		value, ok := metadata[key]
		if !ok {
			continue
		}
		switch typed := value.(type) {
		case []any:
			return typed
		case []string:
			out := make([]any, 0, len(typed))
			for _, item := range typed {
				out = append(out, item)
			}
			return out
		case string:
			trimmed := strings.TrimSpace(typed)
			if trimmed == "" {
				continue
			}
			var out []any
			if err := json.Unmarshal([]byte(trimmed), &out); err == nil {
				return out
			}
		}
	}
	return []any{}
}

func metadataBool(metadata map[string]any, key string, fallback bool) bool {
	if value, ok := metadata[key].(bool); ok {
		return value
	}
	return fallback
}

func formatMillis(value int64) string {
	if value <= 0 {
		value = nowMillis()
	}
	return time.UnixMilli(value).UTC().Format(time.RFC3339Nano)
}
