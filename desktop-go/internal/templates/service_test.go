package templates

import (
	"database/sql"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestImportBuiltinTemplatesIsIdempotentAndSetsDefault(t *testing.T) {
	service := newTestService(t)

	result, err := service.ImportBuiltinTemplates()
	if err != nil {
		t.Fatalf("ImportBuiltinTemplates failed: %v", err)
	}
	if !strings.Contains(result, "新增 6 个") {
		t.Fatalf("expected six lean builtins to be added, got %q", result)
	}

	templates, err := service.GetAllCustomTemplates()
	if err != nil {
		t.Fatalf("GetAllCustomTemplates failed: %v", err)
	}
	if len(templates) != 6 {
		t.Fatalf("expected 6 templates, got %d", len(templates))
	}
	defaultID, err := service.GetDefaultTemplateID()
	if err != nil {
		t.Fatalf("GetDefaultTemplateID failed: %v", err)
	}
	if defaultID == nil || *defaultID != "builtin-minimal" {
		t.Fatalf("expected builtin-minimal default id, got %#v", defaultID)
	}

	result, err = service.ImportBuiltinTemplates()
	if err != nil {
		t.Fatalf("second ImportBuiltinTemplates failed: %v", err)
	}
	if !strings.Contains(result, "跳过 6 个") {
		t.Fatalf("expected second import to skip existing templates, got %q", result)
	}

	reopened, err := NewService(t.TempDir())
	if err != nil {
		t.Fatalf("empty service reopen failed: %v", err)
	}
	emptyTemplates, err := reopened.GetAllCustomTemplates()
	if err != nil {
		t.Fatalf("empty list failed: %v", err)
	}
	if len(emptyTemplates) != 0 {
		t.Fatalf("expected empty store to list zero templates, got %d", len(emptyTemplates))
	}
}

func TestTemplateCRUDVersioningPersistenceAndExport(t *testing.T) {
	dataDir := t.TempDir()
	service, err := NewService(dataDir)
	if err != nil {
		t.Fatalf("NewService failed: %v", err)
	}

	id, err := service.CreateCustomTemplate(map[string]any{
		"name":                   "Custom",
		"description":            "desc",
		"fields":                 []any{"Front", "Back"},
		"front_template":         `<div onclick="evil()"><script>alert(1)</script>{{Front}}</div>`,
		"back_template":          `<a href="javascript:alert(1)">{{Back}}</a>`,
		"css_style":              `.card { color: red; } @import url("x");`,
		"field_extraction_rules": map[string]any{"Front": map[string]any{"field_type": "Text"}},
	})
	if err != nil {
		t.Fatalf("CreateCustomTemplate failed: %v", err)
	}

	templates, err := service.GetAllCustomTemplates()
	if err != nil {
		t.Fatalf("GetAllCustomTemplates failed: %v", err)
	}
	created := requireTemplate(t, templates, id)
	if created["is_built_in"] != false {
		t.Fatalf("expected custom template to not be built in: %#v", created["is_built_in"])
	}
	if strings.Contains(stringValue(created["front_template"]), "script") || strings.Contains(stringValue(created["front_template"]), "onclick") {
		t.Fatalf("front template was not defanged: %q", created["front_template"])
	}
	if strings.Contains(strings.ToLower(stringValue(created["back_template"])), "javascript:") {
		t.Fatalf("back template was not defanged: %q", created["back_template"])
	}
	if strings.Contains(strings.ToLower(stringValue(created["css_style"])), "@import") {
		t.Fatalf("css style was not defanged: %q", created["css_style"])
	}

	if err := service.SetDefaultTemplate(id); err != nil {
		t.Fatalf("SetDefaultTemplate failed: %v", err)
	}
	if err := service.UpdateCustomTemplate(id, map[string]any{
		"expected_version": created["version"],
		"description":      "updated",
	}); err != nil {
		t.Fatalf("UpdateCustomTemplate failed: %v", err)
	}
	afterUpdate, err := service.GetAllCustomTemplates()
	if err != nil {
		t.Fatalf("GetAllCustomTemplates after update failed: %v", err)
	}
	updated := requireTemplate(t, afterUpdate, id)
	if updated["description"] != "updated" {
		t.Fatalf("description was not updated: %#v", updated["description"])
	}
	if updated["version"] == created["version"] {
		t.Fatalf("expected version bump, still %v", updated["version"])
	}
	if err := service.UpdateCustomTemplate(id, map[string]any{
		"expected_version": created["version"],
		"description":      "stale",
	}); err == nil {
		t.Fatal("expected stale expected_version to fail")
	}

	exported, err := service.ExportTemplate(id)
	if err != nil {
		t.Fatalf("ExportTemplate failed: %v", err)
	}
	var exportedObject map[string]any
	if err := json.Unmarshal([]byte(exported.TemplateData), &exportedObject); err != nil {
		t.Fatalf("exported template data is invalid JSON: %v", err)
	}
	if exportedObject["id"] != id {
		t.Fatalf("exported wrong template id: %#v", exportedObject["id"])
	}

	reopened, err := NewService(dataDir)
	if err != nil {
		t.Fatalf("reopen failed: %v", err)
	}
	defaultID, err := reopened.GetDefaultTemplateID()
	if err != nil {
		t.Fatalf("reopen default failed: %v", err)
	}
	if defaultID == nil || *defaultID != id {
		t.Fatalf("expected default id to persist as %s, got %#v", id, defaultID)
	}
	if err := reopened.DeleteCustomTemplate(id); err != nil {
		t.Fatalf("DeleteCustomTemplate failed: %v", err)
	}
	defaultID, err = reopened.GetDefaultTemplateID()
	if err != nil {
		t.Fatalf("default after delete failed: %v", err)
	}
	if defaultID != nil {
		t.Fatalf("expected custom default to clear after delete, got %#v", defaultID)
	}
}

func TestBulkImportNormalizesLegacyFieldsJSON(t *testing.T) {
	service := newTestService(t)
	payload := `[
		{
			"id": "legacy-one",
			"name": "Legacy One",
			"description": "legacy",
			"fields_json": "[\"Front\",\"Back\"]",
			"field_extraction_rules_json": "{\"Front\":{\"field_type\":\"Text\",\"is_required\":true,\"description\":\"front\"}}",
			"front_template": "{{Front}}",
			"back_template": "{{Back}}",
			"css_style": ".card { color: blue; }",
			"is_built_in": false
		}
	]`

	result, err := service.ImportCustomTemplatesBulk(payload, true, true)
	if err != nil {
		t.Fatalf("ImportCustomTemplatesBulk failed: %v", err)
	}
	if !strings.Contains(result, "新增 1 个") {
		t.Fatalf("expected one import, got %q", result)
	}
	templates, err := service.GetAllCustomTemplates()
	if err != nil {
		t.Fatalf("GetAllCustomTemplates failed: %v", err)
	}
	imported := requireTemplate(t, templates, "legacy-one")
	fields, ok := imported["fields"].([]any)
	if !ok || len(fields) != 2 || fields[0] != "Front" {
		t.Fatalf("fields were not normalized: %#v", imported["fields"])
	}
	if _, exists := imported["fields_json"]; exists {
		t.Fatalf("legacy fields_json should not be stored: %#v", imported)
	}
	if imported["is_built_in"] != true {
		t.Fatalf("strict builtin import should mark template built-in: %#v", imported["is_built_in"])
	}

	result, err = service.ImportCustomTemplatesBulk(payload, false, true)
	if err != nil {
		t.Fatalf("second ImportCustomTemplatesBulk failed: %v", err)
	}
	if !strings.Contains(result, "跳过 1 个") {
		t.Fatalf("expected duplicate skip, got %q", result)
	}
}

func TestNewServiceMigratesLegacySQLiteCustomTemplates(t *testing.T) {
	dataDir := t.TempDir()
	createLegacyTemplateSQLite(t, filepath.Join(dataDir, legacySQLiteFileName), []legacyTemplateSeed{
		{
			ID:          "legacy-custom",
			Name:        "Legacy Custom",
			Description: "from old sqlite",
			FieldsJSON:  `["Front","Back","Extra"]`,
			RulesJSON:   `{"Front":{"field_type":"Text","is_required":true,"description":"front"}}`,
			IsActive:    true,
			IsBuiltIn:   false,
		},
		{
			ID:          "legacy-built-in",
			Name:        "Legacy Builtin",
			Description: "old builtin corpus entry",
			FieldsJSON:  `["Front","Back"]`,
			RulesJSON:   `{}`,
			IsActive:    true,
			IsBuiltIn:   true,
		},
	}, "legacy-custom")

	service, err := NewService(dataDir)
	if err != nil {
		t.Fatalf("NewService failed: %v", err)
	}
	templates, err := service.GetAllCustomTemplates()
	if err != nil {
		t.Fatalf("GetAllCustomTemplates failed: %v", err)
	}
	imported := requireTemplate(t, templates, "legacy-custom")
	if imported["description"] != "from old sqlite" {
		t.Fatalf("legacy description was not preserved: %#v", imported)
	}
	if imported["is_built_in"] != false {
		t.Fatalf("legacy custom should stay custom: %#v", imported["is_built_in"])
	}
	fields, ok := imported["fields"].([]any)
	if !ok || len(fields) != 3 || fields[2] != "Extra" {
		t.Fatalf("legacy fields_json was not normalized: %#v", imported["fields"])
	}
	if _, exists := imported["fields_json"]; exists {
		t.Fatalf("legacy fields_json should not be persisted: %#v", imported)
	}
	for _, template := range templates {
		if stringValue(template["id"]) == "legacy-built-in" {
			t.Fatalf("old builtin corpus entry should not be migrated into lean Go store: %#v", template)
		}
	}
	defaultID, err := service.GetDefaultTemplateID()
	if err != nil {
		t.Fatalf("GetDefaultTemplateID failed: %v", err)
	}
	if defaultID == nil || *defaultID != "legacy-custom" {
		t.Fatalf("expected legacy default to migrate, got %#v", defaultID)
	}
	if service.state.LegacyMigration == nil {
		t.Fatal("expected legacy migration metadata to be persisted")
	}
	if service.state.LegacyMigration.Imported != 1 || service.state.LegacyMigration.Skipped != 1 || service.state.LegacyMigration.Failed != 0 {
		t.Fatalf("unexpected migration counts: %#v", service.state.LegacyMigration)
	}

	reopened, err := NewService(dataDir)
	if err != nil {
		t.Fatalf("reopen failed: %v", err)
	}
	reopenedTemplates, err := reopened.GetAllCustomTemplates()
	if err != nil {
		t.Fatalf("GetAllCustomTemplates after reopen failed: %v", err)
	}
	legacyCount := 0
	for _, template := range reopenedTemplates {
		if stringValue(template["id"]) == "legacy-custom" {
			legacyCount++
		}
	}
	if legacyCount != 1 {
		t.Fatalf("legacy migration should be idempotent, found %d copies", legacyCount)
	}
}

func TestLegacySQLiteMigrationPreservesExistingGoTemplates(t *testing.T) {
	dataDir := t.TempDir()
	existingDefault := "go-existing"
	existingStore := store{
		Templates: []Template{
			{
				"id":                     "go-existing",
				"name":                   "Existing",
				"description":            "keep me",
				"fields":                 []string{"Front", "Back"},
				"field_extraction_rules": map[string]any{},
				"front_template":         "{{Front}}",
				"back_template":          "{{Back}}",
				"css_style":              ".card {}",
				"is_active":              true,
				"is_built_in":            false,
				"created_at":             nowISO(),
				"updated_at":             nowISO(),
			},
		},
		DefaultTemplateID: &existingDefault,
	}
	if err := os.MkdirAll(dataDir, 0o700); err != nil {
		t.Fatalf("mkdir failed: %v", err)
	}
	writeStoreBytes, err := json.Marshal(existingStore)
	if err != nil {
		t.Fatalf("marshal existing store failed: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dataDir, "templates-go.json"), writeStoreBytes, 0o600); err != nil {
		t.Fatalf("write existing store failed: %v", err)
	}
	createLegacyTemplateSQLite(t, filepath.Join(dataDir, legacySQLiteFileName), []legacyTemplateSeed{
		{
			ID:          "go-existing",
			Name:        "Existing",
			Description: "old sqlite should not overwrite",
			FieldsJSON:  `["Front","Back"]`,
			RulesJSON:   `{}`,
			IsActive:    true,
			IsBuiltIn:   false,
		},
		{
			ID:          "legacy-new",
			Name:        "Legacy New",
			Description: "missing template should import",
			FieldsJSON:  `["Question","Answer"]`,
			RulesJSON:   `{}`,
			IsActive:    true,
			IsBuiltIn:   false,
		},
	}, "legacy-new")

	service, err := NewService(dataDir)
	if err != nil {
		t.Fatalf("NewService failed: %v", err)
	}
	templates, err := service.GetAllCustomTemplates()
	if err != nil {
		t.Fatalf("GetAllCustomTemplates failed: %v", err)
	}
	existing := requireTemplate(t, templates, "go-existing")
	if existing["description"] != "keep me" {
		t.Fatalf("existing Go template was overwritten: %#v", existing)
	}
	requireTemplate(t, templates, "legacy-new")
	defaultID, err := service.GetDefaultTemplateID()
	if err != nil {
		t.Fatalf("GetDefaultTemplateID failed: %v", err)
	}
	if defaultID == nil || *defaultID != "go-existing" {
		t.Fatalf("existing Go default should win over legacy default, got %#v", defaultID)
	}
	if service.state.LegacyMigration.Imported != 1 || service.state.LegacyMigration.Skipped != 1 {
		t.Fatalf("unexpected preserve-existing migration counts: %#v", service.state.LegacyMigration)
	}
}

func TestLegacySQLiteMigrationHandlesMissingColumnsAndBadJSON(t *testing.T) {
	dataDir := t.TempDir()
	dbPath := filepath.Join(dataDir, legacySQLiteFileName)
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		t.Fatalf("open legacy sqlite failed: %v", err)
	}
	defer db.Close()
	_, err = db.Exec(`
		CREATE TABLE custom_anki_templates (
			id TEXT PRIMARY KEY,
			name TEXT NOT NULL UNIQUE,
			description TEXT,
			preview_front TEXT NOT NULL,
			preview_back TEXT NOT NULL,
			note_type TEXT NOT NULL DEFAULT 'Basic',
			fields_json TEXT NOT NULL DEFAULT '[]',
			generation_prompt TEXT NOT NULL,
			front_template TEXT NOT NULL,
			back_template TEXT NOT NULL,
			css_style TEXT NOT NULL,
			field_extraction_rules_json TEXT NOT NULL DEFAULT '{}',
			created_at TEXT NOT NULL,
			updated_at TEXT NOT NULL,
			is_active INTEGER NOT NULL DEFAULT 1,
			is_built_in INTEGER NOT NULL DEFAULT 0
		);
		INSERT INTO custom_anki_templates
		(id, name, description, preview_front, preview_back, note_type, fields_json,
		 generation_prompt, front_template, back_template, css_style, field_extraction_rules_json,
		 created_at, updated_at, is_active, is_built_in)
		VALUES
		('legacy-bad-json', 'Legacy Bad JSON', 'old partial schema', '{{Front}}', '{{Back}}', 'Basic',
		 'not-json', 'legacy prompt', '<div>{{Front}}</div>', '<div>{{Back}}</div>', '.card {}', '{bad',
		 'not-a-date', 'not-a-date', 1, 0);
	`)
	if err != nil {
		t.Fatalf("create partial legacy schema failed: %v", err)
	}

	service, err := NewService(dataDir)
	if err != nil {
		t.Fatalf("NewService failed: %v", err)
	}
	templates, err := service.GetAllCustomTemplates()
	if err != nil {
		t.Fatalf("GetAllCustomTemplates failed: %v", err)
	}
	imported := requireTemplate(t, templates, "legacy-bad-json")
	if imported["author"] != "Deep Student" {
		t.Fatalf("missing author column should fall back to Deep Student: %#v", imported["author"])
	}
	if imported["version"] != builtinTemplateVersion {
		t.Fatalf("missing version column should fall back to builtinTemplateVersion: %#v", imported["version"])
	}
	fields, ok := imported["fields"].([]any)
	if !ok || len(fields) != 2 || fields[0] != "Front" || fields[1] != "Back" {
		t.Fatalf("bad fields_json should fall back to Front/Back: %#v", imported["fields"])
	}
	rules, ok := imported["field_extraction_rules"].(map[string]any)
	if !ok || rules["Front"] == nil || rules["Back"] == nil {
		t.Fatalf("bad rules JSON should fall back to generated rules: %#v", imported["field_extraction_rules"])
	}
	if _, exists := imported["preview_data_json"]; exists {
		t.Fatalf("missing preview_data_json column should not create stale preview data: %#v", imported)
	}
	if service.state.LegacyMigration == nil || service.state.LegacyMigration.Imported != 1 || service.state.LegacyMigration.Failed != 0 {
		t.Fatalf("unexpected migration metadata for partial old db: %#v", service.state.LegacyMigration)
	}
}

func newTestService(t *testing.T) *Service {
	t.Helper()
	service, err := NewService(t.TempDir())
	if err != nil {
		t.Fatalf("NewService failed: %v", err)
	}
	return service
}

func requireTemplate(t *testing.T, templates []Template, id string) Template {
	t.Helper()
	for _, template := range templates {
		if stringValue(template["id"]) == id {
			return template
		}
	}
	t.Fatalf("template %s not found in %#v", id, templates)
	return nil
}

type legacyTemplateSeed struct {
	ID          string
	Name        string
	Description string
	FieldsJSON  string
	RulesJSON   string
	IsActive    bool
	IsBuiltIn   bool
}

func createLegacyTemplateSQLite(t *testing.T, dbPath string, templates []legacyTemplateSeed, defaultTemplateID string) {
	t.Helper()
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		t.Fatalf("open legacy sqlite failed: %v", err)
	}
	defer db.Close()
	_, err = db.Exec(`
		CREATE TABLE custom_anki_templates (
			id TEXT PRIMARY KEY,
			name TEXT NOT NULL UNIQUE,
			description TEXT,
			author TEXT,
			version TEXT NOT NULL DEFAULT '1.0.0',
			preview_front TEXT NOT NULL,
			preview_back TEXT NOT NULL,
			note_type TEXT NOT NULL DEFAULT 'Basic',
			fields_json TEXT NOT NULL DEFAULT '[]',
			generation_prompt TEXT NOT NULL,
			front_template TEXT NOT NULL,
			back_template TEXT NOT NULL,
			css_style TEXT NOT NULL,
			field_extraction_rules_json TEXT NOT NULL DEFAULT '{}',
			created_at TEXT NOT NULL,
			updated_at TEXT NOT NULL,
			is_active INTEGER NOT NULL DEFAULT 1,
			is_built_in INTEGER NOT NULL DEFAULT 0,
			preview_data_json TEXT
		);
		CREATE TABLE settings (
			key TEXT PRIMARY KEY,
			value TEXT,
			updated_at TEXT
		);
	`)
	if err != nil {
		t.Fatalf("create legacy schema failed: %v", err)
	}
	for _, template := range templates {
		_, err = db.Exec(`
			INSERT INTO custom_anki_templates
			(id, name, description, author, version, preview_front, preview_back, note_type,
			 fields_json, generation_prompt, front_template, back_template, css_style,
			 field_extraction_rules_json, created_at, updated_at, is_active, is_built_in, preview_data_json)
			VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		`,
			template.ID,
			template.Name,
			template.Description,
			"Legacy User",
			"2.3.4",
			"{{Front}}",
			"{{Back}}",
			"Basic",
			template.FieldsJSON,
			"legacy prompt",
			"<div>{{Front}}</div>",
			"<div>{{Back}}</div>",
			".card { color: blue; }",
			template.RulesJSON,
			"2024-01-02T03:04:05.000Z",
			"2024-01-03T03:04:05.000Z",
			boolInt(template.IsActive),
			boolInt(template.IsBuiltIn),
			`{"Front":"preview"}`,
		)
		if err != nil {
			t.Fatalf("insert legacy template %s failed: %v", template.ID, err)
		}
	}
	if defaultTemplateID != "" {
		_, err = db.Exec(
			"INSERT INTO settings (key, value, updated_at) VALUES ('default_template_id', ?, '2024-01-04T03:04:05.000Z')",
			defaultTemplateID,
		)
		if err != nil {
			t.Fatalf("insert legacy default failed: %v", err)
		}
	}
}

func boolInt(value bool) int {
	if value {
		return 1
	}
	return 0
}
