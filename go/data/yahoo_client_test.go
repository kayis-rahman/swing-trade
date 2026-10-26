package data

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestYahooClientFetchCandlesBuildsJavaCompatibleChartRequest(t *testing.T) {
	stamp := time.Date(2026, 10, 23, 4, 0, 0, 0, time.UTC).Unix()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.EscapedPath() != "/v8/finance/chart/%5ENSEI" {
			t.Errorf("path = %q", r.URL.EscapedPath())
		}
		if r.URL.Query().Get("period1") != fmt.Sprint(time.Date(2026, 10, 23, 0, 0, 0, 0, time.UTC).Unix()) {
			t.Errorf("period1 = %q", r.URL.Query().Get("period1"))
		}
		if r.URL.Query().Get("period2") != fmt.Sprint(time.Date(2026, 10, 25, 0, 0, 0, 0, time.UTC).Unix()) {
			t.Errorf("period2 = %q", r.URL.Query().Get("period2"))
		}
		if r.URL.Query().Get("interval") != "1d" || r.URL.Query().Get("events") != "history" || r.URL.Query().Get("includePrePost") != "false" {
			t.Errorf("query = %v", r.URL.Query())
		}
		if r.Header.Get("User-Agent") == "" {
			t.Error("missing Yahoo user agent")
		}
		if _, err := fmt.Fprintf(w, `{"chart":{"result":[{"timestamp":[%d],"meta":{"timezone":"Asia/Kolkata"},"indicators":{"quote":[{"open":[10],"high":[12],"low":[9],"close":[11],"volume":[5]}]}}]}}`, stamp); err != nil {
			t.Errorf("write chart fixture: %v", err)
		}
	}))
	defer server.Close()
	client, err := NewYahooClient(server.URL, NewRequestPacer(0, nil))
	if err != nil {
		t.Fatal(err)
	}
	candles, err := client.FetchCandles(context.Background(), "NIFTY 50", time.Date(2026, 10, 23, 0, 0, 0, 0, time.UTC), time.Date(2026, 10, 24, 0, 0, 0, 0, time.UTC), "1d")
	if err != nil {
		t.Fatal(err)
	}
	if len(candles) != 1 || candles[0].Symbol != "NIFTY 50" {
		t.Fatalf("candles = %+v", candles)
	}
}

func TestYahooClientTreatsNotFoundAsEmptyAndRejectsOtherClientErrors(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { http.Error(w, "missing", http.StatusNotFound) }))
	defer server.Close()
	client, err := NewYahooClient(server.URL, NewRequestPacer(0, nil))
	if err != nil {
		t.Fatal(err)
	}
	candles, err := client.FetchCandles(context.Background(), "UNKNOWN", time.Now(), time.Now(), "1d")
	if err != nil || len(candles) != 0 {
		t.Fatalf("404 result = %v, %v", candles, err)
	}
	server.Config.Handler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { http.Error(w, "bad request", http.StatusBadRequest) })
	_, err = client.FetchCandles(context.Background(), "UNKNOWN", time.Now(), time.Now(), "1d")
	if err == nil {
		t.Fatal("expected non-404 client error")
	}
}

func TestYahooClientRetriesTransientServerErrors(t *testing.T) {
	attempts := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		attempts++
		if attempts < 3 {
			http.Error(w, "temporary", http.StatusServiceUnavailable)
			return
		}
		if _, err := fmt.Fprint(w, `{"chart":{"result":[]}}`); err != nil {
			t.Errorf("write empty chart fixture: %v", err)
		}
	}))
	defer server.Close()
	client, err := NewYahooClient(server.URL, NewRequestPacer(0, nil))
	if err != nil {
		t.Fatal(err)
	}
	client.retryDelay = 0
	candles, err := client.FetchCandles(context.Background(), "ABC", time.Now(), time.Now(), "1d")
	if err != nil || len(candles) != 0 {
		t.Fatalf("retry result = %v, %v", candles, err)
	}
	if attempts != 3 {
		t.Fatalf("attempts = %d, want 3", attempts)
	}
}

func TestYahooClientHonorsContextCancellation(t *testing.T) {
	client, err := NewYahooClient("https://query1.finance.yahoo.com", NewRequestPacer(0, nil))
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	_, err = client.FetchCandles(ctx, "ABC", time.Now(), time.Now(), "1d")
	if err == nil {
		t.Fatal("expected canceled request error")
	}
}
