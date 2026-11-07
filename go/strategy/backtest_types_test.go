package strategy

import (
	"testing"
	"time"

	coredecimal "github.com/kayis-rahman/swing-trade/go/core/decimal"
)

func TestIntradayBacktestConfigValidation(t *testing.T) {
	if _, err := NewIntradayBacktestConfig(coredecimal.Decimal{}, 1, 1, coredecimal.Decimal{}); err == nil {
		t.Fatal("expected invalid config error")
	}
	cfg := IntradayBacktestConfigDefaults()
	if cfg.Quantity <= 0 || cfg.MaxTradesPerDay <= 0 {
		t.Fatalf("bad defaults: %#v", cfg)
	}
}

func TestIntradayBacktestTradeAndResult(t *testing.T) {
	now := time.Now()
	trade := IntradayBacktestTrade{Strategy: "s", Quantity: 2, EntryTime: now, ExitTime: now}
	result := IntradayBacktestResult{Trades: []IntradayBacktestTrade{trade}, InitialCapital: mustParse("100")}
	if len(result.Trades) != 1 || result.Trades[0].Strategy != "s" {
		t.Fatal("trade not retained")
	}
}

func TestBacktestCostModel(t *testing.T) {
	model := BacktestCostModel(func(entry, exit coredecimal.Decimal, qty int, brokerage coredecimal.Decimal) coredecimal.Decimal {
		return mustParse("1.25")
	})
	if model(mustParse("10"), mustParse("11"), 1, mustParse("0")).Cmp(mustParse("1.25")) != 0 {
		t.Fatal("cost model result mismatch")
	}
}

func TestWalkForwardTypes(t *testing.T) {
	fold := WalkForwardFold{Result: "r"}
	eval := NewWalkForwardEvaluation([]Fold{{Result: "x"}}, 1, 2, 3)
	if !eval.IsComplete(1) || fold.Result == nil {
		t.Fatal("walk-forward type mismatch")
	}
}

func mustParse(s string) coredecimal.Decimal {
	d, err := coredecimal.Parse(s)
	if err != nil {
		panic(err)
	}
	return d
}
