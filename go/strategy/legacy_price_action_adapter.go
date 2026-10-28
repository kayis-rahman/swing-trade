package strategy

import (
	coredecimal "github.com/kayis-rahman/swing-trade/go/core/decimal"
	"github.com/kayis-rahman/swing-trade/go/core/domain"
)

// LegacyPriceActionAdapter mirrors Java's LegacyPriceActionAdapter: it wraps
// the hardcoded price-action rules behind the SignalStrategy SPI as type
// BREAKOUT, with params equal to StrategyParams' constants and an
// entryScoreThreshold of 1.0 - the same "4 of 4" confluence the live engine
// enforces today. It is additive only; the legacy engine keeps running.
type LegacyPriceActionAdapter struct {
	legacyRules TradingStrategy
}

// LegacyPriceActionType is the stable type identifier.
const LegacyPriceActionType = "BREAKOUT"

// NewLegacyPriceActionAdapter builds the adapter over the legacy rules.
func NewLegacyPriceActionAdapter(legacyRules TradingStrategy) *LegacyPriceActionAdapter {
	return &LegacyPriceActionAdapter{legacyRules: legacyRules}
}

// DefaultLegacyPriceActionParams mirrors Java's defaultParams: StrategyParams'
// constants plus conservative exit defaults that the legacy engine computes or
// does not use today.
func DefaultLegacyPriceActionParams() StrategyParamsView {
	return StrategyParamsViewOf(map[string]any{
		"emaFast":             domain.EMAFast,
		"emaSlow":             domain.EMASlow,
		"rsiPeriod":           domain.RSIPeriod,
		"rsiMin":              domain.RSILower,
		"rsiMax":              domain.RSIUpper,
		"volumeMaPeriod":      domain.VolumeMAPeriod,
		"volMult":             domain.VolumeMultiplier,
		"weeklyHighPeriod":    domain.FiftyTwoWeekTradingDays,
		"highProximity":       domain.HighProximity,
		"entryScoreThreshold": oneDecimal(),
		"atrPeriod":           domain.ATRPeriod,
		"atrStopMult":         decimalDefault("2"),
		"rewardRisk":          decimalDefault("2"),
		"maxHoldDays":         20,
		"trailAtrMult":        zeroDecimal(),
	})
}

// Type mirrors Java's type().
func (*LegacyPriceActionAdapter) Type() string { return LegacyPriceActionType }

// ParamSchema mirrors Java's paramSchema().
func (*LegacyPriceActionAdapter) ParamSchema() ParamSchema {
	return NewParamSchema([]ParamDef{
		{Name: "emaFast", Type: ParamTypeInt, Min: decimalPtr("5"), Max: decimalPtr("100"), DefaultValue: domain.EMAFast, Description: "Fast EMA period", Group: "trend"},
		{Name: "emaSlow", Type: ParamTypeInt, Min: decimalPtr("20"), Max: decimalPtr("250"), DefaultValue: domain.EMASlow, Description: "Slow EMA period", Group: "trend"},
		{Name: "rsiPeriod", Type: ParamTypeInt, Min: decimalPtr("2"), Max: decimalPtr("50"), DefaultValue: domain.RSIPeriod, Description: "RSI lookback period", Group: "momentum"},
		{Name: "rsiMin", Type: ParamTypeDecimal, Min: decimalPtr("30"), Max: decimalPtr("80"), DefaultValue: domain.RSILower, Description: "RSI entry band lower bound", Group: "momentum"},
		{Name: "rsiMax", Type: ParamTypeDecimal, Min: decimalPtr("30"), Max: decimalPtr("80"), DefaultValue: domain.RSIUpper, Description: "RSI entry band upper bound", Group: "momentum"},
		{Name: "volumeMaPeriod", Type: ParamTypeInt, Min: decimalPtr("5"), Max: decimalPtr("60"), DefaultValue: domain.VolumeMAPeriod, Description: "Volume moving-average period", Group: "volume"},
		{Name: "volMult", Type: ParamTypeDecimal, Min: decimalPtr("1.0"), Max: decimalPtr("4.0"), DefaultValue: domain.VolumeMultiplier, Description: "Volume surge multiplier over its MA", Group: "volume"},
		{Name: "weeklyHighPeriod", Type: ParamTypeInt, Min: decimalPtr("20"), Max: decimalPtr("300"), DefaultValue: domain.FiftyTwoWeekTradingDays, Description: "Lookback for the weekly (52w) high", Group: "trend"},
		{Name: "highProximity", Type: ParamTypeDecimal, Min: decimalPtr("0.85"), Max: decimalPtr("1.0"), DefaultValue: domain.HighProximity, Description: "Required proximity to the weekly high", Group: "trend"},
		{Name: "entryScoreThreshold", Type: ParamTypeDecimal, Min: decimalPtr("0.5"), Max: decimalPtr("1"), DefaultValue: oneDecimal(), Description: "Minimum score to fire a BUY", Group: "scoring"},
		{Name: "atrPeriod", Type: ParamTypeInt, Min: decimalPtr("2"), Max: decimalPtr("50"), DefaultValue: domain.ATRPeriod, Description: "ATR lookback period", Group: "exits"},
		{Name: "atrStopMult", Type: ParamTypeDecimal, Min: decimalPtr("0"), Max: decimalPtr("10"), DefaultValue: decimalDefault("2"), Description: "ATR multiplier for the stop distance", Group: "exits"},
		{Name: "rewardRisk", Type: ParamTypeDecimal, Min: decimalPtr("0.5"), Max: decimalPtr("10"), DefaultValue: decimalDefault("2"), Description: "Reward:risk ratio for the target", Group: "exits"},
		{Name: "maxHoldDays", Type: ParamTypeInt, Min: decimalPtr("1"), Max: decimalPtr("120"), DefaultValue: 20, Description: "Maximum holding period in bars", Group: "exits"},
		{Name: "trailAtrMult", Type: ParamTypeDecimal, Min: decimalPtr("0"), Max: decimalPtr("10"), DefaultValue: zeroDecimal(), Description: "ATR multiplier for the trailing stop (0 disables it)", Group: "exits"},
	})
}

// CrossFieldRules mirrors Java's crossFieldRules().
func (*LegacyPriceActionAdapter) CrossFieldRules() []CrossFieldRule {
	return []CrossFieldRule{
		{Message: "emaFast must be less than emaSlow", Rule: func(params map[string]any) bool {
			return paramInt(params, "emaFast") < paramInt(params, "emaSlow")
		}},
		{Message: "rsiMin must be less than rsiMax", Rule: func(params map[string]any) bool {
			return paramDecimal(params, "rsiMin").Cmp(paramDecimal(params, "rsiMax")) < 0
		}},
	}
}

// RequiredIndicators mirrors Java's requiredIndicators().
func (*LegacyPriceActionAdapter) RequiredIndicators(StrategyParamsView) []IndicatorKey {
	return []IndicatorKey{
		IndicatorKeyEMA, IndicatorKeyRSI, IndicatorKeyATR,
		IndicatorKeyVolume, IndicatorKeyVolumeMA, IndicatorKeyHighestHigh,
	}
}

// WarmupBars mirrors Java's warmupBars().
func (*LegacyPriceActionAdapter) WarmupBars(params StrategyParamsView) int {
	return maxInt(params.GetInt("emaSlow"), params.GetInt("weeklyHighPeriod"))
}

// EvaluateEntry mirrors Java's evaluateEntry().
func (adapter *LegacyPriceActionAdapter) EvaluateEntry(ctx MarketContext, barIndex int, params StrategyParamsView) (Decision, error) {
	view := ctx.View(barIndex)
	indicators := adapter.readIndicators(view, params)

	rules := []RuleOutcome{
		ruleOutcome("trendAligned", adapter.legacyRules.TrendAligned(indicators), true,
			"price>ema"+itoa(params.GetInt("emaFast"))+">ema"+itoa(params.GetInt("emaSlow"))),
		ruleOutcome("rsiInRange", adapter.legacyRules.RsiInEntryRange(indicators), true,
			"rsi in ["+params.GetDecimal("rsiMin").String()+","+params.GetDecimal("rsiMax").String()+"]"),
		ruleOutcome("volumeSurge", adapter.legacyRules.VolumeSurge(indicators), true,
			"volume > "+params.GetDecimal("volMult").String()+"x volumeMA"),
		ruleOutcome("nearWeeklyHigh", adapter.legacyRules.NearWeeklyHigh(indicators), true,
			"price >= "+params.GetDecimal("highProximity").String()+"x weeklyHigh"),
	}

	score := scoreRules(rules)
	buy := allMandatoryPassed(rules) && score.Cmp(params.GetDecimal("entryScoreThreshold")) >= 0
	decisionType := domain.SignalTypeHold
	reasoning := "Entry rules failed: " + failedRuleKeys(rules)
	if buy {
		decisionType = domain.SignalTypeBuy
		reasoning = "All entry rules passed"
	}

	var stop, target *coredecimal.Decimal
	if buy {
		price := view.Close()
		atr := view.ATR(params.GetInt("atrPeriod"))
		stopValue := price.Sub(decimalProduct(atr, params.GetDecimal("atrStopMult")))
		risk := price.Sub(stopValue)
		targetValue := price.Add(decimalProduct(risk, params.GetDecimal("rewardRisk")))
		stop, target = &stopValue, &targetValue
	}

	return NewDecision(decisionType, score, rules, stop, target, reasoning)
}

// EvaluateExit mirrors Java's evaluateExit(): the legacy signal-exit
// confluence feeds the shared uniform exit evaluator.
func (adapter *LegacyPriceActionAdapter) EvaluateExit(ctx MarketContext, barIndex int, position OpenPosition, params StrategyParamsView) (ExitDecision, error) {
	view := ctx.View(barIndex)
	signalExit := adapter.legacyRules.IsSignalExit(adapter.exitIndicators(view, params))
	return UniformExitEvaluate(view, position, params, signalExit)
}

// ClassifyLegacyStyle reproduces the legacy engine's per-bar BUY/SELL/HOLD
// classification for the golden parity test. It is not part of the SPI.
func (adapter *LegacyPriceActionAdapter) ClassifyLegacyStyle(ctx MarketContext, barIndex int, params StrategyParamsView) (domain.SignalType, error) {
	entry, err := adapter.EvaluateEntry(ctx, barIndex, params)
	if err != nil {
		return "", err
	}
	if entry.Type == domain.SignalTypeBuy {
		return domain.SignalTypeBuy, nil
	}
	view := ctx.View(barIndex)
	if adapter.legacyRules.IsSignalExit(adapter.exitIndicators(view, params)) {
		return domain.SignalTypeSell, nil
	}
	return domain.SignalTypeHold, nil
}

func (adapter *LegacyPriceActionAdapter) readIndicators(view View, params StrategyParamsView) *Indicators {
	weeklyHighPeriod := params.GetInt("weeklyHighPeriod")
	if weeklyHighPeriod > view.BarIndex()+1 {
		weeklyHighPeriod = view.BarIndex() + 1
	}
	return &Indicators{
		Price:      view.Close(),
		Ema20:      view.EMA(params.GetInt("emaFast")),
		Ema50:      view.EMA(params.GetInt("emaSlow")),
		Rsi:        view.RSI(params.GetInt("rsiPeriod")),
		Volume:     view.Volume(),
		VolumeMA:   view.VolumeMA(params.GetInt("volumeMaPeriod")),
		WeeklyHigh: view.HighestHigh(weeklyHighPeriod),
	}
}

func (adapter *LegacyPriceActionAdapter) exitIndicators(view View, params StrategyParamsView) *Indicators {
	return &Indicators{
		Price: view.Close(),
		Ema20: view.EMA(params.GetInt("emaFast")),
		Ema50: view.EMA(params.GetInt("emaSlow")),
		Rsi:   view.RSI(params.GetInt("rsiPeriod")),
	}
}
