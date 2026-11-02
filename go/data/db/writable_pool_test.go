package db

import (
	"testing"
)

func TestWritablePoolConfigEnablesWrites(t *testing.T) {
	config, err := WritablePoolConfig("postgres://dev-user@192.168.0.100:5435/swingtrade_db")
	if err != nil {
		t.Fatal(err)
	}
	if got := config.ConnConfig.RuntimeParams["default_transaction_read_only"]; got != "off" {
		t.Fatalf("default_transaction_read_only = %q, want off", got)
	}
}

func TestStageWritablePoolConfigIsWritable(t *testing.T) {
	config, err := WritablePoolConfig("postgres://stage-user@swing_trade_stage_postgres:5432/swingtrade_stage")
	if err != nil || config.ConnConfig.RuntimeParams["default_transaction_read_only"] != "off" {
		t.Fatalf("stage config = %v, error = %v", config, err)
	}
	if _, err := WritablePoolConfig("postgres://stage-user@swing_trade_stage_postgres:5432/swingtrade_db"); err == nil {
		t.Fatal("accepted non-stage database")
	}
}

func TestNewWritablePoolRejectsInvalidURLBeforeConnecting(t *testing.T) {
	if _, err := NewWritablePool(t.Context(), "not-a-postgres-url"); err == nil {
		t.Fatal("NewWritablePool() succeeded with an invalid URL")
	}
}

func TestWritablePoolConfigRejectsUnauthorizedTarget(t *testing.T) {
	if _, err := WritablePoolConfig("postgres://dev-user@localhost:5435/swingtrade_db"); err == nil {
		t.Fatal("WritablePoolConfig() accepted a non-development target")
	}
}

func TestWritablePoolConfigRejectsEmptyURL(t *testing.T) {
	if _, err := WritablePoolConfig(""); err == nil {
		t.Fatal("WritablePoolConfig() succeeded with empty URL")
	}
}

func TestWritableAuthorizedProductionURLStillAcceptedByConfig(t *testing.T) {
	if _, err := WritablePoolConfig(authorizedProductionURL); err != nil {
		t.Fatalf("authorized production URL rejected: %v", err)
	}
}
