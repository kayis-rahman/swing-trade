package db

import (
	"context"
	"errors"
	"sync"
	"testing"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type advisoryFakeRow struct {
	err   error
	value any
}

func (r advisoryFakeRow) Scan(dest ...any) error {
	if r.err != nil {
		return r.err
	}
	if len(dest) > 0 {
		if d, ok := dest[0].(*bool); ok {
			*d = r.value.(bool)
			_ = d
		}
	}
	return nil
}

type advisoryFakeConn struct {
	tryLockRow  advisoryFakeRow
	unlockRow   advisoryFakeRow
	released    bool
	releaseOnce sync.Mutex
}

func (c *advisoryFakeConn) Release() {
	c.releaseOnce.Lock()
	defer c.releaseOnce.Unlock()
	c.released = true
}

func (c *advisoryFakeConn) QueryRow(_ context.Context, sql string, _ ...any) pgx.Row {
	if contains(sql, "pg_try_advisory_lock") {
		return c.tryLockRow
	}
	return c.unlockRow
}

type advisoryFakePool struct {
	conn *advisoryFakeConn
	err  error
}

func (p *advisoryFakePool) Acquire(_ context.Context) (any, error) {
	if p.err != nil {
		return nil, p.err
	}
	return p.conn, nil
}

func TestWithAdvisoryLockCallbackNil(t *testing.T) {
	_, err := withAdvisoryLock(context.Background(), func(context.Context) (advisoryLockConn, error) { return nil, errors.New("nil pool") }, 1, nil)
	if err == nil || err.Error() != "advisory-lock callback is required" {
		t.Fatalf("err = %v", err)
	}
}

func TestWithAdvisoryLockAcquireError(t *testing.T) {
	want := errors.New("acquire failed")
	pool := &advisoryFakePool{err: want}
	_, err := withAdvisoryLock(context.Background(), func(context.Context) (advisoryLockConn, error) { return nil, pool.err }, 1, func(context.Context) error { return nil })
	if !errors.Is(err, want) {
		t.Fatalf("err = %v", err)
	}
}

func TestWithAdvisoryLockLockScanError(t *testing.T) {
	want := errors.New("scan failed")
	pool := &advisoryFakePool{conn: &advisoryFakeConn{tryLockRow: advisoryFakeRow{err: want}}}
	_, err := withAdvisoryLock(context.Background(), func(context.Context) (advisoryLockConn, error) { return pool.conn, nil }, 1, func(context.Context) error { return nil })
	if !errors.Is(err, want) {
		t.Fatalf("err = %v", err)
	}
}

func TestWithAdvisoryLockNotAcquired(t *testing.T) {
	pool := &advisoryFakePool{conn: &advisoryFakeConn{tryLockRow: advisoryFakeRow{value: false}}}
	acquired, err := withAdvisoryLock(context.Background(), func(context.Context) (advisoryLockConn, error) { return pool.conn, nil }, 1, func(context.Context) error { return nil })
	if err != nil || acquired {
		t.Fatalf("acquired=%v err=%v", acquired, err)
	}
}

func TestWithAdvisoryLockRunsCallbackAndUnlocks(t *testing.T) {
	conn := &advisoryFakeConn{
		tryLockRow: advisoryFakeRow{value: true},
		unlockRow:  advisoryFakeRow{value: true},
	}
	pool := &advisoryFakePool{conn: conn}
	called := false
	acquired, err := withAdvisoryLock(context.Background(), func(context.Context) (advisoryLockConn, error) {
		return pool.conn, nil
	}, 1, func(context.Context) error {
		called = true
		return nil
	})
	if err != nil || !acquired || !called {
		t.Fatalf("acquired=%v err=%v called=%v", acquired, err, called)
	}
	if !conn.released {
		t.Fatal("connection was not released")
	}
}

func TestWithAdvisoryLockPropagatesCallbackError(t *testing.T) {
	want := errors.New("callback failed")
	conn := &advisoryFakeConn{
		tryLockRow: advisoryFakeRow{value: true},
		unlockRow:  advisoryFakeRow{value: true},
	}
	pool := &advisoryFakePool{conn: conn}
	acquired, err := withAdvisoryLock(context.Background(), func(context.Context) (advisoryLockConn, error) {
		return pool.conn, nil
	}, 1, func(context.Context) error { return want })
	if !acquired || !errors.Is(err, want) {
		t.Fatalf("acquired=%v err=%v", acquired, err)
	}
}

func TestWithAdvisoryLockUnlockScanError(t *testing.T) {
	want := errors.New("unlock failed")
	conn := &advisoryFakeConn{
		tryLockRow: advisoryFakeRow{value: true},
		unlockRow:  advisoryFakeRow{err: want},
	}
	pool := &advisoryFakePool{conn: conn}
	_, err := withAdvisoryLock(context.Background(), func(context.Context) (advisoryLockConn, error) { return pool.conn, nil }, 1, func(context.Context) error { return nil })
	if !errors.Is(err, want) {
		t.Fatalf("err = %v", err)
	}
}

func TestWithAdvisoryLockUnlockNotHeld(t *testing.T) {
	conn := &advisoryFakeConn{
		tryLockRow: advisoryFakeRow{value: true},
		unlockRow:  advisoryFakeRow{value: false},
	}
	pool := &advisoryFakePool{conn: conn}
	_, err := withAdvisoryLock(context.Background(), func(context.Context) (advisoryLockConn, error) { return pool.conn, nil }, 1, func(context.Context) error { return nil })
	if err == nil || err.Error() != "PostgreSQL advisory lock was not held during release" {
		t.Fatalf("err = %v", err)
	}
}

func contains(s, sub string) bool {
	return len(s) >= len(sub) && (func() bool {
		for i := 0; i+len(sub) <= len(s); i++ {
			if s[i:i+len(sub)] == sub {
				return true
			}
		}
		return false
	})()
}

func TestWithAdvisoryLockExportedValidation(t *testing.T) {
	if _, err := WithAdvisoryLock(context.Background(), nil, 1, func(context.Context) error { return nil }); err == nil {
		t.Fatal("expected nil pool error")
	}
	_, _ = WithAdvisoryLock(context.Background(), nil, 1, nil)
	_, _ = WithAdvisoryLock(context.Background(), &pgxpool.Pool{}, 1, nil)
}
