-- name: CleanupMonitorResults :exec
DELETE FROM monitor_results
WHERE checked_at < CURRENT_TIMESTAMP - ($1::int || ' days')::interval;

-- name: CleanupNotificationDeliveries :exec
DELETE FROM notification_deliveries
WHERE created_at < CURRENT_TIMESTAMP - ($1::int || ' days')::interval;

-- name: CleanupAuditLogs :exec
DELETE FROM audit_log
WHERE created_at < CURRENT_TIMESTAMP - ($1::int || ' days')::interval;

-- name: CleanupDiscoveryResults :exec
DELETE FROM discovery_results
WHERE seen_at < CURRENT_TIMESTAMP - ($1::int || ' days')::interval;

-- name: CleanupDiscoveryScans :exec
DELETE FROM discovery_scans
WHERE created_at < CURRENT_TIMESTAMP - ($1::int || ' days')::interval;
