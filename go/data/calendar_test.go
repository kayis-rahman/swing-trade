package data

import (
	"context"
	"errors"
	"testing"
	"time"
)

type calendarHolidayStore struct {
	holiday *Holiday
	err     error
}

func (s calendarHolidayStore) GetHolidayByDate(context.Context, time.Time) (*Holiday, error) {
	return s.holiday, s.err
}

func (calendarHolidayStore) ListHolidaysByDateRange(context.Context, time.Time, time.Time) ([]Holiday, error) {
	return nil, nil
}

func (calendarHolidayStore) ListUpcomingHolidays(context.Context, time.Time) ([]Holiday, error) {
	return nil, nil
}

func TestIsNSETradingSessionMatchesHolidayAndWeekendRules(t *testing.T) {
	cases := []struct {
		name    string
		date    time.Time
		holiday *Holiday
		want    bool
	}{
		{name: "weekday", date: time.Date(2026, 10, 26, 0, 0, 0, 0, time.UTC), want: true},
		{name: "weekend", date: time.Date(2026, 10, 25, 0, 0, 0, 0, time.UTC), want: false},
		{name: "full holiday", date: time.Date(2026, 10, 26, 0, 0, 0, 0, time.UTC), holiday: &Holiday{Type: "FULL"}, want: false},
		{name: "mixed case full holiday", date: time.Date(2026, 10, 26, 0, 0, 0, 0, time.UTC), holiday: &Holiday{Type: "fUlL"}, want: false},
		{name: "partial holiday", date: time.Date(2026, 10, 25, 0, 0, 0, 0, time.UTC), holiday: &Holiday{Type: "PARTIAL"}, want: true},
	}
	for _, test := range cases {
		t.Run(test.name, func(t *testing.T) {
			got, err := IsNSETradingSession(context.Background(), test.date, calendarHolidayStore{holiday: test.holiday})
			if err != nil || got != test.want {
				t.Fatalf("IsNSETradingSession() = %v, %v; want %v", got, err, test.want)
			}
		})
	}
}

func TestIsNSETradingSessionPropagatesStoreFailures(t *testing.T) {
	wantErr := errors.New("db down")
	if _, err := IsNSETradingSession(context.Background(), time.Time{}, calendarHolidayStore{err: wantErr}); !errors.Is(err, wantErr) {
		t.Fatalf("error = %v, want wrapped db error", err)
	}
	if _, err := IsNSETradingSession(context.Background(), time.Time{}, nil); !errors.Is(err, errHolidayStoreUnavailable) {
		t.Fatalf("nil store error = %v", err)
	}
}
