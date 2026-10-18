package db

import (
	"context"
	"fmt"
	"os"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"
)

const (
	allowLocalTestDatabaseEnv = "SWINGTRADE_TEST_ALLOW_LOCAL_DATABASE"
	localTestDatabaseHost     = "127.0.0.1"
	localTestDatabasePort     = 55433
	localTestDatabaseName     = "go_port_test"
	localTestDatabaseUser     = "go_port_test"
)

// integrationTestPoolConfig preserves the production dev-target guard by default.
// An explicit test-only opt-in can additionally select one fixed loopback test DB.
func integrationTestPoolConfig(databaseURL string) (*pgxpool.Config, error) {
	config, err := ReadOnlyPoolConfig(databaseURL)
	if err == nil {
		return config, nil
	}
	if os.Getenv(allowLocalTestDatabaseEnv) != "1" {
		return nil, fmt.Errorf("database URL is not an authorized integration-test target")
	}

	config, err = pgxpool.ParseConfig(databaseURL)
	if err != nil {
		return nil, fmt.Errorf("parse local integration-test database URL")
	}
	conn := config.ConnConfig
	if conn.Host != localTestDatabaseHost || conn.Port != localTestDatabasePort ||
		conn.Database != localTestDatabaseName || conn.User != localTestDatabaseUser || conn.Password != "" {
		return nil, fmt.Errorf("local integration tests require the fixed loopback-only test database")
	}
	conn.RuntimeParams["default_transaction_read_only"] = "on"
	return config, nil
}

func TestIntegrationTestPoolConfigRequiresExplicitFixedLocalTarget(t *testing.T) {
	localURL := "postgresql://go_port_test@127.0.0.1:55433/go_port_test?sslmode=disable"
	t.Setenv(allowLocalTestDatabaseEnv, "")
	if _, err := integrationTestPoolConfig(localURL); err == nil {
		t.Fatal("local target was accepted without explicit test opt-in")
	}

	t.Setenv(allowLocalTestDatabaseEnv, "1")
	config, err := integrationTestPoolConfig(localURL)
	if err != nil {
		t.Fatalf("fixed local test target was rejected: %v", err)
	}
	if got := config.ConnConfig.RuntimeParams["default_transaction_read_only"]; got != "on" {
		t.Fatalf("default_transaction_read_only = %q, want on", got)
	}

	for _, invalidURL := range []string{
		"postgresql://go_port_test@localhost:55433/go_port_test?sslmode=disable",
		"postgresql://go_port_test@127.0.0.1:55434/go_port_test?sslmode=disable",
		"postgresql://go_port_test@127.0.0.1:55433/postgres?sslmode=disable",
		"postgresql://other@127.0.0.1:55433/go_port_test?sslmode=disable",
		"postgresql://go_port_test:dummy@127.0.0.1:55433/go_port_test?sslmode=disable",
	} {
		if _, err := integrationTestPoolConfig(invalidURL); err == nil {
			t.Fatal("non-fixed local test target was accepted")
		}
	}
}

func newIntegrationTestReadOnlyPool(ctx context.Context, databaseURL string) (*pgxpool.Pool, error) {
	config, err := integrationTestPoolConfig(databaseURL)
	if err != nil {
		return nil, err
	}
	if config.ConnConfig.Host == "192.168.0.100" {
		return NewReadOnlyPool(ctx, databaseURL)
	}

	pool, err := pgxpool.NewWithConfig(ctx, config)
	if err != nil {
		return nil, fmt.Errorf("create local integration-test pool")
	}
	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("ping local integration-test database")
	}
	return pool, nil
}
