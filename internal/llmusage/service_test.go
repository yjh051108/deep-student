package llmusage

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

func TestRecordAndQuery(t *testing.T) {
	s := newSvc(t)
	l, err := s.Record(LogEntry{
		Provider:         "openai",
		Model:            "gpt-4o-mini",
		PromptTokens:     100,
		CompletionTokens: 50,
		CallerType:       "chat",
		SessionID:        "sess-1",
	})
	if err != nil {
		t.Fatal(err)
	}
	if l.TotalTokens != 150 {
		t.Fatalf("total=%d", l.TotalTokens)
	}
	if l.Status != StatusSuccess || l.TokenSource != TokenSourceAPI {
		t.Fatalf("log=%+v", l)
	}

	logs, err := s.Query(LogFilter{Provider: "openai"})
	if err != nil {
		t.Fatal(err)
	}
	if len(logs) != 1 {
		t.Fatalf("logs=%d", len(logs))
	}
	if logs[0].TotalTokens != 150 || logs[0].SessionID != "sess-1" {
		t.Fatalf("log=%+v", logs[0])
	}

	// 过滤不匹配
	logs, _ = s.Query(LogFilter{Provider: "anthropic"})
	if len(logs) != 0 {
		t.Fatalf("filter failed: %d", len(logs))
	}
}

func TestErrorRecord(t *testing.T) {
	s := newSvc(t)
	msg := "rate limit"
	_, err := s.Record(LogEntry{
		Provider:     "deepseek",
		Model:        "deepseek-chat",
		PromptTokens: 10,
		CallerType:   "translation",
		Status:       StatusError,
		ErrorMessage: msg,
	})
	if err != nil {
		t.Fatal(err)
	}
	logs, err := s.Query(LogFilter{Status: StatusError})
	if err != nil {
		t.Fatal(err)
	}
	if len(logs) != 1 || logs[0].ErrorMessage != msg {
		t.Fatalf("logs=%+v", logs)
	}
}

func TestDailyAggregation(t *testing.T) {
	s := newSvc(t)
	for i := 0; i < 3; i++ {
		_, _ = s.Record(LogEntry{
			Provider:         "openai",
			Model:            "gpt-4o",
			PromptTokens:     100,
			CompletionTokens: 200,
			CallerType:       "chat",
		})
	}
	// 不同 model 会生成另一行聚合
	_, _ = s.Record(LogEntry{
		Provider: "openai",
		Model:    "gpt-4o-mini",
		PromptTokens: 5,
		CallerType: "chat",
	})

	daily, err := s.QueryDaily(DailyFilter{Model: "gpt-4o"})
	if err != nil {
		t.Fatal(err)
	}
	if len(daily) != 1 {
		t.Fatalf("daily=%+v", daily)
	}
	d := daily[0]
	if d.RequestCount != 3 || d.TotalTokens != 900 {
		t.Fatalf("daily=%+v", d)
	}
	if d.SuccessCount != 3 || d.ErrorCount != 0 {
		t.Fatalf("daily=%+v", d)
	}
}

func TestSummary(t *testing.T) {
	s := newSvc(t)
	_, _ = s.Record(LogEntry{Provider: "openai", Model: "m1", PromptTokens: 10, CompletionTokens: 10, CallerType: "chat"})
	_, _ = s.Record(LogEntry{Provider: "openai", Model: "m2", PromptTokens: 20, CompletionTokens: 5, CallerType: "anki"})

	sum, err := s.Summary()
	if err != nil {
		t.Fatal(err)
	}
	if sum.TotalRequests != 2 || sum.TotalTokens != 45 {
		t.Fatalf("summary=%+v", sum)
	}
	if sum.TodayRequests != 2 || sum.TotalPromptTokens != 30 {
		t.Fatalf("summary=%+v", sum)
	}
}

func TestCleanup(t *testing.T) {
	s := newSvc(t)
	_, _ = s.Record(LogEntry{Provider: "openai", Model: "m1", PromptTokens: 1, CallerType: "chat"})
	n, err := s.CleanupOlderThan(time.Now().UTC().Add(24 * time.Hour))
	if err != nil {
		t.Fatal(err)
	}
	if n != 1 {
		t.Fatalf("cleaned=%d", n)
	}
	logs, _ := s.Query(LogFilter{})
	if len(logs) != 0 {
		t.Fatalf("logs after cleanup=%d", len(logs))
	}
}

func TestRecordChatHelper(t *testing.T) {
	s := newSvc(t)
	_, err := s.RecordChat("chat", "deepseek", "deepseek-chat", "sess-9", llm.Usage{PromptTokens: 7, CompletionTokens: 3, TotalTokens: 10})
	if err != nil {
		t.Fatal(err)
	}
	logs, _ := s.Query(LogFilter{CallerType: "chat"})
	if len(logs) != 1 || logs[0].TotalTokens != 10 {
		t.Fatalf("logs=%+v", logs)
	}
}
