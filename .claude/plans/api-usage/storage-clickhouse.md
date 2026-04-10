# Storage: ClickHouse

## When to choose this
- ClickHouse is already in the stack (zero new dependency)
- Expected write volume is very high (millions of requests/day)
- No need to join usage data with relational tables in a single query

## Migration

**New file**: `infra/ch_migrations/00002_create_api_usage_table.sql`

```sql
CREATE TABLE IF NOT EXISTS api_usage (
    timestamp DateTime NOT NULL,
    user_id   Int64    NOT NULL,
    operation String   NOT NULL
) ENGINE = MergeTree()
ORDER BY (user_id, timestamp);
```

- Ordered by `(user_id, timestamp)` — optimal for per-user date-range scans.
- `MergeTree` is the standard ClickHouse engine for append-only analytics data.

## Repository Implementation

**New file**: `internal/usage/repos/ch_usage.go`

Uses `*sql.DB` with the ClickHouse driver — same pattern as `internal/events/repos/ch_event.go`.

```go
type chUsageRepository struct {
    db *sql.DB
}

func NewClickHouseUsageRepository(db *sql.DB) UsageRepository {
    return &chUsageRepository{db: db}
}

func (r *chUsageRepository) Record(ctx context.Context, entry models.ApiUsage) error {
    _, err := r.db.ExecContext(ctx,
        "INSERT INTO api_usage (timestamp, user_id, operation) VALUES (?, ?, ?)",
        entry.Timestamp, entry.UserID, entry.Operation,
    )
    return err
}

func (r *chUsageRepository) GetDailyStats(ctx context.Context, userID int64, from, to time.Time) ([]models.DailyStats, error) {
    rows, err := r.db.QueryContext(ctx, `
        SELECT toDate(timestamp) AS date, count() AS count
        FROM api_usage
        WHERE user_id = ? AND timestamp >= ? AND timestamp < ?
        GROUP BY date
        ORDER BY date
    `, userID, from, to)
    if err != nil {
        return nil, err
    }
    defer rows.Close()

    var stats []models.DailyStats
    for rows.Next() {
        var s models.DailyStats
        if err := rows.Scan(&s.Date, &s.Count); err != nil {
            return nil, err
        }
        stats = append(stats, s)
    }
    return stats, rows.Err()
}
```

## Router Wiring

```go
usageRepo := usagerepos.NewClickHouseUsageRepository(chDB)
```

Pass `chDB` (`*sql.DB`) — already available in `NewRouter`.

## Trade-offs

| Pro | Con |
|-----|-----|
| Already in stack | Single-row inserts are inefficient at massive scale (batching recommended) |
| Extremely fast aggregation at scale | Cannot join with PostgreSQL user/key data |
| Columnar storage, low disk usage | Separate operational concern from main DB |
