package controller

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/kayis-rahman/swing-trade/go/data"
)

func discardIngestionLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(io.Discard, nil))
}

type fakeIngestionStatus struct {
	entries []data.IngestionStatusEntry
	err     error
}

func (f fakeIngestionStatus) IngestionStatus(context.Context) ([]data.IngestionStatusEntry, error) {
	if f.err != nil {
		return nil, f.err
	}
	return f.entries, nil
}

func TestIngestionStatusEndpointMatchesJavaEnvelopeAndFormats(t *testing.T) {
	earliest := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)
	latest := time.Date(2026, 10, 24, 0, 0, 0, 0, time.UTC)
	synced := time.Date(2026, 10, 24, 16, 30, 0, 0, time.UTC)
	status := fakeIngestionStatus{entries: []data.IngestionStatusEntry{{
		Symbol: "TCS", Name: stringPointer("Tata Consultancy"), Exchange: stringPointer("NSE"),
		IsActive: boolPointer(true), CandleCount: 1000,
		LastCandleDate: &latest, EarliestCandleDate: &earliest, LastSyncedAt: &synced,
		HasData: true, DataQuality: "excellent",
	}, {
		Symbol: "NEW", IsActive: boolPointer(false), DataQuality: "no_data",
	}}}
	mux := http.NewServeMux()
	RegisterIngestionRoutes(mux, status, discardIngestionLogger(), fixedClock{now: time.Date(2026, 10, 25, 7, 0, 0, 0, time.UTC)})

	response := httptest.NewRecorder()
	mux.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/api/ingestion/status", nil))
	if response.Code != http.StatusOK {
		t.Fatalf("GET /api/ingestion/status = %d: %s", response.Code, response.Body.String())
	}
	body := response.Body.String()
	for _, want := range []string{
		`"earliestCandleDate":"2025-01-01"`,
		`"lastCandleDate":"2026-10-24"`,
		`"candleCount":1000`,
		`"dataQuality":"excellent"`,
		`"hasData":true`,
		`"isActive":true`,
		`"lastSyncedAt":"2026-10-24T16:30:00.000000"`,
		`"name":"Tata Consultancy"`,
		`"symbol":"NEW"`,
		`"dataQuality":"no_data"`,
		`"lastCandleDate":null`,
		`"success":true`,
		`"error":null`,
		`"timestamp":"2026-10-25T07:00:00.000000"`,
	} {
		if !strings.Contains(body, want) {
			t.Fatalf("response missing %s: %s", want, body)
		}
	}
	var envelope map[string]json.RawMessage
	if err := json.Unmarshal(response.Body.Bytes(), &envelope); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(envelope["data"]), "\"symbol\":\"TCS\"") {
		t.Fatalf("data payload = %s", envelope["data"])
	}
}

func TestIngestionStatusEndpointRequiresStore(t *testing.T) {
	mux := http.NewServeMux()
	RegisterIngestionRoutes(mux, nil, discardIngestionLogger(), fixedClock{now: time.Now()})
	response := httptest.NewRecorder()
	mux.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/api/ingestion/status", nil))
	if response.Code != http.StatusServiceUnavailable || !strings.Contains(response.Body.String(), "unavailable") {
		t.Fatalf("missing-store response = %d %s", response.Code, response.Body.String())
	}
}

func TestIngestionStatusEndpointSanitizesFailures(t *testing.T) {
	mux := http.NewServeMux()
	RegisterIngestionRoutes(mux, fakeIngestionStatus{err: errors.New("secret database detail")},
		discardIngestionLogger(), fixedClock{now: time.Now()})
	response := httptest.NewRecorder()
	mux.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/api/ingestion/status", nil))
	if response.Code != http.StatusInternalServerError {
		t.Fatalf("failed read = %d: %s", response.Code, response.Body.String())
	}
	body := response.Body.String()
	if !strings.Contains(body, "Failed to get status") || strings.Contains(body, "secret database detail") {
		t.Fatalf("failure response = %s", body)
	}
}
