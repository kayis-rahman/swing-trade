package strategy

import (
	"testing"

	coredecimal "github.com/kayis-rahman/swing-trade/go/core/decimal"
)

func TestNewIntradayCostModelRejectsNegative(t *testing.T) {
	if _, err := NewIntradayCostModel(mustRuleDecimal("-1"), mustRuleDecimal("0.1")); err == nil {
		t.Fatal("expected error")
	}
	if _, err := NewIntradayCostModel(mustRuleDecimal("1"), mustRuleDecimal("-0.1")); err == nil {
		t.Fatal("expected error")
	}
}

func TestIntradayCostModelBuySellFill(t *testing.T) {
	model := DefaultIntradayCostModel()
	buy := model.BuyFill(mustRuleDecimal("100"))
	sell := model.SellFill(mustRuleDecimal("100"))
	if buy.Cmp(mustRuleDecimal("100.05")) != 0 {
		t.Fatalf("buy = %s", buy)
	}
	if sell.Cmp(mustRuleDecimal("99.95")) != 0 {
		t.Fatalf("sell = %s", sell)
	}
}

func TestIntradayCostModelRoundTripCostPositive(t *testing.T) {
	model := DefaultIntradayCostModel()
	fees, err := model.RoundTripCost(mustRuleDecimal("100.05"), mustRuleDecimal("99.95"), 10)
	if err != nil {
		t.Fatal(err)
	}
	if fees.Cmp(coredecimal.Decimal{}) <= 0 {
		t.Fatalf("fees = %s", fees)
	}
	if _, err := model.RoundTripCost(mustRuleDecimal("100"), mustRuleDecimal("100"), 0); err == nil {
		t.Fatal("expected quantity error")
	}
}
