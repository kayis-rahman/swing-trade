package strategy

import coredecimal "github.com/kayis-rahman/swing-trade/go/core/decimal"

// BacktestConfig contains the capital and execution assumptions for a signal backtest.
type BacktestConfig struct {
	InitialCapital    coredecimal.Decimal
	RiskPerTradePct   coredecimal.Decimal
	SlippagePct       coredecimal.Decimal
	BrokeragePerTrade coredecimal.Decimal
}

// BacktestConfigDefaults returns the standard single-strategy backtest settings.
func BacktestConfigDefaults() BacktestConfig {
	return BacktestConfig{
		InitialCapital: mustParseDecimal("100000"),
		RiskPerTradePct: mustParseDecimal("0.01"),
		SlippagePct: mustParseDecimal("0.001"),
		BrokeragePerTrade: mustParseDecimal("20"),
	}
}
