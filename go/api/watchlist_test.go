package api

import (
	"net/http"
	"net/http/httptest"
	"testing"

	coreclock "github.com/kayis-rahman/swing-trade/go/core/clock"
)

func TestWatchlistReadEndpointsReturnServiceUnavailableWithoutStore(t *testing.T) {
	handler, err := NewHandlerWithWatchlist(testConfig(), nil, coreclock.System{}, nil)
	if err != nil {
		t.Fatal(err)
	}
	for _, path := range []string{"/api/watchlist", "/api/watchlist/all"} {
		response := httptest.NewRecorder()
		handler.ServeHTTP(response, httptest.NewRequest(http.MethodGet, path, nil))
		if response.Code != http.StatusServiceUnavailable {
			t.Errorf("GET %s = %d, want 503", path, response.Code)
		}
	}
}
