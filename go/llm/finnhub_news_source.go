package llm

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/kayis-rahman/swing-trade/go/core/domain"
)

const finnhubNewsURL = "https://finnhub.io/api/v1/news"

// FinnhubNewsSource fetches news articles from Finnhub.
type FinnhubNewsSource struct {
	apiKey      string
	maxArticles int
	httpClient  *http.Client
	baseURL     string
}

// NewFinnhubNewsSource creates a FinnhubNewsSource.
func NewFinnhubNewsSource(apiKey string, maxArticles int, httpClient *http.Client) *FinnhubNewsSource {
	if maxArticles <= 0 {
		maxArticles = 15
	}
	if httpClient == nil {
		httpClient = &http.Client{Timeout: 8 * time.Second}
	}
	return &FinnhubNewsSource{
		apiKey:      strings.TrimSpace(apiKey),
		maxArticles: maxArticles,
		httpClient:  httpClient,
		baseURL:     finnhubNewsURL,
	}
}

// Type returns the source type identifier.
func (s *FinnhubNewsSource) Type() string { return "finnhub" }

// Fetch retrieves news articles for symbol.
func (s *FinnhubNewsSource) Fetch(ctx context.Context, symbol string) ([]domain.NewsArticle, error) {
	if s.apiKey == "" {
		return nil, nil
	}
	if ctx == nil {
		ctx = context.Background()
	}
	reqURL := fmt.Sprintf("%s?category=general&symbol=%s&token=%s", s.baseURL, url.QueryEscape(symbol), url.QueryEscape(s.apiKey))
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, reqURL, nil)
	if err != nil {
		return nil, err
	}
	resp, err := s.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("finnhub returned HTTP %d", resp.StatusCode)
	}
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	out, err := parseFinnhubArticles(body, symbol, s.maxArticles)
	if err != nil {
		return nil, err
	}
	return out, nil
}

type finnhubArticle struct {
	Headline string `json:"headline"`
	URL      string `json:"url"`
	Summary  string `json:"summary"`
	Datetime int64  `json:"datetime"`
}

func parseFinnhubArticles(body []byte, symbol string, maxArticles int) ([]domain.NewsArticle, error) {
	var root []finnhubArticle
	if err := json.Unmarshal(body, &root); err != nil {
		return nil, err
	}
	seen := map[string]bool{}
	var out []domain.NewsArticle
	for _, a := range root {
		headline := strings.TrimSpace(a.Headline)
		if headline == "" {
			continue
		}
		key := strings.ToLower(headline)
		if seen[key] {
			continue
		}
		if len(out) >= maxArticles {
			continue
		}
		seen[key] = true
		summary := strings.TrimSpace(a.Summary)
		if len(summary) > 1000 {
			summary = summary[:1000]
		}
		var pub *time.Time
		if a.Datetime > 0 {
			tt := time.Unix(a.Datetime, 0).UTC()
			pub = &tt
		} else {
			now := time.Now().UTC()
			pub = &now
		}
		out = append(out, domain.NewsArticle{
			Symbol:        symbol,
			Title:         headline,
			Link:          a.URL,
			Description:   summary,
			PublishedDate: pub,
			Source:        "finnhub",
			RawContent:    summary,
		})
	}
	return out, nil

}
