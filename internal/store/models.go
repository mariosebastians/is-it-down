package store

import "time"

type Monitor struct {
	ID              string    `json:"id"`
	Name            string    `json:"name"`
	URL             string    `json:"url"`
	IntervalSeconds int       `json:"interval_seconds"`
	TimeoutSeconds  int       `json:"timeout_seconds"`
	CreatedAt       time.Time `json:"created_at"`
}

type Check struct {
	ID             string    `json:"id"`
	MonitorID      string    `json:"monitor_id"`
	Status         string    `json:"status"`
	StatusCode     *int      `json:"status_code,omitempty"`
	ResponseTimeMS *int      `json:"response_time_ms,omitempty"`
	Error          *string   `json:"error,omitempty"`
	CheckedAt      time.Time `json:"checked_at"`
}
