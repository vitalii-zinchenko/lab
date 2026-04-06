package repository

import (
	"context"

	"gorm.io/gorm"

	"github.com/vitaliizinchenko/lab/model"
	"github.com/vitaliizinchenko/lab/repository/query"
)

// UserRepository defines persistence operations for users.
type UserRepository interface {
	Create(ctx context.Context, user model.User) (model.User, error)
}

type gormUserRepository struct {
	q *query.Query
}

// NewUserRepository returns a UserRepository backed by the given *gorm.DB.
func NewUserRepository(db *gorm.DB) UserRepository {
	return &gormUserRepository{q: query.Use(db)}
}

func (r *gormUserRepository) Create(ctx context.Context, user model.User) (model.User, error) {
	if err := r.q.User.WithContext(ctx).Create(&user); err != nil {
		return model.User{}, err
	}
	return user, nil
}
