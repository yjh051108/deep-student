// Package logger 提供结构化日志（基于 log/slog）并按日滚动。
package logger

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"sync"
	"time"
)

var (
	mu     sync.RWMutex
	global *slog.Logger
)

// Init 初始化全局 logger。
func Init(logDir, level string) error {
	mu.Lock()
	defer mu.Unlock()

	if err := os.MkdirAll(logDir, 0o755); err != nil {
		return fmt.Errorf("create log dir: %w", err)
	}

	lvl := parseLevel(level)
	rotator := &dailyRotator{Dir: logDir, MaxDays: 14}
	mw := io.MultiWriter(rotator, os.Stdout)
	handler := slog.NewJSONHandler(mw, &slog.HandlerOptions{Level: lvl})
	global = slog.New(handler).With("app", "deepstudent")
	return nil
}

func parseLevel(s string) slog.Level {
	switch s {
	case "debug", "DEBUG":
		return slog.LevelDebug
	case "warn", "WARN":
		return slog.LevelWarn
	case "error", "ERROR":
		return slog.LevelError
	default:
		return slog.LevelInfo
	}
}

// L 返回全局 logger。
func L() *slog.Logger {
	mu.RLock()
	defer mu.RUnlock()
	if global == nil {
		return slog.Default()
	}
	return global
}

// With 返回带字段的 logger（不修改全局）。
func With(args ...any) *slog.Logger {
	return L().With(args...)
}

// dailyRotator 简单的按日轮转 writer。
type dailyRotator struct {
	Dir     string
	MaxDays int
	mu      sync.Mutex
	cur     *os.File
	day     string
}

func (r *dailyRotator) Write(p []byte) (int, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	day := time.Now().Format("2006-01-02")
	if r.day != day || r.cur == nil {
		if r.cur != nil {
			_ = r.cur.Close()
		}
		f, err := os.OpenFile(filepath.Join(r.Dir, "app-"+day+".log"), os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
		if err != nil {
			return 0, err
		}
		r.cur = f
		r.day = day
	}
	return r.cur.Write(p)
}

// Debug / Info / Warn / Error 便捷入口。
func Debug(msg string, args ...any) { L().Debug(msg, args...) }
func Info(msg string, args ...any)  { L().Info(msg, args...) }
func Warn(msg string, args ...any)  { L().Warn(msg, args...) }
func Error(msg string, args ...any) { L().Error(msg, args...) }

// CtxWithLogger 把 logger 放进 ctx。
func CtxWithLogger(ctx context.Context, l *slog.Logger) context.Context {
	return context.WithValue(ctx, loggerKey{}, l)
}

// FromContext 从 ctx 取出 logger。
func FromContext(ctx context.Context) *slog.Logger {
	if l, ok := ctx.Value(loggerKey{}).(*slog.Logger); ok && l != nil {
		return l
	}
	return L()
}

type loggerKey struct{}
