package handler

import (
	"context"
	"sync"
	"time"

	"github.com/google/uuid"
	openapi_types "github.com/oapi-codegen/runtime/types"
	"github.com/vitaliizinchenko/lab/gen"
)

// Handler implements gen.StrictServerInterface.
// Currently uses in-memory storage — replace with a real DB later.
type Handler struct {
	mu    sync.RWMutex
	items map[openapi_types.UUID]gen.Item
}

func New() *Handler {
	return &Handler{
		items: make(map[openapi_types.UUID]gen.Item),
	}
}

func (h *Handler) GetHealth(_ context.Context, _ gen.GetHealthRequestObject) (gen.GetHealthResponseObject, error) {
	return gen.GetHealth200JSONResponse{Status: "ok"}, nil
}

func (h *Handler) ListItems(_ context.Context, req gen.ListItemsRequestObject) (gen.ListItemsResponseObject, error) {
	h.mu.RLock()
	defer h.mu.RUnlock()

	limit := 20
	if req.Params.Limit != nil {
		limit = *req.Params.Limit
	}

	result := make([]gen.Item, 0, len(h.items))
	for _, item := range h.items {
		result = append(result, item)
		if len(result) >= limit {
			break
		}
	}

	return gen.ListItems200JSONResponse(result), nil
}

func (h *Handler) CreateItem(_ context.Context, req gen.CreateItemRequestObject) (gen.CreateItemResponseObject, error) {
	h.mu.Lock()
	defer h.mu.Unlock()

	id := openapi_types.UUID(uuid.New())
	item := gen.Item{
		Id:          id,
		Name:        req.Body.Name,
		Description: req.Body.Description,
		CreatedAt:   time.Now().UTC(),
	}
	h.items[id] = item

	return gen.CreateItem201JSONResponse(item), nil
}

func (h *Handler) GetItem(_ context.Context, req gen.GetItemRequestObject) (gen.GetItemResponseObject, error) {
	h.mu.RLock()
	defer h.mu.RUnlock()

	item, ok := h.items[req.Id]
	if !ok {
		return gen.GetItem404JSONResponse{Message: "item not found"}, nil
	}

	return gen.GetItem200JSONResponse(item), nil
}

func (h *Handler) DeleteItem(_ context.Context, req gen.DeleteItemRequestObject) (gen.DeleteItemResponseObject, error) {
	h.mu.Lock()
	defer h.mu.Unlock()

	if _, ok := h.items[req.Id]; !ok {
		return gen.DeleteItem404JSONResponse{Message: "item not found"}, nil
	}

	delete(h.items, req.Id)

	return gen.DeleteItem204Response{}, nil
}
