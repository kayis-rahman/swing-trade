package pipeline

import (
	"testing"

	"github.com/kayis-rahman/swing-trade/go/core/decimal"
	"github.com/kayis-rahman/swing-trade/go/core/domain"
)

func TestPipelineNew(t *testing.T) {
	p := New(nil)
	if p == nil {
		t.Fatal("nil pipeline")
	}
}

func TestArbiterPickWrapper(t *testing.T) {
	d, _ := decimal.Parse("0.9000")
	conf := d
	outcomes := []SignalOutcome{
		{VariantID: "v1", Type: domain.SignalTypeBuy, Persisted: true, Confidence: &conf},
		{VariantID: "v2", Type: domain.SignalTypeBuy, Persisted: true, Confidence: &conf},
	}
	w := ArbiterPick(outcomes, RuleHighestConfidence, map[string]float64{}, map[string]struct{}{}, nil)
	if w == nil || w.VariantID != "v1" {
		t.Fatalf("unexpected winner: %+v", w)
	}
}

func TestArbiterPickVariantWrapper(t *testing.T) {
	d, _ := decimal.Parse("0.9000")
	res := ArbiterPickVariant([]Candidate{{VariantID: "v1", Confidence: d}}, RuleHighestConfidence, map[string]float64{})
	if res == nil || *res != "v1" {
		t.Fatalf("unexpected: %v", res)
	}
}
