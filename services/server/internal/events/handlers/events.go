package handlers

import (
	"context"
	"time"

	"github.com/google/uuid"
	openapi_types "github.com/oapi-codegen/runtime/types"

	"github.com/vitaliizinchenko/lab/internal/shared"
	"github.com/vitaliizinchenko/lab/internal/events/models"
	"github.com/vitaliizinchenko/lab/internal/events/repos"
)

// EventsHandler handles Postgres event_history endpoints.
type EventsHandler struct {
	eventRepo repos.EventHistoryRepository
}

// NewEventsHandler creates an EventsHandler with the given repository.
func NewEventsHandler(eventRepo repos.EventHistoryRepository) *EventsHandler {
	return &EventsHandler{eventRepo: eventRepo}
}

func toGenEvent(e models.EventHistory) shared.EventHistory {
	return shared.EventHistory{
		Id:        openapi_types.UUID(e.ID),
		Level:     shared.EventLevel(e.Level),
		EventType: e.EventType,
		Details:   e.Details,
		CreatedAt: e.CreatedAt,
	}
}

func (h *EventsHandler) CreateEvent(ctx context.Context, req shared.CreateEventRequestObject) (shared.CreateEventResponseObject, error) {
	event := models.EventHistory{
		ID:        uuid.New(),
		Level:     models.EventLevel(req.Body.Level),
		EventType: req.Body.EventType,
		Details:   req.Body.Details,
		CreatedAt: time.Now().UTC(),
	}

	created, err := h.eventRepo.Create(ctx, event)
	if err != nil {
		return nil, err
	}

	return shared.CreateEvent201JSONResponse(toGenEvent(created)), nil
}
