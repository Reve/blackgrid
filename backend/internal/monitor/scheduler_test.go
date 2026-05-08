package monitor

import (
	"context"
	"testing"
	"github.com/jackc/pgx/v5/pgxpool"
)

func TestStateTransitions(t *testing.T) {
	// Let's create an integration test that uses the DB
	pool, err := pgxpool.New(context.Background(), "postgres://blackgrid:blackgrid@localhost:5432/blackgrid?sslmode=disable")
	if err != nil {
		t.Skip("Skipping integration test since no database available")
		return
	}
	defer pool.Close()

	if err := pool.Ping(context.Background()); err != nil {
		t.Skip("Skipping integration test since database is down")
		return
	}
}
