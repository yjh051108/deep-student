// llmusage 包的 SQLite 持久化层。
//
// llm_usage_logs / llm_usage_daily 表结构对齐 Rust 原版
// llm_usage/V20260130__init.sql。写入日志时自动 UPSERT 当日聚合行。

package llmusage

import (
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/helixnow/deep-student-go/pkg/store"
)

// Store llmusage 包专用存储层。
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

// Migrate 创建 llm_usage 相关表与索引（幂等）。
func (s *Store) Migrate() error {
	if s.db == nil {
		return errors.New("llmusage: nil db")
	}
	stmts := []string{
		`CREATE TABLE IF NOT EXISTS llm_usage_logs (
			id TEXT PRIMARY KEY,
			timestamp TEXT NOT NULL,
			provider TEXT NOT NULL,
			model TEXT NOT NULL,
			adapter TEXT,
			api_config_id TEXT,
			prompt_tokens INTEGER NOT NULL DEFAULT 0,
			completion_tokens INTEGER NOT NULL DEFAULT 0,
			total_tokens INTEGER NOT NULL DEFAULT 0,
			reasoning_tokens INTEGER,
			cached_tokens INTEGER,
			token_source TEXT NOT NULL DEFAULT 'api',
			duration_ms INTEGER,
			request_bytes INTEGER,
			response_bytes INTEGER,
			first_token_ms INTEGER,
			caller_type TEXT NOT NULL,
			session_id TEXT,
			status TEXT NOT NULL DEFAULT 'success',
			error_message TEXT,
			cost_estimate REAL,
			date_key TEXT GENERATED ALWAYS AS (substr(timestamp, 1, 10)) STORED,
			hour_key TEXT GENERATED ALWAYS AS (substr(timestamp, 1, 13)) STORED
		)`,
		`CREATE INDEX IF NOT EXISTS idx_llm_usage_logs_timestamp ON llm_usage_logs(timestamp DESC)`,
		`CREATE INDEX IF NOT EXISTS idx_llm_usage_logs_date_key ON llm_usage_logs(date_key)`,
		`CREATE INDEX IF NOT EXISTS idx_llm_usage_logs_caller_type ON llm_usage_logs(caller_type)`,
		`CREATE INDEX IF NOT EXISTS idx_llm_usage_logs_model ON llm_usage_logs(model)`,
		`CREATE INDEX IF NOT EXISTS idx_llm_usage_logs_provider ON llm_usage_logs(provider)`,
		`CREATE INDEX IF NOT EXISTS idx_llm_usage_logs_status ON llm_usage_logs(status)`,
		`CREATE INDEX IF NOT EXISTS idx_llm_usage_logs_session_id ON llm_usage_logs(session_id) WHERE session_id IS NOT NULL`,
		`CREATE TABLE IF NOT EXISTS llm_usage_daily (
			date TEXT NOT NULL,
			caller_type TEXT NOT NULL,
			model TEXT NOT NULL,
			provider TEXT NOT NULL,
			request_count INTEGER NOT NULL DEFAULT 0,
			success_count INTEGER NOT NULL DEFAULT 0,
			error_count INTEGER NOT NULL DEFAULT 0,
			total_prompt_tokens INTEGER NOT NULL DEFAULT 0,
			total_completion_tokens INTEGER NOT NULL DEFAULT 0,
			total_tokens INTEGER NOT NULL DEFAULT 0,
			total_reasoning_tokens INTEGER DEFAULT 0,
			total_cached_tokens INTEGER DEFAULT 0,
			total_cost_estimate REAL DEFAULT 0.0,
			total_duration_ms INTEGER DEFAULT 0,
			created_at TEXT NOT NULL DEFAULT (datetime('now')),
			updated_at TEXT NOT NULL DEFAULT (datetime('now')),
			PRIMARY KEY (date, caller_type, model, provider)
		)`,
		`CREATE INDEX IF NOT EXISTS idx_llm_usage_daily_date ON llm_usage_daily(date DESC)`,
	}
	for _, stmt := range stmts {
		if _, err := s.db.Exec(stmt); err != nil {
			return fmt.Errorf("llmusage migrate: %w", err)
		}
	}
	return nil
}

// InsertLog 写入调用日志并同步更新当日聚合。
func (s *Store) InsertLog(l *Log) error {
	if s.db == nil {
		return errors.New("llmusage: nil db")
	}
	tx, err := s.db.Begin()
	if err != nil {
		return fmt.Errorf("llmusage: begin tx: %w", err)
	}
	_, err = tx.Exec(`INSERT INTO llm_usage_logs
		(id, timestamp, provider, model, adapter, api_config_id, prompt_tokens,
		 completion_tokens, total_tokens, reasoning_tokens, cached_tokens, token_source,
		 duration_ms, request_bytes, response_bytes, first_token_ms, caller_type,
		 session_id, status, error_message, cost_estimate)
		VALUES (?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?)`,
		l.ID, formatTime(l.Timestamp), l.Provider, l.Model, nullableStr(l.Adapter),
		nullableStr(l.APIConfigID), l.PromptTokens, l.CompletionTokens, l.TotalTokens,
		nullableInt(l.ReasoningTokens), nullableInt(l.CachedTokens), string(l.TokenSource),
		nullableInt64(l.DurationMs), nullableInt64(l.RequestBytes), nullableInt64(l.ResponseBytes),
		nullableInt64(l.FirstTokenMs), l.CallerType, nullableStr(l.SessionID),
		string(l.Status), nullableStr(l.ErrorMessage), nullableFloat(l.CostEstimate),
	)
	if err != nil {
		_ = tx.Rollback()
		return fmt.Errorf("llmusage: insert log: %w", err)
	}
	date := l.Timestamp.UTC().Format("2006-01-02")
	success := 0
	errorCount := 0
	if l.Status == StatusSuccess {
		success = 1
	} else {
		errorCount = 1
	}
	cost := 0.0
	if l.CostEstimate != nil {
		cost = *l.CostEstimate
	}
	_, err = tx.Exec(`INSERT INTO llm_usage_daily
		(date, caller_type, model, provider, request_count, success_count, error_count,
		 total_prompt_tokens, total_completion_tokens, total_tokens,
		 total_reasoning_tokens, total_cached_tokens, total_cost_estimate, total_duration_ms)
		VALUES (?,?,?,?,1,?,?,?,?,?,?,?,?,?)
		ON CONFLICT(date, caller_type, model, provider) DO UPDATE SET
			request_count = request_count + 1,
			success_count = success_count + excluded.success_count,
			error_count = error_count + excluded.error_count,
			total_prompt_tokens = total_prompt_tokens + excluded.total_prompt_tokens,
			total_completion_tokens = total_completion_tokens + excluded.total_completion_tokens,
			total_tokens = total_tokens + excluded.total_tokens,
			total_reasoning_tokens = total_reasoning_tokens + excluded.total_reasoning_tokens,
			total_cached_tokens = total_cached_tokens + excluded.total_cached_tokens,
			total_cost_estimate = total_cost_estimate + excluded.total_cost_estimate,
			total_duration_ms = total_duration_ms + excluded.total_duration_ms,
			updated_at = datetime('now')`,
		date, l.CallerType, l.Model, l.Provider, success, errorCount,
		l.PromptTokens, l.CompletionTokens, l.TotalTokens,
		nullableIntOrZero(l.ReasoningTokens), nullableIntOrZero(l.CachedTokens),
		cost, nullableInt64OrZero(l.DurationMs),
	)
	if err != nil {
		_ = tx.Rollback()
		return fmt.Errorf("llmusage: upsert daily: %w", err)
	}
	return tx.Commit()
}

// QueryLogs 查询日志（按时间倒序，支持 provider/model/caller/status 过滤）。
func (s *Store) QueryLogs(filter LogFilter) ([]Log, error) {
	if s.db == nil {
		return nil, errors.New("llmusage: nil db")
	}
	var clauses []string
	var args []interface{}
	if filter.Provider != "" {
		clauses = append(clauses, "provider = ?")
		args = append(args, filter.Provider)
	}
	if filter.Model != "" {
		clauses = append(clauses, "model = ?")
		args = append(args, filter.Model)
	}
	if filter.CallerType != "" {
		clauses = append(clauses, "caller_type = ?")
		args = append(args, filter.CallerType)
	}
	if filter.Status != "" {
		clauses = append(clauses, "status = ?")
		args = append(args, string(filter.Status))
	}
	if filter.Since != nil {
		clauses = append(clauses, "timestamp >= ?")
		args = append(args, formatTime(*filter.Since))
	}
	if filter.Until != nil {
		clauses = append(clauses, "timestamp <= ?")
		args = append(args, formatTime(*filter.Until))
	}
	limit := filter.Limit
	if limit <= 0 {
		limit = 100
	}
	query := `SELECT id, timestamp, provider, model, adapter, api_config_id, prompt_tokens,
		completion_tokens, total_tokens, reasoning_tokens, cached_tokens, token_source,
		duration_ms, request_bytes, response_bytes, first_token_ms, caller_type,
		session_id, status, error_message, cost_estimate FROM llm_usage_logs`
	if len(clauses) > 0 {
		query += ` WHERE ` + strings.Join(clauses, " AND ")
	}
	query += ` ORDER BY timestamp DESC LIMIT ?`
	args = append(args, limit)
	rows, err := s.db.Query(query, args...)
	if err != nil {
		return nil, fmt.Errorf("llmusage: query logs: %w", err)
	}
	defer rows.Close()
	return scanLogs(rows)
}

// QueryDaily 查询按日聚合（支持 caller/model 过滤与日期范围）。
func (s *Store) QueryDaily(filter DailyFilter) ([]DailyAggregate, error) {
	if s.db == nil {
		return nil, errors.New("llmusage: nil db")
	}
	var clauses []string
	var args []interface{}
	if filter.DateStart != "" {
		clauses = append(clauses, "date >= ?")
		args = append(args, filter.DateStart)
	}
	if filter.DateEnd != "" {
		clauses = append(clauses, "date <= ?")
		args = append(args, filter.DateEnd)
	}
	if filter.CallerType != "" {
		clauses = append(clauses, "caller_type = ?")
		args = append(args, filter.CallerType)
	}
	if filter.Model != "" {
		clauses = append(clauses, "model = ?")
		args = append(args, filter.Model)
	}
	query := `SELECT date, caller_type, model, provider, request_count, success_count, error_count,
		total_prompt_tokens, total_completion_tokens, total_tokens, total_reasoning_tokens,
		total_cached_tokens, total_cost_estimate, total_duration_ms FROM llm_usage_daily`
	if len(clauses) > 0 {
		query += ` WHERE ` + strings.Join(clauses, " AND ")
	}
	query += ` ORDER BY date DESC, total_tokens DESC`
	rows, err := s.db.Query(query, args...)
	if err != nil {
		return nil, fmt.Errorf("llmusage: query daily: %w", err)
	}
	defer rows.Close()
	var out []DailyAggregate
	for rows.Next() {
		var d DailyAggregate
		if err := rows.Scan(&d.Date, &d.CallerType, &d.Model, &d.Provider,
			&d.RequestCount, &d.SuccessCount, &d.ErrorCount,
			&d.TotalPromptTokens, &d.TotalCompletionTokens, &d.TotalTokens,
			&d.TotalReasoningTokens, &d.TotalCachedTokens, &d.TotalCostEstimate,
			&d.TotalDurationMs); err != nil {
			return nil, fmt.Errorf("llmusage: scan daily: %w", err)
		}
		out = append(out, d)
	}
	return out, rows.Err()
}

// Aggregate 汇总（总请求/总 token/今日/近 7 天）。
func (s *Store) Aggregate() (*Summary, error) {
	if s.db == nil {
		return nil, errors.New("llmusage: nil db")
	}
	sum := &Summary{}
	err := s.db.QueryRow(`SELECT COUNT(*), COALESCE(SUM(total_tokens),0),
		COALESCE(SUM(prompt_tokens),0), COALESCE(SUM(completion_tokens),0),
		COALESCE(SUM(cost_estimate),0) FROM llm_usage_logs`).
		Scan(&sum.TotalRequests, &sum.TotalTokens, &sum.TotalPromptTokens, &sum.TotalCompletionTokens, &sum.TotalCost)
	if err != nil {
		return nil, fmt.Errorf("llmusage: aggregate total: %w", err)
	}
	today := time.Now().UTC().Format("2006-01-02")
	err = s.db.QueryRow(`SELECT COUNT(*), COALESCE(SUM(total_tokens),0) FROM llm_usage_logs WHERE date_key=?`, today).
		Scan(&sum.TodayRequests, &sum.TodayTokens)
	if err != nil {
		return nil, fmt.Errorf("llmusage: aggregate today: %w", err)
	}
	weekStart := time.Now().UTC().AddDate(0, 0, -6).Format("2006-01-02")
	err = s.db.QueryRow(`SELECT COUNT(*), COALESCE(SUM(total_tokens),0) FROM llm_usage_logs WHERE date_key >= ?`, weekStart).
		Scan(&sum.Last7DaysRequests, &sum.Last7DaysTokens)
	if err != nil {
		return nil, fmt.Errorf("llmusage: aggregate week: %w", err)
	}
	return sum, nil
}

// DeleteOlderThan 清理指定日期之前的日志，返回删除数量。
func (s *Store) DeleteOlderThan(before time.Time) (int64, error) {
	if s.db == nil {
		return 0, errors.New("llmusage: nil db")
	}
	res, err := s.db.Exec(`DELETE FROM llm_usage_logs WHERE timestamp < ?`, formatTime(before))
	if err != nil {
		return 0, fmt.Errorf("llmusage: delete older: %w", err)
	}
	n, _ := res.RowsAffected()
	return n, nil
}

// ===================== 查询参数 / 扫描 =====================

// LogFilter 日志查询过滤。
type LogFilter struct {
	Provider   string    `json:"provider,omitempty"`
	Model      string    `json:"model,omitempty"`
	CallerType string    `json:"callerType,omitempty"`
	Status     Status    `json:"status,omitempty"`
	Since      *time.Time `json:"since,omitempty"`
	Until      *time.Time `json:"until,omitempty"`
	Limit      int       `json:"limit,omitempty"`
}

// DailyFilter 按日聚合查询过滤。
type DailyFilter struct {
	DateStart  string `json:"dateStart,omitempty"`
	DateEnd    string `json:"dateEnd,omitempty"`
	CallerType string `json:"callerType,omitempty"`
	Model      string `json:"model,omitempty"`
}

type scanner interface {
	Scan(dest ...interface{}) error
}

func scanLog(row scanner) (*Log, error) {
	var l Log
	var adapter, apiConfigID, sessionID, errMsg sql.NullString
	var reasoning, cached sql.NullInt64
	var duration, reqBytes, respBytes, firstToken sql.NullInt64
	var cost sql.NullFloat64
	var ts, tokenSource, status string
	err := row.Scan(&l.ID, &ts, &l.Provider, &l.Model, &adapter, &apiConfigID,
		&l.PromptTokens, &l.CompletionTokens, &l.TotalTokens, &reasoning, &cached,
		&tokenSource, &duration, &reqBytes, &respBytes, &firstToken,
		&l.CallerType, &sessionID, &status, &errMsg, &cost)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, fmt.Errorf("llmusage: log not found")
		}
		return nil, fmt.Errorf("llmusage: scan log: %w", err)
	}
	l.Timestamp = parseTime(ts)
	l.Adapter = adapter.String
	l.APIConfigID = apiConfigID.String
	l.TokenSource = TokenSource(tokenSource)
	if reasoning.Valid {
		v := int(reasoning.Int64)
		l.ReasoningTokens = &v
	}
	if cached.Valid {
		v := int(cached.Int64)
		l.CachedTokens = &v
	}
	l.DurationMs = &duration.Int64
	l.RequestBytes = &reqBytes.Int64
	l.ResponseBytes = &respBytes.Int64
	l.FirstTokenMs = &firstToken.Int64
	l.SessionID = sessionID.String
	l.Status = Status(status)
	l.ErrorMessage = errMsg.String
	if cost.Valid {
		l.CostEstimate = &cost.Float64
	}
	return &l, nil
}

func scanLogs(rows *sql.Rows) ([]Log, error) {
	var out []Log
	for rows.Next() {
		l, err := scanLog(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, *l)
	}
	return out, rows.Err()
}

// ===================== 辅助 =====================

func nullableStr(s string) sql.NullString {
	if s == "" {
		return sql.NullString{}
	}
	return sql.NullString{String: s, Valid: true}
}

func nullableInt(p *int) sql.NullInt64 {
	if p == nil {
		return sql.NullInt64{}
	}
	return sql.NullInt64{Int64: int64(*p), Valid: true}
}

func nullableInt64(p *int64) sql.NullInt64 {
	if p == nil {
		return sql.NullInt64{}
	}
	return sql.NullInt64{Int64: *p, Valid: true}
}

func nullableInt64OrZero(p *int64) int64 {
	if p == nil {
		return 0
	}
	return *p
}

func nullableIntOrZero(p *int) int {
	if p == nil {
		return 0
	}
	return *p
}

func nullableFloat(p *float64) sql.NullFloat64 {
	if p == nil {
		return sql.NullFloat64{}
	}
	return sql.NullFloat64{Float64: *p, Valid: true}
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
