package llm

import "testing"

func TestParseSynthesisResponse(t *testing.T) {
	text := `{"narrative":"n","recommendation":"BUY","confidence":0.8,"keyDrivers":["k"],"bullishFactors":["b"],"bearishFactors":null,"conflictDetected":true,"eventRiskDetected":false,"eventRiskReason":""}`
	got, err := ParseSynthesisResponse(text)
	if err != nil || got.Recommendation != "BUY" || !got.ConflictDetected {
		t.Fatalf("got %+v err %v", got, err)
	}
	if got.BearishFactors == nil {
		t.Fatal("nil bearish")
	}
}
