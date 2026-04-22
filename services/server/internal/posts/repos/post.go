package repos

import (
	"context"
	"errors"
	"time"

	"gorm.io/gorm"

	"github.com/vitaliizinchenko/lab/internal/posts/models"
	"github.com/vitaliizinchenko/lab/internal/posts/repos/query"
	"github.com/vitaliizinchenko/lab/internal/shared"
)

// PostRepository defines persistence operations for posts.
type PostRepository interface {
	Create(ctx context.Context, post models.Post) (models.Post, error)
	GetByID(ctx context.Context, id int64) (models.Post, error)
	// Update sets Content on the post identified by (id, userID).
	// Returns ErrNotFound if no row matched — either the post does not exist
	// or it is not owned by the given user.
	Update(ctx context.Context, id, userID int64, content string) (models.Post, error)
	// ListByUserID returns posts for a user ordered by (created_at DESC, id DESC).
	// afterCreatedAt + afterID form the compound cursor for the next page.
	ListByUserID(ctx context.Context, userID int64, afterCreatedAt *time.Time, afterID int64, limit int) ([]models.Post, error)
}

type gormPostRepository struct {
	q *query.Query
}

// NewPostRepository returns a PostRepository backed by the given *gorm.DB.
func NewPostRepository(db *gorm.DB) PostRepository {
	return &gormPostRepository{q: query.Use(db)}
}

func (r *gormPostRepository) Create(ctx context.Context, post models.Post) (models.Post, error) {
	if err := r.q.Post.WithContext(ctx).Create(&post); err != nil {
		return models.Post{}, err
	}
	return post, nil
}

func (r *gormPostRepository) GetByID(ctx context.Context, id int64) (models.Post, error) {
	qp := r.q.Post
	row, err := qp.WithContext(ctx).Where(qp.ID.Eq(id)).First()
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return models.Post{}, shared.ErrNotFound
	}
	if err != nil {
		return models.Post{}, err
	}
	return *row, nil
}

func (r *gormPostRepository) Update(ctx context.Context, id, userID int64, content string) (models.Post, error) {
	qp := r.q.Post
	info, err := qp.WithContext(ctx).
		Where(qp.ID.Eq(id), qp.UserID.Eq(userID)).
		Updates(models.Post{Content: content})
	if err != nil {
		return models.Post{}, err
	}
	if info.RowsAffected == 0 {
		return models.Post{}, shared.ErrNotFound
	}
	return r.GetByID(ctx, id)
}

func (r *gormPostRepository) ListByUserID(ctx context.Context, userID int64, afterCreatedAt *time.Time, afterID int64, limit int) ([]models.Post, error) {
	qp := r.q.Post
	stmt := qp.WithContext(ctx).Where(qp.UserID.Eq(userID))

	if afterCreatedAt != nil {
		stmt = stmt.Where(
			qp.WithContext(ctx).Where(qp.CreatedAt.Lt(*afterCreatedAt)).
				Or(qp.CreatedAt.Eq(*afterCreatedAt), qp.ID.Lt(afterID)),
		)
	}

	rows, err := stmt.Order(qp.CreatedAt.Desc(), qp.ID.Desc()).Limit(limit).Find()
	if err != nil {
		return nil, err
	}

	posts := make([]models.Post, len(rows))
	for i, row := range rows {
		posts[i] = *row
	}
	return posts, nil
}
