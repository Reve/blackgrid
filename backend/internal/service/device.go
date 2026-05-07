package service

import (
	"context"

	"blackgrid/internal/db"
	"github.com/jackc/pgx/v5/pgtype"
)

type DeviceService struct {
	q *db.Queries
}

func NewDeviceService(q *db.Queries) *DeviceService {
	return &DeviceService{q: q}
}

func (s *DeviceService) GetDevices(ctx context.Context) ([]db.Device, error) {
	return s.q.GetDevices(ctx)
}

func (s *DeviceService) GetDevice(ctx context.Context, id pgtype.UUID) (db.Device, error) {
	return s.q.GetDevice(ctx, id)
}

func (s *DeviceService) CreateDevice(ctx context.Context, req db.CreateDeviceParams) (db.Device, error) {
	return s.q.CreateDevice(ctx, req)
}

func (s *DeviceService) UpdateDevice(ctx context.Context, req db.UpdateDeviceParams) (db.Device, error) {
	return s.q.UpdateDevice(ctx, req)
}

func (s *DeviceService) DeleteDevice(ctx context.Context, id pgtype.UUID) error {
	return s.q.DeleteDevice(ctx, id)
}
