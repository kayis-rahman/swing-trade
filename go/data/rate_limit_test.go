package data

import (
	"context"
	"errors"
	"testing"

	coreclock "github.com/kayis-rahman/swing-trade/go/core/clock"
)

func TestRequestPacerAllowsFirstRequestAndPacesLaterRequests(t *testing.T) {
	pacer := NewRequestPacer(60_000_000_000, coreclock.System{})
	if err := pacer.Acquire(context.Background()); err != nil {
		t.Fatalf("first Acquire() error = %v", err)
	}
	if err := pacer.Acquire(context.Background()); err != nil {
		t.Fatalf("second Acquire() error = %v", err)
	}
}

func TestRequestPacerCancellationInterruptsWait(t *testing.T) {
	pacer := NewRequestPacer(1, coreclock.System{})
	if err := pacer.Acquire(context.Background()); err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if err := pacer.Acquire(ctx); !errors.Is(err, context.Canceled) {
		t.Fatalf("Acquire() error = %v, want context canceled", err)
	}
}

func TestRequestPacerNonpositiveQuotaIsUnlimited(t *testing.T) {
	for _, quota := range []int64{0, -1} {
		pacer := NewRequestPacer(quota, nil)
		for range 3 {
			if err := pacer.Acquire(context.Background()); err != nil {
				t.Fatalf("quota %d Acquire() error = %v", quota, err)
			}
		}
	}
}
