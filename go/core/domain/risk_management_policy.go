package domain

import (
	"fmt"
	"unicode"

	coredecimal "github.com/kayis-rahman/swing-trade/go/core/decimal"
)

// RiskManagementPolicy mirrors Java's com.swingtrade.domain.RiskManagementPolicy.
type RiskManagementPolicy func(ctx RiskManagementContext) RiskManagementDecision

func RiskManagementPolicyNone() RiskManagementPolicy {
	return func(ctx RiskManagementContext) RiskManagementDecision { return RiskManagementDecisionHold() }
}

type RiskManagementContext struct {
	EntryPrice            coredecimal.Decimal
	InitialStop           coredecimal.Decimal
	Target                coredecimal.Decimal
	CurrentClose          coredecimal.Decimal
	CurrentLow            coredecimal.Decimal
	CurrentHigh           coredecimal.Decimal
	HighestCloseBeforeBar coredecimal.Decimal
	HoldingDays           int
	PartialExitTaken      bool
	CurrentATR            coredecimal.Decimal
	HasCurrentATR         bool
}

func NewRiskManagementContextFull(entryPrice, initialStop, target, currentClose, currentLow, currentHigh, highestCloseBeforeBar coredecimal.Decimal, holdingDays int, partialExitTaken bool, currentAtr coredecimal.Decimal, hasAtr bool) (RiskManagementContext, error) {
	ctx := RiskManagementContext{
		EntryPrice:            entryPrice,
		InitialStop:           initialStop,
		Target:                target,
		CurrentClose:          currentClose,
		CurrentLow:            currentLow,
		CurrentHigh:           currentHigh,
		HighestCloseBeforeBar: highestCloseBeforeBar,
		HoldingDays:           holdingDays,
		PartialExitTaken:      partialExitTaken,
		CurrentATR:            currentAtr,
		HasCurrentATR:         hasAtr,
	}
	if err := ctx.validate(); err != nil {
		return RiskManagementContext{}, err
	}
	return ctx, nil
}

func (c RiskManagementContext) validate() error {
	if c.HoldingDays < 0 {
		return fmt.Errorf("Risk-management context is invalid")
	}
	return nil
}

type RiskManagementDecision struct {
	Exit             bool
	StopPrice        coredecimal.Decimal
	HasStopPrice     bool
	Reason           string
	PartialExitRatio coredecimal.Decimal
	HasPartialRatio  bool
}

func RiskManagementDecisionHold() RiskManagementDecision {
	return RiskManagementDecision{Exit: false, Reason: "", StopPrice: coredecimal.Decimal{}, HasStopPrice: false}
}

func RiskManagementDecisionExit(stopPrice coredecimal.Decimal, reason string) (RiskManagementDecision, error) {
	blank := true
	for _, r := range reason {
		if !unicode.IsSpace(r) {
			blank = false
			break
		}
	}
	if blank {
		return RiskManagementDecision{}, fmt.Errorf("An exit decision requires a stop price and reason")
	}
	return RiskManagementDecision{Exit: true, StopPrice: stopPrice, HasStopPrice: true, Reason: reason}, nil
}

func RiskManagementDecisionPartialExit(exitPrice coredecimal.Decimal, ratio coredecimal.Decimal) (RiskManagementDecision, error) {
	one, _ := coredecimal.Parse("1")
	if ratio.Cmp(coredecimal.Decimal{}) <= 0 || ratio.Cmp(one) > 0 {
		return RiskManagementDecision{}, fmt.Errorf("Partial exit ratio must be between 0 and 1")
	}
	return RiskManagementDecision{Exit: false, StopPrice: exitPrice, HasStopPrice: true, Reason: "PARTIAL_TARGET", PartialExitRatio: ratio, HasPartialRatio: true}, nil
}
