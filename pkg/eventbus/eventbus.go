// Package eventbus 提供进程内事件总线（订阅、广播、取消）。
package eventbus

import (
	"context"
	"sync"
)

// Event 通用事件载荷。
type Event struct {
	Topic   string
	Payload any
}

// Handler 事件处理函数。
type Handler func(ctx context.Context, e Event) error

// Bus 简单同步/异步混合事件总线。
type Bus struct {
	mu     sync.RWMutex
	subs   map[string][]subEntry
	wg     sync.WaitGroup
	closed bool
}

type subEntry struct {
	h     Handler
	async bool
}

// New 创建一个事件总线。
func New() *Bus {
	return &Bus{subs: map[string][]subEntry{}}
}

// Subscribe 订阅一个 topic，async=true 时异步派发。
func (b *Bus) Subscribe(topic string, h Handler, async bool) {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.subs[topic] = append(b.subs[topic], subEntry{h: h, async: async})
}

// Publish 同步广播到所有订阅者。
func (b *Bus) Publish(ctx context.Context, topic string, payload any) []error {
	b.mu.RLock()
	entries := append([]subEntry(nil), b.subs[topic]...)
	b.mu.RUnlock()
	errs := make([]error, 0, len(entries))
	for _, e := range entries {
		if e.async {
			b.wg.Add(1)
			go func(h Handler) {
				defer b.wg.Done()
				_ = h(ctx, Event{Topic: topic, Payload: payload})
			}(e.h)
			continue
		}
		if err := e.h(ctx, Event{Topic: topic, Payload: payload}); err != nil {
			errs = append(errs, err)
		}
	}
	return errs
}

// PublishAsync 异步广播。
func (b *Bus) PublishAsync(ctx context.Context, topic string, payload any) {
	go func() { _ = b.Publish(ctx, topic, payload) }()
}

// Wait 等待所有异步 handler 完成。
func (b *Bus) Wait() { b.wg.Wait() }

// Close 标记关闭并等待。
func (b *Bus) Close() {
	b.mu.Lock()
	b.closed = true
	b.mu.Unlock()
	b.wg.Wait()
}

// Topics 返回当前已订阅的所有 topic。
func (b *Bus) Topics() []string {
	b.mu.RLock()
	defer b.mu.RUnlock()
	out := make([]string, 0, len(b.subs))
	for t := range b.subs {
		out = append(out, t)
	}
	return out
}
