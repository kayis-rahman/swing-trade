package llm

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/kayis-rahman/swing-trade/go/core/domain"
)

//nolint:funlen
func TestCoverageMethods(t *testing.T) {
	// backend
	for _, tc := range []struct {
		in  string
		out Backend
	}{
		{"local", BackendLocal}, {"pi_ssh", BackendPiSSH}, {"openai", BackendOpenAI},
		{"ollama", BackendOllama}, {"laya", BackendLaya}, {"pi_agent", BackendPiAgent},
		{"gpuhub", BackendOpenAI}, {"x", BackendLocal},
	} {
		if BackendFromKey(tc.in) != tc.out {
			t.Fatal("backend")
		}
	}

	// NewClient + Chat
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"choices":[{"message":{"content":"hi"}}]}`))
	}))
	defer srv.Close()
	c := NewClient(srv.URL, "key")
	resp, err := c.Chat(context.Background(), ChatRequest{Messages: []map[string]any{{"role": "user", "content": "x"}}})
	if err != nil || resp.Choices[0].Message.Content != "hi" {
		t.Fatalf("chat %v", err)
	}

	// SentimentAnalyzer prompt methods
	a := SentimentAnalyzer{}
	if a.GetSystemPrompt() == "" {
		t.Fatal("sys")
	}
	msgs := a.CreatePrompt("AAPL", "news")
	if len(msgs) != 2 {
		t.Fatal("createprompt")
	}
	msgs2 := a.CreatePromptMulti("AAPL", []string{"h"}, "")
	if len(msgs2) != 2 {
		t.Fatal("createpromptmulti")
	}

	// Parse JSON via reasoning extraction
	jsonResp := "<think>x</think>\n```json\n{\"sentiment\":\"NEGATIVE\",\"summary\":\"bad\",\"confidence\":0.2,\"red_flags\":[],\"catalysts\":[]}\n```"
	got := a.ParseResponse(jsonResp)
	if got.Sentiment != domain.SentimentScoreNegative {
		t.Fatalf("got %s", got.Sentiment)
	}

	// plain text
	got2 := a.ParseResponse("positive bullish outlook")
	if got2.Sentiment != domain.SentimentScorePositive {
		t.Fatalf("got %s", got2.Sentiment)
	}

	// empty
	got3 := a.ParseResponse("")
	if got3.Sentiment != domain.SentimentScoreUnknown {
		t.Fatalf("got %s", got3.Sentiment)
	}

	// invalid field types
	got4 := parseWithJackson(`{"score": "NEUTRAL", "confidence": "0.7", "summary": "s"}`)
	if got4.Sentiment != domain.SentimentScoreNeutral {
		t.Fatalf("got %s", got4.Sentiment)
	}

	// extractJSONFromReasoning with no brace
	if extractJSONFromReasoning("no json") != "no json" {
		t.Fatal("extract")
	}

	// NewSentimentOutput
	out := NewSentimentOutput(domain.SentimentScorePositive, "r", floatPtr(0.5), nil, nil, "")
	if out.Source != "DEFAULT" || len(out.RedFlags) != 0 {
		t.Fatalf("out %+v", out)
	}
	b, err := out.MarshalJSON()
	if err != nil || !strings.Contains(string(b), "score") {
		t.Fatal("marshal")
	}

	// groundedItems direct
	if got := groundedItems([]string{"x [1]"}, 1); len(got) != 1 {
		t.Fatal("grounded")
	}

	// prompt loader negative? mustLoadPrompt panic not tested.
	_ = SentimentPromptLoader{}
}
