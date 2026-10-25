package api

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"sort"
	"testing"
	"time"

	coreclock "github.com/kayis-rahman/swing-trade/go/core/clock"
	"github.com/kayis-rahman/swing-trade/go/core/config"
	"github.com/kayis-rahman/swing-trade/go/data"
	"github.com/kayis-rahman/swing-trade/go/data/db"
)

func TestReadEndpointsAgainstAuthorizedDevelopmentDatabase(t *testing.T) {
	databaseURL := os.Getenv("SWINGTRADE_TEST_DATABASE_URL")
	if databaseURL == "" {
		t.Skip("authorized read-only development database URL is unavailable")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	pool, err := db.NewReadOnlyPool(ctx, databaseURL)
	if err != nil {
		t.Fatal("authorized development database connection failed")
	}
	defer pool.Close()

	cfg := config.Config{HTTPAddr: "127.0.0.1:0", PublicRateLimitPerMinute: 100}
	handler, err := NewHandlerWithStores(cfg, nil, coreclock.System{}, data.NewWatchlistReader(pool), data.NewHolidayReader(pool), data.NewSettingsReader(pool))
	if err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct {
		path   string
		active bool
	}{
		{path: "/api/holidays"},
		{path: "/api/holidays/today"},
		{path: "/api/watchlist", active: true},
		{path: "/api/watchlist/all"},
	} {
		response := httptest.NewRecorder()
		handler.ServeHTTP(response, httptest.NewRequest(http.MethodGet, tc.path, nil).WithContext(ctx))
		if response.Code != http.StatusOK {
			t.Fatalf("GET %s = %d", tc.path, response.Code)
		}
		if tc.path == "/api/holidays" || tc.path == "/api/holidays/today" {
			var body struct {
				Data    json.RawMessage `json:"data"`
				Success bool            `json:"success"`
			}
			if err := json.Unmarshal(response.Body.Bytes(), &body); err != nil {
				t.Fatal("holiday response was not valid JSON")
			}
			if !body.Success || len(body.Data) == 0 || string(body.Data) == "null" {
				t.Fatalf("GET %s returned invalid holiday data", tc.path)
			}
			continue
		}
		var body struct {
			Data []struct {
				ID       int64  `json:"id"`
				Symbol   string `json:"symbol"`
				IsActive *bool  `json:"isActive"`
			} `json:"data"`
			Success bool `json:"success"`
		}
		if err := json.Unmarshal(response.Body.Bytes(), &body); err != nil {
			t.Fatal("watchlist response was not valid JSON")
		}
		if !body.Success || body.Data == nil {
			t.Fatalf("GET %s returned an invalid API envelope", tc.path)
		}
		for i, entry := range body.Data {
			if tc.active && (entry.IsActive == nil || !*entry.IsActive) {
				t.Fatalf("active watchlist included %q with isActive=%v", entry.Symbol, entry.IsActive)
			}
			if i > 0 {
				previous := body.Data[i-1]
				ordered := entry.ID >= previous.ID
				if tc.active {
					ordered = sort.StringsAreSorted([]string{previous.Symbol, entry.Symbol})
				}
				if !ordered {
					t.Fatalf("GET %s returned out-of-order rows", tc.path)
				}
			}
		}
	}
}
