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
