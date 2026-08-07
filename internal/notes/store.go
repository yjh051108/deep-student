// notes 包的 SQLite 持久化层。
//
// 建立独立的 notes / note_folders / note_assets 三张表，不复用 resources 表。
// 表结构参考 Rust 原版 notes_manager 与 note_repo，但简化为单表存储（无版本表）。
// 时间字段统一使用 RFC3339 UTC 字符串；tags / metadata 使用 JSON 文本。

package notes

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/helixnow/deep-student-go/pkg/store"
)

// Store notes 包专用的存储层，包装 *sql.DB。
type Store struct {
	db *sql.DB
}

// NewStore 从 pkg/store.Store 构造 notes.Store，并执行表迁移。
func NewStore(s *store.Store) *Store {
	if s == nil || s.DB == nil {
		return &Store{}
	}
	return &Store{db: s.DB}
}

// Migrate 创建 notes 相关表与索引（幂等）。
func (s *Store) Migrate() error {
	if s.db == nil {
		return errors.New("notes: nil db")
	}
	stmts := []string{
		`CREATE TABLE IF NOT EXISTS notes (
			id TEXT PRIMARY KEY,
			title TEXT NOT NULL,
			content_md TEXT NOT NULL DEFAULT '',
			tags TEXT NOT NULL DEFAULT '[]',
			folder_id TEXT,
			has_assets INTEGER NOT NULL DEFAULT 0,
			asset_count INTEGER NOT NULL DEFAULT 0,
			is_pinned INTEGER NOT NULL DEFAULT 0,
			is_deleted INTEGER NOT NULL DEFAULT 0,
			deleted_at TEXT,
			word_count INTEGER NOT NULL DEFAULT 0,
			char_count INTEGER NOT NULL DEFAULT 0,
			created_at TEXT NOT NULL,
			updated_at TEXT NOT NULL,
			metadata TEXT NOT NULL DEFAULT '{}'
		)`,
		`CREATE INDEX IF NOT EXISTS idx_notes_folder_id ON notes(folder_id)`,
		`CREATE INDEX IF NOT EXISTS idx_notes_is_deleted ON notes(is_deleted)`,
		`CREATE INDEX IF NOT EXISTS idx_notes_is_pinned ON notes(is_pinned)`,
		`CREATE INDEX IF NOT EXISTS idx_notes_updated_at ON notes(updated_at)`,
		`CREATE INDEX IF NOT EXISTS idx_notes_tags ON notes(tags)`,
		`CREATE TABLE IF NOT EXISTS note_folders (
			id TEXT PRIMARY KEY,
			name TEXT NOT NULL,
			parent_id TEXT,
			sort_order INTEGER NOT NULL DEFAULT 0,
			created_at TEXT NOT NULL,
			updated_at TEXT NOT NULL
		)`,
		`CREATE INDEX IF NOT EXISTS idx_note_folders_parent_id ON note_folders(parent_id)`,
		`CREATE TABLE IF NOT EXISTS note_assets (
			id TEXT PRIMARY KEY,
			note_id TEXT NOT NULL,
			filename TEXT NOT NULL,
			mime_type TEXT NOT NULL DEFAULT 'application/octet-stream',
			size INTEGER NOT NULL DEFAULT 0,
			blob_ref TEXT NOT NULL,
			created_at TEXT NOT NULL
		)`,
		`CREATE INDEX IF NOT EXISTS idx_note_assets_note_id ON note_assets(note_id)`,
	}
	for _, stmt := range stmts {
		if _, err := s.db.Exec(stmt); err != nil {
			return fmt.Errorf("notes migrate: %w", err)
		}
	}
	return nil
}

// ===================== 笔记 CRUD =====================

// CreateNote 插入一条新笔记。
func (s *Store) CreateNote(n *Note) error {
	if s.db == nil {
		return errors.New("notes: nil db")
	}
	tagsJSON, _ := json.Marshal(n.Tags)
	if n.Tags == nil {
		tagsJSON = []byte("[]")
	}
	metaJSON, _ := json.Marshal(n.Metadata)
	if n.Metadata == nil {
		metaJSON = []byte("{}")
	}
	folderID := nullableString(n.FolderID)
	deletedAt := nullableTime(n.DeletedAt)
	_, err := s.db.Exec(`INSERT INTO notes
		(id, title, content_md, tags, folder_id, has_assets, asset_count,
		 is_pinned, is_deleted, deleted_at, word_count, char_count,
		 created_at, updated_at, metadata)
		VALUES (?,?,?,?,?,?,?,?,?,?,?,?,?,?,?)`,
		n.ID, n.Title, n.ContentMD, string(tagsJSON), folderID,
		boolToInt(n.HasAssets), n.AssetCount,
		boolToInt(n.IsPinned), boolToInt(n.IsDeleted), deletedAt,
		n.WordCount, n.CharCount,
		formatTime(n.CreatedAt), formatTime(n.UpdatedAt), string(metaJSON),
	)
	if err != nil {
		return fmt.Errorf("notes: create note: %w", err)
	}
	return nil
}

// GetNote 按 ID 读取单条笔记（含已删除）。
func (s *Store) GetNote(id string) (*Note, error) {
	if s.db == nil {
		return nil, errors.New("notes: nil db")
	}
	row := s.db.QueryRow(noteSelectColumns+` FROM notes WHERE id=?`, id)
	return scanNote(row)
}

// UpdateNote 更新笔记的可变字段。传入的字段为 nil 表示不更新。
func (s *Store) UpdateNote(id string, title *string, contentMD *string, tags *[]string,
	folderID *string, isPinned *bool, wordCount, charCount int, updatedAt time.Time) error {
	if s.db == nil {
		return errors.New("notes: nil db")
	}
	setParts := []string{"updated_at = ?"}
	args := []interface{}{formatTime(updatedAt)}
	if title != nil {
		setParts = append(setParts, "title = ?")
		args = append(args, *title)
	}
	if contentMD != nil {
		setParts = append(setParts, "content_md = ?")
		args = append(args, *contentMD)
	}
	if tags != nil {
		tagsJSON, _ := json.Marshal(*tags)
		setParts = append(setParts, "tags = ?")
		args = append(args, string(tagsJSON))
	}
	if folderID != nil {
		setParts = append(setParts, "folder_id = ?")
		args = append(args, nullableString(folderID))
	}
	if isPinned != nil {
		setParts = append(setParts, "is_pinned = ?")
		args = append(args, boolToInt(*isPinned))
	}
	if wordCount >= 0 {
		setParts = append(setParts, "word_count = ?")
		args = append(args, wordCount)
	}
	if charCount >= 0 {
		setParts = append(setParts, "char_count = ?")
		args = append(args, charCount)
	}
	args = append(args, id)
	query := `UPDATE notes SET ` + strings.Join(setParts, ", ") + ` WHERE id=?`
	res, err := s.db.Exec(query, args...)
	if err != nil {
		return fmt.Errorf("notes: update note: %w", err)
	}
	affected, _ := res.RowsAffected()
	if affected == 0 {
		return fmt.Errorf("notes: note not found: %s", id)
	}
	return nil
}

// noteSelectColumns 笔记查询列（不含 FROM 子句），与 scanNote 对齐。
const noteSelectColumns = `SELECT id, title, content_md, tags, folder_id, has_assets,
	asset_count, is_pinned, is_deleted, deleted_at, word_count, char_count,
	created_at, updated_at, metadata`

// ListNotes 按选项查询笔记列表。
//
// contentMd=false 时返回的 Note.ContentMD 为空字符串，用于列表场景降低载荷。
func (s *Store) ListNotes(opts ListOptions, includeContent bool) ([]*Note, error) {
	if s.db == nil {
		return nil, errors.New("notes: nil db")
	}
	where, args := buildWhere(opts)
	limit, offset := opts.Limit, opts.Offset
	if limit <= 0 {
		limit = 50
	}
	if offset < 0 {
		offset = 0
	}
	orderBy := buildOrderBy(opts.SortBy, opts.SortDir)
	cols := noteSelectColumns
	if !includeContent {
		// 用空字符串占位 content_md，保持列顺序与 scanNote 一致
		cols = `SELECT id, title, '' AS content_md, tags, folder_id, has_assets,
			asset_count, is_pinned, is_deleted, deleted_at, word_count, char_count,
			created_at, updated_at, metadata`
	}
	query := cols + ` FROM notes` + where + orderBy + ` LIMIT ? OFFSET ?`
	args = append(args, limit, offset)
	rows, err := s.db.Query(query, args...)
	if err != nil {
		return nil, fmt.Errorf("notes: list query: %w", err)
	}
	defer rows.Close()
	return scanNotes(rows)
}

// CountNotes 统计符合条件的笔记总数（与 ListNotes 使用同一 WHERE 子句）。
func (s *Store) CountNotes(opts ListOptions) (int64, error) {
	if s.db == nil {
		return 0, errors.New("notes: nil db")
	}
	where, args := buildWhere(opts)
	query := `SELECT COUNT(*) FROM notes` + where
	var count int64
	err := s.db.QueryRow(query, args...).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("notes: count: %w", err)
	}
	return count, nil
}

// SoftDelete 标记笔记为已删除（移入回收站）。
func (s *Store) SoftDelete(id string, at time.Time) error {
	if s.db == nil {
		return errors.New("notes: nil db")
	}
	res, err := s.db.Exec(
		`UPDATE notes SET is_deleted=1, deleted_at=?, updated_at=? WHERE id=? AND is_deleted=0`,
		formatTime(at), formatTime(at), id,
	)
	if err != nil {
		return fmt.Errorf("notes: soft delete: %w", err)
	}
	affected, _ := res.RowsAffected()
	if affected == 0 {
		return fmt.Errorf("notes: note not found or already deleted: %s", id)
	}
	return nil
}

// Restore 从回收站恢复笔记。
func (s *Store) Restore(id string, at time.Time) error {
	if s.db == nil {
		return errors.New("notes: nil db")
	}
	res, err := s.db.Exec(
		`UPDATE notes SET is_deleted=0, deleted_at=NULL, updated_at=? WHERE id=? AND is_deleted=1`,
		formatTime(at), id,
	)
	if err != nil {
		return fmt.Errorf("notes: restore: %w", err)
	}
	affected, _ := res.RowsAffected()
	if affected == 0 {
		return fmt.Errorf("notes: note not found or not in trash: %s", id)
	}
	return nil
}

// HardDelete 永久删除笔记（数据库行）。资产由 service 层负责清理。
func (s *Store) HardDelete(id string) error {
	if s.db == nil {
		return errors.New("notes: nil db")
	}
	res, err := s.db.Exec(`DELETE FROM notes WHERE id=?`, id)
	if err != nil {
		return fmt.Errorf("notes: hard delete: %w", err)
	}
	affected, _ := res.RowsAffected()
	if affected == 0 {
		return fmt.Errorf("notes: note not found: %s", id)
	}
	return nil
}

// EmptyTrash 清空回收站，返回被删除的笔记 ID 列表（供 service 层清理资产）。
func (s *Store) EmptyTrash() ([]string, error) {
	if s.db == nil {
		return nil, errors.New("notes: nil db")
	}
	rows, err := s.db.Query(`SELECT id FROM notes WHERE is_deleted=1`)
	if err != nil {
		return nil, fmt.Errorf("notes: empty trash query: %w", err)
	}
	var ids []string
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			rows.Close()
			return nil, err
		}
		ids = append(ids, id)
	}
	rows.Close()
	for _, id := range ids {
		if _, err := s.db.Exec(`DELETE FROM notes WHERE id=?`, id); err != nil {
			return ids, err
		}
	}
	return ids, nil
}

// GetTrashCount 统计回收站笔记数量。
func (s *Store) GetTrashCount() (int64, error) {
	if s.db == nil {
		return 0, errors.New("notes: nil db")
	}
	var count int64
	err := s.db.QueryRow(`SELECT COUNT(*) FROM notes WHERE is_deleted=1`).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("notes: trash count: %w", err)
	}
	return count, nil
}

// UpdateNoteAssetStats 更新笔记的资产计数标志。
func (s *Store) UpdateNoteAssetStats(noteID string, hasAssets bool, assetCount int) error {
	if s.db == nil {
		return errors.New("notes: nil db")
	}
	_, err := s.db.Exec(
		`UPDATE notes SET has_assets=?, asset_count=?, updated_at=? WHERE id=?`,
		boolToInt(hasAssets), assetCount, formatTime(time.Now().UTC()), noteID,
	)
	if err != nil {
		return fmt.Errorf("notes: update asset stats: %w", err)
	}
	return nil
}

// ===================== 文件夹 CRUD =====================

// ListFolders 列出全部文件夹，按 sort_order, name 排序。
func (s *Store) ListFolders() ([]*Folder, error) {
	if s.db == nil {
		return nil, errors.New("notes: nil db")
	}
	rows, err := s.db.Query(
		`SELECT id, name, parent_id, sort_order, created_at, updated_at
		 FROM note_folders ORDER BY sort_order ASC, name ASC`,
	)
	if err != nil {
		return nil, fmt.Errorf("notes: list folders: %w", err)
	}
	defer rows.Close()
	var out []*Folder
	for rows.Next() {
		f, err := scanFolder(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, f)
	}
	return out, rows.Err()
}

// CreateFolder 插入新文件夹。
func (s *Store) CreateFolder(f *Folder) error {
	if s.db == nil {
		return errors.New("notes: nil db")
	}
	_, err := s.db.Exec(
		`INSERT INTO note_folders (id, name, parent_id, sort_order, created_at, updated_at)
		 VALUES (?,?,?,?,?,?)`,
		f.ID, f.Name, nullableString(f.ParentID), f.SortOrder,
		formatTime(f.CreatedAt), formatTime(f.UpdatedAt),
	)
	if err != nil {
		return fmt.Errorf("notes: create folder: %w", err)
	}
	return nil
}

// UpdateFolder 更新文件夹名称。
func (s *Store) UpdateFolder(id, name string, updatedAt time.Time) error {
	if s.db == nil {
		return errors.New("notes: nil db")
	}
	res, err := s.db.Exec(
		`UPDATE note_folders SET name=?, updated_at=? WHERE id=?`,
		name, formatTime(updatedAt), id,
	)
	if err != nil {
		return fmt.Errorf("notes: update folder: %w", err)
	}
	affected, _ := res.RowsAffected()
	if affected == 0 {
		return fmt.Errorf("notes: folder not found: %s", id)
	}
	return nil
}

// DeleteFolder 删除文件夹，并将该文件夹下的笔记移到根目录。
func (s *Store) DeleteFolder(id string) error {
	if s.db == nil {
		return errors.New("notes: nil db")
	}
	tx, err := s.db.Begin()
	if err != nil {
		return fmt.Errorf("notes: begin tx: %w", err)
	}
	// 把该文件夹下的笔记移到根
	if _, err := tx.Exec(`UPDATE notes SET folder_id=NULL WHERE folder_id=?`, id); err != nil {
		_ = tx.Rollback()
		return fmt.Errorf("notes: detach notes: %w", err)
	}
	// 把子文件夹移到被删文件夹的父级（避免孤儿）
	if _, err := tx.Exec(
		`UPDATE note_folders SET parent_id=(SELECT parent_id FROM note_folders WHERE id=?)
		 WHERE parent_id=?`, id, id,
	); err != nil {
		_ = tx.Rollback()
		return fmt.Errorf("notes: reparent folders: %w", err)
	}
	if _, err := tx.Exec(`DELETE FROM note_folders WHERE id=?`, id); err != nil {
		_ = tx.Rollback()
		return fmt.Errorf("notes: delete folder: %w", err)
	}
	return tx.Commit()
}

// ===================== 资产 CRUD =====================

// AddAsset 插入资产记录。
func (s *Store) AddAsset(a *Asset) error {
	if s.db == nil {
		return errors.New("notes: nil db")
	}
	_, err := s.db.Exec(
		`INSERT INTO note_assets (id, note_id, filename, mime_type, size, blob_ref, created_at)
		 VALUES (?,?,?,?,?,?,?)`,
		a.ID, a.NoteID, a.Filename, a.MIMEType, a.Size, a.BlobRef, formatTime(a.CreatedAt),
	)
	if err != nil {
		return fmt.Errorf("notes: add asset: %w", err)
	}
	return nil
}

// ListAssets 列出指定笔记的全部资产。
func (s *Store) ListAssets(noteID string) ([]*Asset, error) {
	if s.db == nil {
		return nil, errors.New("notes: nil db")
	}
	rows, err := s.db.Query(
		`SELECT id, note_id, filename, mime_type, size, blob_ref, created_at
		 FROM note_assets WHERE note_id=? ORDER BY created_at ASC`,
		noteID,
	)
	if err != nil {
		return nil, fmt.Errorf("notes: list assets: %w", err)
	}
	defer rows.Close()
	var out []*Asset
	for rows.Next() {
		a, err := scanAsset(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, a)
	}
	return out, rows.Err()
}

// GetAsset 读取单条资产。
func (s *Store) GetAsset(assetID string) (*Asset, error) {
	if s.db == nil {
		return nil, errors.New("notes: nil db")
	}
	row := s.db.QueryRow(
		`SELECT id, note_id, filename, mime_type, size, blob_ref, created_at
		 FROM note_assets WHERE id=?`,
		assetID,
	)
	return scanAsset(row)
}

// DeleteAsset 删除资产记录。
func (s *Store) DeleteAsset(assetID string) error {
	if s.db == nil {
		return errors.New("notes: nil db")
	}
	res, err := s.db.Exec(`DELETE FROM note_assets WHERE id=?`, assetID)
	if err != nil {
		return fmt.Errorf("notes: delete asset: %w", err)
	}
	affected, _ := res.RowsAffected()
	if affected == 0 {
		return fmt.Errorf("notes: asset not found: %s", assetID)
	}
	return nil
}

// ListAssetsByNote 批量查询多个笔记的资产计数（用于列表视图）。
func (s *Store) ListAssetsByNote(noteIDs []string) (map[string]int, error) {
	if s.db == nil {
		return nil, errors.New("notes: nil db")
	}
	out := make(map[string]int)
	if len(noteIDs) == 0 {
		return out, nil
	}
	placeholders := strings.Repeat("?,", len(noteIDs))
	placeholders = placeholders[:len(placeholders)-1]
	args := make([]interface{}, len(noteIDs))
	for i, id := range noteIDs {
		args[i] = id
	}
	query := `SELECT note_id, COUNT(*) FROM note_assets WHERE note_id IN (` + placeholders + `) GROUP BY note_id`
	rows, err := s.db.Query(query, args...)
	if err != nil {
		return nil, fmt.Errorf("notes: list assets by note: %w", err)
	}
	defer rows.Close()
	for rows.Next() {
		var noteID string
		var count int
		if err := rows.Scan(&noteID, &count); err != nil {
			return nil, err
		}
		out[noteID] = count
	}
	return out, rows.Err()
}

// ===================== 辅助函数 =====================

// scanner 兼容 *sql.Row 与 *sql.Rows。
type scanner interface {
	Scan(dest ...interface{}) error
}

func scanNote(row scanner) (*Note, error) {
	var n Note
	var tagsJSON, metaJSON string
	var folderID sql.NullString
	var deletedAt sql.NullString
	var createdAt, updatedAt string
	var hasAssets, isPinned, isDeleted int
	err := row.Scan(
		&n.ID, &n.Title, &n.ContentMD, &tagsJSON, &folderID,
		&hasAssets, &n.AssetCount, &isPinned, &isDeleted, &deletedAt,
		&n.WordCount, &n.CharCount, &createdAt, &updatedAt, &metaJSON,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, fmt.Errorf("notes: note not found")
		}
		return nil, fmt.Errorf("notes: scan note: %w", err)
	}
	_ = json.Unmarshal([]byte(tagsJSON), &n.Tags)
	if n.Tags == nil {
		n.Tags = []string{}
	}
	_ = json.Unmarshal([]byte(metaJSON), &n.Metadata)
	if folderID.Valid {
	 fid := folderID.String
	 n.FolderID = &fid
	}
	n.HasAssets = hasAssets != 0
	n.IsPinned = isPinned != 0
	n.IsDeleted = isDeleted != 0
	if deletedAt.Valid {
		if t, err := time.Parse(time.RFC3339, deletedAt.String); err == nil {
			n.DeletedAt = &t
		}
	}
	n.CreatedAt = parseTime(createdAt)
	n.UpdatedAt = parseTime(updatedAt)
	return &n, nil
}

func scanNotes(rows *sql.Rows) ([]*Note, error) {
	var out []*Note
	for rows.Next() {
		n, err := scanNote(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, n)
	}
	return out, rows.Err()
}

func scanFolder(row scanner) (*Folder, error) {
	var f Folder
	var parentID sql.NullString
	var createdAt, updatedAt string
	err := row.Scan(&f.ID, &f.Name, &parentID, &f.SortOrder, &createdAt, &updatedAt)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, fmt.Errorf("notes: folder not found")
		}
		return nil, fmt.Errorf("notes: scan folder: %w", err)
	}
	if parentID.Valid {
		pid := parentID.String
		f.ParentID = &pid
	}
	f.CreatedAt = parseTime(createdAt)
	f.UpdatedAt = parseTime(updatedAt)
	return &f, nil
}

func scanAsset(row scanner) (*Asset, error) {
	var a Asset
	var createdAt string
	err := row.Scan(&a.ID, &a.NoteID, &a.Filename, &a.MIMEType, &a.Size, &a.BlobRef, &createdAt)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, fmt.Errorf("notes: asset not found")
		}
		return nil, fmt.Errorf("notes: scan asset: %w", err)
	}
	a.CreatedAt = parseTime(createdAt)
	return &a, nil
}

// buildWhere 根据 ListOptions 构造 WHERE 子句与参数。
func buildWhere(opts ListOptions) (string, []interface{}) {
	var clauses []string
	var args []interface{}
	// 软删除过滤：默认只显示未删除；IncludeDeleted=true 显示全部；OnlyDeleted=true 仅回收站
	switch {
	case opts.OnlyDeleted:
		clauses = append(clauses, "is_deleted=1")
	case !opts.IncludeDeleted:
		clauses = append(clauses, "is_deleted=0")
	}
	// 文件夹过滤
	if opts.FolderID != nil {
		if *opts.FolderID == "" {
			clauses = append(clauses, "folder_id IS NULL")
		} else {
			clauses = append(clauses, "folder_id=?")
			args = append(args, *opts.FolderID)
		}
	}
	// 关键字（标题 + 正文 LIKE）
	if kw := strings.TrimSpace(opts.Keyword); kw != "" {
		clauses = append(clauses, "(title LIKE ? OR content_md LIKE ?)")
		pattern := "%" + escapeLike(kw) + "%"
		args = append(args, pattern, pattern)
	}
	// 标签（AND 关系，匹配 tags JSON 数组中的元素）
	for _, tag := range opts.Tags {
		tag = strings.TrimSpace(tag)
		if tag == "" {
			continue
		}
		clauses = append(clauses, "tags LIKE ?")
		args = append(args, "%\""+escapeLike(tag)+"\"%")
	}
	// 资产过滤
	if opts.HasAssets != nil {
		if *opts.HasAssets {
			clauses = append(clauses, "asset_count > 0")
		} else {
			clauses = append(clauses, "asset_count = 0")
		}
	}
	// 日期范围（按 updated_at）
	if opts.DateStart != nil {
		clauses = append(clauses, "datetime(updated_at) >= datetime(?)")
		args = append(args, formatTime(*opts.DateStart))
	}
	if opts.DateEnd != nil {
		clauses = append(clauses, "datetime(updated_at) <= datetime(?)")
		args = append(args, formatTime(*opts.DateEnd))
	}
	if len(clauses) == 0 {
		return "", args
	}
	return " WHERE " + strings.Join(clauses, " AND "), args
}

// buildOrderBy 构造 ORDER BY 子句。
func buildOrderBy(sortBy, sortDir string) string {
	col := "updated_at"
	switch sortBy {
	case "created":
		col = "created_at"
	case "title":
		col = "title"
	case "wordCount":
		col = "word_count"
	case "updated":
		col = "updated_at"
	}
	dir := "DESC"
	if strings.ToLower(sortDir) == "asc" {
		dir = "ASC"
	}
	// 标题排序时统一用 ASC 作为默认更友好；其它字段默认 DESC
	if sortBy == "title" && sortDir == "" {
		dir = "ASC"
	}
	return " ORDER BY " + col + " " + dir
}

// escapeLike 转义 LIKE 模式中的特殊字符，防止通配符注入。
func escapeLike(s string) string {
	s = strings.ReplaceAll(s, `\`, `\\`)
	s = strings.ReplaceAll(s, `%`, `\%`)
	s = strings.ReplaceAll(s, `_`, `\_`)
	return s
}

// nullableString 将 *string 转为 sql.NullString。
func nullableString(s *string) sql.NullString {
	if s == nil {
		return sql.NullString{}
	}
	return sql.NullString{String: *s, Valid: true}
}

// nullableTime 将 *time.Time 转为 sql.NullString（RFC3339）。
func nullableTime(t *time.Time) sql.NullString {
	if t == nil {
		return sql.NullString{}
	}
	return sql.NullString{String: formatTime(*t), Valid: true}
}

// boolToInt Go bool → SQLite INTEGER。
func boolToInt(b bool) int {
	if b {
		return 1
	}
	return 0
}

// formatTime 时间转 RFC3339Nano UTC 字符串（保留亚秒精度，避免同秒内的更新被判定为同一时间）。
func formatTime(t time.Time) string {
	return t.UTC().Format(time.RFC3339Nano)
}

// parseTime 解析 RFC3339/RFC3339Nano 时间字符串，失败返回零值。
// time.Parse 对 RFC3339 布局会自动接受带 fractional second 的输入。
func parseTime(s string) time.Time {
	if s == "" {
		return time.Time{}
	}
	t, err := time.Parse(time.RFC3339Nano, s)
	if err != nil {
		// 兼容老数据：尝试不带 fractional 的 RFC3339
		t, err = time.Parse(time.RFC3339, s)
		if err != nil {
			return time.Time{}
		}
	}
	return t
}
