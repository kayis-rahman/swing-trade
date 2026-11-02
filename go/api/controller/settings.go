package controller

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"math/big"
	"net/http"
	"strconv"
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

// RegisterSettingsWriteRoutes registers settings write endpoints behind a configured writer.
func RegisterSettingsWriteRoutes(mux *http.ServeMux, writer data.SettingsWriteStore, logger *slog.Logger, clk clock.Clock) {
	if writer == nil {
		return
	}
	mux.HandleFunc("POST /api/settings/broker", func(w http.ResponseWriter, r *http.Request) {
		body, err := decodeStringMap(r)
		if err != nil {
			writeResponseJSON(w, http.StatusBadRequest, apiEnvelope{
				Data: nil, Error: stringValuePtr("invalid request body"), Timestamp: javaLocalDateTime(clk.Now()),
			})
			return
		}
		broker := body["broker"]
		if strings.TrimSpace(broker) == "" {
			writeResponseJSON(w, http.StatusBadRequest, apiEnvelope{
				Data: nil, Error: stringValuePtr("broker is required"), Timestamp: javaLocalDateTime(clk.Now()),
			})
			return
		}
		if err := writer.SetSetting(r.Context(), "selectedBroker", broker); err != nil {
			logger.ErrorContext(r.Context(), "settings write failed", "module", "api", "key", "selectedBroker", "error", err)
			writeResponseJSON(w, http.StatusInternalServerError, apiEnvelope{
				Data: nil, Error: stringValuePtr("Unable to write settings"), Timestamp: javaLocalDateTime(clk.Now()),
			})
			return
		}
		writeResponseJSON(w, http.StatusOK, apiEnvelope{
			Data: map[string]string{"selectedBroker": broker}, Error: nil, Success: true, Timestamp: javaLocalDateTime(clk.Now()),
		})
	})
	mux.HandleFunc("PUT /api/settings/trading", func(w http.ResponseWriter, r *http.Request) {
		body, err := decodeStringMap(r)
		if err != nil {
			writeResponseJSON(w, http.StatusBadRequest, apiEnvelope{
				Data: nil, Error: stringValuePtr("invalid request body"), Timestamp: javaLocalDateTime(clk.Now()),
			})
			return
		}
		if raw, ok := body["trading.initial_capital"]; ok {
			if !validPositiveDecimal(raw) {
				writeResponseJSON(w, http.StatusBadRequest, apiEnvelope{
					Data: nil, Error: stringValuePtr("Initial capital must be greater than zero"), Timestamp: javaLocalDateTime(clk.Now()),
				})
				return
			}
		}
		for key, value := range body {
			if err := writer.SetSetting(r.Context(), key, value); err != nil {
				logger.ErrorContext(r.Context(), "settings write failed", "module", "api", "key", key, "error", err)
				writeResponseJSON(w, http.StatusInternalServerError, apiEnvelope{
					Data: nil, Error: stringValuePtr("Unable to write settings"), Timestamp: javaLocalDateTime(clk.Now()),
				})
				return
			}
		}
		writeResponseJSON(w, http.StatusOK, apiEnvelope{
			Data: body, Error: nil, Success: true, Timestamp: javaLocalDateTime(clk.Now()),
		})
	})
	mux.HandleFunc("PUT /api/settings/scanning", func(w http.ResponseWriter, r *http.Request) {
		body, err := decodeStringMap(r)
		if err != nil {
			writeResponseJSON(w, http.StatusBadRequest, apiEnvelope{
				Data: nil, Error: stringValuePtr("invalid request body"), Timestamp: javaLocalDateTime(clk.Now()),
			})
			return
		}
		raw := body["candidate-scan.max-concurrent"]
		workers, parseErr := parsePositiveInt(raw)
		if parseErr != nil || workers < 1 || workers > 12 {
			writeResponseJSON(w, http.StatusBadRequest, apiEnvelope{
				Data: nil, Error: stringValuePtr("Candidate scan workers must be between 1 and 12"), Timestamp: javaLocalDateTime(clk.Now()),
			})
			return
		}
		if err := writer.SetSetting(r.Context(), "candidate-scan.max-concurrent", raw); err != nil {
			logger.ErrorContext(r.Context(), "settings write failed", "module", "api", "key", "candidate-scan.max-concurrent", "error", err)
			writeResponseJSON(w, http.StatusInternalServerError, apiEnvelope{
				Data: nil, Error: stringValuePtr("Unable to write settings"), Timestamp: javaLocalDateTime(clk.Now()),
			})
			return
		}
		writeResponseJSON(w, http.StatusOK, apiEnvelope{
			Data: map[string]string{"candidate-scan.max-concurrent": raw}, Error: nil, Success: true, Timestamp: javaLocalDateTime(clk.Now()),
		})
	})
	mux.HandleFunc("PUT /api/settings/discord", func(w http.ResponseWriter, r *http.Request) {
		body, err := decodeStringMap(r)
		if err != nil {
			writeResponseJSON(w, http.StatusBadRequest, apiEnvelope{
				Data: nil, Error: stringValuePtr("invalid request body"), Timestamp: javaLocalDateTime(clk.Now()),
			})
			return
		}
		for key, value := range body {
			if err := writer.SetSetting(r.Context(), key, value); err != nil {
				logger.ErrorContext(r.Context(), "settings write failed", "module", "api", "key", key, "error", err)
				writeResponseJSON(w, http.StatusInternalServerError, apiEnvelope{
					Data: nil, Error: stringValuePtr("Unable to write settings"), Timestamp: javaLocalDateTime(clk.Now()),
				})
				return
			}
		}
		writeResponseJSON(w, http.StatusOK, apiEnvelope{
			Data: body, Error: nil, Success: true, Timestamp: javaLocalDateTime(clk.Now()),
		})
	})
}

func decodeStringMap(r *http.Request) (map[string]string, error) {
	decoder := json.NewDecoder(r.Body)
	var raw map[string]any
	if err := decoder.Decode(&raw); err != nil {
		return nil, err
	}
	result := make(map[string]string, len(raw))
	for key, value := range raw {
		if value == nil {
			continue
		}
		result[key] = fmt.Sprint(value)
	}
	return result, nil
}

func validPositiveDecimal(raw string) bool {
	for _, r := range raw {
		allowed := (r >= '0' && r <= '9') || r == '.' || r == '+' || r == '-' || r == 'e' || r == 'E'
		if !allowed {
			return false
		}
	}
	n, ok := new(big.Float).SetString(raw)
	if !ok || n.IsInf() {
		return false
	}
	return n.Sign() > 0
}

func parsePositiveInt(raw string) (int, error) {
	i, err := strconv.Atoi(strings.TrimSpace(raw))
	if err != nil {
		return 0, err
	}
	return i, nil
}
