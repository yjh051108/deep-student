package vfs

import (
	"path/filepath"
	"testing"

	"github.com/helixnow/deep-student-go/pkg/store/blob"
)

func TestParseURI(t *testing.T) {
	cases := []struct {
		raw  string
		typ  ResourceType
		id   string
		fail bool
	}{
		{"vfs://note/abc", TypeNote, "abc", false},
		{"vfs://textbook/中文", TypeTextbook, "中文", false},
		{"http://example.com", "", "", true},
		{"vfs://note/", "", "", true},
		// BUG-002 回归：含 # 不应被当作 fragment 截断
		{"vfs://note/abc#tag1", TypeNote, "abc#tag1", false},
		{"vfs://note/C#代码笔记", TypeNote, "C#代码笔记", false},
	}
	for _, c := range cases {
		u, err := Parse(c.raw)
		if c.fail {
			if err == nil {
				t.Fatalf("expected fail: %s", c.raw)
			}
			continue
		}
		if err != nil {
			t.Fatalf("%s: %v", c.raw, err)
		}
		if u.Type != c.typ || u.ID != c.id {
			t.Fatalf("mismatch %s: got (%s, %s)", c.raw, u.Type, u.ID)
		}
	}
}

// TestHashRoundTrip BUG-002 回归：通过 Put/Get 验证含 # 的 ID 能正确存储与读取。
func TestHashRoundTrip(t *testing.T) {
	dir := t.TempDir()
	bs, _ := blob.New(filepath.Join(dir, "b"))
	fs := NewFS(bs)
	uri := "vfs://note/C#笔记"
	if _, err := fs.Put(uri, []byte("content"), map[string]string{"title": "C#"}); err != nil {
		t.Fatal(err)
	}
	data, _, err := fs.Get(uri)
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != "content" {
		t.Fatalf("data: %s", data)
	}
}

func TestPutGetList(t *testing.T) {
	dir := t.TempDir()
	bs, err := blob.New(filepath.Join(dir, "b"))
	if err != nil {
		t.Fatal(err)
	}
	fs := NewFS(bs)
	uri := "vfs://note/n1"
	e, err := fs.Put(uri, []byte("hello"), map[string]string{"title": "Hi", "tags": "a,b"})
	if err != nil {
		t.Fatal(err)
	}
	if e.URI != uri {
		t.Fatalf("uri mismatch: %s", e.URI)
	}
	data, _, err := fs.Get(uri)
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != "hello" {
		t.Fatalf("data: %s", data)
	}
	list := fs.List(TypeNote)
	if len(list) != 1 {
		t.Fatalf("len=%d", len(list))
	}
	res := fs.Search(TypeNote, "a")
	if len(res) != 1 {
		t.Fatalf("search len=%d", len(res))
	}
}
