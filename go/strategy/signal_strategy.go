package strategy

import "fmt"

import (
	"github.com/kayis-rahman/swing-trade/go/core/domain"
)

// IndicatorKey for signal strategies mirrors Java's enum.
type IndicatorKey int

const (
	IndicatorKeyEMA IndicatorKey = iota
	IndicatorKeyRSI
	IndicatorKeyATR
	IndicatorKeyVolume
	IndicatorKeyVolumeMA
	IndicatorKeyHighestHigh
	IndicatorKeyLowestLow
	IndicatorKeyBBMiddle
	IndicatorKeyBBUpper
	IndicatorKeyBBLower
	IndicatorKeyKeltnerMiddle
	IndicatorKeyKeltnerUpper
	IndicatorKeyIndexClose
	IndicatorKeyIndexEMA
)

// SignalStrategy mirrors Java interface.
type SignalStrategy interface {
	Type() string
	ParamSchema() ParamSchema
	RequiredIndicators(params StrategyParamsView) []IndicatorKey
	WarmupBars(params StrategyParamsView) int
	EvaluateEntry(ctx MarketContext, barIndex int, params StrategyParamsView) (Decision, error)
	EvaluateExit(ctx MarketContext, barIndex int, position OpenPosition, params StrategyParamsView) (ExitDecision, error)
	CrossFieldRules() []CrossFieldRule
}

// StrategyRegistry minimal placeholder.
type StrategyRegistry struct {
	strategies map[string]SignalStrategy
}

func NewStrategyRegistry() *StrategyRegistry {
	return &StrategyRegistry{strategies: map[string]SignalStrategy{}}
}
func (r *StrategyRegistry) Register(s SignalStrategy) { r.strategies[s.Type()] = s }
func (r *StrategyRegistry) Lookup(typ string) (SignalStrategy, bool) {
	s, ok := r.strategies[typ]
	return s, ok
}

// StrategyResolver minimal placeholder.
type StrategyResolver struct {
	registry *StrategyRegistry
}

func NewStrategyResolver(registry *StrategyRegistry) *StrategyResolver {
	return &StrategyResolver{registry: registry}
}

// ResolvedStrategy placeholder.
type ResolvedStrategy struct {
	Type     string
	Strategy SignalStrategy
	Params   StrategyParamsView
}

func (r *StrategyResolver) Resolve(typ string, params map[string]any) (ResolvedStrategy, error) {
	s, ok := r.registry.Lookup(typ)
	if !ok {
		return ResolvedStrategy{}, fmt.Errorf("unknown strategy type: %s", typ)
	}
	_ = domain.SignalTypeBuy
	return ResolvedStrategy{Type: typ, Strategy: s, Params: StrategyParamsViewOf(params)}, nil
}
