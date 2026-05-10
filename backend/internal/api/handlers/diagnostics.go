package handlers

import (
	"net/http"
	"time"

	"blackgrid/internal/events"
	"blackgrid/internal/monitor"
	"blackgrid/internal/service"
	"blackgrid/internal/version"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/labstack/echo/v4"
)

// DiagnosticsHandler exposes admin-only operational diagnostics.
//
// It deliberately does not surface secrets (no env var dump, no notification
// channel configs). The shape mirrors what the Settings → Diagnostics tab
// renders so the UI does not need to stitch multiple endpoints together.
type DiagnosticsHandler struct {
	pool             *pgxpool.Pool
	bus              *events.EventBus
	monitorScheduler *monitor.Scheduler
	discoverySvc     *service.DiscoveryService
	retentionCfg     service.RetentionConfig
}

func NewDiagnosticsHandler(
	pool *pgxpool.Pool,
	bus *events.EventBus,
	ms *monitor.Scheduler,
	ds *service.DiscoveryService,
	retentionCfg service.RetentionConfig,
) *DiagnosticsHandler {
	return &DiagnosticsHandler{
		pool:             pool,
		bus:              bus,
		monitorScheduler: ms,
		discoverySvc:     ds,
		retentionCfg:     retentionCfg,
	}
}

func (h *DiagnosticsHandler) Get(c echo.Context) error {
	ctx := c.Request().Context()

	dbStatus := "ok"
	var dbErr string
	if err := h.pool.Ping(ctx); err != nil {
		dbStatus = "error"
		dbErr = err.Error()
	}

	mStats := h.monitorScheduler.Stats()
	dStats := h.discoverySvc.Stats()

	var nextDue *time.Time
	if t, err := h.monitorScheduler.NextDueCheck(ctx); err == nil && !t.IsZero() {
		nextDue = &t
	}

	role := GetAuthRole(c)

	return c.JSON(http.StatusOK, map[string]any{
		"version": version.Get(),
		"database": map[string]any{
			"status": dbStatus,
			"error":  dbErr,
		},
		"monitor_scheduler": map[string]any{
			"running":       mStats.Running,
			"worker_count":  mStats.WorkerCount,
			"last_tick_at":  nullableTime(mStats.LastTickAt),
			"next_due_at":   nextDue,
		},
		"discovery_scheduler": map[string]any{
			"running":        dStats.SchedulerRunning,
			"worker_count":   dStats.WorkerCount,
			"last_tick_at":   nullableTime(dStats.LastTickAt),
			"running_scans":  dStats.RunningScans,
		},
		"events": map[string]any{
			"sse_clients": h.bus.SubscriberCount(),
		},
		"retention": map[string]any{
			"monitor_results_days":         h.retentionCfg.MonitorResultsDays,
			"notification_deliveries_days": h.retentionCfg.NotificationDeliveriesDays,
			"audit_log_days":               h.retentionCfg.AuditLogDays,
			"discovery_results_days":       h.retentionCfg.DiscoveryResultsDays,
			"discovery_scans_days":         h.retentionCfg.DiscoveryScansDays,
			"interval_hours":               h.retentionCfg.IntervalHours,
		},
		"current_user_role": role,
		"server_time":       time.Now().UTC(),
	})
}

func nullableTime(t time.Time) any {
	if t.IsZero() {
		return nil
	}
	return t.UTC()
}
