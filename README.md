# is-it-down

Self-hosted uptime and status monitoring service. Polls a set of HTTP endpoints on independent schedules, records response time and status history, and exposes a status feed you can build a public status page on top of.

[![Go Version](https://img.shields.io/badge/go-1.27-00ADD8?logo=go&logoColor=white)](https://go.dev)
[![PostgreSQL](https://img.shields.io/badge/postgres-16-336791?logo=postgresql&logoColor=white)](https://www.postgresql.org)
[![GORM](https://img.shields.io/badge/orm-GORM-00ADD8)](https://gorm.io)
[![Swagger](https://img.shields.io/badge/docs-swagger-85EA2D?logo=swagger&logoColor=white)](docs/swagger.yaml)
[![License: MIT](https://img.shields.io/badge/license-MIT-blue.svg)](LICENSE)

## Table of Contents

- [Overview](#overview)
- [Architecture](#architecture)
- [Tech Stack](#tech-stack)
- [Prerequisites](#prerequisites)
- [Getting Started](#getting-started)
- [Configuration](#configuration)
- [Project Structure](#project-structure)
- [API Documentation](#api-documentation)
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
| ORM            | [GORM](https://gorm.io) (`AutoMigrate`-managed schema, Postgres driver over `pgx`) |
| API docs       | [Swaggo](https://github.com/swaggo/swag) (annotations → OpenAPI 2.0 → Swagger UI) |
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

3. **Run the API**

   ```bash
   go run ./cmd/api
   ```

   Listens on `:8080` by default. On startup it runs `GORM AutoMigrate` against the `store.Monitor` and `store.Check` models, so the schema is created/updated automatically — no separate migration step needed. Verify with:

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
│   ├── db/                # GORM connection + AutoMigrate helper
│   ├── httpapi/          # HTTP routes, handlers, and Swaggo annotations
│   ├── scheduler/       # Per-monitor concurrent polling scheduler
│   └── store/             # GORM models and data access layer
├── docs/                  # Generated by `swag init` — do not edit by hand
├── docker-compose.yml   # Local Postgres service
└── .env.example          # Environment variable template
```

## API Documentation

Interactive Swagger UI is served by the API itself once it's running:

```
http://localhost:8080/swagger/index.html
```

It's generated from `@Summary`/`@Router`/etc. annotations on the handlers in [`internal/httpapi`](internal/httpapi/handlers.go) and the general API info block in [`cmd/api/main.go`](cmd/api/main.go). After changing any of those annotations, regenerate the spec:

```bash
go tool swag init -g cmd/api/main.go -o docs
```

This overwrites [`docs/docs.go`](docs/docs.go), `docs/swagger.json`, and `docs/swagger.yaml` — regenerated files are committed so the project builds without anyone needing `swag` installed separately (it's declared as a `go.mod` tool dependency, invoked above via `go tool`).

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

Defined as GORM models in [`internal/store/models.go`](internal/store/models.go); the tables below are what `AutoMigrate` produces from them. There is no separate migration file to keep in sync — the struct tags *are* the schema.

**monitors** — the endpoints being watched

| Column             | Type        |
|--------------------|-------------|
| `id`                 | UUID (PK), default `gen_random_uuid()` |
| `name`              | TEXT, not null |
| `url`                | TEXT, not null |
| `interval_seconds` | BIGINT, not null, default `60` |
| `timeout_seconds`  | BIGINT, not null, default `10` |
| `created_at`        | TIMESTAMPTZ |

**checks** — one row per poll of a monitor, `ON DELETE CASCADE` from `monitors`

| Column               | Type                    |
|----------------------|--------------------------|
| `id`                    | UUID (PK), default `gen_random_uuid()` |
| `monitor_id`          | UUID (FK → `monitors.id`) |
| `status`               | TEXT (`up` / `down`), not null |
| `status_code`         | BIGINT, nullable       |
| `response_time_ms`   | BIGINT, nullable       |
| `error`                 | TEXT, nullable          |
| `checked_at`           | TIMESTAMPTZ             |

`checks` has a composite index on `(monitor_id, checked_at DESC)` to keep per-monitor history lookups fast.

## Development

```bash
go build ./...   # compile everything
go vet ./...     # static analysis
gofmt -l .        # list files needing formatting
```

## Roadmap

- [x] Core data model, CRUD API, and concurrent polling worker
- [x] GORM data layer with `AutoMigrate`
- [x] Swagger/OpenAPI documentation via Swaggo
- [ ] Alert notifications on status change (email / webhook)
- [ ] Dockerfiles for `api` and `worker`
- [ ] GitHub Actions CI (build, vet, test)
- [ ] Reverse proxy + TLS (Caddy or Nginx)
- [ ] Production deployment on a VPS
- [ ] Public status page frontend

## License

MIT
