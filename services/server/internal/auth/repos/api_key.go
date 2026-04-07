package repos

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/vitaliizinchenko/lab/internal/auth/models"
	"github.com/vitaliizinchenko/lab/internal/auth/repos/query"
	"github.com/vitaliizinchenko/lab/internal/shared"
)

// ApiKeyRepository defines persistence operations for API keys.
type ApiKeyRepository interface {
	Create(ctx context.Context, key models.ApiKey) (models.ApiKey, error)
	GetByClientID(ctx context.Context, clientID uuid.UUID) (models.ApiKey, error)
	ListByUserID(ctx context.Context, userID int64) ([]models.ApiKey, error)
	Revoke(ctx context.Context, clientID uuid.UUID, userID int64) error
	UpdateLastUsedAt(ctx context.Context, id uuid.UUID, t time.Time) error
}

type gormApiKeyRepository struct {
	q *query.Query
}

// NewApiKeyRepository returns an ApiKeyRepository backed by the given *gorm.DB.
func NewApiKeyRepository(db *gorm.DB) ApiKeyRepository {
	return &gormApiKeyRepository{q: query.Use(db)}
}

func (r *gormApiKeyRepository) Create(ctx context.Context, key models.ApiKey) (models.ApiKey, error) {
	if err := r.q.ApiKey.WithContext(ctx).Create(&key); err != nil {
		return models.ApiKey{}, err
	}
	return key, nil
}

func (r *gormApiKeyRepository) GetByClientID(ctx context.Context, clientID uuid.UUID) (models.ApiKey, error) {
	qk := r.q.ApiKey
	row, err := qk.WithContext(ctx).Where(qk.ClientID.Eq(clientID)).First()
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return models.ApiKey{}, shared.ErrNotFound
	}
	if err != nil {
		return models.ApiKey{}, err
	}
	return *row, nil
}

func (r *gormApiKeyRepository) ListByUserID(ctx context.Context, userID int64) ([]models.ApiKey, error) {
	qk := r.q.ApiKey
	rows, err := qk.WithContext(ctx).Where(qk.UserID.Eq(userID)).Order(qk.CreatedAt.Desc()).Find()
	if err != nil {
		return nil, err
	}
	keys := make([]models.ApiKey, len(rows))
	for i, row := range rows {
		keys[i] = *row
	}
	return keys, nil
}

func (r *gormApiKeyRepository) Revoke(ctx context.Context, clientID uuid.UUID, userID int64) error {
	qk := r.q.ApiKey
	now := time.Now().UTC()
	info, err := qk.WithContext(ctx).
		Where(qk.ClientID.Eq(clientID), qk.UserID.Eq(userID)).
		UpdateSimple(qk.RevokedAt.Value(now))
	if err != nil {
		return err
	}
	if info.RowsAffected == 0 {
		return shared.ErrNotFound
	}
	return nil
}

func (r *gormApiKeyRepository) UpdateLastUsedAt(ctx context.Context, id uuid.UUID, t time.Time) error {
	qk := r.q.ApiKey
	_, err := qk.WithContext(ctx).
		Where(qk.ID.Eq(id)).
		UpdateSimple(qk.LastUsedAt.Value(t))
	return err
}
