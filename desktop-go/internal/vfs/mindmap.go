package vfs

import (
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"strings"
)

type MindMapViewType string

type VfsMindMap struct {
	ID          string         `json:"id"`
	ResourceID  string         `json:"resourceId"`
	Title       string         `json:"title"`
	Description *string        `json:"description,omitempty"`
	IsFavorite  bool           `json:"isFavorite"`
	DefaultView string         `json:"defaultView"`
	Theme       *string        `json:"theme,omitempty"`
	Settings    map[string]any `json:"settings,omitempty"`
	CreatedAt   string         `json:"createdAt"`
	UpdatedAt   string         `json:"updatedAt"`
	DeletedAt   *string        `json:"deletedAt,omitempty"`
}

type VfsMindMapVersion struct {
	VersionID  string  `json:"versionId"`
	MindMapID  string  `json:"mindmapId"`
	ResourceID string  `json:"resourceId"`
	Title      string  `json:"title"`
	Label      *string `json:"label,omitempty"`
	Source     *string `json:"source,omitempty"`
	CreatedAt  string  `json:"createdAt"`
}

type CreateMindMapInput struct {
	Title       string  `json:"title"`
	Description *string `json:"description,omitempty"`
	Content     string  `json:"content,omitempty"`
	DefaultView *string `json:"defaultView,omitempty"`
	Theme       *string `json:"theme,omitempty"`
	FolderID    *string `json:"folderId,omitempty"`
}

type UpdateMindMapInput struct {
	Title             *string        `json:"title,omitempty"`
	Description       *string        `json:"description,omitempty"`
	Content           *string        `json:"content,omitempty"`
	DefaultView       *string        `json:"defaultView,omitempty"`
	Theme             *string        `json:"theme,omitempty"`
	Settings          map[string]any `json:"settings,omitempty"`
	ExpectedUpdatedAt *string        `json:"expectedUpdatedAt,omitempty"`
}

func (s *Service) CreateMindMap(params CreateMindMapInput) (VfsMindMap, error) {
	title := strings.TrimSpace(params.Title)
	if title == "" {
		return VfsMindMap{}, errors.New("mindmap title is required")
	}
	content := strings.TrimSpace(params.Content)
	if content == "" {
		content = defaultMindMapContent(title)
	}
	if err := validateMindMapJSON(content); err != nil {
		return VfsMindMap{}, err
	}

	now := nowMillis()
	mindmapID := "mm_" + randomToken(12)
	metadata := mindMapMetadata(mindmapID, "", title, params.Description, viewOrDefault(params.DefaultView), params.Theme, nil, params.FolderID, false, now, now)

	created, err := s.createMindMapResource(mindmapID, content, metadata)
	if err != nil {
		return VfsMindMap{}, err
	}

	s.mu.RLock()
	defer s.mu.RUnlock()
	resource, ok := s.findResourceByAnyIDLocked(created.ResourceID)
	if !ok {
		return VfsMindMap{}, fmt.Errorf("created mindmap resource not found: %s", created.ResourceID)
	}
	return resourceToMindMap(resource), nil
}

func (s *Service) GetMindMap(mindmapID string) (*VfsMindMap, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	resource, ok := s.findMindMapResourceLocked(mindmapID)
	if !ok || resourceIsDeleted(resource) {
		return nil, nil
	}
	mindmap := resourceToMindMap(resource)
	return &mindmap, nil
}

func (s *Service) GetMindMapContent(mindmapID string) (*string, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	resource, ok := s.findMindMapResourceLocked(mindmapID)
	if !ok || resourceIsDeleted(resource) {
		return nil, nil
	}
	hydrated, err := s.hydrateResourceData(resource)
	if err != nil {
		return nil, err
	}
	return hydrated.Data, nil
}

func (s *Service) UpdateMindMap(mindmapID string, params UpdateMindMapInput) (VfsMindMap, error) {
	mindmapID = strings.TrimSpace(mindmapID)
	if !strings.HasPrefix(mindmapID, "mm_") {
		return VfsMindMap{}, fmt.Errorf("invalid mindmap ID format: %s", mindmapID)
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	index, ok := s.findMindMapResourceIndexLocked(mindmapID)
	if !ok || resourceIsDeleted(s.state.Resources[index]) {
		return VfsMindMap{}, fmt.Errorf("mindmap not found: %s", mindmapID)
	}
	current := s.state.Resources[index]
	if params.ExpectedUpdatedAt != nil && strings.TrimSpace(*params.ExpectedUpdatedAt) != "" {
		actual := metadataString(current.Metadata, "updatedAt", formatMillis(current.UpdatedAt))
		if strings.TrimSpace(*params.ExpectedUpdatedAt) != actual {
			return VfsMindMap{}, fmt.Errorf("mindmap update conflict: expected updatedAt %s, actual %s", strings.TrimSpace(*params.ExpectedUpdatedAt), actual)
		}
	}

	now := nowMillis()
	nextMetadata := normalizeMetadata(current.Metadata)
	if nextMetadata == nil {
		nextMetadata = map[string]any{}
	}
	if params.Title != nil {
		title := strings.TrimSpace(*params.Title)
		if title == "" {
			return VfsMindMap{}, errors.New("mindmap title cannot be empty")
		}
		nextMetadata["title"] = title
		nextMetadata["name"] = title
	}
	if params.Description != nil {
		nextMetadata["description"] = strings.TrimSpace(*params.Description)
	}
	if params.DefaultView != nil {
		nextMetadata["defaultView"] = viewOrDefault(params.DefaultView)
	}
	if params.Theme != nil {
		nextMetadata["theme"] = strings.TrimSpace(*params.Theme)
	}
	if params.Settings != nil {
		nextMetadata["settings"] = normalizeMetadata(params.Settings)
	}
	nextMetadata["updatedAt"] = formatMillis(now)

	if params.Content != nil {
		content := strings.TrimSpace(*params.Content)
		if content == "" {
			content = defaultMindMapContent(metadataString(nextMetadata, "title", "Mind Map"))
		}
		if err := validateMindMapJSON(content); err != nil {
			return VfsMindMap{}, err
		}
		previousContent := ""
		if hydrated, err := s.hydrateResourceData(current); err == nil && hydrated.Data != nil {
			previousContent = *hydrated.Data
		}
		if previousContent != "" && previousContent != content {
			if err := s.appendMindMapVersionLocked(current, previousContent, "manual", now); err != nil {
				return VfsMindMap{}, err
			}
		}
		hash := computeHash(content)
		relativePath, encoding, err := s.writeResourceData("mindmap", hash, content, nextMetadata)
		if err != nil {
			return VfsMindMap{}, err
		}
		externalHash := hash
		current.Hash = hash
		current.ExternalHash = &externalHash
		current.ExternalPath = &relativePath
		current.ContentEncoding = encoding
	}

	current.Metadata = nextMetadata
	current.UpdatedAt = now
	s.state.Resources[index] = current
	if err := s.flushLocked(); err != nil {
		return VfsMindMap{}, err
	}
	return resourceToMindMap(current), nil
}

func (s *Service) DeleteMindMap(mindmapID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	index, ok := s.findMindMapResourceIndexLocked(mindmapID)
	if !ok || resourceIsDeleted(s.state.Resources[index]) {
		return nil
	}
	now := nowMillis()
	metadata := normalizeMetadata(s.state.Resources[index].Metadata)
	if metadata == nil {
		metadata = map[string]any{}
	}
	metadata["status"] = "deleted"
	metadata["deletedAt"] = formatMillis(now)
	metadata["updatedAt"] = formatMillis(now)
	s.state.Resources[index].Metadata = metadata
	s.state.Resources[index].UpdatedAt = now
	return s.flushLocked()
}

func (s *Service) ListMindMaps() ([]VfsMindMap, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	items := make([]VfsMindMap, 0)
	for _, resource := range s.state.Resources {
		if resource.Type != "mindmap" || !metadataBool(resource.Metadata, "mindmapRecord", false) || resourceIsDeleted(resource) {
			continue
		}
		items = append(items, resourceToMindMap(resource))
	}
	sort.SliceStable(items, func(left int, right int) bool {
		return items[left].UpdatedAt > items[right].UpdatedAt
	})
	return items, nil
}

func (s *Service) SetMindMapFavorite(mindmapID string, isFavorite bool) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	index, ok := s.findMindMapResourceIndexLocked(mindmapID)
	if !ok || resourceIsDeleted(s.state.Resources[index]) {
		return fmt.Errorf("mindmap not found: %s", mindmapID)
	}
	now := nowMillis()
	metadata := normalizeMetadata(s.state.Resources[index].Metadata)
	if metadata == nil {
		metadata = map[string]any{}
	}
	metadata["isFavorite"] = isFavorite
	metadata["updatedAt"] = formatMillis(now)
	s.state.Resources[index].Metadata = metadata
	s.state.Resources[index].UpdatedAt = now
	return s.flushLocked()
}

func (s *Service) GetMindMapVersions(mindmapID string) ([]VfsMindMapVersion, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	versions := make([]VfsMindMapVersion, 0)
	for _, resource := range s.state.Resources {
		if resource.Type != "mindmap" || !metadataBool(resource.Metadata, "mindmapVersion", false) || resourceIsDeleted(resource) {
			continue
		}
		if metadataString(resource.Metadata, "mindmapId", "") != strings.TrimSpace(mindmapID) {
			continue
		}
		versions = append(versions, resourceToMindMapVersion(resource))
	}
	sort.SliceStable(versions, func(left int, right int) bool {
		return versions[left].CreatedAt > versions[right].CreatedAt
	})
	return versions, nil
}

func (s *Service) GetMindMapVersion(versionID string) (*VfsMindMapVersion, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	resource, ok := s.findMindMapVersionResourceLocked(versionID)
	if !ok || resourceIsDeleted(resource) {
		return nil, nil
	}
	version := resourceToMindMapVersion(resource)
	return &version, nil
}

func (s *Service) GetMindMapVersionContent(versionID string) (*string, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	resource, ok := s.findMindMapVersionResourceLocked(versionID)
	if !ok || resourceIsDeleted(resource) {
		return nil, nil
	}
	hydrated, err := s.hydrateResourceData(resource)
	if err != nil {
		return nil, err
	}
	return hydrated.Data, nil
}

func (s *Service) createMindMapResource(sourceID string, content string, metadata map[string]any) (CreateResourceResult, error) {
	return s.CreateOrUpdateSource(CreateResourceInput{
		Type:     "mindmap",
		Data:     content,
		SourceID: &sourceID,
		Metadata: metadata,
	})
}

func (s *Service) appendMindMapVersionLocked(mindmapResource Resource, content string, source string, now int64) error {
	versionID := "mv_" + randomToken(12)
	title := metadataString(mindmapResource.Metadata, "title", resourceName(mindmapResource))
	mindmapID := resourceSourceID(mindmapResource)
	resourceID := mindmapResource.ID
	label := metadataString(mindmapResource.Metadata, "updatedAt", formatMillis(mindmapResource.UpdatedAt))
	metadata := map[string]any{
		"sourceType":       "mindmap",
		"resourceType":     "mindmap",
		"mindmapVersion":   true,
		"versionId":        versionID,
		"mindmapId":        mindmapID,
		"parentResourceId": resourceID,
		"resourceId":       resourceID,
		"title":            title,
		"name":             title,
		"label":            label,
		"source":           source,
		"createdAt":        formatMillis(now),
		"updatedAt":        formatMillis(now),
	}
	hash := computeHash(content)
	relativePath, encoding, err := s.writeResourceData("mindmap", hash, content, metadata)
	if err != nil {
		return err
	}
	externalHash := hash
	sourceID := versionID
	versionResource := Resource{
		ID:              "res_" + randomToken(12),
		Hash:            hash,
		Type:            "mindmap",
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
	s.state.Resources = append(s.state.Resources, versionResource)
	return nil
}

func (s *Service) findMindMapResourceLocked(id string) (Resource, bool) {
	resource, ok := s.findResourceByAnyIDLocked(id)
	if !ok || resource.Type != "mindmap" || !metadataBool(resource.Metadata, "mindmapRecord", false) {
		return Resource{}, false
	}
	return resource, true
}

func (s *Service) findMindMapResourceIndexLocked(id string) (int, bool) {
	index, ok := s.findResourceIndexByAnyIDLocked(id)
	if !ok {
		return -1, false
	}
	resource := s.state.Resources[index]
	if resource.Type != "mindmap" || !metadataBool(resource.Metadata, "mindmapRecord", false) {
		return -1, false
	}
	return index, true
}

func (s *Service) findMindMapVersionResourceLocked(id string) (Resource, bool) {
	resource, ok := s.findResourceByAnyIDLocked(id)
	if !ok || resource.Type != "mindmap" || !metadataBool(resource.Metadata, "mindmapVersion", false) {
		return Resource{}, false
	}
	return resource, true
}

func mindMapMetadata(mindmapID string, resourceID string, title string, description *string, defaultView string, theme *string, settings map[string]any, folderID *string, favorite bool, createdAt int64, updatedAt int64) map[string]any {
	metadata := map[string]any{
		"sourceType":     "mindmap",
		"resourceType":   "mindmap",
		"mindmapRecord":  true,
		"mindmapId":      mindmapID,
		"title":          title,
		"name":           title,
		"defaultView":    defaultView,
		"isFavorite":     favorite,
		"createdAt":      formatMillis(createdAt),
		"updatedAt":      formatMillis(updatedAt),
		"contentType":    "application/json",
		"mimeType":       "application/json",
		"extension":      "json",
		"schema":         "MindMapDocument",
		"storageSurface": "hybrid-vfs",
	}
	if resourceID != "" {
		metadata["resourceId"] = resourceID
	}
	if description != nil {
		metadata["description"] = strings.TrimSpace(*description)
	}
	if theme != nil {
		metadata["theme"] = strings.TrimSpace(*theme)
	}
	if settings != nil {
		metadata["settings"] = normalizeMetadata(settings)
	}
	if value := normalizeOptionalString(folderID); value != nil {
		metadata["folderId"] = *value
	}
	return metadata
}

func resourceToMindMap(resource Resource) VfsMindMap {
	resourceID := metadataString(resource.Metadata, "resourceId", resource.ID)
	description := firstMetadataString(resource.Metadata, "description")
	theme := firstMetadataString(resource.Metadata, "theme")
	settings := metadataMap(resource.Metadata, "settings")
	return VfsMindMap{
		ID:          resourceSourceID(resource),
		ResourceID:  resourceID,
		Title:       metadataString(resource.Metadata, "title", resourceName(resource)),
		Description: description,
		IsFavorite:  metadataBool(resource.Metadata, "isFavorite", false),
		DefaultView: metadataString(resource.Metadata, "defaultView", "mindmap"),
		Theme:       theme,
		Settings:    settings,
		CreatedAt:   metadataString(resource.Metadata, "createdAt", formatMillis(resource.CreatedAt)),
		UpdatedAt:   metadataString(resource.Metadata, "updatedAt", formatMillis(resource.UpdatedAt)),
		DeletedAt:   firstMetadataString(resource.Metadata, "deletedAt"),
	}
}

func resourceToMindMapVersion(resource Resource) VfsMindMapVersion {
	return VfsMindMapVersion{
		VersionID:  metadataString(resource.Metadata, "versionId", resourceSourceID(resource)),
		MindMapID:  metadataString(resource.Metadata, "mindmapId", ""),
		ResourceID: metadataString(resource.Metadata, "resourceId", metadataString(resource.Metadata, "parentResourceId", resource.ID)),
		Title:      metadataString(resource.Metadata, "title", resourceName(resource)),
		Label:      firstMetadataString(resource.Metadata, "label"),
		Source:     firstMetadataString(resource.Metadata, "source"),
		CreatedAt:  metadataString(resource.Metadata, "createdAt", formatMillis(resource.CreatedAt)),
	}
}

func defaultMindMapContent(title string) string {
	escaped, _ := json.Marshal(strings.TrimSpace(title))
	if string(escaped) == `""` {
		escaped = []byte(`"Mind Map"`)
	}
	now := formatMillis(nowMillis())
	return fmt.Sprintf(`{"version":"1.0","root":{"id":"root","text":%s,"children":[]},"meta":{"createdAt":%q,"updatedAt":%q}}`, string(escaped), now, now)
}

func validateMindMapJSON(content string) error {
	var decoded map[string]any
	if err := json.Unmarshal([]byte(content), &decoded); err != nil {
		return fmt.Errorf("invalid mindmap content JSON: %w", err)
	}
	root, ok := decoded["root"].(map[string]any)
	if !ok {
		return errors.New("invalid mindmap content: root is required")
	}
	if _, ok := root["children"]; !ok {
		root["children"] = []any{}
	}
	return nil
}

func viewOrDefault(value *string) string {
	if value == nil {
		return "mindmap"
	}
	switch strings.ToLower(strings.TrimSpace(*value)) {
	case "outline":
		return "outline"
	default:
		return "mindmap"
	}
}

func metadataMap(metadata map[string]any, key string) map[string]any {
	value, ok := metadata[key]
	if !ok || value == nil {
		return nil
	}
	if typed, ok := value.(map[string]any); ok {
		return normalizeMetadata(typed)
	}
	return nil
}
