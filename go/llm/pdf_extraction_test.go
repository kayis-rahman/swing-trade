package llm

import (
	"context"
	"testing"
)

type fakeLLM struct {
	out string
	err error
}

func (f *fakeLLM) GenerateChatCompletion(ctx context.Context, messages []map[string]string, maxTokens int, temperature float64) (string, error) {
	return f.out, f.err
}

func TestPdfExtract(t *testing.T) {
	s := NewPdfExtractionService(&fakeLLM{out: "  summary  "})
	got, err := s.Extract(context.Background(), "raw pdf text")
	if err != nil || got != "summary" {
		t.Fatalf("got %q err %v", got, err)
	}
}
func TestPdfExtractEmpty(t *testing.T) {
	s := NewPdfExtractionService(&fakeLLM{out: "x"})
	_, err := s.Extract(context.Background(), "   ")
	if err == nil {
		t.Fatal("expected err")
	}
}
