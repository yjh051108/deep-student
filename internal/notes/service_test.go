package notes

import (
	"path/filepath"
	"strings"
	"testing"
	"time"

	"golang.org/x/text/encoding/simplifiedchinese"

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
	return New(st, bs, fs)
}

func TestCreateAndGet(t *testing.T) {
	s := newSvc(t)
	n, err := s.Create(CreateParams{
		Title:     "测试笔记",
		ContentMD: "# 你好世界\n\n这是一段中文内容。",
		Tags:      []string{"test", "中文"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if n.ID == "" {
		t.Fatal("empty id")
	}
	if n.Title != "测试笔记" {
		t.Fatalf("title=%q", n.Title)
	}
	if n.WordCount <= 0 {
		t.Fatalf("wordCount=%d", n.WordCount)
	}
	if n.CharCount <= 0 {
		t.Fatalf("charCount=%d", n.CharCount)
	}

	got, err := s.Get(n.ID)
	if err != nil {
		t.Fatal(err)
	}
	if got.Title != n.Title {
		t.Fatalf("title mismatch: %q vs %q", got.Title, n.Title)
	}
	if len(got.Tags) != 2 {
		t.Fatalf("tags=%v", got.Tags)
	}
}

func TestUpdateAndOptimisticLock(t *testing.T) {
	s := newSvc(t)
	n, _ := s.Create(CreateParams{Title: "原标题", ContentMD: "原内容"})

	// 更新标题与内容
	newTitle := "新标题"
	newContent := "新内容"
	updated, err := s.Update(UpdateParams{
		ID:       n.ID,
		Title:    &newTitle,
		ContentMD: &newContent,
	})
	if err != nil {
		t.Fatal(err)
	}
	if updated.Title != "新标题" {
		t.Fatalf("title=%q", updated.Title)
	}
	if updated.ContentMD != "新内容" {
		t.Fatalf("content=%q", updated.ContentMD)
	}
	if !updated.UpdatedAt.After(n.UpdatedAt) {
		t.Fatal("updatedAt not advanced")
	}

	// 乐观锁：传入正确的 ExpectedUpdate 应成功
	staleUpdate := updated.UpdatedAt
	_, err = s.Update(UpdateParams{
		ID:             n.ID,
		ContentMD:      strPtr("再次更新"),
		ExpectedUpdate: &staleUpdate,
	})
	if err != nil {
		t.Fatalf("optimistic lock with correct ts should succeed: %v", err)
	}

	// 乐观锁：传入过期的 ExpectedUpdate 应失败
	staleTS := n.UpdatedAt
	_, err = s.Update(UpdateParams{
		ID:             n.ID,
		ContentMD:      strPtr("冲突更新"),
		ExpectedUpdate: &staleTS,
	})
	if err == nil || !strings.Contains(err.Error(), "notes.conflict") {
		t.Fatalf("expected conflict error, got %v", err)
	}
}

func TestListAndFilter(t *testing.T) {
	s := newSvc(t)
	folder1, _ := s.CreateFolder("文件夹1", nil)
	s.Create(CreateParams{Title: "笔记A", ContentMD: "内容A", Tags: []string{"tag1"}, FolderID: &folder1.ID})
	s.Create(CreateParams{Title: "笔记B", ContentMD: "内容B", Tags: []string{"tag2"}})
	s.Create(CreateParams{Title: "笔记C", ContentMD: "内容C", Tags: []string{"tag1", "tag2"}})

	// 全量
	res, err := s.List(ListOptions{Limit: 10})
	if err != nil {
		t.Fatal(err)
	}
	if res.Total != 3 {
		t.Fatalf("total=%d", res.Total)
	}

	// 按文件夹
	res, err = s.List(ListOptions{FolderID: &folder1.ID, Limit: 10})
	if err != nil {
		t.Fatal(err)
	}
	if res.Total != 1 {
		t.Fatalf("folder total=%d", res.Total)
	}

	// 按标签
	tag := "tag1"
	res, err = s.List(ListOptions{Tags: []string{tag}, Limit: 10})
	if err != nil {
		t.Fatal(err)
	}
	if res.Total != 2 {
		t.Fatalf("tag total=%d", res.Total)
	}

	// 按关键字
	res, err = s.List(ListOptions{Keyword: "笔记B", Limit: 10})
	if err != nil {
		t.Fatal(err)
	}
	if res.Total != 1 {
		t.Fatalf("keyword total=%d", res.Total)
	}

	// ListMeta 不含正文
	res, err = s.ListMeta(ListOptions{Limit: 10})
	if err != nil {
		t.Fatal(err)
	}
	for _, it := range res.Items {
		if it.ContentMD != "" {
			t.Fatalf("ListMeta should not return content, got %q", it.ContentMD)
		}
	}
}

func TestSoftDeleteRestoreHardDelete(t *testing.T) {
	s := newSvc(t)
	n, _ := s.Create(CreateParams{Title: "待删除", ContentMD: "内容"})

	// 移入回收站
	if err := s.MoveToTrash(n.ID); err != nil {
		t.Fatal(err)
	}
	got, _ := s.Get(n.ID)
	if !got.IsDeleted {
		t.Fatal("should be deleted")
	}

	// 默认列表不包含已删除
	res, _ := s.List(ListOptions{Limit: 10})
	if res.Total != 0 {
		t.Fatalf("total after delete=%d", res.Total)
	}

	// OnlyDeleted
	res, _ = s.List(ListOptions{OnlyDeleted: true, Limit: 10})
	if res.Total != 1 {
		t.Fatalf("trash total=%d", res.Total)
	}

	// TrashCount
	count, _ := s.TrashCount()
	if count != 1 {
		t.Fatalf("trashCount=%d", count)
	}

	// 恢复
	if err := s.Restore(n.ID); err != nil {
		t.Fatal(err)
	}
	got, _ = s.Get(n.ID)
	if got.IsDeleted {
		t.Fatal("should be restored")
	}

	// HardDelete
	if err := s.HardDelete(n.ID); err != nil {
		t.Fatal(err)
	}
	if _, err := s.Get(n.ID); err == nil {
		t.Fatal("should be gone")
	}
}

func TestEmptyTrash(t *testing.T) {
	s := newSvc(t)
	n1, _ := s.Create(CreateParams{Title: "A", ContentMD: "a"})
	n2, _ := s.Create(CreateParams{Title: "B", ContentMD: "b"})
	_ = s.MoveToTrash(n1.ID)
	_ = s.MoveToTrash(n2.ID)

	count, err := s.EmptyTrash()
	if err != nil {
		t.Fatal(err)
	}
	if count != 2 {
		t.Fatalf("emptied=%d", count)
	}
	trash, _ := s.TrashCount()
	if trash != 0 {
		t.Fatalf("trash after empty=%d", trash)
	}
}

func TestFolderCRUD(t *testing.T) {
	s := newSvc(t)
	parent, err := s.CreateFolder("父文件夹", nil)
	if err != nil {
		t.Fatal(err)
	}
	child, err := s.CreateFolder("子文件夹", &parent.ID)
	if err != nil {
		t.Fatal(err)
	}
 folders, err := s.ListFolders()
	if err != nil {
		t.Fatal(err)
	}
	if len(folders) != 2 {
		t.Fatalf("folders=%d", len(folders))
	}
	_ = child

	// 重命名
	if err := s.UpdateFolder(parent.ID, "新名称"); err != nil {
		t.Fatal(err)
	}
	got, _ := s.ListFolders()
	found := false
	for _, f := range got {
		if f.ID == parent.ID && f.Name == "新名称" {
			found = true
		}
	}
	if !found {
		t.Fatal("rename failed")
	}

	// 删除文件夹
	// 先把笔记挂到父文件夹下
	n, _ := s.Create(CreateParams{Title: "X", ContentMD: "x", FolderID: &parent.ID})
	if err := s.DeleteFolder(parent.ID); err != nil {
		t.Fatal(err)
	}
	// 笔记应被移到根
	got2, _ := s.Get(n.ID)
	if got2.FolderID != nil {
		t.Fatalf("note should be moved to root, folderID=%v", got2.FolderID)
	}
}

func TestAssetCRUD(t *testing.T) {
	s := newSvc(t)
	n, _ := s.Create(CreateParams{Title: "有附件", ContentMD: "内容"})

	a, err := s.AddAsset(n.ID, "test.txt", []byte("hello world"), "text/plain")
	if err != nil {
		t.Fatal(err)
	}
	if a.BlobRef == "" {
		t.Fatal("empty blob ref")
	}

	assets, _ := s.ListAssets(n.ID)
	if len(assets) != 1 {
		t.Fatalf("assets=%d", len(assets))
	}

	// 笔记的资产计数应更新
	got, _ := s.Get(n.ID)
	if !got.HasAssets || got.AssetCount != 1 {
		t.Fatalf("hasAssets=%v assetCount=%d", got.HasAssets, got.AssetCount)
	}

	// 读取资产内容
	data, _, err := s.GetAsset(a.ID)
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != "hello world" {
		t.Fatalf("data=%q", data)
	}

	// 删除资产
	if err := s.DeleteAsset(a.ID); err != nil {
		t.Fatal(err)
	}
	assets, _ = s.ListAssets(n.ID)
	if len(assets) != 0 {
		t.Fatalf("assets after delete=%d", len(assets))
	}
}

func TestImportMarkdownEncodings(t *testing.T) {
	s := newSvc(t)
	cases := []struct {
		name       string
		content    []byte
		wantTitle  string // 期望标题（文件名 stem）
		wantSubstr string // 期望正文子串
	}{
		{"utf8.md", []byte("# UTF-8 笔记\n\n中文内容"), "utf8", "内容"},
		{"utf8bom.md", append([]byte{0xEF, 0xBB, 0xBF}, []byte("# BOM 笔记\n\n内容")...), "utf8bom", "内容"},
		{"utf16le.md", utf16LE("# UTF16LE 笔记\n\n内容"), "utf16le", "内容"},
		{"utf16be.md", utf16BE("# UTF16BE 笔记\n\n内容"), "utf16be", "内容"},
		{"gbk.md", gbkEncode("# GBK 笔记\n\n内容"), "gbk", "内容"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			n, err := s.ImportMarkdown(c.name, c.content, nil)
			if err != nil {
				t.Fatal(err)
			}
			// 标题取文件名 stem（对照 Rust derive_markdown_note_title）
			if n.Title != c.wantTitle {
				t.Fatalf("title=%q want %q", n.Title, c.wantTitle)
			}
			if !strings.Contains(n.ContentMD, c.wantSubstr) {
				t.Fatalf("content=%q want substring %q", n.ContentMD, c.wantSubstr)
			}
		})
	}
}

// TestImportMarkdownGenericTitleFallback 验证：当文件名为通用占位符时，从 H1 提取标题。
func TestImportMarkdownGenericTitleFallback(t *testing.T) {
	s := newSvc(t)
	// 纯数字文件名 → 通用占位符 → 退化到 H1
	n, err := s.ImportMarkdown("12345.md", []byte("# 真实标题\n\n内容"), nil)
	if err != nil {
		t.Fatal(err)
	}
	if n.Title != "真实标题" {
		t.Fatalf("title=%q want H1 fallback", n.Title)
	}
	// 空 stem（只有扩展名）→ 通用占位符 → 退化到 H1
	n2, err := s.ImportMarkdown(".md", []byte("# 另一个标题\n\n内容"), nil)
	if err != nil {
		t.Fatal(err)
	}
	if n2.Title != "另一个标题" {
		t.Fatalf("title=%q want H1 fallback", n2.Title)
	}
	// 通用占位符且无 H1 → 时间戳兜底
	n3, err := s.ImportMarkdown("文件.md", []byte("无标题内容"), nil)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(n3.Title, "导入笔记_") {
		t.Fatalf("title=%q want timestamp fallback", n3.Title)
	}
}

func TestImportBatch(t *testing.T) {
	s := newSvc(t)
	files := map[string][]byte{
		"a.md": []byte("# A\n\n内容A"),
		"b.md": []byte("# B\n\n内容B"),
	}
	notes, err := s.ImportBatch(files, nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(notes) != 2 {
		t.Fatalf("imported=%d", len(notes))
	}
}

func TestExportNote(t *testing.T) {
	s := newSvc(t)
	n, _ := s.Create(CreateParams{Title: "导出测试", ContentMD: "# 标题\n\n**粗体** 内容"})

	// Markdown
	md, err := s.ExportNote(n.ID, ExportMarkdown)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(md), "粗体") {
		t.Fatalf("md=%q", md)
	}

	// HTML
	htmlBytes, err := s.ExportNote(n.ID, ExportHTML)
	if err != nil {
		t.Fatal(err)
	}
	htmlStr := string(htmlBytes)
	if !strings.Contains(htmlStr, "<h1>") {
		t.Fatalf("html missing h1: %q", htmlStr)
	}
	if !strings.Contains(htmlStr, "<strong>粗体</strong>") {
		t.Fatalf("html missing strong: %q", htmlStr)
	}

	// JSON
	jsonBytes, err := s.ExportNote(n.ID, ExportJSON)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(jsonBytes), "导出测试") {
		t.Fatalf("json=%q", jsonBytes)
	}
}

func TestExportAll(t *testing.T) {
	s := newSvc(t)
	s.Create(CreateParams{Title: "N1", ContentMD: "内容1"})
	s.Create(CreateParams{Title: "N2", ContentMD: "内容2"})

	md, err := s.ExportAll(ExportMarkdown)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(md), "N1") || !strings.Contains(string(md), "N2") {
		t.Fatalf("md=%q", md)
	}
	if !strings.Contains(string(md), "---") {
		t.Fatal("missing separator")
	}
}

func TestSearch(t *testing.T) {
	s := newSvc(t)
	s.Create(CreateParams{Title: "贝叶斯定理", ContentMD: "概率论内容"})
	s.Create(CreateParams{Title: "线性代数", ContentMD: "矩阵内容"})

	results, err := s.Search("贝叶斯", 10)
	if err != nil {
		t.Fatal(err)
	}
	if len(results) != 1 {
		t.Fatalf("results=%d", len(results))
	}
	if results[0].Title != "贝叶斯定理" {
		t.Fatalf("title=%q", results[0].Title)
	}
}

func TestStats(t *testing.T) {
	s := newSvc(t)
	s.Create(CreateParams{Title: "A", ContentMD: "a"})
	n2, _ := s.Create(CreateParams{Title: "B", ContentMD: "b"})
	// 置顶 n2
	pinned := true
	s.Update(UpdateParams{ID: n2.ID, IsPinned: &pinned})
	// 移入回收站
	s.MoveToTrash(n2.ID)
	// 添加资产
	n3, _ := s.Create(CreateParams{Title: "C", ContentMD: "c"})
	s.AddAsset(n3.ID, "x.txt", []byte("data"), "text/plain")

	stats, err := s.Stats()
	if err != nil {
		t.Fatal(err)
	}
	// 未删除笔记：A + C = 2
	if stats["total"] != 2 {
		t.Fatalf("total=%d", stats["total"])
	}
	if stats["trash"] != 1 {
		t.Fatalf("trash=%d", stats["trash"])
	}
	// pinned：n2 已被回收，不应计入
	if stats["pinned"] != 0 {
		t.Fatalf("pinned=%d", stats["pinned"])
	}
	if stats["assets"] != 1 {
		t.Fatalf("assets=%d", stats["assets"])
	}
}

func TestWordCount(t *testing.T) {
	cases := []struct {
		content string
		minWord int
		minChar int
	}{
		{"hello world", 2, 10},
		{"你好世界", 4, 4},
		{"Hello 世界", 2, 7},
		{"", 0, 0},
		{"  multiple   spaces  ", 2, 16},
	}
	for _, c := range cases {
		w, ch := countWords(c.content)
		if w < c.minWord {
			t.Fatalf("content=%q word=%d < %d", c.content, w, c.minWord)
		}
		if ch < c.minChar {
			t.Fatalf("content=%q char=%d < %d", c.content, ch, c.minChar)
		}
	}
}

func TestDecodeMarkdownBytesFallback(t *testing.T) {
	// 普通 UTF-8
	if got := decodeMarkdownBytes([]byte("hello")); got != "hello" {
		t.Fatalf("utf8=%q", got)
	}
	// UTF-8 BOM
	if got := decodeMarkdownBytes([]byte{0xEF, 0xBB, 0xBF, 'h', 'i'}); got != "hi" {
		t.Fatalf("bom=%q", got)
	}
	// UTF-16LE with BOM
	if got := decodeMarkdownBytes(utf16LE("AB")); !strings.Contains(got, "AB") {
		t.Fatalf("utf16le=%q", got)
	}
	// GBK
	gbk := gbkEncode("中文")
	if got := decodeMarkdownBytes(gbk); !strings.Contains(got, "中文") {
		t.Fatalf("gbk=%q", got)
	}
}

func TestMarkdownToHTML(t *testing.T) {
	html := markdownToHTML("标题", "# H1\n\n段落 **粗体** 内容\n\n- 列表项")
	if !strings.Contains(html, "<h1>标题</h1>") {
		t.Fatal("missing title h1")
	}
	if !strings.Contains(html, "<h1>H1</h1>") {
		t.Fatal("missing content h1")
	}
	if !strings.Contains(html, "<strong>粗体</strong>") {
		t.Fatal("missing strong")
	}
	if !strings.Contains(html, "<li>列表项</li>") {
		t.Fatal("missing li")
	}
}

// ===================== 测试辅助 =====================

func strPtr(s string) *string { return &s }

// utf16LE 编码字符串并加 BOM。
func utf16LE(s string) []byte {
	runes := []rune(s)
	u16 := make([]uint16, 0, len(runes))
	for _, r := range runes {
		u16 = append(u16, uint16(r))
	}
	out := []byte{0xFF, 0xFE}
	for _, v := range u16 {
		out = append(out, byte(v), byte(v>>8))
	}
	return out
}

// utf16BE 编码字符串并加 BOM。
func utf16BE(s string) []byte {
	runes := []rune(s)
	u16 := make([]uint16, 0, len(runes))
	for _, r := range runes {
		u16 = append(u16, uint16(r))
	}
	out := []byte{0xFE, 0xFF}
	for _, v := range u16 {
		out = append(out, byte(v>>8), byte(v))
	}
	return out
}

// gbkEncode 用 GBK 编码字符串。
func gbkEncode(s string) []byte {
	out, err := simplifiedchinese.GBK.NewEncoder().Bytes([]byte(s))
	if err != nil {
		panic(err)
	}
	return out
}

// 抑制未使用导入
var _ = time.Now
