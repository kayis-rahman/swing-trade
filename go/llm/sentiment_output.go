package llm

import (
	"encoding/json"

	"github.com/kayis-rahman/swing-trade/go/core/domain"
)

// SentimentOutput mirrors Java's com.swingtrade.llm.SentimentOutput.
type SentimentOutput struct {
	Sentiment      domain.SentimentScore `json:"score"`
	Reasoning      string                `json:"summary"`
	Confidence     *float64              `json:"confidence"`
	RedFlags       []string              `json:"red_flags"`
	Catalysts      []string              `json:"catalysts"`
	Source         string                `json:"source,omitempty"`
	DegradedReason string                `json:"-"`
}

// NewSentimentOutput handles parity behavior.
func NewSentimentOutput(sentiment domain.SentimentScore, reasoning string, confidence *float64, redFlags, catalysts []string, source string) SentimentOutput {
	if redFlags == nil {
		redFlags = []string{}
	}
	if catalysts == nil {
		catalysts = []string{}
	}
	if source == "" {
		source = "DEFAULT"
	}
	return SentimentOutput{
		Sentiment:  sentiment,
		Reasoning:  reasoning,
		Confidence: confidence,
		RedFlags:   redFlags,
		Catalysts:  catalysts,
		Source:     source,
	}
}

// MarshalJSON handles parity behavior.
func (s *SentimentOutput) MarshalJSON() ([]byte, error) {
	type alias SentimentOutput
	return json.Marshal(alias(*s))
}
