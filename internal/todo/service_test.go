package todo

import (
	"context"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/helixnow/deep-student-go/pkg/llm"
	"github.com/helixnow/deep-student-go/pkg/store"
	"github.com/helixnow/deep-student-go/pkg/store/blob"
	"github.com/helixnow/deep-student-go/pkg/vfs"
)

// mockProv 测试用 LLM Provider：命中拆解 prompt 时返回 JSON 数组。
type mockProv struct{}

func (m *mockProv) Name() string { return "openai" }

func (m *mockProv) Chat(_ context.Context, req llm.ChatRequest) (*llm.ChatResponse, error) {
	sys := ""
	for _, msg := range req.Messages {
		sys += msg.Content
	}
	if strings.Contains(sys, "任务拆解助手") {
		return &llm.ChatResponse{Content: `[{"title":"调研资料","estPomodoros":2},{"title":"写初稿","estPomodoros":3}]`}, nil
	}
	return &llm.ChatResponse{Content: "ok"}, nil
}

func (m *mockProv) Stream(context.Context, llm.ChatRequest) (<-chan llm.Chunk, error) {
	ch := make(chan llm.Chunk, 1)
	ch <- llm.Chunk{Delta: "ok", Done: true}
	close(ch)
	return ch, nil
}

func (m *mockProv) Embed(context.Context, llm.EmbedRequest) (*llm.EmbedResponse, error) {
	return &llm.EmbedResponse{Embeddings: [][]float32{{1, 2, 3}}}, nil
}

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

	reg := llm.NewRegistry()
	reg.Register(&mockProv{})
	return New(fs, st, reg)
}

func TestEnsureInbox(t *testing.T) {
	s := newSvc(t)
	inbox, err := s.EnsureInbox()
	if err != nil {
		t.Fatal(err)
	}
	if !inbox.IsInbox {
		t.Fatal("expected inbox flag")
	}
	// 幂等
	inbox2, err := s.EnsureInbox()
	if err != nil {
		t.Fatal(err)
	}
	if inbox2.ID != inbox.ID {
		t.Fatal("EnsureInbox not idempotent")
	}
}

func TestListCRUD(t *testing.T) {
	s := newSvc(t)
	l, err := s.CreateList(CreateListParams{Name: "学习"})
	if err != nil {
		t.Fatal(err)
	}
	if l.Name != "学习" {
		t.Fatalf("name=%s", l.Name)
	}
	if _, err := s.CreateList(CreateListParams{Name: "  "}); err == nil {
		t.Fatal("empty name should error")
	}

	lists, err := s.ListLists(false)
	if err != nil {
		t.Fatal(err)
	}
	if len(lists) != 1 {
		t.Fatalf("lists=%d", len(lists))
	}

	// 更新
	nn := "学习v2"
	ul, err := s.UpdateList(UpdateListParams{ID: l.ID, Name: &nn})
	if err != nil {
		t.Fatal(err)
	}
	if ul.Name != "学习v2" {
		t.Fatalf("updated name=%s", ul.Name)
	}

	// 删除进回收站
	if err := s.DeleteList(l.ID); err != nil {
		t.Fatal(err)
	}
	dl, err := s.ListDeletedLists()
	if err != nil {
		t.Fatal(err)
	}
	if len(dl) != 1 {
		t.Fatalf("deleted lists=%d", len(dl))
	}
	// 恢复
	if err := s.RestoreList(l.ID); err != nil {
		t.Fatal(err)
	}
	// 彻底删除
	if err := s.PurgeList(l.ID); err != nil {
		t.Fatal(err)
	}
	lists, _ = s.ListLists(true)
	if len(lists) != 0 {
		t.Fatalf("after purge lists=%d", len(lists))
	}
}

func TestItemLifecycle(t *testing.T) {
	s := newSvc(t)
	l, err := s.CreateList(CreateListParams{Name: "工作"})
	if err != nil {
		t.Fatal(err)
	}
	due := time.Now().UTC().Add(24 * time.Hour)
	it, err := s.CreateItem(CreateItemParams{
		ListID: l.ID,
		Title:  "写周报",
		DueAt:  &due,
		Priority: 2,
		Tags:   []string{"重要"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if it.Priority != 2 || len(it.Tags) != 1 {
		t.Fatalf("item=%+v", it)
	}

	// 子任务
	sub, err := s.CreateItem(CreateItemParams{ListID: l.ID, Title: "收集数据", ParentID: &it.ID})
	if err != nil {
		t.Fatal(err)
	}
	_ = sub

	// 完成
	toggled, err := s.ToggleItem(it.ID)
	if err != nil {
		t.Fatal(err)
	}
	if toggled.CompletedAt == nil {
		t.Fatal("expected completed")
	}
	// 再切回
	toggled, err = s.ToggleItem(it.ID)
	if err != nil {
		t.Fatal(err)
	}
	if toggled.CompletedAt != nil {
		t.Fatal("expected not completed")
	}

	// 软删进回收站
	if err := s.DeleteItem(it.ID); err != nil {
		t.Fatal(err)
	}
	dels, err := s.ListDeletedItems()
	if err != nil {
		t.Fatal(err)
	}
	if len(dels) != 1 {
		t.Fatalf("deleted items=%d", len(dels))
	}
	if err := s.RestoreItem(it.ID); err != nil {
		t.Fatal(err)
	}
	// 彻底删除（含子任务遗留由 purge 级联处理在列表级，单条删除不级联）
	if err := s.PurgeItem(it.ID); err != nil {
		t.Fatal(err)
	}
	if err := s.PurgeItem(sub.ID); err != nil {
		t.Fatal(err)
	}
}

func TestSubtaskSameList(t *testing.T) {
	s := newSvc(t)
	l1, _ := s.CreateList(CreateListParams{Name: "A"})
	l2, _ := s.CreateList(CreateListParams{Name: "B"})
	it, _ := s.CreateItem(CreateItemParams{ListID: l1.ID, Title: "父"})
	_, err := s.CreateItem(CreateItemParams{ListID: l2.ID, Title: "子", ParentID: &it.ID})
	if err == nil {
		t.Fatal("cross-list subtask should error")
	}
}

func TestViews(t *testing.T) {
	s := newSvc(t)
	l, _ := s.CreateList(CreateListParams{Name: "视图"})

	now := time.Now().UTC()
	// 今天
	today := now
	s.CreateItem(CreateItemParams{ListID: l.ID, Title: "今日事", DueAt: &today})
	// 昨天 → 逾期
	yesterday := now.Add(-24 * time.Hour)
	s.CreateItem(CreateItemParams{ListID: l.ID, Title: "昨日事", DueAt: &yesterday})

	todayItems, err := s.ListToday()
	if err != nil {
		t.Fatal(err)
	}
	if len(todayItems) != 1 {
		t.Fatalf("today=%d", len(todayItems))
	}
	overdue, err := s.ListOverdue()
	if err != nil {
		t.Fatal(err)
	}
	if len(overdue) != 1 {
		t.Fatalf("overdue=%d", len(overdue))
	}
	upcoming, err := s.ListUpcoming()
	if err != nil {
		t.Fatal(err)
	}
	if len(upcoming) != 1 {
		t.Fatalf("upcoming=%d", len(upcoming))
	}

	sum, err := s.Summary()
	if err != nil {
		t.Fatal(err)
	}
	if sum.TotalPending != 2 || sum.OverdueCount != 1 {
		t.Fatalf("summary=%+v", sum)
	}
}

func TestSearch(t *testing.T) {
	s := newSvc(t)
	l, _ := s.CreateList(CreateListParams{Name: "S"})
	s.CreateItem(CreateItemParams{ListID: l.ID, Title: "读论文《注意力》"})
	s.CreateItem(CreateItemParams{ListID: l.ID, Title: "健身"})

	items, err := s.Search("论文", 10)
	if err != nil {
		t.Fatal(err)
	}
	if len(items) != 1 || items[0].Title != "读论文《注意力》" {
		t.Fatalf("search=%+v", items)
	}
}

func TestReorder(t *testing.T) {
	s := newSvc(t)
	l, _ := s.CreateList(CreateListParams{Name: "R"})
	a, _ := s.CreateItem(CreateItemParams{ListID: l.ID, Title: "A"})
	b, _ := s.CreateItem(CreateItemParams{ListID: l.ID, Title: "B"})
	if err := s.ReorderItems(l.ID, []string{b.ID, a.ID}); err != nil {
		t.Fatal(err)
	}
	items, err := s.ListItems(l.ID, FilterAll)
	if err != nil {
		t.Fatal(err)
	}
	if items[0].ID != b.ID {
		t.Fatalf("reorder failed: first=%s", items[0].ID)
	}
}

func TestAIBreakdown(t *testing.T) {
	s := newSvc(t)
	ctx := context.Background()
	items, err := s.AIBreakdown(ctx, "完成毕业设计", "")
	if err != nil {
		t.Fatal(err)
	}
	if len(items) != 2 {
		t.Fatalf("breakdown=%d", len(items))
	}
	if items[0].Title != "调研资料" {
		t.Fatalf("first=%s", items[0].Title)
	}
}

func TestPurgeDeleted(t *testing.T) {
	s := newSvc(t)
	l, _ := s.CreateList(CreateListParams{Name: "P"})
	it, _ := s.CreateItem(CreateItemParams{ListID: l.ID, Title: "x"})
	_ = s.DeleteItem(it.ID)
	n, err := s.PurgeDeletedItems()
	if err != nil {
		t.Fatal(err)
	}
	if n != 1 {
		t.Fatalf("purged=%d", n)
	}
}
