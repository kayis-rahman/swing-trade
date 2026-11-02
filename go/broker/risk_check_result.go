package broker

import "fmt"

// RiskCheckResult captures the outcome of a pre-trade risk check.
type RiskCheckResult struct {
	Passed    bool
	Messages  []string
	CheckType string
}

// NewRiskCheckResult creates a result with the given passed state.
// NewRiskCheckResult implements the corresponding risk result operation.
func NewRiskCheckResult(passed bool, message ...string) *RiskCheckResult {
	r := &RiskCheckResult{Passed: passed, Messages: []string{}}
	if len(message) > 0 && message[0] != "" {
		r.Messages = append(r.Messages, message[0])
	}
	return r
}

// AddMessage implements the corresponding risk result operation.
func (r *RiskCheckResult) AddMessage(message string) *RiskCheckResult {
	r.Messages = append(r.Messages, message)
	return r
}

// AddWarning implements the corresponding risk result operation.
func (r *RiskCheckResult) AddWarning(format string, args ...any) *RiskCheckResult {
	r.Passed = false
	r.Messages = append(r.Messages, "WARNING: "+sprintf(format, args))
	return r
}

// AddError implements the corresponding risk result operation.
func (r *RiskCheckResult) AddError(format string, args ...any) *RiskCheckResult {
	r.Passed = false
	r.Messages = append(r.Messages, "ERROR: "+sprintf(format, args))
	return r
}

// AddInfo implements the corresponding risk result operation.
func (r *RiskCheckResult) AddInfo(format string, args ...any) *RiskCheckResult {
	r.Messages = append(r.Messages, "INFO: "+sprintf(format, args))
	return r
}

// HasWarnings implements the corresponding risk result operation.
func (r *RiskCheckResult) HasWarnings() bool {
	for _, m := range r.Messages {
		if len(m) >= 9 && m[:9] == "WARNING: " {
			return true
		}
	}
	return false
}

// HasErrors implements the corresponding risk result operation.
func (r *RiskCheckResult) HasErrors() bool {
	for _, m := range r.Messages {
		if len(m) >= 7 && m[:7] == "ERROR: " {
			return true
		}
	}
	return false
}

// IsPassed implements the corresponding risk result operation.
func (r *RiskCheckResult) IsPassed() bool { return r.Passed }

// GetMessages implements the corresponding risk result operation.
func (r *RiskCheckResult) GetMessages() []string { return r.Messages }

// GetCheckType implements the corresponding risk result operation.
func (r *RiskCheckResult) GetCheckType() string { return r.CheckType }

// SetCheckType implements the corresponding risk result operation.
func (r *RiskCheckResult) SetCheckType(ct string) { r.CheckType = ct }

// SetPassed implements the corresponding risk result operation.
func (r *RiskCheckResult) SetPassed(p bool) { r.Passed = p }

// SetMessages implements the corresponding risk result operation.
func (r *RiskCheckResult) SetMessages(m []string) { r.Messages = m }

func sprintf(format string, args ...any) string {
	// Minimal formatting: support {} placeholders by converting to %s-ish.
	// Delegate to fmt.Sprintf after replacing {} with %v and escaping %.
	if len(args) == 0 {
		return format
	}
	out := format
	// Replace {} with %v one at a time.
	for i := 0; i < len(args); i++ {
		idx := indexOf(out, "{}")
		if idx < 0 {
			break
		}
		out = out[:idx] + "%v" + out[idx+2:]
	}
	// Escape existing percent signs not followed by v/s/d/f? Simpler: double them.
	// But formats may contain %s? Java uses {} mostly. We'll just Sprintf.
	return fmt.Sprintf(out, args...)
}

func indexOf(s, sub string) int {
	for i := 0; i+len(sub) <= len(s); i++ {
		if s[i:i+len(sub)] == sub {
			return i
		}
	}
	return -1
}
