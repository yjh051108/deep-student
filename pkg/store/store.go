// Package store 提供 SQLite 关系数据封装。
package store

import (
	"database/sql"
	"errors"
	"fmt"

	_ "modernc.org/sqlite"
)

// Store 关系数据库封装（modernc.org/sqlite，纯 Go、无 cgo）。
type Store struct {
	DB *sql.DB
}

// Open 打开/创建数据库。
func Open(path string) (*Store, error) {
	dsn := fmt.Sprintf("file:%s?_pragma=journal_mode(WAL)&_pragma=busy_timeout(5000)&_pragma=foreign_keys(1)", path)
	db, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, err
	}
	if err := db.Ping(); err != nil {
		return nil, err
	}
	s := &Store{DB: db}
	if err := s.migrate(); err != nil {
		return nil, err
	}
	return s, nil
}

// Close 关闭数据库。
func (s *Store) Close() error {
	if s == nil || s.DB == nil {
		return nil
	}
	return s.DB.Close()
}

// Backup 把当前 SQLite 拷贝到目标路径；用 SQL Online Backup API 保证一致性，
// 避免在 VFS 写入中复制得到损坏的数据库。
func (s *Store) Backup(target string) error {
	if s == nil || s.DB == nil {
		return errors.New("store: db not open")
	}
	_, err := s.DB.Exec("VACUUM INTO ?", target)
	return err
}

func (s *Store) migrate() error {
	stmts := []string{
		`CREATE TABLE IF NOT EXISTS schema_version (version INTEGER NOT NULL)`,
		`INSERT INTO schema_version(version) SELECT 1 WHERE NOT EXISTS (SELECT 1 FROM schema_version)`,
		`CREATE TABLE IF NOT EXISTS resources (
			uri TEXT PRIMARY KEY,
			type TEXT NOT NULL,
			id TEXT NOT NULL,
			title TEXT,
			tags TEXT,
			metadata TEXT,
			blob_ref TEXT,
			size INTEGER,
			created_at INTEGER,
			updated_at INTEGER
		)`,
		`CREATE INDEX IF NOT EXISTS idx_resources_type ON resources(type)`,
		`CREATE INDEX IF NOT EXISTS idx_resources_tags ON resources(tags)`,
		`CREATE TABLE IF NOT EXISTS chunks (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			uri TEXT NOT NULL,
			pos INTEGER NOT NULL,
			content TEXT NOT NULL,
			token_count INTEGER,
			embedding BLOB
		)`,
		`CREATE INDEX IF NOT EXISTS idx_chunks_uri ON chunks(uri)`,
		`CREATE TABLE IF NOT EXISTS audit_log (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			actor TEXT,
			action TEXT NOT NULL,
			detail TEXT,
			ts INTEGER NOT NULL
		)`,
	}
	for _, s2 := range stmts {
		if _, err := s.DB.Exec(s2); err != nil {
			return fmt.Errorf("migrate: %w", err)
		}
	}
	return nil
}

// SaveResource 写入资源元数据。
func (s *Store) SaveResource(uri, typ, id, title, tags, metadata, blobRef string, size, ts int64) error {
	if s == nil || s.DB == nil {
		return errors.New("store: nil")
	}
	_, err := s.DB.Exec(`INSERT OR REPLACE INTO resources(uri, type, id, title, tags, metadata, blob_ref, size, created_at, updated_at)
		VALUES (?,?,?,?,?,?,?,?,?,?)`,
		uri, typ, id, title, tags, metadata, blobRef, size, ts, ts)
	return err
}

// DeleteResource 删除资源。
func (s *Store) DeleteResource(uri string) error {
	if s == nil || s.DB == nil {
		return errors.New("store: nil")
	}
	_, err := s.DB.Exec(`DELETE FROM resources WHERE uri=?`, uri)
	return err
}

// ListResources 按类型列出。
func (s *Store) ListResources(typ string) ([]ResourceRow, error) {
	if s == nil || s.DB == nil {
		return nil, errors.New("store: nil")
	}
	rows, err := s.DB.Query(`SELECT uri, type, id, title, tags, metadata, blob_ref, size, created_at, updated_at
		FROM resources WHERE (? = '' OR type=?) ORDER BY updated_at DESC`, typ, typ)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []ResourceRow
	for rows.Next() {
		var r ResourceRow
		if err := rows.Scan(&r.URI, &r.Type, &r.ID, &r.Title, &r.Tags, &r.Metadata, &r.BlobRef, &r.Size, &r.CreatedAt, &r.UpdatedAt); err != nil {
			return nil, err
		}
		out = append(out, r)
	}
	return out, rows.Err()
}

// ResourceRow 行结构。
type ResourceRow struct {
	URI       string
	Type      string
	ID        string
	Title     string
	Tags      string
	Metadata  string
	BlobRef   string
	Size      int64
	CreatedAt int64
	UpdatedAt int64
}

// AppendChunk 写入切片。
func (s *Store) AppendChunk(uri string, pos int, content string, tokenCount int, embedding []byte) error {
	if s == nil || s.DB == nil {
		return errors.New("store: nil")
	}
	_, err := s.DB.Exec(`INSERT INTO chunks(uri, pos, content, token_count, embedding) VALUES (?,?,?,?,?)`,
		uri, pos, content, tokenCount, embedding)
	return err
}

// Chunks 读取切片。
func (s *Store) Chunks(uri string) ([]Chunk, error) {
	if s == nil || s.DB == nil {
		return nil, errors.New("store: nil")
	}
	rows, err := s.DB.Query(`SELECT pos, content, token_count, embedding FROM chunks WHERE uri=? ORDER BY pos`, uri)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []Chunk
	for rows.Next() {
		var c Chunk
		if err := rows.Scan(&c.Pos, &c.Content, &c.TokenCount, &c.Embedding); err != nil {
			return nil, err
		}
		out = append(out, c)
	}
	return out, rows.Err()
}

// Chunk 切片记录。
type Chunk struct {
	Pos        int
	Content    string
	TokenCount int
	Embedding  []byte
}

// LogAudit 写入审计日志。
func (s *Store) LogAudit(actor, action, detail string, ts int64) error {
	if s == nil || s.DB == nil {
		return errors.New("store: nil")
	}
	_, err := s.DB.Exec(`INSERT INTO audit_log(actor, action, detail, ts) VALUES (?,?,?,?)`,
		actor, action, detail, ts)
	return err
}

// AuditLogs 读取审计日志。
func (s *Store) AuditLogs(limit int) ([]AuditEntry, error) {
	if s == nil || s.DB == nil {
		return nil, errors.New("store: nil")
	}
	if limit <= 0 {
		limit = 100
	}
	rows, err := s.DB.Query(`SELECT actor, action, detail, ts FROM audit_log ORDER BY ts DESC LIMIT ?`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []AuditEntry
	for rows.Next() {
		var a AuditEntry
		if err := rows.Scan(&a.Actor, &a.Action, &a.Detail, &a.TS); err != nil {
			return nil, err
		}
		out = append(out, a)
	}
	return out, rows.Err()
}

// AuditEntry 审计日志条目。
type AuditEntry struct {
	Actor  string
	Action string
	Detail string
	TS     int64
}
