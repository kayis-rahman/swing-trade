package controller

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/kayis-rahman/swing-trade/go/data"
)

type fakeSignalReader struct {
	rows []data.SignalEntry
	err  error
}

func (f fakeSignalReader) ListLatestSignals(context.Context) ([]data.SignalEntry, error) {
	if f.err != nil {
		return nil, f.err
	}
	return f.rows, nil
}

func signalFixtureRows() []data.SignalEntry {
	date := time.Date(2026, 10, 27, 0, 0, 0, 0, time.UTC)
	confidence := "0.75"
	reasoning := "All entry rules passed"
	entry := "123.4500"
	stop := "118.0000"
	target := "134.3500"
	riskReward := "2.00"
	indicators := "ema20,rsi"
	sentimentScore := "NEUTRAL"
	sentimentReasoning := "No news"
	return []data.SignalEntry{{
		ID: 42, Symbol: "INFY", Date: date, SignalType: "BUY", Confidence: &confidence,
		Reasoning: &reasoning, EntryPrice: &entry, StopLoss: &stop, Target: &target,
		RiskRewardRatio: &riskReward, Indicators: &indicators, GeneratedAt: &date,
		Strategy: "PULLBACK", StrategyVersion: 3, SentimentScore: &sentimentScore,
		SentimentReasoning: &sentimentReasoning,
	}, {
		ID: 43, Symbol: "TCS", Date: date, SignalType: "HOLD", Strategy: "DEFAULT", StrategyVersion: 1,
	}}
}

func serveSignalsLatest(t *testing.T, store data.SignalReadStore) *httptest.ResponseRecorder {
	t.Helper()
	clock := fixedClock{now: time.Date(2026, 10, 27, 0, 0, 0, 0, time.UTC)}
	mux := http.NewServeMux()
	RegisterSignalRoutes(mux, store, slog.New(slog.NewTextHandler(io.Discard, nil)), clock)
	response := httptest.NewRecorder()
	mux.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/api/signals/latest", nil))
	return response
}

func TestSignalsLatestMatchesJavaDirectArrayShape(t *testing.T) {
	response := serveSignalsLatest(t, fakeSignalReader{rows: signalFixtureRows()})
	if response.Code != http.StatusOK {
		t.Fatalf("GET /api/signals/latest = %d: %s", response.Code, response.Body.String())
	}
	if !strings.Contains(response.Header().Get("Content-Type"), "application/json") {
		t.Fatalf("content type = %q", response.Header().Get("Content-Type"))
	}
	var body []map[string]any
	if err := json.Unmarshal(response.Body.Bytes(), &body); err != nil {
		t.Fatalf("response is not a direct JSON array: %v: %s", err, response.Body.String())
	}
	if len(body) != 2 {
		t.Fatalf("len = %d, want 2", len(body))
	}
	assertPopulatedSignal(t, body[0])
	assertEmptySignal(t, body[1])
}

func assertPopulatedSignal(t *testing.T, signal map[string]any) {
	t.Helper()
	for _, key := range []string{
		"id", "symbol", "date", "signalType", "confidence", "reasoning", "entryPrice",
		"stopLoss", "target", "riskRewardRatio", "indicators", "generatedAt", "strategy",
		"strategyVersion", "sentimentScore", "sentimentReasoning",
	} {
		if _, ok := signal[key]; !ok {
			t.Errorf("response missing Java field %q: %v", key, signal)
		}
	}
	if signal["date"] != "2026-10-27" || signal["generatedAt"] != "2026-10-27" {
		t.Errorf("dates = %v/%v, want 2026-10-27", signal["date"], signal["generatedAt"])
	}
	if signal["confidence"] != 0.75 {
		t.Errorf("confidence = %v, want numeric 0.75", signal["confidence"])
	}
	if signal["riskRewardRatio"] != 2.0 {
		t.Errorf("riskRewardRatio = %v, want numeric 2", signal["riskRewardRatio"])
	}
	indicators, ok := signal["indicators"].([]any)
	if !ok || len(indicators) != 2 || indicators[0] != "ema20" {
		t.Errorf("indicators = %v, want [ema20 rsi]", signal["indicators"])
	}
}

func assertEmptySignal(t *testing.T, signal map[string]any) {
	t.Helper()
	if signal["confidence"] != nil || signal["generatedAt"] != nil {
		t.Errorf("nullable fields = %v/%v, want null", signal["confidence"], signal["generatedAt"])
	}
	indicators, ok := signal["indicators"].([]any)
	if !ok || len(indicators) != 0 {
		t.Errorf("indicators = %v, want []", signal["indicators"])
	}
}

func TestSignalsLatestStoreUnavailable(t *testing.T) {
	response := serveSignalsLatest(t, nil)
	if response.Code != http.StatusServiceUnavailable {
		t.Fatalf("nil store status = %d, want 503", response.Code)
	}
}

func TestSignalsLatestReadFailure(t *testing.T) {
	response := serveSignalsLatest(t, fakeSignalReader{err: errors.New("boom")})
	if response.Code != http.StatusInternalServerError {
		t.Fatalf("error status = %d, want 500", response.Code)
	}
	if !strings.Contains(response.Body.String(), `"success":false`) {
		t.Fatalf("error envelope = %s", response.Body.String())
	}
}
