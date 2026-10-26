package strategy

import "testing"

func TestExitDecisionValidationMatchesJava(t *testing.T) {
	price := mustParseDecisionDecimal(t, "99.5")
	if _, err := NewExitDecision(true, "", &price, "x"); err == nil ||
		err.Error() != "An exiting decision must carry a reason and exitPrice" {
		t.Fatalf("exit without reason = %v", err)
	}
	if _, err := NewExitDecision(true, ExitReasonStopLoss, nil, "x"); err == nil ||
		err.Error() != "An exiting decision must carry a reason and exitPrice" {
		t.Fatalf("exit without price = %v", err)
	}
	if _, err := NewExitDecision(false, ExitReasonStopLoss, nil, "x"); err == nil ||
		err.Error() != "A non-exiting decision must not carry a reason or exitPrice" {
		t.Fatalf("hold with reason = %v", err)
	}
	if _, err := NewExitDecision(false, "", &price, "x"); err == nil ||
		err.Error() != "A non-exiting decision must not carry a reason or exitPrice" {
		t.Fatalf("hold with price = %v", err)
	}
}

func TestExitDecisionFactories(t *testing.T) {
	hold := ExitHold()
	if hold.Exit || hold.Reason != "" || hold.ExitPrice != nil || hold.Detail != "No exit condition met" {
		t.Fatalf("hold = %+v", hold)
	}
	price := mustParseDecisionDecimal(t, "101.25")
	exit := ExitWith(ExitReasonTargetHit, price, "target reached")
	if !exit.Exit || exit.Reason != ExitReasonTargetHit || exit.ExitPrice == nil ||
		*exit.ExitPrice != price || exit.Detail != "target reached" {
		t.Fatalf("exit = %+v", exit)
	}
	// Java enum names stay stable for wire and persistence consumers.
	for reason, name := range map[ExitReason]string{
		ExitReasonStopLoss: "STOP_LOSS", ExitReasonTargetHit: "TARGET_HIT",
		ExitReasonSignalExit: "SIGNAL_EXIT", ExitReasonTimeStop: "TIME_STOP",
		ExitReasonTrendBreak: "TREND_BREAK", ExitReasonBreakevenStop: "BREAKEVEN_STOP",
		ExitReasonTrailingStop: "TRAILING_STOP", ExitReasonManual: "MANUAL",
	} {
		if string(reason) != name {
			t.Fatalf("reason = %q, want %q", reason, name)
		}
	}
}
