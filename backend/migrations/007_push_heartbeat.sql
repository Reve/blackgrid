-- +goose Up

-- Dedicated heartbeat timestamp for push monitors. We can no longer overload
-- last_checked_at because that field is also bumped by the scheduler when it
-- evaluates a push monitor for overdue, which would otherwise mask a missing
-- heartbeat as "fresh".
ALTER TABLE monitors ADD COLUMN IF NOT EXISTS last_heartbeat_at TIMESTAMPTZ;

-- Backfill: for existing push monitors, seed last_heartbeat_at from
-- last_checked_at so the upgrade does not flip them to "down".
UPDATE monitors
SET last_heartbeat_at = last_checked_at
WHERE monitor_type = 'push' AND last_heartbeat_at IS NULL;

-- +goose Down
ALTER TABLE monitors DROP COLUMN IF EXISTS last_heartbeat_at;
