package handlers

import (
	"errors"
	"net/http"
	"os"

	"blackgrid/internal/service"

	"github.com/labstack/echo/v4"
)

// GetDiscoveryDiagnostics returns a snapshot of the discovery service config
// and the runtime environment. Operator/admin only. Does not surface secrets.
func (h *Handlers) GetDiscoveryDiagnostics(c echo.Context) error {
	svc := h.DiscoveryService

	hostname, _ := os.Hostname()

	resp := map[string]any{
		"worker_count":   svc.WorkerCount(),
		"default_ports":  svc.DefaultPorts(),
		"tcp_timeout_ms": svc.TCPTimeoutMs(),
		"ping_supported": svc.PingEnabled(),
		"runtime": map[string]any{
			"inside_container": insideContainer(),
			"hostname":         hostname,
		},
	}
	return c.JSON(http.StatusOK, resp)
}

// insideContainer is a best-effort heuristic. It does not affect security; it
// is purely informational on the diagnostics screen.
func insideContainer() bool {
	if _, err := os.Stat("/.dockerenv"); err == nil {
		return true
	}
	if _, err := os.Stat("/run/.containerenv"); err == nil {
		return true
	}
	return false
}

type probeRequest struct {
	Address string `json:"address"`
	Ports   []int  `json:"ports"`
}

// PostDiscoveryProbe runs a one-off probe against an address that belongs to a
// stored prefix. Operator/admin only.
func (h *Handlers) PostDiscoveryProbe(c echo.Context) error {
	var req probeRequest
	if err := c.Bind(&req); err != nil {
		return Error(c, ErrCodeValidation, "invalid request", nil)
	}
	if req.Address == "" {
		return Error(c, ErrCodeValidation, "address is required", nil)
	}
	for _, p := range req.Ports {
		if p < 1 || p > 65535 {
			return Error(c, ErrCodeValidation, "invalid port in list", nil)
		}
	}

	res, err := h.DiscoveryService.ProbeAddress(c.Request().Context(), req.Address, req.Ports)
	if err != nil {
		if errors.Is(err, service.ErrInvalidCIDR) {
			return Error(c, ErrCodeValidation, "invalid address", nil)
		}
		if errors.Is(err, service.ErrUnknownPrefix) {
			return Error(c, ErrCodeValidation, "address is not within a stored prefix", nil)
		}
		return Error(c, ErrCodeInternal, "internal error", nil)
	}

	openPorts := res.OpenPorts
	if openPorts == nil {
		openPorts = []int{}
	}

	return c.JSON(http.StatusOK, map[string]any{
		"address":     req.Address,
		"seen":        res.Seen,
		"open_ports":  openPorts,
		"latency_ms":  res.LatencyMs,
		"reverse_dns": res.ReverseDNS,
	})
}
