package bindings

import (
	"deep-student-go/internal/app"
	"deep-student-go/internal/system"
)

type SystemService struct {
	app *app.App
}

func NewSystemService(app *app.App) *SystemService {
	return &SystemService{app: app}
}

func (s *SystemService) AppDataDir() string {
	return s.app.System.AppDataDir()
}

func (s *SystemService) EnsureDebugLogDir() (string, error) {
	return s.app.System.EnsureDebugLogDir()
}

func (s *SystemService) OpenLogsFolder(logType string) error {
	return s.app.System.OpenLogsFolder(logType)
}

func (s *SystemService) ReportFrontendLog(payload system.FrontendLogPayload) error {
	return s.app.System.ReportFrontendLog(payload)
}

func (s *SystemService) SaveWebviewSettings(settings map[string]any) (system.SaveWebviewSettingsResult, error) {
	return s.app.System.SaveWebviewSettings(settings)
}
