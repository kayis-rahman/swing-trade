package util

import (
	"context"
	"errors"
	"testing"
)

func TestRetryOptimisticRetriesOnlyVersionConflicts(t *testing.T) {
	attempts := 0
	err := RetryOptimistic(context.Background(), func(context.Context) error {
		attempts++
		if attempts < 3 {
			return ErrOptimisticLock
		}
		return nil
	})
	if err != nil || attempts != 3 {
		t.Fatalf("RetryOptimistic() = %v after %d attempts, want success after 3", err, attempts)
	}
}

func TestRetryOptimisticReturnsNonConflictWithoutRetry(t *testing.T) {
	want := errors.New("database unavailable")
	attempts := 0
	err := RetryOptimistic(context.Background(), func(context.Context) error {
		attempts++
		return want
	})
	if !errors.Is(err, want) || attempts != 1 {
		t.Fatalf("RetryOptimistic() = %v after %d attempts, want original error after 1", err, attempts)
	}
}

func TestRetryOptimisticStopsWhenContextIsCancelled(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	attempts := 0
	err := RetryOptimistic(ctx, func(context.Context) error {
		attempts++
		return ErrOptimisticLock
	})
	if !errors.Is(err, context.Canceled) || attempts != 1 {
		t.Fatalf("RetryOptimistic() = %v after %d attempts, want cancellation after 1", err, attempts)
	}
}
