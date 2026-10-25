package data

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"
)

var errHolidayStoreUnavailable = errors.New("holiday store is unavailable")

// IsNSETradingSession mirrors Java MarketCalendar's holiday-first decision.
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
