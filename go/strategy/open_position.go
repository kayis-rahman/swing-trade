package strategy

import (
	"fmt"
	"time"

	coredecimal "github.com/kayis-rahman/swing-trade/go/core/decimal"
)

// OpenPosition mirrors Java's public SPI record for an open paper/live
// position as seen by exit evaluation. Go keeps the price fields as values:
// the Java null guard on entry/stop/target becomes a constructor contract,
// since every caller supplies them and BigDecimal.ZERO was always legal.
type OpenPosition struct {
	EntryIndex    int
	EntryDate     time.Time
	EntryPrice    coredecimal.Decimal
	StopLoss      coredecimal.Decimal
	Target        coredecimal.Decimal
	Quantity      int
	HighWaterMark coredecimal.Decimal
}

// NewOpenPosition mirrors OpenPosition.open: the high-water mark is seeded at
// the entry price unless supplied.
func NewOpenPosition(
	entryIndex int,
	entryDate time.Time,
	entryPrice, stopLoss, target coredecimal.Decimal,
	quantity int,
	highWaterMark *coredecimal.Decimal,
) (OpenPosition, error) {
	if entryDate.IsZero() {
		return OpenPosition{}, fmt.Errorf("entryDate/entryPrice/stopLoss/target cannot be null")
	}
	seeded := entryPrice
	if highWaterMark != nil {
		seeded = *highWaterMark
	}
	return OpenPosition{
		EntryIndex: entryIndex, EntryDate: entryDate,
		EntryPrice: entryPrice, StopLoss: stopLoss, Target: target,
		Quantity: quantity, HighWaterMark: seeded,
	}, nil
}

// OpenPositionAt mirrors OpenPosition.open: high-water mark at the entry
// price.
func OpenPositionAt(
	entryIndex int,
	entryDate time.Time,
	entryPrice, stopLoss, target coredecimal.Decimal,
	quantity int,
) (OpenPosition, error) {
	return NewOpenPosition(entryIndex, entryDate, entryPrice, stopLoss, target, quantity, nil)
}

// AdvanceHighWaterMark mirrors Java: returns a copy with the mark advanced to
// the candidate when it is higher.
func (position OpenPosition) AdvanceHighWaterMark(candidate coredecimal.Decimal) OpenPosition {
	next := candidate
	if position.HighWaterMark.Cmp(candidate) > 0 {
		next = position.HighWaterMark
	}
	position.HighWaterMark = next
	return position
}
