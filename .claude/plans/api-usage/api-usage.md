# Plan: API Usage Statistics — General

## Context

Track API call counts per user across all their API keys and expose a `GET /usage` endpoint
returning daily call counts for a given date range.

`oapi-codegen`'s strict server already passes `operationID` as a string to every `StrictMiddlewareFunc`,
so we capture it there — no template overrides or external maps needed.

Storage implementation is separate — see `storage-clickhouse.md` or `storage-postgres.md`.

---

## Architecture

```
Request → Gin (auth middleware) → strict handler wrapper → UsageTrackingMiddleware → handler
                                                                    ↓
                                                           repo.Record() async

GET /usage → UsageHandler → repo.GetDailyStats() → [ {date, count}, ... ]
```

---

## Repository Interface

**New file**: `internal/usage/repos/usage.go`

```go
type UsageRepository interface {
    Record(ctx context.Context, entry models.ApiUsage) error
    GetDailyStats(ctx context.Context, userID int64, from, to time.Time) ([]models.DailyStats, error)
}
```

The implementation lives in the storage-specific plan.

---

## Models

**New file**: `internal/usage/models/api_usage.go`

```go
type ApiUsage struct {
    Timestamp time.Time
    UserID    int64
    Operation string
}

type DailyStats struct {
    Date  time.Time
    Count int64
}
```

---

## Usage Tracking Middleware

**New file**: `internal/usage/handlers/middleware.go`

```go
func UsageTrackingMiddleware(repo repos.UsageRepository) shared.StrictMiddlewareFunc {
    return func(f shared.StrictGinHandlerFunc, operationID string) shared.StrictGinHandlerFunc {
        return func(ctx *gin.Context, request interface{}) (interface{}, error) {
            response, err := f(ctx, request)
            userID, ok := authhandlers.GetUserID(ctx.Request.Context())
            if ok {
                go func() {
                    _ = repo.Record(context.Background(), models.ApiUsage{
                        Timestamp: time.Now().UTC(),
                        UserID:    userID,
                        Operation: operationID,
                    })
                }()
            }
            return response, err
        }
    }
}
```

- Only records authenticated requests (user_id from JWT context).
- Fire-and-forget goroutine — zero added latency.
- Naturally excludes `GetHealth`, `CreateToken` (unauthenticated, no user_id).

---

## Usage Handler

**New file**: `internal/usage/handlers/usage.go`

```go
type UsageHandler struct {
    repo repos.UsageRepository
}

func NewUsageHandler(repo repos.UsageRepository) *UsageHandler {
    return &UsageHandler{repo: repo}
}

func (h *UsageHandler) GetUsage(ctx context.Context, req shared.GetUsageRequestObject) (shared.GetUsageResponseObject, error) {
    userID, ok := authhandlers.GetUserID(ctx)
    if !ok {
        return shared.GetUsage401JSONResponse{Message: "unauthorized"}, nil
    }
    stats, err := h.repo.GetDailyStats(ctx, userID, req.Params.From.Time, req.Params.To.Time)
    if err != nil {
        return nil, err
    }
    data := make([]shared.DailyUsage, len(stats))
    for i, s := range stats {
        data[i] = shared.DailyUsage{Date: openapi_types.Date{Time: s.Date}, Count: int(s.Count)}
    }
    return shared.GetUsage200JSONResponse{Data: data}, nil
}
```

---

## OpenAPI Spec

### `internal/usage/spec_schemas.yaml`

```yaml
components:
  schemas:
    DailyUsage:
      type: object
      required: [date, count]
      properties:
        date:
          type: string
          format: date
        count:
          type: integer

    UsageStats:
      type: object
      required: [data]
      properties:
        data:
          type: array
          items:
            $ref: "./spec_schemas.yaml#/components/schemas/DailyUsage"
```

### `internal/usage/spec_paths.yaml`

```yaml
paths:
  /usage:
    get:
      operationId: GetUsage
      summary: Get API usage statistics
      security:
        - BearerAuth: []
      parameters:
        - name: from
          in: query
          required: true
          schema:
            type: string
            format: date-time
        - name: to
          in: query
          required: true
          schema:
            type: string
            format: date-time
      responses:
        "200":
          description: Daily usage statistics
          content:
            application/json:
              schema:
                $ref: "./spec_schemas.yaml#/components/schemas/UsageStats"
        "401":
          description: Unauthorized
          content:
            application/json:
              schema:
                $ref: "../../shared/spec_schemas.yaml#/components/schemas/Error"
```

### Update `cmd/server/spec.yaml`

Add the `/usage` path reference.

---

## Router Wiring

**Modified file**: `internal/shared/server/router.go`

1. Add `*usagehandlers.UsageHandler` to `appHandler` struct.
2. Create `usageRepo` from the chosen storage implementation.
3. Pass tracking middleware to `NewStrictHandler`:
   ```go
   strictHandler := shared.NewStrictHandler(h, []shared.StrictMiddlewareFunc{
       usagehandlers.UsageTrackingMiddleware(usageRepo),
   })
   ```

---

## Files Summary

| Action | File |
|--------|------|
| New | `internal/usage/models/api_usage.go` |
| New | `internal/usage/repos/usage.go` (interface only) |
| New | `internal/usage/handlers/middleware.go` |
| New | `internal/usage/handlers/usage.go` |
| New | `internal/usage/spec_paths.yaml` |
| New | `internal/usage/spec_schemas.yaml` |
| Modify | `cmd/server/spec.yaml` |
| Modify | `internal/shared/server/router.go` |

Reuse:
- `internal/auth/handlers/middleware.go` → `GetUserID(ctx)`
- `internal/items/handlers/items.go` → handler pattern
- `internal/events/repos/ch_event.go` → ClickHouse repo pattern (if using CH)

---

## Verification

1. Run chosen storage migration.
2. `go run ./generate.go` to regenerate `api.gen.go`.
3. Make authenticated requests to any endpoint.
4. `GET /usage?from=...&to=...` with valid JWT → daily counts.
5. `GET /usage` without auth → 401.
6. `from > to` → empty `data` array (not an error).
