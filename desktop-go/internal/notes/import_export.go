package notes

import (
	"archive/zip"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"
)

const notesExportSchemaVersion = 2

var exportAssetReferencePattern = regexp.MustCompile(`notes_assets/[^\s\])"'<>]+`)

type ExportRequest struct {
	OutputPath      *string `json:"output_path,omitempty"`
	IncludeVersions *bool   `json:"include_versions,omitempty"`
}

type ExportSingleRequest struct {
	NoteID          string  `json:"note_id"`
	OutputPath      *string `json:"output_path,omitempty"`
	IncludeVersions *bool   `json:"include_versions,omitempty"`
}

type ExportResult struct {
	OutputPath      string `json:"output_path"`
	NoteCount       int    `json:"note_count"`
	AttachmentCount int    `json:"attachment_count"`
}

type ImportRequest struct {
	FilePath         string  `json:"file_path"`
	ConflictStrategy *string `json:"conflict_strategy,omitempty"`
}

type ImportResult struct {
	SubjectCount     int `json:"subject_count"`
	NoteCount        int `json:"note_count"`
	AttachmentCount  int `json:"attachment_count"`
	SkippedCount     int `json:"skipped_count"`
	OverwrittenCount int `json:"overwritten_count"`
}

type ImportArchive struct {
	Notes           []ExportNoteRecord
	AttachmentCount int
}

type ExportNoteRecord struct {
	ID         string         `json:"id"`
	Title      string         `json:"title"`
	ContentMD  string         `json:"content_md"`
	Tags       []string       `json:"tags"`
	CreatedAt  string         `json:"created_at"`
	UpdatedAt  string         `json:"updated_at"`
	IsFavorite bool           `json:"is_favorite"`
	Metadata   map[string]any `json:"metadata,omitempty"`
}

type exportManifest struct {
	SchemaVersion   int                     `json:"schema_version"`
	ExportedAt      string                  `json:"exported_at"`
	AppVersion      string                  `json:"app_version"`
	NoteCount       int                     `json:"note_count"`
	AttachmentCount int                     `json:"attachment_count"`
	VersionCount    int                     `json:"version_count"`
	Preferences     []exportManifestPref    `json:"preferences"`
	Subjects        []exportManifestSubject `json:"subjects"`
}

type exportManifestPref struct {
	Key   string `json:"key"`
	File  string `json:"file"`
	Bytes int    `json:"bytes"`
}

type exportManifestSubject struct {
	Subject         string               `json:"subject"`
	Slug            string               `json:"slug"`
	NoteCount       int                  `json:"note_count"`
	Preferences     []exportManifestPref `json:"preferences"`
	NotesFile       *string              `json:"notes_file,omitempty"`
	AttachmentsRoot *string              `json:"attachments_root,omitempty"`
}

func (s *Service) Export(records []ExportNoteRecord, request ExportRequest) (ExportResult, error) {
	return s.exportRecords(records, request.OutputPath, "notes_export")
}

func (s *Service) ExportSingle(record ExportNoteRecord, request ExportSingleRequest) (ExportResult, error) {
	if strings.TrimSpace(record.ID) == "" {
		return ExportResult{}, errors.New("note_id cannot be empty")
	}
	return s.exportRecords([]ExportNoteRecord{record}, request.OutputPath, "note_export")
}

func (s *Service) ReadImportArchive(request ImportRequest) (ImportArchive, error) {
	filePath := strings.TrimSpace(request.FilePath)
	if filePath == "" {
		return ImportArchive{}, errors.New("file_path cannot be empty")
	}
	reader, err := zip.OpenReader(filePath)
	if err != nil {
		return ImportArchive{}, err
	}
	defer reader.Close()

	notes := []ExportNoteRecord{}
	attachmentCount := 0
	hasNotesJSON := false
	for _, file := range reader.File {
		name, ok := safeArchivePath(file.Name)
		if !ok || strings.HasSuffix(name, "/") {
			continue
		}
		switch {
		case name == "_notes.json":
			parsed, err := readNotesJSON(file)
			if err != nil {
				return ImportArchive{}, err
			}
			notes = parsed
			hasNotesJSON = true
		case strings.HasPrefix(name, "notes/") && strings.HasSuffix(strings.ToLower(name), ".md"):
			if !hasNotesJSON {
				note, err := readMarkdownNote(file)
				if err != nil {
					return ImportArchive{}, err
				}
				notes = append(notes, note)
			}
		case strings.HasPrefix(name, "notes_assets/"):
			if copied, err := s.copyArchiveAsset(file, name); err != nil {
				return ImportArchive{}, err
			} else if copied {
				attachmentCount++
			}
		case strings.HasPrefix(name, "assets/notes_assets/"):
			relative := strings.TrimPrefix(name, "assets/")
			if copied, err := s.copyArchiveAsset(file, relative); err != nil {
				return ImportArchive{}, err
			} else if copied {
				attachmentCount++
			}
		}
	}
	return ImportArchive{
		Notes:           normalizeExportRecords(notes),
		AttachmentCount: attachmentCount,
	}, nil
}

func (s *Service) exportRecords(records []ExportNoteRecord, outputPath *string, prefix string) (ExportResult, error) {
	records = normalizeExportRecords(records)
	outPath := strings.TrimSpace("")
	if outputPath != nil {
		outPath = strings.TrimSpace(*outputPath)
	}
	if outPath == "" {
		outPath = filepath.Join(s.dataDir, "exports", fmt.Sprintf("%s_%s.zip", prefix, time.Now().UTC().Format("20060102_150405")))
	}
	if parent := filepath.Dir(outPath); parent != "" && parent != "." {
		if err := os.MkdirAll(parent, 0o700); err != nil {
			return ExportResult{}, err
		}
	}

	assets, err := s.assetsForRecords(records)
	if err != nil {
		return ExportResult{}, err
	}
	file, err := os.Create(outPath)
	if err != nil {
		return ExportResult{}, err
	}
	defer file.Close()

	writer := zip.NewWriter(file)
	if err := writeJSONEntry(writer, "manifest.json", exportManifest{
		SchemaVersion:   notesExportSchemaVersion,
		ExportedAt:      time.Now().UTC().Format(time.RFC3339Nano),
		AppVersion:      "deep-student-go",
		NoteCount:       len(records),
		AttachmentCount: len(assets),
		VersionCount:    0,
		Preferences:     []exportManifestPref{},
		Subjects:        []exportManifestSubject{},
	}); err != nil {
		_ = writer.Close()
		return ExportResult{}, err
	}
	if err := writeJSONEntry(writer, "_notes.json", records); err != nil {
		_ = writer.Close()
		return ExportResult{}, err
	}
	if err := writeREADME(writer); err != nil {
		_ = writer.Close()
		return ExportResult{}, err
	}
	usedNames := map[string]int{}
	for _, record := range records {
		name := noteMarkdownEntryName(record, usedNames)
		if err := writeTextEntry(writer, name, renderMarkdownExport(record)); err != nil {
			_ = writer.Close()
			return ExportResult{}, err
		}
	}
	for _, asset := range assets {
		if err := writeFileEntry(writer, asset.RelativePath, asset.AbsolutePath); err != nil {
			_ = writer.Close()
			return ExportResult{}, err
		}
	}
	if err := writer.Close(); err != nil {
		return ExportResult{}, err
	}
	return ExportResult{
		OutputPath:      outPath,
		NoteCount:       len(records),
		AttachmentCount: len(assets),
	}, nil
}

func (s *Service) assetsForRecords(records []ExportNoteRecord) ([]AssetRef, error) {
	seen := map[string]AssetRef{}
	for _, record := range records {
		for _, ref := range exportAssetReferencePattern.FindAllString(record.ContentMD, -1) {
			relative := normalizeAssetReference(ref)
			if relative == "" {
				continue
			}
			absolute, err := s.ResolveAssetPath(relative)
			if err != nil {
				continue
			}
			info, err := os.Stat(absolute)
			if err != nil || info.IsDir() {
				continue
			}
			seen[relative] = AssetRef{AbsolutePath: absolute, RelativePath: relative}
		}
		if strings.TrimSpace(record.ID) == "" {
			continue
		}
		assets, err := s.ListAssets("_global", record.ID)
		if err != nil {
			return nil, err
		}
		for _, asset := range assets {
			seen[normalizeAssetReference(asset.RelativePath)] = asset
		}
	}
	keys := make([]string, 0, len(seen))
	for key := range seen {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	out := make([]AssetRef, 0, len(keys))
	for _, key := range keys {
		out = append(out, seen[key])
	}
	return out, nil
}

func (s *Service) copyArchiveAsset(file *zip.File, relativePath string) (bool, error) {
	relativePath = normalizeAssetReference(relativePath)
	if relativePath == "" {
		return false, nil
	}
	if _, ok := safeArchivePath(relativePath); !ok {
		return false, nil
	}
	absolutePath, err := s.resolveRelative(relativePath)
	if err != nil {
		return false, err
	}
	if err := os.MkdirAll(filepath.Dir(absolutePath), 0o700); err != nil {
		return false, err
	}
	source, err := file.Open()
	if err != nil {
		return false, err
	}
	defer source.Close()
	destination, err := os.Create(absolutePath)
	if err != nil {
		return false, err
	}
	defer destination.Close()
	if _, err := io.Copy(destination, source); err != nil {
		return false, err
	}
	return true, nil
}

func writeJSONEntry(writer *zip.Writer, name string, value any) error {
	bytes, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		return err
	}
	return writeBytesEntry(writer, name, append(bytes, '\n'))
}

func writeREADME(writer *zip.Writer) error {
	return writeTextEntry(writer, "README.md", "# Deep Student Notes Export\n\nThis archive contains Markdown notes, a compact _notes.json metadata file, and visible notes_assets files.\n")
}

func writeTextEntry(writer *zip.Writer, name string, text string) error {
	return writeBytesEntry(writer, name, []byte(text))
}

func writeFileEntry(writer *zip.Writer, name string, path string) error {
	bytes, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	return writeBytesEntry(writer, name, bytes)
}

func writeBytesEntry(writer *zip.Writer, name string, bytes []byte) error {
	name, ok := safeArchivePath(name)
	if !ok {
		return fmt.Errorf("unsafe archive path: %s", name)
	}
	entry, err := writer.Create(name)
	if err != nil {
		return err
	}
	_, err = entry.Write(bytes)
	return err
}

func readNotesJSON(file *zip.File) ([]ExportNoteRecord, error) {
	reader, err := file.Open()
	if err != nil {
		return nil, err
	}
	defer reader.Close()
	bytes, err := io.ReadAll(reader)
	if err != nil {
		return nil, err
	}
	var records []ExportNoteRecord
	if err := json.Unmarshal(bytes, &records); err != nil {
		return nil, err
	}
	return records, nil
}

func readMarkdownNote(file *zip.File) (ExportNoteRecord, error) {
	reader, err := file.Open()
	if err != nil {
		return ExportNoteRecord{}, err
	}
	defer reader.Close()
	bytes, err := io.ReadAll(reader)
	if err != nil {
		return ExportNoteRecord{}, err
	}
	record := parseMarkdownExport(string(bytes))
	if record.Title == "" {
		base := strings.TrimSuffix(filepath.Base(file.Name), filepath.Ext(file.Name))
		record.Title = strings.TrimSpace(base)
	}
	return record, nil
}

func renderMarkdownExport(record ExportNoteRecord) string {
	tags, _ := json.Marshal(record.Tags)
	title, _ := json.Marshal(record.Title)
	created, _ := json.Marshal(record.CreatedAt)
	updated, _ := json.Marshal(record.UpdatedAt)
	return fmt.Sprintf("---\nid: %s\ntitle: %s\ntags: %s\ncreated_at: %s\nupdated_at: %s\nis_favorite: %t\n---\n\n%s",
		record.ID,
		string(title),
		string(tags),
		string(created),
		string(updated),
		record.IsFavorite,
		record.ContentMD,
	)
}

func parseMarkdownExport(markdown string) ExportNoteRecord {
	record := ExportNoteRecord{}
	body := markdown
	if strings.HasPrefix(markdown, "---") {
		rest := strings.TrimPrefix(markdown, "---")
		rest = strings.TrimPrefix(rest, "\r\n")
		rest = strings.TrimPrefix(rest, "\n")
		if index := strings.Index(rest, "\n---"); index >= 0 {
			frontmatter := rest[:index]
			body = strings.TrimPrefix(rest[index+len("\n---"):], "\r\n")
			body = strings.TrimPrefix(body, "\n")
			for _, line := range strings.Split(frontmatter, "\n") {
				key, value, ok := strings.Cut(line, ":")
				if !ok {
					continue
				}
				key = strings.TrimSpace(key)
				value = strings.TrimSpace(value)
				switch key {
				case "id":
					record.ID = strings.Trim(value, `"'`)
				case "title":
					record.Title = decodeJSONOrTrim(value)
				case "created_at":
					record.CreatedAt = decodeJSONOrTrim(value)
				case "updated_at":
					record.UpdatedAt = decodeJSONOrTrim(value)
				case "is_favorite":
					record.IsFavorite = strings.EqualFold(value, "true")
				case "tags":
					record.Tags = parseTagList(value)
				}
			}
		}
	}
	record.ContentMD = body
	return record
}

func decodeJSONOrTrim(value string) string {
	var decoded string
	if err := json.Unmarshal([]byte(value), &decoded); err == nil {
		return decoded
	}
	return strings.Trim(value, `"'`)
}

func parseTagList(value string) []string {
	var tags []string
	if err := json.Unmarshal([]byte(value), &tags); err == nil {
		return normalizeTags(tags)
	}
	parts := strings.Split(strings.Trim(value, "[]"), ",")
	return normalizeTags(parts)
}

func normalizeExportRecords(records []ExportNoteRecord) []ExportNoteRecord {
	out := make([]ExportNoteRecord, 0, len(records))
	for _, record := range records {
		record.ID = strings.TrimSpace(record.ID)
		record.Title = strings.TrimSpace(record.Title)
		if record.Title == "" {
			record.Title = "Untitled"
		}
		record.Tags = normalizeTags(record.Tags)
		if record.Metadata == nil {
			record.Metadata = map[string]any{}
		}
		out = append(out, record)
	}
	return out
}

func normalizeTags(tags []string) []string {
	out := make([]string, 0, len(tags))
	seen := map[string]bool{}
	for _, tag := range tags {
		trimmed := strings.Trim(strings.TrimSpace(tag), `"'`)
		if trimmed == "" {
			continue
		}
		key := strings.ToLower(trimmed)
		if seen[key] {
			continue
		}
		seen[key] = true
		out = append(out, trimmed)
	}
	return out
}

func noteMarkdownEntryName(record ExportNoteRecord, used map[string]int) string {
	base := safeSegment(record.Title, "note")
	if record.ID != "" {
		base += "_" + safeSegment(record.ID, "id")
	}
	count := used[base]
	used[base] = count + 1
	if count > 0 {
		base = fmt.Sprintf("%s_%d", base, count+1)
	}
	return filepath.ToSlash(filepath.Join("notes", base+".md"))
}

func safeArchivePath(name string) (string, bool) {
	normalized := filepath.ToSlash(strings.TrimSpace(name))
	normalized = strings.TrimPrefix(normalized, "/")
	if normalized == "" || normalized == "." || strings.Contains(normalized, "\x00") {
		return "", false
	}
	cleaned := filepath.ToSlash(filepath.Clean(normalized))
	if cleaned == "." || strings.HasPrefix(cleaned, "../") || strings.Contains(cleaned, "/../") || filepath.IsAbs(cleaned) {
		return "", false
	}
	return cleaned, true
}
