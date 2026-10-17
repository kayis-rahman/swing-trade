package db

import (
	"context"
	"os"
	"testing"
	"time"
)

func TestTestSchemaNameValidation(t *testing.T) {
	for name, valid := range map[string]bool{
		"go_port_test_0123456789abcdef":       true,
		"public":                              false,
		"go_port_test_0123456789abcdeg":       false,
		"go_port_test_0123456789abcdef_extra": false,
	} {
		if got := isTestSchemaName(name); got != valid {
			t.Errorf("isTestSchemaName(%q) = %t, want %t", name, got, valid)
		}
	}
}

func TestDevelopmentDatabaseConnectionsAreReadOnly(t *testing.T) {
	databaseURL := os.Getenv("SWINGTRADE_TEST_DATABASE_URL")
	if databaseURL == "" {
		t.Skip("SWINGTRADE_TEST_DATABASE_URL is not available; no development database connection attempted")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	pool, err := NewReadOnlyPool(ctx, databaseURL)
	if err != nil {
		t.Fatal("read-only development database connection failed")
	}
	defer pool.Close()
	var mode string
	if err := pool.QueryRow(ctx, "SHOW default_transaction_read_only").Scan(&mode); err != nil {
		t.Fatal("could not verify database read-only mode")
	}
	if mode != "on" {
		t.Fatalf("default_transaction_read_only = %q, want on", mode)
	}
}

func TestDevelopmentDatabaseWritesStayInIsolatedSchema(t *testing.T) {
	databaseURL := os.Getenv("SWINGTRADE_TEST_DATABASE_URL")
	if databaseURL == "" {
		t.Skip("SWINGTRADE_TEST_DATABASE_URL is not available; no development database connection attempted")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	schema, err := NewTestSchema(ctx, databaseURL)
	if err != nil {
		t.Fatal("could not create isolated development test schema")
	}
	t.Cleanup(func() {
		if err := schema.Close(context.Background()); err != nil {
			t.Errorf("could not drop isolated test schema %s", schema.Name)
		}
	})
	var current string
	if err := schema.Pool.QueryRow(ctx, "SELECT current_schema()").Scan(&current); err != nil {
		t.Fatal("could not inspect isolated test search_path")
	}
	if current != schema.Name {
		t.Fatalf("current schema = %q, want the isolated test schema", current)
	}
}
