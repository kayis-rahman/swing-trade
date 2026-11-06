package controller

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	strategy "github.com/kayis-rahman/swing-trade/go/strategy"
)

func TestStrategyTypeRouteListsRegisteredTypes(t *testing.T) {
	now := time.Date(2026, 11, 6, 12, 0, 0, 0, time.UTC)
	mux := http.NewServeMux()
	RegisterStrategyTypeRoutes(mux, nil, strategy.ParamSchemaValidator{}, nil, fixedClock{now: now})
	response := httptest.NewRecorder()
	mux.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/api/strategy-types", nil))
	if response.Code != http.StatusOK {
		t.Fatalf("status = %d body=%s", response.Code, response.Body.String())
	}
	var body struct {
		Data []struct {
			Type string `json:"type"`
		} `json:"data"`
	}
	if err := json.Unmarshal(response.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	if len(body.Data) < 6 {
		t.Fatalf("types = %d, want at least 6: %#v", len(body.Data), body.Data)
	}
}
