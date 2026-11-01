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

	"github.com/kayis-rahman/swing-trade/go/core/clock"
	"github.com/kayis-rahman/swing-trade/go/data"
)

type fakeStrategies struct {
	rows []data.StrategyConfigEntry
	err  error
}

func (f fakeStrategies) ListStrategyConfigs(context.Context) ([]data.StrategyConfigEntry, error) {
	return f.rows, f.err
}

func TestStrategyConfigs(t *testing.T) {
	row := data.StrategyConfigEntry{ID: 1, VariantID: "alpha", Version: 2, Params: json.RawMessage(`{}`), Overlays: json.RawMessage(`{}`), PaperCapital: "100.00", Mode: "SHADOW"}
	cases := []struct {
		path          string
		store         data.StrategyConfigReadStore
		status, count int
	}{
		{"/api/strategy-configs", fakeStrategies{rows: []data.StrategyConfigEntry{row}}, 200, 1},
		{"/api/strategy-configs?mode=OFF", fakeStrategies{rows: []data.StrategyConfigEntry{row}}, 200, 0},
		{"/api/strategy-configs?variantId=other", fakeStrategies{rows: []data.StrategyConfigEntry{row}}, 200, 0},
		{"/api/strategy-configs?mode=bad", fakeStrategies{}, 400, 0},
		{"/api/strategy-configs", nil, 503, 0},
		{"/api/strategy-configs", fakeStrategies{err: errors.New("private")}, 500, 0},
	}
	for _, tc := range cases {
		mux := http.NewServeMux()
		RegisterStrategyConfigRoutes(mux, tc.store, slog.New(slog.NewTextHandler(io.Discard, nil)), clock.System{})
		w := httptest.NewRecorder()
		mux.ServeHTTP(w, httptest.NewRequest(http.MethodGet, tc.path, nil))
		if w.Code != tc.status {
			t.Fatalf("%s: got %d, want %d", tc.path, w.Code, tc.status)
		}
		if tc.status == 200 {
			var body struct {
				Success bool                       `json:"success"`
				Data    []data.StrategyConfigEntry `json:"data"`
			}
			if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil || !body.Success || len(body.Data) != tc.count {
				t.Fatalf("unexpected response %s: %v", w.Body.String(), err)
			}
		}
	}
}
