package handlers

import (
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"blackgrid/internal/db"
	"blackgrid/internal/monitor"
	"blackgrid/internal/service"

	"github.com/jackc/pgx/v5/pgtype"
	"github.com/labstack/echo/v4"
)

type MonitorHandler struct {
	queries      *db.Queries
	runner       *monitor.Runner
	AuditService *service.AuditService
}

func NewMonitorHandler(queries *db.Queries, runner *monitor.Runner, audit *service.AuditService) *MonitorHandler {
	return &MonitorHandler{
		queries:      queries,
		runner:       runner,
		AuditService: audit,
	}
}

// validMonitorTypes is the full allowed set for phase 7.
var validMonitorTypes = map[string]bool{
	"http":     true,
	"tcp":      true,
	"ping":     true,
	"dns":      true,
	"tls":      true,
	"push":     true,
	"postgres": true,
}

// monitorResponse is a safe view of a Monitor that masks secrets.
type monitorResponse struct {
	ID                 string          `json:"id"`
	Name               string          `json:"name"`
	Slug               string          `json:"slug"`
	MonitorType        string          `json:"monitor_type"`
	Target             string          `json:"target"`
	Config             json.RawMessage `json:"config"`
	IpAddressID        *string         `json:"ip_address_id"`
	DeviceID           *string         `json:"device_id"`
	IntervalSeconds    int32           `json:"interval_seconds"`
	TimeoutSeconds     int32           `json:"timeout_seconds"`
	RetryCount         int32           `json:"retry_count"`
	Enabled            bool            `json:"enabled"`
	Status             string          `json:"status"`
	LastCheckedAt      *string         `json:"last_checked_at"`
	LastStatusChangeAt *string         `json:"last_status_change_at"`
	CreatedAt          string          `json:"created_at"`
	UpdatedAt          string          `json:"updated_at"`
}

func toMonitorResponse(m db.Monitor) monitorResponse {
	r := monitorResponse{
		Name:            m.Name,
		Slug:            m.Slug,
		MonitorType:     m.MonitorType,
		Target:          m.Target,
		IntervalSeconds: m.IntervalSeconds,
		TimeoutSeconds:  m.TimeoutSeconds,
		RetryCount:      m.RetryCount,
		Enabled:         m.Enabled,
		Status:          m.Status,
	}

	// UUID → string
	if b, err := m.ID.MarshalJSON(); err == nil {
		var s string
		if json.Unmarshal(b, &s) == nil {
			r.ID = s
		}
	}
	if m.IpAddressID.Valid {
		if b, err := m.IpAddressID.MarshalJSON(); err == nil {
			var s string
			if json.Unmarshal(b, &s) == nil {
				r.IpAddressID = &s
			}
		}
	}
	if m.DeviceID.Valid {
		if b, err := m.DeviceID.MarshalJSON(); err == nil {
			var s string
			if json.Unmarshal(b, &s) == nil {
				r.DeviceID = &s
			}
		}
	}

	// Timestamps
	if m.LastCheckedAt.Valid {
		s := m.LastCheckedAt.Time.Format(time.RFC3339)
		r.LastCheckedAt = &s
	}
	if m.LastStatusChangeAt.Valid {
		s := m.LastStatusChangeAt.Time.Format(time.RFC3339)
		r.LastStatusChangeAt = &s
	}
	r.CreatedAt = m.CreatedAt.Time.Format(time.RFC3339)
	r.UpdatedAt = m.UpdatedAt.Time.Format(time.RFC3339)

	// Mask config
	r.Config = monitor.MaskConfig(m.Config)

	return r
}

func (h *MonitorHandler) GetMonitors(c echo.Context) error {
	ctx := c.Request().Context()
	monitors, err := h.queries.GetMonitors(ctx)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}
	out := make([]monitorResponse, len(monitors))
	for i, m := range monitors {
		out[i] = toMonitorResponse(m)
	}
	return c.JSON(http.StatusOK, out)
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

	return c.JSON(http.StatusOK, toMonitorResponse(m))
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

// createMonitorResponse extends monitorResponse with optional plaintext token (shown once).
type createMonitorResponse struct {
	monitorResponse
	GeneratedPushToken string `json:"generated_push_token,omitempty"`
}

func (h *MonitorHandler) CreateMonitor(c echo.Context) error {
	ctx := c.Request().Context()
	var req createMonitorRequest
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": err.Error()})
	}

	if req.Name == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "name is required"})
	}
	if !validMonitorTypes[req.MonitorType] {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid monitor type"})
	}

	// Push monitors don't require a network target
	if req.MonitorType != "push" && req.Target == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "target is required"})
	}

	slug := req.Slug
	if slug == "" {
		slug = strings.ToLower(strings.ReplaceAll(req.Name, " ", "-"))
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
		ipID.Scan(*req.IpAddressID) //nolint:errcheck
	}

	var deviceID pgtype.UUID
	if req.DeviceID != nil {
		deviceID.Scan(*req.DeviceID) //nolint:errcheck
	}

	// Handle push token generation
	var plainToken string
	var tokenHash pgtype.Text
	config := req.Config

	if req.MonitorType == "push" {
		// Parse existing config for grace_seconds
		var pushCfg monitor.PushConfig
		if config != nil {
			json.Unmarshal(config, &pushCfg) //nolint:errcheck
		}
		if pushCfg.GraceSeconds <= 0 {
			pushCfg.GraceSeconds = 120
		}

		tok, err := monitor.GeneratePushToken()
		if err != nil {
			return c.JSON(http.StatusInternalServerError, map[string]string{"error": "failed to generate push token"})
		}
		plainToken = tok
		hash := monitor.HashToken(tok)
		tokenHash = pgtype.Text{String: hash, Valid: true}

		// Store config without the token (token is in push_token_hash column)
		cfgBytes, _ := json.Marshal(map[string]any{
			"grace_seconds": pushCfg.GraceSeconds,
		})
		config = cfgBytes
	}

	// Validate postgres requires dsn
	if req.MonitorType == "postgres" {
		var pgCfg struct {
			DSN string `json:"dsn"`
		}
		if config != nil {
			json.Unmarshal(config, &pgCfg) //nolint:errcheck
		}
		if pgCfg.DSN == "" {
			return c.JSON(http.StatusBadRequest, map[string]string{"error": "dsn is required for postgres monitors"})
		}
	}

	m, err := h.queries.CreateMonitor(ctx, db.CreateMonitorParams{
		Name:            req.Name,
		Slug:            slug,
		MonitorType:     req.MonitorType,
		Target:          req.Target,
		Config:          config,
		IpAddressID:     ipID,
		DeviceID:        deviceID,
		IntervalSeconds: interval,
		TimeoutSeconds:  timeout,
		RetryCount:      retryCount,
		Enabled:         enabled,
		Status:          status,
		PushTokenHash:   tokenHash,
	})
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	resp := createMonitorResponse{
		monitorResponse:    toMonitorResponse(m),
		GeneratedPushToken: plainToken,
	}
	return c.JSON(http.StatusCreated, resp)
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

	if req.Name != "" {
		m.Name = req.Name
	}
	if req.Slug != "" {
		m.Slug = req.Slug
	}
	if req.MonitorType != "" {
		if !validMonitorTypes[req.MonitorType] {
			return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid monitor type"})
		}
		m.MonitorType = req.MonitorType
	}
	if req.Target != "" {
		m.Target = req.Target
	}

	// Config update: preserve secrets if blank fields submitted
	if req.Config != nil {
		newConfig := mergeConfig(m.MonitorType, m.Config, req.Config)
		m.Config = newConfig
	}

	if req.IntervalSeconds >= 10 {
		m.IntervalSeconds = req.IntervalSeconds
	}
	if req.TimeoutSeconds >= 1 {
		m.TimeoutSeconds = req.TimeoutSeconds
	}
	if req.RetryCount >= 1 {
		m.RetryCount = req.RetryCount
	}

	if req.IpAddressID != nil {
		m.IpAddressID.Scan(*req.IpAddressID) //nolint:errcheck
	}
	if req.DeviceID != nil {
		m.DeviceID.Scan(*req.DeviceID) //nolint:errcheck
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
		PushTokenHash:      m.PushTokenHash,
	})

	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	return c.JSON(http.StatusOK, toMonitorResponse(updated))
}

// mergeConfig merges incoming config into existing config, preserving sensitive fields if not overwritten.
func mergeConfig(monitorType string, existing, incoming []byte) []byte {
	if len(existing) == 0 {
		return incoming
	}
	if len(incoming) == 0 {
		return existing
	}

	var existMap, incomingMap map[string]any
	if err := json.Unmarshal(existing, &existMap); err != nil {
		return incoming
	}
	if err := json.Unmarshal(incoming, &incomingMap); err != nil {
		return existing
	}

	sensitiveKeys := []string{"password", "token", "secret", "api_key", "authorization"}
	// For postgres dsn specifically
	if monitorType == "postgres" {
		sensitiveKeys = append(sensitiveKeys, "dsn")
	}

	merged := make(map[string]any)
	// Start with existing
	for k, v := range existMap {
		merged[k] = v
	}
	// Apply incoming, but skip blank sensitive keys
	for k, v := range incomingMap {
		kLower := strings.ToLower(k)
		isSensitive := false
		for _, s := range sensitiveKeys {
			if strings.Contains(kLower, s) {
				isSensitive = true
				break
			}
		}
		if isSensitive {
			if strVal, ok := v.(string); ok && (strVal == "" || strVal == "***") {
				// Blank or masked — preserve existing
				continue
			}
		}
		merged[k] = v
	}

	out, _ := json.Marshal(merged)
	return out
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
		PushTokenHash:      m.PushTokenHash,
	})

	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	return c.JSON(http.StatusOK, toMonitorResponse(updated))
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

	// Push monitors cannot be actively tested
	if m.MonitorType == "push" {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "push monitors cannot be actively tested; use the push endpoint"})
	}

	result, err := h.runner.Run(ctx, m)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	// Update last_checked_at manually (test does not update status)
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
		Status:             m.Status,
		LastCheckedAt:      now,
		LastStatusChangeAt: m.LastStatusChangeAt,
		PushTokenHash:      m.PushTokenHash,
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

// ReceivePushHeartbeat handles GET/POST /push/{token}
// This endpoint does NOT require session/API auth; the token itself is the credential.
func (h *MonitorHandler) ReceivePushHeartbeat(c echo.Context) error {
	ctx := c.Request().Context()
	rawToken := c.Param("token")
	if rawToken == "" {
		return c.JSON(http.StatusNotFound, map[string]string{"error": "not found"})
	}

	hash := monitor.HashToken(rawToken)
	m, err := h.queries.GetMonitorByPushTokenHash(ctx, hash)
	if err != nil {
		// Return generic 404 to not reveal token existence
		return c.JSON(http.StatusNotFound, map[string]string{"error": "not found"})
	}

	if !m.Enabled {
		return c.JSON(http.StatusOK, map[string]string{"status": "monitor paused"})
	}

	// Parse optional parameters
	pushedStatus := c.QueryParam("status")
	if pushedStatus == "" {
		// Try body
		var body struct {
			Status    string  `json:"status"`
			Message   string  `json:"message"`
			LatencyMs float64 `json:"latency_ms"`
		}
		c.Bind(&body) //nolint:errcheck
		if body.Status != "" {
			pushedStatus = body.Status
		}
	}
	if pushedStatus == "" {
		pushedStatus = "up"
	}
	if pushedStatus != "up" && pushedStatus != "down" && pushedStatus != "degraded" {
		pushedStatus = "up"
	}

	now := pgtype.Timestamptz{Time: time.Now(), Valid: true}

	// Store a monitor result
	var latency pgtype.Int4
	_, _ = h.queries.CreateMonitorResult(ctx, db.CreateMonitorResultParams{
		MonitorID:    m.ID,
		Status:       pushedStatus,
		LatencyMs:    latency,
		ErrorMessage: pgtype.Text{},
		Details: func() []byte {
			b, _ := json.Marshal(map[string]any{"source": "push"})
			return b
		}(),
	})

	// Determine new monitor status and update
	newStatus := pushedStatus
	lastStatusChangeAt := m.LastStatusChangeAt
	if newStatus != m.Status {
		lastStatusChangeAt = now
	}

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
		Status:             newStatus,
		LastCheckedAt:      now,
		LastStatusChangeAt: lastStatusChangeAt,
		PushTokenHash:      m.PushTokenHash,
	})

	return c.JSON(http.StatusOK, map[string]string{"status": "ok"})
}

// RotatePushToken generates a new push token for a push monitor.
// Requires operator/admin role.
func (h *MonitorHandler) RotatePushToken(c echo.Context) error {
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

	if m.MonitorType != "push" {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "only push monitors support token rotation"})
	}

	tok, err := monitor.GeneratePushToken()
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "failed to generate token"})
	}

	hash := monitor.HashToken(tok)

	_, err = h.queries.UpdateMonitor(ctx, db.UpdateMonitorParams{
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
		PushTokenHash:      pgtype.Text{String: hash, Valid: true},
	})
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	h.AuditService.Log(ctx, service.AuditParams{
		Action:     "rotate_token",
		EntityType: "monitor",
		EntityID:   m.ID,
		After:      map[string]any{"monitor_name": m.Name},
	})

	return c.JSON(http.StatusOK, map[string]any{
		"token":      tok,
		"message":    "Push token rotated. Store this token securely; it will not be shown again.",
		"push_url":   "/push/" + tok,
	})
}
