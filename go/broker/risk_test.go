package broker

import (
	"testing"

	coredecimal "github.com/kayis-rahman/swing-trade/go/core/decimal"
)

func dec(s string) coredecimal.Decimal {
	v, _ := coredecimal.Parse(s)
	return v
}

func TestRiskCheckResultMethods(t *testing.T) {
	r := NewRiskCheckResult(true, "msg")
	r.AddWarning("warn %s", "x")
	r.AddError("err")
	r.AddInfo("info {}")
	r.AddMessage("m")
	if r.IsPassed() {
		t.Fatal("expected false after warning")
	}
	if !r.HasWarnings() || !r.HasErrors() {
		t.Fatal("expected warnings/errors")
	}
	r.SetCheckType("T")
	r.SetPassed(true)
	r.SetMessages([]string{"a"})
	if !r.IsPassed() || len(r.GetMessages()) != 1 {
		t.Fatal("set failed")
	}
}

func TestKillSwitch(t *testing.T) {
	ks := NewKillSwitch()
	if ks.Engaged() {
		t.Fatal("should be disengaged")
	}
	ks.Engage()
	if !ks.Engaged() {
		t.Fatal("should engage")
	}
	ks.Disengage()
	if ks.Engaged() {
		t.Fatal("should disengage")
	}
}

func TestPaperEngine(t *testing.T) {
	pe := NewPaperEngine(dec("100"))
	if err := pe.PlaceBuy(2, dec("10")); err != nil {
		t.Fatalf("buy: %v", err)
	}
	if err := pe.PlaceBuy(2, dec("1000")); err == nil {
		t.Fatal("expected insufficient")
	}
	if err := pe.PlaceBuy(0, dec("1")); err == nil {
		t.Fatal("expected quantity")
	}
}

func TestCapitalTracker(t *testing.T) {
	ct := NewCapitalTracker(dec("50000"), 3, dec("10000"))
	positions := []Position{}
	res := ct.EnforceLimits(positions, dec("5000"))
	if !res.IsPassed() {
		t.Fatalf("expected pass: %v", res.GetMessages())
	}
	// max positions
	ps := []Position{{Quantity: 10, EntryPrice: dec("1000")}, {Quantity: 5, EntryPrice: dec("2000")}, {Quantity: 20, EntryPrice: dec("500")}}
	res = ct.EnforceLimits(ps, dec("1000"))
	if res.IsPassed() {
		t.Fatal("expected fail")
	}
	// per position
	res = ct.EnforceLimits([]Position{}, dec("15000"))
	if res.IsPassed() {
		t.Fatal("expected fail")
	}
	if !ct.CanAddPosition([]Position{}, dec("5000")) {
		t.Fatal("can add")
	}
	if ct.GetRemainingPositionCapacity(ps) != 0 {
		t.Fatal("remaining")
	}
	if ct.GetCurrentPositionCount(ps) != 3 {
		t.Fatal("count")
	}
	_ = ct.GetDeployedCapital(ps)
	_ = ct.GetAvailableCapital(ps)
	_ = ct.GetRemainingCapitalPerPosition(ps)
	_ = ct.GetInitialCapital()
	_ = ct.GetMaxPositions()
	_ = ct.GetMaxCapitalPerPosition()
}

func TestPositionSizeValidator(t *testing.T) {
	props := RiskConfig{InitialCapital: dec("50000"), MaxCapitalPerTrade: dec("20000"), MaxPositionSizePercentage: dec("20"), MinPositionSizePercentage: dec("1")}
	v := NewPositionSizeValidator(props)
	res := v.ValidatePositionSize(dec("5000"))
	if !res.IsPassed() {
		t.Fatalf("expected pass: %v", res.GetMessages())
	}
	res = v.ValidatePositionSize(dec("0"))
	if res.IsPassed() {
		t.Fatal("expected fail")
	}
	res = v.ValidatePositionSize(dec("30000"))
	if res.IsPassed() {
		t.Fatal("expected fail")
	}
	_ = v.CalculatePositionPercent(dec("5000"))
	_ = v.GetMaxPositionSize()
	_ = v.CalculateRecommendedPositionSize(dec("2"))
}

func TestPositionLimitChecker(t *testing.T) {
	plc := NewPositionLimitChecker(3, dec("10000"))
	res := plc.CanAddPosition(2, dec("5000"))
	if !res.IsPassed() {
		t.Fatalf("expected pass: %v", res.GetMessages())
	}
	res = plc.CanAddPosition(3, dec("5000"))
	if res.IsPassed() {
		t.Fatal("expected fail")
	}
	res = plc.CanAddPosition(1, dec("15000"))
	if res.IsPassed() {
		t.Fatal("expected fail")
	}
	if plc.GetRemainingPositionCapacity(3) != 0 {
		t.Fatal("remaining")
	}
}

func TestDailyLossCircuitBreaker(t *testing.T) {
	cb := NewDailyLossCircuitBreaker(dec("5"), dec("50000"))
	if !cb.IsTradingAllowed() {
		t.Fatal("should allow")
	}
	res := cb.CanPlaceTrade(dec("1000"), dec("100"))
	if !res.IsPassed() {
		t.Fatalf("expected pass: %v", res.GetMessages())
	}
	// trip
	cb.UpdateWithCurrentPositions(dec("-3000"), dec("0"))
	if !cb.IsCircuitOpen() {
		t.Fatal("expected open")
	}
	if cb.CanPlaceTrade(dec("100"), dec("10")).IsPassed() {
		t.Fatal("expected blocked")
	}
	cb.CloseCircuit()
	if cb.IsCircuitOpen() {
		t.Fatal("expected closed")
	}
	cb.OpenCircuit()
	if !cb.IsCircuitOpen() {
		t.Fatal("expected open")
	}
	_ = cb.GetCircuitOpenTime()
	_ = cb.GetLossAtCircuitOpen()
	_ = cb.GetLossPercent()
	_ = cb.GetCurrentDailyPnL()
}
