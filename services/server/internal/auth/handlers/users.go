package handlers

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5/pgconn"

	"github.com/vitaliizinchenko/lab/internal/shared"
	"github.com/vitaliizinchenko/lab/internal/auth/models"
	"github.com/vitaliizinchenko/lab/internal/auth/repos"
)

// UsersHandler handles user management endpoints.
type UsersHandler struct {
	userRepo repos.UserRepository
}

// NewUsersHandler creates a UsersHandler with the given repository.
func NewUsersHandler(userRepo repos.UserRepository) *UsersHandler {
	return &UsersHandler{userRepo: userRepo}
}

func toGenUser(u models.User) shared.User {
	return shared.User{
		Id:        u.ID,
		Username:  u.Username,
		Email:     u.Email,
		CreatedAt: u.CreatedAt,
	}
}

func (h *UsersHandler) CreateUser(ctx context.Context, req shared.CreateUserRequestObject) (shared.CreateUserResponseObject, error) {
	user := models.User{
		Username:  req.Body.Username,
		Email:     string(req.Body.Email),
		CreatedAt: time.Now().UTC(),
	}

	created, err := h.userRepo.Create(ctx, user)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			return shared.CreateUser409JSONResponse{Message: "username or email already taken"}, nil
		}
		return nil, err
	}

	return shared.CreateUser201JSONResponse(toGenUser(created)), nil
}
