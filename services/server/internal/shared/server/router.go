package server

import (
	"database/sql"
	"fmt"
	"net/http"
	"strings"

	"github.com/getkin/kin-openapi/openapi3filter"
	"github.com/gin-gonic/gin"
	ginmiddleware "github.com/oapi-codegen/gin-middleware"
	"go.uber.org/fx"
	"gorm.io/gorm"

	authhandlers "github.com/vitaliizinchenko/lab/internal/auth/handlers"
	eventhandlers "github.com/vitaliizinchenko/lab/internal/events/handlers"
	itemhandlers "github.com/vitaliizinchenko/lab/internal/items/handlers"
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
	*authhandlers.LoginHandler
	*usagehandlers.UsageHandler
}

// RouterParams holds all dependencies required to build the router.
// fx resolves and injects each field automatically.
type RouterParams struct {
	fx.In

	DB        *gorm.DB
	ChDB      *sql.DB
	JWTSecret []byte

	// PrometheusMiddleware is the Gin handler for Prometheus metrics collection.
	// It is named to avoid ambiguity with any other gin.HandlerFunc providers.
	PrometheusMiddleware gin.HandlerFunc `name:"prometheus"`

	ItemsHandler    *itemhandlers.ItemsHandler
	EventsHandler   *eventhandlers.EventsHandler
	ChEventsHandler *eventhandlers.ChEventsHandler
	TokenHandler    *authhandlers.TokenHandler
	ApiKeysHandler  *authhandlers.ApiKeysHandler
	UsersHandler    *authhandlers.UsersHandler
	LoginHandler    *authhandlers.LoginHandler
	UsageHandler    *usagehandlers.UsageHandler

	// UsageRepo is injected separately because UsageTrackingMiddleware needs it
	// directly, independently of UsageHandler.
	UsageRepo usagerepos.UsageRepository
}

// NewRouter builds a configured Gin engine with OpenAPI validation, auth middleware,
// and all route handlers registered.
func NewRouter(p RouterParams) *gin.Engine {
	swagger, err := shared.GetSwagger()
	if err != nil {
		panic(fmt.Sprintf("failed to load swagger spec: %v", err))
	}
	swagger.Servers = nil

	router := gin.New()
	router.ContextWithFallback = true
	router.Use(p.PrometheusMiddleware)
	router.Use(ginmiddleware.OapiRequestValidatorWithOptions(swagger, &ginmiddleware.Options{
		Options: openapi3filter.Options{
			AuthenticationFunc: authhandlers.NewAuthenticationFunc(p.JWTSecret),
		},
		ErrorHandler: func(c *gin.Context, message string, statusCode int) {
			if strings.Contains(message, "SecurityRequirementsError") {
				c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
				return
			}
			c.AbortWithStatusJSON(statusCode, gin.H{"error": message})
		},
	}))

	h := &appHandler{
		healthHandler:   &healthHandler{},
		ItemsHandler:    p.ItemsHandler,
		EventsHandler:   p.EventsHandler,
		ChEventsHandler: p.ChEventsHandler,
		TokenHandler:    p.TokenHandler,
		ApiKeysHandler:  p.ApiKeysHandler,
		UsersHandler:    p.UsersHandler,
		LoginHandler:    p.LoginHandler,
		UsageHandler:    p.UsageHandler,
	}

	strictHandler := shared.NewStrictHandler(h, []shared.StrictMiddlewareFunc{
		usagehandlers.UsageTrackingMiddleware(p.UsageRepo),
	})
	shared.RegisterHandlers(router, strictHandler)

	return router
}
