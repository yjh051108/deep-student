package skills

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
)

const (
	skillFileName     = "SKILL.md"
	maxSkillFileBytes = 512 * 1024
)

type DirectoryEntry struct {
	Name string `json:"name"`
	Path string `json:"path"`
}

type FileContent struct {
	Content string `json:"content"`
	Path    string `json:"path"`
}

type Service struct {
	dataDir string
}

func NewService(dataDir string) *Service {
	return &Service{dataDir: filepath.Clean(dataDir)}
}

func (s *Service) ListDirectories(path string) ([]DirectoryEntry, error) {
	dir, err := s.validateSkillRoot(path, false)
	if err != nil {
		return nil, err
	}
	info, err := os.Stat(dir)
	if errors.Is(err, os.ErrNotExist) {
		return []DirectoryEntry{}, nil
	}
	if err != nil {
		return nil, err
	}
	if !info.IsDir() {
		return nil, fmt.Errorf("path is not a directory: %s", dir)
	}

	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}

	result := make([]DirectoryEntry, 0, len(entries))
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		if entry.Type()&os.ModeSymlink != 0 {
			continue
		}
		name := entry.Name()
		if strings.HasPrefix(name, ".") {
			continue
		}
		result = append(result, DirectoryEntry{
			Name: name,
			Path: filepath.Join(dir, name),
		})
	}
	sort.Slice(result, func(i, j int) bool {
		return strings.ToLower(result[i].Name) < strings.ToLower(result[j].Name)
	})
	return result, nil
}

func (s *Service) ReadFile(path string) (FileContent, error) {
	file, err := s.validateSkillFilePath(path, false)
	if err != nil {
		return FileContent{}, err
	}
	info, err := os.Lstat(file)
	if err != nil {
		return FileContent{}, err
	}
	if info.IsDir() {
		return FileContent{}, fmt.Errorf("path is not a file: %s", file)
	}
	if info.Size() > maxSkillFileBytes {
		return FileContent{}, fmt.Errorf("skill file exceeds %d bytes: %s", maxSkillFileBytes, file)
	}
	content, err := os.ReadFile(file)
	if err != nil {
		return FileContent{}, err
	}
	return FileContent{Content: string(content), Path: file}, nil
}

func (s *Service) Create(basePath string, skillID string, content string) (FileContent, error) {
	if err := validateSkillID(skillID); err != nil {
		return FileContent{}, err
	}
	if err := validateContentSize(content); err != nil {
		return FileContent{}, err
	}
	base, err := s.validateSkillRoot(basePath, true)
	if err != nil {
		return FileContent{}, err
	}
	skillDir := filepath.Join(base, skillID)
	skillFile := filepath.Join(skillDir, skillFileName)
	if _, err := s.validateSkillFilePath(skillFile, true); err != nil {
		return FileContent{}, err
	}
	if _, err := os.Lstat(skillDir); err == nil {
		return FileContent{}, fmt.Errorf("skill directory already exists: %s", skillDir)
	} else if err != nil && !errors.Is(err, os.ErrNotExist) {
		return FileContent{}, err
	}
	if err := os.MkdirAll(base, 0o700); err != nil {
		return FileContent{}, err
	}
	if err := os.Mkdir(skillDir, 0o700); err != nil {
		return FileContent{}, err
	}
	if err := os.WriteFile(skillFile, []byte(content), 0o600); err != nil {
		return FileContent{}, err
	}
	return FileContent{Content: content, Path: skillFile}, nil
}

func (s *Service) Update(path string, content string) (FileContent, error) {
	if err := validateContentSize(content); err != nil {
		return FileContent{}, err
	}
	file, err := s.validateSkillFilePath(path, true)
	if err != nil {
		return FileContent{}, err
	}
	info, err := os.Lstat(file)
	if err != nil {
		return FileContent{}, err
	}
	if info.IsDir() {
		return FileContent{}, fmt.Errorf("path is not a file: %s", file)
	}
	if err := os.WriteFile(file, []byte(content), 0o600); err != nil {
		return FileContent{}, err
	}
	return FileContent{Content: content, Path: file}, nil
}

func (s *Service) Delete(path string) error {
	dir, err := s.validateSkillDirForDelete(path)
	if err != nil {
		return err
	}
	info, err := os.Lstat(dir)
	if err != nil {
		return err
	}
	if !info.IsDir() {
		return fmt.Errorf("path is not a directory: %s", dir)
	}
	skillFile := filepath.Join(dir, skillFileName)
	if _, err := os.Stat(skillFile); err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return fmt.Errorf("not a valid skill directory, missing %s: %s", skillFileName, dir)
		}
		return err
	}
	return os.RemoveAll(dir)
}

func (s *Service) expandPath(path string) (string, error) {
	trimmed := strings.TrimSpace(path)
	if trimmed == "" {
		return "", errors.New("path cannot be empty")
	}
	if strings.Contains(trimmed, "://") {
		return "", errors.New("only local skill paths are supported")
	}

	if trimmed == "~" || strings.HasPrefix(trimmed, "~/") || strings.HasPrefix(trimmed, `~\`) {
		home, err := os.UserHomeDir()
		if err != nil || home == "" {
			return "", errors.New("cannot resolve user home directory")
		}
		if trimmed == "~" {
			return filepath.Clean(home), nil
		}
		return filepath.Clean(filepath.Join(home, trimmed[2:])), nil
	}

	if filepath.IsAbs(trimmed) {
		return filepath.Clean(trimmed), nil
	}

	cwd, err := os.Getwd()
	if err != nil {
		return "", err
	}
	return filepath.Clean(filepath.Join(cwd, trimmed)), nil
}

func (s *Service) validateSkillRoot(path string, writable bool) (string, error) {
	candidate, err := s.expandPath(path)
	if err != nil {
		return "", err
	}
	candidate, err = cleanAbs(candidate)
	if err != nil {
		return "", err
	}
	if _, err := s.matchAllowedRoot(candidate, writable); err != nil {
		return "", err
	}
	if err := rejectSymlinkedExistingPath(candidate); err != nil {
		return "", err
	}
	return candidate, nil
}

func (s *Service) validateSkillFilePath(path string, writable bool) (string, error) {
	file, err := s.expandPath(path)
	if err != nil {
		return "", err
	}
	file, err = cleanAbs(file)
	if err != nil {
		return "", err
	}
	if filepath.Base(file) != skillFileName {
		return "", fmt.Errorf("skill file path must end with %s: %s", skillFileName, file)
	}

	skillDir := filepath.Dir(file)
	skillID := filepath.Base(skillDir)
	if err := validateSkillID(skillID); err != nil {
		return "", err
	}
	root := filepath.Dir(skillDir)
	if _, err := s.matchAllowedRoot(root, writable); err != nil {
		return "", err
	}
	if err := rejectSymlinkedExistingPath(file); err != nil {
		return "", err
	}
	return file, nil
}

func (s *Service) validateSkillDirForDelete(path string) (string, error) {
	dir, err := s.expandPath(path)
	if err != nil {
		return "", err
	}
	dir, err = cleanAbs(dir)
	if err != nil {
		return "", err
	}
	if err := validateSkillID(filepath.Base(dir)); err != nil {
		return "", err
	}
	root := filepath.Dir(dir)
	if _, err := s.matchAllowedRoot(root, true); err != nil {
		return "", err
	}
	if err := rejectSymlinkedExistingPath(dir); err != nil {
		return "", err
	}
	if _, err := s.validateSkillFilePath(filepath.Join(dir, skillFileName), true); err != nil {
		return "", err
	}
	return dir, nil
}

func (s *Service) matchAllowedRoot(path string, writable bool) (string, error) {
	candidate, err := cleanAbs(path)
	if err != nil {
		return "", err
	}
	for _, root := range s.allowedRoots() {
		if writable && !root.writable {
			continue
		}
		base, err := cleanAbs(root.path)
		if err != nil {
			continue
		}
		if samePath(candidate, base) {
			return base, nil
		}
	}
	access := "read"
	if writable {
		access = "write"
	}
	return "", fmt.Errorf("skill path is outside allowed %s roots: %s", access, path)
}

type allowedRoot struct {
	path     string
	writable bool
}

func (s *Service) allowedRoots() []allowedRoot {
	roots := []allowedRoot{}
	if home, err := os.UserHomeDir(); err == nil && home != "" {
		roots = append(roots,
			allowedRoot{path: filepath.Join(home, ".cursor", "skills-cursor"), writable: false},
			allowedRoot{path: filepath.Join(home, ".deep-student", "skills"), writable: true},
		)
	}
	if s.dataDir != "" {
		roots = append(roots,
			allowedRoot{path: filepath.Join(s.dataDir, "skills"), writable: true},
			allowedRoot{path: filepath.Join(s.dataDir, ".skills"), writable: true},
		)
	}
	if runtime.GOOS == "windows" {
		if local := os.Getenv("LOCALAPPDATA"); local != "" {
			roots = append(roots,
				allowedRoot{path: filepath.Join(local, "ds91", "skills"), writable: true},
				allowedRoot{path: filepath.Join(local, "deep-student", "skills"), writable: true},
			)
		}
	}
	if cwd, err := os.Getwd(); err == nil && cwd != "" {
		roots = append(roots, allowedRoot{path: filepath.Join(cwd, ".skills"), writable: true})
	}
	return roots
}

func cleanAbs(path string) (string, error) {
	cleaned := filepath.Clean(path)
	if !filepath.IsAbs(cleaned) {
		cwd, err := os.Getwd()
		if err != nil {
			return "", err
		}
		cleaned = filepath.Join(cwd, cleaned)
	}
	return filepath.Clean(cleaned), nil
}

func rejectSymlinkedExistingPath(path string) error {
	candidate, err := cleanAbs(path)
	if err != nil {
		return err
	}

	existing := candidate
	existingPaths := []string{}
	for {
		if _, err := os.Lstat(existing); err == nil {
			existingPaths = append(existingPaths, existing)
			parent := filepath.Dir(existing)
			if parent == existing {
				break
			}
			existing = parent
			continue
		} else if !errors.Is(err, os.ErrNotExist) {
			return err
		}
		parent := filepath.Dir(existing)
		if parent == existing {
			return nil
		}
		existing = parent
	}

	for _, existingPath := range existingPaths {
		linkInfo, err := os.Lstat(existingPath)
		if err != nil {
			return err
		}
		targetInfo, err := os.Stat(existingPath)
		if err != nil {
			return fmt.Errorf("skill path uses a symlink or reparse point: %s", path)
		}
		if !os.SameFile(linkInfo, targetInfo) {
			return fmt.Errorf("skill path uses a symlink or reparse point: %s", path)
		}
	}
	return nil
}

func samePath(a string, b string) bool {
	a = filepath.Clean(a)
	b = filepath.Clean(b)
	if runtime.GOOS == "windows" {
		return strings.EqualFold(a, b)
	}
	return a == b
}

func validateContentSize(content string) error {
	if len(content) > maxSkillFileBytes {
		return fmt.Errorf("skill content exceeds %d bytes", maxSkillFileBytes)
	}
	return nil
}

func validateSkillID(skillID string) error {
	trimmed := strings.TrimSpace(skillID)
	if trimmed == "" {
		return errors.New("skill ID cannot be empty")
	}
	if trimmed != skillID {
		return errors.New("skill ID cannot contain leading or trailing whitespace")
	}
	if len(trimmed) > 64 {
		return errors.New("skill ID cannot exceed 64 characters")
	}
	if isWindowsReservedName(trimmed) {
		return fmt.Errorf("skill ID uses a reserved filename: %s", trimmed)
	}
	for i, r := range trimmed {
		if r > 127 {
			return errors.New("skill ID can only contain ASCII letters, numbers, hyphens, and underscores")
		}
		isAlpha := (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z')
		isDigit := r >= '0' && r <= '9'
		if i == 0 && !isAlpha && !isDigit {
			return errors.New("skill ID must start with a letter or number")
		}
		if isAlpha || isDigit || r == '-' || r == '_' {
			continue
		}
		return errors.New("skill ID can only contain letters, numbers, hyphens, and underscores")
	}
	return nil
}

func isWindowsReservedName(name string) bool {
	upper := strings.ToUpper(name)
	switch upper {
	case "CON", "PRN", "AUX", "NUL", "CLOCK$":
		return true
	}
	if len(upper) == 4 {
		prefix := upper[:3]
		suffix := upper[3]
		if (prefix == "COM" || prefix == "LPT") && suffix >= '1' && suffix <= '9' {
			return true
		}
	}
	return false
}
