package store

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Store struct {
	pool *pgxpool.Pool
}

func New(pool *pgxpool.Pool) *Store {
	return &Store{pool: pool}
}

func (s *Store) CreateMonitor(ctx context.Context, m Monitor) (Monitor, error) {
	const q = `
		INSERT INTO monitors (name, url, interval_seconds, timeout_seconds)
		VALUES ($1, $2, $3, $4)
		RETURNING id, name, url, interval_seconds, timeout_seconds, created_at`

	var out Monitor
	err := s.pool.QueryRow(ctx, q, m.Name, m.URL, m.IntervalSeconds, m.TimeoutSeconds).Scan(
		&out.ID, &out.Name, &out.URL, &out.IntervalSeconds, &out.TimeoutSeconds, &out.CreatedAt,
	)
	if err != nil {
		return Monitor{}, fmt.Errorf("create monitor: %w", err)
	}
	return out, nil
}

func (s *Store) ListMonitors(ctx context.Context) ([]Monitor, error) {
	const q = `
		SELECT id, name, url, interval_seconds, timeout_seconds, created_at
		FROM monitors
		ORDER BY created_at DESC`

	rows, err := s.pool.Query(ctx, q)
	if err != nil {
		return nil, fmt.Errorf("list monitors: %w", err)
	}
	defer rows.Close()

	monitors, err := pgx.CollectRows(rows, pgx.RowToStructByName[Monitor])
	if err != nil {
		return nil, fmt.Errorf("scan monitors: %w", err)
	}
	return monitors, nil
}

func (s *Store) GetMonitor(ctx context.Context, id string) (Monitor, error) {
	const q = `
		SELECT id, name, url, interval_seconds, timeout_seconds, created_at
		FROM monitors
		WHERE id = $1`

	var out Monitor
	err := s.pool.QueryRow(ctx, q, id).Scan(
		&out.ID, &out.Name, &out.URL, &out.IntervalSeconds, &out.TimeoutSeconds, &out.CreatedAt,
	)
	if err != nil {
		return Monitor{}, fmt.Errorf("get monitor: %w", err)
	}
	return out, nil
}

func (s *Store) UpdateMonitor(ctx context.Context, id string, m Monitor) (Monitor, error) {
	const q = `
		UPDATE monitors
		SET name = $2, url = $3, interval_seconds = $4, timeout_seconds = $5
		WHERE id = $1
		RETURNING id, name, url, interval_seconds, timeout_seconds, created_at`

	var out Monitor
	err := s.pool.QueryRow(ctx, q, id, m.Name, m.URL, m.IntervalSeconds, m.TimeoutSeconds).Scan(
		&out.ID, &out.Name, &out.URL, &out.IntervalSeconds, &out.TimeoutSeconds, &out.CreatedAt,
	)
	if err != nil {
		return Monitor{}, fmt.Errorf("update monitor: %w", err)
	}
	return out, nil
}

func (s *Store) DeleteMonitor(ctx context.Context, id string) error {
	const q = `DELETE FROM monitors WHERE id = $1`

	tag, err := s.pool.Exec(ctx, q, id)
	if err != nil {
		return fmt.Errorf("delete monitor: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return pgx.ErrNoRows
	}
	return nil
}

func (s *Store) CreateCheck(ctx context.Context, c Check) error {
	const q = `
		INSERT INTO checks (monitor_id, status, status_code, response_time_ms, error)
		VALUES ($1, $2, $3, $4, $5)`

	_, err := s.pool.Exec(ctx, q, c.MonitorID, c.Status, c.StatusCode, c.ResponseTimeMS, c.Error)
	if err != nil {
		return fmt.Errorf("create check: %w", err)
	}
	return nil
}

func (s *Store) ListChecks(ctx context.Context, monitorID string, limit int) ([]Check, error) {
	const q = `
		SELECT id, monitor_id, status, status_code, response_time_ms, error, checked_at
		FROM checks
		WHERE monitor_id = $1
		ORDER BY checked_at DESC
		LIMIT $2`

	rows, err := s.pool.Query(ctx, q, monitorID, limit)
	if err != nil {
		return nil, fmt.Errorf("list checks: %w", err)
	}
	defer rows.Close()

	checks, err := pgx.CollectRows(rows, pgx.RowToStructByName[Check])
	if err != nil {
		return nil, fmt.Errorf("scan checks: %w", err)
	}
	return checks, nil
}

func (s *Store) LatestCheckByMonitor(ctx context.Context) (map[string]Check, error) {
	const q = `
		SELECT DISTINCT ON (monitor_id)
			id, monitor_id, status, status_code, response_time_ms, error, checked_at
		FROM checks
		ORDER BY monitor_id, checked_at DESC`

	rows, err := s.pool.Query(ctx, q)
	if err != nil {
		return nil, fmt.Errorf("latest checks: %w", err)
	}
	defer rows.Close()

	checks, err := pgx.CollectRows(rows, pgx.RowToStructByName[Check])
	if err != nil {
		return nil, fmt.Errorf("scan latest checks: %w", err)
	}

	out := make(map[string]Check, len(checks))
	for _, c := range checks {
		out[c.MonitorID] = c
	}
	return out, nil
}
