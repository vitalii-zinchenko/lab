package models

import (
	"time"

	"github.com/google/uuid"
)

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
