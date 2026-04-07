package main

import (
	"context"

	"github.com/vitaliizinchenko/lab/internal/shared"
)

type HealthHandler struct{}

func (h *HealthHandler) GetHealth(_ context.Context, _ shared.GetHealthRequestObject) (shared.GetHealthResponseObject, error) {
	return shared.GetHealth200JSONResponse{Status: "ok"}, nil
}
