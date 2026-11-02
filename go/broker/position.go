package broker

import (
	coredecimal "github.com/kayis-rahman/swing-trade/go/core/decimal"
)

// Position is a minimal domain position used by risk helpers.
type Position struct {
	Symbol        string
	Quantity      int
	EntryPrice    coredecimal.Decimal
	UnrealizedPnL coredecimal.Decimal
	RealizedPnL   coredecimal.Decimal
}

// PositionValue returns quantity * entryPrice.
// PositionValue implements the corresponding risk operation.
func (p Position) PositionValue() coredecimal.Decimal {
	return p.EntryPrice.MulInt(p.Quantity)
}
