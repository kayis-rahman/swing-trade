package controller

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"time"

	"github.com/kayis-rahman/swing-trade/go/core/clock"
	"github.com/kayis-rahman/swing-trade/go/data"
)

type watchlistResponse struct {
	AddedAt      *string `json:"addedAt"`
	CandleCount  *int32  `json:"candleCount"`
	Exchange     *string `json:"exchange"`
	ID           int64   `json:"id"`
	IsActive     *bool   `json:"isActive"`
	LastSyncedAt *string `json:"lastSyncedAt"`
	Name         *string `json:"name"`
	Symbol       string  `json:"symbol"`
}

type apiEnvelope struct {
	Data      any     `json:"data"`
	Error     *string `json:"error"`
	Success   bool    `json:"success"`
	Timestamp string  `json:"timestamp"`
}

// RegisterWatchlistRoutes adds the Java-compatible read-only watchlist endpoints.
func RegisterWatchlistRoutes(mux *http.ServeMux, store data.WatchlistReadStore, logger *slog.Logger, clk clock.Clock) {
	mux.HandleFunc("GET /api/watchlist", func(w http.ResponseWriter, r *http.Request) {
		serveWatchlist(w, r, store, logger, clk, false)
	})
	mux.HandleFunc("GET /api/watchlist/all", func(w http.ResponseWriter, r *http.Request) {
		serveWatchlist(w, r, store, logger, clk, true)
	})
}

func serveWatchlist(w http.ResponseWriter, r *http.Request, store data.WatchlistReadStore, logger *slog.Logger, clk clock.Clock, all bool) {
	if store == nil {
		writeResponseJSON(w, http.StatusServiceUnavailable, apiEnvelope{
			Data: nil, Error: stringValuePtr("watchlist data store is unavailable"), Timestamp: javaLocalDateTime(clk.Now()),
		})
		return
	}
	var rows []data.WatchlistEntry
	var err error
	if all {
		rows, err = store.ListAllWatchlist(r.Context())
	} else {
		rows, err = store.ListActiveWatchlist(r.Context())
	}
	if err != nil {
		logger.ErrorContext(r.Context(), "watchlist read failed", "module", "api")
		writeResponseJSON(w, http.StatusInternalServerError, apiEnvelope{
			Data: nil, Error: stringValuePtr("Unable to read watchlist"), Timestamp: javaLocalDateTime(clk.Now()),
		})
		return
	}
	response := make([]watchlistResponse, 0, len(rows))
	for _, row := range rows {
		response = append(response, watchlistResponse{
			AddedAt: formatOptionalTimestamp(row.AddedAt), CandleCount: row.CandleCount,
			Exchange: row.Exchange, ID: row.ID, IsActive: row.IsActive,
			LastSyncedAt: formatOptionalTimestamp(row.LastSyncedAt), Name: row.Name, Symbol: row.Symbol,
		})
	}
	writeResponseJSON(w, http.StatusOK, apiEnvelope{
		Data: response, Error: nil, Success: true, Timestamp: javaLocalDateTime(clk.Now()),
	})
}

func writeResponseJSON(w http.ResponseWriter, status int, body any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(body)
}

func formatOptionalTimestamp(value *time.Time) *string {
	if value == nil {
		return nil
	}
	formatted := javaLocalDateTime(*value)
	return &formatted
}

func javaLocalDateTime(value time.Time) string {
	return value.Format("2006-01-02T15:04:05.000000")
}

func stringValuePtr(value string) *string { return &value }
