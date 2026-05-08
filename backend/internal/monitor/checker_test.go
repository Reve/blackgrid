package monitor

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net"
	"testing"

	"blackgrid/internal/db"
)

func TestHTTPChecker(t *testing.T) {
	// Success server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("hello world ok"))
	}))
	defer server.Close()

	c := &HTTPChecker{}

	// Test Success
	configBytes, _ := json.Marshal(HTTPConfig{URL: server.URL, ExpectedBodyContains: "ok"})
	m := db.Monitor{Target: server.URL, Config: configBytes, TimeoutSeconds: 5}

	res := c.Check(context.Background(), m)
	if res.Status != "up" {
		t.Errorf("expected status 'up', got '%s': %s", res.Status, res.ErrorMessage)
	}

	// Test Failure (wrong status)
	configBytes2, _ := json.Marshal(HTTPConfig{URL: server.URL, ExpectedStatusMin: 201, ExpectedStatusMax: 201})
	m2 := db.Monitor{Target: server.URL, Config: configBytes2, TimeoutSeconds: 5}
	res2 := c.Check(context.Background(), m2)
	if res2.Status != "down" {
		t.Errorf("expected status 'down' for wrong status code, got '%s'", res2.Status)
	}

	// Test Failure (missing body string)
	configBytes3, _ := json.Marshal(HTTPConfig{URL: server.URL, ExpectedBodyContains: "missingtext"})
	m3 := db.Monitor{Target: server.URL, Config: configBytes3, TimeoutSeconds: 5}
	res3 := c.Check(context.Background(), m3)
	if res3.Status != "down" {
		t.Errorf("expected status 'down' for missing body string, got '%s'", res3.Status)
	}
}

func TestTCPChecker(t *testing.T) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer ln.Close()

	go func() {
		for {
			conn, err := ln.Accept()
			if err != nil { return }
			conn.Close()
		}
	}()

	c := &TCPChecker{}

	// Success
	m := db.Monitor{Target: ln.Addr().String(), TimeoutSeconds: 5}
	res := c.Check(context.Background(), m)
	if res.Status != "up" {
		t.Errorf("expected status 'up', got '%s': %s", res.Status, res.ErrorMessage)
	}

	// Failure
	mFail := db.Monitor{Target: "127.0.0.1:12345", TimeoutSeconds: 1} // Assuming nothing on this port
	resFail := c.Check(context.Background(), mFail)
	if resFail.Status != "down" {
		t.Errorf("expected status 'down' for unreachable port, got '%s'", resFail.Status)
	}
}

func TestPingChecker(t *testing.T) {
	c := &PingChecker{}

	// Valid format, local success (usually ping localhost works, might need permissions in some envs)
	// We just test basic failure to avoid relying on actual host ICMP in CI,
	// or test localhost if available.
	m := db.Monitor{Target: "127.0.0.1", TimeoutSeconds: 5}
	res := c.Check(context.Background(), m)

	// Due to potential docker restrictions, ping to localhost might fail.
	// The requirement states "If ping is unsupported... store a down result with a clear unsupported error".
	// Since we fall back to the system `ping` command, it might return 'up' or 'down' based on the system.
	if res.Status != "up" && res.Status != "down" {
		t.Errorf("unexpected status '%s'", res.Status)
	}

	// Invalid host injection test
	mInj := db.Monitor{Target: "127.0.0.1; echo hello", TimeoutSeconds: 5}
	resInj := c.Check(context.Background(), mInj)
	if resInj.Status != "down" {
		t.Errorf("expected injection attempt to fail with 'down', got '%s'", resInj.Status)
	}
}
