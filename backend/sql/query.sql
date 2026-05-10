-- name: GetSites :many
SELECT * FROM sites ORDER BY name;

-- name: GetSite :one
SELECT * FROM sites WHERE id = $1 LIMIT 1;

-- name: CreateSite :one
INSERT INTO sites (name, description) VALUES ($1, $2) RETURNING *;

-- name: UpdateSite :one
UPDATE sites SET name = $2, description = $3, updated_at = CURRENT_TIMESTAMP WHERE id = $1 RETURNING *;

-- name: DeleteSite :exec
DELETE FROM sites WHERE id = $1;


-- name: GetVlans :many
SELECT * FROM vlans ORDER BY vlan_id;

-- name: GetVlan :one
SELECT * FROM vlans WHERE id = $1 LIMIT 1;

-- name: CreateVlan :one
INSERT INTO vlans (site_id, vlan_id, name, description) VALUES ($1, $2, $3, $4) RETURNING *;

-- name: UpdateVlan :one
UPDATE vlans SET site_id = $2, vlan_id = $3, name = $4, description = $5, updated_at = CURRENT_TIMESTAMP WHERE id = $1 RETURNING *;

-- name: DeleteVlan :exec
DELETE FROM vlans WHERE id = $1;


-- name: GetPrefixes :many
SELECT * FROM prefixes ORDER BY prefix;

-- name: GetPrefix :one
SELECT * FROM prefixes WHERE id = $1 LIMIT 1;

-- name: CreatePrefix :one
INSERT INTO prefixes (site_id, vlan_id, prefix, description) VALUES ($1, $2, $3, $4) RETURNING *;

-- name: UpdatePrefix :one
UPDATE prefixes SET site_id = $2, vlan_id = $3, prefix = $4, description = $5, updated_at = CURRENT_TIMESTAMP WHERE id = $1 RETURNING *;

-- name: DeletePrefix :exec
DELETE FROM prefixes WHERE id = $1;


-- name: GetIPAddresses :many
SELECT * FROM ip_addresses ORDER BY ip_address;

-- name: GetIPAddress :one
SELECT * FROM ip_addresses WHERE id = $1 LIMIT 1;

-- name: CreateIPAddress :one
INSERT INTO ip_addresses (prefix_id, ip_address, interface_id, status, description) VALUES ($1, $2, $3, $4, $5) RETURNING *;

-- name: UpdateIPAddress :one
UPDATE ip_addresses SET prefix_id = $2, ip_address = $3, interface_id = $4, status = $5, description = $6, updated_at = CURRENT_TIMESTAMP WHERE id = $1 RETURNING *;

-- name: DeleteIPAddress :exec
DELETE FROM ip_addresses WHERE id = $1;

-- name: GetIPAddressesByPrefix :many
SELECT * FROM ip_addresses WHERE prefix_id = $1 ORDER BY ip_address;

-- name: UpdateIPAddressLastSeen :one
UPDATE ip_addresses SET last_seen_at = CURRENT_TIMESTAMP, updated_at = CURRENT_TIMESTAMP WHERE id = $1 RETURNING *;

-- name: UpdateIPAddressStatus :one
UPDATE ip_addresses SET status = $2, updated_at = CURRENT_TIMESTAMP WHERE id = $1 RETURNING *;

-- name: UpdatePrefixScanConfig :one
UPDATE prefixes SET scan_enabled = $2, scan_interval_seconds = $3, updated_at = CURRENT_TIMESTAMP WHERE id = $1 RETURNING *;

-- name: GetLatestScanForPrefix :one
SELECT * FROM discovery_scans WHERE prefix_id = $1 ORDER BY created_at DESC LIMIT 1;


-- name: GetDevices :many
SELECT * FROM devices ORDER BY name;

-- name: GetDevice :one
SELECT * FROM devices WHERE id = $1 LIMIT 1;

-- name: CreateDevice :one
INSERT INTO devices (name, site_id, description, status) VALUES ($1, $2, $3, $4) RETURNING *;

-- name: UpdateDevice :one
UPDATE devices SET name = $2, site_id = $3, description = $4, status = $5, updated_at = CURRENT_TIMESTAMP WHERE id = $1 RETURNING *;

-- name: DeleteDevice :exec
DELETE FROM devices WHERE id = $1;

-- name: GetMonitors :many
SELECT * FROM monitors ORDER BY name;

-- name: GetMonitor :one
SELECT * FROM monitors WHERE id = $1 LIMIT 1;

-- name: CreateMonitor :one
INSERT INTO monitors (name, slug, monitor_type, target, config, ip_address_id, device_id, interval_seconds, timeout_seconds, retry_count, enabled, status, push_token_hash)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13) RETURNING *;

-- name: UpdateMonitor :one
UPDATE monitors
SET name = $2, slug = $3, monitor_type = $4, target = $5, config = $6, ip_address_id = $7, device_id = $8, interval_seconds = $9, timeout_seconds = $10, retry_count = $11, enabled = $12, status = $13, last_checked_at = $14, last_status_change_at = $15, push_token_hash = $16, updated_at = CURRENT_TIMESTAMP
WHERE id = $1 RETURNING *;

-- name: DeleteMonitor :exec
DELETE FROM monitors WHERE id = $1;

-- name: GetMonitorsDueForCheck :many
SELECT * FROM monitors
WHERE enabled = true
  AND status != 'paused'
  AND (last_checked_at IS NULL OR last_checked_at < CURRENT_TIMESTAMP - (interval_seconds || ' seconds')::interval)
ORDER BY last_checked_at ASC NULLS FIRST;

-- name: GetMonitorResults :many
SELECT * FROM monitor_results WHERE monitor_id = $1 ORDER BY checked_at DESC LIMIT $2 OFFSET $3;

-- name: GetMonitorByPushTokenHash :one
SELECT * FROM monitors WHERE push_token_hash = $1 LIMIT 1;

-- name: RecordPushHeartbeat :one
UPDATE monitors
SET last_heartbeat_at = $2,
    last_checked_at = $2,
    status = $3,
    last_status_change_at = $4,
    updated_at = CURRENT_TIMESTAMP
WHERE id = $1 RETURNING *;

-- name: CreateMonitorResult :one
INSERT INTO monitor_results (monitor_id, status, latency_ms, error_message, details)
VALUES ($1, $2, $3, $4, $5) RETURNING *;
