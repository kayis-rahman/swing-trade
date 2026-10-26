package db

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

// WithAdvisoryLock runs fn only after acquiring a session-scoped PostgreSQL lock.
// A false acquired result means another worker already owns the lock.
func WithAdvisoryLock(ctx context.Context, pool *pgxpool.Pool, key int64, fn func(context.Context) error) (acquired bool, err error) {
	if pool == nil {
		return false, fmt.Errorf("advisory-lock pool is required")
	}
	if fn == nil {
		return false, fmt.Errorf("advisory-lock callback is required")
	}
	connection, err := pool.Acquire(ctx)
	if err != nil {
		return false, fmt.Errorf("acquire advisory-lock connection: %w", err)
	}
	defer connection.Release()
	if err := connection.QueryRow(ctx, "SELECT pg_try_advisory_lock($1)", key).Scan(&acquired); err != nil {
		return false, fmt.Errorf("acquire PostgreSQL advisory lock: %w", err)
	}
	if !acquired {
		return false, nil
	}
	defer func() {
		unlockCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), 5*time.Second)
		defer cancel()
		var unlocked bool
		if unlockErr := connection.QueryRow(unlockCtx, "SELECT pg_advisory_unlock($1)", key).Scan(&unlocked); unlockErr != nil {
			if err == nil {
				err = fmt.Errorf("release PostgreSQL advisory lock: %w", unlockErr)
			}
			return
		}
		if !unlocked && err == nil {
			err = fmt.Errorf("PostgreSQL advisory lock was not held during release")
		}
	}()
	if err := fn(ctx); err != nil {
		return true, err
	}
	return true, nil
}
