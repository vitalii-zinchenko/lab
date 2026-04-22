package handlers

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"time"

	authhandlers "github.com/vitaliizinchenko/lab/internal/auth/handlers"
	"github.com/vitaliizinchenko/lab/internal/posts/models"
	"github.com/vitaliizinchenko/lab/internal/posts/repos"
	"github.com/vitaliizinchenko/lab/internal/shared"
)

type postCursor struct {
	CreatedAt time.Time `json:"ca"`
	ID        int64     `json:"id"`
}

func encodeCursor(createdAt time.Time, id int64) string {
	b, _ := json.Marshal(postCursor{CreatedAt: createdAt, ID: id})
	return base64.StdEncoding.EncodeToString(b)
}

func decodeCursor(s string) (createdAt time.Time, id int64, ok bool) {
	b, err := base64.StdEncoding.DecodeString(s)
	if err != nil {
		return
	}
	var c postCursor
	if err := json.Unmarshal(b, &c); err != nil {
		return
	}
	return c.CreatedAt, c.ID, true
}

func toPost(p models.Post) shared.Post {
	return shared.Post{
		Id:        p.ID,
		UserId:    p.UserID,
		Content:   p.Content,
		CreatedAt: p.CreatedAt,
		UpdatedAt: p.UpdatedAt,
	}
}

// PostsHandler handles post CRUD endpoints.
type PostsHandler struct {
	repo repos.PostRepository
}

// NewPostsHandler creates a PostsHandler with the given repository.
func NewPostsHandler(repo repos.PostRepository) *PostsHandler {
	return &PostsHandler{repo: repo}
}

func (h *PostsHandler) CreatePost(ctx context.Context, req shared.CreatePostRequestObject) (shared.CreatePostResponseObject, error) {
	userID, ok := authhandlers.GetUserID(ctx)
	if !ok {
		return shared.CreatePost401JSONResponse{Message: "unauthorized"}, nil
	}

	post, err := h.repo.Create(ctx, models.Post{
		UserID:  userID,
		Content: req.Body.Content,
	})
	if err != nil {
		return nil, err
	}

	return shared.CreatePost201JSONResponse(toPost(post)), nil
}

func (h *PostsHandler) GetPost(ctx context.Context, req shared.GetPostRequestObject) (shared.GetPostResponseObject, error) {
	post, err := h.repo.GetByID(ctx, req.Id)
	if errors.Is(err, shared.ErrNotFound) {
		return shared.GetPost404JSONResponse{Message: "post not found"}, nil
	}
	if err != nil {
		return nil, err
	}
	return shared.GetPost200JSONResponse(toPost(post)), nil
}

func (h *PostsHandler) ListPosts(ctx context.Context, req shared.ListPostsRequestObject) (shared.ListPostsResponseObject, error) {
	limit := 20
	if req.Params.Limit != nil {
		limit = *req.Params.Limit
	}

	var afterCreatedAt *time.Time
	var afterID int64
	if req.Params.Cursor != nil && *req.Params.Cursor != "" {
		if ca, id, ok := decodeCursor(*req.Params.Cursor); ok {
			afterCreatedAt = &ca
			afterID = id
		}
	}

	posts, err := h.repo.ListByUserID(ctx, req.Params.UserId, afterCreatedAt, afterID, limit)
	if err != nil {
		return nil, err
	}

	data := make([]shared.Post, len(posts))
	for i, p := range posts {
		data[i] = toPost(p)
	}

	var nextCursor *string
	if len(posts) == limit {
		last := posts[len(posts)-1]
		s := encodeCursor(last.CreatedAt, last.ID)
		nextCursor = &s
	}

	return shared.ListPosts200JSONResponse{Data: data, NextCursor: nextCursor}, nil
}

func (h *PostsHandler) UpdatePost(ctx context.Context, req shared.UpdatePostRequestObject) (shared.UpdatePostResponseObject, error) {
	userID, ok := authhandlers.GetUserID(ctx)
	if !ok {
		return shared.UpdatePost401JSONResponse{Message: "unauthorized"}, nil
	}

	post, err := h.repo.Update(ctx, req.Id, userID, req.Body.Content)
	if errors.Is(err, shared.ErrNotFound) {
		return shared.UpdatePost404JSONResponse{Message: "post not found or not owned by caller"}, nil
	}
	if err != nil {
		return nil, err
	}

	return shared.UpdatePost200JSONResponse(toPost(post)), nil
}
