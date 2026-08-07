package pomodoro

import (
	"path/filepath"
	"testing"
	"time"

	"github.com/helixnow/deep-student-go/pkg/llm"
	"github.com/helixnow/deep-student-go/pkg/store"
	"github.com/helixnow/deep-student-go/pkg/store/blob"
	"github.com/helixnow/deep-student-go/pkg/vfs"
)

// newSvc 构造测试用 Service。
func newSvc(t *testing.T) *Service {
	t.Helper()
	dir := t.TempDir()
	bs, err := blob.New(filepath.Join(dir, "b"))
	if err != nil {
		t.Fatal(err)
	}
	fs := vfs.NewFS(bs)
	st, err := store.Open(filepath.Join(dir, "x.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = st.Close() })
	return New(fs, st, llm.NewRegistry())
}

func TestCreateDefaults(t *testing.T) {
	s := newSvc(t)
	r, err := s.Create(CreateParams{})
	if err != nil {
		t.Fatal(err)
	}
	if r.Type != TypeWork {
		t.Fatalf("type=%s", r.Type)
	}
	if r.Status != StatusCompleted {
		t.Fatalf("status=%s", r.Status)
	}
	if r.Duration != DefaultWorkSeconds {
		t.Fatalf("duration=%d", r.Duration)
	}
	if r.EndTime == nil {
		t.Fatal("completed record should have end time")
	}
	// 关联待办
	tid := "todo-1"
	r2, err := s.Create(CreateParams{TodoItemID: &tid, ActualDuration: 1500})
	if err != nil {
		t.Fatal(err)
	}
	if r2.TodoItemID == nil || *r2.TodoItemID != tid {
		t.Fatalf("todoItemId=%v", r2.TodoItemID)
	}
}

func TestInvalidTypeAndStatus(t *testing.T) {
	s := newSvc(t)
	if _, err := s.Create(CreateParams{Type: "weird"}); err == nil {
		t.Fatal("invalid type should error")
	}
	if _, err := s.Create(CreateParams{Status: "weird"}); err == nil {
		t.Fatal("invalid status should error")
	}
	// 非正 duration 在 service 层兜底为默认值
	r, err := s.Create(CreateParams{Duration: -5})
	if err != nil {
		t.Fatal(err)
	}
	if r.Duration != DefaultWorkSeconds {
		t.Fatalf("duration=%d", r.Duration)
	}
}

func TestInterruptedNoEnd(t *testing.T) {
	s := newSvc(t)
	r, err := s.Create(CreateParams{Status: StatusInterrupted, ActualDuration: 300})
	if err != nil {
		t.Fatal(err)
	}
	if r.EndTime != nil {
		t.Fatal("interrupted record should not have end time")
	}
}

func TestListByTodo(t *testing.T) {
	s := newSvc(t)
	tid := "todo-9"
	_, _ = s.Create(CreateParams{TodoItemID: &tid})
	_, _ = s.Create(CreateParams{TodoItemID: &tid})
	records, err := s.ListByTodo(tid)
	if err != nil {
		t.Fatal(err)
	}
	if len(records) != 2 {
		t.Fatalf("records=%d", len(records))
	}
}

func TestTodayStats(t *testing.T) {
	s := newSvc(t)
	_, _ = s.Create(CreateParams{ActualDuration: 1500}) // 今日 work completed
	_, _ = s.Create(CreateParams{ActualDuration: 300, Status: StatusInterrupted})
	// 昨天的一条
	yesterday := time.Now().UTC().Add(-24 * time.Hour)
	_, _ = s.Create(CreateParams{StartTime: &yesterday, ActualDuration: 1500})

	stats, err := s.TodayStats()
	if err != nil {
		t.Fatal(err)
	}
	if stats.CompletedCount != 1 || stats.InterruptedCount != 1 {
		t.Fatalf("stats=%+v", stats)
	}
	if stats.TotalSeconds != 1800 {
		t.Fatalf("total=%d", stats.TotalSeconds)
	}
	if stats.Date != time.Now().UTC().Format("2006-01-02") {
		t.Fatalf("date=%s", stats.Date)
	}
}

func TestDailyStats(t *testing.T) {
	s := newSvc(t)
	for i := 0; i < 3; i++ {
		_, _ = s.Create(CreateParams{ActualDuration: 1500})
	}
	daily, err := s.DailyStats(7)
	if err != nil {
		t.Fatal(err)
	}
	if len(daily) == 0 {
		t.Fatal("expected daily stats")
	}
	found := false
	for _, d := range daily {
		if d.Date == time.Now().UTC().Format("2006-01-02") && d.Count == 3 {
			found = true
		}
	}
	if !found {
		t.Fatalf("daily=%+v", daily)
	}
}

func TestListToday(t *testing.T) {
	s := newSvc(t)
	_, _ = s.Create(CreateParams{})
	yesterday := time.Now().UTC().Add(-24 * time.Hour)
	_, _ = s.Create(CreateParams{StartTime: &yesterday})
	today, err := s.ListToday()
	if err != nil {
		t.Fatal(err)
	}
	if len(today) != 1 {
		t.Fatalf("today=%d", len(today))
	}
}
