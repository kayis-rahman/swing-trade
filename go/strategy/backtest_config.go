package strategy

import (
	coredecimal "github.com/kayis-rahman/swing-trade/go/core/decimal"
	domain "github.com/kayis-rahman/swing-trade/go/core/domain"
)

// BacktestConfig mirrors Java's com.swingtrade.strategy.BacktestConfig.
type BacktestConfig struct {
	SlippagePct            coredecimal.Decimal
	BrokeragePerTrade       coredecimal.Decimal
	RiskPerTradePct        coredecimal.Decimal
	InitialCapital         coredecimal.Decimal
	MaxConcurrentPositions int
	ATRMultiplierStop      float64
	RewardRiskRatio        float64
	MaxHoldingDays         int
	SignalExitEnabled      bool
	TrendBreakStreakDays   int
	RiskManagementPolicy   domain.RiskManagementPolicy
	PortfolioExposurePolicy domain.PortfolioExposurePolicy
}

// NewBacktestConfig mirrors the Java BigDecimal-based canonical constructor.
func NewBacktestConfig(slippagePct, brokeragePerTrade, riskPerTradePct, initialCapital coredecimal.Decimal,
	maxConcurrentPositions int, atrMultiplierStop, rewardRiskRatio float64, maxHoldingDays int,
	signalExitEnabled bool, trendBreakStreakDays int, riskManagementPolicy domain.RiskManagementPolicy,
	portfolioExposurePolicy domain.PortfolioExposurePolicy) BacktestConfig {
	cfg := BacktestConfig{
		SlippagePct:            slippagePct,
		BrokeragePerTrade:      brokeragePerTrade,
		RiskPerTradePct:        riskPerTradePct,
		InitialCapital:         initialCapital,
		MaxConcurrentPositions: maxConcurrentPositions,
		ATRMultiplierStop:      atrMultiplierStop,
		RewardRiskRatio:        rewardRiskRatio,
		MaxHoldingDays:         maxHoldingDays,
		SignalExitEnabled:      signalExitEnabled,
		TrendBreakStreakDays:   trendBreakStreakDays,
		RiskManagementPolicy:   riskManagementPolicy,
		PortfolioExposurePolicy: portfolioExposurePolicy,
	}
	if cfg.RiskManagementPolicy == nil {
		cfg.RiskManagementPolicy = domain.RiskManagementPolicyNone()
	}
	if cfg.PortfolioExposurePolicy == nil {
		cfg.PortfolioExposurePolicy = domain.PortfolioExposurePolicyNone()
	}
	return cfg
}

// BacktestConfigDefaults mirrors Java BacktestConfig.defaults().
func BacktestConfigDefaults() BacktestConfig {
	return BacktestConfig{
		SlippagePct:            mustParseDecimal("0.001"),
		BrokeragePerTrade:      mustParseDecimal("20.0"),
		RiskPerTradePct:        mustParseDecimal("0.01"),
		InitialCapital:         mustParseDecimal("500000.0"),
		MaxConcurrentPositions: 5,
		ATRMultiplierStop:      2.0,
		RewardRiskRatio:        2.5,
		MaxHoldingDays:         20,
		SignalExitEnabled:      false,
		TrendBreakStreakDays:   21,
		RiskManagementPolicy:   domain.RiskManagementPolicyNone(),
		PortfolioExposurePolicy: domain.PortfolioExposurePolicyNone(),
	}
}
