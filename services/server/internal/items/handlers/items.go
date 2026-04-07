package handlers

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	openapi_types "github.com/oapi-codegen/runtime/types"

	"github.com/vitaliizinchenko/lab/internal/shared"
	"github.com/vitaliizinchenko/lab/internal/items/models"
	"github.com/vitaliizinchenko/lab/internal/items/repos"
)

// ItemsHandler handles item CRUD endpoints.
type ItemsHandler struct {
	itemRepo repos.ItemRepository
}

// NewItemsHandler creates an ItemsHandler with the given repository.
func NewItemsHandler(itemRepo repos.ItemRepository) *ItemsHandler {
	return &ItemsHandler{itemRepo: itemRepo}
}

func toGenItem(r models.Item) shared.Item {
	return shared.Item{
		Id:          openapi_types.UUID(r.ID),
		Name:        r.Name,
		Description: r.Description,
		CreatedAt:   r.CreatedAt,
	}
}

func (h *ItemsHandler) ListItems(ctx context.Context, req shared.ListItemsRequestObject) (shared.ListItemsResponseObject, error) {
	limit := 20
	if req.Params.Limit != nil {
		limit = *req.Params.Limit
	}

	repoItems, err := h.itemRepo.List(ctx, limit)
	if err != nil {
		return nil, err
	}

	result := make([]shared.Item, len(repoItems))
	for i, item := range repoItems {
		result[i] = toGenItem(item)
	}
	return shared.ListItems200JSONResponse(result), nil
}

func (h *ItemsHandler) CreateItem(ctx context.Context, req shared.CreateItemRequestObject) (shared.CreateItemResponseObject, error) {
	newItem := models.Item{
		ID:          uuid.New(),
		Name:        req.Body.Name,
		Description: req.Body.Description,
		CreatedAt:   time.Now().UTC(),
	}

	created, err := h.itemRepo.Create(ctx, newItem)
	if err != nil {
		return nil, err
	}

	return shared.CreateItem201JSONResponse(toGenItem(created)), nil
}

func (h *ItemsHandler) GetItem(ctx context.Context, req shared.GetItemRequestObject) (shared.GetItemResponseObject, error) {
	item, err := h.itemRepo.GetByID(ctx, uuid.UUID(req.Id))
	if errors.Is(err, shared.ErrNotFound) {
		return shared.GetItem404JSONResponse{Message: "item not found"}, nil
	}
	if err != nil {
		return nil, err
	}
	return shared.GetItem200JSONResponse(toGenItem(item)), nil
}

func (h *ItemsHandler) DeleteItem(ctx context.Context, req shared.DeleteItemRequestObject) (shared.DeleteItemResponseObject, error) {
	err := h.itemRepo.Delete(ctx, uuid.UUID(req.Id))
	if errors.Is(err, shared.ErrNotFound) {
		return shared.DeleteItem404JSONResponse{Message: "item not found"}, nil
	}
	if err != nil {
		return nil, err
	}
	return shared.DeleteItem204Response{}, nil
}
