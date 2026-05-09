package service

import (
	"context"
	"errors"
	"fmt"
	"log"

	"blackgrid/internal/db"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
)

var (
	ErrIncidentAlreadyResolved = errors.New("incident already resolved")
	ErrIncidentNotFound        = errors.New("incident not found")
)

// Notifier is the small surface IncidentService needs from notifications.
// Defined here to avoid an import cycle.
type Notifier interface {
	SendIncidentOpened(ctx context.Context, incident db.Incident, monitor db.Monitor)
	SendIncidentResolved(ctx context.Context, incident db.Incident, monitor db.Monitor)
}

type IncidentService struct {
	q        *db.Queries
	notifier Notifier
}

func NewIncidentService(q *db.Queries) *IncidentService {
	return &IncidentService{q: q}
}

func (s *IncidentService) SetNotifier(n Notifier) {
	s.notifier = n
}

// OpenForMonitor opens a new incident for the given monitor if no open or
// acknowledged incident already exists. Idempotent.
func (s *IncidentService) OpenForMonitor(ctx context.Context, monitor db.Monitor, severity, summary, details string) (db.Incident, bool, error) {
	if severity != "critical" && severity != "warning" && severity != "info" {
		return db.Incident{}, false, fmt.Errorf("invalid severity: %s", severity)
	}

	existing, err := s.q.GetOpenIncidentForMonitor(ctx, monitor.ID)
	if err == nil {
		return existing, false, nil
	}
	if !errors.Is(err, pgx.ErrNoRows) {
		return db.Incident{}, false, err
	}

	var detailsCol pgtype.Text
	if details != "" {
		detailsCol = pgtype.Text{String: details, Valid: true}
	}

	incident, err := s.q.CreateIncident(ctx, db.CreateIncidentParams{
		MonitorID: monitor.ID,
		Severity:  severity,
		Summary:   summary,
		Details:   detailsCol,
	})
	if err != nil {
		return db.Incident{}, false, err
	}

	if s.notifier != nil {
		go func(inc db.Incident, m db.Monitor) {
			defer func() {
				if r := recover(); r != nil {
					log.Printf("notifier panic on open: %v", r)
				}
			}()
			s.notifier.SendIncidentOpened(context.Background(), inc, m)
		}(incident, monitor)
	}

	return incident, true, nil
}

// ResolveForMonitor resolves any open or acknowledged incident for the monitor.
// Idempotent: no-op if there's nothing to resolve.
func (s *IncidentService) ResolveForMonitor(ctx context.Context, monitor db.Monitor, reason string) (db.Incident, bool, error) {
	existing, err := s.q.GetOpenIncidentForMonitor(ctx, monitor.ID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return db.Incident{}, false, nil
		}
		return db.Incident{}, false, err
	}

	resolved, err := s.q.ResolveIncident(ctx, existing.ID, reason)
	if err != nil {
		return db.Incident{}, false, err
	}

	if s.notifier != nil {
		go func(inc db.Incident, m db.Monitor) {
			defer func() {
				if r := recover(); r != nil {
					log.Printf("notifier panic on resolve: %v", r)
				}
			}()
			s.notifier.SendIncidentResolved(context.Background(), inc, m)
		}(resolved, monitor)
	}

	return resolved, true, nil
}

func (s *IncidentService) Acknowledge(ctx context.Context, id pgtype.UUID, note string) (db.Incident, error) {
	current, err := s.q.GetIncident(ctx, id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return db.Incident{}, ErrIncidentNotFound
		}
		return db.Incident{}, err
	}

	switch current.Status {
	case "resolved":
		return current, ErrIncidentAlreadyResolved
	case "acknowledged":
		return current, nil
	}

	return s.q.AcknowledgeIncident(ctx, id, note)
}

func (s *IncidentService) Resolve(ctx context.Context, id pgtype.UUID, note string) (db.Incident, error) {
	current, err := s.q.GetIncident(ctx, id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return db.Incident{}, ErrIncidentNotFound
		}
		return db.Incident{}, err
	}

	if current.Status == "resolved" {
		return current, nil
	}

	resolved, err := s.q.ResolveIncident(ctx, id, note)
	if err != nil {
		return db.Incident{}, err
	}

	if s.notifier != nil {
		monitor, mErr := s.q.GetMonitor(ctx, resolved.MonitorID)
		if mErr == nil {
			go func(inc db.Incident, m db.Monitor) {
				defer func() {
					if r := recover(); r != nil {
						log.Printf("notifier panic on resolve: %v", r)
					}
				}()
				s.notifier.SendIncidentResolved(context.Background(), inc, m)
			}(resolved, monitor)
		}
	}

	return resolved, nil
}

func (s *IncidentService) List(ctx context.Context, params db.ListIncidentsParams) ([]db.Incident, error) {
	if params.Limit <= 0 {
		params.Limit = 100
	}
	items, err := s.q.ListIncidents(ctx, params)
	if err != nil {
		return nil, err
	}
	if items == nil {
		items = []db.Incident{}
	}
	return items, nil
}

func (s *IncidentService) Get(ctx context.Context, id pgtype.UUID) (db.Incident, error) {
	inc, err := s.q.GetIncident(ctx, id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return db.Incident{}, ErrIncidentNotFound
		}
		return db.Incident{}, err
	}
	return inc, nil
}

func (s *IncidentService) Counts(ctx context.Context) (db.IncidentCounts, error) {
	return s.q.CountIncidentsByStatus(ctx)
}
