.PHONY: infra-up infra-down db-migrate-up server-up server-down perf-install perf-smoke perf-load perf-stress perf-insert-usage perf-get-usage perf-get-usage-stats dashboards dashboards-renderer deploy-dashboards tilt-setup app test mock-data mock-data-fill mock-data-reindex mock-create-api-keys generate

POSTGRES_USER             ?= postgres
POSTGRES_PASSWORD         ?= postgres
POSTGRES_DB               ?= app
PGBOUNCER_MAX_CLIENT_CONN ?= 200
PGBOUNCER_DEFAULT_POOL_SIZE ?= 20
PGBOUNCER_POOL_MODE       ?= transaction

export POSTGRES_USER POSTGRES_PASSWORD POSTGRES_DB
export PGBOUNCER_MAX_CLIENT_CONN PGBOUNCER_DEFAULT_POOL_SIZE PGBOUNCER_POOL_MODE

DATABASE_URL_DIRECT = postgres://$(POSTGRES_USER):$(POSTGRES_PASSWORD)@localhost:5432/$(POSTGRES_DB)?sslmode=disable

infra-up:
	docker compose -f infra/docker-compose.yml up -d
	@echo ""
	@echo "Waiting for Postgres to be ready..."
	@until docker compose -f infra/docker-compose.yml exec -T postgres pg_isready -U $(POSTGRES_USER) > /dev/null 2>&1; do \
		sleep 1; \
	done
	@echo "Postgres is ready!"
	@$(MAKE) db-migrate-up
	@echo ""
	@echo "Services running:"
	@echo "  Prometheus: http://localhost:9090"
	@echo "  Grafana:    http://localhost:3000"
	@echo "  pgAdmin:    http://localhost:5050  (admin@admin.com / admin)"
	@echo "  Postgres:   localhost:5432"
	@echo "  PgBouncer:  localhost:6432"

infra-down:
	docker compose -f infra/docker-compose.yml down

db-migrate-up:
	@echo "Running database migrations..."
	@cd infra/migrate && \
		DATABASE_URL="$(DATABASE_URL_DIRECT)" \
		MIGRATIONS_DIR="$(CURDIR)/infra/migrations" \
		go run .

server-up:
	docker compose -f services/server/docker-compose.yml up -d --build
	@echo ""
	@echo "Services running:"
	@echo "  API: http://localhost:8080"

server-down:
	docker compose -f services/server/docker-compose.yml down

# Grafana dashboards — render Jsonnet → JSON via Docker (no local tooling needed)
# On first run jb downloads grafonnet (~30 s); subsequent runs use the cached image.
dashboards-renderer:
	docker build -t jsonnet-renderer infra/grafana/dashboards/

dashboards: dashboards-renderer
	@echo "Rendering Grafana dashboards..."
	@mkdir -p infra/grafana/provisioning/dashboards
	@docker run --rm \
	  -v "$(CURDIR)/infra/grafana/dashboards":/src \
	  -v "$(CURDIR)/infra/grafana/provisioning/dashboards":/out \
	  -w /src \
	  jsonnet-renderer \
	  sh -c "jb install && \
	    jsonnet -J vendor go-runtime.jsonnet  > /out/go-runtime.json && \
	    jsonnet -J vendor api-red.jsonnet     > /out/api-red.json && \
	    jsonnet -J vendor postgres.jsonnet    > /out/postgres.json && \
	    jsonnet -J vendor pgbouncer.jsonnet   > /out/pgbouncer.json"
	@echo "Dashboards written to infra/grafana/provisioning/dashboards/"

# Tilt — local dev orchestrator
tilt-setup:
	@command -v tilt >/dev/null 2>&1 || { echo "Installing Tilt..."; brew install tilt-dev/tap/tilt; }
	@command -v air >/dev/null 2>&1 || { echo "Installing Air..."; go install github.com/air-verse/air@latest; }
	@echo "Tilt and Air are ready."

up:
	tilt up

deploy-dashboards:
	./scripts/deploy-dashboards.sh

# Performance tests (requires k6: brew install k6)
BASE_URL ?= http://localhost:8080

perf-install:
	cd perf/k6 && npm install

perf-smoke:
	cd perf/k6 && npx webpack && k6 run dist/smoke.js -e BASE_URL=$(BASE_URL)

perf-load:
	cd perf/k6 && npx webpack && k6 run dist/load.js -e BASE_URL=$(BASE_URL)

perf-stress:
	cd perf/k6 && npx webpack && k6 run dist/stress.js -e BASE_URL=$(BASE_URL)

perf-insert-usage:
	cd perf/k6 && npx webpack && k6 run dist/insert-usage.js -e BASE_URL=$(BASE_URL)

perf-get-usage:
	cd perf/k6 && npx webpack && k6 run dist/get-usage.js -e BASE_URL=$(BASE_URL)

perf-get-usage-stats:
	cd perf/k6 && npx webpack && k6 run dist/get-usage-stats.js -e BASE_URL=$(BASE_URL)

generate:
	cd services/server && go generate ./...

test:
	cd services/server && go test -v -timeout 120s ./internal/...

# Mock data — pass any subcommand and flags via CMD:
#   make mock-data CMD="fill --workers 4"
#   make mock-data CMD="fill --target 500000000 --workers 8"
#   make mock-data CMD="reindex --work-mem 2GB"
# Shortcut targets with sensible defaults:
#   make mock-data-fill   [TARGET=1000000000] [MOCK_USERS=1000] [MOCK_BATCH=100000]
#   make mock-data-reindex
TARGET ?= 1000000000
MOCK_USERS ?= 1000
MOCK_BATCH ?= 100000
CMD ?= fill --target $(TARGET) --users $(MOCK_USERS) --batch $(MOCK_BATCH)

mock-data:
	cd perf/mock-data && \
		DATABASE_URL="$(DATABASE_URL_DIRECT)" \
		go run . $(CMD)

mock-data-fill:
	cd perf/mock-data && \
		DATABASE_URL="$(DATABASE_URL_DIRECT)" \
		go run . fill --target $(TARGET) --users $(MOCK_USERS) --batch $(MOCK_BATCH)

mock-data-reindex:
	cd perf/mock-data && \
		DATABASE_URL="$(DATABASE_URL_DIRECT)" \
		go run . reindex

# Creates one API key per mock user and writes perf/mock-data/users-credentials.json.
# Idempotent: re-run safely to refresh the file without recreating existing keys.
mock-create-api-keys:
	cd perf/mock-data && \
		DATABASE_URL="$(DATABASE_URL_DIRECT)" \
		go run . create-api-keys --out users-credentials.json
