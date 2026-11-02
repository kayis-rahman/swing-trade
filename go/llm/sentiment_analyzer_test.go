package llm

import (
	"strings"
	"testing"

	"github.com/kayis-rahman/swing-trade/go/core/domain"
)

func TestSentimentAnalyzerParseResponse(t *testing.T) {
	jsonResp := `{"score":"POSITIVE","summary":"good","confidence":0.9,"red_flags":["r [1]"],"catalysts":["c [2]"]}`
	got := SentimentAnalyzer{}.ParseResponse(jsonResp)
	if got.Sentiment != domain.SentimentScorePositive {
		t.Fatalf("got %s", got.Sentiment)
	}
	got2 := SentimentAnalyzer{}.ParseResponseWithCount(jsonResp, 0)
	if len(got2.RedFlags) != 0 {
		t.Fatalf("red flags not filtered: %+v", got2.RedFlags)
	}
	_ = strings.TrimSpace
}

func TestSentimentAnalyzerPlainTextBranches(t *testing.T) {
	cases := []struct {
		in   string
		want domain.SentimentScore
	}{
		{"mixed results, no major news", domain.SentimentScoreNeutral},
		{"earnings miss and guidance cut", domain.SentimentScoreNegative},
		{"positive only for peers", domain.SentimentScoreUnknown},
		{"not positive", domain.SentimentScoreUnknown},
	}
	for _, tc := range cases {
		got := SentimentAnalyzer{}.ParseResponse(tc.in)
		if got.Sentiment != tc.want {
			t.Fatalf("input %q: got %s want %s", tc.in, got.Sentiment, tc.want)
		}
	}
	// long text truncation branch
	long := strings.Repeat("a", 200)
	got := SentimentAnalyzer{}.ParseResponse(long)
	if got.Reasoning != long[:150] {
		t.Fatalf("reasoning not truncated: %d", len(got.Reasoning))
	}
}

func TestExtractJSONFromReasoningStripsBlocks(t *testing.T) {
	input := "<thinking>brace { inside</thinking>\n```json\n{\"score\":\"NEUTRAL\",\"summary\":\"s\",\"confidence\":0.3,\"red_flags\":[],\"catalysts\":[]}\n```"
	got := SentimentAnalyzer{}.ParseResponse(input)
	if got.Sentiment != domain.SentimentScoreNeutral {
		t.Fatalf("got %s", got.Sentiment)
	}
	if extractJSONFromReasoning("no json here") != "no json here" {
		t.Fatal("extract should return original when no brace")
	}
}

func TestParseWithJacksonInvalidConfidenceFallsBack(t *testing.T) {
	got := parseWithJackson(`{"score":"POSITIVE","confidence":"notanumber","summary":"s"}`)
	if got.Sentiment != domain.SentimentScorePositive {
		// falls back to plain text, which still sees the score keyword
		t.Fatalf("got %s", got.Sentiment)
	}
	got2 := parseWithJackson(`{"score":"FOO","summary":"s"}`)
	if got2.Sentiment != domain.SentimentScoreUnknown {
		t.Fatalf("got %s", got2.Sentiment)
	}
}
