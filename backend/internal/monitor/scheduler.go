package monitor

import (
	"context"
	"log"
	"sync"
	"time"

	"blackgrid/internal/db"
	"github.com/jackc/pgx/v5/pgtype"
)

// IncidentHook is invoked after a scheduled monitor check completes so that
// callers (e.g. the incident service) can react to status transitions.
type IncidentHook interface {
	OnScheduledStatusChange(ctx context.Context, monitor db.Monitor, oldStatus, newStatus string)
}

type Scheduler struct {
	queries     *db.Queries
	runner      *Runner
	workerCount int
	stopChan    chan struct{}
	wg          sync.WaitGroup
	hook        IncidentHook
}

func NewScheduler(queries *db.Queries, runner *Runner, workerCount int) *Scheduler {
	if workerCount <= 0 {
		workerCount = 10
	}
	return &Scheduler{
		queries:     queries,
		runner:      runner,
		workerCount: workerCount,
		stopChan:    make(chan struct{}),
	}
}

func (s *Scheduler) SetIncidentHook(h IncidentHook) {
	s.hook = h
}

func (s *Scheduler) Start() {
	s.wg.Add(1)
	go s.loop()
}

func (s *Scheduler) Stop() {
	close(s.stopChan)
	s.wg.Wait()
}

func (s *Scheduler) loop() {
	defer s.wg.Done()
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	// In-progress map to avoid duplicate checks
	inProgress := make(map[pgtype.UUID]bool)
	var mu sync.Mutex

	jobsChan := make(chan db.Monitor, s.workerCount*2)

	// Start workers
	for i := 0; i < s.workerCount; i++ {
		s.wg.Add(1)
		go s.worker(jobsChan, inProgress, &mu)
	}

	for {
		select {
		case <-s.stopChan:
			close(jobsChan)
			return
		case <-ticker.C:
			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			monitors, err := s.queries.GetMonitorsDueForCheck(ctx)
			cancel()

			if err != nil {
				log.Printf("Scheduler error fetching due monitors: %v", err)
				continue
			}

			for _, m := range monitors {
				mu.Lock()
				if !inProgress[m.ID] {
					inProgress[m.ID] = true
					mu.Unlock()

					// Send job to worker pool, non-blocking if possible
					select {
					case jobsChan <- m:
					default:
						// If channel is full, drop it for this cycle, it will be picked up next time
						mu.Lock()
						delete(inProgress, m.ID)
						mu.Unlock()
					}
				} else {
					mu.Unlock()
				}
			}
		}
	}
}

func (s *Scheduler) worker(jobsChan <-chan db.Monitor, inProgress map[pgtype.UUID]bool, mu *sync.Mutex) {
	defer s.wg.Done()

	for m := range jobsChan {
		s.executeCheck(m)

		mu.Lock()
		delete(inProgress, m.ID)
		mu.Unlock()
	}
}

func (s *Scheduler) executeCheck(m db.Monitor) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(m.TimeoutSeconds+5)*time.Second)
	defer cancel()

	result, err := s.runner.Run(ctx, m)
	if err != nil {
		log.Printf("Monitor %s (%s) check error: %v", m.ID, m.Name, err)
		return
	}

	// Calculate state transitions
	newStatus := result.Status

	// Handle retry logic for down status
	if newStatus == "down" && (m.Status == "up" || m.Status == "degraded" || m.Status == "unknown") {
		// Calculate how many failures have occurred consecutively
		results, _ := s.queries.GetMonitorResults(ctx, db.GetMonitorResultsParams{
			MonitorID: m.ID,
			Limit:     m.RetryCount,
			Offset:    0,
		})

		failures := 0
		for _, r := range results {
			if r.Status == "down" {
				failures++
			} else {
				break
			}
		}

		if int32(failures) < m.RetryCount {
			newStatus = "degraded"
		}
	}

	lastStatusChangeAt := m.LastStatusChangeAt
	if newStatus != m.Status {
		now := pgtype.Timestamptz{Time: time.Now(), Valid: true}
		lastStatusChangeAt = now
	}

	now := pgtype.Timestamptz{Time: time.Now(), Valid: true}

	updated, err := s.queries.UpdateMonitor(ctx, db.UpdateMonitorParams{
		ID:                 m.ID,
		Name:               m.Name,
		Slug:               m.Slug,
		MonitorType:        m.MonitorType,
		Target:             m.Target,
		Config:             m.Config,
		IpAddressID:        m.IpAddressID,
		DeviceID:           m.DeviceID,
		IntervalSeconds:    m.IntervalSeconds,
		TimeoutSeconds:     m.TimeoutSeconds,
		RetryCount:         m.RetryCount,
		Enabled:            m.Enabled,
		Status:             newStatus,
		LastCheckedAt:      now,
		LastStatusChangeAt: lastStatusChangeAt,
		PushTokenHash:      m.PushTokenHash,
	})

	if err != nil {
		log.Printf("Monitor %s update state error: %v", m.ID, err)
		return
	}

	if s.hook != nil && newStatus != m.Status {
		hookCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		s.hook.OnScheduledStatusChange(hookCtx, updated, m.Status, newStatus)
	}
}
