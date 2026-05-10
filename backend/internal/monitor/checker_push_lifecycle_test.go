package monitor

import (
	"context"
	"strings"
	"testing"
	"time"

	"blackgrid/internal/db"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
)

// pushTestPool returns a connection to the test database, or skips the test.
func pushTestPool(t *testing.T) *pgxpool.Pool {
	t.Helper()
	pool, err := pgxpool.New(context.Background(), "postgres://blackgrid:blackgrid@localhost:5432/blackgrid?sslmode=disable")
	if err != nil {
		t.Skip("Skipping integration test: cannot connect to db")
	}
	if err := pool.Ping(context.Background()); err != nil {
		pool.Close()
		t.Skip("Skipping integration test: db unreachable")
	}
	return pool
}

// requirePushHeartbeatColumn skips if migration 007 has not been applied.
func requirePushHeartbeatColumn(t *testing.T, pool *pgxpool.Pool) {
	t.Helper()
	var exists bool
	err := pool.QueryRow(context.Background(),
		`SELECT EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name='monitors' AND column_name='last_heartbeat_at')`).
		Scan(&exists)
	if err != nil || !exists {
		t.Skip("Skipping: monitors.last_heartbeat_at column missing (run migration 007)")
	}
}

func newPushMonitor(t *testing.T, q *db.Queries) db.Monitor {
	t.Helper()
	suffix := strings.ReplaceAll(time.Now().Format("150405.000000"), ".", "")
	cfg := []byte(`{"grace_seconds":30}`)
	tok, _ := GeneratePushToken()
	hash := pgtype.Text{String: HashToken(tok), Valid: true}
	m, err := q.CreateMonitor(context.Background(), db.CreateMonitorParams{
		Name:            "push-" + suffix,
		Slug:            "push-" + suffix,
		MonitorType:     "push",
		Target:          "",
		Config:          cfg,
		IntervalSeconds: 30,
		TimeoutSeconds:  5,
		RetryCount:      1,
		Enabled:         true,
		Status:          "unknown",
		PushTokenHash:   hash,
	})
	if err != nil {
		t.Fatalf("create push monitor: %v", err)
	}
	t.Cleanup(func() { _ = q.DeleteMonitor(context.Background(), m.ID) })
	return m
}

// TestPushMonitor_NoHeartbeatStaysDown ensures the scheduled overdue check
// does not flip a never-heartbeated monitor to up just because it ran.
func TestPushMonitor_NoHeartbeatStaysDown(t *testing.T) {
	pool := pushTestPool(t)
	defer pool.Close()
	requirePushHeartbeatColumn(t, pool)
	q := db.New(pool)
	m := newPushMonitor(t, q)

	c := &PushChecker{}
	first := c.Check(context.Background(), m)
	if first.Status != "down" {
		t.Fatalf("expected down for never-heartbeated monitor, got %s", first.Status)
	}

	// Simulate the scheduler bumping last_checked_at without a heartbeat
	// (the dedicated check column moves; the heartbeat column does NOT).
	now := pgtype.Timestamptz{Time: time.Now(), Valid: true}
	_, err := q.UpdateMonitor(context.Background(), db.UpdateMonitorParams{
		ID: m.ID, Name: m.Name, Slug: m.Slug, MonitorType: m.MonitorType,
		Target: m.Target, Config: m.Config, IpAddressID: m.IpAddressID,
		DeviceID: m.DeviceID, IntervalSeconds: m.IntervalSeconds,
		TimeoutSeconds: m.TimeoutSeconds, RetryCount: m.RetryCount,
		Enabled: m.Enabled, Status: "down", LastCheckedAt: now,
		LastStatusChangeAt: now, PushTokenHash: m.PushTokenHash,
	})
	if err != nil {
		t.Fatalf("update: %v", err)
	}

	reloaded, err := q.GetMonitor(context.Background(), m.ID)
	if err != nil {
		t.Fatalf("reload: %v", err)
	}
	if reloaded.LastHeartbeatAt.Valid {
		t.Fatal("scheduler must NOT set last_heartbeat_at; only push handler does")
	}
	again := c.Check(context.Background(), reloaded)
	if again.Status != "down" {
		t.Errorf("monitor must remain down after scheduler check, got %s", again.Status)
	}
}

// TestPushMonitor_RecentHeartbeatIsUp verifies that a fresh heartbeat
// recorded via RecordPushHeartbeat results in an "up" check.
func TestPushMonitor_RecentHeartbeatIsUp(t *testing.T) {
	pool := pushTestPool(t)
	defer pool.Close()
	requirePushHeartbeatColumn(t, pool)
	q := db.New(pool)
	m := newPushMonitor(t, q)

	now := pgtype.Timestamptz{Time: time.Now(), Valid: true}
	updated, err := q.RecordPushHeartbeat(context.Background(), db.RecordPushHeartbeatParams{
		ID: m.ID, LastHeartbeatAt: now, Status: "up", LastStatusChangeAt: now,
	})
	if err != nil {
		t.Fatalf("record heartbeat: %v", err)
	}
	if !updated.LastHeartbeatAt.Valid {
		t.Fatal("RecordPushHeartbeat did not set last_heartbeat_at")
	}

	c := &PushChecker{}
	res := c.Check(context.Background(), updated)
	if res.Status != "up" {
		t.Errorf("expected up, got %s (%s)", res.Status, res.ErrorMessage)
	}
}

// TestPushMonitor_BecomesDownAfterGrace verifies the grace_seconds threshold.
func TestPushMonitor_BecomesDownAfterGrace(t *testing.T) {
	pool := pushTestPool(t)
	defer pool.Close()
	requirePushHeartbeatColumn(t, pool)
	q := db.New(pool)
	m := newPushMonitor(t, q) // grace_seconds=30

	old := pgtype.Timestamptz{Time: time.Now().Add(-90 * time.Second), Valid: true}
	updated, err := q.RecordPushHeartbeat(context.Background(), db.RecordPushHeartbeatParams{
		ID: m.ID, LastHeartbeatAt: old, Status: "up", LastStatusChangeAt: old,
	})
	if err != nil {
		t.Fatalf("record heartbeat: %v", err)
	}

	c := &PushChecker{}
	res := c.Check(context.Background(), updated)
	if res.Status != "down" {
		t.Errorf("expected down past grace, got %s", res.Status)
	}
	if res.ErrorMessage != "heartbeat overdue" {
		t.Errorf("expected 'heartbeat overdue', got %q", res.ErrorMessage)
	}
}
