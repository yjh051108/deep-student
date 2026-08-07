// pomodoro 包的 SQLite 持久化层。
//
// pomodoro_records 表结构对齐 Rust 原版（含枚举/数值校验触发器）。
// 时间字段统一 RFC3339Nano UTC 字符串（带 Z），满足云同步时间基准要求。

package pomodoro

import (
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/helixnow/deep-student-go/pkg/store"
)

// Store pomodoro 包专用存储层。
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

// Migrate 创建 pomodoro_records 表与索引（幂等）。
func (s *Store) Migrate() error {
	if s.db == nil {
		return errors.New("pomodoro: nil db")
	}
	stmts := []string{
		`CREATE TABLE IF NOT EXISTS pomodoro_records (
			id TEXT PRIMARY KEY NOT NULL,
			todo_item_id TEXT,
			start_time TEXT NOT NULL,
			end_time TEXT,
			duration INTEGER NOT NULL,
			actual_duration INTEGER NOT NULL DEFAULT 0,
			type TEXT NOT NULL DEFAULT 'work',
			status TEXT NOT NULL DEFAULT 'completed',
			created_at TEXT NOT NULL
		)`,
		`CREATE INDEX IF NOT EXISTS idx_pomodoro_item ON pomodoro_records(todo_item_id)`,
		`CREATE INDEX IF NOT EXISTS idx_pomodoro_type ON pomodoro_records(type)`,
		`CREATE INDEX IF NOT EXISTS idx_pomodoro_status ON pomodoro_records(status)`,
		`CREATE INDEX IF NOT EXISTS idx_pomodoro_created ON pomodoro_records(created_at DESC)`,
		// 枚举/数值校验（对齐 Rust V20260613 触发器）
		`CREATE TRIGGER IF NOT EXISTS trg_pomodoro_records_validate_insert
		 BEFORE INSERT ON pomodoro_records
		 FOR EACH ROW
		 BEGIN
			SELECT RAISE(ABORT, 'pomodoro_records.type is invalid')
			WHERE NEW.type NOT IN ('work', 'short_break', 'long_break');
			SELECT RAISE(ABORT, 'pomodoro_records.status is invalid')
			WHERE NEW.status NOT IN ('completed', 'interrupted');
			SELECT RAISE(ABORT, 'pomodoro_records.duration must be positive')
			WHERE NEW.duration <= 0;
			SELECT RAISE(ABORT, 'pomodoro_records.actual_duration must be non-negative')
			WHERE NEW.actual_duration < 0;
		 END`,
	}
	for _, stmt := range stmts {
		if _, err := s.db.Exec(stmt); err != nil {
			return fmt.Errorf("pomodoro migrate: %w", err)
		}
	}
	return nil
}

// CreateRecord 插入记录。
func (s *Store) CreateRecord(r *Record) error {
	if s.db == nil {
		return errors.New("pomodoro: nil db")
	}
	_, err := s.db.Exec(`INSERT INTO pomodoro_records
		(id, todo_item_id, start_time, end_time, duration, actual_duration, type, status, created_at)
		VALUES (?,?,?,?,?,?,?,?,?)`,
		r.ID, nullableStr(r.TodoItemID), formatTime(r.StartTime), nullableTime(r.EndTime),
		r.Duration, r.ActualDuration, string(r.Type), string(r.Status), formatTime(r.CreatedAt),
	)
	if err != nil {
		return fmt.Errorf("pomodoro: create record: %w", err)
	}
	return nil
}

// GetRecord 按 ID 读取。
func (s *Store) GetRecord(id string) (*Record, error) {
	if s.db == nil {
		return nil, errors.New("pomodoro: nil db")
	}
	row := s.db.QueryRow(recordSelectCols+` FROM pomodoro_records WHERE id=?`, id)
	return scanRecord(row)
}

// ListByTodo 列出关联某条待办的记录。
func (s *Store) ListByTodo(todoItemID string) ([]Record, error) {
	if s.db == nil {
		return nil, errors.New("pomodoro: nil db")
	}
	rows, err := s.db.Query(
		recordSelectCols+` FROM pomodoro_records WHERE todo_item_id=? ORDER BY start_time DESC`, todoItemID)
	if err != nil {
		return nil, fmt.Errorf("pomodoro: list by todo: %w", err)
	}
	defer rows.Close()
	return scanRecords(rows)
}

// ListBetween 查询 [start, end] 时间范围内的记录（按 start_time）。
func (s *Store) ListBetween(start, end *time.Time) ([]Record, error) {
	if s.db == nil {
		return nil, errors.New("pomodoro: nil db")
	}
	var clauses []string
	var args []interface{}
	if start != nil {
		clauses = append(clauses, "datetime(start_time) >= datetime(?)")
		args = append(args, formatTime(*start))
	}
	if end != nil {
		clauses = append(clauses, "datetime(start_time) <= datetime(?)")
		args = append(args, formatTime(*end))
	}
	query := recordSelectCols + ` FROM pomodoro_records`
	if len(clauses) > 0 {
		query += ` WHERE ` + strings.Join(clauses, " AND ")
	}
	query += ` ORDER BY start_time DESC`
	rows, err := s.db.Query(query, args...)
	if err != nil {
		return nil, fmt.Errorf("pomodoro: list between: %w", err)
	}
	defer rows.Close()
	return scanRecords(rows)
}

// DailyAggregate 按日聚合实际专注秒数与记录数。
func (s *Store) DailyAggregate(days int) ([]DailyStat, error) {
	if s.db == nil {
		return nil, errors.New("pomodoro: nil db")
	}
	if days <= 0 {
		days = 7
	}
	rows, err := s.db.Query(
		`SELECT substr(start_time, 1, 10) AS day, SUM(actual_duration), COUNT(*)
		 FROM pomodoro_records
		 WHERE type='work' AND status='completed'
		   AND start_time >= strftime('%Y-%m-%dT%H:%M:%fZ', 'now', ?)
		 GROUP BY day ORDER BY day DESC LIMIT ?`,
		fmt.Sprintf("-%d days", days-1), days,
	)
	if err != nil {
		return nil, fmt.Errorf("pomodoro: daily aggregate: %w", err)
	}
	defer rows.Close()
	var out []DailyStat
	for rows.Next() {
		var d DailyStat
		var seconds sql.NullInt64
		if err := rows.Scan(&d.Date, &seconds, &d.Count); err != nil {
			return nil, err
		}
		d.TotalSeconds = int(seconds.Int64)
		out = append(out, d)
	}
	return out, rows.Err()
}

const recordSelectCols = `SELECT id, todo_item_id, start_time, end_time, duration,
	actual_duration, type, status, created_at`

type scanner interface {
	Scan(dest ...interface{}) error
}

func scanRecord(row scanner) (*Record, error) {
	var r Record
	var todoID, endTime, startTime, createdAt sql.NullString
	var typ, status string
	err := row.Scan(&r.ID, &todoID, &startTime, &endTime, &r.Duration,
		&r.ActualDuration, &typ, &status, &createdAt)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, fmt.Errorf("pomodoro: record not found")
		}
		return nil, fmt.Errorf("pomodoro: scan record: %w", err)
	}
	if todoID.Valid {
		id := todoID.String
		r.TodoItemID = &id
	}
	r.StartTime = parseTime(startTime.String)
	r.EndTime = parseNullableTime(endTime)
	r.Type = RecordType(typ)
	r.Status = RecordStatus(status)
	r.CreatedAt = parseTime(createdAt.String)
	return &r, nil
}

func scanRecords(rows *sql.Rows) ([]Record, error) {
	var out []Record
	for rows.Next() {
		r, err := scanRecord(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, *r)
	}
	return out, rows.Err()
}

// ===================== 辅助 =====================

func nullableStr(s *string) sql.NullString {
	if s == nil {
		return sql.NullString{}
	}
	return sql.NullString{String: *s, Valid: true}
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
