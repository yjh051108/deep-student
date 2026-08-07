package plugins

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/helixnow/deep-student-go/pkg/store"
)

func newMgr(t *testing.T) *Manager {
	t.Helper()
	dir := t.TempDir()
	st, err := store.Open(filepath.Join(dir, "x.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = st.Close() })
	return New(st, filepath.Join(dir, "vault"))
}

func TestInstallListEnableUninstall(t *testing.T) {
	m := newMgr(t)
	p, err := m.Install("hello", []byte(`{"name":"Hello","version":"1.0.0","description":"demo","command":"echo hi"}`), map[string][]byte{
		"index.js": []byte("console.log('hi')"),
	})
	if err != nil {
		t.Fatal(err)
	}
	if p.Name != "Hello" || p.Version != "1.0.0" {
		t.Fatalf("plugin=%+v", p)
	}
	// 文件落盘
	if _, err := os.Stat(filepath.Join(m.Dir("hello"), "index.js")); err != nil {
		t.Fatal("plugin file missing")
	}
	if _, err := os.Stat(filepath.Join(m.Dir("hello"), "plugin.json")); err != nil {
		t.Fatal("manifest missing")
	}
	// 列表
	list, err := m.List()
	if err != nil {
		t.Fatal(err)
	}
	if len(list) != 1 {
		t.Fatalf("list=%d", len(list))
	}
	// 启用
	if err := m.SetEnabled("hello", true); err != nil {
		t.Fatal(err)
	}
	got, _ := m.Get("hello")
	if !got.Enabled {
		t.Fatal("not enabled")
	}
	// 卸载
	if err := m.Uninstall("hello"); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(m.Dir("hello")); !os.IsNotExist(err) {
		t.Fatal("plugin dir not removed")
	}
}

func TestInvalidName(t *testing.T) {
	m := newMgr(t)
	if _, err := m.Install("../evil", nil, nil); err == nil {
		t.Fatal("path traversal name should error")
	}
	if _, err := m.Install("a/b", nil, nil); err == nil {
		t.Fatal("slash name should error")
	}
}

func TestScanVault(t *testing.T) {
	m := newMgr(t)
	// 手工放一个外部插件目录
	dir := filepath.Join(m.pluginsDir(), "external")
	os.MkdirAll(dir, 0o755)
	os.WriteFile(filepath.Join(dir, "plugin.json"), []byte(`{"name":"External","version":"0.1.0"}`), 0o644)
	found, err := m.ScanVault()
	if err != nil {
		t.Fatal(err)
	}
	if len(found) != 1 || found[0] != "external" {
		t.Fatalf("found=%v", found)
	}
	// 已登记 → 不再重复
	found2, _ := m.ScanVault()
	if len(found2) != 0 {
		t.Fatalf("dup found=%v", found2)
	}
}
