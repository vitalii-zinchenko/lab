package models

import "time"

// Usage is the GORM model for the usage table.
type Usage struct {
	ID        int64     `gorm:"primaryKey;autoIncrement"`
	Timestamp time.Time `gorm:"type:timestamptz;not null"`
	UserID    int64     `gorm:"type:bigint;not null"`
	Operation string    `gorm:"type:text;not null"`
}

func (Usage) TableName() string {
	return "usage"
}

// DailyStats holds the result of a daily aggregation query.
type DailyStats struct {
	Date  time.Time
	Count int64
}
