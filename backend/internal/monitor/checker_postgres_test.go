package monitor

import (
	"context"
	"testing"

	"blackgrid/internal/db"
)

func TestPostgresChecker_Helper(t *testing.T) {
	dsn := "postgres://user:pass@host:5432/dbname?sslmode=disable"
	host, dbName := parseDSNSafe(dsn)
	if host != "host" {
		t.Errorf("expected host, got %s", host)
	}
	if dbName != "dbname" {
		t.Errorf("expected dbname, got %s", dbName)
	}
}

func TestPostgresChecker_CheckFailures(t *testing.T) {
	c := &PostgresChecker{}

	t.Run("missing dsn", func(t *testing.T) {
		m := db.Monitor{Config: []byte("{}")}
		res := c.Check(context.Background(), m)
		if res.Status != "down" || res.ErrorMessage != "dsn is required" {
			t.Errorf("expected dsn is required error, got %s: %s", res.Status, res.ErrorMessage)
		}
	})

	t.Run("invalid connection", func(t *testing.T) {
		// This should fail to connect/ping immediately
		m := db.Monitor{
			Config:         []byte(`{"dsn": "postgres://user:pass@localhost:9999/none"}`),
			TimeoutSeconds: 1,
		}
		res := c.Check(context.Background(), m)
		if res.Status != "down" {
			t.Errorf("expected status down for invalid connection, got %s", res.Status)
		}
	})
}
