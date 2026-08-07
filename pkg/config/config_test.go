package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadDefaults(t *testing.T) {
	Reset()
	t.Setenv("DEEPSTUDENT_DATA", "")
	cfg, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	if cfg.AppName == "" {
		t.Fatal("AppName empty")
	}
	if cfg.DataDir == "" {
		t.Fatal("DataDir empty")
	}
	if cfg.LogLevel == "" {
		t.Fatal("LogLevel empty")
	}
}

func TestLoadOverride(t *testing.T) {
	Reset()
	dir := filepath.Join(os.TempDir(), "dsgo-test")
	t.Setenv("DEEPSTUDENT_DATA", dir)
	cfg, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	if cfg.DataDir != dir {
		t.Fatalf("expected %s, got %s", dir, cfg.DataDir)
	}
}

// TestLoadCreatesDirs BUG-005 回归：DataDir 不存在时 Load 主动 MkdirAll
// 创建所有关键子目录；中文 / 空格 / unicode 路径同样要可创建。
func TestLoadCreatesDirs(t *testing.T) {
	Reset()
	// 用 t.TempDir() 的子路径制造"多级不存在"的情况
	parent := t.TempDir()
	dataDir := filepath.Join(parent, "deepstudent-go", "data")
	t.Setenv("DEEPSTUDENT_DATA", dataDir)
	cfg, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	want := []string{
		cfg.DataDir,
		cfg.CacheDir,
		cfg.LogDir,
		cfg.BackupDir,
		filepath.Join(cfg.DataDir, "blob"),
		filepath.Join(cfg.DataDir, "vector"),
	}
	for _, d := range want {
		st, err := os.Stat(d)
		if err != nil {
			t.Fatalf("missing dir %s: %v", d, err)
		}
		if !st.IsDir() {
			t.Fatalf("%s is not a directory", d)
		}
	}
}

// TestLoadCreatesDirsUnicode BUG-005 回归（中文/空格/unicode）：
// 当 DataDir 含中文或空格时，Load 仍能 MkdirAll 出所有子目录。
func TestLoadCreatesDirsUnicode(t *testing.T) {
	Reset()
	parent := t.TempDir()
	// 真实 Windows / macOS / Linux 文件系统都支持中文 + 空格。
	dataDir := filepath.Join(parent, "中文 目录 with space", "αβγ")
	t.Setenv("DEEPSTUDENT_DATA", dataDir)
	cfg, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(cfg.DataDir); err != nil {
		t.Fatalf("data dir not created: %v", err)
	}
	if _, err := os.Stat(filepath.Join(cfg.DataDir, "blob")); err != nil {
		t.Fatalf("blob dir not created: %v", err)
	}
}
