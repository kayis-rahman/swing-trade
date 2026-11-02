package llm

import "testing"

func TestPromptLoaders(t *testing.T) {
	s := NewSentimentPromptLoader()
	if s.SystemPrompt == "" || s.UserPrompt == "" {
		t.Fatal("empty sentiment prompts")
	}
	sy := NewSynthesisPromptLoader()
	if sy.SystemPrompt == "" || sy.UserPromptTemplate == "" {
		t.Fatal("empty synthesis prompts")
	}
	p := NewPdfExtractionPromptLoader()
	if p.Prompt == "" {
		t.Fatal("empty pdf prompt")
	}
}
