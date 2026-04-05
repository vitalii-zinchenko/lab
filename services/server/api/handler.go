package api

import "github.com/vitaliizinchenko/lab/repository"

// Handler implements StrictServerInterface via embedded sub-handlers.
type Handler struct {
	*HealthHandler
	*ItemsHandler
	*EventsHandler
	*ChEventsHandler
}

// New creates a new Handler with the given repositories.
func New(itemRepo repository.ItemRepository, eventRepo repository.EventHistoryRepository, chEventRepo repository.ChEventRepository) *Handler {
	return &Handler{
		HealthHandler:   &HealthHandler{},
		ItemsHandler:    &ItemsHandler{itemRepo: itemRepo},
		EventsHandler:   &EventsHandler{eventRepo: eventRepo},
		ChEventsHandler: &ChEventsHandler{chEventRepo: chEventRepo},
	}
}
