// Package notes 提供独立的笔记系统：CRUD、回收站、文件夹、附件、导入导出。
//
// 设计参考 Rust 原版 src-tauri/src/notes_manager.rs 与 vfs/repos/note_repo.rs，
// 但使用独立 SQLite 表（不复用 resources 表），简化版本管理与向量索引，
// 聚焦于 P1 笔记能力的完整复刻。
package notes

import "time"

// Note 笔记主体
type Note struct {
	ID         string            `json:"id"`
	Title      string            `json:"title"`
	ContentMD  string            `json:"contentMd"` // Markdown 正文
	Tags       []string          `json:"tags"`
	FolderID   *string           `json:"folderId,omitempty"` // 所属文件夹
	HasAssets  bool              `json:"hasAssets"`
	AssetCount int               `json:"assetCount"`
	IsPinned   bool              `json:"isPinned"`
	IsDeleted  bool              `json:"isDeleted"` // 回收站标志
	DeletedAt  *time.Time        `json:"deletedAt,omitempty"`
	WordCount  int               `json:"wordCount"`
	CharCount  int               `json:"charCount"`
	CreatedAt  time.Time         `json:"createdAt"`
	UpdatedAt  time.Time         `json:"updatedAt"`
	Metadata   map[string]string `json:"metadata,omitempty"`
}

// Folder 文件夹
type Folder struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	ParentID  *string   `json:"parentId,omitempty"`
	SortOrder int       `json:"sortOrder"`
	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
}

// Asset 笔记附件
type Asset struct {
	ID        string    `json:"id"`
	NoteID    string    `json:"noteId"`
	Filename  string    `json:"filename"`
	MIMEType  string    `json:"mimeType"`
	Size      int64     `json:"size"`
	BlobRef   string    `json:"blobRef"`
	CreatedAt time.Time `json:"createdAt"`
}

// ListOptions 列表查询选项
type ListOptions struct {
	FolderID       *string   `json:"folderId,omitempty"`
	Tags           []string  `json:"tags,omitempty"`
	Keyword        string    `json:"keyword,omitempty"`
	IncludeDeleted bool      `json:"includeDeleted,omitempty"`
	OnlyDeleted    bool      `json:"onlyDeleted,omitempty"`
	HasAssets      *bool     `json:"hasAssets,omitempty"`
	DateStart      *time.Time `json:"dateStart,omitempty"`
	DateEnd        *time.Time `json:"dateEnd,omitempty"`
	SortBy         string    `json:"sortBy,omitempty"` // "updated" | "created" | "title" | "wordCount"
	SortDir        string    `json:"sortDir,omitempty"` // "asc" | "desc"
	Limit          int       `json:"limit,omitempty"`
	Offset         int       `json:"offset,omitempty"`
}

// ListResult 列表结果
type ListResult struct {
	Items  []Note `json:"items"`
	Total  int64  `json:"total"`
	Limit  int    `json:"limit"`
	Offset int    `json:"offset"`
}

// CreateParams 创建参数
type CreateParams struct {
	Title     string   `json:"title"`
	ContentMD string   `json:"contentMd"`
	Tags      []string `json:"tags"`
	FolderID  *string  `json:"folderId,omitempty"`
}

// UpdateParams 更新参数（部分字段可空）
type UpdateParams struct {
	ID             string     `json:"id"`
	Title          *string    `json:"title,omitempty"`
	ContentMD      *string    `json:"contentMd,omitempty"`
	Tags           *[]string  `json:"tags,omitempty"`
	FolderID       *string    `json:"folderId,omitempty"`
	IsPinned       *bool      `json:"isPinned,omitempty"`
	ExpectedUpdate *time.Time `json:"expectedUpdate,omitempty"` // 乐观锁
}

// ExportFormat 导出格式
type ExportFormat string

const (
	ExportMarkdown ExportFormat = "markdown"
	ExportHTML     ExportFormat = "html"
	ExportJSON     ExportFormat = "json"
)
