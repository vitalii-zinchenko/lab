#!/usr/bin/env bash
set -euo pipefail

# ---------------------------------------------------------------------------
# Creates the replication user and pg_hba.conf rule for streaming replication.
# This script runs once on first initdb (when pgdata volume is empty).
# ---------------------------------------------------------------------------

# 1. Create the replication user
# SET synchronous_commit = local to avoid hanging if synchronous_standby_names=*
# is already active but no standby is connected yet (chicken-and-egg on fresh start).
psql -v ON_ERROR_STOP=1 --username "$POSTGRES_USER" --dbname "$POSTGRES_DB" <<-EOSQL
  SET synchronous_commit = local;
  DO \$\$
  BEGIN
    IF NOT EXISTS (SELECT FROM pg_catalog.pg_roles WHERE rolname = 'replicator') THEN
      CREATE ROLE replicator WITH REPLICATION LOGIN PASSWORD '${REPL_PASSWORD:-replicator_password}';
    END IF;
  END
  \$\$;
EOSQL

# 2. Allow the replication user to connect from anywhere in the Docker network
cat >> "$PGDATA/pg_hba.conf" <<-EOF

# Streaming replication — allow 'replicator' from Docker network
host    replication     replicator      0.0.0.0/0               scram-sha-256
EOF

# Set synchronous replication — takes effect after real server starts (ALTER SYSTEM
# writes to postgresql.auto.conf, which is read on the next startup).
# We do NOT pass these as -c flags because they would be active during the init
# phase, causing Docker's own CREATE DATABASE to hang with no standby connected.
psql -v ON_ERROR_STOP=1 --username "$POSTGRES_USER" --dbname "$POSTGRES_DB" <<-EOSQL
  ALTER SYSTEM SET synchronous_commit = 'remote_write';
  ALTER SYSTEM SET synchronous_standby_names = '*';
EOSQL

echo "Replication user, pg_hba rule, and synchronous replication settings applied."
