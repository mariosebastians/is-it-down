package store

import (
	"context"
	"fmt"

	"gorm.io/gorm"
)

type Store struct {
	db *gorm.DB
}

func New(db *gorm.DB) *Store {
	return &Store{db: db}
}

func (s *Store) CreateMonitor(ctx context.Context, m Monitor) (Monitor, error) {
	if err := s.db.WithContext(ctx).Create(&m).Error; err != nil {
		return Monitor{}, fmt.Errorf("create monitor: %w", err)
	}
	return m, nil
}

func (s *Store) ListMonitors(ctx context.Context) ([]Monitor, error) {
	var monitors []Monitor
	if err := s.db.WithContext(ctx).Order("created_at DESC").Find(&monitors).Error; err != nil {
		return nil, fmt.Errorf("list monitors: %w", err)
	}
	return monitors, nil
}

func (s *Store) GetMonitor(ctx context.Context, id string) (Monitor, error) {
	var m Monitor
	if err := s.db.WithContext(ctx).First(&m, "id = ?", id).Error; err != nil {
		return Monitor{}, fmt.Errorf("get monitor: %w", err)
	}
	return m, nil
}

func (s *Store) UpdateMonitor(ctx context.Context, id string, m Monitor) (Monitor, error) {
	updates := map[string]any{
		"name":             m.Name,
		"url":              m.URL,
		"interval_seconds": m.IntervalSeconds,
		"timeout_seconds":  m.TimeoutSeconds,
	}

	result := s.db.WithContext(ctx).Model(&Monitor{}).Where("id = ?", id).Updates(updates)
	if result.Error != nil {
		return Monitor{}, fmt.Errorf("update monitor: %w", result.Error)
	}
	if result.RowsAffected == 0 {
		return Monitor{}, gorm.ErrRecordNotFound
	}
	return s.GetMonitor(ctx, id)
}

func (s *Store) DeleteMonitor(ctx context.Context, id string) error {
	result := s.db.WithContext(ctx).Delete(&Monitor{}, "id = ?", id)
	if result.Error != nil {
		return fmt.Errorf("delete monitor: %w", result.Error)
	}
	if result.RowsAffected == 0 {
		return gorm.ErrRecordNotFound
	}
	return nil
}

func (s *Store) CreateCheck(ctx context.Context, c Check) error {
	if err := s.db.WithContext(ctx).Create(&c).Error; err != nil {
		return fmt.Errorf("create check: %w", err)
	}
	return nil
}

func (s *Store) ListChecks(ctx context.Context, monitorID string, limit int) ([]Check, error) {
	var checks []Check
	err := s.db.WithContext(ctx).
		Where("monitor_id = ?", monitorID).
		Order("checked_at DESC").
		Limit(limit).
		Find(&checks).Error
	if err != nil {
		return nil, fmt.Errorf("list checks: %w", err)
	}
	return checks, nil
}

// LatestCheckByMonitor returns the most recent check for every monitor that
// has at least one, keyed by monitor ID. DISTINCT ON is Postgres-specific,
// so this goes through Raw rather than the query builder.
func (s *Store) LatestCheckByMonitor(ctx context.Context) (map[string]Check, error) {
	var checks []Check
	err := s.db.WithContext(ctx).Raw(`
		SELECT DISTINCT ON (monitor_id)
			id, monitor_id, status, status_code, response_time_ms, error, checked_at
		FROM checks
		ORDER BY monitor_id, checked_at DESC`).Scan(&checks).Error
	if err != nil {
		return nil, fmt.Errorf("latest checks: %w", err)
	}

	out := make(map[string]Check, len(checks))
	for _, c := range checks {
		out[c.MonitorID] = c
	}
	return out, nil
}
