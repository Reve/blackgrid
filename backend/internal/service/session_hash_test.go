package service

import (
	"context"
	"testing"

	"blackgrid/internal/db"
)

// TestSession_StoredAsHashNotPlaintext verifies that the value persisted in
// sessions.session_hash is the SHA-256 hex of the cookie token, not the
// plaintext token itself. A DB leak must not yield usable session creds.
func TestSession_StoredAsHashNotPlaintext(t *testing.T) {
	pool := newTestPool(t)
	defer pool.Close()
	requireAuthSchema2(t, pool)

	q := db.New(pool)
	audit := NewAuditService(q, nil)
	svc := NewAuthService(q, AuthConfig{SessionTTLHours: 1}, audit, nil)
	// Ensure a clean state up-front so CreateFirstAdmin succeeds.
	pool.Exec(context.Background(), "DELETE FROM sessions")
	pool.Exec(context.Background(), "DELETE FROM api_tokens")
	pool.Exec(context.Background(), "DELETE FROM users")
	cleanupAuthData(t, pool)

	email := authTestEmail()
	_, _ = svc.CreateFirstAdmin(context.Background(), email, "A", "securepassword123")
	_, plaintext, err := svc.Login(context.Background(), email, "securepassword123")
	if err != nil {
		t.Fatalf("login: %v", err)
	}

	var stored string
	row := pool.QueryRow(context.Background(), `SELECT session_hash FROM sessions ORDER BY created_at DESC LIMIT 1`)
	if err := row.Scan(&stored); err != nil {
		t.Fatalf("read session row: %v", err)
	}
	if stored == plaintext {
		t.Fatal("session_hash column contains plaintext token; expected SHA-256 hex")
	}
	if want := sha256HexToken(plaintext); stored != want {
		t.Fatalf("session_hash mismatch: got %q want %q", stored, want)
	}

	// Plaintext must still resolve.
	if _, _, err := svc.ResolveSession(context.Background(), plaintext); err != nil {
		t.Fatalf("ResolveSession with plaintext failed: %v", err)
	}

	// The stored hash must NOT resolve as if it were plaintext.
	if _, _, err := svc.ResolveSession(context.Background(), stored); err == nil {
		t.Fatal("ResolveSession accepted the at-rest hash as a plaintext token; this would defeat hashing")
	}
}
