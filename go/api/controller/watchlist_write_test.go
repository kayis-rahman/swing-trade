package controller

import (
	"bytes"
	"context"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/kayis-rahman/swing-trade/go/data"
)

type fakeWatchlistWriter struct {
	addRows   data.WatchlistEntry
	err       error
	removed   string
	toggled   data.WatchlistEntry
	toggleErr error
	removeErr error
	addErr    error
}

func (f *fakeWatchlistWriter) AddWatchlist(_ context.Context, _, _, _ string) (data.WatchlistEntry, error) {
	if f.addErr != nil {
		return data.WatchlistEntry{}, f.addErr
	}
	return f.addRows, nil
}
func (f *fakeWatchlistWriter) RemoveWatchlist(_ context.Context, symbol string) error {
	f.removed = symbol
	return f.removeErr
}
func (f *fakeWatchlistWriter) ToggleWatchlist(_ context.Context, _ string, _ bool) (data.WatchlistEntry, error) {
	if f.toggleErr != nil {
		return data.WatchlistEntry{}, f.toggleErr
	}
	return f.toggled, nil
}

func TestWatchlistWrites(t *testing.T) {
	writer := &fakeWatchlistWriter{addRows: data.WatchlistEntry{ID: 1, Symbol: "TCS"}, toggled: data.WatchlistEntry{ID: 1, Symbol: "TCS"}}
	mux := http.NewServeMux()
	RegisterWatchlistWriteRoutes(mux, writer, slog.New(slog.NewTextHandler(io.Discard, nil)), fixedClock{now: time.Now()})
	// add too long symbol
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, httptest.NewRequest(http.MethodPost, "/api/watchlist?symbol=TOOLONGSYMBOL123", nil))
	if w.Code != http.StatusBadRequest {
		t.Fatalf("add long status = %d", w.Code)
	}
	w = httptest.NewRecorder()
	mux.ServeHTTP(w, httptest.NewRequest(http.MethodPost, "/api/watchlist?symbol=tcs", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("add status = %d: %s", w.Code, w.Body.String())
	}
	// remove
	w = httptest.NewRecorder()
	mux.ServeHTTP(w, httptest.NewRequest(http.MethodDelete, "/api/watchlist/TCS", nil))
	if w.Code != http.StatusOK || writer.removed != "TCS" {
		t.Fatalf("remove status = %d", w.Code)
	}
	// toggle not found
	writer2 := &fakeWatchlistWriter{toggleErr: errors.New("not found")}
	mux2 := http.NewServeMux()
	RegisterWatchlistWriteRoutes(mux2, writer2, slog.New(slog.NewTextHandler(io.Discard, nil)), fixedClock{now: time.Now()})
	w = httptest.NewRecorder()
	mux2.ServeHTTP(w, httptest.NewRequest(http.MethodPatch, "/api/watchlist/TCS/toggle?activate=true", nil))
	if w.Code != http.StatusNotFound {
		t.Fatalf("toggle status = %d", w.Code)
	}
	_ = bytes.NewBuffer
}

func TestWatchlistWritesNil(t *testing.T) {
	mux := http.NewServeMux()
	RegisterWatchlistWriteRoutes(mux, nil, slog.New(slog.NewTextHandler(io.Discard, nil)), fixedClock{now: time.Now()})
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, httptest.NewRequest(http.MethodPost, "/api/watchlist?symbol=TCS", nil))
	if w.Code != http.StatusNotFound {
		t.Fatalf("nil writer status = %d", w.Code)
	}
}
