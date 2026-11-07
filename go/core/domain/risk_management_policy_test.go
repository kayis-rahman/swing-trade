package domain

import (
	"testing"

	coredecimal "github.com/kayis-rahman/swing-trade/go/core/decimal"
)

func decimal(t *testing.T, value string) coredecimal.Decimal {
	t.Helper()
	d, err := coredecimal.Parse(value)
	if err != nil {
		t.Fatalf("parse %q: %v", value, err)
	}
	return d
}

func TestRiskManagementContextRejectsNegativeHoldingDays(t *testing.T) {
	_, err := NewRiskManagementContextFull(decimal(t, "100"), decimal(t, "99"), decimal(t, "110"), decimal(t, "101"), decimal(t, "100"), decimal(t, "102"), decimal(t, "100"), -1, false, coredecimal.Decimal{}, false)
	if err == nil {
		t.Fatal("expected invalid context for negative holding days")
	}
}

func TestRiskManagementDecisionExitRejectsBlankReasons(t *testing.T) {
	stop := decimal(t, "99")
	cases := []string{"", " ", "\t", "\n", "\u00A0"}
	for _, reason := range cases {
		_, err := RiskManagementDecisionExit(stop, reason)
		if err == nil {
			t.Fatalf("expected rejection for blank reason %q", reason)
		}
	}
	decision, err := RiskManagementDecisionExit(stop, "stop hit")
	if err != nil || !decision.Exit || !decision.HasStopPrice {
		t.Fatalf("valid exit rejected: %v %+v", err, decision)
	}
}

func TestRiskManagementDecisionPartialExitBoundary(t *testing.T) {
	exitPrice := decimal(t, "105")
	for _, tc := range []struct {
		ratio string
		ok    bool
	}{
		{"0", false},
		{"-1", false},
		{"0.5", true},
		{"1", true},
		{"1.0001", false},
	} {
		ratio, _ := coredecimal.Parse(tc.ratio)
		decision, err := RiskManagementDecisionPartialExit(exitPrice, ratio)
		if tc.ok {
			if err != nil || !decision.HasPartialRatio || decision.PartialExitRatio.Cmp(ratio) != 0 {
				t.Fatalf("ratio %s rejected: %v %+v", tc.ratio, err, decision)
			}
		} else {
			if err == nil {
				t.Fatalf("ratio %s accepted", tc.ratio)
			}
		}
	}
}

func TestRiskManagementDecisionHold(t *testing.T) {
	decision := RiskManagementDecisionHold()
	if decision.Exit || decision.HasStopPrice || decision.HasPartialRatio {
		t.Fatalf("hold should not carry stop or partial exit: %+v", decision)
	}
	if decision.Reason != "" {
		t.Fatalf("hold reason = %q, want empty", decision.Reason)
	}
}

func TestRiskManagementPolicyNone(t *testing.T) {
	policy := RiskManagementPolicyNone()
	decision := policy(RiskManagementContext{HoldingDays: 0})
	if decision.Exit {
		t.Fatal("none policy should hold")
	}
}
