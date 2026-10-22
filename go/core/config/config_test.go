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
	if got.APIKeyEnabled || got.APIKey != "" || got.DatabaseURL != "" {
		t.Fatalf("unexpected enabled integration in defaults: %+v", got)
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
	}
	got, err := Load(func(key string) (string, bool) { value, ok := values[key]; return value, ok })
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if got.HTTPAddr != "127.0.0.1:9090" || !got.APIKeyEnabled || got.APIKey != "local-test-key" || got.PublicRateLimitPerMinute != 7 {
		t.Fatalf("environment mapping mismatch: %+v", got)
	}
	if len(got.CORSAllowedOrigins) != 2 || got.CORSAllowedOrigins[1] != "https://dashboard.example" {
		t.Fatalf("CORS origins were not normalized: %#v", got.CORSAllowedOrigins)
	}
}

func TestLoadRejectsInvalidBoundaryConfiguration(t *testing.T) {
	for name, values := range map[string]map[string]string{
		"invalid port":            {"SERVER_PORT": "70000"},
		"invalid limit":           {"API_PUBLIC_RATE_LIMIT_PER_MINUTE": "0"},
		"missing enabled API key": {"API_KEY_AUTH_ENABLED": "true"},
		"bad boolean":             {"API_KEY_AUTH_ENABLED": "sometimes"},
	} {
		t.Run(name, func(t *testing.T) {
			_, err := Load(func(key string) (string, bool) { value, ok := values[key]; return value, ok })
			if err == nil {
				t.Fatal("Load() succeeded, want validation error")
			}
		})
	}
}
