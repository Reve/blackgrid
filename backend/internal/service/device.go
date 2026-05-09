package service

import (
	"context"

	"blackgrid/internal/db"
	"blackgrid/internal/events"

	"github.com/jackc/pgx/v5/pgtype"
)

type DeviceService struct {
	q   *db.Queries
	bus *events.EventBus
}

func NewDeviceService(q *db.Queries, bus *events.EventBus) *DeviceService {
	return &DeviceService{q: q, bus: bus}
}

func (s *DeviceService) GetDevices(ctx context.Context) ([]db.Device, error) {
	return s.q.GetDevices(ctx)
}

func (s *DeviceService) GetDevice(ctx context.Context, id pgtype.UUID) (db.Device, error) {
	return s.q.GetDevice(ctx, id)
}

func (s *DeviceService) CreateDevice(ctx context.Context, req db.CreateDeviceParams) (db.Device, error) {
	device, err := s.q.CreateDevice(ctx, req)
	if err == nil && s.bus != nil {
		s.bus.Publish(ctx, events.Event{
			Type:       events.IPAMDeviceChanged,
			ObjectType: "device",
			ObjectID:   events.FormatUUID(device.ID),
			Payload: map[string]any{
				"action": "created",
				"name":   device.Hostname,
			},
		})
	}
	return device, err
}

func (s *DeviceService) UpdateDevice(ctx context.Context, req db.UpdateDeviceParams) (db.Device, error) {
	device, err := s.q.UpdateDevice(ctx, req)
	if err == nil && s.bus != nil {
		s.bus.Publish(ctx, events.Event{
			Type:       events.IPAMDeviceChanged,
			ObjectType: "device",
			ObjectID:   events.FormatUUID(device.ID),
			Payload: map[string]any{
				"action": "updated",
				"name":   device.Hostname,
			},
		})
	}
	return device, err
}

func (s *DeviceService) DeleteDevice(ctx context.Context, id pgtype.UUID) error {
	err := s.q.DeleteDevice(ctx, id)
	if err == nil && s.bus != nil {
		s.bus.Publish(ctx, events.Event{
			Type:       events.IPAMDeviceChanged,
			ObjectType: "device",
			ObjectID:   events.FormatUUID(id),
			Payload: map[string]any{
				"action": "deleted",
			},
		})
	}
	return err
}
