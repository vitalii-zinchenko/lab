package main

import (
	"context"
	"fmt"
	"log"

	"github.com/jackc/pgx/v5/pgxpool"
)

// dropIndex drops idx_usage_user_timestamp if it exists.
func dropIndex(ctx context.Context, pool *pgxpool.Pool) error {
	_, err := pool.Exec(ctx, `DROP INDEX IF EXISTS idx_usage_user_timestamp`)
	return err
}

// recreateIndex rebuilds idx_usage_user_timestamp using CONCURRENTLY so it
// does not hold an exclusive lock. SET maintenance_work_mem applies only to
// this session to speed up the sort phase.
func recreateIndex(ctx context.Context, pool *pgxpool.Pool, workMem string) error {
	log.Println("rebuilding idx_usage_user_timestamp (this may take a while)...")

	conn, err := pool.Acquire(ctx)
	if err != nil {
		return fmt.Errorf("acquire connection: %w", err)
	}
	defer conn.Release()

	log.Printf("setting maintenance_work_mem = '%s' for this session", workMem)
	if _, err := conn.Exec(ctx, "SET maintenance_work_mem = '"+workMem+"'"); err != nil {
		return fmt.Errorf("set maintenance_work_mem: %w", err)
	}

	_, err = conn.Exec(ctx,
		`CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_usage_user_timestamp
		 ON usage (user_id, timestamp DESC)`,
	)
	return err
}
