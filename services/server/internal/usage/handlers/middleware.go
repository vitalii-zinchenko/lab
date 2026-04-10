package handlers

import (
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
				if recordErr := repo.Record(ctx.Request.Context(), models.Usage{
					Timestamp: time.Now().UTC(),
					UserID:    userID,
					Operation: operationID,
				}); recordErr != nil {
					return nil, recordErr
				}
			}

			return response, err
		}
	}
}
