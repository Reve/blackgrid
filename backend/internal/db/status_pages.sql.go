// Hand-written queries for status pages and their attached monitors.
// Mirrors sqlc style; regenerate from sql/query.sql when sqlc is run.

package db

import (
	"context"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
)

const statusPageColumns = `id, name, slug, description, public, show_uptime, show_incidents, created_at, updated_at`

func scanStatusPage(row pgx.Row, p *StatusPage) error {
	return row.Scan(
		&p.ID,
		&p.Name,
		&p.Slug,
		&p.Description,
		&p.Public,
		&p.ShowUptime,
		&p.ShowIncidents,
		&p.CreatedAt,
		&p.UpdatedAt,
	)
}

const createStatusPage = `INSERT INTO status_pages (name, slug, description, public, show_uptime, show_incidents)
VALUES ($1, $2, $3, $4, $5, $6)
RETURNING ` + statusPageColumns

type CreateStatusPageParams struct {
	Name          string      `json:"name"`
	Slug          string      `json:"slug"`
	Description   pgtype.Text `json:"description"`
	Public        bool        `json:"public"`
	ShowUptime    bool        `json:"show_uptime"`
	ShowIncidents bool        `json:"show_incidents"`
}

func (q *Queries) CreateStatusPage(ctx context.Context, arg CreateStatusPageParams) (StatusPage, error) {
	row := q.db.QueryRow(ctx, createStatusPage, arg.Name, arg.Slug, arg.Description, arg.Public, arg.ShowUptime, arg.ShowIncidents)
	var p StatusPage
	err := scanStatusPage(row, &p)
	return p, err
}

const getStatusPage = `SELECT ` + statusPageColumns + ` FROM status_pages WHERE id = $1 LIMIT 1`

func (q *Queries) GetStatusPage(ctx context.Context, id pgtype.UUID) (StatusPage, error) {
	row := q.db.QueryRow(ctx, getStatusPage, id)
	var p StatusPage
	err := scanStatusPage(row, &p)
	return p, err
}

const getStatusPageBySlug = `SELECT ` + statusPageColumns + ` FROM status_pages WHERE slug = $1 LIMIT 1`

func (q *Queries) GetStatusPageBySlug(ctx context.Context, slug string) (StatusPage, error) {
	row := q.db.QueryRow(ctx, getStatusPageBySlug, slug)
	var p StatusPage
	err := scanStatusPage(row, &p)
	return p, err
}

const listStatusPages = `SELECT ` + statusPageColumns + ` FROM status_pages ORDER BY created_at DESC`

func (q *Queries) ListStatusPages(ctx context.Context) ([]StatusPage, error) {
	rows, err := q.db.Query(ctx, listStatusPages)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var items []StatusPage
	for rows.Next() {
		var p StatusPage
		if err := scanStatusPage(rows, &p); err != nil {
			return nil, err
		}
		items = append(items, p)
	}
	return items, rows.Err()
}

const updateStatusPage = `UPDATE status_pages
SET name = $2,
    slug = $3,
    description = $4,
    public = $5,
    show_uptime = $6,
    show_incidents = $7,
    updated_at = CURRENT_TIMESTAMP
WHERE id = $1
RETURNING ` + statusPageColumns

type UpdateStatusPageParams struct {
	ID            pgtype.UUID `json:"id"`
	Name          string      `json:"name"`
	Slug          string      `json:"slug"`
	Description   pgtype.Text `json:"description"`
	Public        bool        `json:"public"`
	ShowUptime    bool        `json:"show_uptime"`
	ShowIncidents bool        `json:"show_incidents"`
}

func (q *Queries) UpdateStatusPage(ctx context.Context, arg UpdateStatusPageParams) (StatusPage, error) {
	row := q.db.QueryRow(ctx, updateStatusPage,
		arg.ID, arg.Name, arg.Slug, arg.Description,
		arg.Public, arg.ShowUptime, arg.ShowIncidents,
	)
	var p StatusPage
	err := scanStatusPage(row, &p)
	return p, err
}

const deleteStatusPage = `DELETE FROM status_pages WHERE id = $1`

func (q *Queries) DeleteStatusPage(ctx context.Context, id pgtype.UUID) error {
	_, err := q.db.Exec(ctx, deleteStatusPage, id)
	return err
}

const statusPageMonitorColumns = `status_page_id, monitor_id, display_name, display_order, created_at`

func scanStatusPageMonitor(row pgx.Row, m *StatusPageMonitor) error {
	return row.Scan(&m.StatusPageID, &m.MonitorID, &m.DisplayName, &m.DisplayOrder, &m.CreatedAt)
}

const attachStatusPageMonitor = `INSERT INTO status_page_monitors (status_page_id, monitor_id, display_name, display_order)
VALUES ($1, $2, $3, $4)
RETURNING ` + statusPageMonitorColumns

type AttachStatusPageMonitorParams struct {
	StatusPageID pgtype.UUID `json:"status_page_id"`
	MonitorID    pgtype.UUID `json:"monitor_id"`
	DisplayName  pgtype.Text `json:"display_name"`
	DisplayOrder int32       `json:"display_order"`
}

func (q *Queries) AttachStatusPageMonitor(ctx context.Context, arg AttachStatusPageMonitorParams) (StatusPageMonitor, error) {
	row := q.db.QueryRow(ctx, attachStatusPageMonitor,
		arg.StatusPageID, arg.MonitorID, arg.DisplayName, arg.DisplayOrder,
	)
	var m StatusPageMonitor
	err := scanStatusPageMonitor(row, &m)
	return m, err
}

const updateStatusPageMonitor = `UPDATE status_page_monitors
SET display_name = $3,
    display_order = $4
WHERE status_page_id = $1 AND monitor_id = $2
RETURNING ` + statusPageMonitorColumns

type UpdateStatusPageMonitorParams struct {
	StatusPageID pgtype.UUID `json:"status_page_id"`
	MonitorID    pgtype.UUID `json:"monitor_id"`
	DisplayName  pgtype.Text `json:"display_name"`
	DisplayOrder int32       `json:"display_order"`
}

func (q *Queries) UpdateStatusPageMonitor(ctx context.Context, arg UpdateStatusPageMonitorParams) (StatusPageMonitor, error) {
	row := q.db.QueryRow(ctx, updateStatusPageMonitor,
		arg.StatusPageID, arg.MonitorID, arg.DisplayName, arg.DisplayOrder,
	)
	var m StatusPageMonitor
	err := scanStatusPageMonitor(row, &m)
	return m, err
}

const removeStatusPageMonitor = `DELETE FROM status_page_monitors
WHERE status_page_id = $1 AND monitor_id = $2`

func (q *Queries) RemoveStatusPageMonitor(ctx context.Context, statusPageID, monitorID pgtype.UUID) error {
	_, err := q.db.Exec(ctx, removeStatusPageMonitor, statusPageID, monitorID)
	return err
}

const getStatusPageMonitor = `SELECT ` + statusPageMonitorColumns + `
FROM status_page_monitors
WHERE status_page_id = $1 AND monitor_id = $2
LIMIT 1`

func (q *Queries) GetStatusPageMonitor(ctx context.Context, statusPageID, monitorID pgtype.UUID) (StatusPageMonitor, error) {
	row := q.db.QueryRow(ctx, getStatusPageMonitor, statusPageID, monitorID)
	var m StatusPageMonitor
	err := scanStatusPageMonitor(row, &m)
	return m, err
}

const listStatusPageMonitors = `SELECT ` + statusPageMonitorColumns + `
FROM status_page_monitors
WHERE status_page_id = $1
ORDER BY display_order ASC, created_at ASC`

func (q *Queries) ListStatusPageMonitors(ctx context.Context, statusPageID pgtype.UUID) ([]StatusPageMonitor, error) {
	rows, err := q.db.Query(ctx, listStatusPageMonitors, statusPageID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var items []StatusPageMonitor
	for rows.Next() {
		var m StatusPageMonitor
		if err := scanStatusPageMonitor(rows, &m); err != nil {
			return nil, err
		}
		items = append(items, m)
	}
	return items, rows.Err()
}

const maxStatusPageMonitorOrder = `SELECT COALESCE(MAX(display_order), 0) FROM status_page_monitors WHERE status_page_id = $1`

func (q *Queries) MaxStatusPageMonitorOrder(ctx context.Context, statusPageID pgtype.UUID) (int32, error) {
	row := q.db.QueryRow(ctx, maxStatusPageMonitorOrder, statusPageID)
	var v int32
	err := row.Scan(&v)
	return v, err
}

// MonitorUptimeWindow returns up_count and total_count for monitor_results in
// [now()-windowSeconds, now()].
const monitorUptimeWindow = `SELECT
    COUNT(*) FILTER (WHERE status = 'up') AS up_count,
    COUNT(*) AS total_count
FROM monitor_results
WHERE monitor_id = $1
  AND checked_at >= NOW() - ($2::int || ' seconds')::interval`

type UptimeCounts struct {
	UpCount    int64
	TotalCount int64
}

func (q *Queries) MonitorUptimeWindow(ctx context.Context, monitorID pgtype.UUID, windowSeconds int64) (UptimeCounts, error) {
	row := q.db.QueryRow(ctx, monitorUptimeWindow, monitorID, windowSeconds)
	var c UptimeCounts
	err := row.Scan(&c.UpCount, &c.TotalCount)
	return c, err
}

// ListIncidentsForMonitorsSince returns recent incidents for a set of monitors,
// since the given timestamp. Open, acknowledged, and resolved are all included.
const listIncidentsForMonitorsSince = `SELECT ` + incidentColumns + `
FROM incidents
WHERE monitor_id = ANY($1::uuid[])
  AND (started_at >= $2 OR (status IN ('open', 'acknowledged')))
ORDER BY started_at DESC
LIMIT $3`

func (q *Queries) ListIncidentsForMonitorsSince(ctx context.Context, monitorIDs []pgtype.UUID, since pgtype.Timestamptz, limit int32) ([]Incident, error) {
	rows, err := q.db.Query(ctx, listIncidentsForMonitorsSince, monitorIDs, since, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var items []Incident
	for rows.Next() {
		var i Incident
		if err := scanIncident(rows, &i); err != nil {
			return nil, err
		}
		items = append(items, i)
	}
	return items, rows.Err()
}
