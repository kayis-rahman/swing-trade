package decimal

import "testing"

func TestCmpComparesNumerically(t *testing.T) {
	one, err := Parse("1.00")
	if err != nil {
		t.Fatal(err)
	}
	two, err := Parse("2.00")
	if err != nil {
		t.Fatal(err)
	}
	oneCopy := one
	if one.Cmp(two) != -1 || two.Cmp(one) != 1 || one.Cmp(oneCopy) != 0 {
		t.Fatal("Cmp returned unexpected ordering")
	}
}

func TestParseUsesFinancialScaleAndHalfUp(t *testing.T) {
	got, err := Parse("1.00005")
	if err != nil {
		t.Fatal(err)
	}
	if got.String() != "1.0001" {
		t.Fatalf("Parse() = %s, want 1.0001", got)
	}
}

func TestPercentOfAndZeroDenominatorMatchFinancialScale(t *testing.T) {
	numerator, _ := Parse("1")
	denominator, _ := Parse("6")
	if got := PercentOf(numerator, denominator).String(); got != "16.6667" {
		t.Fatalf("PercentOf() = %s, want 16.6667", got)
	}
	if got := PercentOf(numerator, Decimal{}).String(); got != "0.0000" {
		t.Fatalf("PercentOf(zero denominator) = %s, want 0.0000", got)
	}
}

func TestIndicatorScaleUsesHalfUp(t *testing.T) {
	got, err := RoundHalfUp("1.2345675", IndicatorScale)
	if err != nil {
		t.Fatal(err)
	}
	if got != "1.234568" {
		t.Fatalf("RoundHalfUp() = %s, want 1.234568", got)
	}
}

func TestWholeSharesTruncatesAndSaturatesLikeJava(t *testing.T) {
	value, err := Parse("12.999")
	if err != nil {
		t.Fatal(err)
	}
	if got := WholeShares(value); got != 12 {
		t.Fatalf("WholeShares() = %d, want 12", got)
	}
	overflow, err := Parse("2147483648")
	if err != nil {
		t.Fatal(err)
	}
	if got := WholeShares(overflow); got != Int32Max {
		t.Fatalf("WholeShares(overflow) = %d, want %d", got, Int32Max)
	}
}

func FuzzParseRoundTrip(f *testing.F) {
	for _, input := range []string{"0.1234", "-0.00005", ".5", "NaN", "1e4", "", "999999999999999999999999"} {
		f.Add(input)
	}
	f.Fuzz(func(t *testing.T, input string) {
		value, err := Parse(input)
		if err != nil {
			return
		}
		roundTrip, err := Parse(value.String())
		if err != nil {
			t.Fatalf("Parse(%q) failed after canonical formatting: %v", value.String(), err)
		}
		if roundTrip.String() != value.String() {
			t.Fatalf("round trip = %q, want %q", roundTrip.String(), value.String())
		}
	})
}

func TestAddDoesNotMutateOperands(t *testing.T) {
	left, _ := Parse("1.25")
	right, _ := Parse("2.75")
	if got := left.Add(right).String(); got != "4.0000" {
		t.Fatalf("Add() = %s, want 4.0000", got)
	}
	if left.String() != "1.2500" || right.String() != "2.7500" {
		t.Fatalf("Add() mutated operands: left=%s right=%s", left.String(), right.String())
	}
}
