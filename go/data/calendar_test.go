package data

import (
	"context"
	"errors"
	"testing"
	"time"
)

type calendarHolidayStore struct {
	holiday       *Holiday
	holidayByDate map[string]*Holiday
	err           error
}

func (s calendarHolidayStore) GetHolidayByDate(_ context.Context, date time.Time) (*Holiday, error) {
	if s.holidayByDate != nil {
		return s.holidayByDate[date.Format("2006-01-02")], s.err
	}
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

func TestExpectedNSETradingSessionsUsesInclusiveRangeAndHolidayOverrides(t *testing.T) {
	ist := time.FixedZone("IST", 5*60*60+30*60)
	from := time.Date(2026, 10, 24, 0, 0, 0, 0, ist) // Saturday
	to := time.Date(2026, 10, 26, 0, 0, 0, 0, ist)   // Monday
	store := calendarHolidayStore{holidayByDate: map[string]*Holiday{
		"2026-10-25": {Type: "PARTIAL"}, // Sunday Muhurat session
		"2026-10-26": {Type: "FULL"},
	}}
	sessions, err := ExpectedNSETradingSessions(context.Background(), from, to, store)
	if err != nil {
		t.Fatal(err)
	}
	if len(sessions) != 1 || sessions[0].Format("2006-01-02") != "2026-10-25" {
		t.Fatalf("sessions = %v, want only partial Sunday session", sessions)
	}
	empty, err := ExpectedNSETradingSessions(context.Background(), to, from, store)
	if err != nil || empty == nil || len(empty) != 0 {
		t.Fatalf("reversed range sessions = %v, %v; want empty", empty, err)
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
