package api

import "github.com/vitaliizinchenko/lab/repository"

// Handler implements StrictServerInterface via embedded sub-handlers.
type Handler struct {
	*HealthHandler
	*ItemsHandler
	*EventsHandler
}

// New creates a new Handler with the given repositories.
func New(itemRepo repository.ItemRepository, eventRepo repository.EventHistoryRepository) *Handler {
	return &Handler{
		HealthHandler: &HealthHandler{},
		ItemsHandler:  &ItemsHandler{itemRepo: itemRepo},
		EventsHandler: &EventsHandler{eventRepo: eventRepo},
	}
}
