package vault

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/helixnow/deep-student-go/pkg/store"
	"github.com/helixnow/deep-student-go/pkg/store/blob"
)

func TestFrontMatterRoundTrip(t *testing.T) {
	fm := FrontMatter{
		Title:   "测试笔记",
		Tags:    []string{"go", "中文"},
		Created: time.Date(2026, 8, 1, 0, 0, 0, 0, time.UTC),
		Updated: time.Date(2026, 8, 7, 12, 0, 0, 0, time.UTC),
		DSID:    "abc-123",
		DSType:  "note",
		Extra:   map[string]string{"ext": "md"},
	}
	data, err := WriteFrontMatter(fm, "# 正文\n\nHello **world**")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(string(data), "---\n") {
		t.Fatal("missing frontmatter open")
	}
	got, body, hasFM := ReadFrontMatter(data)
	if !hasFM {
		t.Fatal("frontmatter not detected")
	}
	if got.DSID != "abc-123" || got.Title != "测试笔记" || got.DSType != "note" {
		t.Fatalf("fm=%+v", got)
	}
	if len(got.Tags) != 2 || got.Tags[0] != "go" {
		t.Fatalf("tags=%v", got.Tags)
	}
	if !strings.Contains(body, "# 正文") || strings.Contains(body, "---") {
		t.Fatalf("body=%q", body)
	}
}

func TestFrontMatterAbsent(t *testing.T) {
	_, body, hasFM := ReadFrontMatter([]byte("plain text\n"))
	if hasFM {
		t.Fatal("plain text should not have frontmatter")
	}
	if body != "plain text\n" {
		t.Fatalf("body=%q", body)
	}
}

func TestSanitizeFilename(t *testing.T) {
	cases := map[string]string{
		"学习笔记":            "学习笔记",
		`foo/bar:baz?*qux`: "foo-bar-baz-qux",
		"  spaced  ":       "spaced",
	}
	for in, want := range cases {
		if got := SanitizeFilename(in); got != want {
			t.Fatalf("SanitizeFilename(%q)=%q want %q", in, got, want)
		}
	}
}

func TestScanVault(t *testing.T) {
	dir := t.TempDir()
	v, err := New(filepath.Join(dir, "vault"))
	if err != nil {
		t.Fatal(err)
	}
	// 造两个笔记文件：一个带 frontmatter，一个普通 md
	notesDir := filepath.Join(dir, "vault", "notes")
	os.MkdirAll(notesDir, 0o755)
	fm := FrontMatter{Title: "A", DSID: "id-a", DSType: "note", Tags: []string{"x"}}
	data, _ := WriteFrontMatter(fm, "content a")
	os.WriteFile(filepath.Join(notesDir, "A.md"), data, 0o644)
	os.WriteFile(filepath.Join(notesDir, "普通笔记.md"), []byte("hello"), 0o644)

	entries, errs := v.Scan()
	if len(errs) != 0 {
		t.Fatalf("errors=%v", errs)
	}
	if len(entries) != 2 {
		t.Fatalf("entries=%d", len(entries))
	}
	foundFM, foundPlain := false, false
	for _, e := range entries {
		if e.ID == "id-a" {
			foundFM = true
			if e.Type != TypeNote || e.Title != "A" {
				t.Fatalf("entry=%+v", e)
			}
		} else if e.Title == "普通笔记" {
			foundPlain = true
		}
	}
	if !foundFM || !foundPlain {
		t.Fatalf("foundFM=%v foundPlain=%v entries=%+v", foundFM, foundPlain, entries)
	}
}

func TestAllocatePathDedup(t *testing.T) {
	dir := t.TempDir()
	v, err := New(dir)
	if err != nil {
		t.Fatal(err)
	}
	p1 := v.AllocatePath(TypeNote, "标题", ".md", "")
	// 模拟已写入文件，触发去重
	os.MkdirAll(filepath.Dir(p1), 0o755)
	if err := os.WriteFile(p1, []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	p2 := v.AllocatePath(TypeNote, "标题", ".md", "")
	if p1 == p2 {
		t.Fatalf("expected distinct paths, got %s", p1)
	}
	// 复用已有路径
	p3 := v.AllocatePath(TypeNote, "标题", ".md", p1)
	if p3 != p1 {
		t.Fatalf("existing path should be reused, got %s", p3)
	}
}

func TestMigrateFromBlob(t *testing.T) {
	dir := t.TempDir()
	v, err := New(filepath.Join(dir, "vault"))
	if err != nil {
		t.Fatal(err)
	}
	// 造 store + blob
	st, err := store.Open(filepath.Join(dir, "x.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer st.Close()
	bs, err := blob.New(filepath.Join(dir, "blob"))
	if err != nil {
		t.Fatal(err)
	}
	// 两个资源：一个 md 笔记、一个 pdf
	ref1, _, _ := bs.Put([]byte("# note body"))
	st.SaveResource("vfs://note/n1", "note", "n1", "第一条笔记", "tag1", "", ref1, 10, time.Now().Unix())
	ref2, _, _ := bs.Put([]byte("%PDF-1.4 fake"))
	st.SaveResource("vfs://textbook/t1", "textbook", "t1", "教材", "", `{"ext":"pdf"}`, ref2, 20, time.Now().Unix())

	res, err := v.MigrateFromBlob(st, bs)
	if err != nil {
		t.Fatal(err)
	}
	if !res.DidRun || res.Migrated != 2 {
		t.Fatalf("result=%+v", res)
	}
	// 验证文件落盘
	if _, err := os.Stat(filepath.Join(dir, "vault", "notes", "第一条笔记.md")); err != nil {
		t.Fatal("note md missing: ", err)
	}
	if _, err := os.Stat(filepath.Join(dir, "vault", "resources", "教材.pdf")); err != nil {
		t.Fatal("pdf missing: ", err)
	}
	// 二次迁移应跳过（vault 已有资源）
	res2, err := v.MigrateFromBlob(st, bs)
	if err != nil {
		t.Fatal(err)
	}
	if res2.DidRun {
		t.Fatal("second migration should skip")
	}
	// 迁移后可扫描
	entries, _ := v.Scan()
	if len(entries) != 2 {
		t.Fatalf("scan after migrate=%d", len(entries))
	}
}
