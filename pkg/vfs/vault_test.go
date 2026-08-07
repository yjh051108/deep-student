package vfs

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/helixnow/deep-student-go/pkg/store/blob"
	"github.com/helixnow/deep-student-go/pkg/vault"
)

// newVaultFS 构造 vault 模式 VFS（临时目录）。
func newVaultFS(t *testing.T) *FS {
	t.Helper()
	dir := t.TempDir()
	bs, err := blob.New(filepath.Join(dir, "blob"))
	if err != nil {
		t.Fatal(err)
	}
	fs, err := NewVaultFS(filepath.Join(dir, "vault"), bs)
	if err != nil {
		t.Fatal(err)
	}
	return fs
}

func TestVaultPutGet(t *testing.T) {
	fs := newVaultFS(t)
	e, err := fs.Put("vfs://note/abc", []byte("# 标题\n内容"), map[string]string{"title": "第一条笔记", "tags": "go,测试"})
	if err != nil {
		t.Fatal(err)
	}
	if e.Title != "第一条笔记" || e.FilePath == "" {
		t.Fatalf("entry=%+v", e)
	}
	// 文件真实落盘
	if _, err := os.Stat(e.FilePath); err != nil {
		t.Fatal("file missing: ", err)
	}
	// 读取（Get 剥离 frontmatter 返回正文；文件真实内容带尾随换行）
	data, ge, err := fs.Get("vfs://note/abc")
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != "# 标题\n内容\n" {
		t.Fatalf("data=%q", string(data))
	}
	if ge.ID != "abc" || len(ge.Tags) != 2 {
		t.Fatalf("entry=%+v", ge)
	}
}

func TestVaultListSearchDelete(t *testing.T) {
	fs := newVaultFS(t)
	fs.Put("vfs://note/a1", []byte("A"), map[string]string{"title": "A", "tags": "x"})
	fs.Put("vfs://note/a2", []byte("B"), map[string]string{"title": "B", "tags": "y"})

	all := fs.List(TypeNote)
	if len(all) != 2 {
		t.Fatalf("list=%d", len(all))
	}
	sr := fs.Search(TypeNote, "x")
	if len(sr) != 1 || sr[0].ID != "a1" {
		t.Fatalf("search=%+v", sr)
	}
	if err := fs.Delete("vfs://note/a1"); err != nil {
		t.Fatal(err)
	}
	if _, ok := fs.Stat("vfs://note/a1"); ok {
		t.Fatal("should be deleted")
	}
	// 文件同步删除
	if _, err := os.Stat(sr[0].FilePath); !os.IsNotExist(err) {
		t.Fatal("file should be deleted")
	}
}

func TestVaultPersistAcrossReload(t *testing.T) {
	dir := t.TempDir()
	bs, err := blob.New(filepath.Join(dir, "blob"))
	if err != nil {
		t.Fatal(err)
	}
	fs, err := NewVaultFS(filepath.Join(dir, "vault"), bs)
	if err != nil {
		t.Fatal(err)
	}
	fs.Put("vfs://mindmap/m1", []byte("# 图"), map[string]string{"title": "学习导图"})

	// 模拟重启：新建 VFS 重新扫描 vault
	fs2, err := NewVaultFS(filepath.Join(dir, "vault"), bs)
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := fs2.Stat("vfs://mindmap/m1"); !ok {
		t.Fatal("resource lost after reload")
	}
	// Reload 保持幂等
	if err := fs2.Reload(); err != nil {
		t.Fatal(err)
	}
	if _, ok := fs2.Stat("vfs://mindmap/m1"); !ok {
		t.Fatal("resource lost after Reload")
	}
}

func TestVaultExternalEdit(t *testing.T) {
	dir := t.TempDir()
	bs, _ := blob.New(filepath.Join(dir, "blob"))
	fs, _ := NewVaultFS(filepath.Join(dir, "vault"), bs)
	e, _ := fs.Put("vfs://note/n1", []byte("原始"), map[string]string{"title": "外部编辑测试"})

	// 模拟 Obsidian 外部编辑：保留 frontmatter，只改正文
	raw, err := os.ReadFile(e.FilePath)
	if err != nil {
		t.Fatal(err)
	}
	_, _, hasFM := vault.ReadFrontMatter(raw)
	if !hasFM {
		t.Fatal("expected frontmatter in file")
	}
	// 重写正文（frontmatter 保留，通过 WriteFrontMatter 重新生成）
	fm, _, _ := vault.ReadFrontMatter(raw)
	edited, _ := vault.WriteFrontMatter(fm, "外部修改后的内容")
	if err := os.WriteFile(e.FilePath, edited, 0o644); err != nil {
		t.Fatal(err)
	}
	if err := fs.Reload(); err != nil {
		t.Fatal(err)
	}
	data, _, err := fs.Get("vfs://note/n1")
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != "外部修改后的内容\n" {
		t.Fatalf("external edit not picked up: %q", string(data))
	}
}

func TestVaultNonMarkdown(t *testing.T) {
	fs := newVaultFS(t)
	e, err := fs.Put("vfs://textbook/t1", []byte("%PDF-1.4 fake"), map[string]string{"title": "教材PDF", "ext": "pdf"})
	if err != nil {
		t.Fatal(err)
	}
	if filepath.Ext(e.FilePath) != ".pdf" {
		t.Fatalf("ext=%s", filepath.Ext(e.FilePath))
	}
	// 非 md 读取不剥离
	data, _, err := fs.Get("vfs://textbook/t1")
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != "%PDF-1.4 fake" {
		t.Fatalf("data=%q", string(data))
	}
	// 重启后 sidecar 保证资源仍被识别
	fs2, err := NewVaultFS(filepath.Dir(filepath.Dir(e.FilePath)), fs.blob)
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := fs2.Stat("vfs://textbook/t1"); !ok {
		t.Fatal("pdf resource lost after reload (sidecar missing?)")
	}
}

func TestVaultOverwriteSameID(t *testing.T) {
	fs := newVaultFS(t)
	e1, _ := fs.Put("vfs://note/n1", []byte("v1"), map[string]string{"title": "同ID"})
	e2, err := fs.Put("vfs://note/n1", []byte("v2"), map[string]string{"title": "同ID"})
	if err != nil {
		t.Fatal(err)
	}
	if e2.FilePath != e1.FilePath {
		t.Fatalf("overwrite should reuse path: %s vs %s", e1.FilePath, e2.FilePath)
	}
	// 只有一份文件
	entries := fs.List(TypeNote)
	if len(entries) != 1 {
		t.Fatalf("entries=%d", len(entries))
	}
}

func TestVaultLinksAndBacklinks(t *testing.T) {
	dir := t.TempDir()
	bs, _ := blob.New(filepath.Join(dir, "blob"))
	fs, _ := NewVaultFS(filepath.Join(dir, "vault"), bs)
	fs.Put("vfs://note/a", []byte("参见 [[B 笔记]]"), map[string]string{"title": "A 笔记"})
	fs.Put("vfs://note/b", []byte("回到 [[A 笔记]]"), map[string]string{"title": "B 笔记"})

	links := fs.Links("vfs://note/a")
	if len(links) != 1 || links[0].TargetURI != "vfs://note/b" {
		t.Fatalf("links=%+v", links)
	}
	back := fs.Backlinks("vfs://note/b")
	if len(back) != 1 || back[0].SourceURI != "vfs://note/a" {
		t.Fatalf("backlinks=%+v", back)
	}
	graph := fs.Graph()
	if len(graph) != 2 {
		t.Fatalf("graph=%+v", graph)
	}
}
