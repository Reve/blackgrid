package monitor

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"net/http/httptest"
	"testing"

	"blackgrid/internal/db"
)

func TestTLSChecker(t *testing.T) {
	// Create a test server with TLS
	ts := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer ts.Close()

	// The test server URL is like https://127.0.0.1:12345
	// We need to extract host and port
	addr := ts.Listener.Addr().String()
	host, portStr, _ := net.SplitHostPort(addr)
	var port int
	fmt.Sscanf(portStr, "%d", &port)

	c := &TLSChecker{}

	t.Run("valid certificate", func(t *testing.T) {
		verify := false
		cfg, _ := json.Marshal(TLSConfig{
			Host:      host,
			Port:      port,
			VerifyTLS: &verify,
		})
		m := db.Monitor{Target: addr, Config: cfg, TimeoutSeconds: 5}
		res := c.Check(context.Background(), m)

		if res.Status != "up" {
			t.Errorf("expected status up, got %s: %s", res.Status, res.ErrorMessage)
		}
		if res.Details["days_remaining"] == nil {
			t.Error("expected days_remaining in details")
		}
	})

	t.Run("invalid target", func(t *testing.T) {
		m := db.Monitor{Target: "invalid-host-that-does-not-exist:9999", TimeoutSeconds: 1}
		res := c.Check(context.Background(), m)
		if res.Status != "down" {
			t.Errorf("expected status down for invalid host, got %s", res.Status)
		}
	})
}

func TestResolveTLSTarget(t *testing.T) {
	tests := []struct {
		target string
		cfg    TLSConfig
		wHost  string
		wPort  int
	}{
		{"example.com", TLSConfig{}, "example.com", 443},
		{"example.com:8443", TLSConfig{}, "example.com", 8443},
		{"https://example.com/path", TLSConfig{}, "example.com", 443},
		{"", TLSConfig{Host: "custom.com", Port: 999}, "custom.com", 999},
	}

	for _, tt := range tests {
		h, p := resolveTLSTarget(tt.target, tt.cfg)
		if h != tt.wHost || p != tt.wPort {
			t.Errorf("resolveTLSTarget(%s) = %s:%d, want %s:%d", tt.target, h, p, tt.wHost, tt.wPort)
		}
	}
}
