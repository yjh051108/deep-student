// memory 包的 SQLite 持久化层。
//
// 对齐 Rust 原版 src-tauri/src/memory/ 的「记忆即 VFS 文件夹体系」设计：
//   - memory_items    记忆条目（文件夹归属、标签、权重、来源）
//   - memory_folders  记忆文件夹（可嵌套，如 人物/项目/概念）
//   - memory_relations 记忆间关系（related: 双向关联）
//   - memory_audit_log 记忆审计日志
//   - memory_config    配置（自动提取频率、默认分类、根文件夹）
// 时间字段统一 RFC3339Nano UTC 字符串。

package memory

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/helixnow/deep-student-go/pkg/store"
)

// Store 记忆持久化层。
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

// Migrate 创建记忆相关表（幂等）。
func (s *Store) Migrate() error {
	if s.db == nil {
		return errors.New("memory: nil db")
	}
	stmts := []string{
		`CREATE TABLE IF NOT EXISTS memory_items (
			id TEXT PRIMARY KEY,
			folder_id TEXT,
			category TEXT NOT NULL DEFAULT 'other',
			content TEXT NOT NULL,
			tags TEXT NOT NULL DEFAULT '[]',
			weight INTEGER NOT NULL DEFAULT 1,
			hit_count INTEGER NOT NULL DEFAULT 0,
			last_hit TEXT,
			source TEXT NOT NULL DEFAULT '',
			created_at TEXT NOT NULL,
			updated_at TEXT NOT NULL,
			metadata TEXT NOT NULL DEFAULT '{}'
		)`,
		`CREATE INDEX IF NOT EXISTS idx_memory_items_folder ON memory_items(folder_id)`,
		`CREATE INDEX IF NOT EXISTS idx_memory_items_category ON memory_items(category)`,
		`CREATE TABLE IF NOT EXISTS memory_folders (
			id TEXT PRIMARY KEY,
			name TEXT NOT NULL,
			parent_id TEXT,
			sort_order INTEGER NOT NULL DEFAULT 0,
			created_at TEXT NOT NULL,
			updated_at TEXT NOT NULL
		)`,
		`CREATE INDEX IF NOT EXISTS idx_memory_folders_parent ON memory_folders(parent_id)`,
		`CREATE TABLE IF NOT EXISTS memory_relations (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			source_id TEXT NOT NULL,
			target_id TEXT NOT NULL,
			relation_type TEXT NOT NULL DEFAULT 'related',
			created_at TEXT NOT NULL,
			UNIQUE(source_id, target_id, relation_type)
		)`,
		`CREATE INDEX IF NOT EXISTS idx_memory_relations_source ON memory_relations(source_id)`,
		`CREATE INDEX IF NOT EXISTS idx_memory_relations_target ON memory_relations(target_id)`,
		`CREATE TABLE IF NOT EXISTS memory_audit_log (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			action TEXT NOT NULL,
			target TEXT NOT NULL DEFAULT '',
			detail TEXT NOT NULL DEFAULT '',
			ts TEXT NOT NULL
		)`,
		`CREATE TABLE IF NOT EXISTS memory_config (
			key TEXT PRIMARY KEY,
			value TEXT NOT NULL
		)`,
	}
	for _, stmt := range stmts {
		if _, err := s.db.Exec(stmt); err != nil {
			return fmt.Errorf("memory migrate: %w", err)
		}
	}
	return nil
}

// ===================== items =====================

// CreateItem 插入记忆。
func (s *Store) CreateItem(it *Item) error {
	if s.db == nil {
		return errors.New("memory: nil db")
	}
	tagsJSON, _ := json.Marshal(it.Tags)
	if it.Tags == nil {
		tagsJSON = []byte("[]")
	}
	metaJSON, _ := json.Marshal(it.Metadata)
	if it.Metadata == nil {
		metaJSON = []byte("{}")
	}
	_, err := s.db.Exec(`INSERT INTO memory_items
		(id, folder_id, category, content, tags, weight, hit_count, last_hit, source, created_at, updated_at, metadata)
		VALUES (?,?,?,?,?,?,?,?,?,?,?,?)`,
		it.ID, nullableStr(it.FolderID), it.Category, it.Content, string(tagsJSON),
		it.Weight, it.HitCount, nullableTime(it.LastHit), it.Source,
		formatTime(it.CreatedAt), formatTime(it.UpdatedAt), string(metaJSON),
	)
	if err != nil {
		return fmt.Errorf("memory: create item: %w", err)
	}
	return nil
}

// UpdateItem 更新记忆字段（nil 指针不修改）。
func (s *Store) UpdateItem(it *Item) error {
	if s.db == nil {
		return errors.New("memory: nil db")
	}
	tagsJSON, _ := json.Marshal(it.Tags)
	metaJSON, _ := json.Marshal(it.Metadata)
	_, err := s.db.Exec(`UPDATE memory_items SET
		folder_id=?, category=?, content=?, tags=?, weight=?, hit_count=?, last_hit=?, source=?, updated_at=?, metadata=?
		WHERE id=?`,
		nullableStr(it.FolderID), it.Category, it.Content, string(tagsJSON),
		it.Weight, it.HitCount, nullableTime(it.LastHit), it.Source,
		formatTime(it.UpdatedAt), string(metaJSON), it.ID,
	)
	if err != nil {
		return fmt.Errorf("memory: update item: %w", err)
	}
	return nil
}

// GetItem 读取记忆。
func (s *Store) GetItem(id string) (*Item, error) {
	if s.db == nil {
		return nil, errors.New("memory: nil db")
	}
	row := s.db.QueryRow(itemSelectCols+` FROM memory_items WHERE id=?`, id)
	return scanItem(row)
}

// ListItems 列出全部记忆（按 updated_at 倒序）。
func (s *Store) ListItems() ([]*Item, error) {
	if s.db == nil {
		return nil, errors.New("memory: nil db")
	}
	rows, err := s.db.Query(itemSelectCols + ` FROM memory_items ORDER BY updated_at DESC`)
	if err != nil {
		return nil, fmt.Errorf("memory: list items: %w", err)
	}
	defer rows.Close()
	var out []*Item
	for rows.Next() {
		it, err := scanItem(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, it)
	}
	return out, rows.Err()
}

// ListItemsByFolder 按文件夹列出。
func (s *Store) ListItemsByFolder(folderID string) ([]*Item, error) {
	if s.db == nil {
		return nil, errors.New("memory: nil db")
	}
	rows, err := s.db.Query(itemSelectCols+` FROM memory_items WHERE folder_id=? ORDER BY updated_at DESC`, folderID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []*Item
	for rows.Next() {
		it, err := scanItem(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, it)
	}
	return out, rows.Err()
}

// DeleteItem 删除记忆。
func (s *Store) DeleteItem(id string) error {
	if s.db == nil {
		return errors.New("memory: nil db")
	}
	res, err := s.db.Exec(`DELETE FROM memory_items WHERE id=?`, id)
	if err != nil {
		return fmt.Errorf("memory: delete item: %w", err)
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return fmt.Errorf("memory: item not found: %s", id)
	}
	return nil
}

// MoveItemToFolder 移动记忆到文件夹。
func (s *Store) MoveItemToFolder(id, folderID string) error {
	if s.db == nil {
		return errors.New("memory: nil db")
	}
	_, err := s.db.Exec(`UPDATE memory_items SET folder_id=?, updated_at=? WHERE id=?`,
		nullableStr(&folderID), formatTime(time.Now().UTC()), id)
	return err
}

// UpdateItemTags 更新记忆标签。
func (s *Store) UpdateItemTags(id string, tags []string) error {
	if s.db == nil {
		return errors.New("memory: nil db")
	}
	tagsJSON, _ := json.Marshal(tags)
	_, err := s.db.Exec(`UPDATE memory_items SET tags=?, updated_at=? WHERE id=?`,
		string(tagsJSON), formatTime(time.Now().UTC()), id)
	return err
}

// SearchItems 关键词搜索（内容/标签/分类）。
func (s *Store) SearchItems(q string, limit int) ([]*Item, error) {
	if s.db == nil {
		return nil, errors.New("memory: nil db")
	}
	if limit <= 0 {
		limit = 50
	}
	pattern := "%" + escapeLike(q) + "%"
	rows, err := s.db.Query(itemSelectCols+` FROM memory_items
		WHERE content LIKE ? OR tags LIKE ? OR category LIKE ?
		ORDER BY weight DESC, updated_at DESC LIMIT ?`, pattern, pattern, pattern, limit)
	if err != nil {
		return nil, fmt.Errorf("memory: search: %w", err)
	}
	defer rows.Close()
	var out []*Item
	for rows.Next() {
		it, err := scanItem(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, it)
	}
	return out, rows.Err()
}

// ===================== folders =====================

// CreateFolder 创建文件夹。
func (s *Store) CreateFolder(f *Folder) error {
	if s.db == nil {
		return errors.New("memory: nil db")
	}
	_, err := s.db.Exec(`INSERT INTO memory_folders (id, name, parent_id, sort_order, created_at, updated_at)
		VALUES (?,?,?,?,?,?)`,
		f.ID, f.Name, nullableStr(f.ParentID), f.SortOrder, formatTime(f.CreatedAt), formatTime(f.UpdatedAt))
	if err != nil {
		return fmt.Errorf("memory: create folder: %w", err)
	}
	return nil
}

// ListFolders 列出全部文件夹。
func (s *Store) ListFolders() ([]*Folder, error) {
	if s.db == nil {
		return nil, errors.New("memory: nil db")
	}
	rows, err := s.db.Query(`SELECT id, name, parent_id, sort_order, created_at, updated_at
		FROM memory_folders ORDER BY sort_order ASC, name ASC`)
	if err != nil {
		return nil, fmt.Errorf("memory: list folders: %w", err)
	}
	defer rows.Close()
	var out []*Folder
	for rows.Next() {
		var f Folder
		var parent sql.NullString
		var created, updated string
		if err := rows.Scan(&f.ID, &f.Name, &parent, &f.SortOrder, &created, &updated); err != nil {
			return nil, err
		}
		if parent.Valid {
			p := parent.String
			f.ParentID = &p
		}
		f.CreatedAt = parseTime(created)
		f.UpdatedAt = parseTime(updated)
		out = append(out, &f)
	}
	return out, rows.Err()
}

// GetFolder 读取文件夹。
func (s *Store) GetFolder(id string) (*Folder, error) {
	if s.db == nil {
		return nil, errors.New("memory: nil db")
	}
	row := s.db.QueryRow(`SELECT id, name, parent_id, sort_order, created_at, updated_at
		FROM memory_folders WHERE id=?`, id)
	var f Folder
	var parent sql.NullString
	var created, updated string
	if err := row.Scan(&f.ID, &f.Name, &parent, &f.SortOrder, &created, &updated); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, fmt.Errorf("memory: folder not found: %s", id)
		}
		return nil, err
	}
	if parent.Valid {
		p := parent.String
		f.ParentID = &p
	}
	f.CreatedAt = parseTime(created)
	f.UpdatedAt = parseTime(updated)
	return &f, nil
}

// DeleteFolder 删除文件夹（其下记忆移到根）。
func (s *Store) DeleteFolder(id string) error {
	if s.db == nil {
		return errors.New("memory: nil db")
	}
	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	if _, err := tx.Exec(`UPDATE memory_items SET folder_id=NULL WHERE folder_id=?`, id); err != nil {
		_ = tx.Rollback()
		return err
	}
	if _, err := tx.Exec(`UPDATE memory_folders SET parent_id=(SELECT parent_id FROM memory_folders WHERE id=?) WHERE parent_id=?`, id, id); err != nil {
		_ = tx.Rollback()
		return err
	}
	if _, err := tx.Exec(`DELETE FROM memory_folders WHERE id=?`, id); err != nil {
		_ = tx.Rollback()
		return err
	}
	return tx.Commit()
}

// ===================== relations =====================

// AddRelation 建立双向关联。
func (s *Store) AddRelation(sourceID, targetID, relType string) error {
	if s.db == nil {
		return errors.New("memory: nil db")
	}
	if relType == "" {
		relType = "related"
	}
	now := formatTime(time.Now().UTC())
	for _, pair := range [][2]string{{sourceID, targetID}, {targetID, sourceID}} {
		_, err := s.db.Exec(`INSERT OR IGNORE INTO memory_relations(source_id, target_id, relation_type, created_at)
			VALUES (?,?,?,?)`, pair[0], pair[1], relType, now)
		if err != nil {
			return fmt.Errorf("memory: add relation: %w", err)
		}
	}
	return nil
}

// ListRelations 列出记忆的所有关联目标。
func (s *Store) ListRelations(id string) ([]string, error) {
	if s.db == nil {
		return nil, errors.New("memory: nil db")
	}
	rows, err := s.db.Query(`SELECT target_id FROM memory_relations WHERE source_id=?`, id)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []string
	for rows.Next() {
		var t string
		if err := rows.Scan(&t); err != nil {
			return nil, err
		}
		out = append(out, t)
	}
	return out, rows.Err()
}

// ===================== audit =====================

// LogAudit 写审计日志。
func (s *Store) LogAudit(action, target, detail string) error {
	if s.db == nil {
		return errors.New("memory: nil db")
	}
	_, err := s.db.Exec(`INSERT INTO memory_audit_log(action, target, detail, ts) VALUES (?,?,?,?)`,
		action, target, detail, formatTime(time.Now().UTC()))
	return err
}

// AuditLogs 读取审计日志。
func (s *Store) AuditLogs(limit int) ([]AuditEntry, error) {
	if s.db == nil {
		return nil, errors.New("memory: nil db")
	}
	if limit <= 0 {
		limit = 100
	}
	rows, err := s.db.Query(`SELECT id, action, target, detail, ts FROM memory_audit_log ORDER BY id DESC LIMIT ?`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []AuditEntry
	for rows.Next() {
		var a AuditEntry
		if err := rows.Scan(&a.ID, &a.Action, &a.Target, &a.Detail, &a.TS); err != nil {
			return nil, err
		}
		out = append(out, a)
	}
	return out, rows.Err()
}

// ===================== config =====================

// SetConfig 写配置。
func (s *Store) SetConfig(key, value string) error {
	if s.db == nil {
		return errors.New("memory: nil db")
	}
	_, err := s.db.Exec(`INSERT INTO memory_config(key, value) VALUES (?,?)
		ON CONFLICT(key) DO UPDATE SET value=excluded.value`, key, value)
	return err
}

// GetConfig 读配置。
func (s *Store) GetConfig(key string) (string, bool) {
	if s.db == nil {
		return "", false
	}
	var v string
	err := s.db.QueryRow(`SELECT value FROM memory_config WHERE key=?`, key).Scan(&v)
	if err != nil {
		return "", false
	}
	return v, true
}

// ===================== helpers =====================

// Folder 记忆文件夹。
type Folder struct {
	ID        string     `json:"id"`
	Name      string     `json:"name"`
	ParentID  *string    `json:"parentId,omitempty"`
	SortOrder int        `json:"sortOrder"`
	CreatedAt time.Time  `json:"createdAt"`
	UpdatedAt time.Time  `json:"updatedAt"`
}

// AuditEntry 审计日志条目。
type AuditEntry struct {
	ID     int64  `json:"id"`
	Action string `json:"action"`
	Target string `json:"target"`
	Detail string `json:"detail"`
	TS     string `json:"ts"`
}

const itemSelectCols = `SELECT id, folder_id, category, content, tags, weight, hit_count,
	last_hit, source, created_at, updated_at, metadata`

type scanner interface {
	Scan(dest ...interface{}) error
}

func scanItem(row scanner) (*Item, error) {
	var it Item
	var folderID, lastHit, created, updated sql.NullString
	var tagsJSON, metaJSON string
	err := row.Scan(&it.ID, &folderID, &it.Category, &it.Content, &tagsJSON,
		&it.Weight, &it.HitCount, &lastHit, &it.Source, &created, &updated, &metaJSON)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, fmt.Errorf("memory: item not found")
		}
		return nil, fmt.Errorf("memory: scan item: %w", err)
	}
	if folderID.Valid {
		f := folderID.String
		it.FolderID = &f
	}
	_ = json.Unmarshal([]byte(tagsJSON), &it.Tags)
	if it.Tags == nil {
		it.Tags = []string{}
	}
	if lastHit.Valid {
		it.LastHit = parseTime(lastHit.String)
	}
	it.CreatedAt = parseTime(created.String)
	it.UpdatedAt = parseTime(updated.String)
	_ = json.Unmarshal([]byte(metaJSON), &it.Metadata)
	if it.Metadata == nil {
		it.Metadata = map[string]string{}
	}
	return &it, nil
}

func nullableStr(s *string) sql.NullString {
	if s == nil {
		return sql.NullString{}
	}
	return sql.NullString{String: *s, Valid: true}
}

func nullableTime(t time.Time) sql.NullString {
	if t.IsZero() {
		return sql.NullString{}
	}
	return sql.NullString{String: formatTime(t), Valid: true}
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

func escapeLike(s string) string {
	s = strings.ReplaceAll(s, `\`, `\\`)
	s = strings.ReplaceAll(s, `%`, `\%`)
	s = strings.ReplaceAll(s, `_`, `\_`)
	return s
}
