package templatemgr

import (
	"encoding/json"
	"path/filepath"
	"strings"
	"testing"

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

func TestSeedBuiltins(t *testing.T) {
	s := newSvc(t)
	ts, err := s.List()
	if err != nil {
		t.Fatal(err)
	}
	if len(ts) < 4 {
		t.Fatalf("builtins=%d", len(ts))
	}
	// 幂等：重建服务不重复 seed
	s2 := newSvc(t)
	ts2, _ := s2.List()
	if len(ts2) != len(ts) {
		t.Fatalf("seed not idempotent: %d vs %d", len(ts2), len(ts))
	}
}

func TestCRUD(t *testing.T) {
	s := newSvc(t)
	tpl, err := s.Create(CreateParams{
		Name: "我的模板", FrontTmpl: "<div>{{Front}}</div>", BackTmpl: "<div>{{Back}}</div>",
		SharedCSS: ".card {}",
	})
	if err != nil {
		t.Fatal(err)
	}
	if tpl.Name != "我的模板" {
		t.Fatalf("name=%s", tpl.Name)
	}
	// 更新
	nn := "改名"
	upd, err := s.Update(UpdateParams{ID: tpl.ID, Name: &nn})
	if err != nil {
		t.Fatal(err)
	}
	if upd.Name != "改名" {
		t.Fatalf("updated=%s", upd.Name)
	}
	// 校验必填
	if _, err := s.Create(CreateParams{Name: "", FrontTmpl: "x", BackTmpl: "y"}); err == nil {
		t.Fatal("empty name should error")
	}
	// 删除
	if err := s.Delete(tpl.ID); err != nil {
		t.Fatal(err)
	}
	// 内置保护
	if err := s.Delete("default"); err == nil {
		t.Fatal("builtin delete should error")
	}
}

func TestExportImport(t *testing.T) {
	s := newSvc(t)
	tpl, _ := s.Create(CreateParams{Name: "导出模板", FrontTmpl: "F", BackTmpl: "B", Style: "S", SharedCSS: "C"})
	data, err := s.Export(tpl.ID)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(data), "导出模板") {
		t.Fatalf("export=%s", string(data))
	}
	// 导入
	imp, err := s.Import(data)
	if err != nil {
		t.Fatal(err)
	}
	if imp.Name != "导出模板" || imp.SharedCSS != "C" {
		t.Fatalf("imported=%+v", imp)
	}
	// 批量导入
	bulk := []ImportEntry{
		{Name: "A", FrontTmpl: "f1", BackTmpl: "b1"},
		{Name: "B", FrontTmpl: "f2", BackTmpl: "b2"},
	}
	data2, _ := json.Marshal(bulk)
	imported, failed, err := s.ImportBulk(data2)
	if err != nil {
		t.Fatal(err)
	}
	if imported != 2 || failed != 0 {
		t.Fatalf("imported=%d failed=%d", imported, failed)
	}
}

func TestDefaultTemplate(t *testing.T) {
	s := newSvc(t)
	id, err := s.DefaultID()
	if err != nil {
		t.Fatal(err)
	}
	if id != "default" {
		t.Fatalf("default=%s", id)
	}
	tpl, err := s.Create(CreateParams{Name: "新默认", FrontTmpl: "F", BackTmpl: "B"})
	if err != nil {
		t.Fatal(err)
	}
	if err := s.SetDefault(tpl.ID); err != nil {
		t.Fatal(err)
	}
	id2, _ := s.DefaultID()
	if id2 != tpl.ID {
		t.Fatalf("default2=%s", id2)
	}
	// Template("") 返回默认
	got, err := s.Template("")
	if err != nil {
		t.Fatal(err)
	}
	if got.ID != tpl.ID {
		t.Fatalf("template()=%s", got.ID)
	}
	// 删除默认后回退
	if err := s.Delete(tpl.ID); err != nil {
		t.Fatal(err)
	}
	id3, _ := s.DefaultID()
	if id3 != "default" {
		t.Fatalf("fallback default=%s", id3)
	}
}

func TestImportBuiltins(t *testing.T) {
	s := newSvc(t)
	// 用 SQL 直接删掉内置模板（绕过保护），验证 ImportBuiltins 补回
	if _, err := s.store.db.Exec(`DELETE FROM custom_anki_templates WHERE id='cloze'`); err != nil {
		t.Fatal(err)
	}
	added, err := s.ImportBuiltins()
	if err != nil {
		t.Fatal(err)
	}
	if added != 1 {
		t.Fatalf("added=%d (expect 1)", added)
	}
	// 再次导入不重复
	added2, _ := s.ImportBuiltins()
	if added2 != 0 {
		t.Fatalf("dup added=%d", added2)
	}
}
