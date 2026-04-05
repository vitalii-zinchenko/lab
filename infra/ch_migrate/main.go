package main

import (
	"database/sql"
	"fmt"
	"log"
	"os"

	_ "github.com/ClickHouse/clickhouse-go/v2"
	"github.com/pressly/goose/v3"
)

func main() {
	chURL := os.Getenv("CLICKHOUSE_URL")
	if chURL == "" {
		log.Fatal("CLICKHOUSE_URL environment variable is required")
	}

	migrationsDir := os.Getenv("MIGRATIONS_DIR")
	if migrationsDir == "" {
		migrationsDir = "migrations"
	}

	db, err := sql.Open("clickhouse", chURL)
	if err != nil {
		log.Fatalf("failed to open database: %v", err)
	}
	defer db.Close()

	if err := db.Ping(); err != nil {
		log.Fatalf("failed to connect to database: %v", err)
	}

	if err := goose.SetDialect("clickhouse"); err != nil {
		log.Fatalf("failed to set goose dialect: %v", err)
	}

	if err := goose.Up(db, migrationsDir); err != nil {
		log.Fatalf("failed to run migrations: %v", err)
	}

	fmt.Println("ClickHouse migrations applied successfully!")
}
