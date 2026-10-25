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

type fakeReconciler struct {
	reports map[string]data.ReconciliationReport
	err     error
	calls   int
}

func (f *fakeReconciler) PreviewReconciliation(_ context.Context, symbol string, _, _ time.Time) (data.ReconciliationReport, error) {
	f.calls++
	if f.err != nil {
		return data.ReconciliationReport{}, f.err
	}
	return f.reports[symbol], nil
}

func reconcileRequest(path string) *http.Request {
	return httptest.NewRequest(http.MethodGet, path, nil)
}

func TestReconcileEndpointValidatesParameters(t *testing.T) {
	mux := http.NewServeMux()
	RegisterIngestionReconcileRoute(mux, fakeWatchlistReader{}, &fakeReconciler{}, discardIngestionLogger(), fixedClock{now: time.Now()})

	for name, path := range map[string]string{
		"missing dates":  "/api/ingestion/reconcile",
		"malformed date": "/api/ingestion/reconcile?from=14-09-2026&to=2026-09-18",
		"reversed":       "/api/ingestion/reconcile?from=2026-09-18&to=2026-09-14",
		"over ten days":  "/api/ingestion/reconcile?from=2026-09-01&to=2026-09-12",
	} {
		t.Run(name, func(t *testing.T) {
			response := httptest.NewRecorder()
			mux.ServeHTTP(response, reconcileRequest(path))
			if response.Code != http.StatusBadRequest {
				t.Fatalf("GET %s = %d: %s", path, response.Code, response.Body.String())
			}
			body := response.Body.String()
			if !strings.Contains(body, `"error":"VALIDATION_ERROR"`) || !strings.Contains(body, `"status":400`) {
				t.Fatalf("validation envelope = %s", body)
			}
		})
	}
}

func TestReconcileEndpointServesSingleSymbolReport(t *testing.T) {
	reconciler := &fakeReconciler{reports: map[string]data.ReconciliationReport{
		"TCS": {
			Symbol: "TCS", From: time.Date(2026, 9, 14, 0, 0, 0, 0, time.UTC),
			To:            time.Date(2026, 9, 18, 0, 0, 0, 0, time.UTC),
			Expected:      []time.Time{time.Date(2026, 9, 15, 0, 0, 0, 0, time.UTC)},
			Missing:       []time.Time{time.Date(2026, 9, 15, 0, 0, 0, 0, time.UTC)},
			QualityStatus: "MISSING", Count: 0, Status: "SOURCE_INCOMPLETE",
		},
	}}
	mux := http.NewServeMux()
	RegisterIngestionReconcileRoute(mux, fakeWatchlistReader{}, reconciler, discardIngestionLogger(), fixedClock{now: time.Now()})

	response := httptest.NewRecorder()
	mux.ServeHTTP(response, reconcileRequest("/api/ingestion/reconcile?from=2026-09-14&to=2026-09-18&symbol=tcs"))
	if response.Code != http.StatusOK {
		t.Fatalf("GET reconcile = %d: %s", response.Code, response.Body.String())
	}
	body := response.Body.String()
	for _, want := range []string{
		`"symbol":"TCS"`,
		`"from":"2026-09-14"`,
		`"to":"2026-09-18"`,
		`"expected":["2026-09-15"]`,
		`"missing":["2026-09-15"]`,
		`"nonTradingDayDates":[]`,
		`"invalidStored":[]`,
		`"qualityStatus":"MISSING"`,
		`"status":"SOURCE_INCOMPLETE"`,
		`"earliestDate":null`,
		`"success":true`,
	} {
		if !strings.Contains(body, want) {
			t.Fatalf("single report missing %s: %s", want, body)
		}
	}
	if reconciler.calls != 1 {
		t.Fatalf("reconciler calls = %d", reconciler.calls)
	}
}

func TestReconcileEndpointAggregatesWatchlistReports(t *testing.T) {
	completed := data.ReconciliationReport{
		Symbol: "TCS", From: time.Date(2026, 9, 14, 0, 0, 0, 0, time.UTC),
		To: time.Date(2026, 9, 18, 0, 0, 0, 0, time.UTC), QualityStatus: "GOOD", Status: "COMPLETED",
	}
	incomplete := data.ReconciliationReport{
		Symbol: "INFY", From: time.Date(2026, 9, 14, 0, 0, 0, 0, time.UTC),
		To: time.Date(2026, 9, 18, 0, 0, 0, 0, time.UTC), Status: "CALENDAR_INCOMPLETE",
	}
	reconciler := &fakeReconciler{reports: map[string]data.ReconciliationReport{
		"TCS": completed, "INFY": incomplete,
	}}
	watchlist := fakeWatchlistReader{rows: []data.WatchlistEntry{{Symbol: "TCS", IsActive: boolPointer(true)}, {Symbol: "INFY", IsActive: boolPointer(true)}}}
	mux := http.NewServeMux()
	RegisterIngestionReconcileRoute(mux, watchlist, reconciler, discardIngestionLogger(), fixedClock{now: time.Now()})

	response := httptest.NewRecorder()
	mux.ServeHTTP(response, reconcileRequest("/api/ingestion/reconcile?from=2026-09-14&to=2026-09-18"))
	if response.Code != http.StatusOK {
		t.Fatalf("GET reconcile = %d: %s", response.Code, response.Body.String())
	}
	body := response.Body.String()
	if !strings.Contains(body, `"reports":[`) || !strings.Contains(body, `"symbol":"TCS"`) ||
		!strings.Contains(body, `"symbol":"INFY"`) || !strings.Contains(body, `"status":"CALENDAR_INCOMPLETE"`) {
		t.Fatalf("aggregated envelope = %s", body)
	}
	if reconciler.calls != 2 {
		t.Fatalf("reconciler calls = %d, want one per active symbol", reconciler.calls)
	}
}

func TestReconcileEndpointCalendarIncompleteKeepsMinimalKeys(t *testing.T) {
	reconciler := &fakeReconciler{reports: map[string]data.ReconciliationReport{
		"INFY": {Symbol: "INFY", From: time.Date(2026, 9, 14, 0, 0, 0, 0, time.UTC),
			To: time.Date(2026, 9, 18, 0, 0, 0, 0, time.UTC), Status: "CALENDAR_INCOMPLETE"},
	}}
	mux := http.NewServeMux()
	RegisterIngestionReconcileRoute(mux, nil, reconciler, discardIngestionLogger(), fixedClock{now: time.Now()})

	response := httptest.NewRecorder()
	mux.ServeHTTP(response, reconcileRequest("/api/ingestion/reconcile?from=2026-09-14&to=2026-09-18&symbol=INFY"))
	if response.Code != http.StatusOK {
		t.Fatalf("GET reconcile = %d: %s", response.Code, response.Body.String())
	}
	body := response.Body.String()
	if !strings.Contains(body, `"status":"CALENDAR_INCOMPLETE"`) || !strings.Contains(body, `"from":"2026-09-14"`) {
		t.Fatalf("calendar-incomplete report = %s", body)
	}
	if strings.Contains(body, `"expected"`) || strings.Contains(body, `"qualityStatus"`) {
		t.Fatalf("calendar-incomplete report carries full keys: %s", body)
	}
}

func TestReconcileEndpointRequiresPreviewStore(t *testing.T) {
	mux := http.NewServeMux()
	RegisterIngestionReconcileRoute(mux, fakeWatchlistReader{}, nil, discardIngestionLogger(), fixedClock{now: time.Now()})
	response := httptest.NewRecorder()
	mux.ServeHTTP(response, reconcileRequest("/api/ingestion/reconcile?from=2026-09-14&to=2026-09-18"))
	if response.Code != http.StatusServiceUnavailable {
		t.Fatalf("missing preview store = %d: %s", response.Code, response.Body.String())
	}
}

func TestReconcileEndpointSanitizesFailures(t *testing.T) {
	mux := http.NewServeMux()
	RegisterIngestionReconcileRoute(mux, fakeWatchlistReader{},
		&fakeReconciler{err: errors.New("secret provider detail")},
		discardIngestionLogger(), fixedClock{now: time.Now()})
	response := httptest.NewRecorder()
	mux.ServeHTTP(response, reconcileRequest("/api/ingestion/reconcile?from=2026-09-14&to=2026-09-18&symbol=TCS"))
	if response.Code != http.StatusInternalServerError {
		t.Fatalf("failed preview = %d: %s", response.Code, response.Body.String())
	}
	body := response.Body.String()
	if !strings.Contains(body, "Failed to reconcile") || strings.Contains(body, "secret provider detail") {
		t.Fatalf("failure envelope = %s", body)
	}
}
