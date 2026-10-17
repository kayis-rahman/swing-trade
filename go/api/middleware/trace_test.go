package middleware

import (
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestRequestSymbolMatchesTraceParameterAndPathFallbacks(t *testing.T) {
	for _, test := range []struct {
		url  string
		want string
	}{
		{"/api/health?symbol=TCS", "TCS"},
		{"/api/health?code=INFY", "INFY"},
		{"/api/signals/HDFCBANK", "HDFCBANK"},
		{"/api/positions/RELIANCE", "RELIANCE"},
		{"/api/signals/ABC?symbol=", ""},
		{"/api/health", ""},
	} {
		request := httptest.NewRequest(http.MethodGet, test.url, nil)
		if got := requestSymbol(request); got != test.want {
			t.Errorf("requestSymbol(%q) = %q, want %q", test.url, got, test.want)
		}
	}
}

func TestTraceAddsEightCharacterIDToContext(t *testing.T) {
	clock := fakeClock{now: time.Date(2026, 10, 17, 12, 0, 0, 0, time.UTC)}
	handler := Trace(slog.New(slog.NewTextHandler(discardWriter{}, nil)), clock)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		id := TraceID(r.Context())
		if len(id) != 8 {
			t.Errorf("trace id length = %d, want 8", len(id))
		}
		w.WriteHeader(http.StatusNoContent)
	}))
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/api/health?symbol=TCS", nil))
	if response.Code != http.StatusNoContent {
		t.Fatalf("status = %d", response.Code)
	}
}

type fakeClock struct{ now time.Time }

func (c fakeClock) Now() time.Time { return c.now }

type discardWriter struct{}

func (discardWriter) Write(p []byte) (int, error) { return len(p), nil }
