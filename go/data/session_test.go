package data

import (
	"testing"
	"time"
)

func TestExpectedNSESessionBarStartsMatchJavaSessionGrid(t *testing.T) {
	date := time.Date(2026, 10, 26, 0, 0, 0, 0, time.UTC)
	bars := ExpectedNSESessionBarStarts(date, true)
	if len(bars) != 25 {
		t.Fatalf("bar count = %d, want 25", len(bars))
	}
	ist := time.FixedZone("IST", 5*60*60+30*60)
	if got := bars[0].In(ist).Format("15:04"); got != "09:15" {
		t.Fatalf("first bar = %s, want 09:15 IST", got)
	}
	if got := bars[24].In(ist).Format("15:04"); got != "15:15" {
		t.Fatalf("last bar = %s, want 15:15 IST", got)
	}
	for i := 1; i < len(bars); i++ {
		if got := bars[i].Sub(bars[i-1]); got != 15*time.Minute {
			t.Errorf("bar interval %d = %s, want 15m", i, got)
		}
	}
}

func TestExpectedNSESessionBarStartsEmptyForNonTradingDate(t *testing.T) {
	bars := ExpectedNSESessionBarStarts(time.Date(2026, 10, 25, 0, 0, 0, 0, time.UTC), false)
	if bars == nil || len(bars) != 0 {
		t.Fatalf("non-trading session bars = %#v, want empty slice", bars)
	}
}

func TestExpectedNSESessionBarStartsUsesISTCalendarDate(t *testing.T) {
	date := time.Date(2026, 10, 25, 20, 0, 0, 0, time.UTC) // 2026-10-26 in IST
	bars := ExpectedNSESessionBarStarts(date, true)
	if got := bars[0].Format("2006-01-02 15:04 MST"); got != "2026-10-26 09:15 IST" {
		t.Fatalf("first bar local timestamp = %s", got)
	}
}

func TestIsNormalSessionBarTimeMatchesJavaWindow(t *testing.T) {
	for name, tc := range map[string]struct {
		barTime time.Time
		want    bool
	}{
		"pre-open bar is rejected":    {barTime: time.Date(2000, 1, 1, 9, 0, 0, 0, time.UTC), want: false},
		"first session bar is kept":   {barTime: time.Date(2000, 1, 1, 9, 15, 0, 0, time.UTC), want: true},
		"mid-session bar is kept":     {barTime: time.Date(2000, 1, 1, 12, 0, 0, 0, time.UTC), want: true},
		"last session bar is kept":    {barTime: time.Date(2000, 1, 1, 15, 15, 0, 0, time.UTC), want: true},
		"exact close is rejected":     {barTime: time.Date(2000, 1, 1, 15, 30, 0, 0, time.UTC), want: false},
		"closing auction is rejected": {barTime: time.Date(2000, 1, 1, 15, 45, 0, 0, time.UTC), want: false},
	} {
		t.Run(name, func(t *testing.T) {
			if got := IsNormalSessionBarTime(tc.barTime); got != tc.want {
				t.Fatalf("IsNormalSessionBarTime(%s) = %v, want %v", tc.barTime.Format("15:04:05"), got, tc.want)
			}
		})
	}
}
