package service

import (
	"context"
	"errors"
	"net"
	"net/netip"
	"strconv"
	"testing"
	"time"

	"blackgrid/internal/db"
)

func TestValidateScanInterval(t *testing.T) {
	if err := ValidateScanInterval(60); err != nil {
		t.Fatalf("60s should be valid: %v", err)
	}
	if err := ValidateScanInterval(3600); err != nil {
		t.Fatalf("3600s should be valid: %v", err)
	}
	if err := ValidateScanInterval(59); !errors.Is(err, ErrInvalidInterval) {
		t.Fatalf("59s should be invalid, got %v", err)
	}
}

func TestValidatePrefixForScan(t *testing.T) {
	cases := []struct {
		name string
		cidr string
		max  int
		want error
	}{
		{"valid /24", "192.168.1.0/24", 22, nil},
		{"valid /22 boundary", "10.0.0.0/22", 22, nil},
		{"too large /16", "10.0.0.0/16", 22, ErrPrefixTooLarge},
		{"too large /8", "10.0.0.0/8", 22, ErrPrefixTooLarge},
		{"ipv6 rejected", "2001:db8::/64", 22, ErrIPv6Unsupported},
		{"invalid cidr", "not-a-cidr", 22, ErrInvalidCIDR},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := ValidatePrefixForScan(tc.cidr, tc.max)
			if !errors.Is(got, tc.want) {
				t.Fatalf("want %v got %v", tc.want, got)
			}
		})
	}
}

func TestEnumerateIPv4Hosts(t *testing.T) {
	hosts, err := EnumerateIPv4Hosts("192.168.1.0/30")
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	// /30 has 4 addresses: .0 network, .1, .2 hosts, .3 broadcast
	if len(hosts) != 2 {
		t.Fatalf("want 2 hosts, got %d (%v)", len(hosts), hosts)
	}
	if hosts[0].String() != "192.168.1.1" || hosts[1].String() != "192.168.1.2" {
		t.Fatalf("unexpected hosts: %v", hosts)
	}

	// /31 includes both addresses (point-to-point).
	hosts, err = EnumerateIPv4Hosts("192.168.1.0/31")
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if len(hosts) != 2 {
		t.Fatalf("/31 should yield 2, got %d", len(hosts))
	}

	if _, err := EnumerateIPv4Hosts("2001:db8::/64"); !errors.Is(err, ErrIPv6Unsupported) {
		t.Fatalf("ipv6 should be unsupported, got %v", err)
	}
	if _, err := EnumerateIPv4Hosts("garbage"); !errors.Is(err, ErrInvalidCIDR) {
		t.Fatalf("invalid cidr expected, got %v", err)
	}
}

func TestClassifySeen(t *testing.T) {
	known := db.IpAddress{}
	if got := ClassifySeen(ProbeResult{OpenPorts: []int{80}}, db.IpAddress{}, false, nil); got != "new" {
		t.Fatalf("expected new for unknown IP, got %s", got)
	}
	if got := ClassifySeen(ProbeResult{OpenPorts: []int{80}}, known, true, nil); got != "known" {
		t.Fatalf("expected known when no prior, got %s", got)
	}
	if got := ClassifySeen(ProbeResult{OpenPorts: []int{80}}, known, true, []int{80}); got != "known" {
		t.Fatalf("expected known when ports unchanged, got %s", got)
	}
	// Order shouldn't matter.
	if got := ClassifySeen(ProbeResult{OpenPorts: []int{443, 80}}, known, true, []int{80, 443}); got != "known" {
		t.Fatalf("expected known despite differing slice order, got %s", got)
	}
	if got := ClassifySeen(ProbeResult{OpenPorts: []int{80, 443}}, known, true, []int{80}); got != "changed" {
		t.Fatalf("expected changed when ports differ, got %s", got)
	}
}

func TestNormalizeIP(t *testing.T) {
	if normalizeIP("10.0.0.5") != "10.0.0.5" {
		t.Fatal("plain v4 should round-trip")
	}
	if normalizeIP("10.0.0.5/24") != "10.0.0.5" {
		t.Fatalf("CIDR form should strip mask, got %q", normalizeIP("10.0.0.5/24"))
	}
}

func TestTCPProberDetectsListener(t *testing.T) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	defer ln.Close()
	go func() {
		for {
			conn, err := ln.Accept()
			if err != nil {
				return
			}
			conn.Close()
		}
	}()

	_, portStr, _ := net.SplitHostPort(ln.Addr().String())
	port, _ := strconv.Atoi(portStr)

	prober := &TCPProber{TimeoutMs: 250, Ports: []int{port}}
	res := prober.Probe(context.Background(), netip.MustParseAddr("127.0.0.1"))
	if !res.Seen {
		t.Fatal("listener should be detected")
	}
	if len(res.OpenPorts) != 1 || res.OpenPorts[0] != port {
		t.Fatalf("expected open port %d, got %v", port, res.OpenPorts)
	}
}

func TestTCPProberNoListener(t *testing.T) {
	// Pick a port that is almost certainly closed.
	prober := &TCPProber{TimeoutMs: 100, Ports: []int{1}}
	res := prober.Probe(context.Background(), netip.MustParseAddr("127.0.0.1"))
	if res.Seen {
		t.Fatal("closed port should not be marked seen")
	}
}

func TestParsePortsList(t *testing.T) {
	cases := []struct {
		in       string
		wantOK   bool
		wantList []int
	}{
		{"", false, nil},
		{"   ", false, nil},
		{"abc", false, nil},
		{"0,99999", false, nil},
		{"22,80,443", true, []int{22, 80, 443}},
		{"443,22,80", true, []int{22, 80, 443}},
		{"22, 22, 22, 80", true, []int{22, 80}},
		{"22,abc,80", true, []int{22, 80}},
		{"22,-1,80", true, []int{22, 80}},
		{"22,65535", true, []int{22, 65535}},
	}
	for _, tc := range cases {
		got, ok := ParsePortsList(tc.in)
		if ok != tc.wantOK {
			t.Errorf("ParsePortsList(%q) ok=%v want %v", tc.in, ok, tc.wantOK)
			continue
		}
		if !ok {
			continue
		}
		if len(got) != len(tc.wantList) {
			t.Errorf("ParsePortsList(%q) = %v want %v", tc.in, got, tc.wantList)
			continue
		}
		for i := range got {
			if got[i] != tc.wantList[i] {
				t.Errorf("ParsePortsList(%q)[%d] = %d want %d", tc.in, i, got[i], tc.wantList[i])
			}
		}
	}
}

func TestTCPProberReverseDNSFailureNotFatal(t *testing.T) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	defer ln.Close()
	go func() {
		for {
			conn, err := ln.Accept()
			if err != nil {
				return
			}
			conn.Close()
		}
	}()
	_, portStr, _ := net.SplitHostPort(ln.Addr().String())
	port, _ := strconv.Atoi(portStr)

	prober := &TCPProber{TimeoutMs: 250, Ports: []int{port}}

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	res := prober.Probe(ctx, netip.MustParseAddr("127.0.0.1"))
	if !res.Seen {
		t.Fatal("listener should be detected even if reverse DNS fails")
	}
	// ReverseDNS may or may not resolve depending on host; we only assert the
	// scan didn't fail.
}
