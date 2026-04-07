package main

import (
	"database/sql"
	"log"
	"net/http"
	"os"
	"time"

	_ "github.com/ClickHouse/clickhouse-go/v2"
	"github.com/gin-gonic/gin"
	ginmiddleware "github.com/oapi-codegen/gin-middleware"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/collectors"
	ginprometheus "github.com/zsais/go-gin-prometheus"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"

	authhandlers "github.com/vitaliizinchenko/lab/internal/auth/handlers"
	authrepos "github.com/vitaliizinchenko/lab/internal/auth/repos"
	eventhandlers "github.com/vitaliizinchenko/lab/internal/events/handlers"
	eventrepos "github.com/vitaliizinchenko/lab/internal/events/repos"
	itemhandlers "github.com/vitaliizinchenko/lab/internal/items/handlers"
	itemrepos "github.com/vitaliizinchenko/lab/internal/items/repos"
	"github.com/vitaliizinchenko/lab/internal/shared"
)

func init() {
	// Swap the default Go collector for the extended one so we get the full
	// runtime metric set: go_gc_cycles_*, go_gc_pauses_*, go_sched_*, etc.
	prometheus.Unregister(collectors.NewGoCollector())
	prometheus.MustRegister(collectors.NewGoCollector(
		collectors.WithGoCollectorRuntimeMetrics(collectors.MetricsAll),
	))
}

// appHandler composes all domain sub-handlers to satisfy shared.StrictServerInterface.
type appHandler struct {
	*HealthHandler
	*itemhandlers.ItemsHandler
	*eventhandlers.EventsHandler
	*eventhandlers.ChEventsHandler
	*authhandlers.TokenHandler
	*authhandlers.ApiKeysHandler
	*authhandlers.UsersHandler
}

func main() {
	// --- Database ---
	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		log.Fatal("DATABASE_URL environment variable is required")
	}

	db, err := gorm.Open(postgres.Open(dbURL), &gorm.Config{})
	if err != nil {
		log.Fatalf("failed to connect to database: %v", err)
	}

	sqlDB, err := db.DB()
	if err != nil {
		log.Fatalf("failed to get sql.DB from gorm: %v", err)
	}
	sqlDB.SetMaxOpenConns(100)
	sqlDB.SetMaxIdleConns(25)
	sqlDB.SetConnMaxLifetime(5 * time.Minute)

	jwtSecret := []byte(os.Getenv("JWT_SECRET"))
	if len(jwtSecret) == 0 {
		log.Fatal("JWT_SECRET environment variable is required")
	}

	itemRepo := itemrepos.NewItemRepository(db)
	eventRepo := eventrepos.NewEventHistoryRepository(db)
	userRepo := authrepos.NewUserRepository(db)
	apiKeyRepo := authrepos.NewApiKeyRepository(db)

	// --- ClickHouse ---
	chURL := os.Getenv("CLICKHOUSE_URL")
	if chURL == "" {
		log.Fatal("CLICKHOUSE_URL environment variable is required")
	}

	chDB, err := sql.Open("clickhouse", chURL)
	if err != nil {
		log.Fatalf("failed to connect to clickhouse: %v", err)
	}
	defer chDB.Close()

	chEventRepo := eventrepos.NewChEventRepository(chDB)

	// --- OpenAPI / Swagger ---
	swagger, err := shared.GetSwagger()
	if err != nil {
		log.Fatalf("failed to load swagger spec: %v", err)
	}
	// Clear servers so the validator doesn't enforce host/scheme matching
	swagger.Servers = nil

	// --- Router ---
	router := gin.Default()

	// Prometheus metrics middleware — must be registered before other middleware
	// so it captures all requests including validation failures.
	// ReqCntURLLabelMappingFn uses the route template (e.g. /items/:id) instead
	// of the actual path, preventing cardinality explosion from UUID path params.
	p := ginprometheus.NewPrometheus("gin")
	p.ReqCntURLLabelMappingFn = func(c *gin.Context) string {
		return c.FullPath()
	}
	p.Use(router)

	// Validate all incoming requests against the OpenAPI spec
	router.Use(ginmiddleware.OapiRequestValidator(swagger))
	router.Use(authhandlers.Auth(jwtSecret))

	h := &appHandler{
		HealthHandler:   &HealthHandler{},
		ItemsHandler:    itemhandlers.NewItemsHandler(itemRepo),
		EventsHandler:   eventhandlers.NewEventsHandler(eventRepo),
		ChEventsHandler: eventhandlers.NewChEventsHandler(chEventRepo),
		TokenHandler:    authhandlers.NewTokenHandler(apiKeyRepo, jwtSecret),
		ApiKeysHandler:  authhandlers.NewApiKeysHandler(apiKeyRepo),
		UsersHandler:    authhandlers.NewUsersHandler(userRepo),
	}
	strictHandler := shared.NewStrictHandler(h, nil)
	shared.RegisterHandlers(router, strictHandler)

	srv := &http.Server{
		Addr:    ":8080",
		Handler: router,
	}

	log.Println("Server listening on :8080")
	if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Fatalf("server error: %v", err)
	}
}
