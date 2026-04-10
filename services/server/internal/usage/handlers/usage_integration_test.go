package handlers_test

import (
	"context"
	"net/http"
	"testing"
	"time"

	"gorm.io/gorm"

	"github.com/vitaliizinchenko/lab/internal/shared/apiclient"
)

// waitForUsageCount polls api_usage until at least wantAtLeast rows exist for userID,
// or the timeout expires. Needed because the tracking middleware inserts asynchronously.
func waitForUsageCount(t *testing.T, db *gorm.DB, userID int64, wantAtLeast int, timeout time.Duration) {
	t.Helper()
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		var count int64
		db.WithContext(context.Background()).
			Raw("SELECT COUNT(*) FROM api_usage WHERE user_id = ?", userID).
			Scan(&count)
		if int(count) >= wantAtLeast {
			return
		}
		time.Sleep(50 * time.Millisecond)
	}
	t.Fatalf("waitForUsageCount: expected at least %d records for user %d within %s", wantAtLeast, userID, timeout)
}

func TestGetUsage_Unauthorized(t *testing.T) {
	ctx := context.Background()

	now := time.Now().UTC()
	resp, err := fixture.Client().GetUsageWithResponse(ctx, &apiclient.GetUsageParams{
		From: now.Add(-24 * time.Hour),
		To:   now,
	})
	if err != nil {
		t.Fatalf("GetUsage: %v", err)
	}
	if resp.StatusCode() != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d: %s", resp.StatusCode(), string(resp.Body))
	}
}

func TestGetUsage_EmptyWhenNoCallsInRange(t *testing.T) {
	ctx := context.Background()
	t.Cleanup(func() { fixture.TruncateTables(t) })

	user := fixture.CreateTestUser(t, ctx, "usageempty", "usageempty@example.com")
	key := fixture.CreateTestApiKey(t, ctx, user.JSON201.Id)
	token := fixture.GetToken(t, ctx, key.JSON201.ClientId.String(), key.JSON201.ClientSecret)

	// Query a range in the past with no activity.
	past := time.Now().UTC().Add(-48 * time.Hour)
	resp, err := fixture.AuthClient(token).GetUsageWithResponse(ctx, &apiclient.GetUsageParams{
		From: past,
		To:   past.Add(time.Hour),
	})
	if err != nil {
		t.Fatalf("GetUsage: %v", err)
	}
	if resp.StatusCode() != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", resp.StatusCode(), string(resp.Body))
	}
	if len(resp.JSON200.Data) != 0 {
		t.Errorf("expected empty data, got %d entries", len(resp.JSON200.Data))
	}
}

func TestGetUsage_CountsAuthenticatedCalls(t *testing.T) {
	ctx := context.Background()
	t.Cleanup(func() { fixture.TruncateTables(t) })

	user := fixture.CreateTestUser(t, ctx, "usagecount", "usagecount@example.com")
	key := fixture.CreateTestApiKey(t, ctx, user.JSON201.Id)
	token := fixture.GetToken(t, ctx, key.JSON201.ClientId.String(), key.JSON201.ClientSecret)

	client := fixture.AuthClient(token)

	const wantCalls = 3
	for i := 0; i < wantCalls; i++ {
		if _, err := client.ListApiKeysWithResponse(ctx); err != nil {
			t.Fatalf("ListApiKeys call %d: %v", i+1, err)
		}
	}
	waitForUsageCount(t, fixture.DB, user.JSON201.Id, wantCalls, 3*time.Second)

	now := time.Now().UTC()
	resp, err := client.GetUsageWithResponse(ctx, &apiclient.GetUsageParams{
		From: now.Add(-time.Hour),
		To:   now.Add(time.Hour),
	})
	if err != nil {
		t.Fatalf("GetUsage: %v", err)
	}
	if resp.StatusCode() != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", resp.StatusCode(), string(resp.Body))
	}
	if len(resp.JSON200.Data) == 0 {
		t.Fatal("expected at least one day bucket")
	}

	var total int
	for _, d := range resp.JSON200.Data {
		total += d.Count
	}
	if total < wantCalls {
		t.Errorf("expected total count >= %d, got %d", wantCalls, total)
	}
}

func TestGetUsage_FromAfterTo_ReturnsEmpty(t *testing.T) {
	ctx := context.Background()
	t.Cleanup(func() { fixture.TruncateTables(t) })

	user := fixture.CreateTestUser(t, ctx, "usagefromto", "usagefromto@example.com")
	key := fixture.CreateTestApiKey(t, ctx, user.JSON201.Id)
	token := fixture.GetToken(t, ctx, key.JSON201.ClientId.String(), key.JSON201.ClientSecret)

	now := time.Now().UTC()
	resp, err := fixture.AuthClient(token).GetUsageWithResponse(ctx, &apiclient.GetUsageParams{
		From: now,
		To:   now.Add(-24 * time.Hour), // from > to
	})
	if err != nil {
		t.Fatalf("GetUsage: %v", err)
	}
	if resp.StatusCode() != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", resp.StatusCode(), string(resp.Body))
	}
	if len(resp.JSON200.Data) != 0 {
		t.Errorf("expected empty data when from > to, got %d entries", len(resp.JSON200.Data))
	}
}

func TestGetUsage_IsolatedByUser(t *testing.T) {
	ctx := context.Background()
	t.Cleanup(func() { fixture.TruncateTables(t) })

	// User A makes several authenticated calls.
	userA := fixture.CreateTestUser(t, ctx, "usageA", "usageA@example.com")
	keyA := fixture.CreateTestApiKey(t, ctx, userA.JSON201.Id)
	tokenA := fixture.GetToken(t, ctx, keyA.JSON201.ClientId.String(), keyA.JSON201.ClientSecret)

	// User B makes no calls but queries usage.
	userB := fixture.CreateTestUser(t, ctx, "usageB", "usageB@example.com")
	keyB := fixture.CreateTestApiKey(t, ctx, userB.JSON201.Id)
	tokenB := fixture.GetToken(t, ctx, keyB.JSON201.ClientId.String(), keyB.JSON201.ClientSecret)

	clientA := fixture.AuthClient(tokenA)
	for i := 0; i < 2; i++ {
		if _, err := clientA.ListApiKeysWithResponse(ctx); err != nil {
			t.Fatalf("ListApiKeys: %v", err)
		}
	}
	waitForUsageCount(t, fixture.DB, userA.JSON201.Id, 2, 3*time.Second)

	now := time.Now().UTC()
	resp, err := fixture.AuthClient(tokenB).GetUsageWithResponse(ctx, &apiclient.GetUsageParams{
		From: now.Add(-time.Hour),
		To:   now.Add(time.Hour),
	})
	if err != nil {
		t.Fatalf("GetUsage: %v", err)
	}
	if resp.StatusCode() != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", resp.StatusCode(), string(resp.Body))
	}
	if len(resp.JSON200.Data) != 0 {
		t.Errorf("user B should see no usage, got %d entries", len(resp.JSON200.Data))
	}
}

func TestGetUsage_UnauthenticatedCallsNotTracked(t *testing.T) {
	ctx := context.Background()
	t.Cleanup(func() { fixture.TruncateTables(t) })

	user := fixture.CreateTestUser(t, ctx, "usageunauth", "usageunauth@example.com")
	key := fixture.CreateTestApiKey(t, ctx, user.JSON201.Id)
	token := fixture.GetToken(t, ctx, key.JSON201.ClientId.String(), key.JSON201.ClientSecret)

	// Make unauthenticated calls — these must not appear in usage.
	for i := 0; i < 3; i++ {
		if _, err := fixture.Client().ListItemsWithResponse(ctx, &apiclient.ListItemsParams{}); err != nil {
			t.Fatalf("ListItems: %v", err)
		}
	}

	now := time.Now().UTC()
	resp, err := fixture.AuthClient(token).GetUsageWithResponse(ctx, &apiclient.GetUsageParams{
		From: now.Add(-time.Hour),
		To:   now.Add(time.Hour),
	})
	if err != nil {
		t.Fatalf("GetUsage: %v", err)
	}
	if resp.StatusCode() != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", resp.StatusCode(), string(resp.Body))
	}
	// Only the GetUsage call itself may appear (it's authenticated), unauthenticated ListItems should not.
	for _, d := range resp.JSON200.Data {
		if d.Count > 1 {
			t.Errorf("expected at most 1 call (GetUsage itself), got %d", d.Count)
		}
	}
}
