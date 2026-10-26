package db

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
)

type transactionBeginner interface {
	Begin(context.Context) (pgx.Tx, error)
}

// InTransaction commits fn on success and rolls back on error or cancellation.
func InTransaction(ctx context.Context, pool transactionBeginner, fn func(context.Context, pgx.Tx) error) error {
	if pool == nil {
		return fmt.Errorf("transaction pool is required")
	}
	if fn == nil {
		return fmt.Errorf("transaction callback is required")
	}
	tx, err := pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin transaction: %w", err)
	}
	defer func() {
		rollbackCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), 5*time.Second)
		defer cancel()
		_ = tx.Rollback(rollbackCtx)
	}()
	if err := fn(ctx, tx); err != nil {
		return fmt.Errorf("transaction callback: %w", err)
	}
	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit transaction: %w", err)
	}
	return nil
}
