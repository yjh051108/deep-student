package templates

import (
	"database/sql"
	"errors"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	_ "modernc.org/sqlite"
)

const legacySQLiteFileName = "mistakes.db"

var legacyTemplateColumns = []string{
	"id",
	"name",
	"description",
	"author",
	"version",
	"preview_front",
	"preview_back",
	"note_type",
	"fields_json",
	"generation_prompt",
	"front_template",
	"back_template",
	"css_style",
	"field_extraction_rules_json",
	"created_at",
	"updated_at",
	"is_active",
	"is_built_in",
	"preview_data_json",
}

type legacyReadResult struct {
	templates       []Template
	defaultTemplate *string
	skippedBuiltins int
}

func (s *Service) migrateLegacySQLiteIfPresent(legacyRoots []string) {
	for _, dbPath := range legacySQLiteCandidatePaths(s.dataDir, legacyRoots) {
		if _, err := os.Stat(dbPath); errors.Is(err, os.ErrNotExist) {
			continue
		}
		result, err := s.MigrateLegacySQLite(dbPath, false)
		if err != nil {
			s.recordLegacyMigrationError(dbPath, err)
			continue
		}
		if result.MigratedAt != "" {
			return
		}
	}
}

func (s *Service) MigrateLegacySQLite(dbPath string, includeBuiltins bool) (Migration, error) {
	info, err := os.Stat(dbPath)
	if errors.Is(err, os.ErrNotExist) {
		return Migration{}, nil
	}
	if err != nil {
		return Migration{}, err
	}

	sourcePath, err := filepath.Abs(dbPath)
	if err != nil {
		sourcePath = filepath.Clean(dbPath)
	}
	source := migrationForSource(sourcePath, info)

	s.mu.RLock()
	if s.state.LegacyMigration != nil && sameMigrationSource(*s.state.LegacyMigration, source) && s.state.LegacyMigration.LastError == "" {
		current := *s.state.LegacyMigration
		s.mu.RUnlock()
		return current, nil
	}
	s.mu.RUnlock()

	legacy, err := readLegacySQLiteTemplates(sourcePath, includeBuiltins)
	if err != nil {
		return source, err
	}
	if len(legacy.templates) == 0 && legacy.defaultTemplate == nil && legacy.skippedBuiltins == 0 {
		return source, nil
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	now := nowISO()
	result := source
	result.MigratedAt = now
	result.Skipped = legacy.skippedBuiltins

	for _, raw := range legacy.templates {
		template := normalizeTemplate(raw, now, boolValue(raw["is_built_in"], false))
		id := stringValue(template["id"])
		if id == "" {
			result.Failed++
			continue
		}
		idx := s.findIndexByID(id)
		if idx < 0 {
			idx = s.findIndexByName(stringValue(template["name"]))
		}
		if idx >= 0 {
			result.Skipped++
			continue
		}
		s.state.Templates = append(s.state.Templates, template)
		result.Imported++
	}

	if legacy.defaultTemplate != nil && (s.state.DefaultTemplateID == nil || s.findIndexByID(*s.state.DefaultTemplateID) < 0) {
		if idx := s.findIndexByID(*legacy.defaultTemplate); idx >= 0 && boolValue(s.state.Templates[idx]["is_active"], true) {
			defaultID := *legacy.defaultTemplate
			s.state.DefaultTemplateID = &defaultID
			result.DefaultID = &defaultID
		}
	}

	s.state.LegacyMigration = &result
	if err := s.flushLocked(); err != nil {
		return result, err
	}
	return result, nil
}

func (s *Service) recordLegacyMigrationError(dbPath string, migrationErr error) {
	info, statErr := os.Stat(dbPath)
	if statErr != nil {
		return
	}
	sourcePath, err := filepath.Abs(dbPath)
	if err != nil {
		sourcePath = filepath.Clean(dbPath)
	}
	result := migrationForSource(sourcePath, info)
	result.MigratedAt = nowISO()
	result.LastError = migrationErr.Error()

	s.mu.Lock()
	defer s.mu.Unlock()
	s.state.LegacyMigration = &result
	_ = s.flushLocked()
}

func readLegacySQLiteTemplates(dbPath string, includeBuiltins bool) (legacyReadResult, error) {
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		return legacyReadResult{}, err
	}
	defer db.Close()

	_, _ = db.Exec("PRAGMA query_only = ON")

	exists, err := sqliteTableExists(db, "custom_anki_templates")
	if err != nil || !exists {
		return legacyReadResult{}, err
	}
	columns, err := sqliteColumns(db, "custom_anki_templates")
	if err != nil {
		return legacyReadResult{}, err
	}

	selects := make([]string, 0, len(legacyTemplateColumns))
	for _, column := range legacyTemplateColumns {
		if columns[column] {
			selects = append(selects, "CAST("+quoteSQLiteIdent(column)+" AS TEXT) AS "+quoteSQLiteIdent(column))
			continue
		}
		selects = append(selects, "NULL AS "+quoteSQLiteIdent(column))
	}

	query := "SELECT " + strings.Join(selects, ", ") + " FROM custom_anki_templates ORDER BY created_at DESC"
	rows, err := db.Query(query)
	if err != nil {
		return legacyReadResult{}, err
	}
	defer rows.Close()

	result := legacyReadResult{}
	for rows.Next() {
		values := make([]sql.NullString, len(legacyTemplateColumns))
		dest := make([]any, len(values))
		for i := range values {
			dest[i] = &values[i]
		}
		if err := rows.Scan(dest...); err != nil {
			return legacyReadResult{}, err
		}
		raw := make(Template, len(legacyTemplateColumns))
		for i, column := range legacyTemplateColumns {
			if values[i].Valid {
				raw[column] = values[i].String
			}
		}
		isBuiltIn := sqliteBool(raw["is_built_in"], false)
		if isBuiltIn && !includeBuiltins {
			result.skippedBuiltins++
			continue
		}
		raw["is_active"] = sqliteBool(raw["is_active"], true)
		raw["is_built_in"] = isBuiltIn
		result.templates = append(result.templates, raw)
	}
	if err := rows.Err(); err != nil {
		return legacyReadResult{}, err
	}

	defaultID, err := readLegacyDefaultTemplateID(db)
	if err != nil {
		return legacyReadResult{}, err
	}
	result.defaultTemplate = defaultID
	return result, nil
}

func readLegacyDefaultTemplateID(db *sql.DB) (*string, error) {
	exists, err := sqliteTableExists(db, "settings")
	if err != nil || !exists {
		return nil, err
	}
	var value sql.NullString
	err = db.QueryRow("SELECT CAST(value AS TEXT) FROM settings WHERE key = 'default_template_id'").Scan(&value)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	if !value.Valid || strings.TrimSpace(value.String) == "" {
		return nil, nil
	}
	defaultID := strings.TrimSpace(value.String)
	return &defaultID, nil
}

func sqliteTableExists(db *sql.DB, tableName string) (bool, error) {
	var count int
	err := db.QueryRow("SELECT COUNT(*) FROM sqlite_master WHERE type = 'table' AND name = ?", tableName).Scan(&count)
	return count > 0, err
}

func sqliteColumns(db *sql.DB, tableName string) (map[string]bool, error) {
	rows, err := db.Query("PRAGMA table_info(" + quoteSQLiteIdent(tableName) + ")")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	columns := map[string]bool{}
	for rows.Next() {
		var cid int
		var name string
		var typ string
		var notNull int
		var defaultValue sql.NullString
		var pk int
		if err := rows.Scan(&cid, &name, &typ, &notNull, &defaultValue, &pk); err != nil {
			return nil, err
		}
		columns[name] = true
	}
	return columns, rows.Err()
}

func quoteSQLiteIdent(value string) string {
	return `"` + strings.ReplaceAll(value, `"`, `""`) + `"`
}

func sqliteBool(value any, fallback bool) bool {
	switch typed := value.(type) {
	case bool:
		return typed
	case string:
		switch strings.ToLower(strings.TrimSpace(typed)) {
		case "1", "true", "t", "yes", "y", "on":
			return true
		case "0", "false", "f", "no", "n", "off":
			return false
		default:
			return fallback
		}
	default:
		return fallback
	}
}

func migrationForSource(path string, info os.FileInfo) Migration {
	return Migration{
		SourcePath:    filepath.Clean(path),
		SourceSize:    info.Size(),
		SourceModTime: info.ModTime().UTC().Format(time.RFC3339Nano),
	}
}

func sameMigrationSource(left Migration, right Migration) bool {
	return left.SourcePath == right.SourcePath &&
		left.SourceSize == right.SourceSize &&
		left.SourceModTime == right.SourceModTime
}

func legacySQLiteCandidatePaths(dataDir string, legacyRoots []string) []string {
	paths := []string{}
	seen := map[string]bool{}
	appendPath := func(path string) {
		cleaned := filepath.Clean(path)
		key := strings.ToLower(cleaned)
		if cleaned == "." || seen[key] {
			return
		}
		seen[key] = true
		paths = append(paths, cleaned)
	}
	appendRoot := func(root string, preferSlots bool) {
		root = strings.TrimSpace(root)
		if root == "" {
			return
		}
		if preferSlots {
			appendSlotCandidates(root, appendPath)
			appendPath(filepath.Join(root, legacySQLiteFileName))
			return
		}
		appendPath(filepath.Join(root, legacySQLiteFileName))
		appendSlotCandidates(root, appendPath)
	}

	appendRoot(dataDir, false)
	for _, root := range legacyRoots {
		appendRoot(root, true)
	}
	return paths
}

func appendSlotCandidates(root string, appendPath func(string)) {
	slotsRoot := filepath.Join(root, "slots")
	appendPath(filepath.Join(slotsRoot, "slotA", legacySQLiteFileName))
	appendPath(filepath.Join(slotsRoot, "slotB", legacySQLiteFileName))

	entries, err := os.ReadDir(slotsRoot)
	if err != nil {
		return
	}
	names := make([]string, 0, len(entries))
	for _, entry := range entries {
		if entry.IsDir() {
			names = append(names, entry.Name())
		}
	}
	sort.Strings(names)
	for _, name := range names {
		appendPath(filepath.Join(slotsRoot, name, legacySQLiteFileName))
	}
}
