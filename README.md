# is-it-down

Self-hosted uptime and status monitoring service. Polls a set of HTTP endpoints on independent schedules, records response time and status history, and exposes a status feed you can build a public status page on top of.

[![Go Version](https://img.shields.io/badge/go-1.27-00ADD8?logo=go&logoColor=white)](https://go.dev)
[![PostgreSQL](https://img.shields.io/badge/postgres-16-336791?logo=postgresql&logoColor=white)](https://www.postgresql.org)
[![License: MIT](https://img.shields.io/badge/license-MIT-blue.svg)](LICENSE)

## Table of Contents

- [Overview](#overview)
- [Architecture](#architecture)
- [Tech Stack](#tech-stack)
- [Prerequisites](#prerequisites)
- [Getting Started](#getting-started)
- [Configuration](#configuration)
- [Project Structure](#project-structure)
- [API Reference](#api-reference)
- [Data Model](#data-model)
- [Development](#development)
- [Roadmap](#roadmap)
- [License](#license)

## Overview

`is-it-down` is made of two independent processes sharing one Postgres database:

| Process    | Responsibility                                                                 |
|------------|----------------------------------------------------------------------------------|
| **api**    | REST API for managing monitors and reading check history / status              |
| **worker** | Runs one polling goroutine per monitor and records the result of every check    |

Monitors are polled concurrently and independently, each on its own interval. The worker reconciles its running goroutines against the database every 10 seconds, so creating, editing, or deleting a monitor through the API takes effect without restarting the worker.

## Architecture

```
                     ┌─────────────┐
   HTTP clients ───▶ │   api       │
                     │ (REST CRUD) │
                     └──────┬──────┘
                            │
                            ▼
                     ┌─────────────┐
                     │  PostgreSQL │◀──────┐
                     │  monitors   │       │
                     │  checks     │       │
                     └─────────────┘       │
                                            │
                     ┌─────────────┐       │
                     │   worker    │───────┘
                     │ (scheduler) │  writes check results
                     └──────┬──────┘
                            │  concurrent HTTP probes
                            ▼
                   external endpoints
                   being monitored
```

## Tech Stack

| Layer          | Choice                                                        |
|----------------|-----------------------------------------------------------------|
| Language       | Go 1.27                                                        |
| Database       | PostgreSQL 16                                                  |
| DB driver      | [pgx/v5](https://github.com/jackc/pgx) (`pgxpool`)             |
| HTTP routing   | Standard library `net/http` (Go 1.22+ method/path patterns)    |
| Concurrency    | Goroutines + `context` per monitor, ticker-driven scheduling   |
| Local infra    | Docker Compose (Postgres)                                      |

## Prerequisites

- [Go 1.27+](https://go.dev/dl/)
- [Docker Desktop](https://www.docker.com/products/docker-desktop/) (for local Postgres)

## Getting Started

1. **Clone and configure environment variables**

   ```bash
   git clone https://github.com/mariosebastians/is-it-down.git
   cd is-it-down
   cp .env.example .env
   ```

2. **Start Postgres**

   ```bash
   docker compose up -d db
   ```

   The schema in [`migrations/0001_init.sql`](migrations/0001_init.sql) is applied automatically the first time the container initializes its data volume.

3. **Run the API**

   ```bash
   go run ./cmd/api
   ```

   Listens on `:8080` by default. Verify with:

   ```bash
   curl http://localhost:8080/healthz
   ```

4. **Run the worker** (in a separate terminal)

   ```bash
   go run ./cmd/worker
   ```

5. **Create a monitor and watch it get polled**

   ```bash
   curl -X POST http://localhost:8080/monitors \
     -H "Content-Type: application/json" \
     -d '{"name":"Example","url":"https://example.com","interval_seconds":30,"timeout_seconds":5}'

   curl http://localhost:8080/status
   ```

## Configuration

Both `api` and `worker` read the same environment variables (loaded from `.env` via [godotenv](https://github.com/joho/godotenv) if present):

| Variable       | Description                              | Default                                                              |
|----------------|-------------------------------------------|------------------------------------------------------------------------|
| `DATABASE_URL` | Postgres connection string                | `postgres://isitdown:isitdown@localhost:5432/isitdown?sslmode=disable` |
| `API_ADDR`     | Address the API server listens on         | `:8080`                                                               |

## Project Structure

```
.
├── cmd/
│   ├── api/            # API server entrypoint
│   └── worker/          # Polling worker entrypoint
├── internal/
│   ├── checker/         # Single HTTP probe logic
│   ├── config/           # Environment configuration loading
│   ├── db/                # Postgres connection pool setup
│   ├── httpapi/          # HTTP routes and handlers
│   ├── scheduler/       # Per-monitor concurrent polling scheduler
│   └── store/             # Data access layer (monitors, checks)
├── migrations/           # SQL schema, auto-applied on first DB boot
├── docker-compose.yml   # Local Postgres service
└── .env.example          # Environment variable template
```

## API Reference

Base URL: `http://localhost:8080`

| Method   | Path                     | Description                                   |
|----------|--------------------------|------------------------------------------------|
| `GET`    | `/healthz`               | Liveness check                                |
| `GET`    | `/status`                | All monitors with their latest check result   |
| `POST`   | `/monitors`               | Create a monitor                              |
| `GET`    | `/monitors`               | List all monitors                             |
| `GET`    | `/monitors/{id}`          | Get a single monitor                          |
| `PUT`    | `/monitors/{id}`          | Update a monitor                              |
| `DELETE` | `/monitors/{id}`          | Delete a monitor (cascades its check history)|
| `GET`    | `/monitors/{id}/checks`   | List recent checks for a monitor (last 50)   |

**Create/update monitor body:**

```json
{
  "name": "Example",
  "url": "https://example.com",
  "interval_seconds": 60,
  "timeout_seconds": 10
}
```

## Data Model

**monitors** — the endpoints being watched

| Column             | Type        |
|--------------------|-------------|
| `id`                 | UUID (PK)  |
| `name`              | TEXT       |
| `url`                | TEXT       |
| `interval_seconds` | INTEGER     |
| `timeout_seconds`  | INTEGER     |
| `created_at`        | TIMESTAMPTZ |

**checks** — one row per poll of a monitor

| Column               | Type                    |
|----------------------|--------------------------|
| `id`                    | UUID (PK)              |
| `monitor_id`          | UUID (FK → monitors)   |
| `status`               | TEXT (`up` / `down`)   |
| `status_code`         | INTEGER, nullable       |
| `response_time_ms`   | INTEGER, nullable       |
| `error`                 | TEXT, nullable          |
| `checked_at`           | TIMESTAMPTZ             |

## Development

```bash
go build ./...   # compile everything
go vet ./...     # static analysis
gofmt -l .        # list files needing formatting
```

## Roadmap

- [x] Core data model, CRUD API, and concurrent polling worker
- [ ] Alert notifications on status change (email / webhook)
- [ ] Dockerfiles for `api` and `worker`
- [ ] GitHub Actions CI (build, vet, test)
- [ ] Reverse proxy + TLS (Caddy or Nginx)
- [ ] Production deployment on a VPS
- [ ] Public status page frontend

## License

MIT
