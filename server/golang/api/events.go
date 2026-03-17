package api

import (
	"context"
	"time"

	"github.com/google/uuid"
	openapi_types "github.com/oapi-codegen/runtime/types"
	"github.com/vitaliizinchenko/lab/model"
	"github.com/vitaliizinchenko/lab/repository"
)

type EventsHandler struct {
	eventRepo repository.EventHistoryRepository
}

func toGenEvent(e model.EventHistory) EventHistory {
	return EventHistory{
		Id:        openapi_types.UUID(e.ID),
		Level:     EventLevel(e.Level),
		EventType: e.EventType,
		Details:   e.Details,
		CreatedAt: e.CreatedAt,
	}
}

func (h *EventsHandler) CreateEvent(ctx context.Context, req CreateEventRequestObject) (CreateEventResponseObject, error) {
	event := model.EventHistory{
		ID:        uuid.New(),
		Level:     model.EventLevel(req.Body.Level),
		EventType: req.Body.EventType,
		Details:   req.Body.Details,
		CreatedAt: time.Now().UTC(),
	}

	created, err := h.eventRepo.Create(ctx, event)
	if err != nil {
		return nil, err
	}

	return CreateEvent201JSONResponse(toGenEvent(created)), nil
}
