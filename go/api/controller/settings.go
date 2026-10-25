package controller

import (
	"log/slog"
	"net/http"

	"github.com/kayis-rahman/swing-trade/go/core/clock"
	"github.com/kayis-rahman/swing-trade/go/data"
)

// RegisterSettingsRoutes registers the read-only selected-provider settings endpoint.
func RegisterSettingsRoutes(mux *http.ServeMux, store data.SettingsReadStore, logger *slog.Logger, clk clock.Clock) {
	mux.HandleFunc("GET /api/settings", func(w http.ResponseWriter, r *http.Request) {
		if store == nil {
			writeResponseJSON(w, http.StatusServiceUnavailable, apiEnvelope{
				Data: nil, Error: stringValuePtr("settings data store is unavailable"), Timestamp: javaLocalDateTime(clk.Now()),
			})
			return
		}
		broker, err := store.GetSelectedBroker(r.Context())
		if err != nil {
			logger.ErrorContext(r.Context(), "settings read failed", "module", "api")
			writeResponseJSON(w, http.StatusInternalServerError, apiEnvelope{
				Data: nil, Error: stringValuePtr("Unable to read settings"), Timestamp: javaLocalDateTime(clk.Now()),
			})
			return
		}
		writeResponseJSON(w, http.StatusOK, apiEnvelope{
			Data: map[string]string{"selectedBroker": broker}, Error: nil, Success: true, Timestamp: javaLocalDateTime(clk.Now()),
		})
	})
	mux.HandleFunc("GET /api/settings/discord", func(w http.ResponseWriter, r *http.Request) {
		if store == nil {
			writeResponseJSON(w, http.StatusServiceUnavailable, apiEnvelope{
				Data: nil, Error: stringValuePtr("settings data store is unavailable"), Timestamp: javaLocalDateTime(clk.Now()),
			})
			return
		}
		webhookURL, err := store.GetSetting(r.Context(), "discord.webhook.url")
		if err != nil {
			writeSettingsReadError(w, r, logger, clk)
			return
		}
		webhookEnabled, err := store.GetSetting(r.Context(), "discord.webhook.enabled")
		if err != nil {
			writeSettingsReadError(w, r, logger, clk)
			return
		}
		if webhookEnabled == "" {
			webhookEnabled = "false"
		}
		writeResponseJSON(w, http.StatusOK, apiEnvelope{
			Data:  map[string]string{"discord.webhook.url": webhookURL, "discord.webhook.enabled": webhookEnabled},
			Error: nil, Success: true, Timestamp: javaLocalDateTime(clk.Now()),
		})
	})
}

func writeSettingsReadError(w http.ResponseWriter, r *http.Request, logger *slog.Logger, clk clock.Clock) {
	logger.ErrorContext(r.Context(), "settings read failed", "module", "api")
	writeResponseJSON(w, http.StatusInternalServerError, apiEnvelope{
		Data: nil, Error: stringValuePtr("Unable to read settings"), Timestamp: javaLocalDateTime(clk.Now()),
	})
}
