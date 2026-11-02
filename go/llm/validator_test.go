package llm

import "testing"

func TestValidateSentimentOutput(t *testing.T) {
	out := SentimentOutput{RedFlags: []string{"bad [2]", "no cite", "bad [5]"}, Catalysts: []string{"good [1]"}}
	got := ValidateSentimentOutput(out, 2)
	if len(got.RedFlags) != 1 || got.RedFlags[0] != "bad [2]" {
		t.Fatalf("got %+v", got.RedFlags)
	}
	if len(got.Catalysts) != 1 {
		t.Fatalf("got %+v", got.Catalysts)
	}
}
