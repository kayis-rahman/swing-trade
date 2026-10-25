package controller

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

type fakeSettingsReader struct {
	broker string
	err    error
}

func (f fakeSettingsReader) GetSelectedBroker(context.Context) (string, error) {
	return f.broker, f.err
}

func TestSettingsReadReturnsSelectedBrokerInJavaEnvelope(t *testing.T) {
	mux := http.NewServeMux()
	clk := fixedClock{now: time.Date(2026, 10, 24, 12, 1, 2, 0, time.UTC)}
	RegisterSettingsRoutes(mux, fakeSettingsReader{broker: "fyers"}, slog.New(slog.NewTextHandler(io.Discard, nil)), clk)
	response := httptest.NewRecorder()
	mux.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/api/settings", nil))
	var body struct {
		Data      map[string]string `json:"data"`
		Error     any               `json:"error"`
		Success   bool              `json:"success"`
		Timestamp string            `json:"timestamp"`
	}
	if err := json.Unmarshal(response.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	if response.Code != http.StatusOK || body.Data["selectedBroker"] != "fyers" || !body.Success || body.Error != nil || body.Timestamp != "2026-10-24T12:01:02.000000" {
		t.Fatalf("settings response = %d %#v", response.Code, body)
	}
}

func TestSettingsReadWithoutStoreReturnsServiceUnavailable(t *testing.T) {
	mux := http.NewServeMux()
	RegisterSettingsRoutes(mux, nil, nil, fixedClock{now: time.Now()})
	response := httptest.NewRecorder()
	mux.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/api/settings", nil))
	if response.Code != http.StatusServiceUnavailable {
		t.Fatalf("GET /api/settings without store = %d, want 503", response.Code)
	}
}

func TestSettingsReadFailureReturnsJavaApiEnvelope(t *testing.T) {
	mux := http.NewServeMux()
	RegisterSettingsRoutes(mux, fakeSettingsReader{err: errors.New("database unavailable")}, slog.New(slog.NewTextHandler(io.Discard, nil)), fixedClock{now: time.Now()})
	response := httptest.NewRecorder()
	mux.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/api/settings", nil))
	var body struct {
		Data    any    `json:"data"`
		Error   string `json:"error"`
		Success bool   `json:"success"`
	}
	if err := json.Unmarshal(response.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	if response.Code != http.StatusInternalServerError || body.Data != nil || body.Success || body.Error != "Unable to read settings" {
		t.Fatalf("settings error response = %d %#v", response.Code, body)
	}
}
