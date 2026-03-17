package api

import "context"

type HealthHandler struct{}

func (h *HealthHandler) GetHealth(_ context.Context, _ GetHealthRequestObject) (GetHealthResponseObject, error) {
	return GetHealth200JSONResponse{Status: "ok"}, nil
}
