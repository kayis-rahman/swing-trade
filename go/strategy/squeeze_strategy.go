package strategy

import (
	coredecimal "github.com/kayis-rahman/swing-trade/go/core/decimal"
	"github.com/kayis-rahman/swing-trade/go/core/domain"
)

// SqueezeStrategy mirrors Java's SQUEEZE type: buy the expansion out of a
// Bollinger-Band volatility contraction, confirmed by a volume surge and an
// established uptrend.
type SqueezeStrategy struct{}

// SqueezeType is the stable type identifier.
const SqueezeType = "SQUEEZE"

// Squeeze constants matching Java's private static fields.
var (
	squeezeBBK       = decimalDefault("2")
	squeezeKCATRMult = decimalDefault("1.5")
)

// NewSqueezeStrategy builds the SQUEEZE type.
func NewSqueezeStrategy() *SqueezeStrategy { return &SqueezeStrategy{} }

// Type mirrors Java's type().
func (*SqueezeStrategy) Type() string { return SqueezeType }

// ParamSchema mirrors Java's paramSchema().
func (*SqueezeStrategy) ParamSchema() ParamSchema {
	return NewParamSchema([]ParamDef{
		{Name: "bbPeriod", Type: ParamTypeInt, Min: decimalPtr("5"), Max: decimalPtr("60"), DefaultValue: 20, Description: "Bollinger Bands period", Group: "volatility"},
		{Name: "trendEma", Type: ParamTypeInt, Min: decimalPtr("10"), Max: decimalPtr("200"), DefaultValue: 50, Description: "EMA period defining the required uptrend", Group: "trend"},
		{Name: "squeezeDefinition", Type: ParamTypeEnum, DefaultValue: "BB_PCTILE", Description: "Squeeze detection: BB_PCTILE or BB_IN_KC", Group: "volatility"},
		{Name: "squeezeLookback", Type: ParamTypeInt, Min: decimalPtr("20"), Max: decimalPtr("500"), DefaultValue: 120, Description: "Bars over which the BB-width percentile is computed", Group: "volatility"},
		{Name: "squeezePctile", Type: ParamTypeDecimal, Min: decimalPtr("0"), Max: decimalPtr("100"), DefaultValue: decimalDefault("20"), Description: "Max BB-width percentile counted as a squeeze", Group: "volatility"},
		{Name: "squeezeRecency", Type: ParamTypeInt, Min: decimalPtr("1"), Max: decimalPtr("60"), DefaultValue: 5, Description: "How many recent bars are searched for a squeeze condition", Group: "volatility"},
		{Name: "volumeMaPeriod", Type: ParamTypeInt, Min: decimalPtr("5"), Max: decimalPtr("60"), DefaultValue: 20, Description: "Volume moving-average period", Group: "volume"},
		{Name: "volMult", Type: ParamTypeDecimal, Min: decimalPtr("1.0"), Max: decimalPtr("4.0"), DefaultValue: decimalDefault("1.5"), Description: "Volume surge multiplier over its MA", Group: "volume"},
		{Name: "entryScoreThreshold", Type: ParamTypeDecimal, Min: decimalPtr("0.5"), Max: decimalPtr("1"), DefaultValue: decimalDefault("0.75"), Description: "Minimum score to fire a BUY", Group: "scoring"},
		{Name: "atrPeriod", Type: ParamTypeInt, Min: decimalPtr("2"), Max: decimalPtr("50"), DefaultValue: 14, Description: "ATR lookback period", Group: "exits"},
		{Name: "atrStopMult", Type: ParamTypeDecimal, Min: decimalPtr("0"), Max: decimalPtr("10"), DefaultValue: decimalDefault("2"), Description: "ATR multiplier for the stop distance", Group: "exits"},
		{Name: "rewardRisk", Type: ParamTypeDecimal, Min: decimalPtr("0.5"), Max: decimalPtr("10"), DefaultValue: decimalDefault("2"), Description: "Reward:risk ratio for the target", Group: "exits"},
		{Name: "maxHoldDays", Type: ParamTypeInt, Min: decimalPtr("1"), Max: decimalPtr("120"), DefaultValue: 20, Description: "Maximum holding period in bars", Group: "exits"},
		{Name: "trailAtrMult", Type: ParamTypeDecimal, Min: decimalPtr("0"), Max: decimalPtr("10"), DefaultValue: zeroDecimal(), Description: "ATR multiplier for the trailing stop (0 disables it)", Group: "exits"},
	})
}

// CrossFieldRules mirrors Java's crossFieldRules().
func (*SqueezeStrategy) CrossFieldRules() []CrossFieldRule {
	return []CrossFieldRule{
		{Message: "squeezeDefinition must be BB_PCTILE or BB_IN_KC", Rule: func(params map[string]any) bool {
			value, _ := params["squeezeDefinition"].(string)
			return value == "BB_PCTILE" || value == "BB_IN_KC"
		}},
		{Message: "squeezePctile must be in [0,100]", Rule: func(params map[string]any) bool {
			value := paramDecimal(params, "squeezePctile")
			return value.Cmp(zeroDecimal()) >= 0 && value.Cmp(decimalDefault("100")) <= 0
		}},
		{Message: "squeezeRecency must be <= squeezeLookback", Rule: func(params map[string]any) bool {
			return paramInt(params, "squeezeRecency") <= paramInt(params, "squeezeLookback")
		}},
	}
}

// RequiredIndicators mirrors Java's requiredIndicators().
func (*SqueezeStrategy) RequiredIndicators(StrategyParamsView) []IndicatorKey {
	return []IndicatorKey{
		IndicatorKeyEMA, IndicatorKeyATR, IndicatorKeyVolume, IndicatorKeyVolumeMA,
		IndicatorKeyBBMiddle, IndicatorKeyBBUpper, IndicatorKeyBBLower, IndicatorKeyKeltnerUpper,
	}
}

// WarmupBars mirrors Java's warmupBars().
func (*SqueezeStrategy) WarmupBars(params StrategyParamsView) int {
	return maxInt(params.GetInt("trendEma"), params.GetInt("squeezeLookback")) + params.GetInt("bbPeriod")
}

// EvaluateEntry mirrors Java's evaluateEntry().
func (*SqueezeStrategy) EvaluateEntry(ctx MarketContext, barIndex int, params StrategyParamsView) (Decision, error) {
	view := ctx.View(barIndex)

	bbPeriod := params.GetInt("bbPeriod")
	trendEma := params.GetInt("trendEma")
	squeezeDefinition := params.GetString("squeezeDefinition")
	squeezeLookback := minInt(params.GetInt("squeezeLookback"), barIndex+1)
	squeezePctile := params.GetDecimal("squeezePctile")
	squeezeRecency := params.GetInt("squeezeRecency")
	volumeMaPeriod := params.GetInt("volumeMaPeriod")
	volMult := params.GetDecimal("volMult")

	trendAligned := view.Close().Cmp(view.EMA(trendEma)) > 0

	var squeezed bool
	if squeezeDefinition == "BB_IN_KC" {
		squeezed = recentBBInsideKeltner(view, barIndex, bbPeriod, squeezeRecency)
	} else {
		squeezed = recentBBWidthPercentileBelow(view, barIndex, bbPeriod, squeezeLookback, squeezeRecency, squeezePctile)
	}

	expansion := view.Close().Cmp(view.BBUpper(bbPeriod, squeezeBBK)) > 0 ||
		view.Close().Cmp(view.KeltnerUpper(bbPeriod, squeezeKCATRMult)) > 0
	volumeSurge := CompareProduct(view.VolumeMA(volumeMaPeriod), volMult, view.Volume()) < 0

	rules := []RuleOutcome{
		ruleOutcome("trendAligned", trendAligned, true, "close > EMA"+itoa(trendEma)),
		ruleOutcome("squeezed", squeezed, true,
			"squeeze ("+squeezeDefinition+") within last "+itoa(squeezeRecency)+" bars"),
		ruleOutcome("expansion", expansion, true, "close > upper BB or > upper Keltner"),
		ruleOutcome("volumeSurge", volumeSurge, true, "volume > "+volMult.String()+"x volumeMA"),
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

func recentBBInsideKeltner(view View, barIndex, bbPeriod, recency int) bool {
	start := maxInt(0, barIndex-recency+1)
	for i := start; i <= barIndex; i++ {
		if view.BBUpper(bbPeriod, squeezeBBK, i).Cmp(view.KeltnerUpper(bbPeriod, squeezeKCATRMult, i)) <= 0 {
			return true
		}
	}
	return false
}

func recentBBWidthPercentileBelow(view View, barIndex, bbPeriod, lookback, recency int, pctileThreshold coredecimal.Decimal) bool {
	start := maxInt(0, barIndex-recency+1)
	for i := start; i <= barIndex; i++ {
		percentile := widthPercentile(view, i, bbPeriod, lookback)
		if percentile.Cmp(pctileThreshold) <= 0 {
			return true
		}
	}
	return false
}

// widthPercentile is the percentile rank (0-100) of the width at index within
// the trailing lookback bars, rounded at scale 6 HALF_UP like Java.
func widthPercentile(view View, index, bbPeriod, lookback int) coredecimal.Decimal {
	windowLookback := minInt(lookback, index+1)
	start := maxInt(0, index-windowLookback+1)
	currentWidth := view.bbWidthNum(bbPeriod, squeezeBBK, index)
	countLessOrEqual := 0
	total := 0
	for i := start; i <= index; i++ {
		total++
		if view.bbWidthNum(bbPeriod, squeezeBBK, i).compare(currentWidth) <= 0 {
			countLessOrEqual++
		}
	}
	if total == 0 {
		return zeroDecimal()
	}
	countDecimal := decimalDefault(itoa(countLessOrEqual))
	hundred := decimalDefault("100")
	totalDecimal := decimalDefault(itoa(total))
	return decimalQuotientRounded(decimalProduct(countDecimal, hundred), totalDecimal, 6)
}

// EvaluateExit mirrors Java's evaluateExit().
func (*SqueezeStrategy) EvaluateExit(ctx MarketContext, barIndex int, position OpenPosition, params StrategyParamsView) (ExitDecision, error) {
	view := ctx.View(barIndex)
	signalExit := view.Close().Cmp(view.EMA(params.GetInt("trendEma"))) < 0
	return UniformExitEvaluate(view, position, params, signalExit)
}
