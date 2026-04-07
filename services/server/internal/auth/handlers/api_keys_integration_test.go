package handlers_test

import (
	"context"
	"net/http"
	"testing"
	"time"

	"github.com/google/uuid"
	openapi_types "github.com/oapi-codegen/runtime/types"

	"github.com/vitaliizinchenko/lab/internal/shared/apiclient"
)

func TestCreateApiKey_Success(t *testing.T) {
	ctx := context.Background()
	t.Cleanup(func() { fixture.TruncateTables(t) })

	user := fixture.CreateTestUser(t, ctx, "keyuser", "keyuser@example.com")
	resp := fixture.CreateTestApiKey(t, ctx, user.JSON201.Id)

	if resp.StatusCode() != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", resp.StatusCode(), string(resp.Body))
	}
	if resp.JSON201 == nil {
		t.Fatal("expected JSON body")
	}
	if resp.JSON201.ClientSecret == "" {
		t.Error("expected non-empty client_secret")
	}
	if resp.JSON201.ClientId == (openapi_types.UUID{}) {
		t.Error("expected non-zero client_id")
	}
	if resp.JSON201.CreatedAt.IsZero() {
		t.Error("expected non-zero created_at")
	}
}

func TestCreateApiKey_UserNotFound(t *testing.T) {
	ctx := context.Background()

	resp, err := fixture.Client().CreateApiKeyWithResponse(ctx, apiclient.InternalAuthSpecSchemasNewApiKey{
		UserId: 999999,
	})
	if err != nil {
		t.Fatalf("CreateApiKey: %v", err)
	}

	if resp.StatusCode() != http.StatusNotFound {
		t.Fatalf("expected 404, got %d: %s", resp.StatusCode(), string(resp.Body))
	}
}

func TestCreateApiKey_WithFutureExpiry(t *testing.T) {
	ctx := context.Background()
	t.Cleanup(func() { fixture.TruncateTables(t) })

	user := fixture.CreateTestUser(t, ctx, "expiryuser", "expiryuser@example.com")
	future := time.Now().UTC().Add(24 * time.Hour)

	resp, err := fixture.Client().CreateApiKeyWithResponse(ctx, apiclient.InternalAuthSpecSchemasNewApiKey{
		UserId:    user.JSON201.Id,
		ExpiresAt: &future,
	})
	if err != nil || resp.StatusCode() != http.StatusCreated {
		t.Fatalf("CreateApiKey: status=%d err=%v", resp.StatusCode(), err)
	}

	// Key with a future expiry should still be usable for token exchange.
	tokenResp, err := fixture.Client().CreateTokenWithResponse(ctx, apiclient.InternalAuthSpecSchemasTokenRequest{
		ClientId:     resp.JSON201.ClientId,
		ClientSecret: resp.JSON201.ClientSecret,
		GrantType:    apiclient.ClientCredentials,
	})
	if err != nil || tokenResp.StatusCode() != http.StatusOK {
		t.Fatalf("CreateToken with future expiry key: status=%d err=%v", tokenResp.StatusCode(), err)
	}
}

func TestListApiKeys_Success(t *testing.T) {
	ctx := context.Background()
	t.Cleanup(func() { fixture.TruncateTables(t) })

	user := fixture.CreateTestUser(t, ctx, "listuser", "listuser@example.com")
	fixture.CreateTestApiKey(t, ctx, user.JSON201.Id)
	fixture.CreateTestApiKey(t, ctx, user.JSON201.Id)

	// Need a token to list keys.
	key := fixture.CreateTestApiKey(t, ctx, user.JSON201.Id)
	token := fixture.GetToken(t, ctx, key.JSON201.ClientId.String(), key.JSON201.ClientSecret)

	resp, err := fixture.AuthClient(token).ListApiKeysWithResponse(ctx)
	if err != nil {
		t.Fatalf("ListApiKeys: %v", err)
	}
	if resp.StatusCode() != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", resp.StatusCode(), string(resp.Body))
	}
	if len(*resp.JSON200) != 3 {
		t.Errorf("expected 3 keys, got %d", len(*resp.JSON200))
	}
}

func TestListApiKeys_Unauthenticated(t *testing.T) {
	ctx := context.Background()

	resp, err := fixture.Client().ListApiKeysWithResponse(ctx)
	if err != nil {
		t.Fatalf("ListApiKeys: %v", err)
	}

	if resp.StatusCode() != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d: %s", resp.StatusCode(), string(resp.Body))
	}
}

func TestListApiKeys_OnlyOwnKeys(t *testing.T) {
	ctx := context.Background()
	t.Cleanup(func() { fixture.TruncateTables(t) })

	userA := fixture.CreateTestUser(t, ctx, "userA", "userA@example.com")
	userB := fixture.CreateTestUser(t, ctx, "userB", "userB@example.com")

	// Create 2 keys for A, 3 keys for B.
	for i := range 2 {
		_ = i
		fixture.CreateTestApiKey(t, ctx, userA.JSON201.Id)
	}
	for i := range 3 {
		_ = i
		fixture.CreateTestApiKey(t, ctx, userB.JSON201.Id)
	}

	// Use a fresh key to get tokens for each user.
	keyA := fixture.CreateTestApiKey(t, ctx, userA.JSON201.Id)
	keyB := fixture.CreateTestApiKey(t, ctx, userB.JSON201.Id)
	tokenA := fixture.GetToken(t, ctx, keyA.JSON201.ClientId.String(), keyA.JSON201.ClientSecret)
	tokenB := fixture.GetToken(t, ctx, keyB.JSON201.ClientId.String(), keyB.JSON201.ClientSecret)

	respA, _ := fixture.AuthClient(tokenA).ListApiKeysWithResponse(ctx)
	respB, _ := fixture.AuthClient(tokenB).ListApiKeysWithResponse(ctx)

	// A has 3 keys (2 + the auth key), B has 4 (3 + the auth key).
	if len(*respA.JSON200) != 3 {
		t.Errorf("user A: expected 3 keys, got %d", len(*respA.JSON200))
	}
	if len(*respB.JSON200) != 4 {
		t.Errorf("user B: expected 4 keys, got %d", len(*respB.JSON200))
	}
}

func TestRevokeApiKey_Success(t *testing.T) {
	ctx := context.Background()
	t.Cleanup(func() { fixture.TruncateTables(t) })

	user := fixture.CreateTestUser(t, ctx, "revoker", "revoker@example.com")
	key := fixture.CreateTestApiKey(t, ctx, user.JSON201.Id)
	token := fixture.GetToken(t, ctx, key.JSON201.ClientId.String(), key.JSON201.ClientSecret)

	// Create a second key to revoke.
	target := fixture.CreateTestApiKey(t, ctx, user.JSON201.Id)

	resp, err := fixture.AuthClient(token).RevokeApiKeyWithResponse(ctx, target.JSON201.ClientId)
	if err != nil {
		t.Fatalf("RevokeApiKey: %v", err)
	}
	if resp.StatusCode() != http.StatusNoContent {
		t.Fatalf("expected 204, got %d: %s", resp.StatusCode(), string(resp.Body))
	}

	// Token exchange with the revoked key should now fail.
	tokenResp, _ := fixture.Client().CreateTokenWithResponse(ctx, apiclient.InternalAuthSpecSchemasTokenRequest{
		ClientId:     target.JSON201.ClientId,
		ClientSecret: target.JSON201.ClientSecret,
		GrantType:    apiclient.ClientCredentials,
	})
	if tokenResp.StatusCode() != http.StatusUnauthorized {
		t.Errorf("expected 401 after revoke, got %d", tokenResp.StatusCode())
	}
}

func TestRevokeApiKey_NotFound(t *testing.T) {
	ctx := context.Background()
	t.Cleanup(func() { fixture.TruncateTables(t) })

	user := fixture.CreateTestUser(t, ctx, "revokenotfound", "revokenotfound@example.com")
	key := fixture.CreateTestApiKey(t, ctx, user.JSON201.Id)
	token := fixture.GetToken(t, ctx, key.JSON201.ClientId.String(), key.JSON201.ClientSecret)

	resp, err := fixture.AuthClient(token).RevokeApiKeyWithResponse(ctx,
		openapi_types.UUID(uuid.MustParse("00000000-0000-0000-0000-000000000000")),
	)
	if err != nil {
		t.Fatalf("RevokeApiKey: %v", err)
	}
	if resp.StatusCode() != http.StatusNotFound {
		t.Fatalf("expected 404, got %d: %s", resp.StatusCode(), string(resp.Body))
	}
}

func TestRevokeApiKey_OtherUsersKey(t *testing.T) {
	ctx := context.Background()
	t.Cleanup(func() { fixture.TruncateTables(t) })

	// User A creates a key.
	userA := fixture.CreateTestUser(t, ctx, "ownerA", "ownerA@example.com")
	keyA := fixture.CreateTestApiKey(t, ctx, userA.JSON201.Id)

	// User B tries to revoke A's key.
	userB := fixture.CreateTestUser(t, ctx, "ownerB", "ownerB@example.com")
	keyB := fixture.CreateTestApiKey(t, ctx, userB.JSON201.Id)
	tokenB := fixture.GetToken(t, ctx, keyB.JSON201.ClientId.String(), keyB.JSON201.ClientSecret)

	resp, err := fixture.AuthClient(tokenB).RevokeApiKeyWithResponse(ctx, keyA.JSON201.ClientId)
	if err != nil {
		t.Fatalf("RevokeApiKey: %v", err)
	}
	if resp.StatusCode() != http.StatusNotFound {
		t.Fatalf("expected 404 when revoking another user's key, got %d: %s", resp.StatusCode(), string(resp.Body))
	}
}

func TestRevokeApiKey_Unauthenticated(t *testing.T) {
	ctx := context.Background()
	t.Cleanup(func() { fixture.TruncateTables(t) })

	user := fixture.CreateTestUser(t, ctx, "unauthrevoke", "unauthrevoke@example.com")
	key := fixture.CreateTestApiKey(t, ctx, user.JSON201.Id)

	resp, err := fixture.Client().RevokeApiKeyWithResponse(ctx, key.JSON201.ClientId)
	if err != nil {
		t.Fatalf("RevokeApiKey: %v", err)
	}
	if resp.StatusCode() != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d: %s", resp.StatusCode(), string(resp.Body))
	}
}
