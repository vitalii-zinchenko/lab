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

const (
	tokenTTL  = time.Hour
	jwtIssuer = "lab-api"
)

// TokenHandler handles token issuance via the OAuth client credentials flow.
type TokenHandler struct {
	apiKeyRepo repos.ApiKeyRepository
	jwtSecret  []byte
}

// NewTokenHandler creates a TokenHandler with the given dependencies.
func NewTokenHandler(apiKeyRepo repos.ApiKeyRepository, jwtSecret []byte) *TokenHandler {
	return &TokenHandler{apiKeyRepo: apiKeyRepo, jwtSecret: jwtSecret}
}

// CreateToken exchanges a client_id + client_secret for a signed JWT.
func (h *TokenHandler) CreateToken(ctx context.Context, req shared.CreateTokenRequestObject) (shared.CreateTokenResponseObject, error) {
	clientID := uuid.UUID(req.Body.ClientId)
	key, err := h.apiKeyRepo.GetByClientID(ctx, clientID)
	if errors.Is(err, shared.ErrNotFound) {
		return shared.CreateToken401JSONResponse{Message: "invalid credentials"}, nil
	}
	if err != nil {
		return nil, err
	}

	// Reject revoked keys.
	if key.RevokedAt != nil {
		return shared.CreateToken401JSONResponse{Message: "api key has been revoked"}, nil
	}

	// Reject expired keys.
	if key.ExpiresAt != nil && key.ExpiresAt.Before(time.Now()) {
		return shared.CreateToken401JSONResponse{Message: "api key has expired"}, nil
	}

	// Validate the secret.
	if err := bcryptCompare(key.ClientSecretHash, req.Body.ClientSecret); err != nil {
		return shared.CreateToken401JSONResponse{Message: "invalid credentials"}, nil
	}

	// Issue JWT.
	now := time.Now().UTC()
	claims := jwt.RegisteredClaims{
		Issuer:    jwtIssuer,
		Subject:   strconv.FormatInt(key.UserID, 10),
		IssuedAt:  jwt.NewNumericDate(now),
		ExpiresAt: jwt.NewNumericDate(now.Add(tokenTTL)),
		ID:        uuid.New().String(),
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	signed, err := token.SignedString(h.jwtSecret)
	if err != nil {
		return nil, fmt.Errorf("signing token: %w", err)
	}

	if err := h.apiKeyRepo.UpdateLastUsedAt(ctx, key.ID, now); err != nil {
		return nil, fmt.Errorf("updating last_used_at: %w", err)
	}

	return shared.CreateToken200JSONResponse{
		AccessToken: signed,
		TokenType:   "Bearer",
		ExpiresIn:   int64(tokenTTL.Seconds()),
	}, nil
}
