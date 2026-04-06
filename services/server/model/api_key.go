package model

import (
	"time"

	"github.com/google/uuid"
)

// ApiKey is the GORM model for the api_keys table.
type ApiKey struct {
	ID               uuid.UUID  `gorm:"type:uuid;primaryKey"`
	UserID           int64      `gorm:"not null;index"`
	ClientID         uuid.UUID  `gorm:"type:uuid;not null;uniqueIndex"`
	ClientSecretHash string     `gorm:"type:varchar(255);not null"`
	Name             *string    `gorm:"type:varchar(255)"`
	CreatedAt        time.Time  `gorm:"type:timestamptz;not null"`
	ExpiresAt        *time.Time `gorm:"type:timestamptz"`
	RevokedAt        *time.Time `gorm:"type:timestamptz"`
	LastUsedAt       *time.Time `gorm:"type:timestamptz"`
}

func (ApiKey) TableName() string {
	return "api_keys"
}
