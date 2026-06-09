package runtime

import "testing"

func TestEventBusListenEmitAndUnsubscribe(t *testing.T) {
	bus := NewEventBus()
	var seen int

	stop := bus.Listen("ready", func(event Event) {
		if event.Name != "ready" {
			t.Fatalf("unexpected event name %q", event.Name)
		}
		seen++
	})

	bus.Emit("ready", nil)
	if seen != 1 {
		t.Fatalf("expected one event, got %d", seen)
	}

	stop()
	bus.Emit("ready", nil)
	if seen != 1 {
		t.Fatalf("expected unsubscribe to stop events, got %d", seen)
	}
}

func TestEventBusWildcardListener(t *testing.T) {
	bus := NewEventBus()
	seen := []string{}

	stop := bus.Listen("*", func(event Event) {
		seen = append(seen, event.Name)
	})

	bus.Emit("first", nil)
	bus.Emit("second", nil)
	stop()
	bus.Emit("third", nil)

	if len(seen) != 2 {
		t.Fatalf("expected two wildcard events, got %d", len(seen))
	}
	if seen[0] != "first" || seen[1] != "second" {
		t.Fatalf("unexpected wildcard event order: %+v", seen)
	}
}
