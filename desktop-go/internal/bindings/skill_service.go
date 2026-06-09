package bindings

import (
	"deep-student-go/internal/app"
	"deep-student-go/internal/skills"
)

type SkillService struct {
	app *app.App
}

func NewSkillService(app *app.App) *SkillService {
	return &SkillService{app: app}
}

func (s *SkillService) ListDirectories(path string) ([]skills.DirectoryEntry, error) {
	return s.app.Skills.ListDirectories(path)
}

func (s *SkillService) ReadFile(path string) (skills.FileContent, error) {
	return s.app.Skills.ReadFile(path)
}

func (s *SkillService) Create(basePath string, skillID string, content string) (skills.FileContent, error) {
	return s.app.Skills.Create(basePath, skillID, content)
}

func (s *SkillService) Update(path string, content string) (skills.FileContent, error) {
	return s.app.Skills.Update(path, content)
}

func (s *SkillService) Delete(path string) error {
	return s.app.Skills.Delete(path)
}
