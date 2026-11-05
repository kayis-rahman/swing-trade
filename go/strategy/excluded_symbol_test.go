package strategy

import "testing"

func TestNewExcludedSymbol(t *testing.T) {
	e, err := NewExcludedSymbol("AAA", "dq")
	if err != nil || e.Symbol != "AAA" {
		t.Fatal(err)
	}
	if _, err := NewExcludedSymbol("", "r"); err == nil {
		t.Fatal("expected error")
	}
}
