package handlers

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"time"

	openapi_types "github.com/oapi-codegen/runtime/types"

	authhandlers "github.com/vitaliizinchenko/lab/internal/auth/handlers"
	"github.com/vitaliizinchenko/lab/internal/shared"
	"github.com/vitaliizinchenko/lab/internal/usage/models"
	"github.com/vitaliizinchenko/lab/internal/usage/repos"
)

type usageCursor struct {
	Timestamp time.Time `json:"ts"`
	ID        int64     `json:"id"`
}

func encodeCursor(ts time.Time, id int64) string {
	b, _ := json.Marshal(usageCursor{Timestamp: ts, ID: id})
	return base64.StdEncoding.EncodeToString(b)
}

func decodeCursor(s string) (ts time.Time, id int64, ok bool) {
	b, err := base64.StdEncoding.DecodeString(s)
	if err != nil {
		return
	}
	var c usageCursor
	if err := json.Unmarshal(b, &c); err != nil {
		return
	}
	return c.Timestamp, c.ID, true
}

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

	limit := 100
	if req.Params.Limit != nil {
		limit = *req.Params.Limit
	}

	var afterTimestamp *time.Time
	var afterID int64
	if req.Params.Cursor != nil && *req.Params.Cursor != "" {
		if ts, id, ok := decodeCursor(*req.Params.Cursor); ok {
			afterTimestamp = &ts
			afterID = id
		}
	}

	records, err := h.repo.ListRecords(ctx, userID, req.Params.From, req.Params.To, afterTimestamp, afterID, limit)
	if err != nil {
		return nil, err
	}

	data := make([]shared.UsageItem, len(records))
	for i, r := range records {
		data[i] = shared.UsageItem{
			Id:        r.ID,
			UserId:    r.UserID,
			Timestamp: r.Timestamp,
			Operation: r.Operation,
		}
	}

	var nextCursor *string
	if len(records) == limit {
		last := records[len(records)-1]
		s := encodeCursor(last.Timestamp, last.ID)
		nextCursor = &s
	}

	return shared.GetUsage200JSONResponse{
		Data:       data,
		NextCursor: nextCursor,
	}, nil
}

func (h *UsageHandler) GetUsageStats(ctx context.Context, req shared.GetUsageStatsRequestObject) (shared.GetUsageStatsResponseObject, error) {
	userID, ok := authhandlers.GetUserID(ctx)
	if !ok {
		return shared.GetUsageStats401JSONResponse{Message: "unauthorized"}, nil
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

	return shared.GetUsageStats200JSONResponse{Data: data}, nil
}
