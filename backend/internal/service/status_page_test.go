package service

import (
	"context"
	"strings"
	"testing"
	"time"

	"blackgrid/internal/db"

	"github.com/jackc/pgx/v5/pgtype"
)

// requireStatusPageSchema skips if migration 004 hasn't been applied.
func requireStatusPageSchema(t *testing.T, q *db.Queries) {
	t.Helper()
	_, err := q.ListStatusPages(context.Background())
	if err != nil {
		t.Skipf("Skipping: status_pages table missing (run migration 004): %v", err)
	}
}

func cleanupStatusPage(t *testing.T, svc *StatusPageService, id pgtype.UUID) {
	t.Helper()
	t.Cleanup(func() {
		_ = svc.DeleteStatusPage(context.Background(), id)
	})
}

func ptrStr(s string) *string { return &s }
func ptrBool(b bool) *bool    { return &b }
func ptrI32(v int32) *int32   { return &v }

func TestStatusPage_CreateGeneratesSlug(t *testing.T) {
	pool := newTestPool(t)
	defer pool.Close()
	q := db.New(pool)
	requireStatusPageSchema(t, q)
	svc := NewStatusPageService(q, nil)

	page, err := svc.CreateStatusPage(context.Background(), StatusPageInput{
		Name: "Homelab Core Services " + strings.ReplaceAll(time.Now().Format("150405.000000"), ".", ""),
	})
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	cleanupStatusPage(t, svc, page.ID)

	if page.Slug == "" {
		t.Fatal("expected slug to be generated")
	}
	if !strings.Contains(page.Slug, "homelab-core-services") {
		t.Fatalf("slug %q should be derived from name", page.Slug)
	}
}

func TestStatusPage_DuplicateSlugRejected(t *testing.T) {
	pool := newTestPool(t)
	defer pool.Close()
	q := db.New(pool)
	requireStatusPageSchema(t, q)
	svc := NewStatusPageService(q, nil)

	slug := "dup-" + strings.ReplaceAll(time.Now().Format("150405.000000"), ".", "")
	p1, err := svc.CreateStatusPage(context.Background(), StatusPageInput{Name: "page1", Slug: slug})
	if err != nil {
		t.Fatalf("create1: %v", err)
	}
	cleanupStatusPage(t, svc, p1.ID)

	_, err = svc.CreateStatusPage(context.Background(), StatusPageInput{Name: "page2", Slug: slug})
	if err != ErrStatusPageDuplicateSlug {
		t.Fatalf("expected ErrStatusPageDuplicateSlug, got %v", err)
	}
}

func TestStatusPage_InvalidSlugRejected(t *testing.T) {
	pool := newTestPool(t)
	defer pool.Close()
	q := db.New(pool)
	requireStatusPageSchema(t, q)
	svc := NewStatusPageService(q, nil)

	_, err := svc.CreateStatusPage(context.Background(), StatusPageInput{Name: "x", Slug: "Invalid Slug!"})
	if err != ErrStatusPageInvalidSlug {
		t.Fatalf("expected ErrStatusPageInvalidSlug, got %v", err)
	}
}

func TestStatusPage_UpdateAndDelete(t *testing.T) {
	pool := newTestPool(t)
	defer pool.Close()
	q := db.New(pool)
	requireStatusPageSchema(t, q)
	svc := NewStatusPageService(q, nil)

	page, err := svc.CreateStatusPage(context.Background(), StatusPageInput{
		Name: "page-" + strings.ReplaceAll(time.Now().Format("150405.000000"), ".", ""),
	})
	if err != nil {
		t.Fatalf("create: %v", err)
	}

	updated, err := svc.UpdateStatusPage(context.Background(), page.ID, StatusPageInput{
		Name: "Renamed", Public: ptrBool(true),
	})
	if err != nil {
		t.Fatalf("update: %v", err)
	}
	if updated.Name != "Renamed" || !updated.Public {
		t.Fatalf("update did not apply: %+v", updated)
	}

	if err := svc.DeleteStatusPage(context.Background(), page.ID); err != nil {
		t.Fatalf("delete: %v", err)
	}
	if _, err := svc.GetAdminStatusPage(context.Background(), page.ID); err != ErrStatusPageNotFound {
		t.Fatalf("expected not found after delete, got %v", err)
	}
}

func TestStatusPage_DeleteCascadesMonitorLinks(t *testing.T) {
	pool := newTestPool(t)
	defer pool.Close()
	q := db.New(pool)
	requireStatusPageSchema(t, q)
	requireSchema(t, pool)
	svc := NewStatusPageService(q, nil)

	m := createTestMonitor(t, q)
	page, err := svc.CreateStatusPage(context.Background(), StatusPageInput{Name: "p-" + strings.ReplaceAll(time.Now().Format("150405.000000"), ".", "")})
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	if _, err := svc.AttachMonitor(context.Background(), page.ID, AttachMonitorInput{MonitorID: m.ID}); err != nil {
		t.Fatalf("attach: %v", err)
	}
	if err := svc.DeleteStatusPage(context.Background(), page.ID); err != nil {
		t.Fatalf("delete: %v", err)
	}
	links, err := q.ListStatusPageMonitors(context.Background(), page.ID)
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(links) != 0 {
		t.Fatalf("expected cascade to remove links, got %d", len(links))
	}
}

func TestStatusPage_AttachAndDuplicate(t *testing.T) {
	pool := newTestPool(t)
	defer pool.Close()
	q := db.New(pool)
	requireStatusPageSchema(t, q)
	requireSchema(t, pool)
	svc := NewStatusPageService(q, nil)

	page, err := svc.CreateStatusPage(context.Background(), StatusPageInput{Name: "p-" + strings.ReplaceAll(time.Now().Format("150405.000000"), ".", "")})
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	cleanupStatusPage(t, svc, page.ID)

	m := createTestMonitor(t, q)
	if _, err := svc.AttachMonitor(context.Background(), page.ID, AttachMonitorInput{MonitorID: m.ID}); err != nil {
		t.Fatalf("attach: %v", err)
	}
	if _, err := svc.AttachMonitor(context.Background(), page.ID, AttachMonitorInput{MonitorID: m.ID}); err != ErrMonitorAlreadyAttached {
		t.Fatalf("expected ErrMonitorAlreadyAttached, got %v", err)
	}

	if err := svc.RemoveMonitor(context.Background(), page.ID, m.ID); err != nil {
		t.Fatalf("remove: %v", err)
	}
	if err := svc.RemoveMonitor(context.Background(), page.ID, m.ID); err != ErrMonitorNotAttached {
		t.Fatalf("expected ErrMonitorNotAttached, got %v", err)
	}
}

func TestStatusPage_ReorderRejectsUnattached(t *testing.T) {
	pool := newTestPool(t)
	defer pool.Close()
	q := db.New(pool)
	requireStatusPageSchema(t, q)
	requireSchema(t, pool)
	svc := NewStatusPageService(q, nil)

	page, _ := svc.CreateStatusPage(context.Background(), StatusPageInput{Name: "p-" + strings.ReplaceAll(time.Now().Format("150405.000000"), ".", "")})
	cleanupStatusPage(t, svc, page.ID)

	m1 := createTestMonitor(t, q)
	m2 := createTestMonitor(t, q) // not attached
	svc.AttachMonitor(context.Background(), page.ID, AttachMonitorInput{MonitorID: m1.ID})

	err := svc.ReorderMonitors(context.Background(), page.ID, []pgtype.UUID{m1.ID, m2.ID})
	if err != ErrReorderMonitorMismatched {
		t.Fatalf("expected ErrReorderMonitorMismatched, got %v", err)
	}
}

func TestStatusPage_ReorderUpdatesOrder(t *testing.T) {
	pool := newTestPool(t)
	defer pool.Close()
	q := db.New(pool)
	requireStatusPageSchema(t, q)
	requireSchema(t, pool)
	svc := NewStatusPageService(q, nil)

	page, _ := svc.CreateStatusPage(context.Background(), StatusPageInput{Name: "p-" + strings.ReplaceAll(time.Now().Format("150405.000000"), ".", "")})
	cleanupStatusPage(t, svc, page.ID)

	m1 := createTestMonitor(t, q)
	m2 := createTestMonitor(t, q)
	m3 := createTestMonitor(t, q)
	svc.AttachMonitor(context.Background(), page.ID, AttachMonitorInput{MonitorID: m1.ID})
	svc.AttachMonitor(context.Background(), page.ID, AttachMonitorInput{MonitorID: m2.ID})
	svc.AttachMonitor(context.Background(), page.ID, AttachMonitorInput{MonitorID: m3.ID})

	if err := svc.ReorderMonitors(context.Background(), page.ID, []pgtype.UUID{m3.ID, m1.ID, m2.ID}); err != nil {
		t.Fatalf("reorder: %v", err)
	}
	links, _ := q.ListStatusPageMonitors(context.Background(), page.ID)
	if len(links) != 3 {
		t.Fatalf("links: %d", len(links))
	}
	expectedOrder := []pgtype.UUID{m3.ID, m1.ID, m2.ID}
	for i, l := range links {
		if l.MonitorID != expectedOrder[i] {
			t.Fatalf("position %d: got %v want %v", i, l.MonitorID, expectedOrder[i])
		}
		if l.DisplayOrder != int32((i+1)*10) {
			t.Fatalf("position %d display_order: got %d want %d", i, l.DisplayOrder, (i+1)*10)
		}
	}
}

func TestComputeAggregateStatus(t *testing.T) {
	cases := []struct {
		name   string
		input  []string
		expect string
	}{
		{"empty", []string{}, "empty"},
		{"all up", []string{"up", "up", "up"}, "up"},
		{"any down", []string{"up", "down", "up"}, "down"},
		{"degraded", []string{"up", "degraded"}, "degraded"},
		{"unknown counts as degraded", []string{"up", "unknown"}, "degraded"},
		{"down beats degraded", []string{"down", "degraded"}, "down"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := ComputeAggregateStatus(tc.input); got != tc.expect {
				t.Fatalf("got %q want %q", got, tc.expect)
			}
		})
	}
}

func TestStatusPage_PrivateNotExposedPublicly(t *testing.T) {
	pool := newTestPool(t)
	defer pool.Close()
	q := db.New(pool)
	requireStatusPageSchema(t, q)
	svc := NewStatusPageService(q, nil)

	slug := "priv-" + strings.ReplaceAll(time.Now().Format("150405.000000"), ".", "")
	page, err := svc.CreateStatusPage(context.Background(), StatusPageInput{
		Name: "Private", Slug: slug, Public: ptrBool(false),
	})
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	cleanupStatusPage(t, svc, page.ID)

	if _, err := svc.GetPublicStatusPage(context.Background(), slug); err != ErrStatusPageNotFound {
		t.Fatalf("expected ErrStatusPageNotFound for private page, got %v", err)
	}
}

func TestStatusPage_PublicReturnsAttachedInOrder(t *testing.T) {
	pool := newTestPool(t)
	defer pool.Close()
	q := db.New(pool)
	requireStatusPageSchema(t, q)
	requireSchema(t, pool)
	svc := NewStatusPageService(q, nil)

	slug := "pub-" + strings.ReplaceAll(time.Now().Format("150405.000000"), ".", "")
	page, _ := svc.CreateStatusPage(context.Background(), StatusPageInput{
		Name: "Public", Slug: slug, Public: ptrBool(true), ShowIncidents: ptrBool(false),
	})
	cleanupStatusPage(t, svc, page.ID)

	m1 := createTestMonitor(t, q)
	m2 := createTestMonitor(t, q)
	svc.AttachMonitor(context.Background(), page.ID, AttachMonitorInput{MonitorID: m1.ID, DisplayOrder: ptrI32(20), DisplayName: ptrStr("Service B")})
	svc.AttachMonitor(context.Background(), page.ID, AttachMonitorInput{MonitorID: m2.ID, DisplayOrder: ptrI32(10), DisplayName: ptrStr("Service A")})

	resp, err := svc.GetPublicStatusPage(context.Background(), slug)
	if err != nil {
		t.Fatalf("get public: %v", err)
	}
	if len(resp.Monitors) != 2 {
		t.Fatalf("expected 2 monitors, got %d", len(resp.Monitors))
	}
	if resp.Monitors[0].DisplayName != "Service A" || resp.Monitors[1].DisplayName != "Service B" {
		t.Fatalf("wrong order: %+v", resp.Monitors)
	}
	if resp.Incidents != nil {
		t.Fatalf("incidents should be nil when show_incidents=false")
	}
}

func TestStatusPage_UptimeNoResultsReturnsNil(t *testing.T) {
	pool := newTestPool(t)
	defer pool.Close()
	q := db.New(pool)
	requireStatusPageSchema(t, q)
	requireSchema(t, pool)
	svc := NewStatusPageService(q, nil)

	m := createTestMonitor(t, q)
	u, err := svc.ComputeMonitorUptime(context.Background(), m.ID, 24*60*60)
	if err != nil {
		t.Fatalf("uptime: %v", err)
	}
	if u != nil {
		t.Fatalf("expected nil uptime, got %v", *u)
	}
}

func TestStatusPage_UptimePercentage(t *testing.T) {
	pool := newTestPool(t)
	defer pool.Close()
	q := db.New(pool)
	requireStatusPageSchema(t, q)
	requireSchema(t, pool)
	svc := NewStatusPageService(q, nil)

	m := createTestMonitor(t, q)
	ctx := context.Background()
	t.Cleanup(func() { pool.Exec(ctx, "DELETE FROM monitor_results WHERE monitor_id=$1", m.ID) })

	for i := 0; i < 8; i++ {
		_, err := pool.Exec(ctx,
			`INSERT INTO monitor_results (monitor_id, status, latency_ms, checked_at) VALUES ($1, 'up', 1, NOW())`, m.ID)
		if err != nil {
			t.Fatalf("insert: %v", err)
		}
	}
	for i := 0; i < 2; i++ {
		_, err := pool.Exec(ctx,
			`INSERT INTO monitor_results (monitor_id, status, latency_ms, checked_at) VALUES ($1, 'down', 0, NOW())`, m.ID)
		if err != nil {
			t.Fatalf("insert: %v", err)
		}
	}

	u, err := svc.ComputeMonitorUptime(ctx, m.ID, 24*60*60)
	if err != nil {
		t.Fatalf("uptime: %v", err)
	}
	if u == nil {
		t.Fatal("expected uptime, got nil")
	}
	if *u < 79.9 || *u > 80.1 {
		t.Fatalf("expected ~80%%, got %v", *u)
	}
}

func TestStatusPage_PublicIncidentsForAttachedOnly(t *testing.T) {
	pool := newTestPool(t)
	defer pool.Close()
	q := db.New(pool)
	requireStatusPageSchema(t, q)
	requireSchema(t, pool)
	svc := NewStatusPageService(q, nil)
	incSvc := NewIncidentService(q, nil)

	slug := "incs-" + strings.ReplaceAll(time.Now().Format("150405.000000"), ".", "")
	page, _ := svc.CreateStatusPage(context.Background(), StatusPageInput{
		Name: "P", Slug: slug, Public: ptrBool(true),
	})
	cleanupStatusPage(t, svc, page.ID)

	mAttached := createTestMonitor(t, q)
	mUnattached := createTestMonitor(t, q)
	t.Cleanup(func() {
		pool.Exec(context.Background(), "DELETE FROM incidents WHERE monitor_id IN ($1,$2)", mAttached.ID, mUnattached.ID)
	})

	svc.AttachMonitor(context.Background(), page.ID, AttachMonitorInput{MonitorID: mAttached.ID, DisplayName: ptrStr("Attached")})

	if _, _, err := incSvc.OpenForMonitor(context.Background(), mAttached, "critical", "down attached", ""); err != nil {
		t.Fatalf("open: %v", err)
	}
	if _, _, err := incSvc.OpenForMonitor(context.Background(), mUnattached, "critical", "down unattached", ""); err != nil {
		t.Fatalf("open: %v", err)
	}

	resp, err := svc.GetPublicStatusPage(context.Background(), slug)
	if err != nil {
		t.Fatalf("get public: %v", err)
	}
	if len(resp.Incidents) == 0 {
		t.Fatal("expected incidents")
	}
	for _, i := range resp.Incidents {
		if strings.Contains(i.Summary, "unattached") {
			t.Fatalf("public response leaked unattached incident: %+v", i)
		}
	}
}
