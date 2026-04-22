package models

import "time"

// Post is the GORM model for the posts table.
type Post struct {
	ID        int64     `gorm:"primaryKey;autoIncrement"`
	UserID    int64     `gorm:"type:integer;not null;index"`
	Content   string    `gorm:"type:text;not null"`
	CreatedAt time.Time `gorm:"type:timestamptz;not null;autoCreateTime"`
	UpdatedAt time.Time `gorm:"type:timestamptz;not null;autoUpdateTime"`
}

func (Post) TableName() string {
	return "posts"
}
