package server

import (
	"context"

	"github.com/vitaliizinchenko/lab/internal/shared"
)

type healthHandler struct{}

func (h *healthHandler) GetHealth(_ context.Context, _ shared.GetHealthRequestObject) (shared.GetHealthResponseObject, error) {
	return shared.GetHealth200JSONResponse{Status: "ok"}, nil
}
