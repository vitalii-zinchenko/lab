#!/usr/bin/env bash
set -euo pipefail

PRIMARY_HOST="${PRIMARY_HOST:-postgres}"
PRIMARY_PORT="${PRIMARY_PORT:-5432}"
REPL_USER="${REPL_USER:-replicator}"
REPL_PASSWORD="${REPL_PASSWORD:-replicator_password}"
PGDATA="${PGDATA:-/var/lib/postgresql/data}"

# ---------------------------------------------------------------------------
# If data directory is fully initialized (has PG_VERSION), skip pg_basebackup.
# If partially initialized (no PG_VERSION but not empty — a failed prior run),
# clean it so pg_basebackup can start fresh.
# ---------------------------------------------------------------------------
if [ -f "$PGDATA/PG_VERSION" ]; then
  echo "Data directory already initialized — starting standby..."
  exec gosu postgres postgres \
    -c hot_standby=on \
    -c max_connections=150
elif [ -n "$(ls -A "$PGDATA" 2>/dev/null)" ]; then
  echo "Partial data directory found (no PG_VERSION). Cleaning for fresh basebackup..."
  rm -rf "${PGDATA:?}"/*
fi

# ---------------------------------------------------------------------------
# Wait for the primary to be ready to accept connections (up to 60s).
# ---------------------------------------------------------------------------
echo "Waiting for primary at $PRIMARY_HOST:$PRIMARY_PORT..."
for i in $(seq 1 60); do
  if PGPASSWORD="$REPL_PASSWORD" pg_isready -h "$PRIMARY_HOST" -p "$PRIMARY_PORT" -U "$REPL_USER" -q; then
    echo "Primary is ready."
    break
  fi
  if [ "$i" -eq 60 ]; then
    echo "ERROR: Primary did not become ready within 60 seconds." >&2
    exit 1
  fi
  sleep 1
done

# ---------------------------------------------------------------------------
# Clone the primary using pg_basebackup.
#   -Xs  stream WAL during backup (keeps backup consistent without extra slot)
#   -R   write standby.signal + primary_conninfo into postgresql.auto.conf
#   -P   show progress
# ---------------------------------------------------------------------------
echo "Running pg_basebackup from $PRIMARY_HOST..."
PGPASSWORD="$REPL_PASSWORD" pg_basebackup \
  -h "$PRIMARY_HOST" \
  -p "$PRIMARY_PORT" \
  -U "$REPL_USER" \
  -D "$PGDATA" \
  -P \
  -Xs \
  -R

# ---------------------------------------------------------------------------
# pg_basebackup -R writes primary_conninfo but omits the password for security.
# Append full primary_conninfo with credentials for the lab environment.
# Also set hot_standby to allow read queries while replaying WAL.
# ---------------------------------------------------------------------------
cat >> "$PGDATA/postgresql.auto.conf" <<-EOF

# Standby settings written by entrypoint.sh
hot_standby = on
primary_conninfo = 'host=$PRIMARY_HOST port=$PRIMARY_PORT user=$REPL_USER password=$REPL_PASSWORD application_name=standby1'
EOF

# Fix ownership in case pg_basebackup ran as root
chown -R postgres:postgres "$PGDATA"
chmod 700 "$PGDATA"

echo "Starting standby postgres..."
exec gosu postgres postgres \
  -c hot_standby=on \
  -c max_connections=150
