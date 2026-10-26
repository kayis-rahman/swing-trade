package data

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

const (
	yahooChartPath = "/v8/finance/chart/"
	yahooUserAgent = "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36"
)

// YahooClient fetches chart candles using Yahoo's public chart API.
type YahooClient struct {
	baseURL    *url.URL
	httpClient *http.Client
	pacer      *RequestPacer
	retryDelay time.Duration
}

// NewYahooClient creates a Yahoo client with request throttling and Java's 15-second timeout.
func NewYahooClient(baseURL string, pacer *RequestPacer) (*YahooClient, error) {
	parsed, err := url.Parse(baseURL)
	if err != nil || parsed.Scheme == "" || parsed.Host == "" {
		return nil, fmt.Errorf("invalid Yahoo base URL %q", baseURL)
	}
	if pacer == nil {
		pacer = NewRequestPacer(60, nil)
	}
	return &YahooClient{
		baseURL:    parsed,
		httpClient: &http.Client{Timeout: 15 * time.Second},
		pacer:      pacer,
		retryDelay: time.Second,
	}, nil
}

// FetchCandles fetches an inclusive daily or intraday interval from Yahoo.
func (client *YahooClient) FetchCandles(ctx context.Context, symbol string, startDate, endDate time.Time, interval string) ([]YahooCandle, error) {
	if symbol == "" || interval == "" {
		return nil, fmt.Errorf("symbol and interval are required")
	}
	if startDate.After(endDate) {
		return []YahooCandle{}, nil
	}
	requestURL := *client.baseURL
	requestURL.Path = strings.TrimRight(requestURL.Path, "/") + yahooChartPath + yahooSymbol(symbol)
	query := requestURL.Query()
	query.Set("period1", fmt.Sprintf("%d", startDate.UTC().Truncate(24*time.Hour).Unix()))
	query.Set("period2", fmt.Sprintf("%d", endDate.UTC().Truncate(24*time.Hour).Add(24*time.Hour).Unix()))
	query.Set("interval", interval)
	query.Set("events", "history")
	query.Set("includePrePost", "false")
	requestURL.RawQuery = query.Encode()

	payload, status, err := client.get(ctx, requestURL.String())
	if err != nil {
		return nil, err
	}
	if status == http.StatusNotFound {
		return []YahooCandle{}, nil
	}
	return ParseYahooChart(payload, symbol, startDate, endDate, interval)
}

func (client *YahooClient) get(ctx context.Context, target string) ([]byte, int, error) {
	for attempt := 0; attempt < 4; attempt++ {
		if err := client.pacer.Acquire(ctx); err != nil {
			return nil, 0, err
		}
		request, err := http.NewRequestWithContext(ctx, http.MethodGet, target, nil)
		if err != nil {
			return nil, 0, fmt.Errorf("build Yahoo request: %w", err)
		}
		request.Header.Set("User-Agent", yahooUserAgent)
		response, err := client.httpClient.Do(request)
		if err != nil {
			if attempt == 3 || ctx.Err() != nil {
				return nil, 0, fmt.Errorf("request Yahoo chart: %w", err)
			}
			if err := waitRetry(ctx, client.retryDelay*time.Duration(1<<attempt)); err != nil {
				return nil, 0, err
			}
			continue
		}
		body, readErr := io.ReadAll(response.Body)
		closeErr := response.Body.Close()
		if readErr != nil {
			return nil, response.StatusCode, fmt.Errorf("read Yahoo response: %w", readErr)
		}
		if closeErr != nil {
			return nil, response.StatusCode, fmt.Errorf("close Yahoo response: %w", closeErr)
		}
		if response.StatusCode == http.StatusNotFound {
			return nil, response.StatusCode, nil
		}
		if response.StatusCode == http.StatusTooManyRequests || response.StatusCode >= 500 {
			if attempt < 3 {
				if err := waitRetry(ctx, client.retryDelay*time.Duration(1<<attempt)); err != nil {
					return nil, 0, err
				}
				continue
			}
		}
		if response.StatusCode >= 400 {
			return nil, response.StatusCode, fmt.Errorf("yahoo chart returned HTTP %d", response.StatusCode)
		}
		return body, response.StatusCode, nil
	}
	return nil, 0, fmt.Errorf("yahoo chart retries exhausted")
}

func waitRetry(ctx context.Context, delay time.Duration) error {
	timer := time.NewTimer(delay)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-timer.C:
		return nil
	}
}

func yahooSymbol(symbol string) string {
	if strings.EqualFold(symbol, "NIFTY50") || strings.EqualFold(symbol, "NIFTY 50") {
		return "^NSEI"
	}
	if strings.HasSuffix(symbol, ".NS") || strings.HasSuffix(symbol, ".BO") {
		return symbol
	}
	return symbol + ".NS"
}
