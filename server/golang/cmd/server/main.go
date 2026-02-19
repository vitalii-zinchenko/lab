package main

import (
	"log"
	"net/http"

	"github.com/gin-gonic/gin"
	ginmiddleware "github.com/oapi-codegen/gin-middleware"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/collectors"
	ginprometheus "github.com/zsais/go-gin-prometheus"
	"github.com/vitaliizinchenko/lab/gen"
	"github.com/vitaliizinchenko/lab/handler"
)

func init() {
	// Swap the default Go collector for the extended one so we get the full
	// runtime metric set: go_gc_cycles_*, go_gc_pauses_*, go_sched_*, etc.
	prometheus.Unregister(collectors.NewGoCollector())
	prometheus.MustRegister(collectors.NewGoCollector(
		collectors.WithGoCollectorRuntimeMetrics(collectors.MetricsAll),
	))
}

func main() {
	swagger, err := gen.GetSwagger()
	if err != nil {
		log.Fatalf("failed to load swagger spec: %v", err)
	}
	// Clear servers so the validator doesn't enforce host/scheme matching
	swagger.Servers = nil

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

	h := handler.New()
	strictHandler := gen.NewStrictHandler(h, nil)
	gen.RegisterHandlers(router, strictHandler)

	srv := &http.Server{
		Addr:    ":8080",
		Handler: router,
	}

	log.Println("Server listening on :8080")
	if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Fatalf("server error: %v", err)
	}
}
