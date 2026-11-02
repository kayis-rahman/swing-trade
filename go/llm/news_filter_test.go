package llm

import (
	"testing"
	"time"

	"github.com/kayis-rahman/swing-trade/go/core/domain"
)

func TestNewsFilterService(t *testing.T) {
	loc, _ := time.LoadLocation("Asia/Kolkata")
	now := time.Now().In(loc)
	svc := NewNewsFilterService()
	n := func(x time.Time) *time.Time { return &x }

	article := func(content string, published time.Time, source string) domain.NewsArticle {
		p := published
		return domain.NewsArticle{
			Symbol:        "TCS",
			Title:         content,
			Description:   content,
			PublishedDate: &p,
			Source:        source,
			RawContent:    content,
		}
	}
	_ = n

	useful := article("Earnings beat and revenue growth drive a strong outlook", now.Add(-2*time.Hour), "Reuters")
	spam := article("Guaranteed no risk buy now", now.Add(-1*time.Hour), "Blog")
	old := article("Earnings beat", now.Add(-8*24*time.Hour), "Reuters")
	future := article("Earnings beat", now.Add(2*time.Hour), "Reuters")
	irrelevant := article("Company office renovation update", now.Add(-1*time.Hour), "Blog")

	got := svc.FilterRelevantArticles([]domain.NewsArticle{useful, spam, old, future, irrelevant})
	if len(got) != 1 || got[0].Title != useful.Title {
		t.Fatalf("expected only useful, got %+v", got)
	}

	positive := article("upgrade and bullish expansion", now.Add(-1*time.Hour), "NSE India")
	negative := article("downgrade and investigation", now.Add(-1*time.Hour), "Blog")
	mixed := article("bullish but lawsuit concerns", now.Add(-1*time.Hour), "Blog")
	neutral := article("quarterly results", now.Add(-1*time.Hour), "Blog")

	got2 := svc.FilterRelevantArticles([]domain.NewsArticle{positive, negative})
	if len(got2) != 2 || got2[0].Title != positive.Title || got2[1].Title != negative.Title {
		t.Fatalf("ranking mismatch: %+v", got2)
	}
	analysis := svc.AnalyzeArticleSentiment([]domain.NewsArticle{positive, negative, mixed, neutral})
	if analysis.PositiveCount != 1 || analysis.NegativeCount != 1 || analysis.NeutralCount != 2 {
		t.Fatalf("analysis=%+v", analysis)
	}
	summary := svc.GetSentimentSummary([]domain.NewsArticle{positive, negative, mixed, neutral})
	if !containsStr(summary, "Positive: 25%") || !containsStr(summary, "Negative: 25%") || !containsStr(summary, "Neutral: 50%") {
		t.Fatalf("summary=%s", summary)
	}
	if svc.GetSentimentSummary(nil) != "No articles available for analysis" {
		t.Fatal("empty summary mismatch")
	}
}

func containsStr(s, sub string) bool {
	return len(s) >= len(sub) && (func() bool {
		for i := 0; i+len(sub) <= len(s); i++ {
			if s[i:i+len(sub)] == sub {
				return true
			}
		}
		return false
	})()
}
