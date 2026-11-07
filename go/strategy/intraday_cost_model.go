package strategy

import (
	"fmt"
	"math/big"

	coredecimal "github.com/kayis-rahman/swing-trade/go/core/decimal"
)

// IntradayCostModel mirrors Java's NSE intraday-equity costs; it is intentionally
// independent of the delivery cost model.
type IntradayCostModel struct {
	brokeragePerOrder coredecimal.Decimal
	slippageRate      coredecimal.Decimal
}

var (
	intradaySTTSell  = mustRuleDecimal("0.00025")
	intradayStampBuy = mustRuleDecimal("0.00003")
	intradayExchange = mustRuleDecimal("0.0000307")
	intradaySEBI     = mustRuleDecimal("0.0000001")
	intradayGST      = mustRuleDecimal("0.18")
)

// NewIntradayCostModel mirrors Java's two-arg constructor and validation.
func NewIntradayCostModel(brokeragePerOrder, slippageRate coredecimal.Decimal) (IntradayCostModel, error) {
	if brokeragePerOrder.Cmp(coredecimal.Decimal{}) < 0 || slippageRate.Cmp(coredecimal.Decimal{}) < 0 {
		return IntradayCostModel{}, fmt.Errorf("costs must be non-negative")
	}
	return IntradayCostModel{brokeragePerOrder: brokeragePerOrder, slippageRate: slippageRate}, nil
}

// DefaultIntradayCostModel mirrors Java's no-arg constructor.
func DefaultIntradayCostModel() IntradayCostModel {
	model, _ := NewIntradayCostModel(mustRuleDecimal("20"), mustRuleDecimal("0.0005"))
	return model
}

// BrokeragePerOrder returns the per-order brokerage.
func (m IntradayCostModel) BrokeragePerOrder() coredecimal.Decimal { return m.brokeragePerOrder }

// SlippageRate returns the slippage rate.
func (m IntradayCostModel) SlippageRate() coredecimal.Decimal { return m.slippageRate }

// BuyFill mirrors BigDecimal reference * (1 + slippageRate).
func (m IntradayCostModel) BuyFill(reference coredecimal.Decimal) coredecimal.Decimal {
	scale := reference.Scale() + 4
	factor := new(big.Rat).Add(big.NewRat(1, 1), ratOf(m.slippageRate))
	result := new(big.Rat).Mul(ratOf(reference), factor)
	parsed, err := coredecimal.ParseScale(result.FloatString(scale), scale)
	if err != nil {
		return coredecimal.Decimal{}
	}
	return parsed
}

// SellFill mirrors BigDecimal reference * (1 - slippageRate).
func (m IntradayCostModel) SellFill(reference coredecimal.Decimal) coredecimal.Decimal {
	scale := reference.Scale() + 4
	factor := new(big.Rat).Sub(big.NewRat(1, 1), ratOf(m.slippageRate))
	result := new(big.Rat).Mul(ratOf(reference), factor)
	parsed, err := coredecimal.ParseScale(result.FloatString(scale), scale)
	if err != nil {
		return coredecimal.Decimal{}
	}
	return parsed
}

// RoundTripCost mirrors Java's roundTripCost.
func (m IntradayCostModel) RoundTripCost(buy, sell coredecimal.Decimal, quantity int) (coredecimal.Decimal, error) {
	if quantity <= 0 {
		return coredecimal.Decimal{}, fmt.Errorf("quantity must be positive")
	}
	buyTurnover := new(big.Rat).Mul(ratOf(buy), big.NewRat(int64(quantity), 1))
	sellTurnover := new(big.Rat).Mul(ratOf(sell), big.NewRat(int64(quantity), 1))
	brokerage := new(big.Rat).Mul(ratOf(m.brokeragePerOrder), big.NewRat(2, 1))
	exchange := new(big.Rat).Mul(new(big.Rat).Add(buyTurnover, sellTurnover), ratOf(intradayExchange))
	sebi := new(big.Rat).Mul(new(big.Rat).Add(buyTurnover, sellTurnover), ratOf(intradaySEBI))
	taxable := new(big.Rat).Add(new(big.Rat).Add(brokerage, exchange), sebi)
	gst := new(big.Rat).Mul(taxable, ratOf(intradayGST))
	total := new(big.Rat).Add(new(big.Rat).Add(new(big.Rat).Add(brokerage, exchange), sebi), gst)
	total = new(big.Rat).Add(total, new(big.Rat).Mul(sellTurnover, ratOf(intradaySTTSell)))
	total = new(big.Rat).Add(total, new(big.Rat).Mul(buyTurnover, ratOf(intradayStampBuy)))
	parsed, err := coredecimal.ParseScale(total.FloatString(2), 2)
	if err != nil {
		return coredecimal.Decimal{}, err
	}
	return parsed, nil
}
