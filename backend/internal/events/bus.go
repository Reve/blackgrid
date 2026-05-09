package events

import (
	"context"
	"sync"

	"github.com/google/uuid"
	"blackgrid/internal/metrics"
)

type Subscriber struct {
	ch     chan Event
	filter EventFilter
	ctx    context.Context
}

type EventBus struct {
	subscribers map[string]*Subscriber
	mu          sync.RWMutex
	closed      bool
}

func NewEventBus() *EventBus {
	return &EventBus{
		subscribers: make(map[string]*Subscriber),
	}
}

func (b *EventBus) Publish(ctx context.Context, event Event) {
	b.mu.RLock()
	defer b.mu.RUnlock()

	if b.closed {
		return
	}

	if event.ID == "" {
		event.ID = uuid.New().String()
	}

	metrics.EventBusEventsTotal.WithLabelValues(string(event.Type)).Inc()

	for _, sub := range b.subscribers {
		// Apply filters if any
		if !b.matches(event, sub.filter) {
			continue
		}

		select {
		case sub.ch <- event:
		default:
			// Buffer full, drop event or disconnect?
			// Prompt says: "If a subscriber buffer fills, drop events for that subscriber or disconnect it gracefully."
			// For now, we drop the event to avoid blocking publishers.
		}
	}
}

func (b *EventBus) Subscribe(ctx context.Context, filter EventFilter) (<-chan Event, func()) {
	b.mu.Lock()
	defer b.mu.Unlock()

	id := uuid.New().String()
	// Use buffered channel to avoid blocking publishers
	ch := make(chan Event, 100)

	sub := &Subscriber{
		ch:     ch,
		filter: filter,
		ctx:    ctx,
	}
	b.subscribers[id] = sub

	unsubscribe := func() {
		b.mu.Lock()
		defer b.mu.Unlock()
		if _, ok := b.subscribers[id]; ok {
			close(ch)
			delete(b.subscribers, id)
		}
	}

	return ch, unsubscribe
}

func (b *EventBus) matches(event Event, filter EventFilter) bool {
	if len(filter.Types) > 0 {
		match := false
		for _, t := range filter.Types {
			if t == event.Type {
				match = true
				break
			}
		}
		if !match {
			return false
		}
	}

	if len(filter.ObjectTypes) > 0 {
		match := false
		for _, ot := range filter.ObjectTypes {
			if ot == event.ObjectType {
				match = true
				break
			}
		}
		if !match {
			return false
		}
	}

	return true
}

func (b *EventBus) Shutdown() {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.closed = true
	for id, sub := range b.subscribers {
		close(sub.ch)
		delete(b.subscribers, id)
	}
}
