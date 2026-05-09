package monitor

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"sort"
	"strings"
	"time"

	"blackgrid/internal/db"
)

// DNSConfig holds configuration for the DNS monitor type.
type DNSConfig struct {
	Resolver       string   `json:"resolver"`
	RecordType     string   `json:"record_type"`
	Hostname       string   `json:"hostname"`
	ExpectedValues []string `json:"expected_values"`
	MatchMode      string   `json:"match_mode"` // any, all, exact
}

// DNSChecker implements Checker for DNS monitors.
type DNSChecker struct{}

func (c *DNSChecker) Check(ctx context.Context, m db.Monitor) CheckResult {
	start := time.Now()

	var cfg DNSConfig
	if m.Config != nil {
		if err := json.Unmarshal(m.Config, &cfg); err != nil {
			return CheckResult{Status: "down", ErrorMessage: fmt.Sprintf("invalid config: %v", err)}
		}
	}

	hostname := cfg.Hostname
	if hostname == "" {
		hostname = m.Target
	}
	if hostname == "" {
		return CheckResult{Status: "down", ErrorMessage: "no hostname configured"}
	}

	recordType := strings.ToUpper(cfg.RecordType)
	if recordType == "" {
		recordType = "A"
	}

	matchMode := cfg.MatchMode
	if matchMode == "" {
		matchMode = "any"
	}

	// Build resolver
	resolver := buildResolver(cfg.Resolver)

	// Apply timeout from monitor config
	timeout := time.Duration(m.TimeoutSeconds) * time.Second
	if timeout <= 0 {
		timeout = 10 * time.Second
	}
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	resolverUsed := cfg.Resolver
	if resolverUsed == "" {
		resolverUsed = "system"
	}

	// Perform lookup based on record type
	var returnedValues []string
	var lookupErr error

	switch recordType {
	case "A":
		addrs, err := resolver.LookupHost(ctx, hostname)
		lookupErr = err
		if err == nil {
			for _, a := range addrs {
				ip := net.ParseIP(a)
				if ip != nil && ip.To4() != nil {
					returnedValues = append(returnedValues, a)
				}
			}
		}
	case "AAAA":
		addrs, err := resolver.LookupHost(ctx, hostname)
		lookupErr = err
		if err == nil {
			for _, a := range addrs {
				ip := net.ParseIP(a)
				if ip != nil && ip.To4() == nil && ip.To16() != nil {
					returnedValues = append(returnedValues, a)
				}
			}
		}
	case "CNAME":
		cname, err := resolver.LookupCNAME(ctx, hostname)
		lookupErr = err
		if err == nil {
			returnedValues = []string{strings.TrimSuffix(cname, ".")}
		}
	case "MX":
		mxs, err := resolver.LookupMX(ctx, hostname)
		lookupErr = err
		if err == nil {
			for _, mx := range mxs {
				returnedValues = append(returnedValues, strings.TrimSuffix(mx.Host, "."))
			}
		}
	case "TXT":
		txts, err := resolver.LookupTXT(ctx, hostname)
		lookupErr = err
		if err == nil {
			returnedValues = txts
		}
	case "NS":
		nss, err := resolver.LookupNS(ctx, hostname)
		lookupErr = err
		if err == nil {
			for _, ns := range nss {
				returnedValues = append(returnedValues, strings.TrimSuffix(ns.Host, "."))
			}
		}
	case "PTR":
		ptrs, err := resolver.LookupAddr(ctx, hostname)
		lookupErr = err
		if err == nil {
			for _, p := range ptrs {
				returnedValues = append(returnedValues, strings.TrimSuffix(p, "."))
			}
		}
	default:
		return CheckResult{Status: "down", ErrorMessage: fmt.Sprintf("unsupported record type: %s", recordType)}
	}

	latencyMs := int32(time.Since(start).Milliseconds())

	if lookupErr != nil {
		return CheckResult{
			Status:       "down",
			LatencyMs:    latencyMs,
			ErrorMessage: fmt.Sprintf("DNS lookup failed: %v", lookupErr),
			Details: map[string]any{
				"resolver":    resolverUsed,
				"record_type": recordType,
				"hostname":    hostname,
			},
		}
	}

	// Validate against expected values if provided
	if len(cfg.ExpectedValues) > 0 {
		ok, reason := matchValues(returnedValues, cfg.ExpectedValues, matchMode)
		if !ok {
			return CheckResult{
				Status:       "down",
				LatencyMs:    latencyMs,
				ErrorMessage: fmt.Sprintf("value match failed (%s): %s", matchMode, reason),
				Details: map[string]any{
					"resolver":        resolverUsed,
					"record_type":     recordType,
					"hostname":        hostname,
					"returned_values": returnedValues,
					"expected_values": cfg.ExpectedValues,
					"match_mode":      matchMode,
				},
			}
		}
	}

	return CheckResult{
		Status:    "up",
		LatencyMs: latencyMs,
		Details: map[string]any{
			"resolver":        resolverUsed,
			"record_type":     recordType,
			"hostname":        hostname,
			"returned_values": returnedValues,
		},
	}
}

// buildResolver constructs a net.Resolver using the given address.
// If addr is empty, the system resolver is used.
func buildResolver(addr string) *net.Resolver {
	if addr == "" {
		return net.DefaultResolver
	}
	// Ensure port is present
	if !strings.Contains(addr, ":") {
		addr = addr + ":53"
	}
	return &net.Resolver{
		PreferGo: true,
		Dial: func(ctx context.Context, network, _ string) (net.Conn, error) {
			d := net.Dialer{}
			return d.DialContext(ctx, "udp", addr)
		},
	}
}

// matchValues checks whether returned values satisfy the expected set given matchMode.
func matchValues(returned, expected []string, mode string) (bool, string) {
	retSet := make(map[string]bool)
	for _, v := range returned {
		retSet[strings.ToLower(v)] = true
	}

	switch mode {
	case "all":
		for _, e := range expected {
			if !retSet[strings.ToLower(e)] {
				return false, fmt.Sprintf("missing expected value: %s", e)
			}
		}
		return true, ""

	case "exact":
		expSet := make(map[string]bool)
		for _, e := range expected {
			expSet[strings.ToLower(e)] = true
		}
		// returned must equal expected as sets
		sortedRet := sortedLower(returned)
		sortedExp := sortedLower(expected)
		if strings.Join(sortedRet, ",") != strings.Join(sortedExp, ",") {
			return false, fmt.Sprintf("exact match failed: got %v, expected %v", returned, expected)
		}
		return true, ""

	default: // "any"
		for _, e := range expected {
			if retSet[strings.ToLower(e)] {
				return true, ""
			}
		}
		return false, fmt.Sprintf("none of expected values found: %v", expected)
	}
}

func sortedLower(ss []string) []string {
	out := make([]string, len(ss))
	for i, s := range ss {
		out[i] = strings.ToLower(s)
	}
	sort.Strings(out)
	return out
}
