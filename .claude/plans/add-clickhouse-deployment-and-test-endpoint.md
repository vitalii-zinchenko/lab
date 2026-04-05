# Add ClickHouse Deployment and Test Endpoint

## Context

This is a lab project for experimenting with infrastructure and Go services. We already have PostgreSQL with an `event_history` table. The goal is to add ClickHouse as an analytical data store, wire it into the local dev environment (Tilt + Docker Compose), and expose a test endpoint that inserts/returns events — mirroring the existing Postgres event flow.

Goose v3 natively supports the `clickhouse` dialect, so we can use the same migration approach as Postgres but with a separate runner and migration directory.

---

## Plan

### Step 1: Add ClickHouse service to Docker Compose

**File:** `infra/docker-compose.yml`

- Add `clickhouse` service using `clickhouse/clickhouse-server:24`
- Expose ports `8123` (HTTP) and `9000` (native TCP)
- Volume: `clickhouse_data:/var/lib/clickhouse`
- Healthcheck via `clickhouse-client --query 'SELECT 1'`
- Join `lab-network`
- Add `clickhouse_data` to volumes section

### Step 2: Create ClickHouse migration infrastructure

**New file:** `infra/ch_migrations/00001_create_ch_events_table.sql`

- Create `ch_events` table mirroring `event_history` schema
- Fields: `id UUID`, `level LowCardinality(String)`, `event_type String`, `details Nullable(String)`, `created_at DateTime`
- Engine: `MergeTree() ORDER BY (created_at, id)` (append-only event log)

**New file:** `infra/ch_migrate/main.go`

- Mirror the pattern from `infra/migrate/main.go`
- Read `CLICKHOUSE_URL` and `MIGRATIONS_DIR` env vars
- Use `goose.SetDialect("clickhouse")` + `clickhouse-go/v2` driver
- Run `goose.Up()`

**New file:** `infra/ch_migrate/go.mod` + run `go mod tidy`

### Step 3: Wire ClickHouse into Tiltfile

**File:** `Tiltfile`

- Add `dc_resource('clickhouse', labels=['infrastructure'])`
- Add `ch-migrate` local_resource (depends on `clickhouse`, watches `infra/ch_migrations/`)
- Update `server` resource: add `CLICKHOUSE_URL` env var, add `ch-migrate` to deps

### Step 4: Add `/ch-events` endpoint to OpenAPI spec

**File:** `services/server/api/openapi.yaml`

- Add `POST /ch-events` path
- Reuse existing `NewEvent` request schema and `EventHistory` response schema (same fields)

### Step 5: Regenerate API code

- Run `go generate ./...` from `services/server/` to update `api/api.gen.go`

### Step 6: Add ClickHouse event repository

**New file:** `services/server/repository/ch_event.go`

- Interface: `ChEventRepository` with `Create(ctx, event) (event, error)`
- Implementation using `database/sql` directly (not GORM — ClickHouse doesn't need it)
- Reuse existing `model.EventHistory` struct since fields match
- Use `?` placeholder syntax for clickhouse-go/v2

### Step 7: Add ClickHouse event handler

**New file:** `services/server/api/ch_events.go`

- `ChEventsHandler` struct with `chEventRepo` field
- `CreateChEvent()` method following the pattern from `events.go`
- Reuse `toGenEvent()` helper from `events.go`

### Step 8: Wire into composite handler

**File:** `services/server/api/handler.go`

- Add `*ChEventsHandler` to `Handler` struct
- Update `New()` to accept `repository.ChEventRepository` parameter

### Step 9: Wire ClickHouse connection in server main

**File:** `services/server/cmd/server/main.go`

- Add `database/sql` + `clickhouse-go/v2` imports
- Open ClickHouse connection via `CLICKHOUSE_URL` env var
- Create `ChEventRepository`, pass to `api.New()`

### Step 10: Update Go dependencies

- `go get github.com/ClickHouse/clickhouse-go/v2` in `services/server/`

---

## Key files to modify

| File | Action |
|------|--------|
| `infra/docker-compose.yml` | Add clickhouse service + volume |
| `infra/ch_migrations/00001_create_ch_events_table.sql` | Create (new) |
| `infra/ch_migrate/main.go` | Create (new) |
| `infra/ch_migrate/go.mod` | Create (new) |
| `Tiltfile` | Add clickhouse resource, ch-migrate, update server |
| `services/server/api/openapi.yaml` | Add POST /ch-events |
| `services/server/api/api.gen.go` | Regenerate |
| `services/server/repository/ch_event.go` | Create (new) |
| `services/server/api/ch_events.go` | Create (new) |
| `services/server/api/handler.go` | Add ChEventsHandler |
| `services/server/cmd/server/main.go` | Add ClickHouse connection |
| `services/server/go.mod` | Add clickhouse-go dependency |

## Reusable existing code

- `toGenEvent()` in `services/server/api/events.go:17` — converts model to API type
- `model.EventHistory` in `services/server/model/event_history.go` — reuse as-is
- Migration runner pattern from `infra/migrate/main.go`

## Verification

1. Run `tilt up` — verify ClickHouse container starts and is healthy
2. Check `ch-migrate` resource — should apply migration successfully
3. Check `server` resource — should start without errors
4. Test the endpoint:
   ```bash
   curl -X POST http://localhost:8080/ch-events \
     -H 'Content-Type: application/json' \
     -d '{"level":"info","event_type":"test","details":"hello clickhouse"}'
   ```
   Should return 201 with the inserted event JSON
5. Verify data in ClickHouse:
   ```bash
   docker exec clickhouse clickhouse-client --query "SELECT * FROM ch_events"
   ```
