package strategy

import (
	"math/big"

	coredecimal "github.com/kayis-rahman/swing-trade/go/core/decimal"
	"github.com/kayis-rahman/swing-trade/go/core/domain"
)

// IntradayORBType is the stable type identifier.
const IntradayORBType = "INTRADAY_ORB"

// IntradayVWAPType is the stable type identifier.
const IntradayVWAPType = "INTRADAY_VWAP_PULLBACK"

// IntradayRSI2Type is the stable type identifier.
const IntradayRSI2Type = "INTRADAY_RSI2"

func intradayEmptySchema() ParamSchema { return ParamSchema{Params: []ParamDef{}} }

func mulDecimal(a, b coredecimal.Decimal) coredecimal.Decimal {
	rat := new(big.Rat).Mul(ratOf(a), ratOf(b))
	rounded, err := coredecimal.RoundHalfUp(rat.FloatString(12), 4)
	if err != nil {
		return coredecimal.Decimal{}
	}
	d, err := coredecimal.Parse(rounded)
	if err != nil {
		return coredecimal.Decimal{}
	}
	return d
}

func intradayDecision(buy bool, key, detail string, price coredecimal.Decimal) Decision {
	rule, _ := NewRuleOutcome(key, buy, mustRuleDecimal("1"), true, detail)
	score := mustRuleDecimal("0")
	if buy {
		score = mustRuleDecimal("1")
	}
	signalType := domain.SignalTypeHold
	var stop, target *coredecimal.Decimal
	if buy {
		signalType = domain.SignalTypeBuy
		s := mulDecimal(price, mustRuleDecimal("0.99"))
		t := mulDecimal(price, mustRuleDecimal("1.02"))
		stop = &s
		target = &t
	}
	decision, _ := NewDecision(signalType, score, []RuleOutcome{rule}, stop, target, detail)
	return decision
}

// --- IntradayOpeningRangeBreakoutStrategy ---

// IntradayOpeningRangeBreakoutStrategy mirrors Java.
type IntradayOpeningRangeBreakoutStrategy struct{}

// Type mirrors Java.
func (IntradayOpeningRangeBreakoutStrategy) Type() string { return IntradayORBType }

// ParamSchema mirrors Java.
func (IntradayOpeningRangeBreakoutStrategy) ParamSchema() ParamSchema {
	return intradayEmptySchema()
}

// RequiredIndicators mirrors Java.
func (IntradayOpeningRangeBreakoutStrategy) RequiredIndicators(StrategyParamsView) []IndicatorKey {
	return []IndicatorKey{}
}

// WarmupBars mirrors Java.
func (IntradayOpeningRangeBreakoutStrategy) WarmupBars(StrategyParamsView) int { return 3 }

// CrossFieldRules mirrors Java.
func (IntradayOpeningRangeBreakoutStrategy) CrossFieldRules() []CrossFieldRule {
	return []CrossFieldRule{}
}

// EvaluateEntry mirrors Java.
func (IntradayOpeningRangeBreakoutStrategy) EvaluateEntry(ctx MarketContext, index int, params StrategyParamsView) (Decision, error) {
	view := ctx.View(index)
	first := index
	for first > 0 && view.Date(first-1).Equal(view.Date()) {
		first--
	}
	if index-first < 2 {
		return intradayDecision(false, "openingRangeBreakout", "Opening range not complete", view.Close()), nil
	}
	st := view.SessionTime(first)
	if st == nil || st.Hour() != 9 || st.Minute() != 15 {
		return intradayDecision(false, "openingRangeBreakout", "Opening bars unavailable", view.Close()), nil
	}
	st2 := view.SessionTime(first + 1)
	if st2 == nil || st2.Hour() != 9 || st2.Minute() != 30 {
		return intradayDecision(false, "openingRangeBreakout", "Opening bars unavailable", view.Close()), nil
	}
	rangeHigh := view.High(first)
	if view.High(first+1).Cmp(rangeHigh) > 0 {
		rangeHigh = view.High(first + 1)
	}
	breakout := view.Close().Cmp(rangeHigh) > 0
	return intradayDecision(breakout, "openingRangeBreakout", "Close above first two 15-minute bars' high", view.Close()), nil
}

// EvaluateExit mirrors Java.
func (IntradayOpeningRangeBreakoutStrategy) EvaluateExit(ctx MarketContext, index int, position OpenPosition, params StrategyParamsView) (ExitDecision, error) {
	close := ctx.View(index).Close()
	if close.Cmp(position.StopLoss) <= 0 {
		return ExitWith(ExitReasonStopLoss, close, "Intraday stop"), nil
	}
	if close.Cmp(position.Target) >= 0 {
		return ExitWith(ExitReasonTargetHit, close, "Intraday target"), nil
	}
	return ExitHold(), nil
}

// --- IntradayVwapPullbackStrategy ---

// IntradayVwapPullbackStrategy mirrors Java.
type IntradayVwapPullbackStrategy struct{}

// Type mirrors Java.
func (IntradayVwapPullbackStrategy) Type() string { return IntradayVWAPType }

// ParamSchema mirrors Java.
func (IntradayVwapPullbackStrategy) ParamSchema() ParamSchema { return intradayEmptySchema() }

// RequiredIndicators mirrors Java.
func (IntradayVwapPullbackStrategy) RequiredIndicators(StrategyParamsView) []IndicatorKey {
	return []IndicatorKey{}
}

// WarmupBars mirrors Java.
func (IntradayVwapPullbackStrategy) WarmupBars(StrategyParamsView) int { return 2 }

// CrossFieldRules mirrors Java.
func (IntradayVwapPullbackStrategy) CrossFieldRules() []CrossFieldRule { return []CrossFieldRule{} }

// EvaluateEntry mirrors Java.
func (IntradayVwapPullbackStrategy) EvaluateEntry(ctx MarketContext, index int, params StrategyParamsView) (Decision, error) {
	view := ctx.View(index)
	first := index
	for first > 0 && view.Date(first-1).Equal(view.Date()) {
		first--
	}
	if index == first {
		return intradayDecision(false, "vwapReclaim", "Need a prior session bar", view.Close()), nil
	}
	pv := new(big.Float)
	volume := new(big.Float)
	priorPv := new(big.Float)
	priorVolume := new(big.Float)
	for i := first; i <= index; i++ {
		h, _ := new(big.Float).SetString(view.High(i).String())
		l, _ := new(big.Float).SetString(view.Low(i).String())
		c, _ := new(big.Float).SetString(view.Close(i).String())
		v, _ := new(big.Float).SetString(view.Volume(i).String())
		typical := new(big.Float).Add(h, l)
		typical.Add(typical, c)
		typical.Quo(typical, big.NewFloat(3))
		pv.Add(pv, new(big.Float).Mul(typical, v))
		volume.Add(volume, v)
		if i == index-1 {
			priorPv = new(big.Float).Set(pv)
			priorVolume = new(big.Float).Set(volume)
		}
	}
	if priorVolume.Sign() == 0 {
		return intradayDecision(false, "vwapReclaim", "No prior session volume", view.Close()), nil
	}
	priorVwap := new(big.Float).Quo(priorPv, priorVolume)
	vwap := new(big.Float).Quo(pv, volume)
	priorClose, _ := new(big.Float).SetString(view.Close(index - 1).String())
	currClose, _ := new(big.Float).SetString(view.Close().String())
	reclaim := priorClose.Cmp(priorVwap) < 0 && currClose.Cmp(vwap) >= 0
	return intradayDecision(reclaim, "vwapReclaim", "Previous close below and current close at/above session VWAP", view.Close()), nil
}

// EvaluateExit mirrors Java.
func (IntradayVwapPullbackStrategy) EvaluateExit(ctx MarketContext, index int, position OpenPosition, params StrategyParamsView) (ExitDecision, error) {
	close := ctx.View(index).Close()
	if close.Cmp(position.StopLoss) <= 0 {
		return ExitWith(ExitReasonStopLoss, close, "Intraday stop"), nil
	}
	if close.Cmp(position.Target) >= 0 {
		return ExitWith(ExitReasonTargetHit, close, "Intraday target"), nil
	}
	return ExitHold(), nil
}

// --- IntradayRsi2Strategy ---

// IntradayRsi2Strategy mirrors Java.
type IntradayRsi2Strategy struct{}

// Type mirrors Java.
func (IntradayRsi2Strategy) Type() string { return IntradayRSI2Type }

// ParamSchema mirrors Java.
func (IntradayRsi2Strategy) ParamSchema() ParamSchema { return intradayEmptySchema() }

// RequiredIndicators mirrors Java.
func (IntradayRsi2Strategy) RequiredIndicators(StrategyParamsView) []IndicatorKey {
	return []IndicatorKey{}
}

// WarmupBars mirrors Java.
func (IntradayRsi2Strategy) WarmupBars(StrategyParamsView) int { return 3 }

// CrossFieldRules mirrors Java.
func (IntradayRsi2Strategy) CrossFieldRules() []CrossFieldRule { return []CrossFieldRule{} }

func sessionRsi2(view View, index int) coredecimal.Decimal {
	gains := new(big.Float)
	losses := new(big.Float)
	for i := index - 1; i >= index-2; i-- {
		a, _ := new(big.Float).SetString(view.Close(i + 1).String())
		b, _ := new(big.Float).SetString(view.Close(i).String())
		change := new(big.Float).Sub(a, b)
		if change.Sign() > 0 {
			gains.Add(gains, change)
		} else {
			losses.Add(losses, new(big.Float).Abs(change))
		}
	}
	total := new(big.Float).Add(gains, losses)
	if total.Sign() == 0 {
		// 50
		d, _ := coredecimal.Parse("50")
		return d
	}
	rsiRat := new(big.Float).Mul(gains, big.NewFloat(100))
	rsiRat.Quo(rsiRat, total)
	rounded, _ := coredecimal.RoundHalfUp(rsiRat.Text('f', 12), 8)
	d, _ := coredecimal.Parse(rounded)
	return d
}

// EvaluateEntry mirrors Java.
func (IntradayRsi2Strategy) EvaluateEntry(ctx MarketContext, index int, params StrategyParamsView) (Decision, error) {
	view := ctx.View(index)
	first := index
	for first > 0 && view.Date(first-1).Equal(view.Date()) {
		first--
	}
	if index-first < 2 {
		return intradayDecision(false, "rsi2Oversold", "Need two same-session price changes", view.Close()), nil
	}
	rsi := sessionRsi2(view, index)
	buy := rsi.Cmp(mustRuleDecimal("10")) < 0
	return intradayDecision(buy, "rsi2Oversold", "Session RSI(2) below 10", view.Close()), nil
}

// EvaluateExit mirrors Java.
func (IntradayRsi2Strategy) EvaluateExit(ctx MarketContext, index int, position OpenPosition, params StrategyParamsView) (ExitDecision, error) {
	view := ctx.View(index)
	close := view.Close()
	if close.Cmp(position.StopLoss) <= 0 {
		return ExitWith(ExitReasonStopLoss, close, "Intraday stop"), nil
	}
	if close.Cmp(position.Target) >= 0 {
		return ExitWith(ExitReasonTargetHit, close, "Intraday target"), nil
	}
	first := index
	for first > 0 && view.Date(first-1).Equal(view.Date()) {
		first--
	}
	if index-first >= 2 {
		rsi := sessionRsi2(view, index)
		if rsi.Cmp(mustRuleDecimal("60")) >= 0 {
			return ExitWith(ExitReasonSignalExit, close, "Session RSI(2) recovered"), nil
		}
	}
	return ExitHold(), nil
}
