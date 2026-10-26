package strategy

import (
	"fmt"
	"strings"
	"time"

	coredecimal "github.com/kayis-rahman/swing-trade/go/core/decimal"
	"github.com/kayis-rahman/swing-trade/go/core/domain"
)

// RegimeGateMode mirrors Java's RegimeGateConfig.Mode enum.
type RegimeGateMode string

// Regime gate modes matching Java's enum constants.
const (
	RegimeGateModeCloseAbove RegimeGateMode = "CLOSE_ABOVE"
	RegimeGateModeEMASlopeUp RegimeGateMode = "EMA_SLOPE_UP"
)

// RegimeGateConfig mirrors Java's RegimeGateConfig record with its
// compact-constructor defaults.
type RegimeGateConfig struct {
	IndexEma          int
	Mode              RegimeGateMode
	SlopeLookbackDays int
}

// NewRegimeGateConfig mirrors Java's validation: a non-positive index EMA is
// rejected, a missing mode defaults to CLOSE_ABOVE, and a non-positive slope
// lookback defaults to10 days.
func NewRegimeGateConfig(indexEma int, mode RegimeGateMode, slopeLookbackDays int) (RegimeGateConfig, error) {
	if indexEma <= 0 {
		return RegimeGateConfig{}, fmt.Errorf("indexEma must be positive")
	}
	if mode == "" {
		mode = RegimeGateModeCloseAbove
	}
	if slopeLookbackDays <= 0 {
		slopeLookbackDays = 10
	}
	return RegimeGateConfig{IndexEma: indexEma, Mode: mode, SlopeLookbackDays: slopeLookbackDays}, nil
}

// RegimeGateDefaults mirrors RegimeGateConfig.defaults().
func RegimeGateDefaults() RegimeGateConfig {
	config, _ := NewRegimeGateConfig(200, RegimeGateModeCloseAbove, 10)
	return config
}

// SentimentGateConfig mirrors Java's SentimentGateConfig record with its
// compact-constructor defaults (NEUTRAL minimum,3-day freshness).
type SentimentGateConfig struct {
	MinScore   domain.SentimentScore
	MaxAgeDays int
}

// NewSentimentGateConfig mirrors Java's validation: a missing score defaults
// to NEUTRAL and a non-positive age defaults to3 days (no rejection).
func NewSentimentGateConfig(minScore domain.SentimentScore, maxAgeDays int) SentimentGateConfig {
	if minScore == "" {
		minScore = domain.SentimentScoreNeutral
	}
	if maxAgeDays <= 0 {
		maxAgeDays = 3
	}
	return SentimentGateConfig{MinScore: minScore, MaxAgeDays: maxAgeDays}
}

// SentimentGateDefaults mirrors SentimentGateConfig.defaults().
func SentimentGateDefaults() SentimentGateConfig {
	return NewSentimentGateConfig(domain.SentimentScoreNeutral, 3)
}

// GateOutcome mirrors Java's GateOutcome record, shaped for the
// signals.gate_outcomes JSONB column (gate, blocked, reason, score).
type GateOutcome struct {
	Gate    string               `json:"gate"`
	Blocked bool                 `json:"blocked"`
	Reason  string               `json:"reason"`
	Score   *coredecimal.Decimal `json:"score"`
}

// NewGateOutcome mirrors the Java compact constructor's gate validation.
func NewGateOutcome(gate string, blocked bool, reason string, score *coredecimal.Decimal) (GateOutcome, error) {
	if strings.TrimSpace(gate) == "" {
		return GateOutcome{}, fmt.Errorf("gate cannot be null or blank")
	}
	return GateOutcome{Gate: gate, Blocked: blocked, Reason: reason, Score: score}, nil
}

// GateOutcomePass mirrors GateOutcome.pass (never blocked, no score).
func GateOutcomePass(gate, reason string) GateOutcome {
	outcome, _ := NewGateOutcome(gate, false, reason, nil)
	return outcome
}

// GateOutcomeBlock mirrors GateOutcome.block (always blocked, no score).
func GateOutcomeBlock(gate, reason string) GateOutcome {
	outcome, _ := NewGateOutcome(gate, true, reason, nil)
	return outcome
}

// GateResult mirrors Java's GateEvaluator.GateResult record: the decision
// after the overlay (possibly downgraded to HOLD) plus every gate outcome.
type GateResult struct {
	Decision Decision
	Outcomes []GateOutcome
}

// NewGateResult normalizes nil outcomes to an empty list like Java's record.
func NewGateResult(decision Decision, outcomes []GateOutcome) GateResult {
	if outcomes == nil {
		outcomes = []GateOutcome{}
	}
	return GateResult{Decision: decision, Outcomes: append([]GateOutcome{}, outcomes...)}
}

// AnyBlocked reports whether any recorded outcome blocked the BUY.
func (result GateResult) AnyBlocked() bool {
	for _, outcome := range result.Outcomes {
		if outcome.Blocked {
			return true
		}
	}
	return false
}

// IndexSeries mirrors Java's IndexSeries interface: the fakeable read-only
// view the regime gate needs. A real Nifty-backed implementation is deferred
// to the phase that ingests index candles (Java documents the same call).
type IndexSeries interface {
	HasDataOn(date time.Time) bool
	Close(date time.Time) coredecimal.Decimal
	EMA(period int, date time.Time) coredecimal.Decimal
	EMALookback(period int, date time.Time, lookbackDays int) coredecimal.Decimal
}

// GateEvaluator mirrors Java's GateEvaluator: the composable regime and
// sentiment overlays that can each block a BUY while recording every outcome,
// passes included. It evaluates already-resolved inputs and fetches nothing.
type GateEvaluator struct{}

// ApplyRegimeGate mirrors applyRegimeGate: only BUY decisions with a
// configured gate are evaluated; missing index data skips the gate with a
// recorded pass.
func (GateEvaluator) ApplyRegimeGate(
	decision Decision,
	date time.Time,
	index IndexSeries,
	config *RegimeGateConfig,
) GateResult {
	if decision.Type != domain.SignalTypeBuy || config == nil {
		return NewGateResult(decision, []GateOutcome{})
	}
	if index == nil || !index.HasDataOn(date) {
		return NewGateResult(decision, []GateOutcome{
			GateOutcomePass("regimeGate", "No index data available; gate skipped"),
		})
	}

	var regimeOK bool
	switch config.Mode {
	case RegimeGateModeEMASlopeUp:
		emaNow := index.EMA(config.IndexEma, date)
		emaLookback := index.EMALookback(config.IndexEma, date, config.SlopeLookbackDays)
		regimeOK = emaNow.Cmp(emaLookback) > 0
	default:
		regimeOK = index.Close(date).Cmp(index.EMA(config.IndexEma, date)) > 0
	}

	if regimeOK {
		return NewGateResult(decision, []GateOutcome{
			GateOutcomePass("regimeGate", fmt.Sprintf("Index regime is up (%s)", config.Mode)),
		})
	}
	outcome := GateOutcomeBlock("regimeGate", fmt.Sprintf("Index regime is down (%s); BUY blocked", config.Mode))
	return NewGateResult(downgradeDecision(decision, outcome.Reason), []GateOutcome{outcome})
}

// ApplySentimentGate mirrors applySentimentGate: only configured BUY
// decisions are evaluated; a missing, stale, or too-negative sentiment blocks.
func (GateEvaluator) ApplySentimentGate(
	decision Decision,
	date time.Time,
	sentiment *domain.SentimentResult,
	config *SentimentGateConfig,
) GateResult {
	if decision.Type != domain.SignalTypeBuy || config == nil {
		return NewGateResult(decision, []GateOutcome{})
	}
	if sentiment == nil {
		outcome := GateOutcomeBlock("sentimentGate", "No sentiment available; BUY blocked")
		return NewGateResult(downgradeDecision(decision, outcome.Reason), []GateOutcome{outcome})
	}

	ageDays := int(date.Sub(sentiment.Date).Hours() / 24)
	if ageDays > config.MaxAgeDays {
		outcome := GateOutcomeBlock("sentimentGate",
			fmt.Sprintf("Sentiment is %d days stale (max %d); BUY blocked", ageDays, config.MaxAgeDays))
		return NewGateResult(downgradeDecision(decision, outcome.Reason), []GateOutcome{outcome})
	}

	if sentiment.Score.Rank() < config.MinScore.Rank() {
		outcome := GateOutcomeBlock("sentimentGate",
			fmt.Sprintf("Sentiment %s below minimum %s; BUY blocked", sentiment.Score, config.MinScore))
		return NewGateResult(downgradeDecision(decision, outcome.Reason), []GateOutcome{outcome})
	}

	return NewGateResult(decision, []GateOutcome{
		GateOutcomePass("sentimentGate", fmt.Sprintf("Sentiment %s meets minimum", sentiment.Score)),
	})
}

// ApplyAll mirrors applyAll: regime first, then sentiment over the regime
// result's decision, aggregating every outcome for recording.
func (GateEvaluator) ApplyAll(
	decision Decision,
	date time.Time,
	index IndexSeries,
	regimeConfig *RegimeGateConfig,
	sentiment *domain.SentimentResult,
	sentimentConfig *SentimentGateConfig,
) GateResult {
	var outcomes []GateOutcome
	afterRegime := GateEvaluator{}.ApplyRegimeGate(decision, date, index, regimeConfig)
	outcomes = append(outcomes, afterRegime.Outcomes...)

	afterSentiment := GateEvaluator{}.ApplySentimentGate(afterRegime.Decision, date, sentiment, sentimentConfig)
	outcomes = append(outcomes, afterSentiment.Outcomes...)

	return NewGateResult(afterSentiment.Decision, outcomes)
}

// downgradeDecision mirrors Java's private downgrade: the decision becomes
// HOLD with stop/target cleared and the reasoning prefixed. The score was
// already validated when the decision was created.
func downgradeDecision(decision Decision, reason string) Decision {
	downgraded, _ := NewDecision(
		domain.SignalTypeHold, decision.Score, decision.Rules,
		nil, nil, "Gate-blocked: "+reason,
	)
	return downgraded
}
