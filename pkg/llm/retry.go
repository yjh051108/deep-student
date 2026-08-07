package llm

import (
	"context"
	"errors"
	"math/rand"
	"time"
)

// WithRetry 在 err 可重试时重试。
func WithRetry(ctx context.Context, max int, backoff time.Duration, fn func() error) error {
	if max < 1 {
		max = 1
	}
	var err error
	for i := 0; i < max; i++ {
		err = fn()
		if err == nil {
			return nil
		}
		if !IsRetryable(err) {
			return err
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(backoff + time.Duration(rand.Int63n(int64(backoff)))):
		}
		backoff *= 2
	}
	return err
}

// IsRetryable 简单判断。
func IsRetryable(err error) bool {
	if err == nil {
		return false
	}
	if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
		return false
	}
	s := err.Error()
	for _, kw := range []string{"429", "500", "502", "503", "504", "rate limit", "timeout"} {
		if contains(s, kw) {
			return true
		}
	}
	return false
}

func contains(s, sub string) bool {
	if len(sub) == 0 {
		return true
	}
	for i := 0; i+len(sub) <= len(s); i++ {
		if s[i:i+len(sub)] == sub {
			return true
		}
	}
	return false
}

// Cost 简易成本估算（USD / 1K tokens）。
type Cost struct {
	PromptPer1K     float64
	CompletionPer1K float64
}

// EstimateUSD 估算成本。
func (c Cost) EstimateUSD(u Usage) float64 {
	return float64(u.PromptTokens)/1000.0*c.PromptPer1K + float64(u.CompletionTokens)/1000.0*c.CompletionPer1K
}

// Costs 常见模型成本表（近似）。
var Costs = map[string]Cost{
	"gpt-4o-mini":       {0.00015, 0.0006},
	"gpt-4o":            {0.005, 0.015},
	"claude-3-5-sonnet": {0.003, 0.015},
	"deepseek-chat":     {0.00014, 0.00028},
	"qwen2.5-72b":       {0.0004, 0.0004},
}
