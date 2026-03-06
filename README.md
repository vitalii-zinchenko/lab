# lab

A local-only environment for running experiments — testing ideas, exploring tools, and benchmarking behaviour under different conditions.

> **Local use only.** This setup has no security hardening and must not be exposed to the internet.

## Running locally

```
make infra-up    # start Postgres, pgBouncer, Prometheus, Grafana, pgAdmin
make server-up   # start the API server
```

Services available after `make infra-up`:

| Service    | URL / address                         |
|------------|---------------------------------------|
| API        | http://localhost:8080                 |
| Prometheus | http://localhost:9090                 |
| Grafana    | http://localhost:3000                 |
| pgAdmin    | http://localhost:5050  (admin@admin.com / admin) |
| Postgres   | localhost:5432                        |
| PgBouncer  | localhost:6432                        |

## Performance tests

```
make perf-smoke    # quick sanity check
make perf-load     # sustained load test
make perf-stress   # ramp to breaking point
```
