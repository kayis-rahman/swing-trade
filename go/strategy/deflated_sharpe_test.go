package strategy

import (
	"math"
	"testing"
)

func TestDeflatedSharpeRatioMoreTrialsLowersDsr(t *testing.T) {
	observed := 0.6
	obs := 250
	se := 0.5
	d1, _ := ComputeDeflatedSharpeRatio(observed, 1, obs, se)
	d2, _ := ComputeDeflatedSharpeRatio(observed, 2, obs, se)
	d5, _ := ComputeDeflatedSharpeRatio(observed, 5, obs, se)
	d10, _ := ComputeDeflatedSharpeRatio(observed, 10, obs, se)
	if !(d1 > d2 && d2 > d5 && d5 > d10) {
		t.Fatalf("monotonicity failed: %v %v %v %v", d1, d2, d5, d10)
	}
}

func TestDeflatedSharpeRatioN1(t *testing.T) {
	dsr, _ := ComputeDeflatedSharpeRatio(1.2, 1, 300, 0.3)
	expected := normalCdf(1.2 * math.Sqrt(299))
	if math.Abs(dsr-expected) > 1e-9 {
		t.Fatalf("got %v want %v", dsr, expected)
	}
}

func TestDeflatedSharpeRatioHighN(t *testing.T) {
	dsr, _ := ComputeDeflatedSharpeRatio(1.0, 1000000, 500, 0.3)
	if dsr >= 0.05 {
		t.Fatalf("got %v", dsr)
	}
}

func TestDeflatedSharpeRatioBounded(t *testing.T) {
	for _, n := range []int{1, 5, 50, 5000} {
		dsr, _ := ComputeDeflatedSharpeRatio(2.5, n, 1000, 0.5)
		if dsr < 0 || dsr > 1 {
			t.Fatalf("n=%d got %v", n, dsr)
		}
	}
}

func TestDeflatedSharpeRatioInvalid(t *testing.T) {
	if _, err := ComputeDeflatedSharpeRatio(1.0, 0, 100, 0.3); err == nil {
		t.Fatal("expected error")
	}
	if _, err := ComputeDeflatedSharpeRatio(1.0, 1, 1, 0.3); err == nil {
		t.Fatal("expected error")
	}
}
