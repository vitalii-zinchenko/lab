package models

import "time"

// ApiUsage is the GORM model for the api_usage table.
type ApiUsage struct {
	ID        int64     `gorm:"primaryKey;autoIncrement"`
	Timestamp time.Time `gorm:"type:timestamptz;not null"`
	UserID    int64     `gorm:"type:bigint;not null"`
	Operation string    `gorm:"type:text;not null"`
}

func (ApiUsage) TableName() string {
	return "api_usage"
}

// DailyStats holds the result of a daily aggregation query.
type DailyStats struct {
	Date  time.Time
	Count int64
}
