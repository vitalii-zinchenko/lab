# -*- mode: Python -*-
# Tiltfile — local development orchestrator
# Groups: infrastructure, services, setup

# Allow running without a k8s cluster — we only use docker-compose + local processes
allow_k8s_contexts(k8s_context())

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
local_resource(
    'go-server',
    cmd='./scripts/kill-port.sh 8080',
    serve_cmd=str(local('go env GOPATH', quiet=True)).strip() + '/bin/air',
    serve_dir='server/golang',
    serve_env={
        'DATABASE_URL': 'postgres://postgres:postgres@localhost:6432/app?sslmode=disable',
    },
    deps=['server/golang/cmd', 'server/golang/api', 'server/golang/repository', 'server/golang/model'],
    resource_deps=['db-migrate', 'pgbouncer'],
    labels=['services'],
    readiness_probe=probe(
        period_secs=5,
        http_get=http_get_action(port=8080, path='/health'),
    ),
)
