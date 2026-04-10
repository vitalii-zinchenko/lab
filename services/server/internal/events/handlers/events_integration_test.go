package handlers_test

import (
	"context"
	"net/http"
	"testing"

	"github.com/vitaliizinchenko/lab/internal/shared/apiclient"
)

// --- Postgres events ---

func TestCreateEvent_Success(t *testing.T) {
	ctx := context.Background()
	t.Cleanup(func() { fixture.TruncateTables(t) })

	resp, err := fixture.Client().CreateEventWithResponse(ctx, apiclient.NewEvent{
		Level:     apiclient.Info,
		EventType: "user.created",
	})
	if err != nil {
		t.Fatalf("CreateEvent: %v", err)
	}

	if resp.StatusCode() != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", resp.StatusCode(), string(resp.Body))
	}
	if resp.JSON201 == nil {
		t.Fatal("expected JSON body")
	}
	if resp.JSON201.EventType != "user.created" {
		t.Errorf("event_type: want user.created, got %s", resp.JSON201.EventType)
	}
	if string(resp.JSON201.Level) != "info" {
		t.Errorf("level: want info, got %s", resp.JSON201.Level)
	}
	if resp.JSON201.Id == (apiclient.EventHistory{}).Id {
		t.Error("expected non-zero id")
	}
	if resp.JSON201.CreatedAt.IsZero() {
		t.Error("expected non-zero created_at")
	}
	if resp.JSON201.Details != nil {
		t.Errorf("expected nil details, got %v", *resp.JSON201.Details)
	}
}

func TestCreateEvent_WithDetails(t *testing.T) {
	ctx := context.Background()
	t.Cleanup(func() { fixture.TruncateTables(t) })

	details := `{"user_id": 42}`
	resp, err := fixture.Client().CreateEventWithResponse(ctx, apiclient.NewEvent{
		Level:     apiclient.Error,
		EventType: "payment.failed",
		Details:   &details,
	})
	if err != nil {
		t.Fatalf("CreateEvent: %v", err)
	}
	if resp.StatusCode() != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", resp.StatusCode(), string(resp.Body))
	}
	if resp.JSON201.Details == nil || *resp.JSON201.Details != details {
		t.Errorf("details: want %q, got %v", details, resp.JSON201.Details)
	}
}

func TestCreateEvent_AllLevels(t *testing.T) {
	ctx := context.Background()
	t.Cleanup(func() { fixture.TruncateTables(t) })

	for _, level := range []apiclient.EventLevel{
		apiclient.Info,
		apiclient.Warn,
		apiclient.Error,
	} {
		t.Run(string(level), func(t *testing.T) {
			resp, err := fixture.Client().CreateEventWithResponse(ctx, apiclient.NewEvent{
				Level:     level,
				EventType: "test.event",
			})
			if err != nil {
				t.Fatalf("CreateEvent: %v", err)
			}
			if resp.StatusCode() != http.StatusCreated {
				t.Fatalf("expected 201, got %d: %s", resp.StatusCode(), string(resp.Body))
			}
		})
	}
}

func TestCreateEvent_InvalidLevel(t *testing.T) {
	ctx := context.Background()

	// Send raw JSON to bypass the typed client's enum validation.
	body := `{"level":"critical","event_type":"test"}`
	resp, err := fixture.Client().CreateEventWithBodyWithResponse(ctx, "application/json",
		mustStringReader(body))
	if err != nil {
		t.Fatalf("CreateEvent: %v", err)
	}

	// OpenAPI validator rejects the unknown enum value before hitting the handler.
	if resp.StatusCode() != http.StatusBadRequest {
		t.Fatalf("expected 400 for invalid level, got %d: %s", resp.StatusCode(), string(resp.Body))
	}
}

func TestCreateEvent_MissingEventType(t *testing.T) {
	ctx := context.Background()

	body := `{"level":"info"}`
	resp, err := fixture.Client().CreateEventWithBodyWithResponse(ctx, "application/json",
		mustStringReader(body))
	if err != nil {
		t.Fatalf("CreateEvent: %v", err)
	}
	if resp.StatusCode() != http.StatusBadRequest {
		t.Fatalf("expected 400 for missing event_type, got %d: %s", resp.StatusCode(), string(resp.Body))
	}
}

// --- ClickHouse events ---

func TestCreateChEvent_Success(t *testing.T) {
	ctx := context.Background()

	resp, err := fixture.Client().CreateChEventWithResponse(ctx, apiclient.NewEvent{
		Level:     apiclient.Warn,
		EventType: "cache.miss",
	})
	if err != nil {
		t.Fatalf("CreateChEvent: %v", err)
	}

	if resp.StatusCode() != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", resp.StatusCode(), string(resp.Body))
	}
	if resp.JSON201 == nil {
		t.Fatal("expected JSON body")
	}
	if resp.JSON201.EventType != "cache.miss" {
		t.Errorf("event_type: want cache.miss, got %s", resp.JSON201.EventType)
	}
	if string(resp.JSON201.Level) != "warn" {
		t.Errorf("level: want warn, got %s", resp.JSON201.Level)
	}
}

func TestCreateChEvent_WithDetails(t *testing.T) {
	ctx := context.Background()

	details := "key=homepage ttl=300"
	resp, err := fixture.Client().CreateChEventWithResponse(ctx, apiclient.NewEvent{
		Level:     apiclient.Info,
		EventType: "cache.hit",
		Details:   &details,
	})
	if err != nil {
		t.Fatalf("CreateChEvent: %v", err)
	}
	if resp.StatusCode() != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", resp.StatusCode(), string(resp.Body))
	}
	if resp.JSON201.Details == nil || *resp.JSON201.Details != details {
		t.Errorf("details: want %q, got %v", details, resp.JSON201.Details)
	}
}

func TestCreateChEvent_InvalidLevel(t *testing.T) {
	ctx := context.Background()

	body := `{"level":"debug","event_type":"test"}`
	resp, err := fixture.Client().CreateChEventWithBodyWithResponse(ctx, "application/json",
		mustStringReader(body))
	if err != nil {
		t.Fatalf("CreateChEvent: %v", err)
	}
	if resp.StatusCode() != http.StatusBadRequest {
		t.Fatalf("expected 400 for invalid level, got %d: %s", resp.StatusCode(), string(resp.Body))
	}
}
