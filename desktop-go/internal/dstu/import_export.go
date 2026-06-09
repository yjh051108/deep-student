package dstu

import (
	"deep-student-go/internal/notes"
	"strings"
	"time"
)

func (s *Service) ExportNoteRecords(includeDeleted bool) []notes.ExportNoteRecord {
	s.mu.RLock()
	noteRecords := append([]NoteRecord(nil), s.state.Notes...)
	folders := append([]VfsFolder(nil), s.state.Folders...)
	folderItems := append([]VfsFolderItem(nil), s.state.FolderItems...)
	s.mu.RUnlock()

	itemLookup := folderItemLookup(folderItems)
	folderPathIDs := folderPathIDLookup(folders)
	folderPaths := folderTitlePathLookup(folders)
	out := make([]notes.ExportNoteRecord, 0, len(noteRecords))
	for _, note := range noteRecords {
		if !includeDeleted && noteIsDeleted(note) {
			continue
		}
		node := noteToNode(note)
		applyFolderItemToNode(&node, itemLookup, folderPathIDs, folderPaths)
		out = append(out, exportRecordFromNode(note, node))
	}
	return out
}

func (s *Service) ExportNoteRecord(noteID string, includeDeleted bool) (notes.ExportNoteRecord, bool) {
	id := idFromPath(noteID)
	if id == "" {
		return notes.ExportNoteRecord{}, false
	}
	s.mu.RLock()
	note, ok := s.findNoteLocked(id)
	folders := append([]VfsFolder(nil), s.state.Folders...)
	folderItems := append([]VfsFolderItem(nil), s.state.FolderItems...)
	s.mu.RUnlock()
	if !ok || (!includeDeleted && noteIsDeleted(note)) {
		return notes.ExportNoteRecord{}, false
	}
	node := noteToNode(note)
	applyFolderItemToNode(&node, folderItemLookup(folderItems), folderPathIDLookup(folders), folderTitlePathLookup(folders))
	return exportRecordFromNode(note, node), true
}

func (s *Service) ImportNoteRecords(records []notes.ExportNoteRecord, conflictStrategy string) (notes.ImportResult, error) {
	strategy := strings.ToLower(strings.TrimSpace(conflictStrategy))
	if strategy == "" {
		strategy = "skip"
	}
	if strategy != "skip" && strategy != "overwrite" && strategy != "merge_keep_newer" {
		strategy = "skip"
	}

	now := nowMillis()
	result := notes.ImportResult{}
	s.mu.Lock()
	defer s.mu.Unlock()

	for _, record := range records {
		record.ID = cleanImportedNoteID(record.ID)
		if record.ID == "" {
			record.ID = "note_" + randomToken(16)
		}
		if strings.TrimSpace(record.Title) == "" {
			record.Title = "Untitled"
		}
		metadata := normalizeMetadata(record.Metadata)
		metadata["tags"] = append([]string(nil), record.Tags...)
		metadata["isFavorite"] = record.IsFavorite
		metadata["status"] = "active"
		metadata["deletedAt"] = ""
		createdAt := parseExportTimeMillis(record.CreatedAt, now)
		updatedAt := parseExportTimeMillis(record.UpdatedAt, createdAt)
		if updatedAt <= 0 {
			updatedAt = now
		}
		if createdAt <= 0 {
			createdAt = updatedAt
		}

		if index, exists := s.findNoteIndexLocked(record.ID); exists {
			switch strategy {
			case "skip":
				result.SkippedCount++
				continue
			case "merge_keep_newer":
				if updatedAt <= s.state.Notes[index].UpdatedAt {
					result.SkippedCount++
					continue
				}
			}
			previous := s.state.Notes[index]
			s.state.Notes[index].Name = record.Title
			s.state.Notes[index].Content = record.ContentMD
			s.state.Notes[index].Metadata = metadata
			s.state.Notes[index].CreatedAt = createdAt
			s.state.Notes[index].UpdatedAt = updatedAt
			if err := s.syncNoteResourceLocked(index); err != nil {
				s.state.Notes[index] = previous
				return result, err
			}
			result.NoteCount++
			result.OverwrittenCount++
			continue
		}

		imported := NoteRecord{
			ID:        record.ID,
			Name:      record.Title,
			Content:   record.ContentMD,
			Metadata:  metadata,
			CreatedAt: createdAt,
			UpdatedAt: updatedAt,
		}
		s.state.Notes = append(s.state.Notes, imported)
		index := len(s.state.Notes) - 1
		if err := s.syncNoteResourceLocked(index); err != nil {
			s.state.Notes = s.state.Notes[:index]
			return result, err
		}
		result.NoteCount++
	}

	if err := s.flushLocked(); err != nil {
		return result, err
	}
	return result, nil
}

func exportRecordFromNode(note NoteRecord, node Node) notes.ExportNoteRecord {
	metadata := normalizeMetadata(node.Metadata)
	return notes.ExportNoteRecord{
		ID:         note.ID,
		Title:      note.Name,
		ContentMD:  note.Content,
		Tags:       metadataTags(metadata),
		CreatedAt:  formatMillis(note.CreatedAt),
		UpdatedAt:  formatMillis(note.UpdatedAt),
		IsFavorite: metadataFavorite(metadata),
		Metadata:   metadata,
	}
}

func cleanImportedNoteID(value string) string {
	trimmed := idFromPath(value)
	if trimmed == "" {
		return ""
	}
	for _, char := range trimmed {
		if (char >= 'A' && char <= 'Z') || (char >= 'a' && char <= 'z') || (char >= '0' && char <= '9') || char == '_' || char == '-' {
			continue
		}
		return ""
	}
	return trimmed
}

func parseExportTimeMillis(value string, fallback int64) int64 {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return fallback
	}
	for _, layout := range []string{time.RFC3339Nano, time.RFC3339, "2006-01-02 15:04:05"} {
		parsed, err := time.Parse(layout, trimmed)
		if err == nil {
			return parsed.UnixMilli()
		}
	}
	return fallback
}
