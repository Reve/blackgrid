-- +goose Up

-- Add details JSONB column to monitor_results for type-specific check output
ALTER TABLE monitor_results ADD COLUMN IF NOT EXISTS details JSONB;

-- Add push_token_hash to monitors for push heartbeat monitor type
-- The hash is stored instead of the plaintext token.
ALTER TABLE monitors ADD COLUMN IF NOT EXISTS push_token_hash TEXT;

-- Update monitor_type comment to reflect all supported types
COMMENT ON COLUMN monitors.monitor_type IS 'http, tcp, ping, dns, tls, push, postgres';

-- +goose Down
ALTER TABLE monitor_results DROP COLUMN IF EXISTS details;
ALTER TABLE monitors DROP COLUMN IF EXISTS push_token_hash;
