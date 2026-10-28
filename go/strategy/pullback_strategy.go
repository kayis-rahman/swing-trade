package strategy

import (
	coredecimal "github.com/kayis-rahman/swing-trade/go/core/decimal"
	"github.com/kayis-rahman/swing-trade/go/core/domain"
)

// PullbackStrategy mirrors Java's PULLBACK type: buy a shallow dip back to the
// pullback EMA inside an established trend EMA uptrend, confirmed by RSI
// recovering out of a dip band.
type PullbackStrategy struct{}

// PullbackType is the stable type identifier.
const PullbackType = "PULLBACK"

// NewPullbackStrategy builds the PULLBACK type.
func NewPullbackStrategy() *PullbackStrategy { return &PullbackStrategy{} }

// Type mirrors Java's type().
func (*PullbackStrategy) Type() string { return PullbackType }

// ParamSchema mirrors Java's paramSchema().
func (*PullbackStrategy) ParamSchema() ParamSchema {
	return NewParamSchema([]ParamDef{
		{Name: "trendEma", Type: ParamTypeInt, Min: decimalPtr("10"), Max: decimalPtr("200"), DefaultValue: 50, Description: "EMA period defining the primary uptrend", Group: "trend"},
		{Name: "pullbackEma", Type: ParamTypeInt, Min: decimalPtr("5"), Max: decimalPtr("100"), DefaultValue: 20, Description: "EMA period the pullback is measured against", Group: "trend"},
		{Name: "touchPct", Type: ParamTypeDecimal, Min: decimalPtr("0.001"), Max: decimalPtr("0.10"), DefaultValue: decimalDefault("0.01"), Description: "Max proximity of low to pullbackEma to count as a touch", Group: "trend"},
		{Name: "touchLookback", Type: ParamTypeInt, Min: decimalPtr("1"), Max: decimalPtr("30"), DefaultValue: 3, Description: "Bars to look back for a pullback-EMA touch and RSI recovery", Group: "trend"},
		{Name: "rsiPeriod", Type: ParamTypeInt, Min: decimalPtr("2"), Max: decimalPtr("50"), DefaultValue: 14, Description: "RSI lookback period", Group: "momentum"},
		{Name: "rsiDip", Type: ParamTypeDecimal, Min: decimalPtr("10"), Max: decimalPtr("50"), DefaultValue: decimalDefault("40"), Description: "RSI level that must have been dipped below", Group: "momentum"},
		{Name: "rsiTrigger", Type: ParamTypeDecimal, Min: decimalPtr("20"), Max: decimalPtr("70"), DefaultValue: decimalDefault("45"), Description: "RSI level the recovery must cross back above", Group: "momentum"},
		{Name: "slopeLookback", Type: ParamTypeInt, Min: decimalPtr("1"), Max: decimalPtr("60"), DefaultValue: 10, Description: "Bars over which trendEma's slope must be positive", Group: "trend"},
		{Name: "requireConfirmation", Type: ParamTypeBool, DefaultValue: false, Description: "If true, close > prior high becomes mandatory instead of scored", Group: "confirmation"},
		{Name: "entryScoreThreshold", Type: ParamTypeDecimal, Min: decimalPtr("0.5"), Max: decimalPtr("1"), DefaultValue: decimalDefault("0.75"), Description: "Minimum score to fire a BUY", Group: "scoring"},
		{Name: "atrPeriod", Type: ParamTypeInt, Min: decimalPtr("2"), Max: decimalPtr("50"), DefaultValue: 14, Description: "ATR lookback period", Group: "exits"},
		{Name: "atrStopMult", Type: ParamTypeDecimal, Min: decimalPtr("0"), Max: decimalPtr("10"), DefaultValue: decimalDefault("2"), Description: "ATR multiplier for the stop distance", Group: "exits"},
		{Name: "rewardRisk", Type: ParamTypeDecimal, Min: decimalPtr("0.5"), Max: decimalPtr("10"), DefaultValue: decimalDefault("2"), Description: "Reward:risk ratio for the target", Group: "exits"},
		{Name: "maxHoldDays", Type: ParamTypeInt, Min: decimalPtr("1"), Max: decimalPtr("120"), DefaultValue: 20, Description: "Maximum holding period in bars", Group: "exits"},
		{Name: "trailAtrMult", Type: ParamTypeDecimal, Min: decimalPtr("0"), Max: decimalPtr("10"), DefaultValue: zeroDecimal(), Description: "ATR multiplier for the trailing stop (0 disables it)", Group: "exits"},
	})
}

// CrossFieldRules mirrors Java's crossFieldRules().
func (*PullbackStrategy) CrossFieldRules() []CrossFieldRule {
	return []CrossFieldRule{
		{Message: "pullbackEma must be less than trendEma", Rule: func(params map[string]any) bool {
			return paramInt(params, "pullbackEma") < paramInt(params, "trendEma")
		}},
		{Message: "rsiDip must be less than rsiTrigger", Rule: func(params map[string]any) bool {
			return paramDecimal(params, "rsiDip").Cmp(paramDecimal(params, "rsiTrigger")) < 0
		}},
		{Message: "touchLookback must be positive", Rule: func(params map[string]any) bool {
			return paramInt(params, "touchLookback") > 0
		}},
		{Message: "slopeLookback must be positive", Rule: func(params map[string]any) bool {
			return paramInt(params, "slopeLookback") > 0
		}},
	}
}

// RequiredIndicators mirrors Java's requiredIndicators().
func (*PullbackStrategy) RequiredIndicators(StrategyParamsView) []IndicatorKey {
	return []IndicatorKey{IndicatorKeyEMA, IndicatorKeyRSI, IndicatorKeyATR, IndicatorKeyLowestLow}
}

// WarmupBars mirrors Java's warmupBars().
func (*PullbackStrategy) WarmupBars(params StrategyParamsView) int {
	return params.GetInt("trendEma") + params.GetInt("slopeLookback")
}

// EvaluateEntry mirrors Java's evaluateEntry().
func (*PullbackStrategy) EvaluateEntry(ctx MarketContext, barIndex int, params StrategyParamsView) (Decision, error) {
	view := ctx.View(barIndex)

	trendEma := params.GetInt("trendEma")
	touchLookback := params.GetInt("touchLookback")
	touchPct := params.GetDecimal("touchPct")
	requireConfirmation := params.GetBool("requireConfirmation")

	trendUp := pullbackTrendUp(view, barIndex, trendEma, params.GetInt("slopeLookback"))
	touchedPullbackEma := pullbackTouched(view, barIndex, params.GetInt("pullbackEma"), touchLookback, touchPct)
	rsiRecovered := pullbackRsiRecovered(view, barIndex, params.GetInt("rsiPeriod"),
		params.GetDecimal("rsiDip"), params.GetDecimal("rsiTrigger"), touchLookback)
	confirmation := barIndex > 0 && view.Close().Cmp(view.High(barIndex-1)) > 0

	rules := []RuleOutcome{
		ruleOutcome("trendUp", trendUp, true,
			"close > EMA"+itoa(trendEma)+" and EMA"+itoa(trendEma)+" slope up over "+
				itoa(params.GetInt("slopeLookback"))+" bars"),
		ruleOutcome("touchedPullbackEma", touchedPullbackEma, false,
			"low within "+touchPct.String()+" of EMA"+itoa(params.GetInt("pullbackEma"))+
				" in last "+itoa(touchLookback)+" bars"),
		ruleOutcome("rsiRecovered", rsiRecovered, false,
			"RSI dipped below "+params.GetDecimal("rsiDip").String()+" then crossed back above "+
				params.GetDecimal("rsiTrigger").String()+" within last "+itoa(touchLookback)+" bars"),
		ruleOutcome("confirmation", confirmation, requireConfirmation, "close > prior bar's high"),
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
		stop, target = pullbackStops(view, params, touchLookback)
	}

	return NewDecision(decisionType, score, rules, stop, target, reasoning)
}

// pullbackTrendUp is the mandatory close-above-trend-EMA plus positive-slope rule.
func pullbackTrendUp(view View, barIndex, trendEma, slopeLookback int) bool {
	closeAboveTrend := view.Close().Cmp(view.EMA(trendEma)) > 0
	slopeStartIndex := maxInt(0, barIndex-slopeLookback)
	trendSlopeUp := view.EMA(trendEma).Cmp(view.EMA(trendEma, slopeStartIndex)) > 0
	return closeAboveTrend && trendSlopeUp
}

// pullbackTouched reports whether a recent low came within touchPct of the
// pullback EMA.
func pullbackTouched(view View, barIndex, pullbackEma, touchLookback int, touchPct coredecimal.Decimal) bool {
	start := maxInt(0, barIndex-touchLookback+1)
	for i := start; i <= barIndex; i++ {
		ema := view.EMA(pullbackEma, i)
		low := view.Low(i)
		denominator := ema
		if denominator.Cmp(coredecimal.Decimal{}) == 0 {
			denominator = oneDecimal()
		}
		proximity := decimalQuotientRounded(absDecimal(ema.Sub(low)), denominator, 10)
		if proximity.Cmp(touchPct) <= 0 {
			return true
		}
	}
	return false
}

// pullbackRsiRecovered reports whether RSI dipped below rsiDip and then crossed
// back above rsiTrigger within the touch lookback.
func pullbackRsiRecovered(view View, barIndex, rsiPeriod int, rsiDip, rsiTrigger coredecimal.Decimal, touchLookback int) bool {
	start := maxInt(0, barIndex-touchLookback+1)
	sawDip := false
	for i := start; i <= barIndex; i++ {
		rsi := view.RSI(rsiPeriod, i)
		if !sawDip && rsi.Cmp(rsiDip) < 0 {
			sawDip = true
			continue
		}
		if sawDip && rsi.Cmp(rsiTrigger) >= 0 {
			return true
		}
	}
	return false
}

// pullbackStops mirrors Java's suggested stop: min(swing low, entry - ATR*mult).
func pullbackStops(view View, params StrategyParamsView, touchLookback int) (*coredecimal.Decimal, *coredecimal.Decimal) {
	price := view.Close()
	atr := view.ATR(params.GetInt("atrPeriod"))
	atrStop := price.Sub(decimalProduct(atr, params.GetDecimal("atrStopMult")))
	swingLow := view.LowestLow(touchLookback)
	stopValue := swingLow
	if atrStop.Cmp(swingLow) < 0 {
		stopValue = atrStop
	}
	risk := price.Sub(stopValue)
	targetValue := price.Add(decimalProduct(risk, params.GetDecimal("rewardRisk")))
	return &stopValue, &targetValue
}

// EvaluateExit mirrors Java's evaluateExit().
func (*PullbackStrategy) EvaluateExit(ctx MarketContext, barIndex int, position OpenPosition, params StrategyParamsView) (ExitDecision, error) {
	view := ctx.View(barIndex)
	signalExit := view.Close().Cmp(view.EMA(params.GetInt("trendEma"))) < 0
	return UniformExitEvaluate(view, position, params, signalExit)
}
