package api

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	openapi_types "github.com/oapi-codegen/runtime/types"
	"github.com/vitaliizinchenko/lab/model"
	"github.com/vitaliizinchenko/lab/repository"
)

type ItemsHandler struct {
	itemRepo repository.ItemRepository
}

func toGenItem(r model.Item) Item {
	return Item{
		Id:          openapi_types.UUID(r.ID),
		Name:        r.Name,
		Description: r.Description,
		CreatedAt:   r.CreatedAt,
	}
}

func (h *ItemsHandler) ListItems(ctx context.Context, req ListItemsRequestObject) (ListItemsResponseObject, error) {
	limit := 20
	if req.Params.Limit != nil {
		limit = *req.Params.Limit
	}

	repoItems, err := h.itemRepo.List(ctx, limit)
	if err != nil {
		return nil, err
	}

	result := make([]Item, len(repoItems))
	for i, item := range repoItems {
		result[i] = toGenItem(item)
	}

	return ListItems200JSONResponse(result), nil
}

func (h *ItemsHandler) CreateItem(ctx context.Context, req CreateItemRequestObject) (CreateItemResponseObject, error) {
	newItem := model.Item{
		ID:          uuid.New(),
		Name:        req.Body.Name,
		Description: req.Body.Description,
		CreatedAt:   time.Now().UTC(),
	}

	created, err := h.itemRepo.Create(ctx, newItem)
	if err != nil {
		return nil, err
	}

	return CreateItem201JSONResponse(toGenItem(created)), nil
}

func (h *ItemsHandler) GetItem(ctx context.Context, req GetItemRequestObject) (GetItemResponseObject, error) {
	item, err := h.itemRepo.GetByID(ctx, uuid.UUID(req.Id))
	if errors.Is(err, repository.ErrNotFound) {
		return GetItem404JSONResponse{Message: "item not found"}, nil
	}
	if err != nil {
		return nil, err
	}

	return GetItem200JSONResponse(toGenItem(item)), nil
}

func (h *ItemsHandler) DeleteItem(ctx context.Context, req DeleteItemRequestObject) (DeleteItemResponseObject, error) {
	err := h.itemRepo.Delete(ctx, uuid.UUID(req.Id))
	if errors.Is(err, repository.ErrNotFound) {
		return DeleteItem404JSONResponse{Message: "item not found"}, nil
	}
	if err != nil {
		return nil, err
	}

	return DeleteItem204Response{}, nil
}
