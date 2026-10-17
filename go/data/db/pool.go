package db

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"
)

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
