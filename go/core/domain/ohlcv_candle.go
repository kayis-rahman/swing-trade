package domain

import (
	"math/big"
	"time"

	coredecimal "github.com/kayis-rahman/swing-trade/go/core/decimal"
)

// OhlcvCandle mirrors Java's com.swingtrade.domain.OhlcvCandle.
type OhlcvCandle struct {
	Symbol   string
	Date     time.Time
	Open     coredecimal.Decimal
	High     coredecimal.Decimal
	Low      coredecimal.Decimal
	Close    coredecimal.Decimal
	Volume   *int64
	AdjClose coredecimal.Decimal
}

func NewOhlcvCandle(symbol string, date time.Time, open, high, low, close coredecimal.Decimal, volume int64, adjClose coredecimal.Decimal) OhlcvCandle {
	v := volume
	return OhlcvCandle{Symbol: symbol, Date: date, Open: open, High: high, Low: low, Close: close, Volume: &v, AdjClose: adjClose}
}

func OhlcvCandleOf(symbol string, date time.Time, open, high, low, close coredecimal.Decimal, volume int64) OhlcvCandle {
	return NewOhlcvCandle(symbol, date, open, high, low, close, volume, close)
}

func (c OhlcvCandle) AdjustedForAnalysis() OhlcvCandle {
	if c.Close.Cmp(coredecimal.Decimal{}) <= 0 || c.AdjClose.Cmp(coredecimal.Decimal{}) <= 0 {
		return c
	}
	factorRat := new(big.Rat).Quo(decimalToRat(c.AdjClose), decimalToRat(c.Close))
	out := c
	out.Open = ratToDecimal(new(big.Rat).Mul(decimalToRat(c.Open), factorRat))
	out.High = ratToDecimal(new(big.Rat).Mul(decimalToRat(c.High), factorRat))
	out.Low = ratToDecimal(new(big.Rat).Mul(decimalToRat(c.Low), factorRat))
	out.Close = c.AdjClose
	out.AdjClose = c.AdjClose
	return out
}

func (c OhlcvCandle) GetRange() coredecimal.Decimal { return c.High.Sub(c.Low) }

func (c OhlcvCandle) IsBullish() bool { return c.Close.Cmp(c.Open) > 0 }

func (c OhlcvCandle) IsBearish() bool { return c.Close.Cmp(c.Open) < 0 }

func decimalToRat(d coredecimal.Decimal) *big.Rat {
	return new(big.Rat).SetFrac(d.Units(), new(big.Int).Exp(big.NewInt(10), big.NewInt(4), nil))
}

func ratToDecimal(r *big.Rat) coredecimal.Decimal {
	dec, _ := coredecimal.Parse(r.FloatString(4))
	return dec
}
