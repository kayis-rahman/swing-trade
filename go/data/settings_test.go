package data

import (
	"context"
	"errors"
	"testing"

	"github.com/jackc/pgx/v5"
	"github.com/pashagolub/pgxmock/v4"
)

func TestSettingsReaderSelectedBrokerDefaultsAndValidates(t *testing.T) {
	for _, test := range []struct {
		name string
		row  func(pgxmock.PgxPoolIface)
		want string
	}{
		{name: "configured", row: func(mock pgxmock.PgxPoolIface) {
			mock.ExpectQuery("FROM app_settings").WithArgs("selectedBroker").WillReturnRows(pgxmock.NewRows([]string{"value"}).AddRow("fyers"))
		}, want: "fyers"},
		{name: "null value", row: func(mock pgxmock.PgxPoolIface) {
			mock.ExpectQuery("FROM app_settings").WithArgs("selectedBroker").WillReturnRows(pgxmock.NewRows([]string{"value"}).AddRow(nil))
		}, want: "yahoo"},
		{name: "missing value", row: func(mock pgxmock.PgxPoolIface) {
			mock.ExpectQuery("FROM app_settings").WithArgs("selectedBroker").WillReturnError(pgx.ErrNoRows)
		}, want: "yahoo"},
		{name: "unsupported value", row: func(mock pgxmock.PgxPoolIface) {
			mock.ExpectQuery("FROM app_settings").WithArgs("selectedBroker").WillReturnRows(pgxmock.NewRows([]string{"value"}).AddRow("unknown"))
		}, want: "yahoo"},
		{name: "whitespace value", row: func(mock pgxmock.PgxPoolIface) {
			mock.ExpectQuery("FROM app_settings").WithArgs("selectedBroker").WillReturnRows(pgxmock.NewRows([]string{"value"}).AddRow(" yahoo "))
		}, want: "yahoo"},
	} {
		t.Run(test.name, func(t *testing.T) {
			mock, err := pgxmock.NewPool()
			if err != nil {
				t.Fatal(err)
			}
			defer mock.Close()
			test.row(mock)
			got, err := NewSettingsReader(mock).GetSelectedBroker(context.Background())
			if err != nil || got != test.want {
				t.Fatalf("GetSelectedBroker() = %q, %v; want %q", got, err, test.want)
			}
			if err := mock.ExpectationsWereMet(); err != nil {
				t.Fatal(err)
			}
		})
	}
}

func TestSettingsReaderWrapsDatabaseFailure(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatal(err)
	}
	defer mock.Close()
	mock.ExpectQuery("FROM app_settings").WithArgs("selectedBroker").WillReturnError(errors.New("database unavailable"))
	if _, err := NewSettingsReader(mock).GetSelectedBroker(context.Background()); err == nil {
		t.Fatal("GetSelectedBroker() error = nil, want database error")
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}
