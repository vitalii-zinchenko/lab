#!/usr/bin/env bash
# Render Jsonnet Grafana dashboards to JSON via Docker.
# On first run, jb downloads grafonnet (~30s); subsequent runs use the cached image.
set -euo pipefail

REPO_ROOT="$(cd "$(dirname "$0")/.." && pwd)"

echo "Building jsonnet-renderer image..."
docker build -t jsonnet-renderer "$REPO_ROOT/infra/grafana/dashboards/"

echo "Rendering Grafana dashboards..."
mkdir -p "$REPO_ROOT/infra/grafana/provisioning/dashboards"

docker run --rm \
  -v "$REPO_ROOT/infra/grafana/dashboards":/src \
  -v "$REPO_ROOT/infra/grafana/provisioning/dashboards":/out \
  -w /src \
  jsonnet-renderer \
  sh -c "jb install && \
    jsonnet -J vendor go-runtime.jsonnet  > /out/go-runtime.json && \
    jsonnet -J vendor api-red.jsonnet     > /out/api-red.json && \
    jsonnet -J vendor postgres.jsonnet    > /out/postgres.json && \
    jsonnet -J vendor pgbouncer.jsonnet   > /out/pgbouncer.json"

echo "Dashboards written to infra/grafana/provisioning/dashboards/"
