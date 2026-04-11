package handlers_test

import (
	"context"
	"net/http"
	"testing"
	"time"

	"github.com/vitaliizinchenko/lab/internal/shared/apiclient"
)

func TestCreateUsage_NoAuthRequired(t *testing.T) {
	ctx := context.Background()
	t.Cleanup(func() { fixture.TruncateTables(t) })

	now := time.Now().UTC()
	resp, err := fixture.Client().CreateUsageWithResponse(ctx, apiclient.NewUsage{
		UserId:    42,
		Timestamp: now,
		Operation: "test.op",
	})
	if err != nil {
		t.Fatalf("CreateUsage: %v", err)
	}
	// Should not be 401 — no auth required
	if resp.StatusCode() == http.StatusUnauthorized {
		t.Fatal("expected no auth requirement, got 401")
	}
	if resp.StatusCode() != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", resp.StatusCode(), string(resp.Body))
	}
}

func TestCreateUsage_ReturnsCreatedItem(t *testing.T) {
	ctx := context.Background()
	t.Cleanup(func() { fixture.TruncateTables(t) })

	now := time.Now().UTC().Truncate(time.Millisecond)
	resp, err := fixture.Client().CreateUsageWithResponse(ctx, apiclient.NewUsage{
		UserId:    99,
		Timestamp: now,
		Operation: "items.create",
	})
	if err != nil {
		t.Fatalf("CreateUsage: %v", err)
	}
	if resp.StatusCode() != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", resp.StatusCode(), string(resp.Body))
	}
	if resp.JSON201 == nil {
		t.Fatal("expected JSON body, got nil")
	}

	item := resp.JSON201
	if item.Id <= 0 {
		t.Errorf("expected positive id, got %d", item.Id)
	}
	if item.UserId != 99 {
		t.Errorf("expected user_id=99, got %d", item.UserId)
	}
	if item.Operation != "items.create" {
		t.Errorf("expected operation=items.create, got %q", item.Operation)
	}
	if !item.Timestamp.Equal(now) {
		t.Errorf("expected timestamp=%v, got %v", now, item.Timestamp)
	}
}

func TestCreateUsage_DistinctIdsPerRecord(t *testing.T) {
	ctx := context.Background()
	t.Cleanup(func() { fixture.TruncateTables(t) })

	now := time.Now().UTC()
	first, err := fixture.Client().CreateUsageWithResponse(ctx, apiclient.NewUsage{
		UserId:    1,
		Timestamp: now,
		Operation: "op.a",
	})
	if err != nil || first.StatusCode() != http.StatusCreated {
		t.Fatalf("first CreateUsage failed: %v / %d", err, first.StatusCode())
	}

	second, err := fixture.Client().CreateUsageWithResponse(ctx, apiclient.NewUsage{
		UserId:    1,
		Timestamp: now,
		Operation: "op.b",
	})
	if err != nil || second.StatusCode() != http.StatusCreated {
		t.Fatalf("second CreateUsage failed: %v / %d", err, second.StatusCode())
	}

	if first.JSON201.Id == second.JSON201.Id {
		t.Errorf("expected distinct IDs, both got %d", first.JSON201.Id)
	}
}

func TestCreateUsage_FreeFormOperation(t *testing.T) {
	ctx := context.Background()
	t.Cleanup(func() { fixture.TruncateTables(t) })

	op := "some/arbitrary::operation-string_v2"
	resp, err := fixture.Client().CreateUsageWithResponse(ctx, apiclient.NewUsage{
		UserId:    7,
		Timestamp: time.Now().UTC(),
		Operation: op,
	})
	if err != nil {
		t.Fatalf("CreateUsage: %v", err)
	}
	if resp.StatusCode() != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", resp.StatusCode(), string(resp.Body))
	}
	if resp.JSON201.Operation != op {
		t.Errorf("expected operation=%q, got %q", op, resp.JSON201.Operation)
	}
}
