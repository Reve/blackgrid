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
