package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"blackgrid/internal/events"
	"blackgrid/internal/metrics"

	"github.com/labstack/echo/v4"
)

type EventHandler struct {
	bus *events.EventBus
}

func NewEventHandler(bus *events.EventBus) *EventHandler {
	return &EventHandler{
		bus: bus,
	}
}

func (h *EventHandler) StreamEvents(c echo.Context) error {
	w := c.Response().Writer
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("Transfer-Encoding", "chunked")

	// Parse filters from query params
	filter := events.EventFilter{}
	if types := c.QueryParam("types"); types != "" {
		parts := strings.Split(types, ",")
		for _, p := range parts {
			filter.Types = append(filter.Types, events.EventType(p))
		}
	}
	if objectTypes := c.QueryParam("object_types"); objectTypes != "" {
		filter.ObjectTypes = strings.Split(objectTypes, ",")
	}

	ch, unsubscribe := h.bus.Subscribe(c.Request().Context(), filter)
	metrics.SseClientsCurrent.Inc()
	defer func() {
		unsubscribe()
		metrics.SseClientsCurrent.Dec()
	}()

	// Flush the headers
	c.Response().Flush()

	ticker := time.NewTicker(20 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-c.Request().Context().Done():
			return nil
		case <-ticker.C:
			// Send keepalive comment
			if _, err := fmt.Fprint(w, ": keepalive\n\n"); err != nil {
				return nil
			}
			c.Response().Flush()
		case event, ok := <-ch:
			if !ok {
				return nil
			}

			data, err := json.Marshal(event)
			if err != nil {
				continue
			}

			if _, err := fmt.Fprintf(w, "id: %s\ndata: %s\n\n", event.ID, string(data)); err != nil {
				return nil
			}
			c.Response().Flush()
		}
	}
}
