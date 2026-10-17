package decimal

import "testing"

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

func TestWholeSharesTruncatesTowardZero(t *testing.T) {
	value, err := Parse("12.999")
	if err != nil {
		t.Fatal(err)
	}
	if got := WholeShares(value); got != 12 {
		t.Fatalf("WholeShares() = %d, want 12", got)
	}
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
