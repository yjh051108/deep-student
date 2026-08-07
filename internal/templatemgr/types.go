// Package templatemgr 提供 Anki 自定义模板管理：CRUD / 导入导出 / 内置模板。
//
// 对齐 Rust 原版 custom_anki_templates 表（migrations/vfs）与
// commands.rs 的 *_custom_template 命令。模板持久化到 SQLite，
// anki 制卡服务可通过 Service.Template(id) 取用。
package templatemgr

import "time"

// Template Anki 模板（与 internal/anki.Template 结构兼容 + 管理字段）。
type Template struct {
	ID         string    `json:"id"`
	Name       string    `json:"name"`
	FrontTmpl  string    `json:"front"`
	BackTmpl   string    `json:"back"`
	Style      string    `json:"style"`
	SharedCSS  string    `json:"css"`
	IsBuiltin  bool      `json:"isBuiltin"`  // 内置模板不可删除
	Preview    string    `json:"preview,omitempty"` // 预览数据
	SortOrder  int       `json:"sortOrder"`
	CreatedAt  time.Time `json:"createdAt"`
	UpdatedAt  time.Time `json:"updatedAt"`
}

// CreateParams 创建参数。
type CreateParams struct {
	Name      string `json:"name"`
	FrontTmpl string `json:"front"`
	BackTmpl  string `json:"back"`
	Style     string `json:"style,omitempty"`
	SharedCSS string `json:"css,omitempty"`
	Preview   string `json:"preview,omitempty"`
}

// UpdateParams 更新参数（nil 字段不修改）。
type UpdateParams struct {
	ID        string  `json:"id"`
	Name      *string `json:"name,omitempty"`
	FrontTmpl *string `json:"front,omitempty"`
	BackTmpl  *string `json:"back,omitempty"`
	Style     *string `json:"style,omitempty"`
	SharedCSS *string `json:"css,omitempty"`
	Preview   *string `json:"preview,omitempty"`
}

// ImportEntry 导入模板的 JSON 结构。
type ImportEntry struct {
	Name      string `json:"name"`
	FrontTmpl string `json:"front"`
	BackTmpl  string `json:"back"`
	Style     string `json:"style,omitempty"`
	SharedCSS string `json:"css,omitempty"`
	Preview   string `json:"preview,omitempty"`
}
