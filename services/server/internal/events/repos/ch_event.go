package repos

import (
	"context"
	"database/sql"
	"time"

	"github.com/google/uuid"

	"github.com/vitaliizinchenko/lab/internal/events/models"
)

// ChEventRepository defines persistence operations for ClickHouse events.
type ChEventRepository interface {
	Create(ctx context.Context, event models.EventHistory) (models.EventHistory, error)
}

type chEventRepository struct {
	db *sql.DB
}

// NewChEventRepository returns a ChEventRepository backed by the given *sql.DB.
func NewChEventRepository(db *sql.DB) ChEventRepository {
	return &chEventRepository{db: db}
}

func (r *chEventRepository) Create(ctx context.Context, event models.EventHistory) (models.EventHistory, error) {
	if event.ID == uuid.Nil {
		event.ID = uuid.New()
	}
	if event.CreatedAt.IsZero() {
		event.CreatedAt = time.Now().UTC()
	}

	_, err := r.db.ExecContext(ctx,
		"INSERT INTO ch_events (id, level, event_type, details, created_at) VALUES (?, ?, ?, ?, ?)",
		event.ID, string(event.Level), event.EventType, event.Details, event.CreatedAt,
	)
	if err != nil {
		return models.EventHistory{}, err
	}

	return event, nil
}
