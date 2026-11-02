package pipeline

import (
	"context"

	"github.com/kayis-rahman/swing-trade/go/api/orchestrator"
	"github.com/kayis-rahman/swing-trade/go/strategy"
)

// SignalOutcome is the pipeline's view of a variant signal outcome that feeds
// the arbiter. It is a thin alias of strategy.VariantSignalOutcome.
type SignalOutcome = strategy.VariantSignalOutcome

// Candidate is a competing BUY candidate for the arbiter.
type Candidate = strategy.Candidate

// ArbiterRule selects an arbitration rule.
type ArbiterRule = strategy.ArbitrationRule

const (
	RuleHighestConfidence = strategy.ArbitrationRuleHighestConfidence
	RuleEvidenceRanked    = strategy.ArbitrationRuleEvidenceRanked
)

// StageResult aliases the orchestrator stage result so pipeline stages integrate
// with the real orchestrator.
type StageResult = orchestrator.StageExecutionResult

// ArbiterPick wraps strategy.PickFull, keeping a thin adapter boundary so callers
// do not need to deal with nullable pointers.
func ArbiterPick(outcomes []SignalOutcome, rule ArbiterRule, evidence map[string]float64,
	openPositionVariants map[string]struct{}, champion *string) *SignalOutcome {
	return strategy.PickFull(outcomes, rule, evidence, &openPositionVariants, champion)
}

// ArbiterPickVariant wraps strategy.PickVariant.
func ArbiterPickVariant(candidates []Candidate, rule ArbiterRule, evidence map[string]float64) *string {
	return strategy.PickVariant(candidates, rule, evidence)
}

// SignalPersister is the port used by the pipeline to persist produced signals.
type SignalPersister interface {
	Persist(ctx context.Context, symbol string, outcome SignalOutcome) error
}

// Pipeline coordinates stage execution inputs that feed the arbiter; it does not
// own broker logic. Consumers drive StageRunner values against the orchestrator.
type Pipeline struct {
	Persister SignalPersister
}

func New(persister SignalPersister) *Pipeline {
	return &Pipeline{Persister: persister}
}
