package data

import "time"

const (
	nseSessionBarMinutes = 15
	nseSessionBarCount   = 25
)

// IsNormalSessionBarTime reports whether a bar start falls inside the normal
// 09:15 to 15:30 IST session window, matching Java's
// NseSessionCalendar.isNormalSessionBarTime: not before 09:15 and strictly
// before 15:30, so pre-open and closing-auction bars are rejected. The
// bar-time components are compared as stored, exactly like Java's zone-less
// LocalTime check.
func IsNormalSessionBarTime(barTime time.Time) bool {
	clock := time.Date(2000, 1, 1, barTime.Hour(), barTime.Minute(), barTime.Second(), 0, time.UTC)
	start := time.Date(2000, 1, 1, 9, 15, 0, 0, time.UTC)
	end := time.Date(2000, 1, 1, 15, 30, 0, 0, time.UTC)
	return !clock.Before(start) && clock.Before(end)
}

// ExpectedNSESessionBarStarts returns the 25 regular-session 15-minute bar starts
// from 09:15 through 15:15 IST. The caller supplies the exchange-session decision
// because holiday/Muhurat policy belongs to the market-calendar service.
func ExpectedNSESessionBarStarts(date time.Time, isTradingSession bool) []time.Time {
	if !isTradingSession {
		return []time.Time{}
	}
	ist := time.FixedZone("IST", 5*60*60+30*60)
	localDate := date.In(ist)
	barTime := time.Date(localDate.Year(), localDate.Month(), localDate.Day(), 9, 15, 0, 0, ist)
	bars := make([]time.Time, nseSessionBarCount)
	for i := range bars {
		bars[i] = barTime
		barTime = barTime.Add(nseSessionBarMinutes * time.Minute)
	}
	return bars
}
