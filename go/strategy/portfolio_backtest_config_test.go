package strategy

import (
	"testing"

	coredecimal "github.com/kayis-rahman/swing-trade/go/core/decimal"
)

func TestNewPortfolioBacktestConfigValidAndDefaults(t *testing.T) {
	cfg, err := NewPortfolioBacktestConfig(
		mustParseDecimal("100000"),
		4,
		mustParseDecimal("0.25"),
		mustParseDecimal("0.02"),
		mustParseDecimal("-0.001"),
		nil,
		0,
		mustParseDecimal("-20"),
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.InitialCapital.Cmp(mustParseDecimal("100000")) != 0 {
		t.Fatalf("initial capital mismatch: %s", cfg.InitialCapital.String())
	}
	if cfg.MaxConcurrentPositions != 4 {
		t.Fatalf("max concurrent positions mismatch: %d", cfg.MaxConcurrentPositions)
	}
	if cfg.MaxCapitalPerPosition.Cmp(mustParseDecimal("0.25")) != 0 {
		t.Fatalf("max capital per position mismatch: %s", cfg.MaxCapitalPerPosition.String())
	}
	if cfg.RiskPerTradePct.Cmp(mustParseDecimal("0.02")) != 0 {
		t.Fatalf("risk per trade pct mismatch: %s", cfg.RiskPerTradePct.String())
	}
	if cfg.SlippagePct.Cmp(mustParseDecimal("-0.001")) != 0 {
		t.Fatalf("slippage pct not preserved: %s", cfg.SlippagePct.String())
	}
	if cfg.CostModel == nil {
		t.Fatal("cost model should default when nil")
	}
	if cfg.ATRPeriodForRanking != 14 {
		t.Fatalf("atr period default mismatch: %d", cfg.ATRPeriodForRanking)
	}
	if cfg.BrokeragePerTrade.Cmp(mustParseDecimal("-20")) != 0 {
		t.Fatalf("brokerage not preserved: %s", cfg.BrokeragePerTrade.String())
	}
}

func TestPortfolioBacktestConfigDefaults(t *testing.T) {
	cfg := PortfolioBacktestConfigDefaults(mustParseDecimal("50000"))
	if cfg.InitialCapital.Cmp(mustParseDecimal("50000")) != 0 {
		t.Fatalf("initial capital mismatch: %s", cfg.InitialCapital.String())
	}
	if cfg.MaxConcurrentPositions != 12 {
		t.Fatalf("max concurrent positions mismatch: %d", cfg.MaxConcurrentPositions)
	}
	if cfg.ATRPeriodForRanking != 14 {
		t.Fatalf("atr period mismatch: %d", cfg.ATRPeriodForRanking)
	}
	if cfg.CostModel == nil {
		t.Fatal("cost model nil")
	}
}

func TestNewPortfolioBacktestConfigInvalidInputs(t *testing.T) {
	cases := []struct {
		name string
		run  func() error
	}{
		{"non-positive capital", func() error {
			_, err := NewPortfolioBacktestConfig(coredecimal.Decimal{}, 1, mustParseDecimal("0.1"), mustParseDecimal("0.01"), coredecimal.Decimal{}, nil, 14, coredecimal.Decimal{})
			return err
		}},
		{"non-positive positions", func() error {
			_, err := NewPortfolioBacktestConfig(mustParseDecimal("1"), 0, mustParseDecimal("0.1"), mustParseDecimal("0.01"), coredecimal.Decimal{}, nil, 14, coredecimal.Decimal{})
			return err
		}},
		{"invalid max capital pct", func() error {
			_, err := NewPortfolioBacktestConfig(mustParseDecimal("1"), 1, coredecimal.Decimal{}, mustParseDecimal("0.01"), coredecimal.Decimal{}, nil, 14, coredecimal.Decimal{})
			return err
		}},
		{"invalid risk pct", func() error {
			_, err := NewPortfolioBacktestConfig(mustParseDecimal("1"), 1, mustParseDecimal("0.1"), coredecimal.Decimal{}, coredecimal.Decimal{}, nil, 14, coredecimal.Decimal{})
			return err
		}},
	}
	for _, c := range cases {
		if err := c.run(); err == nil {
			t.Fatalf("%s: expected error", c.name)
		}
	}
}
