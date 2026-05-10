-- name: CreateUser :one
INSERT INTO users (email, display_name, password_hash, role, enabled)
VALUES ($1, $2, $3, $4, $5)
RETURNING id, email, display_name, password_hash, role, enabled, last_login_at, created_at, updated_at;

-- name: GetUserByID :one
SELECT id, email, display_name, password_hash, role, enabled, last_login_at, created_at, updated_at
FROM users WHERE id = $1 LIMIT 1;

-- name: GetUserByEmail :one
SELECT id, email, display_name, password_hash, role, enabled, last_login_at, created_at, updated_at
FROM users WHERE email = $1 LIMIT 1;

-- name: ListUsers :many
SELECT id, email, display_name, password_hash, role, enabled, last_login_at, created_at, updated_at
FROM users ORDER BY created_at ASC;

-- name: UpdateUser :one
UPDATE users
SET display_name = $2, role = $3, enabled = $4, updated_at = CURRENT_TIMESTAMP
WHERE id = $1
RETURNING id, email, display_name, password_hash, role, enabled, last_login_at, created_at, updated_at;

-- name: UpdateUserPassword :one
UPDATE users
SET password_hash = $2, updated_at = CURRENT_TIMESTAMP
WHERE id = $1
RETURNING id, email, display_name, password_hash, role, enabled, last_login_at, created_at, updated_at;

-- name: UpdateUserLastLogin :exec
UPDATE users SET last_login_at = CURRENT_TIMESTAMP WHERE id = $1;

-- name: DeleteUser :exec
DELETE FROM users WHERE id = $1;

-- name: CountUsers :one
SELECT COUNT(*) FROM users;

-- name: CountEnabledAdmins :one
SELECT COUNT(*) FROM users WHERE role = 'admin' AND enabled = true;

-- name: CreateSession :one
INSERT INTO sessions (user_id, session_hash, expires_at)
VALUES ($1, $2, $3)
RETURNING id, user_id, session_hash, expires_at, created_at, last_seen_at;

-- name: GetSessionByHash :one
SELECT id, user_id, session_hash, expires_at, created_at, last_seen_at
FROM sessions WHERE session_hash = $1 LIMIT 1;

-- name: TouchSession :exec
UPDATE sessions SET last_seen_at = CURRENT_TIMESTAMP WHERE id = $1;

-- name: DeleteSession :exec
DELETE FROM sessions WHERE id = $1;

-- name: DeleteExpiredSessions :exec
DELETE FROM sessions WHERE expires_at < CURRENT_TIMESTAMP;

-- name: CreateAPIToken :one
INSERT INTO api_tokens (user_id, name, token_hash, role, expires_at)
VALUES ($1, $2, $3, $4, $5)
RETURNING id, user_id, name, token_hash, role, last_used_at, expires_at, created_at;

-- name: GetAPITokenByHash :one
SELECT id, user_id, name, token_hash, role, last_used_at, expires_at, created_at
FROM api_tokens WHERE token_hash = $1 LIMIT 1;

-- name: GetAPITokenByID :one
SELECT id, user_id, name, token_hash, role, last_used_at, expires_at, created_at
FROM api_tokens WHERE id = $1 LIMIT 1;

-- name: ListAPITokens :many
SELECT id, user_id, name, token_hash, role, last_used_at, expires_at, created_at
FROM api_tokens ORDER BY created_at DESC;

-- name: TouchAPIToken :exec
UPDATE api_tokens SET last_used_at = CURRENT_TIMESTAMP WHERE id = $1;

-- name: DeleteAPIToken :exec
DELETE FROM api_tokens WHERE id = $1;

-- name: CreateAuditLog :one
INSERT INTO audit_log (action, entity_type, entity_id, changes, actor_user_id, actor_type, actor_api_token_id, request_id, ip_address, object_type, object_id, before_state, after_state)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13)
RETURNING id, action, entity_type, entity_id, changes, actor_user_id, actor_type, actor_api_token_id, request_id, ip_address, object_type, object_id, before_state, after_state, created_at;

-- name: ListAuditLogs :many
SELECT id, action, entity_type, entity_id, changes, actor_user_id, actor_type, actor_api_token_id, request_id, ip_address, object_type, object_id, before_state, after_state, created_at
FROM audit_log
WHERE
    (sqlc.narg(actor_user_id)::uuid IS NULL OR actor_user_id = sqlc.narg(actor_user_id))
    AND (sqlc.narg(action)::text IS NULL OR action = sqlc.narg(action))
    AND (sqlc.narg(object_type)::text IS NULL OR object_type = sqlc.narg(object_type))
    AND (sqlc.narg(object_id)::uuid IS NULL OR object_id = sqlc.narg(object_id))
ORDER BY created_at DESC
LIMIT sqlc.arg(lim) OFFSET sqlc.arg(off);
