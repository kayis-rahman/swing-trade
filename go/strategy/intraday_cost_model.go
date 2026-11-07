package strategy

import (
	"fmt"

	coredecimal "github.com/kayis-rahman/swing-trade/go/core/decimal"
)

// IntradayCostModel mirrors Java's NSE intraday-equity costs; intentionally
// independent of the delivery cost model.
type IntradayCostModel struct {
	brokeragePerOrder coredecimal.Decimal
	slippageRate      coredecimal.Decimal
}

var (
	intradaySTTSell  = mustParseDecimalScale("0.00025", 10)
	intradayStampBuy = mustParseDecimalScale("0.00003", 10)
	intradayExchange = mustParseDecimalScale("0.0000307", 10)
	intradaySebi     = mustParseDecimalScale("0.0000001", 10)
	intradayGstRate  = mustParseDecimalScale("0.18", 10)
)

func mustParseDecimal(s string) coredecimal.Decimal {
	d, err := coredecimal.Parse(s)
	if err != nil {
		panic(err)
	}
	return d
}

func mustParseDecimalScale(s string, scale int) coredecimal.Decimal {
	d, err := coredecimal.ParseScale(s, scale)
	if err != nil {
		panic(err)
	}
	return d
}

func qScalar(quantity int) coredecimal.Decimal {
	q, _ := coredecimal.Parse(fmt.Sprintf("%d", quantity))
	return q
}

// NewIntradayCostModel constructs a cost model with the supplied brokerage and
// slippage, mirroring Java's constructor.
func NewIntradayCostModel(brokeragePerOrder, slippageRate coredecimal.Decimal) (IntradayCostModel, error) {
	if brokeragePerOrder.Cmp(coredecimal.Decimal{}) < 0 || slippageRate.Cmp(coredecimal.Decimal{}) < 0 {
		return IntradayCostModel{}, fmt.Errorf("Costs must be non-negative") //nolint:staticcheck // exact Java IllegalArgumentException message for parity
	}
	return IntradayCostModel{brokeragePerOrder: brokeragePerOrder, slippageRate: slippageRate}, nil
}

// DefaultIntradayCostModel mirrors Java's no-arg constructor.
func DefaultIntradayCostModel() IntradayCostModel {
	m, _ := NewIntradayCostModel(mustParseDecimal("20"), mustParseDecimal("0.0005"))
	return m
}

// BuyFill returns the buy price after slippage.
func (c IntradayCostModel) BuyFill(reference coredecimal.Decimal) coredecimal.Decimal {
	one := mustParseDecimal("1")
	return decimalProduct(reference, one.Add(c.slippageRate))
}

// SellFill returns the sell price after slippage.
func (c IntradayCostModel) SellFill(reference coredecimal.Decimal) coredecimal.Decimal {
	one := mustParseDecimal("1")
	return decimalProduct(reference, one.Sub(c.slippageRate))
}

// RoundTripCost computes round-trip transaction costs.
func (c IntradayCostModel) RoundTripCost(buy, sell coredecimal.Decimal, quantity int) (coredecimal.Decimal, error) {
	if quantity <= 0 {
		return coredecimal.Decimal{}, fmt.Errorf("Quantity must be positive") //nolint:staticcheck // exact Java IllegalArgumentException message for parity
	}
	q := qScalar(quantity)
	buyTurnover := decimalProduct(buy, q)
	sellTurnover := decimalProduct(sell, q)
	brokerage := decimalProduct(c.brokeragePerOrder, mustParseDecimal("2"))
	exchange := decimalProduct(buyTurnover.Add(sellTurnover), intradayExchange)
	sebi := decimalProduct(buyTurnover.Add(sellTurnover), intradaySebi)
	taxable := brokerage.Add(exchange).Add(sebi)
	gst := decimalProduct(taxable, intradayGstRate)
	total := brokerage.Add(exchange).Add(sebi).Add(gst).
		Add(decimalProduct(sellTurnover, intradaySTTSell)).
		Add(decimalProduct(buyTurnover, intradayStampBuy))
	roundedStr, err := coredecimal.RoundHalfUp(total.String(), 2)
	if err != nil {
		return coredecimal.Decimal{}, err
	}
	rounded, err := coredecimal.Parse(roundedStr)
	if err != nil {
		return coredecimal.Decimal{}, err
	}
	return rounded, nil
}
