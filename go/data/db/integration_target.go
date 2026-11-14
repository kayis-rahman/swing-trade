package db

import (
	"fmt"
	"os"

	"github.com/jackc/pgx/v5/pgxpool"
)

const (
	localTestDatabaseHost = "127.0.0.1"
	localTestDatabasePort = 55433
	localTestDatabaseName = "go_port_test"
	localTestDatabaseUser = "go_port_test"
)

// IntegrationTestPoolConfig is for integration tests only. It preserves the
// production dev-target guard by default; the explicit
// SWINGTRADE_TEST_ALLOW_LOCAL_DATABASE=1 opt-in can additionally select the one
// fixed loopback throwaway database that bin/go-verify provisions locally.
func IntegrationTestPoolConfig(databaseURL string) (*pgxpool.Config, error) {
	config, err := ReadOnlyPoolConfig(databaseURL)
	if err == nil && config.ConnConfig.Database == "swingtrade" {
		return config, nil
	}
	if os.Getenv(localTestDatabaseEnv) != "1" {
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
