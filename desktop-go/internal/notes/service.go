package notes

import (
	"crypto/rand"
	"deep-student-go/internal/storage"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"math/big"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
	"time"
)

var safeSegmentPattern = regexp.MustCompile(`[^A-Za-z0-9._-]+`)

type Service struct {
	mu      sync.RWMutex
	dataDir string
	path    string
	state   store
}

type store struct {
	Prefs map[string]string `json:"prefs"`
}

type AssetRef struct {
	AbsolutePath string `json:"absolute_path"`
	RelativePath string `json:"relative_path"`
}

type DBStats struct {
	DBPath        string `json:"db_path"`
	FileSizeBytes int64  `json:"file_size_bytes"`
	TotalNotes    int    `json:"total_notes"`
	TotalVersions int    `json:"total_versions"`
	TotalAssets   int    `json:"total_assets"`
}

func NewService(dataDir string) (*Service, error) {
	service := &Service{
		dataDir: filepath.Clean(dataDir),
		path:    filepath.Join(dataDir, "notes-go.json"),
		state: store{
			Prefs: map[string]string{},
		},
	}
	if err := service.load(); err != nil {
		return nil, err
	}
	return service, nil
}

func (s *Service) SetPref(key string, value string) (bool, error) {
	if key == "" {
		return false, errors.New("notes preference key cannot be empty")
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	s.state.Prefs[key] = value
	return true, s.flushLocked()
}

func (s *Service) GetPref(key string) (string, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	value, ok := s.state.Prefs[key]
	return value, ok
}

func (s *Service) SaveAsset(subject string, noteID string, base64Data string, defaultExt string) (AssetRef, error) {
	if strings.TrimSpace(noteID) == "" {
		return AssetRef{}, errors.New("noteId cannot be empty")
	}
	cleanNoteID := safeSegment(noteID, "note")
	cleanSubject := safeSegment(subject, "_global")
	cleanExt := safeExtension(defaultExt)
	bytes, err := decodeBase64Payload(base64Data)
	if err != nil {
		return AssetRef{}, err
	}

	relativeDir := filepath.Join("notes_assets", cleanSubject, cleanNoteID)
	absoluteDir, err := s.resolveRelative(relativeDir)
	if err != nil {
		return AssetRef{}, err
	}
	if err := os.MkdirAll(absoluteDir, 0o700); err != nil {
		return AssetRef{}, err
	}

	fileName := fmt.Sprintf("%s.%s", newToken(12), cleanExt)
	relativePath := filepath.ToSlash(filepath.Join(relativeDir, fileName))
	absolutePath, err := s.resolveRelative(relativePath)
	if err != nil {
		return AssetRef{}, err
	}
	if err := os.WriteFile(absolutePath, bytes, 0o600); err != nil {
		return AssetRef{}, err
	}

	return AssetRef{AbsolutePath: absolutePath, RelativePath: relativePath}, nil
}

func (s *Service) ListAssets(subject string, noteID string) ([]AssetRef, error) {
	if strings.TrimSpace(noteID) == "" {
		return nil, errors.New("noteId cannot be empty")
	}
	cleanNoteID := safeSegment(noteID, "note")
	cleanSubject := safeSegment(subject, "_global")
	relativeDir := filepath.Join("notes_assets", cleanSubject, cleanNoteID)
	absoluteDir, err := s.resolveRelative(relativeDir)
	if err != nil {
		return nil, err
	}

	entries, err := os.ReadDir(absoluteDir)
	if errors.Is(err, os.ErrNotExist) {
		return []AssetRef{}, nil
	}
	if err != nil {
		return nil, err
	}

	out := make([]AssetRef, 0, len(entries))
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		relativePath := filepath.ToSlash(filepath.Join(relativeDir, entry.Name()))
		absolutePath, err := s.resolveRelative(relativePath)
		if err != nil {
			continue
		}
		out = append(out, AssetRef{AbsolutePath: absolutePath, RelativePath: relativePath})
	}
	return out, nil
}

func (s *Service) AssetsIndexScan(noteID string) (int, error) {
	assets, err := s.ListAssets("_global", noteID)
	if err != nil {
		return 0, err
	}
	return len(assets), nil
}

func (s *Service) ScanOrphanAssets(referencedPaths []string) ([]string, error) {
	referenced := map[string]bool{}
	for _, path := range referencedPaths {
		if normalized := normalizeAssetReference(path); normalized != "" {
			referenced[normalized] = true
		}
	}
	assets, err := s.listAllAssets()
	if err != nil {
		return nil, err
	}
	orphans := []string{}
	for _, asset := range assets {
		if !referenced[normalizeAssetReference(asset.RelativePath)] {
			orphans = append(orphans, asset.RelativePath)
		}
	}
	return orphans, nil
}

func (s *Service) BulkDeleteAssets(paths []string) (int, error) {
	count := 0
	for _, path := range paths {
		deleted, err := s.DeleteAsset(path)
		if err != nil {
			return count, err
		}
		if deleted {
			count++
		}
	}
	return count, nil
}

func (s *Service) DBStats(totalNotes int) (DBStats, error) {
	size := int64(0)
	if info, err := os.Stat(s.path); err == nil {
		size = info.Size()
	} else if err != nil && !errors.Is(err, os.ErrNotExist) {
		return DBStats{}, err
	}
	totalAssets, err := s.countAllAssets()
	if err != nil {
		return DBStats{}, err
	}
	return DBStats{
		DBPath:        s.path,
		FileSizeBytes: size,
		TotalNotes:    totalNotes,
		TotalVersions: 0,
		TotalAssets:   totalAssets,
	}, nil
}

func (s *Service) DBVacuum() (bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return true, s.flushLocked()
}

func (s *Service) DeleteAsset(relativePath string) (bool, error) {
	absolutePath, err := s.ResolveAssetPath(relativePath)
	if err != nil {
		return false, err
	}
	info, err := os.Stat(absolutePath)
	if errors.Is(err, os.ErrNotExist) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	if info.IsDir() {
		return false, errors.New("asset path is a directory")
	}
	if err := os.Remove(absolutePath); err != nil {
		return false, err
	}
	return true, nil
}

func (s *Service) ResolveAssetPath(relativePath string) (string, error) {
	trimmed := strings.TrimSpace(relativePath)
	if trimmed == "" {
		return "", errors.New("asset path cannot be empty")
	}
	if filepath.IsAbs(trimmed) {
		return s.ensureInsideDataDir(trimmed)
	}
	return s.resolveRelative(trimmed)
}

func (s *Service) GetImageAsBase64(relativePath string) (string, error) {
	absolutePath, err := s.ResolveAssetPath(stripAssetURL(relativePath))
	if err != nil {
		return "", err
	}
	bytes, err := os.ReadFile(absolutePath)
	if err != nil {
		return "", err
	}
	mime := mimeFromExtension(filepath.Ext(absolutePath))
	return fmt.Sprintf("data:%s;base64,%s", mime, base64.StdEncoding.EncodeToString(bytes)), nil
}

func (s *Service) listAllAssets() ([]AssetRef, error) {
	root, err := s.resolveRelative("notes_assets")
	if err != nil {
		return nil, err
	}
	refs := []AssetRef{}
	err = filepath.WalkDir(root, func(path string, entry os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if entry.IsDir() {
			return nil
		}
		relative, err := filepath.Rel(s.dataDir, path)
		if err != nil {
			return err
		}
		refs = append(refs, AssetRef{
			AbsolutePath: path,
			RelativePath: filepath.ToSlash(relative),
		})
		return nil
	})
	if errors.Is(err, os.ErrNotExist) {
		return []AssetRef{}, nil
	}
	return refs, err
}

func (s *Service) countAllAssets() (int, error) {
	assets, err := s.listAllAssets()
	if err != nil {
		return 0, err
	}
	return len(assets), nil
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
	if s.state.Prefs == nil {
		s.state.Prefs = map[string]string{}
	}
	return nil
}

func (s *Service) flushLocked() error {
	return storage.WriteJSONAtomic(s.path, s.state)
}

func (s *Service) resolveRelative(relativePath string) (string, error) {
	cleaned := filepath.Clean(relativePath)
	if filepath.IsAbs(cleaned) {
		return s.ensureInsideDataDir(cleaned)
	}
	return s.ensureInsideDataDir(filepath.Join(s.dataDir, cleaned))
}

func (s *Service) ensureInsideDataDir(target string) (string, error) {
	baseAbs, err := filepath.Abs(s.dataDir)
	if err != nil {
		return "", err
	}
	targetAbs, err := filepath.Abs(filepath.Clean(target))
	if err != nil {
		return "", err
	}
	rel, err := filepath.Rel(baseAbs, targetAbs)
	if err != nil {
		return "", err
	}
	if rel == "." || (!strings.HasPrefix(rel, "..") && !filepath.IsAbs(rel)) {
		return targetAbs, nil
	}
	return "", errors.New("asset path escapes app data directory")
}

func decodeBase64Payload(payload string) ([]byte, error) {
	trimmed := strings.TrimSpace(payload)
	if comma := strings.Index(trimmed, ","); strings.HasPrefix(trimmed, "data:") && comma >= 0 {
		trimmed = trimmed[comma+1:]
	}
	bytes, err := base64.StdEncoding.DecodeString(trimmed)
	if err != nil {
		return nil, fmt.Errorf("invalid base64 asset payload: %w", err)
	}
	return bytes, nil
}

func safeSegment(value string, fallback string) string {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		trimmed = fallback
	}
	trimmed = strings.ReplaceAll(trimmed, "\\", "_")
	trimmed = strings.ReplaceAll(trimmed, "/", "_")
	trimmed = safeSegmentPattern.ReplaceAllString(trimmed, "_")
	return strings.Trim(trimmed, "._-")
}

func safeExtension(value string) string {
	ext := strings.TrimSpace(strings.TrimPrefix(value, "."))
	ext = strings.ToLower(safeSegmentPattern.ReplaceAllString(ext, ""))
	if ext == "" {
		return "png"
	}
	if len(ext) > 12 {
		return ext[:12]
	}
	return ext
}

func newToken(length int) string {
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

func stripAssetURL(path string) string {
	trimmed := strings.TrimSpace(path)
	for _, prefix := range []string{"asset://localhost/", "asset://", "tauri://localhost/", "tauri://"} {
		if strings.HasPrefix(trimmed, prefix) {
			return strings.TrimPrefix(trimmed, prefix)
		}
	}
	return trimmed
}

func normalizeAssetReference(path string) string {
	trimmed := strings.TrimSpace(stripAssetURL(path))
	trimmed = strings.Trim(trimmed, `"'()[]<>`)
	trimmed = strings.ReplaceAll(trimmed, "\\", "/")
	if index := strings.Index(trimmed, "notes_assets/"); index >= 0 {
		trimmed = trimmed[index:]
	}
	return strings.TrimSpace(trimmed)
}

func mimeFromExtension(ext string) string {
	switch strings.ToLower(strings.TrimPrefix(ext, ".")) {
	case "jpg", "jpeg":
		return "image/jpeg"
	case "gif":
		return "image/gif"
	case "webp":
		return "image/webp"
	case "bmp":
		return "image/bmp"
	case "svg":
		return "image/svg+xml"
	case "heic":
		return "image/heic"
	case "heif":
		return "image/heif"
	case "png":
		return "image/png"
	default:
		return "application/octet-stream"
	}
}
