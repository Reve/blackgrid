package service

import (
	"context"

	"blackgrid/internal/db"
	"github.com/jackc/pgx/v5/pgtype"
)

type VlanService struct {
	q *db.Queries
}

func NewVlanService(q *db.Queries) *VlanService {
	return &VlanService{q: q}
}

func (s *VlanService) GetVlans(ctx context.Context) ([]db.Vlan, error) {
	return s.q.GetVlans(ctx)
}

func (s *VlanService) GetVlan(ctx context.Context, id pgtype.UUID) (db.Vlan, error) {
	return s.q.GetVlan(ctx, id)
}

func (s *VlanService) CreateVlan(ctx context.Context, req db.CreateVlanParams) (db.Vlan, error) {
	return s.q.CreateVlan(ctx, req)
}

func (s *VlanService) UpdateVlan(ctx context.Context, req db.UpdateVlanParams) (db.Vlan, error) {
	return s.q.UpdateVlan(ctx, req)
}

func (s *VlanService) DeleteVlan(ctx context.Context, id pgtype.UUID) error {
	return s.q.DeleteVlan(ctx, id)
}
