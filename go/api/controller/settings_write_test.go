package controller

import (
	"bytes"
	"context"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

type fakeSettingsWriter struct {
	values map[string]string
	err    error
}

func (f *fakeSettingsWriter) SetSetting(_ context.Context, key, value string) error {
	if f.err != nil {
		return f.err
	}
	if f.values == nil {
		f.values = map[string]string{}
	}
	f.values[key] = value
	return nil
}

func TestSettingsBrokerWritePersistsSelectedBroker(t *testing.T) {
	writer := &fakeSettingsWriter{}
	mux := http.NewServeMux()
	RegisterSettingsWriteRoutes(mux, writer, slog.New(slog.NewTextHandler(io.Discard, nil)), fixedClock{now: time.Now()})
	response := httptest.NewRecorder()
	mux.ServeHTTP(response, httptest.NewRequest(http.MethodPost, "/api/settings/broker", bytes.NewBufferString(`{"broker":"yahoo"}`)))
	if response.Code != http.StatusOK || writer.values["selectedBroker"] != "yahoo" {
		t.Fatalf("broker write = %d %#v", response.Code, writer.values)
	}
}

func TestSettingsBrokerWriteRequiresBroker(t *testing.T) {
	writer := &fakeSettingsWriter{}
	mux := http.NewServeMux()
	RegisterSettingsWriteRoutes(mux, writer, slog.New(slog.NewTextHandler(io.Discard, nil)), fixedClock{now: time.Now()})
	response := httptest.NewRecorder()
	mux.ServeHTTP(response, httptest.NewRequest(http.MethodPost, "/api/settings/broker", bytes.NewBufferString(`{}`)))
	if response.Code != http.StatusBadRequest {
		t.Fatalf("broker write status = %d", response.Code)
	}
}

func TestSettingsWritesPropagateFailure(t *testing.T) {
	writer := &fakeSettingsWriter{err: errors.New("boom")}
	mux := http.NewServeMux()
	RegisterSettingsWriteRoutes(mux, writer, slog.New(slog.NewTextHandler(io.Discard, nil)), fixedClock{now: time.Now()})
	response := httptest.NewRecorder()
	mux.ServeHTTP(response, httptest.NewRequest(http.MethodPost, "/api/settings/broker", bytes.NewBufferString(`{"broker":"fyers"}`)))
	if response.Code != http.StatusInternalServerError {
		t.Fatalf("broker write status = %d", response.Code)
	}
}

func TestSettingsTradingWriteValidatesInitialCapital(t *testing.T) {
	writer := &fakeSettingsWriter{}
	mux := http.NewServeMux()
	RegisterSettingsWriteRoutes(mux, writer, slog.New(slog.NewTextHandler(io.Discard, nil)), fixedClock{now: time.Now()})
	response := httptest.NewRecorder()
	mux.ServeHTTP(response, httptest.NewRequest(http.MethodPut, "/api/settings/trading", bytes.NewBufferString(`{"trading.initial_capital":"0"}`)))
	if response.Code != http.StatusBadRequest {
		t.Fatalf("trading validation status = %d", response.Code)
	}
}

func TestSettingsScanningWriteValidatesWorkers(t *testing.T) {
	writer := &fakeSettingsWriter{}
	mux := http.NewServeMux()
	RegisterSettingsWriteRoutes(mux, writer, slog.New(slog.NewTextHandler(io.Discard, nil)), fixedClock{now: time.Now()})
	response := httptest.NewRecorder()
	mux.ServeHTTP(response, httptest.NewRequest(http.MethodPut, "/api/settings/scanning", bytes.NewBufferString(`{"candidate-scan.max-concurrent":"20"}`)))
	if response.Code != http.StatusBadRequest {
		t.Fatalf("scanning validation status = %d", response.Code)
	}
	badBody := bytes.NewBufferString(`not json`)
	response = httptest.NewRecorder()
	mux.ServeHTTP(response, httptest.NewRequest(http.MethodPut, "/api/settings/scanning", badBody))
	if response.Code != http.StatusBadRequest {
		t.Fatalf("scanning invalid json status = %d", response.Code)
	}
}

func TestSettingsWriteRoutesNoOpWhenWriterNil(t *testing.T) {
	mux := http.NewServeMux()
	RegisterSettingsWriteRoutes(mux, nil, slog.New(slog.NewTextHandler(io.Discard, nil)), fixedClock{now: time.Now()})
	response := httptest.NewRecorder()
	mux.ServeHTTP(response, httptest.NewRequest(http.MethodPost, "/api/settings/broker", bytes.NewBufferString(`{"broker":"x"}`)))
	if response.Code != http.StatusNotFound {
		t.Fatalf("nil writer status = %d", response.Code)
	}
}

func TestSettingsDiscordWritePersistsValues(t *testing.T) {
	writer := &fakeSettingsWriter{}
	mux := http.NewServeMux()
	RegisterSettingsWriteRoutes(mux, writer, slog.New(slog.NewTextHandler(io.Discard, nil)), fixedClock{now: time.Now()})
	payload := `{"discord.webhook.url":"https://hooks.example.test/abc","discord.webhook.enabled":"true"}`
	response := httptest.NewRecorder()
	mux.ServeHTTP(response, httptest.NewRequest(http.MethodPut, "/api/settings/discord", strings.NewReader(payload)))
	if response.Code != http.StatusOK {
		t.Fatalf("discord write status = %d: %s", response.Code, response.Body.String())
	}
	if writer.values["discord.webhook.url"] != "https://hooks.example.test/abc" {
		t.Fatalf("discord values = %#v", writer.values)
	}
}

func TestDecodeStringMapNilValueSkipped(t *testing.T) {
	req := httptest.NewRequest(http.MethodPut, "/api/settings/discord", bytes.NewBufferString(`{"discord.webhook.url":null}`))
	m, err := decodeStringMap(req)
	if err != nil || len(m) != 0 {
		t.Fatalf("decode = %#v %v", m, err)
	}
}

func TestValidPositiveDecimalAndParseIntHelpers(t *testing.T) {
	if !validPositiveDecimal("") || !validPositiveDecimal("1.5") || validPositiveDecimal("abc") || validPositiveDecimal("-1") {
		t.Fatal("validPositiveDecimal mismatch")
	}
	if v, err := parsePositiveInt(" 3 "); err != nil || v != 3 {
		t.Fatalf("parsePositiveInt = %d %v", v, err)
	}
	if _, err := parsePositiveInt("x"); err == nil {
		t.Fatal("parsePositiveInt accepted invalid")
	}
}
