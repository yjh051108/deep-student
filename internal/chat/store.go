// chat 包的 SQLite 持久化层（chat_v2）。
//
// 对齐 Rust 原版 migrations/chat_v2 的核心表：
//   - chat_v2_groups   会话分组（软删除）
//   - chat_v2_sessions 会话（软删除/回收站/分支/标签/模型配置）
//   - chat_v2_messages 块式消息（block_ids_json 引用块，简化版直接存文本）
// 时间字段统一 RFC3339Nano UTC 字符串。

package chat

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/helixnow/deep-student-go/pkg/store"
)

// Store chat 持久化层。
type Store struct {
	db *sql.DB
}

// NewStore 构造。
func NewStore(s *store.Store) *Store {
	if s == nil || s.DB == nil {
		return &Store{}
	}
	return &Store{db: s.DB}
}

// Migrate 建表（幂等）。
func (s *Store) Migrate() error {
	if s.db == nil {
		return errors.New("chat: nil db")
	}
	stmts := []string{
		`CREATE TABLE IF NOT EXISTS chat_v2_groups (
			id TEXT PRIMARY KEY,
			name TEXT NOT NULL,
			system_hint TEXT NOT NULL DEFAULT '',
			default_skill TEXT NOT NULL DEFAULT '',
			tags TEXT NOT NULL DEFAULT '[]',
			is_deleted INTEGER NOT NULL DEFAULT 0,
			deleted_at TEXT,
			sort_order INTEGER NOT NULL DEFAULT 0,
			created_at TEXT NOT NULL,
			updated_at TEXT NOT NULL
		)`,
		`CREATE TABLE IF NOT EXISTS chat_v2_sessions (
			id TEXT PRIMARY KEY,
			group_id TEXT,
			title TEXT NOT NULL,
			branch_of TEXT,
			tags TEXT NOT NULL DEFAULT '[]',
			model TEXT NOT NULL DEFAULT 'gpt-4o-mini',
			provider TEXT NOT NULL DEFAULT 'openai',
			system_hint TEXT NOT NULL DEFAULT '',
			is_deleted INTEGER NOT NULL DEFAULT 0,
			deleted_at TEXT,
			pinned INTEGER NOT NULL DEFAULT 0,
			created_at TEXT NOT NULL,
			updated_at TEXT NOT NULL
		)`,
		`CREATE INDEX IF NOT EXISTS idx_c2s_group ON chat_v2_sessions(group_id)`,
		`CREATE INDEX IF NOT EXISTS idx_c2s_deleted ON chat_v2_sessions(is_deleted)`,
		`CREATE INDEX IF NOT EXISTS idx_c2s_updated ON chat_v2_sessions(updated_at DESC)`,
		`CREATE TABLE IF NOT EXISTS chat_v2_messages (
			id TEXT PRIMARY KEY,
			session_id TEXT NOT NULL,
			role TEXT NOT NULL,
			content TEXT NOT NULL DEFAULT '',
			reasoning TEXT NOT NULL DEFAULT '',
			refs TEXT NOT NULL DEFAULT '[]',
			model TEXT NOT NULL DEFAULT '',
			created_at TEXT NOT NULL
		)`,
		`CREATE INDEX IF NOT EXISTS idx_c2m_session ON chat_v2_messages(session_id, created_at)`,
		`CREATE TABLE IF NOT EXISTS chat_v2_tags (
			session_id TEXT NOT NULL,
			tag TEXT NOT NULL,
			PRIMARY KEY(session_id, tag)
		)`,
	}
	for _, stmt := range stmts {
		if _, err := s.db.Exec(stmt); err != nil {
			return fmt.Errorf("chat migrate: %w", err)
		}
	}
	return nil
}

// ===================== groups =====================

// SaveGroup 写入分组。
func (s *Store) SaveGroup(g *Group) error {
	if s.db == nil {
		return errors.New("chat: nil db")
	}
	tagsJSON, _ := json.Marshal(g.Tags)
	_, err := s.db.Exec(`INSERT INTO chat_v2_groups
		(id, name, system_hint, default_skill, tags, is_deleted, deleted_at, sort_order, created_at, updated_at)
		VALUES (?,?,?,?,?,?,?,?,?,?)
		ON CONFLICT(id) DO UPDATE SET name=excluded.name, system_hint=excluded.system_hint,
			default_skill=excluded.default_skill, tags=excluded.tags, updated_at=excluded.updated_at`,
		g.ID, g.Name, g.SystemHint, g.DefaultSkill, string(tagsJSON),
		boolToInt(g.IsDeleted), nullableTime(g.DeletedAt), g.SortOrder,
		formatTime(g.CreatedAt), formatTime(g.UpdatedAt))
	return err
}

// ListGroups 列出分组（含回收站）。
func (s *Store) ListGroups(includeDeleted bool) ([]*Group, error) {
	if s.db == nil {
		return nil, errors.New("chat: nil db")
	}
	q := `SELECT id, name, system_hint, default_skill, tags, is_deleted, deleted_at, sort_order, created_at, updated_at
		FROM chat_v2_groups`
	if !includeDeleted {
		q += ` WHERE is_deleted=0`
	}
	q += ` ORDER BY sort_order ASC, name ASC`
	rows, err := s.db.Query(q)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []*Group
	for rows.Next() {
		var g Group
		var tagsJSON, deletedAt, created, updated string
		var isDeleted, sortOrder int
		if err := rows.Scan(&g.ID, &g.Name, &g.SystemHint, &g.DefaultSkill, &tagsJSON,
			&isDeleted, &deletedAt, &sortOrder, &created, &updated); err != nil {
			return nil, err
		}
		g.Tags = parseStringSlice(tagsJSON)
		g.IsDeleted = isDeleted != 0
		g.SortOrder = sortOrder
		g.CreatedAt = parseTime(created)
		g.UpdatedAt = parseTime(updated)
		if deletedAt != "" {
			t := parseTime(deletedAt)
			g.DeletedAt = &t
		}
		out = append(out, &g)
	}
	return out, rows.Err()
}

// DeleteGroup 软删除分组。
func (s *Store) DeleteGroup(id string) error {
	return s.toggleGroupDeleted(id, true)
}

// RestoreGroup 恢复分组。
func (s *Store) RestoreGroup(id string) error {
	return s.toggleGroupDeleted(id, false)
}

func (s *Store) toggleGroupDeleted(id string, deleted bool) error {
	if s.db == nil {
		return errors.New("chat: nil db")
	}
	_, err := s.db.Exec(`UPDATE chat_v2_groups SET is_deleted=?, deleted_at=?, updated_at=? WHERE id=?`,
		boolToInt(deleted), nullableStrIf(formatTime(time.Now().UTC()), deleted), formatTime(time.Now().UTC()), id)
	return err
}

// PurgeGroup 彻底删除分组。
func (s *Store) PurgeGroup(id string) error {
	if s.db == nil {
		return errors.New("chat: nil db")
	}
	_, err := s.db.Exec(`DELETE FROM chat_v2_groups WHERE id=?`, id)
	return err
}

// ===================== sessions =====================

// SaveSession 写入会话。
func (s *Store) SaveSession(se *Session) error {
	if s.db == nil {
		return errors.New("chat: nil db")
	}
	tagsJSON, _ := json.Marshal(se.Tags)
	_, err := s.db.Exec(`INSERT INTO chat_v2_sessions
		(id, group_id, title, branch_of, tags, model, provider, system_hint, is_deleted, deleted_at, pinned, created_at, updated_at)
		VALUES (?,?,?,?,?,?,?,?,?,?,?,?,?)
		ON CONFLICT(id) DO UPDATE SET group_id=excluded.group_id, title=excluded.title,
			tags=excluded.tags, model=excluded.model, provider=excluded.provider,
			system_hint=excluded.system_hint, updated_at=excluded.updated_at`,
		se.ID, se.GroupID, se.Title, se.BranchOf, string(tagsJSON),
		se.Model, se.Provider, se.SystemHint, boolToInt(se.IsDeleted), nullableTime(se.DeletedAt),
		boolToInt(se.Pinned), formatTime(se.CreatedAt), formatTime(se.UpdatedAt))
	if err != nil {
		return err
	}
	// 同步标签表
	if _, err := s.db.Exec(`DELETE FROM chat_v2_tags WHERE session_id=?`, se.ID); err != nil {
		return err
	}
	for _, tag := range se.Tags {
		if _, err := s.db.Exec(`INSERT OR IGNORE INTO chat_v2_tags(session_id, tag) VALUES (?,?)`, se.ID, tag); err != nil {
			return err
		}
	}
	return nil
}

// GetSession 读取会话（含消息）。
func (s *Store) GetSession(id string) (*Session, error) {
	if s.db == nil {
		return nil, errors.New("chat: nil db")
	}
	row := s.db.QueryRow(`SELECT id, group_id, title, branch_of, tags, model, provider, system_hint,
		is_deleted, deleted_at, pinned, created_at, updated_at FROM chat_v2_sessions WHERE id=?`, id)
	var se Session
	var tagsJSON, created, updated string
	var deletedAt sql.NullString
	var isDeleted, pinned int
	if err := row.Scan(&se.ID, &se.GroupID, &se.Title, &se.BranchOf, &tagsJSON,
		&se.Model, &se.Provider, &se.SystemHint, &isDeleted, &deletedAt, &pinned,
		&created, &updated); err != nil {
		return nil, fmt.Errorf("chat: session not found: %w", err)
	}
	se.Tags = parseStringSlice(tagsJSON)
	se.IsDeleted = isDeleted != 0
	se.Pinned = pinned != 0
	se.CreatedAt = parseTime(created)
	se.UpdatedAt = parseTime(updated)
	if deletedAt.Valid {
		t := parseTime(deletedAt.String)
		se.DeletedAt = &t
	}
	// 消息
	msgRows, err := s.db.Query(`SELECT id, role, content, reasoning, refs, model, created_at
		FROM chat_v2_messages WHERE session_id=? ORDER BY created_at ASC, rowid ASC`, id)
	if err != nil {
		return nil, err
	}
	defer msgRows.Close()
	for msgRows.Next() {
		var m Message
		var refsJSON, created string
		if err := msgRows.Scan(&m.ID, &m.Role, &m.Content, &m.Reasoning, &refsJSON, &m.Model, &created); err != nil {
			return nil, err
		}
		m.Refs = parseStringSlice(refsJSON)
		m.CreatedAt = parseTime(created)
		se.Messages = append(se.Messages, m)
	}
	return &se, nil
}

// ListSessions 列出会话（可过滤）。
func (s *Store) ListSessions(filter SessionFilter) ([]*Session, error) {
	if s.db == nil {
		return nil, errors.New("chat: nil db")
	}
	var clauses []string
	var args []interface{}
	if filter.GroupID != "" {
		clauses = append(clauses, "group_id = ?")
		args = append(args, filter.GroupID)
	}
	if filter.OnlyDeleted {
		clauses = append(clauses, "is_deleted = 1")
	} else if !filter.IncludeDeleted {
		clauses = append(clauses, "is_deleted = 0")
	}
	if filter.Keyword != "" {
		clauses = append(clauses, "title LIKE ?")
		args = append(args, "%"+escapeLike(filter.Keyword)+"%")
	}
	if len(filter.Tags) > 0 {
		for _, tag := range filter.Tags {
			clauses = append(clauses, "id IN (SELECT session_id FROM chat_v2_tags WHERE tag=?)")
			args = append(args, tag)
		}
	}
	q := `SELECT id, group_id, title, branch_of, tags, model, provider, system_hint,
		is_deleted, deleted_at, pinned, created_at, updated_at FROM chat_v2_sessions`
	if len(clauses) > 0 {
		q += ` WHERE ` + strings.Join(clauses, " AND ")
	}
	q += ` ORDER BY pinned DESC, updated_at DESC LIMIT ?`
	args = append(args, limitOr(filter.Limit, 200))
	rows, err := s.db.Query(q, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := make([]*Session, 0)
	for rows.Next() {
		var se Session
		var tagsJSON, created, updated string
		var deletedAt sql.NullString
		var isDeleted, pinned int
		if err := rows.Scan(&se.ID, &se.GroupID, &se.Title, &se.BranchOf, &tagsJSON,
			&se.Model, &se.Provider, &se.SystemHint, &isDeleted, &deletedAt, &pinned,
			&created, &updated); err != nil {
			return nil, err
		}
		se.Tags = parseStringSlice(tagsJSON)
		se.IsDeleted = isDeleted != 0
		se.Pinned = pinned != 0
		se.CreatedAt = parseTime(created)
		se.UpdatedAt = parseTime(updated)
		if deletedAt.Valid {
			t := parseTime(deletedAt.String)
			se.DeletedAt = &t
		}
		out = append(out, &se)
	}
	return out, rows.Err()
}

// SoftDeleteSession 软删除会话。
func (s *Store) SoftDeleteSession(id string) error {
	return s.toggleSessionDeleted(id, true)
}

// RestoreSession 恢复会话。
func (s *Store) RestoreSession(id string) error {
	return s.toggleSessionDeleted(id, false)
}

func (s *Store) toggleSessionDeleted(id string, deleted bool) error {
	if s.db == nil {
		return errors.New("chat: nil db")
	}
	_, err := s.db.Exec(`UPDATE chat_v2_sessions SET is_deleted=?, deleted_at=?, updated_at=? WHERE id=?`,
		boolToInt(deleted), nullableStrIf(formatTime(time.Now().UTC()), deleted), formatTime(time.Now().UTC()), id)
	return err
}

// PurgeSession 彻底删除会话（含消息与标签）。
func (s *Store) PurgeSession(id string) error {
	if s.db == nil {
		return errors.New("chat: nil db")
	}
	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	for _, stmt := range []string{
		`DELETE FROM chat_v2_messages WHERE session_id=?`,
		`DELETE FROM chat_v2_tags WHERE session_id=?`,
		`DELETE FROM chat_v2_sessions WHERE id=?`,
	} {
		if _, err := tx.Exec(stmt, id); err != nil {
			_ = tx.Rollback()
			return err
		}
	}
	return tx.Commit()
}

// PinSession 置顶/取消置顶。
func (s *Store) PinSession(id string, pinned bool) error {
	if s.db == nil {
		return errors.New("chat: nil db")
	}
	_, err := s.db.Exec(`UPDATE chat_v2_sessions SET pinned=?, updated_at=? WHERE id=?`,
		boolToInt(pinned), formatTime(time.Now().UTC()), id)
	return err
}

// UpdateSessionTitle 改标题。
func (s *Store) UpdateSessionTitle(id, title string) error {
	if s.db == nil {
		return errors.New("chat: nil db")
	}
	_, err := s.db.Exec(`UPDATE chat_v2_sessions SET title=?, updated_at=? WHERE id=?`,
		title, formatTime(time.Now().UTC()), id)
	return err
}

// ===================== messages =====================

// AppendMessage 追加消息。
func (s *Store) AppendMessage(m *Message) error {
	if s.db == nil {
		return errors.New("chat: nil db")
	}
	refsJSON, _ := json.Marshal(m.Refs)
	if m.Refs == nil {
		refsJSON = []byte("[]")
	}
	_, err := s.db.Exec(`INSERT INTO chat_v2_messages (id, session_id, role, content, reasoning, refs, model, created_at)
		VALUES (?,?,?,?,?,?,?,?)`,
		m.ID, m.SessionID, m.Role, m.Content, m.Reasoning, string(refsJSON), m.Model, formatTime(m.CreatedAt))
	if err != nil {
		return fmt.Errorf("chat: append message: %w", err)
	}
	// 更新会话时间戳
	_, _ = s.db.Exec(`UPDATE chat_v2_sessions SET updated_at=? WHERE id=?`, formatTime(m.CreatedAt), m.SessionID)
	return nil
}

// DeleteMessage 删除单条消息。
func (s *Store) DeleteMessage(sessionID, messageID string) error {
	if s.db == nil {
		return errors.New("chat: nil db")
	}
	_, err := s.db.Exec(`DELETE FROM chat_v2_messages WHERE id=? AND session_id=?`, messageID, sessionID)
	return err
}

// SearchMessages 全文搜索消息内容。
func (s *Store) SearchMessages(keyword string, limit int) ([]SearchHit, error) {
	if s.db == nil {
		return nil, errors.New("chat: nil db")
	}
	if limit <= 0 {
		limit = 50
	}
	pattern := "%" + escapeLike(keyword) + "%"
	rows, err := s.db.Query(`SELECT m.id, m.session_id, s.title, m.role, m.content, m.created_at
		FROM chat_v2_messages m LEFT JOIN chat_v2_sessions s ON s.id = m.session_id
		WHERE m.content LIKE ? AND s.is_deleted = 0
		ORDER BY m.created_at DESC LIMIT ?`, pattern, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []SearchHit
	for rows.Next() {
		var h SearchHit
		var created string
		if err := rows.Scan(&h.MessageID, &h.SessionID, &h.SessionTitle, &h.Role, &h.Content, &created); err != nil {
			return nil, err
		}
		h.CreatedAt = parseTime(created)
		out = append(out, h)
	}
	return out, rows.Err()
}

// CountSessions 统计会话数。
func (s *Store) CountSessions() (int64, error) {
	var c int64
	err := s.db.QueryRow(`SELECT COUNT(*) FROM chat_v2_sessions WHERE is_deleted=0`).Scan(&c)
	return c, err
}

// ===================== helpers =====================

// SearchHit 消息搜索命中。
type SearchHit struct {
	MessageID    string    `json:"messageId"`
	SessionID    string    `json:"sessionId"`
	SessionTitle string    `json:"sessionTitle"`
	Role         string    `json:"role"`
	Content      string    `json:"content"`
	CreatedAt    time.Time `json:"createdAt"`
}

// SessionFilter 会话列表过滤。
type SessionFilter struct {
	GroupID        string   `json:"groupId,omitempty"`
	Keyword        string   `json:"keyword,omitempty"`
	Tags           []string `json:"tags,omitempty"`
	IncludeDeleted bool     `json:"includeDeleted,omitempty"`
	OnlyDeleted    bool     `json:"onlyDeleted,omitempty"`
	Limit          int      `json:"limit,omitempty"`
}

func parseStringSlice(s string) []string {
	var out []string
	_ = json.Unmarshal([]byte(s), &out)
	if out == nil {
		out = []string{}
	}
	return out
}

func limitOr(v, def int) int {
	if v <= 0 {
		return def
	}
	return v
}

func boolToInt(b bool) int {
	if b {
		return 1
	}
	return 0
}

func nullableTime(t *time.Time) sql.NullString {
	if t == nil {
		return sql.NullString{}
	}
	return sql.NullString{String: formatTime(*t), Valid: true}
}

func nullableStrIf(s string, cond bool) sql.NullString {
	if !cond {
		return sql.NullString{}
	}
	return sql.NullString{String: s, Valid: true}
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
