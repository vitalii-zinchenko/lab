# Storage: Plain PostgreSQL

## When to choose this
- Want a single database to operate
- Need to join usage data with users/api_keys in one query (future)
- Scale is moderate (tens of millions of rows, not billions)

## Migration

**New file**: `infra/migrations/00006_create_api_usage_table.sql`

```sql
-- +goose Up
CREATE TABLE IF NOT EXISTS api_usage (
    timestamp   TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    user_id     BIGINT      NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    operation   TEXT        NOT NULL
);

CREATE INDEX idx_api_usage_user_timestamp ON api_usage (user_id, timestamp DESC);

-- +goose Down
DROP TABLE IF EXISTS api_usage;
```

- Index on `(user_id, timestamp DESC)` covers the `WHERE user_id = ? AND timestamp BETWEEN ? AND ?` query pattern.
- No primary key — append-only table, no need for row identity.
- Foreign key to `users` gives referential integrity and cascading deletes.

## Code Generation

**Modify**: `cmd/gen/main.go` — add a usage section:

```go
// Usage
g = gen.NewGenerator(gen.Config{
    OutPath:      "./internal/usage/repos/query",
    ModelPkgPath: "./internal/usage/models",
    Mode:         gen.WithoutContext | gen.WithDefaultQuery | gen.WithQueryInterface,
})
g.ApplyBasic(usagemodels.ApiUsage{})
g.Execute()
```

Then run `go run ./generate.go` to produce `internal/usage/repos/query/`.

## Repository Implementation

**New file**: `internal/usage/repos/pg_usage.go`

Follows the same pattern as `internal/items/repos/item.go` — holds a `*query.Query` from the generated package.
For standard writes, use the generated DAO. For the aggregation query, use `UnderlyingDB()` to access the underlying `*gorm.DB`.

```go
type pgUsageRepository struct {
    q *query.Query
}

func NewPostgresUsageRepository(db *gorm.DB) UsageRepository {
    return &pgUsageRepository{q: query.Use(db)}
}

func (r *pgUsageRepository) Record(ctx context.Context, entry models.ApiUsage) error {
    return r.q.ApiUsage.WithContext(ctx).Create(&entry)
}

func (r *pgUsageRepository) GetDailyStats(ctx context.Context, userID int64, from, to time.Time) ([]models.DailyStats, error) {
    var stats []models.DailyStats
    err := r.q.ApiUsage.WithContext(ctx).UnderlyingDB().
        Select("date_trunc('day', timestamp) AS date, COUNT(*) AS count").
        Where("user_id = ? AND timestamp >= ? AND timestamp < ?", userID, from, to).
        Group("date_trunc('day', timestamp)").
        Order("date").
        Scan(&stats).Error
    return stats, err
}
```

## Router Wiring

```go
usageRepo := usagerepos.NewPostgresUsageRepository(db)
```

Pass `db` (`*gorm.DB`) — already available in `NewRouter`.

## Trade-offs

| Pro | Con |
|-----|-----|
| Single database to operate | Row-based storage — slower aggregations at very large scale |
| Can join usage with users/api_keys | Needs index tuning as rows grow |
| Same tooling, driver, migrations as rest of app | No columnar compression |
| Referential integrity via foreign key | |
