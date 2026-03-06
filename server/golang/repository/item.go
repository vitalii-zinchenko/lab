package repository

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// ErrNotFound is returned when a queried item does not exist.
var ErrNotFound = errors.New("item not found")

// Item is the GORM model for the items table.
type Item struct {
	ID          uuid.UUID `gorm:"type:uuid;primaryKey"`
	Name        string    `gorm:"type:varchar(255);not null"`
	Description *string   `gorm:"type:text"`
	CreatedAt   time.Time `gorm:"type:timestamptz;not null"`
}

func (Item) TableName() string {
	return "items"
}

// ItemRepository defines persistence operations for items.
type ItemRepository interface {
	List(ctx context.Context, limit int) ([]Item, error)
	Create(ctx context.Context, item Item) (Item, error)
	GetByID(ctx context.Context, id uuid.UUID) (Item, error)
	Delete(ctx context.Context, id uuid.UUID) error
}

type gormItemRepository struct {
	db *gorm.DB
}

// NewItemRepository returns an ItemRepository backed by the given *gorm.DB.
func NewItemRepository(db *gorm.DB) ItemRepository {
	return &gormItemRepository{db: db}
}

func (r *gormItemRepository) List(ctx context.Context, limit int) ([]Item, error) {
	var items []Item
	result := r.db.WithContext(ctx).
		Order("created_at DESC").
		Limit(limit).
		Find(&items)
	if result.Error != nil {
		return nil, result.Error
	}
	return items, nil
}

func (r *gormItemRepository) Create(ctx context.Context, item Item) (Item, error) {
	result := r.db.WithContext(ctx).Create(&item)
	if result.Error != nil {
		return Item{}, result.Error
	}
	return item, nil
}

func (r *gormItemRepository) GetByID(ctx context.Context, id uuid.UUID) (Item, error) {
	var item Item
	result := r.db.WithContext(ctx).First(&item, "id = ?", id)
	if errors.Is(result.Error, gorm.ErrRecordNotFound) {
		return Item{}, ErrNotFound
	}
	if result.Error != nil {
		return Item{}, result.Error
	}
	return item, nil
}

func (r *gormItemRepository) Delete(ctx context.Context, id uuid.UUID) error {
	result := r.db.WithContext(ctx).
		Where("id = ?", id).
		Delete(&Item{})
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return ErrNotFound
	}
	return nil
}
