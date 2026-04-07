package repos

import (
	"context"

	"gorm.io/gorm"

	"github.com/vitaliizinchenko/lab/internal/auth/models"
	"github.com/vitaliizinchenko/lab/internal/auth/repos/query"
)

// UserRepository defines persistence operations for users.
type UserRepository interface {
	Create(ctx context.Context, user models.User) (models.User, error)
}

type gormUserRepository struct {
	q *query.Query
}

// NewUserRepository returns a UserRepository backed by the given *gorm.DB.
func NewUserRepository(db *gorm.DB) UserRepository {
	return &gormUserRepository{q: query.Use(db)}
}

func (r *gormUserRepository) Create(ctx context.Context, user models.User) (models.User, error) {
	if err := r.q.User.WithContext(ctx).Create(&user); err != nil {
		return models.User{}, err
	}
	return user, nil
}
