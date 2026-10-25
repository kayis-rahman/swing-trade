package config

import "testing"

func TestLoadUsesSafeLocalDefaults(t *testing.T) {
	got, err := Load(func(string) (string, bool) { return "", false })
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if got.HTTPAddr != "127.0.0.1:8081" || got.PublicRateLimitPerMinute != 100 {
		t.Fatalf("unexpected defaults: %+v", got)
	}
	if got.APIKeyEnabled || got.APIKey != "" || got.DatabaseURL != "" || got.GPUHubAPIKeyConfigured {
		t.Fatalf("unexpected enabled integration in defaults: %+v", got)
	}
	if got.EODEnabled || got.BenchmarkBackfillOnStartup || got.BenchmarkBackfillYears != 10 || got.FyersSymbolMasterRefreshEnabled {
		t.Fatalf("Go ingestion jobs must default off with a 10-year benchmark window: %+v", got)
	}
}

func TestLoadMapsExistingJavaEnvironmentProperties(t *testing.T) {
	values := map[string]string{
		"SERVER_PORT":                      "9090",
		"API_KEY_AUTH_ENABLED":             "true",
		"API_KEY":                          "local-test-key",
		"API_PUBLIC_RATE_LIMIT_PER_MINUTE": "7",
		"CORS_ALLOWED_ORIGINS":             "http://localhost:3000, https://dashboard.example",
		"DATABASE_URL":                     "postgres://localhost/test",
		"GPUHUB_API_KEY":                   " configured-secret ",
	}
	got, err := Load(func(key string) (string, bool) { value, ok := values[key]; return value, ok })
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if got.HTTPAddr != "127.0.0.1:9090" || !got.APIKeyEnabled || got.APIKey != "local-test-key" || got.PublicRateLimitPerMinute != 7 || !got.GPUHubAPIKeyConfigured {
		t.Fatalf("environment mapping mismatch: %+v", got)
	}
	if len(got.CORSAllowedOrigins) != 2 || got.CORSAllowedOrigins[1] != "https://dashboard.example" {
		t.Fatalf("CORS origins were not normalized: %#v", got.CORSAllowedOrigins)
	}
}

func TestLoadMapsIngestionJobSettings(t *testing.T) {
	values := map[string]string{
		"EOD_INGESTION_ENABLED":               "true",
		"BENCHMARK_BACKFILL_ON_STARTUP":       "true",
		"BENCHMARK_BACKFILL_YEARS":            "7",
		"FYERS_SYMBOL_MASTER_REFRESH_ENABLED": "true",
	}
	got, err := Load(func(key string) (string, bool) { value, ok := values[key]; return value, ok })
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if !got.EODEnabled || !got.BenchmarkBackfillOnStartup || got.BenchmarkBackfillYears != 7 || !got.FyersSymbolMasterRefreshEnabled {
		t.Fatalf("ingestion job settings mismatch: %+v", got)
	}
}

func TestLoadRejectsInvalidBoundaryConfiguration(t *testing.T) {
	for name, values := range map[string]map[string]string{
		"invalid port":            {"SERVER_PORT": "70000"},
		"invalid limit":           {"API_PUBLIC_RATE_LIMIT_PER_MINUTE": "0"},
		"missing enabled API key": {"API_KEY_AUTH_ENABLED": "true"},
		"bad boolean":             {"API_KEY_AUTH_ENABLED": "sometimes"},
		"bad eod boolean":         {"EOD_INGESTION_ENABLED": "later"},
		"bad backfill boolean":    {"BENCHMARK_BACKFILL_ON_STARTUP": "maybe"},
		"bad backfill years":      {"BENCHMARK_BACKFILL_YEARS": "0"},
		"bad symbol master flag":  {"FYERS_SYMBOL_MASTER_REFRESH_ENABLED": "sometimes"},
	} {
		t.Run(name, func(t *testing.T) {
			_, err := Load(func(key string) (string, bool) { value, ok := values[key]; return value, ok })
			if err == nil {
				t.Fatal("Load() succeeded, want validation error")
			}
		})
	}
}
