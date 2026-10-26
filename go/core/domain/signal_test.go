package domain

import "testing"

// Ported from SignalTest.SignalTypeEnumTests: descriptions and value identity.
func TestSignalTypeDescriptionsMatchJava(t *testing.T) {
	for signalType, want := range map[SignalType]string{
		SignalTypeBuy:  "Buy Signal",
		SignalTypeSell: "Sell Signal",
		SignalTypeHold: "Hold",
	} {
		if got := signalType.Description(); got != want {
			t.Fatalf("%s description = %q, want %q", signalType, got, want)
		}
	}
}

func TestSignalTypeValuesCompareByEnumIdentity(t *testing.T) {
	if SignalTypeBuy == SignalTypeSell || SignalTypeSell == SignalTypeHold || SignalTypeBuy == SignalTypeHold {
		t.Fatal("distinct signal types compared equal")
	}
	if SignalTypeBuy != SignalType("BUY") {
		t.Fatal("signal type wire name changed")
	}
	if SignalType("UNKNOWN").Description() != "" {
		t.Fatal("unknown signal type should have no description")
	}
}
