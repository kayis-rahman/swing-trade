package controller

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"strings"

	"github.com/kayis-rahman/swing-trade/go/core/clock"
	"github.com/kayis-rahman/swing-trade/go/data"
)

// StrategyConfigWriteStore persists strategy configuration writes with optimistic locking.
type StrategyConfigWriteStore interface {
	SaveStrategyConfig(ctx context.Context, entry data.StrategyConfigEntry, lockVersion int64) (data.StrategyConfigEntry, error)
}

// RegisterStrategyConfigWriteRoutes registers strategy configuration write endpoints.
func RegisterStrategyConfigWriteRoutes(mux *http.ServeMux, writer StrategyConfigWriteStore, logger *slog.Logger, clk clock.Clock) {
	if writer == nil {
		return
	}
	if logger == nil {
		logger = slog.New(slog.NewTextHandler(discardWriter{}, nil))
	}
	mux.HandleFunc("PUT /api/strategy-configs/{variantId}", func(w http.ResponseWriter, r *http.Request) {
		variantID := strings.TrimSpace(r.PathValue("variantId"))
		var req struct {
			LockVersion int64  `json:"lockVersion"`
			Mode        string `json:"mode"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeResponseJSON(w, http.StatusBadRequest, apiEnvelope{Error: stringValuePtr("invalid request body"), Timestamp: javaLocalDateTime(clk.Now())})
			return
		}
		entry := data.StrategyConfigEntry{VariantID: variantID, Mode: req.Mode}
		saved, err := writer.SaveStrategyConfig(r.Context(), entry, req.LockVersion)
		if err != nil {
			if errors.Is(err, data.ErrOptimisticLock) {
				writeResponseJSON(w, http.StatusConflict, apiEnvelope{Error: stringValuePtr("optimistic lock conflict"), Timestamp: javaLocalDateTime(clk.Now())})
				return
			}
			logger.ErrorContext(r.Context(), "strategy config write failed", "module", "api", "error", err)
			writeResponseJSON(w, http.StatusInternalServerError, apiEnvelope{Error: stringValuePtr("Unable to write strategy config"), Timestamp: javaLocalDateTime(clk.Now())})
			return
		}
		writeResponseJSON(w, http.StatusOK, apiEnvelope{Success: true, Data: saved, Timestamp: javaLocalDateTime(clk.Now())})
	})
}

type discardWriter struct{}

func (discardWriter) Write(p []byte) (int, error) { return len(p), nil }
