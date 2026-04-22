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

var postContents = []string{
	"Just had the best coffee of my life. No notes.",
	"Shipping code at midnight hits different.",
	"Hot take: tabs are better than spaces and I will die on this hill.",
	"The docs lied to me again. Classic.",
	"Three hours of debugging. It was a missing semicolon.",
	"Unpopular opinion: meetings could have been a README.",
	"My PR got merged on the first try. Today was a good day.",
	"Rewrote the whole thing from scratch. Now it's worse somehow.",
	"Found a bug introduced in 2019 that nobody noticed. Living in fear.",
	"Deleted more code than I wrote today. Progress.",
	"The tests pass locally. Deploying to prod. What could go wrong?",
	"Just discovered a feature that apparently nobody on the team knew existed.",
	"Note to self: read the error message before Googling for three hours.",
	"Refactored the utils file. Again. It keeps growing back.",
	"Someone pushed directly to main. We do not speak of this.",
	"The estimate was two days. It is now day six.",
	"Fixed a flaky test by making it less flaky. Still not sure how.",
	"Documentation written. Future self is now someone else's problem.",
	"Asked the rubber duck. The rubber duck knew.",
	"New laptop smell + fresh git clone = unstoppable.",
}

// fillPosts counts rows in posts and COPYs the gap up to target using
// numWorkers parallel COPY streams.
func fillPosts(ctx context.Context, pool *pgxpool.Pool, target, batchSize int64, numWorkers int, userIDs []int64) error {
	var current int64
	if err := pool.QueryRow(ctx, `SELECT COUNT(*) FROM posts`).Scan(&current); err != nil {
		return fmt.Errorf("count posts rows: %w", err)
	}
	log.Printf("current posts rows: %d, target: %d", current, target)

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
			if err := runPostsWorker(ctx, pool, workerID, share, batchSize, gap, userIDs, window, twoYearsAgo, &totalInserted); err != nil {
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
	log.Printf("all workers done — inserted %d rows, posts table now has ~%d rows", inserted, current+inserted)

	return nil
}

func runPostsWorker(
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
	contentCount := len(postContents)
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
			content := postContents[rng.Intn(contentCount)]
			rows[i] = []any{userID, content, ts, ts}
		}

		batchStart := time.Now()
		n, err := pool.CopyFrom(
			ctx,
			pgx.Identifier{"posts"},
			[]string{"user_id", "content", "created_at", "updated_at"},
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
