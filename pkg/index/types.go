// Package index 提供 P0-B 索引系统：FTS5 全文索引 + 向量嵌入 + RAG 检索。
//
// 100% 复刻 Rust 原版 src-tauri/src/vfs 的索引能力，
// 使用 modernc.org/sqlite（纯 Go，无 cgo）内置 FTS5 扩展。
package index

import "time"

// Chunk 切片单元。
type Chunk struct {
	ID         int64     `json:"id"`
	URI        string    `json:"uri"`
	Pos        int       `json:"pos"` // 切片在原文中的位置
	Content    string    `json:"content"`
	TokenCount int       `json:"tokenCount"`
	Embedding  []float32 `json:"embedding,omitempty"`
	CreatedAt  time.Time `json:"createdAt"`
}

// IndexOptions 索引选项。
type IndexOptions struct {
	ChunkSize    int    // 切片大小（字符数），默认 800
	ChunkOverlap int    // 切片重叠（字符数），默认 100
	MinChunkSize int    // 最小切片大小，默认 50
	Embed        bool   // 是否生成向量嵌入
	EmbedModel   string // 嵌入模型 ID（如 "text-embedding-3-small"）
}

// DefaultOptions 默认选项。
func DefaultOptions() IndexOptions {
	return IndexOptions{
		ChunkSize:    800,
		ChunkOverlap: 100,
		MinChunkSize: 50,
		Embed:        false,
		EmbedModel:   "",
	}
}

// SearchQuery 搜索查询。
type SearchQuery struct {
	Query     string   `json:"query"`
	Types     []string `json:"types,omitempty"`  // 资源类型过滤
	Limit     int      `json:"limit"`            // 默认 10
	MinScore  float64  `json:"minScore"`         // 默认 0.0
	UseFTS    bool     `json:"useFts"`           // 使用 FTS
	UseVector bool     `json:"useVector"`        // 使用向量
	UseRerank bool     `json:"useRerank"`        // 是否重排序
}

// DefaultSearchQuery 返回默认搜索查询（FTS + 向量混合）。
func DefaultSearchQuery(query string) SearchQuery {
	return SearchQuery{
		Query:     query,
		Limit:     10,
		MinScore:  0.0,
		UseFTS:    true,
		UseVector: true,
		UseRerank: false,
	}
}

// SearchResult 搜索结果。
type SearchResult struct {
	URI      string  `json:"uri"`
	Type     string  `json:"type"`
	Title    string  `json:"title"`
	Snippet  string  `json:"snippet"` // 高亮片段
	Score    float64 `json:"score"`   // 综合得分 0-1
	FTSScore float64 `json:"ftsScore,omitempty"`
	VecScore float64 `json:"vecScore,omitempty"`
	ChunkPos int     `json:"chunkPos,omitempty"`
}

// IndexStats 索引统计。
type IndexStats struct {
	TotalResources   int `json:"totalResources"`
	IndexedResources int `json:"indexedResources"`
	TotalChunks      int `json:"totalChunks"`
	EmbeddedChunks   int `json:"embeddedChunks"`
	FTSRows          int `json:"ftsRows"`
	AvgChunkTokens   int `json:"avgChunkTokens"`
}

// IndexTask 索引任务状态。
type IndexTask struct {
	URI       string  `json:"uri"`
	Status    string  `json:"status"` // "pending" | "indexing" | "embedded" | "done" | "error"
	Progress  float64 `json:"progress"`
	Error     string  `json:"error,omitempty"`
	StartedAt int64   `json:"startedAt"`
}

// BatchIndexResult 批量索引结果。
type BatchIndexResult struct {
	Total      int          `json:"total"`
	Success    int          `json:"success"`
	Failed     int          `json:"failed"`
	Skipped    int          `json:"skipped"`
	Tasks      []IndexTask  `json:"tasks"`
	Errors     []IndexError `json:"errors,omitempty"`
}

// IndexError 索引错误记录。
type IndexError struct {
	URI   string `json:"uri"`
	Error string `json:"error"`
}

// RAGContext RAG 检索上下文。
type RAGContext struct {
	Query   string
	Chunks  []SearchResult
	Context string // 构造好的上下文字符串
}

// 任务状态常量。
const (
	TaskStatusPending  = "pending"
	TaskStatusIndexing = "indexing"
	TaskStatusEmbedded = "embedded"
	TaskStatusDone     = "done"
	TaskStatusError    = "error"
)
