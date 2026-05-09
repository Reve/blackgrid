// Hand-written queries for discovery/IPAM reconciliation.
// Mirrors sqlc style; regenerate from sql/query.sql when sqlc is run.

package db

import (
	"context"

	"github.com/jackc/pgx/v5/pgtype"
)

const updateIPAddressLastSeen = `-- name: UpdateIPAddressLastSeen :one
UPDATE ip_addresses SET last_seen_at = CURRENT_TIMESTAMP, updated_at = CURRENT_TIMESTAMP WHERE id = $1 RETURNING id, prefix_id, ip_address, interface_id, status, description, created_at, updated_at, last_seen_at
`

func (q *Queries) UpdateIPAddressLastSeen(ctx context.Context, id pgtype.UUID) (IpAddress, error) {
	row := q.db.QueryRow(ctx, updateIPAddressLastSeen, id)
	var i IpAddress
	err := row.Scan(
		&i.ID,
		&i.PrefixID,
		&i.IpAddress,
		&i.InterfaceID,
		&i.Status,
		&i.Description,
		&i.CreatedAt,
		&i.UpdatedAt,
		&i.LastSeenAt,
	)
	return i, err
}

const updateIPAddressStatus = `-- name: UpdateIPAddressStatus :one
UPDATE ip_addresses SET status = $2, updated_at = CURRENT_TIMESTAMP WHERE id = $1 RETURNING id, prefix_id, ip_address, interface_id, status, description, created_at, updated_at, last_seen_at
`

type UpdateIPAddressStatusParams struct {
	ID     pgtype.UUID `json:"id"`
	Status pgtype.Text `json:"status"`
}

const updatePrefixScanConfig = `-- name: UpdatePrefixScanConfig :one
UPDATE prefixes SET scan_enabled = $2, scan_interval_seconds = $3, updated_at = CURRENT_TIMESTAMP WHERE id = $1 RETURNING id, site_id, vlan_id, prefix, description, created_at, updated_at, scan_enabled, scan_interval_seconds
`

type UpdatePrefixScanConfigParams struct {
	ID                  pgtype.UUID `json:"id"`
	ScanEnabled         bool        `json:"scan_enabled"`
	ScanIntervalSeconds int32       `json:"scan_interval_seconds"`
}

func (q *Queries) UpdatePrefixScanConfig(ctx context.Context, arg UpdatePrefixScanConfigParams) (Prefix, error) {
	row := q.db.QueryRow(ctx, updatePrefixScanConfig, arg.ID, arg.ScanEnabled, arg.ScanIntervalSeconds)
	var i Prefix
	err := row.Scan(
		&i.ID,
		&i.SiteID,
		&i.VlanID,
		&i.Prefix,
		&i.Description,
		&i.CreatedAt,
		&i.UpdatedAt,
		&i.ScanEnabled,
		&i.ScanIntervalSeconds,
	)
	return i, err
}

const getLatestScanForPrefix = `-- name: GetLatestScanForPrefix :one
SELECT id, prefix_id, status, started_at, completed_at, error, created_at, updated_at FROM discovery_scans WHERE prefix_id = $1 ORDER BY created_at DESC LIMIT 1
`

func (q *Queries) GetLatestScanForPrefix(ctx context.Context, prefixID pgtype.UUID) (DiscoveryScan, error) {
	row := q.db.QueryRow(ctx, getLatestScanForPrefix, prefixID)
	var i DiscoveryScan
	err := row.Scan(
		&i.ID,
		&i.PrefixID,
		&i.Status,
		&i.StartedAt,
		&i.CompletedAt,
		&i.Error,
		&i.CreatedAt,
		&i.UpdatedAt,
	)
	return i, err
}

func (q *Queries) UpdateIPAddressStatus(ctx context.Context, arg UpdateIPAddressStatusParams) (IpAddress, error) {
	row := q.db.QueryRow(ctx, updateIPAddressStatus, arg.ID, arg.Status)
	var i IpAddress
	err := row.Scan(
		&i.ID,
		&i.PrefixID,
		&i.IpAddress,
		&i.InterfaceID,
		&i.Status,
		&i.Description,
		&i.CreatedAt,
		&i.UpdatedAt,
		&i.LastSeenAt,
	)
	return i, err
}
