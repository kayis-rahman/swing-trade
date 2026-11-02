package llm

// SynthesisOutput mirrors Java's com.swingtrade.llm.SynthesisOutput.
type SynthesisOutput struct {
	Narrative         string   `json:"narrative"`
	Recommendation    string   `json:"recommendation"`
	Confidence        *float64 `json:"confidence"`
	KeyDrivers        []string `json:"keyDrivers"`
	BullishFactors    []string `json:"bullishFactors"`
	BearishFactors    []string `json:"bearishFactors"`
	ConflictDetected  bool     `json:"conflictDetected"`
	EventRiskDetected bool     `json:"eventRiskDetected"`
	EventRiskReason   string   `json:"eventRiskReason"`
}

// NewSynthesisOutput parity.
func NewSynthesisOutput() SynthesisOutput {
	return SynthesisOutput{KeyDrivers: []string{}, BullishFactors: []string{}, BearishFactors: []string{}}
}

// ParseSynthesisResponse extracts JSON and unmarshals.
func ParseSynthesisResponse(text string) (SynthesisOutput, error) {
	content := extractJSONFromReasoning(text)
	out := NewSynthesisOutput()
	if content == "" {
		return out, nil
	}
	var raw map[string]any
	_ = raw
	if err := unmarshalJSON(content, &out); err != nil {
		return out, err
	}
	if out.KeyDrivers == nil {
		out.KeyDrivers = []string{}
	}
	if out.BullishFactors == nil {
		out.BullishFactors = []string{}
	}
	if out.BearishFactors == nil {
		out.BearishFactors = []string{}
	}
	return out, nil
}

func unmarshalJSON(content string, v any) error {
	// local wrapper to avoid import cycle; use encoding/json
	return jsonUnmarshal([]byte(content), v)
}
