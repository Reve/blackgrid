package service

import (
	"context"

	"blackgrid/internal/db"
	"blackgrid/internal/events"

	"github.com/jackc/pgx/v5/pgtype"
)

type VlanService struct {
	q   *db.Queries
	bus *events.EventBus
}

func NewVlanService(q *db.Queries, bus *events.EventBus) *VlanService {
	return &VlanService{q: q, bus: bus}
}

func (s *VlanService) GetVlans(ctx context.Context) ([]db.Vlan, error) {
	return s.q.GetVlans(ctx)
}

func (s *VlanService) GetVlan(ctx context.Context, id pgtype.UUID) (db.Vlan, error) {
	return s.q.GetVlan(ctx, id)
}

func (s *VlanService) CreateVlan(ctx context.Context, req db.CreateVlanParams) (db.Vlan, error) {
	vlan, err := s.q.CreateVlan(ctx, req)
	if err == nil && s.bus != nil {
		s.bus.Publish(ctx, events.Event{
			Type:       events.IPAMVlanChanged,
			ObjectType: "vlan",
			ObjectID:   events.FormatUUID(vlan.ID),
			Payload: map[string]any{
				"action": "created",
				"vid":    vlan.Vid,
				"name":   vlan.Name,
			},
		})
	}
	return vlan, err
}

func (s *VlanService) UpdateVlan(ctx context.Context, req db.UpdateVlanParams) (db.Vlan, error) {
	vlan, err := s.q.UpdateVlan(ctx, req)
	if err == nil && s.bus != nil {
		s.bus.Publish(ctx, events.Event{
			Type:       events.IPAMVlanChanged,
			ObjectType: "vlan",
			ObjectID:   events.FormatUUID(vlan.ID),
			Payload: map[string]any{
				"action": "updated",
				"vid":    vlan.Vid,
				"name":   vlan.Name,
			},
		})
	}
	return vlan, err
}

func (s *VlanService) DeleteVlan(ctx context.Context, id pgtype.UUID) error {
	err := s.q.DeleteVlan(ctx, id)
	if err == nil && s.bus != nil {
		s.bus.Publish(ctx, events.Event{
			Type:       events.IPAMVlanChanged,
			ObjectType: "vlan",
			ObjectID:   events.FormatUUID(id),
			Payload: map[string]any{
				"action": "deleted",
			},
		})
	}
	return err
}
