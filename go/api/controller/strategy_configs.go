package controller

import (
	"log/slog"
	"net/http"
	"strings"

	"github.com/kayis-rahman/swing-trade/go/core/clock"
	"github.com/kayis-rahman/swing-trade/go/data"
)

// RegisterStrategyConfigRoutes registers the read-only configuration list.
func RegisterStrategyConfigRoutes(mux *http.ServeMux, store data.StrategyConfigReadStore, logger *slog.Logger, clk clock.Clock) {
	mux.HandleFunc("GET /api/strategy-configs", func(w http.ResponseWriter, r *http.Request) {
		mode := r.URL.Query().Get("mode")
		if mode != "" && mode != "OFF" && mode != "BACKTEST_ONLY" && mode != "SHADOW" && mode != "CHAMPION" {
			writeResponseJSON(w, http.StatusBadRequest, apiEnvelope{Error: stringValuePtr("Invalid strategy mode"), Timestamp: javaLocalDateTime(clk.Now())})
			return
		}
		if store == nil {
			writeResponseJSON(w, http.StatusServiceUnavailable, apiEnvelope{Error: stringValuePtr("strategy config data store is unavailable"), Timestamp: javaLocalDateTime(clk.Now())})
			return
		}
		rows, err := store.ListStrategyConfigs(r.Context())
		if err != nil {
			logger.ErrorContext(r.Context(), "strategy config read failed", "module", "api")
			writeResponseJSON(w, http.StatusInternalServerError, apiEnvelope{Error: stringValuePtr("Unable to read strategy configs"), Timestamp: javaLocalDateTime(clk.Now())})
			return
		}
		filtered := make([]data.StrategyConfigEntry, 0, len(rows))
		variant := r.URL.Query().Get("variantId")
		for _, row := range rows {
			if (strings.TrimSpace(variant) == "" || row.VariantID == variant) && (mode == "" || row.Mode == mode) {
				filtered = append(filtered, row)
			}
		}
		writeResponseJSON(w, http.StatusOK, apiEnvelope{Success: true, Data: filtered, Timestamp: javaLocalDateTime(clk.Now())})
	})
}
