package memory

import (
	"testing"

	"github.com/helixnow/deep-student-go/pkg/llm"
	"github.com/helixnow/deep-student-go/pkg/store"
	"github.com/helixnow/deep-student-go/pkg/store/blob"
	"github.com/helixnow/deep-student-go/pkg/vfs"
)

// 补充：Memory-as-VFS 扩展测试（文件夹/关系/批量写/持久化）。

func TestMemoryVFSBasics(t *testing.T) {
	s := newMemSvc(t)
	// 批量写
	entries := []WriteEntry{
		{Content: "喜欢咖啡", Category: "preference", Source: "chat"},
		{Content: "目标：过六级", Category: "goal", Source: "chat"},
	}
	items, err := s.WriteBatch(entries)
	if err != nil {
		t.Fatal(err)
	}
	if len(items) != 2 {
		t.Fatalf("items=%d", len(items))
	}
	// 文件夹
	f, err := s.CreateFolder("项目", nil)
	if err != nil {
		t.Fatal(err)
	}
	sub, err := s.CreateFolder("子项目", &f.ID)
	if err != nil {
		t.Fatal(err)
	}
	// 移动
	moved, err := s.MoveToFolder(items[0].ID, sub.ID)
	if err != nil {
		t.Fatal(err)
	}
	if moved.FolderID == nil || *moved.FolderID != sub.ID {
		t.Fatalf("moved=%+v", moved)
	}
	// 树
	tree, err := s.GetTree()
	if err != nil {
		t.Fatal(err)
	}
	if len(tree) != 1 || len(tree[0].Children) != 1 {
		t.Fatalf("tree=%+v", tree)
	}
	// 关系
	if err := s.AddRelation(items[0].ID, items[1].ID, "related"); err != nil {
		t.Fatal(err)
	}
	related, err := s.GetRelated(items[0].ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(related) != 1 {
		t.Fatalf("related=%d", len(related))
	}
	// 更新
	upd, err := s.UpdateContent(items[0].ID, strPtr("更喜欢冷萃"), nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	if upd.Content != "更喜欢冷萃" {
		t.Fatalf("content=%s", upd.Content)
	}
	// 删除
	if err := s.Delete(items[1].ID); err != nil {
		t.Fatal(err)
	}
	// 审计
	logs, err := s.GetAuditLogs(10)
	if err != nil {
		t.Fatal(err)
	}
	if len(logs) < 3 {
		t.Fatalf("audit=%d", len(logs))
	}
}

func TestMemoryPersistence(t *testing.T) {
	dir := t.TempDir()
	bs, _ := blob.New(dir + "/b")
	fs := vfs.NewFS(bs)
	st, err := store.Open(dir + "/x.db")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = st.Close() })
	s := New(fs, st, llm.NewRegistry())
	if _, err := s.Write("持久化测试", "other", "", "test", nil); err != nil {
		t.Fatal(err)
	}
	// 重新构造（同一 db）验证持久化
	s2 := New(fs, st, llm.NewRegistry())
	items := s2.List()
	if len(items) != 1 || items[0].Content != "持久化测试" {
		t.Fatalf("items=%+v", items)
	}
}

func strPtr(s string) *string { return &s }
