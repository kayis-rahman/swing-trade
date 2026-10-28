package strategy

import (
	"strings"

	"github.com/kayis-rahman/swing-trade/go/core/domain"
)

// IndicatorKey mirrors Java's IndicatorKey enum: the family of indicators a
// SignalStrategy declares it needs. The INDEX_* keys are reserved for the
// relative-strength types and are not produced by any Phase 4 strategy.
type IndicatorKey int

// Indicator families matching Java's enum order.
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

// SignalStrategy mirrors Java's SignalStrategy SPI. Both live signal
// generation (last bar) and backtesting (every bar) call EvaluateEntry and
// EvaluateExit with an explicit bar index, and implementations read bar data
// only through MarketContext.View(barIndex), so the two paths cannot drift and
// cannot look ahead.
type SignalStrategy interface {
	// Type is the unique, stable identifier such as "BREAKOUT".
	Type() string
	// ParamSchema describes this type's configurable parameters.
	ParamSchema() ParamSchema
	// RequiredIndicators lists the indicators this strategy needs for params.
	RequiredIndicators(params StrategyParamsView) []IndicatorKey
	// WarmupBars is the minimum number of leading bars before entry is trusted.
	WarmupBars(params StrategyParamsView) int
	// EvaluateEntry evaluates entry rules at barIndex without look-ahead.
	EvaluateEntry(ctx MarketContext, barIndex int, params StrategyParamsView) (Decision, error)
	// EvaluateExit evaluates exit rules for an open position at barIndex.
	EvaluateExit(ctx MarketContext, barIndex int, position OpenPosition, params StrategyParamsView) (ExitDecision, error)
	// CrossFieldRules lists multi-parameter validation rules; may be empty.
	CrossFieldRules() []CrossFieldRule
}

// StrategyTypeRegistry mirrors Java's StrategyTypeRegistry: lookup of
// registered SignalStrategy implementations by their stable type.
//
//nolint:revive // name mirrors Java's StrategyTypeRegistry
type StrategyTypeRegistry struct {
	byType map[string]SignalStrategy
}

// NewStrategyTypeRegistry builds a registry from the supplied strategies.
func NewStrategyTypeRegistry(strategies ...SignalStrategy) *StrategyTypeRegistry {
	byType := make(map[string]SignalStrategy, len(strategies))
	for _, strategy := range strategies {
		byType[strategy.Type()] = strategy
	}
	return &StrategyTypeRegistry{byType: byType}
}

// FindByType mirrors Java's findByType.
func (registry *StrategyTypeRegistry) FindByType(strategyType string) (SignalStrategy, bool) {
	if registry == nil {
		return nil, false
	}
	strategy, ok := registry.byType[strategyType]
	return strategy, ok
}

// All mirrors Java's all(): every registered strategy.
func (registry *StrategyTypeRegistry) All() []SignalStrategy {
	if registry == nil {
		return nil
	}
	out := make([]SignalStrategy, 0, len(registry.byType))
	for _, strategy := range registry.byType {
		out = append(out, strategy)
	}
	return out
}

// LegacyStrategyRegistry mirrors the name-lookup part of Java's
// StrategyRegistry. Parameter-driven legacy construction (the confluence and
// relative-strength types) remains part of the later resolver slice.
type LegacyStrategyRegistry struct {
	byName map[string]TradingStrategy
}

// NewLegacyStrategyRegistry builds a registry from the supplied strategies.
func NewLegacyStrategyRegistry(strategies ...TradingStrategy) *LegacyStrategyRegistry {
	byName := make(map[string]TradingStrategy, len(strategies))
	for _, strategy := range strategies {
		byName[strategy.Name()] = strategy
	}
	return &LegacyStrategyRegistry{byName: byName}
}

// Resolve mirrors Java's StrategyRegistry.resolve for name-registered types.
func (registry *LegacyStrategyRegistry) Resolve(config domain.StrategyConfig) (TradingStrategy, bool) {
	if registry == nil {
		return nil, false
	}
	strategy, ok := registry.byName[config.StrategyType]
	return strategy, ok
}

// ResolvedStrategyKind identifies which ResolvedStrategy variant is set.
type ResolvedStrategyKind int

// ResolvedStrategy variants mirroring Java's sealed interface.
const (
	ResolvedStrategySignal ResolvedStrategyKind = iota
	ResolvedStrategyLegacy
	ResolvedStrategyUnresolved
)

// ResolvedStrategy mirrors Java's ResolvedStrategy sealed interface. Exactly
// one variant is meaningful, selected by Kind: Signal carries Strategy, Params
// and ResolvedParams; Legacy carries Legacy; Unresolved carries Reason.
type ResolvedStrategy struct {
	Kind           ResolvedStrategyKind
	Strategy       SignalStrategy
	Params         StrategyParamsView
	ResolvedParams map[string]any
	Legacy         TradingStrategy
	Reason         string
}

// StrategyResolver mirrors Java's StrategyResolver: it turns a persisted
// StrategyConfig into a runnable strategy, never returning null and never
// throwing for bad input.
//
//nolint:revive // name mirrors Java's StrategyResolver
type StrategyResolver struct {
	typeRegistry   *StrategyTypeRegistry
	legacyRegistry *LegacyStrategyRegistry
	validator      ParamSchemaValidator
}

// NewStrategyResolver builds a resolver over the two registries.
func NewStrategyResolver(typeRegistry *StrategyTypeRegistry, legacyRegistry *LegacyStrategyRegistry) *StrategyResolver {
	return &StrategyResolver{typeRegistry: typeRegistry, legacyRegistry: legacyRegistry}
}

// Resolve mirrors Java's resolve: signal types first, then legacy names, with
// explicit Unresolved reasons instead of silent drops.
func (resolver *StrategyResolver) Resolve(config *domain.StrategyConfig) ResolvedStrategy {
	if config == nil {
		return ResolvedStrategy{Kind: ResolvedStrategyUnresolved, Reason: "no strategy config supplied"}
	}
	strategyType := config.StrategyType
	if strategy, ok := resolver.typeRegistry.FindByType(strategyType); ok {
		return resolver.resolveSignalStrategy(strategy, *config)
	}
	if legacy, ok := resolver.legacyRegistry.Resolve(*config); ok {
		return ResolvedStrategy{Kind: ResolvedStrategyLegacy, Legacy: legacy}
	}
	return ResolvedStrategy{
		Kind:   ResolvedStrategyUnresolved,
		Reason: "unsupported strategy type " + strategyType,
	}
}

func (resolver *StrategyResolver) resolveSignalStrategy(strategy SignalStrategy, config domain.StrategyConfig) ResolvedStrategy {
	validation := resolver.validator.Validate(strategy.ParamSchema(), strategy.CrossFieldRules(), config.Params())
	if !validation.Valid {
		return ResolvedStrategy{
			Kind:   ResolvedStrategyUnresolved,
			Reason: "invalid parameters for " + strategy.Type() + ": " + strings.Join(validation.Errors, "; "),
		}
	}
	params := StrategyParamsViewOf(validation.ResolvedParams)
	for _, key := range strategy.RequiredIndicators(params) {
		if key == IndicatorKeyIndexClose || key == IndicatorKeyIndexEMA {
			return ResolvedStrategy{
				Kind: ResolvedStrategyUnresolved,
				Reason: strategy.Type() +
					" requires an index series, which live evaluation does not supply yet",
			}
		}
	}
	return ResolvedStrategy{
		Kind:           ResolvedStrategySignal,
		Strategy:       strategy,
		Params:         params,
		ResolvedParams: validation.ResolvedParams,
	}
}
