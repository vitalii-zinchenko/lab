package handlers

import (
	"context"
	"time"

	"github.com/google/uuid"

	"github.com/vitaliizinchenko/lab/internal/shared"
	"github.com/vitaliizinchenko/lab/internal/events/models"
	"github.com/vitaliizinchenko/lab/internal/events/repos"
)

// ChEventsHandler handles ClickHouse event endpoints.
type ChEventsHandler struct {
	chEventRepo repos.ChEventRepository
}

// NewChEventsHandler creates a ChEventsHandler with the given repository.
func NewChEventsHandler(chEventRepo repos.ChEventRepository) *ChEventsHandler {
	return &ChEventsHandler{chEventRepo: chEventRepo}
}

func (h *ChEventsHandler) CreateChEvent(ctx context.Context, req shared.CreateChEventRequestObject) (shared.CreateChEventResponseObject, error) {
	event := models.EventHistory{
		ID:        uuid.New(),
		Level:     models.EventLevel(req.Body.Level),
		EventType: req.Body.EventType,
		Details:   req.Body.Details,
		CreatedAt: time.Now().UTC(),
	}

	created, err := h.chEventRepo.Create(ctx, event)
	if err != nil {
		return nil, err
	}

	return shared.CreateChEvent201JSONResponse(toGenEvent(created)), nil
}
