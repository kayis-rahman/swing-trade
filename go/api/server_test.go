package api

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/kayis-rahman/swing-trade/go/api/middleware"
	"github.com/kayis-rahman/swing-trade/go/core/config"
)

func TestInfrastructureRoutes(t *testing.T) {
	handler := handlerFor(t, testConfig())
	for _, path := range []string{
		"/api/health",
		"/api/health/details?verbose=true",
		"/api/health/database",
		"/api/health/market-data",
		"/api/health/llm",
		"/api/health/full",
		"/api/admin/health",
		"/actuator/health",
		"/actuator/health/liveness",
		"/actuator/health/readiness",
		"/actuator/prometheus",
	} {
		t.Run(path, func(t *testing.T) {
			response := httptest.NewRecorder()
			handler.ServeHTTP(response, httptest.NewRequest(http.MethodGet, path, nil))
			if response.Code != http.StatusOK {
				t.Fatalf("GET %s = %d, want 200: %s", path, response.Code, response.Body.String())
			}
			if path != "/actuator/prometheus" && !strings.Contains(response.Header().Get("Content-Type"), "application/json") {
				t.Fatalf("GET %s Content-Type = %q, want JSON", path, response.Header().Get("Content-Type"))
			}
		})
	}
}

func TestPublicRateLimitHasJavaCompatibleHeaders(t *testing.T) {
	cfg := testConfig()
	cfg.PublicRateLimitPerMinute = 1
	handler := handlerFor(t, cfg)
	for i, wantStatus := range []int{http.StatusOK, http.StatusTooManyRequests} {
		response := httptest.NewRecorder()
		request := httptest.NewRequest(http.MethodGet, "/api/health", nil)
		request.RemoteAddr = "192.0.2.10:1234"
		handler.ServeHTTP(response, request)
		if response.Code != wantStatus {
			t.Fatalf("request %d status = %d, want %d", i+1, response.Code, wantStatus)
		}
		if response.Header().Get("X-RateLimit-Limit") != "1" {
			t.Fatalf("X-RateLimit-Limit = %q", response.Header().Get("X-RateLimit-Limit"))
		}
	}
	response := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/api/health", nil)
	request.RemoteAddr = "192.0.2.10:1234"
	handler.ServeHTTP(response, request)
	if response.Header().Get("Retry-After") == "" || response.Header().Get("X-RateLimit-Remaining") != "0" {
		t.Fatalf("missing 429 headers: %#v", response.Header())
	}
}

func TestAdminKeyFailsClosedAndAuthorizedKeyPasses(t *testing.T) {
	cfg := testConfig()
	cfg.APIKeyEnabled = true
	cfg.APIKey = "test-only-key"
	handler := handlerFor(t, cfg)
	request := httptest.NewRequest(http.MethodGet, "/api/admin/health", nil)
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	if response.Code != http.StatusUnauthorized || response.Header().Get("WWW-Authenticate") != "X-API-Key" {
		t.Fatalf("missing-key response = %d %v", response.Code, response.Header())
	}
	request = httptest.NewRequest(http.MethodGet, "/api/admin/health", nil)
	request.Header.Set("X-API-Key", cfg.APIKey)
	response = httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	if response.Code != http.StatusOK {
		t.Fatalf("authorized request status = %d", response.Code)
	}
}

func TestCorsPreflightBypassesAuthenticationAndRateLimit(t *testing.T) {
	cfg := testConfig()
	cfg.APIKeyEnabled = true
	cfg.APIKey = "test-only-key"
	cfg.PublicRateLimitPerMinute = 1
	handler := handlerFor(t, cfg)
	request := httptest.NewRequest(http.MethodOptions, "/api/admin/health", nil)
	request.Header.Set("Origin", "http://localhost:3000")
	request.Header.Set("Access-Control-Request-Method", "GET")
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	if response.Code != http.StatusNoContent {
		t.Fatalf("OPTIONS status = %d, want 204", response.Code)
	}
	if response.Header().Get("Access-Control-Allow-Origin") != "http://localhost:3000" {
		t.Fatalf("CORS origin = %q", response.Header().Get("Access-Control-Allow-Origin"))
	}
}

func TestMetricsEndpointUsesPrometheusText(t *testing.T) {
	handler := handlerFor(t, testConfig())
	handler.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(http.MethodGet, "/api/health", nil))
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/actuator/prometheus", nil))
	if response.Code != http.StatusOK || !strings.Contains(response.Header().Get("Content-Type"), "text/plain") {
		t.Fatalf("metrics response = %d %q", response.Code, response.Header().Get("Content-Type"))
	}
	for _, series := range []string{"http_server_requests_seconds_count", "api_request_duration_seconds_count", `application="swing-trade-api"`} {
		if !strings.Contains(response.Body.String(), series) {
			t.Fatalf("series %q missing from scrape: %s", series, response.Body.String())
		}
	}
}

func TestCorsRejectsUnconfiguredOrigin(t *testing.T) {
	handler := handlerFor(t, testConfig())
	request := httptest.NewRequest(http.MethodGet, "/api/health", nil)
	request.Header.Set("Origin", "https://not-allowed.example")
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	if response.Header().Get("Access-Control-Allow-Origin") != "" {
		t.Fatalf("unexpected CORS origin %q", response.Header().Get("Access-Control-Allow-Origin"))
	}
}

func TestRecoverMiddlewareReturnsSanitizedInternalError(t *testing.T) {
	handler := middleware.Recover(nil)(http.HandlerFunc(func(http.ResponseWriter, *http.Request) { panic("secret internal detail") }))
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/test", nil))
	if response.Code != http.StatusInternalServerError || strings.Contains(response.Body.String(), "secret") {
		t.Fatalf("panic response leaked or had wrong status: %d %s", response.Code, response.Body.String())
	}
}

func handlerFor(t *testing.T, cfg config.Config) http.Handler {
	t.Helper()
	handler, err := NewHandler(cfg)
	if err != nil {
		t.Fatalf("NewHandler() error = %v", err)
	}
	return handler
}

func testConfig() config.Config {
	return config.Config{
		HTTPAddr:                 "127.0.0.1:0",
		PublicRateLimitPerMinute: 100,
		CORSAllowedOrigins:       []string{"http://localhost:3000"},
	}
}
