package service

import (
	"context"
	"errors"
	"fmt"
	"regexp"
	"strings"
	"time"

	"blackgrid/internal/db"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
)

var (
	ErrStatusPageNotFound       = errors.New("status page not found")
	ErrStatusPageDuplicateSlug  = errors.New("status page slug already in use")
	ErrStatusPageInvalidSlug    = errors.New("status page slug must only contain lowercase letters, numbers, and hyphens")
	ErrStatusPageNameRequired   = errors.New("status page name is required")
	ErrMonitorAlreadyAttached   = errors.New("monitor already attached to status page")
	ErrMonitorNotAttached       = errors.New("monitor not attached to status page")
	ErrReorderMonitorMismatched = errors.New("reorder monitor list does not match attached monitors")
)

var slugRegex = regexp.MustCompile(`^[a-z0-9]+(?:-[a-z0-9]+)*$`)

type StatusPageService struct {
	q *db.Queries
}

func NewStatusPageService(q *db.Queries) *StatusPageService {
	return &StatusPageService{q: q}
}

// ----- DTOs -----

type StatusPageInput struct {
	Name          string
	Slug          string
	Description   *string
	Public        *bool
	ShowUptime    *bool
	ShowIncidents *bool
}

type AttachMonitorInput struct {
	MonitorID    pgtype.UUID
	DisplayName  *string
	DisplayOrder *int32
}

type UpdateAttachedMonitorInput struct {
	DisplayName  *string
	DisplayOrder *int32
}

// AttachedMonitor pairs a monitor with its status_page_monitor metadata.
type AttachedMonitor struct {
	Monitor      db.Monitor          `json:"monitor"`
	DisplayName  pgtype.Text         `json:"display_name"`
	DisplayOrder int32               `json:"display_order"`
	CreatedAt    pgtype.Timestamptz  `json:"created_at"`
}

// AdminStatusPage is the admin-facing status page representation, including monitors.
type AdminStatusPage struct {
	Page     db.StatusPage     `json:"page"`
	Monitors []AttachedMonitor `json:"monitors"`
}

// PublicStatusMonitor is a public-safe single-service entry.
type PublicStatusMonitor struct {
	DisplayName   string     `json:"display_name"`
	MonitorType   string     `json:"monitor_type"`
	Status        string     `json:"status"`
	LastCheckedAt *time.Time `json:"last_checked_at"`
	Uptime24h     *float64   `json:"uptime_24h"`
	Uptime30d     *float64   `json:"uptime_30d"`
}

// PublicIncident is a public-safe incident summary.
type PublicIncident struct {
	MonitorDisplayName string     `json:"monitor_display_name"`
	Severity           string     `json:"severity"`
	Status             string     `json:"status"`
	StartedAt          *time.Time `json:"started_at"`
	ResolvedAt         *time.Time `json:"resolved_at"`
	Summary            string     `json:"summary"`
}

// PublicStatusPage is the public-safe page response.
type PublicStatusPage struct {
	Name            string                `json:"name"`
	Slug            string                `json:"slug"`
	Description     string                `json:"description"`
	AggregateStatus string                `json:"aggregate_status"`
	Monitors        []PublicStatusMonitor `json:"monitors"`
	Incidents       []PublicIncident      `json:"incidents,omitempty"`
}

// ----- Slug helpers -----

func generateSlugFromName(name string) string {
	s := strings.ToLower(strings.TrimSpace(name))
	var b strings.Builder
	prevDash := false
	for _, r := range s {
		switch {
		case (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9'):
			b.WriteRune(r)
			prevDash = false
		case r == '-' || r == ' ' || r == '_':
			if !prevDash && b.Len() > 0 {
				b.WriteByte('-')
				prevDash = true
			}
		}
	}
	return strings.Trim(b.String(), "-")
}

func validateSlug(slug string) error {
	if slug == "" {
		return ErrStatusPageInvalidSlug
	}
	if !slugRegex.MatchString(slug) {
		return ErrStatusPageInvalidSlug
	}
	return nil
}

// ----- CRUD -----

func (s *StatusPageService) CreateStatusPage(ctx context.Context, in StatusPageInput) (db.StatusPage, error) {
	name := strings.TrimSpace(in.Name)
	if name == "" {
		return db.StatusPage{}, ErrStatusPageNameRequired
	}
	slug := strings.TrimSpace(in.Slug)
	if slug == "" {
		slug = generateSlugFromName(name)
	}
	if err := validateSlug(slug); err != nil {
		return db.StatusPage{}, err
	}

	desc := pgtype.Text{}
	if in.Description != nil && strings.TrimSpace(*in.Description) != "" {
		desc = pgtype.Text{String: *in.Description, Valid: true}
	}
	public := false
	if in.Public != nil {
		public = *in.Public
	}
	showUptime := true
	if in.ShowUptime != nil {
		showUptime = *in.ShowUptime
	}
	showIncidents := true
	if in.ShowIncidents != nil {
		showIncidents = *in.ShowIncidents
	}

	page, err := s.q.CreateStatusPage(ctx, db.CreateStatusPageParams{
		Name:          name,
		Slug:          slug,
		Description:   desc,
		Public:        public,
		ShowUptime:    showUptime,
		ShowIncidents: showIncidents,
	})
	if err != nil {
		if isUniqueViolation(err) {
			return db.StatusPage{}, ErrStatusPageDuplicateSlug
		}
		return db.StatusPage{}, err
	}
	return page, nil
}

func (s *StatusPageService) UpdateStatusPage(ctx context.Context, id pgtype.UUID, in StatusPageInput) (db.StatusPage, error) {
	current, err := s.q.GetStatusPage(ctx, id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return db.StatusPage{}, ErrStatusPageNotFound
		}
		return db.StatusPage{}, err
	}

	name := current.Name
	if strings.TrimSpace(in.Name) != "" {
		name = strings.TrimSpace(in.Name)
	}

	slug := current.Slug
	if strings.TrimSpace(in.Slug) != "" {
		slug = strings.TrimSpace(in.Slug)
		if err := validateSlug(slug); err != nil {
			return db.StatusPage{}, err
		}
	}

	desc := current.Description
	if in.Description != nil {
		if strings.TrimSpace(*in.Description) == "" {
			desc = pgtype.Text{}
		} else {
			desc = pgtype.Text{String: *in.Description, Valid: true}
		}
	}

	public := current.Public
	if in.Public != nil {
		public = *in.Public
	}
	showUptime := current.ShowUptime
	if in.ShowUptime != nil {
		showUptime = *in.ShowUptime
	}
	showIncidents := current.ShowIncidents
	if in.ShowIncidents != nil {
		showIncidents = *in.ShowIncidents
	}

	updated, err := s.q.UpdateStatusPage(ctx, db.UpdateStatusPageParams{
		ID:            id,
		Name:          name,
		Slug:          slug,
		Description:   desc,
		Public:        public,
		ShowUptime:    showUptime,
		ShowIncidents: showIncidents,
	})
	if err != nil {
		if isUniqueViolation(err) {
			return db.StatusPage{}, ErrStatusPageDuplicateSlug
		}
		return db.StatusPage{}, err
	}
	return updated, nil
}

func (s *StatusPageService) DeleteStatusPage(ctx context.Context, id pgtype.UUID) error {
	if _, err := s.q.GetStatusPage(ctx, id); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return ErrStatusPageNotFound
		}
		return err
	}
	return s.q.DeleteStatusPage(ctx, id)
}

func (s *StatusPageService) ListStatusPages(ctx context.Context) ([]db.StatusPage, error) {
	items, err := s.q.ListStatusPages(ctx)
	if err != nil {
		return nil, err
	}
	if items == nil {
		items = []db.StatusPage{}
	}
	return items, nil
}

func (s *StatusPageService) GetAdminStatusPage(ctx context.Context, id pgtype.UUID) (AdminStatusPage, error) {
	page, err := s.q.GetStatusPage(ctx, id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return AdminStatusPage{}, ErrStatusPageNotFound
		}
		return AdminStatusPage{}, err
	}

	links, err := s.q.ListStatusPageMonitors(ctx, id)
	if err != nil {
		return AdminStatusPage{}, err
	}

	attached := make([]AttachedMonitor, 0, len(links))
	for _, l := range links {
		mon, err := s.q.GetMonitor(ctx, l.MonitorID)
		if err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				continue
			}
			return AdminStatusPage{}, err
		}
		attached = append(attached, AttachedMonitor{
			Monitor:      mon,
			DisplayName:  l.DisplayName,
			DisplayOrder: l.DisplayOrder,
			CreatedAt:    l.CreatedAt,
		})
	}

	return AdminStatusPage{Page: page, Monitors: attached}, nil
}

// ----- Monitor attachment -----

func (s *StatusPageService) AttachMonitor(ctx context.Context, pageID pgtype.UUID, in AttachMonitorInput) (db.StatusPageMonitor, error) {
	if _, err := s.q.GetStatusPage(ctx, pageID); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return db.StatusPageMonitor{}, ErrStatusPageNotFound
		}
		return db.StatusPageMonitor{}, err
	}

	if _, err := s.q.GetStatusPageMonitor(ctx, pageID, in.MonitorID); err == nil {
		return db.StatusPageMonitor{}, ErrMonitorAlreadyAttached
	} else if !errors.Is(err, pgx.ErrNoRows) {
		return db.StatusPageMonitor{}, err
	}

	displayName := pgtype.Text{}
	if in.DisplayName != nil && strings.TrimSpace(*in.DisplayName) != "" {
		displayName = pgtype.Text{String: *in.DisplayName, Valid: true}
	}

	var order int32
	if in.DisplayOrder != nil {
		order = *in.DisplayOrder
	} else {
		max, err := s.q.MaxStatusPageMonitorOrder(ctx, pageID)
		if err != nil {
			return db.StatusPageMonitor{}, err
		}
		order = max + 10
	}

	link, err := s.q.AttachStatusPageMonitor(ctx, db.AttachStatusPageMonitorParams{
		StatusPageID: pageID,
		MonitorID:    in.MonitorID,
		DisplayName:  displayName,
		DisplayOrder: order,
	})
	if err != nil {
		if isUniqueViolation(err) {
			return db.StatusPageMonitor{}, ErrMonitorAlreadyAttached
		}
		return db.StatusPageMonitor{}, err
	}
	return link, nil
}

func (s *StatusPageService) UpdateAttachedMonitor(ctx context.Context, pageID, monitorID pgtype.UUID, in UpdateAttachedMonitorInput) (db.StatusPageMonitor, error) {
	current, err := s.q.GetStatusPageMonitor(ctx, pageID, monitorID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return db.StatusPageMonitor{}, ErrMonitorNotAttached
		}
		return db.StatusPageMonitor{}, err
	}

	displayName := current.DisplayName
	if in.DisplayName != nil {
		if strings.TrimSpace(*in.DisplayName) == "" {
			displayName = pgtype.Text{}
		} else {
			displayName = pgtype.Text{String: *in.DisplayName, Valid: true}
		}
	}

	order := current.DisplayOrder
	if in.DisplayOrder != nil {
		order = *in.DisplayOrder
	}

	return s.q.UpdateStatusPageMonitor(ctx, db.UpdateStatusPageMonitorParams{
		StatusPageID: pageID,
		MonitorID:    monitorID,
		DisplayName:  displayName,
		DisplayOrder: order,
	})
}

func (s *StatusPageService) RemoveMonitor(ctx context.Context, pageID, monitorID pgtype.UUID) error {
	if _, err := s.q.GetStatusPageMonitor(ctx, pageID, monitorID); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return ErrMonitorNotAttached
		}
		return err
	}
	return s.q.RemoveStatusPageMonitor(ctx, pageID, monitorID)
}

// ReorderMonitors sets display_order to 10, 20, 30, ... according to the
// provided list. Every supplied monitor ID must currently be attached;
// any unattached ID causes the entire reorder to be rejected.
func (s *StatusPageService) ReorderMonitors(ctx context.Context, pageID pgtype.UUID, monitorIDs []pgtype.UUID) error {
	if _, err := s.q.GetStatusPage(ctx, pageID); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return ErrStatusPageNotFound
		}
		return err
	}
	for _, mid := range monitorIDs {
		if _, err := s.q.GetStatusPageMonitor(ctx, pageID, mid); err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				return ErrReorderMonitorMismatched
			}
			return err
		}
	}
	for i, mid := range monitorIDs {
		current, err := s.q.GetStatusPageMonitor(ctx, pageID, mid)
		if err != nil {
			return err
		}
		_, err = s.q.UpdateStatusPageMonitor(ctx, db.UpdateStatusPageMonitorParams{
			StatusPageID: pageID,
			MonitorID:    mid,
			DisplayName:  current.DisplayName,
			DisplayOrder: int32((i + 1) * 10),
		})
		if err != nil {
			return err
		}
	}
	return nil
}

// ----- Aggregate / uptime -----

// ComputeAggregateStatus returns a status string computed from a list of
// monitor statuses ("up", "down", "degraded", "unknown", "paused", ...).
func ComputeAggregateStatus(statuses []string) string {
	if len(statuses) == 0 {
		return "empty"
	}
	hasDown := false
	hasDegraded := false
	allUp := true
	for _, s := range statuses {
		switch s {
		case "down":
			hasDown = true
			allUp = false
		case "degraded", "unknown":
			hasDegraded = true
			allUp = false
		case "up":
			// no-op
		default:
			// paused etc — treat as not-up but not-degraded
			allUp = false
		}
	}
	if hasDown {
		return "down"
	}
	if hasDegraded {
		return "degraded"
	}
	if allUp {
		return "up"
	}
	return "degraded"
}

// ComputeMonitorUptime returns the uptime percentage over the given window in
// seconds, or nil if there are no results.
func (s *StatusPageService) ComputeMonitorUptime(ctx context.Context, monitorID pgtype.UUID, windowSeconds int64) (*float64, error) {
	c, err := s.q.MonitorUptimeWindow(ctx, monitorID, windowSeconds)
	if err != nil {
		return nil, err
	}
	if c.TotalCount == 0 {
		return nil, nil
	}
	pct := (float64(c.UpCount) / float64(c.TotalCount)) * 100.0
	return &pct, nil
}

// GetPublicStatusPage returns the public-safe view, or ErrStatusPageNotFound
// if the slug does not exist OR the page is private.
func (s *StatusPageService) GetPublicStatusPage(ctx context.Context, slug string) (PublicStatusPage, error) {
	page, err := s.q.GetStatusPageBySlug(ctx, slug)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return PublicStatusPage{}, ErrStatusPageNotFound
		}
		return PublicStatusPage{}, err
	}
	if !page.Public {
		// Treat private as not-found to avoid leaking existence.
		return PublicStatusPage{}, ErrStatusPageNotFound
	}

	links, err := s.q.ListStatusPageMonitors(ctx, page.ID)
	if err != nil {
		return PublicStatusPage{}, err
	}

	monitorByID := map[string]db.Monitor{}
	monitorDisplayName := map[string]string{}
	statuses := make([]string, 0, len(links))
	publicMonitors := make([]PublicStatusMonitor, 0, len(links))
	monitorIDs := make([]pgtype.UUID, 0, len(links))

	for _, l := range links {
		mon, err := s.q.GetMonitor(ctx, l.MonitorID)
		if err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				continue
			}
			return PublicStatusPage{}, err
		}
		display := mon.Name
		if l.DisplayName.Valid && strings.TrimSpace(l.DisplayName.String) != "" {
			display = l.DisplayName.String
		}
		monitorByID[uuidKey(mon.ID)] = mon
		monitorDisplayName[uuidKey(mon.ID)] = display
		monitorIDs = append(monitorIDs, mon.ID)
		statuses = append(statuses, mon.Status)

		entry := PublicStatusMonitor{
			DisplayName:   display,
			MonitorType:   mon.MonitorType,
			Status:        mon.Status,
			LastCheckedAt: timestamptzPtr(mon.LastCheckedAt),
		}
		if page.ShowUptime {
			u24, err := s.ComputeMonitorUptime(ctx, mon.ID, 24*60*60)
			if err != nil {
				return PublicStatusPage{}, err
			}
			u30d, err := s.ComputeMonitorUptime(ctx, mon.ID, 30*24*60*60)
			if err != nil {
				return PublicStatusPage{}, err
			}
			entry.Uptime24h = u24
			entry.Uptime30d = u30d
		}
		publicMonitors = append(publicMonitors, entry)
	}

	resp := PublicStatusPage{
		Name:            page.Name,
		Slug:            page.Slug,
		AggregateStatus: ComputeAggregateStatus(statuses),
		Monitors:        publicMonitors,
	}
	if page.Description.Valid {
		resp.Description = page.Description.String
	}

	if page.ShowIncidents && len(monitorIDs) > 0 {
		since := pgtype.Timestamptz{Time: time.Now().Add(-30 * 24 * time.Hour), Valid: true}
		incs, err := s.q.ListIncidentsForMonitorsSince(ctx, monitorIDs, since, 50)
		if err != nil {
			return PublicStatusPage{}, err
		}
		publicIncs := make([]PublicIncident, 0, len(incs))
		for _, i := range incs {
			name := monitorDisplayName[uuidKey(i.MonitorID)]
			if name == "" {
				name = "service"
			}
			publicIncs = append(publicIncs, PublicIncident{
				MonitorDisplayName: name,
				Severity:           i.Severity,
				Status:             i.Status,
				StartedAt:          timestamptzPtr(i.StartedAt),
				ResolvedAt:         timestamptzPtr(i.ResolvedAt),
				Summary:            i.Summary,
			})
		}
		resp.Incidents = publicIncs
	}

	return resp, nil
}

// ----- helpers -----

func timestamptzPtr(t pgtype.Timestamptz) *time.Time {
	if !t.Valid {
		return nil
	}
	v := t.Time
	return &v
}

func uuidKey(u pgtype.UUID) string {
	return fmt.Sprintf("%x", u.Bytes)
}

// isUniqueViolation matches pgx unique violation errors without importing the
// pgconn package directly here (kept loose to avoid an extra dep edge).
func isUniqueViolation(err error) bool {
	if err == nil {
		return false
	}
	msg := err.Error()
	return strings.Contains(msg, "unique constraint") || strings.Contains(msg, "23505")
}
