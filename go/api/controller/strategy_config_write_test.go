package controller

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/kayis-rahman/swing-trade/go/data"
)

type fakeStrategyConfigWriter struct {
	entry data.StrategyConfigEntry
	err   error
}

func (f fakeStrategyConfigWriter) SaveStrategyConfig(_ context.Context, _ data.StrategyConfigEntry, _ int64) (data.StrategyConfigEntry, error) {
	if f.err != nil {
		return data.StrategyConfigEntry{}, f.err
	}
	return f.entry, nil
}

func TestStrategyConfigWritesConflict(t *testing.T) {
	writer := fakeStrategyConfigWriter{err: data.ErrOptimisticLock}
	mux := http.NewServeMux()
	RegisterStrategyConfigWriteRoutes(mux, writer, slog.New(slog.NewTextHandler(io.Discard, nil)), fixedClock{now: time.Now()})
	body := bytes.NewBufferString(`{"lockVersion":1,"mode":"SHADOW"}`)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, httptest.NewRequest(http.MethodPut, "/api/strategy-configs/alpha", body))
	if w.Code != http.StatusConflict {
		t.Fatalf("status = %d: %s", w.Code, w.Body.String())
	}
	// success
	writer2 := fakeStrategyConfigWriter{entry: data.StrategyConfigEntry{VariantID: "alpha", Mode: "SHADOW"}}
	mux2 := http.NewServeMux()
	RegisterStrategyConfigWriteRoutes(mux2, writer2, slog.New(slog.NewTextHandler(io.Discard, nil)), fixedClock{now: time.Now()})
	w2 := httptest.NewRecorder()
	mux2.ServeHTTP(w2, httptest.NewRequest(http.MethodPut, "/api/strategy-configs/alpha", bytes.NewBufferString(`{"lockVersion":1,"mode":"SHADOW"}`)))
	if w2.Code != http.StatusOK {
		t.Fatalf("status = %d: %s", w2.Code, w2.Body.String())
	}
	_ = json.Marshal
	_ = errors.New
}

func TestStrategyConfigWritesNil(t *testing.T) {
	mux := http.NewServeMux()
	RegisterStrategyConfigWriteRoutes(mux, nil, slog.New(slog.NewTextHandler(io.Discard, nil)), fixedClock{now: time.Now()})
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, httptest.NewRequest(http.MethodPut, "/api/strategy-configs/alpha", bytes.NewBufferString(`{}`)))
	if w.Code != http.StatusNotFound {
		t.Fatalf("status = %d", w.Code)
	}
}
