package bindings

import (
	"deep-student-go/internal/app"
	"deep-student-go/internal/templates"
)

type TemplateService struct {
	app *app.App
}

func NewTemplateService(app *app.App) *TemplateService {
	return &TemplateService{app: app}
}

func (s *TemplateService) ImportBuiltinTemplates() (string, error) {
	return s.app.Templates.ImportBuiltinTemplates()
}

func (s *TemplateService) GetAllCustomTemplates() ([]templates.Template, error) {
	return s.app.Templates.GetAllCustomTemplates()
}

func (s *TemplateService) GetDefaultTemplateID() (*string, error) {
	return s.app.Templates.GetDefaultTemplateID()
}

func (s *TemplateService) CreateCustomTemplate(request map[string]any) (string, error) {
	return s.app.Templates.CreateCustomTemplate(request)
}

func (s *TemplateService) UpdateCustomTemplate(templateID string, request map[string]any) error {
	return s.app.Templates.UpdateCustomTemplate(templateID, request)
}

func (s *TemplateService) DeleteCustomTemplate(templateID string) error {
	return s.app.Templates.DeleteCustomTemplate(templateID)
}

func (s *TemplateService) SetDefaultTemplate(templateID string) error {
	return s.app.Templates.SetDefaultTemplate(templateID)
}

func (s *TemplateService) ImportCustomTemplatesBulk(templateData string, overwriteExisting bool, strictBuiltin bool) (string, error) {
	return s.app.Templates.ImportCustomTemplatesBulk(templateData, overwriteExisting, strictBuiltin)
}

func (s *TemplateService) ExportTemplate(templateID string) (templates.TemplateExportResponse, error) {
	return s.app.Templates.ExportTemplate(templateID)
}
