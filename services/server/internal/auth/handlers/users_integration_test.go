package handlers_test

import (
	"context"
	"net/http"
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

	resp := fixture.CreateTestUser(t, ctx, "dave", "not-an-email")

	// OpenAPI validator rejects malformed email before hitting the handler.
	if resp.StatusCode() != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d: %s", resp.StatusCode(), string(resp.Body))
	}
}

func TestCreateUser_EmptyUsername(t *testing.T) {
	ctx := context.Background()

	resp := fixture.CreateTestUser(t, ctx, "", "empty@example.com")

	if resp.StatusCode() != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d: %s", resp.StatusCode(), string(resp.Body))
	}
}
