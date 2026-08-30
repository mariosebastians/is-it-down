package scheduler

import (
	"context"
	"log"
	"time"

	"isitdown/internal/checker"
	"isitdown/internal/store"
)

// Scheduler keeps one polling goroutine running per monitor, reconciling
// against the current monitor list on every reconcile tick so monitors
// added, edited, or deleted via the API are picked up without a restart.
type runningMonitor struct {
	cancel  context.CancelFunc
	monitor store.Monitor
}

type Scheduler struct {
	store             *store.Store
	reconcileInterval time.Duration

	running map[string]runningMonitor
}

func New(s *store.Store, reconcileInterval time.Duration) *Scheduler {
	return &Scheduler{
		store:             s,
		reconcileInterval: reconcileInterval,
		running:           make(map[string]runningMonitor),
	}
}

func (s *Scheduler) Run(ctx context.Context) {
	ticker := time.NewTicker(s.reconcileInterval)
	defer ticker.Stop()

	s.reconcile(ctx)
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			s.reconcile(ctx)
		}
	}
}

func (s *Scheduler) reconcile(ctx context.Context) {
	monitors, err := s.store.ListMonitors(ctx)
	if err != nil {
		log.Printf("scheduler: list monitors: %v", err)
		return
	}

	seen := make(map[string]bool, len(monitors))
	for _, m := range monitors {
		seen[m.ID] = true

		if existing, ok := s.running[m.ID]; ok {
			if existing.monitor == m {
				continue
			}
			existing.cancel()
		}

		mctx, cancel := context.WithCancel(ctx)
		s.running[m.ID] = runningMonitor{cancel: cancel, monitor: m}
		go s.pollMonitor(mctx, m)
	}

	for id, rm := range s.running {
		if !seen[id] {
			rm.cancel()
			delete(s.running, id)
		}
	}
}

func (s *Scheduler) pollMonitor(ctx context.Context, m store.Monitor) {
	interval := time.Duration(m.IntervalSeconds) * time.Second
	timeout := time.Duration(m.TimeoutSeconds) * time.Second

	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	s.runCheck(ctx, m, timeout)
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			s.runCheck(ctx, m, timeout)
		}
	}
}

func (s *Scheduler) runCheck(ctx context.Context, m store.Monitor, timeout time.Duration) {
	result := checker.Check(ctx, m.URL, timeout)
	if err := s.store.CreateCheck(ctx, result.ToCheck(m.ID)); err != nil {
		log.Printf("scheduler: record check for monitor %s: %v", m.ID, err)
	}
}
