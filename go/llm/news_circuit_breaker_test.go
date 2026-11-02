package llm

import "testing"

func TestNewsCircuitBreaker(t *testing.T) {
	cb := NewNewsCircuitBreaker(2)
	if err := cb.Allow(); err != nil {
		t.Fatal(err)
	}
	cb.Failure()
	cb.Failure()
	if err := cb.Allow(); err != ErrCircuitOpen {
		t.Fatalf("expected open, got %v", err)
	}
	cb.Success()
	if err := cb.Allow(); err != nil {
		t.Fatal(err)
	}
}
