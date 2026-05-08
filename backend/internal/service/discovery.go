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
	"strconv"
	"sync"
	"time"

	"blackgrid/internal/db"

	"github.com/jackc/pgx/v5/pgtype"
)

var (
	ErrUnknownPrefix      = errors.New("unknown prefix")
	ErrScanAlreadyRunning = errors.New("scan already queued or running for this prefix")
	ErrPrefixTooLarge     = errors.New("prefix is too large for manual scan")
	ErrIPv6Unsupported    = errors.New("IPv6 full scanning is not supported")
	ErrInvalidCIDR        = errors.New("invalid CIDR")
)

type DiscoveryService struct {
	q *db.Queries

	workers           int
	maxIPv4PrefixSize int
	tcpTimeoutMs      int
}

func NewDiscoveryService(q *db.Queries) *DiscoveryService {
	return &DiscoveryService{
		q:                 q,
		workers:           getEnvAsInt("DISCOVERY_WORKERS", 64),
		maxIPv4PrefixSize: getEnvAsInt("DISCOVERY_MAX_IPV4_PREFIX_SIZE", 22),
		tcpTimeoutMs:      getEnvAsInt("DISCOVERY_TCP_TIMEOUT_MS", 750),
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

// StartManualScan starts or queues a manual scan for the given stored prefix.
func (s *DiscoveryService) StartManualScan(ctx context.Context, prefixID pgtype.UUID) (db.DiscoveryScan, error) {
	// Reject unknown prefix_id
	prefix, err := s.q.GetPrefix(ctx, prefixID)
	if err != nil {
		return db.DiscoveryScan{}, ErrUnknownPrefix
	}

	// Reject manual scan if prefix is larger than /22 for IPv4 unless allowed
	ipNet, err := netip.ParsePrefix(prefix.Prefix)
	if err != nil {
		return db.DiscoveryScan{}, ErrInvalidCIDR
	}

	if ipNet.Addr().Is4() {
		ones := ipNet.Bits()
		if ones < s.maxIPv4PrefixSize {
			return db.DiscoveryScan{}, ErrPrefixTooLarge
		}
	} else {
		return db.DiscoveryScan{}, ErrIPv6Unsupported
	}

	// Reject if another scan is currently queued or running for the same prefix
	runningScans, err := s.q.GetRunningOrQueuedScansForPrefix(ctx, prefixID)
	if err != nil {
		return db.DiscoveryScan{}, err
	}
	if len(runningScans) > 0 {
		return db.DiscoveryScan{}, ErrScanAlreadyRunning
	}

	scan, err := s.q.CreateDiscoveryScan(ctx, db.CreateDiscoveryScanParams{
		PrefixID: prefixID,
		Status:   "queued",
	})
	if err != nil {
		return db.DiscoveryScan{}, err
	}

	// Run scan asynchronously
	go func() {
		err := s.RunScan(context.Background(), scan.ID)
		if err != nil {
			log.Printf("Scan %s failed: %v", dbUUIDToString(scan.ID), err)
		}
	}()

	return scan, nil
}

// RunScan executes the discovery scan logic
func (s *DiscoveryService) RunScan(ctx context.Context, scanID pgtype.UUID) error {
	scan, err := s.q.GetDiscoveryScan(ctx, scanID)
	if err != nil {
		return err
	}

	prefix, err := s.q.GetPrefix(ctx, scan.PrefixID)
	if err != nil {
		s.failScan(ctx, scanID, err)
		return err
	}

	_, err = s.q.UpdateDiscoveryScanStatus(ctx, db.UpdateDiscoveryScanStatusParams{
		ID:        scan.ID,
		Status:    "running",
		StartedAt: pgtype.Timestamptz{Time: time.Now(), Valid: true},
	})
	if err != nil {
		return err
	}

	ips, err := getIpsToScan(prefix.Prefix)
	if err != nil {
		s.failScan(ctx, scanID, err)
		return err
	}

	// Load existing IPs for classification
	existingIPs, err := s.q.GetIPAddressesByPrefix(ctx, prefix.ID)
	if err != nil {
		s.failScan(ctx, scanID, err)
		return err
	}

	existingIPMap := make(map[string]db.IpAddress)
	for _, ip := range existingIPs {
		existingIPMap[ip.IpAddress] = ip
	}

	// Setup bounded worker pool
	workQueue := make(chan netip.Addr, len(ips))
	for _, ip := range ips {
		workQueue <- ip
	}
	close(workQueue)

	var wg sync.WaitGroup
	var mu sync.Mutex
	seenIPs := make(map[string]bool)

	for i := 0; i < s.workers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for ip := range workQueue {
				res := s.scanIP(ctx, ip)
				if res.seen {
					mu.Lock()
					seenIPs[ip.String()] = true
					err := s.processScanResult(ctx, scan.ID, prefix.ID, res, existingIPMap)
					if err != nil {
						log.Printf("Failed to process scan result for %s: %v", ip, err)
					}
					mu.Unlock()
				}
			}
		}()
	}

	wg.Wait()

	// Handle stale classifications
	for ipStr, existingIP := range existingIPMap {
		if !seenIPs[ipStr] {
			if existingIP.Status.String == "active" || existingIP.Status.String == "assigned" || existingIP.Status.String == "discovered" || existingIP.Status.String == "dhcp" {
				addr, _ := netip.ParseAddr(ipStr)
				_ = s.processScanResult(ctx, scan.ID, prefix.ID, scanResult{ip: addr, seen: false, isStale: true}, existingIPMap)
			}
		}
	}

	_, err = s.q.UpdateDiscoveryScanStatus(ctx, db.UpdateDiscoveryScanStatusParams{
		ID:          scan.ID,
		Status:      "completed",
		CompletedAt: pgtype.Timestamptz{Time: time.Now(), Valid: true},
	})

	return err
}

type scanResult struct {
	ip         netip.Addr
	seen       bool
	openPorts  []int
	latencyMs  int
	reverseDns string
	isStale    bool
}

var defaultPorts = []int{22, 53, 80, 443, 5432, 6379, 8000, 8080, 9000, 9443}

func (s *DiscoveryService) scanIP(ctx context.Context, ip netip.Addr) scanResult {
	res := scanResult{ip: ip, openPorts: []int{}}

	var wg sync.WaitGroup
	var mu sync.Mutex

	timeout := time.Duration(s.tcpTimeoutMs) * time.Millisecond

	for _, port := range defaultPorts {
		wg.Add(1)
		go func(p int) {
			defer wg.Done()
			target := fmt.Sprintf("%s:%d", ip.String(), p)
			start := time.Now()

			d := net.Dialer{Timeout: timeout}
			conn, err := d.DialContext(ctx, "tcp", target)

			if err == nil {
				conn.Close()
				latency := int(time.Since(start).Milliseconds())

				mu.Lock()
				res.seen = true
				res.openPorts = append(res.openPorts, p)
				if res.latencyMs == 0 || latency < res.latencyMs {
					res.latencyMs = latency
				}
				mu.Unlock()
			}
		}(port)
	}

	wg.Wait()

	if res.seen {
		names, _ := net.LookupAddr(ip.String())
		if len(names) > 0 {
			res.reverseDns = names[0]
		}
	}

	return res
}

func (s *DiscoveryService) processScanResult(ctx context.Context, scanID, prefixID pgtype.UUID, res scanResult, existingMap map[string]db.IpAddress) error {
	ipStr := res.ip.String()
	existingIP, exists := existingMap[ipStr]

	classification := "new"
	if exists {
		classification = "known"

		// Update existing IP's last seen at if known
		if !res.isStale {
			_, _ = s.q.UpdateIPAddressLastSeen(ctx, existingIP.ID)
		}
	}

	if res.isStale {
		classification = "stale"
	}

	if exists && !res.isStale {
		// check for 'changed' state
		recentResults, err := s.q.GetRecentDiscoveryResultsByAddress(ctx, db.GetRecentDiscoveryResultsByAddressParams{
			PrefixID: prefixID,
			Address:  res.ip,
		})

		if err == nil && len(recentResults) > 0 {
			lastRes := recentResults[0]
			var lastPorts []int
			_ = json.Unmarshal(lastRes.OpenPorts, &lastPorts)

			if fmt.Sprintf("%v", lastPorts) != fmt.Sprintf("%v", res.openPorts) {
				classification = "changed"
			}
		}
	}

	portsJSON, _ := json.Marshal(res.openPorts)

	_, err := s.q.CreateDiscoveryResult(ctx, db.CreateDiscoveryResultParams{
		ScanID:         scanID,
		PrefixID:       prefixID,
		Address:        res.ip,
		ReverseDns:     pgtype.Text{String: res.reverseDns, Valid: res.reverseDns != ""},
		OpenPorts:      portsJSON,
		LatencyMs:      pgtype.Int4{Int32: int32(res.latencyMs), Valid: res.latencyMs > 0},
		Classification: classification,
		Raw:            []byte("{}"),
	})

	return err
}

func (s *DiscoveryService) failScan(ctx context.Context, scanID pgtype.UUID, scanErr error) {
	_, _ = s.q.UpdateDiscoveryScanStatus(ctx, db.UpdateDiscoveryScanStatusParams{
		ID:          scanID,
		Status:      "failed",
		CompletedAt: pgtype.Timestamptz{Time: time.Now(), Valid: true},
		Error:       pgtype.Text{String: scanErr.Error(), Valid: true},
	})
}

func dbUUIDToString(id pgtype.UUID) string {
	if !id.Valid {
		return ""
	}
	return fmt.Sprintf("%x-%x-%x-%x-%x", id.Bytes[0:4], id.Bytes[4:6], id.Bytes[6:8], id.Bytes[8:10], id.Bytes[10:16])
}

func getIpsToScan(cidr string) ([]netip.Addr, error) {
	prefix, err := netip.ParsePrefix(cidr)
	if err != nil {
		return nil, err
	}

	var addrs []netip.Addr

	addr := prefix.Addr()
	if addr.Is4() && prefix.Bits() < 31 {
		addr = addr.Next() // skip network
	}

	var broadcast netip.Addr
	if prefix.Addr().Is4() && prefix.Bits() < 31 {
		b := prefix.Addr().As4()
		maskLen := prefix.Bits()
		hostBits := 32 - maskLen
		for i := 0; i < hostBits; i++ {
			byteIdx := 3 - (i / 8)
			bitIdx := i % 8
			b[byteIdx] |= (1 << bitIdx)
		}
		broadcast = netip.AddrFrom4(b)
	}

	for prefix.Contains(addr) {
		if addr.Is4() && addr.Compare(broadcast) == 0 {
			break
		}
		addrs = append(addrs, addr)
		addr = addr.Next()
	}

	return addrs, nil
}

// AcceptResultInput accept input for result
type AcceptResultInput struct {
	Hostname string `json:"hostname"`
	FQDN     string `json:"fqdn"`
	Status   string `json:"status"`
}

// ListScans returns a list of scans
func (s *DiscoveryService) ListScans(ctx context.Context, prefixID pgtype.UUID, status string, limit, offset int32) ([]db.DiscoveryScan, error) {
	return s.q.ListDiscoveryScans(ctx, db.ListDiscoveryScansParams{
		Column1: prefixID,
		Column2: status,
		Offset:  offset,
		Limit:   limit,
	})
}

// GetScan returns scan by id
func (s *DiscoveryService) GetScan(ctx context.Context, id pgtype.UUID) (db.DiscoveryScan, error) {
	return s.q.GetDiscoveryScan(ctx, id)
}

// ListResults returns a list of results
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

// AcceptResult accepts a discovered host into IPAM
func (s *DiscoveryService) AcceptResult(ctx context.Context, resultID pgtype.UUID, input AcceptResultInput) (db.IpAddress, error) {
	res, err := s.q.GetDiscoveryResult(ctx, resultID)
	if err != nil {
		return db.IpAddress{}, err
	}

	if res.CreatedIpAddressID.Valid {
		return s.q.GetIPAddress(ctx, res.CreatedIpAddressID)
	}

	// Check if IP already exists to avoid duplicate
	existingIPs, err := s.q.GetIPAddressesByPrefix(ctx, res.PrefixID)
	if err == nil {
		for _, ip := range existingIPs {
			if ip.IpAddress == res.Address.String() {
				// Link and return existing
				_, _ = s.q.UpdateDiscoveryResultAccepted(ctx, db.UpdateDiscoveryResultAcceptedParams{
					ID:                 res.ID,
					CreatedIpAddressID: ip.ID,
				})
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
		IpAddress:   res.Address.String(),
		Status:      pgtype.Text{String: status, Valid: true},
		Description: pgtype.Text{String: desc, Valid: desc != ""},
	})
	if err != nil {
		return db.IpAddress{}, err
	}

	_, err = s.q.UpdateIPAddressLastSeen(ctx, newIP.ID)
	if err != nil {
		log.Printf("Failed to update last seen for newly accepted IP: %v", err)
	}

	_, err = s.q.UpdateDiscoveryResultAccepted(ctx, db.UpdateDiscoveryResultAcceptedParams{
		ID:                 res.ID,
		CreatedIpAddressID: newIP.ID,
	})
	if err != nil {
		return newIP, fmt.Errorf("ip created but failed to link to result: %w", err)
	}

	return newIP, nil
}

// IgnoreResult marks a result as ignored
func (s *DiscoveryService) IgnoreResult(ctx context.Context, resultID pgtype.UUID) (db.DiscoveryResult, error) {
	return s.q.UpdateDiscoveryResultIgnored(ctx, resultID)
}

// RunScheduledScans loop that triggers scans periodically
func (s *DiscoveryService) RunScheduledScans(ctx context.Context) {
	ticker := time.NewTicker(1 * time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			prefixes, err := s.q.GetPrefixesForScheduledScans(ctx)
			if err != nil {
				log.Printf("Error fetching scheduled prefixes: %v", err)
				continue
			}

			for _, p := range prefixes {
				log.Printf("Starting scheduled scan for prefix %s", p.Prefix)
				_, err := s.StartManualScan(ctx, p.ID)
				if err != nil && !errors.Is(err, ErrScanAlreadyRunning) {
					log.Printf("Failed to start scheduled scan for %s: %v", p.Prefix, err)
				}
			}
		}
	}
}
