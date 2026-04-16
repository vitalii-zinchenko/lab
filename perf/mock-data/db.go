package main

import (
	"context"
	"log"

	"github.com/jackc/pgx/v5/pgxpool"
)

// mustConnect creates a pgxpool and panics on failure.
func mustConnect(dbURL string) *pgxpool.Pool {
	pool, err := pgxpool.New(context.Background(), dbURL)
	if err != nil {
		log.Fatalf("failed to connect to database: %v", err)
	}
	if err := pool.Ping(context.Background()); err != nil {
		log.Fatalf("failed to ping database: %v", err)
	}
	log.Println("connected to database")
	return pool
}
