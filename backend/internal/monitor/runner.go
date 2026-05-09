package monitor

import (
	"context"
	"encoding/json"
	"fmt"

	"blackgrid/internal/db"
	"github.com/jackc/pgx/v5/pgtype"
)

type Runner struct {
	queries *db.Queries
}

func NewRunner(queries *db.Queries) *Runner {
	return &Runner{
		queries: queries,
	}
}

func (r *Runner) Run(ctx context.Context, monitor db.Monitor) (CheckResult, error) {
	var checker Checker

	switch monitor.MonitorType {
	case "http":
		checker = &HTTPChecker{}
	case "tcp":
		checker = &TCPChecker{}
	case "ping":
		checker = &PingChecker{}
	case "dns":
		checker = &DNSChecker{}
	case "tls":
		checker = &TLSChecker{}
	case "push":
		// Push monitors are not actively probed by the runner.
		// The scheduler uses a special overdue-check; active runs are skipped here.
		checker = &PushChecker{}
	case "postgres":
		checker = &PostgresChecker{}
	default:
		return CheckResult{}, fmt.Errorf("unsupported monitor type: %s", monitor.MonitorType)
	}

	result := checker.Check(ctx, monitor)

	var errorMsg pgtype.Text
	if result.ErrorMessage != "" {
		errorMsg = pgtype.Text{String: result.ErrorMessage, Valid: true}
	}

	var latency pgtype.Int4
	if result.LatencyMs > 0 {
		latency = pgtype.Int4{Int32: result.LatencyMs, Valid: true}
	}

	var detailsBytes []byte
	if result.Details != nil {
		b, _ := json.Marshal(result.Details)
		detailsBytes = b
	}

	_, err := r.queries.CreateMonitorResult(ctx, db.CreateMonitorResultParams{
		MonitorID:    monitor.ID,
		Status:       result.Status,
		LatencyMs:    latency,
		ErrorMessage: errorMsg,
		Details:      detailsBytes,
	})
	if err != nil {
		return result, fmt.Errorf("failed to store result: %w", err)
	}

	return result, nil
}
