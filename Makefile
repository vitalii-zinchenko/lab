.PHONY: infra-up infra-down db-migrate-up server-up server-down perf-install perf-smoke perf-load perf-stress

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
	docker compose -f server/golang/docker-compose.yml up --build
	@echo ""
	@echo "Services running:"
	@echo "  API: http://localhost:8080"

server-down:
	docker compose -f server/golang/docker-compose.yml down

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
