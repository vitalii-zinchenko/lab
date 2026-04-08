package handlers

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	openapi_types "github.com/oapi-codegen/runtime/types"
	"golang.org/x/crypto/bcrypt"

	"github.com/vitaliizinchenko/lab/internal/auth/models"
	"github.com/vitaliizinchenko/lab/internal/auth/repos"
	"github.com/vitaliizinchenko/lab/internal/shared"
)

const (
	bcryptCost    = 12
	secretByteLen = 32
)

// ApiKeysHandler handles API key creation, listing, and revocation.
type ApiKeysHandler struct {
	apiKeyRepo repos.ApiKeyRepository
}

// NewApiKeysHandler creates an ApiKeysHandler with the given dependencies.
func NewApiKeysHandler(apiKeyRepo repos.ApiKeyRepository) *ApiKeysHandler {
	return &ApiKeysHandler{apiKeyRepo: apiKeyRepo}
}

// CreateApiKey creates a new API key for the given user.
// The raw client secret is returned exactly once; it is not stored.
func (h *ApiKeysHandler) CreateApiKey(ctx context.Context, req shared.CreateApiKeyRequestObject) (shared.CreateApiKeyResponseObject, error) {
	rawSecret, err := generateSecret()
	if err != nil {
		return nil, fmt.Errorf("generating secret: %w", err)
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(rawSecret), bcryptCost)
	if err != nil {
		return nil, fmt.Errorf("hashing secret: %w", err)
	}

	now := time.Now().UTC()
	key := models.ApiKey{
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
		if shared.IsForeignKeyViolation(err) {
			return shared.CreateApiKey404JSONResponse{Message: "user not found"}, nil
		}
		return nil, err
	}

	return shared.CreateApiKey201JSONResponse{
		ClientId:     openapi_types.UUID(created.ClientID),
		ClientSecret: rawSecret,
		Name:         created.Name,
		CreatedAt:    created.CreatedAt,
		ExpiresAt:    created.ExpiresAt,
	}, nil
}

// ListApiKeys returns all API keys for the authenticated user.
func (h *ApiKeysHandler) ListApiKeys(ctx context.Context, _ shared.ListApiKeysRequestObject) (shared.ListApiKeysResponseObject, error) {
	userID, ok := GetUserID(ctx)
	if !ok {
		return shared.ListApiKeys401JSONResponse{Message: "authentication required"}, nil
	}

	keys, err := h.apiKeyRepo.ListByUserID(ctx, userID)
	if err != nil {
		return nil, err
	}

	result := make([]shared.ApiKey, len(keys))
	for i, k := range keys {
		result[i] = toGenApiKey(k)
	}
	return shared.ListApiKeys200JSONResponse(result), nil
}

// RevokeApiKey sets revoked_at on the given key, scoped to the authenticated user.
func (h *ApiKeysHandler) RevokeApiKey(ctx context.Context, req shared.RevokeApiKeyRequestObject) (shared.RevokeApiKeyResponseObject, error) {
	userID, ok := GetUserID(ctx)
	if !ok {
		return shared.RevokeApiKey401JSONResponse{Message: "authentication required"}, nil
	}

	err := h.apiKeyRepo.Revoke(ctx, uuid.UUID(req.ClientId), userID)
	if errors.Is(err, shared.ErrNotFound) {
		return shared.RevokeApiKey404JSONResponse{Message: "api key not found"}, nil
	}
	if err != nil {
		return nil, err
	}

	return shared.RevokeApiKey204Response{}, nil
}

// toGenApiKey converts a models.ApiKey to the OpenAPI ApiKey schema type.
func toGenApiKey(k models.ApiKey) shared.ApiKey {
	return shared.ApiKey{
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

// bcryptCompare wraps bcrypt.CompareHashAndPassword for use across handlers.
func bcryptCompare(hash, secret string) error {
	return bcrypt.CompareHashAndPassword([]byte(hash), []byte(secret))
}
