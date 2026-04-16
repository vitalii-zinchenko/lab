package main

import (
	"context"
	"fmt"
	"log"
	"os"

	"github.com/spf13/cobra"
)

const (
	defaultTarget    = 1_000_000_000
	defaultUserCount = 1_000
	defaultBatch     = 100_000
	defaultWorkers   = 1
	defaultWorkMem   = "256MB"
	defaultDBURL     = "postgres://postgres:postgres@localhost:5432/app?sslmode=disable"

	usernamePrefix = "mock_user_"
)

func main() {
	var dbURL string

	// ----- root -----
	root := &cobra.Command{
		Use:   "mock-data",
		Short: "Mock data tools for the lab project",
	}
	root.PersistentFlags().StringVar(&dbURL, "db", defaultDBURL, "database connection URL")

	// ----- fill -----
	var (
		fillTarget    int64
		fillUserCount int64
		fillBatch     int64
		fillWorkers   int
	)
	fillCmd := &cobra.Command{
		Use:   "fill",
		Short: "Fill the usage table up to the target row count",
		RunE: func(cmd *cobra.Command, args []string) error {
			pool := mustConnect(dbURL)
			defer pool.Close()
			userIDs, err := ensureUsers(context.Background(), pool, fillUserCount)
			if err != nil {
				return fmt.Errorf("ensure users: %w", err)
			}
			return fillUsage(context.Background(), pool, fillTarget, fillBatch, fillWorkers, userIDs)
		},
	}
	fillCmd.Flags().Int64Var(&fillTarget, "target", defaultTarget, "target total rows in the usage table")
	fillCmd.Flags().Int64Var(&fillUserCount, "users", defaultUserCount, "number of mock users to maintain")
	fillCmd.Flags().Int64Var(&fillBatch, "batch", defaultBatch, "rows per COPY batch")
	fillCmd.Flags().IntVar(&fillWorkers, "workers", defaultWorkers, "number of parallel COPY workers")

	// ----- create-api-keys -----
	var createApiKeysOut string
	createApiKeysCmd := &cobra.Command{
		Use:   "create-api-keys",
		Short: "Create one API key per mock user and write a credentials JSON file",
		Long: `Ensures every mock user has one non-revoked API key named "mock_key".
Existing keys are reused when the output file already contains their secret;
otherwise the stale key is deleted and a fresh one is created.
The full credentials list is written to --out on success.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			pool := mustConnect(dbURL)
			defer pool.Close()
			return createApiKeys(context.Background(), pool, createApiKeysOut)
		},
	}
	createApiKeysCmd.Flags().StringVar(&createApiKeysOut, "out", "users-credentials.json", "output JSON file path")

	// ----- drop-index -----
	dropIndexCmd := &cobra.Command{
		Use:   "drop-index",
		Short: "Drop idx_usage_user_timestamp",
		RunE: func(cmd *cobra.Command, args []string) error {
			pool := mustConnect(dbURL)
			defer pool.Close()
			if err := dropIndex(context.Background(), pool); err != nil {
				return err
			}
			log.Println("index dropped")
			return nil
		},
	}

	// ----- reindex -----
	var reindexWorkMem string
	reindexCmd := &cobra.Command{
		Use:   "reindex",
		Short: "Drop and recreate idx_usage_user_timestamp",
		RunE: func(cmd *cobra.Command, args []string) error {
			pool := mustConnect(dbURL)
			defer pool.Close()
			if err := recreateIndex(context.Background(), pool, reindexWorkMem); err != nil {
				return err
			}
			log.Println("index rebuilt")
			return nil
		},
	}
	reindexCmd.Flags().StringVar(&reindexWorkMem, "work-mem", defaultWorkMem, "PostgreSQL maintenance_work_mem for the index build (e.g. 1GB)")

	root.AddCommand(fillCmd, createApiKeysCmd, dropIndexCmd, reindexCmd)

	if err := root.Execute(); err != nil {
		os.Exit(1)
	}
}
