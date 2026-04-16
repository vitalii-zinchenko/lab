package repos

import (
	"context"
	"errors"

	"gorm.io/gorm"

	"github.com/vitaliizinchenko/lab/internal/auth/models"
	"github.com/vitaliizinchenko/lab/internal/auth/repos/query"
	"github.com/vitaliizinchenko/lab/internal/shared"
)

// UserRepository defines persistence operations for users.
type UserRepository interface {
	Create(ctx context.Context, user models.User) (models.User, error)
	GetByEmail(ctx context.Context, email string) (models.User, error)
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

func (r *gormUserRepository) GetByEmail(ctx context.Context, email string) (models.User, error) {
	u, err := r.q.User.WithContext(ctx).Where(r.q.User.Email.Eq(email)).First()
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return models.User{}, shared.ErrNotFound
	}
	if err != nil {
		return models.User{}, err
	}
	return *u, nil
}
