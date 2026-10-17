package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/prometheus/client_golang/prometheus"
)

func TestRequestMetricsUsesStableRoutePatternAndStatus(t *testing.T) {
	clock := testClock{now: time.Date(2026, 10, 17, 12, 0, 0, 0, time.UTC)}
	registry := prometheus.NewRegistry()
	requests := prometheus.NewHistogramVec(prometheus.HistogramOpts{Name: "http_server_requests_seconds", Help: "test"}, []string{"method", "uri", "status", "outcome", "exception"})
	apiRequests := prometheus.NewSummaryVec(prometheus.SummaryOpts{Name: "api_request_duration_seconds", Help: "test"}, []string{"method", "status"})
	if err := registry.Register(requests); err != nil {
		t.Fatal(err)
	}
	if err := registry.Register(apiRequests); err != nil {
		t.Fatal(err)
	}
	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/items/{id}", func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusNoContent) })
	handler := RequestMetrics(clock, requests, apiRequests)(mux)
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/api/items/123", nil))
	if response.Code != http.StatusNoContent {
		t.Fatalf("route status = %d", response.Code)
	}
	families, err := registry.Gather()
	if err != nil {
		t.Fatal(err)
	}
	if len(families) != 2 {
		t.Fatalf("got %d metric families, want 2", len(families))
	}
	for _, family := range families {
		switch family.GetName() {
		case "http_server_requests_seconds":
			if len(family.Metric) != 1 {
				t.Fatalf("got %d HTTP histogram series, want 1", len(family.Metric))
			}
			metric := family.Metric[0]
			labels := make(map[string]string)
			for _, label := range metric.Label {
				labels[label.GetName()] = label.GetValue()
			}
			for key, want := range map[string]string{
				"method": "GET", "uri": "/api/items/{id}", "status": "204", "outcome": "SUCCESS", "exception": "",
			} {
				if labels[key] != want {
					t.Errorf("label %q = %q, want %q", key, labels[key], want)
				}
			}
			if count := metric.GetHistogram().GetSampleCount(); count != 1 {
				t.Fatalf("HTTP histogram sample count = %d, want 1", count)
			}
		case "api_request_duration_seconds":
			if len(family.Metric) != 1 || family.Metric[0].GetSummary().GetSampleCount() != 1 {
				t.Fatalf("API filter summary did not record one sample: %v", family.Metric)
			}
		default:
			t.Errorf("unexpected metric family %q", family.GetName())
		}
	}
}

type testClock struct{ now time.Time }

func (c testClock) Now() time.Time { return c.now }
