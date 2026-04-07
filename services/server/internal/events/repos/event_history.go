package repos

import (
	"context"
	"encoding/base64"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"github.com/vitaliizinchenko/lab/internal/events/models"
	"github.com/vitaliizinchenko/lab/internal/events/repos/query"
)

// EventCursor holds the decoded position for cursor-based pagination.
type EventCursor struct {
	CreatedAt time.Time
	ID        uuid.UUID
}

// EncodeCursor encodes an EventCursor to an opaque base64 string.
func EncodeCursor(c EventCursor) string {
	raw := fmt.Sprintf("%d:%s", c.CreatedAt.UnixNano(), c.ID.String())
	return base64.StdEncoding.EncodeToString([]byte(raw))
}

// DecodeCursor decodes an opaque cursor string back to EventCursor.
func DecodeCursor(s string) (EventCursor, error) {
	b, err := base64.StdEncoding.DecodeString(s)
	if err != nil {
		return EventCursor{}, fmt.Errorf("invalid cursor: %w", err)
	}
	parts := strings.SplitN(string(b), ":", 2)
	if len(parts) != 2 {
		return EventCursor{}, fmt.Errorf("invalid cursor format")
	}
	var nanos int64
	if _, err := fmt.Sscanf(parts[0], "%d", &nanos); err != nil {
		return EventCursor{}, fmt.Errorf("invalid cursor timestamp: %w", err)
	}
	id, err := uuid.Parse(parts[1])
	if err != nil {
		return EventCursor{}, fmt.Errorf("invalid cursor id: %w", err)
	}
	return EventCursor{
		CreatedAt: time.Unix(0, nanos).UTC(),
		ID:        id,
	}, nil
}

// EventFilter holds optional filters for listing events.
type EventFilter struct {
	Level     *models.EventLevel
	EventType *string
	From      *time.Time
	To        *time.Time
	Cursor    *EventCursor
	Limit     int
}

// EventHistoryRepository defines persistence operations for event_history.
type EventHistoryRepository interface {
	Create(ctx context.Context, event models.EventHistory) (models.EventHistory, error)
	List(ctx context.Context, filter EventFilter) ([]models.EventHistory, error)
}

type gormEventHistoryRepository struct {
	q *query.Query
}

// NewEventHistoryRepository returns an EventHistoryRepository backed by the given *gorm.DB.
func NewEventHistoryRepository(db *gorm.DB) EventHistoryRepository {
	return &gormEventHistoryRepository{q: query.Use(db)}
}

func (r *gormEventHistoryRepository) Create(ctx context.Context, event models.EventHistory) (models.EventHistory, error) {
	if err := r.q.EventHistory.WithContext(ctx).Create(&event); err != nil {
		return models.EventHistory{}, err
	}
	return event, nil
}

func (r *gormEventHistoryRepository) List(ctx context.Context, filter EventFilter) ([]models.EventHistory, error) {
	qe := r.q.EventHistory
	do := qe.WithContext(ctx)

	if filter.Level != nil {
		do = do.Where(qe.Level.Eq(string(*filter.Level)))
	}
	if filter.EventType != nil {
		do = do.Where(qe.EventType.Eq(*filter.EventType))
	}
	if filter.From != nil {
		do = do.Where(qe.CreatedAt.Gte(*filter.From))
	}
	if filter.To != nil {
		do = do.Where(qe.CreatedAt.Lte(*filter.To))
	}
	if filter.Cursor != nil {
		do = do.Clauses(clause.Expr{
			SQL:  "(created_at, id) < (?, ?)",
			Vars: []interface{}{filter.Cursor.CreatedAt, filter.Cursor.ID},
		})
	}

	rows, err := do.Order(qe.CreatedAt.Desc(), qe.ID.Desc()).Limit(filter.Limit).Find()
	if err != nil {
		return nil, err
	}
	events := make([]models.EventHistory, len(rows))
	for i, row := range rows {
		events[i] = *row
	}
	return events, nil
}
