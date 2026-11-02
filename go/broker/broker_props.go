package broker

import coredecimal "github.com/kayis-rahman/swing-trade/go/core/decimal"

// RiskConfig holds risk configuration (Go port).
type RiskConfig struct {
	InitialCapital            coredecimal.Decimal
	MaxConcurrentPositions    int
	MaxCapitalPerPosition     coredecimal.Decimal
	MaxCapitalPerTrade        coredecimal.Decimal
	MaxPositionSizePercentage coredecimal.Decimal
	MinPositionSizePercentage coredecimal.Decimal
	DailyLossCircuitBreaker   coredecimal.Decimal
	KillSwitchEnabled         bool
	KillSwitchActive          bool
}

// IsKillSwitchEnabled implements the corresponding risk operation.
func (b RiskConfig) IsKillSwitchEnabled() bool { return b.KillSwitchEnabled }
