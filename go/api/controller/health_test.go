package controller

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"runtime"
	"strings"
	"testing"
	"time"

	coreclock "github.com/kayis-rahman/swing-trade/go/core/clock"
)

func TestHealthReadEndpointsPreserveLegacyWireShape(t *testing.T) {
	now := time.Date(2026, 10, 24, 12, 0, 0, 0, time.UTC)
	clk := fixedClock{now: now}
	mux := http.NewServeMux()
	RegisterHealthRoutes(mux, true, clk, now.Add(-time.Hour))

	for _, path := range []string{
		"/api/health", "/api/health/details", "/api/health/database",
		"/api/health/market-data", "/api/health/llm", "/api/health/full",
	} {
		t.Run(path, func(t *testing.T) {
			response := httptest.NewRecorder()
			mux.ServeHTTP(response, httptest.NewRequest(http.MethodGet, path, nil))
			if response.Code != http.StatusOK {
				t.Fatalf("GET %s = %d: %s", path, response.Code, response.Body.String())
			}
			var body map[string]any
			if err := json.Unmarshal(response.Body.Bytes(), &body); err != nil {
				t.Fatal(err)
			}
			if body["status"] != "UP" || body["timestamp"] != "2026-10-24T12:00:00.000000" {
				t.Fatalf("health envelope = %#v", body)
			}
			info, ok := body["systemInfo"].(map[string]any)
			if !ok {
				t.Fatalf("systemInfo missing from legacy response shape: %#v", body)
			}
			for _, key := range []string{
				"javaVersion", "osName", "osVersion", "architecture", "availableProcessors",
				"freeMemoryMB", "maxMemoryMB", "totalMemoryMB",
			} {
				if _, exists := info[key]; !exists {
					t.Errorf("legacy systemInfo field %q missing: %#v", key, info)
				}
			}
			if info["javaVersion"] != runtime.Version() || info["architecture"] != runtime.GOARCH {
				t.Errorf("runtime compatibility values = %#v", info)
			}
			if strings.Contains(response.Body.String(), "goVersion") {
				t.Errorf("wire schema was renamed: %s", response.Body.String())
			}
		})
	}
}

func TestHealthDetailsVerboseIncludesRuntimeDiagnostics(t *testing.T) {
	now := time.Date(2026, 10, 24, 12, 0, 0, 0, time.UTC)
	mux := http.NewServeMux()
	RegisterHealthRoutes(mux, false, fixedClock{now: now}, now.Add(-time.Hour))
	response := httptest.NewRecorder()
	mux.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/api/health/details?verbose=true", nil))
	var body struct {
		Components map[string]struct {
			Details map[string]any `json:"details"`
		} `json:"components"`
	}
	if err := json.Unmarshal(response.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	if body.Components["api"].Details["javaVersion"] != runtime.Version() || body.Components["api"].Details["uptime"] != "1h 0m" {
		t.Fatalf("verbose diagnostics = %#v", body.Components["api"].Details)
	}
}

func TestMemoryMegabytesConvertsAndSaturatesSafely(t *testing.T) {
	if got := memoryMegabytes(uint64(memoryMegabyte)); got != 1 {
		t.Fatalf("memoryMegabytes(1 MB) = %d, want 1", got)
	}
	if got, want := memoryMegabytes(^uint64(0)), int64(^uint64(0)/uint64(memoryMegabyte)); got != want {
		t.Fatalf("memoryMegabytes(max uint64) = %d, want %d", got, want)
	}
}

func TestHealthDatabaseReflectsStoreConfiguration(t *testing.T) {
	mux := http.NewServeMux()
	RegisterHealthRoutes(mux, false, fixedClock{now: time.Now()}, time.Now())
	response := httptest.NewRecorder()
	mux.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/api/health/database", nil))
	if !strings.Contains(response.Body.String(), `"description":"Database Service (not configured)"`) {
		t.Fatalf("unconfigured database health = %s", response.Body.String())
	}
}

var _ coreclock.Clock = fixedClock{}
