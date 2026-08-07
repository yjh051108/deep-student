// Package todo 提供多列表待办系统：列表/条目/子任务/优先级/截止/提醒/回收站/AI 拆解。
//
// 数据模型对齐 Rust 原版 src-tauri/src/vfs/todo_handlers.rs（todo_lists / todo_items），
// 使用独立 SQLite 表持久化（非内存）。时间字段统一 RFC3339Nano UTC 字符串；
// tags / repeat 使用 JSON 文本存储。
package todo

import "time"

// List 待办列表。
type List struct {
	ID         string     `json:"id"`
	Name       string     `json:"name"`
	Color      string     `json:"color,omitempty"`
	Icon       string     `json:"icon,omitempty"`
	IsInbox    bool       `json:"isInbox"`    // 内置收件箱（不可删除）
	IsFavorite bool       `json:"isFavorite"` // 收藏置顶
	IsDeleted  bool       `json:"isDeleted"`
	DeletedAt  *time.Time `json:"deletedAt,omitempty"`
	SortOrder  int        `json:"sortOrder"`
	CreatedAt  time.Time  `json:"createdAt"`
	UpdatedAt  time.Time  `json:"updatedAt"`
	// ItemCount 统计字段（列表查询时由 service 填充）。
	ItemCount      int `json:"itemCount,omitempty"`
	PendingCount   int `json:"pendingCount,omitempty"`
	CompletedCount int `json:"completedCount,omitempty"`
}

// Priority 优先级（对齐 Rust：0=无, 1=低, 2=中, 3=高）。
type Priority int

// Repeat 重复规则（JSON 存储，如 {"frequency":"daily"} / {"frequency":"weekly","days":[1,3]}）。
type Repeat struct {
	Frequency string `json:"frequency,omitempty"` // none | daily | weekly | monthly | custom
	Days      []int  `json:"days,omitempty"`      // 每周几（0=周日）
	Interval  int    `json:"interval,omitempty"`  // 间隔
}

// Item 待办条目。
type Item struct {
	ID           string     `json:"id"`
	ListID       string     `json:"listId"`
	Title        string     `json:"title"`
	Notes        string     `json:"notes,omitempty"`
	DueAt        *time.Time `json:"dueAt,omitempty"`
	CompletedAt  *time.Time `json:"completedAt,omitempty"`
	Priority     Priority   `json:"priority"`
	Tags         []string   `json:"tags,omitempty"`
	ParentID     *string    `json:"parentId,omitempty"` // 子任务
	EstPomodoros int        `json:"estPomodoros"`
	DonePomodoros int       `json:"donePomodoros"`
	Repeat       *Repeat    `json:"repeat,omitempty"`
	RemindAt     *time.Time `json:"remindAt,omitempty"`
	IsDeleted    bool       `json:"isDeleted"`
	DeletedAt    *time.Time `json:"deletedAt,omitempty"`
	SortOrder    int        `json:"sortOrder"`
	CreatedAt    time.Time  `json:"createdAt"`
	UpdatedAt    time.Time  `json:"updatedAt"`
	// SubCount 统计字段（service 填充）。
	SubCount int `json:"subCount,omitempty"`
}

// CreateListParams 创建列表参数。
type CreateListParams struct {
	Name  string `json:"name"`
	Color string `json:"color,omitempty"`
	Icon  string `json:"icon,omitempty"`
}

// UpdateListParams 更新列表参数（指针字段 nil 表示不修改）。
type UpdateListParams struct {
	ID       string  `json:"id"`
	Name     *string `json:"name,omitempty"`
	Color    *string `json:"color,omitempty"`
	Icon     *string `json:"icon,omitempty"`
	Favorite *bool   `json:"favorite,omitempty"`
}

// CreateItemParams 创建条目参数。
type CreateItemParams struct {
	ListID      string     `json:"listId"`
	Title       string     `json:"title"`
	Notes       string     `json:"notes,omitempty"`
	DueAt       *time.Time `json:"dueAt,omitempty"`
	Priority    Priority   `json:"priority"`
	Tags        []string   `json:"tags,omitempty"`
	ParentID    *string    `json:"parentId,omitempty"`
	EstPomodoros int       `json:"estPomodoros,omitempty"`
	Repeat      *Repeat    `json:"repeat,omitempty"`
	RemindAt    *time.Time `json:"remindAt,omitempty"`
}

// UpdateItemParams 更新条目参数（指针字段 nil 表示不修改）。
type UpdateItemParams struct {
	ID            string     `json:"id"`
	ListID        *string    `json:"listId,omitempty"`
	Title         *string    `json:"title,omitempty"`
	Notes         *string    `json:"notes,omitempty"`
	DueAt         *time.Time `json:"dueAt,omitempty"`
	Priority      *Priority  `json:"priority,omitempty"`
	Tags          *[]string  `json:"tags,omitempty"`
	ParentID      *string    `json:"parentId,omitempty"`
	EstPomodoros  *int       `json:"estPomodoros,omitempty"`
	DonePomodoros *int       `json:"donePomodoros,omitempty"`
	Repeat        *Repeat    `json:"repeat,omitempty"`
	RemindAt      *time.Time `json:"remindAt,omitempty"`
}

// ItemFilter 条目查询过滤。
type ItemFilter string

const (
	FilterAll       ItemFilter = "all"
	FilterPending   ItemFilter = "pending"
	FilterCompleted ItemFilter = "completed"
	FilterDeleted   ItemFilter = "deleted"
)

// Summary 活跃待办总览。
type Summary struct {
	TotalPending   int `json:"totalPending"`
	TotalCompleted int `json:"totalCompleted"`
	OverdueCount   int `json:"overdueCount"`
	DueTodayCount  int `json:"dueTodayCount"`
	Lists          []List `json:"lists"`
}
