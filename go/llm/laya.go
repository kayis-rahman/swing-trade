package llm

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"strings"
	"time"
)

// LayaResult mirrors Java LayaResult.
// LayaResult parity.
type LayaResult struct {
	Sentiment  string
	Confidence float64
}

// LayaClient is an optional fast classifier client.
// LayaClient parity.
type LayaClient struct {
	BaseURL string
	APIKey  string
	Timeout time.Duration
	HTTP    *http.Client
}

// NewLayaClient parity.
func NewLayaClient(baseURL, apiKey string, timeout time.Duration) *LayaClient {
	if timeout <= 0 {
		timeout = 5 * time.Second
	}
	return &LayaClient{BaseURL: stringsTrim(baseURL), APIKey: apiKey, Timeout: timeout, HTTP: &http.Client{Timeout: timeout}}
}

// Classify parity.
func (c *LayaClient) Classify(ctx context.Context, text string) (LayaResult, bool) {
	if stringsTrim(c.BaseURL) == "" {
		return LayaResult{}, false
	}
	if ctx == nil {
		ctx = context.Background()
	}
	ctx, cancel := context.WithTimeout(ctx, c.Timeout)
	defer cancel()

	body, _ := json.Marshal(map[string]string{"text": text})
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.BaseURL+"/classify", bytes.NewReader(body))
	if err != nil {
		return LayaResult{}, false
	}
	req.Header.Set("Content-Type", "application/json")
	if c.APIKey != "" {
		req.Header.Set("X-API-Key", c.APIKey)
	}
	resp, err := c.HTTP.Do(req)
	if err != nil {
		return LayaResult{}, false
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode >= 300 {
		return LayaResult{}, false
	}
	var raw map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&raw); err != nil {
		return LayaResult{}, false
	}
	sentiment, _ := raw["sentiment"].(string)
	confRaw, ok := raw["confidence"]
	if !ok {
		return LayaResult{}, false
	}
	conf, ok := layaToFloat(confRaw)
	if !ok {
		return LayaResult{}, false
	}
	if sentiment == "" {
		return LayaResult{}, false
	}
	return LayaResult{Sentiment: stringsUpper(sentiment), Confidence: conf}, true
}

func layaToFloat(v any) (float64, bool) {
	switch x := v.(type) {
	case float64:
		return x, true
	case float32:
		return float64(x), true
	case int:
		return float64(x), true
	case int64:
		return float64(x), true
	case json.Number:
		f, err := x.Float64()
		if err != nil {
			return 0, false
		}
		return f, true
	}
	return 0, false
}

func stringsTrim(s string) string  { return strings.TrimSpace(s) }
func stringsUpper(s string) string { return strings.ToUpper(strings.TrimSpace(s)) }
