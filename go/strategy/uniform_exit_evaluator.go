package strategy

import (
	"fmt"

	coredecimal "github.com/kayis-rahman/swing-trade/go/core/decimal"
)

// UniformExitEvaluate applies the shared stop, target, signal, trailing, and
// time exits to one bar, matching Java's UniformExitEvaluator.
func UniformExitEvaluate(view View, position OpenPosition, params StrategyParamsView, signalExitTriggered bool) (ExitDecision, error) {
	low := view.Low()
	high := view.High()
	open := view.Open()
	closev := view.Close()

	if low.Cmp(position.StopLoss) <= 0 {
		exitPrice := position.StopLoss
		if open.Cmp(position.StopLoss) <= 0 {
			exitPrice = open
		}
		return ExitWith(ExitReasonStopLoss, exitPrice, fmt.Sprintf("Bar low %s breached stop loss %s", low, position.StopLoss)), nil
	}

	if high.Cmp(position.Target) >= 0 {
		exitPrice := position.Target
		if open.Cmp(position.Target) >= 0 {
			exitPrice = open
		}
		return ExitWith(ExitReasonTargetHit, exitPrice, fmt.Sprintf("Bar high %s reached target %s", high, position.Target)), nil
	}

	if signalExitTriggered {
		return ExitWith(ExitReasonSignalExit, closev, "Strategy signal-exit rule triggered"), nil
	}

	trailAtrMult := params.GetDecimal("trailAtrMult")
	if trailAtrMult.Cmp(coredecimal.Decimal{}) > 0 {
		atrPeriod := params.GetInt("atrPeriod")
		atr := view.ATR(atrPeriod)
		hwm := position.HighWaterMark
		if closev.Cmp(hwm) > 0 {
			hwm = closev
		}
		dTrail, _ := newDecimalNum(trailAtrMult.String())
		dAtr, _ := newDecimalNum(atr.String())
		dHwm, _ := newDecimalNum(hwm.String())
		product, _ := dAtr.multiply(dTrail).divide(decimalNumFromInt(1)) // exact
		trailing := dHwm.sub(product)
		trailingCore, _ := coredecimal.Parse(trailing.String())
		if closev.Cmp(trailingCore) <= 0 {
			return ExitWith(ExitReasonTrailingStop, closev, fmt.Sprintf("Close %s breached trailing stop %s", closev, trailingCore)), nil
		}
	}

	maxHoldDays := params.GetInt("maxHoldDays")
	if view.BarIndex()-position.EntryIndex >= maxHoldDays {
		return ExitWith(ExitReasonTimeStop, closev, fmt.Sprintf("Held %d bars, max is %d", view.BarIndex()-position.EntryIndex, maxHoldDays)), nil
	}

	return ExitHold(), nil
}

// UniformExitEvaluatorEvaluate is the Java-style alias for UniformExitEvaluate.
func UniformExitEvaluatorEvaluate(view View, position OpenPosition, params StrategyParamsView, signalExitTriggered bool) (ExitDecision, error) {
	return UniformExitEvaluate(view, position, params, signalExitTriggered)
}

// UniformExitEvaluator exposes Evaluate for callers that keep a Java-shaped
// evaluator instance.
type UniformExitEvaluator struct{}

// Evaluate delegates to UniformExitEvaluate.
func (UniformExitEvaluator) Evaluate(view View, position OpenPosition, params StrategyParamsView, signalExitTriggered bool) (ExitDecision, error) {
	return UniformExitEvaluate(view, position, params, signalExitTriggered)
}
