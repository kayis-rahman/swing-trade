package llm

import (
	"regexp"
	"strconv"
	"strings"
)

// articleCitation parity.
var articleCitation = regexp.MustCompile(`\[(\d+)\]`)

// ValidateSentimentOutput applies evidence boundary like Java SentimentOutputValidator.
func ValidateSentimentOutput(out SentimentOutput, articleCount int) SentimentOutput {
	groundedRed := groundedItems(out.RedFlags, articleCount)
	groundedCat := groundedItems(out.Catalysts, articleCount)
	return SentimentOutput{
		Sentiment:      out.Sentiment,
		Reasoning:      out.Reasoning,
		Confidence:     out.Confidence,
		RedFlags:       groundedRed,
		Catalysts:      groundedCat,
		Source:         out.Source,
		DegradedReason: out.DegradedReason,
	}
}

// groundedItems parity.
func groundedItems(items []string, articleCount int) []string {
	if len(items) == 0 || articleCount < 1 {
		return []string{}
	}
	out := []string{}
	for _, item := range items {
		if strings.TrimSpace(item) == "" {
			continue
		}
		if citesOnlyAvailableArticle(item, articleCount) {
			out = append(out, item)
		}
	}
	return out
}

// citesOnlyAvailableArticle parity.
func citesOnlyAvailableArticle(item string, articleCount int) bool {
	matches := articleCitation.FindAllStringSubmatch(item, -1)
	if len(matches) == 0 {
		return false
	}
	for _, m := range matches {
		idx, err := strconv.Atoi(m[1])
		if err != nil {
			return false
		}
		if idx < 1 || idx > articleCount {
			return false
		}
	}
	return true
}
