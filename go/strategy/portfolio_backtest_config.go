package strategy

import (
	"fmt"

	coredecimal "github.com/kayis-rahman/swing-trade/go/core/decimal"
)

// PortfolioBacktestConfig mirrors Java record.
type PortfolioBacktestConfig struct {
	InitialCapital         coredecimal.Decimal
	MaxConcurrentPositions int
	MaxCapitalPerPosition  coredecimal.Decimal
	RiskPerTradePct        coredecimal.Decimal
	SlippagePct            coredecimal.Decimal
	CostModel              BacktestCostModel
	ATRPeriodForRanking    int
	BrokeragePerTrade      coredecimal.Decimal
}

// NewPortfolioBacktestConfig mirrors Java constructor validation.
func NewPortfolioBacktestConfig(initialCapital coredecimal.Decimal, maxConcurrentPositions int,
	maxCapitalPerPositionPct, riskPerTradePct, slippagePct coredecimal.Decimal, costModel BacktestCostModel,
	atrPeriodForRanking int, brokeragePerTrade coredecimal.Decimal) (PortfolioBacktestConfig, error) {
	if initialCapital.Cmp(coredecimal.Decimal{}) <= 0 {
		return PortfolioBacktestConfig{}, fmt.Errorf("initialCapital must be positive")
	}
	if maxConcurrentPositions <= 0 {
		return PortfolioBacktestConfig{}, fmt.Errorf("maxConcurrentPositions must be positive")
	}
	if maxCapitalPerPositionPct.Cmp(coredecimal.Decimal{}) <= 0 || maxCapitalPerPositionPct.Cmp(mustParseDecimal("1")) > 0 {
		return PortfolioBacktestConfig{}, fmt.Errorf("maxCapitalPerPositionPct must be in (0,1]")
	}
	if riskPerTradePct.Cmp(coredecimal.Decimal{}) <= 0 || riskPerTradePct.Cmp(mustParseDecimal("1")) > 0 {
		return PortfolioBacktestConfig{}, fmt.Errorf("riskPerTradePct must be in (0,1]")
	}
	if slippagePct.Cmp(coredecimal.Decimal{}) < 0 {
		slippagePct = coredecimal.Decimal{}
	}
	if costModel == nil {
		costModel = ZerodhaDeliveryCostModel()
	}
	if atrPeriodForRanking <= 0 {
		atrPeriodForRanking = 14
	}
	if brokeragePerTrade.Cmp(coredecimal.Decimal{}) < 0 {
		brokeragePerTrade = coredecimal.Decimal{}
	}
	return PortfolioBacktestConfig{
		InitialCapital:         initialCapital,
		MaxConcurrentPositions: maxConcurrentPositions,
		MaxCapitalPerPosition:  maxCapitalPerPositionPct,
		RiskPerTradePct:        riskPerTradePct,
		SlippagePct:            slippagePct,
		CostModel:              costModel,
		ATRPeriodForRanking:    atrPeriodForRanking,
		BrokeragePerTrade:      brokeragePerTrade,
	}, nil
}

// PortfolioBacktestConfigDefaults mirrors Java defaults().
func PortfolioBacktestConfigDefaults(initialCapital coredecimal.Decimal) PortfolioBacktestConfig {
	c, _ := NewPortfolioBacktestConfig(initialCapital, 12,
		mustParseDecimal("0.10"), mustParseDecimal("0.01"), mustParseDecimal("0.001"),
		ZerodhaDeliveryCostModel(), 14, mustParseDecimal("20"))
	return c
}
