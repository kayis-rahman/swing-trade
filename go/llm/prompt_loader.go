package llm

import (
	"embed"
	"fmt"
	"strings"
)

//go:embed prompts/*.md
var promptFS embed.FS

// SentimentPromptLoader parity.
type SentimentPromptLoader struct {
	SystemPrompt string
	UserPrompt   string
}

// NewSentimentPromptLoader parity.
func NewSentimentPromptLoader() SentimentPromptLoader {
	return SentimentPromptLoader{
		SystemPrompt: mustLoadPrompt("prompts/sentiment-system.md"),
		UserPrompt:   mustLoadPrompt("prompts/sentiment-user.md"),
	}
}

// SynthesisPromptLoader parity.
type SynthesisPromptLoader struct {
	SystemPrompt       string
	UserPromptTemplate string
}

// NewSynthesisPromptLoader parity.
func NewSynthesisPromptLoader() SynthesisPromptLoader {
	return SynthesisPromptLoader{
		SystemPrompt:       mustLoadPrompt("prompts/synthesis-system.md"),
		UserPromptTemplate: mustLoadPrompt("prompts/synthesis-user.md"),
	}
}

// PdfExtractionPromptLoader parity.
type PdfExtractionPromptLoader struct {
	Prompt string
}

// NewPdfExtractionPromptLoader parity.
func NewPdfExtractionPromptLoader() PdfExtractionPromptLoader {
	return PdfExtractionPromptLoader{
		Prompt: mustLoadPrompt("prompts/pdf-extraction.md"),
	}
}

// mustLoadPrompt parity.
func mustLoadPrompt(path string) string {
	b, err := promptFS.ReadFile(path)
	if err != nil {
		panic(fmt.Sprintf("failed to load prompt %s: %v", path, err))
	}
	return strings.TrimSpace(string(b))
}
