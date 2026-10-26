package db

import (
	"context"
	"errors"
	"os"
	"testing"
	"time"
)

func TestWithAdvisoryLockSerializesWorkersAndReleases(t *testing.T) {
	databaseURL := os.Getenv("SWINGTRADE_TEST_DATABASE_URL")
	if databaseURL == "" {
		t.Skip("authorized integration database URL is unavailable")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	schema, err := NewTestSchema(ctx, databaseURL)
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		closeCtx, closeCancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer closeCancel()
		if err := schema.Close(closeCtx); err != nil {
			t.Errorf("close isolated schema: %v", err)
		}
	}()
	key := time.Now().UnixNano()
	called := false
	acquired, err := WithAdvisoryLock(ctx, schema.Pool, key, func(context.Context) error {
		called = true
		secondCalled := false
		secondAcquired, err := WithAdvisoryLock(ctx, schema.Pool, key, func(context.Context) error { secondCalled = true; return nil })
		if err != nil {
			return err
		}
		if secondAcquired || secondCalled {
			t.Error("second worker acquired a held lock")
		}
		return nil
	})
	if err != nil || !acquired || !called {
		t.Fatalf("acquired=%v called=%v err=%v", acquired, called, err)
	}
	acquired, err = WithAdvisoryLock(ctx, schema.Pool, key, func(context.Context) error { return errors.New("job failed") })
	if !acquired || err == nil || err.Error() != "job failed" {
		t.Fatalf("callback error path: acquired=%v err=%v", acquired, err)
	}
	acquired, err = WithAdvisoryLock(ctx, schema.Pool, key, func(context.Context) error { return nil })
	if err != nil || !acquired {
		t.Fatalf("lock was not released after callback error: acquired=%v err=%v", acquired, err)
	}
}

func TestWithAdvisoryLockRejectsMissingArguments(t *testing.T) {
	if _, err := WithAdvisoryLock(context.Background(), nil, 1, func(context.Context) error { return nil }); err == nil {
		t.Fatal("expected nil pool error")
	}
	if _, err := WithAdvisoryLock(context.Background(), nil, 1, nil); err == nil {
		t.Fatal("expected nil pool error")
	}
}
