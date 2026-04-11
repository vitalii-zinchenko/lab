package repos

import (
	"context"
	"time"

	"gorm.io/gorm"

	"github.com/vitaliizinchenko/lab/internal/usage/models"
	"github.com/vitaliizinchenko/lab/internal/usage/repos/query"
)

type pgUsageRepository struct {
	q *query.Query
}

// NewPostgresUsageRepository returns a UsageRepository backed by the given *gorm.DB.
func NewPostgresUsageRepository(db *gorm.DB) UsageRepository {
	return &pgUsageRepository{q: query.Use(db)}
}

func (r *pgUsageRepository) Record(ctx context.Context, entry *models.Usage) error {
	return r.q.Usage.WithContext(ctx).Create(entry)
}

func (r *pgUsageRepository) GetDailyStats(ctx context.Context, userID int64, from, to time.Time) ([]models.DailyStats, error) {
	var stats []models.DailyStats
	err := r.q.Usage.WithContext(ctx).UnderlyingDB().
		Select("date_trunc('day', timestamp) AS date, COUNT(*) AS count").
		Where("user_id = ? AND timestamp >= ? AND timestamp < ?", userID, from, to).
		Group("date_trunc('day', timestamp)").
		Order("date").
		Scan(&stats).Error
	return stats, err
}
