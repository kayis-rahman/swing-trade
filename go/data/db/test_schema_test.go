package db

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

// TestSchema owns one isolated schema and the pool whose search_path is restricted to it.
type TestSchema struct {
	Name  string
	Pool  *pgxpool.Pool
	admin *pgxpool.Pool
}

// NewTestSchema creates a unique schema for an integration test. The returned pool cannot
// resolve objects from public because its search_path contains only the generated schema.
func NewTestSchema(ctx context.Context, databaseURL string) (*TestSchema, error) {
	config, err := integrationTestPoolConfig(databaseURL)
	if err != nil {
		return nil, fmt.Errorf("parse authorized test database URL")
	}
	config.ConnConfig.RuntimeParams["default_transaction_read_only"] = "off"
	admin, err := pgxpool.NewWithConfig(ctx, config)
	if err != nil {
		return nil, fmt.Errorf("connect to test database: %w", err)
	}
	if err := admin.Ping(ctx); err != nil {
		admin.Close()
		return nil, fmt.Errorf("ping test database: %w", err)
	}

	var random [8]byte
	if _, err := rand.Read(random[:]); err != nil {
		admin.Close()
		return nil, fmt.Errorf("generate isolated schema name: %w", err)
	}
	name := "go_port_test_" + hex.EncodeToString(random[:])
	if !isTestSchemaName(name) {
		admin.Close()
		return nil, fmt.Errorf("generated invalid test schema name")
	}
	if _, err := admin.Exec(ctx, "CREATE SCHEMA "+name); err != nil {
		admin.Close()
		return nil, fmt.Errorf("create isolated test schema: %w", err)
	}

	testConfig, err := integrationTestPoolConfig(databaseURL)
	if err != nil {
		_ = dropSchema(ctx, admin, name)
		admin.Close()
		return nil, fmt.Errorf("parse authorized test database URL")
	}
	testConfig.ConnConfig.RuntimeParams["default_transaction_read_only"] = "off"
	testConfig.ConnConfig.RuntimeParams["search_path"] = name
	pool, err := pgxpool.NewWithConfig(ctx, testConfig)
	if err != nil {
		_ = dropSchema(ctx, admin, name)
		admin.Close()
		return nil, fmt.Errorf("connect to isolated test schema: %w", err)
	}
	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		_ = dropSchema(ctx, admin, name)
		admin.Close()
		return nil, fmt.Errorf("ping isolated test schema: %w", err)
	}
	return &TestSchema{Name: name, Pool: pool, admin: admin}, nil
}

// Close drops only the schema created by NewTestSchema and closes both pools.
func (s *TestSchema) Close(ctx context.Context) error {
	if s == nil {
		return nil
	}
	if s.Pool != nil {
		s.Pool.Close()
	}
	var err error
	if s.admin != nil {
		if isTestSchemaName(s.Name) {
			err = dropSchema(ctx, s.admin, s.Name)
		} else {
			err = fmt.Errorf("refusing to drop non-test schema")
		}
		s.admin.Close()
	}
	return err
}

func isTestSchemaName(name string) bool {
	const prefix = "go_port_test_"
	if !strings.HasPrefix(name, prefix) || len(name) != len(prefix)+16 {
		return false
	}
	_, err := hex.DecodeString(strings.TrimPrefix(name, prefix))
	return err == nil
}

func dropSchema(ctx context.Context, pool *pgxpool.Pool, name string) error {
	ctx, cancel := context.WithTimeout(context.WithoutCancel(ctx), 5*time.Second)
	defer cancel()
	if !isTestSchemaName(name) {
		return fmt.Errorf("refusing to drop non-test schema")
	}
	_, err := pool.Exec(ctx, "DROP SCHEMA "+name+" CASCADE")
	if err != nil {
		return fmt.Errorf("drop isolated test schema: %w", err)
	}
	return nil
}
