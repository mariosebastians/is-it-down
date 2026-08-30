package store

import "time"

type Monitor struct {
	ID              string    `gorm:"type:uuid;default:gen_random_uuid();primaryKey" json:"id"`
	Name            string    `gorm:"not null" json:"name"`
	URL             string    `gorm:"not null" json:"url"`
	IntervalSeconds int       `gorm:"not null;default:60" json:"interval_seconds"`
	TimeoutSeconds  int       `gorm:"not null;default:10" json:"timeout_seconds"`
	CreatedAt       time.Time `gorm:"autoCreateTime" json:"created_at"`

	Checks []Check `gorm:"foreignKey:MonitorID;constraint:OnDelete:CASCADE" json:"-"`
}

type Check struct {
	ID             string    `gorm:"type:uuid;default:gen_random_uuid();primaryKey" json:"id"`
	MonitorID      string    `gorm:"type:uuid;not null;index:idx_checks_monitor_checked_at,priority:1" json:"monitor_id"`
	Status         string    `gorm:"not null" json:"status"`
	StatusCode     *int      `json:"status_code,omitempty"`
	ResponseTimeMS *int      `json:"response_time_ms,omitempty"`
	Error          *string   `json:"error,omitempty"`
	CheckedAt      time.Time `gorm:"column:checked_at;autoCreateTime;index:idx_checks_monitor_checked_at,priority:2,sort:desc" json:"checked_at"`
}
