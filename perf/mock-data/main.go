package main

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"log"
	mathrand "math/rand"
	"os"
	"sync"
	"sync/atomic"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/spf13/cobra"
	"golang.org/x/crypto/bcrypt"
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

var operations = []string{"read", "write", "delete", "create"}

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

// ensureUsers guarantees that exactly `count` mock users exist.
// Returns the slice of their IDs.
func ensureUsers(ctx context.Context, pool *pgxpool.Pool, count int64) ([]int64, error) {
	rows, err := pool.Query(ctx,
		`SELECT id FROM users WHERE username LIKE $1 ORDER BY id`,
		usernamePrefix+"%",
	)
	if err != nil {
		return nil, fmt.Errorf("query existing users: %w", err)
	}
	existingIDs, err := pgx.CollectRows(rows, pgx.RowTo[int64])
	if err != nil {
		return nil, fmt.Errorf("collect existing user ids: %w", err)
	}

	existing := int64(len(existingIDs))
	log.Printf("found %d existing mock users (target: %d)", existing, count)

	if existing >= count {
		return existingIDs[:count], nil
	}

	missing := count - existing
	log.Printf("creating %d new mock users via COPY...", missing)

	now := time.Now().UTC()
	copyRows := make([][]any, 0, missing)
	for i := existing + 1; i <= count; i++ {
		copyRows = append(copyRows, []any{
			fmt.Sprintf("%s%04d", usernamePrefix, i),
			fmt.Sprintf("mock_user_%04d@example.com", i),
			now,
		})
	}

	inserted, err := pool.CopyFrom(
		ctx,
		pgx.Identifier{"users"},
		[]string{"username", "email", "created_at"},
		pgx.CopyFromRows(copyRows),
	)
	if err != nil {
		return nil, fmt.Errorf("copy users: %w", err)
	}
	log.Printf("inserted %d new mock users", inserted)

	rows, err = pool.Query(ctx,
		`SELECT id FROM users WHERE username LIKE $1 ORDER BY id LIMIT $2`,
		usernamePrefix+"%", count,
	)
	if err != nil {
		return nil, fmt.Errorf("re-query user ids: %w", err)
	}
	allIDs, err := pgx.CollectRows(rows, pgx.RowTo[int64])
	if err != nil {
		return nil, fmt.Errorf("collect all user ids: %w", err)
	}

	return allIDs, nil
}

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

// fillUsage counts rows in usage and COPYs the gap up to target using
// numWorkers parallel COPY streams.
func fillUsage(ctx context.Context, pool *pgxpool.Pool, target, batchSize int64, numWorkers int, userIDs []int64) error {
	var current int64
	if err := pool.QueryRow(ctx, `SELECT COUNT(*) FROM usage`).Scan(&current); err != nil {
		return fmt.Errorf("count usage rows: %w", err)
	}
	log.Printf("current usage rows: %d, target: %d", current, target)

	gap := target - current
	if gap <= 0 {
		log.Println("already at or above target, nothing to do")
		return nil
	}
	log.Printf("need to insert %d rows using %d worker(s), batch size %d", gap, numWorkers, batchSize)

	now := time.Now().UTC()
	twoYearsAgo := now.Add(-2 * 365 * 24 * time.Hour)
	window := now.Sub(twoYearsAgo).Seconds()

	var totalInserted atomic.Int64

	shareBase := gap / int64(numWorkers)
	remainder := gap % int64(numWorkers)

	var wg sync.WaitGroup
	errCh := make(chan error, numWorkers)

	for w := 0; w < numWorkers; w++ {
		share := shareBase
		if int64(w) < remainder {
			share++
		}
		wg.Add(1)
		go func(workerID int, share int64) {
			defer wg.Done()
			if err := runWorker(ctx, pool, workerID, share, batchSize, gap, userIDs, window, twoYearsAgo, &totalInserted); err != nil {
				errCh <- fmt.Errorf("worker %d: %w", workerID, err)
			}
		}(w, share)
	}

	wg.Wait()
	close(errCh)

	if err := <-errCh; err != nil {
		return err
	}

	inserted := totalInserted.Load()
	log.Printf("all workers done — inserted %d rows, usage table now has ~%d rows", inserted, current+inserted)

	return nil
}

func runWorker(
	ctx context.Context,
	pool *pgxpool.Pool,
	workerID int,
	share, batchSize, gap int64,
	userIDs []int64,
	window float64,
	twoYearsAgo time.Time,
	totalInserted *atomic.Int64,
) error {
	rng := mathrand.New(mathrand.NewSource(time.Now().UnixNano() + int64(workerID)*1_000_000))
	userCount := int64(len(userIDs))
	inserted := int64(0)
	batchNum := 0

	for inserted < share {
		size := batchSize
		if remaining := share - inserted; remaining < size {
			size = remaining
		}
		batchNum++

		rows := make([][]any, size)
		for i := int64(0); i < size; i++ {
			offsetSec := rng.Float64() * window
			ts := twoYearsAgo.Add(time.Duration(offsetSec * float64(time.Second)))
			userID := userIDs[rng.Int63n(userCount)]
			op := operations[rng.Intn(len(operations))]
			rows[i] = []any{ts, userID, op}
		}

		batchStart := time.Now()
		n, err := pool.CopyFrom(
			ctx,
			pgx.Identifier{"usage"},
			[]string{"timestamp", "user_id", "operation"},
			pgx.CopyFromRows(rows),
		)
		if err != nil {
			return fmt.Errorf("batch %d: %w", batchNum, err)
		}
		elapsed := time.Since(batchStart)

		inserted += int64(n)
		globalTotal := totalInserted.Add(int64(n))
		pct := float64(globalTotal) / float64(gap) * 100
		rowsPerSec := float64(n) / elapsed.Seconds()
		log.Printf("worker %d | batch %d: inserted %d rows in %s (%.0f rows/s) — total: %d / %d (%.1f%%)",
			workerID, batchNum, n, elapsed.Round(time.Millisecond), rowsPerSec, globalTotal, gap, pct)
	}

	return nil
}

// userCredential is one entry in the output JSON file.
type userCredential struct {
	UserID       int64  `json:"user_id"`
	Email        string `json:"email"`
	Password     string `json:"password"`
	ClientID     string `json:"client_id"`
	ClientSecret string `json:"client_secret"`
}

// createApiKeys ensures every mock user has exactly one non-revoked API key
// named "mock_key" and a password stored in password_hash. Existing keys and
// passwords are reused when the output file already records them; otherwise
// they are recreated. The full credentials list is written to --out on success.
func createApiKeys(ctx context.Context, pool *pgxpool.Pool, outPath string) error {
	existing := map[int64]userCredential{}
	if data, err := os.ReadFile(outPath); err == nil {
		var creds []userCredential
		if json.Unmarshal(data, &creds) == nil {
			for _, c := range creds {
				existing[c.UserID] = c
			}
		}
	}

	type userRow struct {
		ID    int64
		Email string
	}
	rows, err := pool.Query(ctx,
		`SELECT id, email FROM users WHERE username LIKE $1 ORDER BY id`,
		usernamePrefix+"%",
	)
	if err != nil {
		return fmt.Errorf("query users: %w", err)
	}
	users, err := pgx.CollectRows(rows, pgx.RowToStructByPos[userRow])
	if err != nil {
		return fmt.Errorf("collect users: %w", err)
	}
	log.Printf("found %d mock users", len(users))

	result := make([]userCredential, 0, len(users))

	for _, u := range users {
		cred := userCredential{UserID: u.ID, Email: u.Email}

		// ---- password ----
		var hasPassword bool
		pool.QueryRow(ctx,
			`SELECT password_hash IS NOT NULL FROM users WHERE id = $1`,
			u.ID,
		).Scan(&hasPassword)

		if prev, ok := existing[u.ID]; ok && prev.Password != "" && hasPassword {
			cred.Password = prev.Password
		} else {
			rawPassword, err := generateSecret()
			if err != nil {
				return fmt.Errorf("generate password for user %d: %w", u.ID, err)
			}
			hash, err := bcrypt.GenerateFromPassword([]byte(rawPassword), bcrypt.DefaultCost)
			if err != nil {
				return fmt.Errorf("hash password for user %d: %w", u.ID, err)
			}
			if _, err := pool.Exec(ctx,
				`UPDATE users SET password_hash = $1 WHERE id = $2`,
				string(hash), u.ID,
			); err != nil {
				return fmt.Errorf("update password_hash for user %d: %w", u.ID, err)
			}
			cred.Password = rawPassword
		}

		// ---- api key ----
		var dbClientID string
		keyErr := pool.QueryRow(ctx,
			`SELECT client_id FROM api_keys
			 WHERE user_id = $1 AND name = 'mock_key' AND revoked_at IS NULL
			 LIMIT 1`,
			u.ID,
		).Scan(&dbClientID)

		keyExists := keyErr == nil

		if keyExists {
			if prev, ok := existing[u.ID]; ok && prev.ClientID == dbClientID {
				cred.ClientID = prev.ClientID
				cred.ClientSecret = prev.ClientSecret
				result = append(result, cred)
				continue
			}
			if _, err := pool.Exec(ctx,
				`DELETE FROM api_keys WHERE user_id = $1 AND name = 'mock_key'`,
				u.ID,
			); err != nil {
				return fmt.Errorf("delete stale key for user %d: %w", u.ID, err)
			}
		}

		rawSecret, err := generateSecret()
		if err != nil {
			return fmt.Errorf("generate secret for user %d: %w", u.ID, err)
		}
		hash, err := bcrypt.GenerateFromPassword([]byte(rawSecret), bcrypt.DefaultCost)
		if err != nil {
			return fmt.Errorf("hash secret for user %d: %w", u.ID, err)
		}

		clientID := uuid.New()
		now := time.Now().UTC()
		keyName := "mock_key"
		if _, err := pool.Exec(ctx,
			`INSERT INTO api_keys (id, user_id, client_id, client_secret_hash, name, created_at)
			 VALUES ($1, $2, $3, $4, $5, $6)`,
			uuid.New(), u.ID, clientID, string(hash), keyName, now,
		); err != nil {
			return fmt.Errorf("insert api key for user %d: %w", u.ID, err)
		}

		cred.ClientID = clientID.String()
		cred.ClientSecret = rawSecret
		result = append(result, cred)
	}

	out, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal credentials: %w", err)
	}
	if err := os.WriteFile(outPath, out, 0600); err != nil {
		return fmt.Errorf("write credentials file: %w", err)
	}
	log.Printf("wrote %d credentials to %s", len(result), outPath)
	return nil
}

// generateSecret produces a cryptographically random URL-safe base64 string.
func generateSecret() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return base64.URLEncoding.EncodeToString(b), nil
}
