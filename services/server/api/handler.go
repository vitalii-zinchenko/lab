package api

import "github.com/vitaliizinchenko/lab/repository"

// Handler implements StrictServerInterface via embedded sub-handlers.
type Handler struct {
	*HealthHandler
	*ItemsHandler
	*EventsHandler
	*ChEventsHandler
	*UsersHandler
	*AuthHandler
}

// New creates a new Handler with the given repositories.
func New(
	itemRepo repository.ItemRepository,
	eventRepo repository.EventHistoryRepository,
	chEventRepo repository.ChEventRepository,
	userRepo repository.UserRepository,
	apiKeyRepo repository.ApiKeyRepository,
	jwtSecret []byte,
) *Handler {
	return &Handler{
		HealthHandler:   &HealthHandler{},
		ItemsHandler:    &ItemsHandler{itemRepo: itemRepo},
		EventsHandler:   &EventsHandler{eventRepo: eventRepo},
		ChEventsHandler: &ChEventsHandler{chEventRepo: chEventRepo},
		UsersHandler:    &UsersHandler{userRepo: userRepo},
		AuthHandler:     &AuthHandler{apiKeyRepo: apiKeyRepo, userRepo: userRepo, jwtSecret: jwtSecret},
	}
}
