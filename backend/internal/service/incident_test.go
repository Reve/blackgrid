package service

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"blackgrid/internal/db"

	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
)

func newTestPool(t *testing.T) *pgxpool.Pool {
	t.Helper()
	url := os.Getenv("BLACKGRID_TEST_DATABASE_URL")
	if url == "" {
		url = "postgres://blackgrid:blackgrid@localhost:5432/blackgrid?sslmode=disable"
	}
	pool, err := pgxpool.New(context.Background(), url)
	if err != nil {
		t.Skip("Skipping integration test: cannot connect to db")
	}
	if err := pool.Ping(context.Background()); err != nil {
		pool.Close()
		t.Skip("Skipping integration test: db unreachable")
	}
	return pool
}

// requireSchema skips the test if Phase 3 columns aren't present.
func requireSchema(t *testing.T, pool *pgxpool.Pool) {
	t.Helper()
	var exists bool
	err := pool.QueryRow(context.Background(),
		`SELECT EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name='incidents' AND column_name='severity')`).
		Scan(&exists)
	if err != nil || !exists {
		t.Skip("Skipping: incidents table not migrated to Phase 3 schema (run migration 003)")
	}
}

// createTestMonitor inserts a throwaway monitor and returns it. Cleaned up via t.Cleanup.
func createTestMonitor(t *testing.T, q *db.Queries) db.Monitor {
	t.Helper()
	ctx := context.Background()
	suffix := strings.ReplaceAll(time.Now().Format("150405.000000"), ".", "")
	m, err := q.CreateMonitor(ctx, db.CreateMonitorParams{
		Name:            "test-mon-" + suffix,
		Slug:            "test-mon-" + suffix,
		MonitorType:     "tcp",
		Target:          "127.0.0.1:1",
		Config:          []byte("{}"),
		IntervalSeconds: 60,
		TimeoutSeconds:  5,
		RetryCount:      1,
		Enabled:         true,
		Status:          "unknown",
	})
	if err != nil {
		t.Fatalf("create monitor: %v", err)
	}
	t.Cleanup(func() { _ = q.DeleteMonitor(context.Background(), m.ID) })
	return m
}

func TestIncidentLifecycle_OpensAndDoesNotDuplicate(t *testing.T) {
	pool := newTestPool(t)
	defer pool.Close()
	requireSchema(t, pool)

	q := db.New(pool)
	svc := NewIncidentService(q, nil)
	m := createTestMonitor(t, q)

	inc1, created1, err := svc.OpenForMonitor(context.Background(), m, "critical", "down", "")
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	if !created1 {
		t.Fatal("expected first OpenForMonitor to create")
	}
	t.Cleanup(func() {
		pool.Exec(context.Background(), "DELETE FROM incidents WHERE monitor_id=$1", m.ID)
	})

	_, created2, err := svc.OpenForMonitor(context.Background(), m, "critical", "down again", "")
	if err != nil {
		t.Fatalf("open2: %v", err)
	}
	if created2 {
		t.Fatal("expected second OpenForMonitor to be a no-op (no duplicate)")
	}

	if inc1.Status != "open" {
		t.Errorf("first incident status %q want open", inc1.Status)
	}
}

func TestIncidentLifecycle_ResolveOnRecovery(t *testing.T) {
	pool := newTestPool(t)
	defer pool.Close()
	requireSchema(t, pool)

	q := db.New(pool)
	svc := NewIncidentService(q, nil)
	m := createTestMonitor(t, q)
	t.Cleanup(func() { pool.Exec(context.Background(), "DELETE FROM incidents WHERE monitor_id=$1", m.ID) })

	if _, _, err := svc.OpenForMonitor(context.Background(), m, "critical", "down", ""); err != nil {
		t.Fatalf("open: %v", err)
	}
	resolved, didResolve, err := svc.ResolveForMonitor(context.Background(), m, "recovered")
	if err != nil {
		t.Fatalf("resolve: %v", err)
	}
	if !didResolve || resolved.Status != "resolved" {
		t.Fatalf("expected resolution; got %v %+v", didResolve, resolved)
	}

	// Resolving again is no-op.
	_, didResolve2, _ := svc.ResolveForMonitor(context.Background(), m, "")
	if didResolve2 {
		t.Fatal("second ResolveForMonitor should be no-op")
	}
}

func TestIncidentLifecycle_ResolvesAcknowledged(t *testing.T) {
	pool := newTestPool(t)
	defer pool.Close()
	requireSchema(t, pool)

	q := db.New(pool)
	svc := NewIncidentService(q, nil)
	m := createTestMonitor(t, q)
	t.Cleanup(func() { pool.Exec(context.Background(), "DELETE FROM incidents WHERE monitor_id=$1", m.ID) })

	inc, _, _ := svc.OpenForMonitor(context.Background(), m, "critical", "down", "")
	acked, err := svc.Acknowledge(context.Background(), inc.ID, "looking into it")
	if err != nil {
		t.Fatalf("ack: %v", err)
	}
	if acked.Status != "acknowledged" || !acked.AcknowledgedAt.Valid {
		t.Fatalf("ack did not change status: %+v", acked)
	}

	resolved, _, err := svc.ResolveForMonitor(context.Background(), m, "")
	if err != nil {
		t.Fatalf("resolve: %v", err)
	}
	if resolved.Status != "resolved" {
		t.Fatalf("acknowledged incident should resolve on recovery: %v", resolved.Status)
	}
}

func TestIncidentLifecycle_NewIncidentAfterRecovery(t *testing.T) {
	pool := newTestPool(t)
	defer pool.Close()
	requireSchema(t, pool)

	q := db.New(pool)
	svc := NewIncidentService(q, nil)
	m := createTestMonitor(t, q)
	t.Cleanup(func() { pool.Exec(context.Background(), "DELETE FROM incidents WHERE monitor_id=$1", m.ID) })

	inc1, _, _ := svc.OpenForMonitor(context.Background(), m, "critical", "down", "")
	svc.ResolveForMonitor(context.Background(), m, "")
	inc2, created, _ := svc.OpenForMonitor(context.Background(), m, "critical", "down again", "")
	if !created {
		t.Fatal("expected new incident after previous resolved")
	}
	if inc1.ID == inc2.ID {
		t.Fatal("expected new incident id, got same as previous")
	}
}

func TestAcknowledgeResolvedReturnsError(t *testing.T) {
	pool := newTestPool(t)
	defer pool.Close()
	requireSchema(t, pool)

	q := db.New(pool)
	svc := NewIncidentService(q, nil)
	m := createTestMonitor(t, q)
	t.Cleanup(func() { pool.Exec(context.Background(), "DELETE FROM incidents WHERE monitor_id=$1", m.ID) })

	inc, _, _ := svc.OpenForMonitor(context.Background(), m, "critical", "down", "")
	svc.Resolve(context.Background(), inc.ID, "")
	_, err := svc.Acknowledge(context.Background(), inc.ID, "")
	if err != ErrIncidentAlreadyResolved {
		t.Fatalf("expected ErrIncidentAlreadyResolved, got %v", err)
	}
}

func TestResolveAlreadyResolvedIsIdempotent(t *testing.T) {
	pool := newTestPool(t)
	defer pool.Close()
	requireSchema(t, pool)

	q := db.New(pool)
	svc := NewIncidentService(q, nil)
	m := createTestMonitor(t, q)
	t.Cleanup(func() { pool.Exec(context.Background(), "DELETE FROM incidents WHERE monitor_id=$1", m.ID) })

	inc, _, _ := svc.OpenForMonitor(context.Background(), m, "critical", "down", "")
	if _, err := svc.Resolve(context.Background(), inc.ID, ""); err != nil {
		t.Fatalf("first resolve: %v", err)
	}
	again, err := svc.Resolve(context.Background(), inc.ID, "")
	if err != nil {
		t.Fatalf("second resolve should be idempotent, got: %v", err)
	}
	if again.Status != "resolved" {
		t.Fatalf("expected resolved status, got %s", again.Status)
	}
}

func TestNotifierIsCalledOnOpen(t *testing.T) {
	pool := newTestPool(t)
	defer pool.Close()
	requireSchema(t, pool)

	q := db.New(pool)
	svc := NewIncidentService(q, nil)
	m := createTestMonitor(t, q)
	t.Cleanup(func() { pool.Exec(context.Background(), "DELETE FROM incidents WHERE monitor_id=$1", m.ID) })

	mock := &mockNotifier{}
	svc.SetNotifier(mock)
	if _, _, err := svc.OpenForMonitor(context.Background(), m, "critical", "down", ""); err != nil {
		t.Fatalf("open: %v", err)
	}

	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		if atomic.LoadInt32(&mock.opened) > 0 {
			break
		}
		time.Sleep(10 * time.Millisecond)
	}
	if atomic.LoadInt32(&mock.opened) != 1 {
		t.Fatalf("expected SendIncidentOpened called once, got %d", mock.opened)
	}
}

// TestNotifierIsNotCalledOnRepeatedOpen ensures that calling OpenForMonitor
// repeatedly while an incident is already open does not produce additional
// notification deliveries. This guards against a regression where every
// failed monitor check would page the on-call channel until the incident
// resolved.
func TestNotifierIsNotCalledOnRepeatedOpen(t *testing.T) {
	pool := newTestPool(t)
	defer pool.Close()
	requireSchema(t, pool)

	q := db.New(pool)
	svc := NewIncidentService(q, nil)
	m := createTestMonitor(t, q)
	t.Cleanup(func() { pool.Exec(context.Background(), "DELETE FROM incidents WHERE monitor_id=$1", m.ID) })

	mock := &mockNotifier{}
	svc.SetNotifier(mock)

	for i := 0; i < 5; i++ {
		if _, _, err := svc.OpenForMonitor(context.Background(), m, "critical", "down", ""); err != nil {
			t.Fatalf("open[%d]: %v", i, err)
		}
	}

	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		if atomic.LoadInt32(&mock.opened) > 0 {
			break
		}
		time.Sleep(10 * time.Millisecond)
	}
	// allow late goroutine deliveries to settle
	time.Sleep(100 * time.Millisecond)
	if got := atomic.LoadInt32(&mock.opened); got != 1 {
		t.Fatalf("expected SendIncidentOpened called exactly once across 5 OpenForMonitor calls, got %d", got)
	}
}

// TestNotifierFiresOnceOnOpenAndOnceOnResolve verifies the contract that
// notifications happen on incident opened and resolved transitions only —
// not on intermediate status changes or repeated calls.
func TestNotifierFiresOnceOnOpenAndOnceOnResolve(t *testing.T) {
	pool := newTestPool(t)
	defer pool.Close()
	requireSchema(t, pool)

	q := db.New(pool)
	svc := NewIncidentService(q, nil)
	m := createTestMonitor(t, q)
	t.Cleanup(func() { pool.Exec(context.Background(), "DELETE FROM incidents WHERE monitor_id=$1", m.ID) })

	mock := &mockNotifier{}
	svc.SetNotifier(mock)

	if _, _, err := svc.OpenForMonitor(context.Background(), m, "critical", "down", ""); err != nil {
		t.Fatalf("open: %v", err)
	}
	// Repeat opens (simulating sustained "down" checks) must not re-notify.
	for i := 0; i < 3; i++ {
		if _, _, err := svc.OpenForMonitor(context.Background(), m, "critical", "still down", ""); err != nil {
			t.Fatalf("repeat open[%d]: %v", i, err)
		}
	}
	if _, _, err := svc.ResolveForMonitor(context.Background(), m, "recovered"); err != nil {
		t.Fatalf("resolve: %v", err)
	}
	// Repeat resolves must not re-notify either.
	for i := 0; i < 3; i++ {
		if _, _, err := svc.ResolveForMonitor(context.Background(), m, "still recovered"); err != nil {
			t.Fatalf("repeat resolve[%d]: %v", i, err)
		}
	}

	time.Sleep(200 * time.Millisecond)
	if got := atomic.LoadInt32(&mock.opened); got != 1 {
		t.Fatalf("opened count: want 1, got %d", got)
	}
	if got := atomic.LoadInt32(&mock.resolved); got != 1 {
		t.Fatalf("resolved count: want 1, got %d", got)
	}
}

type mockNotifier struct {
	opened   int32
	resolved int32
}

func (m *mockNotifier) SendIncidentOpened(ctx context.Context, _ db.Incident, _ db.Monitor) {
	atomic.AddInt32(&m.opened, 1)
}
func (m *mockNotifier) SendIncidentResolved(ctx context.Context, _ db.Incident, _ db.Monitor) {
	atomic.AddInt32(&m.resolved, 1)
}

// TestNotificationDelivery_WebhookFailureStored ensures that a failed webhook
// delivery still produces a stored row with status=failed and last_error set.
func TestNotificationDelivery_WebhookFailureStored(t *testing.T) {
	pool := newTestPool(t)
	defer pool.Close()
	requireSchema(t, pool)
	q := db.New(pool)

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer srv.Close()

	svc := NewNotificationService(q, nil)
	svc.SetHTTPClient(srv.Client())

	// Create a webhook channel.
	ch, err := svc.CreateChannel(context.Background(), "wh", ChannelTypeWebhook, true,
		[]byte(`{"url":"`+srv.URL+`"}`))
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	t.Cleanup(func() { _ = svc.DeleteChannel(context.Background(), ch.ID) })

	delivery, err := svc.TestChannel(context.Background(), ch.ID)
	if err != nil {
		t.Fatalf("test channel: %v", err)
	}
	if delivery.Status != "failed" {
		t.Fatalf("expected failed status, got %s", delivery.Status)
	}
	if !delivery.LastError.Valid || delivery.LastError.String == "" {
		t.Fatal("expected last_error to be populated")
	}
}

// TestNotificationDelivery_OneFailureDoesNotBlockOthers verifies that when one
// channel fails, deliveries are still written for other channels.
func TestNotificationDelivery_OneFailureDoesNotBlockOthers(t *testing.T) {
	pool := newTestPool(t)
	defer pool.Close()
	requireSchema(t, pool)
	q := db.New(pool)

	failingSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer failingSrv.Close()

	var wg sync.WaitGroup
	wg.Add(1)
	successSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		wg.Done()
		w.WriteHeader(http.StatusOK)
	}))
	defer successSrv.Close()

	svc := NewNotificationService(q, nil)
	svc.SetHTTPClient(http.DefaultClient)

	failCh, _ := svc.CreateChannel(context.Background(), "fail", ChannelTypeWebhook, true,
		[]byte(`{"url":"`+failingSrv.URL+`"}`))
	okCh, _ := svc.CreateChannel(context.Background(), "ok", ChannelTypeWebhook, true,
		[]byte(`{"url":"`+successSrv.URL+`"}`))
	t.Cleanup(func() {
		svc.DeleteChannel(context.Background(), failCh.ID)
		svc.DeleteChannel(context.Background(), okCh.ID)
	})

	monitor := db.Monitor{ID: pgtype.UUID{Valid: true}, Name: "x", MonitorType: "tcp", Target: "h:1", Status: "down"}
	incident := db.Incident{Severity: "critical", Status: "open"}
	svc.SendIncidentOpened(context.Background(), incident, monitor)

	done := make(chan struct{})
	go func() { wg.Wait(); close(done) }()
	select {
	case <-done:
	case <-time.After(3 * time.Second):
		t.Fatal("success webhook never received request — failing channel blocked others")
	}
}
