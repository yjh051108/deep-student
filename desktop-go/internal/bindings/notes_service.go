package bindings

import (
	"deep-student-go/internal/app"
	"deep-student-go/internal/notes"
)

type NotesService struct {
	app *app.App
}

func NewNotesService(app *app.App) *NotesService {
	return &NotesService{app: app}
}

func (s *NotesService) SetPref(key string, value string) (bool, error) {
	return s.app.Notes.SetPref(key, value)
}

func (s *NotesService) GetPref(key string) (*string, error) {
	value, ok := s.app.Notes.GetPref(key)
	if !ok {
		return nil, nil
	}
	return &value, nil
}

func (s *NotesService) SaveAsset(subject string, noteID string, base64Data string, defaultExt *string) (notes.AssetRef, error) {
	ext := "png"
	if defaultExt != nil {
		ext = *defaultExt
	}
	return s.app.Notes.SaveAsset(subject, noteID, base64Data, ext)
}

func (s *NotesService) ListAssets(subject string, noteID string) ([]notes.AssetRef, error) {
	return s.app.Notes.ListAssets(subject, noteID)
}

func (s *NotesService) AssetsIndexScan(noteID string) (int, error) {
	return s.app.Notes.AssetsIndexScan(noteID)
}

func (s *NotesService) ScanOrphanAssets() ([]string, error) {
	return s.app.Notes.ScanOrphanAssets(s.app.Dstu.NoteAssetReferences(true))
}

func (s *NotesService) BulkDeleteAssets(paths []string) (int, error) {
	return s.app.Notes.BulkDeleteAssets(paths)
}

func (s *NotesService) DBStats() (notes.DBStats, error) {
	return s.app.Notes.DBStats(s.app.Dstu.CountNotes(false))
}

func (s *NotesService) DBVacuum() (bool, error) {
	return s.app.Notes.DBVacuum()
}

func (s *NotesService) Export(request notes.ExportRequest) (notes.ExportResult, error) {
	return s.app.Notes.Export(s.app.Dstu.ExportNoteRecords(false), request)
}

func (s *NotesService) ExportSingle(request notes.ExportSingleRequest) (notes.ExportResult, error) {
	record, ok := s.app.Dstu.ExportNoteRecord(request.NoteID, false)
	if !ok {
		return notes.ExportResult{}, nil
	}
	return s.app.Notes.ExportSingle(record, request)
}

func (s *NotesService) Import(request notes.ImportRequest) (notes.ImportResult, error) {
	archive, err := s.app.Notes.ReadImportArchive(request)
	if err != nil {
		return notes.ImportResult{}, err
	}
	result, err := s.app.Dstu.ImportNoteRecords(archive.Notes, derefString(request.ConflictStrategy))
	if err != nil {
		return result, err
	}
	result.AttachmentCount = archive.AttachmentCount
	return result, nil
}

func (s *NotesService) DeleteAsset(relativePath string) (bool, error) {
	return s.app.Notes.DeleteAsset(relativePath)
}

func (s *NotesService) ResolveAssetPath(relativePath string) (string, error) {
	return s.app.Notes.ResolveAssetPath(relativePath)
}

func (s *NotesService) GetImageAsBase64(relativePath string) (string, error) {
	return s.app.Notes.GetImageAsBase64(relativePath)
}

func derefString(value *string) string {
	if value == nil {
		return ""
	}
	return *value
}
