// pomodoro 包的业务层：创建记录、按待办/日期查询、统计。
//
// 构造函数 New(fs, store, llmReg) 与既有 internal 包保持一致；
// 持久化交给 Store（SQLite）。番茄钟完成时如关联待办，
// 由前端配合 todo 模块递增 done_pomodoros。

package pomodoro

import (
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/google/uuid"

	"github.com/helixnow/deep-student-go/pkg/llm"
	"github.com/helixnow/deep-student-go/pkg/store"
	"github.com/helixnow/deep-student-go/pkg/vfs"
)

// Service 番茄钟业务服务。
type Service struct {
	store *Store
	mu    sync.RWMutex
}

// New 构造番茄钟服务。vfs 与 llmReg 保留用于与既有构造签名一致。
func New(_ *vfs.FS, s *store.Store, _ *llm.Registry) *Service {
	ps := NewStore(s)
	if err := ps.Migrate(); err != nil {
		fmt.Printf("[pomodoro] migrate failed: %v\n", err)
	}
	return &Service{store: ps}
}

// Create 创建一条番茄钟记录，返回带默认值的完整记录。
func (s *Service) Create(p CreateParams) (*Record, error) {
	now := time.Now().UTC()
	start := now
	if p.StartTime != nil {
		start = p.StartTime.UTC()
	}
	typ := p.Type
	if typ == "" {
		typ = TypeWork
	}
	switch typ {
	case TypeWork, TypeShortBreak, TypeLongBreak:
	default:
		return nil, fmt.Errorf("pomodoro: invalid type %q", typ)
	}
	status := p.Status
	if status == "" {
		status = StatusCompleted
	}
	switch status {
	case StatusCompleted, StatusInterrupted:
	default:
		return nil, fmt.Errorf("pomodoro: invalid status %q", status)
	}
	duration := p.Duration
	if duration <= 0 {
		duration = DefaultWorkSeconds
	}
	actual := p.ActualDuration
	if actual < 0 {
		actual = 0
	}
	var end *time.Time
	if status == StatusCompleted {
		t := start.Add(time.Duration(actual) * time.Second)
		end = &t
	}
	r := &Record{
		ID:             uuid.NewString(),
		TodoItemID:     p.TodoItemID,
		StartTime:      start,
		EndTime:        end,
		Duration:       duration,
		ActualDuration: actual,
		Type:           typ,
		Status:         status,
		CreatedAt:      now,
	}
	if err := s.store.CreateRecord(r); err != nil {
		return nil, err
	}
	return r, nil
}

// Get 读取记录。
func (s *Service) Get(id string) (*Record, error) {
	return s.store.GetRecord(id)
}

// ListByTodo 列出关联待办的记录。
func (s *Service) ListByTodo(todoItemID string) ([]Record, error) {
	return s.store.ListByTodo(todoItemID)
}

// ListToday 今日记录（按 UTC 日）。
func (s *Service) ListToday() ([]Record, error) {
	now := time.Now().UTC()
	start := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.UTC)
	end := start.AddDate(0, 0, 1).Add(-time.Nanosecond)
	return s.store.ListBetween(&start, &end)
}

// TodayStats 今日统计。
func (s *Service) TodayStats() (*Stats, error) {
	now := time.Now().UTC()
	start := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.UTC)
	end := start.AddDate(0, 0, 1).Add(-time.Nanosecond)
	return s.StatsFor(&start, &end)
}

// StatsFor 统计 [start, end] 时间段。
func (s *Service) StatsFor(start, end *time.Time) (*Stats, error) {
	records, err := s.store.ListBetween(start, end)
	if err != nil {
		return nil, err
	}
	date := ""
	if start != nil {
		date = start.UTC().Format("2006-01-02")
	}
	st := &Stats{Date: date}
	for _, r := range records {
		st.PlannedSeconds += r.Duration
		st.TotalSeconds += r.ActualDuration
		if r.Status == StatusCompleted {
			st.CompletedCount++
		} else {
			st.InterruptedCount++
		}
		if r.Type == TypeWork {
			st.WorkCount++
		} else {
			st.BreakCount++
		}
	}
	return st, nil
}

// DailyStats 最近 N 天每日专注统计（work + completed）。
func (s *Service) DailyStats(days int) ([]DailyStat, error) {
	return s.store.DailyAggregate(days)
}

// ErrNotFound 兼容错误（保留占位，具体错误由 store 返回）。
var ErrNotFound = errors.New("pomodoro: not found")
