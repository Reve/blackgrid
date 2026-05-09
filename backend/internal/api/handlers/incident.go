package handlers

import (
	"errors"
	"net/http"
	"strconv"

	"blackgrid/internal/db"
	"blackgrid/internal/service"

	"github.com/jackc/pgx/v5/pgtype"
	"github.com/labstack/echo/v4"
)

type IncidentHandler struct {
	svc *service.IncidentService
}

func NewIncidentHandler(svc *service.IncidentService) *IncidentHandler {
	return &IncidentHandler{svc: svc}
}

func (h *IncidentHandler) ListIncidents(c echo.Context) error {
	ctx := c.Request().Context()

	status := c.QueryParam("status")
	severity := c.QueryParam("severity")

	limit := int32(100)
	offset := int32(0)
	if v := c.QueryParam("limit"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			limit = int32(n)
		}
	}
	if v := c.QueryParam("offset"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n >= 0 {
			offset = int32(n)
		}
	}

	var monitorID pgtype.UUID
	if v := c.QueryParam("monitor_id"); v != "" {
		if err := monitorID.Scan(v); err != nil {
			return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid monitor_id"})
		}
	}

	incidents, err := h.svc.List(ctx, db.ListIncidentsParams{
		Status:    status,
		Severity:  severity,
		MonitorID: monitorID,
		Limit:     limit,
		Offset:    offset,
	})
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}
	return c.JSON(http.StatusOK, incidents)
}

func (h *IncidentHandler) GetIncident(c echo.Context) error {
	ctx := c.Request().Context()
	var id pgtype.UUID
	if err := id.Scan(c.Param("id")); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid id"})
	}

	inc, err := h.svc.Get(ctx, id)
	if err != nil {
		if errors.Is(err, service.ErrIncidentNotFound) {
			return c.JSON(http.StatusNotFound, map[string]string{"error": "incident not found"})
		}
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}
	return c.JSON(http.StatusOK, inc)
}

type incidentNoteRequest struct {
	Note string `json:"note"`
}

func (h *IncidentHandler) AcknowledgeIncident(c echo.Context) error {
	ctx := c.Request().Context()
	var id pgtype.UUID
	if err := id.Scan(c.Param("id")); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid id"})
	}

	var req incidentNoteRequest
	_ = c.Bind(&req)

	inc, err := h.svc.Acknowledge(ctx, id, req.Note)
	if err != nil {
		switch {
		case errors.Is(err, service.ErrIncidentNotFound):
			return c.JSON(http.StatusNotFound, map[string]string{"error": "incident not found"})
		case errors.Is(err, service.ErrIncidentAlreadyResolved):
			return c.JSON(http.StatusConflict, map[string]string{"error": "incident is already resolved"})
		default:
			return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
		}
	}
	return c.JSON(http.StatusOK, inc)
}

func (h *IncidentHandler) ResolveIncident(c echo.Context) error {
	ctx := c.Request().Context()
	var id pgtype.UUID
	if err := id.Scan(c.Param("id")); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid id"})
	}

	var req incidentNoteRequest
	_ = c.Bind(&req)

	inc, err := h.svc.Resolve(ctx, id, req.Note)
	if err != nil {
		if errors.Is(err, service.ErrIncidentNotFound) {
			return c.JSON(http.StatusNotFound, map[string]string{"error": "incident not found"})
		}
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}
	return c.JSON(http.StatusOK, inc)
}

func (h *IncidentHandler) IncidentCounts(c echo.Context) error {
	ctx := c.Request().Context()
	counts, err := h.svc.Counts(ctx)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}
	return c.JSON(http.StatusOK, counts)
}
