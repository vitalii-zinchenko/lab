# PostgreSQL Streaming Replication Setup

## Context

The current infra has a single PostgreSQL 17 primary + PgBouncer for connection pooling. We want to add a read-only hot standby replica using PostgreSQL's built-in streaming replication.

**Replication level chosen: `remote_write`**

PostgreSQL synchronous commit levels (from weakest to strongest):
| Level | Guarantee | Latency |
|---|---|---|
| `off` | Async, fire-and-forget — data loss possible | Lowest |
| `local` | WAL flushed locally only | Low |
| **`remote_write`** ← **our choice** | WAL received into standby OS buffer (not fsynced) | Medium |
| `on` | WAL flushed to disk on standby | Higher |
| `remote_apply` | WAL applied (data visible) on standby | Highest |

`remote_write` means: primary commits when the standby has received WAL into its kernel buffer. Possible data loss only on a simultaneous primary + standby OS crash. Good middle ground.

**Tradeoff**: With `synchronous_standby_names = '*'`, if the standby disconnects, the primary stalls on commits. Recovery: `ALTER SYSTEM SET synchronous_standby_names = ''; SELECT pg_reload_conf();`

## Files to Create

### `infra/postgres/primary/01-replication.sh`
Init script mounted in `/docker-entrypoint-initdb.d/` — runs on first `initdb` only.
- Creates `replicator` role with `REPLICATION LOGIN` privilege
- Appends `pg_hba.conf` rule to allow replication connections from Docker network

### `infra/postgres/standby/entrypoint.sh`
Custom entrypoint for the standby container — replaces the default postgres entrypoint.
- If data dir already initialized (container restart): skip to starting postgres
- Otherwise: poll until primary is ready, then run `pg_basebackup -Xs -R` to clone
- Appends `primary_conninfo` with password to `postgresql.auto.conf`
- Uses `gosu postgres postgres` to drop from root before starting

## Files to Modify

### `infra/docker-compose.yml`
**Primary (`postgres`) — add GUC flags:**
```
-c wal_level=replica
-c max_wal_senders=5
-c wal_keep_size=256
-c synchronous_commit=remote_write
-c synchronous_standby_names=*
```
Also mount `./postgres/primary/01-replication.sh:/docker-entrypoint-initdb.d/01-replication.sh:ro`

**Add `postgres-standby` service:**
- Same `postgres:17` image, custom entrypoint `/entrypoint.sh`
- Env vars: `PRIMARY_HOST=postgres`, `REPL_USER=replicator`, `REPL_PASSWORD`
- Separate `pgdata_standby` volume, port `5433:5432`
- `depends_on: postgres: condition: service_healthy`
- `start_period: 60s` in healthcheck (basebackup takes time)

**Add `pgbouncer-standby` service:**
- Same `edoburu/pgbouncer` image, points to `postgres-standby:5432`
- Port `6433:5432`
- `depends_on: postgres-standby: condition: service_healthy`

**Add `postgres_exporter_standby` service:**
- Same exporter image, connects to `postgres-standby:5432`
- Port `9188:9187`

**Add `pgdata_standby` to `volumes` section**

### `infra/pgadmin/servers.json`
Add server `"2"` pointing to `postgres-standby:5432` with name `"lab-standby"`.
> Note: PgAdmin only reads this file on first startup (empty `pgadmin_data` volume).

### `infra/prometheus/prometheus.yml`
Add scrape job `postgres-standby` targeting `postgres_exporter_standby:9187`.

## Port Reference

| Service | Host Port | Internal |
|---|---|---|
| postgres (primary) | 5432 | 5432 |
| postgres-standby | 5433 | 5432 |
| pgbouncer (primary) | 6432 | 5432 |
| pgbouncer-standby | 6433 | 5432 |
| postgres_exporter (primary) | 9187 | 9187 |
| postgres_exporter_standby | 9188 | 9187 |

## Important: Existing Primary Data

If `pgdata` volume already has data, the `01-replication.sh` init script **will not run** (initdb scripts only run on empty volumes). In that case, run manually before bringing up the stack:

```bash
# Create replication user
docker compose exec postgres psql -U postgres -c \
  "CREATE ROLE replicator WITH REPLICATION LOGIN PASSWORD 'replicator_password';"

# Add pg_hba.conf rule
docker compose exec postgres bash -c \
  "echo 'host replication replicator 0.0.0.0/0 scram-sha-256' >> \$PGDATA/pg_hba.conf"

# Reload config and restart to pick up new -c GUC flags
docker compose exec postgres psql -U postgres -c "SELECT pg_reload_conf();"
docker compose up -d --force-recreate postgres
```

## Verification

After `docker compose up -d`:

```bash
# 1. Confirm standby is streaming with remote_write sync state
docker compose exec postgres psql -U postgres -c \
  "SELECT client_addr, state, sync_state FROM pg_stat_replication;"
# Expected: sync_state = remote_write

# 2. Confirm standby is in recovery (read-only)
docker compose exec postgres-standby psql -U postgres -c \
  "SELECT pg_is_in_recovery();"
# Expected: true

# 3. Check replication lag
docker compose exec postgres-standby psql -U postgres -c \
  "SELECT now() - pg_last_xact_replay_timestamp() AS lag;"

# 4. Confirm standby rejects writes
docker compose exec postgres-standby psql -U postgres -d app -c \
  "CREATE TABLE test_rw (id int);"
# Expected: ERROR: cannot execute CREATE TABLE in a read-only transaction

# 5. Test read path via pgbouncer-standby (from host)
psql postgres://postgres:postgres@localhost:6433/app -c "SELECT 1;"
```
