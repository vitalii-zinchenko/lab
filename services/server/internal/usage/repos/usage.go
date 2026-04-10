package repos

import (
	"context"
	"time"

	"github.com/vitaliizinchenko/lab/internal/usage/models"
)

// UsageRepository defines persistence operations for API usage tracking.
type UsageRepository interface {
	Record(ctx context.Context, entry models.Usage) error
	GetDailyStats(ctx context.Context, userID int64, from, to time.Time) ([]models.DailyStats, error)
}
