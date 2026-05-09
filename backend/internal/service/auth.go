package service

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"
	"time"

	"blackgrid/internal/db"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"golang.org/x/crypto/bcrypt"
)

const (
	RoleAdmin    = "admin"
	RoleOperator = "operator"
	RoleViewer   = "viewer"

	minPasswordLength = 12
	tokenPrefix       = "bg_"
	tokenRawBytes     = 32
	sessionRawBytes   = 32
	bcryptCost        = 12
)

var (
	ErrUserNotFound          = errors.New("user not found")
	ErrUserDisabled          = errors.New("user account is disabled")
	ErrInvalidCredentials    = errors.New("invalid credentials")
	ErrPasswordTooShort      = errors.New("password must be at least 12 characters")
	ErrInvalidRole           = errors.New("invalid role: must be admin, operator, or viewer")
	ErrSetupAlreadyDone      = errors.New("setup already completed")
	ErrLastAdmin             = errors.New("cannot remove or disable the last enabled admin")
	ErrTokenExpired          = errors.New("api token has expired")
	ErrTokenNotFound         = errors.New("api token not found")
	ErrSessionExpired        = errors.New("session has expired")
	ErrSessionNotFound       = errors.New("session not found")
	ErrTokenRoleExceedsOwner = errors.New("api token role cannot exceed owner's role")
	ErrEmailTaken            = errors.New("email already in use")
)

// AuthConfig holds tunable auth settings from environment.
type AuthConfig struct {
	CookieSecure    bool
	SessionTTLHours int
}

func AuthConfigFromEnv() AuthConfig {
	secure := false
	if v := os.Getenv("AUTH_COOKIE_SECURE"); strings.EqualFold(v, "true") {
		secure = true
	}
	ttl := 168
	if v := os.Getenv("AUTH_SESSION_TTL_HOURS"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			ttl = n
		}
	}
	return AuthConfig{CookieSecure: secure, SessionTTLHours: ttl}
}

// AuthService handles users, sessions, API tokens.
type AuthService struct {
	q      *db.Queries
	config AuthConfig
	audit  *AuditService
}

func NewAuthService(q *db.Queries, cfg AuthConfig, audit *AuditService) *AuthService {
	return &AuthService{q: q, config: cfg, audit: audit}
}

// ---- Setup ----

// SetupRequired returns true if no users exist yet.
func (s *AuthService) SetupRequired(ctx context.Context) (bool, error) {
	n, err := s.q.CountUsers(ctx)
	if err != nil {
		return false, err
	}
	return n == 0, nil
}

// CreateFirstAdmin creates the initial admin user. Returns ErrSetupAlreadyDone if users exist.
func (s *AuthService) CreateFirstAdmin(ctx context.Context, email, displayName, password string) (db.User, error) {
	required, err := s.SetupRequired(ctx)
	if err != nil {
		return db.User{}, err
	}
	if !required {
		return db.User{}, ErrSetupAlreadyDone
	}
	if err := validatePassword(password); err != nil {
		return db.User{}, err
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcryptCost)
	if err != nil {
		return db.User{}, fmt.Errorf("hash password: %w", err)
	}
	user, err := s.q.CreateUser(ctx, db.CreateUserParams{
		Email:        strings.ToLower(strings.TrimSpace(email)),
		DisplayName:  displayName,
		PasswordHash: string(hash),
		Role:         RoleAdmin,
		Enabled:      true,
	})
	if err != nil {
		return db.User{}, err
	}
	if s.audit != nil {
		s.audit.Log(ctx, AuditParams{
			Action:     "user.create",
			EntityType: "user",
			EntityID:   user.ID,
			ObjectType: "user",
			ObjectID:   user.ID,
			ActorType:  "system",
		})
	}
	return user, nil
}

// ---- Login / Logout ----

// Login validates credentials and creates a session. Returns session hash.
func (s *AuthService) Login(ctx context.Context, email, password string) (db.User, string, error) {
	user, err := s.q.GetUserByEmail(ctx, strings.ToLower(strings.TrimSpace(email)))
	if err != nil {
		// Constant-time: always compute bcrypt even on a miss to avoid timing oracle.
		bcrypt.CompareHashAndPassword([]byte("$2a$12$notvalidhashXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXX"), []byte(password)) //nolint:errcheck
		if errors.Is(err, pgx.ErrNoRows) {
			return db.User{}, "", ErrInvalidCredentials
		}
		return db.User{}, "", err
	}
	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(password)); err != nil {
		if s.audit != nil {
			s.audit.Log(ctx, AuditParams{
				Action:     "auth.login_failure",
				EntityType: "user",
				EntityID:   user.ID,
				ObjectType: "user",
				ObjectID:   user.ID,
				ActorType:  "system",
			})
		}
		return db.User{}, "", ErrInvalidCredentials
	}
	if !user.Enabled {
		return db.User{}, "", ErrUserDisabled
	}

	rawToken := make([]byte, sessionRawBytes)
	if _, err := rand.Read(rawToken); err != nil {
		return db.User{}, "", fmt.Errorf("generate session: %w", err)
	}
	sessionHash := base64.URLEncoding.EncodeToString(rawToken)

	expiresAt := time.Now().Add(time.Duration(s.config.SessionTTLHours) * time.Hour)
	_, err = s.q.CreateSession(ctx, db.CreateSessionParams{
		UserID:      user.ID,
		SessionHash: sessionHash,
		ExpiresAt:   pgtype.Timestamptz{Time: expiresAt, Valid: true},
	})
	if err != nil {
		return db.User{}, "", fmt.Errorf("create session: %w", err)
	}
	_ = s.q.UpdateUserLastLogin(ctx, user.ID)

	if s.audit != nil {
		s.audit.Log(ctx, AuditParams{
			Action:      "auth.login_success",
			EntityType:  "user",
			EntityID:    user.ID,
			ObjectType:  "user",
			ObjectID:    user.ID,
			ActorType:   "user",
			ActorUserID: user.ID,
		})
	}
	return user, sessionHash, nil
}

// Logout deletes the session by hash.
func (s *AuthService) Logout(ctx context.Context, sessionHash string) error {
	sess, err := s.q.GetSessionByHash(ctx, sessionHash)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil // already gone — idempotent
		}
		return err
	}
	user, _ := s.q.GetUserByID(ctx, sess.UserID)
	if err := s.q.DeleteSession(ctx, sess.ID); err != nil {
		return err
	}
	if s.audit != nil {
		s.audit.Log(ctx, AuditParams{
			Action:      "auth.logout",
			EntityType:  "user",
			EntityID:    user.ID,
			ObjectType:  "user",
			ObjectID:    user.ID,
			ActorType:   "user",
			ActorUserID: user.ID,
		})
	}
	return nil
}

// ResolveSession validates a session cookie hash and returns the user.
func (s *AuthService) ResolveSession(ctx context.Context, hash string) (db.User, db.Session, error) {
	sess, err := s.q.GetSessionByHash(ctx, hash)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return db.User{}, db.Session{}, ErrSessionNotFound
		}
		return db.User{}, db.Session{}, err
	}
	if time.Now().After(sess.ExpiresAt.Time) {
		return db.User{}, db.Session{}, ErrSessionExpired
	}
	user, err := s.q.GetUserByID(ctx, sess.UserID)
	if err != nil {
		return db.User{}, db.Session{}, err
	}
	if !user.Enabled {
		return db.User{}, db.Session{}, ErrUserDisabled
	}
	go func() {
		if terr := s.q.TouchSession(context.Background(), sess.ID); terr != nil {
			log.Printf("touch session: %v", terr)
		}
	}()
	return user, sess, nil
}

// ResolveAPIToken validates a bearer token and returns the user/token.
func (s *AuthService) ResolveAPIToken(ctx context.Context, rawToken string) (db.User, db.ApiToken, error) {
	hash := sha256HexToken(rawToken)
	tok, err := s.q.GetAPITokenByHash(ctx, hash)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return db.User{}, db.ApiToken{}, ErrTokenNotFound
		}
		return db.User{}, db.ApiToken{}, err
	}
	if tok.ExpiresAt.Valid && time.Now().After(tok.ExpiresAt.Time) {
		return db.User{}, db.ApiToken{}, ErrTokenExpired
	}
	user, err := s.q.GetUserByID(ctx, tok.UserID)
	if err != nil {
		return db.User{}, db.ApiToken{}, err
	}
	if !user.Enabled {
		return db.User{}, db.ApiToken{}, ErrUserDisabled
	}
	go func() {
		if terr := s.q.TouchAPIToken(context.Background(), tok.ID); terr != nil {
			log.Printf("touch api token: %v", terr)
		}
	}()
	return user, tok, nil
}

// ---- User management ----

func (s *AuthService) ListUsers(ctx context.Context) ([]db.User, error) {
	users, err := s.q.ListUsers(ctx)
	if err != nil {
		return nil, err
	}
	if users == nil {
		users = []db.User{}
	}
	return users, nil
}

func (s *AuthService) GetUser(ctx context.Context, id pgtype.UUID) (db.User, error) {
	u, err := s.q.GetUserByID(ctx, id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return db.User{}, ErrUserNotFound
		}
		return db.User{}, err
	}
	return u, nil
}

type CreateUserInput struct {
	Email       string
	DisplayName string
	Password    string
	Role        string
	Enabled     bool
}

func (s *AuthService) CreateUser(ctx context.Context, actor db.User, p CreateUserInput) (db.User, error) {
	if err := validateRole(p.Role); err != nil {
		return db.User{}, err
	}
	if err := validatePassword(p.Password); err != nil {
		return db.User{}, err
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(p.Password), bcryptCost)
	if err != nil {
		return db.User{}, err
	}
	user, err := s.q.CreateUser(ctx, db.CreateUserParams{
		Email:        strings.ToLower(strings.TrimSpace(p.Email)),
		DisplayName:  p.DisplayName,
		PasswordHash: string(hash),
		Role:         p.Role,
		Enabled:      p.Enabled,
	})
	if err != nil {
		if strings.Contains(err.Error(), "unique") || strings.Contains(err.Error(), "duplicate") {
			return db.User{}, ErrEmailTaken
		}
		return db.User{}, err
	}
	if s.audit != nil {
		s.audit.Log(ctx, AuditParams{
			Action:      "user.create",
			EntityType:  "user",
			EntityID:    user.ID,
			ObjectType:  "user",
			ObjectID:    user.ID,
			ActorType:   "user",
			ActorUserID: actor.ID,
		})
	}
	return user, nil
}

type UpdateUserInput struct {
	DisplayName string
	Role        string
	Enabled     bool
	Password    string // optional — blank means preserve
}

func (s *AuthService) UpdateUser(ctx context.Context, actor db.User, id pgtype.UUID, p UpdateUserInput) (db.User, error) {
	existing, err := s.GetUser(ctx, id)
	if err != nil {
		return db.User{}, err
	}
	if err := validateRole(p.Role); err != nil {
		return db.User{}, err
	}
	// Safety: do not demote/disable the last enabled admin.
	if existing.Role == RoleAdmin {
		wouldLoseAdmin := p.Role != RoleAdmin || !p.Enabled
		if wouldLoseAdmin {
			if err := s.guardLastAdmin(ctx); err != nil {
				return db.User{}, err
			}
		}
	}
	user, err := s.q.UpdateUser(ctx, db.UpdateUserParams{
		ID:          id,
		DisplayName: p.DisplayName,
		Role:        p.Role,
		Enabled:     p.Enabled,
	})
	if err != nil {
		return db.User{}, err
	}
	if p.Password != "" {
		if err := validatePassword(p.Password); err != nil {
			return db.User{}, err
		}
		pwHash, err := bcrypt.GenerateFromPassword([]byte(p.Password), bcryptCost)
		if err != nil {
			return db.User{}, err
		}
		user, err = s.q.UpdateUserPassword(ctx, id, string(pwHash))
		if err != nil {
			return db.User{}, err
		}
	}
	if s.audit != nil {
		s.audit.Log(ctx, AuditParams{
			Action:      "user.update",
			EntityType:  "user",
			EntityID:    user.ID,
			ObjectType:  "user",
			ObjectID:    user.ID,
			ActorType:   "user",
			ActorUserID: actor.ID,
		})
	}
	return user, nil
}

func (s *AuthService) DeleteUser(ctx context.Context, actor db.User, id pgtype.UUID) error {
	existing, err := s.GetUser(ctx, id)
	if err != nil {
		return err
	}
	if existing.Role == RoleAdmin {
		if err := s.guardLastAdmin(ctx); err != nil {
			return err
		}
	}
	if err := s.q.DeleteUser(ctx, id); err != nil {
		return err
	}
	if s.audit != nil {
		s.audit.Log(ctx, AuditParams{
			Action:      "user.delete",
			EntityType:  "user",
			EntityID:    id,
			ObjectType:  "user",
			ObjectID:    id,
			ActorType:   "user",
			ActorUserID: actor.ID,
		})
	}
	return nil
}

func (s *AuthService) guardLastAdmin(ctx context.Context) error {
	n, err := s.q.CountEnabledAdmins(ctx)
	if err != nil {
		return err
	}
	if n <= 1 {
		return ErrLastAdmin
	}
	return nil
}

// ---- API Tokens ----

type CreateAPITokenInput struct {
	UserID    pgtype.UUID
	Name      string
	Role      string
	ExpiresAt *time.Time
}

type CreateAPITokenResult struct {
	PlaintextToken string
	Token          db.ApiToken
}

func (s *AuthService) CreateAPIToken(ctx context.Context, actor db.User, p CreateAPITokenInput) (CreateAPITokenResult, error) {
	if err := validateRole(p.Role); err != nil {
		return CreateAPITokenResult{}, err
	}
	owner, err := s.q.GetUserByID(ctx, p.UserID)
	if err != nil {
		return CreateAPITokenResult{}, ErrUserNotFound
	}
	if !roleAllows(owner.Role, p.Role) {
		return CreateAPITokenResult{}, ErrTokenRoleExceedsOwner
	}

	rawBytes := make([]byte, tokenRawBytes)
	if _, err := rand.Read(rawBytes); err != nil {
		return CreateAPITokenResult{}, err
	}
	plaintext := tokenPrefix + hex.EncodeToString(rawBytes)
	hash := sha256HexToken(plaintext)

	var expiresAt pgtype.Timestamptz
	if p.ExpiresAt != nil {
		expiresAt = pgtype.Timestamptz{Time: *p.ExpiresAt, Valid: true}
	}
	tok, err := s.q.CreateAPIToken(ctx, db.CreateAPITokenParams{
		UserID:    p.UserID,
		Name:      p.Name,
		TokenHash: hash,
		Role:      p.Role,
		ExpiresAt: expiresAt,
	})
	if err != nil {
		return CreateAPITokenResult{}, err
	}
	if s.audit != nil {
		s.audit.Log(ctx, AuditParams{
			Action:      "api_token.create",
			EntityType:  "api_token",
			EntityID:    tok.ID,
			ObjectType:  "api_token",
			ObjectID:    tok.ID,
			ActorType:   "user",
			ActorUserID: actor.ID,
		})
	}
	return CreateAPITokenResult{PlaintextToken: plaintext, Token: tok}, nil
}

func (s *AuthService) ListAPITokens(ctx context.Context) ([]db.ApiToken, error) {
	tokens, err := s.q.ListAPITokens(ctx)
	if err != nil {
		return nil, err
	}
	if tokens == nil {
		tokens = []db.ApiToken{}
	}
	return tokens, nil
}

func (s *AuthService) DeleteAPIToken(ctx context.Context, actor db.User, id pgtype.UUID) error {
	tok, err := s.q.GetAPITokenByID(ctx, id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return ErrTokenNotFound
		}
		return err
	}
	if err := s.q.DeleteAPIToken(ctx, id); err != nil {
		return err
	}
	if s.audit != nil {
		s.audit.Log(ctx, AuditParams{
			Action:      "api_token.delete",
			EntityType:  "api_token",
			EntityID:    tok.ID,
			ObjectType:  "api_token",
			ObjectID:    tok.ID,
			ActorType:   "user",
			ActorUserID: actor.ID,
		})
	}
	return nil
}

// ---- helpers ----

func validatePassword(p string) error {
	if len(p) < minPasswordLength {
		return ErrPasswordTooShort
	}
	return nil
}

func validateRole(r string) error {
	switch r {
	case RoleAdmin, RoleOperator, RoleViewer:
		return nil
	}
	return ErrInvalidRole
}

// roleAllows returns true if ownerRole can grant or equal grantRole.
func roleAllows(ownerRole, grantRole string) bool {
	rank := map[string]int{RoleViewer: 1, RoleOperator: 2, RoleAdmin: 3}
	return rank[ownerRole] >= rank[grantRole]
}

// sha256HexToken hashes a high-entropy token (session or API token) with SHA-256.
// This is safe because the plaintext tokens themselves are 32 bytes of CSPRNG output.
func sha256HexToken(plaintext string) string {
	sum := sha256.Sum256([]byte(plaintext))
	return hex.EncodeToString(sum[:])
}
