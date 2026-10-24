package controller

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestHealthEndpointComponentSetsMatchJavaSurface(t *testing.T) {
	now := time.Date(2026, 10, 24, 12, 0, 0, 0, time.UTC)
	mux := http.NewServeMux()
	RegisterHealthRoutes(mux, true, fixedClock{now: now}, now)
	cases := []struct {
		path       string
		components []string
	}{
		{path: "/api/health", components: []string{"api"}},
		{path: "/api/health/details", components: []string{"api", "data", "strategy", "llm"}},
		{path: "/api/health/database", components: []string{"database"}},
		{path: "/api/health/market-data", components: []string{"market-data"}},
		{path: "/api/health/llm", components: []string{"llm"}},
		{path: "/api/health/full", components: []string{"api", "database", "market-data", "strategy", "llm"}},
	}
	for _, tc := range cases {
		t.Run(tc.path, func(t *testing.T) {
			response := httptest.NewRecorder()
			mux.ServeHTTP(response, httptest.NewRequest(http.MethodGet, tc.path, nil))
			var body struct {
				Components map[string]struct {
					Name           string         `json:"name"`
					Status         string         `json:"status"`
					ResponseTimeMS *int64         `json:"responseTimeMs"`
					Details        map[string]any `json:"details"`
				} `json:"components"`
			}
			if err := json.Unmarshal(response.Body.Bytes(), &body); err != nil {
				t.Fatal(err)
			}
			if len(body.Components) != len(tc.components) {
				t.Fatalf("component set = %#v", body.Components)
			}
			for _, name := range tc.components {
				component, ok := body.Components[name]
				if !ok || component.Name != name || component.Status != "UP" || component.ResponseTimeMS != nil || component.Details == nil {
					t.Errorf("component %q = %#v, present=%v", name, component, ok)
				}
			}
		})
	}
}
