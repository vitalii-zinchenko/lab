package repository

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/vitaliizinchenko/lab/model"
	"github.com/vitaliizinchenko/lab/repository/query"
)

// ErrNotFound is returned when a queried item does not exist.
var ErrNotFound = errors.New("item not found")

// ItemRepository defines persistence operations for items.
type ItemRepository interface {
	List(ctx context.Context, limit int) ([]model.Item, error)
	Create(ctx context.Context, item model.Item) (model.Item, error)
	GetByID(ctx context.Context, id uuid.UUID) (model.Item, error)
	Delete(ctx context.Context, id uuid.UUID) error
}

type gormItemRepository struct {
	q *query.Query
}

// NewItemRepository returns an ItemRepository backed by the given *gorm.DB.
func NewItemRepository(db *gorm.DB) ItemRepository {
	return &gormItemRepository{q: query.Use(db)}
}

func (r *gormItemRepository) List(ctx context.Context, limit int) ([]model.Item, error) {
	qi := r.q.Item
	rows, err := qi.WithContext(ctx).Order(qi.CreatedAt.Desc()).Limit(limit).Find()
	if err != nil {
		return nil, err
	}
	items := make([]model.Item, len(rows))
	for i, row := range rows {
		items[i] = *row
	}
	return items, nil
}

func (r *gormItemRepository) Create(ctx context.Context, item model.Item) (model.Item, error) {
	if err := r.q.Item.WithContext(ctx).Create(&item); err != nil {
		return model.Item{}, err
	}
	return item, nil
}

func (r *gormItemRepository) GetByID(ctx context.Context, id uuid.UUID) (model.Item, error) {
	qi := r.q.Item
	row, err := qi.WithContext(ctx).Where(qi.ID.Eq(id)).First()
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return model.Item{}, ErrNotFound
	}
	if err != nil {
		return model.Item{}, err
	}
	return *row, nil
}

func (r *gormItemRepository) Delete(ctx context.Context, id uuid.UUID) error {
	qi := r.q.Item
	info, err := qi.WithContext(ctx).Where(qi.ID.Eq(id)).Delete()
	if err != nil {
		return err
	}
	if info.RowsAffected == 0 {
		return ErrNotFound
	}
	return nil
}
