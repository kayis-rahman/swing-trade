package db

import (
	"context"
	"fmt"
	"os"

	"github.com/jackc/pgx/v5/pgxpool"
)

const localTestDatabaseEnv = "SWINGTRADE_TEST_ALLOW_LOCAL_DATABASE"

// ReadOnlyPoolConfig parses a PostgreSQL URL and makes every implicit transaction read-only.
func ReadOnlyPoolConfig(databaseURL string) (*pgxpool.Config, error) {
	if databaseURL == "" {
		return nil, fmt.Errorf("DATABASE_URL is required")
	}
	config, err := pgxpool.ParseConfig(databaseURL)
	if err != nil {
		return nil, fmt.Errorf("parse DATABASE_URL: invalid connection string")
	}
	if config.ConnConfig.Host != "192.168.0.100" || config.ConnConfig.Port != 5435 || config.ConnConfig.Database != "swingtrade_db" {
		return nil, fmt.Errorf("DATABASE_URL must target the authorized development database")
	}
	config.ConnConfig.RuntimeParams["default_transaction_read_only"] = "on"
	return config, nil
}

// NewReadOnlyPool creates a read-only pool. The caller owns and must close it.
func NewReadOnlyPool(ctx context.Context, databaseURL string) (*pgxpool.Pool, error) {
	config, err := ReadOnlyPoolConfig(databaseURL)
	if err != nil {
		return nil, err
	}
	pool, err := pgxpool.NewWithConfig(ctx, config)
	if err != nil {
		return nil, fmt.Errorf("create PostgreSQL pool: invalid configuration")
	}
	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("ping PostgreSQL: configured endpoint unavailable")
	}
	return pool, nil
}

// NewLocalOrReadOnlyPool attempts to connect to the configured endpoint first.
// When SWINGTRADE_TEST_ALLOW_LOCAL_DATABASE=1 and the endpoint is unreachable,
// it falls back to a local test database on 127.0.0.1:55433.
func NewLocalOrReadOnlyPool(ctx context.Context, databaseURL string) (*pgxpool.Pool, error) {
	pool, err := NewReadOnlyPool(ctx, databaseURL)
	if err == nil {
		return pool, nil
	}
	if os.Getenv(localTestDatabaseEnv) != "1" {
		return nil, fmt.Errorf("connect to database: %w", err)
	}
	localURL := os.Getenv("SWINGTRADE_LOCAL_TEST_DATABASE_URL")
	if localURL == "" {
		localURL = "postgresql://go_port_test@127.0.0.1:55433/go_port_test?sslmode=disable"
	}
	config, err := pgxpool.ParseConfig(localURL)
	if err != nil {
		return nil, fmt.Errorf("parse local fallback database URL: %w", err)
	}
	if config.ConnConfig.Host != "127.0.0.1" || config.ConnConfig.Port != 55433 ||
		config.ConnConfig.Database != "go_port_test" || config.ConnConfig.User != "go_port_test" ||
		config.ConnConfig.Password != "" {
		return nil, fmt.Errorf("local fallback database URL must target 127.0.0.1:55433/go_port_test as user go_port_test with no password")
	}
	config.ConnConfig.RuntimeParams["default_transaction_read_only"] = "on"
	pool, err = pgxpool.NewWithConfig(ctx, config)
	if err != nil {
		return nil, fmt.Errorf("create local fallback pool: %w", err)
	}
	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("ping local fallback database: %w", err)
	}
	return pool, nil
}
