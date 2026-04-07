package main

import (
	"database/sql"
	"log"
	"net/http"
	"os"
	"time"

	_ "github.com/ClickHouse/clickhouse-go/v2"
	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/collectors"
	ginprometheus "github.com/zsais/go-gin-prometheus"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"

	internalserver "github.com/vitaliizinchenko/lab/internal/shared/server"
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

	// --- Prometheus middleware ---
	// ReqCntURLLabelMappingFn uses the route template (e.g. /items/:id) instead
	// of the actual path, preventing cardinality explosion from UUID path params.
	p := ginprometheus.NewPrometheus("gin")
	p.ReqCntURLLabelMappingFn = func(c *gin.Context) string {
		return c.FullPath()
	}

	// --- Router ---
	router := internalserver.NewRouter(db, chDB, jwtSecret, p.HandlerFunc())
	p.SetMetricsPath(router)

	srv := &http.Server{
		Addr:    ":8080",
		Handler: router,
	}

	log.Println("Server listening on :8080")
	if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Fatalf("server error: %v", err)
	}
}
