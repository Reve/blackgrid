package monitor

import (
	"context"
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

	_, err := r.queries.CreateMonitorResult(ctx, db.CreateMonitorResultParams{
		MonitorID:    monitor.ID,
		Status:       result.Status,
		LatencyMs:    latency,
		ErrorMessage: errorMsg,
	})
	if err != nil {
		return result, fmt.Errorf("failed to store result: %w", err)
	}

	return result, nil
}
