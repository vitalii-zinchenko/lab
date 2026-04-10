package server

import (
	"database/sql"
	"fmt"

	"github.com/gin-gonic/gin"
	ginmiddleware "github.com/oapi-codegen/gin-middleware"
	"gorm.io/gorm"

	authhandlers "github.com/vitaliizinchenko/lab/internal/auth/handlers"
	authrepos "github.com/vitaliizinchenko/lab/internal/auth/repos"
	eventhandlers "github.com/vitaliizinchenko/lab/internal/events/handlers"
	eventrepos "github.com/vitaliizinchenko/lab/internal/events/repos"
	itemhandlers "github.com/vitaliizinchenko/lab/internal/items/handlers"
	itemrepos "github.com/vitaliizinchenko/lab/internal/items/repos"
	"github.com/vitaliizinchenko/lab/internal/shared"
	usagehandlers "github.com/vitaliizinchenko/lab/internal/usage/handlers"
	usagerepos "github.com/vitaliizinchenko/lab/internal/usage/repos"
)

// appHandler composes all domain sub-handlers to satisfy shared.StrictServerInterface.
type appHandler struct {
	*healthHandler
	*itemhandlers.ItemsHandler
	*eventhandlers.EventsHandler
	*eventhandlers.ChEventsHandler
	*authhandlers.TokenHandler
	*authhandlers.ApiKeysHandler
	*authhandlers.UsersHandler
	*usagehandlers.UsageHandler
}

// NewRouter builds a configured Gin engine with OpenAPI validation, auth middleware, and all
// route handlers registered. Optional pre middleware (e.g. Prometheus) is prepended so it
// runs before the standard stack.
func NewRouter(db *gorm.DB, chDB *sql.DB, jwtSecret []byte, pre ...gin.HandlerFunc) *gin.Engine {
	swagger, err := shared.GetSwagger()
	if err != nil {
		panic(fmt.Sprintf("failed to load swagger spec: %v", err))
	}
	swagger.Servers = nil

	router := gin.New()
	router.Use(pre...)
	router.Use(ginmiddleware.OapiRequestValidator(swagger))
	router.Use(authhandlers.Auth(jwtSecret))

	itemRepo := itemrepos.NewItemRepository(db)
	eventRepo := eventrepos.NewEventHistoryRepository(db)
	userRepo := authrepos.NewUserRepository(db)
	apiKeyRepo := authrepos.NewApiKeyRepository(db)
	chEventRepo := eventrepos.NewChEventRepository(chDB)
	usageRepo := usagerepos.NewPostgresUsageRepository(db)

	h := &appHandler{
		healthHandler:   &healthHandler{},
		ItemsHandler:    itemhandlers.NewItemsHandler(itemRepo),
		EventsHandler:   eventhandlers.NewEventsHandler(eventRepo),
		ChEventsHandler: eventhandlers.NewChEventsHandler(chEventRepo),
		TokenHandler:    authhandlers.NewTokenHandler(apiKeyRepo, jwtSecret),
		ApiKeysHandler:  authhandlers.NewApiKeysHandler(apiKeyRepo),
		UsersHandler:    authhandlers.NewUsersHandler(userRepo),
		UsageHandler:    usagehandlers.NewUsageHandler(usageRepo),
	}

	strictHandler := shared.NewStrictHandler(h, []shared.StrictMiddlewareFunc{
		usagehandlers.UsageTrackingMiddleware(usageRepo),
	})
	shared.RegisterHandlers(router, strictHandler)

	return router
}
