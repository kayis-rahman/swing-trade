package llm

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/kayis-rahman/swing-trade/go/core/domain"
)

// SystemPrompt parity.
const SystemPrompt = `You are a financial analyst specialising in Indian equity markets.
Analyse the following for a swing trade entry decision on {symbol}.
Consider: earnings momentum, regulatory news, management changes,
sector tailwinds, FII/DII activity, promoter actions.

Respond with a JSON object containing sentiment analysis fields.
Start your response with { and end with }.

{
  "score": "POSITIVE|NEUTRAL|NEGATIVE",
  "confidence": 0.0-1.0,
  "summary": "2 sentence max reasoning",
  "red_flags": ["list any specific risks"],
  "catalysts": ["list any upcoming catalysts"]
}

Examples:
POSITIVE: Strong quarterly results, FII buying, sector tailwind
NEUTRAL: Mixed results, no major news
NEGATIVE: Promoter pledge, SEBI action, earnings miss
`

// SentimentAnalyzer mirrors Java's helper.
type SentimentAnalyzer struct{}

// GetSystemPrompt handles parity behavior.
func (SentimentAnalyzer) GetSystemPrompt() string { return SystemPrompt }

// CreatePrompt handles parity behavior.
func (SentimentAnalyzer) CreatePrompt(stockSymbol, newsContent string) []map[string]string {
	msg := fmt.Sprintf(`Analyse the following for a swing trade entry decision on %s.

Recent news headlines (last 7 days):
%s

Task: Determine if news sentiment supports a 1-4 week swing trade entry.

Respond with ONLY a JSON object. Start with { and end with }.

{
  "score": "POSITIVE|NEUTRAL|NEGATIVE",
  "confidence": 0.0-1.0,
  "summary": "2 sentence max reasoning",
  "red_flags": ["list any specific risks"],
  "catalysts": ["list any upcoming catalysts"]
}
`, stockSymbol, newsContent)
	return []map[string]string{
		{"role": "system", "content": SystemPrompt},
		{"role": "user", "content": msg},
	}
}

// CreatePromptMulti handles parity behavior.
func (SentimentAnalyzer) CreatePromptMulti(stockSymbol string, headlines []string, earningsSummary string) []map[string]string {
	earningsText := earningsSummary
	if strings.TrimSpace(earningsText) == "" {
		earningsText = "No earnings data available"
	}
	msg := fmt.Sprintf(`Analyse the following for a swing trade entry decision on %s.

Recent news headlines (last 7 days):
%s

Latest earnings summary:
%s

Task: Determine if news sentiment supports a 1-4 week swing trade entry.

Respond with ONLY a JSON object. Start with { and end with }.

{
  "score": "POSITIVE|NEUTRAL|NEGATIVE",
  "confidence": 0.0-1.0,
  "summary": "2 sentence max reasoning",
  "red_flags": ["list any specific risks"],
  "catalysts": ["list any upcoming catalysts"]
}
`, stockSymbol, strings.Join(headlines, "\n"), earningsText)
	return []map[string]string{
		{"role": "system", "content": SystemPrompt},
		{"role": "user", "content": msg},
	}
}

// ParseResponse handles parity behavior.
func (SentimentAnalyzer) ParseResponse(jsonResponse string) SentimentOutput {
	if jsonResponse == "" {
		return SentimentOutput{Sentiment: domain.SentimentScoreUnknown, Reasoning: "Empty response", Confidence: floatPtr(0), RedFlags: []string{}, Catalysts: []string{}, Source: "DEFAULT", DegradedReason: ""}
	}
	if strings.Contains(jsonResponse, "\"score\"") {
		var conv struct {
			Score      string   `json:"score"`
			Summary    string   `json:"summary"`
			Confidence *float64 `json:"confidence"`
			RedFlags   []string `json:"red_flags"`
			Catalysts  []string `json:"catalysts"`
		}
		if err := json.Unmarshal([]byte(extractJSONFromReasoning(jsonResponse)), &conv); err == nil && conv.Score != "" {
			score := domain.SentimentScore(strings.ToUpper(conv.Score))
			conf := conv.Confidence
			if conf == nil {
				c := 0.5
				conf = &c
			}
			src := "LLM"
			return NewSentimentOutput(score, conv.Summary, conf, conv.RedFlags, conv.Catalysts, src)
		}
	}
	return parseWithJackson(jsonResponse)
}

// ParseResponseWithCount handles parity behavior.
func (SentimentAnalyzer) ParseResponseWithCount(jsonResponse string, articleCount int) SentimentOutput {
	return ValidateSentimentOutput(SentimentAnalyzer{}.ParseResponse(jsonResponse), articleCount)
}

// parseWithJackson handles parity behavior.
func parseWithJackson(text string) SentimentOutput {
	content := extractJSONFromReasoning(text)
	var root map[string]any
	if content != "" {
		if err := json.Unmarshal([]byte(content), &root); err == nil {
			if has(root, "score") || has(root, "sentiment") {
				scoreStr := ""
				if v, ok := root["score"]; ok {
					scoreStr = fmt.Sprint(v)
				} else if v, ok := root["sentiment"]; ok {
					scoreStr = fmt.Sprint(v)
				}
				conf := 0.5
				if v, ok := root["confidence"]; ok {
					if f, ok := toFloat(v); ok {
						conf = f
					}
				}
				summary := "Analysis incomplete"
				if v, ok := root["summary"]; ok {
					summary = fmt.Sprint(v)
				}
				red := []string{}
				if arr, ok := root["red_flags"].([]any); ok {
					for _, x := range arr {
						red = append(red, fmt.Sprint(x))
					}
				}
				cat := []string{}
				if arr, ok := root["catalysts"].([]any); ok {
					for _, x := range arr {
						cat = append(cat, fmt.Sprint(x))
					}
				}
				c := conf
				return NewSentimentOutput(domain.SentimentScore(strings.ToUpper(scoreStr)), summary, &c, red, cat, "LLM")
			}
		}
	}
	// plain text fallback
	return parsePlainText(text)
}

// has handles parity behavior.
func has(m map[string]any, key string) bool {
	_, ok := m[key]
	return ok
}

// toFloat handles parity behavior.
func toFloat(v any) (float64, bool) {
	switch t := v.(type) {
	case float64:
		return t, true
	case int:
		return float64(t), true
	case string:
		var f float64
		_, err := fmt.Sscanf(t, "%f", &f)
		return f, err == nil
	}
	return 0, false
}

// parsePlainText handles parity behavior.
func parsePlainText(text string) SentimentOutput {
	if strings.TrimSpace(text) == "" {
		c := 0.0
		return NewSentimentOutput(domain.SentimentScoreUnknown, "Empty response", &c, []string{}, []string{}, "DEFAULT")
	}
	lower := strings.ToLower(text)
	// very simple
	sentiment := domain.SentimentScoreUnknown
	conf := 0.0
	positive := strings.Contains(lower, "positive") || strings.Contains(lower, "bullish")
	negative := strings.Contains(lower, "negative") || strings.Contains(lower, "bearish")
	switch {
	case positive && !negative:
		sentiment = domain.SentimentScorePositive
		conf = 0.4
	case negative && !positive:
		sentiment = domain.SentimentScoreNegative
		conf = 0.4
	case strings.Contains(lower, "mixed") || strings.Contains(lower, "balanced"):
		sentiment = domain.SentimentScoreNeutral
		conf = 0.2
	}
	summary := text
	if len(summary) > 150 {
		summary = summary[:150]
	}
	summary = strings.TrimSpace(summary)
	c := conf
	return NewSentimentOutput(sentiment, summary, &c, []string{}, []string{}, "KEYWORD")
}

// extractJSONFromReasoning handles parity behavior.
func extractJSONFromReasoning(text string) string {
	if strings.TrimSpace(text) == "" {
		return text
	}
	cleaned := text
	open := firstBrace(cleaned, '{')
	if open < 0 {
		return text
	}
	balance := 0
	inString := false
	escaped := false
	for i := open; i < len(cleaned); i++ {
		ch := cleaned[i]
		if escaped {
			escaped = false
			continue
		}
		if ch == '\\' {
			escaped = true
			continue
		}
		if ch == '"' {
			inString = !inString
			continue
		}
		if !inString {
			switch ch {
			case '{':
				balance++
			case '}':
				balance--
				if balance == 0 {
					candidate := cleaned[open : i+1]
					if json.Valid([]byte(candidate)) {
						return candidate
					}
					return ""
				}
			}
		}
	}
	return ""
}

// firstBrace handles parity behavior.
func firstBrace(text string, c byte) int {
	inString := false
	escaped := false
	for i := 0; i < len(text); i++ {
		if escaped {
			escaped = false
			continue
		}
		ch := text[i]
		if ch == '\\' {
			escaped = true
			continue
		}
		if ch == '"' {
			inString = !inString
			continue
		}
		if !inString && ch == c {
			return i
		}
	}
	return -1
}

// floatPtr handles parity behavior.
func floatPtr(v float64) *float64 { return &v }
