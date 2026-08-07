package sync

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/helixnow/deep-student-go/internal/cloudstorage"
	"github.com/helixnow/deep-student-go/internal/todo"
	"github.com/helixnow/deep-student-go/pkg/llm"
	"github.com/helixnow/deep-student-go/pkg/store"
	"github.com/helixnow/deep-student-go/pkg/store/blob"
	"github.com/helixnow/deep-student-go/pkg/vfs"
)

// newPair 构造两个独立库（模拟两台设备），先建业务表再装同步触发器。
func newPair(t *testing.T) (*Service, *Service, *todo.Service, *todo.Service) {
	t.Helper()
	mk := func(name string) (*Service, *todo.Service) {
		dir := t.TempDir()
		bs, _ := blob.New(filepath.Join(dir, "b"))
		fs := vfs.NewFS(bs)
		st, err := store.Open(filepath.Join(dir, name+".db"))
		if err != nil {
			t.Fatal(err)
		}
		t.Cleanup(func() { _ = st.Close() })
		// 先建业务服务（建表），再装同步触发器
		td := todo.New(fs, st, llm.NewRegistry())
		s := New(st)
		if err := s.EnsureTriggers(); err != nil {
			t.Fatal(err)
		}
		return s, td
	}
	s1, td1 := mk("dev1")
	s2, td2 := mk("dev2")
	return s1, s2, td1, td2
}

func TestTriggersRecordChanges(t *testing.T) {
	s, _, td, _ := newPair(t)
	l, _ := td.CreateList(todo.CreateListParams{Name: "同步测试"})
	changes, err := s.Pending(0, 100)
	if err != nil {
		t.Fatal(err)
	}
	found := false
	for _, c := range changes {
		if c.Table == "todo_lists" && c.RecordID == l.ID && c.Operation == "INSERT" {
			found = true
		}
	}
	if !found {
		t.Fatalf("changes=%+v", changes)
	}
}

func TestExportApplyRoundTrip(t *testing.T) {
	s1, s2, td1, td2 := newPair(t)
	l, _ := td1.CreateList(todo.CreateListParams{Name: "任务A"})
	_, _ = td1.CreateItem(todo.CreateItemParams{ListID: l.ID, Title: "子任务1"})

	// 设备1 导出
	batch, err := s1.ExportChanges("dev1", 0, 100)
	if err != nil {
		t.Fatal(err)
	}
	if len(batch.Changes) < 2 {
		t.Fatalf("changes=%d", len(batch.Changes))
	}
	// 设备2 应用
	applied, quarantined, err := s2.ApplyChanges(batch)
	if err != nil {
		t.Fatal(err)
	}
	if applied < 2 || quarantined != 0 {
		t.Fatalf("applied=%d quarantined=%d", applied, quarantined)
	}
	// 设备2 应能看到同步来的列表
	lists, err := td2.ListLists(false)
	if err != nil {
		t.Fatal(err)
	}
	if len(lists) != 1 || lists[0].Name != "任务A" {
		t.Fatalf("lists=%+v", lists)
	}
	items, _ := td2.ListItems(l.ID, "all")
	if len(items) != 1 {
		t.Fatalf("items=%d", len(items))
	}
}

func TestTombstonePropagation(t *testing.T) {
	s1, s2, td1, td2 := newPair(t)
	l, _ := td1.CreateList(todo.CreateListParams{Name: "将被删除"})
	batch, _ := s1.ExportChanges("dev1", 0, 100)
	s2.ApplyChanges(batch)

	// 设备1 软删除
	if err := td1.DeleteList(l.ID); err != nil {
		t.Fatal(err)
	}
	batch2, _ := s1.ExportChanges("dev1", batch.MaxSeq, 100)
	if len(batch2.Changes) != 1 || batch2.Changes[0].Operation != "UPDATE" {
		t.Fatalf("batch2=%+v", batch2.Changes)
	}
	s2.ApplyChanges(batch2)
	// 设备2 该列表应进入回收站
	lists, _ := td2.ListLists(true)
	found := false
	for _, x := range lists {
		if x.ID == l.ID && x.IsDeleted {
			found = true
		}
	}
	if !found {
		t.Fatalf("tombstone not applied: %+v", lists)
	}
}

func TestConflictLocalWins(t *testing.T) {
	s1, s2, td1, td2 := newPair(t)
	l, _ := td1.CreateList(todo.CreateListParams{Name: "初始"})
	batch, _ := s1.ExportChanges("dev1", 0, 100)
	s2.ApplyChanges(batch)

	// 设备1 先改名（远端较早）
	na := "设备1改名"
	td1.UpdateList(todo.UpdateListParams{ID: l.ID, Name: &na})
	b2, _ := s1.ExportChanges("dev1", batch.MaxSeq, 100)
	// 设备2 后改名（本地较新 → 冲突）
	nb := "设备2改名"
	td2.UpdateList(todo.UpdateListParams{ID: l.ID, Name: &nb})

	// 设备2 应用设备1的较早变更：本地较新 → 冲突进隔离区
	applied, quarantined, err := s2.ApplyChanges(b2)
	if err != nil {
		t.Fatal(err)
	}
	_ = applied
	if quarantined < 1 {
		t.Fatalf("expected quarantine, got %d", quarantined)
	}
	// 本地（设备2）仍是自己的名字
	got, _ := td2.GetList(l.ID)
	if got.Name != "设备2改名" {
		t.Fatalf("local should win: %s", got.Name)
	}
	// 隔离区有记录
	cnt, _ := s2.QuarantineCount()
	if cnt < 1 {
		t.Fatal("quarantine empty")
	}
}

func TestQuarantineRetry(t *testing.T) {
	s1, s2, td1, td2 := newPair(t)
	l, _ := td1.CreateList(todo.CreateListParams{Name: "q"})
	batch, _ := s1.ExportChanges("dev1", 0, 100)
	s2.ApplyChanges(batch)
	// 制造冲突：设备1 先改（较早），设备2 后改（本地较新）
	n1 := "一"
	td1.UpdateList(todo.UpdateListParams{ID: l.ID, Name: &n1})
	b2, _ := s1.ExportChanges("dev1", batch.MaxSeq, 100)
	n2 := "二"
	td2.UpdateList(todo.UpdateListParams{ID: l.ID, Name: &n2})
	s2.ApplyChanges(b2)
	// 重试冲突：本地仍较新 → 再次隔离
	entries, _ := s2.QuarantineList(10)
	if len(entries) == 0 {
		t.Fatal("no quarantine entries")
	}
	if err := s2.RetryQuarantine(entries[0].ID); err == nil {
		t.Fatal("retry should fail while local newer")
	}
	// 丢弃
	if err := s2.DiscardQuarantine(entries[0].ID); err != nil {
		t.Fatal(err)
	}
	// 清空
	_, _ = s2.DiscardAllQuarantine()
}

func TestSyncToCloud(t *testing.T) {
	s1, s2, td1, td2 := newPair(t)
	td1.CreateList(todo.CreateListParams{Name: "云同步A"})

	// 共享 WebDAV mock 作为"云端"
	storeMock := newMockCloud()
	srv := newCloudServer(t, storeMock)
	defer srv.Close()
	be, err := cloudstorage.NewBackend(cloudstorage.Config{
		Provider:  cloudstorage.ProviderWebDAV,
		WebDAVURL: srv.URL,
	})
	if err != nil {
		t.Fatal(err)
	}
	// 设备1 推上去
	out1, err := s1.SyncToCloud(context.Background(), be, "sync-root", "dev1")
	if err != nil {
		t.Fatal(err)
	}
	if out1.Uploaded < 1 {
		t.Fatalf("uploaded=%d", out1.Uploaded)
	}
	// 设备2 拉下来（同一云端）
	out2, err := s2.SyncToCloud(context.Background(), be, "sync-root", "dev2")
	if err != nil {
		t.Fatal(err)
	}
	if out2.Downloaded < 1 {
		t.Fatalf("downloaded=%d", out2.Downloaded)
	}
	lists, _ := td2.ListLists(false)
	if len(lists) != 1 {
		t.Fatalf("dev2 lists=%d", len(lists))
	}
}
