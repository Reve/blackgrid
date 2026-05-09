package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net"
	"net/netip"
	"os"
	"sort"
	"strconv"
	"sync"
	"time"

	"blackgrid/internal/db"
	"blackgrid/internal/events"
	"blackgrid/internal/metrics"

	"github.com/jackc/pgx/v5/pgtype"
)

var (
	ErrUnknownPrefix      = errors.New("unknown prefix")
	ErrScanAlreadyRunning = errors.New("scan already queued or running for this prefix")
	ErrPrefixTooLarge     = errors.New("prefix is too large for manual scan")
	ErrIPv6Unsupported    = errors.New("IPv6 full scanning is not supported")
	ErrInvalidCIDR        = errors.New("invalid CIDR")
	ErrInvalidInterval    = errors.New("scan_interval_seconds must be >= 60")
)

// Prober probes a single IP. Implementations may use TCP, ICMP, etc.
// Returning seen=false means the IP did not respond.
type Prober interface {
	Probe(ctx context.Context, ip netip.Addr) ProbeResult
}

type ProbeResult struct {
	Seen       bool
	OpenPorts  []int
	LatencyMs  int
	ReverseDNS string
}

type DiscoveryService struct {
	q *db.Queries

	workers           int
	maxIPv4PrefixSize int
	tcpTimeoutMs      int
	pingTimeoutMs     int
	prober            Prober
	bus               *events.EventBus
}

func NewDiscoveryService(q *db.Queries, bus *events.EventBus) *DiscoveryService {
	tcpTimeout := getEnvAsInt("DISCOVERY_TCP_TIMEOUT_MS", 750)
	s := &DiscoveryService{
		q:                 q,
		bus:               bus,
		workers:           getEnvAsInt("DISCOVERY_WORKERS", 64),
		maxIPv4PrefixSize: getEnvAsInt("DISCOVERY_MAX_IPV4_PREFIX_SIZE", 22),
		tcpTimeoutMs:      tcpTimeout,
		pingTimeoutMs:     getEnvAsInt("DISCOVERY_PING_TIMEOUT_MS", 750),
	}
	s.prober = &TCPProber{TimeoutMs: tcpTimeout, Ports: DefaultPorts}
	return s
}

// SetProber overrides the prober (used by tests).
func (s *DiscoveryService) SetProber(p Prober) { s.prober = p }

// SetWorkers overrides the worker count (used by tests).
func (s *DiscoveryService) SetWorkers(n int) {
	if n > 0 {
		s.workers = n
	}
}

func getEnvAsInt(key string, fallback int) int {
	if val, ok := os.LookupEnv(key); ok {
		if i, err := strconv.Atoi(val); err == nil {
			return i
		}
	}
	return fallback
}

// DefaultPorts is the TCP probe port list.
var DefaultPorts = []int{22, 53, 80, 443, 5432, 6379, 8000, 8080, 9000, 9443}

// TCPProber implements Prober via parallel TCP connect.
type TCPProber struct {
	TimeoutMs int
	Ports     []int
}

func (p *TCPProber) Probe(ctx context.Context, ip netip.Addr) ProbeResult {
	res := ProbeResult{OpenPorts: []int{}}
	var wg sync.WaitGroup
	var mu sync.Mutex
	timeout := time.Duration(p.TimeoutMs) * time.Millisecond

	for _, port := range p.Ports {
		wg.Add(1)
		go func(prt int) {
			defer wg.Done()
			target := net.JoinHostPort(ip.String(), strconv.Itoa(prt))
			start := time.Now()
			d := net.Dialer{Timeout: timeout}
			conn, err := d.DialContext(ctx, "tcp", target)
			if err != nil {
				return
			}
			conn.Close()
			latency := int(time.Since(start).Milliseconds())
			mu.Lock()
			res.Seen = true
			res.OpenPorts = append(res.OpenPorts, prt)
			if res.LatencyMs == 0 || latency < res.LatencyMs {
				res.LatencyMs = latency
			}
			mu.Unlock()
		}(port)
	}
	wg.Wait()

	if res.Seen {
		sort.Ints(res.OpenPorts)
		// Reverse DNS — failures are not fatal
		ctx2, cancel := context.WithTimeout(ctx, timeout)
		defer cancel()
		var resolver net.Resolver
		names, err := resolver.LookupAddr(ctx2, ip.String())
		if err == nil && len(names) > 0 {
			res.ReverseDNS = names[0]
		}
	}
	return res
}

// ValidatePrefixForScan verifies a stored prefix CIDR is safe to scan.
// Returns nil when the prefix is an IPv4 prefix at or smaller than maxIPv4PrefixSize.
func ValidatePrefixForScan(cidr string, maxIPv4PrefixSize int) error {
	ipNet, err := netip.ParsePrefix(cidr)
	if err != nil {
		return ErrInvalidCIDR
	}
	if !ipNet.Addr().Is4() {
		return ErrIPv6Unsupported
	}
	if ipNet.Bits() < maxIPv4PrefixSize {
		return ErrPrefixTooLarge
	}
	return nil
}

// StartManualScan starts a manual scan for the given stored prefix.
func (s *DiscoveryService) StartManualScan(ctx context.Context, prefixID pgtype.UUID) (db.DiscoveryScan, error) {
	prefix, err := s.q.GetPrefix(ctx, prefixID)
	if err != nil {
		return db.DiscoveryScan{}, ErrUnknownPrefix
	}

	if err := ValidatePrefixForScan(prefix.Prefix, s.maxIPv4PrefixSize); err != nil {
		return db.DiscoveryScan{}, err
	}

	running, err := s.q.GetRunningOrQueuedScansForPrefix(ctx, prefixID)
	if err != nil {
		return db.DiscoveryScan{}, err
	}
	if len(running) > 0 {
		return db.DiscoveryScan{}, ErrScanAlreadyRunning
	}

	scan, err := s.q.CreateDiscoveryScan(ctx, db.CreateDiscoveryScanParams{
		PrefixID: prefixID,
		Status:   "queued",
	})
	if err != nil {
		return db.DiscoveryScan{}, err
	}

	go func() {
		runCtx, cancel := context.WithTimeout(context.Background(), 30*time.Minute)
		defer cancel()
		if err := s.RunScan(runCtx, scan.ID); err != nil {
			log.Printf("scan %s failed: %v", uuidString(scan.ID), err)
		}
	}()

	return scan, nil
}

// RunScan executes a queued scan synchronously.
func (s *DiscoveryService) RunScan(ctx context.Context, scanID pgtype.UUID) error {
	start := time.Now()
	scan, err := s.q.GetDiscoveryScan(ctx, scanID)
	if err != nil {
		return err
	}

	prefix, err := s.q.GetPrefix(ctx, scan.PrefixID)
	if err != nil {
		s.failScan(ctx, scanID, err)
		return err
	}

	if _, err = s.q.UpdateDiscoveryScanStatus(ctx, db.UpdateDiscoveryScanStatusParams{
		ID:        scan.ID,
		Status:    "running",
		StartedAt: pgtype.Timestamptz{Time: time.Now(), Valid: true},
	}); err != nil {
		return err
	}

	if s.bus != nil {
		s.bus.Publish(ctx, events.Event{
			Type:       events.DiscoveryScanStarted,
			ObjectType: "discovery_scan",
			ObjectID:   events.FormatUUID(scan.ID),
			Payload: map[string]any{
				"prefix_id": events.FormatUUID(scan.PrefixID),
			},
		})
	}

	ips, err := EnumerateIPv4Hosts(prefix.Prefix)
	if err != nil {
		s.failScan(ctx, scanID, err)
		return err
	}

	existingIPs, err := s.q.GetIPAddressesByPrefix(ctx, prefix.ID)
	if err != nil {
		s.failScan(ctx, scanID, err)
		return err
	}

	existingByAddr := make(map[string]db.IpAddress, len(existingIPs))
	for _, ip := range existingIPs {
		existingByAddr[normalizeIP(ip.IpAddress)] = ip
	}

	seen := make(map[string]bool)
	var seenMu sync.Mutex

	work := make(chan netip.Addr, len(ips))
	for _, ip := range ips {
		work <- ip
	}
	close(work)

	var wg sync.WaitGroup
	workers := s.workers
	if workers <= 0 {
		workers = 1
	}
	for i := 0; i < workers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for ip := range work {
				select {
				case <-ctx.Done():
					return
				default:
				}
				probe := s.prober.Probe(ctx, ip)
				if !probe.Seen {
					continue
				}
				seenMu.Lock()
				seen[ip.String()] = true
				seenMu.Unlock()
				if err := s.persistSeen(ctx, scan.ID, prefix.ID, ip, probe, existingByAddr); err != nil {
					log.Printf("persist scan result %s: %v", ip, err)
				}
			}
		}()
	}
	wg.Wait()

	// Stale detection: existing IPs not seen this scan.
	for addr, existing := range existingByAddr {
		if seen[addr] {
			continue
		}
		status := existing.Status.String
		if status == "assigned" || status == "discovered" || status == "dhcp" || status == "active" {
			parsed, perr := netip.ParseAddr(addr)
			if perr != nil {
				continue
			}
			if err := s.persistStale(ctx, scan.ID, prefix.ID, parsed); err != nil {
				log.Printf("persist stale %s: %v", addr, err)
			}
		}
	}

	_, err = s.q.UpdateDiscoveryScanStatus(ctx, db.UpdateDiscoveryScanStatusParams{
		ID:          scan.ID,
		Status:      "completed",
		CompletedAt: pgtype.Timestamptz{Time: time.Now(), Valid: true},
	})

	metrics.DiscoveryScansTotal.WithLabelValues("completed").Inc()

	if err == nil && s.bus != nil {
		s.bus.Publish(ctx, events.Event{
			Type:       events.DiscoveryScanCompleted,
			ObjectType: "discovery_scan",
			ObjectID:   events.FormatUUID(scan.ID),
			Payload: map[string]any{
				"prefix_id": events.FormatUUID(scan.PrefixID),
			},
		})
	}
	return err
}

// ClassifySeen returns the classification for a probed IP given existing IPAM
// state and the most recent prior result for that address (may be empty).
func ClassifySeen(probe ProbeResult, existing db.IpAddress, exists bool, prior []int) string {
	if !exists {
		return "new"
	}
	prevSorted := append([]int(nil), prior...)
	sort.Ints(prevSorted)
	curSorted := append([]int(nil), probe.OpenPorts...)
	sort.Ints(curSorted)
	if len(prevSorted) > 0 && !intSliceEq(prevSorted, curSorted) {
		return "changed"
	}
	return "known"
}

func intSliceEq(a, b []int) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

func (s *DiscoveryService) persistSeen(ctx context.Context, scanID, prefixID pgtype.UUID, ip netip.Addr, probe ProbeResult, existing map[string]db.IpAddress) error {
	addr := ip.String()
	existingIP, exists := existing[addr]

	var prior []int
	if exists {
		recent, err := s.q.GetRecentDiscoveryResultsByAddress(ctx, db.GetRecentDiscoveryResultsByAddressParams{
			PrefixID: prefixID,
			Address:  ip,
		})
		if err == nil && len(recent) > 0 {
			_ = json.Unmarshal(recent[0].OpenPorts, &prior)
		}
	}

	classification := ClassifySeen(probe, existingIP, exists, prior)

	if exists {
		// Reconciliation: bump last_seen_at on known IP.
		if _, err := s.q.UpdateIPAddressLastSeen(ctx, existingIP.ID); err != nil {
			log.Printf("update last_seen_at for %s: %v", addr, err)
		}
	}

	portsJSON, _ := json.Marshal(probe.OpenPorts)
	_, err := s.q.CreateDiscoveryResult(ctx, db.CreateDiscoveryResultParams{
		ScanID:         scanID,
		PrefixID:       prefixID,
		Address:        ip,
		ReverseDns:     pgtype.Text{String: probe.ReverseDNS, Valid: probe.ReverseDNS != ""},
		OpenPorts:      portsJSON,
		LatencyMs:      pgtype.Int4{Int32: int32(probe.LatencyMs), Valid: probe.LatencyMs > 0},
		Classification: classification,
		Raw:            []byte("{}"),
	})

	if err == nil && s.bus != nil {
		s.bus.Publish(ctx, events.Event{
			Type:       events.DiscoveryResultCreated,
			ObjectType: "discovery_result",
			ObjectID:   events.FormatUUID(scanID), // Result ID is not returned by sqlc CreateDiscoveryResult in some schemas, using scanID for grouping or just publishing event
			Payload: map[string]any{
				"scan_id":        events.FormatUUID(scanID),
				"prefix_id":      events.FormatUUID(prefixID),
				"address":         addr,
				"classification": classification,
				"reverse_dns":    probe.ReverseDNS,
			},
		})

		if classification == "new" {
			s.bus.Publish(ctx, events.Event{
				Type:       events.DiscoveryNewHost,
				ObjectType: "discovery_result",
				Payload: map[string]any{
					"address":   addr,
					"prefix_id": events.FormatUUID(prefixID),
				},
			})
		} else if classification == "changed" {
			s.bus.Publish(ctx, events.Event{
				Type:       events.DiscoveryConflictDetected,
				ObjectType: "discovery_result",
				Payload: map[string]any{
					"address":   addr,
					"prefix_id": events.FormatUUID(prefixID),
					"details":   "Open ports changed",
				},
			})
		}
	}
	return err
}

func (s *DiscoveryService) persistStale(ctx context.Context, scanID, prefixID pgtype.UUID, ip netip.Addr) error {
	_, err := s.q.CreateDiscoveryResult(ctx, db.CreateDiscoveryResultParams{
		ScanID:         scanID,
		PrefixID:       prefixID,
		Address:        ip,
		OpenPorts:      []byte("[]"),
		Classification: "stale",
		Raw:            []byte("{}"),
	})
	if err == nil && s.bus != nil {
		s.bus.Publish(ctx, events.Event{
			Type:       events.DiscoveryStaleDetected,
			ObjectType: "discovery_result",
			Payload: map[string]any{
				"address":   ip.String(),
				"prefix_id": events.FormatUUID(prefixID),
				"scan_id":    events.FormatUUID(scanID),
			},
		})
	}
	return err
}

func (s *DiscoveryService) failScan(ctx context.Context, scanID pgtype.UUID, scanErr error) {
	_, _ = s.q.UpdateDiscoveryScanStatus(ctx, db.UpdateDiscoveryScanStatusParams{
		ID:          scanID,
		Status:      "failed",
		CompletedAt: pgtype.Timestamptz{Time: time.Now(), Valid: true},
		Error:       pgtype.Text{String: scanErr.Error(), Valid: true},
	})

	metrics.DiscoveryScansTotal.WithLabelValues("failed").Inc()

	if s.bus != nil {
		s.bus.Publish(ctx, events.Event{
			Type:       events.DiscoveryScanFailed,
			ObjectType: "discovery_scan",
			ObjectID:   events.FormatUUID(scanID),
			Payload: map[string]any{
				"error": scanErr.Error(),
			},
		})
	}
}

func uuidString(id pgtype.UUID) string {
	if !id.Valid {
		return ""
	}
	return fmt.Sprintf("%x-%x-%x-%x-%x", id.Bytes[0:4], id.Bytes[4:6], id.Bytes[6:8], id.Bytes[8:10], id.Bytes[10:16])
}

// normalizeIP strips a CIDR mask if present and parses to canonical form.
func normalizeIP(s string) string {
	if a, err := netip.ParseAddr(s); err == nil {
		return a.String()
	}
	if p, err := netip.ParsePrefix(s); err == nil {
		return p.Addr().String()
	}
	return s
}

// EnumerateIPv4Hosts lists host addresses inside an IPv4 CIDR, excluding
// network and broadcast addresses for prefixes shorter than /31.
func EnumerateIPv4Hosts(cidr string) ([]netip.Addr, error) {
	prefix, err := netip.ParsePrefix(cidr)
	if err != nil {
		return nil, ErrInvalidCIDR
	}
	if !prefix.Addr().Is4() {
		return nil, ErrIPv6Unsupported
	}

	first := prefix.Addr()
	bits := prefix.Bits()

	var broadcast netip.Addr
	if bits < 31 {
		b := first.As4()
		hostBits := 32 - bits
		for i := 0; i < hostBits; i++ {
			byteIdx := 3 - (i / 8)
			bitIdx := i % 8
			b[byteIdx] |= 1 << bitIdx
		}
		broadcast = netip.AddrFrom4(b)
		first = first.Next() // skip network
	}

	var out []netip.Addr
	for cur := first; prefix.Contains(cur); cur = cur.Next() {
		if bits < 31 && cur.Compare(broadcast) == 0 {
			break
		}
		out = append(out, cur)
	}
	return out, nil
}

// AcceptResultInput describes the optional accept payload.
type AcceptResultInput struct {
	Hostname string `json:"hostname"`
	FQDN     string `json:"fqdn"`
	Status   string `json:"status"`
}

func (s *DiscoveryService) ListScans(ctx context.Context, prefixID pgtype.UUID, status string, limit, offset int32) ([]db.DiscoveryScan, error) {
	return s.q.ListDiscoveryScans(ctx, db.ListDiscoveryScansParams{
		Column1: prefixID,
		Column2: status,
		Offset:  offset,
		Limit:   limit,
	})
}

func (s *DiscoveryService) GetScan(ctx context.Context, id pgtype.UUID) (db.DiscoveryScan, error) {
	return s.q.GetDiscoveryScan(ctx, id)
}

func (s *DiscoveryService) ListResults(ctx context.Context, scanID, prefixID pgtype.UUID, classification string, ignored *bool, limit, offset int32) ([]db.DiscoveryResult, error) {
	ignoredBool := false
	if ignored != nil {
		ignoredBool = *ignored
	}
	return s.q.ListDiscoveryResults(ctx, db.ListDiscoveryResultsParams{
		Column1: scanID,
		Column2: prefixID,
		Column3: classification,
		Column4: ignoredBool,
		Offset:  offset,
		Limit:   limit,
	})
}

func (s *DiscoveryService) AcceptResult(ctx context.Context, resultID pgtype.UUID, input AcceptResultInput) (db.IpAddress, error) {
	res, err := s.q.GetDiscoveryResult(ctx, resultID)
	if err != nil {
		return db.IpAddress{}, err
	}

	if res.CreatedIpAddressID.Valid {
		return s.q.GetIPAddress(ctx, res.CreatedIpAddressID)
	}

	addr := res.Address.String()
	existingIPs, err := s.q.GetIPAddressesByPrefix(ctx, res.PrefixID)
	if err == nil {
		for _, ip := range existingIPs {
			if normalizeIP(ip.IpAddress) == addr {
				if _, err := s.q.UpdateDiscoveryResultAccepted(ctx, db.UpdateDiscoveryResultAcceptedParams{
					ID:                 res.ID,
					CreatedIpAddressID: ip.ID,
				}); err != nil {
					return ip, err
				}
				_, _ = s.q.UpdateIPAddressLastSeen(ctx, ip.ID)
				return ip, nil
			}
		}
	}

	status := input.Status
	if status == "" {
		status = "discovered"
	}

	desc := input.FQDN
	if desc == "" {
		desc = input.Hostname
	}
	if desc == "" && res.ReverseDns.Valid {
		desc = res.ReverseDns.String
	}
	if desc == "" && res.Hostname.Valid {
		desc = res.Hostname.String
	}

	newIP, err := s.q.CreateIPAddress(ctx, db.CreateIPAddressParams{
		PrefixID:    res.PrefixID,
		IpAddress:   addr,
		Status:      pgtype.Text{String: status, Valid: true},
		Description: pgtype.Text{String: desc, Valid: desc != ""},
	})
	if err != nil {
		return db.IpAddress{}, err
	}

	if _, err := s.q.UpdateIPAddressLastSeen(ctx, newIP.ID); err != nil {
		log.Printf("update last_seen for accepted IP %s: %v", addr, err)
	}

	if _, err := s.q.UpdateDiscoveryResultAccepted(ctx, db.UpdateDiscoveryResultAcceptedParams{
		ID:                 res.ID,
		CreatedIpAddressID: newIP.ID,
	}); err != nil {
		return newIP, fmt.Errorf("ip created but failed to link to result: %w", err)
	}

	if s.bus != nil {
		s.bus.Publish(ctx, events.Event{
			Type:       events.DiscoveryResultAccepted,
			ObjectType: "discovery_result",
			ObjectID:   events.FormatUUID(res.ID),
			Payload: map[string]any{
				"ip_address_id": events.FormatUUID(newIP.ID),
				"address":       newIP.IpAddress,
			},
		})
	}

	return newIP, nil
}

func (s *DiscoveryService) IgnoreResult(ctx context.Context, resultID pgtype.UUID) (db.DiscoveryResult, error) {
	res, err := s.q.UpdateDiscoveryResultIgnored(ctx, resultID)
	if err == nil && s.bus != nil {
		s.bus.Publish(ctx, events.Event{
			Type:       events.DiscoveryResultIgnored,
			ObjectType: "discovery_result",
			ObjectID:   events.FormatUUID(resultID),
		})
	}
	return res, err
}

// RunScheduledScans loops on a 1-minute ticker. It returns when ctx is done.
func (s *DiscoveryService) RunScheduledScans(ctx context.Context) {
	ticker := time.NewTicker(1 * time.Minute)
	defer ticker.Stop()

	s.tickScheduled(ctx)
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			s.tickScheduled(ctx)
		}
	}
}

func (s *DiscoveryService) tickScheduled(ctx context.Context) {
	prefixes, err := s.q.GetPrefixesForScheduledScans(ctx)
	if err != nil {
		log.Printf("scheduled scan: fetch prefixes: %v", err)
		return
	}
	for _, p := range prefixes {
		if _, err := s.StartManualScan(ctx, p.ID); err != nil && !errors.Is(err, ErrScanAlreadyRunning) {
			log.Printf("scheduled scan for %s: %v", p.Prefix, err)
		}
	}
}

// ValidateScanInterval enforces the >= 60 second minimum.
func ValidateScanInterval(seconds int) error {
	if seconds < 60 {
		return ErrInvalidInterval
	}
	return nil
}
