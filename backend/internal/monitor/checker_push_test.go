package monitor

import (
	"context"
	"testing"
	"time"

	"blackgrid/internal/db"
	"github.com/jackc/pgx/v5/pgtype"
)

func TestPushChecker(t *testing.T) {
	c := &PushChecker{}

	t.Run("never checked", func(t *testing.T) {
		m := db.Monitor{
			LastHeartbeatAt: pgtype.Timestamptz{Valid: false},
		}
		res := c.Check(context.Background(), m)
		if res.Status != "down" {
			t.Errorf("expected status down for never checked, got %s", res.Status)
		}
	})

	t.Run("within grace period", func(t *testing.T) {
		m := db.Monitor{
			LastHeartbeatAt: pgtype.Timestamptz{Time: time.Now().Add(-10 * time.Second), Valid: true},
		}
		// default grace is 120s
		res := c.Check(context.Background(), m)
		if res.Status != "up" {
			t.Errorf("expected status up within grace period, got %s", res.Status)
		}
	})

	t.Run("overdue", func(t *testing.T) {
		m := db.Monitor{
			LastHeartbeatAt: pgtype.Timestamptz{Time: time.Now().Add(-300 * time.Second), Valid: true},
		}
		res := c.Check(context.Background(), m)
		if res.Status != "down" || res.ErrorMessage != "heartbeat overdue" {
			t.Errorf("expected heartbeat overdue error, got %s: %s", res.Status, res.ErrorMessage)
		}
	})
}

func TestPushTokenHashing(t *testing.T) {
	token, err := GeneratePushToken()
	if err != nil {
		t.Fatal(err)
	}
	if len(token) == 0 {
		t.Error("generated empty token")
	}

	hash := HashToken(token)
	if hash != HashToken(token) {
		t.Error("hashing is not deterministic")
	}
}
