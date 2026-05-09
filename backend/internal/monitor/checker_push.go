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
// When called by the scheduler the monitor's last_checked_at tells us
// when the last heartbeat was received. If now - last_checked_at > grace_seconds
// the monitor is considered overdue → down.
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

	if !m.LastCheckedAt.Valid {
		// Never received a heartbeat
		return CheckResult{
			Status:       "down",
			ErrorMessage: "heartbeat overdue: no heartbeat received",
			Details: map[string]any{
				"grace_seconds": grace,
			},
		}
	}

	elapsed := time.Since(m.LastCheckedAt.Time)
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
