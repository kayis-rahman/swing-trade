package db

import "testing"

func TestReadOnlyPoolConfigEnablesReadOnlyTransactions(t *testing.T) {
	config, err := ReadOnlyPoolConfig("postgres://dev-user@192.168.0.100:5435/swingtrade_db")
	if err != nil {
		t.Fatal(err)
	}
	if got := config.ConnConfig.RuntimeParams["default_transaction_read_only"]; got != "on" {
		t.Fatalf("default_transaction_read_only = %q, want on", got)
	}
}

func TestNewReadOnlyPoolRejectsInvalidURLBeforeConnecting(t *testing.T) {
	if _, err := NewReadOnlyPool(t.Context(), "not-a-postgres-url"); err == nil {
		t.Fatal("NewReadOnlyPool() succeeded with an invalid URL")
	}
}

func TestReadOnlyPoolConfigRejectsUnauthorizedTarget(t *testing.T) {
	if _, err := ReadOnlyPoolConfig("postgres://dev-user@localhost:5435/swingtrade_db"); err == nil {
		t.Fatal("ReadOnlyPoolConfig() accepted a non-development target")
	}
}

func TestReadOnlyPoolConfigRejectsEmptyURL(t *testing.T) {
	if _, err := ReadOnlyPoolConfig(""); err == nil {
		t.Fatal("ReadOnlyPoolConfig() succeeded with empty URL")
	}
}
