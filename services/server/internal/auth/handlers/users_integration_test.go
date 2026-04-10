package handlers_test

import (
	"context"
	"net/http"
	"strings"
	"testing"
)

func TestCreateUser_Success(t *testing.T) {
	ctx := context.Background()
	t.Cleanup(func() { fixture.TruncateTables(t) })

	resp := fixture.CreateTestUser(t, ctx, "alice", "alice@example.com")

	if resp.StatusCode() != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", resp.StatusCode(), string(resp.Body))
	}
	if resp.JSON201 == nil {
		t.Fatal("expected JSON body in 201 response")
	}
	if resp.JSON201.Username != "alice" {
		t.Errorf("username: want alice, got %s", resp.JSON201.Username)
	}
	if resp.JSON201.Email != "alice@example.com" {
		t.Errorf("email: want alice@example.com, got %s", resp.JSON201.Email)
	}
	if resp.JSON201.Id == 0 {
		t.Error("expected non-zero ID")
	}
	if resp.JSON201.CreatedAt.IsZero() {
		t.Error("expected non-zero created_at")
	}
}

func TestCreateUser_DuplicateUsername(t *testing.T) {
	ctx := context.Background()
	t.Cleanup(func() { fixture.TruncateTables(t) })

	fixture.CreateTestUser(t, ctx, "bob", "bob@example.com")

	resp := fixture.CreateTestUser(t, ctx, "bob", "otherbob@example.com")

	if resp.StatusCode() != http.StatusConflict {
		t.Fatalf("expected 409, got %d: %s", resp.StatusCode(), string(resp.Body))
	}
}

func TestCreateUser_DuplicateEmail(t *testing.T) {
	ctx := context.Background()
	t.Cleanup(func() { fixture.TruncateTables(t) })

	fixture.CreateTestUser(t, ctx, "carol", "carol@example.com")

	resp := fixture.CreateTestUser(t, ctx, "carol2", "carol@example.com")

	if resp.StatusCode() != http.StatusConflict {
		t.Fatalf("expected 409, got %d: %s", resp.StatusCode(), string(resp.Body))
	}
}

func TestCreateUser_InvalidEmail(t *testing.T) {
	ctx := context.Background()

	// Use a raw request to bypass client-side email validation in openapi_types.Email.
	// The OpenAPI validator on the server should reject the malformed email with 400.
	body := strings.NewReader(`{"username":"dave","email":"not-an-email"}`)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, fixture.Server.URL+"/users", body)
	if err != nil {
		t.Fatalf("NewRequest: %v", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("Do: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", resp.StatusCode)
	}
}

func TestCreateUser_EmptyUsername(t *testing.T) {
	ctx := context.Background()

	resp := fixture.CreateTestUser(t, ctx, "", "empty@example.com")

	if resp.StatusCode() != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d: %s", resp.StatusCode(), string(resp.Body))
	}
}
