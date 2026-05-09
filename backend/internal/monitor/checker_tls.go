package monitor

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"net"
	"net/url"
	"strings"
	"time"

	"blackgrid/internal/db"
)

// TLSConfig holds configuration for the TLS certificate monitor.
type TLSConfig struct {
	Host         string `json:"host"`
	Port         int    `json:"port"`
	ServerName   string `json:"server_name"`
	WarningDays  int    `json:"warning_days"`
	CriticalDays int    `json:"critical_days"`
	VerifyTLS    *bool  `json:"verify_tls"`
}

// TLSChecker implements Checker for TLS certificate monitors.
type TLSChecker struct{}

func (c *TLSChecker) Check(ctx context.Context, m db.Monitor) CheckResult {
	start := time.Now()

	var cfg TLSConfig
	if m.Config != nil {
		if err := json.Unmarshal(m.Config, &cfg); err != nil {
			return CheckResult{Status: "down", ErrorMessage: fmt.Sprintf("invalid config: %v", err)}
		}
	}

	// Resolve host and port
	host, port := resolveTLSTarget(m.Target, cfg)

	serverName := cfg.ServerName
	if serverName == "" {
		serverName = host
	}

	warningDays := cfg.WarningDays
	if warningDays <= 0 {
		warningDays = 30
	}
	criticalDays := cfg.CriticalDays
	if criticalDays <= 0 {
		criticalDays = 7
	}

	verifyTLS := true
	if cfg.VerifyTLS != nil {
		verifyTLS = *cfg.VerifyTLS
	}

	timeout := time.Duration(m.TimeoutSeconds) * time.Second
	if timeout <= 0 {
		timeout = 10 * time.Second
	}

	dialer := &net.Dialer{Timeout: timeout}
	connCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	addr := fmt.Sprintf("%s:%d", host, port)
	rawConn, err := dialer.DialContext(connCtx, "tcp", addr)
	if err != nil {
		return CheckResult{
			Status:       "down",
			LatencyMs:    int32(time.Since(start).Milliseconds()),
			ErrorMessage: fmt.Sprintf("connection failed: %v", err),
		}
	}
	defer rawConn.Close()

	tlsCfg := &tls.Config{
		ServerName:         serverName,
		InsecureSkipVerify: !verifyTLS, //nolint:gosec
	}
	tlsConn := tls.Client(rawConn, tlsCfg)
	if err := tlsConn.HandshakeContext(connCtx); err != nil {
		return CheckResult{
			Status:       "down",
			LatencyMs:    int32(time.Since(start).Milliseconds()),
			ErrorMessage: fmt.Sprintf("TLS handshake failed: %v", err),
		}
	}
	latencyMs := int32(time.Since(start).Milliseconds())

	// Inspect certificate
	certs := tlsConn.ConnectionState().PeerCertificates
	if len(certs) == 0 {
		return CheckResult{
			Status:       "down",
			LatencyMs:    latencyMs,
			ErrorMessage: "no certificates returned",
		}
	}

	cert := certs[0]
	now := time.Now()
	daysRemaining := int(cert.NotAfter.Sub(now).Hours() / 24)

	details := map[string]any{
		"subject":        cert.Subject.CommonName,
		"issuer":         cert.Issuer.CommonName,
		"not_before":     cert.NotBefore.Format(time.RFC3339),
		"not_after":      cert.NotAfter.Format(time.RFC3339),
		"days_remaining": daysRemaining,
		"dns_names":      cert.DNSNames,
		"verify_tls":     verifyTLS,
	}

	if now.After(cert.NotAfter) {
		return CheckResult{
			Status:       "down",
			LatencyMs:    latencyMs,
			ErrorMessage: "certificate is expired",
			Details:      details,
		}
	}
	if daysRemaining <= criticalDays {
		return CheckResult{
			Status:       "down",
			LatencyMs:    latencyMs,
			ErrorMessage: fmt.Sprintf("certificate expires in %d days (critical threshold: %d)", daysRemaining, criticalDays),
			Details:      details,
		}
	}
	if daysRemaining <= warningDays {
		return CheckResult{
			Status:       "degraded",
			LatencyMs:    latencyMs,
			ErrorMessage: fmt.Sprintf("certificate expires in %d days (warning threshold: %d)", daysRemaining, warningDays),
			Details:      details,
		}
	}

	return CheckResult{
		Status:    "up",
		LatencyMs: latencyMs,
		Details:   details,
	}
}

// resolveTLSTarget parses the monitor target and config to determine host and port.
func resolveTLSTarget(target string, cfg TLSConfig) (string, int) {
	host := cfg.Host
	port := cfg.Port

	if host == "" {
		// Try to parse target as URL or host[:port]
		if strings.HasPrefix(target, "http://") || strings.HasPrefix(target, "https://") {
			if u, err := url.Parse(target); err == nil {
				h := u.Hostname()
				p := u.Port()
				if h != "" {
					host = h
				}
				if p != "" {
					fmt.Sscanf(p, "%d", &port)
				}
			}
		} else if strings.Contains(target, ":") {
			h, p, err := net.SplitHostPort(target)
			if err == nil {
				host = h
				fmt.Sscanf(p, "%d", &port)
			}
		} else {
			host = target
		}
	}

	if port <= 0 {
		port = 443
	}

	return host, port
}
