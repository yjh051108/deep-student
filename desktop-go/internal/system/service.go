package system

import (
	"encoding/json"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
	"time"
)

var logTypePattern = regexp.MustCompile(`^[A-Za-z0-9_-]+$`)

type Service struct {
	dataDir string
}

type FrontendLogPayload struct {
	Level     string         `json:"level,omitempty"`
	Message   string         `json:"message"`
	Stack     string         `json:"stack,omitempty"`
	Component string         `json:"component,omitempty"`
	Route     string         `json:"route,omitempty"`
	URL       string         `json:"url,omitempty"`
	Line      *uint          `json:"line,omitempty"`
	Column    *uint          `json:"column,omitempty"`
	UserAgent string         `json:"user_agent,omitempty"`
	Extra     map[string]any `json:"extra,omitempty"`
	Kind      string         `json:"kind,omitempty"`
}

type SaveWebviewSettingsResult struct {
	Success bool   `json:"success"`
	Path    string `json:"path"`
	Size    int    `json:"size"`
}

type frontendLogEntry struct {
	Timestamp string             `json:"timestamp"`
	Level     string             `json:"level"`
	Kind      string             `json:"kind"`
	Source    string             `json:"source"`
	Payload   FrontendLogPayload `json:"payload"`
}

func NewService(dataDir string) (*Service, error) {
	cleaned, err := ensureDir(dataDir)
	if err != nil {
		return nil, err
	}
	return &Service{dataDir: cleaned}, nil
}

func (s *Service) AppDataDir() string {
	return s.dataDir
}

func (s *Service) EnsureDebugLogDir() (string, error) {
	return ensureDir(filepath.Join(s.dataDir, "debug-logs"))
}

func (s *Service) LogDir(logType string) (string, error) {
	cleanedType := strings.TrimSpace(logType)
	if cleanedType == "" {
		cleanedType = "backend"
	}
	if !logTypePattern.MatchString(cleanedType) {
		return "", errors.New("invalid log type")
	}

	dir, err := ensureDir(filepath.Join(s.dataDir, "logs", cleanedType))
	if err != nil {
		return "", err
	}
	return ensureInside(s.dataDir, dir)
}

func (s *Service) OpenLogsFolder(logType string) error {
	dir, err := s.LogDir(logType)
	if err != nil {
		return err
	}

	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "windows":
		cmd = exec.Command("explorer", dir)
	case "darwin":
		cmd = exec.Command("open", dir)
	default:
		cmd = exec.Command("xdg-open", dir)
	}
	return cmd.Start()
}

func (s *Service) ReportFrontendLog(payload FrontendLogPayload) error {
	if strings.TrimSpace(payload.Message) == "" {
		return errors.New("frontend log message cannot be empty")
	}

	dir, err := s.LogDir("frontend")
	if err != nil {
		return err
	}

	entry := frontendLogEntry{
		Timestamp: time.Now().UTC().Format(time.RFC3339Nano),
		Level:     normalizeLevel(payload.Level),
		Kind:      normalizeKind(payload.Kind),
		Source:    "FRONTEND",
		Payload:   payload,
	}
	bytes, err := json.Marshal(entry)
	if err != nil {
		return err
	}
	bytes = append(bytes, '\n')

	path := filepath.Join(dir, "frontend-go.jsonl")
	file, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o600)
	if err != nil {
		return err
	}
	defer file.Close()

	_, err = file.Write(bytes)
	return err
}

func (s *Service) SaveWebviewSettings(settings map[string]any) (SaveWebviewSettingsResult, error) {
	bytes, err := json.MarshalIndent(settings, "", "  ")
	if err != nil {
		return SaveWebviewSettingsResult{}, err
	}

	path := filepath.Join(s.dataDir, "webview_settings.json")
	if err := os.WriteFile(path, bytes, 0o600); err != nil {
		return SaveWebviewSettingsResult{}, err
	}

	return SaveWebviewSettingsResult{
		Success: true,
		Path:    path,
		Size:    len(bytes),
	}, nil
}

func normalizeLevel(level string) string {
	switch strings.ToUpper(strings.TrimSpace(level)) {
	case "TRACE", "DEBUG", "INFO", "WARN", "WARNING", "ERROR":
		if strings.ToUpper(strings.TrimSpace(level)) == "WARNING" {
			return "WARN"
		}
		return strings.ToUpper(strings.TrimSpace(level))
	default:
		return "ERROR"
	}
}

func normalizeKind(kind string) string {
	trimmed := strings.TrimSpace(kind)
	if trimmed == "" {
		return "CLIENT_ERROR"
	}
	return trimmed
}

func ensureDir(dir string) (string, error) {
	cleaned := filepath.Clean(dir)
	if err := os.MkdirAll(cleaned, 0o700); err != nil {
		return "", err
	}
	return cleaned, nil
}

func ensureInside(base string, target string) (string, error) {
	baseAbs, err := filepath.Abs(base)
	if err != nil {
		return "", err
	}
	targetAbs, err := filepath.Abs(target)
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
	return "", errors.New("path escapes app data directory")
}
