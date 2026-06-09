package files

import (
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"
)

type Service struct{}

func NewService() *Service {
	return &Service{}
}

func (s *Service) ReadFileBytes(path string) ([]byte, error) {
	cleaned, err := cleanLocalPath(path)
	if err != nil {
		return nil, err
	}
	return os.ReadFile(cleaned)
}

func (s *Service) ReadFileText(path string) (string, error) {
	bytes, err := s.ReadFileBytes(path)
	if err != nil {
		return "", err
	}
	return string(bytes), nil
}

func (s *Service) SaveTextToFile(path string, content string) error {
	cleaned, err := cleanLocalPath(path)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(cleaned), 0o700); err != nil {
		return err
	}
	return os.WriteFile(cleaned, []byte(content), 0o600)
}

func (s *Service) GetFileSize(path string) (int64, error) {
	cleaned, err := cleanLocalPath(path)
	if err != nil {
		return 0, err
	}
	info, err := os.Stat(cleaned)
	if err != nil {
		return 0, err
	}
	if info.IsDir() {
		return 0, errors.New("path is a directory")
	}
	return info.Size(), nil
}

func (s *Service) CopyFile(sourcePath string, destPath string) error {
	source, err := cleanLocalPath(sourcePath)
	if err != nil {
		return err
	}
	dest, err := cleanLocalPath(destPath)
	if err != nil {
		return err
	}
	if source == dest {
		return nil
	}

	in, err := os.Open(source)
	if err != nil {
		return err
	}
	defer in.Close()

	if err := os.MkdirAll(filepath.Dir(dest), 0o700); err != nil {
		return err
	}

	out, err := os.OpenFile(dest, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o600)
	if err != nil {
		return err
	}
	defer out.Close()

	if _, err := io.Copy(out, in); err != nil {
		return err
	}
	return out.Sync()
}

func cleanLocalPath(path string) (string, error) {
	trimmed := strings.TrimSpace(path)
	if trimmed == "" {
		return "", errors.New("path cannot be empty")
	}
	if strings.Contains(trimmed, "://") {
		return "", errors.New("only local file paths are supported by the Go file service")
	}
	return filepath.Clean(trimmed), nil
}
