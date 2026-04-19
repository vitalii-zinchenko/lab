package main

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	_ "github.com/ClickHouse/clickhouse-go/v2"
	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/collectors"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"go.uber.org/fx"
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

func newPostgresDB() (*gorm.DB, error) {
	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		return nil, fmt.Errorf("DATABASE_URL environment variable is required")
	}

	db, err := gorm.Open(postgres.Open(dbURL), &gorm.Config{})
	if err != nil {
		return nil, fmt.Errorf("failed to connect to database: %w", err)
	}

	sqlDB, err := db.DB()
	if err != nil {
		return nil, fmt.Errorf("failed to get sql.DB from gorm: %w", err)
	}
	sqlDB.SetMaxOpenConns(100)
	sqlDB.SetMaxIdleConns(25)
	sqlDB.SetConnMaxLifetime(5 * time.Minute)

	return db, nil
}

func newClickHouseDB(lc fx.Lifecycle) (*sql.DB, error) {
	chURL := os.Getenv("CLICKHOUSE_URL")
	if chURL == "" {
		return nil, fmt.Errorf("CLICKHOUSE_URL environment variable is required")
	}

	db, err := sql.Open("clickhouse", chURL)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to clickhouse: %w", err)
	}

	lc.Append(fx.Hook{
		OnStop: func(ctx context.Context) error { return db.Close() },
	})

	return db, nil
}

func newJWTSecret() ([]byte, error) {
	secret := os.Getenv("JWT_SECRET")
	if secret == "" {
		return nil, fmt.Errorf("JWT_SECRET environment variable is required")
	}
	return []byte(secret), nil
}

func startMetricsServer(lc fx.Lifecycle) {
	mux := http.NewServeMux()
	mux.Handle("/metrics", promhttp.Handler())
	srv := &http.Server{Addr: ":9091", Handler: mux}

	lc.Append(fx.Hook{
		OnStart: func(ctx context.Context) error {
			go func() {
				log.Println("Metrics listening on :9091")
				if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
					log.Printf("metrics server error: %v", err)
				}
			}()
			return nil
		},
		OnStop: func(ctx context.Context) error { return srv.Shutdown(ctx) },
	})
}

func startHTTPServer(lc fx.Lifecycle, router *gin.Engine) {
	srv := &http.Server{Addr: ":8080", Handler: router}

	lc.Append(fx.Hook{
		OnStart: func(ctx context.Context) error {
			go func() {
				log.Println("Server listening on :8080")
				if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
					log.Printf("server error: %v", err)
				}
			}()
			return nil
		},
		OnStop: func(ctx context.Context) error { return srv.Shutdown(ctx) },
	})
}

func main() {
	fx.New(
		internalserver.AppModule,

		fx.Provide(newPostgresDB),
		fx.Provide(newClickHouseDB),
		fx.Provide(newJWTSecret),

		fx.Invoke(startMetricsServer),
		fx.Invoke(startHTTPServer),
	).Run()
}
