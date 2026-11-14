package db

import "testing"

func TestIntegrationTestPoolConfigRejectsStage(t *testing.T) {
	t.Setenv(localTestDatabaseEnv, "")
	if _, err := IntegrationTestPoolConfig("postgres://stage@192.168.0.100:5433/swingtrade_stage?sslmode=require"); err == nil {
		t.Fatal("integration tests must never target stage")
	}
}

func TestIntegrationTestPoolConfigRequiresExplicitFixedLocalTarget(t *testing.T) {
	localURL := "postgresql://go_port_test@127.0.0.1:55433/go_port_test?sslmode=disable"
	t.Setenv(localTestDatabaseEnv, "")
	if _, err := IntegrationTestPoolConfig(localURL); err == nil {
		t.Fatal("local target was accepted without explicit test opt-in")
	}

	t.Setenv(localTestDatabaseEnv, "1")
	config, err := IntegrationTestPoolConfig(localURL)
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
		if _, err := IntegrationTestPoolConfig(invalidURL); err == nil {
			t.Fatal("non-fixed local test target was accepted")
		}
	}
}
