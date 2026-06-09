package runtime

import "sync"

type Event struct {
	Name    string
	Payload any
}

type Unsubscribe func()

type EventBus struct {
	mu        sync.RWMutex
	nextID    int
	listeners map[string]map[int]func(Event)
}

func NewEventBus() *EventBus {
	return &EventBus{
		listeners: make(map[string]map[int]func(Event)),
	}
}

func (b *EventBus) Listen(name string, handler func(Event)) Unsubscribe {
	b.mu.Lock()
	defer b.mu.Unlock()

	b.nextID++
	id := b.nextID
	if b.listeners[name] == nil {
		b.listeners[name] = make(map[int]func(Event))
	}
	b.listeners[name][id] = handler

	return func() {
		b.mu.Lock()
		defer b.mu.Unlock()
		delete(b.listeners[name], id)
		if len(b.listeners[name]) == 0 {
			delete(b.listeners, name)
		}
	}
}

func (b *EventBus) Emit(name string, payload any) {
	b.mu.RLock()
	handlers := make([]func(Event), 0, len(b.listeners[name])+len(b.listeners["*"]))
	for _, handler := range b.listeners[name] {
		handlers = append(handlers, handler)
	}
	for _, handler := range b.listeners["*"] {
		handlers = append(handlers, handler)
	}
	b.mu.RUnlock()

	event := Event{Name: name, Payload: payload}
	for _, handler := range handlers {
		handler(event)
	}
}
