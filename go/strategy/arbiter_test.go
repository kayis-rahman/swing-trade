package strategy

import (
	"testing"
	"time"

	coredecimal "github.com/kayis-rahman/swing-trade/go/core/decimal"
	"github.com/kayis-rahman/swing-trade/go/core/domain"
)

func mustDec(t *testing.T, v string) *coredecimal.Decimal {
	t.Helper()
	d, err := coredecimal.Parse(v)
	if err != nil {
		t.Fatal(err)
	}
	return &d
}

func outcome(id string, typ domain.SignalType, conf string, persisted bool) VariantSignalOutcome {
	c := mustDec(&testing.T{}, conf)
	_ = c
	d, _ := coredecimal.Parse(conf)
	return VariantSignalOutcome{id, 1, typ, false, persisted, &d}
}

func TestArbiterPicksHighestConfidenceBuy(t *testing.T) {
	w := Pick([]VariantSignalOutcome{
		outcome("breakout-v1", domain.SignalTypeBuy, "0.60", true),
		outcome("squeeze-v1", domain.SignalTypeBuy, "0.85", true),
		outcome("pullback-v1", domain.SignalTypeBuy, "0.75", true),
	})
	if w == nil || w.VariantID != "squeeze-v1" {
		t.Fatalf("got %+v", w)
	}
}

func TestArbiterIgnoresNonBuyAndUnpersisted(t *testing.T) {
	w := Pick([]VariantSignalOutcome{
		outcome("a", domain.SignalTypeSell, "0.99", true),
		outcome("b", domain.SignalTypeHold, "0.95", true),
		outcome("c", domain.SignalTypeBuy, "0.90", false),
		outcome("d", domain.SignalTypeBuy, "0.40", true),
	})
	if w == nil || w.VariantID != "d" {
		t.Fatalf("got %+v", w)
	}
}

func TestArbiterTieBreaksVariantId(t *testing.T) {
	w := Pick([]VariantSignalOutcome{
		outcome("zeta", domain.SignalTypeBuy, "0.70", true),
		outcome("alpha", domain.SignalTypeBuy, "0.70", true),
	})
	if w == nil || w.VariantID != "alpha" {
		t.Fatalf("got %+v", w)
	}
}

func TestArbiterOpenPositionWins(t *testing.T) {
	open := map[string]struct{}{"open-position": {}}
	champ := "champion"
	w := PickFull([]VariantSignalOutcome{
		outcome("champion", domain.SignalTypeBuy, "0.60", true),
		outcome("open-position", domain.SignalTypeBuy, "0.40", true),
		outcome("flashy", domain.SignalTypeBuy, "0.99", true),
	}, ArbitrationRuleHighestConfidence, map[string]float64{}, &open, &champ)
	if w == nil || w.VariantID != "open-position" {
		t.Fatalf("got %+v", w)
	}
}

func TestArbiterChampionWins(t *testing.T) {
	champ := "champion"
	w := PickFull([]VariantSignalOutcome{
		outcome("champion", domain.SignalTypeBuy, "0.60", true),
		outcome("flashy", domain.SignalTypeBuy, "0.99", true),
	}, ArbitrationRuleHighestConfidence, map[string]float64{}, nil, &champ)
	if w == nil || w.VariantID != "champion" {
		t.Fatalf("got %+v", w)
	}
}

func TestArbiterFallback(t *testing.T) {
	open := map[string]struct{}{"open-position": {}}
	champ := "champion"
	w := PickFull([]VariantSignalOutcome{
		outcome("champion", domain.SignalTypeHold, "0.99", true),
		outcome("flashy", domain.SignalTypeBuy, "0.75", true),
	}, ArbitrationRuleHighestConfidence, map[string]float64{}, &open, &champ)
	if w == nil || w.VariantID != "flashy" {
		t.Fatalf("got %+v", w)
	}
}

func TestArbiterNoWinner(t *testing.T) {
	w := Pick([]VariantSignalOutcome{outcome("a", domain.SignalTypeHold, "0.5", true)})
	if w != nil {
		t.Fatal("expected nil")
	}
	w = Pick(nil)
	if w != nil {
		t.Fatal("expected nil")
	}
}

func TestArbiterEvidence(t *testing.T) {
	asOf := time.Date(2026, 9, 10, 0, 0, 0, 0, time.UTC)
	trade := func(pnl string, exit time.Time) ShadowClosedTrade {
		d, _ := coredecimal.Parse(pnl)
		entry, _ := coredecimal.Parse("100")
		return ShadowClosedTrade{PortfolioID: "v", ExitDate: &exit, EntryPrice: &entry, Quantity: 10, Pnl: &d}
	}
	enough := make([]ShadowClosedTrade, 5)
	for i := 0; i < 5; i++ {
		enough[i] = trade("50", time.Date(2026, 9, 1, 0, 0, 0, 0, time.UTC).AddDate(0, 0, i))
	}
	val, ok := Evidence(enough, asOf)
	if !ok || val != 5.0 {
		t.Fatalf("got %v %v", val, ok)
	}
	_, ok = Evidence(enough[:4], asOf)
	if ok {
		t.Fatal("expected false")
	}
	withFuture := append([]ShadowClosedTrade{}, enough[:4]...)
	fut := trade("500", time.Date(2026, 9, 20, 0, 0, 0, 0, time.UTC))
	withFuture = append(withFuture, fut)
	_, ok = Evidence(withFuture, asOf)
	if ok {
		t.Fatal("expected false")
	}
}

func TestArbiterPickVariantEvidence(t *testing.T) {
	candidates := []Candidate{
		{VariantID: "flashy", Confidence: mustDecVal("0.90")},
		{VariantID: "proven", Confidence: mustDecVal("0.60")},
	}
	ev := map[string]float64{"proven": 3.0, "flashy": -2.0}
	if v := PickVariant(candidates, ArbitrationRuleHighestConfidence, ev); v == nil || *v != "flashy" {
		t.Fatalf("got %v", v)
	}
	if v := PickVariant(candidates, ArbitrationRuleEvidenceRanked, ev); v == nil || *v != "proven" {
		t.Fatalf("got %v", v)
	}
	// neutral fallback
	c2 := []Candidate{
		{VariantID: "a", Confidence: mustDecVal("0.55")},
		{VariantID: "b", Confidence: mustDecVal("0.80")},
	}
	if v := PickVariant(c2, ArbitrationRuleEvidenceRanked, map[string]float64{}); v == nil || *v != "b" {
		t.Fatalf("got %v", v)
	}
	if v := PickVariant(c2, ArbitrationRuleEvidenceRanked, map[string]float64{"a": 1.0}); v == nil || *v != "a" {
		t.Fatalf("got %v", v)
	}
}

func mustDecVal(s string) coredecimal.Decimal {
	d, _ := coredecimal.Parse(s)
	return d
}
