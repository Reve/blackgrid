package db

import (
	"context"
	"net/netip"

	"github.com/jackc/pgx/v5/pgtype"
)

// ---- User queries ----

const createUser = `-- name: CreateUser :one
INSERT INTO users (email, display_name, password_hash, role, enabled)
VALUES ($1, $2, $3, $4, $5)
RETURNING id, email, display_name, password_hash, role, enabled, last_login_at, created_at, updated_at`

type CreateUserParams struct {
	Email        string `json:"email"`
	DisplayName  string `json:"display_name"`
	PasswordHash string `json:"password_hash"`
	Role         string `json:"role"`
	Enabled      bool   `json:"enabled"`
}

func (q *Queries) CreateUser(ctx context.Context, arg CreateUserParams) (User, error) {
	row := q.db.QueryRow(ctx, createUser, arg.Email, arg.DisplayName, arg.PasswordHash, arg.Role, arg.Enabled)
	var u User
	err := row.Scan(&u.ID, &u.Email, &u.DisplayName, &u.PasswordHash, &u.Role, &u.Enabled, &u.LastLoginAt, &u.CreatedAt, &u.UpdatedAt)
	return u, err
}

const getUserByID = `SELECT id, email, display_name, password_hash, role, enabled, last_login_at, created_at, updated_at FROM users WHERE id = $1 LIMIT 1`

func (q *Queries) GetUserByID(ctx context.Context, id pgtype.UUID) (User, error) {
	row := q.db.QueryRow(ctx, getUserByID, id)
	var u User
	err := row.Scan(&u.ID, &u.Email, &u.DisplayName, &u.PasswordHash, &u.Role, &u.Enabled, &u.LastLoginAt, &u.CreatedAt, &u.UpdatedAt)
	return u, err
}

const getUserByEmail = `SELECT id, email, display_name, password_hash, role, enabled, last_login_at, created_at, updated_at FROM users WHERE email = $1 LIMIT 1`

func (q *Queries) GetUserByEmail(ctx context.Context, email string) (User, error) {
	row := q.db.QueryRow(ctx, getUserByEmail, email)
	var u User
	err := row.Scan(&u.ID, &u.Email, &u.DisplayName, &u.PasswordHash, &u.Role, &u.Enabled, &u.LastLoginAt, &u.CreatedAt, &u.UpdatedAt)
	return u, err
}

const listUsers = `SELECT id, email, display_name, password_hash, role, enabled, last_login_at, created_at, updated_at FROM users ORDER BY created_at ASC`

func (q *Queries) ListUsers(ctx context.Context) ([]User, error) {
	rows, err := q.db.Query(ctx, listUsers)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var items []User
	for rows.Next() {
		var u User
		if err := rows.Scan(&u.ID, &u.Email, &u.DisplayName, &u.PasswordHash, &u.Role, &u.Enabled, &u.LastLoginAt, &u.CreatedAt, &u.UpdatedAt); err != nil {
			return nil, err
		}
		items = append(items, u)
	}
	return items, rows.Err()
}

type UpdateUserParams struct {
	ID          pgtype.UUID `json:"id"`
	DisplayName string      `json:"display_name"`
	Role        string      `json:"role"`
	Enabled     bool        `json:"enabled"`
}

const updateUser = `UPDATE users SET display_name = $2, role = $3, enabled = $4, updated_at = CURRENT_TIMESTAMP WHERE id = $1 RETURNING id, email, display_name, password_hash, role, enabled, last_login_at, created_at, updated_at`

func (q *Queries) UpdateUser(ctx context.Context, arg UpdateUserParams) (User, error) {
	row := q.db.QueryRow(ctx, updateUser, arg.ID, arg.DisplayName, arg.Role, arg.Enabled)
	var u User
	err := row.Scan(&u.ID, &u.Email, &u.DisplayName, &u.PasswordHash, &u.Role, &u.Enabled, &u.LastLoginAt, &u.CreatedAt, &u.UpdatedAt)
	return u, err
}

const updateUserPassword = `UPDATE users SET password_hash = $2, updated_at = CURRENT_TIMESTAMP WHERE id = $1 RETURNING id, email, display_name, password_hash, role, enabled, last_login_at, created_at, updated_at`

func (q *Queries) UpdateUserPassword(ctx context.Context, id pgtype.UUID, passwordHash string) (User, error) {
	row := q.db.QueryRow(ctx, updateUserPassword, id, passwordHash)
	var u User
	err := row.Scan(&u.ID, &u.Email, &u.DisplayName, &u.PasswordHash, &u.Role, &u.Enabled, &u.LastLoginAt, &u.CreatedAt, &u.UpdatedAt)
	return u, err
}

const updateUserLastLogin = `UPDATE users SET last_login_at = CURRENT_TIMESTAMP WHERE id = $1`

func (q *Queries) UpdateUserLastLogin(ctx context.Context, id pgtype.UUID) error {
	_, err := q.db.Exec(ctx, updateUserLastLogin, id)
	return err
}

const deleteUser = `DELETE FROM users WHERE id = $1`

func (q *Queries) DeleteUser(ctx context.Context, id pgtype.UUID) error {
	_, err := q.db.Exec(ctx, deleteUser, id)
	return err
}

const countUsers = `SELECT COUNT(*) FROM users`

func (q *Queries) CountUsers(ctx context.Context) (int64, error) {
	var n int64
	err := q.db.QueryRow(ctx, countUsers).Scan(&n)
	return n, err
}

const countEnabledAdmins = `SELECT COUNT(*) FROM users WHERE role = 'admin' AND enabled = true`

func (q *Queries) CountEnabledAdmins(ctx context.Context) (int64, error) {
	var n int64
	err := q.db.QueryRow(ctx, countEnabledAdmins).Scan(&n)
	return n, err
}

// ---- Session queries ----

type CreateSessionParams struct {
	UserID      pgtype.UUID        `json:"user_id"`
	SessionHash string             `json:"session_hash"`
	ExpiresAt   pgtype.Timestamptz `json:"expires_at"`
}

const createSession = `INSERT INTO sessions (user_id, session_hash, expires_at) VALUES ($1, $2, $3) RETURNING id, user_id, session_hash, expires_at, created_at, last_seen_at`

func (q *Queries) CreateSession(ctx context.Context, arg CreateSessionParams) (Session, error) {
	row := q.db.QueryRow(ctx, createSession, arg.UserID, arg.SessionHash, arg.ExpiresAt)
	var s Session
	err := row.Scan(&s.ID, &s.UserID, &s.SessionHash, &s.ExpiresAt, &s.CreatedAt, &s.LastSeenAt)
	return s, err
}

const getSessionByHash = `SELECT id, user_id, session_hash, expires_at, created_at, last_seen_at FROM sessions WHERE session_hash = $1 LIMIT 1`

func (q *Queries) GetSessionByHash(ctx context.Context, hash string) (Session, error) {
	row := q.db.QueryRow(ctx, getSessionByHash, hash)
	var s Session
	err := row.Scan(&s.ID, &s.UserID, &s.SessionHash, &s.ExpiresAt, &s.CreatedAt, &s.LastSeenAt)
	return s, err
}

const touchSession = `UPDATE sessions SET last_seen_at = CURRENT_TIMESTAMP WHERE id = $1`

func (q *Queries) TouchSession(ctx context.Context, id pgtype.UUID) error {
	_, err := q.db.Exec(ctx, touchSession, id)
	return err
}

const deleteSession = `DELETE FROM sessions WHERE id = $1`

func (q *Queries) DeleteSession(ctx context.Context, id pgtype.UUID) error {
	_, err := q.db.Exec(ctx, deleteSession, id)
	return err
}

const deleteExpiredSessions = `DELETE FROM sessions WHERE expires_at < CURRENT_TIMESTAMP`

func (q *Queries) DeleteExpiredSessions(ctx context.Context) error {
	_, err := q.db.Exec(ctx, deleteExpiredSessions)
	return err
}

// ---- API Token queries ----

type CreateAPITokenParams struct {
	UserID    pgtype.UUID        `json:"user_id"`
	Name      string             `json:"name"`
	TokenHash string             `json:"token_hash"`
	Role      string             `json:"role"`
	ExpiresAt pgtype.Timestamptz `json:"expires_at"`
}

const createAPIToken = `INSERT INTO api_tokens (user_id, name, token_hash, role, expires_at) VALUES ($1, $2, $3, $4, $5) RETURNING id, user_id, name, token_hash, role, last_used_at, expires_at, created_at`

func (q *Queries) CreateAPIToken(ctx context.Context, arg CreateAPITokenParams) (ApiToken, error) {
	row := q.db.QueryRow(ctx, createAPIToken, arg.UserID, arg.Name, arg.TokenHash, arg.Role, arg.ExpiresAt)
	var t ApiToken
	err := row.Scan(&t.ID, &t.UserID, &t.Name, &t.TokenHash, &t.Role, &t.LastUsedAt, &t.ExpiresAt, &t.CreatedAt)
	return t, err
}

const getAPITokenByHash = `SELECT id, user_id, name, token_hash, role, last_used_at, expires_at, created_at FROM api_tokens WHERE token_hash = $1 LIMIT 1`

func (q *Queries) GetAPITokenByHash(ctx context.Context, hash string) (ApiToken, error) {
	row := q.db.QueryRow(ctx, getAPITokenByHash, hash)
	var t ApiToken
	err := row.Scan(&t.ID, &t.UserID, &t.Name, &t.TokenHash, &t.Role, &t.LastUsedAt, &t.ExpiresAt, &t.CreatedAt)
	return t, err
}

const getAPITokenByID = `SELECT id, user_id, name, token_hash, role, last_used_at, expires_at, created_at FROM api_tokens WHERE id = $1 LIMIT 1`

func (q *Queries) GetAPITokenByID(ctx context.Context, id pgtype.UUID) (ApiToken, error) {
	row := q.db.QueryRow(ctx, getAPITokenByID, id)
	var t ApiToken
	err := row.Scan(&t.ID, &t.UserID, &t.Name, &t.TokenHash, &t.Role, &t.LastUsedAt, &t.ExpiresAt, &t.CreatedAt)
	return t, err
}

const listAPITokens = `SELECT id, user_id, name, token_hash, role, last_used_at, expires_at, created_at FROM api_tokens ORDER BY created_at DESC`

func (q *Queries) ListAPITokens(ctx context.Context) ([]ApiToken, error) {
	rows, err := q.db.Query(ctx, listAPITokens)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var items []ApiToken
	for rows.Next() {
		var t ApiToken
		if err := rows.Scan(&t.ID, &t.UserID, &t.Name, &t.TokenHash, &t.Role, &t.LastUsedAt, &t.ExpiresAt, &t.CreatedAt); err != nil {
			return nil, err
		}
		items = append(items, t)
	}
	return items, rows.Err()
}

const touchAPIToken = `UPDATE api_tokens SET last_used_at = CURRENT_TIMESTAMP WHERE id = $1`

func (q *Queries) TouchAPIToken(ctx context.Context, id pgtype.UUID) error {
	_, err := q.db.Exec(ctx, touchAPIToken, id)
	return err
}

const deleteAPIToken = `DELETE FROM api_tokens WHERE id = $1`

func (q *Queries) DeleteAPIToken(ctx context.Context, id pgtype.UUID) error {
	_, err := q.db.Exec(ctx, deleteAPIToken, id)
	return err
}

// ---- Audit log ----

type CreateAuditLogParams struct {
	Action           string             `json:"action"`
	EntityType       string             `json:"entity_type"`
	EntityID         pgtype.UUID        `json:"entity_id"`
	Changes          []byte             `json:"changes"`
	ActorUserID      pgtype.UUID        `json:"actor_user_id"`
	ActorType        pgtype.Text        `json:"actor_type"`
	ActorAPITokenID  pgtype.UUID        `json:"actor_api_token_id"`
	RequestID        pgtype.Text        `json:"request_id"`
	IPAddress        *netip.Addr        `json:"ip_address"`
	ObjectType       pgtype.Text        `json:"object_type"`
	ObjectID         pgtype.UUID        `json:"object_id"`
	BeforeState      []byte             `json:"before_state"`
	AfterState       []byte             `json:"after_state"`
}

const createAuditLog = `INSERT INTO audit_log (action, entity_type, entity_id, changes, actor_user_id, actor_type, actor_api_token_id, request_id, ip_address, object_type, object_id, before_state, after_state)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9::inet, $10, $11, $12, $13)
RETURNING id, action, entity_type, entity_id, changes, actor_user_id, actor_type, actor_api_token_id, request_id, ip_address, object_type, object_id, before_state, after_state, created_at`

func (q *Queries) CreateAuditLog(ctx context.Context, arg CreateAuditLogParams) (AuditLogEntry, error) {
	var ipStr interface{}
	if arg.IPAddress != nil && arg.IPAddress.IsValid() {
		ipStr = arg.IPAddress.String()
	}
	row := q.db.QueryRow(ctx, createAuditLog,
		arg.Action, arg.EntityType, arg.EntityID, arg.Changes,
		arg.ActorUserID, arg.ActorType, arg.ActorAPITokenID,
		arg.RequestID, ipStr, arg.ObjectType, arg.ObjectID,
		arg.BeforeState, arg.AfterState,
	)
	var a AuditLogEntry
	err := row.Scan(&a.ID, &a.Action, &a.EntityType, &a.EntityID, &a.Changes,
		&a.ActorUserID, &a.ActorType, &a.ActorAPITokenID, &a.RequestID,
		&a.IPAddress, &a.ObjectType, &a.ObjectID, &a.BeforeState, &a.AfterState, &a.CreatedAt)
	return a, err
}

type ListAuditLogsParams struct {
	ActorUserID pgtype.UUID
	Action      pgtype.Text
	ObjectType  pgtype.Text
	ObjectID    pgtype.UUID
	Limit       int32
	Offset      int32
}

const listAuditLogs = `SELECT id, action, entity_type, entity_id, changes, actor_user_id, actor_type, actor_api_token_id, request_id, ip_address, object_type, object_id, before_state, after_state, created_at
FROM audit_log
WHERE
    ($1::uuid IS NULL OR actor_user_id = $1)
    AND ($2::text IS NULL OR action = $2)
    AND ($3::text IS NULL OR object_type = $3)
    AND ($4::uuid IS NULL OR object_id = $4)
ORDER BY created_at DESC
LIMIT $5 OFFSET $6`

func (q *Queries) ListAuditLogs(ctx context.Context, arg ListAuditLogsParams) ([]AuditLogEntry, error) {
	rows, err := q.db.Query(ctx, listAuditLogs,
		arg.ActorUserID, arg.Action, arg.ObjectType, arg.ObjectID,
		arg.Limit, arg.Offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var items []AuditLogEntry
	for rows.Next() {
		var a AuditLogEntry
		if err := rows.Scan(&a.ID, &a.Action, &a.EntityType, &a.EntityID, &a.Changes,
			&a.ActorUserID, &a.ActorType, &a.ActorAPITokenID, &a.RequestID,
			&a.IPAddress, &a.ObjectType, &a.ObjectID, &a.BeforeState, &a.AfterState, &a.CreatedAt); err != nil {
			return nil, err
		}
		items = append(items, a)
	}
	return items, rows.Err()
}
