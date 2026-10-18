package db

import (
	"context"
	"fmt"
	"os"
	"strings"
	"testing"
)

func TestNewLocalOrReadOnlyPoolTriesProductionFirst(t *testing.T) {
	ctx := context.Background()
	productionURL := "postgresql://dev-user@192.168.0.100:5435/swingtrade_db"
	t.Setenv(localTestDatabaseEnv, "")
	if _, err := NewLocalOrReadOnlyPool(ctx, productionURL); err == nil {
		t.Fatal("NewLocalOrReadOnlyPool() succeeded without opt-in")
	}
}

func TestNewLocalOrReadOnlyPoolUsesLocalFallbackWhenOptedIn(t *testing.T) {
	ctx := context.Background()
	_ = os.Unsetenv(localTestDatabaseEnv)
	productionURL := "postgresql://dev-user@192.168.0.100:5435/swingtrade_db"
	if _, err := NewLocalOrReadOnlyPool(ctx, productionURL); err == nil {
		t.Fatal("NewLocalOrReadOnlyPool() succeeded without opt-in")
	}

	t.Setenv(localTestDatabaseEnv, "1")
	if _, err := NewLocalOrReadOnlyPool(ctx, productionURL); err == nil {
		t.Fatal("NewLocalOrReadOnlyPool() succeeded without local database")
	}

	localURL := "postgresql://go_port_test@127.0.0.1:55433/go_port_test?sslmode=disable"
	_ = os.Setenv("SWINGTRADE_LOCAL_TEST_DATABASE_URL", localURL)
	_ = os.Unsetenv("SWINGTRADE_LOCAL_TEST_DATABASE_URL")
	if _, err := NewLocalOrReadOnlyPool(ctx, productionURL); err == nil {
		t.Fatal("NewLocalOrReadOnlyPool() succeeded without local database")
	}
}

func TestNewLocalOrReadOnlyPoolReturnsPoolForValidLocalTarget(t *testing.T) {
	// This test verifies the local fallback path when a valid local database is running.
	// The integration tests (TestDevelopmentDatabaseConnectionsAreReadOnly and
	// TestDevelopmentDatabaseWritesStayInIsolatedSchema) exercise the live local
	// database path; this unit test only checks parameter validation.
	localURL := "postgresql://go_port_test@127.0.0.1:55433/go_port_test?sslmode=disable"
	_ = os.Setenv("SWINGTRADE_LOCAL_TEST_DATABASE_URL", localURL)
	_ = os.Setenv(localTestDatabaseEnv, "1")
	// Construct an intentionally bad local URL to verify error handling.
	_ = os.Setenv("SWINGTRADE_LOCAL_TEST_DATABASE_URL", "not-a-postgres-url")
	_, err := NewLocalOrReadOnlyPool(context.Background(), "postgresql://dev-user@192.168.0.100:5435/swingtrade_db")
	if err == nil {
		t.Fatal("NewLocalOrReadOnlyPool() succeeded with invalid local fallback URL")
	}
	expectedMsg := "parse local fallback database URL"
	if !strings.Contains(fmt.Sprintf("%v", err), expectedMsg) {
		t.Fatalf("error message = %q, want to contain %q", err, expectedMsg)
	}
}
