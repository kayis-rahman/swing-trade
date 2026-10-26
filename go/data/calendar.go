package data

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"
)

var errHolidayStoreUnavailable = errors.New("holiday store is unavailable")

// DateOnlyIST maps an instant to its IST calendar date at UTC midnight, the
// date convention the Go candle stores, provider fetches, and schedulers use.
func DateOnlyIST(t time.Time) time.Time {
	ist := time.FixedZone("IST", 5*60*60+30*60)
	local := t.In(ist)
	return time.Date(local.Year(), local.Month(), local.Day(), 0, 0, 0, 0, time.UTC)
}

// ExpectedNSETradingSessions returns the trading dates in the inclusive date range.
func ExpectedNSETradingSessions(ctx context.Context, from, to time.Time, holidays HolidayReadStore) ([]time.Time, error) {
	ist := time.FixedZone("IST", 5*60*60+30*60)
	start, end := from.In(ist), to.In(ist)
	first := time.Date(start.Year(), start.Month(), start.Day(), 0, 0, 0, 0, ist)
	last := time.Date(end.Year(), end.Month(), end.Day(), 0, 0, 0, 0, ist)
	sessions := make([]time.Time, 0)
	for date := first; !date.After(last); date = date.AddDate(0, 0, 1) {
		trading, err := IsNSETradingSession(ctx, date, holidays)
		if err != nil {
			return nil, err
		}
		if trading {
			sessions = append(sessions, date)
		}
	}
	return sessions, nil
}

// IsNSETradingSession reports whether a date is a market session using Java's holiday-first decision.
func IsNSETradingSession(ctx context.Context, date time.Time, holidays HolidayReadStore) (bool, error) {
	if holidays == nil {
		return false, errHolidayStoreUnavailable
	}
	ist := time.FixedZone("IST", 5*60*60+30*60)
	localDate := date.In(ist)
	day := time.Date(localDate.Year(), localDate.Month(), localDate.Day(), 0, 0, 0, 0, time.UTC)
	holiday, err := holidays.GetHolidayByDate(ctx, day)
	if err != nil {
		return false, fmt.Errorf("look up NSE holiday: %w", err)
	}
	if holiday != nil {
		return !strings.EqualFold(holiday.Type, "FULL"), nil
	}
	weekday := localDate.Weekday()
	return weekday != time.Saturday && weekday != time.Sunday, nil
}
