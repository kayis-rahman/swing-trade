package strategy

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"testing"

	coredecimal "github.com/kayis-rahman/swing-trade/go/core/decimal"
)

func sha256HexOf(t *testing.T, payload string) string {
	t.Helper()
	sum := sha256.Sum256([]byte(payload))
	return hex.EncodeToString(sum[:])
}

// The canonical JSON vector was computed independently from Java's Jackson
// serialization: sorted keys at every level, no whitespace, no HTML escaping.
func TestStrategyConfigHasherMatchesJacksonVector(t *testing.T) {
	payload := `{"overlays":{"sentimentGate":true},"params":{"emaFast":20,"emaSlow":50},"strategyType":"BREAKOUT"}`
	got, err := ConfigHasher("BREAKOUT",
		map[string]any{"emaFast": 20, "emaSlow": 50}, map[string]any{"sentimentGate": true})
	if err != nil {
		t.Fatalf("hash error = %v", err)
	}
	if want := sha256HexOf(t, payload); got != want {
		t.Fatalf("hasher = %s, want %s", got, want)
	}
	if len(got) != 64 {
		t.Fatalf("hash length = %d", len(got))
	}
}

func TestStrategyConfigHasherIgnoresKeyOrder(t *testing.T) {
	first, err := ConfigHasher("BREAKOUT", map[string]any{"emaFast": 20, "emaSlow": 50},
		map[string]any{"sentimentGate": true})
	if err != nil {
		t.Fatal(err)
	}
	second, err := ConfigHasher("BREAKOUT", map[string]any{"emaSlow": 50, "emaFast": 20},
		map[string]any{"sentimentGate": true})
	if err != nil {
		t.Fatal(err)
	}
	if first != second {
		t.Fatal("key order changed the hash")
	}
}

func TestStrategyConfigHasherDistinguishesInputs(t *testing.T) {
	base, err := ConfigHasher("BREAKOUT", map[string]any{"emaFast": 20}, map[string]any{})
	if err != nil {
		t.Fatal(err)
	}
	otherParams, err := ConfigHasher("BREAKOUT", map[string]any{"emaFast": 21}, map[string]any{})
	if err != nil {
		t.Fatal(err)
	}
	otherType, err := ConfigHasher("PULLBACK", map[string]any{"emaFast": 20}, map[string]any{})
	if err != nil {
		t.Fatal(err)
	}
	if base == otherParams || base == otherType {
		t.Fatal("distinct inputs collided")
	}
}

// Go must keep Jackson's raw numeric spelling: no HTML escaping and a minimum
// of one decimal place for whole doubles.
func TestStrategyConfigHasherPreservesNumericAndHTMLEscaping(t *testing.T) {
	payload := `{"overlays":{},"params":{"label":"a&b","ratio":20.0},"strategyType":"X"}`
	got, err := ConfigHasher("X", map[string]any{"label": "a&b", "ratio": float64(20)}, map[string]any{})
	if err != nil {
		t.Fatal(err)
	}
	if want := sha256HexOf(t, payload); got != want {
		t.Fatalf("escaped/numeric canonical drifted: got %s want %s", got, want)
	}
}

func TestStrategyConfigHasherEncodesDecimalsAsNumbers(t *testing.T) {
	payload := `{"overlays":{},"params":{"proximity":0.9700},"strategyType":"X"}`
	decimalValue, err := coredecimal.Parse("0.97")
	if err != nil {
		t.Fatal(err)
	}
	got, err := ConfigHasher("X", map[string]any{"proximity": decimalValue}, map[string]any{})
	if err != nil {
		t.Fatal(err)
	}
	if want := sha256HexOf(t, payload); got != want {
		t.Fatalf("decimal canonical drifted: got %s want %s", got, want)
	}
}

func TestDecodeParamsJSONKeepsNumericSpelling(t *testing.T) {
	decoded, err := DecodeParamsJSON([]byte(`{"emaFast":20,"ratio":0.97,"mode":"STRICT","on":true}`))
	if err != nil {
		t.Fatal(err)
	}
	if decoded["emaFast"] != json.Number("20") || decoded["ratio"] != json.Number("0.97") ||
		decoded["mode"] != "STRICT" || decoded["on"] != true {
		t.Fatalf("decoded = %#v", decoded)
	}
	// Integer spellings hash identically whether passed as int or json.Number.
	viaInt, err := ConfigHasher("X", map[string]any{"emaFast": 20}, map[string]any{})
	if err != nil {
		t.Fatal(err)
	}
	viaNumber, err := ConfigHasher("X", map[string]any{"emaFast": json.Number("20")}, map[string]any{})
	if err != nil {
		t.Fatal(err)
	}
	if viaInt != viaNumber {
		t.Fatal("json.Number spelling changed the hash")
	}
	if _, err := DecodeParamsJSON([]byte("{not json")); err == nil {
		t.Fatal("malformed params decoded")
	}
}
