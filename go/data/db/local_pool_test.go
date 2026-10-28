package db

import (
	"context"
	"strings"
	"testing"
)

const (
	authorizedProductionURL = "postgresql://dev-user@192.168.0.100:5435/swingtrade_db"
	validLocalURL           = "postgresql://go_port_test@127.0.0.1:55433/go_port_test?sslmode=disable"
)

// An unauthorized/invalid production URL fails validation before any network
// call, which lets these unit tests exercise the fallback branch without
// depending on whether the development or local database is running.
const alwaysFailingProductionURL = "not-a-postgres-url"

func TestNewLocalOrReadOnlyPoolTriesProductionFirst(t *testing.T) {
	ctx := context.Background()
	t.Setenv(localTestDatabaseEnv, "")
	t.Setenv("SWINGTRADE_LOCAL_TEST_DATABASE_URL", "")
	_, err := NewLocalOrReadOnlyPool(ctx, alwaysFailingProductionURL)
	if err == nil {
		t.Fatal("NewLocalOrReadOnlyPool() succeeded without opt-in")
	}
	if !strings.Contains(err.Error(), "connect to database") {
		t.Fatalf("error = %q, want it to wrap the production failure", err)
	}
}

func TestNewLocalOrReadOnlyPoolUsesLocalFallbackWhenOptedIn(t *testing.T) {
	ctx := context.Background()
	productionURL := alwaysFailingProductionURL

	t.Setenv(localTestDatabaseEnv, "")
	t.Setenv("SWINGTRADE_LOCAL_TEST_DATABASE_URL", "")
	if _, err := NewLocalOrReadOnlyPool(ctx, productionURL); err == nil {
		t.Fatal("NewLocalOrReadOnlyPool() succeeded without opt-in")
	} else if !strings.Contains(err.Error(), "connect to database") {
		t.Fatalf("error = %q, want the production failure without opt-in", err)
	}

	// With opt-in, the production failure hands off to the local URL parser.
	t.Setenv(localTestDatabaseEnv, "1")
	t.Setenv("SWINGTRADE_LOCAL_TEST_DATABASE_URL", "not-a-postgres-url")
	if _, err := NewLocalOrReadOnlyPool(ctx, productionURL); err == nil {
		t.Fatal("NewLocalOrReadOnlyPool() succeeded with an invalid local fallback URL")
	} else if !strings.Contains(err.Error(), "parse local fallback database URL") {
		t.Fatalf("error = %q, want the local fallback parse failure", err)
	}
}

func TestNewLocalOrReadOnlyPoolReturnsPoolForValidLocalTarget(t *testing.T) {
	// A valid local URL must pass validation. Whether it then connects depends
	// on the local database being up, so accept either a pool or a connection
	// error, but never a validation error.
	t.Setenv(localTestDatabaseEnv, "1")
	t.Setenv("SWINGTRADE_LOCAL_TEST_DATABASE_URL", validLocalURL)
	pool, err := NewLocalOrReadOnlyPool(context.Background(), alwaysFailingProductionURL)
	if err != nil {
		if strings.Contains(err.Error(), "local fallback database URL must target") {
			t.Fatalf("valid local URL was rejected by validation: %v", err)
		}
		return
	}
	pool.Close()
}

func TestNewLocalOrReadOnlyPoolRejectsNonValidLocalTarget(t *testing.T) {
	t.Setenv(localTestDatabaseEnv, "1")
	t.Setenv("SWINGTRADE_LOCAL_TEST_DATABASE_URL",
		"postgresql://other@localhost:55433/go_port_test?sslmode=disable")
	_, err := NewLocalOrReadOnlyPool(context.Background(), alwaysFailingProductionURL)
	if err == nil {
		t.Fatal("NewLocalOrReadOnlyPool() accepted non-valid local fallback URL")
	}
	expectedMsg := "local fallback database URL must target 127.0.0.1:55433/go_port_test as user go_port_test with no password"
	if !strings.Contains(err.Error(), expectedMsg) {
		t.Fatalf("error message = %q, want to contain %q", err, expectedMsg)
	}
}

// Keeps the authorized production URL in the test surface so a future change
// to the allow-list is visible here.
func TestAuthorizedProductionURLStillAcceptedByConfig(t *testing.T) {
	if _, err := ReadOnlyPoolConfig(authorizedProductionURL); err != nil {
		t.Fatalf("authorized production URL rejected: %v", err)
	}
}
