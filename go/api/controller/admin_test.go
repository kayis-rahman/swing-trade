package controller

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

type fakeSymbolMaster struct {
	count     int
	err       error
	refreshed bool
}

func (f *fakeSymbolMaster) Refresh(_ context.Context) (int, error) {
	f.refreshed = true
	if f.err != nil {
		return 0, f.err
	}
	return f.count, nil
}
func (f *fakeSymbolMaster) Count(_ context.Context) (int, error) {
	if f.err != nil {
		return 0, f.err
	}
	return f.count, nil
}

type fakeValidator struct {
	report any
	err    error
}

func (f fakeValidator) ValidateDataQuality(_ context.Context, _ string, _, _ time.Time) (any, error) {
	if f.err != nil {
		return nil, f.err
	}
	return f.report, nil
}

func TestAdminSymbolsRefreshAndStatus(t *testing.T) {
	master := &fakeSymbolMaster{count: 42}
	mux := http.NewServeMux()
	RegisterAdminRoutes(mux, master, fakeValidator{}, slog.New(slog.NewTextHandler(io.Discard, nil)), fixedClock{now: time.Now()})
	for _, tc := range []struct {
		method, path string
		want         int
	}{{"POST", "/api/admin/symbols/refresh", 200}, {"GET", "/api/admin/symbols/status", 200}} {
		w := httptest.NewRecorder()
		mux.ServeHTTP(w, httptest.NewRequest(tc.method, tc.path, nil))
		if w.Code != tc.want {
			t.Fatalf("%s %s = %d", tc.method, tc.path, w.Code)
		}
	}
}

func TestAdminDataValidate(t *testing.T) {
	mux := http.NewServeMux()
	RegisterAdminRoutes(mux, &fakeSymbolMaster{}, fakeValidator{report: map[string]any{"ok": true}}, slog.New(slog.NewTextHandler(io.Discard, nil)), fixedClock{now: time.Now()})
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, httptest.NewRequest(http.MethodPost, "/api/admin/data/validate", bytes.NewBufferString(`{"symbol":" tcs ","fromDate":"2026-01-01","toDate":"2026-01-31"}`)))
	if w.Code != http.StatusOK {
		t.Fatalf("status = %d: %s", w.Code, w.Body.String())
	}
	bad := httptest.NewRecorder()
	mux.ServeHTTP(bad, httptest.NewRequest(http.MethodPost, "/api/admin/data/validate", bytes.NewBufferString(`{"symbol":"TCS","fromDate":"2026-02-01","toDate":"2026-01-31"}`)))
	if bad.Code != http.StatusBadRequest {
		t.Fatalf("bad status = %d", bad.Code)
	}
}

func TestAdminNilDependencies(t *testing.T) {
	mux := http.NewServeMux()
	RegisterAdminRoutes(mux, nil, nil, nil, fixedClock{now: time.Now()})
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, httptest.NewRequest(http.MethodPost, "/api/admin/symbols/refresh", nil))
	if w.Code != http.StatusServiceUnavailable {
		t.Fatalf("status = %d", w.Code)
	}
	_ = json.NewEncoder
}
