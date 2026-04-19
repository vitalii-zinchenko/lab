# Backend Service Template

A reference architecture for backend services in this monorepo. Use this as a guide when scaffolding a new service.

---

## Folder Structure

```
services/<service-name>/
├── cmd/
│   ├── server/          # Main entrypoint — starts the HTTP/gRPC/etc. server
│   └── <toolname>/      # Optional auxiliary executables (e.g. code generators, spec bundlers)
├── internal/
│   ├── shared/          # Cross-domain utilities (errors, DB helpers, generated interfaces)
│   │   └── server/      # Router/server wiring (RouterParams, NewRouter, fx Module)
│   ├── <domain-a>/      # One folder per business domain
│   │   ├── handlers/    # Transport layer — handles requests and maps to domain logic
│   │   ├── models/      # Domain entities (mapped to DB schema)
│   │   ├── repos/       # Data access — interface + implementation
│   │   └── module.go    # fx.Module exposing all domain constructors
│   ├── <domain-b>/
│   │   └── ...
│   └── testinfra/       # Shared test utilities (fixtures, containers, helpers)
├── go.mod / go.sum
└── Dockerfile
```

---

## Architecture

The service follows a **layered, domain-driven architecture** with a strict separation of concerns:

```
HTTP / gRPC / etc.
       │
       ▼
  Middleware             (auth, validation, observability, rate limiting)
       │
       ▼
   Handlers              (transport layer — one package per domain)
       │
       ▼
  Repositories           (data access — interface-based, one package per domain)
       │
       ▼
  Domain Models          (entities, value objects — mapped to storage schema)
       │
       ▼
  Storage Backend        (PostgreSQL, ClickHouse, Redis, etc.)
```

The router/server wiring (`internal/shared/server/`) is the single place where all layers are composed together via constructor injection.

---

## Layer Responsibilities

### `cmd/server/`
- Reads environment variables (fail fast if required vars are missing)
- Initialises infrastructure connections (DB, cache, message broker, etc.)
- Sets connection pool parameters
- Starts the primary server and any secondary servers (e.g. metrics endpoint)

### `internal/shared/server/` — Dependency Injection Hub
- Creates all repositories (passing DB connections)
- Creates all handlers (passing repositories and secrets)
- Registers all handlers on the router
- Applies global middleware (auth, request validation, observability)
- Returns a fully wired router

### `internal/<domain>/handlers/`
- Receives requests and maps them to repository calls
- Translates domain/repository errors to appropriate transport-layer responses (e.g. 404, 409)
- Does **not** contain business logic — delegates to repositories or services
- Implements the interface generated from the service contract (OpenAPI, protobuf, etc.)

### `internal/<domain>/repos/`
- Defines an **interface** for each repository (enables testing and swappability)
- Provides a concrete implementation backed by the actual storage driver
- Returns domain-level errors (e.g. `shared.ErrNotFound`) rather than raw driver errors
- Contains query logic — cursor-based pagination, filtering, aggregation

### `internal/<domain>/models/`
- Plain structs representing domain entities
- Tagged for the storage layer (e.g. GORM tags, BSON tags)
- No business logic

### `internal/shared/`
- Sentinel errors shared across domains (e.g. `ErrNotFound`)
- DB/driver error helpers (e.g. detecting unique constraint violations)
- Generated code lives here (server interfaces, types) — **do not edit by hand**

### `internal/testinfra/`
- Shared test helpers: spinning up real DB containers, seeding fixtures
- Integration tests use real infrastructure (no mocks for the DB layer)

---

## Domain Structure

Each business domain is self-contained:

```
internal/<domain>/
├── handlers/
│   └── <domain>_handler.go   # Handler struct + all endpoint methods
├── models/
│   └── <entity>.go           # GORM / plain struct entity
└── repos/
    ├── <domain>_repo.go      # Interface + constructor
    └── <domain>_repo_impl.go # Concrete implementation
```

Domains do **not** import each other. Cross-domain data needs go through the handler layer or a shared package.

---

## Service Contract

The service exposes its API through a **contract-first** approach:

- Define the contract first (OpenAPI spec, protobuf file, etc.) before writing any handler code
- Generate server interfaces and request/response types from the contract
- Handlers implement the generated interface — the compiler enforces completeness
- Split large contracts into per-domain files and bundle them into one at build time

This approach is transport-agnostic: the same pattern applies whether the transport is REST (OpenAPI), RPC (gRPC/protobuf), GraphQL, or anything else.

---

## Dependency Injection

DI is handled by **[`go.uber.org/fx`](https://github.com/uber-go/fx)**. Each domain declares its own `fx.Module`; `main.go` composes them and starts lifecycle-managed servers.

### Domain modules

Every domain exposes a `module.go` at its package root:

```go
// internal/<domain>/module.go
package <domain>

var Module = fx.Module("<domain>",
    fx.Provide(repos.NewXxxRepository),
    fx.Provide(handlers.NewXxxHandler),
)
```

All constructors follow the standard Go pattern — they accept their dependencies as arguments and return the concrete type (or an interface). fx resolves the dependency graph automatically.

### Router wiring

`internal/shared/server/router.go` receives all dependencies through an `fx.In` params struct. This keeps the constructor signature stable as new domains are added:

```go
type RouterParams struct {
    fx.In

    DB        *gorm.DB
    ChDB      *sql.DB
    JWTSecret []byte

    // Named tag prevents ambiguity when multiple gin.HandlerFunc values exist.
    PrometheusMiddleware gin.HandlerFunc `name:"prometheus"`

    FooHandler *foohandlers.FooHandler
    BarHandler *barhandlers.BarHandler
    // ... one field per domain handler
}

func NewRouter(p RouterParams) *gin.Engine { ... }
```

The `appHandler` struct in `router.go` embeds all domain handler structs. Because Go embedding promotes methods, the single `appHandler` automatically satisfies the full server interface composed of all domains.

### Application entrypoint

`cmd/server/main.go` wires everything together:

```go
func main() {
    fx.New(
        // Infrastructure providers (DB, secrets, etc.)
        fx.Provide(newPostgresDB),
        fx.Provide(newJWTSecret),
        fx.Provide(
            fx.Annotate(newPrometheusMiddleware, fx.ResultTags(`name:"prometheus"`)),
        ),

        // Domain modules
        foodomain.Module,
        bardomain.Module,

        // HTTP server module
        internalserver.Module,

        // Start servers via lifecycle hooks (graceful shutdown included)
        fx.Invoke(startMetricsServer),
        fx.Invoke(startHTTPServer),
    ).Run()
}
```

Infrastructure that needs cleanup (DB connections, etc.) registers `OnStop` hooks via `fx.Lifecycle`.

### Tests

Tests bypass fx and construct `RouterParams` directly, passing a no-op middleware for anything irrelevant to the test:

```go
router := internalserver.NewRouter(internalserver.RouterParams{
    DB:                   db,
    JWTSecret:            testSecret,
    PrometheusMiddleware: func(c *gin.Context) { c.Next() },
    FooHandler:           foohandlers.NewFooHandler(fooRepo),
    // ...
})
```

---

## Middleware

Applied globally in the router wiring layer, not inside individual handlers:

| Concern | Where |
|---|---|
| Authentication / identity extraction | Router middleware |
| Request validation (schema, auth scopes) | Router middleware |
| Observability (metrics, tracing) | Router middleware |
| Usage / audit tracking | Strict middleware wrapping all handlers |
| Rate limiting | Router middleware |

Domain-specific middleware (if any) is scoped to that domain's handler group.

---

## Configuration

- All configuration comes from **environment variables**
- Required vars are read at startup; the process exits immediately if any are missing — no silent defaults for secrets or infrastructure URLs
- Optional vars may have sensible defaults (e.g. port, log level)

---

## Observability

- Expose a **metrics endpoint** on a separate port (not the primary API port) so it is never exposed through the API gateway
- Use route templates (not raw paths) as metric labels to avoid cardinality explosion from path parameters
- Health check endpoint (`GET /health`) returns a minimal response with no auth requirement

---

## Testing

- Unit tests: cover handler and repository logic in isolation using interface mocks
- Integration tests: use real infrastructure (spin up containers) — do **not** mock the DB layer
- Test files live alongside the code they test (`*_test.go` in the same package or a `_test` package sibling)
- Shared test utilities live in `internal/testinfra/`

**Red/Green discipline**: write a failing test first, then write the implementation to make it pass.

---

## Code Generation

Services may use code generation for repetitive or contract-derived code. All generated files:

- Are committed to the repository
- Have a header comment marking them as generated (`// Code generated ... DO NOT EDIT`)
- Are regenerated via `go generate ./...` or a documented `make` target
- Live in `internal/shared/` (server interfaces) or alongside their source models (query DSL)

---

## Adding a New Domain

1. Create `internal/<domain>/models/`, `repos/`, `handlers/` packages
2. Define the repository interface in `repos/`
3. Define the contract (spec paths + schemas, or protobuf messages + services)
4. Add domain paths/schemas to the master spec and regenerate server interfaces
5. Implement handlers to satisfy the generated interface
6. Create `internal/<domain>/module.go` exposing an `fx.Module` with all repo and handler constructors
7. Add the new module to `fx.New(...)` in `cmd/server/main.go`
8. Add the new handler fields to `RouterParams` and `appHandler` in `internal/shared/server/router.go`
9. Write integration tests in `internal/<domain>/` using `testinfra` helpers
