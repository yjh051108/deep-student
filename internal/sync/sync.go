// Package sync 提供记录级增量同步引擎。
//
// 对齐 Rust 原版 data_governance/sync（__change_log 触发器 + 记录级增量）：
//   - 核心表通过 SQLite 触发器自动写入 __change_log；
//   - ExportChanges 导出自上次 LSN 以来的变更（含行快照 JSON）；
//   - ApplyChanges 以 LWW（updated_at 字符串比较）应用远端变更，
//     冲突/失败记录进入隔离区（quarantine）；
//   - SyncToCloud 通过 cloudstorage.Backend 上传/下载变更并推进游标。
package sync

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strings"
	"sync"
	"time"

	"github.com/helixnow/deep-student-go/internal/cloudstorage"
	"github.com/helixnow/deep-student-go/pkg/store"
)

// TableSpec 被跟踪表的结构描述（用于导出/应用行快照）。
type TableSpec struct {
	Name        string   // 表名
	IDColumn    string   // 主键列
	UpdatedCol  string   // 更新时间列（RFC3339 字符串，用于 LWW 与冲突）
	SelectCols  string   // SELECT 列（不含表名）
	InsertCols  string   // INSERT 列（不含表名）
	InsertVals  string   // 占位符（? 序列）
	IsDeletedCol string  // 软删除标志列（可空；用于 tombstone）
}

// TrackedTables 参与同步的表。
var TrackedTables = []TableSpec{
	{
		Name: "notes", IDColumn: "id", UpdatedCol: "updated_at",
		SelectCols: "id, title, content_md, tags, folder_id, has_assets, asset_count, is_pinned, is_deleted, deleted_at, word_count, char_count, created_at, updated_at, metadata",
		InsertCols: "id, title, content_md, tags, folder_id, has_assets, asset_count, is_pinned, is_deleted, deleted_at, word_count, char_count, created_at, updated_at, metadata",
		InsertVals: "?,?,?,?,?,?,?,?,?,?,?,?,?,?,?",
		IsDeletedCol: "is_deleted",
	},
	{
		Name: "todo_lists", IDColumn: "id", UpdatedCol: "updated_at",
		SelectCols: "id, name, color, icon, is_inbox, is_favorite, is_deleted, deleted_at, sort_order, created_at, updated_at",
		InsertCols: "id, name, color, icon, is_inbox, is_favorite, is_deleted, deleted_at, sort_order, created_at, updated_at",
		InsertVals: "?,?,?,?,?,?,?,?,?,?,?",
		IsDeletedCol: "is_deleted",
	},
	{
		Name: "todo_items", IDColumn: "id", UpdatedCol: "updated_at",
		SelectCols: "id, list_id, title, notes, due_at, completed_at, priority, tags, parent_id, est_pomodoros, done_pomodoros, repeat, remind_at, is_deleted, deleted_at, sort_order, created_at, updated_at",
		InsertCols: "id, list_id, title, notes, due_at, completed_at, priority, tags, parent_id, est_pomodoros, done_pomodoros, repeat, remind_at, is_deleted, deleted_at, sort_order, created_at, updated_at",
		InsertVals: "?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?",
		IsDeletedCol: "is_deleted",
	},
	{
		Name: "pomodoro_records", IDColumn: "id", UpdatedCol: "created_at",
		SelectCols: "id, todo_item_id, start_time, end_time, duration, actual_duration, type, status, created_at",
		InsertCols: "id, todo_item_id, start_time, end_time, duration, actual_duration, type, status, created_at",
		InsertVals: "?,?,?,?,?,?,?,?,?",
	},
	{
		Name: "custom_anki_templates", IDColumn: "id", UpdatedCol: "updated_at",
		SelectCols: "id, name, front_tmpl, back_tmpl, style, shared_css, is_builtin, preview, sort_order, created_at, updated_at",
		InsertCols: "id, name, front_tmpl, back_tmpl, style, shared_css, is_builtin, preview, sort_order, created_at, updated_at",
		InsertVals: "?,?,?,?,?,?,?,?,?,?,?",
	},
	{
		Name: "llm_usage_logs", IDColumn: "id", UpdatedCol: "timestamp",
		SelectCols: "id, timestamp, provider, model, adapter, api_config_id, prompt_tokens, completion_tokens, total_tokens, reasoning_tokens, cached_tokens, token_source, duration_ms, request_bytes, response_bytes, first_token_ms, caller_type, session_id, status, error_message, cost_estimate",
		InsertCols: "id, timestamp, provider, model, adapter, api_config_id, prompt_tokens, completion_tokens, total_tokens, reasoning_tokens, cached_tokens, token_source, duration_ms, request_bytes, response_bytes, first_token_ms, caller_type, session_id, status, error_message, cost_estimate",
		InsertVals: "?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?",
	},
}

// Change 一条变更日志。
type Change struct {
	ID        int64  `json:"id"`
	Table     string `json:"table"`
	RecordID  string `json:"recordId"`
	Operation string `json:"operation"` // INSERT | UPDATE | DELETE
	ChangedAt string `json:"changedAt"`
}

// RowSnapshot 变更附带的行快照（导出时生成）。
type RowSnapshot struct {
	Table     string            `json:"table"`
	RecordID  string            `json:"recordId"`
	Operation string            `json:"operation"`
	UpdatedAt string            `json:"updatedAt"`
	Columns   map[string]any    `json:"columns,omitempty"`
	Raw       string            `json:"raw,omitempty"` // \x1f 分隔的原始列值（供远端 upsert）
}

// ChangeBatch 一次导出的变更批次。
type ChangeBatch struct {
	DeviceID  string        `json:"deviceId"`
	LastSeq   int64         `json:"lastSeq"` // 本地已同步游标
	MaxSeq    int64         `json:"maxSeq"`  // 本批次最大 seq
	Changes   []RowSnapshot `json:"changes"`
}

// Service 同步服务。
type Service struct {
	db *store.Store
	mu sync.Mutex
}

// New 构造同步服务并安装基础表（__change_log / 隔离区 / 状态）。
// 注意：触发器由 EnsureTriggers 在全部业务表（notes/todo 等）创建后调用，
// 因为 sync 包的装配顺序不保证早于各业务服务。
func New(st *store.Store) *Service {
	s := &Service{db: st}
	if err := s.InstallBase(); err != nil {
		fmt.Printf("[sync] install base failed: %v\n", err)
	}
	return s
}

// InstallBase 创建 __change_log 表与同步辅助表（幂等，不建触发器）。
func (s *Service) InstallBase() error {
	if s.db == nil || s.db.DB == nil {
		return errors.New("sync: nil db")
	}
	stmts := []string{
		`CREATE TABLE IF NOT EXISTS __change_log (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			table_name TEXT NOT NULL,
			record_id TEXT NOT NULL,
			operation TEXT NOT NULL CHECK(operation IN ('INSERT','UPDATE','DELETE')),
			changed_at TEXT NOT NULL
		)`,
		`CREATE INDEX IF NOT EXISTS idx__change_log_id ON __change_log(id)`,
		`CREATE TABLE IF NOT EXISTS sync_quarantine (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			table_name TEXT NOT NULL,
			record_id TEXT NOT NULL,
			reason TEXT NOT NULL,
			payload TEXT NOT NULL,
			created_at TEXT NOT NULL
		)`,
		`CREATE TABLE IF NOT EXISTS sync_state (
			key TEXT PRIMARY KEY,
			value TEXT NOT NULL
		)`,
	}
	for _, stmt := range stmts {
		if _, err := s.db.DB.Exec(stmt); err != nil {
			return fmt.Errorf("sync install base: %w", err)
		}
	}
	return nil
}

// EnsureTriggers 为已存在的跟踪表安装变更触发器（幂等）。
// 表不存在时跳过（该表由业务服务稍后创建；可在装配完成后重调本方法）。
func (s *Service) EnsureTriggers() error {
	if s.db == nil || s.db.DB == nil {
		return errors.New("sync: nil db")
	}
	for _, t := range TrackedTables {
		idCol := t.IDColumn
		for _, op := range []string{"INSERT", "UPDATE", "DELETE"} {
			trigName := "trg__change_log_" + t.Name + "_" + strings.ToLower(op)
			var stmt string
			switch op {
			case "INSERT":
				stmt = fmt.Sprintf(`CREATE TRIGGER IF NOT EXISTS %s AFTER INSERT ON %s
					BEGIN INSERT INTO __change_log(table_name, record_id, operation, changed_at)
					VALUES ('%s', NEW.%s, 'INSERT', strftime('%%Y-%%m-%%dT%%H:%%M:%%fZ','now')); END`,
					trigName, t.Name, t.Name, idCol)
			case "UPDATE":
				stmt = fmt.Sprintf(`CREATE TRIGGER IF NOT EXISTS %s AFTER UPDATE ON %s
					BEGIN INSERT INTO __change_log(table_name, record_id, operation, changed_at)
					VALUES ('%s', NEW.%s, 'UPDATE', strftime('%%Y-%%m-%%dT%%H:%%M:%%fZ','now')); END`,
					trigName, t.Name, t.Name, idCol)
			case "DELETE":
				stmt = fmt.Sprintf(`CREATE TRIGGER IF NOT EXISTS %s AFTER DELETE ON %s
					BEGIN INSERT INTO __change_log(table_name, record_id, operation, changed_at)
					VALUES ('%s', OLD.%s, 'DELETE', strftime('%%Y-%%m-%%dT%%H:%%M:%%fZ','now')); END`,
					trigName, t.Name, t.Name, idCol)
			}
			if _, err := s.db.DB.Exec(stmt); err != nil {
				// 表尚未创建：跳过（调用方装配完成后可重调）
				if isNoSuchTable(err) {
					continue
				}
				return fmt.Errorf("sync trigger %s: %w", trigName, err)
			}
		}
	}
	return nil
}

// isNoSuchTable 判断是否因表不存在而失败。
func isNoSuchTable(err error) bool {
	return err != nil && strings.Contains(err.Error(), "no such table")
}

// Pending 返回自 lastSeq 以来的变更日志。
func (s *Service) Pending(lastSeq int64, limit int) ([]Change, error) {
	if limit <= 0 {
		limit = 500
	}
	rows, err := s.db.DB.Query(`SELECT id, table_name, record_id, operation, changed_at
		FROM __change_log WHERE id > ? ORDER BY id LIMIT ?`, lastSeq, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []Change
	for rows.Next() {
		var c Change
		if err := rows.Scan(&c.ID, &c.Table, &c.RecordID, &c.Operation, &c.ChangedAt); err != nil {
			return nil, err
		}
		out = append(out, c)
	}
	return out, rows.Err()
}

// MaxSeq 返回当前最大 LSN。
func (s *Service) MaxSeq() (int64, error) {
	var v int64
	err := s.db.DB.QueryRow(`SELECT COALESCE(MAX(id),0) FROM __change_log`).Scan(&v)
	return v, err
}

// ExportChanges 导出变更批次（含行快照；DELETE 只带 tombstone 标记）。
func (s *Service) ExportChanges(deviceID string, lastSeq, limit int64) (*ChangeBatch, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	changes, err := s.Pending(lastSeq, int(limit))
	if err != nil {
		return nil, err
	}
	if len(changes) == 0 {
		maxSeq, _ := s.MaxSeq()
		return &ChangeBatch{DeviceID: deviceID, LastSeq: lastSeq, MaxSeq: maxSeq}, nil
	}
	batch := &ChangeBatch{DeviceID: deviceID, LastSeq: lastSeq}
	var maxSeq int64
	for _, c := range changes {
		if c.ID > maxSeq {
			maxSeq = c.ID
		}
		snap, err := s.snapshot(c)
		if err != nil {
			// 快照失败（行已被删等）：记录 tombstone
			snap = RowSnapshot{Table: c.Table, RecordID: c.RecordID, Operation: c.Operation, UpdatedAt: c.ChangedAt}
		}
		batch.Changes = append(batch.Changes, snap)
	}
	batch.MaxSeq = maxSeq
	return batch, nil
}

// snapshot 从当前库读取行快照。
func (s *Service) snapshot(c Change) (RowSnapshot, error) {
	spec, ok := tableSpec(c.Table)
	if !ok {
		return RowSnapshot{}, fmt.Errorf("sync: unknown table %s", c.Table)
	}
	snap := RowSnapshot{Table: c.Table, RecordID: c.RecordID, Operation: c.Operation, UpdatedAt: c.ChangedAt, Columns: map[string]any{}}
	if c.Operation == "DELETE" {
		return snap, nil
	}
	rows, err := s.db.DB.Query(`SELECT `+spec.SelectCols+` FROM `+spec.Name+` WHERE `+spec.IDColumn+`=?`, c.RecordID)
	if err != nil {
		return snap, err
	}
	defer rows.Close()
	cols, err := rows.Columns()
	if err != nil {
		return snap, err
	}
	if !rows.Next() {
		return snap, fmt.Errorf("sync: row gone: %s/%s", c.Table, c.RecordID)
	}
	vals := make([]any, len(cols))
	ptrs := make([]any, len(cols))
	for i := range vals {
		ptrs[i] = &vals[i]
	}
	if err := rows.Scan(ptrs...); err != nil {
		return snap, err
	}
	raw := make([]string, len(cols))
	for i, v := range vals {
		switch tv := v.(type) {
		case nil:
			raw[i] = ""
			snap.Columns[cols[i]] = nil
		case []byte:
			raw[i] = string(tv)
			snap.Columns[cols[i]] = string(tv)
		case int64:
			raw[i] = fmt.Sprintf("%d", tv)
			snap.Columns[cols[i]] = tv
		case float64:
			raw[i] = fmt.Sprintf("%g", tv)
			snap.Columns[cols[i]] = tv
		default:
			raw[i] = fmt.Sprintf("%v", tv)
			snap.Columns[cols[i]] = fmt.Sprintf("%v", tv)
		}
	}
	snap.Raw = strings.Join(raw, "\x1f")
	if i, ok := colIndex(cols, spec.UpdatedCol); ok {
		snap.UpdatedAt = raw[i]
	}
	return snap, nil
}

// ApplyChanges 应用远端变更批次（LWW 合并 + 隔离区）。
func (s *Service) ApplyChanges(batch *ChangeBatch) (applied, quarantined int, err error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, snap := range batch.Changes {
		spec, ok := tableSpec(snap.Table)
		if !ok {
			quarantined++
			_ = s.quarantine(snap.Table, snap.RecordID, "unknown table", snap)
			continue
		}
		if err := s.applyOne(spec, snap); err != nil {
			quarantined++
			_ = s.quarantine(snap.Table, snap.RecordID, err.Error(), snap)
			continue
		}
		applied++
	}
	// 推进游标
	if batch.MaxSeq > 0 {
		cur, _ := s.Cursor()
		if batch.MaxSeq > cur {
			_ = s.SetCursor(batch.MaxSeq)
		}
	}
	return applied, quarantined, nil
}

// applyOne 应用单条变更（LWW）。
func (s *Service) applyOne(spec TableSpec, snap RowSnapshot) error {
	if snap.Operation == "DELETE" {
		// tombstone：软删优先；无软删列则硬删
		if spec.IsDeletedCol != "" {
			_, err := s.db.DB.Exec(fmt.Sprintf(`UPDATE %s SET %s=1, updated_at=strftime('%%Y-%%m-%%dT%%H:%%M:%%fZ','now') WHERE %s=?`, spec.Name, spec.IsDeletedCol, spec.IDColumn), snap.RecordID)
			return err
		}
		_, err := s.db.DB.Exec(fmt.Sprintf(`DELETE FROM %s WHERE %s=?`, spec.Name, spec.IDColumn), snap.RecordID)
		return err
	}
	// INSERT/UPDATE：本地行存在则比较 updated_at
	var localUpdated string
	err := s.db.DB.QueryRow(fmt.Sprintf(`SELECT %s FROM %s WHERE %s=?`, spec.UpdatedCol, spec.Name, spec.IDColumn), snap.RecordID).Scan(&localUpdated)
	if err == nil && localUpdated > snap.UpdatedAt {
		// 本地更新 → 冲突：本地胜出（记录隔离区提示）
		return fmt.Errorf("local newer (%s > %s)", localUpdated, snap.UpdatedAt)
	}
	if snap.Raw == "" {
		return fmt.Errorf("empty snapshot payload")
	}
	cols := strings.Split(spec.SelectCols, ", ")
	vals := strings.Split(snap.Raw, "\x1f")
	if len(cols) != len(vals) {
		return fmt.Errorf("column count mismatch (%d vs %d)", len(cols), len(vals))
	}
	args := make([]any, len(vals))
	for i, v := range vals {
		args[i] = v
	}
	upsert := fmt.Sprintf(`INSERT INTO %s (%s) VALUES (%s)
		ON CONFLICT(%s) DO UPDATE SET %s`,
		spec.Name, spec.InsertCols, spec.InsertVals, spec.IDColumn, upsertSet(spec, cols))
	_, err = s.db.DB.Exec(upsert, args...)
	return err
}

// upsertSet 生成 DO UPDATE SET 子句（跳过主键列）。
func upsertSet(spec TableSpec, cols []string) string {
	var parts []string
	for _, c := range cols {
		if c == spec.IDColumn {
			continue
		}
		parts = append(parts, c+"=excluded."+c)
	}
	return strings.Join(parts, ", ")
}

// quarantine 记录隔离区。
func (s *Service) quarantine(table, recordID, reason string, snap RowSnapshot) error {
	payload, _ := json.Marshal(snap)
	_, err := s.db.DB.Exec(`INSERT INTO sync_quarantine(table_name, record_id, reason, payload, created_at)
		VALUES (?,?,?,?,strftime('%Y-%m-%dT%H:%M:%fZ','now'))`,
		table, recordID, reason, string(payload))
	return err
}

// QuarantineList 列出隔离区。
func (s *Service) QuarantineList(limit int) ([]QuarantineEntry, error) {
	if limit <= 0 {
		limit = 100
	}
	rows, err := s.db.DB.Query(`SELECT id, table_name, record_id, reason, payload, created_at
		FROM sync_quarantine ORDER BY id DESC LIMIT ?`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []QuarantineEntry
	for rows.Next() {
		var q QuarantineEntry
		if err := rows.Scan(&q.ID, &q.Table, &q.RecordID, &q.Reason, &q.Payload, &q.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, q)
	}
	return out, rows.Err()
}

// QuarantineCount 隔离区数量。
func (s *Service) QuarantineCount() (int64, error) {
	var c int64
	err := s.db.DB.QueryRow(`SELECT COUNT(*) FROM sync_quarantine`).Scan(&c)
	return c, err
}

// RetryQuarantine 重试一条隔离记录（重新应用原 payload）。
func (s *Service) RetryQuarantine(id int64) error {
	rows, err := s.db.DB.Query(`SELECT table_name, payload FROM sync_quarantine WHERE id=?`, id)
	if err != nil {
		return err
	}
	defer rows.Close()
	if !rows.Next() {
		return fmt.Errorf("sync: quarantine entry not found: %d", id)
	}
	var table, payload string
	if err := rows.Scan(&table, &payload); err != nil {
		return err
	}
	var snap RowSnapshot
	if err := json.Unmarshal([]byte(payload), &snap); err != nil {
		return err
	}
	spec, ok := tableSpec(table)
	if !ok {
		return fmt.Errorf("sync: unknown table %s", table)
	}
	if err := s.applyOne(spec, snap); err != nil {
		return err
	}
	_, err = s.db.DB.Exec(`DELETE FROM sync_quarantine WHERE id=?`, id)
	return err
}

// DiscardQuarantine 丢弃隔离记录。
func (s *Service) DiscardQuarantine(id int64) error {
	_, err := s.db.DB.Exec(`DELETE FROM sync_quarantine WHERE id=?`, id)
	return err
}

// DiscardAllQuarantine 清空隔离区。
func (s *Service) DiscardAllQuarantine() (int64, error) {
	res, err := s.db.DB.Exec(`DELETE FROM sync_quarantine`)
	if err != nil {
		return 0, err
	}
	return res.RowsAffected()
}

// Cursor 返回已同步游标。
func (s *Service) Cursor() (int64, error) {
	var v int64
	err := s.db.DB.QueryRow(`SELECT value FROM sync_state WHERE key='cursor'`).Scan(&v)
	if err != nil {
		return 0, nil // 未初始化
	}
	return v, nil
}

// SetCursor 设置游标。
func (s *Service) SetCursor(seq int64) error {
	_, err := s.db.DB.Exec(`INSERT INTO sync_state(key, value) VALUES ('cursor', ?)
		ON CONFLICT(key) DO UPDATE SET value=excluded.value`, fmt.Sprintf("%d", seq))
	return err
}

// SyncToCloud 与云端同步：上传本地变更 + 下载并应用远端变更。
// 通过 cloudstorage.Backend 交换 <remoteDir>/sync/<deviceID>/changes-<seq>.json。
// 注意：不在此处加锁（ExportChanges/ApplyChanges 内部各自持锁），避免重入死锁。
func (s *Service) SyncToCloud(ctx context.Context, be cloudstorage.Backend, remoteDir, deviceID string) (*SyncOutcome, error) {
	out := &SyncOutcome{}

	cur, _ := s.Cursor()
	// 1. 导出并上传本地变更
	batch, err := s.ExportChanges(deviceID, cur, 500)
	if err != nil {
		return nil, err
	}
	if len(batch.Changes) > 0 {
		data, _ := json.Marshal(batch)
		key := fmt.Sprintf("%s/sync/%s/changes-%d.json", strings.Trim(remoteDir, "/"), deviceID, batch.MaxSeq)
		if err := be.Put(ctx, key, strings.NewReader(string(data)), int64(len(data))); err != nil {
			return nil, err
		}
		out.Uploaded = len(batch.Changes)
		_ = s.SetCursor(batch.MaxSeq)
	}

	// 2. 下载远端（其它设备）变更并应用
	prefix := fmt.Sprintf("%s/sync/", strings.Trim(remoteDir, "/"))
	objects, err := be.List(ctx, prefix)
	if err != nil {
		return nil, err
	}
	for _, obj := range objects {
		if strings.Contains(obj.Key, "/"+deviceID+"/") {
			continue // 跳过自己的
		}
		rc, _, err := be.Get(ctx, obj.Key)
		if err != nil {
			continue
		}
		data, _ := io.ReadAll(rc)
		rc.Close()
		var remote ChangeBatch
		if err := json.Unmarshal(data, &remote); err != nil {
			continue
		}
		applied, quarantined, aerr := s.ApplyChanges(&remote)
		out.Downloaded += applied
		out.Quarantined += quarantined
		if aerr != nil {
			return nil, aerr
		}
	}
	return out, nil
}

// SyncOutcome 同步结果。
type SyncOutcome struct {
	Uploaded    int `json:"uploaded"`
	Downloaded  int `json:"downloaded"`
	Quarantined int `json:"quarantined"`
}

// QuarantineEntry 隔离区记录。
type QuarantineEntry struct {
	ID        int64  `json:"id"`
	Table     string `json:"table"`
	RecordID  string `json:"recordId"`
	Reason    string `json:"reason"`
	Payload   string `json:"payload"`
	CreatedAt string `json:"createdAt"`
}

// tableSpec 按表名查规格。
func tableSpec(name string) (TableSpec, bool) {
	for _, t := range TrackedTables {
		if t.Name == name {
			return t, true
		}
	}
	return TableSpec{}, false
}

func colIndex(cols []string, name string) (int, bool) {
	for i, c := range cols {
		if c == name {
			return i, true
		}
	}
	return 0, false
}

var _ = time.Now
