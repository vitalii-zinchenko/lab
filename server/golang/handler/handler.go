package handler

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	openapi_types "github.com/oapi-codegen/runtime/types"
	"github.com/vitaliizinchenko/lab/gen"
	"github.com/vitaliizinchenko/lab/repository"
)

// Handler implements gen.StrictServerInterface.
type Handler struct {
	items repository.ItemRepository
}

func New(items repository.ItemRepository) *Handler {
	return &Handler{items: items}
}

// toGenItem converts a repository.Item to the API response type gen.Item.
// openapi_types.UUID and uuid.UUID are both [16]byte — the cast is zero-cost.
func toGenItem(r repository.Item) gen.Item {
	return gen.Item{
		Id:          openapi_types.UUID(r.ID),
		Name:        r.Name,
		Description: r.Description,
		CreatedAt:   r.CreatedAt,
	}
}

func (h *Handler) GetHealth(_ context.Context, _ gen.GetHealthRequestObject) (gen.GetHealthResponseObject, error) {
	return gen.GetHealth200JSONResponse{Status: "ok"}, nil
}

func (h *Handler) ListItems(ctx context.Context, req gen.ListItemsRequestObject) (gen.ListItemsResponseObject, error) {
	limit := 20
	if req.Params.Limit != nil {
		limit = *req.Params.Limit
	}

	repoItems, err := h.items.List(ctx, limit)
	if err != nil {
		return nil, err
	}

	result := make([]gen.Item, len(repoItems))
	for i, item := range repoItems {
		result[i] = toGenItem(item)
	}

	return gen.ListItems200JSONResponse(result), nil
}

func (h *Handler) CreateItem(ctx context.Context, req gen.CreateItemRequestObject) (gen.CreateItemResponseObject, error) {
	newItem := repository.Item{
		ID:          uuid.New(),
		Name:        req.Body.Name,
		Description: req.Body.Description,
		CreatedAt:   time.Now().UTC(),
	}

	created, err := h.items.Create(ctx, newItem)
	if err != nil {
		return nil, err
	}

	return gen.CreateItem201JSONResponse(toGenItem(created)), nil
}

func (h *Handler) GetItem(ctx context.Context, req gen.GetItemRequestObject) (gen.GetItemResponseObject, error) {
	item, err := h.items.GetByID(ctx, uuid.UUID(req.Id))
	if errors.Is(err, repository.ErrNotFound) {
		return gen.GetItem404JSONResponse{Message: "item not found"}, nil
	}
	if err != nil {
		return nil, err
	}

	return gen.GetItem200JSONResponse(toGenItem(item)), nil
}

func (h *Handler) DeleteItem(ctx context.Context, req gen.DeleteItemRequestObject) (gen.DeleteItemResponseObject, error) {
	err := h.items.Delete(ctx, uuid.UUID(req.Id))
	if errors.Is(err, repository.ErrNotFound) {
		return gen.DeleteItem404JSONResponse{Message: "item not found"}, nil
	}
	if err != nil {
		return nil, err
	}

	return gen.DeleteItem204Response{}, nil
}
