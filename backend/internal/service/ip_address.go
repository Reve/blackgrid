package service

import (
	"context"

	"blackgrid/internal/db"
	"blackgrid/pkg/ipam"
	"github.com/jackc/pgx/v5/pgtype"
)

type IPAddressService struct {
	q *db.Queries
}

func NewIPAddressService(q *db.Queries) *IPAddressService {
	return &IPAddressService{q: q}
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
	return s.q.CreateIPAddress(ctx, req)
}

func (s *IPAddressService) UpdateIPAddress(ctx context.Context, req db.UpdateIPAddressParams) (db.IpAddress, error) {
	if err := ipam.ValidateIP(req.IpAddress); err != nil {
		return db.IpAddress{}, err
	}
	return s.q.UpdateIPAddress(ctx, req)
}

func (s *IPAddressService) DeleteIPAddress(ctx context.Context, id pgtype.UUID) error {
	return s.q.DeleteIPAddress(ctx, id)
}
