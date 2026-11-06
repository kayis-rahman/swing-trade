package strategy

import (
	"fmt"
	"strings"
	"time"

	coredecimal "github.com/kayis-rahman/swing-trade/go/core/decimal"
	domain "github.com/kayis-rahman/swing-trade/go/core/domain"
)

// SignalResult mirrors Java record.
type SignalResult struct {
	Symbol    string
	Date      time.Time
	Type      domain.SignalType
	RSI       coredecimal.Decimal
	EMA20     coredecimal.Decimal
	EMA50     coredecimal.Decimal
	ATR       coredecimal.Decimal
	Reasoning string
}

// NewSignalResult mirrors the Java compact constructor, normalizing indicator
// readings to INDICATOR_SCALE decimal places (HALF_UP).
func NewSignalResult(symbol string, date time.Time, signalType domain.SignalType, rsi, ema20, ema50, atr coredecimal.Decimal, reasoning string) (SignalResult, error) {
	if strings.TrimSpace(symbol) == "" {
		return SignalResult{}, fmt.Errorf("symbol is required")
	}
	if date.IsZero() {
		return SignalResult{}, fmt.Errorf("date is required")
	}
	normalize := func(value coredecimal.Decimal) coredecimal.Decimal {
		normalized, err := coredecimal.ParseScale(value.String(), coredecimal.IndicatorScale)
		if err != nil {
			return value
		}
		return normalized
	}
	return SignalResult{
		Symbol:    symbol,
		Date:      date,
		Type:      signalType,
		RSI:       normalize(rsi),
		EMA20:     normalize(ema20),
		EMA50:     normalize(ema50),
		ATR:       normalize(atr),
		Reasoning: reasoning,
	}, nil
}
