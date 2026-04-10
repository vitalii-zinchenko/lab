package handlers_test

import (
	"context"
	"net/http"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	openapi_types "github.com/oapi-codegen/runtime/types"

	"github.com/vitaliizinchenko/lab/internal/shared/apiclient"
)

func TestCreateToken_Success(t *testing.T) {
	ctx := context.Background()
	t.Cleanup(func() { fixture.TruncateTables(t) })

	user := fixture.CreateTestUser(t, ctx, "tokenuser", "tokenuser@example.com")
	key := fixture.CreateTestApiKey(t, ctx, user.JSON201.Id)

	resp, err := fixture.Client().CreateTokenWithResponse(ctx, apiclient.TokenRequest{
		ClientId:     key.JSON201.ClientId,
		ClientSecret: key.JSON201.ClientSecret,
		GrantType:    apiclient.ClientCredentials,
	})
	if err != nil {
		t.Fatalf("CreateToken: %v", err)
	}

	if resp.StatusCode() != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", resp.StatusCode(), string(resp.Body))
	}
	if resp.JSON200.TokenType != "Bearer" {
		t.Errorf("token_type: want Bearer, got %s", resp.JSON200.TokenType)
	}
	if resp.JSON200.AccessToken == "" {
		t.Error("expected non-empty access_token")
	}
	if resp.JSON200.ExpiresIn != 3600 {
		t.Errorf("expires_in: want 3600, got %d", resp.JSON200.ExpiresIn)
	}
}

func TestCreateToken_JWTClaimsValid(t *testing.T) {
	ctx := context.Background()
	t.Cleanup(func() { fixture.TruncateTables(t) })

	user := fixture.CreateTestUser(t, ctx, "claimsuser", "claimsuser@example.com")
	key := fixture.CreateTestApiKey(t, ctx, user.JSON201.Id)
	token := fixture.GetToken(t, ctx, key.JSON201.ClientId.String(), key.JSON201.ClientSecret)

	parsed, _, err := jwt.NewParser().ParseUnverified(token, &jwt.RegisteredClaims{})
	if err != nil {
		t.Fatalf("parse jwt: %v", err)
	}
	claims := parsed.Claims.(*jwt.RegisteredClaims)

	if claims.Issuer != "lab-api" {
		t.Errorf("issuer: want lab-api, got %s", claims.Issuer)
	}
	if claims.Subject == "" {
		t.Error("expected non-empty subject")
	}
	if claims.ExpiresAt == nil {
		t.Fatal("expected non-nil exp claim")
	}
	ttl := time.Until(claims.ExpiresAt.Time)
	if ttl < 59*time.Minute || ttl > 61*time.Minute {
		t.Errorf("expected ~1h TTL, got %s", ttl)
	}
}

func TestCreateToken_WrongClientID(t *testing.T) {
	ctx := context.Background()

	resp, err := fixture.Client().CreateTokenWithResponse(ctx, apiclient.TokenRequest{
		ClientId:     openapi_types.UUID(uuid.MustParse("00000000-0000-0000-0000-000000000000")),
		ClientSecret: "somesecret",
		GrantType:    apiclient.ClientCredentials,
	})
	if err != nil {
		t.Fatalf("CreateToken: %v", err)
	}

	if resp.StatusCode() != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d: %s", resp.StatusCode(), string(resp.Body))
	}
}

func TestCreateToken_WrongSecret(t *testing.T) {
	ctx := context.Background()
	t.Cleanup(func() { fixture.TruncateTables(t) })

	user := fixture.CreateTestUser(t, ctx, "wrongsecret", "wrongsecret@example.com")
	key := fixture.CreateTestApiKey(t, ctx, user.JSON201.Id)

	resp, err := fixture.Client().CreateTokenWithResponse(ctx, apiclient.TokenRequest{
		ClientId:     key.JSON201.ClientId,
		ClientSecret: "this-is-the-wrong-secret",
		GrantType:    apiclient.ClientCredentials,
	})
	if err != nil {
		t.Fatalf("CreateToken: %v", err)
	}

	if resp.StatusCode() != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d: %s", resp.StatusCode(), string(resp.Body))
	}
}

func TestCreateToken_RevokedKey(t *testing.T) {
	ctx := context.Background()
	t.Cleanup(func() { fixture.TruncateTables(t) })

	user := fixture.CreateTestUser(t, ctx, "revokeduser", "revokeduser@example.com")
	key := fixture.CreateTestApiKey(t, ctx, user.JSON201.Id)

	token := fixture.GetToken(t, ctx, key.JSON201.ClientId.String(), key.JSON201.ClientSecret)
	_, err := fixture.AuthClient(token).RevokeApiKeyWithResponse(ctx, key.JSON201.ClientId)
	if err != nil {
		t.Fatalf("RevokeApiKey: %v", err)
	}

	resp, err := fixture.Client().CreateTokenWithResponse(ctx, apiclient.TokenRequest{
		ClientId:     key.JSON201.ClientId,
		ClientSecret: key.JSON201.ClientSecret,
		GrantType:    apiclient.ClientCredentials,
	})
	if err != nil {
		t.Fatalf("CreateToken after revoke: %v", err)
	}
	if resp.StatusCode() != http.StatusUnauthorized {
		t.Fatalf("expected 401 after revoke, got %d: %s", resp.StatusCode(), string(resp.Body))
	}
}

func TestCreateToken_ExpiredKey(t *testing.T) {
	ctx := context.Background()
	t.Cleanup(func() { fixture.TruncateTables(t) })

	user := fixture.CreateTestUser(t, ctx, "expireduser", "expireduser@example.com")
	past := time.Now().UTC().Add(-time.Hour)

	keyResp, err := fixture.Client().CreateApiKeyWithResponse(ctx, apiclient.NewApiKey{
		UserId:    user.JSON201.Id,
		ExpiresAt: &past,
	})
	if err != nil || keyResp.StatusCode() != http.StatusCreated {
		t.Fatalf("CreateApiKey: status=%d err=%v", keyResp.StatusCode(), err)
	}

	resp, err := fixture.Client().CreateTokenWithResponse(ctx, apiclient.TokenRequest{
		ClientId:     keyResp.JSON201.ClientId,
		ClientSecret: keyResp.JSON201.ClientSecret,
		GrantType:    apiclient.ClientCredentials,
	})
	if err != nil {
		t.Fatalf("CreateToken: %v", err)
	}
	if resp.StatusCode() != http.StatusUnauthorized {
		t.Fatalf("expected 401 for expired key, got %d: %s", resp.StatusCode(), string(resp.Body))
	}
}

func TestCreateToken_UpdatesLastUsedAt(t *testing.T) {
	ctx := context.Background()
	t.Cleanup(func() { fixture.TruncateTables(t) })

	user := fixture.CreateTestUser(t, ctx, "lastusedat", "lastusedat@example.com")
	key := fixture.CreateTestApiKey(t, ctx, user.JSON201.Id)

	fixture.GetToken(t, ctx, key.JSON201.ClientId.String(), key.JSON201.ClientSecret)

	// last_used_at is updated asynchronously in the handler; poll until it appears.
	fixture.WaitForLastUsedAt(t, ctx, key.JSON201.ClientId.String(), time.Second)
}
