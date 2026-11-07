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
	intradaySTTSell  = mustParseDecimal("0.00025")
	intradayStampBuy = mustParseDecimal("0.00003")
	intradayExchange = mustParseDecimal("0.0000307")
	intradaySebi     = mustParseDecimal("0.0000001")
	intradayGstRate  = mustParseDecimal("0.18")
)

func mustParseDecimal(s string) coredecimal.Decimal {
	d, err := coredecimal.Parse(s)
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
		return IntradayCostModel{}, fmt.Errorf("Costs must be non-negative")
	}
	return IntradayCostModel{brokeragePerOrder: brokeragePerOrder, slippageRate: slippageRate}, nil
}

// DefaultIntradayCostModel mirrors Java's no-arg constructor.
func DefaultIntradayCostModel() IntradayCostModel {
	m, _ := NewIntradayCostModel(mustParseDecimal("20"), mustParseDecimal("0.0005"))
	return m
}

func (c IntradayCostModel) BuyFill(reference coredecimal.Decimal) coredecimal.Decimal {
	one := mustParseDecimal("1")
	return decimalProduct(reference, one.Add(c.slippageRate))
}

func (c IntradayCostModel) SellFill(reference coredecimal.Decimal) coredecimal.Decimal {
	one := mustParseDecimal("1")
	return decimalProduct(reference, one.Sub(c.slippageRate))
}

func (c IntradayCostModel) RoundTripCost(buy, sell coredecimal.Decimal, quantity int) (coredecimal.Decimal, error) {
	if quantity <= 0 {
		return coredecimal.Decimal{}, fmt.Errorf("Quantity must be positive")
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
