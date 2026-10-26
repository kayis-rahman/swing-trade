package util

import (
	"context"
	"errors"
	"fmt"
	"time"
)

// ErrOptimisticLock marks a version conflict eligible for retry.
var ErrOptimisticLock = errors.New("optimistic lock conflict")

const maxAttempts = 3
const initialBackoff = 100 * time.Millisecond

// RetryOptimistic retries only optimistic-lock conflicts, matching Java's 3 attempts and 100/200ms delays.
func RetryOptimistic(ctx context.Context, action func(context.Context) error) error {
	for attempt := 1; attempt <= maxAttempts; attempt++ {
		err := action(ctx)
		if err == nil {
			return nil
		}
		if !errors.Is(err, ErrOptimisticLock) {
			return err
		}
		if attempt == maxAttempts {
			return fmt.Errorf("optimistic-lock action failed after %d attempts: %w", maxAttempts, err)
		}
		delay := initialBackoff << (attempt - 1)
		timer := time.NewTimer(delay)
		select {
		case <-ctx.Done():
			if !timer.Stop() {
				select {
				case <-timer.C:
				default:
				}
			}
			return fmt.Errorf("optimistic-lock retry interrupted: %w", ctx.Err())
		case <-timer.C:
		}
	}
	return nil
}
