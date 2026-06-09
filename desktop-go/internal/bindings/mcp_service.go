package bindings

import "deep-student-go/internal/app"

type McpService struct {
	app *app.App
}

func NewMcpService(app *app.App) *McpService {
	return &McpService{app: app}
}

func (s *McpService) StartStdioSession(command string, args []string, env map[string]string, framing *string, cwd *string) (string, error) {
	return s.app.Mcp.StartStdioSession(command, args, env, framing, cwd)
}

func (s *McpService) SendStdioMessage(sessionID string, payload string) error {
	return s.app.Mcp.SendStdioMessage(sessionID, payload)
}

func (s *McpService) CloseStdioSession(sessionID string) error {
	return s.app.Mcp.CloseStdioSession(sessionID)
}
