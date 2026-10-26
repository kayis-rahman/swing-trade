package data

import coredecimal "github.com/kayis-rahman/swing-trade/go/core/decimal"

// CandleData is the OHLCV input validated before storage or ingestion.
type CandleData struct {
	Symbol string
	Open   *coredecimal.Decimal
	High   *coredecimal.Decimal
	Low    *coredecimal.Decimal
	Close  *coredecimal.Decimal
	Volume int64
}

// IsValidCandle matches Java CandleValidator's OHLCV consistency rules.
func IsValidCandle(candle *CandleData) bool {
	if candle == nil || candle.Open == nil || candle.High == nil || candle.Low == nil || candle.Close == nil || candle.Volume <= 0 {
		return false
	}
	zero := coredecimal.Decimal{}
	if candle.Open.Cmp(zero) <= 0 || candle.High.Cmp(zero) <= 0 || candle.Low.Cmp(zero) <= 0 || candle.Close.Cmp(zero) <= 0 {
		return false
	}
	if candle.High.Cmp(*candle.Low) < 0 || candle.High.Cmp(*candle.Open) < 0 || candle.High.Cmp(*candle.Close) < 0 {
		return false
	}
	return candle.Low.Cmp(*candle.Open) <= 0 && candle.Low.Cmp(*candle.Close) <= 0
}
