-- name: CleanupMonitorResults :execrows
DELETE FROM monitor_results
WHERE checked_at < CURRENT_TIMESTAMP - ($1::int || ' days')::interval;

-- name: CleanupNotificationDeliveries :execrows
DELETE FROM notification_deliveries
WHERE created_at < CURRENT_TIMESTAMP - ($1::int || ' days')::interval;

-- name: CleanupAuditLogs :execrows
DELETE FROM audit_log
WHERE created_at < CURRENT_TIMESTAMP - ($1::int || ' days')::interval;

-- name: CleanupDiscoveryResults :execrows
DELETE FROM discovery_results
WHERE seen_at < CURRENT_TIMESTAMP - ($1::int || ' days')::interval;

-- name: CleanupDiscoveryScans :execrows
DELETE FROM discovery_scans
WHERE created_at < CURRENT_TIMESTAMP - ($1::int || ' days')::interval;
