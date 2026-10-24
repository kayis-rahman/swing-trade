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

	coreclock "github.com/kayis-rahman/swing-trade/go/core/clock"
	"github.com/kayis-rahman/swing-trade/go/data"
)

type fakeWatchlistReader struct {
	rows []data.WatchlistEntry
	err  error
}

func (f fakeWatchlistReader) ListActiveWatchlist(context.Context) ([]data.WatchlistEntry, error) {
	if f.err != nil {
		return nil, f.err
	}
	rows := make([]data.WatchlistEntry, 0, len(f.rows))
	for _, row := range f.rows {
		if row.IsActive != nil && *row.IsActive {
			rows = append(rows, row)
		}
	}
	return rows, nil
}

func (f fakeWatchlistReader) ListAllWatchlist(context.Context) ([]data.WatchlistEntry, error) {
	if f.err != nil {
		return nil, f.err
	}
	return f.rows, nil
}

func TestWatchlistReadEndpointsMatchJavaGoldenShape(t *testing.T) {
	addedAt := time.Date(2026, 10, 15, 6, 49, 42, 140204000, time.UTC)
	candleCount := int32(0)
	reader := fakeWatchlistReader{rows: []data.WatchlistEntry{{
		ID: 12, Symbol: "AXISBANK", Name: stringPointer("Axis Bank Limited"), Exchange: stringPointer("NSE"),
		IsActive: boolPointer(true), AddedAt: &addedAt, CandleCount: &candleCount,
	}, {
		ID: 13, Symbol: "INACTIVE", IsActive: boolPointer(false),
	}}}
	clock := fixedClock{now: time.Date(2026, 10, 15, 7, 0, 0, 0, time.UTC)}
	mux := http.NewServeMux()
	RegisterWatchlistRoutes(mux, reader, slog.New(slog.NewTextHandler(io.Discard, nil)), clock)

	for _, tc := range []struct {
		path string
		want string
	}{
		{path: "/api/watchlist", want: `"data":[{"addedAt":"2026-10-15T06:49:42.140204","candleCount":0,"exchange":"NSE","id":12,"isActive":true,"lastSyncedAt":null,"name":"Axis Bank Limited","symbol":"AXISBANK"}]`},
		{path: "/api/watchlist/all", want: `"data":[{"addedAt":"2026-10-15T06:49:42.140204","candleCount":0,"exchange":"NSE","id":12,"isActive":true,"lastSyncedAt":null,"name":"Axis Bank Limited","symbol":"AXISBANK"},{"addedAt":null,"candleCount":null,"exchange":null,"id":13,"isActive":false,"lastSyncedAt":null,"name":null,"symbol":"INACTIVE"}]`},
	} {
		t.Run(tc.path, func(t *testing.T) {
			response := httptest.NewRecorder()
			mux.ServeHTTP(response, httptest.NewRequest(http.MethodGet, tc.path, nil))
			if response.Code != http.StatusOK {
				t.Fatalf("GET %s = %d, want 200: %s", tc.path, response.Code, response.Body.String())
			}
			if !strings.Contains(response.Header().Get("Content-Type"), "application/json") || !strings.Contains(response.Body.String(), tc.want) {
				t.Fatalf("response does not match Java golden fields: %s %s", response.Header().Get("Content-Type"), response.Body.String())
			}
			var body map[string]json.RawMessage
			if err := json.Unmarshal(response.Body.Bytes(), &body); err != nil {
				t.Fatal(err)
			}
			if string(body["success"]) != "true" || string(body["error"]) != "null" || string(body["timestamp"]) != `"2026-10-15T07:00:00.000000"` {
				t.Fatalf("envelope does not match Java ApiResponse: %s", response.Body.String())
			}
		})
	}
}

func TestEmptyWatchlistSerializesAsEmptyArray(t *testing.T) {
	mux := http.NewServeMux()
	RegisterWatchlistRoutes(mux, fakeWatchlistReader{}, slog.New(slog.NewTextHandler(io.Discard, nil)), fixedClock{now: time.Now()})
	response := httptest.NewRecorder()
	mux.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/api/watchlist", nil))
	var body map[string]json.RawMessage
	if err := json.Unmarshal(response.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	if string(body["data"]) != "[]" {
		t.Fatalf("empty collection = %s, want []", body["data"])
	}
}

func TestWatchlistReadErrorsAreSanitized(t *testing.T) {
	mux := http.NewServeMux()
	RegisterWatchlistRoutes(mux, fakeWatchlistReader{err: errors.New("secret database detail")}, slog.New(slog.NewTextHandler(io.Discard, nil)), fixedClock{now: time.Now()})
	response := httptest.NewRecorder()
	mux.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/api/watchlist", nil))
	if response.Code != http.StatusInternalServerError || strings.Contains(response.Body.String(), "secret database detail") {
		t.Fatalf("database error response = %d %s", response.Code, response.Body.String())
	}
}

func TestWatchlistReadWithoutStoreReturnsServiceUnavailable(t *testing.T) {
	mux := http.NewServeMux()
	RegisterWatchlistRoutes(mux, nil, slog.New(slog.NewTextHandler(io.Discard, nil)), fixedClock{now: time.Now()})
	response := httptest.NewRecorder()
	mux.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/api/watchlist/all", nil))
	if response.Code != http.StatusServiceUnavailable || !strings.Contains(response.Body.String(), "watchlist data store is unavailable") {
		t.Fatalf("missing-store response = %d %s", response.Code, response.Body.String())
	}
}

type fixedClock struct{ now time.Time }

func (c fixedClock) Now() time.Time { return c.now }

var _ coreclock.Clock = fixedClock{}

func stringPointer(value string) *string { return &value }
func boolPointer(value bool) *bool       { return &value }
