package service

import (
	"context"
	"encoding/json"
	"net"

	"blackgrid/internal/db"
	"blackgrid/internal/events"
	"blackgrid/pkg/ipam"

	"github.com/jackc/pgx/v5/pgtype"
)

type PrefixService struct {
	q   *db.Queries
	bus *events.EventBus
}

func NewPrefixService(q *db.Queries, bus *events.EventBus) *PrefixService {
	return &PrefixService{q: q, bus: bus}
}

func (s *PrefixService) GetPrefixes(ctx context.Context) ([]db.Prefix, error) {
	return s.q.GetPrefixes(ctx)
}

func (s *PrefixService) GetPrefix(ctx context.Context, id pgtype.UUID) (db.Prefix, error) {
	return s.q.GetPrefix(ctx, id)
}

func (s *PrefixService) CreatePrefix(ctx context.Context, req db.CreatePrefixParams) (db.Prefix, error) {
	if err := ipam.ValidateCIDR(req.Prefix); err != nil {
		return db.Prefix{}, err
	}
	prefix, err := s.q.CreatePrefix(ctx, req)
	if err == nil && s.bus != nil {
		s.bus.Publish(ctx, events.Event{
			Type:       events.IPAMPrefixChanged,
			ObjectType: "prefix",
			ObjectID:   events.FormatUUID(prefix.ID),
			Payload: map[string]any{
				"action": "created",
				"prefix": prefix.Prefix,
			},
		})
	}
	return prefix, err
}

func (s *PrefixService) UpdatePrefix(ctx context.Context, req db.UpdatePrefixParams) (db.Prefix, error) {
	if err := ipam.ValidateCIDR(req.Prefix); err != nil {
		return db.Prefix{}, err
	}
	prefix, err := s.q.UpdatePrefix(ctx, req)
	if err == nil && s.bus != nil {
		s.bus.Publish(ctx, events.Event{
			Type:       events.IPAMPrefixChanged,
			ObjectType: "prefix",
			ObjectID:   events.FormatUUID(prefix.ID),
			Payload: map[string]any{
				"action": "updated",
				"prefix": prefix.Prefix,
			},
		})
	}
	return prefix, err
}

func (s *PrefixService) DeletePrefix(ctx context.Context, id pgtype.UUID) error {
	err := s.q.DeletePrefix(ctx, id)
	if err == nil && s.bus != nil {
		s.bus.Publish(ctx, events.Event{
			Type:       events.IPAMPrefixChanged,
			ObjectType: "prefix",
			ObjectID:   events.FormatUUID(id),
			Payload: map[string]any{
				"action": "deleted",
			},
		})
	}
	return err
}

// UpdateScanConfig validates scan_interval_seconds and persists scan settings.
func (s *PrefixService) UpdateScanConfig(ctx context.Context, id pgtype.UUID, enabled bool, interval int32) (db.Prefix, error) {
	if err := ValidateScanInterval(int(interval)); err != nil {
		return db.Prefix{}, err
	}
	prefix, err := s.q.UpdatePrefixScanConfig(ctx, db.UpdatePrefixScanConfigParams{
		ID:                  id,
		ScanEnabled:         enabled,
		ScanIntervalSeconds: interval,
	})
	if err == nil && s.bus != nil {
		s.bus.Publish(ctx, events.Event{
			Type:       events.IPAMPrefixChanged,
			ObjectType: "prefix",
			ObjectID:   events.FormatUUID(id),
			Payload: map[string]any{
				"action":       "scan_config_updated",
				"scan_enabled": enabled,
			},
		})
	}
	return prefix, err
}

// LatestScan returns the most recent scan for a prefix (or pgx.ErrNoRows).
func (s *PrefixService) LatestScan(ctx context.Context, id pgtype.UUID) (db.DiscoveryScan, error) {
	return s.q.GetLatestScanForPrefix(ctx, id)
}

// NextAvailableIP finds the next unassigned IP in a prefix and returns it
func (s *PrefixService) NextAvailableIP(ctx context.Context, id pgtype.UUID) (string, error) {
	prefix, err := s.q.GetPrefix(ctx, id)
	if err != nil {
		return "", err
	}

	ips, err := s.q.GetIPAddressesByPrefix(ctx, id)
	if err != nil {
		return "", err
	}

	var existing []string
	for _, ip := range ips {
		existing = append(existing, ip.IpAddress)
	}

	return ipam.GetNextAvailableIP(prefix.Prefix, existing)
}

// PrefixUtilization summarises usage across a prefix.
type PrefixUtilization struct {
	PrefixID    string  `json:"prefix_id"`
	Prefix      string  `json:"prefix"`
	TotalHosts  int     `json:"total_hosts"`
	Allocated   int     `json:"allocated"`
	Free        int     `json:"free"`
	PercentUsed float64 `json:"percent_used"`
}

// Utilization returns simple counts of allocated vs free addresses recorded
// for a prefix. "Allocated" means there is a row in ip_addresses for that IP
// whose status is not "available". TotalHosts is the addressable host count
// of the CIDR (network/broadcast excluded for IPv4 /<31).
func (s *PrefixService) Utilization(ctx context.Context, id pgtype.UUID) (PrefixUtilization, error) {
	prefix, err := s.q.GetPrefix(ctx, id)
	if err != nil {
		return PrefixUtilization{}, err
	}
	ips, err := s.q.GetIPAddressesByPrefix(ctx, id)
	if err != nil {
		return PrefixUtilization{}, err
	}

	allocated := 0
	for _, ip := range ips {
		if !ip.Status.Valid || ip.Status.String != "available" {
			allocated++
		}
	}

	total := 0
	if _, ipnet, err := net.ParseCIDR(prefix.Prefix); err == nil {
		ones, bits := ipnet.Mask.Size()
		hostBits := bits - ones
		switch {
		case hostBits <= 0:
			total = 1
		case bits == 32 && hostBits >= 2:
			// IPv4 /<31: subtract network + broadcast
			total = (1 << hostBits) - 2
		default:
			total = 1 << hostBits
		}
	}

	free := total - allocated
	if free < 0 {
		free = 0
	}
	pct := 0.0
	if total > 0 {
		pct = (float64(allocated) / float64(total)) * 100
	}

	prefixID := ""
	if b, err := id.MarshalJSON(); err == nil {
		_ = json.Unmarshal(b, &prefixID)
	}

	return PrefixUtilization{
		PrefixID:    prefixID,
		Prefix:      prefix.Prefix,
		TotalHosts:  total,
		Allocated:   allocated,
		Free:        free,
		PercentUsed: pct,
	}, nil
}
