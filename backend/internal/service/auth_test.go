package service

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
	"time"

	"blackgrid/internal/db"

	"github.com/jackc/pgx/v5/pgxpool"
)

func cleanupAuthData(t *testing.T, pool *pgxpool.Pool) {
	t.Helper()
	t.Cleanup(func() {
		ctx := context.Background()
		pool.Exec(ctx, "DELETE FROM sessions")
		pool.Exec(ctx, "DELETE FROM api_tokens")
		pool.Exec(ctx, "DELETE FROM users")
	})
}

func requireAuthSchema2(t *testing.T, pool *pgxpool.Pool) {
	t.Helper()
	var exists bool
	err := pool.QueryRow(context.Background(),
		`SELECT EXISTS (SELECT 1 FROM information_schema.tables WHERE table_name='users')`).
		Scan(&exists)
	if err != nil || !exists {
		t.Skip("Skipping: users table not present (run migration 005)")
	}
}

// ---- Setup tests ----

func TestSetup_RequiredWhenNoUsers(t *testing.T) {
	pool := newTestPool(t)
	defer pool.Close()
	requireAuthSchema2(t, pool)

	q := db.New(pool)
	audit := NewAuditService(q)
	svc := NewAuthService(q, AuthConfig{SessionTTLHours: 1}, audit)
	cleanupAuthData(t, pool)

	required, err := svc.SetupRequired(context.Background())
	if err != nil {
		t.Fatalf("SetupRequired: %v", err)
	}
	if !required {
		t.Fatal("expected setup_required=true when no users")
	}
}

func TestSetup_NotRequiredWhenUserExists(t *testing.T) {
	pool := newTestPool(t)
	defer pool.Close()
	requireAuthSchema2(t, pool)

	q := db.New(pool)
	audit := NewAuditService(q)
	svc := NewAuthService(q, AuthConfig{SessionTTLHours: 1}, audit)
	cleanupAuthData(t, pool)

	_, err := svc.CreateFirstAdmin(context.Background(), authTestEmail(), "Admin", "securepassword123")
	if err != nil {
		t.Fatalf("CreateFirstAdmin: %v", err)
	}
	required, _ := svc.SetupRequired(context.Background())
	if required {
		t.Fatal("expected setup_required=false when user exists")
	}
}

func TestSetup_FirstAdminCreation(t *testing.T) {
	pool := newTestPool(t)
	defer pool.Close()
	requireAuthSchema2(t, pool)

	q := db.New(pool)
	audit := NewAuditService(q)
	svc := NewAuthService(q, AuthConfig{SessionTTLHours: 1}, audit)
	cleanupAuthData(t, pool)

	user, err := svc.CreateFirstAdmin(context.Background(), "admin@setup.local", "Admin", "securepassword123")
	if err != nil {
		t.Fatalf("CreateFirstAdmin: %v", err)
	}
	if user.Role != RoleAdmin {
		t.Errorf("expected role=admin, got %s", user.Role)
	}
	if user.PasswordHash == "" || user.PasswordHash == "securepassword123" {
		t.Error("password hash must be set and not be plaintext")
	}
}

func TestSetup_FirstAdminBlockedAfterUserExists(t *testing.T) {
	pool := newTestPool(t)
	defer pool.Close()
	requireAuthSchema2(t, pool)

	q := db.New(pool)
	audit := NewAuditService(q)
	svc := NewAuthService(q, AuthConfig{SessionTTLHours: 1}, audit)
	cleanupAuthData(t, pool)

	_, _ = svc.CreateFirstAdmin(context.Background(), authTestEmail(), "Admin", "securepassword123")
	_, err := svc.CreateFirstAdmin(context.Background(), "other@setup.local", "Other", "securepassword456")
	if err != ErrSetupAlreadyDone {
		t.Fatalf("expected ErrSetupAlreadyDone, got %v", err)
	}
}

// ---- Auth tests ----

func TestAuth_LoginSuccess(t *testing.T) {
	pool := newTestPool(t)
	defer pool.Close()
	requireAuthSchema2(t, pool)

	q := db.New(pool)
	audit := NewAuditService(q)
	svc := NewAuthService(q, AuthConfig{SessionTTLHours: 24}, audit)
	cleanupAuthData(t, pool)

	email := authTestEmail()
	_, err := svc.CreateFirstAdmin(context.Background(), email, "Admin", "securepassword123")
	if err != nil {
		t.Fatalf("setup: %v", err)
	}

	user, hash, err := svc.Login(context.Background(), email, "securepassword123")
	if err != nil {
		t.Fatalf("login: %v", err)
	}
	if hash == "" {
		t.Fatal("expected non-empty session hash")
	}
	if user.Email != email {
		t.Errorf("email mismatch: got %s, want %s", user.Email, email)
	}

	resolved, _, err := svc.ResolveSession(context.Background(), hash)
	if err != nil {
		t.Fatalf("resolve session: %v", err)
	}
	if resolved.ID != user.ID {
		t.Error("resolved user ID mismatch")
	}
}

func TestAuth_LoginFailureDoesNotRevealUserExistence(t *testing.T) {
	pool := newTestPool(t)
	defer pool.Close()
	requireAuthSchema2(t, pool)

	q := db.New(pool)
	audit := NewAuditService(q)
	svc := NewAuthService(q, AuthConfig{SessionTTLHours: 1}, audit)
	cleanupAuthData(t, pool)

	// Wrong email — user doesn't exist
	_, _, err1 := svc.Login(context.Background(), "nonexistent@x.local", "securepassword123")

	// Wrong password — user exists
	email := authTestEmail()
	_, _ = svc.CreateFirstAdmin(context.Background(), email, "A", "securepassword123")
	_, _, err2 := svc.Login(context.Background(), email, "wrongpassword!")

	if err1 != ErrInvalidCredentials {
		t.Errorf("expected ErrInvalidCredentials for nonexistent email, got %v", err1)
	}
	if err2 != ErrInvalidCredentials {
		t.Errorf("expected ErrInvalidCredentials for wrong password, got %v", err2)
	}
}

func TestAuth_DisabledUserCannotLogin(t *testing.T) {
	pool := newTestPool(t)
	defer pool.Close()
	requireAuthSchema2(t, pool)

	q := db.New(pool)
	audit := NewAuditService(q)
	svc := NewAuthService(q, AuthConfig{SessionTTLHours: 1}, audit)
	cleanupAuthData(t, pool)

	email := authTestEmail()
	user, _ := svc.CreateFirstAdmin(context.Background(), email, "A", "securepassword123")

	// Create a second admin so we can disable the first without hitting lastAdmin guard.
	_, _ = svc.CreateUser(context.Background(), user, CreateUserInput{
		Email: authTestEmail(), DisplayName: "A2", Password: "securepassword456",
		Role: RoleAdmin, Enabled: true,
	})
	_, _ = svc.UpdateUser(context.Background(), user, user.ID, UpdateUserInput{
		DisplayName: user.DisplayName, Role: RoleAdmin, Enabled: false,
	})

	_, _, err := svc.Login(context.Background(), email, "securepassword123")
	if err != ErrInvalidCredentials && err != ErrUserDisabled {
		t.Errorf("expected credentials/disabled error for disabled user, got %v", err)
	}
}

func TestAuth_LogoutDeletesSession(t *testing.T) {
	pool := newTestPool(t)
	defer pool.Close()
	requireAuthSchema2(t, pool)

	q := db.New(pool)
	audit := NewAuditService(q)
	svc := NewAuthService(q, AuthConfig{SessionTTLHours: 1}, audit)
	cleanupAuthData(t, pool)

	email := authTestEmail()
	_, _ = svc.CreateFirstAdmin(context.Background(), email, "A", "securepassword123")
	_, hash, _ := svc.Login(context.Background(), email, "securepassword123")

	if err := svc.Logout(context.Background(), hash); err != nil {
		t.Fatalf("logout: %v", err)
	}
	_, _, err := svc.ResolveSession(context.Background(), hash)
	if err == nil {
		t.Fatal("expected session to be invalid after logout")
	}
}

func TestAuth_PasswordHashNotReturnedInJSON(t *testing.T) {
	pool := newTestPool(t)
	defer pool.Close()
	requireAuthSchema2(t, pool)

	q := db.New(pool)
	audit := NewAuditService(q)
	svc := NewAuthService(q, AuthConfig{SessionTTLHours: 1}, audit)
	cleanupAuthData(t, pool)

	user, _ := svc.CreateFirstAdmin(context.Background(), authTestEmail(), "A", "securepassword123")

	b, err := json.Marshal(user)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	if strings.Contains(string(b), "password") {
		t.Errorf("password leaked in JSON: %s", string(b))
	}
}

// ---- User safety tests ----

func TestUser_CannotDeleteLastEnabledAdmin(t *testing.T) {
	pool := newTestPool(t)
	defer pool.Close()
	requireAuthSchema2(t, pool)

	q := db.New(pool)
	audit := NewAuditService(q)
	svc := NewAuthService(q, AuthConfig{SessionTTLHours: 1}, audit)
	cleanupAuthData(t, pool)

	admin, _ := svc.CreateFirstAdmin(context.Background(), authTestEmail(), "Admin", "securepassword123")

	err := svc.DeleteUser(context.Background(), admin, admin.ID)
	if err != ErrLastAdmin {
		t.Errorf("expected ErrLastAdmin, got %v", err)
	}
}

func TestUser_CannotDisableLastEnabledAdmin(t *testing.T) {
	pool := newTestPool(t)
	defer pool.Close()
	requireAuthSchema2(t, pool)

	q := db.New(pool)
	audit := NewAuditService(q)
	svc := NewAuthService(q, AuthConfig{SessionTTLHours: 1}, audit)
	cleanupAuthData(t, pool)

	admin, _ := svc.CreateFirstAdmin(context.Background(), authTestEmail(), "Admin", "securepassword123")
	_, err := svc.UpdateUser(context.Background(), admin, admin.ID, UpdateUserInput{
		DisplayName: "Admin", Role: RoleAdmin, Enabled: false,
	})
	if err != ErrLastAdmin {
		t.Errorf("expected ErrLastAdmin, got %v", err)
	}
}

func TestUser_CannotDemoteLastEnabledAdmin(t *testing.T) {
	pool := newTestPool(t)
	defer pool.Close()
	requireAuthSchema2(t, pool)

	q := db.New(pool)
	audit := NewAuditService(q)
	svc := NewAuthService(q, AuthConfig{SessionTTLHours: 1}, audit)
	cleanupAuthData(t, pool)

	admin, _ := svc.CreateFirstAdmin(context.Background(), authTestEmail(), "Admin", "securepassword123")
	_, err := svc.UpdateUser(context.Background(), admin, admin.ID, UpdateUserInput{
		DisplayName: "Admin", Role: RoleOperator, Enabled: true,
	})
	if err != ErrLastAdmin {
		t.Errorf("expected ErrLastAdmin when demoting last admin, got %v", err)
	}
}

// ---- API token tests ----

func TestAPIToken_CreateAndResolve(t *testing.T) {
	pool := newTestPool(t)
	defer pool.Close()
	requireAuthSchema2(t, pool)

	q := db.New(pool)
	audit := NewAuditService(q)
	svc := NewAuthService(q, AuthConfig{SessionTTLHours: 1}, audit)
	cleanupAuthData(t, pool)

	admin, _ := svc.CreateFirstAdmin(context.Background(), authTestEmail(), "Admin", "securepassword123")
	result, err := svc.CreateAPIToken(context.Background(), admin, CreateAPITokenInput{
		UserID: admin.ID, Name: "test token", Role: RoleOperator,
	})
	if err != nil {
		t.Fatalf("create token: %v", err)
	}
	if !strings.HasPrefix(result.PlaintextToken, tokenPrefix) {
		t.Errorf("token must start with %q, got %s", tokenPrefix, result.PlaintextToken)
	}
	if result.Token.TokenHash == result.PlaintextToken {
		t.Error("token hash must not equal plaintext")
	}

	user, tok, err := svc.ResolveAPIToken(context.Background(), result.PlaintextToken)
	if err != nil {
		t.Fatalf("resolve token: %v", err)
	}
	if user.ID != admin.ID {
		t.Error("user id mismatch")
	}
	if tok.Role != RoleOperator {
		t.Errorf("expected operator role, got %s", tok.Role)
	}
}

func TestAPIToken_ExpiredTokenRejected(t *testing.T) {
	pool := newTestPool(t)
	defer pool.Close()
	requireAuthSchema2(t, pool)

	q := db.New(pool)
	audit := NewAuditService(q)
	svc := NewAuthService(q, AuthConfig{SessionTTLHours: 1}, audit)
	cleanupAuthData(t, pool)

	admin, _ := svc.CreateFirstAdmin(context.Background(), authTestEmail(), "Admin", "securepassword123")
	past := time.Now().Add(-1 * time.Hour)
	result, _ := svc.CreateAPIToken(context.Background(), admin, CreateAPITokenInput{
		UserID: admin.ID, Name: "expired", Role: RoleViewer, ExpiresAt: &past,
	})

	_, _, err := svc.ResolveAPIToken(context.Background(), result.PlaintextToken)
	if err != ErrTokenExpired {
		t.Errorf("expected ErrTokenExpired, got %v", err)
	}
}

func TestAPIToken_DeleteInvalidates(t *testing.T) {
	pool := newTestPool(t)
	defer pool.Close()
	requireAuthSchema2(t, pool)

	q := db.New(pool)
	audit := NewAuditService(q)
	svc := NewAuthService(q, AuthConfig{SessionTTLHours: 1}, audit)
	cleanupAuthData(t, pool)

	admin, _ := svc.CreateFirstAdmin(context.Background(), authTestEmail(), "Admin", "securepassword123")
	result, _ := svc.CreateAPIToken(context.Background(), admin, CreateAPITokenInput{
		UserID: admin.ID, Name: "del", Role: RoleViewer,
	})
	if err := svc.DeleteAPIToken(context.Background(), admin, result.Token.ID); err != nil {
		t.Fatalf("delete: %v", err)
	}
	_, _, err := svc.ResolveAPIToken(context.Background(), result.PlaintextToken)
	if err == nil {
		t.Fatal("expected error after token deletion")
	}
}

func TestAPIToken_TokenRoleCannotExceedOwner(t *testing.T) {
	pool := newTestPool(t)
	defer pool.Close()
	requireAuthSchema2(t, pool)

	q := db.New(pool)
	audit := NewAuditService(q)
	svc := NewAuthService(q, AuthConfig{SessionTTLHours: 1}, audit)
	cleanupAuthData(t, pool)

	admin, _ := svc.CreateFirstAdmin(context.Background(), authTestEmail(), "Admin", "securepassword123")
	op, _ := svc.CreateUser(context.Background(), admin, CreateUserInput{
		Email: authTestEmail(), DisplayName: "Op", Password: "securepassword123",
		Role: RoleOperator, Enabled: true,
	})
	_, err := svc.CreateAPIToken(context.Background(), op, CreateAPITokenInput{
		UserID: op.ID, Name: "t", Role: RoleAdmin,
	})
	if err != ErrTokenRoleExceedsOwner {
		t.Errorf("expected ErrTokenRoleExceedsOwner, got %v", err)
	}
}

// ---- Audit tests ----

func TestAudit_UserCreateWritesLog(t *testing.T) {
	pool := newTestPool(t)
	defer pool.Close()
	requireAuthSchema2(t, pool)

	q := db.New(pool)
	audit := NewAuditService(q)
	svc := NewAuthService(q, AuthConfig{SessionTTLHours: 1}, audit)
	cleanupAuthData(t, pool)

	_, _ = svc.CreateFirstAdmin(context.Background(), authTestEmail(), "Admin", "securepassword123")
	time.Sleep(300 * time.Millisecond)

	entries, err := audit.List(context.Background(), db.ListAuditLogsParams{Limit: 20})
	if err != nil {
		t.Fatalf("list audit: %v", err)
	}
	found := false
	for _, e := range entries {
		if e.Action == "user.create" {
			found = true
			break
		}
	}
	if !found {
		t.Error("expected user.create audit log entry")
	}
}

func TestAudit_DoesNotContainPasswordOrSecret(t *testing.T) {
	pool := newTestPool(t)
	defer pool.Close()
	requireAuthSchema2(t, pool)

	q := db.New(pool)
	audit := NewAuditService(q)
	svc := NewAuthService(q, AuthConfig{SessionTTLHours: 1}, audit)
	cleanupAuthData(t, pool)

	admin, _ := svc.CreateFirstAdmin(context.Background(), authTestEmail(), "Admin", "securepassword123")
	_, _ = svc.CreateAPIToken(context.Background(), admin, CreateAPITokenInput{
		UserID: admin.ID, Name: "t", Role: RoleViewer,
	})
	time.Sleep(300 * time.Millisecond)

	entries, _ := audit.List(context.Background(), db.ListAuditLogsParams{Limit: 50})
	for _, e := range entries {
		raw := string(e.AfterState) + string(e.BeforeState) + string(e.Changes)
		if strings.Contains(raw, "securepassword123") {
			t.Errorf("audit log entry %s contains password", e.Action)
		}
	}
}

// ---- Secret masking tests ----

func TestNotificationMasking_SMTPPasswordMasked(t *testing.T) {
	pool := newTestPool(t)
	defer pool.Close()
	requireSchema(t, pool)

	q := db.New(pool)
	svc := NewNotificationService(q)

	ch, err := svc.CreateChannel(context.Background(), "smtp-mask-test", ChannelTypeSMTP, true,
		[]byte(`{"host":"smtp.test","port":587,"from":"a@b.com","to":["x@y.com"],"password":"supersecret","use_tls":false}`))
	if err != nil {
		t.Fatalf("create smtp channel: %v", err)
	}
	t.Cleanup(func() { svc.DeleteChannel(context.Background(), ch.ID) })

	masked := MaskConfig(ch)
	if strings.Contains(string(masked), "supersecret") {
		t.Error("SMTP password not masked in channel config")
	}
	if !strings.Contains(string(masked), maskedSecret) {
		t.Error("expected masked secret placeholder in config")
	}
}

func TestWebhookHeadersMasked(t *testing.T) {
	pool := newTestPool(t)
	defer pool.Close()
	requireSchema(t, pool)

	q := db.New(pool)
	svc := NewNotificationService(q)

	ch, err := svc.CreateChannel(context.Background(), "wh-mask-test", ChannelTypeWebhook, true,
		[]byte(`{"url":"http://example.com","headers":{"Authorization":"Bearer secret123","X-Custom":"visible"}}`))
	if err != nil {
		t.Fatalf("create webhook channel: %v", err)
	}
	t.Cleanup(func() { svc.DeleteChannel(context.Background(), ch.ID) })

	masked := MaskWebhookHeaders(ch.Config)
	if strings.Contains(string(masked), "secret123") {
		t.Error("webhook Authorization header should be masked")
	}
}

func TestPublicStatusPage_AccessibleWithoutAuth(t *testing.T) {
	pool := newTestPool(t)
	defer pool.Close()
	requireSchema(t, pool)

	q := db.New(pool)
	svc := NewStatusPageService(q)
	trueBool := true
	slug := "pub-" + strings.ReplaceAll(time.Now().Format("150405.000"), ".", "")
	page, err := svc.CreateStatusPage(context.Background(), StatusPageInput{
		Name:   "Public Test",
		Slug:   slug,
		Public: &trueBool,
	})
	if err != nil {
		t.Fatalf("create page: %v", err)
	}
	t.Cleanup(func() { svc.DeleteStatusPage(context.Background(), page.ID) })

	_, err = svc.GetPublicStatusPage(context.Background(), page.Slug)
	if err != nil {
		t.Errorf("GetPublicStatusPage failed (should be accessible): %v", err)
	}
}

// ---- helpers ----

func authTestEmail() string {
	return "auth" + strings.ReplaceAll(time.Now().Format("150405.000000"), ".", "") + "@test.local"
}
