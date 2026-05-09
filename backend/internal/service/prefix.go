package service

import (
	"context"

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
