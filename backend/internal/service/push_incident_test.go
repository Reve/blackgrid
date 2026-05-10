package service

import (
	"context"
	"strings"
	"testing"
	"time"

	"blackgrid/internal/db"
	"github.com/jackc/pgx/v5/pgtype"
)

func newPushIncidentMonitor(t *testing.T, q *db.Queries) db.Monitor {
	t.Helper()
	suffix := strings.ReplaceAll(time.Now().Format("150405.000000"), ".", "")
	m, err := q.CreateMonitor(context.Background(), db.CreateMonitorParams{
		Name:            "push-inc-" + suffix,
		Slug:            "push-inc-" + suffix,
		MonitorType:     "push",
		Target:          "",
		Config:          []byte(`{"grace_seconds":30}`),
		IntervalSeconds: 30,
		TimeoutSeconds:  5,
		RetryCount:      1,
		Enabled:         true,
		Status:          "up", // start healthy so first push=down is a transition
	})
	if err != nil {
		t.Fatalf("create push monitor: %v", err)
	}
	t.Cleanup(func() { _ = q.DeleteMonitor(context.Background(), m.ID) })
	return m
}

// TestPushIncidentLifecycle verifies that pushed status=down opens an incident
// and pushed status=up resolves it, using the same hook as scheduled checks.
func TestPushIncidentLifecycle(t *testing.T) {
	pool := newTestPool(t)
	defer pool.Close()
	requireSchema(t, pool)

	q := db.New(pool)
	m := newPushIncidentMonitor(t, q)

	incidentSvc := NewIncidentService(q, nil)
	hook := NewIncidentHook(incidentSvc)

	now := pgtype.Timestamptz{Time: time.Now(), Valid: true}

	// Simulate push handler: status=down arrives, monitor transitions up→down.
	updated, err := q.RecordPushHeartbeat(context.Background(), db.RecordPushHeartbeatParams{
		ID: m.ID, LastHeartbeatAt: now, Status: "down", LastStatusChangeAt: now,
	})
	if err != nil {
		t.Fatalf("record down heartbeat: %v", err)
	}
	hook.OnScheduledStatusChange(context.Background(), updated, "up", "down")

	open, err := q.GetOpenIncidentForMonitor(context.Background(), m.ID)
	if err != nil {
		t.Fatalf("expected open incident after push down, got: %v", err)
	}
	if open.Status != "open" && open.Status != "acknowledged" {
		t.Errorf("expected open incident, got status %s", open.Status)
	}
	t.Cleanup(func() {
		_, _ = pool.Exec(context.Background(), `DELETE FROM incidents WHERE id = $1`, open.ID)
	})

	// Recovery: status=up arrives, monitor transitions down→up.
	recovered, err := q.RecordPushHeartbeat(context.Background(), db.RecordPushHeartbeatParams{
		ID: m.ID, LastHeartbeatAt: now, Status: "up", LastStatusChangeAt: now,
	})
	if err != nil {
		t.Fatalf("record up heartbeat: %v", err)
	}
	hook.OnScheduledStatusChange(context.Background(), recovered, "down", "up")

	if _, err := q.GetOpenIncidentForMonitor(context.Background(), m.ID); err == nil {
		t.Error("expected no open incident after recovery, but one is still open")
	}
}

// TestPushScheduledOverdueOpensIncident verifies the scheduled overdue path
// (push monitor with no heartbeat → checker returns down → hook opens incident).
func TestPushScheduledOverdueOpensIncident(t *testing.T) {
	pool := newTestPool(t)
	defer pool.Close()
	requireSchema(t, pool)

	q := db.New(pool)
	m := newPushIncidentMonitor(t, q)

	incidentSvc := NewIncidentService(q, nil)
	hook := NewIncidentHook(incidentSvc)

	// Reload monitor — it has no last_heartbeat_at, status=up.
	m, _ = q.GetMonitor(context.Background(), m.ID)
	if m.LastHeartbeatAt.Valid {
		t.Fatal("expected no heartbeat at start")
	}

	// The scheduler would invoke PushChecker, get "down", call hook with up→down.
	hook.OnScheduledStatusChange(context.Background(), m, "up", "down")

	open, err := q.GetOpenIncidentForMonitor(context.Background(), m.ID)
	if err != nil {
		t.Fatalf("expected incident from scheduled overdue, got: %v", err)
	}
	t.Cleanup(func() {
		_, _ = pool.Exec(context.Background(), `DELETE FROM incidents WHERE id = $1`, open.ID)
	})
}
