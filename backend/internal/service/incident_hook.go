package service

import (
	"context"
	"fmt"
	"log"

	"blackgrid/internal/db"
)

// IncidentHook bridges the monitor scheduler to the incident service so the
// scheduler doesn't need to know about incident concerns directly.
type IncidentHook struct {
	svc *IncidentService
}

func NewIncidentHook(svc *IncidentService) *IncidentHook {
	return &IncidentHook{svc: svc}
}

func (h *IncidentHook) OnScheduledStatusChange(ctx context.Context, monitor db.Monitor, oldStatus, newStatus string) {
	switch newStatus {
	case "down":
		summary := fmt.Sprintf("%s is down", monitor.Name)
		details := fmt.Sprintf("Monitor %s (%s) transitioned from %s to down", monitor.Name, monitor.MonitorType, oldStatus)
		if _, _, err := h.svc.OpenForMonitor(ctx, monitor, "critical", summary, details); err != nil {
			log.Printf("incident open failed for monitor %s: %v", monitor.Name, err)
		}
	case "degraded":
		summary := fmt.Sprintf("%s is degraded", monitor.Name)
		details := fmt.Sprintf("Monitor %s (%s) transitioned from %s to degraded", monitor.Name, monitor.MonitorType, oldStatus)
		if _, _, err := h.svc.OpenForMonitor(ctx, monitor, "warning", summary, details); err != nil {
			log.Printf("incident open failed for monitor %s: %v", monitor.Name, err)
		}
	case "up":
		reason := fmt.Sprintf("Monitor %s recovered (was %s)", monitor.Name, oldStatus)
		if _, _, err := h.svc.ResolveForMonitor(ctx, monitor, reason); err != nil {
			log.Printf("incident resolve failed for monitor %s: %v", monitor.Name, err)
		}
	}
}
