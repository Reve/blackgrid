package monitor

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"os/exec"
	"regexp"
	"strings"
	"time"

	"blackgrid/internal/db"
)

type CheckResult struct {
	Status       string
	LatencyMs    int32
	ErrorMessage string
}

type Checker interface {
	Check(ctx context.Context, monitor db.Monitor) CheckResult
}

// HTTP Checker
type HTTPChecker struct{}

type HTTPConfig struct {
	URL                  string            `json:"url"`
	Method               string            `json:"method"`
	ExpectedStatusMin    int               `json:"expected_status_min"`
	ExpectedStatusMax    int               `json:"expected_status_max"`
	ExpectedBodyContains string            `json:"expected_body_contains"`
	Headers              map[string]string `json:"headers"`
	FollowRedirects      *bool             `json:"follow_redirects"`
	VerifyTLS            *bool             `json:"verify_tls"`
}

func (c *HTTPChecker) Check(ctx context.Context, monitor db.Monitor) CheckResult {
	start := time.Now()
	var config HTTPConfig
	if monitor.Config != nil {
		if err := json.Unmarshal(monitor.Config, &config); err != nil {
			return CheckResult{Status: "down", ErrorMessage: fmt.Sprintf("invalid config: %v", err)}
		}
	}

	targetURL := config.URL
	if targetURL == "" {
		targetURL = monitor.Target
	}

	method := config.Method
	if method == "" {
		method = "GET"
	}

	statusMin := config.ExpectedStatusMin
	if statusMin == 0 {
		statusMin = 200
	}
	statusMax := config.ExpectedStatusMax
	if statusMax == 0 {
		statusMax = 299
	}

	verifyTLS := true
	if config.VerifyTLS != nil {
		verifyTLS = *config.VerifyTLS
	}

	followRedirects := true
	if config.FollowRedirects != nil {
		followRedirects = *config.FollowRedirects
	}

	req, err := http.NewRequestWithContext(ctx, method, targetURL, nil)
	if err != nil {
		return CheckResult{Status: "down", ErrorMessage: fmt.Sprintf("failed to create request: %v", err)}
	}

	for k, v := range config.Headers {
		req.Header.Set(k, v)
	}
	if req.Header.Get("User-Agent") == "" {
		req.Header.Set("User-Agent", "Blackgrid/1.0")
	}

	transport := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: !verifyTLS},
	}

	client := &http.Client{
		Transport: transport,
		Timeout:   time.Duration(monitor.TimeoutSeconds) * time.Second,
	}

	if !followRedirects {
		client.CheckRedirect = func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse
		}
	}

	resp, err := client.Do(req)
	latencyMs := int32(time.Since(start).Milliseconds())
	if err != nil {
		return CheckResult{Status: "down", LatencyMs: latencyMs, ErrorMessage: fmt.Sprintf("request failed: %v", err)}
	}
	defer resp.Body.Close()

	if resp.StatusCode < statusMin || resp.StatusCode > statusMax {
		return CheckResult{Status: "down", LatencyMs: latencyMs, ErrorMessage: fmt.Sprintf("unexpected status code: %d", resp.StatusCode)}
	}

	if config.ExpectedBodyContains != "" {
		bodyBytes, err := io.ReadAll(resp.Body)
		if err != nil {
			return CheckResult{Status: "down", LatencyMs: latencyMs, ErrorMessage: fmt.Sprintf("failed to read body: %v", err)}
		}
		if !strings.Contains(string(bodyBytes), config.ExpectedBodyContains) {
			return CheckResult{Status: "down", LatencyMs: latencyMs, ErrorMessage: "body did not contain expected text"}
		}
	}

	return CheckResult{Status: "up", LatencyMs: latencyMs}
}

// TCP Checker
type TCPChecker struct{}

type TCPConfig struct {
	Host string `json:"host"`
	Port int    `json:"port"`
}

func (c *TCPChecker) Check(ctx context.Context, monitor db.Monitor) CheckResult {
	start := time.Now()
	var config TCPConfig
	if monitor.Config != nil {
		if err := json.Unmarshal(monitor.Config, &config); err != nil {
			return CheckResult{Status: "down", ErrorMessage: fmt.Sprintf("invalid config: %v", err)}
		}
	}

	target := monitor.Target
	if config.Host != "" && config.Port != 0 {
		target = fmt.Sprintf("%s:%d", config.Host, config.Port)
	}

	timeout := time.Duration(monitor.TimeoutSeconds) * time.Second
	dialer := net.Dialer{Timeout: timeout}

	conn, err := dialer.DialContext(ctx, "tcp", target)
	latencyMs := int32(time.Since(start).Milliseconds())

	if err != nil {
		return CheckResult{Status: "down", LatencyMs: latencyMs, ErrorMessage: fmt.Sprintf("connection failed: %v", err)}
	}
	conn.Close()

	return CheckResult{Status: "up", LatencyMs: latencyMs}
}

// Ping Checker
type PingChecker struct{}

type PingConfig struct {
	Host        string `json:"host"`
	PacketCount int    `json:"packet_count"`
}

func (c *PingChecker) Check(ctx context.Context, monitor db.Monitor) CheckResult {
	start := time.Now()
	var config PingConfig
	if monitor.Config != nil {
		if err := json.Unmarshal(monitor.Config, &config); err != nil {
			return CheckResult{Status: "down", ErrorMessage: fmt.Sprintf("invalid config: %v", err)}
		}
	}

	host := config.Host
	if host == "" {
		host = monitor.Target
	}

	// Basic validation to prevent command injection
	matched, _ := regexp.MatchString(`^[a-zA-Z0-9.-]+$`, host)
	if !matched {
		return CheckResult{Status: "down", ErrorMessage: "invalid host format"}
	}

	count := config.PacketCount
	if count <= 0 {
		count = 3
	}

	timeout := monitor.TimeoutSeconds
	if timeout <= 0 {
		timeout = 10
	}

	// Use system ping as fallback
	ctx, cancel := context.WithTimeout(ctx, time.Duration(timeout)*time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctx, "ping", "-c", fmt.Sprintf("%d", count), "-W", "1", host)
	err := cmd.Run()
	latencyMs := int32(time.Since(start).Milliseconds())

	if err != nil {
		return CheckResult{Status: "down", LatencyMs: latencyMs, ErrorMessage: fmt.Sprintf("ping failed: %v", err)}
	}

	return CheckResult{Status: "up", LatencyMs: latencyMs}
}
