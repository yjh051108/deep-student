// llmusage 包的业务层：记录调用、查询日志/聚合、总览、清理。
//
// 构造函数 New(fs, store, llmReg) 与既有 internal 包保持一致。

package llmusage

import (
	"fmt"
	"sync"
	"time"

	"github.com/google/uuid"

	"github.com/helixnow/deep-student-go/pkg/llm"
	"github.com/helixnow/deep-student-go/pkg/store"
	"github.com/helixnow/deep-student-go/pkg/vfs"
)

// Service LLM 用量统计业务服务。
type Service struct {
	store *Store
	mu    sync.RWMutex
}

// New 构造用量统计服务。vfs 与 llmReg 保留用于与既有构造签名一致。
func New(_ *vfs.FS, s *store.Store, _ *llm.Registry) *Service {
	us := NewStore(s)
	if err := us.Migrate(); err != nil {
		fmt.Printf("[llmusage] migrate failed: %v\n", err)
	}
	return &Service{store: us}
}

// Record 记录一次调用。由各领域服务在 LLM 调用完成后调用。
func (s *Service) Record(e LogEntry) (*Log, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	now := time.Now().UTC()
	status := e.Status
	if status == "" {
		status = StatusSuccess
	}
	source := e.TokenSource
	if source == "" {
		source = TokenSourceAPI
	}
	total := e.PromptTokens + e.CompletionTokens
	if total < 0 {
		total = 0
	}
	l := &Log{
		ID:               uuid.NewString(),
		Timestamp:        now,
		Provider:         e.Provider,
		Model:            e.Model,
		Adapter:          e.Adapter,
		APIConfigID:      e.APIConfigID,
		PromptTokens:     e.PromptTokens,
		CompletionTokens: e.CompletionTokens,
		TotalTokens:      total,
		ReasoningTokens:  e.ReasoningTokens,
		CachedTokens:     e.CachedTokens,
		TokenSource:      source,
		DurationMs:       e.DurationMs,
		RequestBytes:     e.RequestBytes,
		ResponseBytes:    e.ResponseBytes,
		FirstTokenMs:     e.FirstTokenMs,
		CallerType:       e.CallerType,
		SessionID:        e.SessionID,
		Status:           status,
		ErrorMessage:     e.ErrorMessage,
		CostEstimate:     e.CostEstimate,
	}
	if l.CallerType == "" {
		l.CallerType = "other"
	}
	if err := s.store.InsertLog(l); err != nil {
		return nil, err
	}
	return l, nil
}

// RecordChat 便捷方法：从 ChatResponse 的 Usage 记录一次成功调用。
func (s *Service) RecordChat(callerType, provider, model, sessionID string, usage llm.Usage) (*Log, error) {
	prompt, completion := usage.PromptTokens, usage.CompletionTokens
	return s.Record(LogEntry{
		Provider:         provider,
		Model:            model,
		PromptTokens:     prompt,
		CompletionTokens: completion,
		CallerType:       callerType,
		SessionID:        sessionID,
		Status:           StatusSuccess,
	})
}

// Query 查询日志。
func (s *Service) Query(filter LogFilter) ([]Log, error) {
	return s.store.QueryLogs(filter)
}

// QueryDaily 查询按日聚合。
func (s *Service) QueryDaily(filter DailyFilter) ([]DailyAggregate, error) {
	return s.store.QueryDaily(filter)
}

// Summary 用量总览。
func (s *Service) Summary() (*Summary, error) {
	return s.store.Aggregate()
}

// CleanupOlderThan 清理指定日期之前的日志，返回删除数量。
func (s *Service) CleanupOlderThan(before time.Time) (int64, error) {
	return s.store.DeleteOlderThan(before)
}
