package service

import (
	"context"
	"encoding/json"
	"log"
	"net/netip"

	"blackgrid/internal/db"
	"blackgrid/internal/events"

	"github.com/jackc/pgx/v5/pgtype"
)

// AuditParams describes a single audit event to be persisted.
type AuditParams struct {
	Action      string
	EntityType  string
	EntityID    pgtype.UUID
	ObjectType  string
	ObjectID    pgtype.UUID
	ActorType   string // user, api_token, system
	ActorUserID pgtype.UUID
	ActorTokenID pgtype.UUID
	RequestID   string
	IPAddress   *netip.Addr
	Before      any
	After       any
}

// AuditService persists audit log entries.
type AuditService struct {
	q   *db.Queries
	bus *events.EventBus
}

func NewAuditService(q *db.Queries, bus *events.EventBus) *AuditService {
	return &AuditService{q: q, bus: bus}
}

// Log writes an audit entry, logging but not returning errors so it never blocks the caller.
func (s *AuditService) Log(ctx context.Context, p AuditParams) {
	go func() {
		if err := s.write(context.Background(), p); err != nil {
			log.Printf("audit log write failed: %v", err)
		}
	}()
}

// LogSync writes synchronously — for tests or critical paths.
func (s *AuditService) LogSync(ctx context.Context, p AuditParams) error {
	return s.write(ctx, p)
}

func (s *AuditService) write(ctx context.Context, p AuditParams) error {
	var beforeBytes, afterBytes []byte
	if p.Before != nil {
		b, _ := json.Marshal(p.Before)
		beforeBytes = b
	}
	if p.After != nil {
		b, _ := json.Marshal(p.After)
		afterBytes = b
	}
	objectType := pgtype.Text{}
	if p.ObjectType != "" {
		objectType = pgtype.Text{String: p.ObjectType, Valid: true}
	} else if p.EntityType != "" {
		objectType = pgtype.Text{String: p.EntityType, Valid: true}
	}

	objectID := p.ObjectID
	if !objectID.Valid {
		objectID = p.EntityID
	}

	actorType := pgtype.Text{}
	if p.ActorType != "" {
		actorType = pgtype.Text{String: p.ActorType, Valid: true}
	}

	requestID := pgtype.Text{}
	if p.RequestID != "" {
		requestID = pgtype.Text{String: p.RequestID, Valid: true}
	}

	entityID := p.EntityID
	if !entityID.Valid {
		entityID = p.ObjectID
	}

	_, err := s.q.CreateAuditLog(ctx, db.CreateAuditLogParams{
		Action:          p.Action,
		EntityType:      p.EntityType,
		EntityID:        entityID,
		Changes:         nil,
		ActorUserID:     p.ActorUserID,
		ActorType:       actorType,
		ActorAPITokenID: p.ActorTokenID,
		RequestID:       requestID,
		IPAddress:       p.IPAddress,
		ObjectType:      objectType,
		ObjectID:        objectID,
		BeforeState:     beforeBytes,
		AfterState:      afterBytes,
	})
	if err == nil && s.bus != nil {
		s.bus.Publish(ctx, events.Event{
			Type:       events.AuditEntryCreated,
			ObjectType: "audit_log",
			ObjectID:   events.FormatUUID(objectID),
			Payload: map[string]any{
				"action":      p.Action,
				"entity_type": p.EntityType,
				"actor_type":  p.ActorType,
			},
		})
	}
	return err
}

// ListAuditLogs returns audit log entries with optional filters.
func (s *AuditService) List(ctx context.Context, p db.ListAuditLogsParams) ([]db.AuditLogEntry, error) {
	if p.Limit <= 0 {
		p.Limit = 100
	}
	items, err := s.q.ListAuditLogs(ctx, p)
	if err != nil {
		return nil, err
	}
	if items == nil {
		items = []db.AuditLogEntry{}
	}
	return items, nil
}
