package handlers

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"

	"github.com/vitaliizinchenko/lab/internal/auth/repos"
	"github.com/vitaliizinchenko/lab/internal/shared"
)

// LoginHandler handles password-based login and issues JWTs.
type LoginHandler struct {
	userRepo  repos.UserRepository
	jwtSecret []byte
}

// NewLoginHandler creates a LoginHandler with the given dependencies.
func NewLoginHandler(userRepo repos.UserRepository, jwtSecret []byte) *LoginHandler {
	return &LoginHandler{userRepo: userRepo, jwtSecret: jwtSecret}
}

// Login exchanges email + password for a signed JWT.
func (h *LoginHandler) Login(ctx context.Context, req shared.LoginRequestObject) (shared.LoginResponseObject, error) {
	user, err := h.userRepo.GetByEmail(ctx, string(req.Body.Email))
	if errors.Is(err, shared.ErrNotFound) {
		return shared.Login401JSONResponse{Message: "invalid email or password"}, nil
	}
	if err != nil {
		return nil, fmt.Errorf("get user by email: %w", err)
	}

	if user.PasswordHash == nil {
		return shared.Login401JSONResponse{Message: "invalid email or password"}, nil
	}

	if err := bcryptCompare(*user.PasswordHash, req.Body.Password); err != nil {
		return shared.Login401JSONResponse{Message: "invalid email or password"}, nil
	}

	now := time.Now().UTC()
	claims := jwt.RegisteredClaims{
		Issuer:    jwtIssuer,
		Subject:   strconv.FormatInt(user.ID, 10),
		IssuedAt:  jwt.NewNumericDate(now),
		ExpiresAt: jwt.NewNumericDate(now.Add(tokenTTL)),
		ID:        uuid.New().String(),
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	signed, err := token.SignedString(h.jwtSecret)
	if err != nil {
		return nil, fmt.Errorf("signing token: %w", err)
	}

	return shared.Login200JSONResponse{
		AccessToken: signed,
		TokenType:   "Bearer",
		ExpiresIn:   int64(tokenTTL.Seconds()),
	}, nil
}
