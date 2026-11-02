package db

import "testing"

func TestReadOnlyPoolConfigEnablesReadOnlyTransactions(t *testing.T) {
	config, err := ReadOnlyPoolConfig("postgres://dev-user@192.168.0.100:5433/swingtrade?sslmode=require")
	if err != nil {
		t.Fatal(err)
	}
	if got := config.ConnConfig.RuntimeParams["default_transaction_read_only"]; got != "on" {
		t.Fatalf("default_transaction_read_only = %q, want on", got)
	}
}

func TestStagePoolConfigIsReadOnly(t *testing.T) {
	config, err := ReadOnlyPoolConfig("postgres://stage-user@192.168.0.100:5433/swingtrade_stage?sslmode=require")
	if err != nil || config.ConnConfig.RuntimeParams["default_transaction_read_only"] != "on" {
		t.Fatalf("stage config = %v, error = %v", config, err)
	}
	if _, err := ReadOnlyPoolConfig("postgres://stage-user@192.168.0.100:5433/other?sslmode=require"); err == nil {
		t.Fatal("accepted non-stage database")
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

func TestPoolConfigsRejectInsecureOrOverriddenTargets(t *testing.T) {
	for _, u := range []string{
		"postgres://u@192.168.0.100:5433/swingtrade",
		"postgres://u@192.168.0.100:5433/swingtrade?sslmode=disable",
		"postgres://u@192.168.0.100:5433/swingtrade?sslmode=prefer",
		"postgres://u@192.168.0.100:5433/swingtrade?sslmode=require&host=elsewhere",
		"postgres://u@192.168.0.100:5434/swingtrade?sslmode=require",
	} {
		if _, err := ReadOnlyPoolConfig(u); err == nil {
			t.Errorf("read-only accepted %s", u)
		}
		if _, err := WritablePoolConfig(u); err == nil {
			t.Errorf("writable accepted %s", u)
		}
	}
}

func TestReadOnlyPoolConfigRejectsEmptyURL(t *testing.T) {
	if _, err := ReadOnlyPoolConfig(""); err == nil {
		t.Fatal("ReadOnlyPoolConfig() succeeded with empty URL")
	}
}
