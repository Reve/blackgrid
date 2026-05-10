package handlers

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"strings"

	"blackgrid/internal/db"
	"blackgrid/internal/service"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/labstack/echo/v4"
)

type scanReq struct {
	PrefixID string `json:"prefix_id"`
}

type discoveryResultResponse struct {
	ID                 string `json:"id"`
	ScanID             string `json:"scan_id"`
	PrefixID           string `json:"prefix_id"`
	Address            string `json:"address"`
	MacAddress         any    `json:"mac_address"`
	Hostname           any    `json:"hostname"`
	ReverseDNS         any    `json:"reverse_dns"`
	OpenPorts          []int  `json:"open_ports"`
	LatencyMs          any    `json:"latency_ms"`
	Classification     string `json:"classification"`
	SeenAt             any    `json:"seen_at"`
	Ignored            bool   `json:"ignored"`
	AcceptedAt         any    `json:"accepted_at"`
	CreatedIPAddressID any    `json:"created_ip_address_id"`
	CreatedAt          any    `json:"created_at"`
	UpdatedAt          any    `json:"updated_at"`
}

func discoveryResultToResponse(r db.DiscoveryResult) discoveryResultResponse {
	return discoveryResultResponse{
		ID:                 uuidStr(r.ID),
		ScanID:             uuidStr(r.ScanID),
		PrefixID:           uuidStr(r.PrefixID),
		Address:            r.Address.String(),
		MacAddress:         bytesStringOrNil(r.MacAddress),
		Hostname:           textOrNil(r.Hostname),
		ReverseDNS:         textOrNil(r.ReverseDns),
		OpenPorts:          openPortsOrEmpty(r.OpenPorts),
		LatencyMs:          int4OrNil(r.LatencyMs),
		Classification:     r.Classification,
		SeenAt:             timeOrNilTZ(r.SeenAt),
		Ignored:            r.Ignored,
		AcceptedAt:         timeOrNilTZ(r.AcceptedAt),
		CreatedIPAddressID: uuidOrNil(r.CreatedIpAddressID),
		CreatedAt:          timeOrNilTZ(r.CreatedAt),
		UpdatedAt:          timeOrNilTZ(r.UpdatedAt),
	}
}

func discoveryResultsToResponse(results []db.DiscoveryResult) []discoveryResultResponse {
	out := make([]discoveryResultResponse, 0, len(results))
	for _, r := range results {
		out = append(out, discoveryResultToResponse(r))
	}
	return out
}

func openPortsOrEmpty(raw []byte) []int {
	if len(raw) == 0 {
		return []int{}
	}
	var ports []int
	if err := json.Unmarshal(raw, &ports); err != nil {
		return []int{}
	}
	return ports
}

func textOrNil(t pgtype.Text) any {
	if !t.Valid {
		return nil
	}
	return t.String
}

func int4OrNil(i pgtype.Int4) any {
	if !i.Valid {
		return nil
	}
	return i.Int32
}

func uuidOrNil(id pgtype.UUID) any {
	if !id.Valid {
		return nil
	}
	return uuidStr(id)
}

func bytesStringOrNil(b []byte) any {
	if len(b) == 0 {
		return nil
	}
	return string(b)
}

func parsePortsFilter(raw string) ([]int32, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return []int32{}, nil
	}
	seen := make(map[int32]bool)
	ports := make([]int32, 0, 4)
	for _, part := range strings.Split(raw, ",") {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		port, err := strconv.ParseInt(part, 10, 32)
		if err != nil || port < 1 || port > 65535 {
			return nil, errors.New("ports must be comma-separated numbers between 1 and 65535")
		}
		p := int32(port)
		if seen[p] {
			continue
		}
		seen[p] = true
		ports = append(ports, p)
	}
	return ports, nil
}

func (h *Handlers) StartScan(c echo.Context) error {
	var req scanReq
	if err := c.Bind(&req); err != nil {
		return Error(c, ErrCodeValidation, "invalid request", nil)
	}

	var prefixID pgtype.UUID
	if err := prefixID.Scan(req.PrefixID); err != nil {
		return Error(c, ErrCodeValidation, "invalid prefix_id", nil)
	}

	scan, err := h.DiscoveryService.StartManualScan(c.Request().Context(), prefixID)
	if err != nil {
		if errors.Is(err, service.ErrUnknownPrefix) || errors.Is(err, service.ErrInvalidCIDR) {
			return Error(c, ErrCodeValidation, err.Error(), nil)
		}
		if errors.Is(err, service.ErrPrefixTooLarge) || errors.Is(err, service.ErrIPv6Unsupported) {
			return Error(c, ErrCodeValidation, err.Error(), nil)
		}
		if errors.Is(err, service.ErrScanAlreadyRunning) {
			return Error(c, ErrCodeConflict, err.Error(), nil)
		}
		return Error(c, ErrCodeInternal, "internal error", nil)
	}

	return c.JSON(http.StatusCreated, scan)
}

func (h *Handlers) StartPrefixScan(c echo.Context) error {
	var prefixID pgtype.UUID
	if err := prefixID.Scan(c.Param("id")); err != nil {
		return Error(c, ErrCodeValidation, "invalid id format", nil)
	}

	scan, err := h.DiscoveryService.StartManualScan(c.Request().Context(), prefixID)
	if err != nil {
		if errors.Is(err, service.ErrUnknownPrefix) || errors.Is(err, service.ErrInvalidCIDR) {
			return Error(c, ErrCodeValidation, err.Error(), nil)
		}
		if errors.Is(err, service.ErrPrefixTooLarge) || errors.Is(err, service.ErrIPv6Unsupported) {
			return Error(c, ErrCodeValidation, err.Error(), nil)
		}
		if errors.Is(err, service.ErrScanAlreadyRunning) {
			return Error(c, ErrCodeConflict, err.Error(), nil)
		}
		return Error(c, ErrCodeInternal, "internal error", nil)
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
		return Error(c, ErrCodeInternal, "internal error", nil)
	}

	if scans == nil {
		return c.JSON(http.StatusOK, []interface{}{})
	}

	return c.JSON(http.StatusOK, scans)
}

func (h *Handlers) GetScan(c echo.Context) error {
	var id pgtype.UUID
	if err := id.Scan(c.Param("id")); err != nil {
		return Error(c, ErrCodeValidation, "invalid id format", nil)
	}

	scan, err := h.DiscoveryService.GetScan(c.Request().Context(), id)
	if err != nil {
		return Error(c, ErrCodeNotFound, "scan not found", nil)
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

	ports, err := parsePortsFilter(c.QueryParam("ports"))
	if err != nil {
		return Error(c, ErrCodeValidation, err.Error(), nil)
	}

	results, err := h.DiscoveryService.ListResults(c.Request().Context(), scanID, prefixID, classification, ignored, ports, limit, offset)
	if err != nil {
		return Error(c, ErrCodeInternal, "internal error", nil)
	}

	if results == nil {
		return c.JSON(http.StatusOK, []interface{}{})
	}

	return c.JSON(http.StatusOK, discoveryResultsToResponse(results))
}

func (h *Handlers) AcceptDiscoveryResult(c echo.Context) error {
	var id pgtype.UUID
	if err := id.Scan(c.Param("id")); err != nil {
		return Error(c, ErrCodeValidation, "invalid id format", nil)
	}

	var req service.AcceptResultInput
	if err := c.Bind(&req); err != nil {
		return Error(c, ErrCodeValidation, "invalid request", nil)
	}

	ip, err := h.DiscoveryService.AcceptResult(c.Request().Context(), id, req)
	if err != nil {
		return Error(c, ErrCodeInternal, "internal error", nil)
	}

	LogAudit(h.AuditService, c, service.AuditParams{
		Action:     "discovery.result_accept",
		EntityType: "discovery_result",
		EntityID:   id,
		After:      map[string]any{"ip_address": ip.IpAddress},
	})
	return c.JSON(http.StatusOK, ip)
}

func (h *Handlers) IgnoreDiscoveryResult(c echo.Context) error {
	var id pgtype.UUID
	if err := id.Scan(c.Param("id")); err != nil {
		return Error(c, ErrCodeValidation, "invalid id format", nil)
	}

	res, err := h.DiscoveryService.IgnoreResult(c.Request().Context(), id)
	if err != nil {
		return Error(c, ErrCodeInternal, "internal error", nil)
	}

	LogAudit(h.AuditService, c, service.AuditParams{
		Action:     "discovery.result_ignore",
		EntityType: "discovery_result",
		EntityID:   id,
	})

	return c.JSON(http.StatusOK, discoveryResultToResponse(res))
}

func getQueryInt32(c echo.Context, key string, fallback int32) int32 {
	if val := c.QueryParam(key); val != "" {
		if i, err := strconv.ParseInt(val, 10, 32); err == nil {
			return int32(i)
		}
	}
	return fallback
}
