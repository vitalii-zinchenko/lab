package models

import "time"

// User is the GORM model for the users table.
type User struct {
	ID           int64     `gorm:"primaryKey;autoIncrement"`
	Username     string    `gorm:"type:varchar(255);not null;uniqueIndex"`
	Email        string    `gorm:"type:varchar(255);not null;uniqueIndex"`
	PasswordHash *string   `gorm:"type:varchar(255)"`
	CreatedAt    time.Time `gorm:"type:timestamptz;not null"`
}

func (User) TableName() string {
	return "users"
}
