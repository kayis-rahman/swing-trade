package db

import (
	"context"
	"errors"
	"testing"

	"github.com/jackc/pgx/v5"
	"github.com/pashagolub/pgxmock/v4"
)

func TestInTransactionCommitsOnSuccess(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatal("create transaction mock")
	}
	defer mock.Close()
	mock.ExpectBegin()
	mock.ExpectExec("UPDATE accounts SET balance = \\$1").WithArgs(42).WillReturnResult(pgxmock.NewResult("UPDATE", 1))
	mock.ExpectCommit()

	err = InTransaction(context.Background(), mock, func(ctx context.Context, tx pgx.Tx) error {
		_, err := tx.Exec(ctx, "UPDATE accounts SET balance = $1", 42)
		return err
	})
	if err != nil {
		t.Fatalf("InTransaction() error = %v", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal("transaction expectations were not met")
	}
}

func TestInTransactionRollsBackCallbackError(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatal("create transaction mock")
	}
	defer mock.Close()
	mock.ExpectBegin()
	mock.ExpectRollback()
	want := errors.New("callback failed")
	err = InTransaction(context.Background(), mock, func(context.Context, pgx.Tx) error { return want })
	if !errors.Is(err, want) {
		t.Fatalf("InTransaction() error = %v, want wrapped callback error", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal("transaction expectations were not met")
	}
}

func TestInTransactionReturnsBeginError(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatal("create transaction mock")
	}
	defer mock.Close()
	want := errors.New("begin failed")
	mock.ExpectBegin().WillReturnError(want)
	if err := InTransaction(context.Background(), mock, func(context.Context, pgx.Tx) error { return nil }); !errors.Is(err, want) {
		t.Fatalf("InTransaction() error = %v, want wrapped begin error", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal("transaction expectations were not met")
	}
}

func TestInTransactionReturnsCommitError(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatal("create transaction mock")
	}
	defer mock.Close()
	want := errors.New("commit failed")
	mock.ExpectBegin()
	mock.ExpectCommit().WillReturnError(want)
	mock.ExpectRollback()
	if err := InTransaction(context.Background(), mock, func(context.Context, pgx.Tx) error { return nil }); !errors.Is(err, want) {
		t.Fatalf("InTransaction() error = %v, want wrapped commit error", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal("transaction expectations were not met")
	}
}

func TestInTransactionRejectsMissingDependencies(t *testing.T) {
	if err := InTransaction(context.Background(), nil, func(context.Context, pgx.Tx) error { return nil }); err == nil {
		t.Fatal("InTransaction() succeeded with nil pool")
	}
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatal("create transaction mock")
	}
	defer mock.Close()
	if err := InTransaction(context.Background(), mock, nil); err == nil {
		t.Fatal("InTransaction() succeeded with nil callback")
	}
}
