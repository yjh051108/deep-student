// Package plugins 提供受管插件/扩展生态：安装、启用/禁用、列出、执行。
//
// 对齐 Rust 原版 src-tauri/src/plugins/ 的设计：插件以目录形式存在于
// vault/plugins/<name>/，含 plugin.json 清单与可执行逻辑（命令脚本）。
// 受管扩展支持启用/禁用状态持久化。

package plugins

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/helixnow/deep-student-go/pkg/store"
)

// Plugin 插件元数据。
type Plugin struct {
	ID          string    `json:"id"`
	Name        string    `json:"name"`
	Version     string    `json:"version"`
	Description string    `json:"description"`
	Command     string    `json:"command"`    // 执行命令（如 node index.js）
	Dir         string    `json:"dir"`        // 插件目录（相对 vault/plugins）
	Enabled     bool      `json:"enabled"`
	InstalledAt time.Time `json:"installedAt"`
}

// Manager 插件管理器。
type Manager struct {
	db      *sql.DB
	vaultDir string
}

// New 构造并建表。
func New(st *store.Store, vaultDir string) *Manager {
	m := &Manager{vaultDir: vaultDir}
	if st != nil && st.DB != nil {
		m.db = st.DB
		if _, err := m.db.Exec(`CREATE TABLE IF NOT EXISTS managed_plugins (
			id TEXT PRIMARY KEY,
			name TEXT NOT NULL,
			version TEXT NOT NULL DEFAULT '0.1.0',
			description TEXT NOT NULL DEFAULT '',
			command TEXT NOT NULL DEFAULT '',
			dir TEXT NOT NULL,
			enabled INTEGER NOT NULL DEFAULT 0,
			installed_at TEXT NOT NULL
		)`); err != nil {
			fmt.Printf("[plugins] migrate: %v\n", err)
		}
	}
	return m
}

// pluginsDir 插件根目录。
func (m *Manager) pluginsDir() string {
	if m.vaultDir == "" {
		return "plugins"
	}
	return filepath.Join(m.vaultDir, "plugins")
}

// Install 安装插件（manifest JSON + 可选内容目录）。
// manifest: {name, version, description, command}
func (m *Manager) Install(name string, manifestJSON []byte, files map[string][]byte) (*Plugin, error) {
	if name == "" || strings.ContainsAny(name, `/\`) {
		return nil, errors.New("plugins: invalid name")
	}
	var meta struct {
		Name        string `json:"name"`
		Version     string `json:"version"`
		Description string `json:"description"`
		Command     string `json:"command"`
	}
	if len(manifestJSON) > 0 {
		if err := json.Unmarshal(manifestJSON, &meta); err != nil {
			return nil, fmt.Errorf("plugins: bad manifest: %w", err)
		}
	}
	if meta.Name == "" {
		meta.Name = name
	}
	dir := filepath.Join(m.pluginsDir(), name)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return nil, err
	}
	// 写插件文件
	for path, data := range files {
		target := filepath.Join(dir, filepath.Clean(path))
		// 防目录穿越
		if !strings.HasPrefix(target, dir) {
			continue
		}
		if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
			return nil, err
		}
		if err := os.WriteFile(target, data, 0o644); err != nil {
			return nil, err
		}
	}
	// 写 manifest
	man, _ := json.MarshalIndent(meta, "", "  ")
	_ = os.WriteFile(filepath.Join(dir, "plugin.json"), man, 0o644)

	p := &Plugin{
		ID: name, Name: meta.Name, Version: meta.Version, Description: meta.Description,
		Command: meta.Command, Dir: name, Enabled: false, InstalledAt: time.Now().UTC(),
	}
	if m.db != nil {
		if _, err := m.db.Exec(`INSERT OR REPLACE INTO managed_plugins(id, name, version, description, command, dir, enabled, installed_at)
			VALUES (?,?,?,?,?,?,?,?)`,
			p.ID, p.Name, p.Version, p.Description, p.Command, p.Dir, 0, p.InstalledAt.UTC().Format(time.RFC3339Nano)); err != nil {
			return nil, err
		}
	}
	return p, nil
}

// List 列出插件。
func (m *Manager) List() ([]*Plugin, error) {
	if m.db == nil {
		return nil, nil
	}
	rows, err := m.db.Query(`SELECT id, name, version, description, command, dir, enabled, installed_at
		FROM managed_plugins ORDER BY name ASC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []*Plugin
	for rows.Next() {
		var p Plugin
		var installed string
		var enabled int
		if err := rows.Scan(&p.ID, &p.Name, &p.Version, &p.Description, &p.Command,
			&p.Dir, &enabled, &installed); err != nil {
			return nil, err
		}
		p.Enabled = enabled != 0
		p.InstalledAt = parseTime(installed)
		out = append(out, &p)
	}
	return out, rows.Err()
}

// Get 读取插件。
func (m *Manager) Get(id string) (*Plugin, error) {
	if m.db == nil {
		return nil, errors.New("plugins: db not ready")
	}
	row := m.db.QueryRow(`SELECT id, name, version, description, command, dir, enabled, installed_at
		FROM managed_plugins WHERE id=?`, id)
	var p Plugin
	var installed string
	var enabled int
	if err := row.Scan(&p.ID, &p.Name, &p.Version, &p.Description, &p.Command,
		&p.Dir, &enabled, &installed); err != nil {
		return nil, fmt.Errorf("plugins: not found: %s", id)
	}
	p.Enabled = enabled != 0
	p.InstalledAt = parseTime(installed)
	return &p, nil
}

// SetEnabled 启用/禁用插件。
func (m *Manager) SetEnabled(id string, enabled bool) error {
	if m.db == nil {
		return errors.New("plugins: db not ready")
	}
	_, err := m.db.Exec(`UPDATE managed_plugins SET enabled=? WHERE id=?`, boolToInt(enabled), id)
	return err
}

// Uninstall 卸载插件（删记录 + 目录）。
func (m *Manager) Uninstall(id string) error {
	if m.db != nil {
		if _, err := m.db.Exec(`DELETE FROM managed_plugins WHERE id=?`, id); err != nil {
			return err
		}
	}
	return os.RemoveAll(filepath.Join(m.pluginsDir(), id))
}

// Dir 返回插件目录绝对路径。
func (m *Manager) Dir(id string) string {
	return filepath.Join(m.pluginsDir(), id)
}

// ScanVault 扫描 vault/plugins 下已安装但未登记到库的插件（外部安装）。
func (m *Manager) ScanVault() ([]string, error) {
	root := m.pluginsDir()
	if _, err := os.Stat(root); os.IsNotExist(err) {
		return nil, nil
	}
	entries, err := os.ReadDir(root)
	if err != nil {
		return nil, err
	}
	var found []string
	for _, e := range entries {
		if !e.IsDir() || strings.HasPrefix(e.Name(), ".") {
			continue
		}
		if _, err := os.Stat(filepath.Join(root, e.Name(), "plugin.json")); err == nil {
			// 未登记则登记
			if m.db != nil {
				var cnt int
				_ = m.db.QueryRow(`SELECT COUNT(*) FROM managed_plugins WHERE id=?`, e.Name()).Scan(&cnt)
				if cnt == 0 {
					man, _ := os.ReadFile(filepath.Join(root, e.Name(), "plugin.json"))
					if _, err := m.Install(e.Name(), man, nil); err == nil {
						found = append(found, e.Name())
					}
				}
			}
		}
	}
	return found, nil
}

func boolToInt(b bool) int {
	if b {
		return 1
	}
	return 0
}

func parseTime(s string) time.Time {
	t, err := time.Parse(time.RFC3339Nano, s)
	if err != nil {
		t, err = time.Parse(time.RFC3339, s)
		if err != nil {
			return time.Time{}
		}
	}
	return t
}
