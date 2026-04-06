package api

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5/pgconn"
	"github.com/vitaliizinchenko/lab/model"
	"github.com/vitaliizinchenko/lab/repository"
)

type UsersHandler struct {
	userRepo repository.UserRepository
}

func toGenUser(u model.User) User {
	return User{
		Id:        u.ID,
		Username:  u.Username,
		Email:     u.Email,
		CreatedAt: u.CreatedAt,
	}
}

func (h *UsersHandler) CreateUser(ctx context.Context, req CreateUserRequestObject) (CreateUserResponseObject, error) {
	user := model.User{
		Username:  req.Body.Username,
		Email:     string(req.Body.Email),
		CreatedAt: time.Now().UTC(),
	}

	created, err := h.userRepo.Create(ctx, user)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			return CreateUser409JSONResponse{Message: "username or email already taken"}, nil
		}
		return nil, err
	}

	return CreateUser201JSONResponse(toGenUser(created)), nil
}
