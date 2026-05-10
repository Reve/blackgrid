-- name: CreateDiscoveryScan :one
INSERT INTO discovery_scans (prefix_id, status)
VALUES ($1, $2)
RETURNING *;

-- name: UpdateDiscoveryScanStatus :one
UPDATE discovery_scans
SET status = $2,
    started_at = COALESCE($3, started_at),
    completed_at = COALESCE($4, completed_at),
    error = COALESCE($5, error),
    updated_at = CURRENT_TIMESTAMP
WHERE id = $1
RETURNING *;

-- name: GetDiscoveryScan :one
SELECT * FROM discovery_scans WHERE id = $1 LIMIT 1;

-- name: ListDiscoveryScans :many
SELECT * FROM discovery_scans
WHERE ($1::uuid IS NULL OR prefix_id = $1)
AND ($2::text = '' OR status = $2)
ORDER BY created_at DESC
LIMIT $4 OFFSET $3;

-- name: CreateDiscoveryResult :one
INSERT INTO discovery_results (
    scan_id, prefix_id, address, mac_address, hostname, reverse_dns,
    open_ports, latency_ms, classification, raw
)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
RETURNING *;

-- name: GetDiscoveryResult :one
SELECT * FROM discovery_results WHERE id = $1 LIMIT 1;

-- name: ListDiscoveryResults :many
SELECT * FROM (
    SELECT DISTINCT ON (prefix_id, address) *
    FROM discovery_results
    WHERE ($1::uuid IS NULL OR scan_id = $1)
    AND ($2::uuid IS NULL OR prefix_id = $2)
    AND ($3::text = '' OR classification = $3)
    AND ($4::boolean IS NULL OR ignored = $4)
    AND (
        cardinality($5::int[]) = 0
        OR EXISTS (
            SELECT 1
            FROM jsonb_array_elements_text(open_ports) AS port(value)
            WHERE port.value::int = ANY($5::int[])
        )
    )
    ORDER BY prefix_id, address, seen_at DESC, created_at DESC
) latest
ORDER BY seen_at DESC
LIMIT $7 OFFSET $6;

-- name: UpdateDiscoveryResultAccepted :one
UPDATE discovery_results
SET accepted_at = CURRENT_TIMESTAMP,
    created_ip_address_id = $2,
    updated_at = CURRENT_TIMESTAMP
WHERE id = $1
RETURNING *;

-- name: UpdateDiscoveryResultIgnored :one
UPDATE discovery_results
SET ignored = true,
    classification = 'ignored',
    updated_at = CURRENT_TIMESTAMP
WHERE id = $1
RETURNING *;

-- name: GetRunningOrQueuedScansForPrefix :many
SELECT * FROM discovery_scans
WHERE prefix_id = $1 AND status IN ('queued', 'running');

-- name: GetPrefixesForScheduledScans :many
SELECT p.* FROM prefixes p
WHERE p.scan_enabled = true
AND NOT EXISTS (
    SELECT 1 FROM discovery_scans ds
    WHERE ds.prefix_id = p.id
    AND ds.status IN ('queued', 'running')
)
AND (
    NOT EXISTS (
        SELECT 1 FROM discovery_scans ds
        WHERE ds.prefix_id = p.id
        AND ds.status = 'completed'
    )
    OR EXISTS (
        SELECT 1 FROM discovery_scans ds
        WHERE ds.prefix_id = p.id
        AND ds.status = 'completed'
        AND ds.completed_at < CURRENT_TIMESTAMP - (p.scan_interval_seconds * interval '1 second')
    )
);

-- name: GetRecentDiscoveryResultsByAddress :many
SELECT * FROM discovery_results
WHERE prefix_id = $1 AND address = $2
ORDER BY seen_at DESC
LIMIT 5;
