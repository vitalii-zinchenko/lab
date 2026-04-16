package main

import (
	"context"
	"fmt"
	"log"
	mathrand "math/rand"
	"sync"
	"sync/atomic"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

var operations = []string{"read", "write", "delete", "create"}

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
