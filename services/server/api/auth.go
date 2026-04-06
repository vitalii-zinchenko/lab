package api

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"fmt"
	"strconv"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	openapi_types "github.com/oapi-codegen/runtime/types"
	"golang.org/x/crypto/bcrypt"

	"github.com/vitaliizinchenko/lab/api/middleware"
	"github.com/vitaliizinchenko/lab/model"
	"github.com/vitaliizinchenko/lab/repository"
)

const (
	tokenTTL       = time.Hour
	bcryptCost     = 12
	secretByteLen  = 32
	jwtIssuer      = "lab-api"
)

// AuthHandler handles token issuance and API key management.
type AuthHandler struct {
	apiKeyRepo repository.ApiKeyRepository
	userRepo   repository.UserRepository
	jwtSecret  []byte
}

// CreateToken exchanges a client_id + client_secret for a signed JWT.
func (h *AuthHandler) CreateToken(ctx context.Context, req CreateTokenRequestObject) (CreateTokenResponseObject, error) {
	clientID := uuid.UUID(req.Body.ClientId)
	key, err := h.apiKeyRepo.GetByClientID(ctx, clientID)
	if errors.Is(err, repository.ErrNotFound) {
		return CreateToken401JSONResponse{Message: "invalid credentials"}, nil
	}
	if err != nil {
		return nil, err
	}

	// Reject revoked keys.
	if key.RevokedAt != nil {
		return CreateToken401JSONResponse{Message: "api key has been revoked"}, nil
	}

	// Reject expired keys.
	if key.ExpiresAt != nil && key.ExpiresAt.Before(time.Now()) {
		return CreateToken401JSONResponse{Message: "api key has expired"}, nil
	}

	// Validate the secret.
	if err := bcrypt.CompareHashAndPassword([]byte(key.ClientSecretHash), []byte(req.Body.ClientSecret)); err != nil {
		return CreateToken401JSONResponse{Message: "invalid credentials"}, nil
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

	// Fire-and-forget last_used_at update — don't fail the request if this errors.
	go func() {
		_ = h.apiKeyRepo.UpdateLastUsedAt(context.Background(), key.ID, now)
	}()

	return CreateToken200JSONResponse{
		AccessToken: signed,
		TokenType:   "Bearer",
		ExpiresIn:   int64(tokenTTL.Seconds()),
	}, nil
}

// CreateApiKey creates a new API key for the given user.
// The raw client secret is returned exactly once; it is not stored.
func (h *AuthHandler) CreateApiKey(ctx context.Context, req CreateApiKeyRequestObject) (CreateApiKeyResponseObject, error) {
	// Verify the user exists by attempting a no-op lookup via the repo.
	// We reuse the user repo's Create to check existence; instead we rely on
	// the FK constraint — if user_id is invalid, the INSERT will fail with a
	// foreign key violation (code 23503).

	rawSecret, err := generateSecret()
	if err != nil {
		return nil, fmt.Errorf("generating secret: %w", err)
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(rawSecret), bcryptCost)
	if err != nil {
		return nil, fmt.Errorf("hashing secret: %w", err)
	}

	now := time.Now().UTC()
	key := model.ApiKey{
		ID:               uuid.New(),
		UserID:           req.Body.UserId,
		ClientID:         uuid.New(),
		ClientSecretHash: string(hash),
		Name:             req.Body.Name,
		CreatedAt:        now,
		ExpiresAt:        req.Body.ExpiresAt,
	}

	created, err := h.apiKeyRepo.Create(ctx, key)
	if err != nil {
		if isForeignKeyViolation(err) {
			return CreateApiKey404JSONResponse{Message: "user not found"}, nil
		}
		return nil, err
	}

	return CreateApiKey201JSONResponse{
		ClientId:     openapi_types.UUID(created.ClientID),
		ClientSecret: rawSecret,
		Name:         created.Name,
		CreatedAt:    created.CreatedAt,
		ExpiresAt:    created.ExpiresAt,
	}, nil
}

// ListApiKeys returns all API keys for the authenticated user.
func (h *AuthHandler) ListApiKeys(ctx context.Context, _ ListApiKeysRequestObject) (ListApiKeysResponseObject, error) {
	userID, ok := middleware.GetUserID(ctx)
	if !ok {
		return ListApiKeys401JSONResponse{Message: "authentication required"}, nil
	}

	keys, err := h.apiKeyRepo.ListByUserID(ctx, userID)
	if err != nil {
		return nil, err
	}

	result := make([]ApiKey, len(keys))
	for i, k := range keys {
		result[i] = toGenApiKey(k)
	}
	return ListApiKeys200JSONResponse(result), nil
}

// RevokeApiKey sets revoked_at on the given key, scoped to the authenticated user.
func (h *AuthHandler) RevokeApiKey(ctx context.Context, req RevokeApiKeyRequestObject) (RevokeApiKeyResponseObject, error) {
	userID, ok := middleware.GetUserID(ctx)
	if !ok {
		return RevokeApiKey401JSONResponse{Message: "authentication required"}, nil
	}

	err := h.apiKeyRepo.Revoke(ctx, uuid.UUID(req.ClientId), userID)
	if errors.Is(err, repository.ErrNotFound) {
		return RevokeApiKey404JSONResponse{Message: "api key not found"}, nil
	}
	if err != nil {
		return nil, err
	}

	return RevokeApiKey204Response{}, nil
}

// toGenApiKey converts a model.ApiKey to the OpenAPI ApiKey schema type.
func toGenApiKey(k model.ApiKey) ApiKey {
	return ApiKey{
		ClientId:   openapi_types.UUID(k.ClientID),
		Name:       k.Name,
		CreatedAt:  k.CreatedAt,
		ExpiresAt:  k.ExpiresAt,
		RevokedAt:  k.RevokedAt,
		LastUsedAt: k.LastUsedAt,
	}
}

// generateSecret produces a cryptographically random URL-safe base64 string.
func generateSecret() (string, error) {
	b := make([]byte, secretByteLen)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return base64.URLEncoding.EncodeToString(b), nil
}

// isForeignKeyViolation returns true for PostgreSQL error code 23503.
func isForeignKeyViolation(err error) bool {
	// Import pgconn via errors.As to detect FK violations without a direct dep in this file.
	type pgErrorCode interface {
		SQLState() string
	}
	var pgErr pgErrorCode
	if errors.As(err, &pgErr) {
		return pgErr.SQLState() == "23503"
	}
	return false
}
