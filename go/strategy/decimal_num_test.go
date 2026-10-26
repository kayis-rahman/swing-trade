package strategy

import "testing"

func TestDecimalNumConstructorsMatchTA4JPrecisionBases(t *testing.T) {
	t.Parallel()

	fromString, err := newDecimalNum("150.123456789")
	if err != nil {
		t.Fatalf("newDecimalNum() error = %v", err)
	}
	assertDecimalNum(t, fromString, "150.123456789", 32)
	assertDecimalNum(t, decimalNumFromInt(7), "7", 32)

	fromBigDecimal, err := decimalNumFromBigDecimal("1.2300")
	if err != nil {
		t.Fatalf("decimalNumFromBigDecimal() error = %v", err)
	}
	assertDecimalNum(t, fromBigDecimal, "1.2300", 5)

	fromPrecision, err := newDecimalNumWithPrecision("1.2300", 5)
	if err != nil {
		t.Fatalf("newDecimalNumWithPrecision() error = %v", err)
	}
	assertDecimalNum(t, fromPrecision, "1.2300", 5)
}

func TestDecimalNumArithmeticMatchesTA4JDecimalNum(t *testing.T) {
	t.Parallel()

	left, err := newDecimalNumWithPrecision("1.23456789", 5)
	if err != nil {
		t.Fatalf("left: %v", err)
	}
	right, err := newDecimalNumWithPrecision("0.000067891234", 7)
	if err != nil {
		t.Fatalf("right: %v", err)
	}

	assertDecimalNum(t, left.add(right), "1.2347", 5)
	assertDecimalNum(t, left.sub(right), "1.2345", 5)
	assertDecimalNum(t, left.multiply(right), "0.000083819", 5)

	quotient, err := left.divide(right)
	if err != nil {
		t.Fatalf("divide: %v", err)
	}
	assertDecimalNum(t, quotient, "18185", 5)
}

func TestDecimalNumDivisionPreservesExactPreferredScaleAndRoundsHalfUp(t *testing.T) {
	t.Parallel()

	oneThird, err := decimalNumFromInt(1).divide(decimalNumFromInt(3))
	if err != nil {
		t.Fatalf("one third: %v", err)
	}
	assertDecimalNum(t, oneThird, "0.33333333333333333333333333333333", 32)

	left, err := newDecimalNum("2.00")
	if err != nil {
		t.Fatalf("left: %v", err)
	}
	right, err := newDecimalNum("4")
	if err != nil {
		t.Fatalf("right: %v", err)
	}
	exact, err := left.divide(right)
	if err != nil {
		t.Fatalf("exact division: %v", err)
	}
	assertDecimalNum(t, exact, "0.50", 32)

	rounded, err := newDecimalNumWithPrecision("99999", 3)
	if err != nil {
		t.Fatalf("rounded source: %v", err)
	}
	assertDecimalNum(t, rounded.add(decimalNumFromInt(1)), "100000", 3)
}

func assertDecimalNum(t *testing.T, actual decimalNum, want string, wantPrecision int) {
	t.Helper()
	if actual.String() != want {
		t.Errorf("String() = %q, want %q", actual.String(), want)
	}
	if actual.precision != wantPrecision {
		t.Errorf("precision = %d, want %d", actual.precision, wantPrecision)
	}
}
