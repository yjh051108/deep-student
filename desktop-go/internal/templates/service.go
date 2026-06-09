package templates

import (
	"crypto/rand"
	"deep-student-go/internal/storage"
	"encoding/json"
	"errors"
	"fmt"
	"math/big"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"
)

const builtinTemplateVersion = "1.0.0"

var (
	scriptBlockRE  = regexp.MustCompile(`(?is)<\s*script[^>]*>.*?<\s*/\s*script\s*>`)
	eventHandlerRE = regexp.MustCompile(`(?is)\s+on[a-z]+\s*=\s*(".*?"|'.*?'|[^\s>]+)`)
	jsURLRE        = regexp.MustCompile(`(?is)javascript\s*:`)
	cssRiskRE      = regexp.MustCompile(`(?is)(expression\s*\(|javascript\s*:|@import\s+)`)
)

type Template map[string]any

type TemplateExportResponse struct {
	TemplateData string `json:"template_data"`
}

type Service struct {
	mu      sync.RWMutex
	dataDir string
	path    string
	state   store
}

type store struct {
	Templates         []Template `json:"templates"`
	DefaultTemplateID *string    `json:"defaultTemplateId,omitempty"`
	LegacyMigration   *Migration `json:"legacyMigration,omitempty"`
}

type Migration struct {
	SourcePath    string  `json:"sourcePath"`
	SourceSize    int64   `json:"sourceSize"`
	SourceModTime string  `json:"sourceModTime"`
	MigratedAt    string  `json:"migratedAt"`
	Imported      int     `json:"imported"`
	Skipped       int     `json:"skipped"`
	Failed        int     `json:"failed"`
	DefaultID     *string `json:"defaultTemplateId,omitempty"`
	LastError     string  `json:"lastError,omitempty"`
}

func NewService(dataDir string) (*Service, error) {
	return newService(dataDir, nil)
}

func NewServiceWithLegacyRoots(dataDir string, legacyRoots []string) (*Service, error) {
	return newService(dataDir, legacyRoots)
}

func newService(dataDir string, legacyRoots []string) (*Service, error) {
	cleanedDataDir := filepath.Clean(dataDir)
	service := &Service{
		dataDir: cleanedDataDir,
		path:    filepath.Join(cleanedDataDir, "templates-go.json"),
		state: store{
			Templates: []Template{},
		},
	}
	if err := service.load(); err != nil {
		return nil, err
	}
	service.migrateLegacySQLiteIfPresent(legacyRoots)
	return service, nil
}

func (s *Service) ImportBuiltinTemplates() (string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	now := nowISO()
	added := 0
	updated := 0
	skipped := 0
	failed := 0

	for _, incoming := range builtinTemplates(now) {
		id := stringValue(incoming["id"])
		if id == "" {
			failed++
			continue
		}
		idx := s.findIndexByID(id)
		if idx < 0 {
			s.state.Templates = append(s.state.Templates, incoming)
			added++
			continue
		}
		current := s.state.Templates[idx]
		if !boolValue(current["is_built_in"], false) {
			skipped++
			continue
		}
		if stringValue(current["version"]) == stringValue(incoming["version"]) {
			skipped++
			continue
		}
		if createdAt := stringValue(current["created_at"]); createdAt != "" {
			incoming["created_at"] = createdAt
		}
		incoming["updated_at"] = now
		s.state.Templates[idx] = incoming
		updated++
	}

	if s.state.DefaultTemplateID == nil || s.findIndexByID(*s.state.DefaultTemplateID) < 0 {
		if defaultID := s.firstActiveBuiltinID(); defaultID != "" {
			s.state.DefaultTemplateID = &defaultID
		}
	}

	if err := s.flushLocked(); err != nil {
		return "", err
	}
	return fmt.Sprintf("导入完成：新增 %d 个，更新 %d 个，跳过 %d 个，%d 个失败", added, updated, skipped, failed), nil
}

func (s *Service) GetAllCustomTemplates() ([]Template, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return cloneTemplates(s.state.Templates), nil
}

func (s *Service) GetDefaultTemplateID() (*string, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if s.state.DefaultTemplateID == nil {
		return nil, nil
	}
	value := *s.state.DefaultTemplateID
	return &value, nil
}

func (s *Service) CreateCustomTemplate(request map[string]any) (string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	now := nowISO()
	template := normalizeTemplate(Template(request), now, false)
	id := stringValue(template["id"])
	if id == "" {
		id = newID("tpl")
		template["id"] = id
	}
	if s.findIndexByID(id) >= 0 {
		id = newID("tpl")
		template["id"] = id
	}
	template["is_built_in"] = false
	template["created_at"] = now
	template["updated_at"] = now

	s.state.Templates = append(s.state.Templates, template)
	return id, s.flushLocked()
}

func (s *Service) UpdateCustomTemplate(templateID string, request map[string]any) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	idx := s.findIndexByID(templateID)
	if idx < 0 {
		return notFound(templateID)
	}

	current := cloneTemplate(s.state.Templates[idx])
	if expected := firstString(request, "expected_version", "expectedVersion"); expected != "" && expected != stringValue(current["version"]) {
		return fmt.Errorf("template version conflict: expected %s, current %s", expected, stringValue(current["version"]))
	}

	for key, value := range request {
		if key == "id" || key == "created_at" || key == "createdAt" || key == "updated_at" || key == "updatedAt" || key == "expected_version" || key == "expectedVersion" {
			continue
		}
		current[key] = value
	}

	now := nowISO()
	current = normalizeTemplate(current, now, boolValue(current["is_built_in"], false))
	current["id"] = templateID
	if _, ok := request["version"]; !ok {
		current["version"] = bumpPatchVersion(stringValue(s.state.Templates[idx]["version"]))
	}
	if createdAt := stringValue(s.state.Templates[idx]["created_at"]); createdAt != "" {
		current["created_at"] = createdAt
	}
	current["updated_at"] = now
	s.state.Templates[idx] = current
	return s.flushLocked()
}

func (s *Service) DeleteCustomTemplate(templateID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	idx := s.findIndexByID(templateID)
	if idx < 0 {
		return notFound(templateID)
	}
	if boolValue(s.state.Templates[idx]["is_built_in"], false) {
		return fmt.Errorf("cannot delete built-in template: %s", templateID)
	}
	s.state.Templates = append(s.state.Templates[:idx], s.state.Templates[idx+1:]...)
	if s.state.DefaultTemplateID != nil && *s.state.DefaultTemplateID == templateID {
		s.state.DefaultTemplateID = nil
		if defaultID := s.firstActiveBuiltinID(); defaultID != "" {
			s.state.DefaultTemplateID = &defaultID
		}
	}
	return s.flushLocked()
}

func (s *Service) SetDefaultTemplate(templateID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	templateID = strings.TrimSpace(templateID)
	if templateID == "" {
		s.state.DefaultTemplateID = nil
		return s.flushLocked()
	}
	idx := s.findIndexByID(templateID)
	if idx < 0 {
		return notFound(templateID)
	}
	if !boolValue(s.state.Templates[idx]["is_active"], true) {
		return fmt.Errorf("cannot set inactive template as default: %s", templateID)
	}
	s.state.DefaultTemplateID = &templateID
	return s.flushLocked()
}

func (s *Service) ImportCustomTemplatesBulk(templateData string, overwriteExisting bool, strictBuiltin bool) (string, error) {
	items, err := decodeTemplateData(templateData)
	if err != nil {
		return "", err
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	now := nowISO()
	added := 0
	updated := 0
	skipped := 0
	failed := 0

	for _, raw := range items {
		template := normalizeTemplate(raw, now, strictBuiltin)
		template["is_built_in"] = strictBuiltin
		id := stringValue(template["id"])
		if id == "" {
			id = newID("tpl")
			template["id"] = id
		}
		idx := s.findIndexByID(id)
		if idx < 0 {
			if nameIdx := s.findIndexByName(stringValue(template["name"])); nameIdx >= 0 {
				idx = nameIdx
			}
		}
		if idx >= 0 {
			if !overwriteExisting {
				skipped++
				continue
			}
			if createdAt := stringValue(s.state.Templates[idx]["created_at"]); createdAt != "" {
				template["created_at"] = createdAt
			}
			template["updated_at"] = now
			s.state.Templates[idx] = template
			updated++
			continue
		}
		template["created_at"] = now
		template["updated_at"] = now
		s.state.Templates = append(s.state.Templates, template)
		added++
	}

	if err := s.flushLocked(); err != nil {
		return "", err
	}
	return fmt.Sprintf("导入完成：新增 %d 个，更新 %d 个，跳过 %d 个，%d 个失败", added, updated, skipped, failed), nil
}

func (s *Service) ExportTemplate(templateID string) (TemplateExportResponse, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	idx := s.findIndexByID(templateID)
	if idx < 0 {
		return TemplateExportResponse{}, notFound(templateID)
	}
	bytes, err := json.MarshalIndent(s.state.Templates[idx], "", "  ")
	if err != nil {
		return TemplateExportResponse{}, err
	}
	return TemplateExportResponse{TemplateData: string(bytes)}, nil
}

func (s *Service) load() error {
	bytes, err := os.ReadFile(s.path)
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	if err != nil {
		return err
	}
	if len(bytes) == 0 {
		return nil
	}
	if err := json.Unmarshal(bytes, &s.state); err != nil {
		return err
	}
	if s.state.Templates == nil {
		s.state.Templates = []Template{}
	}
	now := nowISO()
	for i := range s.state.Templates {
		s.state.Templates[i] = normalizeTemplate(s.state.Templates[i], now, boolValue(s.state.Templates[i]["is_built_in"], false))
	}
	return nil
}

func (s *Service) flushLocked() error {
	return storage.WriteJSONAtomic(s.path, s.state)
}

func (s *Service) findIndexByID(id string) int {
	for i, template := range s.state.Templates {
		if stringValue(template["id"]) == id {
			return i
		}
	}
	return -1
}

func (s *Service) findIndexByName(name string) int {
	if strings.TrimSpace(name) == "" {
		return -1
	}
	for i, template := range s.state.Templates {
		if stringValue(template["name"]) == name {
			return i
		}
	}
	return -1
}

func (s *Service) firstActiveBuiltinID() string {
	for _, template := range s.state.Templates {
		if boolValue(template["is_built_in"], false) && boolValue(template["is_active"], true) {
			return stringValue(template["id"])
		}
	}
	return ""
}

func decodeTemplateData(templateData string) ([]Template, error) {
	trimmed := strings.TrimSpace(templateData)
	if trimmed == "" {
		return nil, errors.New("template data cannot be empty")
	}

	var raw any
	if err := json.Unmarshal([]byte(trimmed), &raw); err != nil {
		return nil, err
	}

	if wrapper, ok := raw.(map[string]any); ok {
		if nested, ok := wrapper["template_data"].(string); ok && strings.TrimSpace(nested) != "" {
			return decodeTemplateData(nested)
		}
		if nested, ok := wrapper["templateData"].(string); ok && strings.TrimSpace(nested) != "" {
			return decodeTemplateData(nested)
		}
		return []Template{Template(wrapper)}, nil
	}

	if list, ok := raw.([]any); ok {
		templates := make([]Template, 0, len(list))
		for _, item := range list {
			object, ok := item.(map[string]any)
			if !ok {
				continue
			}
			templates = append(templates, Template(object))
		}
		if len(templates) == 0 {
			return nil, errors.New("template array does not contain template objects")
		}
		return templates, nil
	}

	return nil, errors.New("template data must be an object or array")
}

func normalizeTemplate(raw Template, now string, defaultBuiltin bool) Template {
	out := cloneTemplate(raw)
	if id := firstString(out, "id"); id != "" {
		out["id"] = id
	}
	if name := firstString(out, "name"); name != "" {
		out["name"] = name
	} else {
		out["name"] = "Untitled Template"
	}
	if description := firstString(out, "description"); description != "" {
		out["description"] = description
	} else {
		out["description"] = ""
	}
	if author := firstString(out, "author"); author != "" {
		out["author"] = author
	} else {
		out["author"] = "Deep Student"
	}
	if version := firstString(out, "version"); version != "" {
		out["version"] = version
	} else {
		out["version"] = builtinTemplateVersion
	}
	if noteType := firstString(out, "note_type", "noteType"); noteType != "" {
		out["note_type"] = noteType
	} else {
		out["note_type"] = "Basic"
	}

	out["fields"] = normalizeFields(out)
	out["field_extraction_rules"] = normalizeRuleMap(out)
	out["preview_front"] = sanitizeHTMLCompat(stringOrDefault(firstString(out, "preview_front", "previewFront"), "{{Front}}"))
	out["preview_back"] = sanitizeHTMLCompat(stringOrDefault(firstString(out, "preview_back", "previewBack"), "{{Back}}"))
	out["generation_prompt"] = stringOrDefault(firstString(out, "generation_prompt", "generationPrompt"), "Generate concise Anki cards as JSON using the template fields.")
	out["front_template"] = sanitizeHTMLCompat(stringOrDefault(firstString(out, "front_template", "frontTemplate"), "<div>{{Front}}</div>"))
	out["back_template"] = sanitizeHTMLCompat(stringOrDefault(firstString(out, "back_template", "backTemplate"), "<div>{{Front}}</div><hr><div>{{Back}}</div>"))
	out["css_style"] = sanitizeCSSCompat(stringOrDefault(firstString(out, "css_style", "cssStyle"), ".card { font-family: Arial, sans-serif; font-size: 20px; text-align: center; }"))
	out["is_active"] = boolValue(out["is_active"], boolValue(out["isActive"], true))
	out["is_built_in"] = boolValue(out["is_built_in"], boolValue(out["isBuiltIn"], defaultBuiltin))
	out["created_at"] = stringOrDefault(firstString(out, "created_at", "createdAt"), now)
	out["updated_at"] = stringOrDefault(firstString(out, "updated_at", "updatedAt"), now)

	delete(out, "fields_json")
	delete(out, "fieldsJson")
	delete(out, "field_extraction_rules_json")
	delete(out, "fieldExtractionRulesJson")
	delete(out, "noteType")
	delete(out, "previewFront")
	delete(out, "previewBack")
	delete(out, "generationPrompt")
	delete(out, "frontTemplate")
	delete(out, "backTemplate")
	delete(out, "cssStyle")
	delete(out, "isActive")
	delete(out, "isBuiltIn")
	delete(out, "createdAt")
	delete(out, "updatedAt")
	return out
}

func normalizeFields(template Template) []string {
	if fields, ok := parseStringSlice(template["fields"]); ok && len(fields) > 0 {
		return fields
	}
	for _, key := range []string{"fields_json", "fieldsJson"} {
		if text := stringValue(template[key]); text != "" {
			var values []string
			if err := json.Unmarshal([]byte(text), &values); err == nil && len(values) > 0 {
				return values
			}
			var anyValues []any
			if err := json.Unmarshal([]byte(text), &anyValues); err == nil {
				if fields, ok := parseStringSlice(anyValues); ok && len(fields) > 0 {
					return fields
				}
			}
		}
	}
	return []string{"Front", "Back"}
}

func normalizeRuleMap(template Template) map[string]any {
	for _, key := range []string{"field_extraction_rules", "fieldExtractionRules"} {
		if rules, ok := template[key].(map[string]any); ok {
			return rules
		}
	}
	for _, key := range []string{"field_extraction_rules_json", "fieldExtractionRulesJson"} {
		if text := stringValue(template[key]); text != "" {
			var rules map[string]any
			if err := json.Unmarshal([]byte(text), &rules); err == nil {
				return rules
			}
		}
	}
	fields := normalizeFields(template)
	rules := make(map[string]any, len(fields))
	for _, field := range fields {
		lower := strings.ToLower(field)
		rules[field] = map[string]any{
			"field_type":    "Text",
			"is_required":   lower == "front" || lower == "back",
			"default_value": "",
			"description":   field + " field content",
		}
	}
	return rules
}

func parseStringSlice(value any) ([]string, bool) {
	switch typed := value.(type) {
	case []string:
		return typed, true
	case []any:
		out := make([]string, 0, len(typed))
		for _, item := range typed {
			if text := strings.TrimSpace(stringValue(item)); text != "" {
				out = append(out, text)
			}
		}
		return out, true
	default:
		return nil, false
	}
}

func cloneTemplates(values []Template) []Template {
	out := make([]Template, 0, len(values))
	for _, value := range values {
		out = append(out, cloneTemplate(value))
	}
	return out
}

func cloneTemplate(value Template) Template {
	if value == nil {
		return Template{}
	}
	bytes, err := json.Marshal(value)
	if err != nil {
		out := make(Template, len(value))
		for key, item := range value {
			out[key] = item
		}
		return out
	}
	var cloned Template
	if err := json.Unmarshal(bytes, &cloned); err != nil {
		out := make(Template, len(value))
		for key, item := range value {
			out[key] = item
		}
		return out
	}
	return cloned
}

func firstString(values map[string]any, keys ...string) string {
	for _, key := range keys {
		if text := strings.TrimSpace(stringValue(values[key])); text != "" {
			return text
		}
	}
	return ""
}

func stringValue(value any) string {
	switch typed := value.(type) {
	case string:
		return typed
	case fmt.Stringer:
		return typed.String()
	default:
		return ""
	}
}

func stringOrDefault(value string, fallback string) string {
	if strings.TrimSpace(value) == "" {
		return fallback
	}
	return value
}

func boolValue(value any, fallback bool) bool {
	switch typed := value.(type) {
	case bool:
		return typed
	case string:
		parsed, err := strconv.ParseBool(strings.TrimSpace(typed))
		if err == nil {
			return parsed
		}
		return fallback
	default:
		return fallback
	}
}

func bumpPatchVersion(version string) string {
	parts := strings.Split(strings.TrimSpace(version), ".")
	if len(parts) == 0 || parts[0] == "" {
		return builtinTemplateVersion
	}
	for len(parts) < 3 {
		parts = append(parts, "0")
	}
	patch, err := strconv.Atoi(parts[2])
	if err != nil {
		return version + ".1"
	}
	parts[2] = strconv.Itoa(patch + 1)
	return strings.Join(parts[:3], ".")
}

func builtinTemplates(now string) []Template {
	return []Template{
		builtinTemplate("builtin-minimal", "极简卡片", "适合概念、定义和短问答。", []string{"Front", "Back"}, "{{Front}}", "{{Back}}", now),
		builtinTemplate("builtin-code", "编程代码卡片", "适合代码片段、输出和解释。", []string{"Question", "Code", "Answer"}, "{{Question}}\n\n{{Code}}", "{{Answer}}", now),
		builtinTemplate("builtin-cloze", "填空题卡片", "适合记忆关键术语和公式。", []string{"Text", "Answer"}, "{{Text}}", "{{Answer}}", now),
		builtinTemplate("builtin-choice", "选择题卡片", "适合单选题和辨析题。", []string{"Question", "Options", "Answer"}, "{{Question}}\n\n{{Options}}", "{{Answer}}", now),
		builtinTemplate("builtin-language", "语言学习卡片", "适合词汇、例句和释义。", []string{"Word", "Meaning", "Example"}, "{{Word}}\n\n{{Example}}", "{{Meaning}}", now),
		builtinTemplate("builtin-law", "法律条文卡片", "适合法条、要件和适用场景。", []string{"Article", "Rule", "Application"}, "{{Article}}\n\n{{Rule}}", "{{Application}}", now),
	}
}

func builtinTemplate(id string, name string, description string, fields []string, front string, back string, now string) Template {
	rules := make(map[string]any, len(fields))
	for _, field := range fields {
		lower := strings.ToLower(field)
		rules[field] = map[string]any{
			"field_type":    "Text",
			"is_required":   lower == "front" || lower == "back" || lower == "question" || lower == "answer",
			"default_value": "",
			"description":   field + " field content",
		}
	}
	return Template{
		"id":                     id,
		"name":                   name,
		"description":            description,
		"author":                 "Deep Student",
		"version":                builtinTemplateVersion,
		"preview_front":          front,
		"preview_back":           back,
		"note_type":              "Basic",
		"fields":                 fields,
		"generation_prompt":      "Generate concise Anki card JSON using these fields: " + strings.Join(fields, ", ") + ".",
		"front_template":         "<div class=\"ds-card-front\">" + front + "</div>",
		"back_template":          "<div class=\"ds-card-front\">" + front + "</div><hr><div class=\"ds-card-back\">" + back + "</div>",
		"css_style":              ".card { font-family: Arial, sans-serif; font-size: 20px; text-align: left; line-height: 1.5; }",
		"field_extraction_rules": rules,
		"preview_data_json":      previewDataJSON(fields),
		"created_at":             now,
		"updated_at":             now,
		"is_active":              true,
		"is_built_in":            true,
	}
}

func previewDataJSON(fields []string) string {
	data := make(map[string]string, len(fields))
	for _, field := range fields {
		data[field] = field + " preview"
	}
	bytes, err := json.Marshal(data)
	if err != nil {
		return "{}"
	}
	return string(bytes)
}

func sanitizeHTMLCompat(value string) string {
	value = scriptBlockRE.ReplaceAllString(value, "")
	value = eventHandlerRE.ReplaceAllString(value, "")
	value = jsURLRE.ReplaceAllString(value, "")
	return value
}

func sanitizeCSSCompat(value string) string {
	return cssRiskRE.ReplaceAllString(value, "")
}

func nowISO() string {
	return time.Now().UTC().Format("2006-01-02T15:04:05.000Z")
}

func newID(prefix string) string {
	return fmt.Sprintf("%s_%s", prefix, randomToken(10))
}

func randomToken(length int) string {
	const alphabet = "0123456789abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ_-"
	out := make([]byte, length)
	max := big.NewInt(int64(len(alphabet)))
	for i := range out {
		n, err := rand.Int(rand.Reader, max)
		if err != nil {
			out[i] = alphabet[int(time.Now().UnixNano())%len(alphabet)]
			continue
		}
		out[i] = alphabet[n.Int64()]
	}
	return string(out)
}

func notFound(templateID string) error {
	return fmt.Errorf("template not found: %s", templateID)
}
