package service

import (
	"context"

	"blackgrid/internal/db"
	"blackgrid/internal/events"

	"github.com/jackc/pgx/v5/pgtype"
)

type SiteService struct {
	q   *db.Queries
	bus *events.EventBus
}

func NewSiteService(q *db.Queries, bus *events.EventBus) *SiteService {
	return &SiteService{q: q, bus: bus}
}

func (s *SiteService) GetSites(ctx context.Context) ([]db.Site, error) {
	return s.q.GetSites(ctx)
}

func (s *SiteService) GetSite(ctx context.Context, id pgtype.UUID) (db.Site, error) {
	return s.q.GetSite(ctx, id)
}

func (s *SiteService) CreateSite(ctx context.Context, req db.CreateSiteParams) (db.Site, error) {
	site, err := s.q.CreateSite(ctx, req)
	if err == nil && s.bus != nil {
		s.bus.Publish(ctx, events.Event{
			Type:       events.IPAMSiteChanged,
			ObjectType: "site",
			ObjectID:   events.FormatUUID(site.ID),
			Payload: map[string]any{
				"action": "created",
				"name":   site.Name,
			},
		})
	}
	return site, err
}

func (s *SiteService) UpdateSite(ctx context.Context, req db.UpdateSiteParams) (db.Site, error) {
	site, err := s.q.UpdateSite(ctx, req)
	if err == nil && s.bus != nil {
		s.bus.Publish(ctx, events.Event{
			Type:       events.IPAMSiteChanged,
			ObjectType: "site",
			ObjectID:   events.FormatUUID(site.ID),
			Payload: map[string]any{
				"action": "updated",
				"name":   site.Name,
			},
		})
	}
	return site, err
}

func (s *SiteService) DeleteSite(ctx context.Context, id pgtype.UUID) error {
	err := s.q.DeleteSite(ctx, id)
	if err == nil && s.bus != nil {
		s.bus.Publish(ctx, events.Event{
			Type:       events.IPAMSiteChanged,
			ObjectType: "site",
			ObjectID:   events.FormatUUID(id),
			Payload: map[string]any{
				"action": "deleted",
			},
		})
	}
	return err
}
