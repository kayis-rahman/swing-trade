package data

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// FyersTokenSource supplies the active app ID/access token and refreshes expired credentials.
type FyersTokenSource interface {
	AccessToken(context.Context) (appID, token string, err error)
	RefreshToken(context.Context) error
}

// FyersHistoryClient reads historical candles from Fyers v3.
type FyersHistoryClient struct {
	baseURL    *url.URL
	auth       FyersTokenSource
	httpClient *http.Client
}

// NewFyersHistoryClient constructs the Fyers history reader.
func NewFyersHistoryClient(baseURL string, auth FyersTokenSource, httpClient *http.Client) (*FyersHistoryClient, error) {
	parsed, err := url.Parse(baseURL)
	if err != nil || parsed.Scheme == "" || parsed.Host == "" {
		return nil, fmt.Errorf("invalid Fyers base URL %q", baseURL)
	}
	if auth == nil {
		return nil, fmt.Errorf("fyers token source is required")
	}
	if httpClient == nil {
		httpClient = &http.Client{Timeout: 15 * time.Second}
	}
	return &FyersHistoryClient{baseURL: parsed, auth: auth, httpClient: httpClient}, nil
}

// FetchCandles fetches an inclusive historical range in Fyers' maximum 365-day windows.
func (client *FyersHistoryClient) FetchCandles(ctx context.Context, symbol string, startDate, endDate time.Time, interval string) ([]YahooCandle, error) {
	if symbol == "" || interval == "" {
		return nil, fmt.Errorf("symbol and interval are required")
	}
	if startDate.After(endDate) {
		return []YahooCandle{}, nil
	}
	candles := make([]YahooCandle, 0)
	for from := startDate; !from.After(endDate); {
		to := from.AddDate(0, 0, 364)
		if to.After(endDate) {
			to = endDate
		}
		body, err := client.getRange(ctx, symbol, from, to, interval)
		if err != nil {
			return nil, err
		}
		parsed, err := ParseFyersHistory(body, symbol, interval)
		if err != nil {
			return nil, err
		}
		candles = append(candles, parsed...)
		from = to.AddDate(0, 0, 1)
	}
	return candles, nil
}

func (client *FyersHistoryClient) getRange(ctx context.Context, symbol string, from, to time.Time, interval string) ([]byte, error) {
	requestURL := *client.baseURL
	requestURL.Path = strings.TrimRight(requestURL.Path, "/") + "/data/history"
	query := requestURL.Query()
	query.Set("symbol", "NSE:"+symbol+"-EQ")
	query.Set("resolution", interval)
	query.Set("date_format", "1")
	query.Set("date[from]", from.Format("2006-01-02"))
	query.Set("date[to]", to.Format("2006-01-02"))
	requestURL.RawQuery = query.Encode()
	for authAttempt := 0; authAttempt < 2; authAttempt++ {
		appID, token, err := client.auth.AccessToken(ctx)
		if err != nil {
			return nil, fmt.Errorf("get Fyers access token: %w", err)
		}
		if token == "" || appID == "" {
			return nil, fmt.Errorf("fyers credentials are unavailable")
		}
		body, status, err := client.request(ctx, requestURL.String(), appID+":"+token)
		if err != nil {
			return nil, err
		}
		if status == http.StatusUnauthorized || fyersAuthError(body) {
			if authAttempt == 1 {
				return nil, fmt.Errorf("fyers authentication failed after refresh")
			}
			if err := client.auth.RefreshToken(ctx); err != nil {
				return nil, fmt.Errorf("refresh Fyers token: %w", err)
			}
			continue
		}
		return body, nil
	}
	return nil, fmt.Errorf("fyers authentication retries exhausted")
}

func (client *FyersHistoryClient) request(ctx context.Context, target, authorization string) ([]byte, int, error) {
	for attempt := 0; attempt < 5; attempt++ {
		request, err := http.NewRequestWithContext(ctx, http.MethodGet, target, nil)
		if err != nil {
			return nil, 0, fmt.Errorf("build Fyers history request: %w", err)
		}
		request.Header.Set("Accept", "application/json")
		request.Header.Set("Authorization", authorization)
		response, err := client.httpClient.Do(request)
		if err != nil {
			if ctx.Err() != nil {
				return nil, 0, ctx.Err()
			}
			if attempt == 4 {
				return nil, 0, fmt.Errorf("request Fyers history: %w", err)
			}
			if err := waitRetry(ctx, time.Second*time.Duration(1<<attempt)); err != nil {
				return nil, 0, err
			}
			continue
		}
		body, readErr := io.ReadAll(response.Body)
		closeErr := response.Body.Close()
		if readErr != nil {
			return nil, response.StatusCode, fmt.Errorf("read Fyers response: %w", readErr)
		}
		if closeErr != nil {
			return nil, response.StatusCode, fmt.Errorf("close Fyers response: %w", closeErr)
		}
		if response.StatusCode == http.StatusTooManyRequests && attempt < 4 {
			if err := waitRetry(ctx, time.Second*time.Duration(1<<attempt)); err != nil {
				return nil, 0, err
			}
			continue
		}
		if response.StatusCode >= 400 && response.StatusCode != http.StatusUnauthorized {
			return nil, response.StatusCode, fmt.Errorf("fyers history returned HTTP %d", response.StatusCode)
		}
		return body, response.StatusCode, nil
	}
	return nil, 0, fmt.Errorf("fyers history retries exhausted")
}

func fyersAuthError(body []byte) bool {
	var response struct {
		Status string `json:"s"`
		Code   int    `json:"code"`
	}
	if err := json.Unmarshal(body, &response); err != nil || response.Status != "error" {
		return false
	}
	return response.Code == -15 || response.Code == -16 || response.Code == -17
}
