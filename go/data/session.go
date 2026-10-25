package data

import "time"

const (
	nseSessionBarMinutes = 15
	nseSessionBarCount   = 25
)

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
