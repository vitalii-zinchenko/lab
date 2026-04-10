package handlers_test

import (
	"context"
	"fmt"
	"log"
	"os"
	"testing"

	"github.com/vitaliizinchenko/lab/internal/shared/testinfra"
)

var fixture *testinfra.Fixture

func TestMain(m *testing.M) {
	ctx := context.Background()

	f, err := setupFixture(ctx)
	if err != nil {
		log.Printf("skipping integration tests: %v", err)
		os.Exit(0)
	}
	fixture = f

	code := m.Run()
	fixture.Cleanup()
	os.Exit(code)
}

func setupFixture(ctx context.Context) (f *testinfra.Fixture, err error) {
	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("fixture panic: %v", r)
		}
	}()
	return testinfra.NewFixture(ctx)
}
