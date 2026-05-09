package service

import (
	"context"
	"log"
	"time"

	"blackgrid/internal/db"
)

type RetentionService struct {
	q *db.Queries
}

func NewRetentionService(q *db.Queries) *RetentionService {
	return &RetentionService{q: q}
}

type RetentionConfig struct {
	MonitorResultsDays       int
	NotificationDeliveriesDays int
	AuditLogDays             int
	DiscoveryResultsDays     int
	DiscoveryScansDays       int
	IntervalHours            int
}

func (s *RetentionService) Start(ctx context.Context, cfg RetentionConfig) {
	interval := time.Duration(cfg.IntervalHours) * time.Hour
	if interval == 0 {
		interval = 24 * time.Hour
	}

	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	// Run once at startup
	s.Run(ctx, cfg)

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			s.Run(ctx, cfg)
		}
	}
}

func (s *RetentionService) Run(ctx context.Context, cfg RetentionConfig) {
	log.Println("Starting data retention cleanup job...")

	if cfg.MonitorResultsDays > 0 {
		count, err := s.q.CleanupMonitorResults(ctx, int32(cfg.MonitorResultsDays))
		if err != nil {
			log.Printf("Failed to cleanup monitor_results: %v", err)
		} else {
			log.Printf("Cleaned up %d monitor_results", count)
		}
	}

	if cfg.NotificationDeliveriesDays > 0 {
		count, err := s.q.CleanupNotificationDeliveries(ctx, int32(cfg.NotificationDeliveriesDays))
		if err != nil {
			log.Printf("Failed to cleanup notification_deliveries: %v", err)
		} else {
			log.Printf("Cleaned up %d notification_deliveries", count)
		}
	}

	if cfg.AuditLogDays > 0 {
		count, err := s.q.CleanupAuditLogs(ctx, int32(cfg.AuditLogDays))
		if err != nil {
			log.Printf("Failed to cleanup audit_log: %v", err)
		} else {
			log.Printf("Cleaned up %d audit_log entries", count)
		}
	}

	if cfg.DiscoveryResultsDays > 0 {
		count, err := s.q.CleanupDiscoveryResults(ctx, int32(cfg.DiscoveryResultsDays))
		if err != nil {
			log.Printf("Failed to cleanup discovery_results: %v", err)
		} else {
			log.Printf("Cleaned up %d discovery_results", count)
		}
	}

	if cfg.DiscoveryScansDays > 0 {
		count, err := s.q.CleanupDiscoveryScans(ctx, int32(cfg.DiscoveryScansDays))
		if err != nil {
			log.Printf("Failed to cleanup discovery_scans: %v", err)
		} else {
			log.Printf("Cleaned up %d discovery_scans", count)
		}
	}

	log.Println("Data retention cleanup job finished.")
}
