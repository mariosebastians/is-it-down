CREATE EXTENSION IF NOT EXISTS pgcrypto;

CREATE TABLE monitors (
    id               UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name             TEXT NOT NULL,
    url              TEXT NOT NULL,
    interval_seconds INTEGER NOT NULL DEFAULT 60,
    timeout_seconds  INTEGER NOT NULL DEFAULT 10,
    created_at       TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE checks (
    id                 UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    monitor_id         UUID NOT NULL REFERENCES monitors(id) ON DELETE CASCADE,
    status             TEXT NOT NULL CHECK (status IN ('up', 'down')),
    status_code        INTEGER,
    response_time_ms   INTEGER,
    error              TEXT,
    checked_at         TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_checks_monitor_id_checked_at ON checks (monitor_id, checked_at DESC);
