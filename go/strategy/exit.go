package strategy

import (
	"fmt"

	coredecimal "github.com/kayis-rahman/swing-trade/go/core/decimal"
)

// ExitReason mirrors Java's ExitReason enum: checked in priority order each
// backtest bar (STOP_LOSS, TARGET_HIT, SIGNAL_EXIT, TRAILING_STOP,
// TIME_STOP), with MANUAL for live manual closes.
type ExitReason string

// Exit reasons matching Java's enum constants.
const (
	ExitReasonStopLoss      ExitReason = "STOP_LOSS"
	ExitReasonTargetHit     ExitReason = "TARGET_HIT"
	ExitReasonSignalExit    ExitReason = "SIGNAL_EXIT"
	ExitReasonTimeStop      ExitReason = "TIME_STOP"
	ExitReasonTrendBreak    ExitReason = "TREND_BREAK"
	ExitReasonBreakevenStop ExitReason = "BREAKEVEN_STOP"
	ExitReasonTrailingStop  ExitReason = "TRAILING_STOP"
	ExitReasonManual        ExitReason = "MANUAL"
)

// ExitDecision mirrors Java's record: either "no exit" or an exit at a price
// with a reason. Go models the absent reason/price as an empty reason and a
// nil price; the validation keeps both Java messages.
type ExitDecision struct {
	Exit      bool
	Reason    ExitReason
	ExitPrice *coredecimal.Decimal
	Detail    string
}

// NewExitDecision mirrors the Java compact constructor's two-way validation.
func NewExitDecision(exit bool, reason ExitReason, exitPrice *coredecimal.Decimal, detail string) (ExitDecision, error) {
	if exit && (reason == "" || exitPrice == nil) {
		return ExitDecision{}, fmt.Errorf("An exiting decision must carry a reason and exitPrice") //nolint:staticcheck // exact Java message
	}
	if !exit && (reason != "" || exitPrice != nil) {
		return ExitDecision{}, fmt.Errorf("A non-exiting decision must not carry a reason or exitPrice") //nolint:staticcheck // exact Java message
	}
	return ExitDecision{Exit: exit, Reason: reason, ExitPrice: exitPrice, Detail: detail}, nil
}

// ExitHold mirrors ExitDecision.hold().
func ExitHold() ExitDecision {
	decision, _ := NewExitDecision(false, "", nil, "No exit condition met")
	return decision
}

// ExitWith mirrors ExitDecision.exit(reason, exitPrice, detail).
func ExitWith(reason ExitReason, exitPrice coredecimal.Decimal, detail string) ExitDecision {
	price := exitPrice
	decision, _ := NewExitDecision(true, reason, &price, detail)
	return decision
}
