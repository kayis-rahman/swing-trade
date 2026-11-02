package db

import (
	"context"
	"strings"
	"testing"
)

func TestNewLocalOrWritablePoolTriesProductionFirst(t *testing.T) {
	ctx := context.Background()
	t.Setenv(localTestDatabaseEnv, "")
	t.Setenv("SWINGTRADE_LOCAL_TEST_DATABASE_URL", "")
	_, err := NewLocalOrWritablePool(ctx, alwaysFailingProductionURL)
	if err == nil {
		t.Fatal("NewLocalOrWritablePool() succeeded without opt-in")
	}
	if !strings.Contains(err.Error(), "connect to database") {
		t.Fatalf("error = %q, want it to wrap the production failure", err)
	}
}

func TestNewLocalOrWritablePoolUsesLocalFallbackWhenOptedIn(t *testing.T) {
	ctx := context.Background()
	productionURL := alwaysFailingProductionURL

	t.Setenv(localTestDatabaseEnv, "")
	t.Setenv("SWINGTRADE_LOCAL_TEST_DATABASE_URL", "")
	if _, err := NewLocalOrWritablePool(ctx, productionURL); err == nil {
		t.Fatal("NewLocalOrWritablePool() succeeded without opt-in")
	} else if !strings.Contains(err.Error(), "connect to database") {
		t.Fatalf("error = %q, want the production failure without opt-in", err)
	}

	t.Setenv(localTestDatabaseEnv, "1")
	t.Setenv("SWINGTRADE_LOCAL_TEST_DATABASE_URL", "not-a-postgres-url")
	if _, err := NewLocalOrWritablePool(ctx, productionURL); err == nil {
		t.Fatal("NewLocalOrWritablePool() succeeded with an invalid local fallback URL")
	} else if !strings.Contains(err.Error(), "parse local fallback database URL") {
		t.Fatalf("error = %q, want the local fallback parse failure", err)
	}
}

func TestNewLocalOrWritablePoolReturnsPoolForValidLocalTarget(t *testing.T) {
	t.Setenv(localTestDatabaseEnv, "1")
	t.Setenv("SWINGTRADE_LOCAL_TEST_DATABASE_URL", validLocalURL)
	pool, err := NewLocalOrWritablePool(context.Background(), alwaysFailingProductionURL)
	if err != nil {
		if strings.Contains(err.Error(), "local fallback database URL must target") {
			t.Fatalf("valid local URL was rejected by validation: %v", err)
		}
		return
	}
	pool.Close()
}

func TestNewLocalOrWritablePoolRejectsNonValidLocalTarget(t *testing.T) {
	t.Setenv(localTestDatabaseEnv, "1")
	t.Setenv("SWINGTRADE_LOCAL_TEST_DATABASE_URL",
		"postgresql://other@localhost:55433/go_port_test?sslmode=disable")
	_, err := NewLocalOrWritablePool(context.Background(), alwaysFailingProductionURL)
	if err == nil {
		t.Fatal("NewLocalOrWritablePool() accepted non-valid local fallback URL")
	}
	expectedMsg := "local fallback database URL must target 127.0.0.1:55433/go_port_test as user go_port_test with no password"
	if !strings.Contains(err.Error(), expectedMsg) {
		t.Fatalf("error message = %q, want to contain %q", err, expectedMsg)
	}
}
