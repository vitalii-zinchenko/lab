package testinfra

import (
	"bufio"
	"context"
	"database/sql"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	_ "github.com/ClickHouse/clickhouse-go/v2"
	"github.com/google/uuid"
	openapi_types "github.com/oapi-codegen/runtime/types"
	"github.com/pressly/goose/v3"
	tcclickhouse "github.com/testcontainers/testcontainers-go/modules/clickhouse"
	tcpostgres "github.com/testcontainers/testcontainers-go/modules/postgres"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"

	"github.com/vitaliizinchenko/lab/internal/shared/apiclient"
	internalserver "github.com/vitaliizinchenko/lab/internal/shared/server"
)

// Fixture holds a fully wired test environment: real Postgres + ClickHouse containers,
// migrations applied, and an httptest.Server running the actual Gin router.
type Fixture struct {
	DB        *gorm.DB
	ChDB      *sql.DB
	Server    *httptest.Server
	JWTSecret []byte
	cleanup   []func()
}

// repoRoot resolves the monorepo root at runtime using the path of this source file.
// This is stable regardless of the working directory the tests are run from.
func repoRoot() string {
	_, filename, _, _ := runtime.Caller(0)
	// filename: .../services/server/internal/testinfra/fixture.go
	// walk 4 levels up to reach lab/
	return filepath.Join(filepath.Dir(filename), "..", "..", "..", "..")
}

// NewFixture starts Postgres and ClickHouse containers, runs all migrations, starts the Gin
// server via httptest, and returns the ready fixture. Call Cleanup() when done.
func NewFixture(ctx context.Context) (*Fixture, error) {
	f := &Fixture{JWTSecret: []byte("test-jwt-secret")}
	root := repoRoot()

	// --- Postgres ---
	pgContainer, err := tcpostgres.Run(ctx, "postgres:16-alpine",
		tcpostgres.WithDatabase("testdb"),
		tcpostgres.WithUsername("test"),
		tcpostgres.WithPassword("test"),
		tcpostgres.BasicWaitStrategies(),
	)
	if err != nil {
		return nil, fmt.Errorf("start postgres container: %w", err)
	}
	f.cleanup = append(f.cleanup, func() { _ = pgContainer.Terminate(ctx) })

	pgDSN, err := pgContainer.ConnectionString(ctx, "sslmode=disable")
	if err != nil {
		return nil, fmt.Errorf("get postgres dsn: %w", err)
	}

	db, err := gorm.Open(postgres.Open(pgDSN), &gorm.Config{})
	if err != nil {
		return nil, fmt.Errorf("open gorm: %w", err)
	}
	f.DB = db

	sqlDB, err := db.DB()
	if err != nil {
		return nil, fmt.Errorf("get sql.DB: %w", err)
	}

	goose.SetLogger(goose.NopLogger())
	if err := goose.SetDialect("postgres"); err != nil {
		return nil, fmt.Errorf("goose set dialect: %w", err)
	}
	if err := goose.Up(sqlDB, filepath.Join(root, "infra", "migrations")); err != nil {
		return nil, fmt.Errorf("run postgres migrations: %w", err)
	}

	// --- ClickHouse ---
	chContainer, err := tcclickhouse.Run(ctx, "clickhouse/clickhouse-server:24-alpine")
	if err != nil {
		return nil, fmt.Errorf("start clickhouse container: %w", err)
	}
	f.cleanup = append(f.cleanup, func() { _ = chContainer.Terminate(ctx) })

	chDSN, err := chContainer.ConnectionString(ctx)
	if err != nil {
		return nil, fmt.Errorf("get clickhouse dsn: %w", err)
	}

	chDB, err := sql.Open("clickhouse", chDSN)
	if err != nil {
		return nil, fmt.Errorf("open clickhouse: %w", err)
	}
	f.ChDB = chDB
	f.cleanup = append(f.cleanup, func() { _ = chDB.Close() })

	// Run ClickHouse migrations by parsing the goose Up sections and executing them directly.
	// Goose's version tracking table requires transaction support which ClickHouse doesn't provide.
	if err := runClickHouseMigrations(ctx, chDB, filepath.Join(root, "infra", "ch_migrations")); err != nil {
		return nil, fmt.Errorf("run clickhouse migrations: %w", err)
	}

	// --- Server ---
	router := internalserver.NewRouter(db, chDB, f.JWTSecret)
	f.Server = httptest.NewServer(router)
	f.cleanup = append(f.cleanup, f.Server.Close)

	return f, nil
}

// Cleanup tears down the httptest server and all containers in reverse order.
func (f *Fixture) Cleanup() {
	for i := len(f.cleanup) - 1; i >= 0; i-- {
		f.cleanup[i]()
	}
}

// Client returns an unauthenticated API client pointing at the test server.
func (f *Fixture) Client() *apiclient.ClientWithResponses {
	c, err := apiclient.NewClientWithResponses(f.Server.URL)
	if err != nil {
		panic(fmt.Sprintf("create api client: %v", err))
	}
	return c
}

// AuthClient returns an API client that sends the given JWT as a Bearer token.
func (f *Fixture) AuthClient(token string) *apiclient.ClientWithResponses {
	c, err := apiclient.NewClientWithResponses(f.Server.URL,
		apiclient.WithRequestEditorFn(func(_ context.Context, req *http.Request) error {
			req.Header.Set("Authorization", "Bearer "+token)
			return nil
		}),
	)
	if err != nil {
		panic(fmt.Sprintf("create auth api client: %v", err))
	}
	return c
}

// CreateTestUser creates a user via the API and returns the response. Fails the test on transport error.
func (f *Fixture) CreateTestUser(t *testing.T, ctx context.Context, username, email string) *apiclient.CreateUserResponse {
	t.Helper()
	resp, err := f.Client().CreateUserWithResponse(ctx, apiclient.NewUser{
		Username: username,
		Email:    openapi_types.Email(email),
	})
	if err != nil {
		t.Fatalf("CreateTestUser: %v", err)
	}
	return resp
}

// CreateTestApiKey creates an API key for the given userID. Fails the test on transport error.
func (f *Fixture) CreateTestApiKey(t *testing.T, ctx context.Context, userID int64) *apiclient.CreateApiKeyResponse {
	t.Helper()
	resp, err := f.Client().CreateApiKeyWithResponse(ctx, apiclient.NewApiKey{
		UserId: userID,
	})
	if err != nil {
		t.Fatalf("CreateTestApiKey: %v", err)
	}
	return resp
}

// GetToken exchanges a client_id + client_secret for a JWT. Fails the test on non-200.
func (f *Fixture) GetToken(t *testing.T, ctx context.Context, clientID, clientSecret string) string {
	t.Helper()
	id, err := uuid.Parse(clientID)
	if err != nil {
		t.Fatalf("GetToken: parse client_id: %v", err)
	}
	resp, err := f.Client().CreateTokenWithResponse(ctx, apiclient.TokenRequest{
		ClientId:     openapi_types.UUID(id),
		ClientSecret: clientSecret,
		GrantType:    apiclient.ClientCredentials,
	})
	if err != nil {
		t.Fatalf("GetToken: %v", err)
	}
	if resp.StatusCode() != http.StatusOK {
		t.Fatalf("GetToken: expected 200, got %d: %s", resp.StatusCode(), string(resp.Body))
	}
	return resp.JSON200.AccessToken
}

// TruncateTables removes all application data. Call via t.Cleanup() for test isolation.
func (f *Fixture) TruncateTables(t *testing.T) {
	t.Helper()
	if err := f.DB.Exec("TRUNCATE users, items, api_keys, event_history RESTART IDENTITY CASCADE").Error; err != nil {
		t.Fatalf("TruncateTables: %v", err)
	}
	if _, err := f.ChDB.ExecContext(context.Background(), "TRUNCATE TABLE ch_events"); err != nil {
		// ClickHouse TRUNCATE may not be supported in all versions; log and continue.
		t.Logf("TruncateTables (ch_events): %v", err)
	}
}

// WaitForLastUsedAt polls until the api_key row has a non-nil last_used_at or the timeout
// expires. Used to test the async last_used_at update in the token handler.
func (f *Fixture) WaitForLastUsedAt(t *testing.T, ctx context.Context, clientID string, timeout time.Duration) *time.Time {
	t.Helper()
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		var lastUsedAt *time.Time
		err := f.DB.WithContext(ctx).
			Raw("SELECT last_used_at FROM api_keys WHERE client_id = ?", clientID).
			Scan(&lastUsedAt).Error
		if err == nil && lastUsedAt != nil {
			return lastUsedAt
		}
		time.Sleep(50 * time.Millisecond)
	}
	t.Fatalf("WaitForLastUsedAt: last_used_at not set within %s", timeout)
	return nil
}

// runClickHouseMigrations reads .sql files from dir and executes the -- +goose Up sections.
// Goose's own version tracking uses transactions which ClickHouse doesn't support.
func runClickHouseMigrations(ctx context.Context, db *sql.DB, dir string) error {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return fmt.Errorf("read migrations dir: %w", err)
	}
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".sql") {
			continue
		}
		stmts, err := parseSQLUpStatements(filepath.Join(dir, entry.Name()))
		if err != nil {
			return fmt.Errorf("parse %s: %w", entry.Name(), err)
		}
		for _, stmt := range stmts {
			if _, err := db.ExecContext(ctx, stmt); err != nil {
				return fmt.Errorf("exec %s: %w", entry.Name(), err)
			}
		}
	}
	return nil
}

// parseSQLUpStatements extracts SQL statements from the -- +goose Up section of a goose file.
func parseSQLUpStatements(path string) ([]string, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	var (
		inUp  bool
		stmts []string
		cur   strings.Builder
	)

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := scanner.Text()
		trimmed := strings.TrimSpace(line)

		if trimmed == "-- +goose Up" {
			inUp = true
			continue
		}
		if trimmed == "-- +goose Down" {
			inUp = false
			continue
		}
		if !inUp {
			continue
		}

		cur.WriteString(line)
		cur.WriteByte('\n')

		if strings.HasSuffix(trimmed, ";") {
			if stmt := strings.TrimSpace(cur.String()); stmt != "" && stmt != ";" {
				stmts = append(stmts, stmt)
			}
			cur.Reset()
		}
	}
	// Flush any trailing statement without a semicolon.
	if stmt := strings.TrimSpace(cur.String()); stmt != "" {
		stmts = append(stmts, stmt)
	}
	return stmts, scanner.Err()
}
