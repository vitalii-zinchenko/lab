package handlers

import (
	"context"
	"time"

	"github.com/gin-gonic/gin"

	authhandlers "github.com/vitaliizinchenko/lab/internal/auth/handlers"
	"github.com/vitaliizinchenko/lab/internal/shared"
	"github.com/vitaliizinchenko/lab/internal/usage/models"
	"github.com/vitaliizinchenko/lab/internal/usage/repos"
)

// UsageTrackingMiddleware returns a StrictMiddlewareFunc that records one row per
// authenticated request. It runs after the handler and does not add latency.
func UsageTrackingMiddleware(repo repos.UsageRepository) shared.StrictMiddlewareFunc {
	return func(f shared.StrictHandlerFunc, operationID string) shared.StrictHandlerFunc {
		return func(ctx *gin.Context, request interface{}) (interface{}, error) {
			response, err := f(ctx, request)

			userID, ok := authhandlers.GetUserID(ctx.Request.Context())
			if ok {
				go func() {
					_ = repo.Record(context.Background(), models.ApiUsage{
						Timestamp: time.Now().UTC(),
						UserID:    userID,
						Operation: operationID,
					})
				}()
			}

			return response, err
		}
	}
}
