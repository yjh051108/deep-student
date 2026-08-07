// todo 包的 SQLite 持久化层。
//
// 建立独立 todo_lists / todo_items 两张表（对齐 Rust 原版迁移
// vfs/V20260308__add_todo_tables.sql 与 V20260309__decouple_todo_from_vfs.sql）。
// 时间字段 RFC3339Nano UTC 字符串；tags / repeat 用 JSON 文本。
// 删除采用软删除（is_deleted + deleted_at），回收站可恢复或彻底清除。

package todo

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/helixnow/deep-student-go/pkg/store"
)

// Store todo 包专用存储层。
type Store struct {
	db *sql.DB
}

// NewStore 从 pkg/store.Store 构造，并执行表迁移。
func NewStore(s *store.Store) *Store {
	if s == nil || s.DB == nil {
		return &Store{}
	}
	return &Store{db: s.DB}
}

// Migrate 创建 todo 相关表与索引（幂等）。
func (s *Store) Migrate() error {
	if s.db == nil {
		return errors.New("todo: nil db")
	}
	stmts := []string{
		`CREATE TABLE IF NOT EXISTS todo_lists (
			id TEXT PRIMARY KEY,
			name TEXT NOT NULL,
			color TEXT NOT NULL DEFAULT '',
			icon TEXT NOT NULL DEFAULT '',
			is_inbox INTEGER NOT NULL DEFAULT 0,
			is_favorite INTEGER NOT NULL DEFAULT 0,
			is_deleted INTEGER NOT NULL DEFAULT 0,
			deleted_at TEXT,
			sort_order INTEGER NOT NULL DEFAULT 0,
			created_at TEXT NOT NULL,
			updated_at TEXT NOT NULL
		)`,
		`CREATE INDEX IF NOT EXISTS idx_todo_lists_deleted ON todo_lists(is_deleted)`,
		`CREATE TABLE IF NOT EXISTS todo_items (
			id TEXT PRIMARY KEY,
			list_id TEXT NOT NULL,
			title TEXT NOT NULL,
			notes TEXT NOT NULL DEFAULT '',
			due_at TEXT,
			completed_at TEXT,
			priority INTEGER NOT NULL DEFAULT 0,
			tags TEXT NOT NULL DEFAULT '[]',
			parent_id TEXT,
			est_pomodoros INTEGER NOT NULL DEFAULT 0,
			done_pomodoros INTEGER NOT NULL DEFAULT 0,
			repeat TEXT,
			remind_at TEXT,
			is_deleted INTEGER NOT NULL DEFAULT 0,
			deleted_at TEXT,
			sort_order INTEGER NOT NULL DEFAULT 0,
			created_at TEXT NOT NULL,
			updated_at TEXT NOT NULL
		)`,
		`CREATE INDEX IF NOT EXISTS idx_todo_items_list_id ON todo_items(list_id)`,
		`CREATE INDEX IF NOT EXISTS idx_todo_items_parent_id ON todo_items(parent_id)`,
		`CREATE INDEX IF NOT EXISTS idx_todo_items_due_at ON todo_items(due_at)`,
		`CREATE INDEX IF NOT EXISTS idx_todo_items_deleted ON todo_items(is_deleted)`,
	}
	for _, stmt := range stmts {
		if _, err := s.db.Exec(stmt); err != nil {
			return fmt.Errorf("todo migrate: %w", err)
		}
	}
	return nil
}

// ===================== 列表 CRUD =====================

// CreateList 插入列表。
func (s *Store) CreateList(l *List) error {
	if s.db == nil {
		return errors.New("todo: nil db")
	}
	_, err := s.db.Exec(`INSERT INTO todo_lists
		(id, name, color, icon, is_inbox, is_favorite, is_deleted, deleted_at,
		 sort_order, created_at, updated_at)
		VALUES (?,?,?,?,?,?,?,?,?,?,?)`,
		l.ID, l.Name, l.Color, l.Icon, boolToInt(l.IsInbox), boolToInt(l.IsFavorite),
		boolToInt(l.IsDeleted), nullableTime(l.DeletedAt), l.SortOrder,
		formatTime(l.CreatedAt), formatTime(l.UpdatedAt),
	)
	if err != nil {
		return fmt.Errorf("todo: create list: %w", err)
	}
	return nil
}

// GetList 按 ID 读取列表。
func (s *Store) GetList(id string) (*List, error) {
	if s.db == nil {
		return nil, errors.New("todo: nil db")
	}
	row := s.db.QueryRow(listSelectCols+` FROM todo_lists WHERE id=?`, id)
	return scanList(row)
}

// ListLists 列出列表；includeDeleted=true 时含回收站列表。
func (s *Store) ListLists(includeDeleted bool) ([]*List, error) {
	if s.db == nil {
		return nil, errors.New("todo: nil db")
	}
	query := listSelectCols + ` FROM todo_lists`
	if !includeDeleted {
		query += ` WHERE is_deleted=0`
	}
	query += ` ORDER BY is_inbox DESC, is_favorite DESC, sort_order ASC, name ASC`
	rows, err := s.db.Query(query)
	if err != nil {
		return nil, fmt.Errorf("todo: list lists: %w", err)
	}
	defer rows.Close()
	return scanLists(rows)
}

// UpdateList 更新列表字段（nil 指针不修改）。
func (s *Store) UpdateList(p UpdateListParams) error {
	if s.db == nil {
		return errors.New("todo: nil db")
	}
	setParts := []string{"updated_at = ?"}
	args := []interface{}{formatTime(time.Now().UTC())}
	if p.Name != nil {
		setParts = append(setParts, "name = ?")
		args = append(args, *p.Name)
	}
	if p.Color != nil {
		setParts = append(setParts, "color = ?")
		args = append(args, *p.Color)
	}
	if p.Icon != nil {
		setParts = append(setParts, "icon = ?")
		args = append(args, *p.Icon)
	}
	if p.Favorite != nil {
		setParts = append(setParts, "is_favorite = ?")
		args = append(args, boolToInt(*p.Favorite))
	}
	args = append(args, p.ID)
	query := `UPDATE todo_lists SET ` + joinSet(setParts) + ` WHERE id=? AND is_deleted=0`
	res, err := s.db.Exec(query, args...)
	if err != nil {
		return fmt.Errorf("todo: update list: %w", err)
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return fmt.Errorf("todo: list not found: %s", p.ID)
	}
	return nil
}

// SoftDeleteList 软删除列表（回收站）。
func (s *Store) SoftDeleteList(id string) error {
	return s.toggleListDeleted(id, true)
}

// RestoreList 从回收站恢复列表。
func (s *Store) RestoreList(id string) error {
	return s.toggleListDeleted(id, false)
}

func (s *Store) toggleListDeleted(id string, deleted bool) error {
	if s.db == nil {
		return errors.New("todo: nil db")
	}
	now := formatTime(time.Now().UTC())
	res, err := s.db.Exec(
		`UPDATE todo_lists SET is_deleted=?, deleted_at=?, updated_at=? WHERE id=? AND is_inbox=0`,
		boolToInt(deleted), nullableStrIf(now, deleted), now, id,
	)
	if err != nil {
		return fmt.Errorf("todo: toggle list deleted: %w", err)
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return fmt.Errorf("todo: list not found or is inbox: %s", id)
	}
	return nil
}

// PurgeList 彻底删除列表（含其下条目）。
func (s *Store) PurgeList(id string) error {
	if s.db == nil {
		return errors.New("todo: nil db")
	}
	tx, err := s.db.Begin()
	if err != nil {
		return fmt.Errorf("todo: begin tx: %w", err)
	}
	if _, err := tx.Exec(`DELETE FROM todo_items WHERE list_id=?`, id); err != nil {
		_ = tx.Rollback()
		return fmt.Errorf("todo: purge items: %w", err)
	}
	if _, err := tx.Exec(`DELETE FROM todo_lists WHERE id=?`, id); err != nil {
		_ = tx.Rollback()
		return fmt.Errorf("todo: purge list: %w", err)
	}
	return tx.Commit()
}

// PurgeDeletedLists 清空回收站列表。
func (s *Store) PurgeDeletedLists() (int64, error) {
	if s.db == nil {
		return 0, errors.New("todo: nil db")
	}
	res, err := s.db.Exec(`DELETE FROM todo_lists WHERE is_deleted=1`)
	if err != nil {
		return 0, fmt.Errorf("todo: purge deleted lists: %w", err)
	}
	n, _ := res.RowsAffected()
	return n, nil
}

// ListCounts 统计各列表的条目数 / 待办数 / 已完成数。
func (s *Store) ListCounts(listIDs []string) (map[string][3]int, error) {
	out := make(map[string][3]int)
	if s.db == nil || len(listIDs) == 0 {
		return out, nil
	}
	rows, err := s.db.Query(
		`SELECT list_id, COUNT(*), SUM(CASE WHEN completed_at IS NULL THEN 1 ELSE 0 END), SUM(CASE WHEN completed_at IS NOT NULL THEN 1 ELSE 0 END)
		 FROM todo_items WHERE is_deleted=0 AND list_id IN (` + placeholders(len(listIDs)) + `) GROUP BY list_id`,
		toAnySlice(listIDs)...,
	)
	if err != nil {
		return nil, fmt.Errorf("todo: list counts: %w", err)
	}
	defer rows.Close()
	for rows.Next() {
		var lid string
		var total, pending, completed sql.NullInt64
		if err := rows.Scan(&lid, &total, &pending, &completed); err != nil {
			return nil, err
		}
		out[lid] = [3]int{int(total.Int64), int(pending.Int64), int(completed.Int64)}
	}
	return out, rows.Err()
}

// ===================== 条目 CRUD =====================

// CreateItem 插入条目。
func (s *Store) CreateItem(it *Item) error {
	if s.db == nil {
		return errors.New("todo: nil db")
	}
	tagsJSON, _ := json.Marshal(it.Tags)
	if it.Tags == nil {
		tagsJSON = []byte("[]")
	}
	repeatJSON := nullableRepeat(it.Repeat)
	_, err := s.db.Exec(`INSERT INTO todo_items
		(id, list_id, title, notes, due_at, completed_at, priority, tags, parent_id,
		 est_pomodoros, done_pomodoros, repeat, remind_at, is_deleted, deleted_at,
		 sort_order, created_at, updated_at)
		VALUES (?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?)`,
		it.ID, it.ListID, it.Title, it.Notes, nullableTime(it.DueAt), nullableTime(it.CompletedAt),
		int(it.Priority), string(tagsJSON), nullableStr(it.ParentID),
		it.EstPomodoros, it.DonePomodoros, repeatJSON, nullableTime(it.RemindAt),
		boolToInt(it.IsDeleted), nullableTime(it.DeletedAt), it.SortOrder,
		formatTime(it.CreatedAt), formatTime(it.UpdatedAt),
	)
	if err != nil {
		return fmt.Errorf("todo: create item: %w", err)
	}
	return nil
}

// GetItem 按 ID 读取条目。
func (s *Store) GetItem(id string) (*Item, error) {
	if s.db == nil {
		return nil, errors.New("todo: nil db")
	}
	row := s.db.QueryRow(itemSelectCols+` FROM todo_items WHERE id=?`, id)
	return scanItem(row)
}

// ListItems 列出条目；listID 为空则跨列表；filter 控制状态。
func (s *Store) ListItems(listID string, filter ItemFilter) ([]*Item, error) {
	if s.db == nil {
		return nil, errors.New("todo: nil db")
	}
	var clauses []string
	var args []interface{}
	if listID != "" {
		clauses = append(clauses, "list_id = ?")
		args = append(args, listID)
	}
	switch filter {
	case FilterDeleted:
		clauses = append(clauses, "is_deleted = 1")
	case FilterCompleted:
		clauses = append(clauses, "is_deleted = 0 AND completed_at IS NOT NULL")
	case FilterPending:
		clauses = append(clauses, "is_deleted = 0 AND completed_at IS NULL")
	default: // all
		clauses = append(clauses, "is_deleted = 0")
	}
	query := itemSelectCols + ` FROM todo_items`
	if len(clauses) > 0 {
		query += ` WHERE ` + joinClauses(clauses)
	}
	query += ` ORDER BY is_deleted ASC, (completed_at IS NOT NULL) ASC, sort_order ASC, created_at DESC`
	rows, err := s.db.Query(query, args...)
	if err != nil {
		return nil, fmt.Errorf("todo: list items: %w", err)
	}
	defer rows.Close()
	return scanItems(rows)
}

// UpdateItem 更新条目字段（nil 指针不修改）。
func (s *Store) UpdateItem(p UpdateItemParams) error {
	if s.db == nil {
		return errors.New("todo: nil db")
	}
	setParts := []string{"updated_at = ?"}
	args := []interface{}{formatTime(time.Now().UTC())}
	if p.ListID != nil {
		setParts = append(setParts, "list_id = ?")
		args = append(args, *p.ListID)
	}
	if p.Title != nil {
		setParts = append(setParts, "title = ?")
		args = append(args, *p.Title)
	}
	if p.Notes != nil {
		setParts = append(setParts, "notes = ?")
		args = append(args, *p.Notes)
	}
	if p.DueAt != nil {
		setParts = append(setParts, "due_at = ?")
		args = append(args, formatTime(*p.DueAt))
	}
	if p.Priority != nil {
		setParts = append(setParts, "priority = ?")
		args = append(args, int(*p.Priority))
	}
	if p.Tags != nil {
		tagsJSON, _ := json.Marshal(*p.Tags)
		setParts = append(setParts, "tags = ?")
		args = append(args, string(tagsJSON))
	}
	if p.ParentID != nil {
		setParts = append(setParts, "parent_id = ?")
		args = append(args, nullableStr(p.ParentID))
	}
	if p.EstPomodoros != nil {
		setParts = append(setParts, "est_pomodoros = ?")
		args = append(args, *p.EstPomodoros)
	}
	if p.DonePomodoros != nil {
		setParts = append(setParts, "done_pomodoros = ?")
		args = append(args, *p.DonePomodoros)
	}
	if p.Repeat != nil {
		setParts = append(setParts, "repeat = ?")
		args = append(args, nullableRepeat(p.Repeat))
	}
	if p.RemindAt != nil {
		setParts = append(setParts, "remind_at = ?")
		args = append(args, formatTime(*p.RemindAt))
	}
	args = append(args, p.ID)
	query := `UPDATE todo_items SET ` + joinSet(setParts) + ` WHERE id=? AND is_deleted=0`
	res, err := s.db.Exec(query, args...)
	if err != nil {
		return fmt.Errorf("todo: update item: %w", err)
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return fmt.Errorf("todo: item not found: %s", p.ID)
	}
	return nil
}

// SetCompleted 设置完成状态（CompletedAt nil 表示未完成）。
func (s *Store) SetCompleted(id string, completedAt *time.Time) error {
	if s.db == nil {
		return errors.New("todo: nil db")
	}
	res, err := s.db.Exec(
		`UPDATE todo_items SET completed_at=?, updated_at=? WHERE id=? AND is_deleted=0`,
		nullableTime(completedAt), formatTime(time.Now().UTC()), id,
	)
	if err != nil {
		return fmt.Errorf("todo: set completed: %w", err)
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return fmt.Errorf("todo: item not found: %s", id)
	}
	return nil
}

// SoftDeleteItem 软删除条目。
func (s *Store) SoftDeleteItem(id string) error {
	if s.db == nil {
		return errors.New("todo: nil db")
	}
	now := formatTime(time.Now().UTC())
	res, err := s.db.Exec(
		`UPDATE todo_items SET is_deleted=1, deleted_at=?, updated_at=? WHERE id=? AND is_deleted=0`,
		now, now, id,
	)
	if err != nil {
		return fmt.Errorf("todo: soft delete item: %w", err)
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return fmt.Errorf("todo: item not found: %s", id)
	}
	return nil
}

// RestoreItem 从回收站恢复条目。
func (s *Store) RestoreItem(id string) error {
	if s.db == nil {
		return errors.New("todo: nil db")
	}
	res, err := s.db.Exec(
		`UPDATE todo_items SET is_deleted=0, deleted_at=NULL, updated_at=? WHERE id=? AND is_deleted=1`,
		formatTime(time.Now().UTC()), id,
	)
	if err != nil {
		return fmt.Errorf("todo: restore item: %w", err)
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return fmt.Errorf("todo: item not found in trash: %s", id)
	}
	return nil
}

// PurgeItem 彻底删除条目。
func (s *Store) PurgeItem(id string) error {
	if s.db == nil {
		return errors.New("todo: nil db")
	}
	res, err := s.db.Exec(`DELETE FROM todo_items WHERE id=?`, id)
	if err != nil {
		return fmt.Errorf("todo: purge item: %w", err)
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return fmt.Errorf("todo: item not found: %s", id)
	}
	return nil
}

// PurgeDeletedItems 清空回收站条目。
func (s *Store) PurgeDeletedItems() (int64, error) {
	if s.db == nil {
		return 0, errors.New("todo: nil db")
	}
	res, err := s.db.Exec(`DELETE FROM todo_items WHERE is_deleted=1`)
	if err != nil {
		return 0, fmt.Errorf("todo: purge deleted items: %w", err)
	}
	n, _ := res.RowsAffected()
	return n, nil
}

// ReorderItems 按给定 ID 顺序设置 sort_order。
func (s *Store) ReorderItems(listID string, ids []string) error {
	if s.db == nil {
		return errors.New("todo: nil db")
	}
	tx, err := s.db.Begin()
	if err != nil {
		return fmt.Errorf("todo: begin tx: %w", err)
	}
	for i, id := range ids {
		if _, err := tx.Exec(
			`UPDATE todo_items SET sort_order=?, updated_at=? WHERE id=? AND list_id=?`,
			i, formatTime(time.Now().UTC()), id, listID,
		); err != nil {
			_ = tx.Rollback()
			return fmt.Errorf("todo: reorder: %w", err)
		}
	}
	return tx.Commit()
}

// SubCounts 统计各条目的子任务数量。
func (s *Store) SubCounts(parentIDs []string) (map[string]int, error) {
	out := make(map[string]int)
	if s.db == nil || len(parentIDs) == 0 {
		return out, nil
	}
	rows, err := s.db.Query(
		`SELECT parent_id, COUNT(*) FROM todo_items
		 WHERE is_deleted=0 AND parent_id IN (` + placeholders(len(parentIDs)) + `) GROUP BY parent_id`,
		toAnySlice(parentIDs)...,
	)
	if err != nil {
		return nil, fmt.Errorf("todo: sub counts: %w", err)
	}
	defer rows.Close()
	for rows.Next() {
		var pid string
		var c int
		if err := rows.Scan(&pid, &c); err != nil {
			return nil, err
		}
		out[pid] = c
	}
	return out, rows.Err()
}

// ===================== 视图查询 =====================

// ListDueBetween 查询截止日期在 [start, end]（含）内的未完成条目；end 为空表示无上限。
func (s *Store) ListDueBetween(start, end *time.Time) ([]*Item, error) {
	if s.db == nil {
		return nil, errors.New("todo: nil db")
	}
	var clauses []string
	var args []interface{}
	clauses = append(clauses, "is_deleted = 0")
	clauses = append(clauses, "completed_at IS NULL")
	clauses = append(clauses, "due_at IS NOT NULL")
	if start != nil {
		clauses = append(clauses, "datetime(due_at) >= datetime(?)")
		args = append(args, formatTime(*start))
	}
	if end != nil {
		clauses = append(clauses, "datetime(due_at) <= datetime(?)")
		args = append(args, formatTime(*end))
	}
	query := itemSelectCols + ` FROM todo_items WHERE ` + joinClauses(clauses) +
		` ORDER BY due_at ASC, priority DESC`
	rows, err := s.db.Query(query, args...)
	if err != nil {
		return nil, fmt.Errorf("todo: list due between: %w", err)
	}
	defer rows.Close()
	return scanItems(rows)
}

// Search 按标题/备注关键字搜索（未删除条目）。
func (s *Store) Search(keyword string, limit int) ([]*Item, error) {
	if s.db == nil {
		return nil, errors.New("todo: nil db")
	}
	if limit <= 0 {
		limit = 100
	}
	pattern := "%" + escapeLike(keyword) + "%"
	rows, err := s.db.Query(
		`SELECT `+itemSelectCols[7:]+` FROM todo_items
		 WHERE is_deleted=0 AND (title LIKE ? OR notes LIKE ? OR tags LIKE ?)
		 ORDER BY (completed_at IS NOT NULL) ASC, updated_at DESC LIMIT ?`,
		pattern, pattern, pattern, limit,
	)
	if err != nil {
		return nil, fmt.Errorf("todo: search: %w", err)
	}
	defer rows.Close()
	return scanItems(rows)
}

// CountPending 统计未完成条目数。
func (s *Store) CountPending() (int, error) {
	if s.db == nil {
		return 0, errors.New("todo: nil db")
	}
	var c int
	err := s.db.QueryRow(`SELECT COUNT(*) FROM todo_items WHERE is_deleted=0 AND completed_at IS NULL`).Scan(&c)
	if err != nil {
		return 0, fmt.Errorf("todo: count pending: %w", err)
	}
	return c, nil
}

// CountCompleted 统计已完成条目数。
func (s *Store) CountCompleted() (int, error) {
	if s.db == nil {
		return 0, errors.New("todo: nil db")
	}
	var c int
	err := s.db.QueryRow(`SELECT COUNT(*) FROM todo_items WHERE is_deleted=0 AND completed_at IS NOT NULL`).Scan(&c)
	if err != nil {
		return 0, fmt.Errorf("todo: count completed: %w", err)
	}
	return c, nil
}

// ===================== 查询列 / 扫描 =====================

const listSelectCols = `SELECT id, name, color, icon, is_inbox, is_favorite, is_deleted, deleted_at,
	sort_order, created_at, updated_at`

const itemSelectCols = `SELECT id, list_id, title, notes, due_at, completed_at, priority, tags, parent_id,
	est_pomodoros, done_pomodoros, repeat, remind_at, is_deleted, deleted_at,
	sort_order, created_at, updated_at`

type scanner interface {
	Scan(dest ...interface{}) error
}

func scanList(row scanner) (*List, error) {
	var l List
	var isInbox, isFavorite, isDeleted int
	var deletedAt, createdAt, updatedAt sql.NullString
	err := row.Scan(&l.ID, &l.Name, &l.Color, &l.Icon, &isInbox, &isFavorite, &isDeleted,
		&deletedAt, &l.SortOrder, &createdAt, &updatedAt)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, fmt.Errorf("todo: list not found")
		}
		return nil, fmt.Errorf("todo: scan list: %w", err)
	}
	l.IsInbox = isInbox != 0
	l.IsFavorite = isFavorite != 0
	l.IsDeleted = isDeleted != 0
	l.DeletedAt = parseNullableTime(deletedAt)
	l.CreatedAt = parseTime(createdAt.String)
	l.UpdatedAt = parseTime(updatedAt.String)
	return &l, nil
}

func scanLists(rows *sql.Rows) ([]*List, error) {
	var out []*List
	for rows.Next() {
		l, err := scanList(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, l)
	}
	return out, rows.Err()
}

func scanItem(row scanner) (*Item, error) {
	var it Item
	var priority int
	var tagsJSON string
	var dueAt, completedAt, parentID, repeatJSON, remindAt, deletedAt, createdAt, updatedAt sql.NullString
	var isDeleted int
	err := row.Scan(&it.ID, &it.ListID, &it.Title, &it.Notes, &dueAt, &completedAt,
		&priority, &tagsJSON, &parentID, &it.EstPomodoros, &it.DonePomodoros,
		&repeatJSON, &remindAt, &isDeleted, &deletedAt, &it.SortOrder,
		&createdAt, &updatedAt)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, fmt.Errorf("todo: item not found")
		}
		return nil, fmt.Errorf("todo: scan item: %w", err)
	}
	it.Priority = Priority(priority)
	it.IsDeleted = isDeleted != 0
	_ = json.Unmarshal([]byte(tagsJSON), &it.Tags)
	if it.Tags == nil {
		it.Tags = []string{}
	}
	it.DueAt = parseNullableTime(dueAt)
	it.CompletedAt = parseNullableTime(completedAt)
	it.ParentID = parseNullableStr(parentID)
	it.RemindAt = parseNullableTime(remindAt)
	it.DeletedAt = parseNullableTime(deletedAt)
	if repeatJSON.Valid && repeatJSON.String != "" {
		var r Repeat
		if err := json.Unmarshal([]byte(repeatJSON.String), &r); err == nil {
			it.Repeat = &r
		}
	}
	it.CreatedAt = parseTime(createdAt.String)
	it.UpdatedAt = parseTime(updatedAt.String)
	return &it, nil
}

func scanItems(rows *sql.Rows) ([]*Item, error) {
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

// ===================== 辅助 =====================

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

func joinClauses(clauses []string) string {
	out := ""
	for i, c := range clauses {
		if i > 0 {
			out += " AND "
		}
		out += c
	}
	return out
}

func placeholders(n int) string {
	out := ""
	for i := 0; i < n; i++ {
		if i > 0 {
			out += ","
		}
		out += "?"
	}
	return out
}

func toAnySlice(ss []string) []interface{} {
	out := make([]interface{}, len(ss))
	for i, s := range ss {
		out[i] = s
	}
	return out
}

func nullableStr(s *string) sql.NullString {
	if s == nil {
		return sql.NullString{}
	}
	return sql.NullString{String: *s, Valid: true}
}

func parseNullableStr(ns sql.NullString) *string {
	if !ns.Valid {
		return nil
	}
	s := ns.String
	return &s
}

func nullableTime(t *time.Time) sql.NullString {
	if t == nil {
		return sql.NullString{}
	}
	return sql.NullString{String: formatTime(*t), Valid: true}
}

func parseNullableTime(ns sql.NullString) *time.Time {
	if !ns.Valid || ns.String == "" {
		return nil
	}
	t := parseTime(ns.String)
	return &t
}

func nullableRepeat(r *Repeat) sql.NullString {
	if r == nil {
		return sql.NullString{}
	}
	b, _ := json.Marshal(r)
	return sql.NullString{String: string(b), Valid: true}
}

func nullableStrIf(s string, cond bool) sql.NullString {
	if !cond {
		return sql.NullString{}
	}
	return sql.NullString{String: s, Valid: true}
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

func escapeLike(s string) string {
	s = strings.ReplaceAll(s, `\`, `\\`)
	s = strings.ReplaceAll(s, `%`, `\%`)
	s = strings.ReplaceAll(s, `_`, `\_`)
	return s
}
