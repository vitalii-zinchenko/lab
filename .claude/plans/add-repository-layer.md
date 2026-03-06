# Plan: Add GORM Repository Layer + Items Migration

## Context
The server currently stores items in an in-memory map with a mutex (handler.go comment: "replace with a real DB later"). The infrastructure already has PostgreSQL + PgBouncer running and DATABASE_URL is configured. This plan wires the existing DB infra into the server by introducing a repository layer and adds the `items` table migration.

---

## Files to Create

### 1. Migration — `/Users/vitaliizinchenko/Projects/lab/infra/migrations/00002_create_items_table.sql`
```sql
-- +goose Up
CREATE TABLE IF NOT EXISTS items (
    id          UUID         PRIMARY KEY DEFAULT gen_random_uuid(),
    name        VARCHAR(255) NOT NULL,
    description TEXT,
    created_at  TIMESTAMPTZ  NOT NULL DEFAULT NOW()
);

-- +goose Down
DROP TABLE IF EXISTS items;
```
- UUID PK (matches the `gen.Item.Id` which is `openapi_types.UUID` = `[16]byte`)
- `description` is nullable (maps to `*string`)
- Follows the same Goose pattern as `00001_create_users_table.sql`

### 2. Repository Package — `/Users/vitaliizinchenko/Projects/lab/server/golang/repository/item.go`

**GORM model:**
```go
type Item struct {
    ID          uuid.UUID `gorm:"type:uuid;primaryKey"`
    Name        string    `gorm:"type:varchar(255);not null"`
    Description *string   `gorm:"type:text"`
    CreatedAt   time.Time `gorm:"type:timestamptz;not null"`
}
func (Item) TableName() string { return "items" }
```

**Sentinel error:**
```go
var ErrNotFound = errors.New("item not found")
```

**Interface:**
```go
type ItemRepository interface {
    List(ctx context.Context, limit int) ([]Item, error)
    Create(ctx context.Context, item Item) (Item, error)
    GetByID(ctx context.Context, id uuid.UUID) (Item, error)
    Delete(ctx context.Context, id uuid.UUID) error
}
```

**Constructor:** `func NewItemRepository(db *gorm.DB) ItemRepository`

**Key implementation notes:**
- `List`: `ORDER BY created_at DESC` + `LIMIT` — deterministic newest-first ordering
- `Create`: UUID and CreatedAt set by the caller (handler), GORM issues a single INSERT
- `GetByID`: translates `gorm.ErrRecordNotFound` → `ErrNotFound`
- `Delete`: checks `RowsAffected == 0` → `ErrNotFound` (GORM Delete never returns ErrRecordNotFound)

---

## Files to Modify

### 3. Handler — `/Users/vitaliizinchenko/Projects/lab/server/golang/handler/handler.go`

- Remove `sync.RWMutex` + in-memory `map`
- `Handler` struct holds `items repository.ItemRepository`
- `New(items repository.ItemRepository) *Handler`
- Add private `toGenItem(r repository.Item) gen.Item` helper (type cast: `openapi_types.UUID(r.ID)` and reverse `uuid.UUID(req.Id)` — both are `[16]byte`, zero-cost)
- All methods pass `ctx` to repository calls
- 404 errors: `errors.Is(err, repository.ErrNotFound)` → return the appropriate 404 response
- Other errors: return `nil, err` → strict handler produces HTTP 500

### 4. Main — `/Users/vitaliizinchenko/Projects/lab/server/golang/cmd/server/main.go`

```go
dbURL := os.Getenv("DATABASE_URL")
if dbURL == "" {
    log.Fatal("DATABASE_URL environment variable is required")
}
db, err := gorm.Open(postgres.Open(dbURL), &gorm.Config{})
if err != nil {
    log.Fatalf("failed to connect to database: %v", err)
}
itemRepo := repository.NewItemRepository(db)
h := handler.New(itemRepo)
```

- No `AutoMigrate` — DDL goes through Goose, not GORM (PgBouncer transaction pool mode incompatible with DDL in transactions)

### 5. go.mod — `/Users/vitaliizinchenko/Projects/lab/server/golang/go.mod`

Add via `go get`:
```
gorm.io/gorm
gorm.io/driver/postgres  (brings in pgx/v5 as indirect dep)
```
Run `go mod tidy` after.

---

## Implementation Order

1. `go get gorm.io/gorm gorm.io/driver/postgres && go mod tidy` in the golang server dir
2. Create migration file
3. Create `repository/item.go`
4. Update `handler/handler.go`
5. Update `cmd/server/main.go`

---

## Verification

1. Apply migration: run the migrate tool against the local DB
2. `docker compose up api` (or air hot reload picks it up)
3. Smoke test:
   ```bash
   curl -X POST http://localhost:8080/items -d '{"name":"test","description":"hello"}' -H 'Content-Type: application/json'
   curl http://localhost:8080/items
   curl http://localhost:8080/items/<returned-uuid>
   curl -X DELETE http://localhost:8080/items/<returned-uuid>
   curl http://localhost:8080/items/<deleted-uuid>  # expect 404
   ```

---

## What Does NOT Change
- `gen/api.gen.go` — generated code, never touched
- `infra/migrate/main.go` — migration runner unchanged
- `infra/docker-compose.yml` — infra unchanged
- `server/golang/docker-compose.yml` — DATABASE_URL already configured
- `api/openapi.yaml` — spec unchanged
