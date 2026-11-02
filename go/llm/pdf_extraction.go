package llm

import (
	"context"
	"fmt"
	"strings"
)

// PdfExtractor extracts readable content from raw text using an LLM client.
// PdfExtractor parity.
type PdfExtractor interface {
	GenerateChatCompletion(ctx context.Context, messages []map[string]string, maxTokens int, temperature float64) (string, error)
}

// PdfExtractionService parity.
type PdfExtractionService struct {
	Loader   PdfExtractionPromptLoader
	Client   PdfExtractor
	MaxChars int
}

// NewPdfExtractionService parity.
func NewPdfExtractionService(client PdfExtractor) *PdfExtractionService {
	return &PdfExtractionService{Loader: NewPdfExtractionPromptLoader(), Client: client, MaxChars: 100000}
}

// Extract parity.
func (s *PdfExtractionService) Extract(ctx context.Context, rawText string) (string, error) {
	if s.Client == nil {
		return "", fmt.Errorf("llm client not configured")
	}
	text := strings.TrimSpace(rawText)
	if text == "" {
		return "", fmt.Errorf("empty pdf text")
	}
	if len(text) > s.MaxChars {
		text = text[:s.MaxChars]
	}
	messages := []map[string]string{
		{"role": "system", "content": s.Loader.Prompt},
		{"role": "user", "content": text},
	}
	out, err := s.Client.GenerateChatCompletion(ctx, messages, 4096, 0.2)
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(out), nil
}
