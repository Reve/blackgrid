package handlers

import (
	"errors"
	"net/http"
	"strconv"

	"blackgrid/internal/service"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/labstack/echo/v4"
)

type scanReq struct {
	PrefixID string `json:"prefix_id"`
}

func (h *Handlers) StartScan(c echo.Context) error {
	var req scanReq
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid request"})
	}

	var prefixID pgtype.UUID
	if err := prefixID.Scan(req.PrefixID); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid prefix_id"})
	}

	scan, err := h.DiscoveryService.StartManualScan(c.Request().Context(), prefixID)
	if err != nil {
		if errors.Is(err, service.ErrUnknownPrefix) || errors.Is(err, service.ErrInvalidCIDR) {
			return c.JSON(http.StatusBadRequest, map[string]string{"error": err.Error()})
		}
		if errors.Is(err, service.ErrPrefixTooLarge) || errors.Is(err, service.ErrIPv6Unsupported) {
			return c.JSON(http.StatusUnprocessableEntity, map[string]string{"error": err.Error()})
		}
		if errors.Is(err, service.ErrScanAlreadyRunning) {
			return c.JSON(http.StatusConflict, map[string]string{"error": err.Error()})
		}
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	return c.JSON(http.StatusCreated, scan)
}

func (h *Handlers) StartPrefixScan(c echo.Context) error {
	var prefixID pgtype.UUID
	if err := prefixID.Scan(c.Param("id")); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid id format"})
	}

	scan, err := h.DiscoveryService.StartManualScan(c.Request().Context(), prefixID)
	if err != nil {
		if errors.Is(err, service.ErrUnknownPrefix) || errors.Is(err, service.ErrInvalidCIDR) {
			return c.JSON(http.StatusBadRequest, map[string]string{"error": err.Error()})
		}
		if errors.Is(err, service.ErrPrefixTooLarge) || errors.Is(err, service.ErrIPv6Unsupported) {
			return c.JSON(http.StatusUnprocessableEntity, map[string]string{"error": err.Error()})
		}
		if errors.Is(err, service.ErrScanAlreadyRunning) {
			return c.JSON(http.StatusConflict, map[string]string{"error": err.Error()})
		}
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	return c.JSON(http.StatusCreated, scan)
}

func (h *Handlers) GetScans(c echo.Context) error {
	var prefixID pgtype.UUID
	if pid := c.QueryParam("prefix_id"); pid != "" {
		_ = prefixID.Scan(pid)
	}

	status := c.QueryParam("status")
	limit := getQueryInt32(c, "limit", 100)
	offset := getQueryInt32(c, "offset", 0)

	scans, err := h.DiscoveryService.ListScans(c.Request().Context(), prefixID, status, limit, offset)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	if scans == nil {
		return c.JSON(http.StatusOK, []interface{}{})
	}

	return c.JSON(http.StatusOK, scans)
}

func (h *Handlers) GetScan(c echo.Context) error {
	var id pgtype.UUID
	if err := id.Scan(c.Param("id")); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid id format"})
	}

	scan, err := h.DiscoveryService.GetScan(c.Request().Context(), id)
	if err != nil {
		return c.JSON(http.StatusNotFound, map[string]string{"error": "scan not found"})
	}

	return c.JSON(http.StatusOK, scan)
}

func (h *Handlers) GetDiscoveryResults(c echo.Context) error {
	var scanID pgtype.UUID
	if sid := c.QueryParam("scan_id"); sid != "" {
		_ = scanID.Scan(sid)
	}

	var prefixID pgtype.UUID
	if pid := c.QueryParam("prefix_id"); pid != "" {
		_ = prefixID.Scan(pid)
	}

	classification := c.QueryParam("classification")

	var ignored *bool
	if ig := c.QueryParam("ignored"); ig != "" {
		if val, err := strconv.ParseBool(ig); err == nil {
			ignored = &val
		}
	}

	limit := getQueryInt32(c, "limit", 100)
	offset := getQueryInt32(c, "offset", 0)

	results, err := h.DiscoveryService.ListResults(c.Request().Context(), scanID, prefixID, classification, ignored, limit, offset)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	if results == nil {
		return c.JSON(http.StatusOK, []interface{}{})
	}

	return c.JSON(http.StatusOK, results)
}

func (h *Handlers) AcceptDiscoveryResult(c echo.Context) error {
	var id pgtype.UUID
	if err := id.Scan(c.Param("id")); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid id format"})
	}

	var req service.AcceptResultInput
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid request"})
	}

	ip, err := h.DiscoveryService.AcceptResult(c.Request().Context(), id, req)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	h.AuditService.Log(c.Request().Context(), service.AuditParams{
		Action:     "accept_result",
		EntityType: "discovery_result",
		EntityID:   id,
		After:      map[string]any{"ip_address": ip.IpAddress},
	})
	return c.JSON(http.StatusOK, ip)
}

func (h *Handlers) IgnoreDiscoveryResult(c echo.Context) error {
	var id pgtype.UUID
	if err := id.Scan(c.Param("id")); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid id format"})
	}

	res, err := h.DiscoveryService.IgnoreResult(c.Request().Context(), id)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	return c.JSON(http.StatusOK, res)
}

func getQueryInt32(c echo.Context, key string, fallback int32) int32 {
	if val := c.QueryParam(key); val != "" {
		if i, err := strconv.ParseInt(val, 10, 32); err == nil {
			return int32(i)
		}
	}
	return fallback
}
