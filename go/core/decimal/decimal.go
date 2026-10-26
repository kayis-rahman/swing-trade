package decimal

import (
	"fmt"
	"math/big"
	"strings"
)

// Financial scales and Java Integer.MAX_VALUE compatibility boundary.
const (
	MoneyScale     = 4
	PercentScale   = 4
	IndicatorScale = 6
	Int32Max       = int32(1<<31 - 1)
)

// Decimal is an immutable fixed-scale decimal. The zero value is numeric zero.
type Decimal struct {
	units *big.Int
}

// Parse converts a base-10 value to MoneyScale places using HALF_UP rounding.
func Parse(value string) (Decimal, error) {
	value = strings.TrimSpace(value)
	units, err := scaledUnits(value, MoneyScale)
	if err != nil {
		return Decimal{}, err
	}
	return Decimal{units: units}, nil
}

// String returns the fixed-scale canonical representation.
func (d Decimal) String() string {
	value := d.value()
	negative := value.Sign() < 0
	value.Abs(value)
	whole, fraction := new(big.Int), new(big.Int)
	whole.QuoRem(value, scaleUnit(MoneyScale), fraction)
	prefix := ""
	if negative {
		prefix = "-"
	}
	return fmt.Sprintf("%s%s.%0*s", prefix, whole.String(), MoneyScale, fraction.String())
}

// Cmp compares two decimals numerically and returns -1, 0, or 1.
func (d Decimal) Cmp(other Decimal) int {
	return d.value().Cmp(other.value())
}

// Add returns the sum without mutating either operand.
func (d Decimal) Add(other Decimal) Decimal {
	return Decimal{units: new(big.Int).Add(d.value(), other.value())}
}

// Sub returns the difference without mutating either operand.
func (d Decimal) Sub(other Decimal) Decimal {
	return Decimal{units: new(big.Int).Sub(d.value(), other.value())}
}

// PercentOf computes numerator / denominator * 100, rounded HALF_UP to PercentScale.
// A zero denominator returns scaled zero, matching Java FinancialScale.percentOf.
func PercentOf(numerator, denominator Decimal) Decimal {
	den := denominator.value()
	if den.Sign() == 0 {
		return Decimal{}
	}
	factor := new(big.Int).Mul(big.NewInt(100), new(big.Int).Exp(big.NewInt(10), big.NewInt(PercentScale), nil))
	num := new(big.Int).Mul(numerator.value(), factor)
	return Decimal{units: roundHalfUp(num, den)}
}

// WholeShares truncates toward zero and saturates at Java's Integer.MAX_VALUE.
func WholeShares(value Decimal) int32 {
	shares := new(big.Int).Quo(value.value(), scaleUnit(MoneyScale))
	if shares.Sign() <= 0 {
		return 0
	}
	if shares.Cmp(big.NewInt(int64(Int32Max))) > 0 {
		return Int32Max
	}
	// #nosec G115 -- the preceding comparison bounds shares to Int32Max.
	return int32(shares.Int64())
}

// RoundHalfUp formats a base-10 value to the requested number of places.
func RoundHalfUp(value string, scale int) (string, error) {
	if scale < 0 || scale > 18 {
		return "", fmt.Errorf("scale must be between 0 and 18")
	}
	units, err := scaledUnits(value, scale)
	if err != nil {
		return "", err
	}
	negative := units.Sign() < 0
	units.Abs(units)
	factor := scaleUnit(scale)
	whole, fraction := new(big.Int), new(big.Int)
	whole.QuoRem(units, factor, fraction)
	prefix := ""
	if negative {
		prefix = "-"
	}
	if scale == 0 {
		return prefix + whole.String(), nil
	}
	return fmt.Sprintf("%s%s.%0*s", prefix, whole.String(), scale, fraction.String()), nil
}

func scaledUnits(value string, scale int) (*big.Int, error) {
	value = strings.TrimSpace(value)
	if !validDecimal(value) {
		return nil, fmt.Errorf("invalid decimal %q", value)
	}
	rational, ok := new(big.Rat).SetString(value)
	if !ok {
		return nil, fmt.Errorf("invalid decimal")
	}
	scaled := new(big.Rat).Mul(rational, new(big.Rat).SetInt(scaleUnit(scale)))
	return roundHalfUp(scaled.Num(), scaled.Denom()), nil
}

func validDecimal(value string) bool {
	if value == "" {
		return false
	}
	start := 0
	if value[0] == '+' || value[0] == '-' {
		start = 1
	}
	digits, decimalPoint := 0, false
	for i := start; i < len(value); i++ {
		char := value[i]
		if char >= '0' && char <= '9' {
			digits++
			continue
		}
		if char == '.' && !decimalPoint {
			decimalPoint = true
			continue
		}
		return false
	}
	return digits > 0
}

func scaleUnit(scale int) *big.Int {
	return new(big.Int).Exp(big.NewInt(10), big.NewInt(int64(scale)), nil)
}

func (d Decimal) value() *big.Int {
	if d.units == nil {
		return new(big.Int)
	}
	return new(big.Int).Set(d.units)
}

func roundHalfUp(numerator, denominator *big.Int) *big.Int {
	quotient, remainder := new(big.Int), new(big.Int)
	quotient.QuoRem(numerator, denominator, remainder)
	if new(big.Int).Lsh(new(big.Int).Abs(remainder), 1).Cmp(new(big.Int).Abs(denominator)) >= 0 {
		if numerator.Sign()*denominator.Sign() >= 0 {
			quotient.Add(quotient, big.NewInt(1))
		} else {
			quotient.Sub(quotient, big.NewInt(1))
		}
	}
	return quotient
}
