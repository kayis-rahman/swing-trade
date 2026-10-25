package data

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

type fyersTestAuth struct {
	token     string
	refreshes int
}

func (a *fyersTestAuth) AccessToken(context.Context) (string, string, error) {
	return "app-id", a.token, nil
}
func (a *fyersTestAuth) RefreshToken(context.Context) error {
	a.refreshes++
	a.token = "fresh"
	return nil
}

func TestFyersHistoryClientBuildsRequestAndUsesRefreshAfterUnauthorized(t *testing.T) {
	stamp := time.Date(2026, 10, 23, 4, 0, 0, 0, time.UTC).Unix()
	auth := &fyersTestAuth{token: "expired"}
	requests := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests++
		if r.URL.Path != "/data/history" {
			t.Errorf("path = %q", r.URL.Path)
		}
		if r.URL.Query().Get("symbol") != "NSE:ABC-EQ" || r.URL.Query().Get("resolution") != "15" || r.URL.Query().Get("date_format") != "1" {
			t.Errorf("query = %v", r.URL.Query())
		}
		if r.URL.Query().Get("date[from]") != "2026-10-23" || r.URL.Query().Get("date[to]") != "2026-10-23" {
			t.Errorf("range = %v", r.URL.Query())
		}
		if requests == 1 {
			if r.Header.Get("Authorization") != "app-id:expired" {
				t.Errorf("first auth = %q", r.Header.Get("Authorization"))
			}
			http.Error(w, "expired", http.StatusUnauthorized)
			return
		}
		if r.Header.Get("Authorization") != "app-id:fresh" {
			t.Errorf("retried auth = %q", r.Header.Get("Authorization"))
		}
		if _, err := fmt.Fprintf(w, `{"s":"ok","candles":[[%d,10,12,9,11,5,0]]}`, stamp); err != nil {
			t.Errorf("write fixture: %v", err)
		}
	}))
	defer server.Close()
	client, err := NewFyersHistoryClient(server.URL, auth, server.Client())
	if err != nil {
		t.Fatal(err)
	}
	candles, err := client.FetchCandles(context.Background(), "ABC", time.Date(2026, 10, 23, 0, 0, 0, 0, time.UTC), time.Date(2026, 10, 23, 0, 0, 0, 0, time.UTC), "15")
	if err != nil {
		t.Fatal(err)
	}
	if len(candles) != 1 || candles[0].Close.String() != "11.0000" {
		t.Fatalf("candles = %+v", candles)
	}
	if auth.refreshes != 1 || requests != 2 {
		t.Fatalf("refreshes=%d requests=%d", auth.refreshes, requests)
	}
}

type fyersRoundTripperFunc func(*http.Request) (*http.Response, error)

func (fn fyersRoundTripperFunc) RoundTrip(request *http.Request) (*http.Response, error) {
	return fn(request)
}

func TestFyersHistoryRequestDoesNotRetryNetworkErrors(t *testing.T) {
	calls := 0
	transport := fyersRoundTripperFunc(func(*http.Request) (*http.Response, error) {
		calls++
		return nil, errors.New("network unavailable")
	})
	client, err := NewFyersHistoryClient("https://example.invalid", &fyersTestAuth{token: "token"}, &http.Client{Transport: transport})
	if err != nil {
		t.Fatal(err)
	}
	_, _, err = client.request(context.Background(), "https://example.invalid/data/history", "app-id:token")
	if err == nil || calls != 1 {
		t.Fatalf("err=%v calls=%d, want immediate single failure", err, calls)
	}
}

func TestFyersHistoryClientRejectsMissingCredentialsAndReversedWindowIsEmpty(t *testing.T) {
	auth := &fyersTestAuth{}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { t.Error("unexpected request") }))
	defer server.Close()
	client, err := NewFyersHistoryClient(server.URL, auth, server.Client())
	if err != nil {
		t.Fatal(err)
	}
	if _, err := client.FetchCandles(context.Background(), "ABC", time.Now(), time.Now(), "D"); err == nil {
		t.Fatal("expected missing-credentials error")
	}
	if _, err := client.FetchCandles(context.Background(), "ABC", time.Now(), time.Now().Add(-24*time.Hour), "D"); err != nil {
		t.Fatalf("reversed range error = %v", err)
	}
}
