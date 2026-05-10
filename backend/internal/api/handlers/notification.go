package handlers

import (
	"encoding/json"
	"errors"
	"net/http"

	"blackgrid/internal/db"
	"blackgrid/internal/service"

	"github.com/jackc/pgx/v5/pgtype"
	"github.com/labstack/echo/v4"
)

type NotificationHandler struct {
	svc   *service.NotificationService
	audit *service.AuditService
}

func NewNotificationHandler(svc *service.NotificationService) *NotificationHandler {
	return &NotificationHandler{svc: svc}
}

// SetAuditService wires the audit service. Channel mutations are written to
// the audit log when this is set; otherwise they pass through silently.
func (h *NotificationHandler) SetAuditService(a *service.AuditService) {
	h.audit = a
}

type channelResponse struct {
	ID          string          `json:"id"`
	Name        string          `json:"name"`
	ChannelType string          `json:"channel_type"`
	Enabled     bool            `json:"enabled"`
	Config      json.RawMessage `json:"config"`
	CreatedAt   any             `json:"created_at"`
	UpdatedAt   any             `json:"updated_at"`
}

func toChannelResponse(c db.NotificationChannel) channelResponse {
	idBytes := c.ID.Bytes
	id := ""
	if c.ID.Valid {
		id = uuidToString(idBytes)
	}
	return channelResponse{
		ID:          id,
		Name:        c.Name,
		ChannelType: c.ChannelType,
		Enabled:     c.Enabled,
		Config:      service.MaskConfig(c),
		CreatedAt:   timestamptzToJSON(c.CreatedAt),
		UpdatedAt:   timestamptzToJSON(c.UpdatedAt),
	}
}

func uuidToString(b [16]byte) string {
	const hex = "0123456789abcdef"
	out := make([]byte, 36)
	pos := 0
	for i, by := range b {
		if i == 4 || i == 6 || i == 8 || i == 10 {
			out[pos] = '-'
			pos++
		}
		out[pos] = hex[by>>4]
		out[pos+1] = hex[by&0x0f]
		pos += 2
	}
	return string(out)
}

func timestamptzToJSON(t pgtype.Timestamptz) any {
	if !t.Valid {
		return nil
	}
	return t.Time
}

type channelRequest struct {
	Name        string          `json:"name"`
	ChannelType string          `json:"channel_type"`
	Enabled     *bool           `json:"enabled"`
	Config      json.RawMessage `json:"config"`
}

func (h *NotificationHandler) ListChannels(c echo.Context) error {
	channels, err := h.svc.ListChannels(c.Request().Context())
	if err != nil {
		return Error(c, ErrCodeInternal, "internal error", nil)
	}
	resp := make([]channelResponse, 0, len(channels))
	for _, ch := range channels {
		resp = append(resp, toChannelResponse(ch))
	}
	return c.JSON(http.StatusOK, resp)
}

func (h *NotificationHandler) GetChannel(c echo.Context) error {
	var id pgtype.UUID
	if err := id.Scan(c.Param("id")); err != nil {
		return Error(c, ErrCodeValidation, "invalid id", nil)
	}
	ch, err := h.svc.GetChannel(c.Request().Context(), id)
	if err != nil {
		if errors.Is(err, service.ErrChannelNotFound) {
			return Error(c, ErrCodeNotFound, "channel not found", nil)
		}
		return Error(c, ErrCodeInternal, "internal error", nil)
	}
	return c.JSON(http.StatusOK, toChannelResponse(ch))
}

func (h *NotificationHandler) CreateChannel(c echo.Context) error {
	var req channelRequest
	if err := c.Bind(&req); err != nil {
		return Error(c, ErrCodeValidation, err.Error(), nil)
	}
	enabled := true
	if req.Enabled != nil {
		enabled = *req.Enabled
	}
	if len(req.Config) == 0 {
		req.Config = json.RawMessage("{}")
	}

	ch, err := h.svc.CreateChannel(c.Request().Context(), req.Name, req.ChannelType, enabled, req.Config)
	if err != nil {
		return Error(c, ErrCodeValidation, err.Error(), nil)
	}
	if h.audit != nil {
		// MaskedConfig() strips sensitive fields like passwords/tokens
		// before they enter the audit trail.
		h.audit.Log(c.Request().Context(), service.AuditParams{
			Action:     "notification_channel.create",
			EntityType: "notification_channel",
			EntityID:   ch.ID,
			After: map[string]any{
				"name":         ch.Name,
				"channel_type": ch.ChannelType,
				"enabled":      ch.Enabled,
				"config":       json.RawMessage(service.MaskConfig(ch)),
			},
		})
	}
	return c.JSON(http.StatusCreated, toChannelResponse(ch))
}

func (h *NotificationHandler) UpdateChannel(c echo.Context) error {
	var id pgtype.UUID
	if err := id.Scan(c.Param("id")); err != nil {
		return Error(c, ErrCodeValidation, "invalid id", nil)
	}

	existing, err := h.svc.GetChannel(c.Request().Context(), id)
	if err != nil {
		if errors.Is(err, service.ErrChannelNotFound) {
			return Error(c, ErrCodeNotFound, "channel not found", nil)
		}
		return Error(c, ErrCodeInternal, "internal error", nil)
	}

	var req channelRequest
	if err := c.Bind(&req); err != nil {
		return Error(c, ErrCodeValidation, err.Error(), nil)
	}

	name := existing.Name
	if req.Name != "" {
		name = req.Name
	}
	channelType := existing.ChannelType
	if req.ChannelType != "" {
		channelType = req.ChannelType
	}
	enabled := existing.Enabled
	if req.Enabled != nil {
		enabled = *req.Enabled
	}
	config := json.RawMessage(existing.Config)
	if len(req.Config) > 0 {
		config = req.Config
	}

	ch, err := h.svc.UpdateChannel(c.Request().Context(), id, name, channelType, enabled, config)
	if err != nil {
		return Error(c, ErrCodeValidation, err.Error(), nil)
	}
	if h.audit != nil {
		h.audit.Log(c.Request().Context(), service.AuditParams{
			Action:     "notification_channel.update",
			EntityType: "notification_channel",
			EntityID:   ch.ID,
			Before:     map[string]any{"name": existing.Name, "enabled": existing.Enabled, "config": json.RawMessage(service.MaskConfig(existing))},
			After:      map[string]any{"name": ch.Name, "enabled": ch.Enabled, "config": json.RawMessage(service.MaskConfig(ch))},
		})
	}
	return c.JSON(http.StatusOK, toChannelResponse(ch))
}

func (h *NotificationHandler) DeleteChannel(c echo.Context) error {
	var id pgtype.UUID
	if err := id.Scan(c.Param("id")); err != nil {
		return Error(c, ErrCodeValidation, "invalid id", nil)
	}
	if err := h.svc.DeleteChannel(c.Request().Context(), id); err != nil {
		return Error(c, ErrCodeInternal, "internal error", nil)
	}
	if h.audit != nil {
		h.audit.Log(c.Request().Context(), service.AuditParams{
			Action:     "notification_channel.delete",
			EntityType: "notification_channel",
			EntityID:   id,
		})
	}
	return c.NoContent(http.StatusNoContent)
}

func (h *NotificationHandler) TestChannel(c echo.Context) error {
	var id pgtype.UUID
	if err := id.Scan(c.Param("id")); err != nil {
		return Error(c, ErrCodeValidation, "invalid id", nil)
	}

	delivery, err := h.svc.TestChannel(c.Request().Context(), id)
	if err != nil {
		if errors.Is(err, service.ErrChannelNotFound) {
			return Error(c, ErrCodeNotFound, "channel not found", nil)
		}
		return Error(c, ErrCodeInternal, "internal error", nil)
	}

	resp := map[string]any{
		"status":     delivery.Status,
		"event_type": delivery.EventType,
	}
	if delivery.LastError.Valid {
		resp["error"] = delivery.LastError.String
	}
	if delivery.SentAt.Valid {
		resp["sent_at"] = delivery.SentAt.Time
	}
	return c.JSON(http.StatusOK, resp)
}
