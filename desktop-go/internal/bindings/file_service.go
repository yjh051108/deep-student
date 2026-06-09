package bindings

import "deep-student-go/internal/app"

type FileService struct {
	app *app.App
}

func NewFileService(app *app.App) *FileService {
	return &FileService{app: app}
}

func (s *FileService) ReadFileBytes(path string) ([]byte, error) {
	return s.app.Files.ReadFileBytes(path)
}

func (s *FileService) ReadFileText(path string) (string, error) {
	return s.app.Files.ReadFileText(path)
}

func (s *FileService) SaveTextToFile(path string, content string) error {
	return s.app.Files.SaveTextToFile(path, content)
}

func (s *FileService) GetFileSize(path string) (int64, error) {
	return s.app.Files.GetFileSize(path)
}

func (s *FileService) CopyFile(sourcePath string, destPath string) error {
	return s.app.Files.CopyFile(sourcePath, destPath)
}
