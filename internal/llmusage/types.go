// Package llmusage 提供 LLM 调用用量统计：调用日志 + 按日聚合。
//
// 表结构对齐 Rust 原版迁移 llm_usage/V20260130__init.sql（llm_usage_logs /
// llm_usage_daily），时间字段 RFC3339Nano UTC；date_key/hour_key 为生成列。
package llmusage

import "time"

// Status 调用状态。
type Status string

const (
	StatusSuccess  Status = "success"
	StatusError    Status = "error"
	StatusTimeout  Status = "timeout"
	StatusCanceled Status = "cancelled"
)

// TokenSource token 来源。
type TokenSource string

const (
	TokenSourceAPI       TokenSource = "api"
	TokenSourceEstimated TokenSource = "estimated"
	TokenSourceTiktoken  TokenSource = "tiktoken"
)

// Log 单次 LLM 调用日志。
type Log struct {
	ID              string      `json:"id"`
	Timestamp       time.Time   `json:"timestamp"`
	Provider        string      `json:"provider"`
	Model           string      `json:"model"`
	Adapter         string      `json:"adapter,omitempty"`
	APIConfigID     string      `json:"apiConfigId,omitempty"`
	PromptTokens    int         `json:"promptTokens"`
	CompletionTokens int        `json:"completionTokens"`
	TotalTokens     int         `json:"totalTokens"`
	ReasoningTokens *int        `json:"reasoningTokens,omitempty"`
	CachedTokens    *int        `json:"cachedTokens,omitempty"`
	TokenSource     TokenSource `json:"tokenSource"`
	DurationMs      *int64      `json:"durationMs,omitempty"`
	RequestBytes    *int64      `json:"requestBytes,omitempty"`
	ResponseBytes   *int64      `json:"responseBytes,omitempty"`
	FirstTokenMs    *int64      `json:"firstTokenMs,omitempty"`
	CallerType      string      `json:"callerType"`
	SessionID       string      `json:"sessionId,omitempty"`
	Status          Status      `json:"status"`
	ErrorMessage    string      `json:"errorMessage,omitempty"`
	CostEstimate    *float64    `json:"costEstimate,omitempty"`
}

// DailyAggregate 按日聚合（llm_usage_daily 行）。
type DailyAggregate struct {
	Date                string `json:"date"`
	CallerType          string `json:"callerType"`
	Model               string `json:"model"`
	Provider            string `json:"provider"`
	RequestCount        int    `json:"requestCount"`
	SuccessCount        int    `json:"successCount"`
	ErrorCount          int    `json:"errorCount"`
	TotalPromptTokens   int    `json:"totalPromptTokens"`
	TotalCompletionTokens int  `json:"totalCompletionTokens"`
	TotalTokens         int    `json:"totalTokens"`
	TotalReasoningTokens int   `json:"totalReasoningTokens"`
	TotalCachedTokens   int    `json:"totalCachedTokens"`
	TotalCostEstimate   float64 `json:"totalCostEstimate"`
	TotalDurationMs     int64  `json:"totalDurationMs"`
}

// LogEntry 写日志参数。
type LogEntry struct {
	Provider         string      `json:"provider"`
	Model            string      `json:"model"`
	Adapter          string      `json:"adapter,omitempty"`
	APIConfigID      string      `json:"apiConfigId,omitempty"`
	PromptTokens     int         `json:"promptTokens"`
	CompletionTokens int         `json:"completionTokens"`
	ReasoningTokens  *int        `json:"reasoningTokens,omitempty"`
	CachedTokens     *int        `json:"cachedTokens,omitempty"`
	TokenSource      TokenSource `json:"tokenSource,omitempty"`
	DurationMs       *int64      `json:"durationMs,omitempty"`
	RequestBytes     *int64      `json:"requestBytes,omitempty"`
	ResponseBytes    *int64      `json:"responseBytes,omitempty"`
	FirstTokenMs     *int64      `json:"firstTokenMs,omitempty"`
	CallerType       string      `json:"callerType"`
	SessionID        string      `json:"sessionId,omitempty"`
	Status           Status      `json:"status,omitempty"`
	ErrorMessage     string      `json:"errorMessage,omitempty"`
	CostEstimate     *float64    `json:"costEstimate,omitempty"`
}

// Summary 总览。
type Summary struct {
	TotalRequests      int   `json:"totalRequests"`
	TotalTokens        int   `json:"totalTokens"`
	TotalPromptTokens  int   `json:"totalPromptTokens"`
	TotalCompletionTokens int `json:"totalCompletionTokens"`
	TotalCost          float64 `json:"totalCost"`
	TodayRequests      int   `json:"todayRequests"`
	TodayTokens        int   `json:"todayTokens"`
	Last7DaysRequests  int   `json:"last7DaysRequests"`
	Last7DaysTokens    int   `json:"last7DaysTokens"`
}
