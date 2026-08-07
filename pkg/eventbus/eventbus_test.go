package eventbus

import (
	"context"
	"sync/atomic"
	"testing"
	"time"
)

func TestPublishSubscribe(t *testing.T) {
	b := New()
	defer b.Close()
	var n int32
	b.Subscribe("x", func(ctx context.Context, e Event) error {
		atomic.AddInt32(&n, 1)
		return nil
	}, false)
	b.Publish(context.Background(), "x", "hello")
	if atomic.LoadInt32(&n) != 1 {
		t.Fatalf("expected 1, got %d", n)
	}
}

func TestAsyncSubscribe(t *testing.T) {
	b := New()
	defer b.Close()
	var n int32
	b.Subscribe("x", func(ctx context.Context, e Event) error {
		atomic.AddInt32(&n, 1)
		return nil
	}, true)
	b.Publish(context.Background(), "x", "hi")
	b.Wait()
	if atomic.LoadInt32(&n) != 1 {
		t.Fatalf("expected 1, got %d", n)
	}
}

func TestCancel(t *testing.T) {
	b := New()
	defer b.Close()
	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()
	errs := b.Publish(ctx, "nope", nil)
	if len(errs) != 0 {
		t.Fatal("expected no errors")
	}
}
