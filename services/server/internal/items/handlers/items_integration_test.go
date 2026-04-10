package handlers_test

import (
	"context"
	"net/http"
	"strings"
	"testing"

	"github.com/google/uuid"
	openapi_types "github.com/oapi-codegen/runtime/types"

	"github.com/vitaliizinchenko/lab/internal/shared/apiclient"
)

func TestCreateItem_Success(t *testing.T) {
	ctx := context.Background()
	t.Cleanup(func() { fixture.TruncateTables(t) })

	resp, err := fixture.Client().CreateItemWithResponse(ctx, apiclient.NewItem{
		Name: "widget",
	})
	if err != nil {
		t.Fatalf("CreateItem: %v", err)
	}

	if resp.StatusCode() != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", resp.StatusCode(), string(resp.Body))
	}
	if resp.JSON201 == nil {
		t.Fatal("expected JSON body")
	}
	if resp.JSON201.Name != "widget" {
		t.Errorf("name: want widget, got %s", resp.JSON201.Name)
	}
	if resp.JSON201.Id == (openapi_types.UUID{}) {
		t.Error("expected non-zero id")
	}
	if resp.JSON201.CreatedAt.IsZero() {
		t.Error("expected non-zero created_at")
	}
	if resp.JSON201.Description != nil {
		t.Errorf("expected nil description, got %v", *resp.JSON201.Description)
	}
}

func TestCreateItem_WithDescription(t *testing.T) {
	ctx := context.Background()
	t.Cleanup(func() { fixture.TruncateTables(t) })

	desc := "a very fine widget"
	resp, err := fixture.Client().CreateItemWithResponse(ctx, apiclient.NewItem{
		Name:        "widget with desc",
		Description: &desc,
	})
	if err != nil {
		t.Fatalf("CreateItem: %v", err)
	}
	if resp.StatusCode() != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", resp.StatusCode(), string(resp.Body))
	}
	if resp.JSON201.Description == nil || *resp.JSON201.Description != desc {
		t.Errorf("description: want %q, got %v", desc, resp.JSON201.Description)
	}
}

func TestCreateItem_MissingName(t *testing.T) {
	ctx := context.Background()

	body := `{"description":"no name here"}`
	resp, err := fixture.Client().CreateItemWithBodyWithResponse(ctx, "application/json",
		strings.NewReader(body))
	if err != nil {
		t.Fatalf("CreateItem: %v", err)
	}
	if resp.StatusCode() != http.StatusBadRequest {
		t.Fatalf("expected 400 for missing name, got %d: %s", resp.StatusCode(), string(resp.Body))
	}
}

func TestListItems_Empty(t *testing.T) {
	ctx := context.Background()
	t.Cleanup(func() { fixture.TruncateTables(t) })

	resp, err := fixture.Client().ListItemsWithResponse(ctx, &apiclient.ListItemsParams{})
	if err != nil {
		t.Fatalf("ListItems: %v", err)
	}
	if resp.StatusCode() != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", resp.StatusCode(), string(resp.Body))
	}
	if len(*resp.JSON200) != 0 {
		t.Errorf("expected empty list, got %d items", len(*resp.JSON200))
	}
}

func TestListItems_WithItems(t *testing.T) {
	ctx := context.Background()
	t.Cleanup(func() { fixture.TruncateTables(t) })

	for _, name := range []string{"alpha", "beta", "gamma"} {
		_, err := fixture.Client().CreateItemWithResponse(ctx, apiclient.NewItem{Name: name})
		if err != nil {
			t.Fatalf("CreateItem %s: %v", name, err)
		}
	}

	resp, err := fixture.Client().ListItemsWithResponse(ctx, &apiclient.ListItemsParams{})
	if err != nil {
		t.Fatalf("ListItems: %v", err)
	}
	if resp.StatusCode() != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", resp.StatusCode(), string(resp.Body))
	}
	if len(*resp.JSON200) != 3 {
		t.Errorf("expected 3 items, got %d", len(*resp.JSON200))
	}
}

func TestListItems_Limit(t *testing.T) {
	ctx := context.Background()
	t.Cleanup(func() { fixture.TruncateTables(t) })

	for _, name := range []string{"one", "two", "three"} {
		_, err := fixture.Client().CreateItemWithResponse(ctx, apiclient.NewItem{Name: name})
		if err != nil {
			t.Fatalf("CreateItem %s: %v", name, err)
		}
	}

	limit := 2
	resp, err := fixture.Client().ListItemsWithResponse(ctx, &apiclient.ListItemsParams{Limit: &limit})
	if err != nil {
		t.Fatalf("ListItems: %v", err)
	}
	if resp.StatusCode() != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", resp.StatusCode(), string(resp.Body))
	}
	if len(*resp.JSON200) != 2 {
		t.Errorf("expected 2 items with limit=2, got %d", len(*resp.JSON200))
	}
}

func TestGetItem_Success(t *testing.T) {
	ctx := context.Background()
	t.Cleanup(func() { fixture.TruncateTables(t) })

	created, err := fixture.Client().CreateItemWithResponse(ctx, apiclient.NewItem{Name: "findme"})
	if err != nil || created.StatusCode() != http.StatusCreated {
		t.Fatalf("CreateItem: status=%d err=%v", created.StatusCode(), err)
	}

	resp, err := fixture.Client().GetItemWithResponse(ctx, created.JSON201.Id)
	if err != nil {
		t.Fatalf("GetItem: %v", err)
	}
	if resp.StatusCode() != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", resp.StatusCode(), string(resp.Body))
	}
	if resp.JSON200.Name != "findme" {
		t.Errorf("name: want findme, got %s", resp.JSON200.Name)
	}
	if resp.JSON200.Id != created.JSON201.Id {
		t.Errorf("id mismatch: want %s, got %s", created.JSON201.Id, resp.JSON200.Id)
	}
}

func TestGetItem_NotFound(t *testing.T) {
	ctx := context.Background()

	resp, err := fixture.Client().GetItemWithResponse(ctx,
		openapi_types.UUID(uuid.MustParse("00000000-0000-0000-0000-000000000000")))
	if err != nil {
		t.Fatalf("GetItem: %v", err)
	}
	if resp.StatusCode() != http.StatusNotFound {
		t.Fatalf("expected 404, got %d: %s", resp.StatusCode(), string(resp.Body))
	}
}

func TestDeleteItem_Success(t *testing.T) {
	ctx := context.Background()
	t.Cleanup(func() { fixture.TruncateTables(t) })

	created, err := fixture.Client().CreateItemWithResponse(ctx, apiclient.NewItem{Name: "deleteme"})
	if err != nil || created.StatusCode() != http.StatusCreated {
		t.Fatalf("CreateItem: status=%d err=%v", created.StatusCode(), err)
	}

	delResp, err := fixture.Client().DeleteItemWithResponse(ctx, created.JSON201.Id)
	if err != nil {
		t.Fatalf("DeleteItem: %v", err)
	}
	if delResp.StatusCode() != http.StatusNoContent {
		t.Fatalf("expected 204, got %d: %s", delResp.StatusCode(), string(delResp.Body))
	}

	// Verify it's gone.
	getResp, err := fixture.Client().GetItemWithResponse(ctx, created.JSON201.Id)
	if err != nil {
		t.Fatalf("GetItem after delete: %v", err)
	}
	if getResp.StatusCode() != http.StatusNotFound {
		t.Errorf("expected 404 after delete, got %d", getResp.StatusCode())
	}
}

func TestDeleteItem_NotFound(t *testing.T) {
	ctx := context.Background()

	resp, err := fixture.Client().DeleteItemWithResponse(ctx,
		openapi_types.UUID(uuid.MustParse("00000000-0000-0000-0000-000000000000")))
	if err != nil {
		t.Fatalf("DeleteItem: %v", err)
	}
	if resp.StatusCode() != http.StatusNotFound {
		t.Fatalf("expected 404, got %d: %s", resp.StatusCode(), string(resp.Body))
	}
}

func TestDeleteItem_AlreadyDeleted(t *testing.T) {
	ctx := context.Background()
	t.Cleanup(func() { fixture.TruncateTables(t) })

	created, err := fixture.Client().CreateItemWithResponse(ctx, apiclient.NewItem{Name: "delete-twice"})
	if err != nil || created.StatusCode() != http.StatusCreated {
		t.Fatalf("CreateItem: status=%d err=%v", created.StatusCode(), err)
	}

	fixture.Client().DeleteItemWithResponse(ctx, created.JSON201.Id) //nolint:errcheck

	resp, err := fixture.Client().DeleteItemWithResponse(ctx, created.JSON201.Id)
	if err != nil {
		t.Fatalf("DeleteItem second call: %v", err)
	}
	if resp.StatusCode() != http.StatusNotFound {
		t.Fatalf("expected 404 on second delete, got %d: %s", resp.StatusCode(), string(resp.Body))
	}
}
