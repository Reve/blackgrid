-- +goose Up

CREATE TABLE users (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    email TEXT NOT NULL UNIQUE,
    display_name TEXT NOT NULL,
    password_hash TEXT NOT NULL,
    role TEXT NOT NULL CHECK (role IN ('admin', 'operator', 'viewer')),
    enabled BOOLEAN NOT NULL DEFAULT true,
    last_login_at TIMESTAMP WITH TIME ZONE,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_users_email ON users (email);
CREATE INDEX idx_users_role ON users (role);

CREATE TABLE sessions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    session_hash TEXT NOT NULL UNIQUE,
    expires_at TIMESTAMP WITH TIME ZONE NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
    last_seen_at TIMESTAMP WITH TIME ZONE
);

CREATE INDEX idx_sessions_user_id ON sessions (user_id);
CREATE INDEX idx_sessions_session_hash ON sessions (session_hash);
CREATE INDEX idx_sessions_expires_at ON sessions (expires_at);

CREATE TABLE api_tokens (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    name TEXT NOT NULL,
    token_hash TEXT NOT NULL UNIQUE,
    role TEXT NOT NULL CHECK (role IN ('admin', 'operator', 'viewer')),
    last_used_at TIMESTAMP WITH TIME ZONE,
    expires_at TIMESTAMP WITH TIME ZONE,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_api_tokens_user_id ON api_tokens (user_id);
CREATE INDEX idx_api_tokens_token_hash ON api_tokens (token_hash);

-- Extend audit_log with actor metadata
ALTER TABLE audit_log
    ADD COLUMN IF NOT EXISTS actor_user_id UUID REFERENCES users(id) ON DELETE SET NULL,
    ADD COLUMN IF NOT EXISTS actor_type TEXT,
    ADD COLUMN IF NOT EXISTS actor_api_token_id UUID REFERENCES api_tokens(id) ON DELETE SET NULL,
    ADD COLUMN IF NOT EXISTS request_id TEXT,
    ADD COLUMN IF NOT EXISTS ip_address INET,
    ADD COLUMN IF NOT EXISTS object_type TEXT,
    ADD COLUMN IF NOT EXISTS object_id UUID,
    ADD COLUMN IF NOT EXISTS before_state JSONB,
    ADD COLUMN IF NOT EXISTS after_state JSONB;

-- Rename audit_log columns to match new schema if needed (entity_type -> object_type collision handled by aliases)
-- entity_type and entity_id remain; object_type and object_id are new aliases for the same purpose
-- We keep both for backward compatibility - service layer will populate object_type/object_id going forward.

-- +goose Down
ALTER TABLE audit_log
    DROP COLUMN IF EXISTS after_state,
    DROP COLUMN IF EXISTS before_state,
    DROP COLUMN IF EXISTS object_id,
    DROP COLUMN IF EXISTS object_type,
    DROP COLUMN IF EXISTS ip_address,
    DROP COLUMN IF EXISTS request_id,
    DROP COLUMN IF EXISTS actor_api_token_id,
    DROP COLUMN IF EXISTS actor_type,
    DROP COLUMN IF EXISTS actor_user_id;

DROP INDEX IF EXISTS idx_api_tokens_token_hash;
DROP INDEX IF EXISTS idx_api_tokens_user_id;
DROP TABLE IF EXISTS api_tokens;

DROP INDEX IF EXISTS idx_sessions_expires_at;
DROP INDEX IF EXISTS idx_sessions_session_hash;
DROP INDEX IF EXISTS idx_sessions_user_id;
DROP TABLE IF EXISTS sessions;

DROP INDEX IF EXISTS idx_users_role;
DROP INDEX IF EXISTS idx_users_email;
DROP TABLE IF EXISTS users;
