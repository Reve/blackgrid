package monitor

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"net/url"
	"strings"
	"time"

	"blackgrid/internal/db"

	_ "github.com/jackc/pgx/v5/stdlib" // pgx stdlib driver
)

// PostgresConfig holds configuration for the PostgreSQL monitor.
type PostgresConfig struct {
	DSN   string `json:"dsn"`
	Query string `json:"query"`
}

// PostgresChecker implements Checker for PostgreSQL monitors.
type PostgresChecker struct{}

func (c *PostgresChecker) Check(ctx context.Context, m db.Monitor) CheckResult {
	start := time.Now()

	var cfg PostgresConfig
	if m.Config != nil {
		if err := json.Unmarshal(m.Config, &cfg); err != nil {
			return CheckResult{Status: "down", ErrorMessage: fmt.Sprintf("invalid config: %v", err)}
		}
	}

	if cfg.DSN == "" {
		return CheckResult{Status: "down", ErrorMessage: "dsn is required"}
	}

	query := cfg.Query
	if query == "" {
		query = "SELECT 1"
	}

	timeout := time.Duration(m.TimeoutSeconds) * time.Second
	if timeout <= 0 {
		timeout = 10 * time.Second
	}

	// Parse safe details (no credentials)
	dbHost, dbName := parseDSNSafe(cfg.DSN)

	connCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	// Open connection using pgx stdlib driver
	sqlDB, err := sql.Open("pgx", cfg.DSN)
	if err != nil {
		latencyMs := int32(time.Since(start).Milliseconds())
		return CheckResult{
			Status:       "down",
			LatencyMs:    latencyMs,
			ErrorMessage: fmt.Sprintf("failed to open connection: %v", err),
			Details: map[string]any{
				"db_host": dbHost,
				"db_name": dbName,
			},
		}
	}
	defer sqlDB.Close()

	sqlDB.SetMaxOpenConns(1)
	sqlDB.SetConnMaxLifetime(timeout)

	if err := sqlDB.PingContext(connCtx); err != nil {
		latencyMs := int32(time.Since(start).Milliseconds())
		return CheckResult{
			Status:       "down",
			LatencyMs:    latencyMs,
			ErrorMessage: fmt.Sprintf("connection failed: %v", err),
			Details: map[string]any{
				"db_host": dbHost,
				"db_name": dbName,
			},
		}
	}

	row := sqlDB.QueryRowContext(connCtx, query)
	var result any
	if err := row.Scan(&result); err != nil {
		latencyMs := int32(time.Since(start).Milliseconds())
		return CheckResult{
			Status:       "down",
			LatencyMs:    latencyMs,
			ErrorMessage: fmt.Sprintf("query failed: %v", err),
			Details: map[string]any{
				"db_host":    dbHost,
				"db_name":    dbName,
				"query_used": true,
			},
		}
	}

	latencyMs := int32(time.Since(start).Milliseconds())
	return CheckResult{
		Status:    "up",
		LatencyMs: latencyMs,
		Details: map[string]any{
			"db_host":    dbHost,
			"db_name":    dbName,
			"query_used": true,
		},
	}
}

// parseDSNSafe extracts host and database name from a DSN without exposing credentials.
func parseDSNSafe(dsn string) (host, dbName string) {
	u, err := url.Parse(dsn)
	if err != nil {
		// Try simple extraction
		return "", ""
	}
	host = u.Hostname()
	dbName = strings.TrimPrefix(u.Path, "/")
	return host, dbName
}
