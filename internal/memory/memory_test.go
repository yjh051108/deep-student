package memory

import (
	"testing"

	"github.com/helixnow/deep-student-go/pkg/llm"
	"github.com/helixnow/deep-student-go/pkg/store"
	"github.com/helixnow/deep-student-go/pkg/store/blob"
	"github.com/helixnow/deep-student-go/pkg/vfs"
)

func newMemSvc(t *testing.T) *Service {
	t.Helper()
	dir := t.TempDir()
	bs, _ := blob.New(dir + "/b")
	fs := vfs.NewFS(bs)
	st, err := store.Open(dir + "/x.db")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = st.Close() })
	return New(fs, st, llm.NewRegistry())
}

func TestApplyFact(t *testing.T) {
	s := newMemSvc(t)
	it := s.ApplyFact("likes tea")
	if it.ID == "" {
		t.Fatal("no id")
	}
	if len(s.Search("tea")) != 1 {
		t.Fatal("search miss")
	}
}

func TestProfile(t *testing.T) {
	s := newMemSvc(t)
	s.ApplyFact("x")
	p := s.Profile()
	if p == "" {
		t.Fatal("empty profile")
	}
}

// TestVFSHashRoundTrip BUG-002 回归：memory 通过 vfs 存/取 # 资源时不被截断。
// memory.go 内部对 vfs 资源只做 get/list，外部使用 vfs.Put；这一条用例验证
// 任何 service 在共享同一 vfs.FS 实例时，# 资源 id 不会被错误解析。
func TestVFSHashRoundTrip(t *testing.T) {
	dir := t.TempDir()
	bs, _ := blob.New(dir + "/b")
	fs := vfs.NewFS(bs)
	uri := "vfs://note/C#笔记#tag1"
	if _, err := fs.Put(uri, []byte("payload"), map[string]string{"title": "C#"}); err != nil {
		t.Fatal(err)
	}
	// 直接通过 Parse 反解，ID 应保留 #
	u, err := vfs.Parse(uri)
	if err != nil {
		t.Fatal(err)
	}
	if u.ID != "C#笔记#tag1" {
		t.Fatalf("id=%q", u.ID)
	}
	// Get 也能取到
	data, _, err := fs.Get(uri)
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != "payload" {
		t.Fatalf("data=%q", data)
	}
	// Delete 也能清掉
	if err := fs.Delete(uri); err != nil {
		t.Fatal(err)
	}
}
