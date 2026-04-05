package api

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/vitaliizinchenko/lab/model"
	"github.com/vitaliizinchenko/lab/repository"
)

type ChEventsHandler struct {
	chEventRepo repository.ChEventRepository
}

func (h *ChEventsHandler) CreateChEvent(ctx context.Context, req CreateChEventRequestObject) (CreateChEventResponseObject, error) {
	event := model.EventHistory{
		ID:        uuid.New(),
		Level:     model.EventLevel(req.Body.Level),
		EventType: req.Body.EventType,
		Details:   req.Body.Details,
		CreatedAt: time.Now().UTC(),
	}

	created, err := h.chEventRepo.Create(ctx, event)
	if err != nil {
		return nil, err
	}

	return CreateChEvent201JSONResponse(toGenEvent(created)), nil
}
