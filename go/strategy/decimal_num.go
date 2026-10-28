package strategy

import (
	"fmt"
	"math/big"
	"strings"
)

// decimalNum reproduces ta4j DecimalNum's BigDecimal delegate and per-instance
// HALF_UP MathContext. It is deliberately separate from core/decimal, whose
// fixed financial scale cannot represent indicator arithmetic.
type decimalNum struct {
	coefficient *big.Int
	scale       int
	precision   int
}

func newDecimalNum(value string) (decimalNum, error) {
	coefficient, scale, err := parseDecimalCoefficient(value)
	if err != nil {
		return decimalNum{}, err
	}
	return newDecimalNumFromParts(coefficient, scale, max(decimalPrecision(coefficient), 32)), nil
}

// decimalNumFromBigDecimal reproduces DecimalNum.valueOf(BigDecimal): the
// delegate remains unrounded and its own significant-digit count becomes the
// MathContext precision.
func decimalNumFromBigDecimal(value string) (decimalNum, error) {
	coefficient, scale, err := parseDecimalCoefficient(value)
	if err != nil {
		return decimalNum{}, err
	}
	return newDecimalNumFromParts(coefficient, scale, decimalPrecision(coefficient)), nil
}

func newDecimalNumWithPrecision(value string, precision int) (decimalNum, error) {
	coefficient, scale, err := parseDecimalCoefficient(value)
	if err != nil {
		return decimalNum{}, err
	}
	if precision <= 0 {
		return decimalNum{}, fmt.Errorf("precision must be positive")
	}
	return newDecimalNumFromPartsRounded(coefficient, scale, precision), nil
}

func decimalNumFromInt(value int) decimalNum {
	return newDecimalNumFromParts(big.NewInt(int64(value)), 0, 32)
}

func newDecimalNumFromParts(coefficient *big.Int, scale, precision int) decimalNum {
	return decimalNum{coefficient: new(big.Int).Set(coefficient), scale: scale, precision: precision}
}

func newDecimalNumFromPartsRounded(coefficient *big.Int, scale, precision int) decimalNum {
	coefficient, scale = roundDecimalToPrecision(coefficient, scale, precision)
	return newDecimalNumFromParts(coefficient, scale, precision)
}

func (value decimalNum) add(other decimalNum) decimalNum {
	scale := max(value.scale, other.scale)
	left := scaleCoefficient(value.coefficient, scale-value.scale)
	right := scaleCoefficient(other.coefficient, scale-other.scale)
	return newDecimalNumFromPartsRounded(new(big.Int).Add(left, right), scale, value.effectivePrecision())
}

func (value decimalNum) sub(other decimalNum) decimalNum {
	scale := max(value.scale, other.scale)
	left := scaleCoefficient(value.coefficient, scale-value.scale)
	right := scaleCoefficient(other.coefficient, scale-other.scale)
	return newDecimalNumFromPartsRounded(new(big.Int).Sub(left, right), scale, value.effectivePrecision())
}

func (value decimalNum) multiply(other decimalNum) decimalNum {
	return newDecimalNumFromPartsRounded(
		new(big.Int).Mul(cloneBigInt(value.coefficient), cloneBigInt(other.coefficient)),
		value.scale+other.scale,
		value.effectivePrecision(),
	)
}

func (value decimalNum) divide(other decimalNum) (decimalNum, error) {
	if cloneBigInt(other.coefficient).Sign() == 0 {
		return decimalNum{}, fmt.Errorf("division by zero")
	}

	numerator, denominator := cloneBigInt(value.coefficient), cloneBigInt(other.coefficient)
	shift := other.scale - value.scale
	if shift >= 0 {
		numerator = scaleCoefficient(numerator, shift)
	} else {
		denominator = scaleCoefficient(denominator, -shift)
	}

	precision := value.effectivePrecision()
	if coefficient, scale, ok := exactDecimalQuotient(numerator, denominator, value.scale-other.scale); ok {
		return newDecimalNumFromPartsRounded(coefficient, scale, precision), nil
	}

	scale := precision - decimalExponent(numerator, denominator) - 1
	quotient, remainder := divideAtScale(numerator, denominator, scale)
	if shouldRoundHalfUp(remainder, denominator) {
		if numerator.Sign()*denominator.Sign() >= 0 {
			quotient.Add(quotient, big.NewInt(1))
		} else {
			quotient.Sub(quotient, big.NewInt(1))
		}
	}
	return newDecimalNumFromPartsRounded(quotient, scale, precision), nil
}

// divideRounded mirrors BigDecimal.divide(other, scale, HALF_UP): a fixed
// result scale independent of either operand's significant digits.
func (value decimalNum) divideRounded(other decimalNum, scale int) (decimalNum, error) {
	if cloneBigInt(other.coefficient).Sign() == 0 {
		return decimalNum{}, fmt.Errorf("division by zero")
	}
	numerator, denominator := cloneBigInt(value.coefficient), cloneBigInt(other.coefficient)
	shift := other.scale - value.scale
	if shift >= 0 {
		numerator = scaleCoefficient(numerator, shift)
	} else {
		denominator = scaleCoefficient(denominator, -shift)
	}
	quotient, remainder := divideAtScale(numerator, denominator, scale)
	if shouldRoundHalfUp(remainder, denominator) {
		if numerator.Sign()*denominator.Sign() >= 0 {
			quotient.Add(quotient, big.NewInt(1))
		} else {
			quotient.Sub(quotient, big.NewInt(1))
		}
	}
	return newDecimalNumFromPartsRounded(quotient, scale, max(scale, value.effectivePrecision())), nil
}

// effectivePrecision makes the zero value usable: Java's DecimalNum always
// carries a MathContext, while a Go zero struct carries none. Without this,
// an accumulator declared as `var sum decimalNum` would round every result to
// zero significant digits.
func (value decimalNum) effectivePrecision() int {
	if value.precision <= 0 {
		return 32
	}
	return value.precision
}

// String is BigDecimal.toPlainString-compatible for the finite values used by
// MarketContext. It deliberately preserves the delegate's trailing zero scale.
func (value decimalNum) String() string {
	coefficient := cloneBigInt(value.coefficient)
	negative := coefficient.Sign() < 0
	coefficient.Abs(coefficient)
	prefix := ""
	if negative {
		prefix = "-"
	}
	digits := coefficient.String()
	if value.scale <= 0 {
		return prefix + digits + strings.Repeat("0", -value.scale)
	}
	if len(digits) <= value.scale {
		return prefix + "0." + strings.Repeat("0", value.scale-len(digits)) + digits
	}
	return prefix + digits[:len(digits)-value.scale] + "." + digits[len(digits)-value.scale:]
}

func parseDecimalCoefficient(value string) (*big.Int, int, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return nil, 0, fmt.Errorf("invalid decimal %q", value)
	}
	negative := false
	if value[0] == '+' || value[0] == '-' {
		negative = value[0] == '-'
		value = value[1:]
	}
	parts := strings.Split(value, ".")
	if len(parts) > 2 || (len(parts) == 2 && parts[0] == "" && parts[1] == "") {
		return nil, 0, fmt.Errorf("invalid decimal %q", value)
	}
	whole := parts[0]
	fraction := ""
	if len(parts) == 2 {
		fraction = parts[1]
	}
	if whole == "" {
		whole = "0"
	}
	for _, character := range whole + fraction {
		if character < '0' || character > '9' {
			return nil, 0, fmt.Errorf("invalid decimal %q", value)
		}
	}
	coefficient, ok := new(big.Int).SetString(whole+fraction, 10)
	if !ok {
		return nil, 0, fmt.Errorf("invalid decimal %q", value)
	}
	if negative {
		coefficient.Neg(coefficient)
	}
	return coefficient, len(fraction), nil
}

func exactDecimalQuotient(numerator, denominator *big.Int, preferredScale int) (*big.Int, int, bool) {
	numerator, denominator = cloneBigInt(numerator), cloneBigInt(denominator)
	if denominator.Sign() < 0 {
		numerator.Neg(numerator)
		denominator.Neg(denominator)
	}
	gcd := new(big.Int).GCD(nil, nil, numerator, denominator)
	numerator.Quo(numerator, gcd)
	denominator.Quo(denominator, gcd)

	twos, fives := 0, 0
	for new(big.Int).Rem(denominator, big.NewInt(2)).Sign() == 0 {
		denominator.Quo(denominator, big.NewInt(2))
		twos++
	}
	for new(big.Int).Rem(denominator, big.NewInt(5)).Sign() == 0 {
		denominator.Quo(denominator, big.NewInt(5))
		fives++
	}
	if denominator.Cmp(big.NewInt(1)) != 0 {
		return nil, 0, false
	}

	scale := max(twos, fives)
	if twos < scale {
		numerator.Mul(numerator, pow(2, scale-twos))
	}
	if fives < scale {
		numerator.Mul(numerator, pow(5, scale-fives))
	}
	if preferredScale > scale {
		numerator.Mul(numerator, decimalNumPow10(preferredScale-scale))
		scale = preferredScale
	}
	return numerator, scale, true
}

func divideAtScale(numerator, denominator *big.Int, scale int) (*big.Int, *big.Int) {
	if scale >= 0 {
		numerator = scaleCoefficient(numerator, scale)
	} else {
		denominator = scaleCoefficient(denominator, -scale)
	}
	return new(big.Int).QuoRem(numerator, denominator, new(big.Int))
}

func decimalExponent(numerator, denominator *big.Int) int {
	numerator, denominator = new(big.Int).Abs(numerator), new(big.Int).Abs(denominator)
	exponent := len(numerator.String()) - len(denominator.String())
	if exponent >= 0 {
		if numerator.Cmp(scaleCoefficient(denominator, exponent)) < 0 {
			exponent--
		}
	} else if scaleCoefficient(numerator, -exponent).Cmp(denominator) < 0 {
		exponent--
	}
	return exponent
}

func roundDecimalToPrecision(coefficient *big.Int, scale, precision int) (*big.Int, int) {
	coefficient = cloneBigInt(coefficient)
	shift := decimalPrecision(coefficient) - precision
	if shift <= 0 {
		return coefficient, scale
	}
	quotient, remainder := new(big.Int).QuoRem(coefficient, decimalNumPow10(shift), new(big.Int))
	if shouldRoundHalfUp(remainder, decimalNumPow10(shift)) {
		if coefficient.Sign() >= 0 {
			quotient.Add(quotient, big.NewInt(1))
		} else {
			quotient.Sub(quotient, big.NewInt(1))
		}
	}
	if decimalPrecision(quotient) > precision {
		quotient.Quo(quotient, big.NewInt(10))
		shift++
	}
	return quotient, scale - shift
}

func shouldRoundHalfUp(remainder, denominator *big.Int) bool {
	return new(big.Int).Lsh(new(big.Int).Abs(remainder), 1).Cmp(new(big.Int).Abs(denominator)) >= 0
}

func decimalPrecision(coefficient *big.Int) int {
	if coefficient.Sign() == 0 {
		return 1
	}
	return len(new(big.Int).Abs(coefficient).String())
}

func scaleCoefficient(coefficient *big.Int, scale int) *big.Int {
	return new(big.Int).Mul(cloneBigInt(coefficient), decimalNumPow10(scale))
}

func decimalNumPow10(exponent int) *big.Int {
	return pow(10, exponent)
}

func pow(base, exponent int) *big.Int {
	return new(big.Int).Exp(big.NewInt(int64(base)), big.NewInt(int64(exponent)), nil)
}

func cloneBigInt(value *big.Int) *big.Int {
	if value == nil {
		return new(big.Int)
	}
	return new(big.Int).Set(value)
}

// compare mirrors BigDecimal.compareTo for indicator values.
func (value decimalNum) compare(other decimalNum) int {
	scale := max(value.scale, other.scale)
	left := scaleCoefficient(value.coefficient, scale-value.scale)
	right := scaleCoefficient(other.coefficient, scale-other.scale)
	return left.Cmp(right)
}
