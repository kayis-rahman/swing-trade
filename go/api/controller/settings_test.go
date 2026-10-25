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
	"time"
)

type fakeSettingsReader struct {
	broker   string
	settings map[string]string
	err      error
}

func (f fakeSettingsReader) GetSelectedBroker(context.Context) (string, error) {
	return f.broker, f.err
}

func (f fakeSettingsReader) GetSetting(_ context.Context, key string) (string, bool, error) {
	value, found := f.settings[key]
	return value, found, f.err
}

func TestSettingsReadReturnsSelectedBrokerInJavaEnvelope(t *testing.T) {
	mux := http.NewServeMux()
	clk := fixedClock{now: time.Date(2026, 10, 24, 12, 1, 2, 0, time.UTC)}
	RegisterSettingsRoutes(mux, fakeSettingsReader{broker: "fyers"}, slog.New(slog.NewTextHandler(io.Discard, nil)), clk)
	response := httptest.NewRecorder()
	mux.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/api/settings", nil))
	var body struct {
		Data      map[string]string `json:"data"`
		Error     any               `json:"error"`
		Success   bool              `json:"success"`
		Timestamp string            `json:"timestamp"`
	}
	if err := json.Unmarshal(response.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	if response.Code != http.StatusOK || body.Data["selectedBroker"] != "fyers" || !body.Success || body.Error != nil || body.Timestamp != "2026-10-24T12:01:02.000000" {
		t.Fatalf("settings response = %d %#v", response.Code, body)
	}
}

func TestDiscordSettingsReadReturnsJavaDefaultsAndSettings(t *testing.T) {
	for _, test := range []struct {
		name     string
		settings map[string]string
		wantURL  string
		wantOn   string
	}{
		{name: "defaults", wantOn: "false"},
		{name: "configured", settings: map[string]string{"discord.webhook.url": "https://hooks.example.test/abc", "discord.webhook.enabled": "true"}, wantURL: "https://hooks.example.test/abc", wantOn: "true"},
	} {
		t.Run(test.name, func(t *testing.T) {
			mux := http.NewServeMux()
			RegisterSettingsRoutes(mux, fakeSettingsReader{settings: test.settings}, slog.New(slog.NewTextHandler(io.Discard, nil)), fixedClock{now: time.Now()})
			response := httptest.NewRecorder()
			mux.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/api/settings/discord", nil))
			var body struct {
				Data map[string]string `json:"data"`
			}
			if err := json.Unmarshal(response.Body.Bytes(), &body); err != nil {
				t.Fatal(err)
			}
			if response.Code != http.StatusOK || body.Data["discord.webhook.url"] != test.wantURL || body.Data["discord.webhook.enabled"] != test.wantOn {
				t.Fatalf("discord settings = %d %#v", response.Code, body.Data)
			}
		})
	}
}

func TestGpuHubSettingsReadExposesOnlyConfiguredFlag(t *testing.T) {
	for _, test := range []struct {
		name   string
		reader fakeSettingsReader
		env    bool
		want   string
	}{
		{name: "unset", want: "false"},
		{name: "database key", reader: fakeSettingsReader{settings: map[string]string{"gpuhub.api_key": "secret"}}, want: "true"},
		{name: "environment key", env: true, want: "true"},
	} {
		t.Run(test.name, func(t *testing.T) {
			mux := http.NewServeMux()
			RegisterSettingsRoutes(mux, test.reader, slog.New(slog.NewTextHandler(io.Discard, nil)), fixedClock{now: time.Now()}, test.env)
			response := httptest.NewRecorder()
			mux.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/api/settings/gpuhub", nil))
			var body struct {
				Data map[string]string `json:"data"`
			}
			if err := json.Unmarshal(response.Body.Bytes(), &body); err != nil {
				t.Fatal(err)
			}
			if response.Code != http.StatusOK || body.Data["gpuhub.api_key.configured"] != test.want {
				t.Fatalf("GPUHub settings = %d %#v", response.Code, body.Data)
			}
			if _, leaked := body.Data["gpuhub.api_key"]; leaked {
				t.Fatal("response exposed the GPUHub API key")
			}
		})
	}
}

func TestSettingsReadsPreserveTradingAndScanningDefaults(t *testing.T) {
	for _, test := range []struct {
		path string
		want map[string]string
	}{
		{path: "/api/settings/trading", want: map[string]string{
			"trading.mode": "paper", "trading.max_position_size": "10", "trading.stop_loss": "5",
			"trading.take_profit": "15", "trading.allocation_per_position": "100000", "trading.initial_capital": "500000",
		}},
		{path: "/api/settings/scanning", want: map[string]string{"candidate-scan.max-concurrent": "3"}},
	} {
		t.Run(test.path, func(t *testing.T) {
			mux := http.NewServeMux()
			RegisterSettingsRoutes(mux, fakeSettingsReader{}, slog.New(slog.NewTextHandler(io.Discard, nil)), fixedClock{now: time.Now()})
			response := httptest.NewRecorder()
			mux.ServeHTTP(response, httptest.NewRequest(http.MethodGet, test.path, nil))
			var body struct {
				Data map[string]string `json:"data"`
			}
			if err := json.Unmarshal(response.Body.Bytes(), &body); err != nil {
				t.Fatal(err)
			}
			if response.Code != http.StatusOK || !equalStringMap(body.Data, test.want) {
				t.Fatalf("settings %s = %d %#v", test.path, response.Code, body.Data)
			}
		})
	}
}

func TestSettingsReadWithoutStoreReturnsServiceUnavailable(t *testing.T) {
	mux := http.NewServeMux()
	RegisterSettingsRoutes(mux, nil, nil, fixedClock{now: time.Now()})
	for _, path := range []string{"/api/settings", "/api/settings/discord", "/api/settings/gpuhub", "/api/settings/trading", "/api/settings/scanning"} {
		response := httptest.NewRecorder()
		mux.ServeHTTP(response, httptest.NewRequest(http.MethodGet, path, nil))
		if response.Code != http.StatusServiceUnavailable {
			t.Errorf("GET %s without store = %d, want 503", path, response.Code)
		}
	}
}

func equalStringMap(left, right map[string]string) bool {
	if len(left) != len(right) {
		return false
	}
	for key, value := range right {
		if left[key] != value {
			return false
		}
	}
	return true
}

func TestSettingsReadFailureReturnsJavaApiEnvelope(t *testing.T) {
	mux := http.NewServeMux()
	RegisterSettingsRoutes(mux, fakeSettingsReader{err: errors.New("database unavailable")}, slog.New(slog.NewTextHandler(io.Discard, nil)), fixedClock{now: time.Now()})
	response := httptest.NewRecorder()
	mux.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/api/settings", nil))
	var body struct {
		Data    any    `json:"data"`
		Error   string `json:"error"`
		Success bool   `json:"success"`
	}
	if err := json.Unmarshal(response.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	if response.Code != http.StatusInternalServerError || body.Data != nil || body.Success || body.Error != "Unable to read settings" {
		t.Fatalf("settings error response = %d %#v", response.Code, body)
	}
}
