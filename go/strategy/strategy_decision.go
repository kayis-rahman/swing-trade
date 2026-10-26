package strategy

import (
	"fmt"
	"strings"

	coredecimal "github.com/kayis-rahman/swing-trade/go/core/decimal"
	"github.com/kayis-rahman/swing-trade/go/core/domain"
)

// RuleOutcome mirrors Java's RuleOutcome record: one named rule's result
// within a decision, including the mandatory flag required by plan §3.2.
type RuleOutcome struct {
	Key       string
	Passed    bool
	Weight    coredecimal.Decimal
	Mandatory bool
	Detail    string
}

// NewRuleOutcome mirrors the Java record's compact-constructor validation.
func NewRuleOutcome(key string, passed bool, weight coredecimal.Decimal, mandatory bool, detail string) (RuleOutcome, error) {
	if strings.TrimSpace(key) == "" {
		return RuleOutcome{}, fmt.Errorf("Rule key cannot be null or blank") //nolint:staticcheck // exact Java message
	}
	zero := coredecimal.Decimal{}
	if weight.Cmp(zero) < 0 {
		return RuleOutcome{}, fmt.Errorf("Rule weight must be non-negative") //nolint:staticcheck // exact Java message
	}
	return RuleOutcome{Key: key, Passed: passed, Weight: weight, Mandatory: mandatory, Detail: detail}, nil
}

// Decision mirrors Java's strategy.StrategyDecision record: the scored outcome
// of one entry evaluation. Entry evaluation never produces SELL.
type Decision struct {
	Type            domain.SignalType
	Score           coredecimal.Decimal
	Rules           []RuleOutcome
	SuggestedStop   *coredecimal.Decimal
	SuggestedTarget *coredecimal.Decimal
	Reasoning       string
}

// NewDecision mirrors the Java StrategyDecision compact constructor: the type is
// required and the score must sit in [0,1]; nil rules become an immutable
// empty list.
func NewDecision(
	signalType domain.SignalType,
	score coredecimal.Decimal,
	rules []RuleOutcome,
	suggestedStop *coredecimal.Decimal,
	suggestedTarget *coredecimal.Decimal,
	reasoning string,
) (Decision, error) {
	if signalType == "" {
		return Decision{}, fmt.Errorf("Decision type cannot be null") //nolint:staticcheck // exact Java message
	}
	zero := coredecimal.Decimal{}
	one, err := coredecimal.Parse("1")
	if err != nil {
		return Decision{}, err
	}
	if score.Cmp(zero) < 0 || score.Cmp(one) > 0 {
		return Decision{}, fmt.Errorf("Score must be in [0,1]: %s", score.String()) //nolint:staticcheck // exact Java message
	}
	return Decision{
		Type:            signalType,
		Score:           score,
		Rules:           append([]RuleOutcome{}, rules...),
		SuggestedStop:   suggestedStop,
		SuggestedTarget: suggestedTarget,
		Reasoning:       reasoning,
	}, nil
}
