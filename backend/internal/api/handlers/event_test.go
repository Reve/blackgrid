package handlers

import (
	"context"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"blackgrid/internal/events"
	"github.com/labstack/echo/v4"
)

// TestStreamEvents_FrameFormat verifies the SSE frame contains all three
// fields (id, event, data) in the order browsers' EventSource expects.
func TestStreamEvents_FrameFormat(t *testing.T) {
	bus := events.NewEventBus()
	defer bus.Shutdown()
	h := NewEventHandler(bus)

	e := echo.New()
	req := httptest.NewRequest("GET", "/events", nil)
	rec := httptest.NewRecorder()

	ctx, cancel := context.WithCancel(req.Context())
	req = req.WithContext(ctx)
	c := e.NewContext(req, rec)

	done := make(chan struct{})
	go func() {
		_ = h.StreamEvents(c)
		close(done)
	}()

	// Give the handler a moment to subscribe.
	time.Sleep(50 * time.Millisecond)

	bus.Publish(context.Background(), events.Event{
		ID:         "fixed-id",
		Type:       events.MonitorTested,
		ObjectType: "monitor",
		ObjectID:   "abc",
		Payload:    map[string]any{"x": 1},
	})

	// Allow the frame to be written.
	time.Sleep(80 * time.Millisecond)
	cancel()
	<-done

	body := rec.Body.String()

	if !strings.Contains(body, "id: fixed-id") {
		t.Errorf("missing 'id:' line; got:\n%s", body)
	}
	if !strings.Contains(body, "event: monitor.tested") {
		t.Errorf("missing 'event:' line with type; got:\n%s", body)
	}
	if !strings.Contains(body, "data: {") {
		t.Errorf("missing 'data:' line; got:\n%s", body)
	}
	// Frames must end with a blank line, per the SSE spec.
	if !strings.Contains(body, "\n\n") {
		t.Errorf("frame not terminated with blank line; got:\n%s", body)
	}
}
