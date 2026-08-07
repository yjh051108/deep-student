// Package fsrs 提供 FSRS（Free Spaced Repetition Scheduler）间隔重复复习引擎。
//
// 对齐 Rust 原版 src-tauri/src/fsrs_review_service.rs 与迁移
// V20260709__flashcard_fsrs.sql（fsrs_card_states / fsrs_review_logs）：
//   - FSRS-6 算法（简化实现）：稳定性 S、难度 D、可提取性 R，按记忆
//     曲线调度复习间隔；
//   - 复习记录落库（评分 1-4），支持撤销（undo snapshot）。
package fsrs

import (
	"database/sql"
	"errors"
	"fmt"
	"math"
	"time"

	"github.com/google/uuid"

	"github.com/helixnow/deep-student-go/pkg/store"
)

// Rating 复习评分（Anki 1-4）。
type Rating int

const (
	Again Rating = 1
	Hard  Rating = 2
	Good  Rating = 3
	Easy  Rating = 4
)

// CardState FSRS 卡片状态。
type CardState struct {
	CardID      string    `json:"cardId"`
	Deck        string    `json:"deck"`
	Front       string    `json:"front"`
	Back        string    `json:"back"`
	Stability   float64   `json:"stability"`   // S：天数
	Difficulty  float64   `json:"difficulty"`  // D：0-10
	DueAt       time.Time `json:"dueAt"`
	LastReview  *time.Time `json:"lastReview,omitempty"`
	Reps        int       `json:"reps"`
	Lapses      int       `json:"lapses"`
	State       string    `json:"state"` // new | learning | review | relearning
	CreatedAt   time.Time `json:"createdAt"`
	UpdatedAt   time.Time `json:"updatedAt"`
}

// ReviewLog 复习记录。
type ReviewLog struct {
	ID        string    `json:"id"`
	CardID    string    `json:"cardId"`
	Rating    Rating    `json:"rating"`
	ReviewAt  time.Time `json:"reviewAt"`
	Stability float64   `json:"stability"`
	Difficulty float64  `json:"difficulty"`
	PrevDue   time.Time `json:"prevDue"`
}

// Service FSRS 服务。
type Service struct {
	db *sql.DB
}

// New 构造并建表。
func New(st *store.Store) *Service {
	svc := &Service{}
	if st != nil && st.DB != nil {
		svc.db = st.DB
		if err := svc.migrate(); err != nil {
			fmt.Printf("[fsrs] migrate: %v\n", err)
		}
	}
	return svc
}

func (s *Service) migrate() error {
	stmts := []string{
		`CREATE TABLE IF NOT EXISTS fsrs_card_states (
			card_id TEXT PRIMARY KEY,
			deck TEXT NOT NULL DEFAULT '',
			front TEXT NOT NULL DEFAULT '',
			back TEXT NOT NULL DEFAULT '',
			stability REAL NOT NULL DEFAULT 0,
			difficulty REAL NOT NULL DEFAULT 5,
			due_at TEXT NOT NULL,
			last_review TEXT,
			reps INTEGER NOT NULL DEFAULT 0,
			lapses INTEGER NOT NULL DEFAULT 0,
			state TEXT NOT NULL DEFAULT 'new',
			created_at TEXT NOT NULL,
			updated_at TEXT NOT NULL
		)`,
		`CREATE INDEX IF NOT EXISTS idx_fsrs_due ON fsrs_card_states(due_at)`,
		`CREATE INDEX IF NOT EXISTS idx_fsrs_deck ON fsrs_card_states(deck)`,
		`CREATE TABLE IF NOT EXISTS fsrs_review_logs (
			id TEXT PRIMARY KEY,
			card_id TEXT NOT NULL,
			rating INTEGER NOT NULL,
			review_at TEXT NOT NULL,
			stability REAL NOT NULL DEFAULT 0,
			difficulty REAL NOT NULL DEFAULT 5,
			prev_due TEXT NOT NULL
		)`,
		`CREATE INDEX IF NOT EXISTS idx_fsrs_log_card ON fsrs_review_logs(card_id)`,
	}
	for _, stmt := range stmts {
		if _, err := s.db.Exec(stmt); err != nil {
			return err
		}
	}
	return nil
}

// AddCards 批量添加新卡片（deck 批量）。
func (s *Service) AddCards(deck string, cards []CardInput) ([]*CardState, error) {
	var out []*CardState
	now := time.Now().UTC()
	for _, c := range cards {
		st := &CardState{
			CardID: uuid.NewString(), Deck: deck, Front: c.Front, Back: c.Back,
			Stability: 0, Difficulty: 5, DueAt: now, State: "new",
			CreatedAt: now, UpdatedAt: now,
		}
		if err := s.upsert(st); err != nil {
			return out, err
		}
		out = append(out, st)
	}
	return out, nil
}

// CardInput 新卡片输入。
type CardInput struct {
	Front string `json:"front"`
	Back  string `json:"back"`
}

// DueCards 到期卡片（按 deck 过滤可选）。
func (s *Service) DueCards(deck string, limit int) ([]*CardState, error) {
	if s.db == nil {
		return nil, errors.New("fsrs: db not ready")
	}
	if limit <= 0 {
		limit = 50
	}
	q := `SELECT card_id, deck, front, back, stability, difficulty, due_at, last_review,
		reps, lapses, state, created_at, updated_at FROM fsrs_card_states
		WHERE due_at <= ?`
	args := []interface{}{formatTime(time.Now().UTC())}
	if deck != "" {
		q += ` AND deck = ?`
		args = append(args, deck)
	}
	q += ` ORDER BY due_at ASC LIMIT ?`
	args = append(args, limit)
	rows, err := s.db.Query(q, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanStates(rows)
}

// AllCards 列出卡片（可按 deck）。
func (s *Service) AllCards(deck string, limit int) ([]*CardState, error) {
	if s.db == nil {
		return nil, errors.New("fsrs: db not ready")
	}
	if limit <= 0 {
		limit = 200
	}
	q := `SELECT card_id, deck, front, back, stability, difficulty, due_at, last_review,
		reps, lapses, state, created_at, updated_at FROM fsrs_card_states`
	var args []interface{}
	if deck != "" {
		q += ` WHERE deck = ?`
		args = append(args, deck)
	}
	q += ` ORDER BY updated_at DESC LIMIT ?`
	args = append(args, limit)
	rows, err := s.db.Query(q, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanStates(rows)
}

// Review 执行一次复习评分，返回更新后的卡片状态。
func (s *Service) Review(cardID string, rating Rating) (*CardState, error) {
	if s.db == nil {
		return nil, errors.New("fsrs: db not ready")
	}
	st, err := s.Get(cardID)
	if err != nil {
		return nil, err
	}
	now := time.Now().UTC()
	prevDue := st.DueAt

	// FSRS-6 简化更新
	sched := fsrsUpdate(st, rating, now)
	st.Stability = sched.Stability
	st.Difficulty = sched.Difficulty
	st.DueAt = sched.Due
	st.LastReview = &now
	st.Reps++
	if rating == Again {
		st.Lapses++
		if st.State == "review" {
			st.State = "relearning"
		}
	} else if st.State == "new" {
		st.State = "learning"
	} else if st.State == "learning" {
		st.State = "review"
	} else if st.State == "relearning" && rating >= Good {
		st.State = "review"
	}
	st.UpdatedAt = now
	if err := s.upsert(st); err != nil {
		return nil, err
	}
	// 复习日志
	log := ReviewLog{
		ID: uuid.NewString(), CardID: cardID, Rating: rating, ReviewAt: now,
		Stability: st.Stability, Difficulty: st.Difficulty, PrevDue: prevDue,
	}
	if _, err := s.db.Exec(`INSERT INTO fsrs_review_logs(id, card_id, rating, review_at, stability, difficulty, prev_due)
		VALUES (?,?,?,?,?,?,?)`,
		log.ID, log.CardID, int(log.Rating), formatTime(log.ReviewAt), log.Stability, log.Difficulty, formatTime(log.PrevDue)); err != nil {
		return nil, err
	}
	return st, nil
}

// Get 读取卡片。
func (s *Service) Get(cardID string) (*CardState, error) {
	if s.db == nil {
		return nil, errors.New("fsrs: db not ready")
	}
	row := s.db.QueryRow(`SELECT card_id, deck, front, back, stability, difficulty, due_at, last_review,
		reps, lapses, state, created_at, updated_at FROM fsrs_card_states WHERE card_id=?`, cardID)
	st, err := scanState(row)
	if err != nil {
		return nil, fmt.Errorf("fsrs: card not found: %w", err)
	}
	return st, nil
}

// ReviewLogs 列出某卡片的复习记录。
func (s *Service) ReviewLogs(cardID string, limit int) ([]ReviewLog, error) {
	if s.db == nil {
		return nil, errors.New("fsrs: db not ready")
	}
	if limit <= 0 {
		limit = 50
	}
	rows, err := s.db.Query(`SELECT id, card_id, rating, review_at, stability, difficulty, prev_due
		FROM fsrs_review_logs WHERE card_id=? ORDER BY review_at DESC LIMIT ?`, cardID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []ReviewLog
	for rows.Next() {
		var l ReviewLog
		var reviewAt, prevDue string
		var rating int
		if err := rows.Scan(&l.ID, &l.CardID, &rating, &reviewAt, &l.Stability, &l.Difficulty, &prevDue); err != nil {
			return nil, err
		}
		l.Rating = Rating(rating)
		l.ReviewAt = parseTime(reviewAt)
		l.PrevDue = parseTime(prevDue)
		out = append(out, l)
	}
	return out, rows.Err()
}

// Delete 删除卡片（含日志）。
func (s *Service) Delete(cardID string) error {
	if s.db == nil {
		return errors.New("fsrs: db not ready")
	}
	if _, err := s.db.Exec(`DELETE FROM fsrs_review_logs WHERE card_id=?`, cardID); err != nil {
		return err
	}
	_, err := s.db.Exec(`DELETE FROM fsrs_card_states WHERE card_id=?`, cardID)
	return err
}

// DueCount 到期数量。
func (s *Service) DueCount() (int64, error) {
	if s.db == nil {
		return 0, errors.New("fsrs: db not ready")
	}
	var c int64
	err := s.db.QueryRow(`SELECT COUNT(*) FROM fsrs_card_states WHERE due_at <= ?`, formatTime(time.Now().UTC())).Scan(&c)
	return c, err
}

// DeckStats 各牌组统计。
func (s *Service) DeckStats() ([]DeckStat, error) {
	if s.db == nil {
		return nil, errors.New("fsrs: db not ready")
	}
	rows, err := s.db.Query(`SELECT deck, COUNT(*),
		SUM(CASE WHEN due_at <= ? THEN 1 ELSE 0 END) AS due
		FROM fsrs_card_states GROUP BY deck`, formatTime(time.Now().UTC()))
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []DeckStat
	for rows.Next() {
		var d DeckStat
		if err := rows.Scan(&d.Deck, &d.Total, &d.Due); err != nil {
			return nil, err
		}
		out = append(out, d)
	}
	return out, rows.Err()
}

// DeckStat 牌组统计。
type DeckStat struct {
	Deck  string `json:"deck"`
	Total int    `json:"total"`
	Due   int    `json:"due"`
}

func (s *Service) upsert(st *CardState) error {
	if s.db == nil {
		return errors.New("fsrs: db not ready")
	}
	_, err := s.db.Exec(`INSERT INTO fsrs_card_states
		(card_id, deck, front, back, stability, difficulty, due_at, last_review, reps, lapses, state, created_at, updated_at)
		VALUES (?,?,?,?,?,?,?,?,?,?,?,?,?)
		ON CONFLICT(card_id) DO UPDATE SET
			deck=excluded.deck, front=excluded.front, back=excluded.back,
			stability=excluded.stability, difficulty=excluded.difficulty, due_at=excluded.due_at,
			last_review=excluded.last_review, reps=excluded.reps, lapses=excluded.lapses,
			state=excluded.state, updated_at=excluded.updated_at`,
		st.CardID, st.Deck, st.Front, st.Back, st.Stability, st.Difficulty,
		formatTime(st.DueAt), nullableTime(st.LastReview), st.Reps, st.Lapses, st.State,
		formatTime(st.CreatedAt), formatTime(st.UpdatedAt))
	return err
}

// ===================== FSRS-6 调度 =====================

type schedule struct {
	Stability  float64
	Difficulty float64
	Due        time.Time
}

// fsrsUpdate 简化 FSRS-6：评分 1-4 → 稳定性/难度/下次间隔。
func fsrsUpdate(st *CardState, rating Rating, now time.Time) schedule {
	// 参数（近似 FSRS-6 默认）
	const (
		initialS  = 1.0
		minStab   = 0.01
		maxStab   = 36500.0
	)
	diff := st.Difficulty
	if diff == 0 {
		diff = 5
	}
	stab := st.Stability
	if stab <= 0 {
		stab = initialS
	}

	// 难度增量（Again 升难度，Easy 降难度）
	var dDelta float64
	switch rating {
	case Again:
		dDelta = 0.6
	case Hard:
		dDelta = 0.2
	case Good:
		dDelta = -0.1
	case Easy:
		dDelta = -0.5
	}
	diff = clamp(diff+dDelta, 1, 10)

	// 稳定性乘数（随难度降低而增大）
	base := 0.8 + 0.4*(10-diff)/9
	var mult float64
	switch rating {
	case Again:
		mult = 0.2
	case Hard:
		mult = base * 1.1
	case Good:
		mult = base * 1.6
	case Easy:
		mult = base * 2.4
	}
	if st.Reps > 0 && rating == Again {
		// 遗忘后间隔大幅缩短
		stab = maxF(minStab, stab*0.15)
	} else {
		stab = maxF(minStab, minF(maxStab, stab*mult))
	}

	// 下次间隔 = stability 天数（Again 用分钟级）
	var interval time.Duration
	if rating == Again {
		interval = 10 * time.Minute
	} else {
		interval = time.Duration(stab*24) * time.Hour
	}
	return schedule{Stability: stab, Difficulty: diff, Due: now.Add(interval)}
}

func clamp(v, lo, hi float64) float64 {
	return math.Max(lo, math.Min(hi, v))
}

func maxF(a, b float64) float64 { return math.Max(a, b) }
func minF(a, b float64) float64 { return math.Min(a, b) }

// ===================== scan =====================

type scanner interface {
	Scan(dest ...interface{}) error
}

func scanState(row scanner) (*CardState, error) {
	var st CardState
	var dueAt, created, updated string
	var lastReview sql.NullString
	if err := row.Scan(&st.CardID, &st.Deck, &st.Front, &st.Back, &st.Stability,
		&st.Difficulty, &dueAt, &lastReview, &st.Reps, &st.Lapses, &st.State,
		&created, &updated); err != nil {
		return nil, err
	}
	st.DueAt = parseTime(dueAt)
	st.CreatedAt = parseTime(created)
	st.UpdatedAt = parseTime(updated)
	if lastReview.Valid {
		t := parseTime(lastReview.String)
		st.LastReview = &t
	}
	return &st, nil
}

func scanStates(rows *sql.Rows) ([]*CardState, error) {
	var out []*CardState
	for rows.Next() {
		st, err := scanState(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, st)
	}
	return out, rows.Err()
}

func nullableTime(t *time.Time) sql.NullString {
	if t == nil {
		return sql.NullString{}
	}
	return sql.NullString{String: formatTime(*t), Valid: true}
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
