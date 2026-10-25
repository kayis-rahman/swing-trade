package sqlc

import (
	"context"
	"errors"
	"testing"

	"github.com/pashagolub/pgxmock/v4"
)

func TestGeneratedGetAppSettingQueryAndScanErrors(t *testing.T) {
	for _, test := range []struct {
		name string
		rows func(pgxmock.PgxPoolIface) error
	}{
		{name: "query", rows: func(mock pgxmock.PgxPoolIface) error {
			mock.ExpectQuery("FROM app_settings").WithArgs("selectedBroker").WillReturnError(errors.New("query failed"))
			_, err := New(mock).GetAppSetting(context.Background(), "selectedBroker")
			return err
		}},
		{name: "scan", rows: func(mock pgxmock.PgxPoolIface) error {
			mock.ExpectQuery("FROM app_settings").WithArgs("selectedBroker").WillReturnRows(pgxmock.NewRows([]string{"value"}).AddRow(struct{}{}))
			_, err := New(mock).GetAppSetting(context.Background(), "selectedBroker")
			return err
		}},
	} {
		t.Run(test.name, func(t *testing.T) {
			mock, err := pgxmock.NewPool()
			if err != nil {
				t.Fatal(err)
			}
			defer mock.Close()
			if err := test.rows(mock); err == nil {
				t.Fatal("GetAppSetting() error = nil, want error")
			}
			if err := mock.ExpectationsWereMet(); err != nil {
				t.Fatal(err)
			}
		})
	}
}

func TestGeneratedGetAppSettingReadsNullableValue(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatal(err)
	}
	defer mock.Close()
	mock.ExpectQuery("FROM app_settings").WithArgs("selectedBroker").WillReturnRows(pgxmock.NewRows([]string{"value"}).AddRow("fyers"))
	value, err := New(mock).GetAppSetting(context.Background(), "selectedBroker")
	if err != nil || !value.Valid || value.String != "fyers" {
		t.Fatalf("GetAppSetting() = %#v, %v", value, err)
	}
	mock.ExpectQuery("FROM app_settings").WithArgs("selectedBroker").WillReturnRows(pgxmock.NewRows([]string{"value"}).AddRow(nil))
	value, err = New(mock).GetAppSetting(context.Background(), "selectedBroker")
	if err != nil || value.Valid {
		t.Fatalf("GetAppSetting(null) = %#v, %v", value, err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}
