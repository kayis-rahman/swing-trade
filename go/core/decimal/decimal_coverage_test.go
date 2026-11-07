package decimal

import (
	"encoding/json"
	"math/big"
	"testing"
)

func TestSubReturnsDifferenceWithoutMutation(t *testing.T) {
	left, err := Parse("5.25")
	if err != nil {
		t.Fatal(err)
	}
	right, err := Parse("2.75")
	if err != nil {
		t.Fatal(err)
	}
	if got := left.Sub(right).String(); got != "2.5000" {
		t.Fatalf("Sub() = %s, want 2.5000", got)
	}
	if left.String() != "5.2500" || right.String() != "2.7500" {
		t.Fatalf("Sub() mutated operands: left=%s right=%s", left.String(), right.String())
	}
}

func TestSubAlignsMixedScales(t *testing.T) {
	left, err := ParseScale("1.5", PercentScale)
	if err != nil {
		t.Fatal(err)
	}
	right, err := ParseScale("0.5", MoneyScale)
	if err != nil {
		t.Fatal(err)
	}
	if got := left.Sub(right).String(); got != "1.0000" {
		t.Fatalf("Sub() mixed scale = %s, want 1.0000", got)
	}
}

func TestMulIntScalesByIntegerQuantity(t *testing.T) {
	value, err := Parse("1.25")
	if err != nil {
		t.Fatal(err)
	}
	if got := value.MulInt(3).String(); got != "3.7500" {
		t.Fatalf("MulInt(3) = %s, want 3.7500", got)
	}
	if got := value.MulInt(0).String(); got != "0.0000" {
		t.Fatalf("MulInt(0) = %s, want 0.0000", got)
	}
}

func TestFromUnitsConstructsMoneyScaleDecimal(t *testing.T) {
	units := newBigInt(12345)
	deferred := FromUnits(units)
	if got := deferred.String(); got != "1.2345" {
		t.Fatalf("FromUnits() = %s, want 1.2345", got)
	}
	units.SetInt64(999)
	if deferred.String() != "1.2345" {
		t.Fatalf("FromUnits() did not copy its input")
	}
}

func TestUnitsReturnsIndependentCopy(t *testing.T) {
	value, err := Parse("1.25")
	if err != nil {
		t.Fatal(err)
	}
	units := value.Units()
	if units.Cmp(newBigInt(12500)) != 0 {
		t.Fatalf("Units() = %s, want 12500", units.String())
	}
	units.SetInt64(0)
	if value.Units().Cmp(newBigInt(12500)) != 0 {
		t.Fatal("Units() returned a shared underlying value")
	}
}

func TestParseScaleConstructsAtRequestedScale(t *testing.T) {
	value, err := ParseScale("1.2345675", IndicatorScale)
	if err != nil {
		t.Fatal(err)
	}
	if value.Scale() != IndicatorScale {
		t.Fatalf("ParseScale scale = %d, want %d", value.Scale(), IndicatorScale)
	}
	if got := value.String(); got != "1.234568" {
		t.Fatalf("ParseScale() = %s, want 1.234568", got)
	}
}

func TestParseScaleRejectsInvalidInput(t *testing.T) {
	if _, err := ParseScale("not-a-number", MoneyScale); err == nil {
		t.Fatal("ParseScale() expected an error for invalid input")
	}
}

func TestRoundHalfUpSupportsZeroScaleAndNegativeValues(t *testing.T) {
	got, err := RoundHalfUp("-1.6", 0)
	if err != nil {
		t.Fatal(err)
	}
	if got != "-2" {
		t.Fatalf("RoundHalfUp() = %s, want -2", got)
	}
	if _, err := RoundHalfUp("1.2", -1); err == nil {
		t.Fatal("RoundHalfUp() expected an error for negative scale")
	}
	if _, err := RoundHalfUp("1.2", 19); err == nil {
		t.Fatal("RoundHalfUp() expected an error for scale above 18")
	}
}

func TestMarshalJSONRendersNumber(t *testing.T) {
	value, err := Parse("1.25")
	if err != nil {
		t.Fatal(err)
	}
	data, err := json.Marshal(value)
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != "1.2500" {
		t.Fatalf("MarshalJSON() = %s, want 1.2500", string(data))
	}
}

func TestWholeSharesHandlesNonPositiveAndZeroValue(t *testing.T) {
	negative, err := Parse("-1.25")
	if err != nil {
		t.Fatal(err)
	}
	if got := WholeShares(negative); got != 0 {
		t.Fatalf("WholeShares(negative) = %d, want 0", got)
	}
	if got := WholeShares(Decimal{}); got != 0 {
		t.Fatalf("WholeShares(zero) = %d, want 0", got)
	}
}

func TestCmpHandlesScaledOperands(t *testing.T) {
	left, err := ParseScale("1", PercentScale)
	if err != nil {
		t.Fatal(err)
	}
	right, err := Parse("1.0001")
	if err != nil {
		t.Fatal(err)
	}
	if left.Cmp(right) != -1 {
		t.Fatalf("Cmp() mixed scales returned %d, want -1", left.Cmp(right))
	}
}

func TestScaleDefaultsToMoneyScaleForZeroValue(t *testing.T) {
	if got := (Decimal{}).Scale(); got != MoneyScale {
		t.Fatalf("Scale() zero value = %d, want %d", got, MoneyScale)
	}
}

func newBigInt(value int64) *big.Int {
	return new(big.Int).SetInt64(value)
}
