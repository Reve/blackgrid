package events

import (
	"context"
	"testing"
	"time"
)

func TestPublish_PopulatesIDAndCreatedAt(t *testing.T) {
	bus := NewEventBus()
	defer bus.Shutdown()

	ch, unsub := bus.Subscribe(context.Background(), EventFilter{})
	defer unsub()

	bus.Publish(context.Background(), Event{Type: MonitorTested, Payload: map[string]any{"x": 1}})

	select {
	case ev := <-ch:
		if ev.ID == "" {
			t.Error("Publish did not populate Event.ID")
		}
		if ev.CreatedAt.IsZero() {
			t.Error("Publish did not populate Event.CreatedAt")
		}
	case <-time.After(time.Second):
		t.Fatal("event never delivered")
	}
}

func TestPublish_DoesNotOverridePreSetCreatedAt(t *testing.T) {
	bus := NewEventBus()
	defer bus.Shutdown()

	ch, unsub := bus.Subscribe(context.Background(), EventFilter{})
	defer unsub()

	preset := time.Date(2025, 1, 2, 3, 4, 5, 0, time.UTC)
	bus.Publish(context.Background(), Event{Type: MonitorTested, CreatedAt: preset})

	select {
	case ev := <-ch:
		if !ev.CreatedAt.Equal(preset) {
			t.Errorf("Publish overwrote pre-set CreatedAt: got %v, want %v", ev.CreatedAt, preset)
		}
	case <-time.After(time.Second):
		t.Fatal("event never delivered")
	}
}

func TestShutdown_ClosesSubscribers(t *testing.T) {
	bus := NewEventBus()
	ch, _ := bus.Subscribe(context.Background(), EventFilter{})
	bus.Shutdown()

	// Channel must close so SSE handler returns instead of hanging.
	select {
	case _, ok := <-ch:
		if ok {
			t.Error("expected channel closed after Shutdown")
		}
	case <-time.After(time.Second):
		t.Fatal("subscriber channel was not closed after Shutdown")
	}

	// Subsequent Publish must be a silent no-op.
	bus.Publish(context.Background(), Event{Type: MonitorTested})
}
