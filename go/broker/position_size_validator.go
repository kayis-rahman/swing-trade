package broker

import (
	"math/big"

	coredecimal "github.com/kayis-rahman/swing-trade/go/core/decimal"
)

// PositionSizeValidator validates trade size.
type PositionSizeValidator struct {
	Props RiskConfig
}

// NewPositionSizeValidator implements the corresponding risk operation.
func NewPositionSizeValidator(props RiskConfig) *PositionSizeValidator {
	return &PositionSizeValidator{Props: props}
}

func decimalToRat(d coredecimal.Decimal) *big.Rat {
	return new(big.Rat).SetFrac(d.Units(), big.NewInt(10000))
}

func ratToDecimal(r *big.Rat) coredecimal.Decimal {
	s := r.FloatString(8)
	rounded, _ := coredecimal.RoundHalfUp(s, coredecimal.MoneyScale)
	v, _ := coredecimal.Parse(rounded)
	return v
}

func percentDecimal(base, percent coredecimal.Decimal) coredecimal.Decimal {
	r := new(big.Rat).Mul(decimalToRat(base), decimalToRat(percent))
	r = r.Quo(r, new(big.Rat).SetInt64(100))
	return ratToDecimal(r)
}

// ValidatePositionSize implements the corresponding risk operation.
func (v *PositionSizeValidator) ValidatePositionSize(tradeValue coredecimal.Decimal) *RiskCheckResult {
	result := NewRiskCheckResult(true)
	result.CheckType = "POSITION_SIZE_VALIDATION"
	if tradeValue.Cmp(coredecimal.Decimal{}) <= 0 {
		result.AddError("Invalid trade value: %s. Must be positive.", tradeValue)
		return result
	}
	if tradeValue.Cmp(v.Props.MaxCapitalPerTrade) > 0 {
		result.AddError("Trade value exceeds maximum per trade: %s > %s", tradeValue, v.Props.MaxCapitalPerTrade)
		return result
	}
	maxAllowed := percentDecimal(v.Props.InitialCapital, v.Props.MaxPositionSizePercentage)
	if tradeValue.Cmp(maxAllowed) > 0 {
		result.AddError("Trade size exceeds maximum position size: %s%% > %s%%", v.CalculatePositionPercent(tradeValue), v.Props.MaxPositionSizePercentage)
		return result
	}
	minAllowed := percentDecimal(v.Props.InitialCapital, v.Props.MinPositionSizePercentage)
	if tradeValue.Cmp(minAllowed) < 0 {
		result.AddWarning("Trade size is below minimum recommended: %s%% < %s%%", v.CalculatePositionPercent(tradeValue), v.Props.MinPositionSizePercentage)
	}
	result.AddInfo("Position size validated: %s (%s%% of capital)", tradeValue, v.CalculatePositionPercent(tradeValue))
	return result
}

// CalculatePositionPercent implements the corresponding risk operation.
func (v *PositionSizeValidator) CalculatePositionPercent(tradeValue coredecimal.Decimal) coredecimal.Decimal {
	return coredecimal.PercentOf(tradeValue, v.Props.InitialCapital)
}

// GetMaxPositionSize implements the corresponding risk operation.
func (v *PositionSizeValidator) GetMaxPositionSize() coredecimal.Decimal {
	return percentDecimal(v.Props.InitialCapital, v.Props.MaxPositionSizePercentage)
}

// CalculateRecommendedPositionSize implements the corresponding risk operation.
func (v *PositionSizeValidator) CalculateRecommendedPositionSize(riskPerTrade coredecimal.Decimal) coredecimal.Decimal {
	return percentDecimal(v.Props.InitialCapital, riskPerTrade)
}
