package models

import (
	"time"

	"github.com/google/uuid"
)

type EventLevel string

const (
	EventLevelError EventLevel = "error"
	EventLevelWarn  EventLevel = "warn"
	EventLevelInfo  EventLevel = "info"
)

// EventHistory is the GORM model for the event_history table.
type EventHistory struct {
	ID        uuid.UUID  `gorm:"type:uuid;primaryKey"`
	Level     EventLevel `gorm:"type:event_level;not null"`
	EventType string     `gorm:"type:text;not null"`
	Details   *string    `gorm:"type:text"`
	CreatedAt time.Time  `gorm:"type:timestamptz;not null"`
}

func (EventHistory) TableName() string {
	return "event_history"
}
