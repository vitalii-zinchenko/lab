# Integration Tests for Auth Handlers

## Context

The server has no tests at all. We need integration tests that spin up real Postgres and ClickHouse
containers (via testcontainers-go), run all migrations, start the real Gin server (via httptest),
and call it through a generated typed API client. Tests live alongside handlers in
`internal/auth/handlers/` and run as normal Go tests with `go test ./...` — no build tags needed.

---

## File Structure

```
services/server/
├── internal/
│   ├── apiclient/
│   │   └── client.gen.go                    # Generated typed HTTP client (oapi-codegen)
│   ├── server/
│   │   └── router.go                        # Extracted NewRouter() shared by main + tests
│   ├── testinfra/
│   │   └── fixture.go                       # Fixture: container + server setup + helper methods
│   └── auth/handlers/
│       ├── testmain_test.go                 # TestMain: start/stop shared Fixture
│       ├── users_integration_test.go
│       ├── token_integration_test.go
│       └── api_keys_integration_test.go
├── oapi-codegen.client.yaml                 # New oapi-codegen config for client generation
└── generate.go                              # Add client generation directive
```

> **Migration path**: `testinfra/fixture.go` uses `runtime.Caller(0)` to find the source file's
> location and walks up to the monorepo root, then points goose at `infra/migrations/` and
> `infra/ch_migrations/` directly. This is stable regardless of where `go test` is invoked from.

---

## New Dependencies

```
github.com/testcontainers/testcontainers-go
github.com/testcontainers/testcontainers-go/modules/postgres
github.com/testcontainers/testcontainers-go/modules/clickhouse
github.com/pressly/goose/v3
```

---

## Step 1 — API Client Generation

New file `oapi-codegen.client.yaml`:
```yaml
package: apiclient
output: internal/apiclient/client.gen.go
generate:
  client: true
  models: true
```

Update `generate.go` to add:
```go
//go:generate oapi-codegen --config=oapi-codegen.client.yaml cmd/server/openapi.gen.json
```

Run: `go generate ./...` to produce `internal/apiclient/client.gen.go`.

The generated client exposes typed methods like:
```go
client.CreateUser(ctx, apiclient.NewUser{Username: "alice", Email: "alice@example.com"})
client.CreateToken(ctx, apiclient.TokenRequest{ClientId: id, ClientSecret: secret})
client.ListApiKeys(ctx) // with Authorization header
```

---

## Step 2 — Router Extraction

Extract router setup from `cmd/server/main.go` into `internal/server/router.go`:

```go
// internal/server/router.go
package server

func NewRouter(db *gorm.DB, chDB *sql.DB, jwtSecret []byte) *gin.Engine {
    swagger, _ := shared.GetSwagger()
    swagger.Servers = nil

    router := gin.New()  // gin.New() not gin.Default() — caller adds what it needs
    router.Use(ginmiddleware.OapiRequestValidator(swagger))
    router.Use(authhandlers.Auth(jwtSecret))
    // ... compose all handlers, register routes
    return router
}
```

- `main.go` calls `server.NewRouter(...)` after setting up Prometheus middleware
- `testinfra/fixture.go` calls `server.NewRouter(...)` directly — no Prometheus (avoids global registry conflicts in tests)

---

## Step 3 — Shared Test Infrastructure (`internal/testinfra/`)

### `fixture.go`

```go
package testinfra

import (
    "context"
    "database/sql"
    "net/http/httptest"
    "path/filepath"
    "runtime"

    _ "github.com/ClickHouse/clickhouse-go/v2"
    tcpostgres "github.com/testcontainers/testcontainers-go/modules/postgres"
    tcclickhouse "github.com/testcontainers/testcontainers-go/modules/clickhouse"
    "github.com/pressly/goose/v3"
    "gorm.io/driver/postgres"
    "gorm.io/gorm"
    "github.com/vitaliizinchenko/lab/internal/server"
)

// repoRoot resolves the monorepo root using the source file's path at compile time.
// Stable regardless of the working directory when tests are run.
func repoRoot() string {
    _, filename, _, _ := runtime.Caller(0)
    // .../services/server/internal/testinfra/fixture.go → walk up to lab/
    return filepath.Join(filepath.Dir(filename), "..", "..", "..", "..", "..")
}

type Fixture struct {
    DB        *gorm.DB
    ChDB      *sql.DB
    Server    *httptest.Server
    JWTSecret []byte
    cleanup   []func()
}

func (f *Fixture) Cleanup() {
    for i := len(f.cleanup) - 1; i >= 0; i-- {
        f.cleanup[i]()
    }
}

func NewFixture(ctx context.Context) (*Fixture, error) {
    f := &Fixture{JWTSecret: []byte("test-jwt-secret")}
    root := repoRoot()

    // 1. Start Postgres container
    pgContainer, _ := tcpostgres.Run(ctx, "postgres:16-alpine",
        tcpostgres.WithDatabase("testdb"),
        tcpostgres.WithUsername("test"),
        tcpostgres.WithPassword("test"),
        tcpostgres.BasicWaitStrategies(),
    )
    f.cleanup = append(f.cleanup, func() { pgContainer.Terminate(ctx) })

    // 2. Connect GORM
    dsn, _ := pgContainer.ConnectionString(ctx, "sslmode=disable")
    db, _ := gorm.Open(postgres.Open(dsn), &gorm.Config{})
    f.DB = db

    // 3. Run Postgres migrations
    sqlDB, _ := db.DB()
    goose.SetDialect("postgres")
    goose.Up(sqlDB, filepath.Join(root, "infra", "migrations"))

    // 4. Start ClickHouse container
    chContainer, _ := tcclickhouse.Run(ctx, "clickhouse/clickhouse-server:24-alpine")
    f.cleanup = append(f.cleanup, func() { chContainer.Terminate(ctx) })

    chDSN, _ := chContainer.ConnectionString(ctx)
    chDB, _ := sql.Open("clickhouse", chDSN)
    f.ChDB = chDB

    // 5. Run ClickHouse migrations
    goose.SetDialect("clickhouse")
    goose.Up(chDB, filepath.Join(root, "infra", "ch_migrations"))

    // 6. Build Gin router + httptest.Server
    router := server.NewRouter(db, chDB, f.JWTSecret)
    f.Server = httptest.NewServer(router)
    f.cleanup = append(f.cleanup, f.Server.Close)

    return f, nil
}
```

These helper methods live on `Fixture` in `fixture.go`:

```go
// CreateTestUser creates a user via the API and returns the response.
func (f *Fixture) CreateTestUser(ctx context.Context, username, email string) (apiclient.User, error)

// CreateTestApiKey creates an API key for the given userID.
func (f *Fixture) CreateTestApiKey(ctx context.Context, userID int64) (apiclient.CreatedApiKey, error)

// GetToken exchanges client credentials for a JWT string.
func (f *Fixture) GetToken(ctx context.Context, clientID, clientSecret string) (string, error)

// TruncateTables clears all tables between tests. Call via t.Cleanup().
func (f *Fixture) TruncateTables(t *testing.T)
```

---

## Step 4 — Test Files

### `testmain_test.go`

```go
package handlers_test

var fixture *testinfra.Fixture

func TestMain(m *testing.M) {
    ctx := context.Background()
    var err error
    fixture, err = testinfra.NewFixture(ctx)
    if err != nil {
        log.Fatalf("setup failed: %v", err)
    }
    code := m.Run()
    fixture.Cleanup()
    os.Exit(code)
}
```

### `users_integration_test.go`

| Test | Scenario | Expected |
|------|----------|----------|
| `TestCreateUser_Success` | Valid username + email | 201, correct fields returned |
| `TestCreateUser_DuplicateUsername` | Same username, different email | 409 |
| `TestCreateUser_DuplicateEmail` | Different username, same email | 409 |
| `TestCreateUser_InvalidEmail` | Malformed email string | 400 (OpenAPI validation) |
| `TestCreateUser_MissingUsername` | Empty username field | 400 |

### `token_integration_test.go`

| Test | Scenario | Expected |
|------|----------|----------|
| `TestCreateToken_Success` | Valid client_id + client_secret | 200, Bearer JWT |
| `TestCreateToken_WrongClientID` | Non-existent client_id | 401 |
| `TestCreateToken_WrongSecret` | Correct client_id, wrong secret | 401 |
| `TestCreateToken_RevokedKey` | Key with revoked_at set | 401 |
| `TestCreateToken_ExpiredKey` | Key with expires_at in the past | 401 |
| `TestCreateToken_JWTClaimsValid` | Decode returned JWT | issuer=lab-api, sub=userID, exp≈1h |
| `TestCreateToken_UpdatesLastUsedAt` | Token exchange, then check DB | last_used_at set (async, short retry) |

### `api_keys_integration_test.go`

| Test | Scenario | Expected |
|------|----------|----------|
| `TestCreateApiKey_Success` | Valid user_id | 201, secret returned once |
| `TestCreateApiKey_UserNotFound` | Non-existent user_id | 404 |
| `TestCreateApiKey_WithExpiry` | ExpiresAt in the future | 201, key usable for token |
| `TestListApiKeys_Success` | Authenticated user with 2 keys | 200, list of 2 |
| `TestListApiKeys_Unauthenticated` | No Authorization header | 401 |
| `TestListApiKeys_OnlyOwnKeys` | Two users, each has keys | Each sees only their own |
| `TestRevokeApiKey_Success` | Own key | 204, subsequent token exchange fails |
| `TestRevokeApiKey_NotFound` | Non-existent clientId | 404 |
| `TestRevokeApiKey_OtherUsersKey` | Key belongs to different user | 404 |
| `TestRevokeApiKey_Unauthenticated` | No Authorization header | 401 |

---

## Running Tests

```bash
# All tests (containers start automatically)
go test ./...

# Auth handlers only, verbose
go test -v -timeout 120s ./internal/auth/handlers/...
```

---

## Verification

1. `go generate ./...` produces `internal/apiclient/client.gen.go` without errors
2. `go build ./...` compiles cleanly
3. `go test -v -timeout 120s ./internal/auth/handlers/...` runs all tests green
4. Each test is independently runnable — tables truncated in `t.Cleanup()`
5. Containers start once per `go test` invocation via `TestMain`, not per test
