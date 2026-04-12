# -*- mode: Python -*-
# Tiltfile — local development orchestrator
# Groups: infrastructure, services, setup

load('ext://dotenv', 'dotenv')
dotenv('services/server/.env')

# Allow running without a k8s cluster — we only use docker-compose + local processes
allow_k8s_contexts(k8s_context())

# ============================================================
# Images
# ============================================================
docker_build('infra_server', 'services/server')

# ============================================================
# Infrastructure (Docker Compose)
# ============================================================
docker_compose('./infra/docker-compose.yml')

dc_resource('prometheus',          labels=['infrastructure'])
dc_resource('grafana',             labels=['infrastructure'], resource_deps=['prometheus'])
dc_resource('postgres',            labels=['infrastructure'])
dc_resource('pgbouncer',           labels=['infrastructure'], resource_deps=['postgres'])
dc_resource('postgres_exporter',   labels=['infrastructure'], resource_deps=['postgres'])
dc_resource('pgbouncer_exporter',  labels=['infrastructure'], resource_deps=['pgbouncer'])
dc_resource('pgadmin',             labels=['infrastructure'], resource_deps=['postgres'])
dc_resource('clickhouse',          labels=['infrastructure'])

# ============================================================
# Setup
# ============================================================
local_resource(
    'db-migrate',
    cmd='cd infra/migrate && DATABASE_URL="postgres://postgres:postgres@localhost:5432/app?sslmode=disable" MIGRATIONS_DIR="' + os.getcwd() + '/infra/migrations" go run .',
    deps=['infra/migrations'],
    resource_deps=['postgres'],
    labels=['setup'],
)

local_resource(
    'ch-migrate',
    cmd='cd infra/ch_migrate && CLICKHOUSE_URL="clickhouse://localhost:9000/default" MIGRATIONS_DIR="' + os.getcwd() + '/infra/ch_migrations" go run .',
    deps=['infra/ch_migrations'],
    resource_deps=['clickhouse'],
    labels=['setup'],
)

local_resource(
    'deploy-dashboards',
    cmd='./scripts/deploy-dashboards.sh',
    deps=[
        'infra/grafana/dashboards/go-runtime.jsonnet',
        'infra/grafana/dashboards/api-red.jsonnet',
        'infra/grafana/dashboards/postgres.jsonnet',
        'infra/grafana/dashboards/pgbouncer.jsonnet',
        'infra/grafana/dashboards/Dockerfile',
    ],
    resource_deps=['grafana'],
    labels=['setup'],
)

# ============================================================
# Services
# ============================================================
dc_resource(
    'server',
    labels=['services'],
    resource_deps=['db-migrate', 'ch-migrate', 'pgbouncer'],
)
