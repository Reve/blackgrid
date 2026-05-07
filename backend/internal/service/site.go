package service

import (
	"context"

	"blackgrid/internal/db"
	"github.com/jackc/pgx/v5/pgtype"
)

type SiteService struct {
	q *db.Queries
}

func NewSiteService(q *db.Queries) *SiteService {
	return &SiteService{q: q}
}

func (s *SiteService) GetSites(ctx context.Context) ([]db.Site, error) {
	return s.q.GetSites(ctx)
}

func (s *SiteService) GetSite(ctx context.Context, id pgtype.UUID) (db.Site, error) {
	return s.q.GetSite(ctx, id)
}

func (s *SiteService) CreateSite(ctx context.Context, req db.CreateSiteParams) (db.Site, error) {
	return s.q.CreateSite(ctx, req)
}

func (s *SiteService) UpdateSite(ctx context.Context, req db.UpdateSiteParams) (db.Site, error) {
	return s.q.UpdateSite(ctx, req)
}

func (s *SiteService) DeleteSite(ctx context.Context, id pgtype.UUID) error {
	return s.q.DeleteSite(ctx, id)
}
