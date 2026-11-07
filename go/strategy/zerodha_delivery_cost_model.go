package strategy

import (
	"fmt"
	"math/big"
	"strings"

	coredecimal "github.com/kayis-rahman/swing-trade/go/core/decimal"
)

const (
	zerodhaSTTRate      = "0.001"
	zerodhaExchangeRate = "0.0000297"
	zerodhaSebiRate     = "0.000001"
	zerodhaStampRate    = "0.00015"
	zerodhaGSTRate      = "0.18"
	zerodhaDPCharge     = "15.93"
)

// NewZerodhaDeliveryCostModel returns the default delivery-equity cost model.
func NewZerodhaDeliveryCostModel() BacktestCostModel {
	return ZerodhaDeliveryCostModel()
}

// ZerodhaDeliveryCostModel returns the default delivery-equity cost model.
func ZerodhaDeliveryCostModel() BacktestCostModel {
	rate := func(s string) *big.Rat {
		r, ok := new(big.Rat).SetString(s)
		if !ok {
			panic(fmt.Sprintf("invalid decimal %s", s))
		}
		return r
	}
	sttRate := rate(zerodhaSTTRate)
	exchangeRate := rate(zerodhaExchangeRate)
	sebiRate := rate(zerodhaSebiRate)
	stampRate := rate(zerodhaStampRate)
	gstRate := rate(zerodhaGSTRate)
	dpCharge := rate(zerodhaDPCharge)
	return func(entryPrice, exitPrice coredecimal.Decimal, quantity int, brokerage coredecimal.Decimal) coredecimal.Decimal {
		q := new(big.Rat).SetInt64(int64(quantity))
		buyTurnover := new(big.Rat).Mul(ratOf(entryPrice), q)
		sellTurnover := new(big.Rat).Mul(ratOf(exitPrice), q)
		totalTurnover := new(big.Rat).Add(buyTurnover, sellTurnover)
		exchangeAndRegulatory := new(big.Rat).Mul(totalTurnover, new(big.Rat).Add(exchangeRate, sebiRate))
		stt := new(big.Rat).Mul(totalTurnover, sttRate)
		stamp := new(big.Rat).Mul(buyTurnover, stampRate)
		brokerageRat := ratOf(brokerage)
		gst := new(big.Rat).Mul(new(big.Rat).Add(brokerageRat, exchangeAndRegulatory), gstRate)
		total := brokerageRat
		total = new(big.Rat).Add(total, stt)
		total = new(big.Rat).Add(total, exchangeAndRegulatory)
		total = new(big.Rat).Add(total, stamp)
		total = new(big.Rat).Add(total, gst)
		total = new(big.Rat).Add(total, dpCharge)
		roundedStr, err := coredecimal.RoundHalfUp(strings.TrimSpace(total.FloatString(18)), 2)
		if err != nil {
			return coredecimal.Decimal{}
		}
		rounded, err := coredecimal.Parse(roundedStr)
		if err != nil {
			return coredecimal.Decimal{}
		}
		return rounded
	}
}

// DefaultBacktestCostModel is an alias for the delivery model.
func DefaultBacktestCostModel() BacktestCostModel {
	return ZerodhaDeliveryCostModel()
}
