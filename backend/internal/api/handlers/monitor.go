package handlers

import (
	"encoding/json"
	"net/http"
	"time"

	"blackgrid/internal/db"
	"blackgrid/internal/monitor"

	"github.com/jackc/pgx/v5/pgtype"
	"github.com/labstack/echo/v4"
)

type MonitorHandler struct {
	queries   *db.Queries
	runner    *monitor.Runner
}

func NewMonitorHandler(queries *db.Queries, runner *monitor.Runner) *MonitorHandler {
	return &MonitorHandler{
		queries:   queries,
		runner:    runner,
	}
}

func (h *MonitorHandler) GetMonitors(c echo.Context) error {
	ctx := c.Request().Context()
	monitors, err := h.queries.GetMonitors(ctx)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}
	return c.JSON(http.StatusOK, monitors)
}

func (h *MonitorHandler) GetMonitor(c echo.Context) error {
	ctx := c.Request().Context()
	id := c.Param("id")

	var uuid pgtype.UUID
	if err := uuid.Scan(id); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid UUID"})
	}

	m, err := h.queries.GetMonitor(ctx, uuid)
	if err != nil {
		return c.JSON(http.StatusNotFound, map[string]string{"error": "monitor not found"})
	}

	return c.JSON(http.StatusOK, m)
}

type createMonitorRequest struct {
	Name            string          `json:"name"`
	Slug            string          `json:"slug"`
	MonitorType     string          `json:"monitor_type"`
	Target          string          `json:"target"`
	Config          json.RawMessage `json:"config"`
	IpAddressID     *string         `json:"ip_address_id"`
	DeviceID        *string         `json:"device_id"`
	IntervalSeconds int32           `json:"interval_seconds"`
	TimeoutSeconds  int32           `json:"timeout_seconds"`
	RetryCount      int32           `json:"retry_count"`
	Enabled         *bool           `json:"enabled"`
}

func (h *MonitorHandler) CreateMonitor(c echo.Context) error {
	ctx := c.Request().Context()
	var req createMonitorRequest
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": err.Error()})
	}

	if req.Name == "" || req.Target == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "name and target are required"})
	}
	if req.MonitorType != "http" && req.MonitorType != "tcp" && req.MonitorType != "ping" {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid monitor type"})
	}

	slug := req.Slug
	if slug == "" {
		slug = req.Name // naive slug generation, could be improved
	}

	interval := req.IntervalSeconds
	if interval < 10 {
		interval = 60
	}
	timeout := req.TimeoutSeconds
	if timeout < 1 {
		timeout = 10
	}
	retryCount := req.RetryCount
	if retryCount < 1 {
		retryCount = 3
	}

	enabled := true
	if req.Enabled != nil {
		enabled = *req.Enabled
	}

	status := "unknown"
	if !enabled {
		status = "paused"
	}

	var ipID pgtype.UUID
	if req.IpAddressID != nil {
		ipID.Scan(*req.IpAddressID)
	}

	var deviceID pgtype.UUID
	if req.DeviceID != nil {
		deviceID.Scan(*req.DeviceID)
	}

	m, err := h.queries.CreateMonitor(ctx, db.CreateMonitorParams{
		Name:            req.Name,
		Slug:            slug,
		MonitorType:     req.MonitorType,
		Target:          req.Target,
		Config:          req.Config,
		IpAddressID:     ipID,
		DeviceID:        deviceID,
		IntervalSeconds: interval,
		TimeoutSeconds:  timeout,
		RetryCount:      retryCount,
		Enabled:         enabled,
		Status:          status,
	})
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	return c.JSON(http.StatusCreated, m)
}

func (h *MonitorHandler) UpdateMonitor(c echo.Context) error {
	ctx := c.Request().Context()
	id := c.Param("id")

	var uuid pgtype.UUID
	if err := uuid.Scan(id); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid UUID"})
	}

	m, err := h.queries.GetMonitor(ctx, uuid)
	if err != nil {
		return c.JSON(http.StatusNotFound, map[string]string{"error": "monitor not found"})
	}

	var req createMonitorRequest
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": err.Error()})
	}

	// For simplicity, update all fields from DB monitor with provided request fields if non-zero
	if req.Name != "" { m.Name = req.Name }
	if req.Slug != "" { m.Slug = req.Slug }
	if req.MonitorType != "" { m.MonitorType = req.MonitorType }
	if req.Target != "" { m.Target = req.Target }
	if req.Config != nil { m.Config = req.Config }
	if req.IntervalSeconds >= 10 { m.IntervalSeconds = req.IntervalSeconds }
	if req.TimeoutSeconds >= 1 { m.TimeoutSeconds = req.TimeoutSeconds }
	if req.RetryCount >= 1 { m.RetryCount = req.RetryCount }

	if req.IpAddressID != nil {
		m.IpAddressID.Scan(*req.IpAddressID)
	}
	if req.DeviceID != nil {
		m.DeviceID.Scan(*req.DeviceID)
	}

	if req.Enabled != nil {
		m.Enabled = *req.Enabled
		if !m.Enabled {
			m.Status = "paused"
		} else if m.Status == "paused" {
			m.Status = "unknown"
		}
	}

	updated, err := h.queries.UpdateMonitor(ctx, db.UpdateMonitorParams{
		ID:                 m.ID,
		Name:               m.Name,
		Slug:               m.Slug,
		MonitorType:        m.MonitorType,
		Target:             m.Target,
		Config:             m.Config,
		IpAddressID:        m.IpAddressID,
		DeviceID:           m.DeviceID,
		IntervalSeconds:    m.IntervalSeconds,
		TimeoutSeconds:     m.TimeoutSeconds,
		RetryCount:         m.RetryCount,
		Enabled:            m.Enabled,
		Status:             m.Status,
		LastCheckedAt:      m.LastCheckedAt,
		LastStatusChangeAt: m.LastStatusChangeAt,
	})

	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	return c.JSON(http.StatusOK, updated)
}

func (h *MonitorHandler) DeleteMonitor(c echo.Context) error {
	ctx := c.Request().Context()
	id := c.Param("id")

	var uuid pgtype.UUID
	if err := uuid.Scan(id); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid UUID"})
	}

	if err := h.queries.DeleteMonitor(ctx, uuid); err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	return c.NoContent(http.StatusNoContent)
}

func (h *MonitorHandler) PauseMonitor(c echo.Context) error {
	return h.setStatus(c, "paused", false)
}

func (h *MonitorHandler) ResumeMonitor(c echo.Context) error {
	return h.setStatus(c, "unknown", true)
}

func (h *MonitorHandler) setStatus(c echo.Context, status string, enabled bool) error {
	ctx := c.Request().Context()
	id := c.Param("id")

	var uuid pgtype.UUID
	if err := uuid.Scan(id); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid UUID"})
	}

	m, err := h.queries.GetMonitor(ctx, uuid)
	if err != nil {
		return c.JSON(http.StatusNotFound, map[string]string{"error": "monitor not found"})
	}

	lastStatusChange := m.LastStatusChangeAt
	if m.Status != status {
		lastStatusChange = pgtype.Timestamptz{Time: time.Now(), Valid: true}
	}

	updated, err := h.queries.UpdateMonitor(ctx, db.UpdateMonitorParams{
		ID:                 m.ID,
		Name:               m.Name,
		Slug:               m.Slug,
		MonitorType:        m.MonitorType,
		Target:             m.Target,
		Config:             m.Config,
		IpAddressID:        m.IpAddressID,
		DeviceID:           m.DeviceID,
		IntervalSeconds:    m.IntervalSeconds,
		TimeoutSeconds:     m.TimeoutSeconds,
		RetryCount:         m.RetryCount,
		Enabled:            enabled,
		Status:             status,
		LastCheckedAt:      m.LastCheckedAt,
		LastStatusChangeAt: lastStatusChange,
	})

	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	return c.JSON(http.StatusOK, updated)
}

func (h *MonitorHandler) TestMonitor(c echo.Context) error {
	ctx := c.Request().Context()
	id := c.Param("id")

	var uuid pgtype.UUID
	if err := uuid.Scan(id); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid UUID"})
	}

	m, err := h.queries.GetMonitor(ctx, uuid)
	if err != nil {
		return c.JSON(http.StatusNotFound, map[string]string{"error": "monitor not found"})
	}

	result, err := h.runner.Run(ctx, m)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	// Update last_checked_at manually here since it's a manual test
	now := pgtype.Timestamptz{Time: time.Now(), Valid: true}
	_, _ = h.queries.UpdateMonitor(ctx, db.UpdateMonitorParams{
		ID:                 m.ID,
		Name:               m.Name,
		Slug:               m.Slug,
		MonitorType:        m.MonitorType,
		Target:             m.Target,
		Config:             m.Config,
		IpAddressID:        m.IpAddressID,
		DeviceID:           m.DeviceID,
		IntervalSeconds:    m.IntervalSeconds,
		TimeoutSeconds:     m.TimeoutSeconds,
		RetryCount:         m.RetryCount,
		Enabled:            m.Enabled,
		Status:             m.Status, // status is intentionally NOT updated per requirements for test
		LastCheckedAt:      now,
		LastStatusChangeAt: m.LastStatusChangeAt,
	})

	return c.JSON(http.StatusOK, result)
}

func (h *MonitorHandler) GetMonitorResults(c echo.Context) error {
	ctx := c.Request().Context()
	id := c.Param("id")

	var uuid pgtype.UUID
	if err := uuid.Scan(id); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid UUID"})
	}

	results, err := h.queries.GetMonitorResults(ctx, db.GetMonitorResultsParams{
		MonitorID: uuid,
		Limit:     100,
		Offset:    0,
	})
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	return c.JSON(http.StatusOK, results)
}
