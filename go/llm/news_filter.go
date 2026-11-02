package llm

import (
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/kayis-rahman/swing-trade/go/core/domain"
)

const (
	recentHours = 24
	freshHours  = 48
	maxHours    = 168
)

var (
	highQualitySources = []string{
		"Economic Times", "MoneyControl", "Reuters", "Bloomberg",
		"Business Standard", "Financial Express", "Livemint",
		"NSE India", "BSE India", "PTI",
	}

	spamPatterns = []*regexp.Regexp{
		regexp.MustCompile(`(?i)\b(congrats|congratulations|celebrate|party)\b`),
		regexp.MustCompile(`(?i)\b(free|guaranteed|100%|no risk)\b`),
		regexp.MustCompile(`(?i)\b(invest now|buy now|call of the day)\b`),
		regexp.MustCompile(`(?i)\b(unconfirmed|rumor|speculation|gossip)\b`),
	}
)

type tradingKeyword struct {
	text   string
	weight int
}

var positiveKeywords = []tradingKeyword{
	{"earnings beat", 3}, {"revenue growth", 3}, {"profit increase", 3},
	{"upgrade", 3}, {"bullish", 3}, {"surge", 2}, {"strong", 2},
	{"record", 2}, {"milestone", 2}, {"expansion", 2}, {"contract", 2},
	{"partnership", 2}, {"positive outlook", 3}, {"guidance raise", 3},
	{"dividend increase", 2}, {"buyback", 2},
}

var negativeKeywords = []tradingKeyword{
	{"earnings miss", 3}, {"revenue decline", 3}, {"loss", 3},
	{"downgrade", 3}, {"bearish", 3}, {"drop", 2}, {"weak", 2},
	{"concern", 2}, {"investigation", 3}, {"lawsuit", 2}, {"fraud", 3},
	{"scandal", 3}, {"negative outlook", 3}, {"guidance cut", 3},
	{"layoff", 2}, {"delay", 2},
}

var neutralKeywords = []tradingKeyword{
	{"quarterly results", 1}, {"board meeting", 1},
	{"annual general meeting", 1}, {"management change", 1},
	{"product launch", 1}, {"partnership announcement", 1},
}

// NewsFilterService filters and ranks news articles by relevance.
type NewsFilterService struct{}

func NewNewsFilterService() *NewsFilterService { return &NewsFilterService{} }

type FilteredArticle struct {
	Article domain.NewsArticle
	Score   float64
}

type ArticleSentimentAnalysis struct {
	PositiveCount int
	NegativeCount int
	NeutralCount  int
	TotalCount    int
}

func (s *NewsFilterService) FilterRelevantArticles(articles []domain.NewsArticle) []domain.NewsArticle {
	var filtered []FilteredArticle
	for _, a := range articles {
		fa := s.evaluateArticle(a)
		if fa != nil {
			filtered = append(filtered, *fa)
		}
	}
	// sort by score desc
	// simple insertion sort
	for i := 1; i < len(filtered); i++ {
		j := i
		for j > 0 && filtered[j-1].Score < filtered[j].Score {
			filtered[j-1], filtered[j] = filtered[j], filtered[j-1]
			j--
		}
	}
	out := make([]domain.NewsArticle, 0, len(filtered))
	for _, f := range filtered {
		out = append(out, f.Article)
	}
	return out
}

func (s *NewsFilterService) evaluateArticle(article domain.NewsArticle) *FilteredArticle {
	content := getArticleContent(article)
	if containsSpamPattern(content) {
		return nil
	}
	ageHours := calculateAgeHours(article.PublishedDate)
	if ageHours > maxHours {
		return nil
	}
	tradingScore := calculateTradingRelevanceScore(content)
	if tradingScore <= 0.2 && !isHighQualitySource(article.Source) && !hasSubstance(article) {
		return nil
	}
	score := s.calculateRelevanceScore(article, ageHours)
	if score < 0.3 {
		return nil
	}
	return &FilteredArticle{Article: article, Score: score}
}

func (s *NewsFilterService) calculateRelevanceScore(article domain.NewsArticle, ageHours int64) float64 {
	score := 1.0
	if ageHours <= recentHours {
		// fresh
	} else if ageHours <= freshHours {
		score *= 0.8
	} else {
		score *= 0.5
	}
	if isHighQualitySource(article.Source) {
		score *= 1.2
	}
	if hasSubstance(article) {
		score *= 1.1
	}
	tradingScore := calculateTradingRelevanceScore(getArticleContent(article))
	score = (score * 0.4) + (tradingScore * 0.6)
	if score > 1.0 {
		score = 1.0
	}
	return score
}

func calculateTradingRelevanceScore(content string) float64 {
	lower := strings.ToLower(content)
	positiveScore := 0.0
	negativeScore := 0.0
	positiveCount := 0
	negativeCount := 0
	for _, kw := range positiveKeywords {
		if strings.Contains(lower, strings.ToLower(kw.text)) {
			positiveScore += float64(kw.weight)
			positiveCount++
		}
	}
	for _, kw := range negativeKeywords {
		if strings.Contains(lower, strings.ToLower(kw.text)) {
			negativeScore += float64(kw.weight)
			negativeCount++
		}
	}
	positiveScore = min(1.0, positiveScore/10.0)
	negativeScore = min(1.0, negativeScore/10.0)
	if positiveCount > 0 || negativeCount > 0 {
		return (positiveScore + negativeScore) / 2.0
	}
	for _, kw := range neutralKeywords {
		if strings.Contains(lower, strings.ToLower(kw.text)) {
			return 0.4
		}
	}
	return 0.2
}

func getArticleContent(article domain.NewsArticle) string {
	var sb strings.Builder
	if article.Title != "" {
		sb.WriteString(article.Title)
		sb.WriteString(" ")
	}
	if article.Description != "" {
		sb.WriteString(article.Description)
		sb.WriteString(" ")
	}
	if article.RawContent != "" {
		sb.WriteString(article.RawContent)
	}
	return sb.String()
}

func containsSpamPattern(content string) bool {
	for _, p := range spamPatterns {
		if p.MatchString(content) {
			return true
		}
	}
	return false
}

func isHighQualitySource(source string) bool {
	if source == "" {
		return false
	}
	lower := strings.ToLower(source)
	for _, hq := range highQualitySources {
		if strings.Contains(lower, strings.ToLower(hq)) {
			return true
		}
	}
	return false
}

func hasSubstance(article domain.NewsArticle) bool {
	content := getArticleContent(article)
	words := strings.Fields(strings.TrimSpace(content))
	return len(words) >= 20
}

func calculateAgeHours(pd *time.Time) int64 {
	if pd == nil {
		return maxHours + 1
	}
	loc := pd.Location()
	if loc == nil {
		loc = time.UTC
	}
	now := time.Now().In(loc)
	if pd.After(now) {
		return maxHours + 1
	}
	return int64(now.Sub(*pd).Hours())
}

func (s *NewsFilterService) AnalyzeArticleSentiment(articles []domain.NewsArticle) ArticleSentimentAnalysis {
	positiveCount := 0
	negativeCount := 0
	neutralCount := 0
	for _, article := range articles {
		content := strings.ToLower(getArticleContent(article))
		hasPositive := false
		hasNegative := false
		for _, kw := range positiveKeywords {
			if strings.Contains(content, strings.ToLower(kw.text)) {
				hasPositive = true
				break
			}
		}
		for _, kw := range negativeKeywords {
			if strings.Contains(content, strings.ToLower(kw.text)) {
				hasNegative = true
				break
			}
		}
		if hasPositive && !hasNegative {
			positiveCount++
		} else if hasNegative && !hasPositive {
			negativeCount++
		} else {
			neutralCount++
		}
	}
	return ArticleSentimentAnalysis{
		PositiveCount: positiveCount,
		NegativeCount: negativeCount,
		NeutralCount:  neutralCount,
		TotalCount:    len(articles),
	}
}

func (s *NewsFilterService) GetSentimentSummary(articles []domain.NewsArticle) string {
	analysis := s.AnalyzeArticleSentiment(articles)
	if len(articles) == 0 {
		return "No articles available for analysis"
	}
	positiveRatio := int(float64(analysis.PositiveCount) / float64(analysis.TotalCount) * 100)
	negativeRatio := int(float64(analysis.NegativeCount) / float64(analysis.TotalCount) * 100)
	neutralRatio := int(float64(analysis.NeutralCount) / float64(analysis.TotalCount) * 100)
	return fmt.Sprintf("Article Sentiment Analysis: Positive: %d%%, Negative: %d%%, Neutral: %d%%",
		positiveRatio, negativeRatio, neutralRatio)
}

func min(a, b float64) float64 {
	if a < b {
		return a
	}
	return b
}
