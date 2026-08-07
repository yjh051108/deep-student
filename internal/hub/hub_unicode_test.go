package hub

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/helixnow/deep-student-go/pkg/store"
	"github.com/helixnow/deep-student-go/pkg/store/blob"
	"github.com/helixnow/deep-student-go/pkg/vfs"
)

// TestUnicodePathEnd2End BUG-007 回归：把数据目录放在 "中文 目录" / "with space" 路径下，
// 跑一次完整的 store + vfs + hub 流水线：open 库、put 中文笔记、list、get、search、delete，
// 然后 close 重开验证 SQLite 文件能正常 reload（中文/空格/unicode 路径都跑通）。
func TestUnicodePathEnd2End(t *testing.T) {
	// 1) 在 tmp 下创建"中文 目录"和"with space"两层
	root := t.TempDir()
	cnDir := filepath.Join(root, "中文 目录 with space", "αβγ")
	if err := os.MkdirAll(cnDir, 0o755); err != nil {
		t.Fatal(err)
	}

	// 2) blob 目录同样含中文/空格
	blobDir := filepath.Join(cnDir, "blob 块")
	if err := os.MkdirAll(blobDir, 0o755); err != nil {
		t.Fatal(err)
	}

	// 3) SQLite 文件名也带中文 + 空格
	dbPath := filepath.Join(cnDir, "学习 data.db")

	// 4) 打开 store + blob + vfs
	bs, err := blob.New(blobDir)
	if err != nil {
		t.Fatal(err)
	}
	st, err := store.Open(dbPath)
	if err != nil {
		t.Fatal(err)
	}
	fs := vfs.NewFS(bs)
	s := New(fs, st, nil)

	// 5) Put 一批中文 + 含 # + 含空格的 uri
	uris := []string{
		"vfs://note/笔记 A",
		"vfs://note/笔记 B#tag1",
		"vfs://note/with space",
		"vfs://textbook/教材 α",
	}
	bodies := [][]byte{
		[]byte(`{"text":"今天学到了 Go 内存模型"}`),
		[]byte(`{"text":"含 # 字符的 id 应当原样存储"}`),
		[]byte(`hello world`),
		[]byte(`{"title":"αβγδ 教材"}`),
	}
	for i, u := range uris {
		if _, err := fs.Put(u, bodies[i], map[string]string{"title": idstrUnicode(i), "tags": "中文,with space"}); err != nil {
			t.Fatal(err)
		}
		// 同步把元数据写一份到 store（hub.ImportResource 走的就是这条路；这里手动写
		// 是为了保留自定义 uri，让 Get/ListResources 行为一致）。
		e, _, err := fs.Get(u)
		if err != nil {
			t.Fatal(err)
		}
		_ = e
		if err := st.SaveResource(u, strings.SplitN(u, "/", 4)[2], idstrUnicode(i), idstrUnicode(i), "中文,with space", "{}", "", int64(len(bodies[i])), 1); err != nil {
			t.Fatal(err)
		}
	}

	// 6) List 验证
	if got := len(s.List(vfs.TypeNote)); got < 3 {
		t.Fatalf("note list len=%d, want >=3", got)
	}

	// 7) Get 验证中文/空格 id 都能取回
	for i, u := range uris {
		data, _, err := s.Get(u)
		if err != nil {
			t.Fatalf("get %s: %v", u, err)
		}
		if !bytes.Equal(data, bodies[i]) {
			t.Fatalf("body mismatch for %s: got %q want %q", u, data, bodies[i])
		}
	}

	// 8) Search 验证（按 tag 搜中文）
	hits := s.Search(vfs.TypeNote, "中文")
	if len(hits) < 1 {
		t.Fatalf("expected at least 1 hit for tag 中文, got %d", len(hits))
	}

	// 9) Delete 验证
	if err := s.Delete(uris[3]); err != nil {
		t.Fatalf("delete: %v", err)
	}
	if _, _, err := s.Get(uris[3]); err == nil {
		t.Fatalf("expected error after delete")
	}

	// 10) close + 重新 open 同一路径，验证 SQLite 文件能被一致读出
	_ = st.Close()
	st2, err := store.Open(dbPath)
	if err != nil {
		t.Fatalf("reopen store: %v", err)
	}
	t.Cleanup(func() { _ = st2.Close() })
	rows, err := st2.ListResources("")
	if err != nil {
		t.Fatalf("list resources: %v", err)
	}
	if len(rows) < 1 {
		t.Fatalf("expected >=1 resource row after reopen, got %d", len(rows))
	}

	// 11) 直接 os.Stat 校验文件确实在中文/空格路径下
	if _, err := os.Stat(dbPath); err != nil {
		t.Fatalf("db file missing at %s: %v", dbPath, err)
	}
	// 校验目录路径含中文字符
	if !strings.Contains(cnDir, "中文") {
		t.Fatalf("path not preserving Chinese: %s", cnDir)
	}
}

// idstrUnicode 把 int 编码成 base-36 风格字符串，避免导入 strconv。
func idstrUnicode(i int) string {
	if i == 0 {
		return "0"
	}
	var buf [20]byte
	pos := len(buf)
	for i > 0 {
		pos--
		buf[pos] = byte('0' + i%10)
		i /= 10
	}
	return string(buf[pos:])
}
