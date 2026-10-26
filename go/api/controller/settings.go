package controller

import (
	"log/slog"
	"net/http"
	"strings"

	"github.com/kayis-rahman/swing-trade/go/core/clock"
	"github.com/kayis-rahman/swing-trade/go/data"
)

// RegisterSettingsRoutes registers read-only settings endpoints.
func RegisterSettingsRoutes(mux *http.ServeMux, store data.SettingsReadStore, logger *slog.Logger, clk clock.Clock, environmentGPUHubKeyConfigured ...bool) {
	gpuHubKeyConfiguredFromEnvironment := len(environmentGPUHubKeyConfigured) > 0 && environmentGPUHubKeyConfigured[0]
	mux.HandleFunc("GET /api/settings", func(w http.ResponseWriter, r *http.Request) {
		if unavailable(w, store, clk) {
			return
		}
		broker, err := store.GetSelectedBroker(r.Context())
		if err != nil {
			writeSettingsReadError(w, r, logger, clk)
			return
		}
		writeSettingsResponse(w, clk, map[string]string{"selectedBroker": broker})
	})
	mux.HandleFunc("GET /api/settings/discord", func(w http.ResponseWriter, r *http.Request) {
		if unavailable(w, store, clk) {
			return
		}
		settings, err := readSettings(r, store, map[string]string{
			"discord.webhook.url": "", "discord.webhook.enabled": "false",
		})
		if err != nil {
			writeSettingsReadError(w, r, logger, clk)
			return
		}
		writeSettingsResponse(w, clk, settings)
	})
	mux.HandleFunc("GET /api/settings/gpuhub", func(w http.ResponseWriter, r *http.Request) {
		if unavailable(w, store, clk) {
			return
		}
		configured := gpuHubKeyConfiguredFromEnvironment
		if !configured {
			key, found, err := store.GetSetting(r.Context(), "gpuhub.api_key")
			if err != nil {
				writeSettingsReadError(w, r, logger, clk)
				return
			}
			configured = found && strings.TrimSpace(key) != ""
		}
		value := "false"
		if configured {
			value = "true"
		}
		writeSettingsResponse(w, clk, map[string]string{"gpuhub.api_key.configured": value})
	})
	mux.HandleFunc("GET /api/settings/trading", func(w http.ResponseWriter, r *http.Request) {
		if unavailable(w, store, clk) {
			return
		}
		settings, err := readSettings(r, store, map[string]string{
			"trading.mode": "paper", "trading.max_position_size": "10", "trading.stop_loss": "5",
			"trading.take_profit": "15", "trading.allocation_per_position": "100000", "trading.initial_capital": "500000",
		})
		if err != nil {
			writeSettingsReadError(w, r, logger, clk)
			return
		}
		writeSettingsResponse(w, clk, settings)
	})
	mux.HandleFunc("GET /api/settings/scanning", func(w http.ResponseWriter, r *http.Request) {
		if unavailable(w, store, clk) {
			return
		}
		settings, err := readSettings(r, store, map[string]string{"candidate-scan.max-concurrent": "3"})
		if err != nil {
			writeSettingsReadError(w, r, logger, clk)
			return
		}
		writeSettingsResponse(w, clk, settings)
	})
}

func unavailable(w http.ResponseWriter, store data.SettingsReadStore, clk clock.Clock) bool {
	if store != nil {
		return false
	}
	writeResponseJSON(w, http.StatusServiceUnavailable, apiEnvelope{
		Data: nil, Error: stringValuePtr("settings data store is unavailable"), Timestamp: javaLocalDateTime(clk.Now()),
	})
	return true
}

func readSettings(r *http.Request, store data.SettingsReadStore, defaults map[string]string) (map[string]string, error) {
	settings := make(map[string]string, len(defaults))
	for key, defaultValue := range defaults {
		value, found, err := store.GetSetting(r.Context(), key)
		if err != nil {
			return nil, err
		}
		if !found {
			value = defaultValue
		}
		settings[key] = value
	}
	return settings, nil
}

func writeSettingsResponse(w http.ResponseWriter, clk clock.Clock, settings map[string]string) {
	writeResponseJSON(w, http.StatusOK, apiEnvelope{
		Data: settings, Error: nil, Success: true, Timestamp: javaLocalDateTime(clk.Now()),
	})
}

func writeSettingsReadError(w http.ResponseWriter, r *http.Request, logger *slog.Logger, clk clock.Clock) {
	logger.ErrorContext(r.Context(), "settings read failed", "module", "api")
	writeResponseJSON(w, http.StatusInternalServerError, apiEnvelope{
		Data: nil, Error: stringValuePtr("Unable to read settings"), Timestamp: javaLocalDateTime(clk.Now()),
	})
}
