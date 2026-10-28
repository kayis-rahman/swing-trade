package strategy

import (
	"testing"
	"time"

	coredecimal "github.com/kayis-rahman/swing-trade/go/core/decimal"
	"github.com/kayis-rahman/swing-trade/go/core/domain"
)

func TestUniformExitStopLoss(t *testing.T) {
	date := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	candles := make([]domain.OhlcvCandle, 0, 31)
	for i := 0; i < 30; i++ {
		candles = append(candles, domain.OhlcvCandleOf("TESTCO", date, mustDecimal("100"), mustDecimal("101"), mustDecimal("99"), mustDecimal("100"), 1000000))
		date = date.AddDate(0, 0, 1)
	}
	candles = append(candles, domain.OhlcvCandleOf("TESTCO", date, mustDecimal("100"), mustDecimal("130"), mustDecimal("80"), mustDecimal("100"), 1000000))
	ctx := MarketContextOf("TESTCO", candles)
	last := ctx.BarCount() - 1
	pos, _ := OpenPositionAt(0, date, mustDecimal("100"), mustDecimal("90"), mustDecimal("120"), 10)
	params := StrategyParamsViewOf(map[string]any{"trailAtrMult": coredecimal.Decimal{}, "atrPeriod": 14, "maxHoldDays": 100})
	dec, _ := UniformExitEvaluate(ctx.View(last), pos, params, true)
	if !dec.Exit || dec.Reason != ExitReasonStopLoss {
		t.Fatalf("expected stop loss, got %+v", dec)
	}
}

