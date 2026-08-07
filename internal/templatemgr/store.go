// templatemgr 包的 SQLite 持久化层。
//
// custom_anki_templates 表对齐 Rust 原版；时间字段 RFC3339Nano UTC。

package templatemgr

import (
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/helixnow/deep-student-go/pkg/store"
)

// Store 模板存储层。
type Store struct {
	db *sql.DB
}

// NewStore 从 pkg/store.Store 构造。
func NewStore(s *store.Store) *Store {
	if s == nil || s.DB == nil {
		return &Store{}
	}
	return &Store{db: s.DB}
}

// Migrate 创建 custom_anki_templates 表（幂等）。
func (s *Store) Migrate() error {
	if s.db == nil {
		return errors.New("templatemgr: nil db")
	}
	stmts := []string{
		`CREATE TABLE IF NOT EXISTS custom_anki_templates (
			id TEXT PRIMARY KEY,
			name TEXT NOT NULL,
			front_tmpl TEXT NOT NULL,
			back_tmpl TEXT NOT NULL,
			style TEXT NOT NULL DEFAULT '',
			shared_css TEXT NOT NULL DEFAULT '',
			is_builtin INTEGER NOT NULL DEFAULT 0,
			preview TEXT NOT NULL DEFAULT '',
			sort_order INTEGER NOT NULL DEFAULT 0,
			created_at TEXT NOT NULL,
			updated_at TEXT NOT NULL
		)`,
		`CREATE INDEX IF NOT EXISTS idx_templates_builtin ON custom_anki_templates(is_builtin)`,
	}
	for _, stmt := range stmts {
		if _, err := s.db.Exec(stmt); err != nil {
			return fmt.Errorf("templatemgr migrate: %w", err)
		}
	}
	return nil
}

// Create 插入模板。
func (s *Store) Create(t *Template) error {
	if s.db == nil {
		return errors.New("templatemgr: nil db")
	}
	_, err := s.db.Exec(`INSERT INTO custom_anki_templates
		(id, name, front_tmpl, back_tmpl, style, shared_css, is_builtin, preview, sort_order, created_at, updated_at)
		VALUES (?,?,?,?,?,?,?,?,?,?,?)`,
		t.ID, t.Name, t.FrontTmpl, t.BackTmpl, t.Style, t.SharedCSS,
		boolToInt(t.IsBuiltin), t.Preview, t.SortOrder, formatTime(t.CreatedAt), formatTime(t.UpdatedAt),
	)
	if err != nil {
		return fmt.Errorf("templatemgr: create: %w", err)
	}
	return nil
}

// Get 按 ID 读取。
func (s *Store) Get(id string) (*Template, error) {
	if s.db == nil {
		return nil, errors.New("templatemgr: nil db")
	}
	row := s.db.QueryRow(tplSelectCols+` FROM custom_anki_templates WHERE id=?`, id)
	return scanTemplate(row)
}

// List 列出全部模板（内置在前，按 sort_order, name）。
func (s *Store) List() ([]*Template, error) {
	if s.db == nil {
		return nil, errors.New("templatemgr: nil db")
	}
	rows, err := s.db.Query(tplSelectCols + ` FROM custom_anki_templates ORDER BY is_builtin DESC, sort_order ASC, name ASC`)
	if err != nil {
		return nil, fmt.Errorf("templatemgr: list: %w", err)
	}
	defer rows.Close()
	var out []*Template
	for rows.Next() {
		t, err := scanTemplate(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, t)
	}
	return out, rows.Err()
}

// Update 更新模板字段（nil 不修改）。
func (s *Store) Update(p UpdateParams) error {
	if s.db == nil {
		return errors.New("templatemgr: nil db")
	}
	setParts := []string{"updated_at = ?"}
	args := []interface{}{formatTime(time.Now().UTC())}
	if p.Name != nil {
		setParts = append(setParts, "name = ?")
		args = append(args, *p.Name)
	}
	if p.FrontTmpl != nil {
		setParts = append(setParts, "front_tmpl = ?")
		args = append(args, *p.FrontTmpl)
	}
	if p.BackTmpl != nil {
		setParts = append(setParts, "back_tmpl = ?")
		args = append(args, *p.BackTmpl)
	}
	if p.Style != nil {
		setParts = append(setParts, "style = ?")
		args = append(args, *p.Style)
	}
	if p.SharedCSS != nil {
		setParts = append(setParts, "shared_css = ?")
		args = append(args, *p.SharedCSS)
	}
	if p.Preview != nil {
		setParts = append(setParts, "preview = ?")
		args = append(args, *p.Preview)
	}
	args = append(args, p.ID)
	query := `UPDATE custom_anki_templates SET ` + joinSet(setParts) + ` WHERE id=?`
	res, err := s.db.Exec(query, args...)
	if err != nil {
		return fmt.Errorf("templatemgr: update: %w", err)
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return fmt.Errorf("templatemgr: template not found: %s", p.ID)
	}
	return nil
}

// Delete 删除模板（内置模板受保护）。
func (s *Store) Delete(id string) error {
	if s.db == nil {
		return errors.New("templatemgr: nil db")
	}
	res, err := s.db.Exec(`DELETE FROM custom_anki_templates WHERE id=? AND is_builtin=0`, id)
	if err != nil {
		return fmt.Errorf("templatemgr: delete: %w", err)
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return fmt.Errorf("templatemgr: template not found or is builtin: %s", id)
	}
	return nil
}

// Count 统计模板数。
func (s *Store) Count() (int64, error) {
	if s.db == nil {
		return 0, errors.New("templatemgr: nil db")
	}
	var c int64
	err := s.db.QueryRow(`SELECT COUNT(*) FROM custom_anki_templates`).Scan(&c)
	if err != nil {
		return 0, fmt.Errorf("templatemgr: count: %w", err)
	}
	return c, nil
}

const tplSelectCols = `SELECT id, name, front_tmpl, back_tmpl, style, shared_css,
	is_builtin, preview, sort_order, created_at, updated_at`

type scanner interface {
	Scan(dest ...interface{}) error
}

func scanTemplate(row scanner) (*Template, error) {
	var t Template
	var isBuiltin int
	var createdAt, updatedAt string
	err := row.Scan(&t.ID, &t.Name, &t.FrontTmpl, &t.BackTmpl, &t.Style, &t.SharedCSS,
		&isBuiltin, &t.Preview, &t.SortOrder, &createdAt, &updatedAt)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, fmt.Errorf("templatemgr: template not found")
		}
		return nil, fmt.Errorf("templatemgr: scan: %w", err)
	}
	t.IsBuiltin = isBuiltin != 0
	t.CreatedAt = parseTime(createdAt)
	t.UpdatedAt = parseTime(updatedAt)
	return &t, nil
}

func joinSet(parts []string) string {
	out := ""
	for i, p := range parts {
		if i > 0 {
			out += ", "
		}
		out += p
	}
	return out
}

func boolToInt(b bool) int {
	if b {
		return 1
	}
	return 0
}

func formatTime(t time.Time) string {
	return t.UTC().Format(time.RFC3339Nano)
}

func parseTime(s string) time.Time {
	if s == "" {
		return time.Time{}
	}
	t, err := time.Parse(time.RFC3339Nano, s)
	if err != nil {
		t, err = time.Parse(time.RFC3339, s)
		if err != nil {
			return time.Time{}
		}
	}
	return t
}
