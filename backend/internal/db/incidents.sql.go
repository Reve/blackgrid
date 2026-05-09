// Hand-written queries for incidents.
// Mirrors sqlc style; regenerate from sql/query.sql when sqlc is run.

package db

import (
	"context"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
)

const incidentColumns = `id, monitor_id, status, severity, started_at, acknowledged_at, resolved_at, summary, details, created_at, updated_at`

func scanIncident(row pgx.Row, i *Incident) error {
	return row.Scan(
		&i.ID,
		&i.MonitorID,
		&i.Status,
		&i.Severity,
		&i.StartedAt,
		&i.AcknowledgedAt,
		&i.ResolvedAt,
		&i.Summary,
		&i.Details,
		&i.CreatedAt,
		&i.UpdatedAt,
	)
}

const createIncident = `INSERT INTO incidents (monitor_id, status, severity, summary, details, started_at)
VALUES ($1, 'open', $2, $3, $4, CURRENT_TIMESTAMP)
RETURNING ` + incidentColumns

type CreateIncidentParams struct {
	MonitorID pgtype.UUID `json:"monitor_id"`
	Severity  string      `json:"severity"`
	Summary   string      `json:"summary"`
	Details   pgtype.Text `json:"details"`
}

func (q *Queries) CreateIncident(ctx context.Context, arg CreateIncidentParams) (Incident, error) {
	row := q.db.QueryRow(ctx, createIncident, arg.MonitorID, arg.Severity, arg.Summary, arg.Details)
	var i Incident
	err := scanIncident(row, &i)
	return i, err
}

const getIncident = `SELECT ` + incidentColumns + ` FROM incidents WHERE id = $1 LIMIT 1`

func (q *Queries) GetIncident(ctx context.Context, id pgtype.UUID) (Incident, error) {
	row := q.db.QueryRow(ctx, getIncident, id)
	var i Incident
	err := scanIncident(row, &i)
	return i, err
}

const getOpenIncidentForMonitor = `SELECT ` + incidentColumns + `
FROM incidents
WHERE monitor_id = $1 AND status IN ('open', 'acknowledged')
ORDER BY started_at DESC
LIMIT 1`

func (q *Queries) GetOpenIncidentForMonitor(ctx context.Context, monitorID pgtype.UUID) (Incident, error) {
	row := q.db.QueryRow(ctx, getOpenIncidentForMonitor, monitorID)
	var i Incident
	err := scanIncident(row, &i)
	return i, err
}

const listIncidents = `SELECT ` + incidentColumns + `
FROM incidents
WHERE ($1::text = '' OR status = $1)
  AND ($2::text = '' OR severity = $2)
  AND ($3::uuid IS NULL OR monitor_id = $3)
ORDER BY started_at DESC
LIMIT $4 OFFSET $5`

type ListIncidentsParams struct {
	Status    string      `json:"status"`
	Severity  string      `json:"severity"`
	MonitorID pgtype.UUID `json:"monitor_id"`
	Limit     int32       `json:"limit"`
	Offset    int32       `json:"offset"`
}

func (q *Queries) ListIncidents(ctx context.Context, arg ListIncidentsParams) ([]Incident, error) {
	rows, err := q.db.Query(ctx, listIncidents, arg.Status, arg.Severity, arg.MonitorID, arg.Limit, arg.Offset)
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

const acknowledgeIncident = `UPDATE incidents
SET status = 'acknowledged',
    acknowledged_at = CURRENT_TIMESTAMP,
    updated_at = CURRENT_TIMESTAMP,
    details = COALESCE(NULLIF($2, ''), details)
WHERE id = $1
RETURNING ` + incidentColumns

func (q *Queries) AcknowledgeIncident(ctx context.Context, id pgtype.UUID, note string) (Incident, error) {
	row := q.db.QueryRow(ctx, acknowledgeIncident, id, note)
	var i Incident
	err := scanIncident(row, &i)
	return i, err
}

const resolveIncident = `UPDATE incidents
SET status = 'resolved',
    resolved_at = CURRENT_TIMESTAMP,
    updated_at = CURRENT_TIMESTAMP,
    details = COALESCE(NULLIF($2, ''), details)
WHERE id = $1
RETURNING ` + incidentColumns

func (q *Queries) ResolveIncident(ctx context.Context, id pgtype.UUID, note string) (Incident, error) {
	row := q.db.QueryRow(ctx, resolveIncident, id, note)
	var i Incident
	err := scanIncident(row, &i)
	return i, err
}

const countIncidentsByStatus = `SELECT
    COUNT(*) FILTER (WHERE status = 'open') AS open_count,
    COUNT(*) FILTER (WHERE status = 'acknowledged') AS acknowledged_count,
    COUNT(*) FILTER (WHERE status = 'open' AND severity = 'critical') AS critical_count,
    COUNT(*) FILTER (WHERE status = 'resolved' AND resolved_at >= NOW() - INTERVAL '24 hours') AS resolved_24h_count
FROM incidents`

type IncidentCounts struct {
	OpenCount         int64 `json:"open_count"`
	AcknowledgedCount int64 `json:"acknowledged_count"`
	CriticalCount     int64 `json:"critical_count"`
	Resolved24hCount  int64 `json:"resolved_24h_count"`
}

func (q *Queries) CountIncidentsByStatus(ctx context.Context) (IncidentCounts, error) {
	row := q.db.QueryRow(ctx, countIncidentsByStatus)
	var c IncidentCounts
	err := row.Scan(&c.OpenCount, &c.AcknowledgedCount, &c.CriticalCount, &c.Resolved24hCount)
	return c, err
}
