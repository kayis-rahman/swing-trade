package data

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/pashagolub/pgxmock/v4"
)

func TestCoverageReaderRequiresEveryYearVerified(t *testing.T) {
	for name, tc := range map[string]struct {
		rows []any // [year, status] pairs
		want bool
	}{
		"all years verified": {rows: []any{int32(2026), "VERIFIED", int32(2027), "VERIFIED"}, want: true},
		"pending year fails": {rows: []any{int32(2026), "VERIFIED", int32(2027), "PENDING"}, want: false},
		"missing year fails": {rows: []any{int32(2026), "VERIFIED"}, want: false},
	} {
		t.Run(name, func(t *testing.T) {
			mock, err := pgxmock.NewPool()
			if err != nil {
				t.Fatal(err)
			}
			defer mock.Close()
			rows := pgxmock.NewRows([]string{"calendar_year", "status"})
			for i := 0; i+1 < len(tc.rows); i += 2 {
				rows.AddRow(tc.rows[i], tc.rows[i+1])
			}
			mock.ExpectQuery("calendar_year").WithArgs(int32(2026), int32(2027)).WillReturnRows(rows)
			got, err := NewCoverageReader(mock).IsCoverageVerified(context.Background(),
				time.Date(2026, 3, 1, 0, 0, 0, 0, time.UTC),
				time.Date(2027, 4, 1, 0, 0, 0, 0, time.UTC))
			if err != nil || got != tc.want {
				t.Fatalf("IsCoverageVerified = %v, %v, want %v", got, err, tc.want)
			}
			if err := mock.ExpectationsWereMet(); err != nil {
				t.Fatal(err)
			}
		})
	}
}

func TestCoverageReaderWrapsDatabaseErrors(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatal(err)
	}
	defer mock.Close()
	mock.ExpectQuery("calendar_year").WillReturnError(errors.New("db down"))
	if _, err := NewCoverageReader(mock).IsCoverageVerified(context.Background(),
		time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
		time.Date(2026, 2, 1, 0, 0, 0, 0, time.UTC)); err == nil {
		t.Fatal("coverage error = nil")
	}
}
