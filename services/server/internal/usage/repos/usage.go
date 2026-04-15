package repos

import (
	"context"
	"time"

	"github.com/vitaliizinchenko/lab/internal/usage/models"
)

// UsageRepository defines persistence operations for API usage tracking.
type UsageRepository interface {
	Record(ctx context.Context, entry *models.Usage) error
	// ListRecords returns raw usage rows for a user within [from, to), ordered by
	// (timestamp ASC, id ASC). afterTimestamp+afterID form the compound cursor —
	// pass nil for the first page. Returns at most limit rows.
	ListRecords(ctx context.Context, userID int64, from, to time.Time, afterTimestamp *time.Time, afterID int64, limit int) ([]models.Usage, error)
	GetDailyStats(ctx context.Context, userID int64, from, to time.Time) ([]models.DailyStats, error)
}
