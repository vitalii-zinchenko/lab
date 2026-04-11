package handlers

import (
	"context"

	openapi_types "github.com/oapi-codegen/runtime/types"

	authhandlers "github.com/vitaliizinchenko/lab/internal/auth/handlers"
	"github.com/vitaliizinchenko/lab/internal/shared"
	"github.com/vitaliizinchenko/lab/internal/usage/models"
	"github.com/vitaliizinchenko/lab/internal/usage/repos"
)

// UsageHandler handles usage statistics endpoints.
type UsageHandler struct {
	repo repos.UsageRepository
}

// NewUsageHandler creates a UsageHandler with the given repository.
func NewUsageHandler(repo repos.UsageRepository) *UsageHandler {
	return &UsageHandler{repo: repo}
}

func (h *UsageHandler) CreateUsage(ctx context.Context, req shared.CreateUsageRequestObject) (shared.CreateUsageResponseObject, error) {
	entry := &models.Usage{
		UserID:    req.Body.UserId,
		Timestamp: req.Body.Timestamp,
		Operation: req.Body.Operation,
	}
	if err := h.repo.Record(ctx, entry); err != nil {
		return nil, err
	}
	return shared.CreateUsage201JSONResponse{
		Id:        entry.ID,
		UserId:    entry.UserID,
		Timestamp: entry.Timestamp,
		Operation: entry.Operation,
	}, nil
}

func (h *UsageHandler) GetUsage(ctx context.Context, req shared.GetUsageRequestObject) (shared.GetUsageResponseObject, error) {
	userID, ok := authhandlers.GetUserID(ctx)
	if !ok {
		return shared.GetUsage401JSONResponse{Message: "unauthorized"}, nil
	}

	stats, err := h.repo.GetDailyStats(ctx, userID, req.Params.From, req.Params.To)
	if err != nil {
		return nil, err
	}

	data := make([]shared.DailyUsage, len(stats))
	for i, s := range stats {
		data[i] = shared.DailyUsage{
			Date:  openapi_types.Date{Time: s.Date},
			Count: int(s.Count),
		}
	}

	return shared.GetUsage200JSONResponse{Data: data}, nil
}
