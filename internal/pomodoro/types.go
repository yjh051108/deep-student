// Package pomodoro 提供番茄钟专注记录：创建/查询/按日统计。
//
// 数据模型对齐 Rust 原版迁移 vfs/V20260310__add_pomodoro.sql
// 与 V20260613__pomodoro_timestamps_and_constraints.sql（pomodoro_records 表），
// 时间字段统一 RFC3339Nano UTC 字符串（带 Z 后缀，满足云同步时间基准要求）。
package pomodoro

import "time"

// RecordType 番茄钟类型。
type RecordType string

const (
	TypeWork       RecordType = "work"
	TypeShortBreak RecordType = "short_break"
	TypeLongBreak  RecordType = "long_break"
)

// RecordStatus 记录状态。
type RecordStatus string

const (
	StatusCompleted   RecordStatus = "completed"
	StatusInterrupted RecordStatus = "interrupted"
)

// DefaultWorkSeconds 默认专注时长（25 分钟）。
const DefaultWorkSeconds = 25 * 60

// Record 番茄钟记录。
type Record struct {
	ID             string       `json:"id"`
	TodoItemID     *string      `json:"todoItemId,omitempty"`
	StartTime      time.Time    `json:"startTime"`
	EndTime        *time.Time   `json:"endTime,omitempty"`
	Duration       int          `json:"duration"`        // 计划时长（秒）
	ActualDuration int          `json:"actualDuration"`  // 实际专注时长（秒）
	Type           RecordType   `json:"type"`
	Status         RecordStatus `json:"status"`
	CreatedAt      time.Time    `json:"createdAt"`
}

// CreateParams 创建记录参数。
type CreateParams struct {
	TodoItemID     *string      `json:"todoItemId,omitempty"`
	StartTime      *time.Time   `json:"startTime,omitempty"` // 默认 now
	Duration       int          `json:"duration,omitempty"`  // 默认 25*60
	ActualDuration int          `json:"actualDuration,omitempty"`
	Type           RecordType   `json:"type,omitempty"`   // 默认 work
	Status         RecordStatus `json:"status,omitempty"` // 默认 completed
}

// Stats 统计结果。
type Stats struct {
	Date             string `json:"date"`             // YYYY-MM-DD（UTC）
	TotalSeconds     int    `json:"totalSeconds"`     // 实际专注总时长
	PlannedSeconds   int    `json:"plannedSeconds"`   // 计划总时长
	CompletedCount   int    `json:"completedCount"`   // 完成个数
	InterruptedCount int    `json:"interruptedCount"` // 中断个数
	WorkCount        int    `json:"workCount"`
	BreakCount       int    `json:"breakCount"`
}

// DailyStat 按日统计条目。
type DailyStat struct {
	Date         string `json:"date"`
	TotalSeconds int    `json:"totalSeconds"`
	Count        int    `json:"count"`
}
