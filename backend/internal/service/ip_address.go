package service

import (
	"context"

	"blackgrid/internal/db"
	"blackgrid/internal/events"
	"blackgrid/pkg/ipam"

	"github.com/jackc/pgx/v5/pgtype"
)

type IPAddressService struct {
	q   *db.Queries
	bus *events.EventBus
}

func NewIPAddressService(q *db.Queries, bus *events.EventBus) *IPAddressService {
	return &IPAddressService{q: q, bus: bus}
}

func (s *IPAddressService) GetIPAddresses(ctx context.Context) ([]db.IpAddress, error) {
	return s.q.GetIPAddresses(ctx)
}

func (s *IPAddressService) GetIPAddress(ctx context.Context, id pgtype.UUID) (db.IpAddress, error) {
	return s.q.GetIPAddress(ctx, id)
}

func (s *IPAddressService) CreateIPAddress(ctx context.Context, req db.CreateIPAddressParams) (db.IpAddress, error) {
	if err := ipam.ValidateIP(req.IpAddress); err != nil {
		return db.IpAddress{}, err
	}
	ip, err := s.q.CreateIPAddress(ctx, req)
	if err == nil && s.bus != nil {
		s.bus.Publish(ctx, events.Event{
			Type:       events.IPAMIPAddressChanged,
			ObjectType: "ip_address",
			ObjectID:   events.FormatUUID(ip.ID),
			Payload: map[string]any{
				"action":     "created",
				"ip_address": ip.IpAddress,
			},
		})
	}
	return ip, err
}

func (s *IPAddressService) UpdateIPAddress(ctx context.Context, req db.UpdateIPAddressParams) (db.IpAddress, error) {
	if err := ipam.ValidateIP(req.IpAddress); err != nil {
		return db.IpAddress{}, err
	}
	ip, err := s.q.UpdateIPAddress(ctx, req)
	if err == nil && s.bus != nil {
		s.bus.Publish(ctx, events.Event{
			Type:       events.IPAMIPAddressChanged,
			ObjectType: "ip_address",
			ObjectID:   events.FormatUUID(ip.ID),
			Payload: map[string]any{
				"action":     "updated",
				"ip_address": ip.IpAddress,
			},
		})
	}
	return ip, err
}

func (s *IPAddressService) DeleteIPAddress(ctx context.Context, id pgtype.UUID) error {
	err := s.q.DeleteIPAddress(ctx, id)
	if err == nil && s.bus != nil {
		s.bus.Publish(ctx, events.Event{
			Type:       events.IPAMIPAddressChanged,
			ObjectType: "ip_address",
			ObjectID:   events.FormatUUID(id),
			Payload: map[string]any{
				"action": "deleted",
			},
		})
	}
	return err
}
