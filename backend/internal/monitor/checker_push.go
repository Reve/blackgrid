package monitor

import (
	"context"
	"encoding/json"
	"time"

	"blackgrid/internal/db"
)

// PushConfig holds configuration for the push heartbeat monitor.
type PushConfig struct {
	// Token is the raw push token — only set on creation, not read back.
	// In storage the token hash is kept in push_token_hash column.
	GraceSeconds int `json:"grace_seconds"`
}

// PushChecker implements the passive overdue check for push monitors.
// It compares now against last_heartbeat_at — *not* last_checked_at, which
// is bumped by the scheduler itself whenever it evaluates this monitor.
// If now - last_heartbeat_at > grace_seconds the monitor is overdue → down.
type PushChecker struct{}

func (c *PushChecker) Check(ctx context.Context, m db.Monitor) CheckResult {
	var cfg PushConfig
	if m.Config != nil {
		json.Unmarshal(m.Config, &cfg) //nolint:errcheck
	}

	grace := cfg.GraceSeconds
	if grace <= 0 {
		grace = 120
	}

	if !m.LastHeartbeatAt.Valid {
		// Never received a heartbeat
		return CheckResult{
			Status:       "down",
			ErrorMessage: "heartbeat overdue: no heartbeat received",
			Details: map[string]any{
				"grace_seconds": grace,
			},
		}
	}

	elapsed := time.Since(m.LastHeartbeatAt.Time)
	if elapsed > time.Duration(grace)*time.Second {
		return CheckResult{
			Status:       "down",
			ErrorMessage: "heartbeat overdue",
			Details: map[string]any{
				"grace_seconds":  grace,
				"elapsed_seconds": int(elapsed.Seconds()),
			},
		}
	}

	return CheckResult{
		Status: "up",
		Details: map[string]any{
			"grace_seconds":  grace,
			"elapsed_seconds": int(elapsed.Seconds()),
		},
	}
}
